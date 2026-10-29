package 非因果

import "math"

const (
	hitNone = iota
	hitWall
	hitBody
	hitMinionGhost
	hitFighterGhost
	hitOut
)

type circle struct {
	id   uint64
	x, y float64
	r    float64
	kind int
}

func isShotBlocker(kind int) bool {
	return kind == hitWall || kind == hitBody || kind == hitOut
}

// stepShot 沿这一小段看最先碰上什么。墙、真人、飞出场都只让弹消失。
func stepShot(x0, y0, x1, y1, bulletR float64, circles []circle, walls []wallSeg, outlineOutside func(x, y float64) (t float64, ok bool)) (kind int, id uint64) {
	bestT := 2.0
	kind = hitNone
	consider := func(t float64, k int, who uint64) {
		if t < -1e-9 || t > 1+1e-9 {
			return
		}
		if t < 0 {
			t = 0
		}
		earlier := t < bestT-1e-9
		// 同时接触时优先挡弹，不能由候选顺序决定是否打进重合的影。
		blockingTie := math.Abs(t-bestT) <= 1e-9 && isShotBlocker(k) && !isShotBlocker(kind)
		if earlier || blockingTie {
			bestT = t
			kind = k
			id = who
		}
	}
	if outlineOutside != nil {
		if t, ok := outlineOutside(x1, y1); ok {
			consider(t, hitOut, 0)
		}
	}
	for _, w := range walls {
		if t, ok := shotHitsWall(x0, y0, x1, y1, bulletR, w); ok {
			consider(t, hitWall, 0)
		}
	}
	for _, c := range circles {
		rad := bulletR + c.r
		if t, ok := shotHitsCircle(x0, y0, x1, y1, c.x, c.y, rad); ok {
			consider(t, c.kind, c.id)
		}
	}
	return kind, id
}

type wallSeg struct {
	x1, y1, x2, y2, r float64
}

func shotHitsCircle(x0, y0, x1, y1, cx, cy, rad float64) (float64, bool) {
	dx, dy := x1-x0, y1-y0
	fx, fy := x0-cx, y0-cy
	if fx*fx+fy*fy <= rad*rad {
		return 0, true
	}
	a := dx*dx + dy*dy
	if a < 1e-12 {
		return 0, false
	}
	b := 2 * (fx*dx + fy*dy)
	c := fx*fx + fy*fy - rad*rad
	disc := b*b - 4*a*c
	if disc < 0 {
		return 0, false
	}
	sd := math.Sqrt(disc)
	t1 := (-b - sd) / (2 * a)
	if t1 >= 0 && t1 <= 1 {
		return t1, true
	}
	t2 := (-b + sd) / (2 * a)
	if t2 >= 0 && t2 <= 1 {
		return t2, true
	}
	return 0, false
}

func shotHitsWall(x0, y0, x1, y1, bulletR float64, w wallSeg) (float64, bool) {
	reach := bulletR + w.r
	if distPointSeg(x0, y0, w.x1, w.y1, w.x2, w.y2) <= reach {
		return 0, true
	}
	// 这一小段里离墙最近的那一点。步长不大，用两端和中点就够。
	for _, t := range []float64{0.25, 0.5, 0.75, 1} {
		x := x0 + (x1-x0)*t
		y := y0 + (y1-y0)*t
		if distPointSeg(x, y, w.x1, w.y1, w.x2, w.y2) <= reach {
			return t, true
		}
	}
	return 0, false
}

func distPointSeg(px, py, x1, y1, x2, y2 float64) float64 {
	vx, vy := x2-x1, y2-y1
	l2 := vx*vx + vy*vy
	if l2 < 1e-12 {
		return math.Hypot(px-x1, py-y1)
	}
	t := ((px-x1)*vx + (py-y1)*vy) / l2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return math.Hypot(px-(x1+t*vx), py-(y1+t*vy))
}
