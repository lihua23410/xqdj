package unit

import "math"

const (
	detourPad  = 4.0 // 端点外侧再留的空隙，避免贴着圆头蹭过去
	detourHold = 0.3 // 记住这一侧这么久，转墙不至于每拍换边
)

// DetourHold 上次绕开的那一侧。Until 之前沿用，不每拍重挑。
type DetourHold struct {
	Until float64
	ID    uint64
	End   int // 1 = (X1,Y1)，2 = (X2,Y2)
}

// Detour 直线到 (tx,ty)。没被墙挡住就返回目标本身，并清空 hold。
// 挡住则改朝第一截挡路墙离目标更近的那个端点外侧。每截墙都算，不看挡视线。
// 场边不在 Walls 里，这里不绕。
func Detour(x, y, radius, tx, ty float64, walls []WallView, hold DetourHold, now float64) (wx, wy float64, next DetourHold) {
	w, ok := firstBlock(x, y, radius, tx, ty, walls)
	if !ok {
		return tx, ty, DetourHold{}
	}
	end := nearerEnd(w, tx, ty)
	if hold.ID == w.ID && now < hold.Until && (hold.End == 1 || hold.End == 2) {
		end = hold.End
		next = hold
	} else {
		next = DetourHold{Until: now + detourHold, ID: w.ID, End: end}
	}
	wx, wy = pastEnd(w, end, radius)
	return wx, wy, next
}

func nearerEnd(w WallView, tx, ty float64) int {
	d1 := math.Hypot(w.X1-tx, w.Y1-ty)
	d2 := math.Hypot(w.X2-tx, w.Y2-ty)
	if d2 < d1 {
		return 2
	}
	return 1
}

func pastEnd(w WallView, end int, radius float64) (float64, float64) {
	ex, ey := w.X1, w.Y1
	ox, oy := w.X2, w.Y2
	if end == 2 {
		ex, ey, ox, oy = ox, oy, ex, ey
	}
	dx, dy := ex-ox, ey-oy
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return ex, ey
	}
	gap := radius + w.Radius + detourPad
	return ex + dx/n*gap, ey + dy/n*gap
}

func firstBlock(x, y, radius, tx, ty float64, walls []WallView) (WallView, bool) {
	var best WallView
	bestT := math.MaxFloat64
	ok := false
	for i := range walls {
		w := walls[i]
		t, dist := approachOn(x, y, tx, ty, w.X1, w.Y1, w.X2, w.Y2)
		if dist > radius+w.Radius {
			continue
		}
		if ok && (t > bestT+1e-9 || (math.Abs(t-bestT) <= 1e-9 && w.ID >= best.ID)) {
			continue
		}
		best, bestT, ok = w, t, true
	}
	return best, ok
}

// approachOn 路径线段到墙线段的最近距离，以及最近点在路径上的参数（0–1）。
func approachOn(p1x, p1y, q1x, q1y, p2x, p2y, q2x, q2y float64) (t, dist float64) {
	ux, uy := q1x-p1x, q1y-p1y
	vx, vy := q2x-p2x, q2y-p2y
	wx, wy := p1x-p2x, p1y-p2y
	a := ux*ux + uy*uy
	b := ux*vx + uy*vy
	c := vx*vx + vy*vy
	d := ux*wx + uy*wy
	e := vx*wx + vy*wy
	D := a*c - b*b
	sN, sD := D, D
	tN, tD := D, D
	if D < 1e-12 {
		sN = 0
		sD = 1
		tN = e
		tD = c
	} else {
		sN = b*e - c*d
		tN = a*e - b*d
		if sN < 0 {
			sN = 0
			tN = e
			tD = c
		} else if sN > sD {
			sN = sD
			tN = e + b
			tD = c
		}
	}
	if tN < 0 {
		tN = 0
		if -d < 0 {
			sN = 0
		} else if -d > a {
			sN = sD
		} else {
			sN = -d
			sD = a
		}
	} else if tN > tD {
		tN = tD
		if -d+b < 0 {
			sN = 0
		} else if -d+b > a {
			sN = sD
		} else {
			sN = -d + b
			sD = a
		}
	}
	sc, tc := 0.0, 0.0
	if math.Abs(sN) > 1e-12 {
		sc = sN / sD
	}
	if math.Abs(tN) > 1e-12 {
		tc = tN / tD
	}
	c1x, c1y := p1x+ux*sc, p1y+uy*sc
	c2x, c2y := p2x+vx*tc, p2y+vy*tc
	return sc, math.Hypot(c1x-c2x, c1y-c2y)
}
