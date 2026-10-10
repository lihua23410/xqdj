// 盾斧（怪猎的充能斧）：盾形态正面减伤吃招 → 剑形态二连斩攒能量 → 金瓶转红盾/红剑/红斧 → 斧形态解放斩。
// 数值一律走下面那个 const 块，单位约定：距离 px、速度 px/s、时间 s、角度 度（unit.Deg 转弧度）、伤害 HP 点。
// 完整数值表见 README.md 的「盾斧」一行；改这里记得两边对齐。
package 盾斧

import (
	"embed"
	"math"
	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const KindAxe = "盾斧"

const (
	// —— 本体 ——
	axeRadius = 18.0   // 和所有小球一样的半径
	axeSpeed  = 165.0  // 盾/剑形态的巡航速度
	axeSlow   = 90.0   // 红斧形态的巡航速度（切斧时 165 → 90）
	axeHP     = 100.0  // 标准血量
	axeVision = 9999.0 // 全场视野
	axeColor  = "#9a9a9a"

	// —— 索敌/判定范围 ——
	// seekR 是剑形态「摸到人就起手」的距离：74 是原始量值，×18/54 是按半径比缩到本球（r=18），≈24.7。
	seekR      = 36
	seekSpan   = 120.0 // 索敌扇面：正面 ±60°
	axeSeekR   = 124.0 // 红斧态索敌半径（大解、追解都按这个范围打）
	chouR      = 124.0 // 超解扇半径
	chouSpan   = 120.0 // 超解扇面：正面 ±30°。比索敌窄，所以要先突刺把球顶进落点
	shieldSpan = 300.0 // 盾减伤生效的角度：正面 ±150°，只剩背后 60° 是空档

	// —— 各招时长（秒）——
	slashBeat  = 0.25  // 二连斩动画每一段（前端 gsap 用），3×0.25 正好是 slashLife
	slashGap   = 0.25  // 二连斩两刀之间
	slashLife  = 0.75  // 二连斩总时长：0s 第一刀、0.25s 第二刀，剩下 0.5s 收招
	thrustSp   = 350.0 // 突刺速度
	thrustLife = 0.8   // 突刺最长 1.5s ≈ 顶出去 525px（场地半径 280，所以基本会撞边或撞人）
	fillStand  = 0.8   // 连招结束后站定灌瓶
	bottleWait = 0.2   // 灌瓶到黄瓶转化的等待
	axeWind    = 0.2   // 切斧起手
	axeStand   = 0.8   // 切斧站定
	spinT      = 0.5   // 大解/追解/超解旋每一拍的时长
	tsuiGap    = 0.2   // 追解两刀之间
	flipDown   = 0.3   // 超解翻转：把斧头压下去
	flipUp     = 0.25  // 超解翻转：翻回来（合计 1.05s，就是 stepChouFlip 的时长）
	waveWait   = 0.3   // 超解旋 → 第一波砸地
	waveGap    = 0.15  // 三波砸地之间
	// 超解前那一下突刺：距离短，只为把球顶进超解的落点里。
	chouThrustLife = 0.5 // 0.5s ≈ 顶 175px；碰到人当场伤 3 并提前接超解旋

	// —— 各招伤害（HP 点）——
	// 剑/突刺这类走 strikeDmg()，红剑态会再加 redBonus；下面这几个是固定值。
	slashDmg    = 5.0  // 二连斩每一刀
	thrustDmg   = 3.0  // 突刺（普通突刺和超解前那一下都是这个）
	redBonus    = 3.0  // 红剑加成：二连斩 5→7、突刺 3→5
	daiDmg      = 18.0 // 大解
	tsuiDmg     = 15.0 // 追解每一刀（两刀）
	chouSpinDmg = 15.0 // 超解旋
	chouFanDmg  = 30.0 // 超解扇（104 半径、正面 ±30° 那一发）
	waveDmg     = 15.0 // 超解砸地每一波

	// 超解砸地：三处**互不重叠**的圆（圆心距离 ≥ 2×slamR），范围比原来的分环更远。
	slamR = 26.0 // 每一处砸地圆的半径，同时就是出伤判定半径（前端照着画同一个圆）

	// 瓶与能量：energy 上限 2（每打中一段 +1），黄瓶才够换红。
	phialEmpty = 0
	phialWhite = 1
	phialGold  = 2
)

// 三处砸地的落点，每项是 {沿面朝方向的距离, 偏转角}：正中 70 一处，左右各偏 32° 的 125 两处。
// 原来那套是「60° 扇环分 3 环」，三个圆叠在一起、又都落在 104 以内；
// 现在摊成三处独立的圆：圆心两两距离 75.4 / 132.5 / 75.4，都 ≥ 2×slamR = 52 → 三个圆互不重叠；
// 最远够到 125+26 = 151（原来 ≤104）。三波按数组顺序一波一处。
var slamSpots = [3][2]float64{
	{70, 0},
	{125, 32},
	{125, -32},
}

// 形态编码：0/1/2 直接发给前端，前端 axeSrc 按它挑 sheild/sword/axe 素材。
const (
	formShield = iota
	formSword
	formAxe
)

// 招式流水线（每一步都靠 a.until 计时推进，数字见上面的 const 块）：
//
//	盾/剑 stepIdle --正面 seekR≈24.7、120° 摸到人--> stepSlash1（二连斩 5+5）
//	  --slashLife 0.75s 后沿 thrustSp 350 突刺 1.5s，碰到人伤 3--> stepSlash2（再一套二连斩）
//	  --打完--> stepFill 站定 1.0s 灌瓶--> stepBottle 等 0.5s 转黄瓶
//	  --> stepAxeIn 0.2s--> stepAxeHold 1.0s--> stepAxeIdle（切斧，巡航降到 90）
//	红斧 stepAxeIdle --104/120° 摸到人--> stepDai 18 --0.35s--> stepTsui1 15 --0.4s--> stepTsui2 15
//	  --> stepChouThrust（350 顶 0.5s，碰到人伤 3）--> stepChouSpin 15
//	  --翻转 0.8+0.25s--> stepChouFan 30 --0.4s--> stepWave1/2/3（各 15、r=26，间隔 0.3s）
//	  --> 红态与瓶全掉、巡航回 165、切回盾
const (
	stepIdle = iota
	stepSlash1
	stepThrust
	stepSlash2
	stepFill
	stepBottle
	stepAxeIn
	stepAxeHold
	stepAxeIdle
	stepDai
	stepTsui1
	stepTsuiWait
	stepTsui2
	stepChouSpin
	stepChouFlip
	stepChouFan
	stepChouWait
	stepWave1
	stepWave2
	stepWave3
	// 超解前那一下突刺。追加在末尾，不改前面几个 step 的编号。
	stepChouThrust
)

func init() {
	p := unit.NewPack(KindAxe, assets)
	// Look.Ghost = 220：速度超过 220px/s 时前端才拉残影（也就是 350 的突刺那一段），平时走位不拉。
	p.Register(unit.Spec{
		Kind:    KindAxe,
		Role:    unit.RoleFighter,
		Radius:  axeRadius,
		MaxHP:   axeHP,
		Speed:   axeSpeed,
		Vision:  axeVision,
		Fighter: true,
		Look:    unit.Look{Color: axeColor, Ghost: 220, FX: []string{"axe"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &盾斧{form: formShield, hx: 0, hy: 1}
	})
}

type 盾斧 struct {
	form      int                   // formShield / formSword / formAxe
	redShield bool                  // 红盾：正面减伤 0.5 → 0.25
	redSword  bool                  // 红剑：strikeDmg 给剑/突刺 +redBonus
	phial     int                   // phialEmpty / phialWhite / phialGold，金瓶才够换红
	energy    int                   // 攒瓶能量，0..2，每打中一段 +1
	step      int                   // stepXxx
	until     float64               // 当前这一步的截止时刻（s.Time 走的那条时间轴）
	slashN    int                   // 二连斩已经出了几刀（0/1/2）
	lockID    uint64                // 锁定目标，突刺/转向优先找它
	thrustDir [2]float64            // 突刺方向（单位向量）
	hx, hy    float64               // 面朝方向（单位向量），hitFan/rotate 都按它算
	x, y      float64               // 自己的位置，front() 判定正面要用
	seen      map[uint64][2]float64 // 见过的单位位置：盾只挡见过的来源，目标脱离视野后还能记住最后位置
	slot      int
	booted    bool
}

func (a *盾斧) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		a.confirm(ctx, e)
	case unit.Collision:
		a.onHit(ctx, e)
	case unit.Sense:
		a.tick(ctx, e)
	}
}

func (a *盾斧) confirm(ctx unit.Context, d unit.IncomingDamage) {
	amt := d.Amount
	if a.form == formShield {
		// 盾形态才减伤，两个条件：来源得是「见过」的，且来源落在正面 shieldSpan=300°（±150°）里。
		// 普通盾吃一半 ×0.5，红盾吃四分之一 ×0.25；背后剩的 60° 是空档，不减。
		if src, ok := a.seen[d.From]; ok && a.front(src[0], src[1]) {
			if a.redShield {
				amt *= 0.25
			} else {
				amt *= 0.5
			}
		}
	}
	ctx.Out <- unit.ConfirmDamage{Token: d.Token, UnitID: ctx.ID, Amount: amt}
}

func (a *盾斧) front(ox, oy float64) bool {
	dx, dy := ox-a.x, oy-a.y
	if math.Hypot(dx, dy) < 1e-6 {
		return true
	}
	fn := math.Hypot(a.hx, a.hy)
	if fn < 1e-6 {
		return true
	}
	dot := (dx*a.hx + dy*a.hy) / (math.Hypot(dx, dy) * fn)
	if dot > 1 {
		dot = 1
	} else if dot < -1 {
		dot = -1
	}
	return math.Acos(dot) <= unit.Deg(shieldSpan)/2
}

func (a *盾斧) onHit(ctx unit.Context, e unit.Collision) {
	if !unit.EnemyTarget(e, a.slot) {
		return
	}
	switch a.step {
	case stepThrust:
		ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: false}
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: a.strikeDmg(thrustDmg)}
		a.beginSlash(ctx, e.Time, true)
	case stepChouThrust:
		// 超解前那一下突刺：撞上人就结 3 伤，然后接超解旋（顶到位是目的，不是打完收招）。
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: a.strikeDmg(thrustDmg)}
		a.beginChouSpin(ctx, e.Time)
	}
}

