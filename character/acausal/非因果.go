package 非因果

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"

	_ "xqdj/map"
)

//go:embed fx
var assets embed.FS

const (
	KindAcausal      = "非因果"
	KindSelfGhost    = "非因果自身影"
	KindFighterGhost = "非因果战斗机影"
	KindMinionGhost  = "非因果活随从影"
	KindLockShot     = "非因果锁影弹"
)

const (
	bodyRadius = 18.0
	bodyHP     = 100.0
	bodySpeed  = 160.0
	bodyVision = 9999.0
	bodyColor  = "#d7c4ff"
	shotWait   = 0.5
	doomDamage = 9999.0
)

func init() {
	p := unit.NewPack(KindAcausal, assets)
	p.Register(unit.Spec{
		Kind:    KindAcausal,
		Role:    unit.RoleFighter,
		Radius:  bodyRadius,
		MaxHP:   bodyHP,
		Speed:   bodySpeed,
		Vision:  bodyVision,
		Fighter: true,
		Look:    unit.Look{Color: bodyColor, FX: []string{"acausal"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &非因果{silent: info.Silent}
	})
	p.Register(unit.Spec{
		Kind:     KindSelfGhost,
		Role:     unit.RoleHelper,
		Radius:   bodyRadius,
		MaxHP:    1,
		Nonsolid: true,
		Look:     unit.Look{Color: bodyColor, FX: []string{"acausal-ghost"}},
	}, func(unit.SpawnInfo) unit.Actor { return selfGhost{} })
	p.Register(unit.Spec{
		Kind:     KindFighterGhost,
		Role:     unit.RoleHelper,
		Radius:   bodyRadius,
		MaxHP:    1,
		Nonsolid: true,
		Look:     unit.Look{Color: bodyColor, FX: []string{"acausal-ghost"}},
	}, func(unit.SpawnInfo) unit.Actor { return fighterGhost{} })
	p.Register(unit.Spec{
		Kind:   KindMinionGhost,
		Role:   unit.RoleMinion,
		Radius: 14,
		MaxHP:  9999,
		Mortal: true,
		Look:   unit.Look{Color: bodyColor, FX: []string{"acausal-ghost"}},
	}, func(unit.SpawnInfo) unit.Actor { return minionGhost{} })
	p.Register(unit.Spec{
		Kind:     KindLockShot,
		Role:     unit.RoleHelper,
		Radius:   6,
		MaxHP:    1,
		Nonsolid: true,
		Look:     unit.Look{Color: bodyColor, FX: []string{"acausal-shot"}},
	}, func(unit.SpawnInfo) unit.Actor { return lockShot{} })
}

type selfGhost struct{}

func (selfGhost) Handle(unit.Context, unit.Event) {}

type fighterGhost struct{}

func (fighterGhost) Handle(unit.Context, unit.Event) {}

type minionGhost struct{}

func (minionGhost) Handle(ctx unit.Context, ev unit.Event) {
	if d, ok := ev.(unit.IncomingDamage); ok {
		unit.ConfirmHit(ctx, d)
	}
}

type lockShot struct{}

func (lockShot) Handle(unit.Context, unit.Event) {}

type ghostRef struct {
	track int
	id    uint64
	hp    float64
	dead  bool
	ready bool
}

type 非因果 struct {
	silent bool
	booted bool
	tape   *Tape
	rng    *rand.Rand
	slot   int
	x, y   float64
	hp     float64

	self     ghostRef
	fighters []ghostRef
	minions  []ghostRef
	shadow   []float64

	seenMinion map[uint64]taggedMinion
	nextIndex  map[minionKey]int

	shotID    uint64
	shotX     float64
	shotY     float64
	shotR     float64
	shotDmg   float64
	shotSpeed float64
	shotBin   int
	shotLive  bool
	shotWait  bool
	nextShot  float64
	wantSpawn bool
	specSent  bool
	lockTrack int
	dead      bool
}

type minionKey struct {
	slot, seat int
	kind       string
}

type taggedMinion struct {
	key   minionKey
	index int
}

func (a *非因果) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		unit.ConfirmHit(ctx, e)
	case unit.Sense:
		if a.silent {
			return
		}
		a.onSense(ctx, e)
	}
}

