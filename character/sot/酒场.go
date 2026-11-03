package 酒翁

import (
	"math"

	"xqdj/internal/unit"
)

const stainRadius = 40.0
const stainLife = 40.0 // 酒会挥发：酒场最多活这么久，不然一局下来堆近百个，每拍都要感知

type 酒场 struct {
	owner  uint64
	slot   int
	inside map[uint64]struct{}
	born   float64
	booted bool
}

func (f *酒场) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !f.booted {
		f.booted = true
		f.born = s.Time
	}
	if s.Time-f.born >= stainLife {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	if f.inside == nil {
		f.inside = map[uint64]struct{}{}
	}
	now := map[uint64]struct{}{}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !f.affects(*o) {
			continue
		}
		if math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) > o.Radius+s.Self.Radius {
			continue
		}
		now[o.ID] = struct{}{}
		if _, was := f.inside[o.ID]; was {
			continue
		}
		mark := unit.StackMark{UnitID: o.ID, Kind: drunkKind, Delta: 1, Icon: drunkIcon}
		if o.ID != f.owner {
			mark.Lifetime = drunkLife
		}
		ctx.Out <- mark
	}
	f.inside = now
}

func (f *酒场) affects(o unit.Snapshot) bool {
	if o.ID == f.owner {
		return true
	}
	return unit.Hittable(o, f.slot)
}
