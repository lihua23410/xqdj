// 蛆：本体不出现，靠保活；场上蛆宝宝→苍蝇→苍蝇卵循环，苍蝇数量光环磨血。
package 蛆

import (
	"embed"
	"math"
	"math/rand/v2"
	"sort"

	"xqdj/internal/unit"
)

const (
	KindMaggot = "蛆"
	KindBaby   = "蛆宝宝"
	KindFly    = "苍蝇"
	KindEgg    = "苍蝇卵"
)

const (
	bodyHP    = 100.0
	bodyColor = "#6b5a4a"

	babyRadius = 12.0
	babyHP     = 25.0
	babyCruise = 120.0
	babyGrow   = 6.0 // 成长基准；实际在 ±morphJitter 内抽
	babyAim    = 16
	babyColor  = "#f0d5c8"
	babySpread = 40.0

	flyRadius   = 12.0
	flyHP       = 25.0
	flyCruise   = 150.0
	flyAim      = 15
	flyColor    = "#4a3428"
	attractR    = 80.0
	overlapR    = 24.0
	overlapNeed = 4.0
	breedCD     = 5.0
	flyVision   = 9999.0

	eggRadius = 10.0
	eggHP     = 15.0
	eggAim    = 17
	eggHatch  = 6.0 // 孵化基准；实际在 ±morphJitter 内抽
	eggColor  = "#7a8f6a"

	morphJitter = 5.0

	keepAliveGrace = 3.0
	auraGap        = 1.0
	auraPerLayer   = 1.0
	bootBabies     = 5

	timeEps = 1e-9
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindMaggot, assets)
	p.Register(unit.Spec{
		Kind:     KindMaggot,
		Role:     unit.RoleFighter,
		Radius:   0,
		MaxHP:    bodyHP,
		Speed:    0,
		Vision:   9999,
		Fighter:  true,
		Nonsolid: true,
		Look:     unit.Look{Color: bodyColor, FX: []string{"maggot"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &蛆{}
	})
	p.Register(unit.Spec{
		Kind:        KindBaby,
		Role:        unit.RoleMinion,
		Radius:      babyRadius,
		MaxHP:       babyHP,
		Speed:       babyCruise,
		Vision:      0,
		Mortal:      true,
		Cruise:      true,
		AimPriority: babyAim,
		Look:        unit.Look{Color: babyColor, FX: []string{"maggot-baby"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &蛆宝宝{owner: info.OwnerID, slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:        KindFly,
		Role:        unit.RoleMinion,
		Radius:      flyRadius,
		MaxHP:       flyHP,
		Speed:       flyCruise,
		Vision:      flyVision,
		Mortal:      true,
		Cruise:      true,
		AimPriority: flyAim,
		Look:        unit.Look{Color: flyColor, FX: []string{"maggot-fly"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &苍蝇{owner: info.OwnerID, slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:        KindEgg,
		Role:        unit.RoleMinion,
		Radius:      eggRadius,
		MaxHP:       eggHP,
		Speed:       0,
		Vision:      0,
		Mortal:      true,
		AimPriority: eggAim,
		Look:        unit.Look{Color: eggColor, Ghost: 160, FX: []string{"maggot-egg"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &苍蝇卵{owner: info.OwnerID, slot: info.Slot}
	})
}

type inheritReq struct {
	x, y, hp float64
}

type flyPair struct {
	a, b uint64
}

type 蛆 struct {
	booted      bool
	awaitSpawn  bool // 开局已 Spawn 宝宝、尚未来到 Nearby
	slot        int
	orphanFrom  float64 // <0：当前有保活
	auraReady   float64 // 0：层数为 0，下次数起需再等 auraGap
	babyReady   map[uint64]float64 // 成长到期时刻
	eggReady    map[uint64]float64 // 孵化到期时刻
	knownFly    map[uint64]struct{}
	pendingHP   []inheritReq
	breedUntil  map[uint64]float64
	overlapFrom map[flyPair]float64
	flyPassing  map[uint64]bool
	flyWasCD    map[uint64]bool // 上一拍是否在繁殖冷却，用于切白眼与飘字
	stenchOn    bool            // 场上己方苍蝇 ≥3 时的全屏臭气
	rng         *rand.Rand
}

func (a *蛆) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	a.onSense(ctx, s)
}

func (a *蛆) onSense(ctx unit.Context, s unit.Sense) {
	a.slot = s.Self.Slot
	if !a.booted {
		a.booted = true
		a.orphanFrom = -1
		a.awaitSpawn = true
		a.babyReady = map[uint64]float64{}
		a.eggReady = map[uint64]float64{}
		a.knownFly = map[uint64]struct{}{}
		a.breedUntil = map[uint64]float64{}
		a.overlapFrom = map[flyPair]float64{}
		a.flyPassing = map[uint64]bool{}
		a.flyWasCD = map[uint64]bool{}
		a.rng = rand.New(rand.NewPCG(ctx.ID*2654435761, 0x9e3779b97f4a7c15))
		ctx.Out <- unit.NoHealthNumbers{UnitID: ctx.ID, Hold: true}
		ctx.Out <- unit.NoFrameFreeze{UnitID: ctx.ID, Hold: true}
		unit.SetAim(ctx, ctx.ID, 0)
		spawnBootBabies(ctx, s)
	}
	a.ensureMaps()
	a.applyInherit(ctx, s)
	a.noteBirths(s)
	a.tickGrowth(ctx, s)
	a.tickHatch(ctx, s)
	a.tickFlies(ctx, s)
	a.tickKeepAlive(ctx, s, a.countKeepAlive(s))
	a.tickAura(ctx, s)
}

func (a *蛆) ensureMaps() {
	if a.babyReady == nil {
		a.babyReady = map[uint64]float64{}
	}
	if a.eggReady == nil {
		a.eggReady = map[uint64]float64{}
	}
	if a.knownFly == nil {
		a.knownFly = map[uint64]struct{}{}
	}
	if a.breedUntil == nil {
		a.breedUntil = map[uint64]float64{}
	}
	if a.overlapFrom == nil {
		a.overlapFrom = map[flyPair]float64{}
	}
	if a.flyPassing == nil {
		a.flyPassing = map[uint64]bool{}
	}
	if a.flyWasCD == nil {
		a.flyWasCD = map[uint64]bool{}
	}
	if a.rng == nil {
		a.rng = rand.New(rand.NewPCG(1, 1))
	}
}

// jitterDur 在 base±morphJitter 均匀抽秒数，下限 0.1。
func (a *蛆) jitterDur(base float64) float64 {
	d := base - morphJitter + a.rng.Float64()*2*morphJitter
	if d < 0.1 {
		d = 0.1
	}
	return d
}

func spawnBootBabies(ctx unit.Context, s unit.Sense) {
	for i := 0; i < bootBabies; i++ {
		ang := float64(i) * 2 * math.Pi / float64(bootBabies)
		dx, dy := math.Cos(ang), math.Sin(ang)
		ctx.Out <- unit.Spawn{
			Kind:    KindBaby,
			X:       s.Self.X + dx*babySpread,
			Y:       s.Self.Y + dy*babySpread,
			VX:      -dy * babyCruise,
			VY:      dx * babyCruise,
			OwnerID: ctx.ID,
			Slot:    s.Self.Slot,
		}
	}
}

func (a *蛆) noteBirths(s unit.Sense) {
	liveBaby := map[uint64]struct{}{}
	liveEgg := map[uint64]struct{}{}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID != s.Self.ID {
			continue
		}
		switch o.Kind {
		case KindBaby:
			liveBaby[o.ID] = struct{}{}
			if _, ok := a.babyReady[o.ID]; !ok {
				a.babyReady[o.ID] = s.Time + a.jitterDur(babyGrow)
			}
		case KindEgg:
			liveEgg[o.ID] = struct{}{}
			if _, ok := a.eggReady[o.ID]; !ok {
				a.eggReady[o.ID] = s.Time + a.jitterDur(eggHatch)
			}
		}
	}
	for id := range a.babyReady {
		if _, ok := liveBaby[id]; !ok {
			delete(a.babyReady, id)
		}
	}
	for id := range a.eggReady {
		if _, ok := liveEgg[id]; !ok {
			delete(a.eggReady, id)
		}
	}
}

func (a *蛆) countKeepAlive(s unit.Sense) int {
	n := 0
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID != s.Self.ID {
			continue
		}
		switch o.Kind {
		case KindBaby, KindFly, KindEgg:
			n++
		}
	}
	return n
}

func (a *蛆) tickKeepAlive(ctx unit.Context, s unit.Sense, alive int) {
	if alive > 0 {
		a.orphanFrom = -1
		a.awaitSpawn = false
		return
	}
	if a.awaitSpawn {
		return
	}
	if a.orphanFrom < 0 {
		a.orphanFrom = s.Time
	}
	if s.Time+timeEps >= a.orphanFrom+keepAliveGrace {
		hp := s.Self.HP
		if hp < 1 {
			hp = bodyHP
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: ctx.ID, Amount: hp}
	}
}

func (a *蛆) countFlies(s unit.Sense) int {
	n := 0
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID == s.Self.ID && o.Kind == KindFly {
			n++
		}
	}
	return n
}

func (a *蛆) syncStench(ctx unit.Context, s unit.Sense, flies int) {
	a.stenchOn = flies >= 3
	// 每拍带苍蝇数；前端 ≥3 开臭气，停更后超时关掉（防本体倒下残留）。
	ctx.Out <- unit.FX{
		Name: "stench", Kind: KindMaggot, Slot: a.slot,
		X: s.Self.X, Y: s.Self.Y, Amount: float64(flies),
	}
}

func (a *蛆) tickAura(ctx unit.Context, s unit.Sense) {
	flies := a.countFlies(s)
	a.syncStench(ctx, s, flies)
	layers := flies / 3
	if layers <= 0 {
		a.auraReady = 0
		return
	}
	if a.auraReady == 0 {
		a.auraReady = s.Time + auraGap
	}
	if s.Time+timeEps < a.auraReady {
		return
	}
	a.auraReady = s.Time + auraGap
	amt := float64(layers) * auraPerLayer
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: amt}
	}
}

func (a *蛆) tickGrowth(ctx unit.Context, s unit.Sense) {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID != s.Self.ID || o.Kind != KindBaby {
			continue
		}
		ready, ok := a.babyReady[o.ID]
		if !ok || s.Time+timeEps < ready {
			continue
		}
		a.pendingHP = append(a.pendingHP, inheritReq{x: o.X, y: o.Y, hp: o.HP})
		ctx.Out <- unit.Spawn{
			Kind: KindFly, X: o.X, Y: o.Y, VX: o.VX, VY: o.VY,
			OwnerID: ctx.ID, Slot: a.slot,
		}
		ctx.Out <- unit.Despawn{UnitID: o.ID}
		delete(a.babyReady, o.ID)
		a.awaitSpawn = true
	}
}

func (a *蛆) tickHatch(ctx unit.Context, s unit.Sense) {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID != s.Self.ID || o.Kind != KindEgg {
			continue
		}
		ready, ok := a.eggReady[o.ID]
		if !ok || s.Time+timeEps < ready {
			continue
		}
		ang := float64(o.ID%628) / 100
		ctx.Out <- unit.Spawn{
			Kind:    KindBaby,
			X:       o.X,
			Y:       o.Y,
			VX:      math.Cos(ang) * babyCruise,
			VY:      math.Sin(ang) * babyCruise,
			OwnerID: ctx.ID,
			Slot:    a.slot,
		}
		ctx.Out <- unit.Despawn{UnitID: o.ID}
		delete(a.eggReady, o.ID)
		a.awaitSpawn = true
	}
}

func (a *蛆) applyInherit(ctx unit.Context, s unit.Sense) {
	if a.knownFly == nil {
		a.knownFly = map[uint64]struct{}{}
	}
	if len(a.pendingHP) > 0 {
		var left []inheritReq
		for _, req := range a.pendingHP {
			bestID := uint64(0)
			bestD := 1e18
			for i := range s.Nearby {
				o := &s.Nearby[i]
				if o.OwnerID != s.Self.ID || o.Kind != KindFly {
					continue
				}
				if _, seen := a.knownFly[o.ID]; seen {
					continue
				}
				d := math.Hypot(o.X-req.x, o.Y-req.y)
				if d < bestD {
					bestD = d
					bestID = o.ID
				}
			}
			if bestID != 0 && bestD <= flyRadius*2+8 {
				hp := req.hp
				if hp > flyHP {
					hp = flyHP
				}
				if hp < 1 {
					hp = 1
				}
				ctx.Out <- unit.SetHP{UnitID: bestID, HP: hp, MaxHP: flyHP}
				a.knownFly[bestID] = struct{}{}
				continue
			}
			left = append(left, req)
		}
		a.pendingHP = left
	}
	live := map[uint64]struct{}{}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID == s.Self.ID && o.Kind == KindFly {
			live[o.ID] = struct{}{}
			a.knownFly[o.ID] = struct{}{}
		}
	}
	for id := range a.knownFly {
		if _, ok := live[id]; !ok {
			delete(a.knownFly, id)
		}
	}
}

