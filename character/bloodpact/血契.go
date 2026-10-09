// 血契：以血为食、以血为薪的近战寄生者。技能烧自己的血（饿扑/收线/血蝠），
// 寄生吸血是唯一的回血——要么吃掉对手，要么流血流干。
//
// 状态机：
//
//	饥饿 ──饿扑(烧5血)──> 咬合建线 ──> 寄生(回流) ──断线──> 蔫置3s ──> 饥饿
//	  │                                    │
//	  └──熔断(HP≤10)：耗血技能全熄，只剩基础弧──┘
//
// 咬合是 Attach 弧单位（小骑士/地慧星那套薄扇环）：常驻挂在身前，打到人就散，
// RearmAttach 按 CD 再挂。弧散那一拍若眼前还站着敌方战斗机就建血线
// （锤击「弧消失那一帧」的同款口径）。
package 血契

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

const KindBloodPact = "血契"

// 数值常量集中在 init 上面，方便调平衡。
const (
	pactRadius = 18.0
	pactHP     = 100.0
	pactSpeed  = 170.0
	pactVision = 9999.0
	pactColor  = "#8c1420" // 暗红

	// 咬合：Attach 薄扇环（地慧星口径）：内 18 / 外 20、110°。
	biteDamage   = 7.0
	biteCD       = 0.30
	biteCDFeed   = 0.25 // 寄生期更快
	biteArcSpan  = 110.0
	biteArcInner = pactRadius
	biteArcOuter = pactRadius + 2
	biteSlack    = 8.0 // 咬合余量：外径 + 对方半径 + 这个，咬距上被挤开一点也算「在嘴边」
	mouthReach   = biteArcOuter + 18 + biteSlack // 嘴边：46。建线量这个，收线起手也量这个

	// 饿扑：烧 5 血，朝索敌目标 400 速扑 0.5 秒。扑空 = 白烧。
	pounceCost  = 5.0
	pounceCD    = 6.0
	pounceSpeed = 400.0
	pounceDur   = 0.5

	// 熔断线：HP ≤ 10 时耗血技能全熄（放招那一刻判定）。所以永远不会烧死自己。
	blackoutHP = 10.0

	// 血线：距离 > 224——场地直径 560 的 2/5——绷断；断线后蔫置 3 秒，巡航 −30，不能重连。
	lineRange = 320.0
	wiltDur   = 3.0
	wiltDrop  = 30.0

	// 寄生回流：按连接时长分档 25% / 50% / 75%。
	feedMidAt   = 5.0
	feedDeepAt  = 10.0
	feedShallow = 0.25
	feedMid     = 0.50
	feedDeep    = 0.75
	feedStep    = 0.5 // 攒够这么多才 Heal 一次（引擎口径：≥0.5 才有治疗特效）

	// 收线：烧 8 血，把敌人朝自己强拉 1.4 秒。拉力压过敌人自身速度，没有抗性；
	// 墙由物理碰撞免费涌现（顶在墙上就是卡住了）。拉到 38 就提前结束。
	// 起手要等对方真的拉开（> mouthReach）：Attach 弧咬人会把对方从咬距上
	// 挤开一点，若 >38 就起手，贴脸互殴会每 3 秒白烧 8 血去拉那一像素。
	reelCost    = 8.0
	reelCD      = 3.0
	reelDur     = 1.4
	reelSpeed   = 60.0
	reelStopGap = 38.0

	// 血蝠：烧 6 血、CD 5s、一次两只、场上至多 3 只。活随从，追着人咬（3 伤、
	// 0.5s CD），6 秒寿终。它咬敌方战斗机的伤走血线回流（来源不挑），
	// 所以蝙蝠是在外勤打工、把血抽回总部；蝙蝠自己挨打的伤不回流（随从不是契约方）。
	batCost      = 6.0
	batCD        = 5.0
	batPerCall   = 1
	batCap       = 3
	batRadius    = 8.0
	batHP        = 10.0
	batSpeed     = 190.0
	batDamage    = 3.0
	batBiteCD    = 0.5
	batLife      = 6.0
	batSeekEvery = 0.4
)

// KindBloodArc 是咬合那截 Attach 薄扇环的类名（打中即散，RearmAttach 再挂）。
const KindBloodArc = "血契弧"

