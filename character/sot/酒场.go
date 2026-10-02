package 酒翁

import (
	"math"

	"xqdj/internal/unit"
)

const stainRadius = 40.0

type 酒场 struct {
	owner  uint64
	slot   int
	inside map[uint64]struct{}
}

func (f *酒场) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
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
		ctx.Out <- unit.StackMark{UnitID: o.ID, Kind: drunkKind, Delta: 1, Icon: drunkIcon}
	}
	f.inside = now
}

func (f *酒场) affects(o unit.Snapshot) bool {
	if o.ID == f.owner {
		return true
	}
	return unit.Hittable(o, f.slot)
}
