// 弓箭（老兵安柏赫）：没有出众的能力——无成长、无资源、无形态，强度全在**时机**。
// 蓄力射击（站定 0.5s → 一箭 → 后摇 0.75s）、后撤步射击（安全向滑步 90 + 撤中一箭，CD 4s）、
// 齐射（站定 1s → 四支贴排同射 → 后摇 1s）、翻滚（挨报价那一下 Block + 短滚 75，CD 2.5s）。
// 单位：距离 px、速度 px/s、时间 s、角度 度（unit.Deg）、伤害 HP 点。
package 弓箭

import (
	"embed"
	"math"

	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const (
	KindArcher      = "弓箭"
	KindArrow       = "箭"
	KindVolleyArrow = "齐射箭"
)

const (
	// —— 本体（凡人面板）——
	archerRadius = 18.0
	archerSpeed  = 178.0 // 小骑士同级：要主动经营距离，腿不能慢
	archerHP     = 100.0
	archerVision = 9999.0    // 安全向算法要看全场敌人和弹幕
	archerColor  = "#38332f" // 老兵本体：深褐
	arrowColor   = "#f5f2e8" // 箭是白的：深色场上棕色小点根本看不见（Trail 关掉，拖尾自己画）

	// —— 箭 ——
	arrowDmg   = 5.0 // 蓄射 / 后撤步那一箭
	volleyDmg  = 5.0 // 齐射每一支（对心四支全中 = 20）
	arrowSpeed = 1024.0
	arrowR     = 4.0
	arrowStun  = 0.2 // 每支箭 0.2s 眩晕；同目标同帧同值，Stun 直接覆盖 → 天然不叠
	arrowGap   = 2.0 // 出膛点离球面留的余量（同远程原型机：radius + 弹半径 + gap）

	// —— 节奏（秒）——
	chargeStand = 0.5 // 蓄力射击的站定
	chargeCd    = 0.5 // 蓄力射击的后摇（就是它的 CD，总节拍 1.0s/箭：最短的一档）
	volleyStand = 1.5 // 齐射的站定（它最大的破绽）
	volleyCd    = 1.0 // 齐射的后摇（= CD，2.0s 一轮）
	retreatCD   = 6.0 // 后撤步：它的价值是白送 90px 位移，按"大招呼"给 6s（同小骑士冲撞/血契饿扑）
	rollCD      = 4

	// —— 位移（后撤/翻滚共用一个速度，只差时长）——
	dashSpeed   = 300.0
	dashRetreat = 0.3  // 撤退 300×0.3 = 90
	dashRoll    = 0.25 // 翻滚 300×0.25 = 75
	retreatDist = dashSpeed * dashRetreat
	rollDist    = dashSpeed * dashRoll

	// —— 齐射排面 ——
	volleyN   = 4
	volleyGap = 8.0 // 相邻箭心间隔：-12/-4/+4/+12 → 含箭半径整体约 32 宽
	// 全中走廊 ±12、掉支带 12~36、超过 36 才整个躲开（几何见设计稿）

	// —— AI 距离档 ——
	closeRange  = 90.0  // < 90：贴脸，后撤步（CD 中则蓄力硬顶）
	volleyRange = 180.0 // > 180 且无弹道威胁：齐射；其余（含中距）走蓄力射击

	// —— 安全向算法 ——
	safeDirs     = 12  // 采样 12 个方向（30° 一格）
	threatBuffer = 8.0 // 弹道走廊的缓冲
	wEnemy       = 1.0 // 三因子权重 1:1:1（首版，手感期再调）
	wBallistic   = 1.0
	wField       = 1.0
	threatFine   = -400.0 // 落点在弹道走廊里的重罚
)

const (
	stepIdle     = iota
	stepCharge   // 蓄力射击：站定中
	stepRecover  // 蓄力射击：后摇
	stepVolley   // 齐射：站定中
	stepVolleyCd // 齐射：后摇
)

func init() {
	p := unit.NewPack(KindArcher, assets)
	p.Register(unit.Spec{
		Kind:    KindArcher,
		Role:    unit.RoleFighter,
		Radius:  archerRadius,
		MaxHP:   archerHP,
		Speed:   archerSpeed,
		Vision:  archerVision,
		Fighter: true,
		// Ghost 260：巡航 178 不拉残影，只有后撤/翻滚那两下（350/300）留残影。
		Look: unit.Look{Color: archerColor, Ghost: 260, FX: []string{"archer"}},
	}, func(info unit.SpawnInfo) unit.Actor { return &弓箭{field: info.Field} })
	// 箭：撞墙即消（收 WallHit → Despawn，不弹射）；撞人出伤 + 0.2s 眩晕 + 自毁。
	p.Register(unit.Spec{
		Kind: KindArrow, Role: unit.RoleProjectile,
		Radius: arrowR, MaxHP: 1, Speed: arrowSpeed, Vision: 0,
		Look: unit.Look{Color: arrowColor, Trail: false, FX: []string{"arrow"}},
	}, func(info unit.SpawnInfo) unit.Actor { return &箭{dmg: arrowDmg, slot: info.Slot} })
	p.Register(unit.Spec{
		Kind: KindVolleyArrow, Role: unit.RoleProjectile,
		Radius: arrowR, MaxHP: 1, Speed: arrowSpeed, Vision: 0,
		Look: unit.Look{Color: arrowColor, Trail: false, FX: []string{"arrow"}},
	}, func(info unit.SpawnInfo) unit.Actor { return &箭{dmg: volleyDmg, slot: info.Slot} })
}

// 箭：两种箭只差伤害，行为同一套。
type 箭 struct {
	dmg  float64
	slot int
}

func (a *箭) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Collision:
		if unit.EnemyTarget(e, a.slot) {
			ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: a.dmg}
			// Until>0 由引擎自动到期；同一波四支箭同帧发同一个 Until，覆盖后还是 0.2s
			ctx.Out <- unit.Stun{UnitID: e.Other.ID, Hold: true, Until: e.Time + arrowStun}
			// 光有 Stun 看不出停顿：它只停感知，球照旧按原速度飞（直线巡航的目标毫无反应）。
			// 要"真的愣一下"得配 HoldStill：它每帧把速度顶成 0，到点把记下的速度还回去
			// （筑墙者(百战) 相撞眩晕用的同一套）。
			ctx.Out <- unit.HoldStill{UnitID: e.Other.ID, Until: e.Time + arrowStun}
		}
		// 碰到任何单位都收箭：不留在场上当幽灵弹
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	case unit.WallHit:
		// 场边 / 硬墙 / 胶囊墙统一：撞墙即消，不弹射
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