func (a *非因果) onSense(ctx unit.Context, s unit.Sense) {
	if a.dead {
		return
	}
	if !a.booted {
		a.boot(ctx, s)
	}
	if a.tape == nil {
		return
	}
	a.x, a.y = s.Self.X, s.Self.Y
	a.hp = s.Self.HP
	a.slot = s.Self.Slot
	a.syncGhosts(ctx, s)
	a.moveGhosts(ctx, s)
	if a.tape.DeathTime >= 0 && s.Time+1e-9 >= a.tape.DeathTime {
		a.dead = true
		if a.hp > 0 {
			ctx.Out <- unit.Damage{From: ctx.ID, To: ctx.ID, Amount: a.hp}
		}
	}
	a.doom(ctx, s)
	if a.dead {
		return
	}
	a.stepBullet(ctx, s)
}

func (a *非因果) boot(ctx unit.Context, s unit.Sense) {
	a.booted = true
	a.rng = unit.NewRNG(ctx.Stream)
	a.tape = Current()
	a.seenMinion = map[uint64]taggedMinion{}
	a.nextIndex = map[minionKey]int{}
	a.lockTrack = -1
	if a.tape == nil {
		return
	}
	a.fighters = make([]ghostRef, len(a.tape.Fighters))
	a.shadow = make([]float64, len(a.tape.Fighters))
	if len(a.tape.Self) > 0 {
		x, y := sampleAt(a.tape.Self, 0)
		ctx.Out <- ctx.Born(unit.Spawn{
			Kind: KindSelfGhost, X: x, Y: y,
			OwnerID: ctx.ID, Slot: s.Self.Slot,
		})
	}
	for i, ft := range a.tape.Fighters {
		a.shadow[i] = ft.MaxHP
		x, y := sampleAt(ft.Samples, 0)
		ctx.Out <- ctx.Born(unit.Spawn{
			Kind: KindFighterGhost, X: x, Y: y,
			OwnerID: ctx.ID, Slot: ft.Slot,
		})
	}
	a.minions = make([]ghostRef, len(a.tape.Minions))
	for i, mt := range a.tape.Minions {
		a.minions[i].hp = mt.MaxHP
		x, y := sampleAt(mt.Samples, 0)
		ctx.Out <- ctx.Born(unit.Spawn{
			Kind: KindMinionGhost, X: x, Y: y,
			OwnerID: ctx.ID, Slot: mt.OwnerSlot,
		})
	}
	a.wantSpawn = true
	a.emitSpec(ctx, s)
}

func (a *非因果) emitSpec(ctx unit.Context, s unit.Sense) {
	if a.specSent || a.tape == nil {
		return
	}
	a.specSent = true
	buckets := Bucket(a.tape.Bins, displayBins)
	for i, h := range buckets {
		ctx.Out <- unit.FX{
			Name: "spec", Kind: ctx.Kind, UnitID: ctx.ID, Slot: s.Self.Slot,
			X: float64(i), Y: float64(displayBins), Amount: h * spectrumCoeff,
		}
	}
}

func (a *非因果) syncGhosts(ctx unit.Context, s unit.Sense) {
	var fighterIDs, minionIDs []uint64
	for _, o := range s.Nearby {
		if o.OwnerID != ctx.ID {
			continue
		}
		switch o.Kind {
		case KindSelfGhost:
			if !a.self.ready {
				a.self.id = o.ID
				a.self.ready = true
				a.self.hp = -1
				ctx.Out <- unit.NoHealthNumbers{UnitID: o.ID, Hold: true}
				ctx.Out <- unit.StackMark{UnitID: o.ID, Kind: KindAcausal, Delta: 1}
			}
		case KindFighterGhost:
			fighterIDs = append(fighterIDs, o.ID)
		case KindMinionGhost:
			minionIDs = append(minionIDs, o.ID)
		case KindLockShot:
			if a.shotLive && a.shotID == 0 {
				a.shotID = o.ID
				ctx.Out <- unit.SetRadius{UnitID: o.ID, Radius: a.shotR}
				ctx.Out <- unit.StackMark{UnitID: o.ID, Kind: "谱", Delta: a.shotBin + 1}
			}
		}
	}
	sortIDs(fighterIDs)
	sortIDs(minionIDs)
	for i := range a.fighters {
		if a.fighters[i].ready {
			if !containsID(fighterIDs, a.fighters[i].id) {
				a.fighters[i].dead = true
			}
			continue
		}
		if i < len(fighterIDs) {
			a.fighters[i].id = fighterIDs[i]
			a.fighters[i].ready = true
			ctx.Out <- unit.NoHealthNumbers{UnitID: fighterIDs[i], Hold: true}
			ctx.Out <- unit.StackMark{UnitID: fighterIDs[i], Kind: a.tape.Fighters[i].Kind, Delta: 1}
			a.pushShadow(ctx, i)
			if r := a.tape.Fighters[i].Radius; r > 0 {
				ctx.Out <- unit.SetRadius{UnitID: fighterIDs[i], Radius: r}
			}
		}
	}
	for i := range a.minions {
		if a.minions[i].ready {
			if !containsID(minionIDs, a.minions[i].id) {
				a.minions[i].dead = true
			}
			continue
		}
		if i < len(minionIDs) {
			id := minionIDs[i]
			a.minions[i].id = id
			a.minions[i].ready = true
			hp := a.tape.Minions[i].MaxHP
			a.minions[i].hp = hp
			ctx.Out <- unit.SetHP{UnitID: id, HP: hp, MaxHP: hp}
			ctx.Out <- unit.Pass{UnitID: id, Hold: true}
			ctx.Out <- unit.StackMark{UnitID: id, Kind: a.tape.Minions[i].Kind, Delta: 1}
			if r := a.tape.Minions[i].Radius; r > 0 {
				ctx.Out <- unit.SetRadius{UnitID: id, Radius: r}
			}
		}
	}
	for _, o := range s.Nearby {
		if o.Kind != KindMinionGhost || o.OwnerID != ctx.ID {
			continue
		}
		for i := range a.minions {
			if a.minions[i].id == o.ID {
				a.minions[i].hp = o.HP
				if o.HP <= 0 {
					a.minions[i].dead = true
				}
			}
		}
	}
}