// KindBloodBat 是烧血召唤的小蝙蝠（活随从）的类名。
const KindBloodBat = "血契蝠"

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindBloodPact, assets)
	p.Register(unit.Spec{
		Kind:    KindBloodPact,
		Role:    unit.RoleFighter,
		Radius:  pactRadius,
		MaxHP:   pactHP,
		Speed:   pactSpeed,
		Vision:  pactVision,
		Fighter: true,
		Look:    unit.Look{Color: pactColor, FX: []string{"bloodpact"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &血契{}
	})

	// 咬合弧：贴身薄扇环（Attach，小骑士/地慧星同款）。碰到敌方战斗机或
	// 敌方活随从结 7、闪一口、就地散架；主人按 CD 再挂。
	p.Register(unit.Spec{
		Kind:     KindBloodArc,
		Role:     unit.RoleProjectile,
		Radius:   biteArcOuter,
		MaxHP:    1,
		Speed:    pactSpeed,
		Vision:   0,
		Fighter:  false,
		Attach:   true,
		ArcSpan:  unit.Deg(biteArcSpan),
		ArcInner: biteArcInner,
		Look:     unit.Look{Color: pactColor, Overlay: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &血契弧{slot: info.Slot}
	})

	// 血蝠：烧血召出的活随从，追着人咬。它咬的伤对血线回流是「敌方战斗机
	// 受到的确认伤害」，来源不挑——所以蝙蝠是在外勤打工、把血抽回总部。
	// Pass：不与单位碰撞（蛆宝宝互吸同款）——巡航会把它往目标身上顶，
	// 实心身体顶不过引擎推挤，会卡着重叠；穿过去由下嘴判定圈管出伤。
	p.Register(unit.Spec{
		Kind:    KindBloodBat,
		Role:    unit.RoleMinion,
		Radius:  batRadius,
		MaxHP:   batHP,
		Speed:   batSpeed,
		Vision:  pactVision,
		Fighter: false,
		Mortal:  true,
		Look:    unit.Look{Color: pactColor, FX: []string{"bloodpact-bat"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &血契蝠{owner: info.OwnerID, slot: info.Slot}
	})
}

type 血契 struct {
	x, y   float64
	radius float64
	slot   int

	faceX, faceY float64

	fsSeq    uint32
	pounceAt float64
	reelAt   float64
	batAt    float64
	rng      *rand.Rand

	arc      unit.AttachState

	// 血线
	lineID    uint64
	lineSince float64
	lineHP    float64
	lineMax   float64
	wiltUntil float64

	reelUntil float64
	pool      float64
	state     int
	booted    bool
}

func (b *血契) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	if s, ok := ev.(unit.Sense); ok {
		b.onSense(ctx, s)
	}
}

func (b *血契) onSense(ctx unit.Context, s unit.Sense) {
	b.x, b.y, b.radius, b.slot = s.Self.X, s.Self.Y, s.Self.Radius, s.Self.Slot
	if !b.booted {
		b.booted = true
		b.pounceAt = s.Time
		b.reelAt = s.Time
		b.batAt = s.Time
		b.state = -1
	}
	if n := math.Hypot(s.Self.VX, s.Self.VY); n > 1e-6 {
		b.faceX, b.faceY = s.Self.VX/n, s.Self.VY/n
	} else if b.faceX == 0 && b.faceY == 0 {
		b.faceX, b.faceY = 1, 0
	}

	b.endWilt(ctx, s)
	b.updateLine(ctx, s)
	b.checkBiteTether(ctx, s)
	b.rearmBite(ctx, s)
	b.reflux(ctx, s)
	b.tryPounce(ctx, s)
	b.reel(ctx, s)
	b.tryCallBats(ctx, s)
	b.syncLook(ctx, s)
}

// endWilt 蔫置到点：巡航还回去。
func (b *血契) endWilt(ctx unit.Context, s unit.Sense) {
	if b.wiltUntil <= 0 || s.Time+1e-9 < b.wiltUntil {
		return
	}
	b.wiltUntil = 0
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: pactSpeed}
}

func (b *血契) wilted(s unit.Sense) bool {
	return b.wiltUntil > 0 && s.Time+1e-9 < b.wiltUntil
}

// updateLine 血线的断法都在这里：目标没了、或距离超过 90 就绷断并蔫置。
func (b *血契) updateLine(ctx unit.Context, s unit.Sense) {
	if b.lineID == 0 {
		return
	}
	o := findNearby(s.Nearby, b.lineID)
	if o == nil || o.Role != unit.RoleFighter {
		b.breakLine(ctx, s)
		return
	}
	if math.Hypot(o.X-b.x, o.Y-b.y) > lineRange {
		b.breakLine(ctx, s)
	}
}

func (b *血契) breakLine(ctx unit.Context, s unit.Sense) {
	if b.lineID == 0 {
		return
	}
	b.lineID = 0
	b.lineHP, b.lineMax, b.pool, b.reelUntil = 0, 0, 0, 0
	b.wiltUntil = s.Time + wiltDur
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: pactSpeed - wiltDrop}
	ctx.Out <- unit.FX{
		Name: "blood-snap", Kind: ctx.Kind, UnitID: ctx.ID, Slot: b.slot,
		X: b.x, Y: b.y,
	}
}

// tether 建线：已经在线上就不动，深度按连接时长自己长。建线只认敌方战斗机。
func (b *血契) tether(ctx unit.Context, s unit.Sense, o *unit.Snapshot) {
	if b.lineID != 0 || b.wilted(s) {
		return
	}
	b.lineID = o.ID
	b.lineSince = s.Time
	b.lineHP, b.lineMax = o.HP, o.MaxHP
	ctx.Out <- unit.FX{
		Name: "blood-bond", Kind: ctx.Kind, UnitID: ctx.ID, Slot: b.slot,
		X: b.x, Y: b.y, VX: o.X, VY: o.Y,
	}
}

// rearmBite 咬合弧的挂载：打到人就散，按 CD 再挂（寄生期快一点）。
func (b *血契) rearmBite(ctx unit.Context, s unit.Sense) {
	cd := biteCD
	if b.lineID != 0 {
		cd = biteCDFeed
	}
	if unit.RearmAttach(s, ctx.ID, KindBloodArc, cd, &b.arc) {
		unit.SpawnAttach(ctx, s, KindBloodArc)
		// 出生这一拍就把下一次可挂的钟压上：弧若在挂出那一拍就咬到人
		// （敌人已经站在扇区里），引擎让它当拍变成非实心并被 reapAttach 收走，
		// 永远活不进任何一次感知，RearmAttach 于是以为 CD 从未跑过——
		// 表现成重叠目标下每拍重挂一咬。压上钟之后无论弧有没有被感知到，
		// 下一把都得等满这个 CD。
		b.arc.ReadyAt = s.Time + cd
	}
}

// checkBiteTether 咬合够得着的范围里站着敌方战斗机（且没有血线、不在荒置），
// 就算咬上了、就地建血线，不等弧真的散那一拍：Attach 弧的物理碰撞会把
// 对方从咬距上挤开一点，按「等弧散」会把推开的嘴边人漏掉。
func (b *血契) checkBiteTether(ctx unit.Context, s unit.Sense) {
	if b.lineID != 0 || b.wilted(s) {
		return
	}
	fx, fy := b.faceUnit()
	cosHalf := math.Cos(unit.Deg(biteArcSpan / 2))
	var fighter *unit.Snapshot
	bestD := math.MaxFloat64
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role != unit.RoleFighter || !unit.Hittable(*o, b.slot) {
			continue
		}
		dx, dy := o.X-b.x, o.Y-b.y
		d := math.Hypot(dx, dy)
		if d < 1e-6 || d > biteArcOuter+o.Radius+biteSlack {
			continue
		}
		if (dx/d)*fx+(dy/d)*fy < cosHalf-1e-8 {
			continue
		}
		if d < bestD {
			fighter, bestD = o, d
		}
	}
	if fighter != nil {
		b.tether(ctx, s, fighter)
	}
}

