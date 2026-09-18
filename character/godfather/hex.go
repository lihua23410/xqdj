package 教父

import (
	"math"
	"math/rand/v2"
	"xqdj/internal/unit"
)

func hexVertex(i int) (float64, float64) {
	a := float64(i) * math.Pi / 3
	return unit.HexRadius * math.Cos(a), unit.HexRadius * math.Sin(a)
}

func hexNormal(i int) (float64, float64) {
	a := (float64(i) + 0.5) * math.Pi / 3
	return math.Cos(a), math.Sin(a)
}

func hexRim(t, inset float64) (float64, float64) {
	for t < 0 {
		t += 6
	}
	side := int(t) % 6
	f := t - math.Floor(t)
	x0, y0 := hexVertex(side)
	x1, y1 := hexVertex(side + 1)
	nx, ny := hexNormal(side)
	return x0 + (x1-x0)*f - nx*inset, y0 + (y1-y0)*f - ny*inset
}

func nearestEdgeDir(x, y float64) (float64, float64) {
	best := math.Inf(-1)
	var bx, by float64
	for i := 0; i < 6; i++ {
		nx, ny := hexNormal(i)
		d := nx*x + ny*y
		if d > best {
			best = d
			bx, by = nx, ny
		}
	}
	return bx, by
}

func edgeKick(rng *rand.Rand, radius float64) (x, y, vx, vy float64) {
	side := 0
	f := 0.5
	ang := 0.0
	if rng != nil {
		side = rng.IntN(6)
		f = rng.Float64()
		ang = rng.Float64() * 2 * math.Pi
	}
	x, y = hexRim(float64(side)+f, radius+6)
	vx, vy = math.Cos(ang)*minionCruise, math.Sin(ang)*minionCruise
	return
}