func (a *盾斧) tick(ctx unit.Context, s unit.Sense) {
	a.x, a.y = s.Self.X, s.Self.Y
	a.slot = s.Self.Slot
	a.note(s)
	a.face(s.Self.VX, s.Self.VY)
	if !a.booted {
		a.booted = true
		a.emitLook(ctx)
		a.emitPose(ctx, 90)
		a.emitPhial(ctx)
	}
	switch a.step {
	case stepIdle:
		if hit := a.firstInFan(s, seekR, seekSpan); hit != nil {
			a.energy = 0
			a.lockID = hit.ID
			a.toSword(ctx)
			a.beginSlash(ctx, s.Time, false)
		}
	case stepSlash1, stepSlash2:
		a.tickSlash(ctx, s)
	case stepThrust:
		a.tickThrust(ctx, s)
	case stepFill:
		if s.Time+1e-9 >= a.until {
			a.fillBottles(ctx)
			a.until = s.Time + bottleWait
			a.step = stepBottle
		}
	case stepBottle:
		if s.Time+1e-9 >= a.until {
			a.convertRed(ctx, s.Time)
		}
	case stepAxeIn:
		if s.Time+1e-9 >= a.until {
			a.hold(ctx, true)
			a.until = s.Time + axeStand
			a.step = stepAxeHold
		}
	case stepAxeHold:
		if s.Time+1e-9 >= a.until {
			a.form = formAxe
			a.step = stepAxeIdle
			a.hold(ctx, false)
			ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: axeSlow}
			a.emitLook(ctx)
			a.emitPose(ctx, 120)
		}
	case stepAxeIdle:
		if hit := a.firstInFan(s, axeSeekR, seekSpan); hit != nil {
			a.lockID = hit.ID
			a.turnTo(ctx, s.Self.X, s.Self.Y, hit.X, hit.Y)
			a.hold(ctx, true)
			a.step = stepDai
			a.until = s.Time + spinT
			a.emitPose(ctx, 300)
			a.emitAnim(ctx, s, "dai", 300, 120, spinT)
		}
	case stepDai:
		if s.Time+1e-9 >= a.until {
			a.turnToEnemy(ctx, s)
			a.hitFan(ctx, s, axeSeekR, seekSpan, daiDmg, false)
			a.step = stepTsui1
			a.until = s.Time + spinT
			a.emitAnim(ctx, s, "tsui", 120, 120, spinT)
		}
	case stepTsui1:
		if s.Time+1e-9 >= a.until {
			a.turnToEnemy(ctx, s)
			a.hitFan(ctx, s, axeSeekR, seekSpan, tsuiDmg, false)
			a.step = stepTsuiWait
			a.until = s.Time + tsuiGap
		}
	case stepTsuiWait:
		if s.Time+1e-9 >= a.until {
			a.turnToEnemy(ctx, s)
			a.step = stepTsui2
			a.until = s.Time + spinT
			a.emitAnim(ctx, s, "tsui-back", 120, 120, spinT)
		}
	case stepTsui2:
		if s.Time+1e-9 >= a.until {
			a.turnToEnemy(ctx, s)
			a.hitFan(ctx, s, axeSeekR, seekSpan, tsuiDmg, false)
			// 追解收完不再直接进超解旋：先补一下突刺顶进去（怪猎那套属性解放突刺）。
			a.beginChouThrust(ctx, s)
		}
	case stepChouThrust:
		a.tickChouThrust(ctx, s)
	case stepChouSpin:
		if s.Time+1e-9 >= a.until {
			a.turnToEnemy(ctx, s)
			a.hitFan(ctx, s, axeSeekR, seekSpan, chouSpinDmg, false)
			a.step = stepChouFlip
			a.until = s.Time + flipDown + flipUp
			a.emitAnim(ctx, s, "flip", flipDown, flipUp, 0)
		}
	case stepChouFlip:
		if s.Time+1e-9 >= a.until {
			a.turnToEnemy(ctx, s)
			a.hitFan(ctx, s, chouR, chouSpan, chouFanDmg, false)
			a.step = stepChouWait
			a.until = s.Time + waveWait
		}
	case stepChouWait:
		if s.Time+1e-9 >= a.until {
			a.turnToEnemy(ctx, s)
			a.wave(ctx, s, 0)
			a.step = stepWave1
			a.until = s.Time + waveGap
		}
	case stepWave1:
		if s.Time+1e-9 >= a.until {
			a.turnToEnemy(ctx, s)
			a.wave(ctx, s, 1)
			a.step = stepWave2
			a.until = s.Time + waveGap
		}
	case stepWave2:
		if s.Time+1e-9 >= a.until {
			a.turnToEnemy(ctx, s)
			a.wave(ctx, s, 2)
			a.dropRed(ctx)
			a.step = stepIdle
			a.hold(ctx, false)
		}
	}
}

