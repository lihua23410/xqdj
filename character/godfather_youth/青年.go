// 教父(青年)自己出伤。战斗状态围绕针剂组织。不拾取药物，不带活随从。
package 教父青年

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

const KindYouth = "教父(青年)"
const KindYouthShot = "教父(青年)手枪弹"
const KindYouthBuffShot = "教父(青年)强化弹"

const (
	bodyRadius = 18.0
	bodyHP     = 100.0
	cruise     = 180.0
	buffCruise = 270.0
	vision     = 9999.0
	bodyColor  = "#3a322c"
	shotColor  = "#c9a227"

	doseCap   = 3
	doseEvery = 30.0
	doseHeal  = 15.0
	doseFloor = 50.0
	hpEdge    = 20.0
	buffDur   = 8.0

	closeR      = 125.0
	markDur     = 2.0
	markAim     = 2
	spread      = 30 * math.Pi / 180
	shotGap     = 0.12
	shotSpeed   = 640.0
	shotRadius  = 4.0
	shotDmg     = 1.0
	buffShotDmg = 1.5
	shotN       = 3
	buffShotN   = 5
	shotRecover = 3.0
	knockSpeed  = 240.0
	knockDur    = 0.2

	// 与暗杀者冲刺同一套：速率 640、距离 280，全程持 Pass。
	ramSpeed   = 640.0
	ramDist    = 280.0
	ramDmg     = 2.5
	ramRecover = 0.45
	buffRamCD  = 0.2
	stunDur    = 0.8
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindYouth, assets)
	p.Register(unit.Spec{
		Kind: KindYouth, Role: unit.RoleFighter, Radius: bodyRadius, MaxHP: bodyHP,
		Speed: cruise, Vision: vision, Fighter: true,
		Look: unit.Look{Color: bodyColor, Ghost: 280, FX: []string{"youth"}},
	}, func(unit.SpawnInfo) unit.Actor { return &教父青年{} })
	regShot(p, KindYouthShot, shotDmg, false)
	regShot(p, KindYouthBuffShot, buffShotDmg, true)
}

func regShot(p *unit.Pack, kind string, dmg float64, knock bool) {
	p.Register(unit.Spec{
		Kind: kind, Role: unit.RoleProjectile, Radius: shotRadius, MaxHP: 1,
		Speed: shotSpeed, Vision: 0,
		Look: unit.Look{Color: shotColor, FX: []string{"youth-shot"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &手枪弹{owner: info.OwnerID, slot: info.Slot, dmg: dmg, knock: knock}
	})
}

type phase uint8

const (
	phIdle phase = iota
	phVolley
	phRam
	phReturn
)

type 教父青年 struct {
	rng       *rand.Rand
	roll      func() float64
	booted    bool
	slot      int
	doses     int
	nextDose  float64
	prevHP    float64
	buffUntil float64
	phase     phase
	readyAt   float64
	shotsLeft int
	shotKind  string
	nextShot  float64
	aimX      float64
	aimY      float64
	haveAim   bool
	ramID     uint64
	ramX      float64
	ramY      float64
	dashLeft  float64
	lastT     float64
	hitReady  float64
	struck    bool
	opened    bool
	inside    map[uint64]bool
	marks     map[uint64]aimMark
}

type aimMark struct {
	until float64
	prev  uint8
}

func (a *教父青年) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		unit.ConfirmHit(ctx, e)
	case unit.WallHit:
		if a.phase == phRam || a.phase == phReturn {
			a.ramX, a.ramY = reflectDir(a.ramX, a.ramY, e.NX, e.NY)
			a.finish(ctx, e.Time)
		}
	case unit.Collision:
		a.connect(ctx, e.Time, e.Other)
	case unit.Sense:
		a.onSense(ctx, e)
	}
}

