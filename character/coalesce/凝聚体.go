// 凝聚体把生命、体积、速度和弹药统一成质量。
// 受伤会崩下可夺回的凝块；主动开火会把当前质量压成会飞回来的重弹。
// 质量越低身体越小、速度越快，但每次出手也越接近解体的边缘。
package 凝聚体

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const (
	KindCoalesce = "凝聚体"
	KindShot     = "凝聚体压缩弹"

	KindMassSmall  = "凝聚体小凝块"
	KindMassMedium = "凝聚体中凝块"
	KindMassHeavy  = "凝聚体重凝块"
	KindMassShot   = "凝聚体弹块"
)

const (
	bodyHP       = 100.0
	bodyRadius   = 28.0
	minRadius    = 11.0
	bodyCruise   = 108.0
	maxCruise    = 208.0
	bodyVision   = 9999.0
	bodyColor    = "#74d8c6"
	shotColor    = "#b6fff0"
	chunkColor   = "#67c9b8"
	shotRadius   = 10.0
	shotSpeed    = 450.0
	shotDamage   = 12.0
	shotCost     = 12.0
	minShotHP    = 24.0
	shotInterval = 1.8
	shotLife     = 3.4
	shotBounces  = 2
	chunkLife    = 9.0
	maxChunks    = 4
	shedGap      = 0.22
	contactGap   = 0.75
	steerGap     = 0.36
	denySlow     = 0.78
	denyTime     = 2.8
	chunkToken   = uint64(0xc0a15eed00000000)
)

type chunkDef struct {
	kind   string
	value  float64
	radius float64
}

var chunkDefs = map[string]chunkDef{
	KindMassSmall:  {kind: KindMassSmall, value: 3, radius: 6},
	KindMassMedium: {kind: KindMassMedium, value: 6, radius: 9},
	KindMassHeavy:  {kind: KindMassHeavy, value: 10, radius: 12},
	KindMassShot:   {kind: KindMassShot, value: shotCost, radius: 13},
}

var chunkList = []chunkDef{
	chunkDefs[KindMassSmall],
	chunkDefs[KindMassMedium],
	chunkDefs[KindMassHeavy],
	chunkDefs[KindMassShot],
}

