// R.缪：一堆相同外形的克隆人互相残杀，活下来的那个就是本体。
// 本体免疫伤害，HP 随存活[缪]数量变化，每帧跟随血量最高的[缪]（位置+速度向量）。
// [缪]白色活随从，近战+开火+突刺。击杀同类永久加速，加速值继承。
package r缪

import (
	"embed"
	"math"
	"math/rand/v2"
	"sync"
	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const (
	KindRMiu    = "R.缪"
	KindMiu     = "缪"
	KindMiuShot = "缪弹"
	KindMiuArc  = "缪弧"
)

const (
	rMiuRadius = miuRadius // 本体与缪同尺寸，伪装成克隆人
	rMiuMaxHP  = miuHP     // 本体血量实时与缪互换，上限与缪一致

	miuRadius    = 18.0
	miuHP        = 100.0
	miuBaseSpeed = 170.0
	miuVision    = 9999.0
	miuColor     = "#f0f0f0"

	miuDamage    = 5.0
	miuMeleeCD   = 0.35
	miuMeleeGap  = 8.0 // 半径之和外的额外近战范围
	miuMeleeSpan = 60.0
	miuArcInner  = miuRadius - 5
	miuArcOuter  = miuRadius
	miuArcColor  = "#111111"

	miuShotSpeed    = 500.0
	miuShotRadius   = 5.0
	miuShotDamage   = 5.0
	miuShotLifetime = 3.0
	miuFireCD       = 12.0
	miuFireCount    = 3
	miuFireGap      = 0.15

	miuThrustSpeed = 640.0
	miuThrustCD    = 6.0
	miuThrustDur   = 0.4

	killBonusPer  = 100.0
	speedDivisor  = 20.0
	miuSpawnCount = 5
)

// 共享状态：击杀加速追踪
var (
	miuMu           sync.Mutex
	miuBonus        = map[uint64]float64{} // 缪 ID → 累计加速值
	miuLastAttacker = map[uint64]uint64{}  // 被攻击的缪 ID → 最后攻击者 ID
	miuClaimed      = map[uint64]bool{}    // 已被认领击杀奖励的缪 ID
)

func init() {
	p := unit.NewPack(KindRMiu, assets)

	// R.缪 战斗机：与缪同外形、有碰撞、能移动、挂近战弧；血量/位置与血最高缪换位
	p.Register(unit.Spec{
		Kind:    KindRMiu,
		Role:    unit.RoleFighter,
		Radius:  rMiuRadius,
		MaxHP:   rMiuMaxHP,
		Speed:   miuBaseSpeed,
		Vision:  miuVision,
		Fighter: true,
		Look:    unit.Look{Color: miuColor, Ghost: 180, FX: []string{"miu"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &R缪{ownerID: info.OwnerID}
	})

	// 缪 活随从：白色、近战+开火+突刺、击杀同类加速
	p.Register(unit.Spec{
		Kind:   KindMiu,
		Role:   unit.RoleMinion,
		Radius: miuRadius,
		MaxHP:  miuHP,
		Speed:  miuBaseSpeed,
		Vision: miuVision,
		Mortal: true,
		Look:   unit.Look{Color: miuColor, Ghost: 180, FX: []string{"miu"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &缪{ownerID: info.OwnerID, slot: info.Slot}
	})

	// 缪弹 投射物：射速快，伤害含速度差
	p.Register(unit.Spec{
		Kind:   KindMiuShot,
		Role:   unit.RoleProjectile,
		Radius: miuShotRadius,
		MaxHP:  1,
		Speed:  miuShotSpeed,
		Vision: miuVision, // 需要 Sense 手动检测同 slot 缪
		Look:   unit.Look{Color: "#1a1a1a", Overlay: true, Trail: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &缪弹{owner: info.OwnerID, slot: info.Slot}
	})

	p.Register(unit.Spec{
		Kind:     KindMiuArc,
		Role:     unit.RoleProjectile,
		Radius:   miuArcOuter,
		MaxHP:    1,
		Speed:    miuBaseSpeed,
		Vision:   0,
		Attach:   true,
		ArcSpan:  unit.Deg(miuMeleeSpan),
		ArcInner: miuArcInner,
		Look:     unit.Look{Color: miuArcColor, Overlay: true},
	}, func(unit.SpawnInfo) unit.Actor {
		return &缪弧{}
	})
}

// ===================== R.缪 战斗机 =====================

type R缪 struct {
	ownerID uint64
	booted  bool

	// 击杀加速（与缪同机制）
	killBonus  float64
	speed      float64
	prevMiuIDs map[uint64]bool

	// 每帧缓存：本体当前状态 + 血最高缪快照（IncomingDamage 里没有 Nearby，需提前记录）
	hp             float64
	x, y, vx, vy   float64
	bestID         uint64
	bestHP         float64
	bestX, bestY   float64
	bestVX, bestVY float64
	hasBest        bool
	marks          []unit.Mark
	bestMarks      []unit.Mark

	// 移动与近战（与缪同款）
	arc            unit.AttachState
	lockedTargetID uint64
	retargetAt     float64
	meleeReadyAt   float64
	skipSync       bool // 本帧刚致死换位，Sense 不再二次互换
}

func (r *R缪) Handle(ctx unit.Context, ev unit.Event) {
	// 本体受击：若这一发会致死且场上还有缪，先保命（换身份，伤害打在替身缪上）
	if d, ok := ev.(unit.IncomingDamage); ok {
		if r.hasBest && r.hp <= d.Amount+1e-9 {
			unit.BlockHit(ctx, d)
			bodyHP := r.hp
			r.emitSwap(ctx, r.bestID, r.bestHP, r.bestX, r.bestY, r.bestVX, r.bestVY, bodyHP, r.bestMarks)
			// 替身先接过本体血，再吃这一击（SetHP 负值会被引擎丢掉）
			ctx.Out <- unit.Damage{From: d.From, To: r.bestID, Amount: d.Amount}
			miuMu.Lock()
			miuLastAttacker[r.bestID] = d.From
			miuMu.Unlock()
			r.skipSync = true
			r.hasBest = false
			return
		}
		unit.ConfirmHit(ctx, d)
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	r.hp = s.Self.HP
	r.x, r.y = s.Self.X, s.Self.Y
	r.vx, r.vy = s.Self.VX, s.Self.VY
	r.marks = cloneMarks(s.Self.Marks)
	if !r.booted {
		r.booted = true
		r.speed = miuBaseSpeed
		r.spawnMinions(ctx, s)
		r.meleeReadyAt = s.Time
		return // 缪下一帧才出生，本帧跳过同步
	}

	// 击杀领奖（与缪同机制）+ 更新巡航速度
	claimMiuKill(ctx, s, ctx.ID, &r.prevMiuIDs, &r.killBonus, r.vx, r.vy)
	r.speed = miuBaseSpeed + r.killBonus
	miuMu.Lock()
	miuBonus[ctx.ID] = r.killBonus
	miuMu.Unlock()
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: r.speed}

	// 自主行为：近战弧 + 索敌移动（与缪同款）
	if unit.RearmAttach(s, ctx.ID, KindMiuArc, 0, &r.arc) {
		unit.SpawnAttach(ctx, s, KindMiuArc)
	}
	r.moveAndMelee(ctx, s)
	if r.skipSync {
		r.skipSync = false
		r.refreshBest(ctx, s)
		return
	}
	r.syncState(ctx, s)
}

func (r *R缪) spawnMinions(ctx unit.Context, s unit.Sense) {
	for i := 0; i < miuSpawnCount; i++ {
		ang := float64(i) * 2 * math.Pi / miuSpawnCount
		dist := miuRadius + rMiuRadius + 20
		ctx.Out <- unit.Spawn{
			Kind:    KindMiu,
			X:       s.Self.X + math.Cos(ang)*dist,
			Y:       s.Self.Y + math.Sin(ang)*dist,
			OwnerID: ctx.ID,
			Slot:    s.Self.Slot,
		}
	}
}

func (r *R缪) moveAndMelee(ctx unit.Context, s unit.Sense) {
	// 找目标：敌方活体 + 同主人缪尔（克隆人互殴，活下来的才是本体）
	targets := r.findEnemies(s)

	// 近战（同缪机制）
	if s.Time+1e-9 >= r.meleeReadyAt {
		hx, hy := r.facing()
		reach := s.Self.Radius + miuMeleeGap
		var best *unit.Snapshot
		bestDist := math.Inf(1)
		for i := range targets {
			t := &targets[i]
			if !inFan(s.Self, hx, hy, *t, reach, miuMeleeSpan) {
				continue
			}
			d := math.Hypot(t.X-s.Self.X, t.Y-s.Self.Y)
			if d < bestDist {
				bestDist = d
				best = t
			}
		}
		if best != nil {
			dmg := miuDamage
			// 速度差加成，与缪一致（用巡航速度）
			sp := r.speed
			tsp := math.Hypot(best.VX, best.VY)
			if sp > tsp {
				dmg += (sp - tsp) / speedDivisor
			}
			ctx.Out <- unit.Damage{From: ctx.ID, To: best.ID, Amount: dmg}
			// 追踪自己击杀的缪
			if best.Kind == KindMiu && best.OwnerID == s.Self.ID {
				miuMu.Lock()
				miuLastAttacker[best.ID] = ctx.ID
				miuMu.Unlock()
			}
			ctx.Out <- unit.FX{
				Name: "miu-melee", Kind: ctx.Kind,
				X: best.X, Y: best.Y, Slot: s.Self.Slot,
			}
			r.meleeReadyAt = s.Time + miuMeleeCD
		}
	}

	// 索敌移动：锁一次目标直线走
	if r.lockedTargetID != 0 {
		alive := false
		for i := range targets {
			if targets[i].ID == r.lockedTargetID {
				alive = true
				break
			}
		}
		if alive && s.Time+1e-9 < r.retargetAt {
			return
		}
		r.lockedTargetID = 0
	}
	if len(targets) == 0 {
		sp := math.Hypot(r.vx, r.vy)
		if sp < 1e-6 {
			ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: r.speed, VY: 0}
		}
		return
	}
	best := &targets[0]
	bestDist := math.Hypot(best.X-s.Self.X, best.Y-s.Self.Y)
	for i := 1; i < len(targets); i++ {
		d := math.Hypot(targets[i].X-s.Self.X, targets[i].Y-s.Self.Y)
		if d < bestDist {
			bestDist = d
			best = &targets[i]
		}
	}
	dx := best.X - s.Self.X
	dy := best.Y - s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	r.lockedTargetID = best.ID
	r.retargetAt = s.Time + 3.0
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: dx / n * r.speed, VY: dy / n * r.speed}
}

func (r *R缪) findEnemies(s unit.Sense) []unit.Snapshot {
	var out []unit.Snapshot
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.ID == s.Self.ID {
			continue
		}
		// 克隆人互殴：自己 spawn 的缪也是目标（最后一个活着的才是本体）
		if o.Kind == KindMiu && o.OwnerID == s.Self.ID {
			out = append(out, *o)
			continue
		}
		// 敌方战斗机或活随从
		if o.Slot != s.Self.Slot && (o.Role == unit.RoleFighter || o.Mortal) {
			out = append(out, *o)
		}
	}
	return out
}

func (r *R缪) facing() (float64, float64) {
	n := math.Hypot(r.vx, r.vy)
	if n > 1e-6 {
		return r.vx / n, r.vy / n
	}
	return 0, 1
}

func (r *R缪) aliveMinions(ctx unit.Context, s unit.Sense) []*unit.Snapshot {
	var alive []*unit.Snapshot
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID == ctx.ID && o.Kind == KindMiu {
			alive = append(alive, o)
		}
	}
	return alive
}

func (r *R缪) refreshBest(ctx unit.Context, s unit.Sense) {
	alive := r.aliveMinions(ctx, s)
	if len(alive) == 0 {
		r.hasBest = false
		return
	}
	bestMiu := alive[0]
	for _, o := range alive[1:] {
		if o.HP > bestMiu.HP {
			bestMiu = o
		}
	}
	r.bestID = bestMiu.ID
	r.bestHP = bestMiu.HP
	r.bestX, r.bestY = bestMiu.X, bestMiu.Y
	r.bestVX, r.bestVY = bestMiu.VX, bestMiu.VY
	r.bestMarks = cloneMarks(bestMiu.Marks)
	r.hasBest = true
}

func (r *R缪) emitSwap(ctx unit.Context, other uint64, ohp, ox, oy, ovx, ovy, selfHP float64, otherMarks []unit.Mark) {
	ctx.Out <- unit.SetHP{UnitID: ctx.ID, HP: ohp, MaxHP: miuHP}
	ctx.Out <- unit.SetHP{UnitID: other, HP: selfHP, MaxHP: miuHP}
	ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: ox, Y: oy}
	ctx.Out <- unit.Teleport{UnitID: other, X: r.x, Y: r.y}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: ovx, VY: ovy}
	ctx.Out <- unit.SetVelocity{UnitID: other, VX: r.vx, VY: r.vy}
	// 剑痕/诅咒留在原来那具身体上：单位走了，标记对调留下。
	r.swapMarks(ctx, other, r.marks, otherMarks)
	r.hp = ohp
	r.x, r.y = ox, oy
	r.vx, r.vy = ovx, ovy
	r.marks = cloneMarks(otherMarks)
	r.lockedTargetID = 0
}

