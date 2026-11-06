// Package winch owns 回收绞盘: its effect and adjacent fx.js artwork.
package winch

import "xqdj/character/engine_heart/parts/api"

func Definition() api.Definition {
	return api.Definition{ID: 10, Key: "winch", Name: "回收绞盘", Install: func(b *api.Build) { b.WinchSpeed = 110 }}
}
