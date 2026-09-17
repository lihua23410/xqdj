package 人偶使

import (
	"math"
	"sync"
	"xqdj/internal/unit"
)

const (
	dollIdle uint8 = iota
	dollStrike
	dollShoot
	dollEllipse
	dollEllipseRecall
	dollOrbit
	dollChase
	dollRecall
)

const (
	poseAa uint8 = iota
	poseAb
	poseAc
	poseAd
	poseAe
	poseAh
	poseAi
)

type dollSpec struct {
	mode     uint8
	pose     uint8
	x, y     float64
	tx, ty   float64
	ang      float64
	ux, uy   float64
	shootAt  float64
	armAt    float64
	until    float64
	recallAt float64
	arriveAt float64
	shots    int
	dmg      float64
	once     bool
	slow     bool
	rangeOn  bool
	follow   bool
	arriveIn float64
	bornGen  int
}

type bombSpec struct {
	kind    uint8
	dmg     float64
	tx, ty  float64
	explode float64
}

const (
	bombBounce uint8 = iota
	bombSeek
)

type chaseMark struct {
	start, until float64
	tx, ty       float64
}

var (
	dollMu   sync.Mutex
	dollQ    []dollSpec
	bombMu   sync.Mutex
	bombQ    []bombSpec
	stateMu  sync.Mutex
	dollSt   = map[uint64]uint8{}
	chaseAt  = map[uint64]chaseMark{}
	recallID = map[uint64]bool{}
)

func spawnDollAt(ctx unit.Context, s unit.Sense, destX, destY float64, spec dollSpec) {
	spec.x, spec.y = destX, destY
	sx, sy := s.Self.X, s.Self.Y
	if math.Hypot(destX-sx, destY-sy) > dollRadius {
		wait := spec.arriveIn
		if wait <= 0 {
			wait = deployLife
		}
		spec.arriveAt = s.Time + wait
	} else {
		sx, sy = destX, destY
	}
	pushDoll(spec)
	ctx.Out <- unit.Spawn{Kind: KindNingyushiDoll, X: sx, Y: sy, OwnerID: ctx.ID, Slot: s.Self.Slot}
}

func pushDoll(s dollSpec) {
	dollMu.Lock()
	dollQ = append(dollQ, s)
	dollMu.Unlock()
}

func popDoll() dollSpec {
	dollMu.Lock()
	defer dollMu.Unlock()
	if len(dollQ) == 0 {
		return dollSpec{mode: dollIdle}
	}
	s := dollQ[0]
	dollQ = dollQ[1:]
	return s
}

func pushBomb(s bombSpec) {
	bombMu.Lock()
	bombQ = append(bombQ, s)
	bombMu.Unlock()
}

func popBomb() bombSpec {
	bombMu.Lock()
	defer bombMu.Unlock()
	if len(bombQ) == 0 {
		return bombSpec{kind: bombBounce, dmg: 6}
	}
	s := bombQ[0]
	bombQ = bombQ[1:]
	return s
}

func setState(id uint64, st uint8) {
	stateMu.Lock()
	dollSt[id] = st
	stateMu.Unlock()
}

func stateOf(id uint64) uint8 {
	stateMu.Lock()
	defer stateMu.Unlock()
	return dollSt[id]
}

func dropState(id uint64) {
	stateMu.Lock()
	delete(dollSt, id)
	delete(chaseAt, id)
	delete(recallID, id)
	stateMu.Unlock()
}

func orderChase(id uint64, start, until, tx, ty float64) {
	stateMu.Lock()
	chaseAt[id] = chaseMark{start: start, until: until, tx: tx, ty: ty}
	stateMu.Unlock()
}

func orderRecall(id uint64) {
	stateMu.Lock()
	recallID[id] = true
	stateMu.Unlock()
}

func takeRecall(id uint64) bool {
	stateMu.Lock()
	defer stateMu.Unlock()
	if !recallID[id] {
		return false
	}
	delete(recallID, id)
	return true
}

func resetQueues() {
	dollMu.Lock()
	dollQ = nil
	dollMu.Unlock()
	bombMu.Lock()
	bombQ = nil
	bombMu.Unlock()
}

