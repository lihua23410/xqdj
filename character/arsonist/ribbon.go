package 纵火者

import "math"

const ribbonStep = surfaceStep

func ribbonHash(seed uint64) float64 {
	seed = (seed ^ (seed >> 30)) * 0xbf58476d1ce4e5b9
	seed = (seed ^ (seed >> 27)) * 0x94d049bb133111eb
	return float64((seed^(seed>>31))>>11)/(1<<53)*2 - 1
}
func ribbonNoise(seed uint64, t float64) float64 {
	i := math.Floor(t)
	f := t - i
	f = f * f * (3 - 2*f)
	a := ribbonHash(seed + uint64(int64(i))*0x9e3779b97f4a7c15)
	b := ribbonHash(seed + uint64(int64(i)+1)*0x9e3779b97f4a7c15)
	return a*(1-f) + b*f
}

// Sample by travelled distance, not Sense count. Subdividing the same movement
// into more frames must not turn a wet ribbon into regularly spaced beads.
func (g *oilSurface) addOilRibbon(x0, y0, x1, y1, r0, r1, distance float64, seed uint64, now float64) {
	dx, dy := x1-x0, y1-y0
	length := math.Hypot(dx, dy)
	if length < 1e-6 {
		return
	}
	first := int(math.Floor((distance+1e-9)/ribbonStep)) + 1
	if distance < 1e-9 {
		first = 0
	}
	last := int(math.Floor((distance + length + 1e-9) / ribbonStep))
	if first > last {
		return
	}
	g.addOilCoverage(now, func(visit func(point)) {
		for i := first; i <= last; i++ {
			drawn := float64(i) * ribbonStep
			t := math.Max(0, math.Min(1, (drawn-distance)/length))
			base := r0 + (r1-r0)*t
			strength := math.Min(1, drawn/14)
			side := base * .11 * ribbonNoise(seed^0x6439, drawn/19) * strength
			radius := base * (1 + .16*ribbonNoise(seed, drawn/13)*strength)
			x, y := x0+dx*t-dy/length*side, y0+dy*t+dx/length*side
			rasterStroke(x, y, x, y, radius, visit)
			if base <= bottleTrailWidth+1e-9 && i%10 == 4 {
				// Occasional tiny drips accompany the fine, still-connected oil filament.
				sign := 1.0
				if ribbonHash(seed+uint64(i)) < 0 {
					sign = -1
				}
				off := (radius + surfaceStep*1.5) * sign
				px, py := x-dy/length*off, y+dx/length*off
				rasterStroke(px, py, px, py, surfaceStep*.8, visit)
			}
		}
	})
}
