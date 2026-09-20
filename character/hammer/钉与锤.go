// 钉与锤：紫色球。木槌近战、五寸钉钉死、藁人形替身转伤。
package 钉与锤

import (
	"embed"
	"math"
	"sync"
	"xqdj/internal/unit"
)

//go:embed fx 诅咒.png
var assets embed.FS

const (
	KindHammer    = "钉与锤"
	KindHammerArc = "钉与锤弧"
	KindNail      = "钉与锤钉"
	KindDoll      = "钉与锤人偶"
)

const (
	fighterRadius = 18.0
	fighterHP     = 100.0
	fighterSpeed  = 170.0
	fighterVision = 200.0
	fighterColor  = "#9966cc"

	arcRadius  = 20.0
	arcInner   = 18.0
	arcSpanDeg = 120.0
	arcColor   = "#999999"
	arcHitCD   = 0.1
	arcDamage  = 7.0

	hammerCD        = 5.0
	hammerStunDur   = 2.0
	hammerKnockback = 350.0
	swingDamage     = arcDamage + 1
	swingReach      = 56.0

	nailCD        = 8.0
	nailSpeed     = 500.0
	nailRadius    = 5.0
	nailDamage    = 5.0
	nailPushSpeed = 600.0
	nailLockDur   = 1.5
	nailLifetime  = 5.0
	nailBounces   = 3
	nailPushWait  = 1.0

	ritualCD     = 16.0
	ritualFollow = 3.0
	ritualWind   = 0.4
	ritualOrbit  = 56.0

	curseKind  = "咒"
	curseIcon  = "/ball/钉与锤/诅咒.png"
	curseMax   = 7
	curseBurst = 22.0

	dollRadius    = 14.0
	dollSpawnTime = 30.0
	dollColor     = "#c4a060"
)

const (
	nailFly = iota
	nailOrbit
	nailWind
)

// 三个固定人偶位置（场内，120° 均分，半径 140）
var dollPositions = [3][2]float64{
	{140, 0},
	{-70, 121.24},
	{-70, -121.24},
}

