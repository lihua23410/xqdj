package 八边形

import (
	"math"

	"xqdj/internal/unit"
)

func init() {
	unit.RegisterOutline(unit.ShapeOctagon, newOctagonOutline)
}

// octagonOutline：截角方形朝向的正八边形。边法线在 k·45°（上下左右四条边正放），
// 顶点在 22.5°+k·45°。Extent 是顶点圆半径；场心到边 = Extent·cos(22.5°)。
type octagonOutline struct {
	nx, ny [8]float64
	apo    float64
	R      float64
}

func newOctagonOutline(extent float64) unit.Outline {
	o := &octagonOutline{R: extent, apo: extent * math.Cos(math.Pi/8)}
	for i := 0; i < 8; i++ {
		a := float64(i) * math.Pi / 4
		o.nx[i] = math.Cos(a)
		o.ny[i] = math.Sin(a)
	}
	return o
}

func (o *octagonOutline) Contains(x, y, radius float64) bool {
	limit := o.apo - radius
	for i := 0; i < 8; i++ {
		if o.nx[i]*x+o.ny[i]*y > limit+1e-6 {
			return false
		}
	}
	return true
}

func (o *octagonOutline) SampleEdge(t, inset float64) (x, y, nx, ny float64) {
	tt := t * 8
	side := int(tt) % 8
	frac := tt - math.Floor(tt)
	// 边 side 的两个端点是顶点 side·45° ± 22.5°。
	a0 := float64(side)*math.Pi/4 - math.Pi/8
	a1 := a0 + math.Pi/4
	x0, y0 := o.R*math.Cos(a0), o.R*math.Sin(a0)
	x1, y1 := o.R*math.Cos(a1), o.R*math.Sin(a1)
	nx, ny = o.nx[side], o.ny[side]
	return x0 + (x1-x0)*frac - nx*inset, y0 + (y1-y0)*frac - ny*inset, nx, ny
}

func (o *octagonOutline) NearestEdgeDir(x, y float64) (dx, dy float64) {
	best := math.Inf(-1)
	for i := 0; i < 8; i++ {
		if d := o.nx[i]*x + o.ny[i]*y; d > best {
			best = d
			dx, dy = o.nx[i], o.ny[i]
		}
	}
	return dx, dy
}

func octSemiExtent(faceX, faceY, radius, nx, ny float64) float64 {
	fl := math.Hypot(faceX, faceY)
	if fl < 1e-12 {
		return radius
	}
	fx, fy := faceX/fl, faceY/fl
	if nx*fx+ny*fy >= -1e-9 {
		return radius
	}
	// |perp(face)·n| * radius
	return radius * math.Abs(-fy*nx + fx*ny)
}

func (o *octagonOutline) Sweep(px, py, vx, vy, faceX, faceY, radius, dt float64, semi bool) (t, nx, ny float64, hit bool) {
	bestT := dt + 1
	var bestNx, bestNy float64
	found := false
	for i := 0; i < 8; i++ {
		nnx, nny := o.nx[i], o.ny[i]
		ext := radius
		if semi {
			ext = octSemiExtent(faceX, faceY, radius, nnx, nny)
		}
		limit := o.apo - ext
		vn := nnx*vx + nny*vy
		if vn <= 1e-9 {
			continue
		}
		dist := limit - (nnx*px + nny*py)
		tt := dist / vn
		if tt < -1e-9 || tt > dt {
			continue
		}
		if tt < 0 {
			tt = 0
		}
		ax, ay := px+vx*tt, py+vy*tt
		ok := true
		for j := 0; j < 8; j++ {
			if j == i {
				continue
			}
			extj := radius
			if semi {
				extj = octSemiExtent(faceX, faceY, radius, o.nx[j], o.ny[j])
			}
			if o.nx[j]*ax+o.ny[j]*ay > o.apo-extj+1e-4 {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		if tt < bestT {
			bestT = tt
			bestNx, bestNy = nnx, nny
			found = true
		}
	}
	return bestT, bestNx, bestNy, found
}

func (o *octagonOutline) Constrain(x, y, radius, hintNx, hintNy, faceX, faceY, skin float64, semi bool) (ox, oy float64) {
	ox, oy = x, y
	if hintNx*hintNx+hintNy*hintNy > 1e-12 {
		ext := radius
		if semi {
			ext = octSemiExtent(faceX, faceY, radius, hintNx, hintNy)
		}
		limit := o.apo - ext - skin
		pen := hintNx*ox + hintNy*oy - limit
		if pen > 0 {
			ox -= hintNx * pen
			oy -= hintNy * pen
		}
		return ox, oy
	}
	for i := 0; i < 8; i++ {
		nnx, nny := o.nx[i], o.ny[i]
		ext := radius
		if semi {
			ext = octSemiExtent(faceX, faceY, radius, nnx, nny)
		}
		limit := o.apo - ext - skin
		pen := nnx*ox + nny*oy - limit
		if pen > 0 {
			ox -= nnx * pen
			oy -= nny * pen
		}
	}
	return ox, oy
}
