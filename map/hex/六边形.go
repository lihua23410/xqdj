package 六边形

import "xqdj/internal/unit"

const Name = "六边形"

func Field() unit.Field {
	return unit.Field{Name: Name, Shape: unit.ShapeHex, Extent: unit.HexRadius}
}

func init() {
	unit.RegisterField(Field())
}
