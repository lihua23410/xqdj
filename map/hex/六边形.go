package 六边形

import (
	"embed"

	"xqdj/internal/unit"
)

const Name = "六边形"

//go:embed fx
var assets embed.FS

func Field() unit.Field {
	return unit.Field{Name: Name, Shape: unit.ShapeHex, Extent: unit.HexRadius}
}

func init() {
	unit.RegisterField(Field())
	unit.NewPack(Name, assets)
}
