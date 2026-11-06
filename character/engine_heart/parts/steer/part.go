// Package steer owns 换向喷口: its effect and adjacent fx.js artwork.
package steer

import (
	"math"
	"xqdj/character/engine_heart/parts/api"
)

func Definition() api.Definition {
	return api.Definition{ID: 9, Key: "steer", Name: "换向喷口", Install: func(b *api.Build) {
		b.Steer = func(x, y float64, beat int) (float64, float64) {
			a := .48
			if beat%2 == 0 {
				a = -a
			}
			return x*math.Cos(a) - y*math.Sin(a), x*math.Sin(a) + y*math.Cos(a)
		}
	}}
}
