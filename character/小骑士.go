package character

import (
	"math"
	"math/rand/v2"
	"xqdj/internal/unit"
)

const KindKnight = "小骑士"

const (
	knightRadius   = 18.0
	knightSpeed    = 175.0
	knightHP       = 100.0
	knightStartHP  = 75.0
	knightVision   = 9999.0
	knightDamage   = 10.0
	knightHitCD    = 0.1
	knightCD       = 7.0
	knightChainGap = 0.5
	ramSpeed       = 340.0
	hexCircum      = 280.0
)

func init() {
	unit.Register(unit.Spec{
		Kind:    KindKnight,
		Role:    unit.RoleFighter,
		Radius:  knightRadius,
		MaxHP:   knightHP,
		StartHP: knightStartHP,
		Speed:   knightSpeed,
		Vision:  knightVision,
		Fighter: true,
	}, func(unit.SpawnInfo) unit.Actor {
		return &小骑士{readyAt: knightCD}
	})
}

type 小骑士 struct {
	readyAt    float64
	hitReadyAt float64
}

func (k *小骑士) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		k.tryRam(ctx, e)
	case unit.Collision:
		k.hit(ctx, e)
	}
}

func (k *小骑士) hit(ctx unit.Context, e unit.Collision) {
	if e.Other.Role != unit.RoleFighter {
		return
	}
	if e.Time < k.hitReadyAt {
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: knightDamage}
	k.hitReadyAt = e.Time + knightHitCD
}

func (k *小骑士) tryRam(ctx unit.Context, sense unit.Sense) {
	if sense.Time+1e-9 < k.readyAt {
		return
	}
	var enemy *unit.Snapshot
	for i := range sense.Nearby {
		o := &sense.Nearby[i]
		if o.Role != unit.RoleFighter || o.Slot == sense.Self.Slot {
			continue
		}
		enemy = o
		break
	}
	if enemy == nil {
		return
	}
	tx, ty := beside(sense.Self, *enemy)
	ctx.Out <- unit.FX{
		Name: "swap", Kind: ctx.Kind,
		X: sense.Self.X, Y: sense.Self.Y, Slot: sense.Self.Slot,
	}
	ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: tx, Y: ty}
	dx := enemy.X - tx
	dy := enemy.Y - ty
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		dx, dy, n = 1, 0, 1
	}
	vx, vy := dx/n*ramSpeed, dy/n*ramSpeed
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: vx, VY: vy}
	ctx.Out <- unit.FX{
		Name: "dash", Kind: ctx.Kind,
		X: tx, Y: ty, VX: vx, VY: vy, Slot: sense.Self.Slot,
	}
	if rand.Float64() < 0.5 {
		k.readyAt = sense.Time + knightChainGap
	} else {
		k.readyAt = sense.Time + knightCD
	}
}

func beside(self, enemy unit.Snapshot) (float64, float64) {
	gap := self.Radius + enemy.Radius + 8
	ex, ey := enemy.X, enemy.Y
	base := math.Atan2(ey, ex)
	for _, extra := range []float64{math.Pi / 2, -math.Pi / 2, math.Pi / 3, -math.Pi / 3, 2 * math.Pi / 3, -2 * math.Pi / 3, math.Pi} {
		ang := base + extra
		x := ex + math.Cos(ang)*gap
		y := ey + math.Sin(ang)*gap
		if insideHex(x, y, self.Radius+4) {
			return x, y
		}
	}
	return 0, 0
}

func insideHex(x, y, radius float64) bool {
	ap := hexCircum * math.Sqrt(3) / 2
	limit := ap - radius
	for i := 0; i < 6; i++ {
		a := (float64(i) + 0.5) * math.Pi / 3
		if math.Cos(a)*x+math.Sin(a)*y > limit+1e-6 {
			return false
		}
	}
	return true
}
