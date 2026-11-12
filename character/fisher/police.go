package 钓鱼佬

import (
	"math"

	"xqdj/internal/unit"
)

const (
	policeDispatchKind = "钓鱼佬出警"
	policeCarKind      = "钓鱼佬警车"
	policeOfficerKind  = "钓鱼佬警察"
	electricFishCount  = 3
	policeDelay        = 3.0
	policeDriveTime    = 1.25
	policeDoorDelay    = 0.2
	policeExitDistance = 18.0
	policeSpeed        = 50.0
	policeExitTime     = policeExitDistance / policeSpeed
	policeRadius       = fisherRadius
)

func registerPolice(p *unit.Pack) {
	p.Register(unit.Spec{
		Kind: policeDispatchKind, Role: unit.RoleHelper, Hidden: true, RequiresOwner: true, Vision: fisherVision,
	}, func(info unit.SpawnInfo) unit.Actor {
		at, _ := info.Data.(float64)
		return &policeDispatch{owner: info.OwnerID, at: at}
	})
	p.Register(unit.Spec{
		Kind: policeCarKind, Role: unit.RoleHelper, RequiresOwner: true, Radius: 20,
		Look: unit.Look{Color: "#233755", FX: []string{"fisher-police-car"}, Overlay: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		lane, _ := info.Data.(policeLane)
		return &policeCar{owner: info.OwnerID, lane: lane}
	})
	p.Register(unit.Spec{
		Kind: policeOfficerKind, Role: unit.RoleHelper, RequiresOwner: true, Radius: policeRadius, Speed: policeSpeed, Vision: fisherVision,
		Look: unit.Look{Color: "#498bd3", FX: []string{"fisher-police-officer"}, Overlay: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		exit, _ := info.Data.(policeExit)
		return &policeOfficer{owner: info.OwnerID, exit: exit}
	})
}

func (a *钓鱼佬) reportElectric(ctx unit.Context, now, x, y float64, n int) {
	if n < electricFishCount {
		return
	}
	ctx.Out <- unit.FX{Name: "electric", Kind: KindFisher, UnitID: ctx.ID, X: x, Y: y, Slot: a.slot}
	if a.reported {
		return
	}
	a.reported = true
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: policeDispatchKind, OwnerID: ctx.ID, Slot: a.slot, X: x, Y: y, Data: now + policeDelay,
	})
}

type policeDispatch struct {
	owner uint64
	at    float64
	sent  bool
}

func (p *policeDispatch) Handle(ctx unit.Context, event unit.Event) {
	s, ok := event.(unit.Sense)
	if !ok || p.sent || s.Time+1e-9 < p.at {
		return
	}
	p.sent = true
	if target, found := policeTarget(s, p.owner); found {
		lane := policeApproach(s.Field, target.X, target.Y, s.Time)
		ctx.Out <- unit.FX{Name: "police-siren", Kind: KindFisher, UnitID: ctx.ID, X: lane.x, Y: lane.y, Slot: target.Slot}
		ctx.Out <- ctx.Born(unit.Spawn{
			Kind: policeCarKind, OwnerID: p.owner, Slot: target.Slot,
			X: lane.x, Y: lane.y, VX: lane.parkX - lane.x, VY: lane.parkY - lane.y, Data: lane,
		})
	}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

type policeLane struct {
	x, y         float64
	parkX, parkY float64
	at           float64
}

func policeApproach(f unit.Field, targetX, targetY, at float64) policeLane {
	// 在真实场边找离目标最近的一条可停车路线，适用于非圆形和有内墙的场地。
	const samples = 72
	bestDistance := math.Inf(1)
	var best policeLane
	for i := 0; i < samples; i++ {
		x, y, nx, ny := f.SampleEdge((float64(i)+0.5)/samples, 0)
		parkX, parkY := x-nx*65, y-ny*65
		if !f.Walkable(parkX, parkY, policeRadius) ||
			!f.Walkable(x-nx*20, y-ny*20, policeRadius) ||
			!f.Walkable(x-nx*42, y-ny*42, policeRadius) {
			continue
		}
		distance := math.Hypot(parkX-targetX, parkY-targetY)
		if distance < bestDistance {
			bestDistance = distance
			best = policeLane{x + nx*34, y + ny*34, parkX, parkY, at}
		}
	}
	if math.IsInf(bestDistance, 1) {
		x, y, nx, ny := f.SampleEdge(0.25, 0)
		parkX, parkY := f.Clamp(x-nx*65, y-ny*65, policeRadius)
		return policeLane{x + nx*34, y + ny*34, parkX, parkY, at}
	}
	return best
}

type policeCar struct {
	owner uint64
	lane  policeLane
	out   bool
}

func (p *policeCar) Handle(ctx unit.Context, event unit.Event) {
	s, ok := event.(unit.Sense)
	if !ok || p.out {
		return
	}
	age := s.Time - p.lane.at
	t := math.Min(1, math.Max(0, age/policeDriveTime))
	progress := t * (2 - t)
	x, y := p.lane.parkX, p.lane.parkY
	if t < 1 || s.Self.X != x || s.Self.Y != y {
		ctx.Out <- unit.Teleport{UnitID: ctx.ID,
			X: p.lane.x + (x-p.lane.x)*progress, Y: p.lane.y + (y-p.lane.y)*progress,
		}
	}
	if age+1e-9 < policeDriveTime+policeDoorDelay {
		return
	}
	p.out = true
	dx, dy := p.lane.parkX-p.lane.x, p.lane.parkY-p.lane.y
	if length := math.Hypot(dx, dy); length > 0 {
		for _, side := range []float64{1, -1} {
			exitX := x - dy/length*side*policeExitDistance
			exitY := y + dx/length*side*policeExitDistance
			if s.Field.Walkable(exitX, exitY, policeRadius) {
				x, y = exitX, exitY
				break
			}
		}
	}
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: policeOfficerKind, OwnerID: p.owner, Slot: s.Self.Slot, X: p.lane.parkX, Y: p.lane.parkY,
		VX: (x - p.lane.parkX) / policeExitTime, VY: (y - p.lane.parkY) / policeExitTime,
		Data: policeExit{fromX: p.lane.parkX, fromY: p.lane.parkY, x: x, y: y, at: s.Time},
	})
}

