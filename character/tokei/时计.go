// 时计：白色时钟，表针就是资源条。
// 每 8s 时针 +2h；仅当读数落在 2/4/6/8/10/12 且该刻度本场从未落下过时刻下时刻。
// 无常态弧；敌方踩进时刻判定圈才瞬移冲撞并短窗挂弧。致命伤才回拨（表针 −2h）。
// 时刻永久耗尽时，本体播放 3 秒死亡演出后用既有 Despawn 指令退场。
// 单位：距离 px、速度 px/s、时间 s、角度 度（unit.Deg 转弧度）、伤害 HP 点。
package 时计

import (
	"embed"
	"math"

	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const (
	KindTokei       = "时计"
	KindTokeiArc    = "时计弧"
	KindTokeiMoment = "时计时刻"
)

const (
	// —— 本体 ——
	tokeiRadius = 18.0      // 标准盘
	tokeiSpeed  = 170.0     // 中庸巡航：它的位移资源在时刻网络里，腿不能抢戏
	tokeiHP     = 100.0     // 标准血
	tokeiVision = 9999.0    // 陷阱要看全场踩影
	tokeiColor  = "#f2f2f2" // 白盘，黑针画在 fx 皮肤里

	// —— 踩影临时弧（小骑士同构，伤降 1；仅冲刺窗口内挂）——
	arcSpan   = 120.0 // 度
	arcDmg    = 8.0
	arcCD     = 0.1
	arcInner  = tokeiRadius
	arcOuter  = tokeiRadius + 2
	arcWindow = 0.5 // 踩影触发后挂弧这么久

	// —— 时刻（存档）——
	saveGap     = 8.0 // 存档节拍；开场先等满一次
	handStep    = 2   // 每拍时针 +2h；第一档记 2:00
	retireDelay = 3.0 // 永久耗尽后的死亡演出
	handCap     = 12  // 走到 12:00 停针

	// —— 陷阱 ——
	tripR       = 22.0  // 时刻判定圈半径（比本体 18 略大）= 时刻实体半径
	tripSilence = 3.0   // 同一时刻被踩后静默这么久，圈边反复进出钓不到第二下
	ramSpeed    = 350.0 // 撞击冲刺（小骑士 ramSpeed 同款）
	ramGap      = 60.0  // 落脚留的跑道（小骑士 beside 同款）
)

func init() {
	p := unit.NewPack(KindTokei, assets)
	p.Register(unit.Spec{
		Kind:    KindTokei,
		Role:    unit.RoleFighter,
		Radius:  tokeiRadius,
		MaxHP:   tokeiHP,
		Speed:   tokeiSpeed,
		Vision:  tokeiVision,
		Fighter: true,
		Look:    unit.Look{Color: tokeiColor, FX: []string{"tokei"}},
	}, func(unit.SpawnInfo) unit.Actor { return &时计{} })
	// 临时弧：仅踩影窗口内 Attach；撞到敌方战斗机/活随从就出伤并自毁。
	p.Register(unit.Spec{
		Kind:     KindTokeiArc,
		Role:     unit.RoleProjectile,
		Radius:   arcOuter,
		MaxHP:    1,
		Speed:    tokeiSpeed,
		Vision:   0,
		Attach:   true,
		ArcSpan:  unit.Deg(arcSpan),
		ArcInner: arcInner,
		Look:     unit.Look{Color: tokeiColor, Overlay: true},
	}, func(info unit.SpawnInfo) unit.Actor { return &时计弧{slot: info.Slot} })
	// 时刻：钉在存档位置的透明 Helper。Nonsolid 所以不实心、不撞人、不进 Hittable（不可瞄准、不吃伤）；
	// 不带 VisionBlock（不挡任何人索敌）；AimPriority 留 0（Seek 索不到）。半径就是判定圈 22。
	p.Register(unit.Spec{
		Kind:     KindTokeiMoment,
		Role:     unit.RoleHelper,
		Radius:   tripR,
		MaxHP:    1,
		Speed:    0,
		Vision:   0,
		Nonsolid: true,
		Look:     unit.Look{Color: tokeiColor, FX: []string{"tokei-moment"}},
	}, func(info unit.SpawnInfo) unit.Actor { return 时刻{} })
}

// moment 是一个档：刻档那一刻能看见的五件套（位置/血/速度/朝向/Marks）+ 钟点 + 陷阱用的圈内状态。
type moment struct {
	hour     int // 落下时的钟点：2/4/6/8/10/12
	x, y     float64
	vx, vy   float64
	hp       float64
	maxHP    float64
	fx, fy   float64 // Face 方向
	marks    []unit.Mark
	pin      uint64  // 认领到的时刻实体 ID；0 = 还没认上
	inCircle bool    // 上一拍判定圈里有没有鱼（边沿触发用）
	silent   float64 // 静默到这个时刻
}

type 时计 struct {
	moments []moment     // 档栈，末尾是最近的档（回拨弹栈顶）
	carved  map[int]bool // 本场已成功落下的钟点；用掉/被偷也不释放
	hand    int          // 已提交的表针读数（小时）
	stopped bool         // 走到 12:00 后停针
	nextAt  float64      // 下一次推针的绝对时刻
	booted  bool

	pending  float64 // 本拍已确认、Sense 还没带回来的伤，用来算致命
	hp       float64 // 最近一次 Sense 的血量
	x, y     float64
	slot     int
	arcUntil float64 // 踩影临时弧窗口截止
	arc      unit.AttachState
	retiring bool
	retireAt float64 // 时计私有的退场截止时间，不扩展公共快照
}

type 时计弧 struct{ slot int }

func (a *时计弧) Handle(ctx unit.Context, ev unit.Event) {
	e, ok := ev.(unit.Collision)
	if !ok || !unit.EnemyTarget(e, a.slot) {
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: arcDmg}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

// 时刻：一块钉住不动的透明表盘，自己没有逻辑（被踩由主人扫 Nearby 判定）。
type 时刻 struct{}

func (时刻) Handle(unit.Context, unit.Event) {}

func (a *时计) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		a.hurt(ctx, e)
	case unit.Sense:
		if a.retiring {
			a.deathTick(ctx, e.Time)
		} else {
			a.tick(ctx, e)
		}
	case unit.Stunned:
		if a.retiring {
			ctx.Out <- unit.Stun{UnitID: ctx.ID, Hold: false}
			a.deathTick(ctx, e.Time)
		}
	}
}

