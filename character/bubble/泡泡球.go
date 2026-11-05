// 泡泡球：一路吹出慢速漂移的泡泡。敌方弹体撞进泡泡会被封存在里面，
// 弹越多泡泡越慢、越大；泡泡互相碰撞会合并，敌方战斗机撞上则按体积受伤。
// 被球撞破的泡泡会把封存弹药和撞击推力一起释放。
package 泡泡球

import (
	"embed"
	"encoding/json"
	"math"
	"math/rand/v2"
	"sync"

	"xqdj/internal/unit"
)

const (
	KindBubbleBall = "泡泡球"
	KindBubble     = "泡泡"
	KindBubbleShot = "泡泡封弹"
	KindBubbleLoad = "泡泡载弹"
)

const (
	bodyRadius = 18.0
	bodyHP     = 100.0
	bodySpeed  = 168.0
	bodyVision = 9999.0
	bodyColor  = "#d5fbf2"

	bubbleRadius   = 18.0
	bubbleSpeed    = 42.0
	bubbleLife     = 10.0
	maxBubbles     = 12
	maxStored      = 24
	maxBubbleSize  = 110.0
	bubbleStateGap = 0.18

	shotSpeed = 420.0
	shotLife  = 1.8

	popPush        = 95.0
	popPushPerAmmo = 20.0
	shotPush       = 42.0
	tickSeconds    = 1.0 / 60.0
)

type bubbleBirth struct {
	radius   float64
	speed    float64
	duration float64
	releaseX float64
	releaseY float64
}

type bubbleAmmo struct {
	Kind      string  `json:"kind"`
	OwnerID   uint64  `json:"ownerId"`
	OwnerKind string  `json:"ownerKind"`
	Slot      int     `json:"slot"`
	VX        float64 `json:"vx"`
	VY        float64 `json:"vy"`
	Damage    float64 `json:"damage"`
}

type bubbleState struct {
	Count int          `json:"count"`
	Ammo  []bubbleAmmo `json:"ammo"`
}

type bubbleBook struct {
	radius float64
	ready  bool
	ammo   []bubbleAmmo
}

type bubbleBooks struct {
	sync.Mutex
	byID map[uint64]bubbleBook
}

func bubbleBookState(ctx unit.Context) *bubbleBooks {
	books := unit.StateFor[bubbleBooks](ctx.State)
	books.Lock()
	if books.byID == nil {
		books.byID = map[uint64]bubbleBook{}
	}
	books.Unlock()
	return books
}

func (b *泡泡) remember(ctx unit.Context) {
	books := bubbleBookState(ctx)
	books.Lock()
	books.byID[ctx.ID] = bubbleBook{radius: b.radius, ready: !b.growing && b.safeUntil <= 0, ammo: append([]bubbleAmmo(nil), b.ammo...)}
	books.Unlock()
}

func (b *泡泡) forget(ctx unit.Context) {
	books := bubbleBookState(ctx)
	books.Lock()
	delete(books.byID, ctx.ID)
	books.Unlock()
}

func readBubbleBook(ctx unit.Context, id uint64) (bubbleBook, bool) {
	books := bubbleBookState(ctx)
	books.Lock()
	defer books.Unlock()
	b, ok := books.byID[id]
	return b, ok
}

func removeBubbleBook(ctx unit.Context, id uint64) {
	books := bubbleBookState(ctx)
	books.Lock()
	delete(books.byID, id)
	books.Unlock()
}