func (a *非因果) moveGhosts(ctx unit.Context, s unit.Sense) {
	if a.self.ready && !a.self.dead {
		x, y := sampleAt(a.tape.Self, s.Time)
		ctx.Out <- unit.Teleport{UnitID: a.self.id, X: x, Y: y}
		hp := sampleHP(a.tape.SelfHP, s.Time)
		if math.Abs(hp-a.self.hp) > 0.05 {
			a.self.hp = hp
			ctx.Out <- unit.SetHP{UnitID: a.self.id, HP: hp, MaxHP: bodyHP}
		}
	}
	for i, g := range a.fighters {
		if !g.ready || g.dead {
			continue
		}
		x, y := sampleAt(a.tape.Fighters[i].Samples, s.Time)
		ctx.Out <- unit.Teleport{UnitID: g.id, X: x, Y: y}
	}
	for i, g := range a.minions {
		if !g.ready || g.dead {
			continue
		}
		x, y := sampleAt(a.tape.Minions[i].Samples, s.Time)
		ctx.Out <- unit.Teleport{UnitID: g.id, X: x, Y: y}
	}
}

func (a *非因果) doom(ctx unit.Context, s unit.Sense) {
	for i := range a.fighters {
		if a.shadow[i] > 0 {
			continue
		}
		if id, ok := a.realFighter(s, a.tape.Fighters[i].Slot, a.tape.Fighters[i].Seat); ok {
			ctx.Out <- unit.Damage{From: ctx.ID, To: id, Amount: doomDamage}
		}
	}
	a.noteRealMinions(s)
	for i, g := range a.minions {
		if !g.dead {
			continue
		}
		if id, ok := a.realMinion(a.tape.Minions[i]); ok {
			if stillThere(s, id) {
				ctx.Out <- unit.Damage{From: ctx.ID, To: id, Amount: doomDamage}
			}
		}
	}
}

func (a *非因果) stepBullet(ctx unit.Context, s unit.Sense) {
	if !a.shotLive {
		if a.wantSpawn || (!a.shotWait && s.Time+1e-9 >= a.nextShot) || (a.shotWait && s.Time+1e-9 >= a.nextShot) {
			a.spawnShot(ctx, s)
		}
		return
	}
	if a.shotID == 0 {
		return
	}
	lock := a.pickLock(s)
	a.lockTrack = lock
	tx, ty := a.x, a.y
	if lock >= 0 {
		tx, ty = sampleAt(a.tape.Fighters[lock].Samples, s.Time)
	}
	dx, dy := tx-a.shotX, ty-a.shotY
	dist := math.Hypot(dx, dy)
	step := a.shotSpeed * tickDT
	nx, ny := tx, ty
	if dist > step && dist > 1e-9 {
		nx = a.shotX + dx/dist*step
		ny = a.shotY + dy/dist*step
	}
	circles, walls, outside := a.hitWorld(s)
	kind, id := stepShot(a.shotX, a.shotY, nx, ny, a.shotR, circles, walls, outside)
	if kind == hitNone {
		a.shotX, a.shotY = nx, ny
		ctx.Out <- unit.Teleport{UnitID: a.shotID, X: nx, Y: ny}
		return
	}
	a.killShot(ctx, s)
	switch kind {
	case hitFighterGhost:
		a.hitFighterGhost(ctx, s, id)
	case hitMinionGhost:
		ctx.Out <- unit.Damage{From: ctx.ID, To: id, Amount: a.shotDmg}
	}
}

