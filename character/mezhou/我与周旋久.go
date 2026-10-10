// 我与周旋久：主球「我」带一个同色半透明随从「周旋久」。两个球互相碰撞会掉血，
// 每次互撞随机进入「误认 / 失败(精神病) / 无事发生」之一；任一方倒下后按当前
// 状态派生虚空、神圣、自杀等终局形态。胜负仍由战斗机「我」决定（引擎口径）。
package 我与周旋久

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const KindMe = "我与周旋久"
const KindZhou = "周旋久"
const KindVoidShot = "虚空弹"

// 状态。前三个是互撞掷出的循环状态，后三个是死亡派生。
const (
	modeNone = iota // 无事发生
	modeMistake     // 误认
	modePsycho      // 失败（精神病）
	modeVoid        // 虚空（无事发生里周旋久先死）
	modeHoly        // 神圣（误认里我挨到致命伤）
	modeSuicide     // 自杀（误认里周旋久先死）
	modeMonster     // 怪物（精神病里我挨到致命伤）
)

const (
	meRadius = 18.0
	meHP     = 200.0
	meSpeed  = 120.0
	meVision = 9999.0
	meDamage = 6.0
	meColor  = "#a06bf5"

	zhouRadius = 18.0
	zhouHP     = 70.0
	zhouSpeed  = 155.0
	zhouVision = 9999.0
	zhouDamage = 6.0
	zhouAim    = 20

	bumpsPerRoll = 1
	bumpCD       = 0.25

	psychoTick  = 0.5
	psychoDrain = 10.0
	knifeDamage = 7.0
	knifeGap    = 0.12
	knifeReach  = 74.0

	voidGap      = 0.22
	voidDamage   = 2.0
	voidSpeed    = 430.0
	voidShotR    = 7.0
	voidShotHP   = 1.0
	voidShotLife = 6.0

	holySpeedMul = 1.6
	holyAuraR    = 165.0
	holyAuraDmg  = 5.0
	holyAuraGap  = 0.4

	suicideDelay = 5.0
	suicideR     = 170.0
	suicideDmg   = 9999.0

	howlPeriod = 2.0
	howlDamage = 10.0
	howlStun   = 1.2
	howlR      = 250.0
)