type shotBirth struct {
	damage float64
}

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindBubbleBall, assets)
	p.Register(unit.Spec{
		Kind:    KindBubbleBall,
		Role:    unit.RoleFighter,
		Radius:  bodyRadius,
		MaxHP:   bodyHP,
		Speed:   bodySpeed,
		Vision:  bodyVision,
		Fighter: true,
		Look:    unit.Look{Color: bodyColor, Glow: true, FX: []string{"bubble-ball"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &泡泡球{rng: unit.NewRNG(info.Stream)}
	})

	p.Register(unit.Spec{
		Kind:   KindBubble,
		Role:   unit.RoleMinion,
		Radius: bubbleRadius,
		MaxHP:  1,
		Speed:  bubbleSpeed,
		Vision: bodyVision,
		Mortal: false,
		Cruise: true,
		Look:   unit.Look{Color: "#d9fbff", Glow: true, FX: []string{"bubble-shell"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		b := bubbleBirth{radius: bubbleRadius, speed: bubbleSpeed}
		if d, ok := info.Data.(bubbleBirth); ok {
			b = d
		}
		start := b.radius
		if b.duration > 0 && start > 4 {
			start = 4
		}
		return &泡泡{owner: info.OwnerID, slot: info.Slot, radius: start, finalRadius: b.radius, baseSpeed: b.speed, growDuration: b.duration, growing: b.duration > 0, releaseX: b.releaseX, releaseY: b.releaseY}
	})

	p.Register(unit.Spec{
		Kind:   KindBubbleShot,
		Role:   unit.RoleProjectile,
		Radius: 5,
		MaxHP:  1,
		Speed:  shotSpeed,
		Vision: 0,
		Pass:   true,
		Look:   unit.Look{Color: "#e8ffff", Glow: true, Trail: true, FX: []string{"bubble-shot"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		amount := 2.0
		if d, ok := info.Data.(shotBirth); ok && d.damage > 0 {
			amount = d.damage
		}
		return &封弹{owner: info.OwnerID, slot: info.Slot, damage: amount}
	})
}

func markStacks(o unit.Snapshot, kind string) int {
	for i := range o.Marks {
		if o.Marks[i].Kind == kind {
			return o.Marks[i].Stacks
		}
	}
	return 0
}

type 泡泡球 struct {
	rng      *rand.Rand
	booted   bool
	slot     int
	nextBlow float64
	nextTurn float64
	x, y     float64
}

func (b *泡泡球) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		unit.ConfirmHit(ctx, e)
	case unit.Sense:
		b.onSense(ctx, e)
	case unit.WallHit:
		b.turnRandom(ctx)
	}
}

func (b *泡泡球) onSense(ctx unit.Context, s unit.Sense) {
	b.slot = s.Self.Slot
	b.x, b.y = s.Self.X, s.Self.Y
	if !b.booted {
		b.booted = true
		b.nextTurn = s.Time + 1.5 + b.rng.Float64()*2.5
		b.turnRandom(ctx)
		b.nextBlow = s.Time
		ctx.Out <- unit.FX{Name: "bubble-idle", Kind: KindBubbleBall, UnitID: ctx.ID, Slot: b.slot, X: b.x, Y: b.y}
	}

	count := 0
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind == KindBubble && o.OwnerID == ctx.ID {
			count++
		}
	}
	if s.Time+1e-9 >= b.nextBlow && count < maxBubbles {
		duration := b.blow(ctx, s)
		b.nextBlow = s.Time + duration + 0.12
	}

	if s.Time+1e-9 >= b.nextTurn {
		b.nextTurn = s.Time + 1.8 + b.rng.Float64()*2.8
		b.turnRandom(ctx)
	}
}

func (b *泡泡球) turnRandom(ctx unit.Context) {
	a := b.rng.Float64() * 2 * math.Pi
	ctx.Out <- unit.SetFSDirection{UnitID: ctx.ID, VX: math.Cos(a), VY: math.Sin(a)}
}

func (b *泡泡球) blow(ctx unit.Context, s unit.Sense) float64 {
	r := 10 + b.rng.Float64()*18
	speed := 65 - (r-10)*1.45 + b.rng.Float64()*5
	duration := 1.1 + (r-10)/18*1.8 + b.rng.Float64()*0.08
	dx, dy := s.Self.VX, s.Self.VY
	l := math.Hypot(dx, dy)
	if l < 1e-6 {
		dx, dy, l = 1, 0, 1
	}
	base := math.Atan2(dy/l, dx/l) + (b.rng.Float64()*2-1)*0.55
	gap := bodyRadius + 19 + 4
	x, y := s.Field.Clamp(s.Self.X+math.Cos(base)*gap, s.Self.Y+math.Sin(base)*gap, 4)
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: KindBubble, X: x, Y: y,
		OwnerID: ctx.ID, Slot: b.slot,
		Data: bubbleBirth{
			radius: r, speed: speed, duration: duration,
			releaseX: math.Cos(base), releaseY: math.Sin(base),
		},
	})
	ctx.Out <- unit.FX{Name: "bubble-blow", Kind: KindBubbleBall, UnitID: ctx.ID, Slot: b.slot, X: x, Y: y, Amount: duration, VX: r, VY: math.Cos(base)}
	return duration
}

