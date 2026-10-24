// 巡回朝随机方向丢出标识物后站定。标识物一路加速、撞墙反弹，
// 直到撞中敌方才折返；折返一路继续加速、继续撞人出伤，回到巡回身边消失。
// 巡回收回标识物后恢复巡航，5 秒后再丢。
package 巡回

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

const KindPatrol = "巡回"
const KindPatrolMarker = "巡回标识物"

const (
	patrolRadius = 18.0
	patrolHP     = 100.0
	patrolCruise = 155.0
	patrolColor  = "#4a7fb5"

	markRadius = 11.0
	markVision = 9999.0
	markGap    = 2.0
	markColor  = "#ffab2e"

	firstThrowDelay = 0.8
	throwCD         = 5.0
	markStartSpeed  = 300.0
	markAccel       = 320.0
	// 去程兜底：一直没撞到人也折返，别让巡回站成雕像。
	markTimeout = 10.0
	// 回程兜底：被墙反复别住时认丢这一枚，主人照样恢复。
	returnTimeout = 6.0
	// 伤害随撞击时速走：飞得越久砸得越重（反重力弹先例）。
	dmgPerSpeed = 0.014
	dmgMin      = 8.0
	dmgMax      = 26.0
	hitPaceCD   = 0.25
	bounceFXCD  = 0.15
	catchPad    = 2.0

	markOutKind = "巡回标识物在外"
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindPatrol, assets)
	p.Register(unit.Spec{
		Kind:    KindPatrol,
		Role:    unit.RoleFighter,
		Radius:  patrolRadius,
		MaxHP:   patrolHP,
		Speed:   patrolCruise,
		Vision:  0,
		Fighter: true,
		Look:    unit.Look{Color: patrolColor, FX: []string{"patrol"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &巡回{slot: info.Slot, rng: unit.NewRNG(info.Stream)}
	})
	p.Register(unit.Spec{
		Kind:    KindPatrolMarker,
		Role:    unit.RoleProjectile,
		Radius:  markRadius,
		MaxHP:   1,
		Speed:   0,
		Vision:  markVision,
		Fighter: false,
		Look:    unit.Look{Color: markColor, Glow: true, Trail: true, Overlay: true, FX: []string{"patrolmark"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &标识物{owner: info.OwnerID, slot: info.Slot, speed: markStartSpeed}
	})
}

// 巡回本体：视野 0 也看得见自己的标识物（随从始终进感知），
// 所以靠轮询 Nearby 判断标识物是否已经回来。
type 巡回 struct {
	slot        int
	rng         *rand.Rand
	waiting     bool
	nextThrowAt float64
}

func (a *巡回) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		a.onSense(ctx, e)
	}
}

func (a *巡回) onSense(ctx unit.Context, s unit.Sense) {
	if a.waiting {
		if a.markerAlive(ctx, s) {
			return
		}
		a.resume(ctx, s)
		return
	}
	if !a.thrownOnce(s) {
		return
	}
	if s.Time >= a.nextThrowAt {
		a.throw(ctx, s)
	}
}

// thrownOnce 首拍只记时刻不出手：nextThrowAt 初始为 0，直接比会开局即丢。
func (a *巡回) thrownOnce(s unit.Sense) bool {
	if a.nextThrowAt == 0 {
		a.nextThrowAt = s.Time + firstThrowDelay
		return false
	}
	return true
}

func (a *巡回) markerAlive(ctx unit.Context, s unit.Sense) bool {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind == KindPatrolMarker && o.OwnerID == ctx.ID {
			return true
		}
	}
	return false
}

func (a *巡回) throw(ctx unit.Context, s unit.Sense) {
	ang := a.rng.Float64() * 2 * math.Pi
	ux, uy := math.Cos(ang), math.Sin(ang)
	d := patrolRadius + markRadius + markGap
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind:    KindPatrolMarker,
		X:       s.Self.X + ux*d,
		Y:       s.Self.Y + uy*d,
		VX:      ux * markStartSpeed,
		VY:      uy * markStartSpeed,
		OwnerID: ctx.ID,
		Slot:    a.slot,
	})
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	ctx.Out <- unit.StackMark{UnitID: ctx.ID, Kind: markOutKind, Delta: 1}
	// 出手特效挂标识物的 Kind：流光是标识物出膛，用琥珀色而不是本体的钢蓝。
	ctx.Out <- unit.FX{Name: "patrol-throw", Kind: KindPatrolMarker, UnitID: ctx.ID, Slot: a.slot,
		X: s.Self.X, Y: s.Self.Y, VX: ux, VY: uy}
	a.waiting = true
}

func (a *巡回) resume(ctx unit.Context, s unit.Sense) {
	a.waiting = false
	a.nextThrowAt = s.Time + throwCD
	ctx.Out <- unit.ClearMarks{UnitID: ctx.ID, Kind: markOutKind}
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}
	ang := a.rng.Float64() * 2 * math.Pi
	ux, uy := math.Cos(ang), math.Sin(ang)
	ctx.Out <- unit.SetFSDirection{UnitID: ctx.ID, VX: ux, VY: uy}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: ux * patrolCruise, VY: uy * patrolCruise}
}

