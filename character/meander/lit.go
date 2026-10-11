package 迷走

import (
	"math"

	"xqdj/internal/unit"
)

// 点亮：钉在当前环线圆心，按半径画亮环，并对蹭到可进入亮弧的敌人刮伤。
type 点亮 struct {
	slot  int
	cx    float64
	cy    float64
	r     float64
	hitAt map[uint64]float64
}

func (a *点亮) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	a.slot = s.Self.Slot
	a.cx, a.cy = s.Self.X, s.Self.Y
	if s.Self.Radius > 1 {
		a.r = s.Self.Radius
	}
	if a.hitAt == nil {
		a.hitAt = map[uint64]float64{}
	}
	half := litWidth * 0.5
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, a.slot) {
			continue
		}
		if t, ok := a.hitAt[o.ID]; ok && s.Time+1e-9 < t {
			continue
		}
		dx, dy := o.X-a.cx, o.Y-a.cy
		d := math.Hypot(dx, dy)
		if d < 1e-6 {
			continue
		}
		circ := math.Abs(d - a.r)
		if circ > o.Radius+half+1e-9 {
			continue
		}
		px := a.cx + dx/d*a.r
		py := a.cy + dy/d*a.r
		if !arcActive(s, px, py) {
			continue
		}
		a.hitAt[o.ID] = s.Time + litHitCD
		ctx.Out <- unit.Damage{
			From:     ctx.ID,
			To:       o.ID,
			Amount:   litDamage,
			NoFreeze: true,
		}
	}
}

func arcActive(s unit.Sense, x, y float64) bool {
	if !s.Field.Walkable(x, y, 0) {
		return false
	}
	for i := range s.Walls {
		w := &s.Walls[i]
		if wallCovers(w, x, y) {
			return false
		}
	}
	return true
}

func wallCovers(w *unit.WallView, x, y float64) bool {
	// Sense.Walls 一律按胶囊粗细量；场边不在 Walls 里，由 Field.Walkable 裁。
	d := distPointSeg(x, y, w.X1, w.Y1, w.X2, w.Y2)
	return d < w.Radius+1e-9
}

func distPointSeg(px, py, x1, y1, x2, y2 float64) float64 {
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