func (a *盾斧) beginSlash(ctx unit.Context, now float64, second bool) {
	if second {
		a.step = stepSlash2
	} else {
		a.step = stepSlash1
	}
	a.slashN = 0
	a.until = now + slashLife
	a.hold(ctx, true)
	a.emitAnim(ctx, unit.Sense{Time: now, Self: unit.Snapshot{X: a.x, Y: a.y, VX: a.hx, VY: a.hy}}, "slash", 150, 30, slashBeat)
}

func (a *盾斧) tickSlash(ctx unit.Context, s unit.Sense) {
	elapsed := s.Time - (a.until - slashLife)
	if a.slashN == 0 {
		a.hitFan(ctx, s, seekR, seekSpan, a.strikeDmg(slashDmg), true)
		a.slashN = 1
	} else if a.slashN == 1 && elapsed+1e-9 >= slashGap {
		a.hitFan(ctx, s, seekR, seekSpan, a.strikeDmg(slashDmg), false)
		a.slashN = 2
	}
	if s.Time+1e-9 < a.until {
		return
	}
	if a.step == stepSlash1 {
		a.aimThrust(s)
		a.step = stepThrust
		a.until = s.Time + thrustLife
		a.hold(ctx, false)
		ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: a.thrustDir[0] * thrustSp, VY: a.thrustDir[1] * thrustSp}
		return
	}
	a.endCombo(ctx, s.Time)
}

