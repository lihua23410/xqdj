// 时计·铭刻：时计的 SP 形态。本体的表针与「致命自动回拨」被整包换成一列卡牌。
//
// 抽卡：每 1s 攒 1 点，满 10 点从牌库顶抽 1 张；手牌无上限（HUD 只画最早 5 张）。
// 牌库空了当场重铸成初始 8 张（不额外发牌）；牌底可以被【烬时之斩】直接抽走，
// 所以牌底躺着的【溯时】不会被重铸吞掉——牌没抽空就不会重铸。
//
// 出牌：每 10s 一个窗口，窗内随机掷 1/2 张，在窗内随机时刻打出，两张至少隔 1s。
// 只从「当前打得出来」的牌里随机（没有记录时的 曜时之旅/溯时 是死牌）。
//
// 记录：唯一槽位、覆盖式、永不过期。铭刻把当前状态记下来并变成【溯时】沉到牌库底；
// 曜时之旅读回属性，溯时连手牌与抽卡进度一起读回。两张回溯卡都不消耗记录。
//
// 命中抽底：20s 冷却就绪时，冲刺攻击真的打到人 → 抽牌库底 1 张 → 开始冷却。
// 冷却剩余时间进记录槽，所以回溯能把冷却倒回去。
//
// 实现口径：没有任何自动保命（挨打直接 ConfirmHit）；记录/还原全部用现有 Cmd 逐条回放，
// 不扩引擎协议（与时计同路）。
package 时计_铭刻

