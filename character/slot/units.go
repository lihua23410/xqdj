package 老虎机

import (
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

type slotBullet struct {
	owner    uint64
	slot     int
	kind     string
	rng      *rand.Rand
	lastNear []unit.Snapshot
}

func (b *slotBullet) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		b.lastNear = append(b.lastNear[:0], e.Nearby...)
		if b.rng == nil {
			b.rng = unit.NewRNG(ctx.ID)
		}
	case unit.Collision:
		if e.Other.ID == b.owner {
			return
		}
		if !unit.Hittable(e.Other, b.slot) {
			if e.Other.Role == unit.RoleProjectile {
				ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			}
			return
		}
		if b.kind == KindSlotBombShot {
			b.explode(ctx, e.Other.X, e.Other.Y, &e.Other)
			return
		}
		amount := b.damage()
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: amount}
		if b.kind == KindSlotLuckyShot {
			ctx.Out <- unit.AddFSComponent{
				UnitID: e.Other.ID, Zone: unit.FSZoneM,
				Token: luckySlowToken + ctx.ID, Value: luckySlow, ExpiresAt: 0,
			}
			ctx.Out <- unit.Stun{UnitID: e.Other.ID, Hold: true, Until: e.Time + luckyStun}
		}
		ctx.Out <- unit.FX{
			Name: "slot-bullet-hit", Kind: KindSlot, UnitID: b.owner, Slot: b.slot,
			X: e.Other.X, Y: e.Other.Y, Amount: amount,
		}
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	case unit.WallHit:
		if b.kind == KindSlotBombShot {
			b.explode(ctx, e.X, e.Y, nil)
			return
		}
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

func (b *slotBullet) damage() float64 {
	switch b.kind {
	case KindSlotCherryShot:
		return cherryDamage
	case KindSlotWatermelonShot:
		return watermelonDamage
	case KindSlotCrownShot:
		return crownDamage
	case KindSlotCoinShot:
		return coinDamage
	case KindSlotBombShot:
		return bombDamage
	case KindSlotJackpotShot:
		return jackpotDamage
	case KindSlotLuckyShot:
		if b.rng == nil {
			b.rng = unit.NewRNG(1)
		}
		return luckyMinDamage + float64(secureIntN(int(luckyMaxDamage-luckyMinDamage)+1, b.rng))
	}
	return 0
}

func (b *slotBullet) explode(ctx unit.Context, x, y float64, extra *unit.Snapshot) {
	ctx.Out <- unit.FX{
		Name: "slot-bomb-bullet", Kind: KindSlot, UnitID: b.owner, Slot: b.slot,
		X: x, Y: y, Amount: bombRadius,
	}
	seen := make(map[uint64]bool, len(b.lastNear)+1)
	if extra != nil && unit.Hittable(*extra, b.slot) {
		seen[extra.ID] = true
		ctx.Out <- unit.Damage{From: ctx.ID, To: extra.ID, Amount: bombDamage}
	}
	for i := range b.lastNear {
		o := &b.lastNear[i]
		if seen[o.ID] || !unit.Hittable(*o, b.slot) {
			continue
		}
		if math.Hypot(o.X-x, o.Y-y) > bombRadius+o.Radius {
			continue
		}
		seen[o.ID] = true
		ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: bombDamage}
	}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

type cherryBall struct {
	owner uint64
	slot  int
	at    float64
}

func (c *cherryBall) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if c.at < 0 {
		c.at = s.Time
	}
	if s.Time+1e-9 < c.at {
		return
	}
	target := unit.Seek(s)
	if target == nil {
		return
	}
	c.at = s.Time + cherryBallCD
	dx, dy := target.X-s.Self.X, target.Y-s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	ux, uy := dx/n, dy/n
	d := cherryBallRadius + 6 + 4
	ctx.Out <- unit.Spawn{
		Kind: KindSlotCherryShot,
		X:    s.Self.X + ux*d, Y: s.Self.Y + uy*d,
		VX: ux * 520, VY: uy * 520, OwnerID: c.owner, Slot: c.slot,
	}
	ctx.Out <- unit.FX{
		Name: "slot-cherry-fire", Kind: KindSlot, UnitID: ctx.ID, Slot: c.slot,
		X: s.Self.X, Y: s.Self.Y, VX: ux, VY: uy,
	}
}
