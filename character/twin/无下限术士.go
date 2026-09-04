package 无下限术士

import (
	"embed"
	"math"
	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const KindTwin = "无下限术士"
const KindTwinArc = "无下限术士弧"
const KindTwinBlueArc = "无下限弧"

const (
	twinRadius    = 18.0
	twinSpeed     = 200.0
	twinHP        = 100.0
	twinDamage    = 7.0
	twinHitCD     = 0.1
	twinGap       = 2.0
	twinKind      = "无下限"
	twinVision    = 9999.0
	twinPull      = 300.0
	twinPush      = -100.0
	twinArcInner  = twinRadius
	twinArcOuter  = twinRadius + 2
	twinArcSpan   = 90.0
	twinRedColor  = "#e24b4b"
	twinBlueColor = "#4b7be2"

	purpleKind      = "紫弹"
	purpleRadius    = 32.0 // 约 5.6 倍体积（相对 r=18）
	purpleSpeed     = 300.0
	purpleDamage    = 12.0
	purpleCD        = 0.5
	purpleOffscreen = 640.0
)

func init() {
	p := unit.NewPack(KindTwin, assets)
	p.Register(unit.Spec{
		Kind:    KindTwin,
		Role:    unit.RoleFighter,
		Radius:  twinRadius,
		MaxHP:   twinHP,
		Speed:   twinSpeed,
		Vision:  twinVision,
		Fighter: true,
		Semi:    true,
		Look:    unit.Look{Color: twinRedColor, FX: []string{"pull", "bond"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &无下限术士{slot: info.Slot}
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
		Kind:    twinKind,
		Role:    unit.RoleTwin,
		Radius:  twinRadius,
		MaxHP:   twinHP,
		Speed:   twinSpeed,
		Vision:  twinVision,
		Fighter: false,
		Semi:    true,
		Look:    unit.Look{Color: twinBlueColor, FX: []string{"push", "bond"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &双生{owner: info.OwnerID, slot: info.Slot}
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
	slot        int
	spawned     bool
	arc         unit.AttachState
	shotReadyAt float64
	selfX       float64
	selfY       float64
	enemyX      float64
	enemyY      float64
	hasEnemy    bool
}

func (d *无下限术士) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		if !d.spawned {
			d.spawned = true
			d.split(ctx, e)
		}
		if unit.RearmAttach(e, ctx.ID, KindTwinArc, twinHitCD, &d.arc) {
			unit.SpawnAttach(ctx, e, KindTwinArc)
		}
		d.remember(e)
		twinField(ctx, e, twinPull)
	case unit.Collision:
		if e.Other.Role == unit.RoleTwin && e.Other.Slot == d.slot {
			d.tryShot(ctx, e)
		}
	}
}

func (d *无下限术士) remember(s unit.Sense) {
	d.selfX, d.selfY = s.Self.X, s.Self.Y
	d.hasEnemy = false
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role != unit.RoleFighter || o.Slot == s.Self.Slot {
			continue
		}
		d.enemyX, d.enemyY = o.X, o.Y
		d.hasEnemy = true
		break
	}
}

func (d *无下限术士) split(ctx unit.Context, s unit.Sense) {
	sp := math.Hypot(s.Self.VX, s.Self.VY)
	vx, vy := s.Self.VX, s.Self.VY
	if sp < 1e-6 {
		sp = twinSpeed
		vx, vy = twinSpeed, 0
	}
	px, py := vx/sp, vy/sp
	ctx.Out <- unit.Spawn{
		Kind:    twinKind,
		X:       s.Self.X - px*twinGap,
		Y:       s.Self.Y - py*twinGap,
		VX:      -vx,
		VY:      -vy,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	}
	ctx.Out <- unit.FX{
		Name: "split", Kind: KindTwin,
		X: s.Self.X, Y: s.Self.Y, VX: vx, VY: vy, Slot: s.Self.Slot,
	}
	ctx.Out <- unit.FX{
		Name: "split", Kind: twinKind,
		X: s.Self.X - px*twinGap, Y: s.Self.Y - py*twinGap, VX: -vx, VY: -vy, Slot: s.Self.Slot,
	}
}

func (d *无下限术士) tryShot(ctx unit.Context, e unit.Collision) {
	if !d.hasEnemy || e.Time < d.shotReadyAt {
		return
	}
	mx := (d.selfX + e.Other.X) / 2
	my := (d.selfY + e.Other.Y) / 2
	dx := d.enemyX - mx
	dy := d.enemyY - my
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	ux, uy := dx/n, dy/n
	ctx.Out <- unit.Spawn{
		Kind:    purpleKind,
		X:       mx,
		Y:       my,
		VX:      ux * purpleSpeed,
		VY:      uy * purpleSpeed,
		OwnerID: ctx.ID,
		Slot:    d.slot,
	}
	ctx.Out <- unit.FX{
		Name:   "void-shot",
		UnitID: ctx.ID,
		Kind:   purpleKind,
		X:      mx,
		Y:      my,
		VX:     ux * purpleSpeed,
		VY:     uy * purpleSpeed,
		Slot:   d.slot,
	}
	d.shotReadyAt = e.Time + purpleCD
}

type 双生 struct {
	owner uint64
	slot  int
	arc   unit.AttachState
}

func (c *双生) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		if unit.RearmAttach(e, ctx.ID, KindTwinBlueArc, twinHitCD, &c.arc) {
			unit.SpawnAttach(ctx, e, KindTwinBlueArc)
		}
		twinField(ctx, e, twinPush)
	}
}

type 术士弧 struct {
	slot int
	dmg  float64
}

func (a *术士弧) Handle(ctx unit.Context, ev unit.Event) {
	e, ok := ev.(unit.Collision)
	if !ok || !unit.EnemyFighter(e, a.slot) {
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
		if e.Other.Slot == b.slot {
			return
		}
		if e.Other.Role != unit.RoleFighter {
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
		if o.Slot == s.Self.Slot || o.Role == unit.RoleProjectile {
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