func chaseWindow(id uint64) (start, until, tx, ty float64, ok bool) {
	stateMu.Lock()
	defer stateMu.Unlock()
	v, ok := chaseAt[id]
	if !ok {
		return 0, 0, 0, 0, false
	}
	return v.start, v.until, v.tx, v.ty, true
}

type 人偶 struct {
	spec        dollSpec
	owner       uint64
	slot        int
	booted      bool
	lastT       float64
	x, y        float64
	next        float64
	left        int
	bursting    bool
	burstN      int
	burstAt     float64
	hitOnce     bool
	ang         float64
	recallUntil float64
}

func newDoll(info unit.SpawnInfo) *人偶 {
	s := popDoll()
	return &人偶{spec: s, owner: info.OwnerID, slot: info.Slot, ang: s.ang, left: s.shots}
}

func (d *人偶) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		d.onSense(ctx, e)
	}
}

func (d *人偶) onSense(ctx unit.Context, s unit.Sense) {
	if !d.booted {
		d.booted = true
		d.lastT = s.Time
		d.x, d.y = s.Self.X, s.Self.Y
		if d.spec.mode == 0 {
			d.spec.mode = dollIdle
		}
		setState(ctx.ID, d.spec.mode)
	}
	dt := s.Time - d.lastT
	if dt < 0 {
		dt = 0
	}
	d.lastT = s.Time
	d.x, d.y = s.Self.X, s.Self.Y
	if takeRecall(ctx.ID) && canForceRecall(d.spec.mode) {
		d.spec.slow = false
		d.spec.arriveAt = 0
		d.beginRecall(ctx)
	}
	if d.spec.follow {
		if o := ownerOf(d.owner); o != nil && o.abortGen != d.spec.bornGen {
			d.spec.follow = false
			d.spec.until = 0
			d.spec.arriveAt = 0
			d.spec.slow = false
			d.beginRecall(ctx)
		}
	}
	if d.spec.arriveAt > 0 && s.Time+1e-9 < d.spec.arriveAt {
		d.tickDeploy(ctx, s, dt)
		d.showRange(ctx, s)
		d.emitPose(ctx, s)
		if d.spec.mode == dollOrbit {
			d.pulseEllipse(ctx, s, enemyOf(s))
		}
		return
	}
	if d.spec.arriveAt > 0 {
		nx, ny := clampHex(d.spec.x, d.spec.y, dollRadius)
		ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: nx, Y: ny}
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
		d.spec.arriveAt = 0
		d.x, d.y = nx, ny
	}
	if d.spec.mode == dollIdle {
		if start, until, tx, ty, ok := chaseWindow(ctx.ID); ok && s.Time+1e-9 >= start {
			d.spec.mode = dollChase
			d.spec.until = until
			d.spec.tx, d.spec.ty = tx, ty
			d.spec.rangeOn = true
			setState(ctx.ID, dollChase)
		}
	}
	d.showRange(ctx, s)
	d.emitPose(ctx, s)
	switch d.spec.mode {
	case dollStrike:
		if d.spec.follow {
			d.tickFollow(ctx, s)
		}
		if d.spec.recallAt > 0 && s.Time+1e-9 >= d.spec.recallAt {
			d.beginRecall(ctx)
		} else if d.spec.until > 0 && s.Time+1e-9 >= d.spec.until {
			d.die(ctx)
		}
	case dollShoot:
		d.tickShoot(ctx, s)
	case dollEllipse, dollEllipseRecall:
		d.tickEllipse(ctx, s, dt)
	case dollOrbit:
		d.tickOrbit(ctx, s, dt)
	case dollChase:
		d.tickChase(ctx, s, dt)
	case dollRecall:
		d.tickRecall(ctx, s, dt)
	}
}

func canForceRecall(mode uint8) bool {
	switch mode {
	case dollIdle, dollShoot, dollStrike:
		return true
	default:
		return false
	}
}

func (d *人偶) showRange(ctx unit.Context, s unit.Sense) {
	armed := d.spec.armAt <= 0 || s.Time+1e-9 >= d.spec.armAt
	boomerang := d.spec.mode == dollRecall && d.spec.slow && d.spec.rangeOn
	on := 0.0
	if d.spec.rangeOn && armed && (d.spec.mode != dollRecall || boomerang) {
		if d.spec.arriveAt <= 0 || d.spec.mode == dollOrbit {
			on = 1
		}
	}
	ctx.Out <- unit.FX{Name: "range", Kind: ctx.Kind, UnitID: ctx.ID, Amount: on, VX: 1, VY: 0, Slot: d.slot}
}

