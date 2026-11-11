package 闪烁

import (
	"math"
	"xqdj/internal/unit"
)

const (
	kindBullet  = "闪烁子弹"
	kindSmart   = "闪烁智能弹"
	kindBoost   = "闪烁强化智能弹"
	bulletSpeed = 580.0
	turnRate    = 2.4 // 每秒弧度；不瞬转、不掉头重锁。
)

type bulletData struct {
	Target        uint64
	Smart, Boost  bool
	Damage, Until float64
}
type projectile struct {
	bulletData
	owner       uint64
	slot        int
	lastT       float64
	clock, dead bool
}

func registerBullets(p *unit.Pack) {
	for _, kind := range []string{kindBullet, kindSmart, kindBoost} {
		color := "#b7fff1"
		skin := "flicker-bullet"
		if kind != kindBullet {
			color = "#68ffe0"
			skin = "flicker-smart"
		}
		if kind == kindBoost {
			color = "#ffd482"
			skin = "flicker-boost"
		}
		p.Register(unit.Spec{Kind: kind, Role: unit.RoleProjectile, Radius: 3, MaxHP: 1, Speed: bulletSpeed, Vision: 400,
			Look: unit.Look{Color: color, FX: []string{skin}}}, func(info unit.SpawnInfo) unit.Actor {
			data, _ := info.Data.(bulletData)
			return &projectile{bulletData: data, owner: info.OwnerID, slot: info.Slot}
		})
	}
}
func (b *projectile) Handle(ctx unit.Context, ev unit.Event) {
	if b.dead {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		if e.Time >= b.Until {
			b.end(ctx)
			return
		}
		dt := 1.0 / 60
		if b.clock {
			dt = math.Max(0, math.Min(.05, e.Time-b.lastT))
		}
		b.lastT, b.clock = e.Time, true
		if !b.Smart || b.Target == 0 {
			return
		}
		target := unit.SeekIf(e, func(o unit.Snapshot) bool { return o.ID == b.Target })
		if target == nil {
			b.Target = 0
			return
		}
		current := math.Atan2(e.Self.VY, e.Self.VX)
		desired := math.Atan2(target.Y-e.Self.Y, target.X-e.Self.X)
		delta := math.Atan2(math.Sin(desired-current), math.Cos(desired-current))
		// 飞过目标后不兜圈，丢失锁定继续直飞。
		if math.Abs(delta) > math.Pi/2 {
			b.Target = 0
			return
		}
		angle := current + math.Max(-turnRate*dt, math.Min(turnRate*dt, delta))
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: math.Cos(angle) * bulletSpeed, VY: math.Sin(angle) * bulletSpeed}
	case unit.Collision:
		if e.Other.ID == b.owner {
			return
		}
		if unit.Hittable(e.Other, b.slot) {
			ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: b.Damage, NoFreeze: !b.Boost || e.Other.Role != unit.RoleFighter}
		}
		b.end(ctx)
	case unit.WallHit:
		b.end(ctx)
	}
}
func (b *projectile) end(ctx unit.Context) { b.dead = true; ctx.Out <- unit.Despawn{UnitID: ctx.ID} }