import (
	"embed"
	"math"
	"math/rand/v2"
	"sort"

	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

// KindInscribed 是选人界面上的名字。
const KindInscribed = "时计·铭刻"

// KindInscribedArc 是它身上那圈常驻弧。
const KindInscribedArc = "时计铭刻弧"

const (
	inscribedRadius = 18.0
	inscribedSpeed  = 165.0
	inscribedHP     = 60.0
	inscribedVision = 400.0
	inscribedColor  = "#f2f2f2"

	// —— 常驻弧：唯一的出伤来源（冲刺只负责把人送过去）——
	// 只砍身前那 90° 一扇（同原型机_近战），不是整圈：从背后贴上来的敌人打不到。
	arcInner   = inscribedRadius
	arcOuter   = inscribedRadius + 2
	arcSpanDeg = 90.0
	arcDamage  = 4.0 // 基础伤害；时痕的 +X 由引擎的 SetDamageBonus 统一加
	arcCD      = 0.6 // 撞碎之后重新长出来要这么久

	// —— 冲刺（只位移，不结算伤害）——
	dashSpeed     = 420.0 // 冲刺速度
	dashMinSpan   = 0.15
	dashMaxSpan   = 0.60
	dashOvershoot = 40.0 // 冲过目标点这么多再收

	// —— 烬时之斩 ——
	emberRadius = 144.0
	emberDamage = 5.0

	// —— buff：都会过期 ——
	// 参考百闻牌的式神复活：那边复活丢 buff，所以回溯稳赚；这边 buff 原本永驻，
	// 回溯只会越回越少。改成限时后，回溯=把过期/即将过期的 buff 重新拉满才成立。
	buffLife  = 20.0 // 每层时痕 / 长守各自活这么久
	traceStep = 3.5  // 时痕：每层造成的伤害 +3.5
	guardPool = 15.0 // 长守：每层给一条 20 点的额外血条

	// —— 节奏 ——
	pointGap  = 1.0  // 每 1s 攒 1 点
	pointCap  = 5.0  // 满 5 点抽 1 张 = 5 秒一张
	windowGap = 5.0  // 出牌窗口长度
	playGap   = 1.0  // 同窗两张牌的最小间隔
	hitDrawCD = 20.0 // 命中抽底牌的冷却

	// —— 出牌判断 ——
	markMinHand   = 3    // 铭刻：手里攒到这么多牌才值得写记录
	recordHealthy = 0.70 // 铭刻：血量不低于 70% 上限才写记录

)

const (
	cardNone      = iota
	cardTrace     // 时痕
	cardGuard     // 长守
	cardJourney   // 曜时之旅
	cardMark      // 铭刻
	cardEmber     // 烬时之斩
	cardBacktrack // 溯时
)

// 标记与图标的公共名：HUD 上按层数显示，回溯时整包重放。
const (
	markTrace = "时痕"
	markGuard = "长守"
	iconTrace = "/ball/" + KindInscribed + "/fx/trace.svg"
	iconGuard = "/ball/" + KindInscribed + "/fx/guard.svg"
)

// 前端 FX 名一律带包前缀：引擎的 IMMEDIATE_AUDIO 里那几个通用名只出声不画。
const (
	fxDash     = "tokei-inscribed-dash"
	fxHit      = "tokei-inscribed-hit"
	fxMark     = "tokei-inscribed-mark"
	fxRewind   = "tokei-inscribed-rewind"
	fxEmber    = "tokei-inscribed-ember"
	fxTrace    = "tokei-inscribed-trace"
	fxGuard    = "tokei-inscribed-guard"
	fxEmberPop = "tokei-inscribed-ember-pop"
)

func init() {
	p := unit.NewPack(KindInscribed, assets)
	p.Register(unit.Spec{
		Kind:    KindInscribed,
		Role:    unit.RoleFighter,
		Radius:  inscribedRadius,
		MaxHP:   inscribedHP,
		Speed:   inscribedSpeed,
		Vision:  inscribedVision,
		Fighter: true,
		Look: unit.Look{
			Color: inscribedColor,
			FX:    []string{"tokei-inscribed"},
		},
	}, func(unit.SpawnInfo) unit.Actor { return &铭刻{} })
	// 常驻弧：一张朝正前方张开的刀面（弧自己按主人速度/朝向转向），撞到敌方
	// 战斗机/活随从就出 arcDamage 点并自毁，主人每 arcCD 秒把它重新长出来。
	// 冲刺本身不结算伤害，全靠这一刀磨。
	p.Register(unit.Spec{
		Kind:     KindInscribedArc,
		Role:     unit.RoleProjectile,
		Radius:   arcOuter,
		MaxHP:    1,
		Speed:    inscribedSpeed,
		Vision:   0,
		Attach:   true,
		ArcSpan:  unit.Deg(arcSpanDeg),
		ArcInner: arcInner,
		Look:     unit.Look{Color: inscribedColor, Overlay: true},
	}, func(info unit.SpawnInfo) unit.Actor { return &铭刻弧{slot: info.Slot} })
}

// 铭刻弧 撞到敌人就结算一次，然后散掉等主人重挂。
type 铭刻弧 struct{ slot int }

func (a *铭刻弧) Handle(ctx unit.Context, ev unit.Event) {
	e, ok := ev.(unit.Collision)
	if !ok || !unit.EnemyTarget(e, a.slot) {
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: arcDamage}
	ctx.Out <- unit.FX{
		Name: fxHit, Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot,
		X: e.Other.X, Y: e.Other.Y,
	}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

// record 是一次「铭刻」拍下来的全部状态。牌库不在里面：回溯不动牌库。
type record struct {
	x, y         float64
	vx, vy       float64
	hp, maxHP    float64
	faceX, faceY float64
	trace, guard int
	guardPool    float64
	hand         []uint8
	progress     float64
	cdLeft       float64 // 记录那一刻的命中抽牌冷却剩余
}

type 铭刻 struct {
	rng    *rand.Rand
	booted bool
	slot   int

	x, y         float64
	hp, maxHP    float64
	faceX, faceY float64

	deck     []uint8
	hand     []uint8
	progress float64
	nextAt   float64 // 下一个抽卡点的绝对时刻

	winStart   float64   // 当前出牌窗口起点
	plan       []float64 // 本窗排好的出牌时刻
	lastPlayAt float64

	traceAt   []float64 // 时痕每层的到期时刻
	guardAt   []float64 // 长守每层的到期时刻（池子另记）
	guardPool float64   // 长守的额外血量池（引擎 Absorb）

	rec  record
	have bool

	dashing   bool
	dashUntil float64
	arc       unit.AttachState

	cdReadyAt float64 // 命中抽牌的冷却就绪时刻；<= now 表示就绪
}

func (a *铭刻) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		// 这套牌没有免费保命：挨打就照单全收。
		unit.AcceptHit(ctx, e)
	case unit.Sense:
		a.tick(ctx, e)
	}
}