func init() {
	p := unit.NewPack(KindMe, assets)
	p.Register(unit.Spec{
		Kind:    KindMe,
		Role:    unit.RoleFighter,
		Radius:  meRadius,
		MaxHP:   meHP,
		Speed:   meSpeed,
		Vision:  meVision,
		Fighter: true,
		Look:    unit.Look{Color: meColor, FX: []string{"mezhou-me"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &我{}
	})
	p.Register(unit.Spec{
		Kind:        KindZhou,
		Role:        unit.RoleMinion,
		Radius:      zhouRadius,
		MaxHP:       zhouHP,
		Speed:       zhouSpeed,
		Vision:      zhouVision,
		Mortal:      true,
		Cruise:      true,
		AimPriority: zhouAim,
		Look:        unit.Look{Color: meColor, FX: []string{"mezhou-zhou"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &周旋久{owner: info.OwnerID, slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:    KindVoidShot,
		Role:    unit.RoleProjectile,
		Radius:  voidShotR,
		MaxHP:   voidShotHP,
		Speed:   voidSpeed,
		Vision:  voidShotLife * voidSpeed,
		Look:    unit.Look{Color: "#c9a6ff", Trail: true, FX: []string{"mezhou-void"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &虚空弹{owner: info.OwnerID, slot: info.Slot}
	})
}

type 我 struct {
	slot  int
	mode  int
	boot  bool
	rng   *rand.Rand
	hp    float64
	x, y  float64
	hx, hy float64

	zhouID     uint64
	zhouHP     float64
	zhouMax    float64
	zhouSeen   bool
	spawnGrace float64

	bumps      int
	lastBumpAt float64

	nextDrainAt float64
	nextKnifeAt float64
	nextVoidAt  float64
	nextAuraAt  float64
	nextHowlAt  float64
	suicideAt   float64
	dead        bool
}

func (t *我) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		t.onIncoming(ctx, e)
	case unit.Collision:
		t.onCollision(ctx, e)
	case unit.Sense:
		t.onSense(ctx, e)
	}
}

func (t *我) onIncoming(ctx unit.Context, d unit.IncomingDamage) {
	lethal := t.hp-d.Amount <= 0
	// 误认：致命伤不致死，转为神圣。
	if lethal && t.mode == modeMistake {
		unit.BlockHit(ctx, d)
		t.enterHoly(ctx)
		return
	}
	// 精神病：致命伤不致死，周旋久接替成为嚎叫怪物。
	if lethal && t.mode == modePsycho {
		unit.BlockHit(ctx, d)
		t.enterMonster(ctx)
		return
	}
	unit.ConfirmHit(ctx, d)
}

func (t *我) onCollision(ctx unit.Context, e unit.Collision) {
	if t.dead {
		return
	}
	if e.Other.Kind == KindZhou {
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: meDamage}
		if t.mode == modeNone && e.Time-t.lastBumpAt >= bumpCD {
			t.lastBumpAt = e.Time
			t.bumps++
			ctx.Out <- unit.FX{Name: "bump", Kind: ctx.Kind, X: e.Other.X, Y: e.Other.Y, Slot: t.slot, Amount: float64(t.bumps)}
			if t.bumps >= bumpsPerRoll {
				t.bumps = 0
				t.roll(ctx, e.Time, e.Other.HP, e.Other.MaxHP)
			}
		}
		return
	}
	if unit.Hittable(e.Other, t.slot) {
		amt := meDamage
		if t.mode == modeHoly {
			amt *= 2
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: amt}
	}
}

func (t *我) onSense(ctx unit.Context, s unit.Sense) {
	t.x, t.y = s.Self.X, s.Self.Y
	t.hp = s.Self.HP
	t.slot = s.Self.Slot
	if sp := math.Hypot(s.Self.VX, s.Self.VY); sp > 1e-6 {
		t.hx, t.hy = s.Self.VX/sp, s.Self.VY/sp
	}
	if !t.boot {
		t.boot = true
		t.spawnGrace = s.Time + 0.6
		if t.rng == nil {
			t.rng = rand.New(rand.NewPCG(ctx.ID*2654435761+1, 0x9e3779b97f4a7c15))
		}
		ctx.Out <- unit.Spawn{
			Kind: KindZhou, X: s.Self.X + 55, Y: s.Self.Y,
			VX: -zhouSpeed * 0.6, VY: zhouSpeed * 0.8,
			OwnerID: ctx.ID, Slot: s.Self.Slot,
		}
		t.emitMode(ctx)
	}
	t.noteZhou(ctx, s)
	switch t.mode {
	case modePsycho:
		t.tickPsycho(ctx, s)
	case modeVoid:
		t.tickVoid(ctx, s)
	case modeHoly:
		t.tickHoly(ctx, s)
	case modeSuicide:
		t.tickSuicide(ctx, s)
	case modeMonster:
		t.tickMonster(ctx, s)
	}
}

func (t *我) noteZhou(ctx unit.Context, s unit.Sense) {
	var z *unit.Snapshot
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID == s.Self.ID && o.Kind == KindZhou {
			z = o
			break
		}
	}
	if z != nil {
		t.zhouSeen = true
		t.zhouID = z.ID
		t.zhouHP = z.HP
		t.zhouMax = z.MaxHP
		return
	}
	if t.zhouSeen && s.Time >= t.spawnGrace {
		t.zhouSeen = false
		t.onZhouDeath(ctx, s)
	}
}

func (t *我) onZhouDeath(ctx unit.Context, s unit.Sense) {
	switch t.mode {
	case modeNone:
		t.enterVoid(ctx)
	case modeMistake:
		t.enterSuicide(ctx, s.Time)
	case modePsycho:
		// 继续追杀敌方（tickPsycho 找不到周旋久会自动改追敌人）
	}
}

func (t *我) roll(ctx unit.Context, now, zhouHP, zhouMax float64) {
	switch t.rng.IntN(3) {
	case 0:
		t.enterMistake(ctx, zhouHP, zhouMax)
	case 1:
		t.enterPsycho(ctx, now)
	default:
		// 无事发生：保持，等下一次碰撞
	}
}

func (t *我) enterMistake(ctx unit.Context, hp, maxHP float64) {
	t.mode = modeMistake
	// 我 = 周旋久：血量与速度都对齐周旋久。
	if maxHP <= 0 {
		maxHP = zhouHP
	}
	if hp <= 0 {
		hp = maxHP
	}
	ctx.Out <- unit.SetHP{UnitID: ctx.ID, HP: hp, MaxHP: maxHP}
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: zhouSpeed}
	t.emitMode(ctx)
}

func (t *我) enterPsycho(ctx unit.Context, now float64) {
	t.mode = modePsycho
	t.nextDrainAt = now + psychoTick
	t.nextKnifeAt = now
	t.emitMode(ctx)
}

func (t *我) enterVoid(ctx unit.Context) {
	t.mode = modeVoid
	t.nextVoidAt = 0
	t.emitMode(ctx)
}

func (t *我) enterHoly(ctx unit.Context) {
	t.mode = modeHoly
	t.inheritZhouHP(ctx)
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: meSpeed * holySpeedMul}
	ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: KindZhou}
	t.emitMode(ctx)
}

