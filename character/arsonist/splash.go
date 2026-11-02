package 纵火者

import (
	"math"

	"xqdj/internal/unit"
)

const splashSamples = 128

type splashDrop struct{ x, y, r float64 }
type oilSplash struct {
	radius [splashSamples]float64
	reach  float64
	drops  []splashDrop
}

// A broad, connected puddle with uneven lobes and a few detached droplets.
// Normalize the total area around the old radius, instead of enlarging the attack.
func makeOilSplash(r float64, seed uint64) oilSplash {
	rng := unit.NewRNG(seed)
	var shape oilSplash
	phase := [4]float64{}
	for i := range phase {
		phase[i] = rng.Float64() * 2 * math.Pi
	}
	type finger struct{ angle, width, length float64 }
	fingers := make([]finger, 3)
	for i := range fingers {
		fingers[i] = finger{rng.Float64() * 2 * math.Pi, .14 + rng.Float64()*.2, .14 + rng.Float64()*.16}
	}
	area := 0.0
	for i := range shape.radius {
		a := float64(i) * 2 * math.Pi / splashSamples
		v := 1 + .10*math.Sin(2*a+phase[0]) + .075*math.Sin(3*a+phase[1]) + .05*math.Sin(5*a+phase[2]) + .025*math.Sin(8*a+phase[3])
		for _, f := range fingers {
			d := math.Remainder(a-f.angle, 2*math.Pi) / f.width
			v += f.length * math.Exp(-d*d*.5)
		}
		v = math.Min(1.4, v)
		shape.radius[i] = v
		area += v * v
	}
	dropArea := 0.0
	for i := 0; i < 3; i++ {
		dr := r * (.045 + .035*rng.Float64())
		shape.drops = append(shape.drops, splashDrop{r: dr})
		dropArea += dr * dr
	}
	scale := math.Sqrt((r*r - dropArea) / (area / splashSamples))
	for i := range shape.radius {
		shape.radius[i] *= scale
		shape.reach = math.Max(shape.reach, shape.radius[i])
	}
	for i := range shape.drops {
		a := fingers[i].angle + (.5-rng.Float64())*.3
		d := &shape.drops[i]
		reach := math.Min(shape.at(a)+d.r+surfaceStep*(1+rng.Float64()), r*1.5-d.r)
		d.x, d.y = math.Cos(a)*reach, math.Sin(a)*reach
	}
	return shape
}
func (s *oilSplash) at(angle float64) float64 {
	if angle < 0 {
		angle += 2 * math.Pi
	}
	f := angle * splashSamples / (2 * math.Pi)
	i := int(f) % splashSamples
	t := f - math.Floor(f)
	return s.radius[i]*(1-t) + s.radius[(i+1)%splashSamples]*t
}
func rasterSplash(x, y, r float64, seed uint64, visit func(point)) {
	shape := makeOilSplash(r, seed)
	for gy := gridCoord(y - shape.reach); gy <= gridCoord(y+shape.reach); gy++ {
		for gx := gridCoord(x - shape.reach); gx <= gridCoord(x+shape.reach); gx++ {
			dx, dy := (float64(gx)+.5)*surfaceStep-x, (float64(gy)+.5)*surfaceStep-y
			radius := shape.at(math.Atan2(dy, dx))
			if dx*dx+dy*dy <= radius*radius {
				visit(point{gx, gy})
			}
		}
	}
	for _, d := range shape.drops {
		rasterStroke(x+d.x, y+d.y, x+d.x, y+d.y, d.r, visit)
	}
}
func (g *oilSurface) addOilSplash(x, y, r float64, seed uint64, now float64) {
	g.addOilCoverage(now, func(visit func(point)) { rasterSplash(x, y, r, seed, visit) })
}
