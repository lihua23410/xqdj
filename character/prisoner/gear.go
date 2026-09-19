package 囚徒

import (
	"math"
	"math/rand/v2"
	"xqdj/internal/unit"
)

const (
	cageInner   = 48.0
	cageWall    = 6.0
	cageRing    = cageInner + cageWall
	cageOuter   = cageInner + 2*cageWall
	cageFit     = cageOuter
	cageColor   = "#3a3a3a"
	cageDamage  = 2.0
	gallowsR    = 16.5
	gallowsDmg  = 1.0
	gallowsCD   = 0.4
	chairR      = 16.5
	currentNMin = 3
	currentNMax = 6
	currentBend = 3
	currentMinL = 70.0
	currentMaxL = 100.0
	currentW    = 4.0
	currentDmg  = 5.0
	currentCD   = 0.5
	chainHitR   = 3.0
)

type 囚笼 struct {
	owner uint64
	slot  int
	ready bool
	born  float64
}

func (a *囚笼) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !a.ready {
		a.ready = true
		a.born = s.Time
		placeCage(ctx, s.Self.X, s.Self.Y, a.owner, a.slot)
	}
	if s.Time+1e-9 >= a.born+gearLife {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

func placeCage(ctx unit.Context, cx, cy float64, owner uint64, slot int) {
	const n = 8
	for i := 0; i < n; i++ {
		a0 := (float64(i) - 0.2) * 2 * math.Pi / n
		a1 := (float64(i) + 1.2) * 2 * math.Pi / n
		ctx.Out <- unit.PlaceWall{
			OwnerID: owner, Slot: slot, Kind: KindCage,
			X1: cx + math.Cos(a0)*cageRing, Y1: cy + math.Sin(a0)*cageRing,
			X2: cx + math.Cos(a1)*cageRing, Y2: cy + math.Sin(a1)*cageRing,
			Radius: cageWall, Life: gearLife, Amount: cageDamage,
		}
	}
}

type 绞刑架 struct {
	owner    uint64
	slot     int
	ready    bool
	born     float64
	locked   uint64
	lockKind string
	lockVX   float64
	lockVY   float64
	nextTick float64
}

func (a *绞刑架) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !a.ready {
		a.ready = true
		a.born = s.Time
		ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
	}
	if a.locked == 0 {
		if o := overlapEnemy(s); o != nil {
			a.locked = o.ID
			a.lockKind = o.Kind
			a.lockVX, a.lockVY = o.VX, o.VY
			a.nextTick = s.Time
		}
	}
	if a.locked != 0 {
		a.hold(ctx, s.Time)
	}
	if s.Time+1e-9 >= a.born+gearLife {
		if a.locked != 0 {
			a.release(ctx)
		}
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

func (a *绞刑架) hold(ctx unit.Context, now float64) {
	ctx.Out <- unit.Stun{UnitID: a.locked, Hold: true}
	ctx.Out <- unit.Stand{UnitID: a.locked, Hold: true}
	ctx.Out <- unit.SetVelocity{UnitID: a.locked, VX: 0, VY: 0}
	if now+1e-9 < a.nextTick {
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: a.locked, Amount: gallowsDmg}
	a.nextTick = now + gallowsCD
}

func (a *绞刑架) release(ctx unit.Context) {
	ctx.Out <- unit.Stun{UnitID: a.locked, Hold: false}
	ctx.Out <- unit.Stand{UnitID: a.locked, Hold: false}
	vx, vy := a.lockVX, a.lockVY
	if math.Hypot(vx, vy) < 1e-6 {
		speed := prisonerCruise
		if spec, ok := unit.Lookup(a.lockKind); ok && spec.Speed > 0 {
			speed = spec.Speed
		}
		vx, vy = speed, 0
	}
	ctx.Out <- unit.SetVelocity{UnitID: a.locked, VX: vx, VY: vy}
}

func overlapEnemy(s unit.Sense) *unit.Snapshot {
	sr := s.Self.Radius
	if sr < 1e-9 {
		sr = gallowsR
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		r := o.Radius
		if r < 1e-9 {
			r = prisonerRadius
		}
		if math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) <= sr+r {
			return o
		}
	}
	return nil
}

type 电椅 struct {
	owner uint64
	slot  int
	ready bool
	born  float64
	next  float64
	hitAt map[uint64]float64
	rng   *rand.Rand
}

func (a *电椅) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !a.ready {
		a.ready = true
		a.born = s.Time
		a.next = s.Time
		ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
	}
	if s.Time+1e-9 >= a.next {
		a.burst(ctx, s)
		a.next = s.Time + currentCD
	}
	if s.Time+1e-9 >= a.born+gearLife {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

func (a *电椅) burst(ctx unit.Context, s unit.Sense) {
	rng := a.rng
	if rng == nil {
		rng = rand.New(rand.NewPCG(uint64(s.Time*1e6)+s.Self.ID, 3))
	}
	a.strike(ctx, s, genBolts(s.Self.X, s.Self.Y, rng))
}

func (a *电椅) strike(ctx unit.Context, s unit.Sense, bolts [][]vec) {
	if a.hitAt == nil {
		a.hitAt = map[uint64]float64{}
	}
	half := currentW / 2
	var hook, body *unit.Snapshot
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind == KindHook && o.OwnerID == a.owner {
			hook = o
		}
		if o.Role == unit.RoleFighter && o.ID == a.owner {
			body = o
		}
	}
	shocked := false
	for _, bolt := range bolts {
		for i := 0; i+1 < len(bolt); i++ {
			p, q := bolt[i], bolt[i+1]
			ctx.Out <- unit.FX{
				Name: "bolt", Kind: ctx.Kind, UnitID: ctx.ID, Slot: s.Self.Slot,
				X: p.x, Y: p.y, VX: q.x, VY: q.y,
			}
			if !shocked && hook != nil && body != nil && chainStruck(p, q, *body, *hook, half) {
				shocked = true
				ctx.Out <- unit.StackMark{UnitID: a.owner, Kind: shockKind, Delta: 1}
			}
			for j := range s.Nearby {
				o := &s.Nearby[j]
				if !unit.Hittable(*o, s.Self.Slot) || o.ID == a.owner {
					continue
				}
				if !segHits(p.x, p.y, q.x, q.y, o.X, o.Y, o.Radius, half) {
					continue
				}
				if at, ok := a.hitAt[o.ID]; ok && s.Time+1e-9 < at+currentCD {
					continue
				}
				a.hitAt[o.ID] = s.Time
				ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: currentDmg}
			}
		}
	}
}

