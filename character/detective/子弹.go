package 探长

import "xqdj/internal/unit"

type 子弹 struct {
	owner  uint64
	slot   int
	damage float64
	dead   bool
}

func (b *子弹) Handle(ctx unit.Context, ev unit.Event) {
	if b.dead {
		return
	}
	switch e := ev.(type) {
	case unit.Collision:
		if e.Other.ID == b.owner || e.Other.Slot == b.slot {
			return
		}
		if unit.Hittable(e.Other, b.slot) {
			ctx.Out <- unit.Damage{
				From: ctx.ID, To: e.Other.ID, Amount: b.damage,
			}
		}
		b.gone(ctx)
	case unit.WallHit:
		b.gone(ctx)
	}
}

func (b *子弹) gone(ctx unit.Context) {
	if b.dead {
		return
	}
	b.dead = true
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}