func init() {
	p := unit.NewPack(KindCoalesce, assets)
	p.Register(unit.Spec{
		Kind: KindCoalesce, Role: unit.RoleFighter,
		Radius: bodyRadius, MaxHP: bodyHP, Speed: bodyCruise, Vision: bodyVision,
		Fighter: true,
		Look:    unit.Look{Color: bodyColor, Glow: true, FX: []string{"coalesce"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &凝聚体{}
	})

	for _, def := range chunkList {
		def := def
		p.Register(unit.Spec{
			Kind: def.kind, Role: unit.RoleHelper,
			Radius: def.radius, MaxHP: 1, Speed: 0, Vision: bodyVision,
			Nonsolid: true,
			Look:     unit.Look{Color: chunkColor, FX: []string{"coalesce-chunk"}},
		}, func(unit.SpawnInfo) unit.Actor {
			return &massChunk{bornAt: -1}
		})
	}

	p.Register(unit.Spec{
		Kind: KindShot, Role: unit.RoleProjectile,
		Radius: shotRadius, MaxHP: 1, Speed: shotSpeed, Vision: bodyVision,
		Look: unit.Look{Color: shotColor, Trail: true, Glow: true, FX: []string{"coalesce-shot"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &massShot{owner: info.OwnerID, slot: info.Slot, bornAt: -1}
	})
}

type point struct {
	x, y float64
}

type 凝聚体 struct {
	booted bool
	slot   int
	rng    *rand.Rand

	x, y       float64
	radius     float64
	hp         float64
	field      unit.Field
	lastCruise float64

	source       map[uint64]point
	knownChunks  int
	nextShotAt   float64
	nextSteerAt  float64
	nextContact  float64
	lastShedTime float64
}

func (a *凝聚体) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		a.onIncoming(ctx, e)
	case unit.Sense:
		a.onSense(ctx, e)
	case unit.Collision:
		a.onCollision(ctx, e)
	}
}

func (a *凝聚体) onSense(ctx unit.Context, sense unit.Sense) {
	a.slot = sense.Self.Slot
	a.x, a.y = sense.Self.X, sense.Self.Y
	a.radius = sense.Self.Radius
	a.hp = sense.Self.HP
	a.field = sense.Field
	if !a.booted {
		a.booted = true
		a.rng = unit.NewRNG(ctx.Stream)
		a.nextShotAt = sense.Time + 1.1
		a.nextSteerAt = sense.Time
	}
	if a.source == nil {
		a.source = make(map[uint64]point)
	}
	for i := range sense.Nearby {
		o := &sense.Nearby[i]
		a.source[o.ID] = point{x: o.X, y: o.Y}
	}

	a.syncMass(ctx, sense)
	a.knownChunks = a.settleChunks(ctx, sense)
	target := unit.Seek(sense)
	a.steer(ctx, sense, target)
	a.maybeFire(ctx, sense, target)
}

func massGeometry(hp float64) (radius, cruise float64) {
	frac := hp / bodyHP
	if frac < 0 {
		frac = 0
	} else if frac > 1 {
		frac = 1
	}
	radius = minRadius + (bodyRadius-minRadius)*math.Sqrt(frac)
	cruise = bodyCruise + (maxCruise-bodyCruise)*(1-frac)
	return radius, cruise
}

func (a *凝聚体) syncMass(ctx unit.Context, sense unit.Sense) {
	wantRadius, wantCruise := massGeometry(sense.Self.HP)
	if math.Abs(sense.Self.Radius-wantRadius) > 0.05 {
		ctx.Out <- unit.SetRadius{UnitID: ctx.ID, Radius: wantRadius}
	}
	if a.lastCruise == 0 || math.Abs(a.lastCruise-wantCruise) > 0.5 {
		ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: wantCruise}
		a.lastCruise = wantCruise
	}
}

func (a *凝聚体) settleChunks(ctx unit.Context, sense unit.Sense) int {
	alive := 0
	for i := range sense.Nearby {
		chunk := &sense.Nearby[i]
		def, ok := chunkDefs[chunk.Kind]
		if !ok || chunk.OwnerID != sense.Self.ID {
			continue
		}

		selfDistance := math.Hypot(chunk.X-a.x, chunk.Y-a.y)
		if selfDistance <= sense.Self.Radius+def.radius+7 {
			ctx.Out <- unit.Heal{UnitID: ctx.ID, Amount: def.value}
			ctx.Out <- unit.Despawn{UnitID: chunk.ID}
			ctx.Out <- unit.FX{
				Name: "coalesce-absorb", Kind: KindCoalesce, UnitID: ctx.ID, Slot: a.slot,
				X: chunk.X, Y: chunk.Y, Amount: def.value,
			}
			continue
		}

		deniedBy := uint64(0)
		for j := range sense.Nearby {
			foe := &sense.Nearby[j]
			if !unit.Hittable(*foe, a.slot) {
				continue
			}
			if math.Hypot(chunk.X-foe.X, chunk.Y-foe.Y) <= def.radius+foe.Radius+5 {
				deniedBy = foe.ID
				break
			}
		}
		if deniedBy != 0 {
			foe := findSnapshot(sense.Nearby, deniedBy)
			ctx.Out <- unit.Despawn{UnitID: chunk.ID}
			ctx.Out <- unit.AddFSComponent{
				UnitID: deniedBy, Zone: unit.FSZoneM, Token: chunkToken,
				Value: denySlow, ExpiresAt: sense.Time + denyTime,
			}
			ctx.Out <- unit.FX{
				Name: "coalesce-deny", Kind: KindCoalesce, UnitID: ctx.ID, Slot: a.slot,
				X: chunk.X, Y: chunk.Y, Amount: def.value,
				VX: float64(deniedBy), VY: foe.Y,
			}
			continue
		}
		alive++
	}
	return alive
}

func findSnapshot(list []unit.Snapshot, id uint64) unit.Snapshot {
	for i := range list {
		if list[i].ID == id {
			return list[i]
		}
	}
	return unit.Snapshot{}
}

func (a *凝聚体) steer(ctx unit.Context, sense unit.Sense, target *unit.Snapshot) {
	if sense.Time+1e-9 < a.nextSteerAt {
		return
	}
	a.nextSteerAt = sense.Time + steerGap
	dx, dy := 0.0, 0.0

	if a.hp < sense.Self.MaxHP-0.5 {
		if chunk, ok := nearestChunk(sense, sense.Self.ID, a.x, a.y); ok {
			dx, dy = chunk.X-a.x, chunk.Y-a.y
		}
	}
	if math.Hypot(dx, dy) < 1e-6 && target != nil {
		tx, ty := target.X-a.x, target.Y-a.y
		distance := math.Hypot(tx, ty)
		switch {
		case distance > 245:
			dx, dy = tx, ty
		case distance < 118:
			dx, dy = -tx, -ty
		case distance > 1e-6:
			dx, dy = -ty, tx
		}
	}
	if math.Hypot(dx, dy) < 1e-6 && math.Hypot(sense.Self.VX, sense.Self.VY) > 1e-6 {
		dx, dy = sense.Self.VX, sense.Self.VY
	}
	length := math.Hypot(dx, dy)
	if length < 1e-6 {
		return
	}
	dx, dy = dx/length, dy/length
	_, speed := massGeometry(sense.Self.HP)
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: dx * speed, VY: dy * speed}
	ctx.Out <- unit.SetFSDirection{UnitID: ctx.ID, VX: dx, VY: dy}
}

