package character

import (
	"math"
	"xqdj/internal/unit"
)

const KindWaller = "筑墙者"

const (
	wallerRadius = 18.0
	wallerSpeed  = 155.0
	wallerHP     = 100.0
	wallerVision = 9999.0
	wallEvery    = 2.7
	wallLife     = 7.5
	wallLen      = 155.0
	wallRadius   = 6.0
	wallDamage   = 3.5
)

func init() {
	unit.Register(unit.Spec{
		Kind:    KindWaller,
		Role:    unit.RoleFighter,
		Radius:  wallerRadius,
		MaxHP:   wallerHP,
		Speed:   wallerSpeed,
		Vision:  wallerVision,
		Fighter: true,
		Look:    unit.Look{Color: "#c4a36a", WallGuide: wallLen},
	}, func(unit.SpawnInfo) unit.Actor {
		return &筑墙者{nextBuild: wallEvery}
	})
}

type 筑墙者 struct {
	nextBuild float64
}

func (w *筑墙者) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if s.Time+1e-9 < w.nextBuild {
		return
	}
	var enemy *unit.Snapshot
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role != unit.RoleFighter {
			continue
		}
		enemy = o
		break
	}
	if enemy == nil {
		return
	}
	dx := enemy.X - s.Self.X
	dy := enemy.Y - s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	px, py := -dy/n, dx/n
	half := wallLen / 2
	mx := (s.Self.X + enemy.X) / 2
	my := (s.Self.Y + enemy.Y) / 2
	ctx.Out <- unit.PlaceWall{
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
		Kind:    ctx.Kind,
		X1:      mx + px*half,
		Y1:      my + py*half,
		X2:      mx - px*half,
		Y2:      my - py*half,
		Radius:  wallRadius,
		Life:    wallLife,
		Amount:  wallDamage,
	}
	w.nextBuild = s.Time + wallEvery
}