func init() {
	p := unit.NewPack(KindHammer, assets)

	// 战斗机
	p.Register(unit.Spec{
		Kind:    KindHammer,
		Role:    unit.RoleFighter,
		Radius:  fighterRadius,
		MaxHP:   fighterHP,
		Speed:   fighterSpeed,
		Vision:  fighterVision,
		Fighter: true,
		Look:    unit.Look{Color: fighterColor, Ghost: 280, VisionRing: true, FX: []string{"hammer"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &钉与锤{}
	})

	// 近战弧（Attach 扇环弹，灰色）
	p.Register(unit.Spec{
		Kind:     KindHammerArc,
		Role:     unit.RoleProjectile,
		Radius:   arcRadius,
		MaxHP:    1,
		Speed:    fighterSpeed,
		Vision:   0,
		Fighter:  false,
		Attach:   true,
		ArcSpan:  unit.Deg(arcSpanDeg),
		ArcInner: arcInner,
		Look:     unit.Look{Color: arcColor, Overlay: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &弧{owner: info.OwnerID, slot: info.Slot}
	})

	// 钉子弹（BreakWalls；人偶只受伤不位移，其它可命中单位会被钉住）
	p.Register(unit.Spec{
		Kind:       KindNail,
		Role:       unit.RoleProjectile,
		Radius:     nailRadius,
		MaxHP:      1,
		Speed:      nailSpeed,
		Vision:     9999,
		Fighter:    false,
		BreakWalls: true,
		Look:       unit.Look{Color: "#c0c0c0", Overlay: true, FX: []string{"nail"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &钉{owner: info.OwnerID, slot: info.Slot, spawnTime: -1}
	})

	// 替身人偶（活随从，不动）
	p.Register(unit.Spec{
		Kind:    KindDoll,
		Role:    unit.RoleMinion,
		Radius:  dollRadius,
		MaxHP:   100, // 占位，生成后由 SetHP 覆盖
		Speed:   0,
		Vision:  0,
		Fighter: false,
		Mortal:  true,
		Look:    unit.Look{Color: dollColor, Overlay: true, FX: []string{"hammer-doll"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &人偶{}
	})
}

// ========== 战斗机 ==========

type 钉与锤 struct {
	arc       unit.AttachState
	enemySlot int

	// 锤击
	hammerReadyAt float64
	hammerArmed   bool
	fsSeq         uint32

	// 钉子弹
	nailReadyAt   float64
	ritualReadyAt float64
	started       bool

	// 人偶
	dollsSpawned  bool
	pendingDollHP float64
	dollHPs       map[uint64]float64
}

func (h *钉与锤) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !h.started {
		h.started = true
		if h.ritualReadyAt == 0 {
			h.ritualReadyAt = s.Time + ritualCD
		}
	}
	h.enemySlot = 1 - s.Self.Slot
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role == unit.RoleFighter && o.Slot != s.Self.Slot {
			rememberFoe(ctx.ID, o.ID)
			break
		}
	}

	h.checkHammer(ctx, s)

	if unit.RearmAttach(s, ctx.ID, KindHammerArc, arcHitCD, &h.arc) {
		h.hammerArmed = s.Time >= h.hammerReadyAt
		unit.SpawnAttach(ctx, s, KindHammerArc)
	}

	if !h.tryRitual(ctx, s) {
		h.tryFireNail(ctx, s)
	}
	h.trySpawnDolls(ctx, s)
	h.trackDolls(ctx, s)
}

// checkHammer：弧消失时，锤击就绪则对近战内可命中单位击退眩晕，并挥锤群攻。
func (h *钉与锤) checkHammer(ctx unit.Context, s unit.Sense) {
	if !h.hammerArmed {
		return
	}
	if unit.HasOwned(s, ctx.ID, KindHammerArc) {
		return
	}
	h.hammerArmed = false
	if s.Time < h.hammerReadyAt {
		return
	}
	var best, any *unit.Snapshot
	bestDist, anyDist := math.Inf(1), math.Inf(1)
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		dx := o.X - s.Self.X
		dy := o.Y - s.Self.Y
		dist := math.Hypot(dx, dy)
		if dist > s.Self.Radius+arcRadius+o.Radius+8 {
			continue
		}
		if dist < anyDist {
			anyDist = dist
			any = o
		}
		if o.Kind == KindDoll {
			continue
		}
		if dist < bestDist {
			bestDist = dist
			best = o
		}
	}
	if any == nil {
		return
	}
	aim := any
	dist := anyDist
	if best != nil {
		aim = best
		dist = bestDist
	}
	if dist < 1e-6 {
		dist = 1
	}
	ux, uy := (aim.X-s.Self.X)/dist, (aim.Y-s.Self.Y)/dist
	h.hammerReadyAt = s.Time + hammerCD
	if best != nil {
		until := s.Time + hammerStunDur
		ctx.Out <- unit.AddFS{
			UnitID: best.ID, DX: ux, DY: uy,
			BaseSpeed: hammerKnockback, OnWall: true,
			ExpiresAt: s.Time + nailPushWait, Token: h.nextFSToken(ctx.ID),
		}
		ctx.Out <- unit.Stun{UnitID: best.ID, Hold: true, Until: until}
	}
	h.swingAround(ctx, s, ux, uy)
}

func (h *钉与锤) swingAround(ctx unit.Context, s unit.Sense, vx, vy float64) {
	ctx.Out <- unit.FX{
		Name: "hammer", Kind: ctx.Kind, UnitID: ctx.ID,
		X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot,
		VX: vx, VY: vy,
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		if math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) > swingReach+o.Radius {
			continue
		}
		hurt(ctx, ctx.ID, ctx.ID, *o, swingDamage)
		if markStacks(*o, curseKind) > 0 {
			applyCurse(ctx, ctx.ID, *o)
		}
	}
}

func (h *钉与锤) tryFireNail(ctx unit.Context, s unit.Sense) {
	if s.Time < h.nailReadyAt {
		return
	}
	target := enemyFighter(s)
	if target == nil {
		return
	}
	dx := target.X - s.Self.X
	dy := target.Y - s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	ux, uy := dx/n, dy/n
	gap := s.Self.Radius + nailRadius + 1.5
	h.swingAround(ctx, s, ux, uy)
	ctx.Out <- unit.Spawn{
		Kind:    KindNail,
		X:       s.Self.X + ux*gap,
		Y:       s.Self.Y + uy*gap,
		VX:      ux * nailSpeed,
		VY:      uy * nailSpeed,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	}
	h.nailReadyAt = s.Time + nailCD
	ctx.Out <- unit.FX{
		Name: "nail-spawn", Kind: ctx.Kind,
		X: s.Self.X + ux*gap, Y: s.Self.Y + uy*gap,
		VX: ux, VY: uy, Slot: s.Self.Slot,
	}
}

func (h *钉与锤) tryRitual(ctx unit.Context, s unit.Sense) bool {
	if s.Time < h.ritualReadyAt {
		return false
	}
	enemy := enemyFighter(s)
	if enemy == nil {
		return false
	}
	for i := 0; i < 3; i++ {
		a := ritualAng(i)
		pushNail(nailJob{orbit: true, ang: a})
		x, y := clampInHex(enemy.X+math.Cos(a)*ritualOrbit, enemy.Y+math.Sin(a)*ritualOrbit, nailRadius)
		ctx.Out <- unit.Spawn{
			Kind:    KindNail,
			X:       x,
			Y:       y,
			OwnerID: ctx.ID,
			Slot:    s.Self.Slot,
		}
	}
	h.ritualReadyAt = s.Time + ritualCD
	return true
}

func (h *钉与锤) trySpawnDolls(ctx unit.Context, s unit.Sense) {
	if h.dollsSpawned || s.Time < dollSpawnTime {
		return
	}
	var enemyHP float64
	if enemy := enemyFighter(s); enemy != nil {
		enemyHP = enemy.HP
	}
	if enemyHP <= 0 {
		return
	}
	h.dollsSpawned = true
	if h.dollHPs == nil {
		h.dollHPs = map[uint64]float64{}
	}
	dollHP := enemyHP / 2
	if dollHP < 1 {
		dollHP = 1
	}
	h.pendingDollHP = dollHP
	for _, pos := range dollPositions {
		ctx.Out <- unit.Spawn{
			Kind:    KindDoll,
			X:       pos[0],
			Y:       pos[1],
			OwnerID: ctx.ID,
			Slot:    h.enemySlot,
		}
		ctx.Out <- unit.FX{
			Name: "doll-spawn", Kind: ctx.Kind,
			X: pos[0], Y: pos[1], Slot: s.Self.Slot,
		}
	}
}

// trackDolls：给新生人偶设 HP。转伤改由近战弧命中时立刻复制，避免等下一帧 Sense。
func (h *钉与锤) trackDolls(ctx unit.Context, s unit.Sense) {
	if h.dollHPs == nil {
		h.dollHPs = map[uint64]float64{}
	}

	if h.pendingDollHP > 0 {
		for i := range s.Nearby {
			o := &s.Nearby[i]
			if o.OwnerID == ctx.ID && o.Kind == KindDoll {
				if _, exists := h.dollHPs[o.ID]; !exists {
					ctx.Out <- unit.SetHP{
						UnitID: o.ID,
						HP:     h.pendingDollHP,
						MaxHP:  h.pendingDollHP,
					}
					h.dollHPs[o.ID] = h.pendingDollHP
				}
			}
		}
		if len(h.dollHPs) >= 3 {
			h.pendingDollHP = 0
		}
	}

	alive := map[uint64]bool{}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID == ctx.ID && o.Kind == KindDoll {
			alive[o.ID] = true
		}
	}
	for id := range h.dollHPs {
		if !alive[id] {
			delete(h.dollHPs, id)
		}
	}
}

// ========== 近战弧 ==========

type 弧 struct {
	owner uint64
	slot  int
}

func (a *弧) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Collision:
		if !unit.EnemyTarget(e, a.slot) {
			return
		}
		hurt(ctx, ctx.ID, a.owner, e.Other, arcDamage)
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

// ========== 钉子弹 ==========

type 钉 struct {
	owner       uint64
	slot        int
	spawnTime   float64
	bounces     int
	phase       int
	ang         float64
	followUntil float64
	shootAt     float64
	lockX       float64
	lockY       float64
	lastX       float64
	lastY       float64
	hasLockPos  bool

	hit     bool
	enemyID uint64
	pushX   float64
	pushY   float64
	hitTime float64
	pushTok uint64
	pinTok  uint64
	struck  map[uint64]bool
	flyX    float64
	flyY    float64
	fsSeq   uint32
}

func (n *钉) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Collision:
		n.onCollision(ctx, e)
	case unit.WallHit:
		n.onWall(ctx, e)
	case unit.Sense:
		n.onSense(ctx, e)
	}
}

