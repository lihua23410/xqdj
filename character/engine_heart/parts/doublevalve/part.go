// Package doublevalve owns 双联阀组: its effect and adjacent fx.js artwork.
package doublevalve

import "xqdj/character/engine_heart/parts/api"

func Definition() api.Definition {
	return api.Definition{ID: 6, Key: "doublevalve", Name: "双联阀组", Install: func(b *api.Build) { b.Echo = true }}
}