func canonPair(a, b uint64) flyPair {
	if a > b {
		a, b = b, a
	}
	return flyPair{a: a, b: b}
}

func (a *蛆) setFlyPass(ctx unit.Context, id uint64, hold bool) {
	if a.flyPassing[id] == hold {
		return
	}
	a.flyPassing[id] = hold
	ctx.Out <- unit.Pass{UnitID: id, Hold: hold}
}

func (a *蛆) onBreedCD(id uint64, now float64) bool {
	until, ok := a.breedUntil[id]
	return ok && now+timeEps < until
}

func (a *蛆) tickFlies(ctx unit.Context, s unit.Sense) {
	type node struct {
		id   uint64
		x, y float64
	}
	var flies []node
	live := map[uint64]struct{}{}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID != s.Self.ID || o.Kind != KindFly {
			continue
		}
		live[o.ID] = struct{}{}
		flies = append(flies, node{id: o.ID, x: o.X, y: o.Y})
	}
	for id := range a.breedUntil {
		if _, ok := live[id]; !ok {
			delete(a.breedUntil, id)
		}
	}
	for id := range a.flyPassing {
		if _, ok := live[id]; !ok {
			delete(a.flyPassing, id)
		}
	}
	for id := range a.flyWasCD {
		if _, ok := live[id]; !ok {
			delete(a.flyWasCD, id)
		}
	}
	for p := range a.overlapFrom {
		if _, ok := live[p.a]; !ok {
			delete(a.overlapFrom, p)
			continue
		}
		if _, ok := live[p.b]; !ok {
			delete(a.overlapFrom, p)
		}
	}

	sort.Slice(flies, func(i, j int) bool { return flies[i].id < flies[j].id })
	matched := map[uint64]uint64{}
	for i := range flies {
		fa := flies[i]
		if a.onBreedCD(fa.id, s.Time) {
			a.setFlyPass(ctx, fa.id, false)
			continue
		}
		if _, ok := matched[fa.id]; ok {
			continue
		}
		best := -1
		bestD := attractR + 1
		for j := range flies {
			if j == i {
				continue
			}
			fb := flies[j]
			if a.onBreedCD(fb.id, s.Time) {
				continue
			}
			if _, ok := matched[fb.id]; ok {
				continue
			}
			d := math.Hypot(fa.x-fb.x, fa.y-fb.y)
			if d <= attractR && d < bestD {
				bestD = d
				best = j
			}
		}
		if best < 0 {
			a.setFlyPass(ctx, fa.id, false)
			continue
		}
		fb := flies[best]
		matched[fa.id] = fb.id
		matched[fb.id] = fa.id
	}

	activePairs := map[flyPair]struct{}{}
	for id, pid := range matched {
		if id > pid {
			continue
		}
		var fa, fb node
		for _, f := range flies {
			if f.id == id {
				fa = f
			}
			if f.id == pid {
				fb = f
			}
		}
		pair := canonPair(fa.id, fb.id)
		activePairs[pair] = struct{}{}
		a.setFlyPass(ctx, fa.id, true)
		a.setFlyPass(ctx, fb.id, true)
		pull(ctx, fa.id, fa.x, fa.y, fb.x, fb.y)
		pull(ctx, fb.id, fb.x, fb.y, fa.x, fa.y)

		dist := math.Hypot(fa.x-fb.x, fa.y-fb.y)
		if dist > overlapR {
			delete(a.overlapFrom, pair)
			continue
		}
		from, ok := a.overlapFrom[pair]
		if !ok {
			a.overlapFrom[pair] = s.Time
			from = s.Time
		}
		ctx.Out <- unit.FX{
			Name: "fly-overlap", Kind: KindFly,
			X: fa.x, Y: fa.y, VX: fb.x, VY: fb.y, Slot: a.slot,
			Amount: s.Time - from,
		}
		if s.Time+timeEps < from+overlapNeed {
			continue
		}
		mx, my := (fa.x+fb.x)/2, (fa.y+fb.y)/2
		ctx.Out <- unit.Spawn{
			Kind: KindEgg, X: mx, Y: my,
			OwnerID: ctx.ID, Slot: a.slot,
		}
		a.awaitSpawn = true
		ctx.Out <- unit.FX{Name: "fly-egg", Kind: KindEgg, X: mx, Y: my, Slot: a.slot}
		a.breedUntil[fa.id] = s.Time + breedCD
		a.breedUntil[fb.id] = s.Time + breedCD
		a.setFlyPass(ctx, fa.id, false)
		a.setFlyPass(ctx, fb.id, false)
		delete(a.overlapFrom, pair)
	}
	for p := range a.overlapFrom {
		if _, ok := activePairs[p]; !ok {
			delete(a.overlapFrom, p)
		}
	}
	for _, f := range flies {
		if _, ok := matched[f.id]; ok {
			continue
		}
		a.setFlyPass(ctx, f.id, false)
	}
	for _, f := range flies {
		a.pulseFlyEye(ctx, s, f.id, f.x, f.y)
	}
}

// pulseFlyEye：冷却中红眼，可繁殖白眼；刚从冷却出来飘「抖擞精神」。
func (a *蛆) pulseFlyEye(ctx unit.Context, s unit.Sense, id uint64, x, y float64) {
	cd := a.onBreedCD(id, s.Time)
	prev, seen := a.flyWasCD[id]
	if seen && prev == cd {
		return
	}
	a.flyWasCD[id] = cd
	amt := 0.0
	if cd {
		amt = 1
	}
	ctx.Out <- unit.FX{
		Name: "fly-eye", Kind: KindFly, UnitID: id,
		X: x, Y: y, Slot: a.slot, Amount: amt,
	}
	if seen && prev && !cd {
		ctx.Out <- unit.FX{
			Name: "spirit", Kind: KindFly, UnitID: id,
			X: x, Y: y, Slot: a.slot,
		}
	}
}

func pull(ctx unit.Context, id uint64, x, y, tx, ty float64) {
	dx, dy := tx-x, ty-y
	dist := math.Hypot(dx, dy)
	if dist < 1e-6 {
		return
	}
	ctx.Out <- unit.SetVelocity{
		UnitID: id,
		VX:     dx / dist * flyCruise,
		VY:     dy / dist * flyCruise,
	}
}