func (n *钉) onWall(ctx unit.Context, e unit.WallHit) {
	if n.hit || n.phase != nailFly {
		return
	}
	n.bounces++
	if n.bounces >= nailBounces {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	ux, uy := n.flyX, n.flyY
	if math.Hypot(ux, uy) < 1e-6 {
		ux, uy = -e.NX, -e.NY
	}
	rx, ry := reflectDir(ux, uy, e.NX, e.NY)
	n.steer(ctx, rx, ry)
}

func (n *钉) steer(ctx unit.Context, ux, uy float64) {
	d := math.Hypot(ux, uy)
	if d < 1e-6 {
		ux, uy, d = 1, 0, 1
	}
	n.flyX, n.flyY = ux/d*nailSpeed, uy/d*nailSpeed
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: n.flyX, VY: n.flyY}
}

func (n *钉) onCollision(ctx unit.Context, e unit.Collision) {
	if n.hit || n.phase != nailFly || e.Other.ID == n.owner {
		return
	}
	if !unit.Hittable(e.Other, n.slot) {
		return
	}
	if n.alreadyStruck(e.Other.ID) {
		return
	}
	n.markStruck(e.Other.ID)
	hurt(ctx, n.owner, n.owner, e.Other, nailDamage)
	applyCurse(ctx, n.owner, e.Other)
	ctx.Out <- unit.FX{
		Name: "nail-hit", Kind: ctx.Kind,
		X: e.Other.X, Y: e.Other.Y, Slot: n.slot,
	}
	if e.Other.Kind == KindDoll {
		return
	}
	n.hit = true
	n.enemyID = e.Other.ID
	n.hitTime = e.Time
	// 推送方向 = 钉子飞行方向：沿命中朝向把敌人钉出去。
	// 旧版用 nearestWallDir（敌人到最近场边），导致位移和钉子朝向不一致。
	px, py := n.flyX, n.flyY
	if d := math.Hypot(px, py); d > 1e-6 {
		px, py = px/d, py/d
	} else {
		px, py = -e.NX, -e.NY
		if d := math.Hypot(px, py); d > 1e-6 {
			px, py = px/d, py/d
		} else {
			px, py = 1, 0
		}
	}
	n.pushX, n.pushY = px, py
	until := e.Time + nailLockDur
	n.pushTok = n.nextFSToken(ctx.ID)
	n.pinTok = n.nextFSToken(ctx.ID)
	ctx.Out <- unit.AddFS{
		UnitID: n.enemyID, DX: n.pushX, DY: n.pushY,
		BaseSpeed: nailPushSpeed, OnWall: true,
		ExpiresAt: e.Time + nailPushWait, Token: n.pushTok,
	}
	ctx.Out <- unit.AddFSComponent{
		UnitID: n.enemyID, Zone: unit.FSZoneM, Token: n.pinTok,
		Value: 0, ExpiresAt: until,
	}
	ctx.Out <- unit.Stun{UnitID: n.enemyID, Hold: true, Until: until}
	n.stickTo(ctx, &e.Other, n.pushX*nailPushSpeed, n.pushY*nailPushSpeed)
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
}

