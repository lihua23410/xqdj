package 血契

import (
	"math"

	"xqdj/internal/unit"
)

// 血契蝠 烧 6 血召出的活随从（一次一只、场上至多 3 只）。
// 绕在最近的敌人周围；每到攻击间隔就朝它穿一次，穿过时咬一口，穿完回到环上。
// 咬的伤打在血线连着的目标身上就按那条线回流。6 秒寿终。
type 血契蝠 struct {
	owner uint64
	slot  int

	bornAt   float64
	nextBite float64
	nextSeek float64
	targetID uint64
	lastT    float64
	booted   bool
	hold     unit.DetourHold

	dashing  bool
	dashLeft float64
	dashX    float64
	dashY    float64
	dashID   uint64
	struck   bool
}

func (bat *血契蝠) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		bat.onSense(ctx, e)
	case unit.Collision:
		bat.strike(ctx, e)
	case unit.WallHit:
		if bat.dashing {
			bat.dashing = false
		}
	}
}

func (bat *血契蝠) onSense(ctx unit.Context, s unit.Sense) {
	// Pass：不跟单位顶牛。穿过那一下靠重叠咬，不靠把人推开。
	if !bat.booted {
		bat.booted = true
		bat.bornAt = s.Time
		bat.nextBite = s.Time
		bat.nextSeek = s.Time
		bat.lastT = s.Time
		ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
		return
	}
	if s.Time-bat.bornAt >= batLife {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	dt := s.Time - bat.lastT
	if dt < 0 {
		dt = 0
	}
	if dt > 0.2 {
		dt = 0.2
	}
	bat.lastT = s.Time
	if bat.dashing {
		bat.dashLeft -= batDash * dt
		if bat.dashLeft <= 0 {
			bat.dashing = false
		} else {
			ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: bat.dashX * batDash, VY: bat.dashY * batDash}
			return
		}
	}
	bat.hunt(ctx, s)
}

// hunt 没在穿的时候：冷却好了就朝当前目标穿过去，否则停在它周围的环位上。
func (bat *血契蝠) hunt(ctx unit.Context, s unit.Sense) {
	if s.Time+1e-9 >= bat.nextSeek {
		bat.nextSeek = s.Time + batSeekEvery
		bat.targetID = 0
	}
	tgt := bat.locked(s)
	if tgt == nil {
		return
	}
	if s.Time+1e-9 >= bat.nextBite {
		bat.startDash(ctx, s, tgt)
		return
	}
	bat.orbit(ctx, s, tgt)
}

func (bat *血契蝠) locked(s unit.Sense) *unit.Snapshot {
	if bat.targetID != 0 {
		tgt := findNearby(s.Nearby, bat.targetID)
		if tgt != nil && unit.Hittable(*tgt, bat.slot) {
			return tgt
		}
		bat.targetID = 0
	}
	bestD := math.MaxFloat64
	var tgt *unit.Snapshot
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, bat.slot) {
			continue
		}
		d := math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y)
		if d < bestD {
			bestD, tgt = d, o
		}
	}
	if tgt != nil {
		bat.targetID = tgt.ID
	}
	return tgt
}

// startDash 方向锁死朝目标圆心，距离够穿到对面环上。咬在重叠时结算一次。
func (bat *血契蝠) startDash(ctx unit.Context, s unit.Sense, tgt *unit.Snapshot) {
	dx, dy := tgt.X-s.Self.X, tgt.Y-s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		dx, dy, n = 1, 0, 1
	}
	bat.dashing = true
	bat.struck = false
	bat.dashID = tgt.ID
	bat.dashX, bat.dashY = dx/n, dy/n
	bat.dashLeft = n + tgt.Radius + s.Self.Radius + batRing
	bat.nextBite = s.Time + batBiteCD
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: bat.dashX * batDash, VY: bat.dashY * batDash}
}

// orbit 停在目标球面外的固定角上。三只蝙蝠各占 120°。直线被墙挡住就绕墙端。
func (bat *血契蝠) orbit(ctx unit.Context, s unit.Sense, tgt *unit.Snapshot) {
	ang := float64(s.Self.ID%3) * 2 * math.Pi / 3
	r := tgt.Radius + s.Self.Radius + batRing
	tx := tgt.X + math.Cos(ang)*r
	ty := tgt.Y + math.Sin(ang)*r
	wx, wy, hold := unit.Detour(s.Self.X, s.Self.Y, s.Self.Radius, tx, ty, s.Walls, bat.hold, s.Time)
	bat.hold = hold
	dx, dy := wx-s.Self.X, wy-s.Self.Y
	gd := math.Hypot(tx-s.Self.X, ty-s.Self.Y)
	vx, vy := 0.0, 0.0
	if d := math.Hypot(dx, dy); d > 1e-6 {
		sp := math.Min(batSpeed, gd*6)
		if sp < 20 {
			sp = 0
		}
		vx, vy = dx/d*sp, dy/d*sp
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: vx, VY: vy}
}

func (bat *血契蝠) strike(ctx unit.Context, e unit.Collision) {
	if !bat.dashing || bat.struck || e.Other.ID != bat.dashID {
		return
	}
	if !unit.Hittable(e.Other, bat.slot) {
		return
	}
	bat.struck = true
	ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: batDamage}
	ctx.Out <- unit.FX{
		Name: "blood-bat-bite", Kind: KindBloodBat, UnitID: ctx.ID, Slot: bat.slot,
		X: e.Other.X, Y: e.Other.Y,
	}
}