func nearestChunk(sense unit.Sense, owner uint64, x, y float64) (unit.Snapshot, bool) {
	best := unit.Snapshot{}
	bestDistance := math.MaxFloat64
	for i := range sense.Nearby {
		o := &sense.Nearby[i]
		if o.OwnerID != owner {
			continue
		}
		if _, ok := chunkDefs[o.Kind]; !ok {
			continue
		}
		d := math.Hypot(o.X-x, o.Y-y)
		if d < bestDistance {
			bestDistance, best = d, *o
		}
	}
	return best, bestDistance < math.MaxFloat64
}

func (a *凝聚体) maybeFire(ctx unit.Context, sense unit.Sense, target *unit.Snapshot) {
	if target == nil || sense.Time+1e-9 < a.nextShotAt || sense.Self.HP <= minShotHP {
		return
	}
	dx, dy := target.X-a.x, target.Y-a.y
	length := math.Hypot(dx, dy)
	if length < 1e-6 {
		return
	}
	lead := math.Min(0.9, length/shotSpeed*0.72)
	dx += target.VX * lead
	dy += target.VY * lead
	length = math.Hypot(dx, dy)
	if length < 1e-6 {
		return
	}
	ux, uy := dx/length, dy/length
	a.nextShotAt = sense.Time + shotInterval
	ctx.Out <- unit.SetHP{UnitID: ctx.ID, HP: sense.Self.HP - shotCost}
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: KindShot,
		X:    a.x + ux*(sense.Self.Radius+shotRadius+4),
		Y:    a.y + uy*(sense.Self.Radius+shotRadius+4),
		VX:   ux * shotSpeed, VY: uy * shotSpeed,
		OwnerID: ctx.ID, Slot: a.slot,
	})
	ctx.Out <- unit.FX{
		Name: "coalesce-compress", Kind: KindCoalesce, UnitID: ctx.ID, Slot: a.slot,
		X: a.x, Y: a.y, VX: ux, VY: uy, Amount: shotCost,
	}
}

func (a *凝聚体) onCollision(ctx unit.Context, hit unit.Collision) {
	if !unit.EnemyTarget(hit, a.slot) || hit.Time+1e-9 < a.nextContact {
		return
	}
	a.nextContact = hit.Time + contactGap
	damage := 4 + a.hp*0.10
	ctx.Out <- unit.Damage{From: ctx.ID, To: hit.Other.ID, Amount: damage}
	ctx.Out <- unit.FX{
		Name: "coalesce-collide", Kind: KindCoalesce, UnitID: ctx.ID, Slot: a.slot,
		X: hit.Other.X, Y: hit.Other.Y, Amount: damage,
		VX: hit.NX, VY: hit.NY,
	}
}

