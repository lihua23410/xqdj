package 外科手术

import (
	"math"
	"xqdj/internal/unit"
)

type scalpel struct {
	owner           uint64
	slot            int
	data            knifePlan
	hits            map[uint64]bool
	done            bool
	copied, started bool
	born            float64
}

func (a *scalpel) Handle(ctx unit.Context, ev unit.Event) {
	if a.done {
		return
	}
	if a.copied {
		a.handleCopy(ctx, ev)
		return
	}
	var cmds commands
	o := unit.StateFor[clinic](ctx.State).operation(a.owner)
	switch e := ev.(type) {
	case unit.Collision:
		if unit.EnemyTarget(e, a.slot) && !a.hits[e.Other.ID] {
			a.hits[e.Other.ID] = true
			cmds.add(unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: 8})
		}
	case unit.Sense:
		p := point{e.Self.X, e.Self.Y}
		d := norm(a.data.End.sub(a.data.Start))
		progress := (p.X-a.data.A.X)*d.X + (p.Y-a.data.A.Y)*d.Y
		o.Lock()
		if w := o.wound(a.data.Wound); w != nil {
			w.Drawn = math.Max(w.Drawn, math.Max(0, math.Min(1, progress/a.data.B.sub(a.data.A).length())))
		}
		remaining := a.data.End.sub(p)
		if remaining.length() < 1 || remaining.X*d.X+remaining.Y*d.Y <= 0 {
			a.done = true
			o.knifeOn = false
			if w := o.wound(a.data.Wound); w != nil {
				w.Drawn = 1
			}
			cmds.add(unit.Despawn{UnitID: ctx.ID})
		} else {
			v := norm(remaining).mul(math.Min(knifeSpeed, remaining.length()/stepTime))
			cmds.add(unit.SetVelocity{UnitID: ctx.ID, VX: v.X, VY: v.Y})
		}
		o.Unlock()
	case unit.WallHit:
		if e.Kind.Edge() {
			a.done = true
			o.Lock()
			o.knifeOn = false
			if w := o.wound(a.data.Wound); w != nil {
				w.Drawn = 1
			}
			o.Unlock()
			cmds.add(unit.Despawn{UnitID: ctx.ID})
		}
	}
	cmds.send(ctx)
}

// Kind-only replays have no incision owner or return point. Keep their launch
// velocity and piercing damage, and leave on the field edge or after 2.5s.
func (a *scalpel) handleCopy(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		if !a.started {
			a.born, a.started = e.Time, true
		}
		if e.Time >= a.born+2.5 {
			a.done = true
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		}
	case unit.Collision:
		if unit.EnemyTarget(e, a.slot) && !a.hits[e.Other.ID] {
			a.hits[e.Other.ID] = true
			ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: 8}
		}
	case unit.WallHit:
		if e.Kind.Edge() {
			a.done = true
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		}
	}
}

type needle struct {
	owner, wound uint64
	slot         int
}

func (a *needle) Handle(ctx unit.Context, ev unit.Event) {
	e, ok := ev.(unit.Collision)
	if !ok || !unit.EnemyTarget(e, a.slot) {
		return
	}
	o := unit.StateFor[clinic](ctx.State).operation(a.owner)
	var cmds commands
	o.Lock()
	w := o.wound(a.wound)
	if o.stage == stitching && o.current == a.wound && o.immune(e.Time) && w != nil {
		h := w.hits[e.Other.ID]
		if h.Count < 3 && (h.Count == 0 || e.Time+1e-9 >= h.At+.2) {
			w.hits[e.Other.ID] = punctureHit{At: e.Time, Count: h.Count + 1}
			cmds.add(unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: 2})
		}
	}
	o.Unlock()
	cmds.send(ctx)
}

type bloodBullet struct {
	born          float64
	started, done bool
}

func (a *bloodBullet) Handle(ctx unit.Context, ev unit.Event) {
	if a.done {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		if !a.started {
			a.born, a.started = e.Time, true
		}
		if e.Time >= a.born+2.5 {
			a.done = true
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		}
	case unit.Collision:
		if e.Other.Nonsolid || !(e.Other.Role == unit.RoleFighter || e.Other.Mortal) {
			return
		}
		a.done = true
		// Keep this source tombstone after Despawn: a recipient can process the
		// offer later in the same frame, including another surgeon's blood.
		unit.StateFor[clinic](ctx.State).markBlood(ctx.ID, e.Time)
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: 2}
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	case unit.WallHit:
		a.done = true
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}
