// Package plate owns 护心夹板: its effect and adjacent fx.js artwork.
package plate

import (
	"math"
	"xqdj/character/engine_heart/parts/api"
)

func Definition() api.Definition {
	return api.Definition{ID: 5, Key: "plate", Name: "护心夹板", Install: func(b *api.Build) {
		b.Guards = append(b.Guards, func(f api.Frame) api.Guard {
			return api.Guard{Key: "plate", Inner: 25, Outer: 31, Span: math.Pi * .6, Angle: f.Time * math.Pi / 6}
		})
	}}
}
