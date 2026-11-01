// 捆：随机巡游留下绳迹。敌人碰到绳子后会被拖上绳路，
// 沿绳路反方向滑向绳尾，途中持续受伤，离开绳尾才恢复自由。
package 捆

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const KindBundle = "捆"

const (
	bodyRadius = 18.0
	bodyHP     = 100.0
	bodySpeed  = 175.0
	bodyVision = 9999.0
	bodyColor  = "#b34c50"

	trailStep     = 9.0
	trailLife     = 4.5
	trailWidth    = 10.0
	trailTickDmg  = 1.0
	trailTickGap  = 0.2
	rideSpeed     = 190.0
	rideFreedom   = 0.9
	maxTrail      = 220
	rideTokenBase = uint64(0xb0a9d1e500000000)

	turnMin  = 1.0
	turnMax  = 2.2
	stateGap = 0.2
)

func init() {
	p := unit.NewPack(KindBundle, assets)
	p.Register(unit.Spec{
		Kind: KindBundle, Role: unit.RoleFighter,
		Radius: bodyRadius, MaxHP: bodyHP, Speed: bodySpeed, Vision: bodyVision,
		Fighter: true,
		Look:    unit.Look{Color: bodyColor, FX: []string{"bundle"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &捆{rng: unit.NewRNG(info.Stream)}
	})
}

type point struct {
	x, y float64
}

type ropeSegment struct {
	id     uint64
	ax, ay float64
	bx, by float64
	bornAt float64
}

type ropeRider struct {
	targetID  uint64
	route     []point
	total     float64
	progress  float64
	lastTime  float64
	nextHitAt float64
}

type 捆 struct {
	slot   int
	booted bool
	rng    *rand.Rand

	x, y   float64
	vx, vy float64

	nextTurnAt float64

	trail        []ropeSegment
	lastTrail    point
	hasLastTrail bool
	nextSegment  uint64
	riders       map[uint64]*ropeRider
	freedomAt    map[uint64]float64
	lastStateAt  float64
}

func (a *捆) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	sense, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	a.slot = sense.Self.Slot
	a.x, a.y = sense.Self.X, sense.Self.Y
	a.vx, a.vy = sense.Self.VX, sense.Self.VY
	if !a.booted {
		a.booted = true
		a.lastTrail = point{x: a.x, y: a.y}
		a.hasLastTrail = true
		a.riders = make(map[uint64]*ropeRider)
		a.freedomAt = make(map[uint64]float64)
		ctx.Out <- unit.FX{Name: "bundle-reset", Kind: KindBundle, UnitID: ctx.ID, Slot: a.slot}
	}

	a.randomSteer(ctx, sense)
	a.tickRiders(ctx, sense)
	a.expireTrail(ctx, sense.Time)
	a.extendTrail(ctx, sense.Time)
	a.catchRiders(ctx, sense)
	a.emitState(ctx, sense.Time)
}

func (a *捆) randomSteer(ctx unit.Context, sense unit.Sense) {
	if sense.Time+1e-9 < a.nextTurnAt {
		return
	}
	ang := 0.0
	if a.rng != nil {
		ang = a.rng.Float64() * 2 * math.Pi
		a.nextTurnAt = sense.Time + turnMin + a.rng.Float64()*(turnMax-turnMin)
	} else {
		a.nextTurnAt = sense.Time + 1.5
	}
	ux, uy := math.Cos(ang), math.Sin(ang)
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: ux * bodySpeed, VY: uy * bodySpeed}
	ctx.Out <- unit.SetFSDirection{UnitID: ctx.ID, VX: ux, VY: uy}
}

func (a *捆) extendTrail(ctx unit.Context, now float64) {
	if a.hasLastTrail && math.Hypot(a.x-a.lastTrail.x, a.y-a.lastTrail.y) < trailStep {
		return
	}
	a.nextSegment++
	seg := ropeSegment{
		id: a.nextSegment,
		ax: a.lastTrail.x, ay: a.lastTrail.y,
		bx: a.x, by: a.y, bornAt: now,
	}
	a.trail = append(a.trail, seg)
	a.lastTrail = point{x: a.x, y: a.y}
	a.hasLastTrail = true
	if len(a.trail) > maxTrail {
		a.trail = append([]ropeSegment(nil), a.trail[len(a.trail)-maxTrail:]...)
	}
	ctx.Out <- unit.FX{
		Name: "bundle-trail-add", Kind: KindBundle, UnitID: ctx.ID, Slot: a.slot,
		X: seg.ax, Y: seg.ay, VX: seg.bx, VY: seg.by, Amount: float64(seg.id),
	}
}

func (a *捆) expireTrail(ctx unit.Context, now float64) {
	if len(a.riders) > 0 {
		// 有人被拖着时保持整条可见绳路，避免受害者在绳尾中途消失。
		return
	}
	kept := a.trail[:0]
	for _, seg := range a.trail {
		if now+1e-9 < seg.bornAt+trailLife {
			kept = append(kept, seg)
			continue
		}
		ctx.Out <- unit.FX{
			Name: "bundle-trail-remove", Kind: KindBundle, UnitID: ctx.ID, Slot: a.slot,
			Amount: float64(seg.id),
		}
	}
	a.trail = kept
}