func (a *盾斧) tickThrust(ctx unit.Context, s unit.Sense) {
	a.aimThrust(s)
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: a.thrustDir[0] * thrustSp, VY: a.thrustDir[1] * thrustSp}
	if s.Time+1e-9 >= a.until {
		a.endCombo(ctx, s.Time)
	}
}

// beginChouThrust 超解前那一下突刺：朝锁定目标 350 速顶过去。
// 命中（伤 3）或超时都接超解旋——这一下是为了把人顶进落点，不是独立的招。
func (a *盾斧) beginChouThrust(ctx unit.Context, s unit.Sense) {
	a.aimThrust(s)
	a.step = stepChouThrust
	a.until = s.Time + chouThrustLife
	a.hold(ctx, false)
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: a.thrustDir[0] * thrustSp, VY: a.thrustDir[1] * thrustSp}
}

func (a *盾斧) tickChouThrust(ctx unit.Context, s unit.Sense) {
	a.aimThrust(s)
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: a.thrustDir[0] * thrustSp, VY: a.thrustDir[1] * thrustSp}
	if s.Time+1e-9 >= a.until {
		a.beginChouSpin(ctx, s.Time)
	}
}

// beginChouSpin 起超解旋：突刺命中或走完都从这里接着往下砸。
func (a *盾斧) beginChouSpin(ctx unit.Context, now float64) {
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: false}
	a.hold(ctx, true) // 顶到位就站定（hold 顺带把速度清零）
	a.step = stepChouSpin
	a.until = now + spinT
	a.emitAnim(ctx, unit.Sense{Time: now, Self: unit.Snapshot{X: a.x, Y: a.y, VX: a.hx, VY: a.hy}}, "chou", 120, 120, spinT)
}