func (a *时计) tick(ctx unit.Context, s unit.Sense) {
	a.hp, a.x, a.y = s.Self.HP, s.Self.X, s.Self.Y
	a.pending = 0
	a.slot = s.Self.Slot
	if a.carved == nil {
		a.carved = map[int]bool{}
	}
	if !a.booted {
		a.booted = true
		a.nextAt = s.Time + saveGap // 开场先等满一次 8s
		a.syncFx(ctx)
	}
	a.bindPins(ctx, s)
	if a.retireIfExhausted(ctx, s.Time) {
		return
	}
	a.save(ctx, s)
	a.trap(ctx, s)
	a.armArc(ctx, s) // 踩影之后再挂，触发当帧就能出第一发弧
}

// armArc 仅在踩影临时窗口内按 CD 挂弧；窗口外拆掉残留弧。
func (a *时计) armArc(ctx unit.Context, s unit.Sense) {
	if s.Time+1e-9 < a.arcUntil {
		if unit.RearmAttach(s, ctx.ID, KindTokeiArc, arcCD, &a.arc) {
			unit.SpawnAttach(ctx, s, KindTokeiArc)
		}
		return
	}
	if unit.HasOwned(s, ctx.ID, KindTokeiArc) {
		ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: KindTokeiArc}
	}
	a.arc = unit.AttachState{}
}