// 标识物：去程方向交给引擎（撞墙反射原样保留），自己只把速率越推越高；
// 持 Pass 撞人不减速，所以撞中谁都不停，回程也一路穿人。
type 标识物 struct {
	owner uint64
	slot  int

	booted bool
	speed  float64
	lastT  float64
	bornAt float64
	flipAt float64

	returning  bool
	struck     map[uint64]bool
	hitReadyAt float64

	detour       unit.DetourHold
	lastBounceFX float64
}

func (b *标识物) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		b.onSense(ctx, e)
	case unit.Collision:
		b.onCollision(ctx, e)
	case unit.WallHit:
		b.onWall(ctx, e)
	}
}

func (b *标识物) onSense(ctx unit.Context, s unit.Sense) {
	if !b.booted {
		b.booted = true
		b.lastT = s.Time
		b.bornAt = s.Time
		ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
		return
	}
	dt := s.Time - b.lastT
	if dt < 0 {
		dt = 0
	}
	b.lastT = s.Time
	b.speed += markAccel * dt

	if b.returning {
		b.homeward(ctx, s, dt)
		return
	}
	if s.Time-b.bornAt < markTimeout {
		if n := math.Hypot(s.Self.VX, s.Self.VY); n > 1e-6 {
			ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: s.Self.VX / n * b.speed, VY: s.Self.VY / n * b.speed}
		}
		return
	}
	b.flip(s.Time)
}

// homeward 回程：朝巡回当前位置直线加速；被墙别住时 Detour 绕端点。
func (b *标识物) homeward(ctx unit.Context, s unit.Sense, dt float64) {
	if s.Time-b.flipAt > returnTimeout {
		b.arrive(ctx, s.Self.X, s.Self.Y)
		return
	}
	owner := ownerOf(s, b.owner)
	if owner == nil {
		return
	}
	wx, wy, hold := unit.Detour(s.Self.X, s.Self.Y, s.Self.Radius, owner.X, owner.Y, s.Walls, b.detour, s.Time)
	b.detour = hold
	dx, dy := wx-s.Self.X, wy-s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: dx / n * b.speed, VY: dy / n * b.speed}

	dist := math.Hypot(owner.X-s.Self.X, owner.Y-s.Self.Y)
	catch := s.Self.Radius + owner.Radius + catchPad
	if dist <= catch || dist <= b.speed*dt {
		b.arrive(ctx, owner.X, owner.Y)
	}
}

func (b *标识物) arrive(ctx unit.Context, x, y float64) {
	ctx.Out <- unit.FX{Name: "patrol-catch", Kind: ctx.Kind, UnitID: b.owner, Slot: b.slot, X: x, Y: y}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

func (b *标识物) flip(t float64) {
	b.returning = true
	b.flipAt = t
	b.struck = make(map[uint64]bool)
	b.hitReadyAt = t + hitPaceCD
}

func (b *标识物) onCollision(ctx unit.Context, e unit.Collision) {
	if !unit.EnemyTarget(e, b.slot) {
		return
	}
	if !b.returning {
		b.strike(ctx, e)
		b.flip(e.Time)
		return
	}
	if b.struck == nil {
		b.struck = make(map[uint64]bool)
	}
	if b.struck[e.Other.ID] || e.Time < b.hitReadyAt {
		return
	}
	b.struck[e.Other.ID] = true
	b.hitReadyAt = e.Time + hitPaceCD
	b.strike(ctx, e)
}

func (b *标识物) strike(ctx unit.Context, e unit.Collision) {
	amt := b.speed * dmgPerSpeed
	if amt < dmgMin {
		amt = dmgMin
	}
	if amt > dmgMax {
		amt = dmgMax
	}
	ctx.Out <- unit.Damage{From: b.owner, To: e.Other.ID, Amount: amt}
	ctx.Out <- unit.FX{Name: "patrol-hit", Kind: ctx.Kind, UnitID: ctx.ID, Slot: b.slot,
		X: e.Other.X, Y: e.Other.Y, VX: e.NX, VY: e.NY, Amount: amt}
}

// onWall 去程的撞墙火花：弹一次亮一次，是"还在加速"的反馈；回程只顾回家，不撒。
func (b *标识物) onWall(ctx unit.Context, w unit.WallHit) {
	if b.returning || w.Time < b.lastBounceFX+bounceFXCD {
		return
	}
	b.lastBounceFX = w.Time
	ctx.Out <- unit.FX{Name: "patrol-bounce", Kind: ctx.Kind, UnitID: ctx.ID, Slot: b.slot,
		X: w.X, Y: w.Y, VX: w.NX, VY: w.NY}
}

func ownerOf(s unit.Sense, id uint64) *unit.Snapshot {
	if s.Self.ID == id {
		return &s.Self
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.ID == id {
			return o
		}
	}
	return nil
}
