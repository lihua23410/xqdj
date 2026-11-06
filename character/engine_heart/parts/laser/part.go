// Package laser owns a telegraphed, direction-locked sustained pressure beam.
package laser

import (
	"math"
	"xqdj/character/engine_heart/parts/api"
	"xqdj/internal/unit"
)

func Definition() api.Definition {
	return api.Definition{ID: 11, Key: "laser", Name: "蓄压光棱镜", Install: func(*api.Build) {}, Weapon: func() unit.Actor { return &weapon{} }}
}

type weapon struct {
	charge              int
	active              bool
	fireAt, until, next float64
	dx, dy              float64
}

func (w *weapon) Handle(ctx unit.Context, ev unit.Event) {
	switch p := ev.(type) {
	case api.Pulse:
		if p.Echo {
			return
		}
		w.charge = min(2, w.charge+1)
		if w.charge < 2 || w.active {
			return
		}
		s := p.Sense
		var target *unit.Snapshot
		best := math.MaxFloat64
		for i := range s.Nearby {
			o := &s.Nearby[i]
			d := math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y)
			if unit.Hittable(*o, s.Self.Slot) && o.HP > 0 && d >= 90 && d <= 360+o.Radius && d < best && api.Clear(s, s.Self.X, s.Self.Y, *o) {
				target = o
				best = d
			}
		}
		if target == nil {
			return
		} // Store pressure until a long-range firing opportunity exists.
		w.charge = 0
		w.active = true
		w.fireAt = s.Time + .28
		w.until = w.fireAt + .6
		w.next = w.fireAt
		w.dx, w.dy = api.Direction(target.X-s.Self.X, target.Y-s.Self.Y)
		w.tick(ctx, s)
	case unit.Sense:
		w.tick(ctx, p)
	case unit.Stunned, api.Detached:
		w.stop(ctx)
		w.charge = 0
	}
}

func (w *weapon) stop(ctx unit.Context) {
	if w.active {
		ctx.Out <- unit.FX{Name: "heart-laser-off", Kind: ctx.Kind, UnitID: ctx.ID}
	}
	w.active = false
}
func (w *weapon) tick(ctx unit.Context, s unit.Sense) {
	if !w.active {
		return
	}
	if s.Time >= w.until-1e-8 {
		w.stop(ctx)
		return
	}
	d := api.Ray(s, s.Self.X, s.Self.Y, w.dx, w.dy, 360)
	ex, ey := s.Self.X+w.dx*d, s.Self.Y+w.dy*d
	if s.Time+1e-8 < w.fireAt {
		api.Flash(ctx, s, "heart-laser-aim", s.Self.X, s.Self.Y, ex, ey, math.Max(0, 1-(w.fireAt-s.Time)/.28))
		return
	}
	api.Flash(ctx, s, "heart-laser", s.Self.X, s.Self.Y, ex, ey, 1)
	if s.Time+1e-8 < w.next {
		return
	}
	// At most one exposure per Sense; skipped time never produces a burst of catch-up hits.
	w.next = math.Max(w.next+.12, s.Time+.12)
	for _, o := range s.Nearby {
		if unit.Hittable(o, s.Self.Slot) && o.HP > 0 && api.Distance(s.Self.X, s.Self.Y, ex, ey, o.X, o.Y) <= o.Radius+6 && api.Clear(s, s.Self.X, s.Self.Y, o) {
			ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: 1.6, NoFreeze: true}
		}
	}
}