func (a *盾斧) aimThrust(s unit.Sense) {
	tx, ty, ok := a.lockPos(s)
	if !ok {
		if hit := a.firstInFan(s, seekR*2, 360); hit != nil {
			a.lockID = hit.ID
			tx, ty, ok = hit.X, hit.Y, true
		}
	}
	if !ok {
		if math.Hypot(a.thrustDir[0], a.thrustDir[1]) < 1e-6 {
			a.thrustDir = [2]float64{a.hx, a.hy}
		}
		return
	}
	dx, dy := tx-s.Self.X, ty-s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	a.thrustDir = [2]float64{dx / n, dy / n}
	a.hx, a.hy = a.thrustDir[0], a.thrustDir[1]
}

func (a *盾斧) lockPos(s unit.Sense) (x, y float64, ok bool) {
	if a.lockID == 0 {
		return 0, 0, false
	}
	for i := range s.Nearby {
		if s.Nearby[i].ID == a.lockID {
			return s.Nearby[i].X, s.Nearby[i].Y, true
		}
	}
	p, ok := a.seen[a.lockID]
	return p[0], p[1], ok
}

func (a *盾斧) endCombo(ctx unit.Context, now float64) {
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: false}
	a.toShield(ctx)
	a.hold(ctx, true)
	a.step = stepFill
	a.until = now + fillStand
	a.emitPhial(ctx)
}

func (a *盾斧) fillBottles(ctx unit.Context) {
	switch a.energy {
	case 1:
		if a.phial == phialWhite {
			a.phial = phialGold
		} else if a.phial == phialEmpty {
			a.phial = phialWhite
		}
	case 2:
		a.phial = phialGold
	}
	a.emitPhial(ctx)
}

func (a *盾斧) convertRed(ctx unit.Context, now float64) {
	if a.phial != phialGold {
		a.step = stepIdle
		a.hold(ctx, false)
		return
	}
	a.phial = phialEmpty
	a.emitPhial(ctx)
	if !a.redShield {
		a.redShield = true
		a.step = stepIdle
		a.hold(ctx, false)
		a.emitLook(ctx)
		return
	}
	if !a.redSword {
		a.redSword = true
		a.step = stepIdle
		a.hold(ctx, false)
		a.emitLook(ctx)
		return
	}
	a.step = stepAxeIn
	a.until = now + axeWind
	a.hold(ctx, false)
	a.emitLook(ctx)
}

