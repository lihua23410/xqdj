package 迷走

import (
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

const (
	ringCount   = 6
	trackCount  = 5
	ringRMin    = 70.0
	ringRMax    = 130.0
	crossMargin = 8.0
	mazeTries   = 80
	centerPad   = 8.0

	guideFrac    = 0.45
	guideFracMin = 0.28
	guideShrink  = 4.0
)

type ring struct {
	x, y, r float64
}

type junction struct {
	x, y float64
	a, b int
}

type maze struct {
	rings  []ring
	junc   []junction
	guideR float64 // 引导圆半径；0 表示无轨环引导（不应出现）
}

func buildMaze(f unit.Field, rng *rand.Rand) (maze, bool) {
	if rng == nil {
		return maze{}, false
	}
	for try := 0; try < mazeTries; try++ {
		hub, ok := pickHub(f, rng)
		if !ok {
			continue
		}
		guideR, tracks, ok := placeTrackAnchors(f, rng)
		if !ok {
			continue
		}
		centers := make([]pt, 0, ringCount)
		centers = append(centers, hub)
		centers = append(centers, tracks...)
		rings := radiiForSpread(centers, rng)
		junc := fieldJunctions(f, rings)
		if connected(ringCount, junc) {
			return maze{rings: rings, junc: junc, guideR: guideR}, true
		}
		for k := 0; k < 12; k++ {
			rings = radiiForSpread(centers, rng)
			junc = fieldJunctions(f, rings)
			if connected(ringCount, junc) {
				return maze{rings: rings, junc: junc, guideR: guideR}, true
			}
		}
	}
	return fallbackMaze(f, rng), true
}

func pickHub(f unit.Field, rng *rand.Rand) (pt, bool) {
	if f.Walkable(0, 0, centerPad) {
		return pt{0, 0}, true
	}
	// 近场心：小范围内找可进入点
	ext := f.Extent
	if ext < unit.MinExtent {
		ext = unit.MinExtent
	}
	span := ext * 0.18
	best, bestD := pt{}, math.Inf(1)
	found := false
	for i := 0; i < 40; i++ {
		x := (rng.Float64()*2 - 1) * span
		y := (rng.Float64()*2 - 1) * span
		x, y = f.Clamp(x, y, centerPad)
		if !f.Walkable(x, y, centerPad) {
			continue
		}
		d := math.Hypot(x, y)
		if d < bestD {
			best, bestD, found = pt{x, y}, d, true
		}
	}
	if found {
		return best, true
	}
	if x, y, ok := f.RandomWalkable(rng, centerPad); ok {
		return pt{x, y}, true
	}
	return pt{}, false
}

func placeTrackAnchors(f unit.Field, rng *rand.Rand) (guideR float64, tracks []pt, ok bool) {
	ext := f.Extent
	if ext < unit.MinExtent {
		ext = unit.MinExtent
	}
	base := rng.Float64() * 2 * math.Pi
	r := ext * guideFrac
	minR := ext * guideFracMin
	for r+1e-9 >= minR {
		pts := make([]pt, trackCount)
		good := true
		for i := 0; i < trackCount; i++ {
			ang := base + float64(i)*2*math.Pi/float64(trackCount)
			x := r * math.Cos(ang)
			y := r * math.Sin(ang)
			if !f.Walkable(x, y, centerPad) {
				good = false
				break
			}
			pts[i] = pt{x, y}
		}
		if good {
			return r, pts, true
		}
		r -= guideShrink
	}
	return 0, nil, false
}

func radiiForSpread(centers []pt, rng *rand.Rand) []ring {
	n := len(centers)
	rings := make([]ring, n)
	for i, c := range centers {
		nearest := math.Inf(1)
		second := math.Inf(1)
		for j, o := range centers {
			if i == j {
				continue
			}
			d := math.Hypot(c.x-o.x, c.y-o.y)
			if d < nearest {
				second = nearest
				nearest = d
			} else if d < second {
				second = d
			}
		}
		target := nearest*0.58 + second*0.12
		target += (rng.Float64()*2 - 1) * 12
		if target < ringRMin {
			target = ringRMin
		}
		if target > ringRMax {
			target = ringRMax
		}
		rings[i] = ring{x: c.x, y: c.y, r: target}
	}
	return rings
}

// fieldJunctions 只保留落在场地轮廓内的交点（pad=0，不把球半径算进去）。
func fieldJunctions(f unit.Field, rings []ring) []junction {
	raw := intersections(rings)
	out := make([]junction, 0, len(raw))
	for _, j := range raw {
		if f.OutlineContains(j.x, j.y, 0) {
			out = append(out, j)
		}
	}
	return out
}

func fallbackMaze(f unit.Field, rng *rand.Rand) maze {
	hub, ok := pickHub(f, rng)
	if !ok {
		hub = pt{0, 0}
	}
	guideR, tracks, ok := placeTrackAnchors(f, rng)
	if !ok {
		ext := f.Extent
		if ext < unit.MinExtent {
			ext = unit.MinExtent
		}
		guideR = ext * guideFracMin
		base := 0.0
		if rng != nil {
			base = rng.Float64() * 2 * math.Pi
		}
		tracks = make([]pt, trackCount)
		for i := 0; i < trackCount; i++ {
			ang := base + float64(i)*2*math.Pi/float64(trackCount)
			x, y := guideR*math.Cos(ang), guideR*math.Sin(ang)
			x, y = f.Clamp(x, y, centerPad)
			tracks[i] = pt{x, y}
		}
	}
	centers := make([]pt, 0, ringCount)
	centers = append(centers, hub)
	centers = append(centers, tracks...)
	if rng != nil {
		for k := 0; k < 40; k++ {
			rings := radiiForSpread(centers, rng)
			junc := fieldJunctions(f, rings)
			if connected(len(rings), junc) {
				return maze{rings: rings, junc: junc, guideR: guideR}
			}
		}
	}
	rings := make([]ring, len(centers))
	for i, c := range centers {
		rings[i] = ring{x: c.x, y: c.y, r: 100}
	}
	rings[0].r = 110
	return maze{rings: rings, junc: fieldJunctions(f, rings), guideR: guideR}
}

func intersections(rings []ring) []junction {
	out := make([]junction, 0, ringCount*2)
	for i := 0; i < len(rings); i++ {
		for j := i + 1; j < len(rings); j++ {
			p1, p2, ok := circleCross(rings[i], rings[j])
			if !ok {
				continue
			}
			out = append(out,
				junction{x: p1.x, y: p1.y, a: i, b: j},
				junction{x: p2.x, y: p2.y, a: i, b: j},
			)
		}
	}
	return out
}

type pt struct{ x, y float64 }

func circleCross(a, b ring) (pt, pt, bool) {
	dx, dy := b.x-a.x, b.y-a.y
	d := math.Hypot(dx, dy)
	if d < 1e-6 {
		return pt{}, pt{}, false
	}
	if d >= a.r+b.r-crossMargin || d <= math.Abs(a.r-b.r)+crossMargin {
		return pt{}, pt{}, false
	}
	aa := (a.r*a.r - b.r*b.r + d*d) / (2 * d)
	h2 := a.r*a.r - aa*aa
	if h2 <= 1e-6 {
		return pt{}, pt{}, false
	}
	h := math.Sqrt(h2)
	ux, uy := dx/d, dy/d
	px, py := a.x+ux*aa, a.y+uy*aa
	nx, ny := -uy*h, ux*h
	return pt{px + nx, py + ny}, pt{px - nx, py - ny}, true
}

func connected(n int, junc []junction) bool {
	if n <= 1 {
		return true
	}
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		if parent[i] != i {
			parent[i] = find(parent[i])
		}
		return parent[i]
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[rb] = ra
		}
	}
	for _, j := range junc {
		union(j.a, j.b)
	}
	root := find(0)
	for i := 1; i < n; i++ {
		if find(i) != root {
			return false
		}
	}
	return true
}

func nearestRing(rings []ring, x, y float64) int {
	best, bestD := 0, math.Inf(1)
	for i, r := range rings {
		d := math.Abs(math.Hypot(x-r.x, y-r.y) - r.r)
		if d < bestD {
			best, bestD = i, d
		}
	}
	return best
}

func pointOnRing(r ring, x, y float64) (px, py, ang float64) {
	ang = math.Atan2(y-r.y, x-r.x)
	return r.x + r.r*math.Cos(ang), r.y + r.r*math.Sin(ang), ang
}

func tangentVel(r ring, ang, spin, speed float64) (vx, vy float64) {
	return -math.Sin(ang) * spin * speed, math.Cos(ang) * spin * speed
}
