package 无下限术士

import (
	"embed"
	"math"
	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const KindTwin = "无下限术士"
const KindRed = "红半"
const KindBlue = "蓝半"
const KindTwinArc = "无下限术士弧"
const KindTwinBlueArc = "无下限弧"

const (
	twinRadius    = 18.0
	twinSpeed     = 200.0
	twinHP        = 100.0
	halfHP        = 9999.0
	twinDamage    = 7.0
	twinHitCD     = 0.1
	twinGap       = 2.0
	twinVision    = 9999.0
	twinPull      = 300.0
	twinPush      = -100.0
	twinArcInner  = twinRadius
	twinArcOuter  = twinRadius + 2
	twinArcSpan   = 90.0
	twinRedColor  = "#e24b4b"
	twinBlueColor = "#4b7be2"

	purpleKind      = "紫弹"
	purpleRadius    = 32.0
	purpleSpeed     = 300.0
	purpleDamage    = 12.0
	purpleCD        = 0.5
	purpleOffscreen = 640.0
)

func init() {
	p := unit.NewPack(KindTwin, assets)
	p.Register(unit.Spec{
		Kind:     KindTwin,
		Role:     unit.RoleFighter,
		Radius:   0,
		MaxHP:    twinHP,
		Speed:    twinSpeed,
		Vision:   twinVision,
		Fighter:  true,
		Nonsolid: true,
		Look:     unit.Look{Color: twinRedColor},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &无下限术士{slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:        KindRed,
		Role:        unit.RoleMinion,
		Radius:      twinRadius,
		MaxHP:       halfHP,
		Speed:       twinSpeed,
		Vision:      twinVision,
		Mortal:      true,
		Semi:        true,
		Cruise:      true,
		AimPriority: unit.DefaultFighterAim,
		Look:        unit.Look{Color: twinRedColor, FX: []string{"pull", "bond"}, ShareHP: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &半{owner: info.OwnerID, slot: info.Slot, red: true}
	})
	p.Register(unit.Spec{
		Kind:     KindTwinArc,
		Role:     unit.RoleProjectile,
		Radius:   twinArcOuter,
		MaxHP:    1,
		Speed:    twinSpeed,
		Vision:   0,
		Fighter:  false,
		Semi:     true,
		Attach:   true,
		ArcSpan:  unit.Deg(twinArcSpan),
		ArcInner: twinArcInner,
		Look:     unit.Look{Color: twinRedColor, Overlay: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &术士弧{slot: info.Slot, dmg: twinDamage}
	})
	p.Register(unit.Spec{
		Kind:        KindBlue,
		Role:        unit.RoleMinion,
		Radius:      twinRadius,
		MaxHP:       halfHP,
		Speed:       twinSpeed,
		Vision:      twinVision,
		Mortal:      true,
		Semi:        true,
		AimPriority: unit.DefaultFighterAim,
		Look:        unit.Look{Color: twinBlueColor, FX: []string{"push", "bond"}, ShareHP: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &半{owner: info.OwnerID, slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:     KindTwinBlueArc,
		Role:     unit.RoleProjectile,
		Radius:   twinArcOuter,
		MaxHP:    1,
		Speed:    twinSpeed,
		Vision:   0,
		Fighter:  false,
		Semi:     true,
		Attach:   true,
		ArcSpan:  unit.Deg(twinArcSpan),
		ArcInner: twinArcInner,
		Look:     unit.Look{Color: twinBlueColor, Overlay: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &术士弧{slot: info.Slot, dmg: twinDamage}
	})
	p.Register(unit.Spec{
		Kind:      purpleKind,
		Role:      unit.RoleProjectile,
		Radius:    purpleRadius,
		MaxHP:     1,
		Speed:     purpleSpeed,
		Vision:    0,
		Fighter:   false,
		PassWalls: true,
		Look:      unit.Look{Color: "#b44cff", Glow: true, Trail: true, Overlay: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &紫弹{slot: info.Slot}
	})
}

type 无下限术士 struct {
	slot    int
	spawned bool
}

func (d *无下限术士) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok || d.spawned {
		return
	}
	d.spawned = true
	ctx.Out <- unit.NoHealthNumbers{UnitID: ctx.ID, Hold: true}
	unit.SetAim(ctx, ctx.ID, 0)
	split(ctx, s)
}

func split(ctx unit.Context, s unit.Sense) {
	sp := math.Hypot(s.Self.VX, s.Self.VY)
	vx, vy := s.Self.VX, s.Self.VY
	if sp < 1e-6 {
		sp = twinSpeed
		vx, vy = twinSpeed, 0
	}
	px, py := vx/sp, vy/sp
	ctx.Out <- unit.Spawn{
		Kind: KindRed, X: s.Self.X, Y: s.Self.Y, VX: vx, VY: vy,
		OwnerID: ctx.ID, Slot: s.Self.Slot,
	}
	ctx.Out <- unit.Spawn{
		Kind:    KindBlue,
		X:       s.Self.X - px*twinGap,
		Y:       s.Self.Y - py*twinGap,
		VX:      -vx,
		VY:      -vy,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	}
	ctx.Out <- unit.FX{
		Name: "split", Kind: KindRed,
		X: s.Self.X, Y: s.Self.Y, VX: vx, VY: vy, Slot: s.Self.Slot,
	}
	ctx.Out <- unit.FX{
		Name: "split", Kind: KindBlue,
		X: s.Self.X - px*twinGap, Y: s.Self.Y - py*twinGap, VX: -vx, VY: -vy, Slot: s.Self.Slot,
	}
}

type 半 struct {
	owner       uint64
	slot        int
	red         bool
	arc         unit.AttachState
	shotReadyAt float64
	selfX       float64
	selfY       float64
	enemyX      float64
	enemyY      float64
	hasEnemy    bool
}

func (h *半) Handle(ctx unit.Context, ev unit.Event) {
	if passHurt(ctx, ev, h.owner) {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		kind := KindTwinBlueArc
		if h.red {
			kind = KindTwinArc
		}
		if unit.RearmAttach(e, ctx.ID, kind, twinHitCD, &h.arc) {
			unit.SpawnAttach(ctx, e, kind)
		}
		h.remember(e)
		str := twinPush
		if h.red {
			str = twinPull
		}
		twinField(ctx, e, str)
	case unit.Collision:
		if h.red && e.Other.Kind == KindBlue && e.Other.Slot == h.slot {
			h.tryShot(ctx, e)
		}
	}
}

func (h *半) remember(s unit.Sense) {
	h.selfX, h.selfY = s.Self.X, s.Self.Y
	h.hasEnemy = false
	if o := unit.Seek(s); o != nil {
		h.enemyX, h.enemyY = o.X, o.Y
		h.hasEnemy = true
	}
}

func (h *半) tryShot(ctx unit.Context, e unit.Collision) {
	if !h.hasEnemy || e.Time < h.shotReadyAt {
		return
	}
	mx := (h.selfX + e.Other.X) / 2
	my := (h.selfY + e.Other.Y) / 2
	dx := h.enemyX - mx
	dy := h.enemyY - my
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	ux, uy := dx/n, dy/n
	ctx.Out <- unit.Spawn{
		Kind: purpleKind, X: mx, Y: my,
		VX: ux * purpleSpeed, VY: uy * purpleSpeed,
		OwnerID: ctx.ID, Slot: h.slot,
	}
	ctx.Out <- unit.FX{
		Name: "void-shot", UnitID: ctx.ID, Kind: purpleKind,
		X: mx, Y: my, VX: ux * purpleSpeed, VY: uy * purpleSpeed, Slot: h.slot,
	}
	h.shotReadyAt = e.Time + purpleCD
}

func passHurt(ctx unit.Context, ev unit.Event, owner uint64) bool {
	d, ok := ev.(unit.IncomingDamage)
	if !ok {
		return false
	}
	unit.ConfirmHit(ctx, d)
	if owner != 0 && d.Amount > 0 {
		ctx.Out <- unit.Damage{From: d.From, To: owner, Amount: d.Amount}
	}
	return true
}

type 术士弧 struct {
	slot int
	dmg  float64
}

func (a *术士弧) Handle(ctx unit.Context, ev unit.Event) {
	e, ok := ev.(unit.Collision)
	if !ok || !unit.EnemyTarget(e, a.slot) {
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: a.dmg}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

type 紫弹 struct {
	slot       int
	hitReadyAt float64
}

func (b *紫弹) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		if math.Abs(e.Self.X) > purpleOffscreen || math.Abs(e.Self.Y) > purpleOffscreen {
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		}
	case unit.Collision:
		if e.Other.Slot == b.slot || !unit.Hittable(e.Other, b.slot) {
			return
		}
		if e.Time < b.hitReadyAt {
			return
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: purpleDamage}
		ctx.Out <- unit.FX{
			Name: "void-hit", Kind: purpleKind,
			X: e.Other.X, Y: e.Other.Y, Slot: e.Other.Slot,
		}
		b.hitReadyAt = e.Time + twinHitCD
	}
}

func twinField(ctx unit.Context, s unit.Sense, strength float64) {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Slot == s.Self.Slot || o.Role == unit.RoleProjectile || o.Nonsolid {
			continue
		}
		dx := s.Self.X - o.X
		dy := s.Self.Y - o.Y
		n := math.Hypot(dx, dy)
		if n < 1e-6 {
			continue
		}
		ctx.Out <- unit.Force{
			UnitID: o.ID,
			AX:     dx / n * strength,
			AY:     dy / n * strength,
		}
	}
}
