package parts

import (
	"math/rand/v2"
	"xqdj/character/engine_heart/parts/api"
	"xqdj/character/engine_heart/parts/arc"
	"xqdj/character/engine_heart/parts/cage"
	"xqdj/character/engine_heart/parts/cam"
	"xqdj/character/engine_heart/parts/doublevalve"
	"xqdj/character/engine_heart/parts/flywheel"
	"xqdj/character/engine_heart/parts/laser"
	"xqdj/character/engine_heart/parts/manifold"
	"xqdj/character/engine_heart/parts/nozzle"
	"xqdj/character/engine_heart/parts/piston"
	"xqdj/character/engine_heart/parts/plate"
	"xqdj/character/engine_heart/parts/ram"
	"xqdj/character/engine_heart/parts/steam"
	"xqdj/character/engine_heart/parts/steer"
	"xqdj/character/engine_heart/parts/turbine"
	"xqdj/character/engine_heart/parts/winch"
)

var catalog = []api.Definition{piston.Definition(), manifold.Definition(), flywheel.Definition(), cam.Definition(), nozzle.Definition(), plate.Definition(), doublevalve.Definition(), turbine.Definition(), cage.Definition(), steer.Definition(), winch.Definition(), laser.Definition(), ram.Definition(), steam.Definition(), arc.Definition()}

func Catalog() []api.Definition { return append([]api.Definition(nil), catalog...) }

func IsWeapon(id int) bool {
	for _, d := range catalog {
		if d.ID == id {
			return d.Weapon != nil
		}
	}
	return false
}

// One reserved weapon plus a without-replacement draw; shuffle its field position too.
func Select(rng *rand.Rand, n int) []api.Definition {
	if n <= 0 {
		return nil
	}
	n = min(n, len(catalog))
	var weapons []int
	for i, d := range catalog {
		if d.Weapon != nil {
			weapons = append(weapons, i)
		}
	}
	chosen := weapons[rng.IntN(len(weapons))]
	result := []api.Definition{catalog[chosen]}
	for _, i := range rng.Perm(len(catalog)) {
		if i != chosen && len(result) < n {
			result = append(result, catalog[i])
		}
	}
	rng.Shuffle(len(result), func(i, j int) { result[i], result[j] = result[j], result[i] })
	return result
}
func Build(ids []int) api.Build {
	b := api.Base()
	selected := map[int]bool{}
	for _, id := range ids {
		selected[id] = true
	}
	for _, d := range catalog {
		if selected[d.ID] {
			d.Install(&b)
		}
	}
	return b
}
