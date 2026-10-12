// 分身者(百战)：本体不出现；四只战影编队出伤；战影消失折损本体。
package 分身者_百战

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const (
	KindDoppelVeteran = "分身者(百战)"
	KindShadow        = "战影"
)

const (
	bodyHP     = 100.0
	bodyVision = 9999.0
	bodyColor  = "#3a4a5a"

	shadowRadius = 16.0
	shadowHP     = 50.0
	shadowCruise = 170.0
	shadowAim    = 15
	shadowColor  = "#4a5a6a"
	shadowSpread = 48.0
	shadowCount  = 4
	levyAmount   = 25.0

	reformDur = 2.0

	weiRadius    = 70.0
	weiPeriod    = 1.2
	weiOrbitDur  = 2.4
	weiPauseDur  = 0.6
	weiDashSp    = 720.0
	weiOvershoot = 180.0
	weiDamage    = 8.0
	weiGatherMax = 2.5

	zhouRadius    = 40.0
	zhouPeriod    = 1.0
	zhouStand     = 0.8
	zhouDashSp    = 640.0
	zhouDashLen   = 420.0 // 原 280，延长 50%
	zhouDamage    = 7.0
	zhouGatherMax = 2.5 // 墙边走不齐则超时强行进准备

	dieStackR    = 2.0
	dieGatherSp  = 420.0
	dieWait      = 0.3
	dieGap       = 1.0
	dieDashSp    = 1100.0
	dieDashLen   = 320.0
	dieDamage    = 8.0
	dieGatherMax = 2.5
	deathHeal    = 15.0 // 任一战影消失时，其余各回 15

	orbitArriveR  = 12.0 // 走到环位算到齐
	weiOrbitDmg   = 3.0  // 招1环刮
	zhouOrbitDmg  = 7.0  // 招2外围三球转圈出伤
	orbitScrapeCD = 0.35

	timeEps = 1e-9
)

func init() {
	p := unit.NewPack(KindDoppelVeteran, assets)
	p.Register(unit.Spec{
		Kind:     KindDoppelVeteran,
		Role:     unit.RoleFighter,
		Radius:   0,
		MaxHP:    bodyHP,
		Speed:    0,
		Vision:   bodyVision,
		Fighter:  true,
		Nonsolid: true,
		Look:     unit.Look{Color: bodyColor},
	}, func(unit.SpawnInfo) unit.Actor {
		return &百战{}
	})
	p.Register(unit.Spec{
		Kind:        KindShadow,
		Role:        unit.RoleMinion,
		Radius:      shadowRadius,
		MaxHP:       shadowHP,
		Speed:       shadowCruise,
		Vision:      0,
		Mortal:      true,
		Cruise:      true,
		AimPriority: shadowAim,
		Look: unit.Look{
			Color: shadowColor,
			FX:    []string{"shadow"},
			Ghost: 400, // 冲刺残影（引擎 dashing）
		},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &战影{owner: info.OwnerID, slot: info.Slot}
	})
}

const (
	phaseBoot = iota
	phaseReform
	phaseWeiGather
	phaseWeiOrbit
	phaseWeiPause
	phaseWeiDash
	phaseZhouGather
	phaseZhouAim
	phaseZhouDash
	phaseDieGather
	phaseDieWait
	phaseDieVolley
)

type 百战 struct {
	booted     bool
	slot       int
	phase      int
	phaseUntil float64
	rng        *rand.Rand
	moveBag    []int
	lastT      float64

	aimX, aimY   float64
	liveX, liveY float64
	hasAim       bool

	orbitAng     map[uint64]float64
	orbitHitAt   map[uint64]map[uint64]float64 // 战影 → 目标 → 下次可刮
	axisID       uint64
	stackID      uint64
	stackX       float64
	stackY       float64
	stackFixed   bool // 招3停住后钉死叠点，不再跟活坐标追
	centerFixed  bool // 招1停转后钉死圆心
	centerX      float64
	centerY      float64
	lockID       uint64 // 围刺锁定的敌方单位；非 0 时不换锁
	order        []uint64
	shotIdx      int
	nextShot     float64
	dashIDs      []uint64
	gatherFrom   float64
	knownShadow  map[uint64]struct{}

	passing  map[uint64]bool
	standing map[uint64]bool
}