func (a *盾斧) dropRed(ctx unit.Context) {
	a.redShield = false
	a.redSword = false
	a.phial = phialEmpty
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: axeSpeed}
	a.toShield(ctx)
	a.emitPhial(ctx)
}

func (a *盾斧) toSword(ctx unit.Context) {
	a.form = formSword
	a.emitLook(ctx)
	pose := 120.0
	a.emitPose(ctx, pose)
}

func (a *盾斧) toShield(ctx unit.Context) {
	a.form = formShield
	a.emitLook(ctx)
	a.emitPose(ctx, 90)
}

func (a *盾斧) turnToEnemy(ctx unit.Context, s unit.Sense) {
	var hit *unit.Snapshot
	if a.lockID != 0 {
		for i := range s.Nearby {
			o := &s.Nearby[i]
			if o.ID == a.lockID && unit.Hittable(*o, s.Self.Slot) {
				hit = o
				break
			}
		}
	}
	if hit == nil {
		hit = unit.Seek(s)
	}
	if hit == nil {
		return
	}
	a.lockID = hit.ID
	a.turnTo(ctx, s.Self.X, s.Self.Y, hit.X, hit.Y)
}

func (a *盾斧) turnTo(ctx unit.Context, x, y, tx, ty float64) {
	dx, dy := tx-x, ty-y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	a.hx, a.hy = dx/n, dy/n
	ctx.Out <- unit.FX{Name: "face", Kind: ctx.Kind, UnitID: ctx.ID, VX: a.hx, VY: a.hy}
}

func (a *盾斧) hold(ctx unit.Context, on bool) {
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: on}
	if on {
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	}
}

// strikeDmg 是剑/突刺类伤害的总入口：红剑态、且不在斧态时加 redBonus=2（二连斩 5→7、突刺 3→5）。
func (a *盾斧) strikeDmg(base float64) float64 {
	if a.redSword && a.form != formAxe {
		return base + redBonus
	}
	return base
}

func (a *盾斧) hitFan(ctx unit.Context, s unit.Sense, r, span, dmg float64, energy bool) {
	hit := false
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) || !inFan(s.Self, a.hx, a.hy, *o, r, span) {
			continue
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: dmg}
		hit = true
	}
	// 只要这一刀造成了伤害就回能量：打活随从也算，不再只认本体。上限 2（到金瓶就不再涨）。
	if energy && hit && a.energy < 2 {
		a.energy++
		a.emitPhial(ctx)
	}
}

// wave 超解砸地：一次落一处圆（i 是 slamSpots 下标，0/1/2 对应三波）。
// 三处摊开摆、两两不重叠，最远推到 151；出伤判定就是这个圆本身——观众看见的圆和挨打的圈是同一个：
// 命中条件是「目标圆心距 ≤ slamR + 目标半径」，也就是两个圆真的碰到了。
func (a *盾斧) wave(ctx unit.Context, s unit.Sense, i int) {
	dist, deg := slamSpots[i][0], slamSpots[i][1]
	ux, uy := a.rotate(deg)
	cx, cy := s.Self.X+ux*dist, s.Self.Y+uy*dist
	for j := range s.Nearby {
		o := &s.Nearby[j]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		if math.Hypot(o.X-cx, o.Y-cy) > slamR+o.Radius {
			continue
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: waveDmg}
	}
	// FX 把「方向 × 半径」压进 VX/VY：前端 hypot(VX,VY) 还原出 slamR=26，
	// 再按世界坐标乘 scale 画直径 2r。Amount 这里前端不用（恒 1，只表示"一处"）。
	ctx.Out <- unit.FX{
		Name: "wave", Kind: ctx.Kind, UnitID: ctx.ID,
		X: cx, Y: cy,
		VX: ux * slamR, VY: uy * slamR,
		Amount: 1, Slot: s.Self.Slot,
	}
}

