// 大六边形：平顶六边形外形，场心到场边三倍默认六边形；无硬墙。
package 大六边形

import (
	"embed"

	"xqdj/internal/unit"
	_ "xqdj/map/hex" // ShapeHex 判定在六边形包
)

const Name = "大六边形"

// Extent 场心到场边最远点。三倍默认六边形；单区 240–320 的例外，不缩回去。
const Extent = 3 * unit.HexRadius

//go:embed fx
var assets embed.FS

func Field() unit.Field {
	return unit.Field{Name: Name, Shape: unit.ShapeHex, Extent: Extent}
}

func init() {
	unit.RegisterField(Field())
	unit.NewPack(Name, assets)
}