func (n *钉) onSense(ctx unit.Context, s unit.Sense) {
	n.boot(ctx, s)
	if n.phase == nailOrbit || n.phase == nailWind {
		n.tickOrbit(ctx, s)
		return
	}
	if !n.hit {
		if sp := math.Hypot(s.Self.VX, s.Self.VY); sp > 8 {
			n.flyX, n.flyY = s.Self.VX, s.Self.VY
		}
		if s.Time > n.spawnTime+nailLifetime {
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		}
		return
	}
	var enemy *unit.Snapshot
	for i := range s.Nearby {
		if s.Nearby[i].ID == n.enemyID {
			enemy = &s.Nearby[i]
			break
		}
	}
	if enemy == nil {
		n.clearPin(ctx)
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	n.stickTo(ctx, enemy, enemy.VX, enemy.VY)
	if s.Time+1e-9 >= n.hitTime+nailLockDur {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

func (n *钉) boot(ctx unit.Context, s unit.Sense) {
	if n.spawnTime >= 0 {
		return
	}
	n.spawnTime = s.Time
	job, ok := popNail()
	if !ok || !job.orbit {
		return
	}
	n.phase = nailOrbit
	n.ang = job.ang
	n.followUntil = s.Time + ritualFollow
	n.shootAt = s.Time + ritualFollow + ritualWind
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
}

func (n *钉) tickOrbit(ctx unit.Context, s unit.Sense) {
	if enemy := enemyFighter(s); enemy != nil {
		n.lastX, n.lastY = enemy.X, enemy.Y
		n.hasLockPos = true
	}
	if n.phase == nailOrbit {
		if s.Time+1e-9 >= n.followUntil {
			n.phase = nailWind
			n.lockX, n.lockY = n.lastX, n.lastY
			ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
			return
		}
		if n.hasLockPos {
			n.placeOrbit(ctx, n.lastX, n.lastY)
		}
		return
	}
	if s.Time+1e-9 < n.shootAt {
		return
	}
	n.phase = nailFly
	n.spawnTime = s.Time
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: false}
	n.steer(ctx, n.lockX-s.Self.X, n.lockY-s.Self.Y)
}

func (n *钉) placeOrbit(ctx unit.Context, cx, cy float64) {
	x, y := clampInHex(cx+math.Cos(n.ang)*ritualOrbit, cy+math.Sin(n.ang)*ritualOrbit, nailRadius)
	ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: x, Y: y}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
}

func (n *钉) stickTo(ctx unit.Context, e *unit.Snapshot, vx, vy float64) {
	if e == nil {
		return
	}
	gap := e.Radius * 0.2
	ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: e.X + n.pushX*gap, Y: e.Y + n.pushY*gap}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: vx, VY: vy}
}

