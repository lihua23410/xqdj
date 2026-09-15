package 优昙华院

import (
	"embed"
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

const (
	udongeinRadius = 18.0
	udongeinSpeed  = 152.0
	udongeinHP     = 100.0
	udongeinVision = 9999.0
	udongeinColor  = "#d5a4d2"

	energyMax   = 5.0
	energyRegen = 0.35 // 每秒恢复 0.35 能量
	energyCost  = 1.0
	publicCD    = 1.5
	skillCD     = 2.0

	mindDamage   = 8.0
	mindScale    = 920.0 // 起步 mindBase，再加 ½ t³ × 920
	mindBase     = 170.0
	mindRadius   = 12.0
	shardCount   = 8
	shardRadius  = 32.0
	shardSpeed   = 200.0
	shardTau     = 0.4  // v = v0 · τ/(τ+t)；τ 越大衰减越慢、飞得更远
	shardFadeCut = 0.08 // 反比透明度降到这视为 0，弹幕消失
	shardDamage  = 3.0
	shotRadius   = 6.0
	shotColor    = "#e23d4a"

	breakRange  = 110.0
	breakDamage = 12.0
	breakKnock  = 300.0

	laserLife   = 1.0
	laserTick   = 0.2
	laserDamage = 2.0
	laserHalf   = 12.0
	laserLen    = 560.0

	aspectGap    = 0.2
	aspectShots  = 6
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

	crownDamage = 10.0
	crownSpeed  = 155.0
	crownBaseR  = 10.0
	crownGrow   = 0.22
	crownMaxR   = 60.0

	doseAtk     = 0.06
	doseDef     = 0.08
	doseMax     = 4
	doseSuicide = 9999.0
)

var atkByOwner sync.Map

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
		return &优昙华院{energy: energyMax, gcdUntil: publicCD}
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
		Kind:    KindCrown,
		Role:    unit.RoleProjectile,
		Radius:  crownBaseR,
		MaxHP:   1,
		Speed:   crownSpeed,
		Vision:  udongeinVision,
		Fighter: false,
		Look:    unit.Look{Color: "#ff6ab8", Trail: true, Glow: true, FX: []string{"udongein-crown"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &花冠{owner: info.OwnerID}
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
	energy   float64
	lastT    float64
	booted   bool
	gcdUntil float64
	skillCD  [SkillCount]float64
	doses    int
	locked   bool  // 测例锁定技能
	force    uint8 // locked 时的技能；空档表示不放

	laserUntil float64
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

	rng *rand.Rand

	slot int
}

func (a *优昙华院) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		a.onHit(ctx, e)
	case unit.Sense:
		a.onSense(ctx, e)
	}
}

func (a *优昙华院) onHit(ctx unit.Context, d unit.IncomingDamage) {
	if a.laserUntil > d.Time {
		a.stopLaser(ctx)
	}
	amt := d.Amount * a.defMul()
	if amt < 0.05 {
		amt = 0.05
	}
	ctx.Out <- unit.ConfirmDamage{Token: d.Token, UnitID: ctx.ID, Amount: amt}
}

func (a *优昙华院) onSense(ctx unit.Context, s unit.Sense) {
	a.slot = s.Self.Slot
	dt := 0.0
	if a.booted {
		dt = s.Time - a.lastT
	}
	a.lastT = s.Time
	if !a.booted {
		a.booted = true
		if a.energy <= 0 {
			a.energy = energyMax
		}
	}
	if dt > 0 {
		a.energy += dt * energyRegen
		if a.energy > energyMax {
			a.energy = energyMax
		}
	}
	atkByOwner.Store(ctx.ID, a.atkMul())
	a.emitHUD(ctx, s)

	if a.laserUntil > s.Time {
		a.lockPose(ctx, s)
		a.tickLaser(ctx, s)
		return
	}
	if a.laserUntil > 0 && s.Time >= a.laserUntil {
		a.stopLaser(ctx)
	}
	if a.aspectBusy(s.Time) {
		a.tickAspect(ctx, s)
	}

	enemy := enemyOf(s)
	if enemy == nil {
		return
	}
	if !a.aspectBusy(s.Time) {
		a.tryCast(ctx, s, *enemy)
	}
}

func (a *优昙华院) aspectBusy(now float64) bool {
	return a.aspectLeft > 0 || now+1e-9 < a.aspectHold
}

// tryCast 在能量和冷却都够的技能里纯随机。测例用 force 锁定；force 为空档则不放。

func (a *优昙华院) tryCast(ctx unit.Context, s unit.Sense, enemy unit.Snapshot) bool {
	if s.Time+1e-9 < a.gcdUntil {
		return false
	}
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
		a.gcdUntil = s.Time + publicCD
		a.skillCD[sk] = s.Time + skillCD
		return true
	}
	return false
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
	out := make([]uint8, 0, SkillCount-1)
	for sk := uint8(SkillMind); sk < SkillCount; sk++ {
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
	if s.Time+1e-9 < a.skillCD[sk] {
		return false
	}
	return a.energy+1e-9 >= skillCost(sk)
}

func (a *优昙华院) invoke(ctx unit.Context, s unit.Sense, enemy unit.Snapshot, act Action, sk uint8) bool {
	switch sk {
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

func scaled(owner uint64, base float64) float64 {
	return base * ownerAtk(owner)
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
