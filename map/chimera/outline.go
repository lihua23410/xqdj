package 奇美拉

import (
	"math"

	"xqdj/internal/unit"
)

func init() {
	unit.RegisterOutline(unit.ShapeChimera, newChimeraOutline)
}

type polyOutline struct {
	pts []pt
}

func newChimeraOutline(float64) unit.Outline {
	return &polyOutline{pts: contour()}
}

func (p *polyOutline) Contains(x, y, radius float64) bool {
	if !p.inside(x, y) {
		return false
	}
	if radius <= 0 {
		return true
	}
	return p.dist(x, y) >= radius-1e-6
}

func (p *polyOutline) inside(x, y float64) bool {
	n := len(p.pts)
	in := false
	j := n - 1
	for i := 0; i < n; i++ {
		xi, yi := p.pts[i].x, p.pts[i].y
		xj, yj := p.pts[j].x, p.pts[j].y
		if (yi > y) != (yj > y) {
			xint := (xj-xi)*(y-yi)/(yj-yi) + xi
			if x < xint {
				in = !in
			}
		}
		j = i
	}
	return in
}

func (p *polyOutline) dist(x, y float64) float64 {
	best := math.Inf(1)
	n := len(p.pts)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		d := distSeg(x, y, p.pts[i], p.pts[j])
		if d < best {
			best = d
		}
	}
	return best
}

func distSeg(px, py float64, a, b pt) float64 {
	dx, dy := b.x-a.x, b.y-a.y
	l2 := dx*dx + dy*dy
	if l2 < 1e-12 {
		return math.Hypot(px-a.x, py-a.y)
	}
	t := ((px-a.x)*dx + (py-a.y)*dy) / l2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return math.Hypot(px-(a.x+dx*t), py-(a.y+dy*t))
}

func (p *polyOutline) edge(i int) (a, b pt, nx, ny, length float64) {
	n := len(p.pts)
	a = p.pts[i]
	b = p.pts[(i+1)%n]
	dx, dy := b.x-a.x, b.y-a.y
	length = math.Hypot(dx, dy)
	if length < 1e-12 {
		return a, b, 1, 0, 0
	}
	// 逆时针轮廓，朝外的法线在前进方向右侧。
	nx, ny = dy/length, -dx/length
	return a, b, nx, ny, length
}

func (p *polyOutline) SampleEdge(t, inset float64) (x, y, nx, ny float64) {
	n := len(p.pts)
	if n == 0 {
		return 0, 0, 1, 0
	}
	total := 0.0
	lens := make([]float64, n)
	for i := 0; i < n; i++ {
		_, _, _, _, L := p.edge(i)
		lens[i] = L
		total += L
	}
	if total < 1e-9 {
		return p.pts[0].x, p.pts[0].y, 1, 0
	}
	if t < 0 {
		t = 0
	}
	t = t - math.Floor(t)
	target := t * total
	acc := 0.0
	for i := 0; i < n; i++ {
		if acc+lens[i] >= target || i == n-1 {
			frac := 0.0
			if lens[i] > 1e-12 {
				frac = (target - acc) / lens[i]
			}
			a, b, nnx, nny, _ := p.edge(i)
			x = a.x + (b.x-a.x)*frac - nnx*inset
			y = a.y + (b.y-a.y)*frac - nny*inset
			return x, y, nnx, nny
		}
		acc += lens[i]
	}
	return p.pts[0].x, p.pts[0].y, 1, 0
}

func (p *polyOutline) NearestEdgeDir(x, y float64) (dx, dy float64) {
	best := math.Inf(1)
	dx, dy = 1, 0
	n := len(p.pts)
	for i := 0; i < n; i++ {
		a, b, nx, ny, _ := p.edge(i)
		d := distSeg(x, y, a, b)
		if d < best {
			best = d
			dx, dy = nx, ny
		}
	}
	return dx, dy
}

func (p *polyOutline) Sweep(px, py, vx, vy, faceX, faceY, radius, dt float64, semi bool) (t, nx, ny float64, hit bool) {
	_ = faceX
	_ = faceY
	_ = semi
	bestT := dt + 1
	var bestNx, bestNy float64
	found := false
	n := len(p.pts)
	for i := 0; i < n; i++ {
		a, b, enx, eny, L := p.edge(i)
		if L < 1e-9 {
			continue
		}
		tt, nnx, nny, ok := sweepEdge(px, py, vx, vy, radius, dt, a, b, enx, eny)
		if !ok || tt >= bestT {
			continue
		}
		bestT = tt
		bestNx, bestNy = nnx, nny
		found = true
	}
	return bestT, bestNx, bestNy, found
}