func (n *钉) alreadyStruck(id uint64) bool {
	return n.struck != nil && n.struck[id]
}

func (n *钉) markStruck(id uint64) {
	if n.struck == nil {
		n.struck = map[uint64]bool{}
	}
	n.struck[id] = true
}

// ========== 替身人偶 ==========

type 人偶 struct{}

func (d *人偶) Handle(unit.Context, unit.Event) {
	// 被动单位，不处理任何事件
}

// ========== 工具函数 ==========

func hurt(ctx unit.Context, from, owner uint64, u unit.Snapshot, amount float64) {
	if amount <= 0 {
		return
	}
	ctx.Out <- unit.Damage{From: from, To: u.ID, Amount: amount}
	copyIfDoll(ctx, owner, u, amount)
}

func copyIfDoll(ctx unit.Context, owner uint64, u unit.Snapshot, amount float64) {
	if u.Kind != KindDoll || amount <= 0 || u.HP <= 0 {
		return
	}
	amt := u.HP
	if amt > amount {
		amt = amount
	}
	if eid := foeOfOwner(owner); eid != 0 {
		ctx.Out <- unit.Damage{From: owner, To: eid, Amount: amt}
	}
}

func applyCurse(ctx unit.Context, from uint64, u unit.Snapshot) {
	n := markStacks(u, curseKind)
	ctx.Out <- unit.StackMark{UnitID: u.ID, Kind: curseKind, Delta: 1, Icon: curseIcon}
	if n+1 < curseMax {
		return
	}
	hurt(ctx, from, from, u, curseBurst)
	ctx.Out <- unit.ClearMarks{UnitID: u.ID, Kind: curseKind}
}