type 泡泡 struct {
	owner        uint64
	slot         int
	radius       float64
	finalRadius  float64
	baseSpeed    float64
	growDuration float64
	growStart    float64
	releaseAt    float64
	releaseX     float64
	releaseY     float64
	safeUntil    float64
	x, y         float64
	vx, vy       float64
	growing      bool
	bornAt       float64
	booted       bool
	done         bool
	stateAt      float64
	nextState    float64
	ammo         []bubbleAmmo
	seen         map[uint64]bool
	nearby       []unit.Snapshot
}

func (b *泡泡) Handle(ctx unit.Context, ev unit.Event) {
	if b.done {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		b.onSense(ctx, e)
	case unit.Collision:
		b.onCollision(ctx, e)
	}
}

func (b *泡泡) onSense(ctx unit.Context, s unit.Sense) {
	b.x, b.y = s.Self.X, s.Self.Y
	b.vx, b.vy = s.Self.VX, s.Self.VY
	b.nearby = append(b.nearby[:0], s.Nearby...)
	if !b.booted {
		b.booted = true
		if b.growDuration > 0 {
			b.growing = true
			b.growStart = s.Time
			b.releaseAt = s.Time + b.growDuration
			ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
			ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
			ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: 0}
		} else {
			b.bornAt = s.Time
		}
		b.nextState = s.Time
		ctx.Out <- unit.SetRadius{UnitID: ctx.ID, Radius: b.radius}
		if !b.growing {
			ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: b.baseSpeed}
		}
		b.remember(ctx)
	}
	if b.growing {
		for i := range s.Nearby {
			o := &s.Nearby[i]
			if o.ID != b.owner {
				continue
			}
			fx, fy := o.VX, o.VY
			if d := math.Hypot(fx, fy); d < 1 {
				fx, fy = o.FaceX, o.FaceY
			}
			if d := math.Hypot(fx, fy); d > 1e-6 {
				fx, fy = fx/d, fy/d
			} else {
				fx, fy = 1, 0
			}
			tip := o.Radius + 19 + b.radius
			x, y := s.Field.Clamp(o.X+fx*tip, o.Y+fy*tip, math.Max(3, b.radius))
			ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: x, Y: y}
			break
		}
		p := math.Max(0, math.Min(1, (s.Time-b.growStart)/b.growDuration))
		grown := 4 + (b.finalRadius-4)*math.Sqrt(p)
		b.radius = grown
		ctx.Out <- unit.SetRadius{UnitID: ctx.ID, Radius: grown}
		if s.Time+1e-9 >= b.releaseAt {
			b.growing = false
			b.bornAt = s.Time
			b.radius = b.finalRadius
			ctx.Out <- unit.SetRadius{UnitID: ctx.ID, Radius: b.finalRadius}
			ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: false}
			ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}
			ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: b.baseSpeed}
			ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: b.releaseX * b.baseSpeed, VY: b.releaseY * b.baseSpeed}
			b.safeUntil = s.Time + 0.65
			b.remember(ctx)
			ctx.Out <- unit.FX{Name: "bubble-release", Kind: KindBubbleBall, UnitID: b.owner, Slot: b.slot, X: b.x, Y: b.y, Amount: b.finalRadius}
		}
		return
	}
	if b.safeUntil > 0 {
		if s.Time+1e-9 >= b.safeUntil {
			b.safeUntil = 0
			ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: false}
			b.remember(ctx)
		}
	}
	if s.Time-b.bornAt >= bubbleLife {
		b.done = true
		b.forget(ctx)
		ctx.Out <- unit.FX{Name: "bubble-fade", Kind: KindBubbleBall, UnitID: b.owner, Slot: b.slot, X: b.x, Y: b.y}
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	if s.Time+1e-9 >= b.nextState {
		b.nextState = s.Time + bubbleStateGap
		data, _ := json.Marshal(bubbleState{Count: len(b.ammo), Ammo: b.ammo})
		ctx.Out <- unit.FX{
			Name: "bubble-state", Data: data, Kind: KindBubbleBall, UnitID: ctx.ID, Slot: b.slot,
			X: b.x, Y: b.y,
		}
	}
}

