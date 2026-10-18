package 无下限术士

import (
	"embed"
	"math"
	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const KindTwin = "无下限术士"
const KindRed = "红半"
const KindBlue = "蓝半"
const KindTwinArc = "无下限术士弧"
const KindTwinBlueArc = "无下限弧"

const (
	twinRadius    = 18.0
	twinSpeed     = 200.0
	twinHP        = 150.0
	halfHP        = 9999.0
	twinDamage    = 7.0
	twinHitCD     = 0.1
	twinGap       = 2.0
	twinVision    = 9999.0
	twinPull      = 300.0
	twinPush      = -100.0
	twinArcInner  = twinRadius
	twinArcOuter  = twinRadius + 2
	twinArcSpan   = 90.0
	twinRedColor  = "#e24b4b"
	twinBlueColor = "#4b7be2"

	purpleKind   = "紫弹"
	purpleRadius = 32.0
	purpleSpeed  = 300.0
	purpleDamage = 12.0
	purpleCD     = 0.5

	// 互相吸引：圆心距在 (attractMin, attractMax] 且不在互吸冷却内时，双向改速度。
	attractMin = 36.0
	attractMax = 80.0
	attractCD  = 7.0

	// 开局 grace 内不做距离互吸；两球交替念偈。grace 结束后无视距离互吸一次。
	// 第一次半圆相撞必出紫弹，挡视线的墙挡不住这一发；之后再撞仍要索敌到人。
	bootGrace = 3.0
	chantGap  = bootGrace / 4
)

var chantPhrases = [...]string{
	"【九纲】",
	"【偏光】",
	"【乌与声明】",
	"【表里之间】",
}

func init() {
	p := unit.NewPack(KindTwin, assets)
	p.Register(unit.Spec{
		Kind:     KindTwin,
		Role:     unit.RoleFighter,
		Radius:   0,
		MaxHP:    twinHP,
		Speed:    twinSpeed,
		Vision:   twinVision,
		Fighter:  true,
		Nonsolid: true,
		Look:     unit.Look{Color: twinRedColor},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &无下限术士{slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:        KindRed,
		Role:        unit.RoleMinion,
		Radius:      twinRadius,
		MaxHP:       halfHP,
		Speed:       twinSpeed,
		Vision:      twinVision,
		Mortal:      true,
		Semi:        true,
		Cruise:      true,
		AimPriority: unit.DefaultFighterAim,
		Look:        unit.Look{Color: twinRedColor, FX: []string{"pull", "bond"}, ShareHP: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &半{owner: info.OwnerID, slot: info.Slot, red: true}
	})
	p.Register(unit.Spec{
		Kind:     KindTwinArc,
		Role:     unit.RoleProjectile,
		Radius:   twinArcOuter,
		MaxHP:    1,
		Speed:    twinSpeed,
		Vision:   0,
		Fighter:  false,
		Semi:     true,
		Attach:   true,
		ArcSpan:  unit.Deg(twinArcSpan),
		ArcInner: twinArcInner,
		Look:     unit.Look{Color: twinRedColor, Overlay: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &术士弧{slot: info.Slot, dmg: twinDamage}
	})
	p.Register(unit.Spec{
		Kind:        KindBlue,
		Role:        unit.RoleMinion,
		Radius:      twinRadius,
		MaxHP:       halfHP,
		Speed:       twinSpeed,
		Vision:      twinVision,
		Mortal:      true,
		Semi:        true,
		AimPriority: unit.DefaultFighterAim,
		Look:        unit.Look{Color: twinBlueColor, FX: []string{"push", "bond"}, ShareHP: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &半{owner: info.OwnerID, slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:     KindTwinBlueArc,
		Role:     unit.RoleProjectile,
		Radius:   twinArcOuter,
		MaxHP:    1,
		Speed:    twinSpeed,
		Vision:   0,
		Fighter:  false,
		Semi:     true,
		Attach:   true,
		ArcSpan:  unit.Deg(twinArcSpan),
		ArcInner: twinArcInner,
		Look:     unit.Look{Color: twinBlueColor, Overlay: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &术士弧{slot: info.Slot, dmg: twinDamage}
	})
	p.Register(unit.Spec{
		Kind:      purpleKind,
		Role:      unit.RoleProjectile,
		Radius:    purpleRadius,
		MaxHP:     1,
		Speed:     purpleSpeed,
		Vision:    0,
		Fighter:   false,
		PassWalls: true,
		Look:      unit.Look{Color: "#b44cff", Glow: true, Trail: true, Overlay: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &紫弹{slot: info.Slot}
	})
}

type 无下限术士 struct {
	slot    int
	spawned bool
}

func (d *无下限术士) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok || d.spawned {
		return
	}
	d.spawned = true
	ctx.Out <- unit.NoHealthNumbers{UnitID: ctx.ID, Hold: true}
	unit.SetAim(ctx, ctx.ID, 0)
	split(ctx, s)
}

func split(ctx unit.Context, s unit.Sense) {
	sp := math.Hypot(s.Self.VX, s.Self.VY)
	vx, vy := s.Self.VX, s.Self.VY
	if sp < 1e-6 {
		sp = twinSpeed
		vx, vy = twinSpeed, 0
	}
	px, py := vx/sp, vy/sp
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: KindRed, X: s.Self.X, Y: s.Self.Y, VX: vx, VY: vy,
		OwnerID: ctx.ID, Slot: s.Self.Slot,
	})
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind:    KindBlue,
		X:       s.Self.X - px*twinGap,
		Y:       s.Self.Y - py*twinGap,
		VX:      -vx,
		VY:      -vy,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	})
	ctx.Out <- unit.FX{
		Name: "split", Kind: KindRed,
		X: s.Self.X, Y: s.Self.Y, VX: vx, VY: vy, Slot: s.Self.Slot,
	}
	ctx.Out <- unit.FX{
		Name: "split", Kind: KindBlue,
		X: s.Self.X - px*twinGap, Y: s.Self.Y - py*twinGap, VX: -vx, VY: -vy, Slot: s.Self.Slot,
	}
}

type 半 struct {
	owner          uint64
	slot           int
	red            bool
	arc            unit.AttachState
	shotReadyAt    float64
	attractReadyAt float64
	attracting     bool
	bootPulling    bool // grace 结束后那一次无视距离互吸
	bootPullDone   bool
	chantOn        bool
	selfX          float64
	selfY          float64
	enemyX         float64
	enemyY         float64
	hasEnemy       bool
	aimX           float64
	aimY           float64
	hasAim         bool // 无视挡视线时的索敌，只给第一次相撞兜底
	opened         bool // 第一次半圆相撞的紫弹已经打出
	hold           unit.DetourHold
}

func (h *半) Handle(ctx unit.Context, ev unit.Event) {
	if passHurt(ctx, ev, h.owner) {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		kind := KindTwinBlueArc
		if h.red {
			kind = KindTwinArc
		}
		if unit.RearmAttach(e, ctx.ID, kind, twinHitCD, &h.arc) {
			unit.SpawnAttach(ctx, e, kind)
		}
		h.remember(e)
		h.tickAttract(ctx, e)
		str := twinPush
		if h.red {
			str = twinPull
		}
		twinField(ctx, e, str)
	case unit.Collision:
		if e.Other.Slot != h.slot {
			return
		}
		sib := KindBlue
		if !h.red {
			sib = KindRed
		}
		if e.Other.Kind != sib {
			return
		}
		// 半圆贴上后圆心距常略大于 36，仍落在互吸带里；不在碰撞时收束会一直改速度、冷却起不来。
		h.endAttract(ctx, e.Time, h.selfX, h.selfY)
		if h.red && h.tryShot(ctx, e, !h.opened) {
			h.opened = true
		}
	}
}

func (h *半) tickAttract(ctx unit.Context, s unit.Sense) {
	if s.Time < bootGrace {
		if h.attracting {
			h.endAttract(ctx, s.Time, s.Self.X, s.Self.Y)
		}
		h.pulseChant(ctx, s)
		return
	}
	h.clearChant(ctx, s)

	sib, ok := findSibling(s, h.red, h.slot)
	if !ok {
		h.endAttract(ctx, s.Time, s.Self.X, s.Self.Y)
		return
	}

	// grace 结束后仅一次：无视距离互吸，直到相撞（或同伴消失）。
	if !h.bootPullDone {
		h.attracting = true
		h.bootPulling = true
		pulseAttract(ctx, s, 1)
		h.pullToward(ctx, s, sib.X, sib.Y)
		return
	}

	d := math.Hypot(s.Self.X-sib.X, s.Self.Y-sib.Y)
	inBand := d > attractMin && d <= attractMax
	if h.attracting {
		if !inBand {
			h.endAttract(ctx, s.Time, s.Self.X, s.Self.Y)
			return
		}
		// 每拍重发，避免包脚本晚于首帧 FX 时观众侧永远进不了互吸态。
		pulseAttract(ctx, s, 1)
		h.pullToward(ctx, s, sib.X, sib.Y)
		return
	}
	if !inBand || s.Time < h.attractReadyAt {
		return
	}
	h.attracting = true
	pulseAttract(ctx, s, 1)
	h.pullToward(ctx, s, sib.X, sib.Y)
}

func (h *半) pulseChant(ctx unit.Context, s unit.Sense) {
	step := int(s.Time / chantGap)
	if step >= len(chantPhrases) {
		step = len(chantPhrases) - 1
	}
	// 两球交替：偶数步红半念，奇数步蓝半念；四句依次是九纲→偏光→乌与声明→表里之间。
	show := (step%2 == 0) == h.red
	if !show {
		h.clearChant(ctx, s)
		return
	}
	h.chantOn = true
	ctx.Out <- unit.FX{
		Name: "twin-chant", UnitID: ctx.ID, Kind: s.Self.Kind,
		X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot, Amount: float64(step),
	}
}

func (h *半) clearChant(ctx unit.Context, s unit.Sense) {
	if !h.chantOn {
		return
	}
	h.chantOn = false
	ctx.Out <- unit.FX{
		Name: "twin-chant", UnitID: ctx.ID, Kind: s.Self.Kind,
		X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot, Amount: -1,
	}
}

func pulseAttract(ctx unit.Context, s unit.Sense, amount float64) {
	ctx.Out <- unit.FX{
		Name: "twin-attract", UnitID: ctx.ID, Kind: s.Self.Kind,
		X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot, Amount: amount,
	}
}

func (h *半) endAttract(ctx unit.Context, now, x, y float64) {
	if !h.attracting {
		return
	}
	h.attracting = false
	if h.bootPulling {
		h.bootPulling = false
		h.bootPullDone = true
	}
	h.attractReadyAt = now + attractCD
	kind := KindBlue
	if h.red {
		kind = KindRed
	}
	ctx.Out <- unit.FX{
		Name: "twin-attract", UnitID: ctx.ID, Kind: kind,
		X: x, Y: y, Slot: h.slot, Amount: 0,
	}
}

func findSibling(s unit.Sense, red bool, slot int) (unit.Snapshot, bool) {
	want := KindBlue
	if !red {
		want = KindRed
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Slot == slot && o.Kind == want {
			return *o, true
		}
	}
	return unit.Snapshot{}, false
}

func (h *半) pullToward(ctx unit.Context, s unit.Sense, tx, ty float64) {
	wx, wy, hold := unit.Detour(s.Self.X, s.Self.Y, s.Self.Radius, tx, ty, s.Walls, h.hold, s.Time)
	h.hold = hold
	dx, dy := wx-s.Self.X, wy-s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	ctx.Out <- unit.SetVelocity{
		UnitID: ctx.ID,
		VX:     dx / n * twinSpeed,
		VY:     dy / n * twinSpeed,
	}
}

func (h *半) remember(s unit.Sense) {
	h.selfX, h.selfY = s.Self.X, s.Self.Y
	h.hasEnemy = false
	if o := unit.Seek(s); o != nil {
		h.enemyX, h.enemyY = o.X, o.Y
		h.hasEnemy = true
	}
	h.aimX, h.aimY, h.hasAim = aimIgnoringWalls(s)
}

// aimIgnoringWalls 按 Seek 的优先度与距离挑人，但不看挡视线。
// 第一次相撞时墙正压在连线上，Seek 会空，这一发仍要有方向。
func aimIgnoringWalls(s unit.Sense) (x, y float64, ok bool) {
	var best *unit.Snapshot
	bestPri := uint8(255)
	bestD2 := math.MaxFloat64
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Aimable(*o, s.Self.Slot) {
			continue
		}
		dx, dy := o.X-s.Self.X, o.Y-s.Self.Y
		d2 := dx*dx + dy*dy
		if best != nil && (o.AimPriority > bestPri || (o.AimPriority == bestPri && d2 >= bestD2)) {
			continue
		}
		best = o
		bestPri = o.AimPriority
		bestD2 = d2
	}
	if best == nil {
		return 0, 0, false
	}
	return best.X, best.Y, true
}

func (h *半) tryShot(ctx unit.Context, e unit.Collision, opening bool) bool {
	if e.Time < h.shotReadyAt {
		return false
	}
	ex, ey := h.enemyX, h.enemyY
	if !h.hasEnemy {
		if !opening || !h.hasAim {
			return false
		}
		ex, ey = h.aimX, h.aimY
	}
	mx := (h.selfX + e.Other.X) / 2
	my := (h.selfY + e.Other.Y) / 2
	dx := ex - mx
	dy := ey - my
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return false
	}
	ux, uy := dx/n, dy/n
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: purpleKind, X: mx, Y: my,
		VX: ux * purpleSpeed, VY: uy * purpleSpeed,
		OwnerID: ctx.ID, Slot: h.slot,
	})
	ctx.Out <- unit.FX{
		Name: "void-shot", UnitID: ctx.ID, Kind: purpleKind,
		X: mx, Y: my, VX: ux * purpleSpeed, VY: uy * purpleSpeed, Slot: h.slot,
	}
	h.shotReadyAt = e.Time + purpleCD
	return true
}

