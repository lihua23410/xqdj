package api

import (
	"math"
	"xqdj/internal/unit"
)

func CoreDashToken(id uint64) uint64 { return unit.DeriveStream(id, 0, 102) }

func Direction(x, y float64) (float64, float64) {
	n := math.Hypot(x, y)
	if n < 1e-9 {
		return 1, 0
	}
	return x / n, y / n
}

// Ray stops at arena boundaries, solid field walls and sensed wall capsules.
func Ray(s unit.Sense, x, y, dx, dy, length float64) float64 {
	n := max(1, int(math.Ceil(length/2)))
	for i := 1; i <= n; i++ {
		d := length * float64(i) / float64(n)
		px, py := x+dx*d, y+dy*d
		blocked := !s.Field.Walkable(px, py, 1)
		for _, w := range s.Walls {
			if Distance(w.X1, w.Y1, w.X2, w.Y2, px, py) <= w.Radius+1 {
				blocked = true
				break
			}
		}
		if blocked {
			return length * float64(i-1) / float64(n)
		}
	}
	return length
}

func Clear(s unit.Sense, x, y float64, o unit.Snapshot) bool {
	dx, dy := Direction(o.X-x, o.Y-y)
	d := math.Max(0, math.Hypot(o.X-x, o.Y-y)-o.Radius)
	return Ray(s, x, y, dx, dy, d) >= d-1e-6
}

func Target(s unit.Sense, reach float64) *unit.Snapshot {
	var best *unit.Snapshot
	for i := range s.Nearby {
		o := &s.Nearby[i]
		d := math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y)
		if !unit.Hittable(*o, s.Self.Slot) || o.HP <= 0 || d > reach+o.Radius || !Clear(s, s.Self.X, s.Self.Y, *o) {
			continue
		}
		if best == nil || d < math.Hypot(best.X-s.Self.X, best.Y-s.Self.Y) {
			best = o
		}
	}
	return best
}

func Distance(ax, ay, bx, by, x, y float64) float64 {
	dx, dy := bx-ax, by-ay
	t := 0.0
	if d := dx*dx + dy*dy; d > 1e-12 {
		t = math.Max(0, math.Min(1, ((x-ax)*dx+(y-ay)*dy)/d))
	}
	return math.Hypot(x-ax-t*dx, y-ay-t*dy)
}

func Flash(ctx unit.Context, s unit.Sense, name string, x, y, ex, ey, power float64) {
	FlashOwner(ctx, ctx.ID, s, name, x, y, ex, ey, power)
}

func FlashOwner(ctx unit.Context, owner uint64, s unit.Sense, name string, x, y, ex, ey, power float64) {
	ctx.Out <- unit.FX{Name: name, Kind: ctx.Kind, UnitID: owner, Slot: s.Self.Slot, X: x, Y: y, VX: ex, VY: ey, Amount: power}
}
