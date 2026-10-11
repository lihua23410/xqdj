// 时计：白色时钟，表针就是资源条。
// 每 5s 把「当下的自己」刻成一个时刻（最多 8 个 = 24:00）；累计 25 点确认伤、或吃到致命伤时回拨到最近一个时刻；
// 敌方踩进时刻判定圈就把时计拉过去撞。全套只用现有引擎指令拼，零引擎改动。
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
	tokeiHP     = 100.0     // 标准血：25 伤 = 四分之一血一个档
	tokeiVision = 9999.0    // 陷阱要看全场踩影
	tokeiColor  = "#f2f2f2" // 白盘，黑针画在 fx 皮肤里

	// —— 常态弧（小骑士同构，伤降 1）——
	arcSpan  = 120.0 // 度
	arcDmg   = 8.0   // 小骑士 9 伤降 1：它有保命系统，输出让半步
	arcCD    = 0.1
	arcInner = tokeiRadius
	arcOuter = tokeiRadius + 2

	// —— 时刻（存档）——
	saveGap  = 5.0 // 存档节拍；开场先等满一次
	saveCap  = 6   // 6 档封顶 = 12:00（表针正好转满一圈，与 0:00 重合，前端另有满仓区分）
	handStep = 2   // 每刻一档，时针 +2h —— 而且是**先推时间再留下时刻**，所以第一档记的是 2:00

	// —— 回拨 ——
	revertAt = 25.0 // 累计确认伤到 25 就回拨；一次伤害只回拨一次

	// —— 陷阱 ——
	tripR       = 22.0  // 时刻判定圈半径（比本体 18 略大）= 时刻实体半径
	tripSilence = 3.0   // 同一时刻被踩后静默这么久，圈边反复进出钓不到第二下
	ramSpeed    = 350.0 // 撞击冲刺（小骑士 ramSpeed 同款）
	ramGap      = 60.0  // 落脚留的跑道：踩圈的人就在档位旁边，贴上去的瞬间不能和它重叠（小骑士 beside 同款），
	//                     但只留 8px 的话"撞"根本看不见——留出 60px 才有冲过去这一段。
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
	// 常态弧：Attach 每帧贴主人，撞到敌方战斗机/活随从就出伤并自毁（同小骑士弧）。
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
	}, func(unit.SpawnInfo) unit.Actor { return 时刻{} })
}

// moment 是一个档：刻档那一刻能看见的五件套（位置/血/速度/朝向/Marks）+ 陷阱用的圈内状态。
type moment struct {
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
	moments []moment // 档栈，末尾是最近的档（回拨弹栈顶）
	nextAt  float64  // 下一次刻档的绝对时刻
	booted  bool

	dealt   float64 // 本档周期内累计的确认伤，回拨后清零
	pending float64 // 本拍已确认、Sense 还没带回来的伤，用来算致命
	hp      float64 // 最近一次 Sense 的血量
	x, y    float64
	slot    int

	arc unit.AttachState
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
		a.tick(ctx, e)
	}
}

func (a *时计) tick(ctx unit.Context, s unit.Sense) {
	a.hp, a.x, a.y = s.Self.HP, s.Self.X, s.Self.Y
	a.pending = 0
	a.slot = s.Self.Slot
	if !a.booted {
		a.booted = true
		a.nextAt = s.Time + saveGap // 开场先等满一次 5s
		a.syncFx(ctx)
	}
	// 常态弧：和主人同款 CD 循环
	if unit.RearmAttach(s, ctx.ID, KindTokeiArc, arcCD, &a.arc) {
		unit.SpawnAttach(ctx, s, KindTokeiArc)
	}
	a.bindPins(ctx, s)
	a.save(ctx, s)
	a.trap(ctx, s)
}

// save 5s 节拍到点就刻一档。满仓时不刻，但节拍照走（空出档位后要重新吃满 5s）。
// 眩晕期间引擎停发 Sense，所以这条节拍自然暂停——档永远是清醒时拍的照。
func (a *时计) save(ctx unit.Context, s unit.Sense) {
	if s.Time+1e-9 < a.nextAt {
		return
	}
	a.nextAt = s.Time + saveGap
	if len(a.moments) >= saveCap {
		return
	}
	m := moment{
		x: s.Self.X, y: s.Self.Y,
		vx: s.Self.VX, vy: s.Self.VY,
		hp: s.Self.HP, maxHP: s.Self.MaxHP,
		fx: s.Self.FaceX, fy: s.Self.FaceY,
		marks: append([]unit.Mark(nil), s.Self.Marks...),
	}
	a.moments = append(a.moments, m)
	// 实体钉在档位；VX/VY 留 0，它不该动
	ctx.Out <- unit.Spawn{
		Kind: KindTokeiMoment, X: m.x, Y: m.y,
		OwnerID: ctx.ID, Slot: s.Self.Slot,
	}
	a.syncFx(ctx)
}

// pinRef 是扫到的一个时刻实体：认领档位时按位置就近配对。
type pinRef struct {
	id   uint64
	x, y float64
}

// bindPins 事后认领自己的时刻实体（Spawn 出门那一刻拿不到 ID，只好像火山绑熔岩摊那样靠位置认）。
// 顺手处理被偷走的档：实体没了就丢档——档池少一、那个回拨锚点直接没了。
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