func (b *泡泡) onCollision(ctx unit.Context, e unit.Collision) {
	if b.growing || b.safeUntil > 0 {
		return
	}
	if e.Other.Kind == KindBubble {
		b.merge(ctx, e.Other, e.Time)
		return
	}
	switch e.Other.Role {
	case unit.RoleProjectile:
		if e.Other.Slot == b.slot || e.Other.ArcSpan > 0 {
			return
		}
		b.capture(ctx, e.Other)
	case unit.RoleFighter:
		if e.Other.ID != b.owner && e.Other.Slot != b.slot {
			b.hitEnemy(ctx, e.Other)
		}
	}
}

func (b *泡泡) capture(ctx unit.Context, bullet unit.Snapshot) {
	if b.seen == nil {
		b.seen = map[uint64]bool{}
	}
	if b.seen[bullet.ID] {
		return
	}
	b.seen[bullet.ID] = true
	ctx.Out <- unit.Despawn{UnitID: bullet.ID}
	if len(b.ammo) >= maxStored {
		return
	}
	ownerKind := ""
	for i := range b.nearby {
		if b.nearby[i].ID == bullet.OwnerID {
			ownerKind = b.nearby[i].Kind
			break
		}
	}
	power := 2.0 + math.Min(4.0, bullet.Radius*0.3)
	b.ammo = append(b.ammo, bubbleAmmo{
		Kind: bullet.Kind, OwnerID: bullet.OwnerID, OwnerKind: ownerKind,
		Slot: bullet.Slot, VX: bullet.VX, VY: bullet.VY, Damage: power,
	})
	n := len(b.ammo)
	ctx.Out <- unit.ClearMarks{UnitID: ctx.ID, Kind: KindBubbleLoad}
	ctx.Out <- unit.StackMark{UnitID: ctx.ID, Kind: KindBubbleLoad, Delta: n, Hidden: true}
	b.radius = b.finalRadius + math.Min(8, float64(n)*0.75)
	ctx.Out <- unit.SetRadius{UnitID: ctx.ID, Radius: b.radius}
	speed := bubbleMoveSpeed(b.radius, n)
	dirX, dirY := b.vx, b.vy
	if d := math.Hypot(dirX, dirY); d > 1e-6 {
		dirX, dirY = dirX/d, dirY/d
	} else {
		dirX, dirY = 1, 0
	}
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: speed}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: dirX * speed, VY: dirY * speed}
	b.remember(ctx)
	ctx.Out <- unit.FX{
		Name: "bubble-capture", Kind: KindBubbleBall, UnitID: ctx.ID, Slot: b.slot,
		X: b.x, Y: b.y, Amount: float64(n),
		Data: mustJSON(bubbleState{Count: n, Ammo: b.ammo}),
	}
}

func bubbleMoveSpeed(radius float64, ammo int) float64 {
	weight := math.Pow(bubbleRadius/math.Max(bubbleRadius, radius), 0.65)
	return math.Max(8, 65*weight-float64(ammo)*1.2)
}

func (b *泡泡) merge(ctx unit.Context, other unit.Snapshot, now float64) {
	if b.radius+0.25 < other.Radius || (math.Abs(b.radius-other.Radius) < 0.25 && ctx.ID < other.ID) {
		return
	}
	book, ok := readBubbleBook(ctx, other.ID)
	if !ok || !book.ready {
		return
	}
	b.ammo = append(b.ammo, book.ammo...)
	if len(b.ammo) > maxStored {
		b.ammo = b.ammo[:maxStored]
	}
	b.radius = math.Min(maxBubbleSize, math.Sqrt(b.radius*b.radius+other.Radius*other.Radius))
	b.finalRadius = b.radius
	b.bornAt = now
	speed := bubbleMoveSpeed(b.radius, len(b.ammo))
	dirX, dirY := b.vx, b.vy
	if d := math.Hypot(dirX, dirY); d > 1e-6 {
		dirX, dirY = dirX/d, dirY/d
	} else {
		dirX, dirY = 1, 0
	}
	ctx.Out <- unit.SetRadius{UnitID: ctx.ID, Radius: b.radius}
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: speed}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: dirX * speed, VY: dirY * speed}
	ctx.Out <- unit.ClearMarks{UnitID: ctx.ID, Kind: KindBubbleLoad}
	ctx.Out <- unit.StackMark{UnitID: ctx.ID, Kind: KindBubbleLoad, Delta: len(b.ammo), Hidden: true}
	b.remember(ctx)
	removeBubbleBook(ctx, other.ID)
	ctx.Out <- unit.Despawn{UnitID: other.ID}
	ctx.Out <- unit.FX{Name: "bubble-merge", Kind: KindBubbleBall, UnitID: ctx.ID, Slot: b.slot, X: b.x, Y: b.y, Amount: b.radius, Data: mustJSON(bubbleState{Count: len(b.ammo), Ammo: b.ammo})}
}