// save 8s 节拍：先推时间再尝试留时刻。已用刻度空过；走到 12:00 停针。
// 眩晕期间引擎停发 Sense，所以这条节拍自然暂停——档永远是清醒时拍的照。
func (a *时计) save(ctx unit.Context, s unit.Sense) {
	if a.stopped || s.Time+1e-9 < a.nextAt {
		return
	}
	a.nextAt = s.Time + saveGap
	a.hand += handStep
	if a.hand > handCap {
		a.hand = handCap
	}
	if !a.carved[a.hand] {
		a.carved[a.hand] = true
		m := moment{
			hour: a.hand,
			x:    s.Self.X, y: s.Self.Y,
			vx: s.Self.VX, vy: s.Self.VY,
			hp: s.Self.HP, maxHP: s.Self.MaxHP,
			fx: s.Self.FaceX, fy: s.Self.FaceY,
			marks: append([]unit.Mark(nil), s.Self.Marks...),
		}
		a.moments = append(a.moments, m)
		ctx.Out <- ctx.Born(unit.Spawn{
			Kind: KindTokeiMoment, X: m.x, Y: m.y,
			OwnerID: ctx.ID, Slot: s.Self.Slot,
		})
		ctx.Out <- unit.FX{
			Name: "carve", Kind: ctx.Kind, UnitID: ctx.ID, Slot: s.Self.Slot,
			Amount: float64(m.hour), X: m.x, Y: m.y,
		}
	}
	if a.hand >= handCap {
		a.stopped = true
	}
	a.syncFx(ctx)
}

// pinRef 是扫到的一个时刻实体：认领档位时按位置就近配对。
type pinRef struct {
	id   uint64
	x, y float64
}

// bindPins 事后认领自己的时刻实体（Spawn 出门那一刻拿不到 ID，只好像火山绑熔岩摊那样靠位置认）。
// 顺手处理被偷走的档：实体没了就丢档——档池少一、那个回拨锚点直接没了（钟点仍算已刻，永不回补）。
func (a *时计) bindPins(ctx unit.Context, s unit.Sense) {
	var live []pinRef
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind != KindTokeiMoment || o.OwnerID != s.Self.ID {
			continue
		}
		live = append(live, pinRef{id: o.ID, x: o.X, y: o.Y})
	}
	lost := false
	if len(a.moments) > 0 {
		kept := a.moments[:0]
		for _, m := range a.moments {
			if m.pin != 0 && !hasPin(live, m.pin) {
				lost = true
				continue
			}
			kept = append(kept, m)
		}
		a.moments = kept
	}
	taken := map[uint64]bool{}
	for _, m := range a.moments {
		if m.pin != 0 {
			taken[m.pin] = true
		}
	}
	for i := range a.moments {
		if a.moments[i].pin != 0 {
			continue
		}
		best, bestD := uint64(0), math.MaxFloat64
		for _, p := range live {
			if taken[p.id] {
				continue
			}
			d := math.Hypot(p.x-a.moments[i].x, p.y-a.moments[i].y)
			if d < bestD {
				best, bestD = p.id, d
			}
		}
		if best != 0 {
			a.moments[i].pin = best
			taken[best] = true
			lost = true // 认领也要补发一次表针（残影的表针停在它刻下时的时间）
		}
	}
	if lost {
		a.syncFx(ctx)
	}
}

func hasPin(live []pinRef, id uint64) bool {
	for _, p := range live {
		if p.id == id {
			return true
		}
	}
	return false
}

// syncFx 把表针与每个残影的针位告诉前端：Amount = 当前表针小时，VX = 停针/满仓标记。
// 残影的 FX 挂在实体自己的 UnitID 上，前端按 ID 存每个残影停的时间。
func (a *时计) syncFx(ctx unit.Context) {
	full := 0.0
	if a.stopped {
		full = 1
	}
	// 停针后窗口留空（to<=from）：前端 phase=0，针停在 12:00
	winFrom, winTo := a.nextAt-saveGap, a.nextAt
	if a.stopped {
		winFrom, winTo = 0, 0
	}
	ctx.Out <- unit.FX{
		Name: "clock", Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot,
		// Amount = 已提交读数；VX = 停针；X/Y = 本拍窗口
		Amount: float64(a.hand), VX: full,
		X: winFrom, Y: winTo,
	}
	for i := range a.moments {
		if a.moments[i].pin == 0 {
			continue
		}
		// X/Y 带这一档的静默窗口：踩影后前端靠它把表盘压暗、把时针从上一小时
		// 慢慢拨回这个时刻。窗口外 X/Y 都是 0，前端按正常残影画。
		coolFrom, coolTo := 0.0, 0.0
		if a.moments[i].silent > 0 {
			coolFrom = a.moments[i].silent - tripSilence
			coolTo = a.moments[i].silent
		}
		ctx.Out <- unit.FX{
			Name: "moment", Kind: ctx.Kind, UnitID: a.moments[i].pin, Slot: a.slot,
			Amount: float64(a.moments[i].hour),
			X:      coolFrom, Y: coolTo,
		}
	}
}

