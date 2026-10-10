package 圆

import (
	"embed"

	"xqdj/internal/unit"
)

const Name = "圆"

//go:embed fx
var assets embed.FS

func Field() unit.Field {
	return unit.Field{
		Name:   Name,
		Shape:  unit.ShapeCircle,
		Extent: unit.HexRadius,
		Walls: []unit.FieldWall{{
			Kind: unit.WallHard,
			X1:   -110, Y1: 0,
			X2: 110, Y2: 0,
			Radius: 6,
			Period: 8,
		}},
	}
}

func init() {
	unit.RegisterField(Field())
	unit.NewPack(Name, assets)
}
