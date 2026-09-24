package 原型机_整合

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const KindIntegrated = "原型机_整合"
const KindIntegratedRound = "原型机_整合弹"

const (
	bodyRadius   = 18.0
	cruise       = 148.0
	vision       = 9999.0
	bodyHP       = 100.0
	nearRadius   = 185.0
	fleeBoost    = 70.0
	fleeExtraMax = 300.0
	trackSeconds = 5.0
	fireInterval = 0.5
	cone         = 60 * math.Pi / 180
	turnStep     = 1.8 * math.Pi / 180

	roundRadius  = 6.0
	roundSpeed   = 185.0
	roundHP      = 1.0
	roundDamage  = 2.0
	roundBounces = 2

	coreColor = "#3dd6c6"
)

func init() {
	p := unit.NewPack(KindIntegrated, assets)
	p.Register(unit.Spec{
		Kind:    KindIntegrated,
		Role:    unit.RoleFighter,
		Radius:  bodyRadius,
		MaxHP:   bodyHP,
		Speed:   cruise,
		Vision:  vision,
		Fighter: true,
		Look:    unit.Look{Color: coreColor, FX: []string{"integrated"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &原型机_整合{}
	})
	p.Register(unit.Spec{
		Kind:    KindIntegratedRound,
		Role:    unit.RoleProjectile,
		Radius:  roundRadius,
		MaxHP:   roundHP,
		Speed:   roundSpeed,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: coreColor, FX: []string{"integrated-shot"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &整合弹{owner: info.OwnerID, slot: info.Slot}
	})
}

type 原型机_整合 struct {
	rng         *rand.Rand
	roll        func() float64
	fireReadyAt float64
	opened      bool
	trackID     uint64
	trackUntil  float64
	prevAim     uint8
	inside      map[uint64]bool
	tracking    bool
}

func (a *原型机_整合) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if a.rng == nil {
		a.rng = rand.New(rand.NewPCG(rand.Uint64()^ctx.ID, rand.Uint64()))
	}
	if a.inside == nil {
		a.inside = map[uint64]bool{}
	}
	a.track(ctx, s)
	if a.trackID != 0 {
		a.steer(ctx, s)
	}
	a.shoot(ctx, s)
}

func (a *原型机_整合) track(ctx unit.Context, s unit.Sense) {
	if !a.opened {
		a.lock(ctx, s, a.insideNow(s))
		a.remember(s)
		a.opened = true
		return
	}
	if a.trackID != 0 && !a.stillTracked(s) {
		a.clear(ctx, s)
	}
	entered := a.entrants(s)
	if a.trackID == 0 {
		a.lock(ctx, s, aimableOf(s, entered))
	}
	a.shove(ctx, s, entered)
	a.remember(s)
}

func (a *原型机_整合) stillTracked(s unit.Sense) bool {
	if s.Time >= a.trackUntil {
		return false
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.ID != a.trackID {
			continue
		}
		return o.HP > 0
	}
	return false
}

func (a *原型机_整合) insideNow(s unit.Sense) []unit.Snapshot {
	var out []unit.Snapshot
	r2 := nearRadius * nearRadius
	for i := range s.Nearby {
		o := s.Nearby[i]
		if !unit.Aimable(o, s.Self.Slot) {
			continue
		}
		dx, dy := o.X-s.Self.X, o.Y-s.Self.Y
		if dx*dx+dy*dy <= r2 {
			out = append(out, o)
		}
	}
	return out
}

func (a *原型机_整合) entrants(s unit.Sense) []unit.Snapshot {
	var out []unit.Snapshot
	r2 := nearRadius * nearRadius
	for i := range s.Nearby {
		o := s.Nearby[i]
		if !unit.Hittable(o, s.Self.Slot) {
			continue
		}
		dx, dy := o.X-s.Self.X, o.Y-s.Self.Y
		nowIn := dx*dx+dy*dy <= r2
		prev, seen := a.inside[o.ID]
		if seen && !prev && nowIn {
			out = append(out, o)
		}
	}
	return out
}

func aimableOf(s unit.Sense, cands []unit.Snapshot) []unit.Snapshot {
	var out []unit.Snapshot
	for i := range cands {
		if unit.Aimable(cands[i], s.Self.Slot) {
			out = append(out, cands[i])
		}
	}
	return out
}

func (a *原型机_整合) shove(ctx unit.Context, s unit.Sense, entered []unit.Snapshot) {
	foe := pick(s, aimableOf(s, entered))
	if foe == nil {
		foe = pick(s, entered)
	}
	if foe == nil {
		return
	}
	dx, dy := s.Self.X-foe.X, s.Self.Y-foe.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	sp := math.Hypot(s.Self.VX, s.Self.VY)
	if sp < 1e-6 {
		sp = cruise
	}
	room := cruise + fleeExtraMax - sp
	if room > 0 {
		add := fleeBoost
		if add > room {
			add = room
		}
		sp += add
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: dx / n * sp, VY: dy / n * sp}
}

func (a *原型机_整合) remember(s unit.Sense) {
	seen := map[uint64]bool{}
	r2 := nearRadius * nearRadius
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		seen[o.ID] = true
		dx, dy := o.X-s.Self.X, o.Y-s.Self.Y
		a.inside[o.ID] = dx*dx+dy*dy <= r2
	}
	for id := range a.inside {
		if !seen[id] {
			delete(a.inside, id)
		}
	}
}

func (a *原型机_整合) lock(ctx unit.Context, s unit.Sense, cands []unit.Snapshot) {
	best := pick(s, cands)
	if best == nil {
		return
	}
	a.trackID = best.ID
	a.trackUntil = s.Time + trackSeconds
	a.prevAim = best.AimPriority
	ctx.Out <- unit.SetAimPriority{From: ctx.ID, UnitID: best.ID, Value: 1}
	if !a.tracking {
		a.tracking = true
		a.emitTrack(ctx, s, 1, best.ID)
	}
}

func (a *原型机_整合) clear(ctx unit.Context, s unit.Sense) {
	id, prev := a.trackID, a.prevAim
	a.trackID = 0
	a.trackUntil = 0
	a.prevAim = 0
	if id != 0 && prev != 0 {
		ctx.Out <- unit.SetAimPriority{From: ctx.ID, UnitID: id, Value: prev}
	}
	if a.tracking {
		a.tracking = false
		a.emitTrack(ctx, s, 0, 0)
	}
}

func (a *原型机_整合) emitTrack(ctx unit.Context, s unit.Sense, amount float64, target uint64) {
	ctx.Out <- unit.FX{
		Name:   "track",
		UnitID: ctx.ID,
		Kind:   ctx.Kind,
		X:      s.Self.X,
		Y:      s.Self.Y,
		VX:     float64(target),
		Slot:   s.Self.Slot,
		Amount: amount,
	}
}

func pick(s unit.Sense, cands []unit.Snapshot) *unit.Snapshot {
	var best *unit.Snapshot
	bestPri := uint8(255)
	bestD2 := math.MaxFloat64
	for i := range cands {
		o := &cands[i]
		dx, dy := o.X-s.Self.X, o.Y-s.Self.Y
		d2 := dx*dx + dy*dy
		if best != nil && (o.AimPriority > bestPri || (o.AimPriority == bestPri && d2 >= bestD2)) {
			continue
		}
		best = o
		bestPri = o.AimPriority
		bestD2 = d2
	}
	return best
}

func (a *原型机_整合) steer(ctx unit.Context, s unit.Sense) {
	var target *unit.Snapshot
	for i := range s.Nearby {
		if s.Nearby[i].ID == a.trackID {
			target = &s.Nearby[i]
			break
		}
	}
	if target == nil {
		return
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID != ctx.ID || o.Kind != KindIntegratedRound {
			continue
		}
		sp := math.Hypot(o.VX, o.VY)
		if sp < 1e-6 {
			continue
		}
		dx, dy := target.X-o.X, target.Y-o.Y
		tn := math.Hypot(dx, dy)
		if tn < 1e-6 {
			continue
		}
		ux, uy := turnToward(o.VX/sp, o.VY/sp, dx/tn, dy/tn, turnStep)
		ctx.Out <- unit.SetVelocity{UnitID: o.ID, VX: ux * sp, VY: uy * sp}
	}
}

func (a *原型机_整合) shoot(ctx unit.Context, s unit.Sense) {
	if s.Time < a.fireReadyAt {
		return
	}
	target := unit.Seek(s)
	if target == nil {
		return
	}
	dx, dy := target.X-s.Self.X, target.Y-s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	ux, uy := dx/n, dy/n
	roll := a.sample()
	ang := (roll*2 - 1) * cone
	c, sn := math.Cos(ang), math.Sin(ang)
	ux, uy = ux*c-uy*sn, ux*sn+uy*c
	gap := s.Self.Radius + roundRadius + 1.5
	ctx.Out <- unit.Spawn{
		Kind:    KindIntegratedRound,
		X:       s.Self.X + ux*gap,
		Y:       s.Self.Y + uy*gap,
		VX:      ux * roundSpeed,
		VY:      uy * roundSpeed,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	}
	a.fireReadyAt = s.Time + fireInterval
	ctx.Out <- unit.FX{
		Name:   "shot",
		UnitID: ctx.ID,
		Kind:   ctx.Kind,
		X:      s.Self.X,
		Y:      s.Self.Y,
		VX:     ux * roundSpeed,
		VY:     uy * roundSpeed,
		Slot:   s.Self.Slot,
	}
}

func (a *原型机_整合) sample() float64 {
	if a.roll != nil {
		return a.roll()
	}
	return a.rng.Float64()
}

func turnToward(ux, uy, tx, ty, maxRad float64) (float64, float64) {
	ang := math.Atan2(ux*ty-uy*tx, ux*tx+uy*ty)
	if ang > maxRad {
		ang = maxRad
	} else if ang < -maxRad {
		ang = -maxRad
	}
	c, s := math.Cos(ang), math.Sin(ang)
	return ux*c - uy*s, ux*s + uy*c
}

type 整合弹 struct {
	owner   uint64
	slot    int
	bounces int
}

func (b *整合弹) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Collision:
		b.onHit(ctx, e.Other)
	case unit.WallHit:
		b.bounces++
		if b.bounces >= roundBounces {
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		}
	}
}

func (b *整合弹) onHit(ctx unit.Context, other unit.Snapshot) {
	if other.ID == b.owner {
		return
	}
	if !unit.Hittable(other, b.slot) {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: other.ID, Amount: roundDamage}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}
