package 优昙华院

import (
	"math"
	"xqdj/internal/unit"
)

type 心弹 struct {
	owner  uint64
	slot   int
	ux, uy float64
	t0     float64
	x, y   float64
	booted bool
	passed bool
	dead   bool
	wallNX float64
	wallNY float64
}

func (b *心弹) Handle(ctx unit.Context, ev unit.Event) {
	if b.dead {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		b.slot = e.Self.Slot
		if !b.booted {
			b.t0 = e.Time
			n := math.Hypot(e.Self.VX, e.Self.VY)
			if n > 1e-6 {
				b.ux, b.uy = e.Self.VX/n, e.Self.VY/n
			} else {
				b.ux, b.uy = 1, 0
			}
			b.booted = true
		}
		if !b.passed {
			b.passed = true
			ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
		}
		sp := mindSpeed(e.Time - b.t0)
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: b.ux * sp, VY: b.uy * sp}
		b.x, b.y = e.Self.X, e.Self.Y
		clearShots(ctx, e, b.owner)
	case unit.Collision:
		b.onHit(ctx, e.Other)
	case unit.WallHit:
		b.wallNX, b.wallNY = e.NX, e.NY
		b.split(ctx)
		b.die(ctx)
	}
}

func (b *心弹) onHit(ctx unit.Context, other unit.Snapshot) {
	if other.ID == b.owner {
		return
	}
	if other.Role == unit.RoleProjectile {
		ctx.Out <- unit.Despawn{UnitID: other.ID}
		return
	}
	if other.Role != unit.RoleFighter {
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: other.ID, Amount: scaled(b.owner, mindDamage)}
	b.die(ctx)
}

func (b *心弹) die(ctx unit.Context) {
	if b.dead {
		return
	}
	b.dead = true
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

func (b *心弹) split(ctx unit.Context) {
	x, y := b.x, b.y
	nx, ny := b.wallNX, b.wallNY
	if n := math.Hypot(nx, ny); n > 1e-6 {
		x -= nx / n * shardRadius * 0.35
		y -= ny / n * shardRadius * 0.35
	}
	for i := 0; i < shardCount; i++ {
		ang := float64(i) * 2 * math.Pi / shardCount
		ux, uy := math.Cos(ang), math.Sin(ang)
		ctx.Out <- unit.Spawn{
			Kind:    KindMindShard,
			X:       x + ux*(shardRadius+2),
			Y:       y + uy*(shardRadius+2),
			VX:      ux * shardSpeed,
			VY:      uy * shardSpeed,
			OwnerID: b.owner,
			Slot:    b.slot,
		}
	}
}

type 碎弹 struct {
	owner  uint64
	ux, uy float64
	t0     float64
	booted bool
	passed bool
	dead   bool
}

func (b *碎弹) Handle(ctx unit.Context, ev unit.Event) {
	if b.dead {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		if !b.booted {
			b.t0 = e.Time
			n := math.Hypot(e.Self.VX, e.Self.VY)
			if n > 1e-6 {
				b.ux, b.uy = e.Self.VX/n, e.Self.VY/n
			} else {
				b.ux, b.uy = 1, 0
			}
			b.booted = true
		}
		if !b.passed {
			b.passed = true
			ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
		}
		age := e.Time - b.t0
		if shardAlpha(age) <= 0 {
			b.dead = true
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			return
		}
		sp := shardSpeedAt(age)
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: b.ux * sp, VY: b.uy * sp}
		ctx.Out <- unit.FX{
			Name: "shard-fade", Kind: ctx.Kind, UnitID: ctx.ID,
			Amount: shardAlpha(age), Slot: e.Self.Slot,
		}
		clearShots(ctx, e, b.owner)
	case unit.Collision:
		if e.Other.ID == b.owner {
			return
		}
		if e.Other.Role == unit.RoleProjectile {
			ctx.Out <- unit.Despawn{UnitID: e.Other.ID}
			return
		}
		if e.Other.Role != unit.RoleFighter {
			return
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: scaled(b.owner, shardDamage)}
		b.dead = true
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	case unit.WallHit:
		b.dead = true
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

type 幻弹 struct{ owner uint64 }

func (b *幻弹) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Collision:
		if hitFighter(ctx, e.Other, b.owner, aspectDamage) {
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		}
	case unit.WallHit:
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

type 花冠 struct {
	owner     uint64
	ox, oy    float64
	originSet bool
	dead      bool
}

func (b *花冠) Handle(ctx unit.Context, ev unit.Event) {
	if b.dead {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		if !b.originSet {
			b.ox, b.oy = e.Self.X, e.Self.Y
			b.originSet = true
		}
		r := crownRadius(math.Hypot(e.Self.X-b.ox, e.Self.Y-b.oy))
		for i := range e.Nearby {
			o := &e.Nearby[i]
			if o.Role != unit.RoleFighter || o.ID == b.owner {
				continue
			}
			if math.Hypot(o.X-e.Self.X, o.Y-e.Self.Y) > r+o.Radius {
				continue
			}
			ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: scaled(b.owner, crownDamage)}
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			b.dead = true
			return
		}
	case unit.Collision:
		if hitFighter(ctx, e.Other, b.owner, crownDamage) {
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			b.dead = true
		}
	case unit.WallHit:
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		b.dead = true
	}
}

type 分身 struct{}

func (分身) Handle(unit.Context, unit.Event) {}

type 瓦斯 struct {
	owner    uint64
	slot     int
	born     float64
	nextTick float64
	booted   bool
	dead     bool
	slowed   map[uint64]float64
}

func (g *瓦斯) Handle(ctx unit.Context, ev unit.Event) {
	if g.dead {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !g.booted {
		g.born = s.Time
		g.nextTick = s.Time
		g.booted = true
		g.slowed = map[uint64]float64{}
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	}
	if s.Time+1e-9 >= g.born+gasLife {
		g.release(ctx)
		g.dead = true
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	g.tick(ctx, s)
}

func (g *瓦斯) tick(ctx unit.Context, s unit.Sense) {
	dmgNow := s.Time+1e-9 >= g.nextTick
	seen := map[uint64]bool{}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role != unit.RoleFighter || o.Slot == g.slot || o.ID == g.owner {
			continue
		}
		if math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) > gasRadius {
			continue
		}
		seen[o.ID] = true
		g.slow(ctx, *o)
		if dmgNow {
			ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: scaled(g.owner, gasDamage)}
		}
	}
	if dmgNow {
		g.nextTick = s.Time + gasTick
	}
	for id, orig := range g.slowed {
		if seen[id] {
			continue
		}
		ctx.Out <- unit.SetCruise{UnitID: id, Speed: orig}
		delete(g.slowed, id)
	}
}