func (d *人偶) emitPose(ctx unit.Context, s unit.Sense) {
	ctx.Out <- unit.FX{Name: "pose", Kind: ctx.Kind, UnitID: ctx.ID, Amount: float64(d.currentPose(s.Time)), Slot: d.slot}
}

func (d *人偶) currentPose(now float64) uint8 {
	if d.spec.arriveAt > now+1e-9 {
		if d.spec.mode == dollOrbit {
			return poseAe
		}
		return poseAb
	}
	switch d.spec.mode {
	case dollIdle:
		return poseAa
	case dollRecall:
		return poseAb
	case dollShoot:
		if d.spec.shootAt > 0 && now+1e-9 < d.spec.shootAt {
			return poseAb
		}
		return poseAc
	case dollStrike:
		if d.spec.pose != 0 {
			return d.spec.pose
		}
		return poseAd
	case dollEllipse, dollEllipseRecall:
		if d.spec.armAt > 0 && now+1e-9 < d.spec.armAt {
			return poseAb
		}
		return poseAe
	default:
		return poseAe
	}
}

func (d *人偶) tickFollow(ctx unit.Context, s unit.Sense) {
	var ox, oy float64
	found := false
	for i := range s.Nearby {
		if s.Nearby[i].ID == d.owner {
			ox, oy = s.Nearby[i].X, s.Nearby[i].Y
			found = true
			break
		}
	}
	if !found {
		return
	}
	x := ox + math.Cos(d.spec.ang)*spell24Ring
	y := oy + math.Sin(d.spec.ang)*spell24Ring
	nx, ny := clampHex(x, y, dollRadius)
	ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: nx, Y: ny}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
}

func (d *人偶) tickDeploy(ctx unit.Context, s unit.Sense, dt float64) {
	dx, dy := d.spec.x-s.Self.X, d.spec.y-s.Self.Y
	n := math.Hypot(dx, dy)
	remain := d.spec.arriveAt - s.Time
	if n < 1e-3 || remain <= 1e-6 {
		nx, ny := clampHex(d.spec.x, d.spec.y, dollRadius)
		ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: nx, Y: ny}
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
		d.spec.arriveAt = 0
		return
	}
	if dt < 1e-6 {
		dt = 1.0 / 60
	}
	sp := n / remain
	ux, uy := dx/n, dy/n
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: ux * sp, VY: uy * sp}
	step := sp * dt
	if step >= n {
		nx, ny := clampHex(d.spec.x, d.spec.y, dollRadius)
		ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: nx, Y: ny}
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
		d.spec.arriveAt = 0
		return
	}
	nx, ny := clampHex(s.Self.X+ux*step, s.Self.Y+uy*step, dollRadius)
	ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: nx, Y: ny}
}

func (d *人偶) tickShoot(ctx unit.Context, s unit.Sense) {
	if s.Time+1e-9 < d.spec.shootAt {
		return
	}
	if d.left <= 0 {
		if idleOthers(s, d.owner, ctx.ID) >= 2 {
			d.spec.slow = false
			d.beginRecall(ctx)
		} else {
			d.spec.mode = dollIdle
			d.spec.rangeOn = false
			setState(ctx.ID, dollIdle)
		}
		return
	}
	if d.next > 0 && s.Time+1e-9 < d.next {
		return
	}
	setState(ctx.ID, dollShoot)
	e := enemyOf(s)
	ux, uy := 1.0, 0.0
	if e != nil {
		ux, uy = toward(s.Self, *e)
	}
	ctx.Out <- unit.Spawn{
		Kind: KindNingyushiShot, X: s.Self.X + ux*(dollRadius+2), Y: s.Self.Y + uy*(dollRadius+2),
		VX: ux * shotSpeed, VY: uy * shotSpeed, OwnerID: d.owner, Slot: d.slot,
	}
	d.left--
	d.next = s.Time + placeLife/float64(placeShots)
}

