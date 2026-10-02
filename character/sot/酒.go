package 酒翁

import (
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

const (
	wineRadius   = 12.0
	wineFall     = 220.0 // Spec 占位；真正往下走靠重力
	wineDmg      = 7.0
	wineG        = 380.0 // 向下加速度，丢坛子那种
	wineToss     = 70.0  // 出生横向初速上限
	wineDrop     = 50.0  // 出生就往下掉一点
	wineWind     = 180.0 // 下落时左右晃
	wineMax      = 520.0
	wineSpawnPad = 20.0 // 离场边再收一点，圆场贴边会当场 WallHit
)

type 酒 struct {
	owner  uint64
	slot   int
	booted bool
	dead   bool
	x, y   float64
	rng    *rand.Rand
}

func (w *酒) Handle(ctx unit.Context, ev unit.Event) {
	if w.dead {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		w.onSense(ctx, e)
	case unit.Collision:
		w.onHit(ctx, e.Other)
	case unit.WallHit:
		w.shatter(ctx, e.X, e.Y)
	}
}

func (w *酒) onSense(ctx unit.Context, s unit.Sense) {
	w.x, w.y = s.Self.X, s.Self.Y
	if !w.booted {
		w.booted = true
		w.rng = rand.New(rand.NewPCG(ctx.ID, ctx.ID))
	}
	ax := (w.rng.Float64()*2 - 1) * wineWind
	ctx.Out <- unit.Force{UnitID: ctx.ID, AX: ax, AY: -wineG}
	sp := math.Hypot(s.Self.VX, s.Self.VY)
	if sp > wineMax {
		ctx.Out <- unit.SetVelocity{
			UnitID: ctx.ID,
			VX:     s.Self.VX / sp * wineMax,
			VY:     s.Self.VY / sp * wineMax,
		}
	}
}

func (w *酒) onHit(ctx unit.Context, other unit.Snapshot) {
	if other.Slot == w.slot {
		return
	}
	if unit.Hittable(other, w.slot) {
		ctx.Out <- unit.Damage{From: ctx.ID, To: other.ID, Amount: wineDmg}
		x, y := w.x, w.y
		if !w.booted {
			x, y = other.X, other.Y
		}
		w.shatter(ctx, x, y)
		return
	}
	if other.Role == unit.RoleProjectile {
		x, y := w.x, w.y
		if !w.booted {
			x, y = other.X, other.Y
		}
		w.shatter(ctx, x, y)
		return
	}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

func (w *酒) shatter(ctx unit.Context, x, y float64) {
	if w.dead {
		return
	}
	w.dead = true
	ctx.Out <- unit.Spawn{
		Kind: KindSotStain, OwnerID: w.owner, Slot: w.slot,
		X: x, Y: y,
	}
	ctx.Out <- unit.FX{Name: "sot-shatter", Kind: KindSot, X: x, Y: y, Slot: w.slot}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}
