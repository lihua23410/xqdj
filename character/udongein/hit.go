package 人偶使

import (
	"math"

	"xqdj/internal/unit"
)

func fanHit(ox, oy, ux, uy, span, reach float64, e unit.Snapshot) bool {
	dx, dy := e.X-ox, e.Y-oy
	dist := math.Hypot(dx, dy)
	if dist > reach+e.Radius+1e-6 {
		return false
	}
	if dist < 1e-6 {
		return true
	}
	cos := (dx*ux + dy*uy) / dist
	if cos > 1 {
		cos = 1
	} else if cos < -1 {
		cos = -1
	}
	ang := math.Acos(cos)
	extra := 0.0
	if e.Radius < dist {
		extra = math.Asin(e.Radius / dist)
	}
	return ang <= span/2+extra+1e-6
}

func rectHit(ox, oy, ux, uy, w, h float64, e unit.Snapshot) bool {
	px, py := e.X-ox, e.Y-oy
	along := px*ux + py*uy
	side := -px*uy + py*ux
	return math.Abs(along) <= w/2+e.Radius+1e-6 && math.Abs(side) <= h/2+e.Radius+1e-6
}

func ellipseHit(ox, oy, rx, ry float64, e unit.Snapshot) bool {
	px, py := e.X-ox, e.Y-oy
	ax, ay := rx+e.Radius, ry+e.Radius
	if ax < 1e-6 || ay < 1e-6 {
		return false
	}
	return (px/ax)*(px/ax)+(py/ay)*(py/ay) <= 1+1e-6
}

func laserHits(x, y, ux, uy, half, length float64, e unit.Snapshot) bool {
	vx, vy := e.X-x, e.Y-y
	along := vx*ux + vy*uy
	if along < -e.Radius || along > length {
		return false
	}
	side := math.Abs(vx*uy - vy*ux)
	return side <= half+e.Radius
}

func segHits(x1, y1, x2, y2, half float64, e unit.Snapshot) bool {
	dx, dy := x2-x1, y2-y1
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return math.Hypot(e.X-x1, e.Y-y1) <= half+e.Radius
	}
	return laserHits(x1, y1, dx/n, dy/n, half, n, e)
}

func clampHex(x, y, radius float64) (float64, float64) {
	if unit.HexContains(x, y, radius) {
		return x, y
	}
	n := math.Hypot(x, y)
	if n < 1e-6 {
		return 0, 0
	}
	limit := unit.HexRadius - radius - 4
	if limit < 8 {
		limit = 8
	}
	s := limit / n
	return x * s, y * s
}