// safeTarget 是安全向算法要用的战场摘要（敌方战斗机/活随从 + 敌方弹）。
// 翻滚发生在 IncomingDamage 那一拍、手里没有 Sense，所以每拍存一份给它用。
type safeTarget struct {
	x, y   float64
	r      float64
	vx, vy float64
	bullet bool
}

type 弓箭 struct {
	step      int
	until     float64 // 当前这一步的截止时刻
	rollAt    float64 // 翻滚 CD
	retAt     float64 // 后撤步 CD
	dashUntil float64 // 滑步（FS）还在生效的截止时刻
	fsTok     uint64
	slot      int
	booted    bool
	x, y      float64
	world     []safeTarget
	field     unit.Field
}

func (a *弓箭) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		a.hurt(ctx, e)
	case unit.Sense:
		a.tick(ctx, e)
	}
}

func (a *弓箭) tick(ctx unit.Context, s unit.Sense) {
	a.x, a.y, a.slot = s.Self.X, s.Self.Y, s.Self.Slot
	a.field = s.Field
	a.note(s)
	if !a.booted {
		a.booted = true
	}
	target := unit.Seek(s) // 走 Seek：挡视线墙后锁不到人就没有目标，全体待机
	dist := math.MaxFloat64
	if target != nil {
		dist = math.Hypot(target.X-s.Self.X, target.Y-s.Self.Y)
	}
	switch a.step {
	case stepCharge:
		if a.abortIfUnlocked(ctx, target) {
			return
		}
		// 站定期间每帧把瞄准方向发出去：弓要跟着目标转，不然出膛方向（开火那拍重新瞄）和弓的朝向不一致
		a.emitFace(ctx, s, target)
		// 蓄力中贴脸：主动取消止损（这箭哑火），去等翻滚 CD
		if dist < closeRange {
			a.cancel(ctx)
			return
		}
		if s.Time+1e-9 >= a.until {
			a.shoot(ctx, s, target, KindArrow, arrowDmg)
			a.step, a.until = stepRecover, s.Time+chargeCd
		}
	case stepRecover:
		if s.Time+1e-9 >= a.until {
			a.step = stepIdle
		}
	case stepVolley:
		if a.abortIfUnlocked(ctx, target) {
			return
		}
		a.emitFace(ctx, s, target) // 齐射蓄力同理：每帧跟目标
		if dist < closeRange {
			a.cancel(ctx)
			return
		}
		if s.Time+1e-9 >= a.until {
			a.fireVolley(ctx, s, target)
			a.step, a.until = stepVolleyCd, s.Time+volleyCd
		}
	case stepVolleyCd:
		if s.Time+1e-9 >= a.until {
			a.step = stepIdle
		}
	}
	if a.step == stepIdle {
		a.choose(ctx, s, target, dist)
	}
}