func (a *凝聚体) onIncoming(ctx unit.Context, damage unit.IncomingDamage) {
	unit.ConfirmHit(ctx, damage)
	if damage.Amount < 4 || damage.Time+1e-9 < a.lastShedTime+shedGap || a.knownChunks >= maxChunks {
		return
	}
	kind := chunkForDamage(damage.Amount)
	def := chunkDefs[kind]
	if def.kind == "" {
		return
	}
	ux, uy := 0.0, 0.0
	if source, ok := a.source[damage.From]; ok {
		ux, uy = a.x-source.x, a.y-source.y
	}
	if math.Hypot(ux, uy) < 1e-6 && a.rng != nil {
		ang := a.rng.Float64() * 2 * math.Pi
		ux, uy = math.Cos(ang), math.Sin(ang)
	}
	length := math.Hypot(ux, uy)
	if length < 1e-6 {
		ux, uy, length = 1, 0, 1
	}
	ux, uy = ux/length, uy/length
	x := a.x + ux*(a.radius+def.radius+3)
	y := a.y + uy*(a.radius+def.radius+3)
	if a.field.Name != "" {
		x, y = a.field.Clamp(x, y, def.radius+2)
	}
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: kind, X: x, Y: y, OwnerID: ctx.ID, Slot: a.slot,
	})
	ctx.Out <- unit.FX{
		Name: "coalesce-shed", Kind: KindCoalesce, UnitID: ctx.ID, Slot: a.slot,
		X: x, Y: y, Amount: def.value, VX: ux, VY: uy,
	}
	a.knownChunks++
	a.lastShedTime = damage.Time
}

func chunkForDamage(amount float64) string {
	switch {
	case amount < 8:
		return KindMassSmall
	case amount < 16:
		return KindMassMedium
	default:
		return KindMassHeavy
	}
}

type massChunk struct {
	bornAt float64
}

func (c *massChunk) Handle(ctx unit.Context, ev unit.Event) {
	sense, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if c.bornAt < 0 {
		c.bornAt = sense.Time
	}
	if sense.Time+1e-9 >= c.bornAt+chunkLife {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

type massShot struct {
	owner uint64
	slot  int

	bornAt    float64
	x, y      float64
	ox, oy    float64
	seenOwner bool
	wallHits  int
	field     unit.Field
	dead      bool
}

func (s *massShot) Handle(ctx unit.Context, ev unit.Event) {
	if s.dead {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		s.x, s.y = e.Self.X, e.Self.Y
		s.field = e.Field
		if s.bornAt < 0 {
			s.bornAt = e.Time
		}
		for i := range e.Nearby {
			if e.Nearby[i].ID == s.owner {
				s.ox, s.oy, s.seenOwner = e.Nearby[i].X, e.Nearby[i].Y, true
				break
			}
		}
		if e.Time+1e-9 >= s.bornAt+shotLife {
			s.returnChunk(ctx, s.x, s.y)
		}
	case unit.Collision:
		if e.Other.ID == s.owner || e.Other.Slot == s.slot {
			return
		}
		if unit.Hittable(e.Other, s.slot) {
			ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: shotDamage}
			ctx.Out <- unit.FX{
				Name: "coalesce-shot-hit", Kind: KindCoalesce, UnitID: s.owner, Slot: s.slot,
				X: e.Other.X, Y: e.Other.Y, Amount: shotDamage,
			}
			x, y := e.Other.X, e.Other.Y
			if s.seenOwner {
				dx, dy := s.ox-x, s.oy-y
				if length := math.Hypot(dx, dy); length > 1e-6 {
					def := chunkDefs[KindMassShot]
					x += dx / length * (e.Other.Radius + def.radius + 6)
					y += dy / length * (e.Other.Radius + def.radius + 6)
				}
			}
			s.returnChunk(ctx, x, y)
			return
		}
		s.returnChunk(ctx, e.Other.X, e.Other.Y)
	case unit.WallHit:
		s.wallHits++
		if s.wallHits >= shotBounces {
			s.returnChunk(ctx, e.X, e.Y)
		}
	}
}

func (s *massShot) returnChunk(ctx unit.Context, x, y float64) {
	if s.dead {
		return
	}
	s.dead = true
	def := chunkDefs[KindMassShot]
	if s.field.Name != "" {
		x, y = s.field.Clamp(x, y, def.radius+2)
	}
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: KindMassShot, X: x, Y: y, OwnerID: s.owner, Slot: s.slot,
	})
	ctx.Out <- unit.FX{
		Name: "coalesce-return", Kind: KindCoalesce, UnitID: s.owner, Slot: s.slot,
		X: x, Y: y, Amount: def.value,
	}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}