func (a *铭刻) tick(ctx unit.Context, s unit.Sense) {
	a.slot = s.Self.Slot
	a.x, a.y = s.Self.X, s.Self.Y
	a.hp, a.maxHP = s.Self.HP, s.Self.MaxHP
	a.faceX, a.faceY = s.Self.FaceX, s.Self.FaceY
	if !a.booted {
		a.boot(ctx, s)
	}
	a.expireBuffs(ctx, s.Time)
	a.charge(ctx, s)
	// 先 play 再 schedule：窗口交界那一拍如果先滚新窗，上一窗排在末尾的那张
	// 会被直接丢掉（实测丢了近两成出牌机会）。
	a.play(ctx, s)
	a.schedule(ctx, s)
	a.armArc(ctx, s)
	a.fight(ctx, s)
	a.touchDraw(ctx, s)
}

// armArc 常驻弧：撞碎之后等 arcCD 再长出来，平时一直挂着。
func (a *铭刻) armArc(ctx unit.Context, s unit.Sense) {
	if unit.RearmAttach(s, ctx.ID, KindInscribedArc, arcCD, &a.arc) {
		unit.SpawnAttach(ctx, s, KindInscribedArc)
	}
}

func (a *铭刻) boot(ctx unit.Context, s unit.Sense) {
	a.booted = true
	if a.rng == nil {
		a.rng = unit.NewRNG(ctx.Stream)
	}
	a.deck = baseDeck(a.rng)
	// 开局手牌：铭刻 + 从牌库顶抽 1 张（牌库剩 7 张）。
	a.hand = []uint8{cardMark}
	a.drawTop()
	a.nextAt = s.Time + pointGap
	a.winStart = windowStart(s.Time)
	a.plan = a.rollPlan(a.winStart)
}

// expireBuffs 两层 buff 各自按层过期：时痕掉层就重算加伤，长守掉层就削一截池子。
// 引擎那边不需要新协议——Absorb 是现成的，标记层数也是现成的。
func (a *铭刻) expireBuffs(ctx unit.Context, now float64) {
	if len(a.traceAt) > 0 {
		kept := a.traceAt[:0]
		gone := 0
		for _, at := range a.traceAt {
			if now+1e-9 >= at {
				gone++
				continue
			}
			kept = append(kept, at)
		}
		a.traceAt = kept
		if gone > 0 {
			ctx.Out <- unit.StackMark{UnitID: ctx.ID, Kind: markTrace, Delta: -gone}
			ctx.Out <- unit.SetDamageBonus{UnitID: ctx.ID, Amount: float64(len(a.traceAt)) * traceStep}
		}
	}
	if len(a.guardAt) > 0 {
		kept := a.guardAt[:0]
		gone := 0
		for _, at := range a.guardAt {
			if now+1e-9 >= at {
				gone++
				continue
			}
			kept = append(kept, at)
		}
		a.guardAt = kept
		if gone > 0 {
			a.guardPool -= guardPool * float64(gone)
			if a.guardPool < 0 {
				a.guardPool = 0
			}
			ctx.Out <- unit.SetAbsorb{UnitID: ctx.ID, Amount: a.guardPool}
		}
	}
}

func baseDeck(r *rand.Rand) []uint8 {
	d := make([]uint8, 0, 8)
	for i := 0; i < 2; i++ {
		d = append(d, cardTrace, cardGuard, cardJourney, cardEmber)
	}
	r.Shuffle(len(d), func(i, j int) { d[i], d[j] = d[j], d[i] })
	return d
}