func (b *泡泡) hitEnemy(ctx unit.Context, other unit.Snapshot) {
	damage := 2 + math.Max(0, b.radius-8)*0.35
	ctx.Out <- unit.Damage{From: ctx.ID, To: other.ID, Amount: damage}
	b.pop(ctx, other)
}

func (b *泡泡) pop(ctx unit.Context, other unit.Snapshot) {
	if b.done {
		return
	}
	b.done = true
	count := len(b.ammo)
	dx, dy := other.X-b.x, other.Y-b.y
	d := math.Hypot(dx, dy)
	if d < 1e-6 {
		dx, dy, d = 1, 0, 1
	}
	dirX, dirY := dx/d, dy/d
	push := popPush + float64(count)*popPushPerAmmo
	ctx.Out <- unit.Force{UnitID: other.ID, AX: dirX * push / tickSeconds, AY: dirY * push / tickSeconds}

	aimX, aimY := dirX, dirY
	best := math.MaxFloat64
	for i := range b.nearby {
		o := &b.nearby[i]
		if !unit.Hittable(*o, b.slot) {
			continue
		}
		if d := math.Hypot(o.X-b.x, o.Y-b.y); d < best {
			best = d
			aimX, aimY = (o.X-b.x)/d, (o.Y-b.y)/d
		}
	}
	for i, a := range b.ammo {
		bx, by := a.VX, a.VY
		bl := math.Hypot(bx, by)
		sx, sy := aimX, aimY
		if bl > 1e-6 {
			sx = aimX*0.7 + bx/bl*0.3
			sy = aimY*0.7 + by/bl*0.3
		}
		sl := math.Hypot(sx, sy)
		if sl < 1e-6 {
			sx, sy, sl = dirX, dirY, 1
		}
		spread := (float64(i) - float64(count-1)/2) * 0.09
		ang := math.Atan2(sy/sl, sx/sl) + spread
		speed := shotSpeed + math.Min(120, float64(count)*8)
		ctx.Out <- ctx.Born(unit.Spawn{
			Kind: KindBubbleShot, X: b.x, Y: b.y,
			VX: math.Cos(ang) * speed, VY: math.Sin(ang) * speed,
			OwnerID: b.owner, Slot: b.slot, Data: shotBirth{damage: a.Damage},
		})
	}
	ctx.Out <- unit.FX{
		Name: "bubble-pop", Kind: KindBubbleBall, UnitID: b.owner, Slot: b.slot,
		X: b.x, Y: b.y, Amount: float64(count), Data: mustJSON(bubbleState{Count: count, Ammo: b.ammo}),
	}
	b.forget(ctx)
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

func mustJSON(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}

type 封弹 struct {
	owner  uint64
	slot   int
	damage float64
	bornAt float64
	booted bool
	done   bool
	x, y   float64
	vx, vy float64
}

func (s *封弹) Handle(ctx unit.Context, ev unit.Event) {
	if s.done {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		s.x, s.y, s.vx, s.vy = e.Self.X, e.Self.Y, e.Self.VX, e.Self.VY
		if !s.booted {
			s.booted = true
			s.bornAt = e.Time
		} else if e.Time-s.bornAt >= shotLife {
			s.done = true
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		}
	case unit.Collision:
		if !unit.Hittable(e.Other, s.slot) {
			return
		}
		s.done = true
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: s.damage, NoFreeze: true}
		if d := math.Hypot(s.vx, s.vy); d > 1e-6 {
			ctx.Out <- unit.Force{UnitID: e.Other.ID, AX: s.vx / d * shotPush / tickSeconds, AY: s.vy / d * shotPush / tickSeconds}
		}
		ctx.Out <- unit.FX{Name: "bubble-shot-hit", Kind: KindBubbleBall, UnitID: s.owner, Slot: s.slot, X: e.Other.X, Y: e.Other.Y, Amount: s.damage}
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	case unit.WallHit:
		s.done = true
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}
