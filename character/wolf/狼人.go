package 狼人

import (
	"embed"
	"math"
	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const KindWolf = "狼人"
const KindMoon = "狼人月亮"
const KindWolfArc = "狼人弧"
const KindWolfBite = "狼人撕咬弧"

const (
	wolfRadius = 18.0
	wolfSpeed  = 172.0
	wolfHP     = 100.0
	wolfVision = 9999.0

	wolfDamage    = 8.0
	wolfHitCD     = 0.3
	wolfArcInner  = wolfRadius
	wolfArcOuter  = wolfRadius + 6
	wolfArcSpan   = 110.0
	wolfBiteBase  = 6.0
	wolfBiteSpan  = 150.0
	wolfBiteTime  = 6.0
	wolfDashSpeed = 640.0
	wolfDashLock  = 0.3
	wolfDashLen   = 180.0
	wolfFanKeep   = 0.4

	moonRadius = 34.0
	phaseCount = 8
	fullMoon   = 4 // 新月0 娥眉1 上弦2 盈凸3 满月4 亏凸5 下弦6 残月7
	moonDry    = 10.0
	howlHold   = 1.0

	wolfColor = "#f4f1ea"
	moonColor = "#e8d9a0"
)

func init() {
	p := unit.NewPack(KindWolf, assets)
	p.Register(unit.Spec{
		Kind:    KindWolf,
		Role:    unit.RoleFighter,
		Radius:  wolfRadius,
		MaxHP:   wolfHP,
		Speed:   wolfSpeed,
		Vision:  wolfVision,
		Fighter: true,
		Look:    unit.Look{Color: wolfColor, Ghost: 280, Glow: true, FX: []string{"wolf"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &狼人{}
	})
	p.Register(unit.Spec{
		Kind:    KindMoon,
		Role:    unit.RoleHelper,
		Radius:  moonRadius,
		MaxHP:   1,
		Speed:   0,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: moonColor, Glow: true, FX: []string{"moon"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return 月亮{}
	})
	p.Register(unit.Spec{
		Kind:     KindWolfArc,
		Role:     unit.RoleProjectile,
		Radius:   wolfArcOuter,
		MaxHP:    1,
		Speed:    wolfSpeed,
		Vision:   0,
		Fighter:  false,
		Attach:   true,
		ArcSpan:  unit.Deg(wolfArcSpan),
		ArcInner: wolfArcInner,
		Look:     unit.Look{Color: wolfColor, Overlay: true, FX: []string{"wolf-arc"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &狼弧{slot: info.Slot, dmg: wolfDamage}
	})
	p.Register(unit.Spec{
		Kind:     KindWolfBite,
		Role:     unit.RoleProjectile,
		Radius:   wolfArcOuter,
		MaxHP:    1,
		Speed:    wolfDashSpeed,
		Vision:   0,
		Fighter:  false,
		Attach:   true,
		ArcSpan:  unit.Deg(wolfBiteSpan),
		ArcInner: wolfArcInner,
		Look:     unit.Look{Color: "#ffd0c8", Overlay: true, FX: []string{"wolf-arc"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return newBiteArc(info)
	})
}

func newBiteArc(info unit.SpawnInfo) *狼弧 {
	dmg, _ := info.Data.(float64)
	if dmg <= 0 {
		dmg = wolfBiteBase
	}
	return &狼弧{slot: info.Slot, dmg: dmg}
}

// biteNone 游荡积月相；biteLock 站定锁敌 0.3s；biteDash 沿锁定方向冲 180。
const (
	biteNone = iota
	biteLock
	biteDash
)

type 狼人 struct {
	arc        unit.AttachState
	phase      int
	bite       int
	biteUntil  float64
	biteDmg    float64
	biteHits   int
	biteArc    bool
	dashArmed  bool
	biteCut    bool
	lockUntil  float64
	dashLeft   float64
	dashX      float64
	dashY      float64
	aimX, aimY float64
	slot       int
	x, y       float64
	insideMoon map[uint64]bool
	booted     bool
	phaseAt    float64
	howling    bool
	howlUntil  float64
}

type 月亮 struct{}

func (月亮) Handle(unit.Context, unit.Event) {}

type 狼弧 struct {
	slot int
	dmg  float64
}

func (a *狼弧) Handle(ctx unit.Context, ev unit.Event) {
	e, ok := ev.(unit.Collision)
	if !ok || !unit.EnemyTarget(e, a.slot) {
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: a.dmg}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

func (w *狼人) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		w.onHit(ctx, e)
	case unit.WallHit:
		w.onWall(ctx, e)
	case unit.Sense:
		w.slot = e.Self.Slot
		w.x, w.y = e.Self.X, e.Self.Y
		w.bootMoon(ctx, e)
		w.noteBite(ctx.ID, e)
		w.checkTouches(ctx, e)
		w.tickBite(ctx, e)
		w.tickHowl(ctx, e)
		w.syncArc(ctx, e)
	}
}

func (w *狼人) onHit(ctx unit.Context, d unit.IncomingDamage) {
	amt := d.Amount
	if w.bite == biteDash {
		amt = d.KeepFan(wolfFanKeep)
	}
	ctx.Out <- unit.ConfirmDamage{Token: d.Token, UnitID: ctx.ID, Amount: amt}
}

func (w *狼人) biting() bool { return w.bite != biteNone }

func (w *狼人) bootMoon(ctx unit.Context, s unit.Sense) {
	if w.booted {
		return
	}
	w.booted = true
	rng := unit.NewRNG(ctx.Stream)
	x, y, ok := s.Field.RandomWalkable(rng, moonRadius)
	if !ok {
		x, y = s.Field.Clamp(0, 80, moonRadius)
	}
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind:    KindMoon,
		X:       x,
		Y:       y,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	})
	w.phaseAt = s.Time
	w.emitPhase(ctx)
}

func (w *狼人) checkTouches(ctx unit.Context, s unit.Sense) {
	moon := findMoon(s, ctx.ID)
	cur := map[uint64]bool{}
	if moon != nil {
		if touching(s.Self, *moon) {
			cur[s.Self.ID] = true
		}
		for i := range s.Nearby {
			o := &s.Nearby[i]
			if !unit.Hittable(*o, s.Self.Slot) {
				continue
			}
			if touching(*o, *moon) {
				cur[o.ID] = true
			}
		}
	}
	if w.insideMoon == nil {
		w.insideMoon = map[uint64]bool{}
	}
	for id := range cur {
		if !w.insideMoon[id] {
			w.onTouch(ctx, s)
		}
	}
	w.insideMoon = cur
}

func (w *狼人) onTouch(ctx unit.Context, s unit.Sense) {
	if w.biting() {
		return
	}
	w.advance(ctx, s)
}

func (w *狼人) advance(ctx unit.Context, s unit.Sense) {
	wasHowl := w.howling
	w.howling = false
	w.phase = (w.phase + 1) % phaseCount
	w.phaseAt = s.Time
	w.emitPhase(ctx)
	if w.phase == fullMoon {
		w.startBite(ctx, s)
		return
	}
	if wasHowl {
		ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}
	}
}

func (w *狼人) tickHowl(ctx unit.Context, s unit.Sense) {
	if !w.booted || w.biting() {
		return
	}
	if w.howling {
		ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
		if s.Time+1e-9 < w.howlUntil {
			return
		}
		w.howling = false
		w.advance(ctx, s)
		if !w.biting() {
			ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}
		}
		return
	}
	if s.Time+1e-9 < w.phaseAt+moonDry {
		return
	}
	w.howling = true
	w.howlUntil = s.Time + howlHold
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	ctx.Out <- unit.FX{
		Name:   "howl",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		X:      s.Self.X,
		Y:      s.Self.Y,
		Slot:   s.Self.Slot,
	}
}

func (w *狼人) startBite(ctx unit.Context, s unit.Sense) {
	if w.biteDmg == 0 {
		w.biteDmg = wolfBiteBase
	}
	w.bite = biteLock
	w.biteUntil = s.Time + wolfBiteTime
	w.lockUntil = s.Time + wolfDashLock
	w.updateAim(s)
	w.holdPass(ctx, true)
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	ctx.Out <- unit.FX{
		Name:   "rage",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		X:      s.Self.X,
		Y:      s.Self.Y,
		Slot:   s.Self.Slot,
		Amount: 1,
	}
}

func (w *狼人) beginLock(ctx unit.Context, t float64) {
	w.bite = biteLock
	w.lockUntil = t + wolfDashLock
	w.dropBiteArc(ctx)
	w.holdPass(ctx, true)
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
}

func (w *狼人) onWall(ctx unit.Context, e unit.WallHit) {
	if w.bite != biteDash {
		return
	}
	w.endSegment(ctx, e.Time, true, e.NX, e.NY)
}

func (w *狼人) endSegment(ctx unit.Context, t float64, wall bool, nx, ny float64) {
	if w.bite != biteDash {
		return
	}
	if t+1e-9 >= w.biteUntil {
		w.finishBite(ctx, wall, nx, ny)
		return
	}
	w.beginLock(ctx, t)
}

func (w *狼人) finishBite(ctx unit.Context, wall bool, nx, ny float64) {
	w.bite = biteNone
	w.phase = 0
	w.dropBiteArc(ctx)
	w.emitPhase(ctx)
	w.holdPass(ctx, false)
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}
	vx, vy := w.cruiseAlongAim()
	if wall {
		vx, vy = cruiseOffWall(w.aimX, w.aimY, nx, ny)
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: vx, VY: vy}
	ctx.Out <- unit.FX{
		Name:   "rage",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		X:      w.x,
		Y:      w.y,
		Slot:   w.slot,
		Amount: 0,
	}
}

func (w *狼人) landBite() {
	if w.biteDmg == 0 {
		w.biteDmg = wolfBiteBase
	}
	w.biteHits++
	if w.biteHits >= 2 {
		w.biteHits = 0
		w.biteDmg++
	}
}

func (w *狼人) noteBite(id uint64, s unit.Sense) {
	has := unit.HasOwned(s, id, KindWolfBite)
	if w.biteCut {
		if !has {
			w.biteCut = false
			w.biteArc = false
		}
		return
	}
	if w.bite != biteDash {
		w.biteArc = false
		return
	}
	if w.biteArc && !has {
		w.landBite()
	}
	w.biteArc = has
}

// dropBiteArc 冲刺结束就把这一道撕咬弧摘掉。自己摘的不算咬中。
func (w *狼人) dropBiteArc(ctx unit.Context) {
	if w.dashArmed || w.biteArc {
		w.biteCut = true
	}
	w.dashArmed = false
	ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: KindWolfBite}
}

func (w *狼人) emitPhase(ctx unit.Context) {
	ctx.Out <- unit.FX{
		Name:   "phase",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		Slot:   w.slot,
		Amount: float64(w.phase),
		X:      0,
		Y:      0,
	}
}

func (w *狼人) tickBite(ctx unit.Context, s unit.Sense) {
	if !w.biting() {
		return
	}
	w.holdPass(ctx, true)
	if w.bite == biteLock {
		w.updateAim(s)
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
		if s.Time+1e-9 >= w.biteUntil {
			w.finishBite(ctx, false, 0, 0)
			return
		}
		if s.Time+1e-9 >= w.lockUntil {
			w.beginDash(ctx, s)
		}
		return
	}
	w.steerDash(ctx)
	w.dashLeft -= math.Hypot(s.Self.X-w.dashX, s.Self.Y-w.dashY)
	w.dashX, w.dashY = s.Self.X, s.Self.Y
	if w.dashLeft <= 1e-6 {
		w.endSegment(ctx, s.Time, false, 0, 0)
	}
}

func (w *狼人) beginDash(ctx unit.Context, s unit.Sense) {
	w.bite = biteDash
	w.dashLeft = wolfDashLen
	w.dashX, w.dashY = s.Self.X, s.Self.Y
	w.steerDash(ctx)
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}
}