func windowStart(t float64) float64 {
	if t < 0 {
		return 0
	}
	return math.Floor(t/windowGap) * windowGap
}

// rollPlan 给一个窗口排出去牌时刻：先均匀掷 2/3 张，再把窗里剩下的空档
// 随机切给这几张牌。这样每一格都排得下（间隔天然 ≥ playGap），不会像
// 「先随机再撑开」那样把最后一张挤到窗外白白丢掉。
func (a *铭刻) rollPlan(win float64) []float64 {
	if a.rng == nil {
		return nil
	}
	n := 2 + a.rng.IntN(2)
	// 窗口里能自由分配的空档：总长减去 (n-1) 个最小间隔。
	slack := (windowGap - float64(n-1)*playGap) * 0.999
	cuts := make([]float64, n-1)
	for i := range cuts {
		cuts[i] = a.rng.Float64() * slack
	}
	sort.Float64s(cuts)
	plan := make([]float64, 0, n)
	t := win
	prev := 0.0
	for i := 0; i < n; i++ {
		if i > 0 {
			t += playGap
			t += cuts[i-1] - prev
			prev = cuts[i-1]
		}
		plan = append(plan, t)
	}
	return plan
}

// charge 每 1s 攒 1 点；满 10 点抽一张。眩晕时收不到 Sense，节拍自然暂停。
func (a *铭刻) charge(ctx unit.Context, s unit.Sense) {
	for i := 0; i < 200 && s.Time+1e-9 >= a.nextAt; i++ {
		a.nextAt += pointGap
		a.progress++
		if a.progress+1e-9 < pointCap {
			continue
		}
		a.progress -= pointCap
		a.drawTop()
	}
}

func (a *铭刻) refill() {
	a.deck = baseDeck(a.rng)
}

// drawTop 从牌库顶抽一张；抽空的那一刻当场重洗成初始八张。
func (a *铭刻) drawTop() {
	if len(a.deck) == 0 {
		a.refill()
	}
	a.hand = append(a.hand, a.deck[0])
	a.deck = a.deck[1:]
	if len(a.deck) == 0 {
		a.refill()
	}
}

// drawBottom 从牌库底抽一张（烬时之斩、命中抽牌）。
func (a *铭刻) drawBottom() uint8 {
	if len(a.deck) == 0 {
		a.refill()
	}
	n := len(a.deck) - 1
	c := a.deck[n]
	a.deck = a.deck[:n]
	a.hand = append(a.hand, c)
	if len(a.deck) == 0 {
		a.refill()
	}
	return c
}

func (a *铭刻) schedule(ctx unit.Context, s unit.Sense) {
	win := windowStart(s.Time)
	if win == a.winStart {
		return
	}
	a.winStart = win
	a.plan = a.rollPlan(win)
}

// play 到点就打一张。只从当前打得出来的牌里随机；打不出来就空过这一格。
func (a *铭刻) play(ctx unit.Context, s unit.Sense) {
	for len(a.plan) > 0 && s.Time+1e-9 >= a.plan[0] {
		a.plan = a.plan[1:]
		if s.Time+1e-9 < a.lastPlayAt+playGap {
			continue
		}
		idx, ok := a.pickPlayable()
		if !ok {
			continue
		}
		a.playCard(ctx, s, idx)
		a.lastPlayAt = s.Time
	}
}

// playable 决定这张牌此刻该不该打。曜时之旅是唯一的回血手段，掉血够多才值得；
// 溯时只回收手牌，记录里的牌比现在多才值得读；铭刻要血够健康、手里攒够牌才写。
// playable 决定这张牌此刻该不该打。曜时之旅是唯一的回血手段，掉血够多才值得；
// 溯时只回收手牌，记录里的牌比现在多才值得读；铭刻要血够健康、手里攒够牌才写。
func (a *铭刻) playable(c uint8) bool {
	switch c {
	case cardJourney:
		// 曜时之旅现在只刷新 buff：哪样掉了才值得读档。
		return a.have && (len(a.traceAt) < a.rec.trace || a.guardPool < a.rec.guardPool-1e-9)
	case cardBacktrack:
		return a.have && len(a.rec.hand) > len(a.hand)
	case cardMark:
		return !a.have || (a.hp >= a.maxHP*recordHealthy && len(a.hand) >= markMinHand)
	case cardNone:
		return false
	default:
		return true
	}
}