func (t *我) enterSuicide(ctx unit.Context, now float64) {
	t.mode = modeSuicide
	t.suicideAt = now + suicideDelay
	t.emitMode(ctx)
}

// enterMonster：精神病里我挨到致命伤，周旋久接替成为嚎叫怪物并退场。
func (t *我) enterMonster(ctx unit.Context) {
	t.mode = modeMonster
	t.nextHowlAt = 0
	t.inheritZhouHP(ctx)
	ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: KindZhou}
	t.emitMode(ctx)
}

// inheritZhouHP：转形态时继承周旋久的血量（神圣/怪物都是「周旋久接替」）。
func (t *我) inheritZhouHP(ctx unit.Context) {
	// 血量按周旋久的上限转换，并回满。
	maxHP := t.zhouMax
	if maxHP <= 0 {
		maxHP = zhouHP
	}
	ctx.Out <- unit.SetHP{UnitID: ctx.ID, HP: maxHP, MaxHP: maxHP}
}

func (t *我) tickPsycho(ctx unit.Context, s unit.Sense) {
	if s.Time+1e-9 >= t.nextDrainAt {
		ctx.Out <- unit.Damage{From: ctx.ID, To: ctx.ID, Amount: psychoDrain, NoFreeze: true}
		t.nextDrainAt += psychoTick
	}
	target, ok := t.pickTarget(s)
	if !ok {
		return
	}
	dx, dy := target.X-t.x, target.Y-t.y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	ux, uy := dx/n, dy/n
	sp := zhouSpeed * 2.2
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: ux * sp, VY: uy * sp}
	if n <= knifeReach+target.Radius && s.Time+1e-9 >= t.nextKnifeAt {
		ctx.Out <- unit.Damage{From: ctx.ID, To: target.ID, Amount: knifeDamage}
		ctx.Out <- unit.FX{
			Name: "knife", Kind: ctx.Kind, UnitID: ctx.ID, Slot: t.slot,
			X: target.X, Y: target.Y, VX: ux, VY: uy, Amount: float64(s.Time),
		}
		t.nextKnifeAt = s.Time + knifeGap
	}
}

// pickTarget：精神病先杀周旋久，周旋久死了再杀敌方；其余形态追敌方。
func (t *我) pickTarget(s unit.Sense) (unit.Snapshot, bool) {
	if t.mode == modePsycho {
		for i := range s.Nearby {
			o := s.Nearby[i]
			if o.OwnerID == s.Self.ID && o.Kind == KindZhou {
				return o, true
			}
		}
	}
	var best unit.Snapshot
	bestD := math.MaxFloat64
	found := false
	for i := range s.Nearby {
		o := s.Nearby[i]
		if !unit.Hittable(o, s.Self.Slot) {
			continue
		}
		if d := math.Hypot(o.X-t.x, o.Y-t.y); d < bestD {
			bestD, best, found = d, o, true
		}
	}
	return best, found
}

func (t *我) tickVoid(ctx unit.Context, s unit.Sense) {
	if s.Time+1e-9 < t.nextVoidAt {
		return
	}
	t.nextVoidAt = s.Time + voidGap
	ang := t.rng.Float64() * 2 * math.Pi
	ux, uy := math.Cos(ang), math.Sin(ang)
	ctx.Out <- unit.Spawn{
		Kind: KindVoidShot, X: t.x + ux*meRadius, Y: t.y + uy*meRadius,
		VX: ux * voidSpeed, VY: uy * voidSpeed, OwnerID: ctx.ID, Slot: t.slot,
	}
	ctx.Out <- unit.FX{Name: "void", Kind: ctx.Kind, UnitID: ctx.ID, Slot: t.slot, X: t.x, Y: t.y, VX: ux, VY: uy}
}

