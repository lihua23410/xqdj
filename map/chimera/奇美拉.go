package 奇美拉

import (
	"embed"

	"xqdj/internal/unit"
	机关房 "xqdj/map/chamber"
	八边形 "xqdj/map/octagon"
	火山 "xqdj/map/volcano"
)

const Name = "奇美拉"

const KindChimera = "奇美拉"

//go:embed fx
var assets embed.FS

func Field() unit.Field {
	cx, _ := East()
	return unit.Field{
		Name: Name, Shape: unit.ShapeChimera, Extent: farthest(),
		BootKind: KindChimera,
		Walls: []unit.FieldWall{{
			Kind: unit.WallHard,
			X1:   cx - 110, Y1: 0,
			X2: cx + 110, Y2: 0,
			Radius: 6,
			Period: 8,
		}},
	}
}

func init() {
	unit.RegisterField(Field())
	p := unit.NewPack(KindChimera, assets)
	p.Register(unit.Spec{
		Kind: KindChimera, Role: unit.RoleHelper, Radius: 0, MaxHP: 1,
		Speed: 0, Vision: 0, Fighter: false, Nonsolid: true,
	}, func(unit.SpawnInfo) unit.Actor { return &奇美拉{} })
}

type 奇美拉 struct {
	booted bool
}

func (a *奇美拉) Handle(ctx unit.Context, ev unit.Event) {
	if _, ok := ev.(unit.Sense); !ok || a.booted {
		return
	}
	a.booted = true
	_, ny := North()
	_, sy := South()
	wx, _ := West()
	ctx.Out <- ctx.Born(unit.Spawn{Kind: 火山.KindVolcano, X: 0, Y: ny})
	ctx.Out <- ctx.Born(unit.Spawn{Kind: 机关房.KindChamber, X: 0, Y: sy})
	ctx.Out <- ctx.Born(unit.Spawn{Kind: 八边形.KindOctagon, X: wx, Y: 0})
}
