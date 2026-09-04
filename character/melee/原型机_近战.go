package 原型机_近战

import (
	"embed"
	"math"
	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const KindMelee = "原型机_近战"
const KindMeleeArc = "原型机_近战弧"

const (
	meleeRadius    = 18.0
	meleeSpeed     = 195.0
	meleeVision    = 185.0
	meleeHP        = 100.0
	meleeDamage    = 7.0
	meleeSeekBoost = 140.0
	meleeSeekTurn  = 15 * math.Pi / 180
	meleeHitCD     = 0.1
	meleeArcInner  = meleeRadius
	meleeArcOuter  = meleeRadius + 2
	meleeArcMin    = 90.0
	meleeArcMax    = 180.0
	meleeColor     = "#3dd6c6"
)

func init() {
	p := unit.NewPack(KindMelee, assets)
	p.Register(unit.Spec{
		Kind:    KindMelee,
		Role:    unit.RoleFighter,
		Radius:  meleeRadius,
		MaxHP:   meleeHP,
		Speed:   meleeSpeed,
		Vision:  meleeVision,
		Fighter: true,
		Look:    unit.Look{Color: meleeColor, Ghost: 280, VisionRing: true},
	}, func(unit.SpawnInfo) unit.Actor {
		return &原型机_近战{}
	})
	p.Register(unit.Spec{
		Kind:     KindMeleeArc,
		Role:     unit.RoleProjectile,
		Radius:   meleeArcOuter,
		MaxHP:    1,
		Speed:    meleeSpeed,
		Vision:   0,
		Fighter:  false,
		Attach:   true,
		ArcSpan:  unit.Deg(meleeArcMin),
		ArcInner: meleeArcInner,
		Look:     unit.Look{Color: meleeColor, Overlay: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &近战弧{slot: info.Slot}
	})
}

type 原型机_近战 struct {
	arc         unit.AttachState
	enemyInside bool
}

func (m *原型机_近战) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		amt := e.Amount * (1 - meleeResist(e.Speed))
		ctx.Out <- unit.ConfirmDamage{Token: e.Token, UnitID: ctx.ID, Amount: amt}
	case unit.Sense:
		if unit.RearmAttach(e, ctx.ID, KindMeleeArc, meleeHitCD, &m.arc) {
			unit.SpawnAttach(ctx, e, KindMeleeArc)
		}
		m.seek(ctx, e)
	}
}

func meleeResist(speed float64) float64 {
	extra := speed - meleeSpeed
	if extra < 0 {
		extra = 0
	}
	r := extra / 150 * 0.05
	if r > 0.25 {
		r = 0.25
	}
	return r
}

func meleeArcSpan(speed float64) float64 {
	r := meleeResist(speed)
	return unit.Deg(meleeArcMin + (meleeArcMax-meleeArcMin)*(r/0.25))
}

type 近战弧 struct {
	slot int
}

func (a *近战弧) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		ctx.Out <- unit.SetArcSpan{UnitID: ctx.ID, Span: meleeArcSpan(math.Hypot(e.Self.VX, e.Self.VY))}
	case unit.Collision:
		if !unit.EnemyFighter(e, a.slot) {
			return
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: meleeDamage}
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

func (m *原型机_近战) seek(ctx unit.Context, s unit.Sense) {
	var target *unit.Snapshot
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role != unit.RoleFighter {
			continue
		}
		target = o
		break
	}
	inside := target != nil
	entered := inside && !m.enemyInside
	m.enemyInside = inside
	if !entered {
		return
	}
	sp := math.Hypot(s.Self.VX, s.Self.VY)
	if sp < 1e-6 {
		return
	}
	dx := target.X - s.Self.X
	dy := target.Y - s.Self.Y
	tn := math.Hypot(dx, dy)
	ux, uy := s.Self.VX/sp, s.Self.VY/sp
	if tn > 1e-6 {
		ux, uy = turnToward(ux, uy, dx/tn, dy/tn, meleeSeekTurn)
	}
	ns := sp + meleeSeekBoost
	vx, vy := ux*ns, uy*ns
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: vx, VY: vy}
	ctx.Out <- unit.FX{
		Name:   "dash",
		UnitID: ctx.ID,
		Kind:   ctx.Kind,
		X:      s.Self.X,
		Y:      s.Self.Y,
		VX:     vx,
		VY:     vy,
		Slot:   s.Self.Slot,
	}
}

func turnToward(ux, uy, tx, ty, maxRad float64) (float64, float64) {
	ang := math.Atan2(ux*ty-uy*tx, ux*tx+uy*ty)
	if ang > maxRad {
		ang = maxRad
	} else if ang < -maxRad {
		ang = -maxRad
	}
	c, s := math.Cos(ang), math.Sin(ang)
	return ux*c - uy*s, ux*s + uy*c
}
