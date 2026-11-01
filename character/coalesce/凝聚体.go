// 凝聚体把生命、体积、速度和弹药统一成质量。
// 受伤会崩下可夺回的凝块；主动开火会把当前质量压成会飞回来的重弹。
// 质量越低身体越小、速度越快，但每次出手也越接近解体的边缘。
package 凝聚体

import (
	"embed"
	"math"
	"math/rand/v2"
	"strconv"

	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const (
	KindCoalesce = "凝聚体"
	KindShot     = "凝聚体压缩弹"

	KindMassBlock = "凝聚体质量块"
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
	shotSpeed    = 450.0
	minShotHP    = 24.0
	shotInterval = 1.8
	shotLife     = 3.4
	shotBounces  = 2
	chunkLife    = 9.0
	maxChunks    = 24
	maxShedBits  = 12
	minChunkMass = 1
	maxChunkMass = 12
	shedMinRange = 78.0
	shedMaxRange = 145.0
	shedGap      = 0.22
	contactGap   = 0.75
	steerGap     = 0.36
)

type chunkDef struct {
	kind   string
	value  float64
	radius float64
}

var chunkDefs = make(map[string]chunkDef, maxChunkMass)

var chunkList []chunkDef

func massKind(value int) string {
	return KindMassBlock + strconv.Itoa(value)
}

func shotKind(value int) string {
	return KindShot + strconv.Itoa(value)
}

func shotRadiusForMass(value int) float64 {
	return 5 + math.Sqrt(float64(value))*1.45
}

func shotSpeedForMass(value int) float64 {
	return shotSpeed - float64(value)*5
}

func initChunkDefs() {
	for value := minChunkMass; value <= maxChunkMass; value++ {
		def := chunkDef{
			kind: massKind(value), value: float64(value), radius: 4.2 + math.Sqrt(float64(value))*1.55,
		}
		chunkDefs[def.kind] = def
		chunkList = append(chunkList, def)
	}
}

func init() {
	initChunkDefs()
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
			Kind: def.kind, Role: unit.RoleMinion,
			Radius: def.radius, MaxHP: 1, Speed: 0, Vision: bodyVision,
			Mortal:      true,
			AimPriority: 255,
			Look:        unit.Look{Color: chunkColor, FX: []string{"coalesce-chunk"}},
		}, func(info unit.SpawnInfo) unit.Actor {
			return &massChunk{bornAt: -1, owner: info.OwnerID, slot: info.Slot}
		})
	}

	for value := minChunkMass; value <= maxChunkMass; value++ {
		value := value
		p.Register(unit.Spec{
			Kind: shotKind(value), Role: unit.RoleProjectile,
			Radius: shotRadiusForMass(value), MaxHP: 1, Speed: shotSpeedForMass(value), Vision: bodyVision,
			Look: unit.Look{Color: shotColor, Trail: true, Glow: true, FX: []string{"coalesce-shot"}},
		}, func(info unit.SpawnInfo) unit.Actor {
			return &massShot{owner: info.OwnerID, slot: info.Slot, bornAt: -1, mass: value}
		})
	}
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
		alive++
	}
	return alive
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
	shotMass := minChunkMass
	if a.rng != nil {
		shotMass += a.rng.IntN(maxChunkMass)
	}
	dx, dy := target.X-a.x, target.Y-a.y
	length := math.Hypot(dx, dy)
	if length < 1e-6 {
		return
	}
	speed := shotSpeedForMass(shotMass)
	lead := math.Min(0.9, length/speed*0.72)
	dx += target.VX * lead
	dy += target.VY * lead
	length = math.Hypot(dx, dy)
	if length < 1e-6 {
		return
	}
	ux, uy := dx/length, dy/length
	a.nextShotAt = sense.Time + shotInterval
	ctx.Out <- unit.SetHP{UnitID: ctx.ID, HP: sense.Self.HP - float64(shotMass)}
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: shotKind(shotMass),
		X:    a.x + ux*(sense.Self.Radius+shotRadiusForMass(shotMass)+4),
		Y:    a.y + uy*(sense.Self.Radius+shotRadiusForMass(shotMass)+4),
		VX:   ux * speed, VY: uy * speed,
		OwnerID: ctx.ID, Slot: a.slot,
	})
	ctx.Out <- unit.FX{
		Name: "coalesce-compress", Kind: KindCoalesce, UnitID: ctx.ID, Slot: a.slot,
		X: a.x, Y: a.y, VX: ux, VY: uy, Amount: float64(shotMass),
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
	totalMass := int(math.Floor(damage.Amount + 1e-6))
	if totalMass < 1 || damage.Time+1e-9 < a.lastShedTime+shedGap || a.knownChunks >= maxChunks {
		return
	}
	available := maxChunks - a.knownChunks
	if cap := available * maxChunkMass; totalMass > cap {
		totalMass = cap
	}
	if cap := maxShedBits * maxChunkMass; totalMass > cap {
		totalMass = cap
	}
	if totalMass <= 0 {
		return
	}

	minPieces := (totalMass + maxChunkMass - 1) / maxChunkMass
	maxPieces := totalMass
	if maxPieces > maxShedBits {
		maxPieces = maxShedBits
	}
	if maxPieces > available {
		maxPieces = available
	}
	pieces := minPieces
	if a.rng != nil && maxPieces > minPieces {
		pieces += a.rng.IntN(maxPieces - minPieces + 1)
	}
	if pieces <= 0 || pieces > available {
		return
	}
	masses := splitMass(totalMass, pieces, a.rng)
	base := 0.0
	if a.rng != nil {
		base = a.rng.Float64() * 2 * math.Pi
	}
	ctx.Out <- unit.FX{
		Name: "coalesce-burst", Kind: KindCoalesce, UnitID: ctx.ID, Slot: a.slot,
		X: a.x, Y: a.y, Amount: float64(totalMass),
	}
	for i, mass := range masses {
		def := chunkDefs[massKind(mass)]
		x, y := a.scatterBit(def, base, i, pieces)
		ctx.Out <- ctx.Born(unit.Spawn{
			Kind: def.kind, X: x, Y: y, OwnerID: ctx.ID, Slot: a.slot,
		})
		ctx.Out <- unit.FX{
			Name: "coalesce-shed", Kind: KindCoalesce, UnitID: ctx.ID, Slot: a.slot,
			X: x, Y: y, Amount: def.value,
		}
	}
	a.knownChunks += pieces
	a.lastShedTime = damage.Time
}