func (a *教父青年) onSense(ctx unit.Context, s unit.Sense) {
	a.slot = s.Self.Slot
	if !a.booted {
		a.boot(ctx, s)
	}
	if s.Self.HP <= 0 {
		a.prevHP = 0
		if a.buffUntil > 0 {
			a.buffUntil = 0
			ctx.Out <- unit.FX{Name: "buff", Kind: KindYouth, UnitID: ctx.ID, Amount: 0, Slot: s.Self.Slot}
		}
		return
	}
	a.drugs(ctx, s)
	a.expireBuff(ctx, s)
	a.markEnter(ctx, s)
	a.markEnter(ctx, s)
	dt := s.Time - a.lastT
	if dt < 0 {
		dt = 0
	}
	a.lastT = s.Time
	if a.phase != phRam && a.phase != phReturn {
		a.steer(ctx, s)
	}
	a.act(ctx, s, dt)
}

func (a *教父青年) boot(ctx unit.Context, s unit.Sense) {
	a.booted = true
	a.doses = doseCap
	a.nextDose = doseEvery
	a.prevHP = s.Self.HP
	a.rng = rand.New(rand.NewPCG(s.Self.ID, uint64(s.Time*1e6)+1))
	a.tellDoses(ctx, s)
	ctx.Out <- unit.FX{Name: "buff", Kind: KindYouth, UnitID: ctx.ID, Amount: 0, Slot: s.Self.Slot}
}

func (a *教父青年) drugs(ctx unit.Context, s unit.Sense) {
	clock := s.Time+1e-9 >= a.nextDose
	edge := a.prevHP >= hpEdge && s.Self.HP < hpEdge
	if a.doses > 0 && (clock || edge) {
		a.spend(ctx, s)
	}
	if clock {
		for a.nextDose <= s.Time+1e-9 {
			a.nextDose += doseEvery
		}
	}
	a.prevHP = s.Self.HP
}

func (a *教父青年) spend(ctx unit.Context, s unit.Sense) {
	a.doses--
	hp, maxHP := s.Self.HP, s.Self.MaxHP
	if maxHP <= 0 {
		maxHP = bodyHP
	}
	heal := doseHeal
	room := maxHP - hp
	if heal > room {
		heal = room
	}
	if heal < 0 {
		heal = 0
	}
	if hp+heal < doseFloor && maxHP >= doseFloor {
		need := doseFloor - (hp + heal)
		if heal+need > room {
			need = room - heal
		}
		if need > 0 {
			heal += need
		}
	}
	if heal > 0 {
		ctx.Out <- unit.Heal{UnitID: ctx.ID, Amount: heal}
	}
	a.buffUntil = s.Time + buffDur
	if a.phase != phRam && a.phase != phReturn {
		ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: buffCruise}
	}
	a.tellDoses(ctx, s)
	ctx.Out <- unit.FX{Name: "buff", Kind: KindYouth, UnitID: ctx.ID, Amount: 1, Slot: s.Self.Slot}
}

func (a *教父青年) expireBuff(ctx unit.Context, s unit.Sense) {
	if a.buffUntil <= 0 || s.Time+1e-9 < a.buffUntil {
		return
	}
	a.buffUntil = 0
	if a.phase != phRam && a.phase != phReturn {
		ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: cruise}
	}
	ctx.Out <- unit.FX{Name: "buff", Kind: KindYouth, UnitID: ctx.ID, Amount: 0, Slot: s.Self.Slot}
}

func (a *教父青年) tellDoses(ctx unit.Context, s unit.Sense) {
	ctx.Out <- unit.FX{Name: "doses", Kind: KindYouth, UnitID: ctx.ID, Amount: float64(a.doses), Slot: s.Self.Slot}
}