func (t *我) tickHoly(ctx unit.Context, s unit.Sense) {
	if s.Time+1e-9 < t.nextAuraAt {
		return
	}
	t.nextAuraAt = s.Time + holyAuraGap
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		if math.Hypot(o.X-t.x, o.Y-t.y) <= holyAuraR+o.Radius {
			ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: holyAuraDmg, NoFreeze: true}
		}
	}
	ctx.Out <- unit.FX{Name: "holy", Kind: ctx.Kind, UnitID: ctx.ID, Slot: t.slot, X: t.x, Y: t.y}
}

func (t *我) tickSuicide(ctx unit.Context, s unit.Sense) {
	if s.Time+1e-9 < t.suicideAt {
		// 闪烁提醒交给前端 mode-suicide 的 CSS，避免每帧刷特效。
		return
	}
	t.dead = true
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		if math.Hypot(o.X-t.x, o.Y-t.y) <= suicideR+o.Radius {
			ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: suicideDmg}
		}
	}
	ctx.Out <- unit.FX{Name: "suicide", Kind: ctx.Kind, UnitID: ctx.ID, Slot: t.slot, X: t.x, Y: t.y, Amount: suicideR}
	ctx.Out <- unit.Damage{From: ctx.ID, To: ctx.ID, Amount: suicideDmg}
}

func (t *我) tickMonster(ctx unit.Context, s unit.Sense) {
	// 追着敌人走，嚎叫才够得着。
	if target, ok := t.pickTarget(s); ok {
		dx, dy := target.X-t.x, target.Y-t.y
		if n := math.Hypot(dx, dy); n > 1e-6 {
			ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: dx / n * meSpeed, VY: dy / n * meSpeed}
		}
	}
	if t.nextHowlAt == 0 {
		t.nextHowlAt = s.Time + howlPeriod
		return
	}
	if s.Time+1e-9 < t.nextHowlAt {
		return
	}
	t.nextHowlAt = s.Time + howlPeriod
	ctx.Out <- unit.FX{Name: "howl", Kind: ctx.Kind, UnitID: ctx.ID, Slot: t.slot, X: t.x, Y: t.y, Amount: howlR}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		if math.Hypot(o.X-t.x, o.Y-t.y) <= howlR+o.Radius {
			ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: howlDamage}
			ctx.Out <- unit.Stun{UnitID: o.ID, Hold: true, Until: s.Time + howlStun}
		}
	}
}

func (t *我) emitMode(ctx unit.Context) {
	ctx.Out <- unit.FX{Name: "mode", Kind: ctx.Kind, UnitID: ctx.ID, Slot: t.slot, X: t.x, Y: t.y, Amount: float64(t.mode)}
}

type 周旋久 struct {
	owner uint64
	slot  int
}

func (z *周旋久) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	e, ok := ev.(unit.Collision)
	if !ok {
		return
	}
	if e.Other.Kind == KindMe {
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: zhouDamage}
		return
	}
	if unit.Hittable(e.Other, z.slot) {
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: zhouDamage}
	}
}

type 虚空弹 struct {
	owner uint64
	slot  int
	born  float64
}

func (b *虚空弹) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		if b.born == 0 {
			b.born = e.Time
		}
		if e.Time-b.born >= voidShotLife {
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			return
		}
		var best *unit.Snapshot
		bestD := math.MaxFloat64
		for i := range e.Nearby {
			o := &e.Nearby[i]
			if !unit.Hittable(*o, b.slot) {
				continue
			}
			if d := math.Hypot(o.X-e.Self.X, o.Y-e.Self.Y); d < bestD {
				bestD, best = d, o
			}
		}
		if best != nil && bestD > 1e-6 {
			dx, dy := best.X-e.Self.X, best.Y-e.Self.Y
			n := math.Hypot(dx, dy)
			ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: dx / n * voidSpeed, VY: dy / n * voidSpeed}
		}
	case unit.Collision:
		if unit.Hittable(e.Other, b.slot) {
			ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: voidDamage, NoFreeze: true}
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		}
	case unit.WallHit:
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}