func splitMass(total, pieces int, rng *rand.Rand) []int {
	masses := make([]int, pieces)
	for i := range masses {
		masses[i] = minChunkMass
	}
	for rest := total - pieces*minChunkMass; rest > 0; {
		idx := 0
		if rng != nil {
			idx = rng.IntN(pieces)
		}
		if masses[idx] >= maxChunkMass {
			idx = (idx + 1) % pieces
			if masses[idx] >= maxChunkMass {
				for i := range masses {
					if masses[i] < maxChunkMass {
						idx = i
						break
					}
				}
			}
		}
		masses[idx]++
		rest--
	}
	return masses
}

func (a *凝聚体) scatterBit(def chunkDef, base float64, index, total int) (float64, float64) {
	sector := 2 * math.Pi / float64(total)
	for attempt := 0; attempt < 20; attempt++ {
		jitter := 0.0
		if a.rng != nil {
			jitter = (a.rng.Float64() - 0.5) * sector * 0.7
		}
		angle := base + float64(index)*sector + jitter
		distance := shedMinRange
		if a.rng != nil {
			distance += a.rng.Float64() * (shedMaxRange - shedMinRange)
		}
		x := a.x + math.Cos(angle)*distance
		y := a.y + math.Sin(angle)*distance
		if a.field.Name == "" || a.field.Walkable(x, y, def.radius+2) {
			return x, y
		}
	}
	angle := base + float64(index)*sector
	return a.field.Clamp(a.x+math.Cos(angle)*shedMaxRange, a.y+math.Sin(angle)*shedMaxRange, def.radius+2)
}

type massChunk struct {
	bornAt float64
	owner  uint64
	slot   int
	x, y   float64
	booted bool
}

func (c *massChunk) Handle(ctx unit.Context, ev unit.Event) {
	if hit, ok := ev.(unit.IncomingDamage); ok {
		ctx.Out <- unit.FX{
			Name: "coalesce-chunk-break", Kind: KindCoalesce, UnitID: c.owner, Slot: c.slot,
			X: c.x, Y: c.y, Amount: hit.Amount,
		}
		unit.ConfirmHit(ctx, hit)
		return
	}
	sense, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	c.x, c.y = sense.Self.X, sense.Self.Y
	if !c.booted {
		c.booted = true
		ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
		ctx.Out <- unit.NoHealthNumbers{UnitID: ctx.ID, Hold: true}
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
	mass  int

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
			amount := float64(s.mass)
			ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: amount}
			ctx.Out <- unit.FX{
				Name: "coalesce-shot-hit", Kind: KindCoalesce, UnitID: s.owner, Slot: s.slot,
				X: e.Other.X, Y: e.Other.Y, Amount: amount,
			}
			x, y := e.Other.X, e.Other.Y
			if s.seenOwner {
				dx, dy := s.ox-x, s.oy-y
				if length := math.Hypot(dx, dy); length > 1e-6 {
					def := chunkDefs[massKind(s.mass)]
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
	def := chunkDefs[massKind(s.mass)]
	if s.field.Name != "" {
		x, y = s.field.Clamp(x, y, def.radius+2)
	}
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: def.kind, X: x, Y: y, OwnerID: s.owner, Slot: s.slot,
	})
	ctx.Out <- unit.FX{
		Name: "coalesce-return", Kind: KindCoalesce, UnitID: s.owner, Slot: s.slot,
		X: x, Y: y, Amount: def.value,
	}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}