func (a *铭刻) pickPlayable() (int, bool) {
	if a.rng == nil {
		return 0, false
	}
	cand := make([]int, 0, len(a.hand))
	for i, c := range a.hand {
		if a.playable(c) {
			cand = append(cand, i)
		}
	}
	if len(cand) == 0 {
		return 0, false
	}
	return cand[a.rng.IntN(len(cand))], true
}

func (a *铭刻) playCard(ctx unit.Context, s unit.Sense, idx int) {
	c := a.hand[idx]
	a.hand = append(a.hand[:idx], a.hand[idx+1:]...)
	switch c {
	case cardTrace:
		a.traceAt = append(a.traceAt, s.Time+buffLife)
		ctx.Out <- unit.StackMark{UnitID: ctx.ID, Kind: markTrace, Delta: 1, Icon: iconTrace}
		ctx.Out <- unit.SetDamageBonus{UnitID: ctx.ID, Amount: float64(len(a.traceAt)) * traceStep}
		ctx.Out <- unit.FX{
			Name: fxTrace, Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot,
			X: a.x, Y: a.y, Amount: float64(len(a.traceAt)),
		}
		a.attack(ctx, s)
	case cardGuard:
		a.guardAt = append(a.guardAt, s.Time+buffLife)
		a.guardPool += guardPool
		ctx.Out <- unit.SetAbsorb{UnitID: ctx.ID, Amount: a.guardPool}
		ctx.Out <- unit.FX{
			Name: fxGuard, Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot,
			X: a.x, Y: a.y, Amount: a.guardPool,
		}
		a.attack(ctx, s)
	case cardJourney:
		// 只回溯属性，牌不给。
		a.applyRecord(ctx, s, false)
		ctx.Out <- unit.FX{
			Name: fxRewind, Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot,
			X: a.x, Y: a.y,
		}
	case cardMark:
		// 先把自己从手牌里拿掉再记录：记录里的手牌不该含着这张铭刻。
		a.snapshot(s)
		a.deck = append(a.deck, cardBacktrack) // 沉到牌库底
		ctx.Out <- unit.FX{
			Name: fxMark, Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot,
			X: a.x, Y: a.y,
		}
	case cardEmber:
		a.ember(ctx, s)
		a.attack(ctx, s)
		a.drawBottom()
		ctx.Out <- unit.FX{
			Name: fxEmberPop, Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot,
			X: a.x, Y: a.y,
		}
	case cardBacktrack:
		a.applyRecord(ctx, s, true)
		a.hand = append(a.hand, cardMark) // 溯时用掉，变回一张铭刻
		ctx.Out <- unit.FX{
			Name: fxRewind, Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot,
			X: a.x, Y: a.y,
		}
	}
}

// ember 烬时之斩的范围伤害：只打敌方、不看墙。
func (a *铭刻) ember(ctx unit.Context, s unit.Sense) {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		if math.Hypot(o.X-a.x, o.Y-a.y) > emberRadius+o.Radius {
			continue
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: emberDamage, NoFreeze: true}
	}
	ctx.Out <- unit.FX{
		Name: fxEmber, Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot,
		X: a.x, Y: a.y, Amount: emberRadius,
	}
}

