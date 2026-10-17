// 谭泳辰老先生：开局空手随机游荡，场上散落铃铛与苦无。自己或同边战斗机碰到都算捡到，效果记在他身上。
// 捡到苦无后每 2 秒丢一发直线苦无（3 伤，约 10% 暴击 10 伤）；有铃铛时累计
// 3 次命中或任意暴击就净化（无敌 1.5s + 回已损失血量 20%）；累计 10 次命中或
// 3 次暴击开 5 秒灵狐场域，场内苦无间隔 0.7s 但不再累计。
package 谭泳辰

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const KindTan = "谭泳辰老先生"
const KindBell = "铃铛"
const KindKunai = "苦无"
const KindDomain = "灵狐场域"
const KindMilk = "奶符"

const (
	tanRadius = 18.0
	tanHP     = 90.0
	tanSpeed  = 165.0
	tanVision = 9999.0
	tanColor  = "#4fd8e8" // 青色

	bellRadius  = 14.0
	kunaiRadius = 14.0

	kunaiInterval       = 2.0
	kunaiIntervalDomain = 0.55
	kunaiRange          = 620.0
	kunaiDamage         = 3.0
	kunaiCritDamage     = 12.0
	kunaiCritChance     = 0.10
	kunaiFlight         = 0.55  // 苦无飞行时间（秒），伤害在飞到落点时结算
	kunaiOvershoot      = 320.0 // 苦无飞过目标后继续飞的世界距离

	bellHitsForPurify = 3
	purifyInvuln      = 1.5
	purifyHealFrac    = 0.20
	purifyTeamR       = 240.0 // 净化顺手回血的队友范围

	domainHits     = 10
	domainCrits    = 3
	domainDuration = 5.0
	domainRadius   = 220.0

	milkRadius = 14.0
	milkHeal   = 5.0
	milkEvery  = 5 // 每这么多次苦无攻击掉一个奶符

	pickupSlack = 10.0
)