func markStacks(u unit.Snapshot, kind string) int {
	for _, m := range u.Marks {
		if m.Kind == kind {
			return m.Stacks
		}
	}
	return 0
}

// ritualAng：三钉绕敌。i=0 在正上方，尖朝下（屏幕 0°），然后每 120° 一枚，尖始终朝圆心。
func ritualAng(i int) float64 {
	return math.Pi/2 + float64(i)*2*math.Pi/3
}

func enemyFighter(s unit.Sense) *unit.Snapshot {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role == unit.RoleFighter && o.Slot != s.Self.Slot {
			return o
		}
	}
	return nil
}

func clampInHex(x, y, r float64) (float64, float64) {
	return unit.LiveField().Clamp(x, y, r)
}

func (h *钉与锤) nextFSToken(owner uint64) uint64 {
	return nextFSToken(owner, &h.fsSeq)
}

func (n *钉) nextFSToken(owner uint64) uint64 {
	return nextFSToken(owner, &n.fsSeq)
}

func nextFSToken(owner uint64, seq *uint32) uint64 {
	*seq++
	return owner<<32 | uint64(*seq)
}

func (n *钉) clearPin(ctx unit.Context) {
	if n.enemyID == 0 {
		return
	}
	if n.pushTok != 0 {
		ctx.Out <- unit.RemoveFS{UnitID: n.enemyID, Token: n.pushTok}
		n.pushTok = 0
	}
	if n.pinTok != 0 {
		ctx.Out <- unit.RemoveFSComponent{UnitID: n.enemyID, Token: n.pinTok}
		n.pinTok = 0
	}
	ctx.Out <- unit.Stun{UnitID: n.enemyID, Hold: false}
}

type nailJob struct {
	orbit bool
	ang   float64
}

var (
	foeMu sync.Mutex
	foeOf = map[uint64]uint64{}

	nailMu sync.Mutex
	nailQ  []nailJob
)

func rememberFoe(owner, enemy uint64) {
	if owner == 0 || enemy == 0 {
		return
	}
	foeMu.Lock()
	foeOf[owner] = enemy
	foeMu.Unlock()
}

func foeOfOwner(owner uint64) uint64 {
	foeMu.Lock()
	defer foeMu.Unlock()
	return foeOf[owner]
}

func pushNail(job nailJob) {
	nailMu.Lock()
	nailQ = append(nailQ, job)
	nailMu.Unlock()
}

func popNail() (nailJob, bool) {
	nailMu.Lock()
	defer nailMu.Unlock()
	if len(nailQ) == 0 {
		return nailJob{}, false
	}
	job := nailQ[0]
	nailQ = nailQ[1:]
	return job, true
}

func resetNailJobs() {
	nailMu.Lock()
	nailQ = nil
	nailMu.Unlock()
}

func reflectDir(ux, uy, nx, ny float64) (float64, float64) {
	dot := ux*nx + uy*ny
	rx, ry := ux-2*dot*nx, uy-2*dot*ny
	n := math.Hypot(rx, ry)
	if n < 1e-6 {
		return ux, uy
	}
	return rx / n, ry / n
}
