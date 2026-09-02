package character

import (
	"math"
	"xqdj/internal/unit"
)

const KindMelee = "原型机_近战"

const (
	meleeRadius    = 18.0
	meleeSpeed     = 200.0
	meleeVision    = 185.0
	meleeHP        = 100.0
	meleeDamage    = 7.0
	meleeSeekBoost = 155.0
	meleeSeekTurn  = 15 * math.Pi / 180
	meleeHitCD     = 0.1
)

func init() {
	unit.Register(unit.Spec{
		Kind:    KindMelee,
		Role:    unit.RoleFighter,
		Radius:  meleeRadius,
		MaxHP:   meleeHP,
		Speed:   meleeSpeed,
		Vision:  meleeVision,
		Fighter: true,
	}, func(unit.SpawnInfo) unit.Actor {
		return &原型机_近战{}
	})
}

type 原型机_近战 struct {
	hitReadyAt  float64
	enemyInside bool
}

func (m *原型机_近战) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		m.seek(ctx, e)
	case unit.Collision:
		m.hit(ctx, e)
	}
}

func (m *原型机_近战) hit(ctx unit.Context, e unit.Collision) {
	if e.Other.Role != unit.RoleFighter {
		return
	}
	if e.Time < m.hitReadyAt {
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: meleeDamage}
	m.hitReadyAt = e.Time + meleeHitCD
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