// hurt 两段式伤害的拦截层。回拨只吃致命伤：有档就回拨 + BlockDamage 整包取消；
// 档空了就照常吃下来——回拨保不了没档的人。
func (a *时计) hurt(ctx unit.Context, d unit.IncomingDamage) {
	if a.retiring {
		unit.BlockHit(ctx, d)
		return
	}
	if a.hp-a.pending-d.Amount <= 1e-9 && len(a.moments) > 0 {
		a.rewind(ctx, d.Time)
		unit.BlockHit(ctx, d)
		return
	}
	a.pending += d.Amount
	unit.ConfirmHit(ctx, d)
}

// rewind 回拨到最近的档（栈顶弹出）。结算顺序照设计稿：
// 位置 → 血 → 速度 → 朝向 → Marks（先清当下的、再装回档里的）→ 解控 → 用掉这个时刻 → 表针 −2h。
func (a *时计) rewind(ctx unit.Context, now float64) {
	if len(a.moments) == 0 {
		return
	}
	m := a.moments[len(a.moments)-1]
	a.moments = a.moments[:len(a.moments)-1]
	// 表针按已提交读数 −2h（回拨表意）；下一拍仍 +2h，已用刻度空过。
	// 压测：committed=4、窗口正中显示 ~5、档顶是 2 → 针到 2，再经 4 空过 → 6。
	a.hand -= handStep
	if a.hand < 0 {
		a.hand = 0
	}
	a.stopped = false
	ctx.Out <- unit.FX{
		Name: "rewind", Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot,
		X: a.x, Y: a.y, VX: m.x, VY: m.y, Amount: float64(a.hand),
	}
	ctx.Out <- unit.FX{
		Name: "revive", Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot,
		X: m.x, Y: m.y,
	}
	ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: m.x, Y: m.y}
	ctx.Out <- unit.SetHP{UnitID: ctx.ID, HP: m.hp, MaxHP: m.maxHP}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: m.vx, VY: m.vy}
	// 引擎没有「直接改 Face」的指令：速度回档本身带着朝向回来（写速度会顺手 aimFace），
	// 静止档再补一条 SetFSDirection，把巡航朝向也摆回档里，下次起步就朝那边。
	ctx.Out <- unit.SetFSDirection{UnitID: ctx.ID, VX: m.fx, VY: m.fy}
	// 当下的眩晕解除：档是清醒时拍的，快照里没有的东西回拨后不该有（推论：回拨附带解控）
	ctx.Out <- unit.Stun{UnitID: ctx.ID, Hold: false}
	ctx.Out <- unit.ClearMarks{UnitID: ctx.ID}
	for _, mk := range m.marks {
		if mk.Stacks <= 0 {
			continue
		}
		ctx.Out <- unit.StackMark{UnitID: ctx.ID, Kind: mk.Kind, Delta: mk.Stacks, Icon: mk.Icon}
	}
	// 用掉这一档：只拆这一个实体。DespawnOwned 会把整池时刻一起清掉，那是错的。
	if m.pin != 0 {
		ctx.Out <- unit.Despawn{UnitID: m.pin}
	}
	a.pending = 0
	a.hp, a.x, a.y = m.hp, m.x, m.y
	a.nextAt = now + saveGap // 重新吃满 8s 节拍，下一拍从 hand+2 走
	a.syncFx(ctx)
	a.retireIfExhausted(ctx, now)
}

