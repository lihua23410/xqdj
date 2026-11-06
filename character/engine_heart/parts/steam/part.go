// Package steam owns lingering, fixed-world pressure clouds with slowing pulses.
package steam

import (
	"math"
	"xqdj/character/engine_heart/parts/api"
	"xqdj/internal/unit"
)

func Definition() api.Definition {
	return api.Definition{ID: 13, Key: "steam", Name: "蒸汽泄压缸", Install: func(*api.Build) {}, Weapon: func() unit.Actor { return &weapon{} }}
}

type weapon struct{ x, y, until, next float64 }

func (w *weapon) tick(ctx unit.Context, s unit.Sense) {
	if w.until == 0 || s.Time >= w.until || s.Time+1e-8 < w.next {
		return
	}
	w.next += .12
	for _, o := range s.Nearby {
		if unit.Hittable(o, s.Self.Slot) && o.HP > 0 && math.Hypot(o.X-w.x, o.Y-w.y) <= 82+o.Radius && api.Clear(s, w.x, w.y, o) {
			ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: 1.5, NoFreeze: true}
			ctx.Out <- unit.AddFSComponent{UnitID: o.ID, Zone: unit.FSZoneM, Token: unit.DeriveStream(ctx.ID, o.ID, 313), Value: .65, ExpiresAt: s.Time + .18}
		}
	}
}
func (w *weapon) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case api.Pulse:
		if e.Echo {
			return
		}
		s := e.Sense
		w.x, w.y, w.until, w.next = s.Self.X, s.Self.Y, s.Time+.36, s.Time
		api.Flash(ctx, s, "heart-steam", w.x, w.y, w.x, w.y, 1)
		w.tick(ctx, s)
	case unit.Sense:
		w.tick(ctx, e)
	case unit.Stunned, api.Detached:
		w.until = 0
	}
}
