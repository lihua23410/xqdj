// Package cam owns 配气凸轮: its effect and adjacent fx.js artwork.
package cam

import "xqdj/character/engine_heart/parts/api"

func Definition() api.Definition {
	return api.Definition{ID: 3, Key: "cam", Name: "配气凸轮", Install: func(b *api.Build) { b.Period *= .8 }}
}
