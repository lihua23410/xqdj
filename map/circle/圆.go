package 圆

import "xqdj/internal/unit"

const Name = "圆"

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
}
