// 八边形：截角方形轮廓与三段挡视线硬墙全在本包。
package 八边形

import (
	"embed"
	"math"

	"xqdj/internal/unit"
)

const (
	Name = "八边形"

	KindOctagon = "八边形"

	wallDist = 110.0 // 场心到墙心
	wallLen  = 60.0  // 墙长（两轴点间距）
	wallR    = 8.0   // 墙粗（胶囊半径，方端即半宽）

	lookColor = "#454b55"
)

//go:embed fx
var assets embed.FS

func Field() unit.Field {
	return unit.Field{
		Name: Name, Shape: unit.ShapeOctagon, Extent: unit.HexRadius,
		BootKind: KindOctagon,
	}
}

func init() {
	unit.RegisterField(Field())

	p := unit.NewPack(KindOctagon, assets)
	// 场地控制器：开局把三段挡视线硬墙砌到位，之后无事可做。
	p.Register(unit.Spec{
		Kind: KindOctagon, Role: unit.RoleHelper, Radius: 1, MaxHP: 1,
		Speed: 0, Vision: 9999, Fighter: false, Nonsolid: true,
		Look: unit.Look{Color: lookColor},
	}, func(unit.SpawnInfo) unit.Actor {
		return &八边形{}
	})
}

type 八边形 struct {
	booted bool
}

func (a *八边形) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok || a.booted {
		return
	}
	a.booted = true
	for i := 0; i < 3; i++ {
		ang := math.Pi/2 + float64(i)*2*math.Pi/3 // 90°、210°、330°：绕场心中心对称
		cx, cy := wallDist*math.Cos(ang), wallDist*math.Sin(ang)
		tx, ty := -math.Sin(ang), math.Cos(ang) // 切向：墙面正对场心
		half := wallLen / 2
		ctx.Out <- unit.PlaceWall{
			OwnerID: ctx.ID, Slot: s.Self.Slot, Kind: KindOctagon,
			X1: cx - tx*half, Y1: cy - ty*half,
			X2: cx + tx*half, Y2: cy + ty*half,
			Radius: wallR, Life: 0, Hard: true,
			VisionBlock: true,
		}
	}
}