func (r *R缪) swapMarks(ctx unit.Context, other uint64, mine, theirs []unit.Mark) {
	ctx.Out <- unit.ClearMarks{UnitID: ctx.ID}
	ctx.Out <- unit.ClearMarks{UnitID: other}
	putMarks(ctx, ctx.ID, theirs)
	putMarks(ctx, other, mine)
}

func putMarks(ctx unit.Context, id uint64, marks []unit.Mark) {
	for _, m := range marks {
		if m.Kind == "" || m.Stacks == 0 {
			continue
		}
		ctx.Out <- unit.StackMark{UnitID: id, Kind: m.Kind, Delta: m.Stacks, Icon: m.Icon}
	}
}

func cloneMarks(in []unit.Mark) []unit.Mark {
	if len(in) == 0 {
		return nil
	}
	out := make([]unit.Mark, len(in))
	copy(out, in)
	return out
}

func (r *R缪) syncState(ctx unit.Context, s unit.Sense) {
	alive := r.aliveMinions(ctx, s)
	if len(alive) == 0 {
		r.hasBest = false
		return
	}
	r.refreshBest(ctx, s)

	best := s.Self
	for _, o := range alive {
		if o.HP > best.HP {
			best = *o
		}
	}
	if best.ID == ctx.ID {
		return
	}
	r.emitSwap(ctx, best.ID, best.HP, best.X, best.Y, best.VX, best.VY, s.Self.HP, best.Marks)
}