func (g *瓦斯) slow(ctx unit.Context, o unit.Snapshot) {
	sp := math.Hypot(o.VX, o.VY)
	orig, ok := g.slowed[o.ID]
	if !ok {
		orig = sp
		if orig < 80 {
			orig = 80
		}
		g.slowed[o.ID] = orig
	}
	cap := orig * gasSlow
	if cap < 24 {
		cap = 24
	}
	ctx.Out <- unit.SetCruise{UnitID: o.ID, Speed: cap}
	if sp > cap+1 {
		s := cap / sp
		ctx.Out <- unit.SetVelocity{UnitID: o.ID, VX: o.VX * s, VY: o.VY * s}
	}
}

func (g *瓦斯) release(ctx unit.Context) {
	for id, orig := range g.slowed {
		ctx.Out <- unit.SetCruise{UnitID: id, Speed: orig}
		delete(g.slowed, id)
	}
}

type 激光 struct {
	born   float64
	booted bool
}

func (l *激光) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !l.booted {
		l.born = s.Time
		l.booted = true
	}
	if s.Time >= l.born+laserLife {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

func clearShots(ctx unit.Context, s unit.Sense, owner uint64) {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.ID == ctx.ID || o.OwnerID == owner {
			continue
		}
		if o.Role != unit.RoleProjectile {
			continue
		}
		if math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) > s.Self.Radius+o.Radius+4 {
			continue
		}
		ctx.Out <- unit.Despawn{UnitID: o.ID}
	}
}

func hitFighter(ctx unit.Context, other unit.Snapshot, owner uint64, base float64) bool {
	if other.ID == owner {
		return false
	}
	if other.Role != unit.RoleFighter {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return true
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: other.ID, Amount: scaled(owner, base)}
	return true
}
