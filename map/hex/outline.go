package 六边形

import (
	"math"

	"xqdj/internal/unit"
)

func init() {
	unit.RegisterOutline(unit.ShapeHex, newHexOutline)
}

type hexOutline struct {
	nx, ny [6]float64
	apo    float64
	R      float64
}

func newHexOutline(extent float64) unit.Outline {
	h := &hexOutline{R: extent, apo: extent * math.Sqrt(3) / 2}
	for i := 0; i < 6; i++ {
		a := (float64(i) + 0.5) * math.Pi / 3
		h.nx[i] = math.Cos(a)
		h.ny[i] = math.Sin(a)
	}
	return h
}

func (h *hexOutline) Contains(x, y, radius float64) bool {
	limit := h.apo - radius
	for i := 0; i < 6; i++ {
		if h.nx[i]*x+h.ny[i]*y > limit+1e-6 {
			return false
		}
	}
	return true
}

func (h *hexOutline) SampleEdge(t, inset float64) (x, y, nx, ny float64) {
	tt := t * 6
	side := int(tt) % 6
	frac := tt - math.Floor(tt)
	a0 := float64(side) * math.Pi / 3
	a1 := float64(side+1) * math.Pi / 3
	x0, y0 := h.R*math.Cos(a0), h.R*math.Sin(a0)
	x1, y1 := h.R*math.Cos(a1), h.R*math.Sin(a1)
	nx, ny = h.nx[side], h.ny[side]
	return x0 + (x1-x0)*frac - nx*inset, y0 + (y1-y0)*frac - ny*inset, nx, ny
}

func (h *hexOutline) NearestEdgeDir(x, y float64) (dx, dy float64) {
	best := math.Inf(-1)
	for i := 0; i < 6; i++ {
		if d := h.nx[i]*x + h.ny[i]*y; d > best {
			best = d
			dx, dy = h.nx[i], h.ny[i]
		}
	}
	return dx, dy
}

func hexSemiExtent(faceX, faceY, radius, nx, ny float64) float64 {
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

func (h *hexOutline) Sweep(px, py, vx, vy, faceX, faceY, radius, dt float64, semi bool) (t, nx, ny float64, hit bool) {
	bestT := dt + 1
	var bestNx, bestNy float64
	found := false
	for i := 0; i < 6; i++ {
		nnx, nny := h.nx[i], h.ny[i]
		ext := radius
		if semi {
			ext = hexSemiExtent(faceX, faceY, radius, nnx, nny)
		}
		limit := h.apo - ext
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
		for j := 0; j < 6; j++ {
			if j == i {
				continue
			}
			extj := radius
			if semi {
				extj = hexSemiExtent(faceX, faceY, radius, h.nx[j], h.ny[j])
			}
			if h.nx[j]*ax+h.ny[j]*ay > h.apo-extj+1e-4 {
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

func (h *hexOutline) Constrain(x, y, radius, hintNx, hintNy, faceX, faceY, skin float64, semi bool) (ox, oy float64) {
	ox, oy = x, y
	if hintNx*hintNx+hintNy*hintNy > 1e-12 {
		ext := radius
		if semi {
			ext = hexSemiExtent(faceX, faceY, radius, hintNx, hintNy)
		}
		limit := h.apo - ext - skin
		pen := hintNx*ox + hintNy*oy - limit
		if pen > 0 {
			ox -= hintNx * pen
			oy -= hintNy * pen
		}
		return ox, oy
	}
	for i := 0; i < 6; i++ {
		nnx, nny := h.nx[i], h.ny[i]
		ext := radius
		if semi {
			ext = hexSemiExtent(faceX, faceY, radius, nnx, nny)
		}
		limit := h.apo - ext - skin
		pen := nnx*ox + nny*oy - limit
		if pen > 0 {
			ox -= nnx * pen
			oy -= nny * pen
		}
	}
	return ox, oy
}