// ===================== 缪 活随从 =====================

type 缪 struct {
	ownerID uint64
	slot    int
	booted  bool

	killBonus float64
	speed     float64

	meleeReadyAt float64

	fireReadyAt float64
	fireQueue   int
	fireNextAt  float64

	thrustReadyAt float64
	thrustUntil   float64
	thrustToken   uint64
	thrustSeq     uint64
	thrusting     bool
	thrustDX      float64
	thrustDY      float64

	x, y float64
	vx   float64
	vy   float64

	lockedTargetID uint64
	retargetAt     float64

	prevMiuIDs map[uint64]bool
	arc        unit.AttachState
}

func (m *缪) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		unit.ConfirmHit(ctx, e)
	case unit.WallHit:
		if m.thrusting {
			bx, by := bounce(m.vx, m.vy, e.NX, e.NY)
			m.endThrust(ctx, bx, by)
		}
	case unit.Collision:
		if m.thrusting {
			bx, by := bounce(m.vx, m.vy, e.NX, e.NY)
			m.endThrust(ctx, bx, by)
		}
	case unit.Sense:
		m.tick(ctx, e)
	}
}

func (m *缪) tick(ctx unit.Context, s unit.Sense) {
	m.x, m.y = s.Self.X, s.Self.Y
	m.vx, m.vy = s.Self.VX, s.Self.VY

	if !m.booted {
		m.booted = true
		m.speed = miuBaseSpeed
		m.meleeReadyAt = s.Time
		m.fireReadyAt = s.Time + miuFireCD + rand.Float64()*2 - 1
		m.thrustReadyAt = s.Time + miuThrustCD + rand.Float64()*2 - 1
	}

	// 更新共享状态
	miuMu.Lock()
	miuBonus[ctx.ID] = m.killBonus
	miuMu.Unlock()

	// 检测击杀（共享机制：本体与缪同用）
	claimMiuKill(ctx, s, m.ownerID, &m.prevMiuIDs, &m.killBonus, m.vx, m.vy)

	// 同步速度
	m.speed = miuBaseSpeed + m.killBonus
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: m.speed}

	if unit.RearmAttach(s, ctx.ID, KindMiuArc, 0, &m.arc) {
		unit.SpawnAttach(ctx, s, KindMiuArc)
	}

	// 突刺中不执行其他行为
	if m.thrusting {
		if s.Time+1e-9 >= m.thrustUntil {
			m.endThrust(ctx, m.vx, m.vy)
		}
		return
	}

	targets := m.findTargets(s)

	m.tryMelee(ctx, s, targets)
	m.tryFire(ctx, s, targets)
	m.tryThrust(ctx, s, targets)
	m.moveToward(ctx, s, targets)
}