type policeExit struct {
	fromX, fromY float64
	x, y         float64
	at           float64
}

type policeOfficer struct {
	owner    uint64
	exit     policeExit
	lastTime float64
	detour   unit.DetourHold
	arrested bool
}

func (p *policeOfficer) Handle(ctx unit.Context, event unit.Event) {
	s, ok := event.(unit.Sense)
	if !ok || p.arrested {
		return
	}
	target, ok := policeTarget(s, p.owner)
	if !ok {
		return
	}
	if age := s.Time - p.exit.at; p.exit.at > 0 && age < policeExitTime {
		t := math.Max(0, age/policeExitTime)
		ctx.Out <- unit.Teleport{UnitID: ctx.ID,
			X: p.exit.fromX + (p.exit.x-p.exit.fromX)*t, Y: p.exit.fromY + (p.exit.y-p.exit.fromY)*t,
		}
		p.lastTime = s.Time
		return
	}
	x, y := s.Self.X, s.Self.Y
	if math.Hypot(target.X-x, target.Y-y) <= target.Radius+policeRadius && policeContactClear(s, target) {
		p.arrested = true
		ctx.Out <- unit.FX{
			Name: "arrest", Kind: KindFisher, UnitID: target.ID, X: target.X, Y: target.Y,
			VX: x - target.X, VY: y - target.Y, Slot: target.Slot, Amount: target.Radius,
		}
		// 沿用战斗机离场和胜负判定：单挑结束，组队只移除本人。
		ctx.Out <- unit.Despawn{UnitID: target.ID}
		return
	}
	elapsed := math.Min(0.05, s.Time-p.lastTime)
	p.lastTime = s.Time
	tx, ty, hold := unit.Detour(x, y, policeRadius, target.X, target.Y, s.Walls, p.detour, s.Time)
	p.detour = hold
	dx, dy := tx-x, ty-y
	distance := math.Hypot(dx, dy)
	if elapsed <= 0 || distance < 1e-6 {
		return
	}
	stride := policeSpeed * elapsed
	step := math.Min(1, stride/distance)
	nextX, nextY := s.Field.Clamp(x+dx*step, y+dy*step, policeRadius)
	if math.Hypot(nextX-x, nextY-y) > stride+1e-9 {
		return
	}
	ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: nextX, Y: nextY}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: (nextX - x) / elapsed, VY: (nextY - y) / elapsed}
}

func policeContactClear(s unit.Sense, target unit.Snapshot) bool {
	x, y, _ := unit.Detour(s.Self.X, s.Self.Y, 0, target.X, target.Y, s.Walls, unit.DetourHold{}, s.Time)
	return x == target.X && y == target.Y
}

func policeTarget(s unit.Sense, id uint64) (unit.Snapshot, bool) {
	for _, u := range s.Nearby {
		if u.ID == id && u.Role == unit.RoleFighter && u.HP > 0 {
			return u, true
		}
	}
	return unit.Snapshot{}, false
}