func (d *人偶) tickEllipse(ctx unit.Context, s unit.Sense, dt float64) {
	if s.Time+1e-9 < d.spec.armAt {
		return
	}
	e := enemyOf(s)
	d.pulseEllipse(ctx, s, e)
	if d.spec.mode == dollEllipseRecall {
		d.spec.slow = true
		d.nudgeRecall(ctx, s, dt)
		return
	}
	if d.spec.until > 0 && s.Time+1e-9 >= d.spec.until {
		d.die(ctx)
	}
}

func (d *人偶) tickChase(ctx unit.Context, s unit.Sense, dt float64) {
	if s.Time+1e-9 >= d.spec.until {
		d.die(ctx)
		return
	}
	if dt < 1e-6 {
		dt = 1.0 / 60
	}
	dx, dy := d.spec.tx-s.Self.X, d.spec.ty-s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-3 {
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	} else {
		mx, my := dx/n, dy/n
		step := chaseSp * dt
		if step >= n {
			nx, ny := clampHex(d.spec.tx, d.spec.ty, dollRadius)
			ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: nx, Y: ny}
			ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
		} else {
			nx, ny := clampHex(s.Self.X+mx*step, s.Self.Y+my*step, dollRadius)
			ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: nx, Y: ny}
			ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: mx * chaseSp, VY: my * chaseSp}
		}
	}
	d.pulseEllipse(ctx, s, enemyOf(s))
}

func (d *人偶) tickOrbit(ctx unit.Context, s unit.Sense, dt float64) {
	if s.Time+1e-9 >= d.spec.until {
		d.die(ctx)
		return
	}
	d.ang += dt * (2 * math.Pi / orbitLife)
	x, y := math.Cos(d.ang)*orbitR, math.Sin(d.ang)*orbitR
	ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: x, Y: y}
	tx, ty := -math.Sin(d.ang), math.Cos(d.ang)
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: tx * (orbitR * 2 * math.Pi / orbitLife), VY: ty * (orbitR * 2 * math.Pi / orbitLife)}
	e := enemyOf(s)
	d.pulseEllipse(ctx, s, e)
}

func (d *人偶) pulseEllipse(ctx unit.Context, s unit.Sense, e *unit.Snapshot) {
	if e == nil {
		return
	}
	inside := ellipseHit(s.Self.X, s.Self.Y, ellipseRX, ellipseRY, *e)
	if d.spec.once {
		if inside && !d.hitOnce {
			d.hitOnce = true
			dmg := d.spec.dmg
			if dmg <= 0 {
				dmg = burstDmg
			}
			deal(ctx, d.owner, e.ID, dmg)
		}
		return
	}
	if !inside {
		return
	}
	d.pulseHits(ctx, s, e, burstDmg)
}

func (d *人偶) pulseHits(ctx unit.Context, s unit.Sense, e *unit.Snapshot, dmg float64) {
	if e == nil {
		return
	}
	if !d.bursting {
		d.bursting = true
		d.burstN = burstHits
		d.burstAt = s.Time
	}
	if s.Time+1e-9 < d.burstAt {
		return
	}
	if d.burstN <= 0 {
		d.bursting = false
		return
	}
	deal(ctx, d.owner, e.ID, dmg)
	d.burstN--
	d.burstAt = s.Time + burstLife/float64(burstHits)
	if d.burstN <= 0 {
		d.bursting = false
	}
}

func (d *人偶) beginRecall(ctx unit.Context) {
	if d.spec.mode == dollRecall {
		return
	}
	if !d.spec.slow {
		d.spec.rangeOn = false
	}
	d.spec.mode = dollRecall
	d.armRecall(d.lastT)
	setState(ctx.ID, dollRecall)
}

func (d *人偶) armRecall(now float64) {
	if d.recallUntil <= 0 {
		d.recallUntil = now + recallLife
	}
}

func (d *人偶) tickRecall(ctx unit.Context, s unit.Sense, dt float64) {
	if d.spec.slow && d.spec.rangeOn {
		d.pulseEllipse(ctx, s, enemyOf(s))
	}
	d.nudgeRecall(ctx, s, dt)
}