func (a *百战) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	a.onSense(ctx, s)
}

func (a *百战) onSense(ctx unit.Context, s unit.Sense) {
	a.slot = s.Self.Slot
	if a.orbitAng == nil {
		a.orbitAng = map[uint64]float64{}
	}
	if a.passing == nil {
		a.passing = map[uint64]bool{}
	}
	if a.standing == nil {
		a.standing = map[uint64]bool{}
	}
	dt := s.Time - a.lastT
	if dt < 0 {
		dt = 0
	}
	a.lastT = s.Time

	if !a.booted {
		a.booted = true
		a.phase = phaseBoot
		// 与狼人月亮同一套：运行时 Uint64 搅种，避免槽位固定种子抽偏
		a.rng = rand.New(rand.NewPCG(rand.Uint64()^ctx.ID, rand.Uint64()))
		ctx.Out <- unit.NoHealthNumbers{UnitID: ctx.ID, Hold: true}
		unit.SetAim(ctx, ctx.ID, 0)
		spawnShadows(ctx, s)
		return
	}
	if a.rng == nil {
		a.rng = rand.New(rand.NewPCG(rand.Uint64()^ctx.ID, rand.Uint64()))
	}

	a.refreshAim(s)
	shadows := a.listShadows(s)
	a.syncShadowDeaths(ctx, shadows)
	if a.phase > phaseReform && len(shadows) == 0 {
		a.clearDashWait()
		a.beginReform(ctx, s.Time, shadows)
		return
	}

	switch a.phase {
	case phaseBoot:
		if len(shadows) == 0 {
			return
		}
		a.beginReform(ctx, s.Time, shadows)
	case phaseReform:
		if s.Time+timeEps < a.phaseUntil {
			return
		}
		if !a.hasAim || len(shadows) == 0 {
			a.phaseUntil = s.Time + reformDur
			return
		}
		a.pickMove(ctx, s, shadows)
	case phaseWeiGather:
		a.tickWeiGather(ctx, s, shadows, dt)
	case phaseWeiOrbit:
		a.tickWeiOrbit(ctx, s, shadows, dt)
	case phaseWeiPause:
		a.tickWeiPause(ctx, s, shadows)
	case phaseWeiDash:
		a.driveDashes(ctx, s)
		a.tickWaitDashes(ctx, s, shadows)
	case phaseZhouGather:
		a.tickZhouGather(ctx, s, shadows, dt)
	case phaseZhouAim:
		a.tickZhouAim(ctx, s, shadows, dt)
	case phaseZhouDash:
		a.driveDashes(ctx, s)
		a.tickZhouDash(ctx, s, shadows, dt)
	case phaseDieGather:
		a.tickDieGather(ctx, s, shadows, dt)
	case phaseDieWait:
		a.tickDieWait(ctx, s, shadows)
	case phaseDieVolley:
		a.tickDieVolley(ctx, s, shadows)
	}
}

func (a *百战) beginReform(ctx unit.Context, now float64, shadows []unit.Snapshot) {
	a.clearPasses(ctx, shadows)
	a.clearStands(ctx, shadows)
	clearAllStackGuards()
	a.kickCruise(ctx, shadows)
	a.phase = phaseReform
	a.phaseUntil = now + reformDur
	a.clearDashWait()
	a.axisID = 0
	a.stackID = 0
	a.stackFixed = false
	a.centerFixed = false
	a.lockID = 0
	a.order = nil
	a.shotIdx = 0
	a.gatherFrom = 0
	a.orbitHitAt = nil
}

func (a *百战) pickMove(ctx unit.Context, s unit.Sense, shadows []unit.Snapshot) {
	move := a.drawMove()
	switch move {
	case 0:
		a.startWei(ctx, s, shadows)
	case 1:
		a.startZhou(ctx, s, shadows)
	default:
		a.startDie(ctx, s, shadows)
	}
}

