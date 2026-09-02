package character

import (
	"math"
	"xqdj/internal/unit"
)

const KindRanged = "原型机_远程"

const (
	rangedRadius       = 18.0
	rangedSpeed        = 140.0
	rangedVision       = 9999.0
	rangedHP           = 100.0
	rangedFireInterval = 2

	kindBullet    = "子弹"
	bulletRadius  = 6.0
	bulletSpeed   = 160.0
	bulletHP      = 1.0
	bulletDamage  = 8.0
	bulletBounces = 3
)

func init() {
	unit.Register(unit.Spec{
		Kind:    KindRanged,
		Role:    unit.RoleFighter,
		Radius:  rangedRadius,
		MaxHP:   rangedHP,
		Speed:   rangedSpeed,
		Vision:  rangedVision,
		Fighter: true,
	}, func(unit.SpawnInfo) unit.Actor {
		return &原型机_远程{}
	})
	unit.Register(unit.Spec{
		Kind:    kindBullet,
		Role:    unit.RoleProjectile,
		Radius:  bulletRadius,
		MaxHP:   bulletHP,
		Speed:   bulletSpeed,
		Vision:  0,
		Fighter: false,
	}, func(info unit.SpawnInfo) unit.Actor {
		return &远程子弹{owner: info.OwnerID}
	})
}

type 原型机_远程 struct {
	fireReadyAt float64
}

func (r *原型机_远程) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		r.shoot(ctx, e)
	}
}

func (r *原型机_远程) shoot(ctx unit.Context, s unit.Sense) {
	if s.Time < r.fireReadyAt {
		return
	}
	var target *unit.Snapshot
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role != unit.RoleFighter {
			continue
		}
		target = o
		break
	}
	if target == nil {
		return
	}
	dx := target.X - s.Self.X
	dy := target.Y - s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	ux, uy := dx/n, dy/n
	gap := s.Self.Radius + bulletRadius + 1.5
	ctx.Out <- unit.Spawn{
		Kind:    kindBullet,
		X:       s.Self.X + ux*gap,
		Y:       s.Self.Y + uy*gap,
		VX:      ux * bulletSpeed,
		VY:      uy * bulletSpeed,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	}
	r.fireReadyAt = s.Time + rangedFireInterval
	sp := math.Hypot(s.Self.VX, s.Self.VY)
	if sp < 1e-6 {
		sp = rangedSpeed
	}
	ctx.Out <- unit.SetVelocity{
		UnitID: ctx.ID,
		VX:     -ux * sp,
		VY:     -uy * sp,
	}
	ctx.Out <- unit.FX{
		Name:   "shot",
		UnitID: ctx.ID,
		X:      s.Self.X,
		Y:      s.Self.Y,
		VX:     ux * bulletSpeed,
		VY:     uy * bulletSpeed,
		Slot:   s.Self.Slot,
	}
}

type 远程子弹 struct {
	owner   uint64
	bounces int
}

func (b *远程子弹) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Collision:
		b.onHit(ctx, e.Other)
	case unit.WallHit:
		b.bounces++
		if b.bounces >= bulletBounces {
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		}
	}
}

func (b *远程子弹) onHit(ctx unit.Context, other unit.Snapshot) {
	if other.ID == b.owner {
		return
	}
	if other.Role != unit.RoleFighter {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: other.ID, Amount: bulletDamage}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}