// rotate 把面朝方向转 deg 度（逆时针为正），得到这一处砸地的朝向。
func (a *盾斧) rotate(deg float64) (float64, float64) {
	rad := unit.Deg(deg)
	c, sn := math.Cos(rad), math.Sin(rad)
	return a.hx*c - a.hy*sn, a.hx*sn + a.hy*c
}

func (a *盾斧) firstInFan(s unit.Sense, r, span float64) *unit.Snapshot {
	return unit.SeekIf(s, func(o unit.Snapshot) bool {
		return inFan(s.Self, a.hx, a.hy, o, r, span)
	})
}

func inFan(self unit.Snapshot, hx, hy float64, o unit.Snapshot, r, spanDeg float64) bool {
	dx, dy := o.X-self.X, o.Y-self.Y
	dist := math.Hypot(dx, dy)
	if dist > r+o.Radius {
		return false
	}
	if dist < 1e-6 {
		return true
	}
	fn := math.Hypot(hx, hy)
	if fn < 1e-6 {
		return true
	}
	dot := (dx*hx + dy*hy) / (dist * fn)
	if dot > 1 {
		dot = 1
	} else if dot < -1 {
		dot = -1
	}
	return math.Acos(dot) <= unit.Deg(spanDeg)/2
}

func (a *盾斧) face(vx, vy float64) {
	if math.Hypot(vx, vy) < 1e-6 {
		return
	}
	n := math.Hypot(vx, vy)
	a.hx, a.hy = vx/n, vy/n
}

func (a *盾斧) note(s unit.Sense) {
	if a.seen == nil {
		a.seen = map[uint64][2]float64{}
	}
	a.seen[s.Self.ID] = [2]float64{s.Self.X, s.Self.Y}
	for i := range s.Nearby {
		o := s.Nearby[i]
		a.seen[o.ID] = [2]float64{o.X, o.Y}
	}
}

// emitLook 三个字段各有约定，前端原样收：Amount=形态(0 盾/1 剑/2 斧)、VX=红盾(0/1)、VY=红剑(0/1)。
func (a *盾斧) emitLook(ctx unit.Context) {
	form := float64(a.form)
	rs, rw := 0.0, 0.0
	if a.redShield {
		rs = 1
	}
	if a.redSword {
		rw = 1
	}
	ctx.Out <- unit.FX{Name: "look", Kind: ctx.Kind, UnitID: ctx.ID, Amount: form, VX: rs, VY: rw}
}

// emitPose 发一个「定格姿态角」（度）：90=盾（素材朝北就是 90）、120=剑/斧、300=大解起手。
// 前端换成 CSS 是 poseCss = 90 - pose。
func (a *盾斧) emitPose(ctx unit.Context, deg float64) {
	ctx.Out <- unit.FX{Name: "pose", Kind: ctx.Kind, UnitID: ctx.ID, Amount: deg}
}

// emitAnim 前端动画三件套：VX/VY = 起止姿态角（度），Amount = 时长（秒）。
// flip 是例外：这三个字段被当成 (压下秒, 翻回秒, 0) 用，见 fx/shot.js 的 flip。
func (a *盾斧) emitAnim(ctx unit.Context, s unit.Sense, name string, from, to, dur float64) {
	ctx.Out <- unit.FX{
		Name: name, Kind: ctx.Kind, UnitID: ctx.ID,
		X: s.Self.X, Y: s.Self.Y, VX: from, VY: to, Amount: dur, Slot: s.Self.Slot,
	}
}

// emitPhial 的 amt 是给前端的图号：斩击那几步发能量值 0..2（空/三瓶/过充），
// 其余时候发瓶色码 0/10/20（空/满瓶/强化瓶）——前端 axePhialSrc 按这些数换图。
func (a *盾斧) emitPhial(ctx unit.Context) {
	amt := 0.0
	if a.step == stepSlash1 || a.step == stepThrust || a.step == stepSlash2 {
		amt = float64(a.energy)
	} else if a.phial == phialWhite {
		amt = 10
	} else if a.phial == phialGold {
		amt = 20
	}
	ctx.Out <- unit.FX{Name: "phial", Kind: ctx.Kind, UnitID: ctx.ID, Amount: amt}
}