func (w *狼人) holdPass(ctx unit.Context, hold bool) {
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: hold}
}

func (w *狼人) steerDash(ctx unit.Context) {
	w.holdPass(ctx, true)
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: w.aimX * wolfDashSpeed, VY: w.aimY * wolfDashSpeed}
}

func (w *狼人) updateAim(s unit.Sense) {
	en := enemyOf(s)
	if en == nil {
		if math.Hypot(w.aimX, w.aimY) >= 1e-6 {
			return
		}
		dx, dy := s.Self.VX, s.Self.VY
		if n := math.Hypot(dx, dy); n >= 1e-6 {
			w.aimX, w.aimY = dx/n, dy/n
			return
		}
		w.aimX, w.aimY = 1, 0
		return
	}
	dx := en.X - s.Self.X
	dy := en.Y - s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	w.aimX, w.aimY = dx/n, dy/n
}

func (w *狼人) syncArc(ctx unit.Context, s unit.Sense) {
	if w.bite == biteDash {
		if unit.HasOwned(s, ctx.ID, KindWolfArc) {
			ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: KindWolfArc}
		}
		if w.dashArmed || unit.HasOwned(s, ctx.ID, KindWolfBite) {
			w.dashArmed = true
			return
		}
		w.spawnArc(ctx, s, KindWolfBite)
		w.dashArmed = true
		w.biteArc = true
		return
	}
	if unit.HasOwned(s, ctx.ID, KindWolfBite) {
		w.biteCut = true
		ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: KindWolfBite}
	}
	w.dashArmed = false
	if w.biting() {
		if unit.HasOwned(s, ctx.ID, KindWolfArc) {
			ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: KindWolfArc}
		}
		w.arc = unit.AttachState{}
		return
	}
	if unit.RearmAttach(s, ctx.ID, KindWolfArc, wolfHitCD, &w.arc) {
		w.spawnArc(ctx, s, KindWolfArc)
	}
}

