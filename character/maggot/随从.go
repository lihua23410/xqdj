package 蛆

import "xqdj/internal/unit"

type 蛆宝宝 struct {
	owner uint64
	slot  int
}

func (b *蛆宝宝) Handle(ctx unit.Context, ev unit.Event) {
	unit.AcceptHit(ctx, ev)
}

type 苍蝇 struct {
	owner uint64
	slot  int
}

func (f *苍蝇) Handle(ctx unit.Context, ev unit.Event) {
	unit.AcceptHit(ctx, ev)
}

type 苍蝇卵 struct {
	owner  uint64
	slot   int
	booted bool
}

func (e *苍蝇卵) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	_, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !e.booted {
		e.booted = true
		ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
}