func (a *教父青年) markEnter(ctx unit.Context, s unit.Sense) {
	if a.inside == nil {
		a.inside = map[uint64]bool{}
	}
	if a.marks == nil {
		a.marks = map[uint64]aimMark{}
	}
	for id, m := range a.marks {
		if s.Time+1e-9 < m.until {
			continue
		}
		if m.prev != 0 {
			ctx.Out <- unit.SetAimPriority{From: ctx.ID, UnitID: id, Value: m.prev}
		}
		delete(a.marks, id)
	}
	r2 := closeR * closeR
	seen := map[uint64]bool{}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) || o.AimPriority == 0 {
			continue
		}
		dx, dy := o.X-s.Self.X, o.Y-s.Self.Y
		nowIn := dx*dx+dy*dy <= r2
		seen[o.ID] = true
		was, known := a.inside[o.ID]
		if nowIn && (!a.opened || (known && !was)) {
			prev := o.AimPriority
			if old, ok := a.marks[o.ID]; ok {
				prev = old.prev
			}
			a.marks[o.ID] = aimMark{until: s.Time + markDur, prev: prev}
			ctx.Out <- unit.SetAimPriority{From: ctx.ID, UnitID: o.ID, Value: markAim}
		}
		a.inside[o.ID] = nowIn
	}
	for id := range a.inside {
		if !seen[id] {
			delete(a.inside, id)
		}
	}
	a.opened = true
}

func (a *教父青年) steer(ctx unit.Context, s unit.Sense) {
	t := unit.Seek(s)
	if t == nil {
		return
	}
	dx, dy := t.X-s.Self.X, t.Y-s.Self.Y
	if math.Hypot(dx, dy) < 1e-6 {
		return
	}
	ctx.Out <- unit.SetFSDirection{UnitID: ctx.ID, VX: dx, VY: dy}
}

func (a *教父青年) act(ctx unit.Context, s unit.Sense, dt float64) {
	switch a.phase {
	case phRam, phReturn:
		a.dashLeft -= ramSpeed * dt
		a.push(ctx)
		if a.dashLeft <= 0 {
			a.finish(ctx, s.Time)
		}
		return
	case phVolley:
		if foe := a.nearestClose(s); foe != nil {
			a.phase = phIdle
			a.startRam(ctx, s, foe)
			return
		}
		if s.Time+1e-9 < a.nextShot {
			return
		}
		a.fire(ctx, s)
	default:
		if foe := a.nearestClose(s); foe != nil {
			a.startRam(ctx, s, foe)
			return
		}
		if s.Time+1e-9 < a.readyAt {
			return
		}
		t := unit.Seek(s)
		if t == nil {
			return
		}
		a.startVolley(ctx, s)
	}
}

func (a *教父青年) nearestClose(s unit.Sense) *unit.Snapshot {
	var best *unit.Snapshot
	bestD := closeR + 1e-9
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		d := dist(s.Self, *o)
		if d <= bestD {
			best = o
			bestD = d
		}
	}
	return best
}

func (a *教父青年) startVolley(ctx unit.Context, s unit.Sense) {
	a.phase = phVolley
	a.shotsLeft = shotN
	a.shotKind = KindYouthShot
	if s.Time+1e-9 < a.buffUntil {
		a.shotsLeft = buffShotN
		a.shotKind = KindYouthBuffShot
	}
	a.fire(ctx, s)
}

func (a *教父青年) fire(ctx unit.Context, s unit.Sense) {
	ux, uy, ok := a.aim(s)
	if ok {
		ang := math.Atan2(uy, ux) + a.spreadAng()
		ux, uy = math.Cos(ang), math.Sin(ang)
		a.aimX, a.aimY, a.haveAim = ux, uy, true
	} else if a.haveAim {
		ux, uy = a.aimX, a.aimY
	} else {
		a.phase = phIdle
		a.readyAt = s.Time + shotRecover
		return
	}
	gap := s.Self.Radius + shotRadius + 1.5
	ctx.Out <- unit.Spawn{
		Kind: a.shotKind,
		X:    s.Self.X + ux*gap, Y: s.Self.Y + uy*gap,
		VX: ux * shotSpeed, VY: uy * shotSpeed,
		OwnerID: ctx.ID, Slot: s.Self.Slot,
	}
	a.shotsLeft--
	if a.shotsLeft > 0 {
		a.nextShot = s.Time + shotGap
		return
	}
	a.phase = phIdle
	a.readyAt = s.Time + shotRecover
}

func (a *教父青年) aim(s unit.Sense) (float64, float64, bool) {
	t := unit.Seek(s)
	if t == nil {
		return 0, 0, false
	}
	dx, dy := t.X-s.Self.X, t.Y-s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return 0, 0, false
	}
	return dx / n, dy / n, true
}