// choose 是 AI 优先级（设计稿那张表）：贴脸先想办法拉开 → 远且无弹道威胁就齐射 → 其余蓄力射击。
func (a *弓箭) choose(ctx unit.Context, s unit.Sense, target *unit.Snapshot, dist float64) {
	if target == nil || s.Time+1e-9 < a.dashUntil {
		return // 没目标 / 滑步还没完：待机
	}
	if dist < closeRange {
		if s.Time+1e-9 >= a.retAt {
			a.retreat(ctx, s, target)
			return
		}
		a.beginCharge(ctx, s, target, stepCharge, chargeStand) // CD 中：硬顶一箭
		return
	}
	if dist > volleyRange && !a.anyThreat(s) {
		a.beginCharge(ctx, s, target, stepVolley, volleyStand)
		return
	}
	a.beginCharge(ctx, s, target, stepCharge, chargeStand)
}

// beginCharge 站定蓄力（盾斧的 hold 先例：Stand + 速度清零）。
func (a *弓箭) beginCharge(ctx unit.Context, s unit.Sense, target *unit.Snapshot, step int, stand float64) {
	a.step = step
	a.until = s.Time + stand
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	ctx.Out <- unit.FX{Name: "draw", Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot, X: s.Self.X, Y: s.Self.Y, Amount: stand}
	a.emitFace(ctx, s, target)
}

// emitFace 把瞄准方向发给前端：站定蓄力时速度是 0，客户端没法从速度推朝向，只能靠这条。
// X/Y 也要带（不带的话前端按 (0,0) 画，特效就会堆在场心）。
func (a *弓箭) emitFace(ctx unit.Context, s unit.Sense, target *unit.Snapshot) {
	ux, uy := a.aimDir(s, target)
	ctx.Out <- unit.FX{Name: "face", Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot, X: s.Self.X, Y: s.Self.Y, VX: ux, VY: uy}
}

// cancel 收弓：蓄力被自己（贴脸止损 / 丢锁）或翻滚打断，这一箭哑火。
// X/Y 必须带：前端拿 ctx.x/ctx.y 画特效，不带就按 (0,0) 画到场心去。
func (a *弓箭) cancel(ctx unit.Context) {
	a.step = stepIdle
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}
	ctx.Out <- unit.FX{Name: "cancel", Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot, X: a.x, Y: a.y}
}

// abortIfUnlocked 蓄力中途 Seek 不到人就收弓，不把箭射出去。
func (a *弓箭) abortIfUnlocked(ctx unit.Context, target *unit.Snapshot) bool {
	if target != nil {
		return false
	}
	a.cancel(ctx)
	return true
}

// shoot 朝锁定目标出一支箭。没有锁就不放。
func (a *弓箭) shoot(ctx unit.Context, s unit.Sense, target *unit.Snapshot, kind string, dmg float64) {
	if target == nil {
		a.cancel(ctx)
		return
	}
	ux, uy := a.aimDir(s, target)
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}
	a.spawnArrow(ctx, s, ux, uy, 0, kind)
	a.emitFace(ctx, s, target)
	ctx.Out <- unit.FX{Name: "shoot", Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot, X: s.Self.X, Y: s.Self.Y, VX: ux, VY: uy, Amount: dmg}
}