func (m *缪) findTargets(s unit.Sense) []unit.Snapshot {
	var out []unit.Snapshot
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.ID == s.Self.ID {
			continue
		}
		// 同主人的其他缪（保持互伤）
		if o.Kind == KindMiu && o.OwnerID == m.ownerID {
			out = append(out, *o)
			continue
		}
		// 本体（同 slot 的 R.缪）也是克隆人之一，互殴
		if o.Kind == KindRMiu && o.Slot == m.slot {
			out = append(out, *o)
			continue
		}
		// 敌方战斗机或活随从
		if o.Slot != m.slot && (o.Role == unit.RoleFighter || o.Mortal) {
			out = append(out, *o)
		}
	}
	return out
}

func (m *缪) tryMelee(ctx unit.Context, s unit.Sense, targets []unit.Snapshot) {
	if s.Time+1e-9 < m.meleeReadyAt {
		return
	}
	hx, hy := m.facing()
	reach := s.Self.Radius + miuMeleeGap
	var best *unit.Snapshot
	bestDist := math.Inf(1)
	for i := range targets {
		t := &targets[i]
		if !inFan(s.Self, hx, hy, *t, reach, miuMeleeSpan) {
			continue
		}
		d := math.Hypot(t.X-s.Self.X, t.Y-s.Self.Y)
		if d < bestDist {
			bestDist = d
			best = t
		}
	}
	if best == nil {
		return
	}
	dmg := miuDamage + m.speedBonus(best)
	ctx.Out <- unit.Damage{From: ctx.ID, To: best.ID, Amount: dmg}

	if best.Kind == KindMiu && best.OwnerID == m.ownerID {
		miuMu.Lock()
		miuLastAttacker[best.ID] = ctx.ID
		miuMu.Unlock()
	}

	ctx.Out <- unit.FX{
		Name: "miu-melee", Kind: ctx.Kind,
		X: best.X, Y: best.Y, Slot: s.Self.Slot,
	}
	m.meleeReadyAt = s.Time + miuMeleeCD
}