func (a *教父青年) spreadAng() float64 {
	u := 0.5
	if a.roll != nil {
		u = a.roll()
	} else if a.rng != nil {
		u = a.rng.Float64()
	}
	if u < 0 {
		u = 0
	}
	if u > 1 {
		u = 1
	}
	return (u*2 - 1) * spread
}

func (a *教父青年) startRam(ctx unit.Context, s unit.Sense, t *unit.Snapshot) {
	dx, dy := t.X-s.Self.X, t.Y-s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		dx, dy, n = 1, 0, 1
	}
	a.phase = phRam
	a.ramID = t.ID
	a.ramX, a.ramY = dx/n, dy/n
	a.dashLeft = ramDist
	a.struck = false
	a.hitReady = 0
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
	a.push(ctx)
}

func (a *教父青年) push(ctx unit.Context) {
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: a.ramX * ramSpeed, VY: a.ramY * ramSpeed}
}

func (a *教父青年) connect(ctx unit.Context, now float64, other unit.Snapshot) {
	if (a.phase != phRam && a.phase != phReturn) || other.ID != a.ramID || a.struck {
		return
	}
	if !unit.Hittable(other, a.slot) {
		return
	}
	empowered := a.phase == phReturn || now+1e-9 < a.buffUntil
	if empowered && now+1e-9 < a.hitReady {
		return
	}
	if empowered {
		a.hitReady = now + buffRamCD
	}
	a.struck = true
	ctx.Out <- unit.Damage{From: ctx.ID, To: other.ID, Amount: ramDmg}
	if a.phase == phRam && now+1e-9 < a.buffUntil {
		ctx.Out <- unit.Stun{UnitID: other.ID, Hold: true, Until: now + stunDur}
		a.ramX, a.ramY = -a.ramX, -a.ramY
		a.phase = phReturn
		a.dashLeft = ramDist
		a.struck = false
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: a.ramX * ramSpeed, VY: a.ramY * ramSpeed}
	}
}

func (a *教父青年) finish(ctx unit.Context, now float64) {
	a.phase = phIdle
	a.readyAt = now + ramRecover
	speed := cruise
	if now+1e-9 < a.buffUntil {
		speed = buffCruise
	}
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: false}
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: speed}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: a.ramX * speed, VY: a.ramY * speed}
}

func dist(a, b unit.Snapshot) float64 {
	return math.Hypot(a.X-b.X, a.Y-b.Y)
}

func reflectDir(ux, uy, nx, ny float64) (float64, float64) {
	dot := ux*nx + uy*ny
	rx, ry := ux-2*dot*nx, uy-2*dot*ny
	n := math.Hypot(rx, ry)
	if n < 1e-6 {
		return ux, uy
	}
	return rx / n, ry / n
}

type 手枪弹 struct {
	owner uint64
	slot  int
	dmg   float64
	knock bool
	ux    float64
	uy    float64
	aimed bool
}

func (b *手枪弹) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		if sp := math.Hypot(e.Self.VX, e.Self.VY); sp > 1e-6 {
			b.ux, b.uy, b.aimed = e.Self.VX/sp, e.Self.VY/sp, true
		}
	case unit.Collision:
		b.hit(ctx, e)
	case unit.WallHit:
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

func (b *手枪弹) hit(ctx unit.Context, e unit.Collision) {
	if e.Other.ID == b.owner {
		return
	}
	if !unit.Hittable(e.Other, b.slot) {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: b.dmg}
	if b.knock {
		ux, uy := b.ux, b.uy
		if !b.aimed {
			ux, uy = -e.NX, -e.NY
		}
		if math.Hypot(ux, uy) < 1e-6 {
			ux, uy = 1, 0
		}
		ctx.Out <- unit.AddFS{
			UnitID: e.Other.ID, DX: ux, DY: uy, BaseSpeed: knockSpeed, OnWall: true,
			ExpiresAt: e.Time + knockDur, Token: b.owner<<32 | e.Other.ID,
		}
	}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}