// drawMove 三招洗牌袋：抽空再洗，避免某一招长期抽不到。
func (a *百战) drawMove() int {
	if a.rng == nil {
		a.rng = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	}
	if len(a.moveBag) == 0 {
		a.moveBag = []int{0, 1, 2}
		a.rng.Shuffle(len(a.moveBag), func(i, j int) {
			a.moveBag[i], a.moveBag[j] = a.moveBag[j], a.moveBag[i]
		})
	}
	n := len(a.moveBag) - 1
	move := a.moveBag[n]
	a.moveBag = a.moveBag[:n]
	return move
}

func (a *百战) refreshAim(s unit.Sense) {
	if a.lockID != 0 {
		for i := range s.Nearby {
			o := &s.Nearby[i]
			if o.ID != a.lockID {
				continue
			}
			a.aimX, a.aimY = o.X, o.Y
			a.liveX, a.liveY = o.X, o.Y
			a.hasAim = true
			return
		}
		// 锁着的人暂时不在 Nearby：保住最后坐标，不改锁别人
		if a.hasAim {
			a.liveX, a.liveY = a.aimX, a.aimY
		}
		return
	}
	if en := unit.Seek(s); en != nil {
		a.aimX, a.aimY = en.X, en.Y
		a.liveX, a.liveY = en.X, en.Y
		a.hasAim = true
		return
	}
	// 招中途丢目标：保留 aim；live 跟着 aim 冻住
	if a.hasAim {
		a.liveX, a.liveY = a.aimX, a.aimY
	}
}

func (a *百战) listShadows(s unit.Sense) []unit.Snapshot {
	out := make([]unit.Snapshot, 0, shadowCount)
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID == s.Self.ID && o.Kind == KindShadow {
			out = append(out, *o)
		}
	}
	return out
}

// syncShadowDeaths：战影从场上消失时，本体折损 + 其余互回。
// 折损放在这里结算，避免战影侧用 Sense 血量判死时与 Confirm 不同步而漏折。
func (a *百战) syncShadowDeaths(ctx unit.Context, shadows []unit.Snapshot) {
	if a.knownShadow == nil {
		a.knownShadow = map[uint64]struct{}{}
	}
	live := make(map[uint64]struct{}, len(shadows))
	for _, sh := range shadows {
		live[sh.ID] = struct{}{}
	}
	for id := range a.knownShadow {
		if _, ok := live[id]; ok {
			continue
		}
		ctx.Out <- unit.Damage{
			From: ctx.ID, To: ctx.ID, Amount: levyAmount, NoFreeze: true,
		}
		for _, sh := range shadows {
			ctx.Out <- unit.Heal{UnitID: sh.ID, Amount: deathHeal}
		}
	}
	a.knownShadow = live
}

func (a *百战) center() (float64, float64) {
	if a.centerFixed {
		return a.centerX, a.centerY
	}
	if a.hasAim {
		return a.liveX, a.liveY
	}
	return a.aimX, a.aimY
}

func (a *百战) freezeCenter() {
	a.centerX, a.centerY = a.center()
	a.centerFixed = true
}

type ringSpot struct {
	x, y float64
	r    float64
}

// scrapeOrbitAt 用环上当下坐标结算重叠出伤（不用 Sense 旧快照，避免 Teleport 后打空）。
func (a *百战) scrapeOrbitAt(ctx unit.Context, s unit.Sense, spots map[uint64]ringSpot, amount float64) {
	if len(spots) == 0 || amount <= 0 {
		return
	}
	if a.orbitHitAt == nil {
		a.orbitHitAt = map[uint64]map[uint64]float64{}
	}
	for id, sp := range spots {
		hit := a.orbitHitAt[id]
		if hit == nil {
			hit = map[uint64]float64{}
			a.orbitHitAt[id] = hit
		}
		for i := range s.Nearby {
			o := &s.Nearby[i]
			if !unit.Hittable(*o, a.slot) {
				continue
			}
			if math.Hypot(o.X-sp.x, o.Y-sp.y) > sp.r+o.Radius {
				continue
			}
			if s.Time+timeEps < hit[o.ID] {
				continue
			}
			hit[o.ID] = s.Time + orbitScrapeCD
			ctx.Out <- unit.Damage{From: id, To: o.ID, Amount: amount}
		}
	}
}

