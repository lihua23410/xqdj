package 靶子

import "xqdj/internal/unit"

const KindDummy = "靶子"

func init() {
	p := unit.NewPack(KindDummy, nil)
	p.Register(unit.Spec{
		Kind:    KindDummy,
		Role:    unit.RoleFighter,
		Radius:  18,
		MaxHP:   99999,
		Speed:   165,
		Vision:  9999,
		Fighter: true,
		Look:    unit.Look{Color: "#4aa3ff"},
	}, func(unit.SpawnInfo) unit.Actor {
		return 靶子{}
	})
}

type 靶子 struct{}

func (靶子) Handle(ctx unit.Context, ev unit.Event) {
	unit.AcceptHit(ctx, ev)
}
