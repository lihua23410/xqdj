// Package nozzle owns 燃油喷嘴: its effect and adjacent fx.js artwork.
package nozzle

import "xqdj/character/engine_heart/parts/api"

func Definition() api.Definition {
	return api.Definition{ID: 4, Key: "nozzle", Name: "燃油喷嘴", Install: func(b *api.Build) { b.Damage += 1.5 }}
}
