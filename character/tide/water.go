package tide

import (
	"math"

	"xqdj/internal/unit"
)

// Evaluate contact and wall shadows in the same narrow bands along the body.
// Each band stops at the closest wall surface anywhere in its forward span.
const (
	bandCount        = 18
	maxWakeSideWidth = 135.0
)

type point struct{ x, y float64 }

type waterFrame struct {
	origin  point
	forward point
}

func (f waterFrame) local(x, y float64) point {
	dx, dy := x-f.origin.x, y-f.origin.y
	return point{dx*f.forward.x + dy*f.forward.y, dy*f.forward.x - dx*f.forward.y}
}

type water struct {
	radius, outer float64
	edges         [bandCount][2]float64 // outward edges for negative and positive sides
}

func newWater(radius, width float64) water {
	w := water{radius: radius, outer: radius + width}
	for i := range w.edges {
		w.edges[i] = [2]float64{w.outer, w.outer}
	}
	return w
}

func (w water) band(i int) (float64, float64) {
	step := 2 * w.radius / bandCount
	return -w.radius + float64(i)*step, -w.radius + float64(i+1)*step
}

func (w *water) clip(frame waterFrame, wall unit.WallView) {
	a := frame.local(wall.X1, wall.Y1)
	b := frame.local(wall.X2, wall.Y2)
	r := wall.Radius
	if math.Min(a.x, b.x)-r > w.radius || math.Max(a.x, b.x)+r < -w.radius ||
		math.Min(a.y, b.y)-r > w.outer || math.Max(a.y, b.y)+r < -w.outer {
		return
	}
	length := math.Hypot(b.x-a.x, b.y-a.y)
	var corners [4]point
	if length > 1e-9 {
		nx, ny := -(b.y-a.y)*r/length, (b.x-a.x)*r/length
		corners = [4]point{{a.x + nx, a.y + ny}, {b.x + nx, b.y + ny}, {b.x - nx, b.y - ny}, {a.x - nx, a.y - ny}}
	}
	for i, edge := range w.edges {
		start, end := w.band(i)
		lo, hi := math.Inf(1), math.Inf(-1)
		if length > 1e-9 {
			lo, hi = slabRange(corners, start, end)
		}
		if !wall.Square || length <= 1e-9 {
			for _, cap := range [2]point{a, b} {
				dx := cap.x - clamp(cap.x, start, end)
				if dx*dx <= r*r {
					dy := math.Sqrt(r*r - dx*dx)
					lo, hi = math.Min(lo, cap.y-dy), math.Max(hi, cap.y+dy)
				}
			}
		}
		if lo < 0 {
			w.edges[i][0] = clamp(-hi, w.radius, edge[0])
		}
		if hi > 0 {
			w.edges[i][1] = clamp(lo, w.radius, edge[1])
		}
	}
}

// Project the part of the wall rectangle in [start, end] onto the lateral axis.
func slabRange(corners [4]point, start, end float64) (lo, hi float64) {
	lo, hi = math.Inf(1), math.Inf(-1)
	for i, a := range corners {
		if a.x >= start && a.x <= end {
			lo, hi = math.Min(lo, a.y), math.Max(hi, a.y)
		}
		b := corners[(i+1)%len(corners)]
		for _, x := range [2]float64{start, end} {
			if x > math.Min(a.x, b.x) && x < math.Max(a.x, b.x) {
				y := a.y + (b.y-a.y)*(x-a.x)/(b.x-a.x)
				lo, hi = math.Min(lo, y), math.Max(hi, y)
			}
		}
	}
	return lo, hi
}

func (w *water) hitSide(center point, radius float64) float64 {
	preferred := 1
	if center.y < 0 {
		preferred = 0
	}
	for _, side := range [2]int{preferred, 1 - preferred} {
		sign := float64(2*side - 1)
		y := sign * center.y
		for i, edge := range w.edges {
			if edge[side] <= w.radius {
				continue
			}
			start, end := w.band(i)
			dx := center.x - clamp(center.x, start, end)
			dy := y - clamp(y, w.radius, edge[side])
			if dx*dx+dy*dy <= radius*radius {
				return sign
			}
		}
	}
	return 0
}

func (w *water) sideWidths() (negW, posW float64) {
	var sumNeg, sumPos float64
	for _, edge := range w.edges {
		sumNeg += clamp(edge[0]-w.radius, 0, maxWakeSideWidth)
		sumPos += clamp(edge[1]-w.radius, 0, maxWakeSideWidth)
	}
	return sumNeg / bandCount, sumPos / bandCount
}

