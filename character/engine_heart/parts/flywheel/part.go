// Package flywheel owns 惯性飞轮: its effect and adjacent fx.js artwork.
package flywheel

import "xqdj/character/engine_heart/parts/api"

func Definition() api.Definition {
	return api.Definition{ID: 2, Key: "flywheel", Name: "惯性飞轮", Install: func(b *api.Build) { b.Dash += 90 }}
}