func (a *捆) catchRiders(ctx unit.Context, sense unit.Sense) {
	for i := range sense.Nearby {
		o := &sense.Nearby[i]
		if !unit.Hittable(*o, a.slot) || a.riders[o.ID] != nil ||
			sense.Time+1e-9 < a.freedomAt[o.ID] {
			continue
		}
		segIndex := contactSegment(*o, a.trail)
		if segIndex < 0 {
			continue
		}
		route := rideRoute(point{x: o.X, y: o.Y}, a.trail, segIndex)
		total := routeLength(route)
		if total < 1 {
			continue
		}
		a.riders[o.ID] = &ropeRider{
			targetID: o.ID, route: route, total: total,
			lastTime: sense.Time, nextHitAt: sense.Time,
		}
		ctx.Out <- unit.FX{
			Name: "bundle-rope-catch", Kind: KindBundle, UnitID: ctx.ID, Slot: a.slot,
			X: o.X, Y: o.Y, VX: float64(o.ID),
		}
	}
}

func (a *捆) tickRiders(ctx unit.Context, sense unit.Sense) {
	if len(a.riders) == 0 {
		return
	}
	visible := make(map[uint64]unit.Snapshot, len(sense.Nearby))
	for i := range sense.Nearby {
		visible[sense.Nearby[i].ID] = sense.Nearby[i]
	}
	for id, rider := range a.riders {
		o, ok := visible[id]
		if !ok {
			delete(a.riders, id)
			continue
		}
		dt := sense.Time - rider.lastTime
		if dt < 0 || dt > 0.08 {
			dt = 1.0 / 60.0
		}
		rider.lastTime = sense.Time
		rider.progress += rideSpeed * dt
		if rider.progress >= rider.total {
			tail := rider.route[len(rider.route)-1]
			a.freedomAt[id] = sense.Time + rideFreedom
			delete(a.riders, id)
			ctx.Out <- unit.RemoveFS{UnitID: id, Token: rideTokenBase ^ id}
			ctx.Out <- unit.FX{
				Name: "bundle-rope-release", Kind: KindBundle, UnitID: ctx.ID, Slot: a.slot,
				X: tail.x, Y: tail.y, VX: float64(id),
			}
			continue
		}
		at := pointAlong(rider.route, rider.progress+16)
		dx, dy := at.x-o.X, at.y-o.Y
		length := math.Hypot(dx, dy)
		if length > 1e-6 {
			ctx.Out <- unit.AddFS{
				UnitID: id, DX: dx / length, DY: dy / length,
				BaseSpeed: rideSpeed, OnWall: true,
				Token: rideTokenBase ^ id, ExpiresAt: sense.Time + 0.16,
			}
		}
		if sense.Time+1e-9 >= rider.nextHitAt {
			rider.nextHitAt = sense.Time + trailTickGap
			ctx.Out <- unit.Damage{From: ctx.ID, To: id, Amount: trailTickDmg, NoFreeze: true}
			ctx.Out <- unit.FX{
				Name: "bundle-rope-burn", Kind: KindBundle, UnitID: ctx.ID, Slot: a.slot,
				X: o.X, Y: o.Y, VX: float64(id),
			}
		}
	}
}

func contactSegment(o unit.Snapshot, trail []ropeSegment) int {
	for i := len(trail) - 1; i >= 0; i-- {
		seg := trail[i]
		if distanceToSegment(o.X, o.Y, seg.ax, seg.ay, seg.bx, seg.by) <= o.Radius+trailWidth {
			return i
		}
	}
	return -1
}

func rideRoute(at point, trail []ropeSegment, index int) []point {
	seg := trail[index]
	start := closestToSegment(at.x, at.y, seg.ax, seg.ay, seg.bx, seg.by)
	route := []point{at, start}
	for i := index; i >= 0; i-- {
		p := point{x: trail[i].ax, y: trail[i].ay}
		if math.Hypot(p.x-route[len(route)-1].x, p.y-route[len(route)-1].y) > 0.5 {
			route = append(route, p)
		}
	}
	return route
}

func closestToSegment(px, py, ax, ay, bx, by float64) point {
	dx, dy := bx-ax, by-ay
	l2 := dx*dx + dy*dy
	if l2 < 1e-12 {
		return point{x: ax, y: ay}
	}
	t := ((px-ax)*dx + (py-ay)*dy) / l2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return point{x: ax + dx*t, y: ay + dy*t}
}

func routeLength(route []point) float64 {
	total := 0.0
	for i := 1; i < len(route); i++ {
		total += math.Hypot(route[i].x-route[i-1].x, route[i].y-route[i-1].y)
	}
	return total
}

func pointAlong(route []point, distance float64) point {
	if len(route) == 0 {
		return point{}
	}
	remain := distance
	for i := 1; i < len(route); i++ {
		dx, dy := route[i].x-route[i-1].x, route[i].y-route[i-1].y
		length := math.Hypot(dx, dy)
		if length < 1e-9 {
			continue
		}
		if remain <= length {
			t := remain / length
			return point{x: route[i-1].x + dx*t, y: route[i-1].y + dy*t}
		}
		remain -= length
	}
	return route[len(route)-1]
}

func distanceToSegment(px, py, ax, ay, bx, by float64) float64 {
	p := closestToSegment(px, py, ax, ay, bx, by)
	return math.Hypot(px-p.x, py-p.y)
}

func (a *捆) emitState(ctx unit.Context, now float64) {
	if now+1e-9 < a.lastStateAt+stateGap {
		return
	}
	a.lastStateAt = now
	ctx.Out <- unit.FX{
		Name: "bundle-state", Kind: KindBundle, UnitID: ctx.ID, Slot: a.slot,
		X: a.x, Y: a.y, Amount: float64(len(a.riders)), VX: float64(len(a.trail)),
	}
}