// attack 一次冲刺索敌：锁视野内最近的可瞄准目标，朝它冲一条**直线**，撞到就停。
// 方向在出手那一刻定死（不追踪），时长按距离折算（速度恒定 420）：
// 死守 0.25s 的 105 px 冲程在真实场地里等于永远扑空；上限保证不会一口气横穿全场。
func (a *铭刻) attack(ctx unit.Context, s unit.Sense) {
	dx, dy := a.faceX, a.faceY
	dist := 0.0
	if target := unit.Seek(s); target != nil {
		// 瞄提前量，但路径仍是直线：目标也以 165 匀速跑，照着它现在的位置冲必然擦肩。
		// 迭代两次就够（第二次只是把"追到预测点要多久"算进预测点本身）。
		dx, dy = target.X-a.x, target.Y-a.y
		dist = math.Hypot(dx, dy)
		for i := 0; i < 2; i++ {
			t := dist / dashSpeed
			dx, dy = target.X+target.VX*t-a.x, target.Y+target.VY*t-a.y
			dist = math.Hypot(dx, dy)
		}
	}
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		dx, dy, n = 1, 0, 1
	}
	ux, uy := dx/n, dy/n
	span := (dist + dashOvershoot) / dashSpeed
	if span < dashMinSpan {
		span = dashMinSpan
	}
	if span > dashMaxSpan {
		span = dashMaxSpan
	}
	a.dashing = true
	a.dashUntil = s.Time + span
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: ux * dashSpeed, VY: uy * dashSpeed}
	ctx.Out <- unit.FX{
		Name: fxDash, Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot,
		X: a.x, Y: a.y, VX: ux * dashSpeed, VY: uy * dashSpeed,
	}
}

// fight 冲刺只管位移：冲满时长就收，伤害全交给常驻弧。
func (a *铭刻) fight(ctx unit.Context, s unit.Sense) {
	if a.dashing {
		if s.Time+1e-9 >= a.dashUntil {
			a.endDash(ctx, s)
		}
		return
	}
	a.steer(ctx, s)
}

// endDash 收冲刺，立刻接回巡航速度——不然引擎要从 0 重新加速 6 秒才回到 165。
func (a *铭刻) endDash(ctx unit.Context, s unit.Sense) {
	a.dashing = false
	if dx, dy, ok := a.desiredDir(s); ok {
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: dx * inscribedSpeed, VY: dy * inscribedSpeed}
		ctx.Out <- unit.SetFSDirection{UnitID: ctx.ID, VX: dx, VY: dy}
		return
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID}
}

// touchDraw 命中抽底：弧贴到人的时候算命中，20s 冷却就绪才抽，抽完才开始冷却。
// 弧自己的结算写在 铭刻弧 里（那时候主人拿不到回执），这里只按同一扇范围判一下。
func (a *铭刻) touchDraw(ctx unit.Context, s unit.Sense) {
	if s.Time+1e-9 < a.cdReadyAt {
		return
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		if math.Hypot(o.X-a.x, o.Y-a.y) > s.Self.Radius+o.Radius+4 {
			continue
		}
		if !inArcCone(s.Self, *o) {
			continue
		}
		a.cdReadyAt = s.Time + hitDrawCD
		a.drawBottom()
		return
	}
}

// arcAim 跟引擎给挂件定朝向的规矩一致：动起来朝速度，站住了朝主人朝向。
func arcAim(s unit.Snapshot) (float64, float64) {
	if n := math.Hypot(s.VX, s.VY); n > 1e-6 {
		return s.VX / n, s.VY / n
	}
	if n := math.Hypot(s.FaceX, s.FaceY); n > 1e-6 {
		return s.FaceX / n, s.FaceY / n
	}
	return 1, 0
}

// inArcCone 目标在不在身前那扇里：和引擎 sweptArcVsCircle 的角度判定同一套。
func inArcCone(s unit.Snapshot, o unit.Snapshot) bool {
	dx, dy := o.X-s.X, o.Y-s.Y
	n := math.Hypot(dx, dy)
	if n <= 1e-9 {
		return true
	}
	ax, ay := arcAim(s)
	return (dx*ax+dy*ay)/n >= math.Cos(unit.Deg(arcSpanDeg)/2)-1e-9
}

