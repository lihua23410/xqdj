// 小幺哥：本体只有 80 血，随机游走并丢出礼物盒。
// 礼物盒会从全部可选战斗机里随机召唤一只临时帮手；帮手正常战斗，但不计入胜负。
package 小幺哥

import (
	"embed"
	"encoding/json"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

const (
	KindXiaoyaoge = "小幺哥"
	KindGiftBox   = "小幺哥礼物盒"
)

const (
	bodyRadius = 18.0
	bodyHP     = 80.0
	bodySpeed  = 152.0
	bodyVision = 9999.0
	bodyColor  = "#ff655f"

	giftFlight = 0.9
	giftSpeed  = 330.0
	giftRadius = 12.0
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindXiaoyaoge, assets)
	p.Register(unit.Spec{
		Kind:    KindXiaoyaoge,
		Role:    unit.RoleFighter,
		Radius:  bodyRadius,
		MaxHP:   bodyHP,
		Speed:   bodySpeed,
		Vision:  bodyVision,
		Fighter: true,
		Look: unit.Look{
			Color:      bodyColor,
			VisionRing: true,
			FX:         []string{"xiaoyaoge"},
		},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &小幺哥{rng: unit.NewRNG(info.Stream)}
	})

	p.Register(unit.Spec{
		Kind:   KindGiftBox,
		Role:   unit.RoleProjectile,
		Radius: giftRadius,
		MaxHP:  1,
		Speed:  giftSpeed,
		Vision: 0,
		Pass:   true,
		Look: unit.Look{
			Color: "#ffcb55",
			Glow:  true,
			FX:    []string{"xiaoyaoge-gift"},
		},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &礼物盒{owner: info.OwnerID, slot: info.Slot, rng: unit.NewRNG(info.Stream)}
	})
}

type 小幺哥 struct {
	rng      *rand.Rand
	booted   bool
	nextGift float64
	thrown   bool
	nextTurn float64
	x, y     float64
	hx, hy   float64
}

func (x *小幺哥) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		unit.ConfirmHit(ctx, e)
	case unit.Sense:
		x.onSense(ctx, e)
	case unit.WallHit:
		x.turnRandom(ctx)
	}
}

func (x *小幺哥) onSense(ctx unit.Context, s unit.Sense) {
	x.x, x.y = s.Self.X, s.Self.Y
	if speed := math.Hypot(s.Self.VX, s.Self.VY); speed > 1e-6 {
		x.hx, x.hy = s.Self.VX/speed, s.Self.VY/speed
	}
	if !x.booted {
		x.booted = true
		if x.rng == nil {
			x.rng = unit.NewRNG(ctx.Stream)
		}
		x.nextGift = s.Time + 1.0
		x.nextTurn = s.Time + 2.0 + x.rng.Float64()*2.2
		x.turnRandom(ctx)
		ctx.Out <- unit.FX{Name: "xiaoyaoge-idle", Kind: KindXiaoyaoge, UnitID: ctx.ID, Slot: s.Self.Slot, X: x.x, Y: x.y}
	}
	if !x.thrown && s.Time+1e-9 >= x.nextGift {
		x.throwGift(ctx, s)
		x.thrown = true
	}
	if s.Time+1e-9 >= x.nextTurn {
		x.nextTurn = s.Time + 2.0 + x.rng.Float64()*2.4
		x.turnRandom(ctx)
	}
}

func (x *小幺哥) turnRandom(ctx unit.Context) {
	if x.rng == nil {
		x.rng = unit.NewRNG(ctx.Stream)
	}
	angle := x.rng.Float64() * 2 * math.Pi
	x.hx, x.hy = math.Cos(angle), math.Sin(angle)
	ctx.Out <- unit.SetFSDirection{UnitID: ctx.ID, VX: x.hx, VY: x.hy}
}

func (x *小幺哥) throwGift(ctx unit.Context, s unit.Sense) {
	base := math.Atan2(x.hy, x.hx)
	if target := unit.Seek(s); target != nil {
		base = math.Atan2(target.Y-x.y, target.X-x.x)
	}
	angle := base + (x.rng.Float64()*2-1)*unit.Deg(28)
	ux, uy := math.Cos(angle), math.Sin(angle)
	start := bodyRadius + giftRadius + 6
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind:    KindGiftBox,
		X:       x.x + ux*start,
		Y:       x.y + uy*start,
		VX:      ux * giftSpeed,
		VY:      uy * giftSpeed,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	})
	ctx.Out <- unit.FX{
		Name: "gift-throw", Kind: KindXiaoyaoge, UnitID: ctx.ID, Slot: s.Self.Slot,
		X: x.x, Y: x.y, VX: ux, VY: uy,
	}
}

type giftReveal struct {
	Helper string `json:"helper"`
}

type 礼物盒 struct {
	owner  uint64
	slot   int
	rng    *rand.Rand
	bornAt float64
	booted bool
	done   bool
	x, y   float64
}

func (g *礼物盒) Handle(ctx unit.Context, ev unit.Event) {
	if g.done {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		g.x, g.y = e.Self.X, e.Self.Y
		if !g.booted {
			g.booted = true
			g.bornAt = e.Time
		} else if e.Time-g.bornAt >= giftFlight {
			g.open(ctx)
		}
	case unit.WallHit:
		g.x, g.y = e.X, e.Y
		g.open(ctx)
	}
}

func (g *礼物盒) open(ctx unit.Context) {
	if g.done {
		return
	}
	g.done = true
	if g.rng == nil {
		g.rng = unit.NewRNG(ctx.Stream)
	}
	pool := make([]string, 0, 64)
	for _, kind := range unit.FighterKinds() {
		if kind == KindXiaoyaoge || kind == "靶子" {
			continue
		}
		pool = append(pool, kind)
	}
	helper := KindXiaoyaoge
	if len(pool) > 0 {
		helper = pool[g.rng.IntN(len(pool))]
	}
	angle := g.rng.Float64() * 2 * math.Pi
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind:      helper,
		X:         g.x,
		Y:         g.y,
		VX:        math.Cos(angle) * 120,
		VY:        math.Sin(angle) * 120,
		OwnerID:   g.owner,
		Slot:      g.slot,
		Temporary: true,
	})
	ctx.Out <- unit.FX{
		Name: "gift-open", Kind: KindXiaoyaoge, UnitID: g.owner, Slot: g.slot,
		X: g.x, Y: g.y, Amount: float64(len(pool)),
		Data: mustJSON(giftReveal{Helper: helper}),
	}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

func mustJSON(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
