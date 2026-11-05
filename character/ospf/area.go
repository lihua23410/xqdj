package ospf

import (
	"math"
	"sync"

	"xqdj/internal/unit"
	奇美拉 "xqdj/map/chimera"
	_ "xqdj/map/hex"
)

type areaMap struct {
	chimera       bool
	radius, angle float64
	areas         []int
}

type matchAreas struct {
	once   sync.Once
	layout areaMap
}

func areasFor(info unit.SpawnInfo) areaMap {
	state := unit.StateFor[matchAreas](info.State)
	state.once.Do(func() {
		state.layout = cutAreas(info.Field, info.MatchSeed)
	})
	return state.layout
}

func cutAreas(field unit.Field, seed uint64) areaMap {
	const areaStream = 0x4f535046
	rng := unit.NewRNG(unit.DeriveStream(seed, 0, areaStream))
	if field.Shape == unit.ShapeChimera {
		return areaMap{chimera: true, areas: rng.Perm(4)}
	}

	// These centered regular outlines have a constant supporting distance.
	// The outward normal projects any point on that edge onto its apothem.
	x, y, nx, ny := field.SampleEdge(0, 0)
	return areaMap{radius: (x*nx + y*ny) / 2, angle: rng.Float64() * 2 * math.Pi, areas: rng.Perm(3)}
}

func (m areaMap) at(x, y float64) int {
	if m.chimera {
		if room := 奇美拉.PeripheralRoom(x, y); room >= 0 {
			return m.areas[room]
		}
		return 0
	}
	if math.Hypot(x, y) <= m.radius+1e-9 {
		return 0
	}
	sector := (math.Atan2(y, x) - m.angle) / (2 * math.Pi / 3)
	// Round off trig error on the cut rays before applying the half-open rule.
	if nearest := math.Round(sector); math.Abs(sector-nearest) < 1e-12 {
		sector = nearest
	}
	return 1 + m.areas[(int(math.Floor(sector))+6)%3]
}