func (a *非因果) spawnShot(ctx unit.Context, s unit.Sense) {
	a.wantSpawn = false
	a.shotWait = false
	u := a.rng.Float64()
	shown, dmg, speed := Draw(a.tape.Bins, u)
	a.shotBin = shown
	a.shotDmg = dmg
	a.shotSpeed = speed
	a.shotR = shotRadius(dmg)
	a.shotX, a.shotY = a.x, a.y
	a.shotLive = true
	a.shotID = 0
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: KindLockShot, X: a.x, Y: a.y,
		OwnerID: ctx.ID, Slot: a.slot,
	})
	ctx.Out <- unit.FX{
		Name: "pick", Kind: ctx.Kind, UnitID: ctx.ID, Slot: s.Self.Slot,
		X: float64(shown), Y: float64(displayBins), Amount: dmg, VX: speed,
	}
}

func (a *非因果) killShot(ctx unit.Context, s unit.Sense) {
	if a.shotID != 0 {
		ctx.Out <- unit.Despawn{UnitID: a.shotID}
	}
	a.shotLive = false
	a.shotID = 0
	a.shotWait = true
	a.nextShot = s.Time + shotWait
}

func (a *非因果) hitFighterGhost(ctx unit.Context, s unit.Sense, id uint64) {
	for i, g := range a.fighters {
		if g.id != id || a.shadow[i] <= 0 {
			continue
		}
		a.shadow[i] -= a.shotDmg
		if a.shadow[i] < 0 {
			a.shadow[i] = 0
		}
		a.pushShadow(ctx, i)
		if a.shadow[i] == 0 {
			if rid, ok := a.realFighter(s, a.tape.Fighters[i].Slot, a.tape.Fighters[i].Seat); ok {
				ctx.Out <- unit.Damage{From: ctx.ID, To: rid, Amount: doomDamage}
			}
		}
		return
	}
}

func (a *非因果) pickLock(s unit.Sense) int {
	best := -1
	bestAim := 999
	bestDist := 0.0
	for i, g := range a.fighters {
		if !g.ready || g.dead || a.shadow[i] <= 0 {
			continue
		}
		x, y := sampleAt(a.tape.Fighters[i].Samples, s.Time)
		aim := 15
		if o, ok := a.realSnap(s, a.tape.Fighters[i].Slot, a.tape.Fighters[i].Seat); ok && o.AimPriority > 0 {
			aim = int(o.AimPriority)
		}
		d := math.Hypot(x-a.x, y-a.y)
		if best < 0 || aim < bestAim || (aim == bestAim && d < bestDist) {
			best = i
			bestAim = aim
			bestDist = d
		}
	}
	return best
}

func (a *非因果) hitWorld(s unit.Sense) ([]circle, []wallSeg, func(x, y float64) (float64, bool)) {
	var circles []circle
	for i, g := range a.fighters {
		if !g.ready || g.dead {
			continue
		}
		r := a.tape.Fighters[i].Radius
		if r <= 0 {
			r = bodyRadius
		}
		x, y := sampleAt(a.tape.Fighters[i].Samples, s.Time)
		circles = append(circles, circle{id: g.id, x: x, y: y, r: r, kind: hitFighterGhost})
	}
	for i, g := range a.minions {
		if !g.ready || g.dead {
			continue
		}
		r := a.tape.Minions[i].Radius
		if r <= 0 {
			r = 14
		}
		x, y := sampleAt(a.tape.Minions[i].Samples, s.Time)
		circles = append(circles, circle{id: g.id, x: x, y: y, r: r, kind: hitMinionGhost})
	}
	for _, o := range s.Nearby {
		if o.Kind == KindLockShot || o.Kind == KindSelfGhost || o.Kind == KindFighterGhost || o.Kind == KindMinionGhost {
			continue
		}
		if o.Role == unit.RoleFighter || o.Mortal {
			circles = append(circles, circle{id: o.ID, x: o.X, y: o.Y, r: o.Radius, kind: hitBody})
		}
	}
	var walls []wallSeg
	for _, w := range s.Walls {
		walls = append(walls, wallSeg{x1: w.X1, y1: w.Y1, x2: w.X2, y2: w.Y2, r: w.Radius})
	}
	outside := func(x, y float64) (float64, bool) {
		if s.Field.Shape == "" {
			return 0, false
		}
		ol := unit.MakeOutline(s.Field.Shape, s.Field.Extent)
		if ol.Contains(x, y, a.shotR) {
			return 0, false
		}
		dt := tickDT
		vx := (x - a.shotX) / dt
		vy := (y - a.shotY) / dt
		t, _, _, hit := ol.Sweep(a.shotX, a.shotY, vx, vy, 1, 0, a.shotR, dt, false)
		if !hit {
			return 1, true
		}
		if dt <= 0 {
			return 0, true
		}
		return t / dt, true
	}
	return circles, walls, outside
}