// trap 踩影：敌方战斗机或敌方活随从「进入」某个时刻的判定圈（边沿触发）就把时计拉过去撞。
// 弹体不在 Hittable 里（朝弹撞毫无意义）；同一时刻被踩后静默 3s；时计眩晕时收不到 Sense，自然不触发。
// 一拍只拉一次，同时踩中多个就按档位顺序取最先的那个。触发后打开临时弧窗口。
func (a *时计) trap(ctx unit.Context, s unit.Sense) {
	for i := range a.moments {
		m := &a.moments[i]
		bait, ok := a.bait(s, *m)
		entered := ok && !m.inCircle
		m.inCircle = ok
		if !entered || s.Time+1e-9 < m.silent {
			continue
		}
		m.silent = s.Time + tripSilence
		a.syncFx(ctx)
		dx, dy := bait.X-m.x, bait.Y-m.y
		if math.Hypot(dx, dy) < 1e-6 {
			dx, dy = 1, 0 // 鱼正好压在档位上：没方向可用，随便朝一边顶
		}
		// 落脚点：不能直接落在档位中心。踩圈的人就站在档位旁边，两边半径 18 一重叠，
		// 引擎当场把这对单位弹开、冲撞速度被反弹覆盖（弧的朝向跟着速度走，于是"根本撞不到"）。
		// 所以沿「档位→鱼」这条线退到刚好够得着之外（留 ramGap 的空隙，小骑士 beside 同款），
		// 档位只当锚点；冲撞方向再从落脚点重新指向鱼。
		sx, sy := a.ramStart(s.Field, bait, dx, dy, *m)
		vx, vy := bait.X-sx, bait.Y-sy
		vn := math.Hypot(vx, vy)
		if vn < 1e-6 {
			vx, vy, vn = 1, 0, 1
		}
		ux, uy := vx/vn, vy/vn
		ctx.Out <- unit.FX{
			Name: "trip", Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot,
			X: m.x, Y: m.y, VX: ux, VY: uy,
		}
		ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: sx, Y: sy}
		// 撞击本身无伤，出伤靠瞬过去之后的临时弧
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: ux * ramSpeed, VY: uy * ramSpeed}
		a.arc = unit.AttachState{} // 窗口一开立刻能挂第一发
		a.arcUntil = s.Time + arcWindow
		return
	}
}

// ramStart 挑落脚点：优先站在档位那一侧（贴回档位，观众看起来还是"被拉到那个时刻"），
// 那里不可行走就绕着鱼转圈找（同小骑士 beside 的兜底），都不行才退回档位中心。
func (a *时计) ramStart(f unit.Field, bait unit.Snapshot, mx, my float64, m moment) (float64, float64) {
	base := math.Atan2(my, mx) // 档位 → 鱼 的方向角
	clearance := tokeiRadius + bait.Radius + ramGap
	for _, off := range []float64{0, 30, -30, 60, -60, 90, -90, 150, -150, 180} {
		ang := base + unit.Deg(off)
		x, y := bait.X-math.Cos(ang)*clearance, bait.Y-math.Sin(ang)*clearance
		if f.Walkable(x, y, tokeiRadius+4) {
			return x, y
		}
	}
	return m.x, m.y
}

// bait 找判定圈里最近的鱼。圈是「两个圆真的碰到」：圆心距 ≤ tripR + 目标半径。
func (a *时计) bait(s unit.Sense, m moment) (unit.Snapshot, bool) {
	var best unit.Snapshot
	found := false
	bestD := 0.0
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		d := math.Hypot(o.X-m.x, o.Y-m.y)
		if d > tripR+o.Radius {
			continue
		}
		if !found || d < bestD {
			best, bestD, found = *o, d, true
		}
	}
	return best, found
}

// 永久耗尽看完整的刻度账，不看当前针位：中途用光仍保留空走机制。
func (a *时计) retireIfExhausted(ctx unit.Context, now float64) bool {
	if a.retiring {
		return true
	}
	if len(a.moments) > 0 {
		return false
	}
	for hour := handStep; hour <= handCap; hour += handStep {
		if !a.carved[hour] {
			return false
		}
	}
	a.retiring = true
	a.retireAt = now + retireDelay
	a.arcUntil = 0
	a.arc = unit.AttachState{}
	ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: KindTokeiArc}
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
	ctx.Out <- unit.SetAimPriority{From: ctx.ID, UnitID: ctx.ID, Value: 0}
	ctx.Out <- unit.NoHealthNumbers{UnitID: ctx.ID, Hold: true}
	a.deathTick(ctx, now)
	return true
}

// 演出用时计现有的 FX 通道传窗口；每拍重发同一窗口，丢帧也不会重开计时。
// 前端与导出都按快照的模拟时间算针位，不引入公共协议或真实时间定时器。
func (a *时计) deathTick(ctx unit.Context, now float64) {
	if now+1e-9 >= a.retireAt {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID}
	ctx.Out <- unit.SetHP{UnitID: ctx.ID, HP: 0}
	ctx.Out <- unit.FX{
		Name: "overclock", Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot,
		X: a.retireAt - retireDelay, Y: a.retireAt,
	}
}