// spawnArrow 出膛：箭心在球面外一点（radius + 箭半径 + 余量），沿 u 方向；off 是横向偏移（齐射用）。
func (a *弓箭) spawnArrow(ctx unit.Context, s unit.Sense, ux, uy, off float64, kind string) {
	px, py := -uy, ux // 垂直于射击方向
	d := archerRadius + arrowR + arrowGap
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: kind,
		X:    s.Self.X + ux*d + px*off,
		Y:    s.Self.Y + uy*d + py*off,
		VX:   ux * arrowSpeed, VY: uy * arrowSpeed,
		OwnerID: ctx.ID, Slot: s.Self.Slot,
	})
}

// fireVolley 四支贴排同射：垂直射击方向展开，箭心间隔 8（-12/-4/+4/+12）。
// 不是扇面——四支平行飞，所以横移躲不开、只能躲远。
func (a *弓箭) fireVolley(ctx unit.Context, s unit.Sense, target *unit.Snapshot) {
	if target == nil {
		a.cancel(ctx)
		return
	}
	ux, uy := a.aimDir(s, target)
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}
	for i := 0; i < volleyN; i++ {
		off := (float64(i) - float64(volleyN-1)/2) * volleyGap
		a.spawnArrow(ctx, s, ux, uy, off, KindVolleyArrow)
	}
	a.emitFace(ctx, s, target)
	ctx.Out <- unit.FX{Name: "volley", Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot, X: s.Self.X, Y: s.Self.Y, VX: ux, VY: uy, Amount: volleyN}
}

// retreat 后撤步射击：无前摇，立刻朝安全向滑步 90，撤的途中朝锁定目标射一支箭。
func (a *弓箭) retreat(ctx unit.Context, s unit.Sense, target *unit.Snapshot) {
	ux, uy := a.safeDir(s, retreatDist)
	a.retAt = s.Time + retreatCD
	a.dashUntil = s.Time + dashRetreat
	a.mint(ctx, ux, uy, s.Time+dashRetreat)
	ctx.Out <- unit.FX{Name: "dash", Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot, X: s.Self.X, Y: s.Self.Y, VX: ux, VY: uy, Amount: retreatDist}
	a.shoot(ctx, s, target, KindArrow, arrowDmg) // 移动射击：不站定，这一拍不浪费
}

// mint 起一条瞬时 FS（滑步）：持着的时候速度就写死成这个向量，巡航带子暂停。
func (a *弓箭) mint(ctx unit.Context, ux, uy, until float64) {
	a.fsTok++
	ctx.Out <- unit.AddFS{
		UnitID: ctx.ID, DX: ux, DY: uy, BaseSpeed: dashSpeed,
		OnWall: true, ExpiresAt: until, Token: a.fsTok,
	}
}

// hurt 翻滚：纯被动，挨报价那一拍才滚（地慧星的同构口径：IncomingDamage 层 Block）。
// CD 中的攻击照常吃；滚出去的同时打断自己的蓄力（这一箭哑火）。
func (a *弓箭) hurt(ctx unit.Context, d unit.IncomingDamage) {
	if d.Time+1e-9 >= a.rollAt {
		ux, uy := a.safeDirCached(rollDist)
		a.rollAt = d.Time + rollCD
		a.dashUntil = d.Time + dashRoll
		a.mint(ctx, ux, uy, d.Time+dashRoll)
		ctx.Out <- unit.FX{Name: "roll", Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot, X: a.x, Y: a.y, VX: ux, VY: uy, Amount: rollDist}
		if a.step == stepCharge || a.step == stepVolley {
			a.cancel(ctx)
		}
		unit.BlockHit(ctx, d)
		return
	}
	unit.ConfirmHit(ctx, d)
}

func (a *弓箭) aimDir(s unit.Sense, target *unit.Snapshot) (float64, float64) {
	if target != nil {
		dx, dy := target.X-s.Self.X, target.Y-s.Self.Y
		if n := math.Hypot(dx, dy); n > 1e-6 {
			return dx / n, dy / n
		}
	}
	vx, vy := s.Self.VX, s.Self.VY
	if n := math.Hypot(vx, vy); n > 1e-6 {
		return vx / n, vy / n
	}
	return 1, 0
}

// note 每拍存一份战场摘要：敌方战斗机/活随从 + 敌方弹（翻滚那一拍没有 Sense 可用）。
func (a *弓箭) note(s unit.Sense) {
	a.world = a.world[:0]
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if unit.Hittable(*o, s.Self.Slot) {
			a.world = append(a.world, safeTarget{x: o.X, y: o.Y, r: o.Radius})
			continue
		}
		if enemyBullet(*o, s.Self.Slot) {
			a.world = append(a.world, safeTarget{x: o.X, y: o.Y, r: o.Radius, vx: o.VX, vy: o.VY, bullet: true})
		}
	}
}

