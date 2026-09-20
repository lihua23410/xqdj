// R.缪：透明球。免疫伤害，HP 随存活[缪]数量变化。位置跟随绑定的[缪]。
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
)

const (
	rMiuRadius = 3.0 // 尽可能小，防骗子弹
	rMiuMaxHP  = 5.0

	miuRadius    = 18.0
	miuHP        = 100.0
	miuBaseSpeed = 170.0
	miuVision    = 9999.0
	miuColor     = "#f0f0f0"

	miuDamage   = 5.0
	miuMeleeCD  = 0.35
	miuMeleeGap = 8.0 // 半径之和外的额外近战范围

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
	speedDivisor  = 50.0
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

	// R.缪 战斗机：透明、免伤、HP=存活缪数、跟随绑定缪位置
	p.Register(unit.Spec{
		Kind:    KindRMiu,
		Role:    unit.RoleFighter,
		Radius:  rMiuRadius,
		MaxHP:   rMiuMaxHP,
		Speed:   0,
		Vision:  0, // 自身随从始终可见
		Fighter: true,
		Look:    unit.Look{Color: "#ffffff", Ghost: 500, Overlay: true},
	}, func(unit.SpawnInfo) unit.Actor {
		return &R缪{}
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
}

// ===================== R.缪 战斗机 =====================

type R缪 struct {
	booted  bool
	boundID uint64
}

func (r *R缪) Handle(ctx unit.Context, ev unit.Event) {
	// 免疫所有伤害
	if d, ok := ev.(unit.IncomingDamage); ok {
		unit.BlockHit(ctx, d)
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !r.booted {
		r.booted = true
		r.spawnMinions(ctx, s)
		return // 缪下一帧才出生，本帧跳过同步
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
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
}

func (r *R缪) syncState(ctx unit.Context, s unit.Sense) {
	// 收集存活缪
	var alive []*unit.Snapshot
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID == ctx.ID && o.Kind == KindMiu {
			alive = append(alive, o)
		}
	}

	// HP = 存活缪数量
	ctx.Out <- unit.SetHP{UnitID: ctx.ID, HP: float64(len(alive)), MaxHP: rMiuMaxHP}

	if len(alive) == 0 {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}

	// 检查绑定缪是否存活
	var bound *unit.Snapshot
	for _, o := range alive {
		if o.ID == r.boundID {
			bound = o
			break
		}
	}
	// 切换绑定
	if bound == nil {
		bound = alive[0]
		r.boundID = bound.ID
	}

	ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: bound.X, Y: bound.Y}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
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

	// 检测击杀
	m.checkKills(ctx, s)

	// 同步速度
	m.speed = miuBaseSpeed + m.killBonus
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: m.speed}

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
		// 同主人的其他缪（友军互伤）
		if o.Kind == KindMiu && o.OwnerID == m.ownerID {
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
	for i := range targets {
		t := &targets[i]
		dist := math.Hypot(t.X-s.Self.X, t.Y-s.Self.Y)
		if dist > s.Self.Radius+t.Radius+miuMeleeGap {
			continue
		}
		dmg := miuDamage + m.speedBonus(t)
		ctx.Out <- unit.Damage{From: ctx.ID, To: t.ID, Amount: dmg}

		// 追踪同主人缪的击杀
		if t.Kind == KindMiu && t.OwnerID == m.ownerID {
			miuMu.Lock()
			miuLastAttacker[t.ID] = ctx.ID
			miuMu.Unlock()
		}

		ctx.Out <- unit.FX{
			Name: "miu-melee", Kind: ctx.Kind,
			X: t.X, Y: t.Y, Slot: s.Self.Slot,
		}
		m.meleeReadyAt = s.Time + miuMeleeCD
		return // 一次只打一个
	}
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

func (m *缪) checkKills(ctx unit.Context, s unit.Sense) {
	currentIDs := map[uint64]bool{}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind == KindMiu && o.OwnerID == m.ownerID {
			currentIDs[o.ID] = true
		}
	}

	if m.prevMiuIDs != nil {
		for id := range m.prevMiuIDs {
			if currentIDs[id] {
				continue
			}
			// 此缪已消失，检查是否自己是最后攻击者
			miuMu.Lock()
			if !miuClaimed[id] && miuLastAttacker[id] == ctx.ID {
				miuClaimed[id] = true
				victimBonus := miuBonus[id]
				miuMu.Unlock()

				m.killBonus += killBonusPer + victimBonus
				m.speed = miuBaseSpeed + m.killBonus
				ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: m.speed}
				// 立刻把当前速度拉到新值（minion 无 decelerateLocked）
				sp := math.Hypot(m.vx, m.vy)
				if sp > 1e-6 {
					ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: m.vx / sp * m.speed, VY: m.vy / sp * m.speed}
				}

				ctx.Out <- unit.FX{
					Name: "miu-powerup", Kind: ctx.Kind,
					X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot,
					Amount: m.killBonus,
				}
			} else {
				miuMu.Unlock()
			}
		}
	}
	m.prevMiuIDs = currentIDs
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
		if e.Other.Kind != KindMiu && !unit.Hittable(e.Other, b.slot) {
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
		if o.Kind != KindMiu || o.Slot != b.slot || o.ID == b.owner {
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
