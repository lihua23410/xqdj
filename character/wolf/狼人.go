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
	wolfHitCD     = 0.1
	wolfArcInner  = wolfRadius
	wolfArcOuter  = wolfRadius + 6
	wolfArcSpan   = 110.0
	wolfBiteDmg   = 13.0
	wolfBiteCD    = 0.3
	wolfBiteSpan  = 150.0
	wolfDashSpeed = 640.0
	wolfDashLock  = 0.1
	wolfDashSeek  = 0.35
	wolfDashCount = 5

	moonRadius = 34.0
	phaseCount = 8
	fullMoon   = 4 // 新月0 娥眉1 上弦2 盈凸3 满月4 亏凸5 下弦6 残月7

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
		Look:     unit.Look{Color: wolfColor, Overlay: true},
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
		Look:     unit.Look{Color: "#ffd0c8", Overlay: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &狼弧{slot: info.Slot, dmg: wolfBiteDmg}
	})
}

// biteNone 游荡积月相；biteLock 站定锁敌（开撕咬 0.1s，撞墙后 0.35s）；biteDash 沿锁定方向冲到墙。
const (
	biteNone = iota
	biteLock
	biteDash
)

type 狼人 struct {
	arc         unit.AttachState
	phase       int
	bite        int
	dashesLeft  int
	lockUntil   float64
	aimX, aimY  float64
	slot        int
	x, y        float64
	selfInMoon  bool
	enemyInMoon bool
	booted      bool
}

type 月亮 struct{}

func (月亮) Handle(unit.Context, unit.Event) {}

type 狼弧 struct {
	slot int
	dmg  float64
}

func (a *狼弧) Handle(ctx unit.Context, ev unit.Event) {
	e, ok := ev.(unit.Collision)
	if !ok || !unit.EnemyFighter(e, a.slot) {
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: a.dmg}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

func (w *狼人) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	switch e := ev.(type) {
	case unit.WallHit:
		w.onWall(ctx, e)
	case unit.Sense:
		w.slot = e.Self.Slot
		w.x, w.y = e.Self.X, e.Self.Y
		w.bootMoon(ctx, e)
		w.checkTouches(ctx, e)
		w.tickBite(ctx, e)
		w.syncArc(ctx, e)
	}
}

func (w *狼人) biting() bool { return w.bite != biteNone }

func (w *狼人) bootMoon(ctx unit.Context, s unit.Sense) {
	if w.booted {
		return
	}
	w.booted = true
	ctx.Out <- unit.Spawn{
		Kind:    KindMoon,
		X:       0,
		Y:       0,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	}
	w.emitPhase(ctx)
}

func (w *狼人) checkTouches(ctx unit.Context, s unit.Sense) {
	moon := findMoon(s, ctx.ID)
	selfHit := moon != nil && touching(s.Self, *moon)
	if selfHit && !w.selfInMoon {
		w.onTouch(ctx, s)
	}
	w.selfInMoon = selfHit

	en := enemyOf(s)
	enHit := moon != nil && en != nil && touching(*en, *moon)
	if enHit && !w.enemyInMoon {
		w.onTouch(ctx, s)
	}
	w.enemyInMoon = enHit
}

func (w *狼人) onTouch(ctx unit.Context, s unit.Sense) {
	if w.biting() {
		return
	}
	w.advance(ctx, s)
}

func (w *狼人) advance(ctx unit.Context, s unit.Sense) {
	w.phase = (w.phase + 1) % phaseCount
	w.emitPhase(ctx)
	if w.phase == fullMoon {
		w.startBite(ctx, s)
	}
}

func (w *狼人) startBite(ctx unit.Context, s unit.Sense) {
	w.bite = biteLock
	w.dashesLeft = wolfDashCount
	w.lockUntil = s.Time + wolfDashLock
	w.updateAim(s)
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

func (w *狼人) beginLock(t float64) {
	w.bite = biteLock
	w.lockUntil = t + wolfDashSeek
}

func (w *狼人) onWall(ctx unit.Context, e unit.WallHit) {
	if w.bite != biteDash {
		return
	}
	w.dashesLeft--
	if w.dashesLeft > 0 {
		w.beginLock(e.Time)
		ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: false}
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
		return
	}
	w.finishBite(ctx, e)
}

func (w *狼人) finishBite(ctx unit.Context, e unit.WallHit) {
	w.bite = biteNone
	w.dashesLeft = 0
	w.phase = 0
	w.emitPhase(ctx)
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: false}
	vx, vy := cruiseOffWall(w.aimX, w.aimY, e.NX, e.NY)
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

func (w *狼人) emitPhase(ctx unit.Context) {
	ctx.Out <- unit.FX{
		Name:   "phase",
		Kind:   ctx.Kind,
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
	if w.bite == biteLock {
		w.updateAim(s)
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
		if s.Time+1e-9 >= w.lockUntil {
			w.bite = biteDash
			ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
			ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: w.aimX * wolfDashSpeed, VY: w.aimY * wolfDashSpeed}
		}
		return
	}
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
	want, drop, cd := KindWolfArc, KindWolfBite, wolfHitCD
	if w.biting() {
		want, drop, cd = KindWolfBite, KindWolfArc, wolfBiteCD
	}
	if unit.HasOwned(s, ctx.ID, drop) {
		ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: drop}
	}
	if unit.RearmAttach(s, ctx.ID, want, cd, &w.arc) {
		unit.SpawnAttach(ctx, s, want)
	}
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
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role == unit.RoleFighter && o.Slot != s.Self.Slot {
			return o
		}
	}
	return nil
}

func touching(a, b unit.Snapshot) bool {
	return math.Hypot(a.X-b.X, a.Y-b.Y) <= a.Radius+b.Radius+1e-6
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
