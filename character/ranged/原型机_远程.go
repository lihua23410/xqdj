package 原型机_远程

import (
	"embed"
	"math"
	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const KindRanged = "原型机_远程"

const (
	rangedRadius       = 18.0
	rangedSpeed        = 148.0
	rangedVision       = 9999.0
	rangedHP           = 100.0
	rangedFireInterval = 1.6

	kindBullet    = "子弹"
	bulletRadius  = 6.0
	bulletMuzzle  = 80.0
	bulletAccel   = 200.0
	bulletHP      = 1.0
	bulletDamage  = 9.0
	bulletBounces = 3
)

func init() {
	p := unit.NewPack(KindRanged, assets)
	p.Register(unit.Spec{
		Kind:    KindRanged,
		Role:    unit.RoleFighter,
		Radius:  rangedRadius,
		MaxHP:   rangedHP,
		Speed:   rangedSpeed,
		Vision:  rangedVision,
		Fighter: true,
		Look:    unit.Look{Color: "#f0a050"},
	}, func(unit.SpawnInfo) unit.Actor {
		return &原型机_远程{}
	})
	p.Register(unit.Spec{
		Kind:    kindBullet,
		Role:    unit.RoleProjectile,
		Radius:  bulletRadius,
		MaxHP:   bulletHP,
		Speed:   bulletMuzzle,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: "#f0a050", FX: []string{"rocket"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &远程子弹{owner: info.OwnerID, slot: info.Slot}
	})
}

type 原型机_远程 struct {
	fireReadyAt float64
}

func (r *原型机_远程) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		r.shoot(ctx, e)
	}
}

func (r *原型机_远程) shoot(ctx unit.Context, s unit.Sense) {
	if s.Time < r.fireReadyAt {
		return
	}
	target := unit.Seek(s)
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
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind:    kindBullet,
		X:       s.Self.X + ux*gap,
		Y:       s.Self.Y + uy*gap,
		VX:      ux * bulletMuzzle,
		VY:      uy * bulletMuzzle,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	})
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
		Kind:   ctx.Kind,
		X:      s.Self.X,
		Y:      s.Self.Y,
		VX:     ux * bulletMuzzle,
		VY:     uy * bulletMuzzle,
		Slot:   s.Self.Slot,
	}
}

type 远程子弹 struct {
	owner    uint64
	slot     int
	bounces  int
	dx, dy   float64
	legAt    float64
	fromRest bool
	aimed    bool
	clock    bool
	dead     bool
}

func (b *远程子弹) Handle(ctx unit.Context, ev unit.Event) {
	if b.dead {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		b.fly(ctx, e)
	case unit.Collision:
		b.onHit(ctx, e.Other)
	case unit.WallHit:
		b.onWall(ctx, e)
	}
}

func (b *远程子弹) fly(ctx unit.Context, s unit.Sense) {
	if !b.aimed {
		b.capture(s.Self.VX, s.Self.VY)
	}
	if !b.aimed {
		return
	}
	if !b.clock {
		b.clock = true
		if !b.fromRest {
			b.legAt = s.Time
		}
	}
	t := s.Time - b.legAt
	if t < 0 {
		t = 0
	}
	v0 := bulletMuzzle
	if b.fromRest {
		v0 = 0
	}
	speed := v0 + bulletAccel*t
	ctx.Out <- unit.SetVelocity{
		UnitID: ctx.ID,
		VX:     b.dx * speed,
		VY:     b.dy * speed,
	}
}

func (b *远程子弹) capture(vx, vy float64) {
	n := math.Hypot(vx, vy)
	if n < 1e-6 {
		return
	}
	b.dx, b.dy = vx/n, vy/n
	b.aimed = true
}

func (b *远程子弹) onWall(ctx unit.Context, e unit.WallHit) {
	b.bounces++
	if b.bounces >= bulletBounces {
		b.dead = true
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	if b.aimed {
		dot := b.dx*e.NX + b.dy*e.NY
		b.dx -= 2 * dot * e.NX
		b.dy -= 2 * dot * e.NY
		n := math.Hypot(b.dx, b.dy)
		if n < 1e-6 {
			b.aimed = false
		} else {
			b.dx /= n
			b.dy /= n
		}
	}
	b.fromRest = true
	b.legAt = e.Time
	b.clock = true
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID}
}

func (b *远程子弹) onHit(ctx unit.Context, other unit.Snapshot) {
	if other.ID == b.owner {
		return
	}
	b.dead = true
	if !unit.Hittable(other, b.slot) {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: other.ID, Amount: bulletDamage}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}
