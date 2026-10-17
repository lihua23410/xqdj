package pink

import (
	"math"

	"xqdj/internal/unit"
)

// crowd 只负责吃伤害。移动完全交给巡航：初速随机，撞墙后由引擎反弹。
// crowd 只负责吃伤害。移动完全交给巡航：初速随机，撞墙后由引擎反弹。
// 只有创建者 Pink 或其弹体/火焰能造成有效伤害，团队队友无法抢杀。
type crowd struct {
	owner uint64
	seen  map[uint64]unit.Snapshot
}

func (c *crowd) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		if c.canTakeDamage(e.From) {
			unit.ConfirmHit(ctx, e)
		} else {
			unit.BlockHit(ctx, e)
		}
	case unit.Sense:
		seen := make(map[uint64]unit.Snapshot, len(e.Nearby))
		for i := range e.Nearby {
			seen[e.Nearby[i].ID] = e.Nearby[i]
		}
		c.seen = seen
	}
}

func (c *crowd) canTakeDamage(from uint64) bool {
	if from == c.owner {
		return true
	}
	src, ok := c.seen[from]
	return ok && src.OwnerID == c.owner
}

// item 是掉落物，只有 pink 本人会因为 OwnerID 匹配而拾取。
type item struct {
	slot int
	kind string
}

func (i *item) Handle(unit.Context, unit.Event) {}

type bottleShot struct {
	owner uint64
	slot  int
}

func (b *bottleShot) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Collision:
		if e.Other.ID == b.owner {
			return
		}
		if !unit.Hittable(e.Other, b.slot) && e.Other.Role != unit.RoleProjectile {
			return
		}
		if unit.Hittable(e.Other, b.slot) {
			ctx.Out <- unit.Damage{From: b.owner, To: e.Other.ID, Amount: bottleImpactDamage}
			ctx.Out <- unit.Stun{UnitID: e.Other.ID, Hold: true, Until: e.Time + bottleStun}
		}
		b.ignite(ctx, e.Other.X, e.Other.Y)
	case unit.WallHit:
		b.ignite(ctx, e.X, e.Y)
	}
}

func (b *bottleShot) ignite(ctx unit.Context, x, y float64) {
	ctx.Out <- unit.Spawn{
		Kind: KindFire, X: x, Y: y, OwnerID: b.owner, Slot: b.slot,
	}
	ctx.Out <- unit.FX{
		Name: "fire", Kind: KindPink, UnitID: b.owner, Slot: b.slot,
		X: x, Y: y,
	}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

type fireArea struct {
	born float64
	next float64
}

func (f *fireArea) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if f.born == 0 {
		f.born = s.Time
		f.next = s.Time
	}
	if s.Time+1e-9 >= f.born+fireLife {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	if s.Time+1e-9 < f.next {
		return
	}
	f.next = s.Time + fireGap
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		if math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) <= fireRadius+o.Radius {
			ctx.Out <- unit.Damage{
				From: s.Self.OwnerID, To: o.ID, Amount: fireDamage,
				NoFreeze: true,
			}
			ctx.Out <- unit.AddFSComponent{
				UnitID: o.ID, Zone: unit.FSZoneM, Token: ctx.ID,
				Value: fireSlow, ExpiresAt: s.Time + fireGap + 0.3,
			}
		}
	}
}

type bullet struct {
	owner    uint64
	slot     int
	kind     string
	damage   float64
	life     float64
	born     float64
	lastNear []unit.Snapshot
}

func (b *bullet) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		if b.born == 0 {
			b.born = e.Time
		}
		b.lastNear = append(b.lastNear[:0], e.Nearby...)
		if b.life > 0 && e.Time-b.born >= b.life {
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		}
	case unit.Collision:
		if e.Other.ID == b.owner {
			return
		}
		if b.kind == KindRocketShot {
			if unit.Hittable(e.Other, b.slot) || e.Other.Role == unit.RoleProjectile {
				b.explode(ctx, e.Other.X, e.Other.Y, &e.Other)
			}
			return
		}
		if !unit.Hittable(e.Other, b.slot) {
			if e.Other.Role == unit.RoleProjectile {
				ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			}
			return
		}
		ctx.Out <- unit.Damage{
			From: ctx.ID, To: e.Other.ID, Amount: b.damage,
			NoFreeze: b.kind == KindShotgunShot,
		}
		ctx.Out <- unit.FX{
			Name: "bullet-hit", Kind: KindPink, UnitID: b.owner, Slot: b.slot,
			X: e.Other.X, Y: e.Other.Y,
		}
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	case unit.WallHit:
		if b.kind == KindRocketShot {
			b.explode(ctx, e.X, e.Y, nil)
			return
		}
		ctx.Out <- unit.FX{
			Name: "bullet-wall", Kind: KindPink, UnitID: b.owner, Slot: b.slot,
			X: e.X, Y: e.Y,
		}
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

func (b *bullet) explode(ctx unit.Context, x, y float64, extra *unit.Snapshot) {
	ctx.Out <- unit.FX{
		Name: "rocket-boom", Kind: KindPink, UnitID: b.owner, Slot: b.slot,
		X: x, Y: y, Amount: rocketR,
	}
	seen := make(map[uint64]bool, len(b.lastNear)+1)
	if extra != nil {
		if unit.Hittable(*extra, b.slot) {
			seen[extra.ID] = true
			ctx.Out <- unit.Damage{From: ctx.ID, To: extra.ID, Amount: rocketDmg}
		}
	}
	for i := range b.lastNear {
		o := &b.lastNear[i]
		if seen[o.ID] || !unit.Hittable(*o, b.slot) {
			continue
		}
		if math.Hypot(o.X-x, o.Y-y) > rocketR+o.Radius {
			continue
		}
		seen[o.ID] = true
		ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: rocketDmg}
	}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}