func (w *狼人) spawnArc(ctx unit.Context, s unit.Sense, kind string) {
	sp := ctx.Born(unit.Spawn{
		Kind:    kind,
		X:       s.Self.X,
		Y:       s.Self.Y,
		VX:      s.Self.VX,
		VY:      s.Self.VY,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	})
	if kind == KindWolfBite {
		if w.biteDmg == 0 {
			w.biteDmg = wolfBiteBase
		}
		sp.Data = w.biteDmg
	}
	ctx.Out <- sp
}

func findMoon(s unit.Sense, owner uint64) *unit.Snapshot {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind == KindMoon && o.OwnerID == owner {
			return o
		}
	}
	return nil
}

func enemyOf(s unit.Sense) *unit.Snapshot {
	return unit.Seek(s)
}

func touching(a, b unit.Snapshot) bool {
	return math.Hypot(a.X-b.X, a.Y-b.Y) <= a.Radius+b.Radius+1e-6
}

func (w *狼人) cruiseAlongAim() (float64, float64) {
	ax, ay := w.aimX, w.aimY
	if math.Hypot(ax, ay) < 1e-6 {
		ax, ay = 1, 0
	}
	n := math.Hypot(ax, ay)
	return ax / n * wolfSpeed, ay / n * wolfSpeed
}

func cruiseOffWall(ax, ay, nx, ny float64) (float64, float64) {
	nn := math.Hypot(nx, ny)
	if nn < 1e-6 {
		nx, ny, nn = 1, 0, 1
	}
	nx, ny = nx/nn, ny/nn
	if math.Hypot(ax, ay) < 1e-6 {
		ax, ay = -nx, -ny
	}
	dot := ax*nx + ay*ny
	rx, ry := ax-2*dot*nx, ay-2*dot*ny
	n := math.Hypot(rx, ry)
	if n < 1e-6 {
		rx, ry, n = -nx, -ny, 1
	}
	return rx / n * wolfSpeed, ry / n * wolfSpeed
}
