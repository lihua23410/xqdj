package 优昙华院

import (
	"embed"
	"math"
	"math/rand/v2"
	"sync"
	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const KindUdongein = "优昙华院"
const KindMindShot = "优昙华院心弹"
const KindMindShard = "优昙华院碎弹"
const KindAspect = "优昙华院分身"
const KindAspectShot = "优昙华院幻弹"
const KindCrown = "优昙华院花冠"
const KindLaser = "优昙华院激光"
const KindGas = "优昙华院毒烟"
const KindSeekShot = "优昙华院索敌弹"
const KindBlast = "优昙华院爆药"

const (
	udongeinRadius = 18.0
	udongeinSpeed  = 152.0
	udongeinHP     = 100.0
	udongeinVision = 9999.0
	udongeinColor  = "#d5a4d2"

	energyMax  = 5.0
	energyCost = 1.0
	energyWall = 1.8

	cardSlots   = 5
	cardFill    = 0.1
	cardUpgrade = 0.20
	deckSize    = 20

	animRing    = 0.55 // fx-udongein-ring
	animCrown   = 0.45 // fx-ring
	animDose    = 0.70
	mindFire    = 0.50 // 单发心弹的射出窗口
	recoverAtk  = 1.00 // 非想天则特技后摇（出完弹到能再出招）
	recoverCard = 1.10

	frontCostDiv = 10.0 // 按初始伤害扣灵力：每 5 点初始伤害 1 灵力
	frontDR      = 0.75 // 前方减伤后剩 75%
	emptyHurt    = 1.5

	mindDamage   = 8.0
	mindScale    = 920.0 // 起步 mindBase，再加 ½ t³ × 920
	mindBase     = 170.0
	mindRadius   = 12.0
	shardCount   = 8
	shardRadius  = 32.0
	shardSpeed   = 200.0
	shardTau     = 0.2  // v = v0 · τ/(τ+t)；τ 越大衰减越慢、飞得更远
	shardFadeCut = 0.08 // 反比透明度降到这视为 0，弹幕消失
	shardDamage  = 3.0
	shotRadius   = 6.0
	shotColor    = "#e23d4a"

	volleyCount  = 3
	volleyDamage = 4.0
	volleySpeed  = 210.0
	volleySpread = 18.0 * math.Pi / 180
	volleyRadius = 6.0
	volleyGap    = 0.4 // 三发间隔；动画到最后一发出完

	breakRange  = 110.0
	breakDamage = 12.0
	breakKnock  = 300.0

	laserLife   = 0.5
	laserWind   = 0.4 // 前摇，给对方闪开的时间
	laserTick   = 0.2
	laserDamage = 2.0
	laserHalf   = 12.0
	laserLen    = 560.0

	aspectGap    = 0.2
	aspectShots  = 5
	aspectDamage = 3.0
	aspectSpeed  = 185.0
	aspectReach  = 92.0
	aspectMax    = 110.0
	aspectMin    = 40.0

	gasRadius = 54.0
	gasLife   = 10.0
	gasTick   = 0.5
	gasDamage = 1.0
	gasSlow   = 0.4

	crownDamage    = 18.0
	crownSpeed     = 155.0
	crownBaseR     = 10.0
	crownGrow      = 0.22
	crownMaxR      = 60.0
	crownOffscreen = 640.0

	doseAtk     = 0.06
	doseDef     = 0.08
	doseMax     = 4
	blastRadius = 168.0
	blastDamage = 40.0
	blastLife   = 0.55
)

var (
	atkByOwner   sync.Map
	skillByOwner sync.Map
	owners       sync.Map
)

func init() {
	p := unit.NewPack(KindUdongein, assets)
	p.Register(unit.Spec{
		Kind:    KindUdongein,
		Role:    unit.RoleFighter,
		Radius:  udongeinRadius,
		MaxHP:   udongeinHP,
		Speed:   udongeinSpeed,
		Vision:  udongeinVision,
		Fighter: true,
		Look:    unit.Look{Color: udongeinColor, Glow: true, FX: []string{"udongein"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &优昙华院{energy: energyMax, energyCap: energyMax}
	})
	p.Register(unit.Spec{
		Kind:    KindMindShot,
		Role:    unit.RoleProjectile,
		Radius:  mindRadius,
		MaxHP:   1,
		Speed:   185,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: shotColor, Glow: true, Trail: true, FX: []string{"udongein-mind"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &心弹{owner: info.OwnerID}
	})
	p.Register(unit.Spec{
		Kind:    KindMindShard,
		Role:    unit.RoleProjectile,
		Radius:  shardRadius,
		MaxHP:   1,
		Speed:   shardSpeed,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: shotColor, Glow: true, FX: []string{"udongein-shard"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &碎弹{owner: info.OwnerID}
	})
	p.Register(unit.Spec{
		Kind:    KindAspectShot,
		Role:    unit.RoleProjectile,
		Radius:  shotRadius,
		MaxHP:   1,
		Speed:   aspectSpeed,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: shotColor, Trail: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &幻弹{owner: info.OwnerID}
	})
	p.Register(unit.Spec{
		Kind:    KindSeekShot,
		Role:    unit.RoleProjectile,
		Radius:  volleyRadius,
		MaxHP:   1,
		Speed:   volleySpeed,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: shotColor, Glow: true, Trail: true, FX: []string{"udongein-seek"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &索敌弹{owner: info.OwnerID}
	})
	p.Register(unit.Spec{
		Kind:      KindCrown,
		Role:      unit.RoleProjectile,
		Radius:    crownBaseR,
		MaxHP:     1,
		Speed:     crownSpeed,
		Vision:    udongeinVision,
		Fighter:   false,
		PassWalls: true,
		Look:      unit.Look{Color: "#ff6ab8", Trail: true, Glow: true, Overlay: true, FX: []string{"udongein-crown"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &花冠{owner: info.OwnerID, slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:    KindAspect,
		Role:    unit.RoleHelper,
		Radius:  udongeinRadius,
		MaxHP:   1,
		Speed:   0,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: udongeinColor, Glow: true, FX: []string{"udongein-ghost"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return 分身{}
	})
	p.Register(unit.Spec{
		Kind:    KindGas,
		Role:    unit.RoleHelper,
		Radius:  gasRadius,
		MaxHP:   1,
		Speed:   0,
		Vision:  gasRadius + 40,
		Fighter: false,
		Look:    unit.Look{Color: "#8fbf3a", Glow: true, Overlay: true, FX: []string{"udongein-gas"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &瓦斯{owner: info.OwnerID, slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:    KindBlast,
		Role:    unit.RoleHelper,
		Radius:  blastRadius,
		MaxHP:   1,
		Speed:   0,
		Vision:  blastRadius + 40,
		Fighter: false,
		Look:    unit.Look{Color: "#ff6ab8", Glow: true, Overlay: true, FX: []string{"udongein-blast"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &爆药{owner: info.OwnerID, slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:    KindLaser,
		Role:    unit.RoleHelper,
		Radius:  1,
		MaxHP:   1,
		Speed:   0,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: shotColor, Overlay: true, FX: []string{"udongein-beam"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &激光{}
	})
}

type 优昙华院 struct {
	cardMu sync.Mutex

	energy    float64
	energyCap float64
	drained   bool
	lastT     float64
	booted    bool
	animUntil float64
	doses     int
	upgrades  [SkillCount]int
	locked    bool  // 测例锁定技能
	force     uint8 // locked 时的技能；空档表示不放

	laserUntil float64
	laserFrom  float64
	laserOn    bool
	laserHitAt float64
	laserUX    float64
	laserUY    float64
	lockX      float64
	lockY      float64
	holdVX     float64
	holdVY     float64

	aspectLeft int
	aspectNext float64
	aspectHold float64
	aspectUX   float64
	aspectUY   float64
	cloneX     float64
	cloneY     float64

	volleyLeft int
	volleyNext float64
	volleyHold float64
	volleyAng  float64

	hx, hy float64
	x, y   float64
	nearby []unit.Snapshot

	deck     []uint8
	hand     []uint8
	progress float64

	rng  *rand.Rand
	slot int
}

func (a *优昙华院) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		a.onHit(ctx, e)
	case unit.WallHit:
		a.onWall()
	case unit.Sense:
		a.onSense(ctx, e)
	}
}

func (a *优昙华院) onWall() {
	cap := a.energyCap
	if cap < 1 {
		cap = energyMax
	}
	a.energy += energyWall
	if a.energy > cap {
		a.energy = cap
	}
	if a.energy+1e-9 >= cap {
		a.drained = false
		a.energyCap = energyMax
	}
}

func (a *优昙华院) enterBreak() {
	if a.drained {
		return
	}
	a.drained = true
	a.energyCap = energyMax - 1
	if a.energyCap < 1 {
		a.energyCap = 1
	}
}

func (a *优昙华院) dumpEnergy() {
	a.energy = 0
	a.enterBreak()
}

func (a *优昙华院) onHit(ctx unit.Context, d unit.IncomingDamage) {
	if a.laserUntil > d.Time {
		a.stopLaser(ctx)
		a.animUntil = d.Time
	}
	raw := d.Amount
	amt := raw
	if a.energy < 1e-9 {
		a.enterBreak()
		amt *= emptyHurt
	} else if a.frontHit(d.From) {
		cost := raw / frontCostDiv
		if cost <= a.energy+1e-9 {
			a.energy -= cost
			amt = raw * frontDR
			if a.energy < 1e-9 {
				a.energy = 0
				a.enterBreak()
			}
		} else {
			a.dumpEnergy()
			amt *= emptyHurt
		}
	}
	amt *= a.defMul()
	if amt < 0.05 {
		amt = 0.05
	}
	ctx.Out <- unit.ConfirmDamage{Token: d.Token, UnitID: ctx.ID, Amount: amt}
	a.charge()
}

func (a *优昙华院) frontHit(from uint64) bool {
	if from == 0 {
		return false
	}
	hx, hy := a.hx, a.hy
	n := math.Hypot(hx, hy)
	if n < 1e-6 {
		return false
	}
	hx, hy = hx/n, hy/n
	ox, oy := 0.0, 0.0
	found := false
	for i := range a.nearby {
		o := &a.nearby[i]
		if o.ID != from {
			continue
		}
		ox, oy = o.X, o.Y
		found = true
		break
	}
	if !found {
		for i := range a.nearby {
			o := &a.nearby[i]
			if o.Role == unit.RoleFighter && o.Slot != a.slot {
				ox, oy = o.X, o.Y
				found = true
				break
			}
		}
	}
	if !found {
		return false
	}
	dx, dy := ox-a.x, oy-a.y
	if math.Hypot(dx, dy) < 1e-6 {
		return true
	}
	return dx*hx+dy*hy >= 0
}

func (a *优昙华院) onSense(ctx unit.Context, s unit.Sense) {
	a.slot = s.Self.Slot
	a.x, a.y = s.Self.X, s.Self.Y
	if n := math.Hypot(s.Self.VX, s.Self.VY); n > 1e-6 {
		a.hx, a.hy = s.Self.VX, s.Self.VY
	}
	a.nearby = append(a.nearby[:0], s.Nearby...)
	if a.booted {
		a.lastT = s.Time
	} else {
		a.boot(ctx, s.Time)
	}
	owners.Store(ctx.ID, a)
	a.publish(ctx.ID)
	a.emitHUD(ctx, s)

	if a.laserUntil > s.Time {
		a.lockPose(ctx, s)
		a.armLaser(ctx, s)
		a.tickLaser(ctx, s)
		return
	}
	if a.laserUntil > 0 && s.Time >= a.laserUntil {
		a.stopLaser(ctx)
	}
	if a.volleyBusy(s.Time) {
		a.tickVolley(ctx, s)
	}
	if a.aspectBusy(s.Time) {
		a.tickAspect(ctx, s)
	}
	if a.busy(s.Time) {
		return
	}
	enemy := enemyOf(s)
	if a.tryCard(ctx, s, enemy) {
		return
	}
	if enemy == nil {
		return
	}
	a.tryCast(ctx, s, *enemy)
}

func (a *优昙华院) boot(ctx unit.Context, now float64) {
	a.booted = true
	a.lastT = now
	if a.energyCap <= 0 {
		a.energyCap = energyMax
	}
	a.ensureRNG(ctx)
	a.cardMu.Lock()
	if a.deck == nil {
		a.initDeckLocked()
	}
	a.cardMu.Unlock()
}

func (a *优昙华院) busy(now float64) bool {
	return a.volleyBusy(now) || a.aspectBusy(now) || now+1e-9 < a.animUntil
}

func (a *优昙华院) volleyBusy(now float64) bool {
	return a.volleyLeft > 0 || now+1e-9 < a.volleyHold
}

func (a *优昙华院) aspectBusy(now float64) bool {
	return a.aspectLeft > 0 || now+1e-9 < a.aspectHold
}

func (a *优昙华院) tryCast(ctx unit.Context, s unit.Sense, enemy unit.Snapshot) bool {
	a.ensureRNG(ctx)
	for _, sk := range a.castOrder(s) {
		act := Action{Skill: sk}
		if !a.locked && sk == SkillGas {
			act.DBin = uint8(a.rng.IntN(dashDirs * dashRungs))
		}
		if !a.invoke(ctx, s, enemy, act, sk) {
			continue
		}
		a.energy -= skillCost(sk)
		if a.energy < 1e-9 {
			a.energy = 0
			a.enterBreak()
		}
		a.lockAnim(s.Time, sk)
		a.charge()
		return true
	}
	return false
}

func (a *优昙华院) lockAnim(now float64, sk uint8) {
	until := now + skillAnim(sk)
	if until > a.animUntil {
		a.animUntil = until
	}
}

func (a *优昙华院) ensureRNG(ctx unit.Context) {
	if a.rng == nil {
		a.rng = rand.New(rand.NewPCG(rand.Uint64()^ctx.ID, rand.Uint64()))
	}
}

func (a *优昙华院) castOrder(s unit.Sense) []uint8 {
	if a.locked {
		if a.force <= SkillSteer || a.force >= SkillCount {
			return nil
		}
		if !a.ready(s, a.force) {
			return nil
		}
		return []uint8{a.force}
	}
	out := make([]uint8, 0, len(attackSkills))
	for _, sk := range attackSkills {
		if a.ready(s, sk) {
			out = append(out, sk)
		}
	}
	a.rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func skillCost(sk uint8) float64 {
	if sk <= SkillSteer || sk >= SkillCount {
		return 0
	}
	return energyCost
}

func (a *优昙华院) ready(s unit.Sense, sk uint8) bool {
	if sk <= SkillSteer || int(sk) >= SkillCount {
		return false
	}
	if a.busy(s.Time) {
		return false
	}
	return a.energy+1e-9 >= skillCost(sk)
}

func (a *优昙华院) invoke(ctx unit.Context, s unit.Sense, enemy unit.Snapshot, act Action, sk uint8) bool {
	switch sk {
	case SkillVolley:
		return a.castVolley(ctx, s, enemy, act)
	case SkillMind:
		return a.castMind(ctx, s, enemy, act)
	case SkillBreak:
		return a.castBreak(ctx, s, enemy)
	case SkillLaser:
		return a.castLaser(ctx, s, enemy, act)
	case SkillAspect:
		return a.castAspect(ctx, s, enemy, act)
	case SkillGas:
		return a.castGas(ctx, s, enemy, act)
	case SkillCrown:
		return a.castCrown(ctx, s, enemy, act)
	case SkillDose:
		return a.castDose(ctx, s)
	}
	return false
}

func (a *优昙华院) atkMul() float64 {
	return 1 + doseAtk*float64(a.doses)
}

func (a *优昙华院) defMul() float64 {
	m := 1 - doseDef*float64(a.doses)
	if m < 0.2 {
		return 0.2
	}
	return m
}

func (a *优昙华院) skillMul(sk uint8) float64 {
	if int(sk) >= SkillCount {
		return 1
	}
	return 1 + cardUpgrade*float64(a.upgrades[sk])
}

func (a *优昙华院) publish(id uint64) {
	atkByOwner.Store(id, a.atkMul())
	var mul [SkillCount]float64
	for sk := uint8(0); sk < SkillCount; sk++ {
		mul[sk] = a.skillMul(sk)
	}
	skillByOwner.Store(id, mul)
}

func (a *优昙华院) emitHUD(ctx unit.Context, s unit.Sense) {
	ctx.Out <- unit.FX{
		Name:   "energy",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		X:      s.Self.X,
		Y:      s.Self.Y,
		Slot:   s.Self.Slot,
		Amount: a.energy,
		VX:     float64(a.doses),
		VY:     a.energyCap,
	}
	packed, prog, n := a.hudCards()
	ctx.Out <- unit.FX{
		Name:   "cards",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		X:      s.Self.X,
		Y:      s.Self.Y,
		Slot:   s.Self.Slot,
		Amount: prog,
		VX:     packed,
		VY:     float64(n),
	}
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

func ownerAtk(id uint64) float64 {
	v, ok := atkByOwner.Load(id)
	if !ok {
		return 1
	}
	m, _ := v.(float64)
	if m < 1 {
		return 1
	}
	return m
}

func ownerSkill(id uint64, sk uint8) float64 {
	v, ok := skillByOwner.Load(id)
	if !ok {
		return 1
	}
	mul, _ := v.([SkillCount]float64)
	if int(sk) >= SkillCount || mul[sk] < 1 {
		return 1
	}
	return mul[sk]
}

func scaled(owner uint64, base float64) float64 {
	return base * ownerAtk(owner)
}

func scaledSkill(owner uint64, sk uint8, base float64) float64 {
	return scaled(owner, base) * ownerSkill(owner, sk)
}

func spawnShot(ctx unit.Context, kind string, x, y, ux, uy, speed, gap float64, slot int) {
	ctx.Out <- unit.Spawn{
		Kind:    kind,
		X:       x + ux*gap,
		Y:       y + uy*gap,
		VX:      ux * speed,
		VY:      uy * speed,
		OwnerID: ctx.ID,
		Slot:    slot,
	}
}

func shotFX(ctx unit.Context, s unit.Sense, ux, uy float64) {
	ctx.Out <- unit.FX{
		Name: "shot", Kind: ctx.Kind, UnitID: ctx.ID,
		X: s.Self.X, Y: s.Self.Y, VX: ux, VY: uy, Slot: s.Self.Slot,
	}
}

func mindSpeed(t float64) float64 {
	if t < 0 {
		t = 0
	}
	return mindBase + 0.5*t*t*t*mindScale
}

func shardFade(t float64) float64 {
	if t < 0 {
		t = 0
	}
	return shardTau / (shardTau + t)
}

func shardAlpha(t float64) float64 {
	f := shardFade(t)
	if f <= shardFadeCut {
		return 0
	}
	return (f - shardFadeCut) / (1 - shardFadeCut)
}

func shardSpeedAt(t float64) float64 {
	return shardSpeed * shardFade(t)
}

func crownRadius(dist float64) float64 {
	r := crownBaseR + dist*crownGrow
	if r > crownMaxR {
		return crownMaxR
	}
	return r
}

func deal(ctx unit.Context, owner, to uint64, sk uint8, base float64) {
	ctx.Out <- unit.Damage{From: ctx.ID, To: to, Amount: scaledSkill(owner, sk, base)}
	noteDealt(owner)
}

func noteDealt(owner uint64) {
	v, ok := owners.Load(owner)
	if !ok {
		return
	}
	a, _ := v.(*优昙华院)
	if a == nil {
		return
	}
	a.charge()
}