func (m *缪) facing() (float64, float64) {
	n := math.Hypot(m.vx, m.vy)
	if n > 1e-6 {
		return m.vx / n, m.vy / n
	}
	n = math.Hypot(m.thrustDX, m.thrustDY)
	if n > 1e-6 {
		return m.thrustDX, m.thrustDY
	}
	return 0, 1
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

func (m *缪) speedBonus(t *unit.Snapshot) float64 {
	selfSp := m.speed
	targetSp := math.Hypot(t.VX, t.VY)
	if selfSp > targetSp {
		return (selfSp - targetSp) / speedDivisor
	}
	return 0
}

func (m *缪) tryFire(ctx unit.Context, s unit.Sense, targets []unit.Snapshot) {
	// 处理排队子弹
	if m.fireQueue > 0 && s.Time+1e-9 >= m.fireNextAt {
		m.fireQueue--
		m.fireBullet(ctx, s, targets)
		m.fireNextAt = s.Time + miuFireGap
	}
	if m.fireQueue > 0 || s.Time+1e-9 < m.fireReadyAt || len(targets) == 0 {
		return
	}
	// 首发立即射出
	m.fireBullet(ctx, s, targets)
	m.fireQueue = miuFireCount - 1
	m.fireNextAt = s.Time + miuFireGap
	m.fireReadyAt = s.Time + miuFireCD
}

func (m *缪) fireBullet(ctx unit.Context, s unit.Sense, targets []unit.Snapshot) {
	t := targets[rand.IntN(len(targets))]
	dx := t.X - s.Self.X
	dy := t.Y - s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	ux, uy := dx/n, dy/n
	gap := s.Self.Radius + miuShotRadius + 1.5
	ctx.Out <- unit.Spawn{
		Kind:    KindMiuShot,
		X:       s.Self.X + ux*gap,
		Y:       s.Self.Y + uy*gap,
		VX:      ux * miuShotSpeed,
		VY:      uy * miuShotSpeed,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	}
	ctx.Out <- unit.FX{
		Name: "miu-shot", Kind: ctx.Kind,
		X: s.Self.X + ux*gap, Y: s.Self.Y + uy*gap,
		VX: ux, VY: uy, Slot: s.Self.Slot,
	}
}

func (m *缪) tryThrust(ctx unit.Context, s unit.Sense, targets []unit.Snapshot) {
	if s.Time+1e-9 < m.thrustReadyAt || len(targets) == 0 {
		return
	}
	t := targets[rand.IntN(len(targets))]
	dx := t.X - s.Self.X
	dy := t.Y - s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	m.thrustSeq++
	m.thrustToken = ctx.ID<<32 | m.thrustSeq
	m.thrusting = true
	m.thrustUntil = s.Time + miuThrustDur
	m.thrustReadyAt = s.Time + miuThrustCD
	m.thrustDX, m.thrustDY = dx/n, dy/n
	ctx.Out <- unit.AddFS{
		UnitID:    ctx.ID,
		DX:        m.thrustDX,
		DY:        m.thrustDY,
		BaseSpeed: miuThrustSpeed,
		OnWall:    true,
		ExpiresAt: s.Time + miuThrustDur,
		Token:     m.thrustToken,
	}
	ctx.Out <- unit.FX{
		Name: "miu-thrust", Kind: ctx.Kind,
		X: s.Self.X, Y: s.Self.Y,
		VX: dx / n, VY: dy / n, Slot: s.Self.Slot,
	}
}

func (m *缪) endThrust(ctx unit.Context, vx, vy float64) {
	if !m.thrusting {
		return
	}
	m.thrusting = false
	ctx.Out <- unit.RemoveFS{UnitID: ctx.ID, Token: m.thrustToken}
	n := math.Hypot(vx, vy)
	if n < 1e-6 {
		vx, vy = m.thrustDX, m.thrustDY
		n = math.Hypot(vx, vy)
	}
	if n < 1e-6 {
		vx, vy, n = 1, 0, 1
	}
	// 活随从没有巡航减速；撤 FS 后必须立刻把速度拉回巡航。
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: m.speed}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: vx / n * m.speed, VY: vy / n * m.speed}
}

