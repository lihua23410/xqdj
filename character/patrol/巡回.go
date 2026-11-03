// 巡回朝随机方向丢出标识物后站定。标识物一路加速、撞墙反弹。
// 去程至少飞满 5 秒。满 5 秒之后，下一次撞墙才折返。
// 折返沿去程轨迹倒着飞，这段穿墙，轨迹走完再飞向主人当前位置。
// 伤害随速率增长，封顶 20。同一次接触只出一次伤，分开后再撞可以再打。回到巡回身边消失。
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
	markStartSpeed  = 0.0
	// 从静止起步，4 秒达到 20 伤对应的速率（约 1428.57）。
	markFullDamageAfter = 4.0
	markAccel           = (dmgMax/dmgPerSpeed - markStartSpeed) / markFullDamageAfter
	// 去程至少 5 秒。满了之后要再撞一次墙才折返。
	markTimeout = 5.0
	// 伤害随撞击时速走：飞得越久砸得越重（反重力弹先例）。
	dmgPerSpeed = 0.014
	dmgMin      = 8.0
	dmgMax      = 20.0
	// 分开超过皮肤厚度才算离开，避免贴着来回触发。
	touchSlop  = 1.0
	bounceFXCD = 0.15
	catchPad   = 2.0

	markOutKind = "巡回标识物在外"

	// 去程球心取样：静止起步时连续几拍几乎重合，不堆重复点。
	trailMinStep = 1e-3
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

		ZeroVelocityWhileOwned: KindPatrolMarker,
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
		Look:    unit.Look{Color: markColor, Overlay: true, FX: []string{"patrolmark"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &标识物{owner: info.OwnerID, slot: info.Slot, speed: markStartSpeed}
	})
}

// 巡回本体：站定跟随自己全部标识物的实际在场状态，眩晕期间也更新。
type 巡回 struct {
	slot          int
	rng           *rand.Rand
	waiting       bool
	resumePending bool
	nextThrowAt   float64
}

func (a *巡回) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		a.onSense(ctx, e)
	case unit.OwnedPresenceChanged:
		a.onOwnedPresence(ctx, e)
	}
}