func chainStruck(p, q vec, body, hook unit.Snapshot, half float64) bool {
	return segsClose(p.x, p.y, q.x, q.y, body.X, body.Y, hook.X, hook.Y, half+chainHitR)
}

func genBolts(cx, cy float64, rng *rand.Rand) [][]vec {
	want := currentNMin + rng.IntN(currentNMax-currentNMin+1)
	out := make([][]vec, 0, want)
	var used [][4]float64
	for n := 0; n < 120 && len(out) < want; n++ {
		pts := genBolt(cx, cy, rng)
		if boltCrosses(pts, used) {
			continue
		}
		out = append(out, pts)
		for i := 0; i+1 < len(pts); i++ {
			used = append(used, [4]float64{pts[i].x, pts[i].y, pts[i+1].x, pts[i+1].y})
		}
	}
	return out
}

func genBolt(cx, cy float64, rng *rand.Rand) []vec {
	bends := 1 + rng.IntN(currentBend)
	nseg := bends + 1
	total := currentMinL + rng.Float64()*(currentMaxL-currentMinL)
	seg := total / float64(nseg)
	ang := rng.Float64() * 2 * math.Pi
	x := cx + math.Cos(ang)*chairR
	y := cy + math.Sin(ang)*chairR
	pts := []vec{{x, y}}
	for i := 0; i < nseg; i++ {
		if i > 0 {
			ang += (rng.Float64()*2 - 1) * (math.Pi * 0.7)
		}
		x += math.Cos(ang) * seg
		y += math.Sin(ang) * seg
		pts = append(pts, vec{x, y})
	}
	return pts
}

func boltCrosses(pts []vec, used [][4]float64) bool {
	for i := 0; i+1 < len(pts); i++ {
		ax, ay, bx, by := pts[i].x, pts[i].y, pts[i+1].x, pts[i+1].y
		for _, u := range used {
			if properIntersect(ax, ay, bx, by, u[0], u[1], u[2], u[3]) {
				return true
			}
		}
	}
	return false
}

func properIntersect(ax, ay, bx, by, cx, cy, dx, dy float64) bool {
	d1 := cross(bx-ax, by-ay, cx-ax, cy-ay)
	d2 := cross(bx-ax, by-ay, dx-ax, dy-ay)
	d3 := cross(dx-cx, dy-cy, ax-cx, ay-cy)
	d4 := cross(dx-cx, dy-cy, bx-cx, by-cy)
	return d1*d2 < -1e-12 && d3*d4 < -1e-12
}

func cross(ax, ay, bx, by float64) float64 { return ax*by - ay*bx }

func segsClose(ax, ay, bx, by, cx, cy, dx, dy, pad float64) bool {
	if segHits(ax, ay, bx, by, cx, cy, 0, pad) || segHits(ax, ay, bx, by, dx, dy, 0, pad) {
		return true
	}
	if segHits(cx, cy, dx, dy, ax, ay, 0, pad) || segHits(cx, cy, dx, dy, bx, by, 0, pad) {
		return true
	}
	return properIntersect(ax, ay, bx, by, cx, cy, dx, dy)
}