func bounce(vx, vy, nx, ny float64) (float64, float64) {
	n := math.Hypot(nx, ny)
	if n < 1e-9 {
		return vx, vy
	}
	nx, ny = nx/n, ny/n
	dot := vx*nx + vy*ny
	return vx - 2*dot*nx, vy - 2*dot*ny
}

func (m *缪) moveToward(ctx unit.Context, s unit.Sense, targets []unit.Snapshot) {
	// 锁定目标还活着且未到重锁时间，保持当前方向不走
	if m.lockedTargetID != 0 {
		alive := false
		for i := range targets {
			if targets[i].ID == m.lockedTargetID {
				alive = true
				break
			}
		}
		if alive && s.Time+1e-9 < m.retargetAt {
			return
		}
		m.lockedTargetID = 0
	}

	if len(targets) == 0 {
		// 无目标时保持游荡
		sp := math.Hypot(m.vx, m.vy)
		if sp < 1e-6 {
			ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: m.speed, VY: 0}
		}
		return
	}

	// 朝最近目标锁定方向
	best := &targets[0]
	bestDist := math.Hypot(best.X-s.Self.X, best.Y-s.Self.Y)
	for i := 1; i < len(targets); i++ {
		d := math.Hypot(targets[i].X-s.Self.X, targets[i].Y-s.Self.Y)
		if d < bestDist {
			bestDist = d
			best = &targets[i]
		}
	}
	dx := best.X - s.Self.X
	dy := best.Y - s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	m.lockedTargetID = best.ID
	m.retargetAt = s.Time + 3.0
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: dx / n * m.speed, VY: dy / n * m.speed}
}