func (b *血契) faceUnit() (float64, float64) {
	if n := math.Hypot(b.faceX, b.faceY); n > 1e-6 {
		return b.faceX / n, b.faceY / n
	}
	return 1, 0
}

// reflux 寄生回流：血线连着期间，敌方战斗机掉多少血就按深度回流多少。
// 只认确认伤害（快照里的 HP 差值），HP 回升和上限变化不算。
func (b *血契) reflux(ctx unit.Context, s unit.Sense) {
	if b.lineID == 0 {
		return
	}
	o := findNearby(s.Nearby, b.lineID)
	if o == nil || o.Role != unit.RoleFighter {
		return
	}
	if o.MaxHP != b.lineMax {
		// 上限变了（勇者升级之类）这一拍不算回流，只重新对表。
		b.lineMax, b.lineHP = o.MaxHP, o.HP
		return
	}
	if b.lineHP > 0 && o.HP < b.lineHP-1e-9 {
		b.pool += (b.lineHP - o.HP) * feedRatio(s.Time-b.lineSince)
	}
	b.lineHP = o.HP
	if b.pool >= feedStep {
		ctx.Out <- unit.Heal{UnitID: ctx.ID, Amount: b.pool}
		b.pool = 0
	}
}

func feedRatio(held float64) float64 {
	switch {
	case held >= feedDeepAt:
		return feedDeep
	case held >= feedMidAt:
		return feedMid
	default:
		return feedShallow
	}
}

