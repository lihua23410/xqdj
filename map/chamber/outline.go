package 机关房

import (
	"math"

	"xqdj/internal/unit"
)

func init() {
	unit.RegisterOutline(unit.ShapeSquare, newSquareOutline)
}

type squareOutline struct {
	half float64 // 场心到边
}

func newSquareOutline(extent float64) unit.Outline {
	// 与九宫一致：以 extent 为外接六边形外接圆半径时的最大内接轴对齐正方形。
	return &squareOutline{half: extent * (3 - math.Sqrt(3)) / 2}
}

func (s *squareOutline) Contains(x, y, radius float64) bool {
	limit := s.half - radius
	return math.Abs(x) <= limit+1e-6 && math.Abs(y) <= limit+1e-6
}

func (s *squareOutline) SampleEdge(t, inset float64) (x, y, nx, ny float64) {
	side := int(t * 4) % 4
	frac := t*4 - math.Floor(t*4)
	r := s.half - inset
	if r < 0 {
		r = 0
	}
	switch side {
	case 0:
		return r, -r + 2*r*frac, 1, 0
	case 1:
		return r - 2*r*frac, r, 0, 1
	case 2:
		return -r, r - 2*r*frac, -1, 0
	default:
		return -r + 2*r*frac, -r, 0, -1
	}
}

func (s *squareOutline) NearestEdgeDir(x, y float64) (dx, dy float64) {
	ax, ay := math.Abs(x), math.Abs(y)
	if ax >= ay {
		if x >= 0 {
			return 1, 0
		}
		return -1, 0
	}
	if y >= 0 {
		return 0, 1
	}
	return 0, -1
}

func (s *squareOutline) Sweep(px, py, vx, vy, faceX, faceY, radius, dt float64, semi bool) (t, nx, ny float64, hit bool) {
	_ = faceX
	_ = faceY
	_ = semi
	limit := s.half - radius
	if limit < 8 {
		limit = 8
	}
	if math.Abs(px) > limit+1e-6 || math.Abs(py) > limit+1e-6 {
		dx, dy := s.NearestEdgeDir(px, py)
		return 0, dx, dy, true
	}
	bestT := dt + 1
	var bestNx, bestNy float64
	found := false
	type face struct {
		limit, comp, vcomp, nnx, nny float64
	}
	faces := []face{
		{limit, px, vx, 1, 0},
		{limit, -px, -vx, -1, 0},
		{limit, py, vy, 0, 1},
		{limit, -py, -vy, 0, -1},
	}
	for _, f := range faces {
		if f.vcomp <= 1e-9 {
			continue
		}
		tt := (f.limit - f.comp) / f.vcomp
		if tt < -1e-9 || tt > dt {
			continue
		}
		if tt < 0 {
			tt = 0
		}
		if tt < bestT {
			bestT = tt
			bestNx, bestNy = f.nnx, f.nny
			found = true
		}
	}
	return bestT, bestNx, bestNy, found
}

func (s *squareOutline) Constrain(x, y, radius, hintNx, hintNy, faceX, faceY, skin float64, semi bool) (ox, oy float64) {
	_ = hintNx
	_ = hintNy
	_ = faceX
	_ = faceY
	_ = semi
	limit := s.half - radius - skin
	if limit < 8 {
		limit = 8
	}
	ox, oy = x, y
	if ox > limit {
		ox = limit
	} else if ox < -limit {
		ox = -limit
	}
	if oy > limit {
		oy = limit
	} else if oy < -limit {
		oy = -limit
	}
	return ox, oy
}
