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
	wolfBiteCD    = 0.08
	wolfBiteSpan  = 150.0
	wolfRageSpeed = 310.0
	wolfRageTime  = 3.0

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
		Speed:    wolfRageSpeed,
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

type 狼人 struct {
	arc         unit.AttachState
	phase       int
	raging      bool
	rageUntil   float64
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
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	w.bootMoon(ctx, s)
	if w.raging && s.Time+1e-9 >= w.rageUntil {
		w.raging = false
		w.advance(ctx, s)
	}
	w.checkTouches(ctx, s)
	w.syncArc(ctx, s)
	if w.raging {
		w.chase(ctx, s)
	}
}

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
	w.emitPhase(ctx, s)
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
	if w.raging {
		return
	}
	w.advance(ctx, s)
}

func (w *狼人) advance(ctx unit.Context, s unit.Sense) {
	w.phase = (w.phase + 1) % phaseCount
	w.emitPhase(ctx, s)
	if w.phase == fullMoon {
		w.startRage(ctx, s)
	}
}

func (w *狼人) startRage(ctx unit.Context, s unit.Sense) {
	w.raging = true
	w.rageUntil = s.Time + wolfRageTime
	ctx.Out <- unit.FX{
		Name:   "rage",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		X:      s.Self.X,
		Y:      s.Self.Y,
		Slot:   s.Self.Slot,
	}
}

func (w *狼人) emitPhase(ctx unit.Context, s unit.Sense) {
	ctx.Out <- unit.FX{
		Name:   "phase",
		Kind:   ctx.Kind,
		Slot:   s.Self.Slot,
		Amount: float64(w.phase),
		X:      0,
		Y:      0,
	}
}

func (w *狼人) syncArc(ctx unit.Context, s unit.Sense) {
	want, drop, cd := KindWolfArc, KindWolfBite, wolfHitCD
	if w.raging {
		want, drop, cd = KindWolfBite, KindWolfArc, wolfBiteCD
	}
	if unit.HasOwned(s, ctx.ID, drop) {
		ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: drop}
	}
	if unit.RearmAttach(s, ctx.ID, want, cd, &w.arc) {
		unit.SpawnAttach(ctx, s, want)
	}
}

func (w *狼人) chase(ctx unit.Context, s unit.Sense) {
	en := enemyOf(s)
	if en == nil {
		return
	}
	dx := en.X - s.Self.X
	dy := en.Y - s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: dx / n * wolfRageSpeed, VY: dy / n * wolfRageSpeed}
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