func blockedByWall(ax, ay, bx, by float64, walls []unit.WallView) bool {
	p0, p1 := point{ax, ay}, point{bx, by}
	for _, w := range walls {
		wa, wb := point{w.X1, w.Y1}, point{w.X2, w.Y2}
		wx, wy := wb.x-wa.x, wb.y-wa.y
		wlen := math.Hypot(wx, wy)
		if w.Square && wlen > 1e-9 {
			ux, uy := wx/wlen, wy/wlen
			nx, ny := -uy, ux
			alx := (ax-wa.x)*ux + (ay-wa.y)*uy
			aly := (ax-wa.x)*nx + (ay-wa.y)*ny
			blx := (bx-wa.x)*ux + (by-wa.y)*uy
			bly := (bx-wa.x)*nx + (by-wa.y)*ny
			if segIntersectsBox(alx, aly, blx, bly, 0, wlen, -w.Radius, w.Radius) {
				return true
			}
			continue
		}
		if segSegDist(p0, p1, wa, wb) <= w.Radius {
			return true
		}
	}
	return false
}

func segIntersectsBox(ax, ay, bx, by, minX, maxX, minY, maxY float64) bool {
	t0, t1 := 0.0, 1.0
	dx, dy := bx-ax, by-ay
	for _, slab := range [2]struct{ p, d, lo, hi float64 }{
		{ax, dx, minX, maxX},
		{ay, dy, minY, maxY},
	} {
		if math.Abs(slab.d) < 1e-12 {
			if slab.p < slab.lo || slab.p > slab.hi {
				return false
			}
			continue
		}
		inv := 1.0 / slab.d
		s0 := (slab.lo - slab.p) * inv
		s1 := (slab.hi - slab.p) * inv
		if s0 > s1 {
			s0, s1 = s1, s0
		}
		if s0 > t0 {
			t0 = s0
		}
		if s1 < t1 {
			t1 = s1
		}
		if t0 > t1 {
			return false
		}
	}
	return true
}

func segSegDist(p1, q1, p2, q2 point) float64 {
	d1x, d1y := q1.x-p1.x, q1.y-p1.y
	d2x, d2y := q2.x-p2.x, q2.y-p2.y
	rx, ry := p1.x-p2.x, p1.y-p2.y
	a := d1x*d1x + d1y*d1y
	e := d2x*d2x + d2y*d2y
	f := d2x*rx + d2y*ry
	var s, t float64
	if a <= 1e-12 && e <= 1e-12 {
		return math.Hypot(rx, ry)
	}
	if a <= 1e-12 {
		s = 0
		t = clamp(f/e, 0, 1)
	} else {
		c := d1x*rx + d1y*ry
		if e <= 1e-12 {
			t = 0
			s = clamp(-c/a, 0, 1)
		} else {
			b := d1x*d2x + d1y*d2y
			denom := a*e - b*b
			if denom != 0 {
				s = clamp((b*f-c*e)/denom, 0, 1)
			} else {
				s = 0
			}
			t = (b*s + f) / e
			if t < 0 {
				t = 0
				s = clamp(-c/a, 0, 1)
			} else if t > 1 {
				t = 1
				s = clamp((b-c)/a, 0, 1)
			}
		}
	}
	cx := rx + d1x*s - d2x*t
	cy := ry + d1y*s - d2y*t
	return math.Hypot(cx, cy)
}

type wakeSegment struct {
	id       uint64
	x, y     float64
	fx, fy   float64
	halfLen  float64
	negW     float64
	posW     float64
	bornAt   float64
	life     float64
	flaredAt float64
	surfedAt float64
}

func (seg wakeSegment) extents(now float64) (negEdge, posEdge float64, alive bool) {
	age := now - seg.bornAt
	if age < 0 {
		age = 0
	}
	if seg.life <= 0 || age >= seg.life {
		return 0, 0, false
	}
	rem := clamp(1-age/seg.life, 0, 1)
	flare := 1.0
	if seg.flaredAt >= 0 && now-seg.flaredAt < 0.45 {
		flare = 1.0 + 0.35*(1-clamp((now-seg.flaredAt)/0.45, 0, 1))
	}
	base := bodyRadius * 0.35
	return base + seg.negW*rem*flare, base + seg.posW*rem*flare, true
}

func (seg wakeSegment) testCircle(tx, ty, tr, now float64) (hit bool, inX, inY float64) {
	negEdge, posEdge, alive := seg.extents(now)
	if !alive {
		return false, 0, 0
	}
	dx, dy := tx-seg.x, ty-seg.y
	lx := dx*seg.fx + dy*seg.fy
	ly := dy*seg.fx - dx*seg.fy
	cx := clamp(lx, -seg.halfLen, seg.halfLen)
	cy := clamp(ly, -negEdge, posEdge)
	ddx, ddy := lx-cx, ly-cy
	if ddx*ddx+ddy*ddy > tr*tr {
		return false, 0, 0
	}
	if ly > 1e-6 {
		return true, seg.fy, -seg.fx
	}
	if ly < -1e-6 {
		return true, -seg.fy, seg.fx
	}
	return true, 0, 0
}

func clamp(x, lo, hi float64) float64 {
	return math.Min(math.Max(x, lo), hi)
}