// claimMiuKill：扫描本阵营消失的缪尔，若最后攻击者是自己则认领击杀加速（本体与缪同机制）
func claimMiuKill(ctx unit.Context, s unit.Sense, ownerID uint64, prev *map[uint64]bool, bonus *float64, vx, vy float64) {
	currentIDs := map[uint64]bool{}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind == KindMiu && o.OwnerID == ownerID {
			currentIDs[o.ID] = true
		}
	}

	if *prev != nil {
		for id := range *prev {
			if currentIDs[id] {
				continue
			}
			// 此缪已消失，检查是否自己是最后攻击者
			miuMu.Lock()
			if !miuClaimed[id] && miuLastAttacker[id] == ctx.ID {
				miuClaimed[id] = true
				victimBonus := miuBonus[id]
				miuMu.Unlock()

				*bonus += killBonusPer + victimBonus
				speed := miuBaseSpeed + *bonus
				ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: speed}
				// 立刻把当前速度拉到新值（minion 无 decelerateLocked）
				sp := math.Hypot(vx, vy)
				if sp > 1e-6 {
					ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: vx / sp * speed, VY: vy / sp * speed}
				}

				ctx.Out <- unit.FX{
					Name: "miu-powerup", Kind: ctx.Kind,
					X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot,
					Amount: *bonus,
				}
			} else {
				miuMu.Unlock()
			}
		}
	}
	*prev = currentIDs
}

// ===================== 缪弹 投射物 =====================

type 缪弹 struct {
	owner     uint64 // 发射此弹的缪 ID
	slot      int
	spawnTime float64
	booted    bool
	px, py    float64
}

func (b *缪弹) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Collision:
		if e.Other.Kind == KindMiu && e.Other.ID == b.owner {
			return
		}
		if e.Other.Kind != KindMiu && e.Other.Kind != KindRMiu && !unit.Hittable(e.Other, b.slot) {
			return
		}
		b.hit(ctx, e.Other)
	case unit.Sense:
		b.onSense(ctx, e)
	}
}

func (b *缪弹) onSense(ctx unit.Context, s unit.Sense) {
	x, y := s.Self.X, s.Self.Y
	if !b.booted {
		b.booted = true
		b.spawnTime = s.Time
		b.px, b.py = x, y
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if (o.Kind != KindMiu && o.Kind != KindRMiu) || o.Slot != b.slot || o.ID == b.owner {
			continue
		}
		if shotHits(b.px, b.py, x, y, s.Self.Radius, o.X, o.Y, o.Radius) {
			b.hit(ctx, *o)
			return
		}
	}
	b.px, b.py = x, y
	if s.Time > b.spawnTime+miuShotLifetime {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

func shotHits(x0, y0, x1, y1, sr, tx, ty, tr float64) bool {
	reach := sr + tr
	if math.Hypot(x1-tx, y1-ty) <= reach || math.Hypot(x0-tx, y0-ty) <= reach {
		return true
	}
	dx, dy := x1-x0, y1-y0
	len2 := dx*dx + dy*dy
	if len2 < 1e-12 {
		return false
	}
	t := ((tx-x0)*dx + (ty-y0)*dy) / len2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return math.Hypot(x0+t*dx-tx, y0+t*dy-ty) <= reach
}

func (b *缪弹) hit(ctx unit.Context, target unit.Snapshot) {
	dmg := miuShotDamage
	miuMu.Lock()
	ownerBonus := miuBonus[b.owner]
	miuMu.Unlock()
	ownerSpeed := miuBaseSpeed + ownerBonus
	targetSpeed := math.Hypot(target.VX, target.VY)
	if ownerSpeed > targetSpeed {
		dmg += (ownerSpeed - targetSpeed) / speedDivisor
	}

	ctx.Out <- unit.Damage{From: b.owner, To: target.ID, Amount: dmg}

	if target.Kind == KindMiu {
		miuMu.Lock()
		miuLastAttacker[target.ID] = b.owner
		miuMu.Unlock()
	}

	ctx.Out <- unit.FX{
		Name: "miu-shot-hit", Kind: ctx.Kind,
		X: target.X, Y: target.Y, Slot: b.slot,
	}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

// ===================== 缪弧 近战范围 =====================

type 缪弧 struct {
	pass bool
}

func (a *缪弧) Handle(ctx unit.Context, ev unit.Event) {
	if _, ok := ev.(unit.Sense); !ok || a.pass {
		return
	}
	a.pass = true
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
}
