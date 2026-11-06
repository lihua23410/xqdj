// Package piston owns 增压活塞: its effect and adjacent fx.js artwork.
package piston

import "xqdj/character/engine_heart/parts/api"

func Definition() api.Definition {
	return api.Definition{ID: 0, Key: "piston", Name: "增压活塞", Install: func(b *api.Build) { b.Pull *= 1.4; b.Push *= 1.4 }}
}
