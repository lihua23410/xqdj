package 圆

import (
	"math"

	"xqdj/internal/unit"
)

func init() {
	unit.RegisterOutline(unit.ShapeCircle, newCircleOutline)
}

type circleOutline struct {
	R float64
}

func newCircleOutline(extent float64) unit.Outline {
	return &circleOutline{R: extent}
}

func (c *circleOutline) Contains(x, y, radius float64) bool {
	return math.Hypot(x, y) <= c.R-radius+1e-6
}

func (c *circleOutline) SampleEdge(t, inset float64) (x, y, nx, ny float64) {
	ang := t * 2 * math.Pi
	nx, ny = math.Cos(ang), math.Sin(ang)
	r := c.R - inset
	if r < 0 {
		r = 0
	}
	return nx * r, ny * r, nx, ny
}

func (c *circleOutline) NearestEdgeDir(x, y float64) (dx, dy float64) {
	n := math.Hypot(x, y)
	if n < 1e-9 {
		return 1, 0
	}
	return x / n, y / n
}

func (c *circleOutline) Sweep(px, py, vx, vy, faceX, faceY, radius, dt float64, semi bool) (t, nx, ny float64, hit bool) {
	_ = faceX
	_ = faceY
	_ = semi
	limit := c.R - radius
	if limit < 8 {
		limit = 8
	}
	r2 := px*px + py*py
	lim2 := limit * limit
	if r2 > lim2+1e-6 {
		n := math.Hypot(px, py)
		if n < 1e-9 {
			return 0, 1, 0, true
		}
		return 0, px / n, py / n, true
	}
	if px*vx+py*vy <= 1e-9 {
		return 0, 0, 0, false
	}
	a := vx*vx + vy*vy
	if a < 1e-16 {
		return 0, 0, 0, false
	}
	b := 2 * (px*vx + py*vy)
	cc := r2 - lim2
	disc := b*b - 4*a*cc
	if disc < 0 {
		return 0, 0, 0, false
	}
	tt := (-b + math.Sqrt(disc)) / (2 * a)
	if tt < -1e-9 || tt > dt {
		return 0, 0, 0, false
	}
	if tt < 0 {
		tt = 0
	}
	ax, ay := px+vx*tt, py+vy*tt
	n := math.Hypot(ax, ay)
	if n < 1e-9 {
		return tt, 1, 0, true
	}
	return tt, ax / n, ay / n, true
}

func (c *circleOutline) Constrain(x, y, radius, hintNx, hintNy, faceX, faceY, skin float64, semi bool) (ox, oy float64) {
	_ = hintNx
	_ = hintNy
	_ = faceX
	_ = faceY
	_ = semi
	limit := c.R - radius - skin
	if limit < 8 {
		limit = 8
	}
	d := math.Hypot(x, y)
	if d <= limit {
		return x, y
	}
	if d < 1e-9 {
		return limit, 0
	}
	s := limit / d
	return x * s, y * s
}