// enemyBullet 敌方弹：Attach 扇环不算弹（跟着人走的近战弧，不是会飞过来的东西）。
func enemyBullet(o unit.Snapshot, slot int) bool {
	return o.Slot != slot && o.Role == unit.RoleProjectile && o.ArcSpan == 0
}

// safeDir 安全向：采样 12 个方向（30° 一格）各自算落点，取分最高的那个单位方向。
func (a *弓箭) safeDir(s unit.Sense, dist float64) (float64, float64) {
	best, bestScore := 0.0, math.Inf(-1)
	f := s.Field
	for i := 0; i < safeDirs; i++ {
		ang := float64(i) * 2 * math.Pi / safeDirs
		lx, ly := s.Self.X+math.Cos(ang)*dist, s.Self.Y+math.Sin(ang)*dist
		if score := a.score(f, lx, ly); score > bestScore {
			bestScore, best = score, ang
		}
	}
	return math.Cos(best), math.Sin(best)
}

// safeDirCached 是翻滚用的版本：世界状态取上一拍的摘要（IncomingDamage 里没有 Sense）。
func (a *弓箭) safeDirCached(dist float64) (float64, float64) {
	best, bestScore := 0.0, math.Inf(-1)
	f := a.field
	s := unit.Sense{Self: unit.Snapshot{X: a.x, Y: a.y, Slot: a.slot}}
	for _, w := range a.world {
		s.Nearby = append(s.Nearby, unit.Snapshot{
			X: w.x, Y: w.y, Radius: w.r, VX: w.vx, VY: w.vy,
			Slot: 1 - a.slot, Role: pickRole(w),
		})
	}
	for i := 0; i < safeDirs; i++ {
		ang := float64(i) * 2 * math.Pi / safeDirs
		lx, ly := a.x+math.Cos(ang)*dist, a.y+math.Sin(ang)*dist
		if score := a.score(f, lx, ly); score > bestScore {
			bestScore, best = score, ang
		}
	}
	return math.Cos(best), math.Sin(best)
}

func pickRole(w safeTarget) string {
	if w.bullet {
		return unit.RoleProjectile
	}
	return unit.RoleFighter
}

// score 三因子 1:1:1：敌距（越远越好）+ 弹道（走廊内重罚）+ 场地（夹回可进入区域的折损）。
func (a *弓箭) score(f unit.Field, lx, ly float64) float64 {
	enemy := 400.0
	ballistic := 240.0
	for _, w := range a.world {
		if w.bullet {
			corridor := w.r + archerRadius + threatBuffer
			d := rayDist(lx, ly, w.x, w.y, w.vx, w.vy)
			if d < corridor {
				ballistic = threatFine
			} else if d-corridor < ballistic {
				ballistic = d - corridor
			}
			continue
		}
		if d := math.Hypot(w.x-lx, w.y-ly); d < enemy {
			enemy = d
		}
	}
	cx, cy := f.Clamp(lx, ly, archerRadius)
	loss := math.Hypot(cx-lx, cy-ly)
	return wEnemy*enemy + wBallistic*ballistic - wField*loss
}

// anyThreat 现在站的这一格有没有弹道威胁（齐射的前置条件：远且没人朝它射）。
func (a *弓箭) anyThreat(s unit.Sense) bool {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !enemyBullet(*o, s.Self.Slot) {
			continue
		}
		if rayDist(s.Self.X, s.Self.Y, o.X, o.Y, o.VX, o.VY) < o.Radius+archerRadius+threatBuffer {
			return true
		}
	}
	return false
}

// rayDist 点到弹道**射线**的距离（只算弹的前方：已经飞过去的不算威胁）。
func rayDist(px, py, ox, oy, vx, vy float64) float64 {
	n := math.Hypot(vx, vy)
	if n < 1e-6 {
		return math.Hypot(px-ox, py-oy)
	}
	ux, uy := vx/n, vy/n
	t := (px-ox)*ux + (py-oy)*uy
	if t < 0 {
		t = 0
	}
	return math.Hypot(px-(ox+ux*t), py-(oy+uy*t))
}
