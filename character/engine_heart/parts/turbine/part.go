// Package turbine owns 偏流叶轮: its effect and adjacent fx.js artwork.
package turbine

import "xqdj/character/engine_heart/parts/api"

func Definition() api.Definition {
	return api.Definition{ID: 7, Key: "turbine", Name: "偏流叶轮", Install: func(b *api.Build) {
		b.Flow = func(x, y, force float64) (float64, float64) { return (x - y*.65) * force, (y + x*.65) * force }
	}}
}