func sweepEdge(px, py, vx, vy, radius, dt float64, a, b pt, enx, eny float64) (t, nx, ny float64, hit bool) {
	bestT := dt + 1
	found := false
	dist := (px-a.x)*enx + (py-a.y)*eny
	vn := vx*enx + vy*eny
	if vn > 1e-9 {
		tt := (-radius - dist) / vn
		if tt >= -1e-9 && tt <= dt {
			if tt < 0 {
				tt = 0
			}
			hx, hy := px+vx*tt, py+vy*tt
			qx, qy := hx-enx*radius, hy-eny*radius
			if onSeg(qx, qy, a, b) && tt < bestT {
				bestT = tt
				nx, ny = enx, eny
				found = true
			}
		}
	}
	for _, v := range []pt{a, b} {
		tt, nnx, nny, ok := sweepPoint(px, py, vx, vy, radius, dt, v.x, v.y)
		if ok && tt < bestT {
			bestT = tt
			nx, ny = nnx, nny
			found = true
		}
	}
	return bestT, nx, ny, found
}

func onSeg(x, y float64, a, b pt) bool {
	dx, dy := b.x-a.x, b.y-a.y
	l2 := dx*dx + dy*dy
	if l2 < 1e-12 {
		return math.Hypot(x-a.x, y-a.y) < 1e-4
	}
	t := ((x-a.x)*dx + (y-a.y)*dy) / l2
	return t >= -1e-4 && t <= 1+1e-4
}

func sweepPoint(px, py, vx, vy, radius, dt, ax, ay float64) (t, nx, ny float64, hit bool) {
	rx, ry := px-ax, py-ay
	aa := vx*vx + vy*vy
	bb := 2 * (rx*vx + ry*vy)
	cc := rx*rx + ry*ry - radius*radius
	if aa < 1e-16 {
		if cc <= 0 {
			n := math.Hypot(rx, ry)
			if n < 1e-9 {
				return 0, 1, 0, true
			}
			return 0, rx / n, ry / n, true
		}
		return 0, 0, 0, false
	}
	disc := bb*bb - 4*aa*cc
	if disc < 0 {
		return 0, 0, 0, false
	}
	tt := (-bb - math.Sqrt(disc)) / (2 * aa)
	if tt < -1e-9 || tt > dt {
		if cc <= 1e-8 {
			n := math.Hypot(rx, ry)
			if n < 1e-9 {
				return 0, 1, 0, true
			}
			return 0, -rx / n, -ry / n, true
		}
		return 0, 0, 0, false
	}
	if tt < 0 {
		tt = 0
	}
	hx, hy := px+vx*tt-ax, py+vy*tt-ay
	n := math.Hypot(hx, hy)
	if n < 1e-9 {
		return tt, 1, 0, true
	}
	// 从顶点指向圆心的反方向是朝外（圆心在场内）。
	return tt, -hx / n, -hy / n, true
}

func (p *polyOutline) Constrain(x, y, radius, hintNx, hintNy, faceX, faceY, skin float64, semi bool) (ox, oy float64) {
	_ = faceX
	_ = faceY
	_ = semi
	need := radius + skin
	if hintNx*hintNx+hintNy*hintNy > 1e-12 {
		return p.constrainHint(x, y, need, hintNx, hintNy)
	}
	ox, oy = x, y
	for iter := 0; iter < 8; iter++ {
		in := p.inside(ox, oy)
		qx, qy, dist := p.closest(ox, oy)
		if dist < 1e-9 {
			return ox, oy
		}
		if in && dist >= need-1e-6 {
			return ox, oy
		}
		var nx, ny float64
		if in {
			nx, ny = (ox-qx)/dist, (oy-qy)/dist
			ox += nx * (need - dist)
			oy += ny * (need - dist)
		} else {
			nx, ny = (qx-ox)/dist, (qy-oy)/dist
			ox += nx * (dist + need)
			oy += ny * (dist + need)
		}
	}
	return ox, oy
}

func (p *polyOutline) constrainHint(x, y, need, nx, ny float64) (float64, float64) {
	if p.Contains(x, y, need) {
		return x, y
	}
	lo, hi := 0.0, need+64
	for i := 0; i < 24; i++ {
		mid := 0.5 * (lo + hi)
		if p.Contains(x-nx*mid, y-ny*mid, need) {
			hi = mid
		} else {
			lo = mid
		}
	}
	return x - nx*hi, y - ny*hi
}

func (p *polyOutline) closest(x, y float64) (qx, qy, dist float64) {
	best := math.Inf(1)
	n := len(p.pts)
	for i := 0; i < n; i++ {
		a := p.pts[i]
		b := p.pts[(i+1)%n]
		dx, dy := b.x-a.x, b.y-a.y
		l2 := dx*dx + dy*dy
		t := 0.0
		if l2 > 1e-12 {
			t = ((x-a.x)*dx + (y-a.y)*dy) / l2
			if t < 0 {
				t = 0
			} else if t > 1 {
				t = 1
			}
		}
		cx, cy := a.x+dx*t, a.y+dy*t
		d := math.Hypot(x-cx, y-cy)
		if d < best {
			best = d
			qx, qy = cx, cy
		}
	}
	return qx, qy, best
}