// syncFx 把表针与每个残影的针位告诉前端：Amount = 小时（档数 × 2），VX = 满仓标记。
// 残影的 FX 挂在实体自己的 UnitID 上，前端按 ID 存每个残影停的时间。
func (a *时计) syncFx(ctx unit.Context) {
	full := 0.0
	if len(a.moments) >= saveCap {
		full = 1
	}
	ctx.Out <- unit.FX{
		Name: "clock", Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot,
		// Amount = 档数 × 2（当前表针读数）、VX = 满仓标记；
		// X/Y = 这一拍窗口的起止模拟时刻——前端拿它把时针从当前读数连续推到下一拍读数（正点落在刻档那一瞬，
		// 也就是「先推时间再留时刻」），分针则一拍正好一圈。
		Amount: float64(len(a.moments) * handStep), VX: full,
		X: a.nextAt - saveGap, Y: a.nextAt,
	}
	for i := range a.moments {
		if a.moments[i].pin == 0 {
			continue
		}
		ctx.Out <- unit.FX{
			Name: "moment", Kind: ctx.Kind, UnitID: a.moments[i].pin, Slot: a.slot,
			// 先推时间再留时刻：第一档记的是 2:00、第二档 4:00……最后一档 12:00
			Amount: float64((i + 1) * handStep),
		}
	}
}

// hurt 两段式伤害的拦截层。两个回拨扳机都在这儿：
//  1. 一次致命伤（当前血 − 这段伤 ≤ 0）：有档就回拨 + BlockDamage 整包取消（不进 25 伤计数）；
//     档空了就照常吃下来——回拨保不了没档的人。
//  2. 累计 25 点确认伤：确认完这一下就回拨。被 Block、被闪、减到 0 的报价压根到不了这里（引擎不发）。
func (a *时计) hurt(ctx unit.Context, d unit.IncomingDamage) {
	if a.hp-a.pending-d.Amount <= 1e-9 && len(a.moments) > 0 {
		a.rewind(ctx, d.Time)
		unit.BlockHit(ctx, d)
		return
	}
	a.pending += d.Amount
	unit.ConfirmHit(ctx, d)
	a.dealt += d.Amount
	if a.dealt+1e-9 >= revertAt {
		// 过门槛就清零，不管这次能不能真回拨：没档时 rewind 会直接返回，
		// 计数要是留成 ≥25，5s 后新刻的那一档会被上一轮的账当场花掉。
		a.dealt = 0
		a.rewind(ctx, d.Time)
	}
}

// rewind 回拨到最近的档（栈顶弹出）。结算顺序照设计稿：
// 位置 → 血 → 速度 → 朝向 → Marks（先清当下的、再装回档里的）→ 解控 → 用掉这个时刻 → 计数器清零。
func (a *时计) rewind(ctx unit.Context, now float64) {
	if len(a.moments) == 0 {
		return
	}
	m := a.moments[len(a.moments)-1]
	a.moments = a.moments[:len(a.moments)-1]
	hours := float64(handStep * len(a.moments)) // 回拨后的表针：正是剩下最后一档记的时间
	ctx.Out <- unit.FX{
		Name: "rewind", Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot,
		X: a.x, Y: a.y, VX: m.x, VY: m.y, Amount: hours,
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
	a.dealt = 0
	a.pending = 0
	a.hp, a.x, a.y = m.hp, m.x, m.y
	a.nextAt = now + saveGap // 档空出来了，重新吃满 5s 节拍
	a.syncFx(ctx)
}

// trap 踩影：敌方战斗机或敌方活随从「进入」某个时刻的判定圈（边沿触发）就把时计拉过去撞。
// 弹体不在 Hittable 里（朝弹撞毫无意义）；同一时刻被踩后静默 3s；时计眩晕时收不到 Sense，自然不触发。
// 一拍只拉一次，同时踩中多个就按档位顺序取最先的那个。
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
		dx, dy := bait.X-m.x, bait.Y-m.y
		if math.Hypot(dx, dy) < 1e-6 {
			dx, dy = 1, 0 // 鱼正好压在档位上：没方向可用，随便朝一边顶
		}
		// 落脚点：不能直接落在档位中心。踩圈的人就站在档位旁边，两边半径 18 一重叠，
		// 引擎当场把这对单位弹开、冲撞速度被反弹覆盖（弧的朝向跟着速度走，于是"根本撞不到"）。
		// 所以沿「档位→鱼」这条线退到刚好够得着之外（留 ramGap 的空隙，小骑士 beside 同款），
		// 档位只当锚点；冲撞方向再从落脚点重新指向鱼。
		sx, sy := a.ramStart(bait, dx, dy, *m)
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
		// 撞击本身无伤，出伤靠瞬过去之后的贴身弧；之后巡航带子自然接管
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: ux * ramSpeed, VY: uy * ramSpeed}
		return
	}
}

// ramStart 挑落脚点：优先站在档位那一侧（贴回档位，观众看起来还是"被拉到那个时刻"），
// 那里不可行走就绕着鱼转圈找（同小骑士 beside 的兜底），都不行才退回档位中心。
func (a *时计) ramStart(bait unit.Snapshot, mx, my float64, m moment) (float64, float64) {
	base := math.Atan2(my, mx) // 档位 → 鱼 的方向角
	clearance := tokeiRadius + bait.Radius + ramGap
	f := unit.LiveField()
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