func init() {
	p := unit.NewPack(KindTan, assets)
	p.Register(unit.Spec{
		Kind:    KindTan,
		Role:    unit.RoleFighter,
		Radius:  tanRadius,
		MaxHP:   tanHP,
		Speed:   tanSpeed,
		Vision:  tanVision,
		Fighter: true,
		Look:    unit.Look{Color: tanColor, FX: []string{"tanyongchen"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &谭泳辰老先生{}
	})
	p.Register(unit.Spec{
		Kind:     KindBell,
		Role:     unit.RoleHelper,
		Radius:   bellRadius,
		MaxHP:    1,
		Speed:    0,
		Vision:   0,
		Nonsolid: true,
		Look:     unit.Look{Color: "#f0c85a", FX: []string{"tan-bell"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &静物{owner: info.OwnerID, slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:     KindKunai,
		Role:     unit.RoleHelper,
		Radius:   kunaiRadius,
		MaxHP:    1,
		Speed:    0,
		Vision:   0,
		Nonsolid: true,
		Look:     unit.Look{Color: "#dce8f5", FX: []string{"tan-kunai"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &静物{owner: info.OwnerID, slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:     KindDomain,
		Role:     unit.RoleHelper,
		Radius:   domainRadius,
		MaxHP:    1,
		Speed:    0,
		Vision:   0,
		Nonsolid: true,
		Look:     unit.Look{Color: "#4fd8e8", FX: []string{"tan-domain"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &灵狐场域{owner: info.OwnerID, slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:     KindMilk,
		Role:     unit.RoleHelper,
		Radius:   milkRadius,
		MaxHP:    1,
		Speed:    0,
		Vision:   80,
		Nonsolid: true,
		Look:     unit.Look{Color: "#eff6ff", FX: []string{"tan-milk"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &奶符{owner: info.OwnerID, slot: info.Slot}
	})
}

type 谭泳辰老先生 struct {
	slot int
	boot bool
	rng  *rand.Rand

	hasBell  bool
	hasKunai bool

	nextAttackAt float64
	bellHits     int
	hits         int
	crits        int
	attacks      int // 苦无总出手次数，用来每 milkEvery 次掉一个奶符

	invulnUntil float64
	hp          float64
	maxHP       float64
	x, y        float64
	pending     []pendingKunai
}

// pendingKunai：一发已出手、正在直线飞行的苦无。逐帧扫掠，飞到谁就结算谁。
type pendingKunai struct {
	start    float64 // 出手时刻
	ox, oy   float64 // 出手位置
	ux, uy   float64 // 飞行方向（单位向量）
	total    float64 // 总飞行距离
	speed    float64 // 飞行速度（世界单位/秒）
	traveled float64 // 已飞距离
	crit     bool
}

func (t *谭泳辰老先生) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		if e.Time < t.invulnUntil {
			unit.BlockHit(ctx, e)
			return
		}
		unit.ConfirmHit(ctx, e)
	case unit.Sense:
		t.onSense(ctx, e)
	}
}

func (t *谭泳辰老先生) onSense(ctx unit.Context, s unit.Sense) {
	t.x, t.y = s.Self.X, s.Self.Y
	t.hp = s.Self.HP
	t.maxHP = s.Self.MaxHP
	t.slot = s.Self.Slot

	if !t.boot {
		t.boot = true
		if t.rng == nil {
			t.rng = unit.NewRNG(ctx.ID)
		}
		t.spawnItems(ctx, s)
		t.nextAttackAt = s.Time
	}

	t.tryPickup(ctx, s)

	inDomain := t.inDomain(s)
	if t.hasKunai && s.Time+1e-9 >= t.nextAttackAt {
		interval := kunaiInterval
		if inDomain {
			interval = kunaiIntervalDomain
		}
		if t.attack(ctx, s, inDomain) {
			t.nextAttackAt = s.Time + interval
		}
	}
	t.resolvePending(ctx, s)
}

func (t *谭泳辰老先生) spawnItems(ctx unit.Context, s unit.Sense) {
	f := unit.LiveField()
	for _, kind := range []string{KindBell, KindKunai} {
		x, y, ok := f.RandomWalkable(t.rng, 20)
		if !ok {
			x, y = f.Clamp(s.Self.X, s.Self.Y, 40)
		}
		ctx.Out <- unit.Spawn{Kind: kind, X: x, Y: y, OwnerID: ctx.ID, Slot: s.Self.Slot}
	}
}

// spawnMilk：在场上随机一个可行走点放一个奶符。
func (t *谭泳辰老先生) spawnMilk(ctx unit.Context) {
	f := unit.LiveField()
	x, y, ok := f.RandomWalkable(t.rng, 20)
	if !ok {
		x, y = f.Clamp(t.x, t.y, 40)
	}
	ctx.Out <- unit.Spawn{Kind: KindMilk, X: x, Y: y, OwnerID: ctx.ID, Slot: t.slot}
}

func (t *谭泳辰老先生) tryPickup(ctx unit.Context, s unit.Sense) {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID != s.Self.ID {
			continue
		}
		if o.Kind != KindBell && o.Kind != KindKunai {
			continue
		}
		// 只有谭泳辰本人能捡，队友和敌人都不能。
		if math.Hypot(o.X-t.x, o.Y-t.y) > tanRadius+o.Radius+pickupSlack {
			continue
		}
		switch o.Kind {
		case KindBell:
			t.hasBell = true
		case KindKunai:
			t.hasKunai = true
		}
		pickupFlag := 1.0
		if o.Kind == KindKunai {
			pickupFlag = 2
		}
		ctx.Out <- unit.Despawn{UnitID: o.ID}
		ctx.Out <- unit.FX{Name: "pickup", Kind: ctx.Kind, UnitID: ctx.ID, Slot: t.slot, X: o.X, Y: o.Y, Amount: pickupFlag}
	}
}

// inDomain：自己是否站在己方灵狐场域里。
func (t *谭泳辰老先生) inDomain(s unit.Sense) bool {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		// 队友（同 Slot）的灵狐场域也算，团队共享攻速。
		if o.Kind == KindDomain && o.Slot == s.Self.Slot {
			if math.Hypot(o.X-t.x, o.Y-t.y) <= o.Radius {
				return true
			}
		}
	}
	return false
}

// attack：朝最近敌人丢一发直线苦无（命中即结算），返回是否真的出手。
func (t *谭泳辰老先生) attack(ctx unit.Context, s unit.Sense, inDomain bool) bool {
	target := unit.Seek(s)
	if target == nil {
		return false
	}
	dx, dy := target.X-t.x, target.Y-t.y
	n := math.Hypot(dx, dy)
	if n > kunaiRange+target.Radius || n < 1e-6 {
		return false
	}
	ux, uy := dx/n, dy/n
	crit := t.rng.Float64() < kunaiCritChance
	// 只预约这一发；伤害等飞到落点那一拍才结算。
	// 苦无飞过目标后继续飞一段；逐帧扫掠，飞到谁就在那一刻结算谁。
	total := n + kunaiOvershoot
	t.pending = append(t.pending, pendingKunai{
		start: s.Time, ox: t.x, oy: t.y, ux: ux, uy: uy,
		total: total, speed: total / kunaiFlight, crit: crit,
	})
	ctx.Out <- unit.FX{Name: "kunai", Kind: ctx.Kind, UnitID: ctx.ID, Slot: t.slot, X: t.x, Y: t.y, VX: ux, VY: uy, Amount: total}

	// 每 milkEvery 次出手掉一个奶符（同队可吃，回 5 血）。
	t.attacks++
	if t.attacks >= milkEvery {
		t.attacks = 0
		t.spawnMilk(ctx)
	}
	return true
}

// resolvePending：苦无飞到落点那一拍结算伤害与累计；目标不在落点就是空。
func (t *谭泳辰老先生) resolvePending(ctx unit.Context, s unit.Sense) {
	if len(t.pending) == 0 {
		return
	}
	keep := t.pending[:0]
	for _, p := range t.pending {
		traveled := p.speed * (s.Time - p.start)
		if traveled >= p.total {
			continue // 飞完，没碰到
		}
		from := p.traveled
		ax := p.ox + p.ux*from
		ay := p.oy + p.uy*from
		bx := p.ox + p.ux*traveled
		by := p.oy + p.uy*traveled

		hit := -1
		bestT := math.MaxFloat64
		for i := range s.Nearby {
			o := &s.Nearby[i]
			if !unit.Hittable(*o, s.Self.Slot) {
				continue
			}
			d, tt := segClosest(ax, ay, bx, by, o.X, o.Y)
			if d > o.Radius+kunaiRadius+6 {
				continue
			}
			if tt < bestT {
				bestT, hit = tt, i
			}
		}
		if hit < 0 {
			// 没碰到人：看这一拍有没有撞墙，撞到就停在墙上。
			blocked := !s.Field.OutlineContains(bx, by, kunaiRadius) // 飞出场地边界也算撞墙
			if !blocked {
				for _, w := range s.Walls {
					if segSegDist(ax, ay, bx, by, w.X1, w.Y1, w.X2, w.Y2) <= w.Radius+kunaiRadius {
						blocked = true
						break
					}
				}
			}
			if blocked {
				ctx.Out <- unit.FX{Name: "kunai-hit", Kind: ctx.Kind, UnitID: ctx.ID, Slot: t.slot, X: bx, Y: by, Amount: -1}
				continue // 被墙挡下，苦无消失
			}
			p.traveled = traveled
			keep = append(keep, p)
			continue
		}
		o := &s.Nearby[hit]
		amt := kunaiDamage
		critFlag := 0.0
		if p.crit {
			amt = kunaiCritDamage
			critFlag = 1
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: amt}
		ctx.Out <- unit.FX{Name: "kunai-hit", Kind: ctx.Kind, UnitID: ctx.ID, Slot: t.slot, X: o.X, Y: o.Y, Amount: critFlag}
		t.accumulate(ctx, s, p.crit)
	}
	t.pending = keep
}

// segClosest：点 (px,py) 到线段 (ax,ay)-(bx,by) 的距离，以及在线段上的参数 t。
func segClosest(ax, ay, bx, by, px, py float64) (dist, t float64) {
	dx, dy := bx-ax, by-ay
	l2 := dx*dx + dy*dy
	if l2 < 1e-9 {
		return math.Hypot(px-ax, py-ay), 0
	}
	tt := ((px-ax)*dx + (py-ay)*dy) / l2
	if tt < 0 {
		tt = 0
	} else if tt > 1 {
		tt = 1
	}
	cx, cy := ax+dx*tt, ay+dy*tt
	return math.Hypot(px-cx, py-cy), tt
}

// segSegDist：两条线段 (p1,q1) 与 (p2,q2) 的最短距离。
func segSegDist(p1x, p1y, q1x, q1y, p2x, p2y, q2x, q2y float64) float64 {
	d1x, d1y := q1x-p1x, q1y-p1y
	d2x, d2y := q2x-p2x, q2y-p2y
	rx, ry := p1x-p2x, p1y-p2y
	a := d1x*d1x + d1y*d1y
	e := d2x*d2x + d2y*d2y
	f := d2x*rx + d2y*ry
	const eps = 1e-9
	var s, t float64
	switch {
	case a <= eps && e <= eps:
		return math.Hypot(rx, ry)
	case a <= eps:
		s = 0
		t = clamp01(f / e)
	default:
		c := d1x*rx + d1y*ry
		if e <= eps {
			t = 0
			s = clamp01(-c / a)
		} else {
			b := d1x*d2x + d1y*d2y
			denom := a*e - b*b
			if denom > eps {
				s = clamp01((b*f - c*e) / denom)
			} else {
				s = 0
			}
			t = (b*s + f) / e
			if t < 0 {
				t = 0
				s = clamp01(-c / a)
			} else if t > 1 {
				t = 1
				s = clamp01((b - c) / a)
			}
		}
	}
	c1x, c1y := p1x+d1x*s, p1y+d1y*s
	c2x, c2y := p2x+d2x*t, p2y+d2y*t
	return math.Hypot(c1x-c2x, c1y-c2y)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// accumulate：一次命中的累计（铃铛净化 / 灵狐场域）。
func (t *谭泳辰老先生) accumulate(ctx unit.Context, s unit.Sense, crit bool) {
	t.hits++
	if crit {
		t.crits++
	}
	if t.hasBell {
		t.bellHits++
		if t.bellHits >= bellHitsForPurify || crit {
			t.purify(ctx, s)
			t.bellHits = 0
		}
	}
	if t.hits >= domainHits || t.crits >= domainCrits {
		t.hits = 0
		t.crits = 0
		ctx.Out <- unit.Spawn{Kind: KindDomain, X: t.x, Y: t.y, OwnerID: ctx.ID, Slot: t.slot}
		ctx.Out <- unit.FX{Name: "domain", Kind: ctx.Kind, UnitID: ctx.ID, Slot: t.slot, X: t.x, Y: t.y, Amount: domainRadius}
	}
}

// purify：白色净化，无敌 1.5 秒并回复已损失生命的 20%。
func (t *谭泳辰老先生) purify(ctx unit.Context, s unit.Sense) {
	t.invulnUntil = s.Time + purifyInvuln
	missing := t.maxHP - t.hp
	if missing < 0 {
		missing = 0
	}
	heal := missing * purifyHealFrac
	if heal > 0 {
		ctx.Out <- unit.Heal{UnitID: ctx.ID, Amount: heal}
	}
	// 净化也顺手照顾身边队友（同 Slot），各回自己已损血量的同样比例。
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.ID == s.Self.ID || o.Slot != s.Self.Slot {
			continue
		}
		if o.Role != unit.RoleFighter && !o.Mortal {
			continue
		}
		if math.Hypot(o.X-t.x, o.Y-t.y) > purifyTeamR {
			continue
		}
		om := o.MaxHP - o.HP
		if om <= 0 {
			continue
		}
		ctx.Out <- unit.Heal{UnitID: o.ID, Amount: om * purifyHealFrac}
		ctx.Out <- unit.FX{Name: "purify", Kind: ctx.Kind, Slot: o.Slot, X: o.X, Y: o.Y, UnitID: o.ID}
	}
	ctx.Out <- unit.FX{Name: "purify", Kind: ctx.Kind, UnitID: ctx.ID, Slot: t.slot, X: t.x, Y: t.y, Amount: purifyInvuln}
}

// 静物：场上的铃铛 / 苦无，不实心、不吃伤害；主人或同边战斗机碰到由主人收走。
type 静物 struct {
	owner uint64
	slot  int
}

func (o *静物) Handle(unit.Context, unit.Event) {}

// 灵狐场域：存在 domainDuration 秒后自己消失。
type 灵狐场域 struct {
	owner uint64
	slot  int
	born  float64
}

func (d *灵狐场域) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if d.born == 0 {
		d.born = s.Time
	}
	if s.Time+1e-9 >= d.born+domainDuration {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

// 奶符：任何人（战斗机或活随从）吃到都回 5 血，然后消失。
type 奶符 struct {
	owner uint64
	slot  int
}

func (m *奶符) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Nonsolid {
			continue
		}
		if o.Slot != m.slot {
			continue // 只回己方队伍（含自己）
		}
		if o.Role != unit.RoleFighter && !o.Mortal {
			continue
		}
		if math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) <= s.Self.Radius+o.Radius+2 {
			ctx.Out <- unit.Heal{UnitID: o.ID, Amount: milkHeal}
			ctx.Out <- unit.FX{Name: "milk", Kind: ctx.Kind, Slot: m.slot, X: o.X, Y: o.Y, UnitID: o.ID}
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			return
		}
	}
}