// tryPounce 饿扑：无血线、不蔫置、不在熔断区时，烧 5 血朝索敌目标猛扑。
func (b *血契) tryPounce(ctx unit.Context, s unit.Sense) {
	if b.lineID != 0 || b.wilted(s) {
		return
	}
	if s.Self.HP <= blackoutHP {
		return
	}
	if s.Time+1e-9 < b.pounceAt {
		return
	}
	target := unit.Seek(s)
	if target == nil {
		return
	}
	ux, uy := unitDir(target.X-b.x, target.Y-b.y)
	b.pounceAt = s.Time + pounceCD
	ctx.Out <- unit.SetHP{UnitID: ctx.ID, HP: s.Self.HP - pounceCost}
	ctx.Out <- unit.AddFS{
		UnitID: ctx.ID, DX: ux, DY: uy, BaseSpeed: pounceSpeed,
		OnWall: true, ExpiresAt: s.Time + pounceDur,
		Token: b.nextFSToken(ctx.ID),
	}
	ctx.Out <- unit.FX{
		Name: "blood-pounce", Kind: ctx.Kind, UnitID: ctx.ID, Slot: b.slot,
		X: b.x, Y: b.y, VX: ux, VY: uy, Amount: pounceCost,
	}
}

// reel 收线：有血线、CD 好、不在熔断区就烧 8 血起手，
// 之后每帧把敌人朝自己强拉，贴到 38 提前结束（剩余时间不拉不烧）。
func (b *血契) reel(ctx unit.Context, s unit.Sense) {
	if b.reelUntil > 0 && s.Time+1e-9 >= b.reelUntil {
		b.reelUntil = 0
	}
	if b.lineID == 0 {
		b.reelUntil = 0
		return
	}
	o := findNearby(s.Nearby, b.lineID)
	if o == nil {
		b.reelUntil = 0
		return
	}
	d := math.Hypot(o.X-b.x, o.Y-b.y)
	if b.reelUntil == 0 {
		// 对方真的拉开了才起手：贴脸时咬合会把人从咬距上挤开一点，
		// >38 就起手会每 3 秒白烧 8 血去拉那一像素。
		if d <= mouthReach {
			return
		}
		if s.Time+1e-9 < b.reelAt || s.Self.HP <= blackoutHP {
			return
		}
		b.reelAt = s.Time + reelCD
		b.reelUntil = s.Time + reelDur
		ctx.Out <- unit.SetHP{UnitID: ctx.ID, HP: s.Self.HP - reelCost}
		ctx.Out <- unit.FX{
			Name: "blood-reel", Kind: ctx.Kind, UnitID: ctx.ID, Slot: b.slot,
			X: b.x, Y: b.y, VX: o.X, VY: o.Y, Amount: reelCost,
		}
	} else if d <= reelStopGap {
		b.reelUntil = 0
		return
	}
	ux, uy := unitDir(b.x-o.X, b.y-o.Y)
	ctx.Out <- unit.SetVelocity{UnitID: o.ID, VX: ux * reelSpeed, VY: uy * reelSpeed}
}

