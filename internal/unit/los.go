package unit

import "math"

// segSegDist 线段 p1→q1 到线段 p2→q2 的最近距离。
// 与 sim 包的 segDist 同一套 clamp 算法：unit 不能反向 import sim，各留一份、口径一致。
func segSegDist(p1x, p1y, q1x, q1y, p2x, p2y, q2x, q2y float64) float64 {
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
	return math.Hypot(c1x-c2x, c1y-c2y)
}

// SightBlocked：s.Self 球心到 o 球心的连线段是否被带挡视线标签的实体压住。
// 墙按胶囊量（轴 + Radius），单位按圆量（中心到线段的距离 ≤ Radius）；相切算挡（≤）。
// 敌我不分：自家带标签单位也挡自家索敌。排除自身和目标本身；标签与实心无关。
// 场地预放墙不带标签，对这里永远透明。
func SightBlocked(s Sense, o Snapshot) bool {
	for i := range s.Walls {
		w := &s.Walls[i]
		if !w.VisionBlock {
			continue
		}
		if segSegDist(s.Self.X, s.Self.Y, o.X, o.Y, w.X1, w.Y1, w.X2, w.Y2) <= w.Radius {
			return true
		}
	}
	for i := range s.Nearby {
		b := &s.Nearby[i]
		if !b.VisionBlock || b.ID == s.Self.ID || b.ID == o.ID {
			continue
		}
		if distToSeg(b.X, b.Y, s.Self.X, s.Self.Y, o.X, o.Y) <= b.Radius {
			return true
		}
	}
	return false
}

// SeekLOS 在 Seek 之上再要求视线不被带挡视线标签的实体挡住：墙后的人跳过，剩下的照
// 瞄准优先度取。没带标签的实体与 Seek 完全一致。挡视线只影响这一层索敌，不进感知
// 过滤、不挡物理；不用 SeekLOS 的角色隔墙照常索敌。
func SeekLOS(s Sense) *Snapshot {
	return SeekIf(s, func(o Snapshot) bool {
		return !SightBlocked(s, o)
	})
}