func (a *巡回) onSense(ctx unit.Context, s unit.Sense) {
	if a.waiting {
		return
	}
	// 最后一颗弹在眩晕中离场，冷却从离场起算；能再次行动时才恢复移动。
	if a.resumePending {
		a.resumePending = false
		a.resumeMovement(ctx)
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

func (a *巡回) onOwnedPresence(ctx unit.Context, e unit.OwnedPresenceChanged) {
	if e.Kind != KindPatrolMarker {
		return
	}
	if e.Present {
		if !a.waiting {
			ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
			ctx.Out <- unit.StackMark{UnitID: ctx.ID, Kind: markOutKind, Delta: 1}
		}
		a.waiting = true
		a.resumePending = false
		return
	}
	if !a.waiting {
		return
	}
	a.waiting = false
	a.nextThrowAt = e.Time + throwCD
	a.resumePending = e.SelfStunned
	ctx.Out <- unit.ClearMarks{UnitID: ctx.ID, Kind: markOutKind}
	if !a.resumePending {
		a.resumeMovement(ctx)
	}
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

func (a *巡回) resumeMovement(ctx unit.Context) {
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}
	ang := a.rng.Float64() * 2 * math.Pi
	ux, uy := math.Cos(ang), math.Sin(ang)
	ctx.Out <- unit.SetFSDirection{UnitID: ctx.ID, VX: ux, VY: uy}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: ux * patrolCruise, VY: uy * patrolCruise}
}

// 标识物：去程方向交给引擎（撞墙反射原样保留），自己只把速率越推越高，并记下球心。
// 持 Pass 撞人不减速。折返起持 PassWalls，沿记下的轨迹倒回，走完再飞向主人。
type 标识物 struct {
	owner uint64
	slot  int

	booted bool
	speed  float64
	dirX   float64
	dirY   float64
	lastT  float64
	bornAt float64

	returning bool
	touching  map[uint64]bool

	trail        []trailPt
	lastBounceFX float64
}

type trailPt struct {
	x, y float64
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
	b.releaseTouch(s)
	if !b.booted {
		b.booted = true
		b.lastT = s.Time
		b.bornAt = s.Time
		// 初速为 0 时，从出生点相对主人的偏移还原随机出手方向。
		dx, dy := s.Self.VX, s.Self.VY
		if math.Hypot(dx, dy) < 1e-6 {
			if owner := ownerOf(s, b.owner); owner != nil {
				dx, dy = s.Self.X-owner.X, s.Self.Y-owner.Y
			}
		}
		b.dirX, b.dirY = 1, 0
		if n := math.Hypot(dx, dy); n > 1e-6 {
			b.dirX, b.dirY = dx/n, dy/n
		}
		b.noteTrail(s.Self.X, s.Self.Y)
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
		b.rewind(ctx, s, dt)
		return
	}
	b.noteTrail(s.Self.X, s.Self.Y)
	if n := math.Hypot(s.Self.VX, s.Self.VY); n > 1e-6 {
		b.dirX, b.dirY = s.Self.VX/n, s.Self.VY/n
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: b.dirX * b.speed, VY: b.dirY * b.speed}
}

func (b *标识物) windowDone(t float64) bool {
	return b.booted && t-b.bornAt+1e-9 >= markTimeout
}

// rewind 回程：沿去程球心折线从新到旧飞出这一拍的行程，走完再飞向主人当前位置。
// 穿墙之后不再绕墙；必须真的靠近才接回，不能因为理论上这一拍够远就消失。
func (b *标识物) rewind(ctx unit.Context, s unit.Sense, dt float64) {
	owner := ownerOf(s, b.owner)
	if owner != nil {
		dist := math.Hypot(owner.X-s.Self.X, owner.Y-s.Self.Y)
		catch := s.Self.Radius + owner.Radius + catchPad
		if dist <= catch {
			b.arrive(ctx, owner.X, owner.Y)
			return
		}
	}
	vx, vy, ok := b.rewindVelocity(s.Self.X, s.Self.Y, b.speed, dt, owner)
	if !ok {
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
		return
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: vx, VY: vy}
}

// rewindVelocity 把这一拍的行程走在轨迹上，剩下的步长接向主人。轨迹点按走完的丢掉。
// 没有可去之处时 ok 为 false，调用方把速度写成 0，避免穿墙后沿着最后一段飞出场。
func (b *标识物) rewindVelocity(x, y, speed, dt float64, owner *unit.Snapshot) (vx, vy float64, ok bool) {
	budget := speed * dt
	for len(b.trail) > 0 {
		last := b.trail[len(b.trail)-1]
		if math.Hypot(last.x-x, last.y-y) >= trailMinStep {
			break
		}
		b.trail = b.trail[:len(b.trail)-1]
	}
	tx, ty := x, y
	for budget > 1e-12 && len(b.trail) > 0 {
		last := b.trail[len(b.trail)-1]
		dx, dy := last.x-tx, last.y-ty
		seg := math.Hypot(dx, dy)
		if seg <= budget {
			budget -= seg
			tx, ty = last.x, last.y
			b.trail = b.trail[:len(b.trail)-1]
			continue
		}
		k := budget / seg
		tx += dx * k
		ty += dy * k
		budget = 0
	}
	if budget > 1e-12 && owner != nil {
		dx, dy := owner.X-tx, owner.Y-ty
		seg := math.Hypot(dx, dy)
		if seg > 1e-12 {
			if seg <= budget {
				tx, ty = owner.X, owner.Y
			} else {
				k := budget / seg
				tx += dx * k
				ty += dy * k
			}
		}
	}
	dx, dy := tx-x, ty-y
	n := math.Hypot(dx, dy)
	if n < 1e-9 {
		return 0, 0, false
	}
	step := speed
	if dt > 0 && n/dt < speed {
		step = n / dt
	}
	return dx / n * step, dy / n * step, true
}

func (b *标识物) noteTrail(x, y float64) {
	if n := len(b.trail); n > 0 {
		last := b.trail[n-1]
		if math.Hypot(x-last.x, y-last.y) < trailMinStep {
			return
		}
	}
	b.trail = append(b.trail, trailPt{x: x, y: y})
}

func (b *标识物) arrive(ctx unit.Context, x, y float64) {
	ctx.Out <- unit.FX{Name: "patrol-catch", Kind: ctx.Kind, UnitID: b.owner, Slot: b.slot, X: x, Y: y}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

func (b *标识物) flip(ctx unit.Context) {
	b.returning = true
	ctx.Out <- unit.PassWalls{UnitID: ctx.ID, Hold: true}
}

func (b *标识物) onCollision(ctx unit.Context, e unit.Collision) {
	if !unit.EnemyTarget(e, b.slot) {
		return
	}
	if b.touching[e.Other.ID] {
		return
	}
	if b.touching == nil {
		b.touching = map[uint64]bool{}
	}
	b.touching[e.Other.ID] = true
	b.strike(ctx, e)
}

// releaseTouch 在感知里确认已经分开的人。还叠在一起时引擎每拍都会再报碰撞。
func (b *标识物) releaseTouch(s unit.Sense) {
	for id := range b.touching {
		o := snapshotByID(s, id)
		if o == nil || math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) > s.Self.Radius+o.Radius+touchSlop {
			delete(b.touching, id)
		}
	}
}

func snapshotByID(s unit.Sense, id uint64) *unit.Snapshot {
	for i := range s.Nearby {
		if s.Nearby[i].ID == id {
			return &s.Nearby[i]
		}
	}
	return nil
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

// onWall 去程的撞墙火花：弹一次亮一次，是"还在加速"的反馈。
// 满 5 秒后的这一下不再撒火花，它负责进入收回。回程穿墙，不会再撞到。
func (b *标识物) onWall(ctx unit.Context, w unit.WallHit) {
	if b.returning {
		return
	}
	if b.windowDone(w.Time) {
		b.noteTrail(w.X, w.Y)
		b.flip(ctx)
		return
	}
	if w.Time < b.lastBounceFX+bounceFXCD {
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
