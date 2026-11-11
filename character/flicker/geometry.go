package 闪烁

import (
	"math"
	"xqdj/internal/unit"
)

func clipReach(px, py, ux, uy, maxLen, halfW float64, field unit.Field, walls []unit.WallView) float64 {
	if maxLen <= 0 {
		return 0
	}
	step := 4.0
	if halfW > step {
		step = halfW
	}
	prev := 0.0
	for t := math.Min(step, maxLen); prev < maxLen; t = math.Min(t+step, maxLen) {
		if t > maxLen {
			t = maxLen
		}
		if !blockedAt(px+ux*t, py+uy*t, halfW, field, walls) {
			prev = t
			if t >= maxLen {
				return maxLen
			}
			continue
		}
		lo, hi := prev, t
		for j := 0; j < 18; j++ {
			mid := 0.5 * (lo + hi)
			if !blockedAt(px+ux*mid, py+uy*mid, halfW, field, walls) {
				lo = mid
			} else {
				hi = mid
			}
		}
		return hi
	}
	return maxLen
}

func blockedAt(x, y, halfW float64, field unit.Field, walls []unit.WallView) bool {
	if !field.Walkable(x, y, halfW) {
		return true
	}
	for i := range walls {
		w := &walls[i]
		if segDist(x, y, w.X1, w.Y1, w.X2, w.Y2) < w.Radius+halfW {
			return true
		}
	}
	return false
}

func segDist(px, py, x1, y1, x2, y2 float64) float64 {
	dx, dy := x2-x1, y2-y1
	l2 := dx*dx + dy*dy
	if l2 < 1e-12 {
		return math.Hypot(px-x1, py-y1)
	}
	t := ((px-x1)*dx + (py-y1)*dy) / l2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return math.Hypot(px-(x1+dx*t), py-(y1+dy*t))
}
