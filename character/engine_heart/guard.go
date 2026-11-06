package 泵之心

import (
	"math"
	"xqdj/character/engine_heart/parts/api"
	"xqdj/internal/unit"
)

type seenAttack struct {
	snapshot unit.Snapshot
	time     float64
}

func (w *world) guards(now float64) []api.Guard {
	b := w.build()
	f := api.Frame{Time: now, Phase: w.phase, Self: w.self, Suspended: w.suspended, Loaded: w.ammo}
	var result []api.Guard
	for _, makeGuard := range b.Guards {
		g := makeGuard(f)
		if g.Outer > g.Inner {
			result = append(result, g)
		}
	}
	return result
}

func ordinaryAttack(o unit.Snapshot) bool {
	spec, ok := unit.Lookup(o.Kind)
	return ok && spec.Role == unit.RoleProjectile && !spec.Nonsolid && !spec.Shell
}

func angular(x, y, angle, span float64) bool {
	return math.Cos(math.Atan2(y, x)-angle) >= math.Cos(span/2)-1e-9
}

// Guard geometry lives in this role. Sweep samples are at most two world units
// apart, including both ends, rather than checking only a fast bullet's endpoint.
func crossesGuard(g api.Guard, ox, oy, nx, ny, r float64) bool {
	n := max(1, int(math.Ceil(math.Hypot(nx-ox, ny-oy)/2)))
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		x, y := ox+(nx-ox)*t, oy+(ny-oy)*t
		d := math.Hypot(x, y)
		if d+r < g.Inner || d-r > g.Outer {
			continue
		}
		pad := math.Asin(math.Min(1, r/math.Max(d, 1)))
		if angular(x, y, g.Angle, g.Span+2*pad) {
			return true
		}
	}
	return false
}

func fanTouches(g api.Guard, o, self unit.Snapshot) bool {
	fx, fy := heading(o)
	face := math.Atan2(fy, fx)
	steps := max(2, int(math.Ceil(g.Span*g.Outer/2)))
	for i := 0; i <= steps; i++ {
		a := g.Angle - g.Span/2 + g.Span*float64(i)/float64(steps)
		r := (g.Inner + g.Outer) / 2
		x, y := self.X+math.Cos(a)*r-o.X, self.Y+math.Sin(a)*r-o.Y
		d := math.Hypot(x, y)
		thickness := (g.Outer - g.Inner) / 2
		if d+thickness >= o.ArcInner && d-thickness <= o.Radius && angular(x, y, face, o.ArcSpan+.08) {
			return true
		}
	}
	return false
}

func (w *world) caught(ctx unit.Context, g api.Guard, o unit.Snapshot, now float64) {
	if w.blocked == nil {
		w.blocked = map[uint64]float64{}
	}
	w.blocked[o.ID] = now + .25
	if g.Capture {
		w.ammo = true
	}
	ctx.Out <- unit.Despawn{UnitID: o.ID}
	ctx.Out <- unit.FX{Name: "heart-block", Kind: KindEngineHeart, UnitID: w.self.ID, X: o.X, Y: o.Y, Slot: w.self.Slot}
}

func (w *world) intercept(ctx unit.Context, s unit.Sense, self unit.Snapshot, dt float64) {
	if w.seen == nil {
		w.seen = map[uint64]seenAttack{}
	}
	for _, o := range s.Nearby {
		old, known := w.seen[o.ID]
		if o.Slot != self.Slot && ordinaryAttack(o) && w.blocked[o.ID] < s.Time {
			for _, g := range w.guards(s.Time) {
				if g.Capture && o.ArcSpan > 0 {
					continue
				}
				hit := false
				if o.ArcSpan > 0 {
					hit = fanTouches(g, o, self)
				} else {
					x, y := o.X-o.VX*dt, o.Y-o.VY*dt
					if known {
						x, y = old.snapshot.X, old.snapshot.Y
					}
					hit = crossesGuard(g, x-self.X, y-self.Y, o.X-self.X, o.Y-self.Y, o.Radius)
				}
				if hit {
					w.caught(ctx, g, o, s.Time)
					break
				}
			}
		}
		w.seen[o.ID] = seenAttack{o, s.Time}
	}
	for id, v := range w.seen {
		if s.Time-v.time > 1 {
			delete(w.seen, id)
		}
	}
	for id, until := range w.blocked {
		if until < s.Time-.5 {
			delete(w.blocked, id)
		}
	}
}

// Damage offers are the fallback for a bullet traversing plate and body within
// one physics tick. Unknown non-fan sources are never treated as projectiles.
// The old protocol cannot recover a vanished, never-observed projectile's path.
func (w *world) blockOffer(ctx unit.Context, e unit.IncomingDamage) bool {
	if until, ok := w.blocked[e.From]; ok && e.Time <= until {
		return true
	}
	seen, ok := w.seen[e.From]
	o := seen.snapshot
	if !ok && e.FromFan() {
		seen, ok = w.seen[e.SourceOwnerID]
		o = seen.snapshot
		o.ArcSpan = e.ArcSpan
	}
	if !ok || (!e.FromFan() && !ordinaryAttack(o)) {
		return false
	}
	dx, dy := o.X-w.self.X, o.Y-w.self.Y
	for _, g := range w.guards(e.Time) {
		if g.Capture && e.FromFan() {
			continue
		}
		if math.Hypot(dx, dy) < g.Inner-o.Radius {
			continue
		}
		pad := math.Asin(math.Min(1, o.Radius/math.Max(g.Inner, 1)))
		if e.FromFan() {
			pad = 0
		} // fan radius is reach, not a bullet's thickness
		if angular(dx, dy, g.Angle, g.Span+2*pad) {
			o.ID = e.From
			w.caught(ctx, g, o, e.Time)
			return true
		}
	}
	return false
}
