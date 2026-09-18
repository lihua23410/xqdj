package 教父

import (
	"math"
	"xqdj/internal/unit"
)

type leaveDrive struct {
	on              bool
	pauseUntil      float64
	fadeUntil       float64
	dx, dy, x, y, r float64
}

func (l *leaveDrive) note(x, y, r float64) {
	l.x, l.y, l.r = x, y, r
}

func (l *leaveDrive) begin(ctx unit.Context, now float64) {
	if l.on {
		return
	}
	l.on = true
	l.pauseUntil = now + leavePause
	l.dx, l.dy = nearestEdgeDir(l.x, l.y)
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
	ctx.Out <- unit.FX{Name: "aim", Kind: KindGodfather, UnitID: ctx.ID, Amount: -1}
}

func (l *leaveDrive) handle(ctx unit.Context, ev unit.Event) bool {
	if !l.on {
		return false
	}
	switch e := ev.(type) {
	case unit.WallHit:
		if l.fadeUntil > 0 || e.Time+1e-9 < l.pauseUntil {
			return true
		}
		if l.dx*e.NX+l.dy*e.NY <= 0.35 {
			return true
		}
		l.fadeUntil = e.Time + leaveFade
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
		ctx.Out <- unit.FX{
			Name: "leave", Kind: KindGodfather, UnitID: ctx.ID,
			X: l.x, Y: l.y, Slot: 0,
		}
	case unit.Sense:
		l.note(e.Self.X, e.Self.Y, e.Self.Radius)
		if e.Time+1e-9 < l.pauseUntil {
			ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
			return true
		}
		if l.fadeUntil > 0 {
			if e.Time+1e-9 >= l.fadeUntil {
				ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			} else {
				ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
			}
			return true
		}
		ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: l.dx * leaveSpeed, VY: l.dy * leaveSpeed}
	}
	return true
}

type 暗杀者 struct {
	owner      uint64
	slot       int
	attacks    int
	booted     bool
	cdUntil    float64
	dashing    bool
	dashLeft   float64
	aimX, aimY float64
	hit        map[uint64]bool
	lastT      float64
	leave      leaveDrive
}

func (a *暗杀者) Handle(ctx unit.Context, ev unit.Event) {
	if a.leave.handle(ctx, ev) {
		return
	}
	if unit.AcceptHit(ctx, ev) {
		return
	}
	switch e := ev.(type) {
	case unit.WallHit:
		a.endDash(ctx, e.Time, e.NX, e.NY, true)
	case unit.Collision:
		a.onHit(ctx, e)
	case unit.Sense:
		a.tick(ctx, e)
	}
}

func (a *暗杀者) tick(ctx unit.Context, s unit.Sense) {
	a.leave.note(s.Self.X, s.Self.Y, s.Self.Radius)
	if !a.booted {
		a.booted = true
		a.cdUntil = s.Time + assassinCD
		a.lastT = s.Time
		a.hit = map[uint64]bool{}
	}
	dt := s.Time - a.lastT
	if dt < 0 {
		dt = 0
	}
	a.lastT = s.Time
	pickupDrug(ctx, s, func() { a.attacks++ })
	if a.dashing {
		a.dashLeft -= assassinSpeed * dt
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: a.aimX * assassinSpeed, VY: a.aimY * assassinSpeed}
		if a.dashLeft <= 0 {
			a.endDash(ctx, s.Time, 0, 0, false)
		}
		return
	}
	if a.attacks <= 0 {
		a.leave.begin(ctx, s.Time)
		return
	}
	if s.Time+1e-9 < a.cdUntil {
		return
	}
	en := enemyFighter(s)
	if en == nil {
		return
	}
	dx, dy := en.X-s.Self.X, en.Y-s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	a.attacks--
	a.dashing = true
	a.dashLeft = assassinDash
	a.aimX, a.aimY = dx/n, dy/n
	a.hit = map[uint64]bool{}
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: a.aimX * assassinSpeed, VY: a.aimY * assassinSpeed}
}

func (a *暗杀者) onHit(ctx unit.Context, e unit.Collision) {
	if !a.dashing || !unit.EnemyTarget(e, a.slot) {
		return
	}
	if a.hit[e.Other.ID] {
		return
	}
	a.hit[e.Other.ID] = true
	ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: assassinDmg}
}