func (a *非因果) pushShadow(ctx unit.Context, i int) {
	g := a.fighters[i]
	if !g.ready || g.id == 0 {
		return
	}
	ctx.Out <- unit.SetHP{
		UnitID: g.id,
		HP:     a.shadow[i],
		MaxHP:  a.tape.Fighters[i].MaxHP,
	}
}

func (a *非因果) realFighter(s unit.Sense, slot, seat int) (uint64, bool) {
	o, ok := a.realSnap(s, slot, seat)
	if !ok {
		return 0, false
	}
	return o.ID, true
}

func (a *非因果) realSnap(s unit.Sense, slot, seat int) (unit.Snapshot, bool) {
	for _, o := range s.Nearby {
		if o.Role == unit.RoleFighter && o.Slot == slot && o.Seat == seat {
			return o, true
		}
	}
	return unit.Snapshot{}, false
}

func (a *非因果) noteRealMinions(s unit.Sense) {
	owners := map[uint64]minionKey{}
	for _, o := range s.Nearby {
		if o.Role == unit.RoleFighter {
			owners[o.ID] = minionKey{slot: o.Slot, seat: o.Seat}
		}
	}
	var ids []uint64
	meta := map[uint64]minionKey{}
	for _, o := range s.Nearby {
		if !o.Mortal || o.Kind == KindMinionGhost {
			continue
		}
		key, ok := owners[o.OwnerID]
		if !ok {
			continue
		}
		key.kind = o.Kind
		ids = append(ids, o.ID)
		meta[o.ID] = key
	}
	sortIDs(ids)
	for _, id := range ids {
		if _, ok := a.seenMinion[id]; ok {
			continue
		}
		key := meta[id]
		n := a.nextIndex[key]
		a.nextIndex[key] = n + 1
		a.seenMinion[id] = taggedMinion{key: key, index: n}
	}
}

func (a *非因果) realMinion(mt MinionTrack) (uint64, bool) {
	want := minionKey{slot: mt.OwnerSlot, seat: mt.OwnerSeat, kind: mt.Kind}
	for id, tag := range a.seenMinion {
		if tag.index == mt.Index && tag.key == want {
			return id, true
		}
	}
	return 0, false
}

func stillThere(s unit.Sense, id uint64) bool {
	for _, o := range s.Nearby {
		if o.ID == id && o.HP > 0 {
			return true
		}
	}
	return false
}

func sampleAt(samples []Sample, time float64) (float64, float64) {
	if len(samples) == 0 {
		return 0, 0
	}
	i := sampleIndex(len(samples), time)
	return samples[i].X, samples[i].Y
}

func sampleHP(hp []float64, time float64) float64 {
	if len(hp) == 0 {
		return 0
	}
	return hp[sampleIndex(len(hp), time)]
}

func sampleIndex(n int, time float64) int {
	if n <= 0 {
		return 0
	}
	i := int(math.Round(time / tickDT))
	if i < 0 {
		i = 0
	}
	if i >= n {
		i = n - 1
	}
	return i
}

func shotRadius(dmg float64) float64 {
	r := 5 + dmg*0.08
	if r < 5 {
		r = 5
	}
	if r > 16 {
		r = 16
	}
	return r
}

func sortIDs(ids []uint64) {
	for i := 1; i < len(ids); i++ {
		j := i
		for j > 0 && ids[j] < ids[j-1] {
			ids[j], ids[j-1] = ids[j-1], ids[j]
			j--
		}
	}
}

func containsID(ids []uint64, id uint64) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}
