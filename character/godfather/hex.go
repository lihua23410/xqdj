package 教父

import (
	"math"
	"math/rand/v2"
	"xqdj/internal/unit"
)

func nearestEdgeDir(field unit.Field, x, y float64) (float64, float64) {
	return field.NearestEdgeDir(x, y)
}

func edgeKick(field unit.Field, rng *rand.Rand, radius float64) (x, y, vx, vy float64) {
	t := 0.5
	ang := 0.0
	if rng != nil {
		t = rng.Float64()
		ang = rng.Float64() * 2 * math.Pi
	}
	x, y, _, _ = field.SampleEdge(t, radius+6)
	vx, vy = math.Cos(ang)*minionCruise, math.Sin(ang)*minionCruise
	return
}
