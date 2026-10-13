// 大六边形：平顶六边形外形，场心到场边原三倍默认再缩四分之一；无硬墙。
package 大六边形

import (
	"embed"

	"xqdj/internal/unit"
	_ "xqdj/map/hex" // ShapeHex 判定在六边形包
)

const Name = "大六边形"

// Extent 场心到场边最远点。要改大小只动这一处，再把 fx/widehex.css 的宽高按 2R 和 √3·R 对齐。
const Extent = 9 * unit.HexRadius / 4

//go:embed fx
var assets embed.FS

func Field() unit.Field {
	return unit.Field{Name: Name, Shape: unit.ShapeHex, Extent: Extent}
}

func init() {
	unit.RegisterField(Field())
	unit.NewPack(Name, assets)
}