func (a *暗杀者) endDash(ctx unit.Context, now, nx, ny float64, wall bool) {
	if !a.dashing {
		return
	}
	a.dashing = false
	a.cdUntil = now + assassinCD
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: false}
	ux, uy := a.aimX, a.aimY
	if wall {
		ux, uy = reflectDir(ux, uy, nx, ny)
		a.aimX, a.aimY = ux, uy
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: ux * minionCruise, VY: uy * minionCruise}
	if a.attacks <= 0 {
		a.leave.begin(ctx, now)
	}
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

type 狙击者 struct {
	owner   uint64
	slot    int
	shots   int
	aimFrom float64
	booted  bool
	leave   leaveDrive
}

func (a *狙击者) Handle(ctx unit.Context, ev unit.Event) {
	if a.leave.handle(ctx, ev) {
		return
	}
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	a.leave.note(s.Self.X, s.Self.Y, s.Self.Radius)
	if !a.booted {
		a.booted = true
		a.aimFrom = s.Time
	}
	pickupDrug(ctx, s, func() { a.shots++ })
	en := enemyFighter(s)
	if en == nil {
		return
	}
	dx, dy := en.X-s.Self.X, en.Y-s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	ux, uy := dx/n, dy/n
	prog := (s.Time - a.aimFrom) / sniperAim
	if prog > 1 {
		prog = 1
	}
	if prog < 0 {
		prog = 0
	}
	ctx.Out <- unit.FX{
		Name: "aim", Kind: KindGodfather, UnitID: ctx.ID, Slot: s.Self.Slot,
		X: s.Self.X, Y: s.Self.Y, VX: en.X, VY: en.Y, Amount: prog,
	}
	if s.Time+1e-9 < a.aimFrom+sniperAim {
		return
	}
	if a.shots <= 0 {
		a.leave.begin(ctx, s.Time)
		return
	}
	gap := s.Self.Radius + shotRadius + 1.5
	ctx.Out <- unit.Spawn{
		Kind: KindGodfatherShot,
		X:    s.Self.X + ux*gap, Y: s.Self.Y + uy*gap,
		VX: ux * shotSpeed, VY: uy * shotSpeed,
		OwnerID: a.owner, Slot: a.slot,
	}
	a.shots--
	if a.shots <= 0 {
		a.leave.begin(ctx, s.Time)
		return
	}
	a.aimFrom = s.Time
}

type 狙击弹 struct {
	slot int
	hit  map[uint64]bool
}

func (b *狙击弹) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Collision:
		if !unit.EnemyTarget(e, b.slot) {
			return
		}
		if b.hit[e.Other.ID] {
			return
		}
		if b.hit == nil {
			b.hit = map[uint64]bool{}
		}
		b.hit[e.Other.ID] = true
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: shotDmg}
	case unit.WallHit:
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

type 贩卖者 struct {
	owner  uint64
	slot   int
	left   int
	nextAt float64
	booted bool
	said   map[uint64]bool
	leave  leaveDrive
}

func (a *贩卖者) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	if s, ok := ev.(unit.Sense); ok {
		a.leave.note(s.Self.X, s.Self.Y, s.Self.Radius)
		if !a.booted {
			a.booted = true
			a.nextAt = s.Time + dealerGap
			a.said = map[uint64]bool{}
		}
		a.refuse(ctx, s)
	}
	if a.leave.handle(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if s.Time+1e-9 < a.nextAt {
		return
	}
	if a.left <= 0 {
		a.leave.begin(ctx, s.Time)
		return
	}
	ctx.Out <- unit.Spawn{
		Kind: KindDrug, X: s.Self.X, Y: s.Self.Y,
		OwnerID: a.owner, Slot: a.slot,
	}
	a.left--
	if a.left <= 0 {
		a.leave.begin(ctx, s.Time)
		return
	}
	a.nextAt = s.Time + dealerGap
}

func (a *贩卖者) refuse(ctx unit.Context, s unit.Sense) {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind != KindDrug || o.Slot != s.Self.Slot {
			continue
		}
		if !overlap(s.Self, *o) {
			continue
		}
		if a.said[o.ID] {
			continue
		}
		a.said[o.ID] = true
		ctx.Out <- unit.FX{
			Name: "scream", Kind: KindGodfather, UnitID: ctx.ID, Slot: s.Self.Slot,
			X: s.Self.X, Y: s.Self.Y,
		}
	}
}

type 药物 struct{}

func (药物) Handle(unit.Context, unit.Event) {}