// desiredDir 决定平时往哪走：笔直朝目标，不绕圈。没有目标就顺着朝向。
func (a *铭刻) desiredDir(s unit.Sense) (float64, float64, bool) {
	dx, dy := a.faceX, a.faceY
	if target := unit.Seek(s); target != nil {
		dx, dy = target.X-a.x, target.Y-a.y
	}
	if math.Hypot(dx, dy) < 1e-6 {
		return 0, 0, false
	}
	// 必须归一化：endDash 会拿它乘速度，原始差值会把速度放大到几千。
	n := math.Hypot(dx, dy)
	return dx / n, dy / n, true
}

// steer 平时的走位。冲刺期间不抢方向。
func (a *铭刻) steer(ctx unit.Context, s unit.Sense) {
	dx, dy, ok := a.desiredDir(s)
	if !ok {
		return
	}
	ctx.Out <- unit.SetFSDirection{UnitID: ctx.ID, VX: dx, VY: dy}
}

// snapshot 记录当前状态（牌库除外）。
func (a *铭刻) snapshot(s unit.Sense) {
	cdLeft := a.cdReadyAt - s.Time
	if cdLeft < 0 {
		cdLeft = 0
	}
	a.rec = record{
		x: a.x, y: a.y, vx: s.Self.VX, vy: s.Self.VY,
		hp: a.hp, maxHP: a.maxHP,
		faceX: s.Self.FaceX, faceY: s.Self.FaceY,
		trace: len(a.traceAt), guard: len(a.guardAt), guardPool: a.guardPool,
		hand:     append([]uint8(nil), a.hand...),
		progress: a.progress,
		cdLeft:   cdLeft,
	}
	a.have = true
}

// applyRecord 铭刻·回溯。两张回溯牌分工不同：
//   - 曜时之旅（handOnly=false）：刷新 buff（层数满寿命、额外血条补回去）+ 回到记录的位置，
//     **但一滴血都不回**——活命全靠长守那条额外血条；
//   - 溯时（handOnly=true）：只回收记录时的手牌（外加抽卡进度与命中抽牌冷却）。
//
// 回溯刷新 buff 是这个角色的核心特色（buff 会过期，读档等于续上），所以曜时之旅
// 每次都把层数重置成满寿命；它不回血，于是「拖沓」只能靠血条本身解决。
func (a *铭刻) applyRecord(ctx unit.Context, s unit.Sense, handOnly bool) {
	if !a.have {
		return
	}
	r := a.rec
	if handOnly {
		a.hand = append(a.hand[:0], r.hand...)
		a.progress = r.progress
		a.cdReadyAt = s.Time + r.cdLeft
		a.dashing = false
		return
	}
	// 只回位置与朝向，不回血量与上限。
	ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: r.x, Y: r.y}
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: inscribedSpeed}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: r.vx, VY: r.vy}
	ctx.Out <- unit.SetFSDirection{UnitID: ctx.ID, VX: r.faceX, VY: r.faceY}
	ctx.Out <- unit.Stun{UnitID: ctx.ID, Hold: false}
	ctx.Out <- unit.ClearMarks{UnitID: ctx.ID}
	// 读回来的 buff 是**满寿命**的——这就是回溯在这套牌里的价值。
	a.traceAt = a.traceAt[:0]
	for i := 0; i < r.trace; i++ {
		a.traceAt = append(a.traceAt, s.Time+buffLife)
	}
	if r.trace > 0 {
		ctx.Out <- unit.StackMark{UnitID: ctx.ID, Kind: markTrace, Delta: r.trace, Icon: iconTrace}
	}
	a.guardAt = a.guardAt[:0]
	a.guardPool = r.guardPool
	for layer := 0; layer < r.guard; layer++ {
		a.guardAt = append(a.guardAt, s.Time+buffLife)
	}
	ctx.Out <- unit.SetAbsorb{UnitID: ctx.ID, Amount: a.guardPool}
	ctx.Out <- unit.SetDamageBonus{UnitID: ctx.ID, Amount: float64(r.trace) * traceStep}
	a.x, a.y = r.x, r.y
	a.dashing = false
}
