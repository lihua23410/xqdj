package 小骑士

import (
	"embed"
	"math"
	"math/rand/v2"
	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const KindKnight = "小骑士"
const KindKnightArc = "小骑士弧"

const (
	knightRadius   = 18.0
	knightSpeed    = 178.0
	knightHP       = 100.0
	knightStartHP  = 85.0
	knightVision   = 9999.0
	knightDamage   = 9.0
	knightHitCD    = 0.1
	knightCD       = 6.0
	knightChainGap = 0.5
	ramSpeed       = 350.0
	knightArcInner = knightRadius
	knightArcOuter = knightRadius + 2
	knightArcSpan  = 120.0
	knightColor    = "#ff2a2a"
)

func init() {
	p := unit.NewPack(KindKnight, assets)
	p.Register(unit.Spec{
		Kind:    KindKnight,
		Role:    unit.RoleFighter,
		Radius:  knightRadius,
		MaxHP:   knightHP,
		StartHP: knightStartHP,
		Speed:   knightSpeed,
		Vision:  knightVision,
		Fighter: true,
		Look:    unit.Look{Color: knightColor, Ghost: 250},
	}, func(unit.SpawnInfo) unit.Actor {
		return &小骑士{readyAt: knightCD}
	})
	p.Register(unit.Spec{
		Kind:     KindKnightArc,
		Role:     unit.RoleProjectile,
		Radius:   knightArcOuter,
		MaxHP:    1,
		Speed:    knightSpeed,
		Vision:   0,
		Fighter:  false,
		Attach:   true,
		ArcSpan:  unit.Deg(knightArcSpan),
		ArcInner: knightArcInner,
		Look:     unit.Look{Color: knightColor, Overlay: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &骑士弧{slot: info.Slot}
	})
}

type 小骑士 struct {
	readyAt float64
	arc     unit.AttachState
}

func (k *小骑士) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if unit.RearmAttach(s, ctx.ID, KindKnightArc, knightHitCD, &k.arc) {
		unit.SpawnAttach(ctx, s, KindKnightArc)
	}
	k.tryRam(ctx, s)
}

type 骑士弧 struct {
	slot int
}

func (a *骑士弧) Handle(ctx unit.Context, ev unit.Event) {
	e, ok := ev.(unit.Collision)
	if !ok || !unit.EnemyFighter(e, a.slot) {
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: knightDamage}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

func (k *小骑士) tryRam(ctx unit.Context, sense unit.Sense) {
	if sense.Time+1e-9 < k.readyAt {
		return
	}
	var enemy *unit.Snapshot
	for i := range sense.Nearby {
		o := &sense.Nearby[i]
		if o.Role != unit.RoleFighter || o.Slot == sense.Self.Slot {
			continue
		}
		enemy = o
		break
	}
	if enemy == nil {
		return
	}
	tx, ty := beside(sense.Self, *enemy)
	ctx.Out <- unit.FX{
		Name: "blink", Kind: ctx.Kind,
		X: sense.Self.X, Y: sense.Self.Y, Slot: sense.Self.Slot,
	}
	ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: tx, Y: ty}
	dx := enemy.X - tx
	dy := enemy.Y - ty
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		dx, dy, n = 1, 0, 1
	}
	vx, vy := dx/n*ramSpeed, dy/n*ramSpeed
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: vx, VY: vy}
	ctx.Out <- unit.FX{
		Name: "dash", Kind: ctx.Kind,
		X: tx, Y: ty, VX: vx, VY: vy, Slot: sense.Self.Slot,
	}
	if rand.Float64() < 0.5 {
		k.readyAt = sense.Time + knightChainGap
	} else {
		k.readyAt = sense.Time + knightCD
	}
}

func beside(self, enemy unit.Snapshot) (float64, float64) {
	gap := self.Radius + enemy.Radius + 8
	ex, ey := enemy.X, enemy.Y
	base := math.Atan2(ey, ex)
	for _, extra := range []float64{math.Pi / 2, -math.Pi / 2, math.Pi / 3, -math.Pi / 3, 2 * math.Pi / 3, -2 * math.Pi / 3, math.Pi} {
		ang := base + extra
		x := ex + math.Cos(ang)*gap
		y := ey + math.Sin(ang)*gap
		if unit.HexContains(x, y, self.Radius+4) {
			return x, y
		}
	}
	return 0, 0
}
