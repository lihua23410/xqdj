package 纵火者

import (
	"math"

	"xqdj/internal/unit"
)

// bottleAim 是扔出那一瞬记下的圆心，以及瓶子出生的位置。
type bottleAim struct {
	X, Y   float64
	SX, SY float64
}

type bottle struct {
	owner  uint64
	slot   int
	fire   bool
	tx, ty float64
	x, y   float64
	have   bool
	dead   bool
}

func newBottle(fire bool) func(unit.SpawnInfo) unit.Actor {
	return func(info unit.SpawnInfo) unit.Actor {
		b := &bottle{owner: info.OwnerID, slot: info.Slot, fire: fire}
		if d, ok := info.Data.(bottleAim); ok {
			b.tx, b.ty = d.X, d.Y
			b.x, b.y = d.SX, d.SY
			b.have = true
		}
		return b
	}
}

func (b *bottle) Handle(ctx unit.Context, ev unit.Event) {
	if b.dead {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		b.onSense(ctx, e)
	case unit.Collision:
		if unit.Hittable(e.Other, b.slot) {
			b.shatter(ctx, b.x, b.y)
		}
	case unit.WallHit:
		b.shatter(ctx, e.X, e.Y)
	}
}

func (b *bottle) onSense(ctx unit.Context, s unit.Sense) {
	x, y := s.Self.X, s.Self.Y
	if !b.have {
		b.have = true
		b.x, b.y = x, y
		if math.Hypot(x-b.tx, y-b.ty) <= bottleRadius {
			b.shatter(ctx, b.tx, b.ty)
		}
		return
	}
	if reached(b.x, b.y, x, y, b.tx, b.ty) {
		b.shatter(ctx, b.tx, b.ty)
		return
	}
	b.x, b.y = x, y
}

func reached(ax, ay, bx, by, tx, ty float64) bool {
	if math.Hypot(bx-tx, by-ty) <= bottleRadius {
		return true
	}
	abx, aby := bx-ax, by-ay
	ab2 := abx*abx + aby*aby
	if ab2 < 1e-8 {
		return false
	}
	t := ((tx-ax)*abx + (ty-ay)*aby) / ab2
	if t < 0 || t > 1 {
		return false
	}
	cx := ax + abx*t
	cy := ay + aby*t
	return math.Hypot(cx-tx, cy-ty) <= bottleRadius
}

func (b *bottle) shatter(ctx unit.Context, x, y float64) {
	if b.dead {
		return
	}
	b.dead = true
	kind := KindArsonOilMark
	sound := "arson-oil-break"
	if b.fire {
		kind = KindArsonFireMark
		sound = "arson-fire-break"
	}
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: kind, OwnerID: b.owner, Slot: b.slot, X: x, Y: y,
	})
	ctx.Out <- unit.FX{Name: sound, UnitID: ctx.ID, Kind: ctx.Kind, X: x, Y: y, Slot: b.slot}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}
