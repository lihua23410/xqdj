// Package manifold owns 分流歧管: its effect and adjacent fx.js artwork.
package manifold

import "xqdj/character/engine_heart/parts/api"

func Definition() api.Definition {
	return api.Definition{ID: 1, Key: "manifold", Name: "分流歧管", Install: func(b *api.Build) { b.Range += 24 }}
}