func passHurt(ctx unit.Context, ev unit.Event, owner uint64) bool {
	d, ok := ev.(unit.IncomingDamage)
	if !ok {
		return false
	}
	unit.ConfirmHit(ctx, d)
	if owner != 0 && d.Amount > 0 {
		ctx.Out <- unit.Damage{From: d.From, To: owner, Amount: d.Amount}
	}
	return true
}

type 术士弧 struct {
	slot int
	dmg  float64
}

func (a *术士弧) Handle(ctx unit.Context, ev unit.Event) {
	e, ok := ev.(unit.Collision)
	if !ok || !unit.EnemyTarget(e, a.slot) {
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: a.dmg}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

type 紫弹 struct {
	slot       int
	hitReadyAt float64
}

func (b *紫弹) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		if !e.Field.OutlineContains(e.Self.X, e.Self.Y, 0) {
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		}
	case unit.Collision:
		if e.Other.Slot == b.slot {
			if e.Other.Role == unit.RoleFighter {
				ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			}
			return
		}
		if !unit.Hittable(e.Other, b.slot) {
			return
		}
		if e.Time < b.hitReadyAt {
			return
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: purpleDamage}
		ctx.Out <- unit.FX{
			Name: "void-hit", Kind: purpleKind,
			X: e.Other.X, Y: e.Other.Y, Slot: e.Other.Slot,
		}
		b.hitReadyAt = e.Time + twinHitCD
	}
}

func twinField(ctx unit.Context, s unit.Sense, strength float64) {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Slot == s.Self.Slot || o.Role == unit.RoleProjectile || o.Nonsolid {
			continue
		}
		dx := s.Self.X - o.X
		dy := s.Self.Y - o.Y
		n := math.Hypot(dx, dy)
		if n < 1e-6 {
			continue
		}
		ctx.Out <- unit.Force{
			UnitID: o.ID,
			AX:     dx / n * strength,
			AY:     dy / n * strength,
		}
	}
}
