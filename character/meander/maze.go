package 迷走

import (
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

const (
	ringCount   = 6
	ringRMin    = 70.0
	ringRMax    = 130.0
	crossMargin = 8.0
	mazeTries   = 80
	centerPad   = 8.0
	minCenterFrac = 0.30
	minCenterAbs  = 80.0
)

type ring struct {
	x, y, r float64
}

type junction struct {
	x, y   float64
	a, b   int
}

type maze struct {
	rings []ring
	junc  []junction
}

func buildMaze(f unit.Field, rng *rand.Rand) (maze, bool) {
	if rng == nil {
		return maze{}, false
	}
	for try := 0; try < mazeTries; try++ {
		centers, ok := spreadCenters(f, rng, ringCount)
		if !ok {
			continue
		}
		rings := radiiForSpread(centers, rng)
		junc := fieldJunctions(f, rings)
		if connected(ringCount, junc) {
			return maze{rings: rings, junc: junc}, true
		}
		for k := 0; k < 12; k++ {
			rings = radiiForSpread(centers, rng)
			junc = fieldJunctions(f, rings)
			if connected(ringCount, junc) {
				return maze{rings: rings, junc: junc}, true
			}
		}
	}
	return fallbackMaze(f, rng), true
}

func spreadCenters(f unit.Field, rng *rand.Rand, n int) ([]pt, bool) {
	cands := centerCandidates(f, rng)
	if len(cands) < n {
		return nil, false
	}
	minD := minCenterDist(f)
	picked := make([]pt, 0, n)
	picked = append(picked, cands[nearestCandidate(cands, 0, 0)])

	for len(picked) < n {
		bestI, bestScore := -1, -1.0
		for i, c := range cands {
			if tooClose(c, picked, minD*0.55) {
				continue
			}
			dmin := minDistTo(c, picked)
			// 略偏外铺满，但不要贴边过狠（圆/八边形交点易出界）。
			edge := math.Hypot(c.x, c.y) / math.Max(f.Extent, 1)
			score := dmin + edge*minD*0.06
			if score > bestScore {
				bestI, bestScore = i, score
			}
		}
		if bestI < 0 {
			bestI, bestScore = -1, -1.0
			for i, c := range cands {
				if containsPt(picked, c) {
					continue
				}
				dmin := minDistTo(c, picked)
				if dmin > bestScore {
					bestI, bestScore = i, dmin
				}
			}
			if bestI < 0 {
				return nil, false
			}
		}
		picked = append(picked, cands[bestI])
	}
	return picked, true
}

func centerCandidates(f unit.Field, rng *rand.Rand) []pt {
	ext := f.Extent
	if ext < unit.MinExtent {
		ext = unit.MinExtent
	}
	span := ext * 0.72
	step := span * 0.24
	if step < 32 {
		step = 32
	}
	out := make([]pt, 0, 96)
	for x := -span; x <= span+1e-9; x += step {
		for y := -span; y <= span+1e-9; y += step {
			jx := x + (rng.Float64()-0.5)*step*0.7
			jy := y + (rng.Float64()-0.5)*step*0.7
			jx, jy = f.Clamp(jx, jy, centerPad)
			if f.Walkable(jx, jy, centerPad) {
				out = append(out, pt{jx, jy})
			}
		}
	}
	for i := 0; i < 28; i++ {
		x, y, ok := f.RandomWalkable(rng, centerPad)
		if ok {
			out = append(out, pt{x, y})
		}
	}
	return out
}

func minCenterDist(f unit.Field) float64 {
	d := f.Extent * minCenterFrac
	if d < minCenterAbs {
		d = minCenterAbs
	}
	return d
}

func nearestCandidate(cands []pt, x, y float64) int {
	best, bestD := 0, math.Inf(1)
	for i, c := range cands {
		d := math.Hypot(c.x-x, c.y-y)
		if d < bestD {
			best, bestD = i, d
		}
	}
	return best
}

func minDistTo(p pt, set []pt) float64 {
	best := math.Inf(1)
	for _, q := range set {
		d := math.Hypot(p.x-q.x, p.y-q.y)
		if d < best {
			best = d
		}
	}
	return best
}

func tooClose(p pt, set []pt, minD float64) bool {
	return minDistTo(p, set) < minD
}

func containsPt(set []pt, p pt) bool {
	for _, q := range set {
		if math.Hypot(p.x-q.x, p.y-q.y) < 1e-6 {
			return true
		}
	}
	return false
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
	ext := f.Extent
	if ext < unit.MinExtent {
		ext = unit.MinExtent
	}
	reach := ext * 0.38
	baseAng := 0.0
	if rng != nil {
		baseAng = rng.Float64() * 2 * math.Pi
	}
	cx, cy := 0.0, 0.0
	if !f.Walkable(cx, cy, centerPad) {
		if x, y, ok := f.RandomWalkable(rng, centerPad); ok {
			cx, cy = x, y
		} else {
			cx, cy = f.Clamp(50, 0, centerPad)
		}
	}
	centers := []pt{{cx, cy}}
	for i := 0; i < ringCount-1; i++ {
		ang := baseAng + float64(i)*2*math.Pi/float64(ringCount-1)
		x := cx + math.Cos(ang)*reach
		y := cy + math.Sin(ang)*reach
		x, y = f.Clamp(x, y, centerPad)
		if !f.Walkable(x, y, centerPad) {
			if rx, ry, ok := f.RandomWalkable(rng, centerPad); ok {
				x, y = rx, ry
			}
		}
		centers = append(centers, pt{x, y})
	}
	if rng != nil {
		for k := 0; k < 40; k++ {
			rings := radiiForSpread(centers, rng)
			junc := fieldJunctions(f, rings)
			if connected(len(rings), junc) {
				return maze{rings: rings, junc: junc}
			}
		}
	}
	rings := make([]ring, len(centers))
	for i, c := range centers {
		rings[i] = ring{x: c.x, y: c.y, r: 100}
	}
	rings[0].r = 110
	return maze{rings: rings, junc: fieldJunctions(f, rings)}
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