// tryCallBats 血蝠：烧 6 血、CD 5s、场上至多 3 只。蝙蝠是活随从——它咬敌方
// 战斗机的伤走血线回流（来源不挑），所以这笔投资的回报是从敌人身上抽血。
// 和饿扑/收线一样受熔断管：HP ≤ 10 时召不动。
func (b *血契) tryCallBats(ctx unit.Context, s unit.Sense) {
	if s.Self.HP <= blackoutHP {
		return
	}
	if s.Time+1e-9 < b.batAt {
		return
	}
	live := 0
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind == KindBloodBat && o.OwnerID == ctx.ID && o.Role == unit.RoleMinion {
			live++
		}
	}
	if live >= batCap {
		return
	}
	if b.rng == nil {
		b.rng = rand.New(rand.NewPCG(ctx.ID*8817264546332525217, 0x9e3779b97f4a7c15))
	}
	b.batAt = s.Time + batCD
	ctx.Out <- unit.SetHP{UnitID: ctx.ID, HP: s.Self.HP - batCost}
	for i := 0; i < batPerCall; i++ {
		// 落点挑到不压进任何人：出生圈的随机方向若撞上敌方战斗机或活随从
		// （引擎出生位本来就贴得近），当拍的重叠会穿透断言；挪远一圈再放。
		for try := 0; try < 8; try++ {
			ang := b.rng.Float64() * 2 * math.Pi
			ux, uy := math.Cos(ang), math.Sin(ang)
			gap := b.radius + batRadius + 4 + float64(try)*batRadius
			x, y := b.x+ux*gap, b.y+uy*gap
			if circleBlocked(s, x, y, batRadius) {
				continue
			}
			ctx.Out <- unit.Spawn{
				Kind: KindBloodBat, OwnerID: ctx.ID, Slot: b.slot,
				X: x, Y: y,
				VX: ux * batSpeed, VY: uy * batSpeed,
			}
			break
		}
	}
	ctx.Out <- unit.FX{
		Name: "blood-bats", Kind: ctx.Kind, UnitID: ctx.ID, Slot: b.slot,
		X: b.x, Y: b.y, Amount: batCost,
	}
}

// circleBlocked 这个圆会不会压进感知里任何实心单位（自己和自己的弧除外）。
func circleBlocked(s unit.Sense, x, y, r float64) bool {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.ID == s.Self.ID || o.OwnerID == s.Self.ID || o.Nonsolid {
			continue
		}
		if math.Hypot(o.X-x, o.Y-y) < o.Radius+r+2 {
			return true
		}
	}
	return false
}

// syncLook 给页面递外观：血线每帧报两端 + 当前深度；状态变了才报一次状态。
func (b *血契) syncLook(ctx unit.Context, s unit.Sense) {
	if b.lineID != 0 {
		if o := findNearby(s.Nearby, b.lineID); o != nil {
			ctx.Out <- unit.FX{
				Name: "blood-line", Kind: ctx.Kind, UnitID: ctx.ID, Slot: b.slot,
				X: b.x, Y: b.y, VX: o.X, VY: o.Y,
				Amount: float64(b.depth(s.Time)) + 1,
			}
		}
	}
	st := b.stateCode(s)
	if st != b.state {
		b.state = st
		ctx.Out <- unit.FX{
			Name: "blood-state", Kind: ctx.Kind, UnitID: ctx.ID, Slot: b.slot,
			X: b.x, Y: b.y, Amount: float64(st),
		}
	}
}

// depth 血线深度：0 细（25%）、1 粗（50%）、2 深红搏动（75%）。
func (b *血契) depth(now float64) int {
	held := now - b.lineSince
	switch {
	case held >= feedDeepAt:
		return 2
	case held >= feedMidAt:
		return 1
	default:
		return 0
	}
}

// stateCode 0 饥饿、1 寄生、2 蔫置、3 熔断。
func (b *血契) stateCode(s unit.Sense) int {
	switch {
	case s.Self.HP <= blackoutHP:
		return 3
	case b.wilted(s):
		return 2
	case b.lineID != 0:
		return 1
	default:
		return 0
	}
}

func (b *血契) nextFSToken(owner uint64) uint64 {
	b.fsSeq++
	return owner<<32 | uint64(b.fsSeq)
}