func (a *百战) setPass(ctx unit.Context, id uint64, hold bool) {
	if a.passing[id] == hold {
		return
	}
	a.passing[id] = hold
	ctx.Out <- unit.Pass{UnitID: id, Hold: hold}
}

func (a *百战) setStand(ctx unit.Context, id uint64, hold bool) {
	if a.standing[id] == hold {
		return
	}
	a.standing[id] = hold
	ctx.Out <- unit.Stand{UnitID: id, Hold: hold}
}

func (a *百战) clearPasses(ctx unit.Context, shadows []unit.Snapshot) {
	live := map[uint64]struct{}{}
	for _, sh := range shadows {
		live[sh.ID] = struct{}{}
		a.setPass(ctx, sh.ID, false)
	}
	for id := range a.passing {
		if _, ok := live[id]; !ok {
			delete(a.passing, id)
		}
	}
}

func (a *百战) clearStands(ctx unit.Context, shadows []unit.Snapshot) {
	live := map[uint64]struct{}{}
	for _, sh := range shadows {
		live[sh.ID] = struct{}{}
		a.setStand(ctx, sh.ID, false)
	}
	for id := range a.standing {
		if _, ok := live[id]; !ok {
			delete(a.standing, id)
		}
	}
}

// kickCruise：重整时一律压回巡航 170，避免环转切向 / 冲刺残留被带进自由飞。
func (a *百战) kickCruise(ctx unit.Context, shadows []unit.Snapshot) {
	n := len(shadows)
	for i, sh := range shadows {
		sp := math.Hypot(sh.VX, sh.VY)
		var ux, uy float64
		if sp >= 1e-3 {
			ux, uy = sh.VX/sp, sh.VY/sp
		} else {
			ang := float64(i) * 2 * math.Pi / float64(n+1)
			ux, uy = math.Cos(ang), math.Sin(ang)
		}
		ctx.Out <- unit.SetVelocity{
			UnitID: sh.ID,
			VX:     ux * shadowCruise,
			VY:     uy * shadowCruise,
		}
	}
}

func (a *百战) clearDashWait() {
	for _, id := range a.dashIDs {
		clearDash(id)
	}
	a.dashIDs = nil
}

func (a *百战) driveDashes(ctx unit.Context, s unit.Sense) {
	for _, id := range a.dashIDs {
		j := getDash(id)
		if j == nil {
			continue
		}
		j.tick(ctx, id, s)
	}
}

func (a *百战) tickWaitDashes(ctx unit.Context, s unit.Sense, shadows []unit.Snapshot) {
	if !dashesDone(a.dashIDs) {
		return
	}
	a.beginReform(ctx, s.Time, shadows)
}

func (a *百战) assignOrbitAngles(shadows []unit.Snapshot) {
	a.orbitAng = map[uint64]float64{}
	n := len(shadows)
	if n == 0 {
		return
	}
	for i, sh := range shadows {
		a.orbitAng[sh.ID] = float64(i) * 2 * math.Pi / float64(n)
	}
}

func placeOnOrbit(ctx unit.Context, id uint64, cx, cy, ang, radius, omega float64) (x, y float64) {
	x = cx + math.Cos(ang)*radius
	y = cy + math.Sin(ang)*radius
	// 切向速度，供拖尾/撞墙观感；位置每拍钉死
	tx := -math.Sin(ang) * omega * radius
	ty := math.Cos(ang) * omega * radius
	ctx.Out <- unit.Teleport{UnitID: id, X: x, Y: y}
	ctx.Out <- unit.SetVelocity{UnitID: id, VX: tx, VY: ty}
	return x, y
}

func spawnShadows(ctx unit.Context, s unit.Sense) {
	for i := 0; i < shadowCount; i++ {
		ang := float64(i) * 2 * math.Pi / float64(shadowCount)
		dx, dy := math.Cos(ang), math.Sin(ang)
		ctx.Out <- unit.Spawn{
			Kind:    KindShadow,
			X:       s.Self.X + dx*shadowSpread,
			Y:       s.Self.Y + dy*shadowSpread,
			VX:      -dy * shadowCruise,
			VY:      dx * shadowCruise,
			OwnerID: ctx.ID,
			Slot:    s.Self.Slot,
		}
	}
}