func (d *人偶) nudgeRecall(ctx unit.Context, s unit.Sense, dt float64) {
	ox, oy := 0.0, 0.0
	found := false
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.ID == d.owner {
			ox, oy = o.X, o.Y
			found = true
			break
		}
	}
	if !found {
		d.die(ctx)
		return
	}
	dx, dy := ox-s.Self.X, oy-s.Self.Y
	n := math.Hypot(dx, dy)
	home := ningyushiRadius + dollRadius
	if n < home {
		d.die(ctx)
		return
	}
	fresh := d.recallUntil <= 0
	d.armRecall(s.Time)
	if fresh {
		dt = 1.0 / 60
	}
	remain := d.recallUntil - s.Time
	if remain <= 1e-6 || dt >= remain {
		d.die(ctx)
		return
	}
	if dt < 1e-6 {
		dt = 1.0 / 60
	}
	sp := n / remain
	ux, uy := dx/n, dy/n
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: ux * sp, VY: uy * sp}
	step := sp * dt
	if step >= n-home {
		d.die(ctx)
		return
	}
	nx, ny := s.Self.X+ux*step, s.Self.Y+uy*step
	ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: nx, Y: ny}
}

func idleOthers(s unit.Sense, owner, self uint64) int {
	n := 0
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind != KindNingyushiDoll || o.OwnerID != owner || o.ID == self {
			continue
		}
		if stateOf(o.ID) == dollIdle {
			n++
		}
	}
	return n
}

func (d *人偶) die(ctx unit.Context) {
	dropState(ctx.ID)
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

type 人偶弹 struct {
	owner uint64
	slot  int
	dead  bool
}

func (b *人偶弹) Handle(ctx unit.Context, ev unit.Event) {
	if b.dead {
		return
	}
	switch e := ev.(type) {
	case unit.Collision:
		if e.Other.ID == b.owner {
			return
		}
		if e.Other.Role == unit.RoleFighter && e.Other.Slot != b.slot {
			deal(ctx, b.owner, e.Other.ID, placeDmg)
			b.dead = true
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		}
	case unit.WallHit:
		b.dead = true
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

type 人偶炸弹 struct {
	spec  bombSpec
	owner uint64
	slot  int
	dead  bool
	x, y  float64
}

func newBomb(info unit.SpawnInfo) *人偶炸弹 {
	return &人偶炸弹{spec: popBomb(), owner: info.OwnerID, slot: info.Slot}
}

func (b *人偶炸弹) Handle(ctx unit.Context, ev unit.Event) {
	if b.dead {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		b.x, b.y = e.Self.X, e.Self.Y
		pose := poseAi
		ctx.Out <- unit.FX{Name: "pose", Kind: ctx.Kind, UnitID: ctx.ID, Amount: float64(pose), Slot: b.slot}
		if b.spec.kind == bombSeek {
			dx, dy := b.spec.tx-e.Self.X, b.spec.ty-e.Self.Y
			if math.Hypot(dx, dy) <= bombR+8 {
				b.explode(ctx, e.Self.X, e.Self.Y, e)
			}
		}
	case unit.Collision:
		if e.Other.ID == b.owner {
			return
		}
		if e.Other.Role == unit.RoleFighter && e.Other.Slot != b.slot {
			if b.spec.kind == bombSeek {
				b.explode(ctx, b.x, b.y, unit.Sense{Self: unit.Snapshot{X: b.x, Y: b.y}, Nearby: []unit.Snapshot{e.Other}})
				return
			}
			deal(ctx, b.owner, e.Other.ID, b.spec.dmg)
			b.die(ctx)
		}
	}
}

func (b *人偶炸弹) explode(ctx unit.Context, x, y float64, s unit.Sense) {
	r := b.spec.explode
	if r <= 0 {
		r = demonR
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role != unit.RoleFighter || o.Slot == b.slot {
			continue
		}
		if math.Hypot(o.X-x, o.Y-y) <= r+o.Radius {
			deal(ctx, b.owner, o.ID, b.spec.dmg)
		}
	}
	ctx.Out <- unit.FX{Name: "blast", Kind: ctx.Kind, UnitID: ctx.ID, X: x, Y: y, Slot: b.slot, Amount: r}
	b.die(ctx)
}

func (b *人偶炸弹) die(ctx unit.Context) {
	if b.dead {
		return
	}
	b.dead = true
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

type 激光 struct {
	booted bool
	until  float64
}

func (b *激光) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !b.booted {
		b.booted = true
		b.until = s.Time + houraiLife
	}
	if s.Time+1e-9 >= b.until {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}
