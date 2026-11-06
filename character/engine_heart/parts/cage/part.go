// Package cage owns 捕弹阀笼: its effect and adjacent fx.js artwork.
package cage

import (
	"math"
	"xqdj/character/engine_heart/parts/api"
)

func Definition() api.Definition {
	return api.Definition{ID: 8, Key: "cage", Name: "捕弹阀笼", Install: func(b *api.Build) {
		b.Capture = true
		b.Guards = append(b.Guards, func(f api.Frame) api.Guard {
			if f.Suspended || f.Loaded || f.Phase >= .4 {
				return api.Guard{}
			}
			x, y := f.Self.VX, f.Self.VY
			if math.Hypot(x, y) < 1e-6 {
				x, y = f.Self.FaceX, f.Self.FaceY
			}
			return api.Guard{Key: "cage", Inner: 32, Outer: 37, Span: math.Pi / 2, Angle: math.Atan2(y, x), Capture: true}
		})
	}}
}
