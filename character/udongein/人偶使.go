package 人偶使

import (
	"embed"
	"math"
	"math/rand/v2"
	"sync"
	"xqdj/internal/unit"
)

//go:embed fx 素材
var assets embed.FS

const KindNingyushi = "人偶使"
const KindNingyushiDoll = "人偶使偶"
const KindNingyushiBomb = "人偶使炸弹"
const KindNingyushiShot = "人偶使弹"
const KindNingyushiBeam = "人偶使激光"

const (
	ningyushiRadius = 18.0
	ningyushiSpeed  = 152.0
	ningyushiHP     = 100.0
	ningyushiVision = 9999.0
	ningyushiColor  = "#fafab2"

	dollRadius = 10.0
	dollColor  = "#f0c43c"

	energyMax  = 5.0
	energyTick = 0.8

	cardSlots = 5
	cardFill  = 0.1
	deckSize  = 20

	frontCostDiv = 10.0 // 有灵力时：每 10 点初始伤害耗 1 灵力
	frontDR      = 0.5  // 有灵力减伤后剩 50%
	emptyHurt    = 1.25 // 空灵力 / 破防挨打 ×1.25

	dollReach = 54.0
	dollFar   = 108.0

	fanR       = 36.0
	atkFanDeg  = 120.0
	skFanDeg   = 100.0
	atkFanHits = 6
	skFanHits  = 7
	atkFanDmg  = 1.4
	skFanDmg   = 1.12
	atkFanWind = 0.1
	atkFanLife = 1.0

	rectW       = 72.0
	rectH       = 36.0
	atkRectWind = 0.2
	atkRectLife = 1.5
	atkRectDmg  = 9.8
	knockDist   = 108.0
	rectStun    = 1.0

	ellipseRX = 36.0
	ellipseRY = 18.0
	chaseSp   = 100.0
	chaseLife = 3.0
	burstHits = 4
	burstLife = 0.5
	burstDmg  = 1.4

	placeWind  = 0.8
	placeShots = 6
	placeLife  = 1.0
	placeDmg   = 1.4
	shotSpeed  = 240.0
	shotRadius = 6.0

	recallLife = 0.4
	deployLife = 0.2

	spell24Wind = 0.2
	spell24Life = 2.0
	spell24Hits = 4
	spell24N    = 6
	spell24Ring = 24.0
	spell24Len  = 108.0
	spell24Rush = ningyushiSpeed * 2.5

	laserHalf = 10.0
	laserLen  = 560.0
	laserHold = 0.4

	houraiLife  = 1.5
	houraiHits  = 16
	houraiBeams = 4

	bombR       = 12.0
	demonR      = 54.0
	demonDmg    = 28.0
	demonSp     = 108.0 / 1.2
	bounceSp    = 75.0
	orbitR      = 200.0
	orbitN      = 8
	orbitLife   = 3.0
	orbitExpand = 1.2
	houraiDmg   = 1.4
)

var (
	owners sync.Map
)

func init() {
	p := unit.NewPack(KindNingyushi, assets)
	p.Register(unit.Spec{
		Kind:    KindNingyushi,
		Role:    unit.RoleFighter,
		Radius:  ningyushiRadius,
		MaxHP:   ningyushiHP,
		Speed:   ningyushiSpeed,
		Vision:  ningyushiVision,
		Fighter: true,
		Look:    unit.Look{Color: ningyushiColor, Glow: true, FX: []string{"ningyushi"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &人偶使{energy: energyMax, energyCap: energyMax}
	})
	p.Register(unit.Spec{
		Kind:    KindNingyushiDoll,
		Role:    unit.RoleHelper,
		Radius:  dollRadius,
		MaxHP:   1,
		Speed:   0,
		Vision:  ningyushiVision,
		Fighter: false,
		Look:    unit.Look{Color: dollColor, Overlay: true, FX: []string{"ningyushi-doll"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return newDoll(info)
	})
	p.Register(unit.Spec{
		Kind:    KindNingyushiBomb,
		Role:    unit.RoleProjectile,
		Radius:  bombR,
		MaxHP:   1,
		Speed:   bounceSp,
		Vision:  ningyushiVision,
		Fighter: false,
		Look:    unit.Look{Color: dollColor, Overlay: true, FX: []string{"ningyushi-bomb"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return newBomb(info)
	})
	p.Register(unit.Spec{
		Kind:    KindNingyushiShot,
		Role:    unit.RoleProjectile,
		Radius:  shotRadius,
		MaxHP:   1,
		Speed:   shotSpeed,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: "#ffe08a", Trail: true, FX: []string{"ningyushi-shot"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &人偶弹{owner: info.OwnerID, slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:    KindNingyushiBeam,
		Role:    unit.RoleHelper,
		Radius:  1,
		MaxHP:   1,
		Speed:   0,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: "#ffe9a0", Overlay: true, FX: []string{"ningyushi-beam"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &激光{}
	})
}

type 人偶使 struct {
	cardMu sync.Mutex

	energy    float64
	energyCap float64
	regenAcc  float64
	spentAcc  float64
	drained   bool
	lastT     float64
	booted    bool
	lockUntil float64

	locked      bool
	force       uint8
	playingCard bool
	level       [SkillCount]int

	hx, hy float64
	x, y   float64
	slot   int
	nearby []unit.Snapshot

	idleDolls int
	liveDolls int

	job      job
	abortGen int

	stunID     uint64
	stunUntil  float64
	stunVX     float64
	stunVY     float64
	stunCruise float64
	stunning   bool

	deck     []uint8
	hand     []uint8
	progress float64

	rng *rand.Rand
}

type job struct {
	kind   uint8
	until  float64
	next   float64
	left   int
	ox, oy float64
	ux, uy float64
	dmg    float64
	span   float64
	reach  float64
	wind   float64
	armed  bool
	knock  bool
	paths  [][2]float64
	idx    int
	selfUX float64
	selfUY float64
	gap    float64
}

func (a *人偶使) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		a.onHit(ctx, e)
	case unit.Collision:
		a.onCollide(ctx, e)
	case unit.WallHit:
		a.onWall(ctx, e)
	case unit.Sense:
		a.onSense(ctx, e)
	}
}

func (a *人偶使) onWall(ctx unit.Context, e unit.WallHit) {
	a.endRush(ctx, e.Time)
}

func (a *人偶使) onCollide(ctx unit.Context, e unit.Collision) {
	if a.job.kind != SkillN24 || a.job.left <= 0 {
		return
	}
	if !unit.EnemyTarget(e, a.slot) {
		return
	}
	deal(ctx, ctx.ID, e.Other.ID, float64(a.job.left)*a.job.dmg)
	a.endRush(ctx, e.Time)
}

func (a *人偶使) endRush(ctx unit.Context, now float64) {
	if a.job.kind != SkillN24 {
		return
	}
	a.restoreCruise(ctx)
	a.job = job{}
	a.lockUntil = now
	a.abortGen++
}

func (a *人偶使) enterBreak() {
	if a.drained {
		return
	}
	a.drained = true
	a.energyCap = energyMax - 1
	if a.energyCap < 1 {
		a.energyCap = 1
	}
}

func (a *人偶使) dumpEnergy() {
	a.energy = 0
	a.enterBreak()
}

func (a *人偶使) onHit(ctx unit.Context, d unit.IncomingDamage) {
	raw := d.Amount
	amt := raw
	if a.energy < 1e-9 {
		a.enterBreak()
		amt *= emptyHurt
	} else {
		cost := raw / frontCostDiv
		if cost <= a.energy+1e-9 {
			a.spend(cost)
			amt = raw * frontDR
			if a.energy < 1e-9 {
				a.energy = 0
				a.enterBreak()
			}
		} else {
			a.dumpEnergy()
			amt *= emptyHurt
		}
	}
	if amt < 0.05 {
		amt = 0.05
	}
	ctx.Out <- unit.ConfirmDamage{Token: d.Token, UnitID: ctx.ID, Amount: amt}
	a.charge()
}

func (a *人偶使) onSense(ctx unit.Context, s unit.Sense) {
	a.slot = s.Self.Slot
	a.x, a.y = s.Self.X, s.Self.Y
	if n := math.Hypot(s.Self.VX, s.Self.VY); n > 1e-6 {
		a.hx, a.hy = s.Self.VX/n, s.Self.VY/n
	}
	a.nearby = append(a.nearby[:0], s.Nearby...)
	a.countDolls(ctx.ID)
	if a.booted {
		if !a.lockedOut(s.Time) {
			dt := s.Time - a.lastT
			if dt < 0 {
				dt = 0
			}
			if a.lockUntil > 0 && a.lastT < a.lockUntil {
				dt = s.Time - a.lockUntil
			}
			a.regen(dt)
		}
		a.lastT = s.Time
	} else {
		a.boot(ctx, s.Time)
	}
	owners.Store(ctx.ID, a)
	a.emitHUD(ctx, s)
	a.holdStun(ctx, s)
	a.tickJob(ctx, s)
	if a.lockedOut(s.Time) {
		return
	}
	enemy := enemyOf(s)
	if a.tryCard(ctx, s, enemy) {
		return
	}
	sk := a.pickCast(s, enemy)
	if sk == SkillNone {
		return
	}
	a.invokePaid(ctx, s, enemy, sk, false)
}

func (a *人偶使) boot(ctx unit.Context, now float64) {
	a.booted = true
	a.lastT = now
	if a.energyCap <= 0 {
		a.energyCap = energyMax
	}
	a.ensureRNG(ctx)
	a.cardMu.Lock()
	if a.deck == nil {
		a.initDeckLocked()
	}
	a.cardMu.Unlock()
	ctx.Out <- unit.NoFrameFreeze{UnitID: ctx.ID, Hold: true}
}

func (a *人偶使) lockedOut(now float64) bool {
	return now+1e-9 < a.lockUntil
}

func (a *人偶使) regen(dt float64) {
	if dt <= 0 || a.energy+1e-9 >= a.energyCap {
		return
	}
	a.regenAcc += dt
	for a.regenAcc+1e-9 >= energyTick {
		a.regenAcc -= energyTick
		a.energy += 1
		if a.energy > a.energyCap {
			a.energy = a.energyCap
		}
		if a.energy+1e-9 >= a.energyCap {
			a.drained = false
			a.energyCap = energyMax
			a.regenAcc = 0
			return
		}
	}
}

func (a *人偶使) spend(amt float64) {
	if amt <= 0 {
		return
	}
	a.energy -= amt
	if a.energy < 0 {
		a.energy = 0
	}
	a.spentAcc += amt
	for a.spentAcc+1e-9 >= 1 {
		a.spentAcc -= 1
		a.charge()
	}
	if a.energy < 1e-9 {
		a.energy = 0
		a.enterBreak()
	}
}

func (a *人偶使) lockUntilAt(now float64, sk uint8) {
	until := now + lockSpan(sk)
	if until > a.lockUntil {
		a.lockUntil = until
	}
}

func (a *人偶使) ensureRNG(ctx unit.Context) {
	if a.rng == nil {
		a.rng = rand.New(rand.NewPCG(rand.Uint64()^ctx.ID, rand.Uint64()))
	}
}

func (a *人偶使) countDolls(id uint64) {
	a.idleDolls = 0
	a.liveDolls = 0
	for i := range a.nearby {
		o := &a.nearby[i]
		if o.Kind != KindNingyushiDoll || o.OwnerID != id {
			continue
		}
		a.liveDolls++
		if stateOf(o.ID) == dollIdle {
			a.idleDolls++
		}
	}
}

func (a *人偶使) emitHUD(ctx unit.Context, s unit.Sense) {
	ctx.Out <- unit.FX{
		Name: "energy", Kind: ctx.Kind, UnitID: ctx.ID,
		X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot,
		Amount: a.energy, VY: a.energyCap,
	}
	packed, prog, n := a.hudCards()
	ctx.Out <- unit.FX{
		Name: "cards", Kind: ctx.Kind, UnitID: ctx.ID,
		X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot,
		Amount: prog, VX: packed, VY: float64(n),
	}
}

func (a *人偶使) invokePaid(ctx unit.Context, s unit.Sense, enemy *unit.Snapshot, sk uint8, fromCard bool) bool {
	a.ensureRNG(ctx)
	var target unit.Snapshot
	if enemy != nil {
		target = *enemy
	}
	a.playingCard = fromCard
	ok := a.invoke(ctx, s, target, sk)
	if ok && shouldCall(sk) {
		a.emitCall(ctx, s, sk)
	}
	a.playingCard = false
	if !ok {
		return false
	}
	cost := energyCostOf(sk)
	if fromCard {
		cost = energyCostOf(sk)
	}
	if !fromCard && isNumberCard(sk) && a.levelOf(sk) >= 1 {
		cost = 1
	}
	if cost > 0 {
		a.spend(cost)
	}
	a.regenAcc = 0
	a.lockUntilAt(s.Time, sk)
	return true
}

func (a *人偶使) emitCall(ctx unit.Context, s unit.Sense, sk uint8) {
	spell := 0.0
	if a.playingCard || (isNumberCard(sk) && a.levelOf(sk) >= 1) {
		spell = 1
	}
	ctx.Out <- unit.FX{
		Name: "call", Kind: ctx.Kind, UnitID: ctx.ID,
		X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot,
		Amount: float64(sk), VY: spell,
	}
}

func enemyOf(s unit.Sense) *unit.Snapshot {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role == unit.RoleFighter && o.Slot != s.Self.Slot {
			return o
		}
	}
	return nil
}

func noteDealt(owner uint64) {
	v, ok := owners.Load(owner)
	if !ok {
		return
	}
	a, _ := v.(*人偶使)
	if a == nil {
		return
	}
	a.charge()
}

func ownerOf(id uint64) *人偶使 {
	v, ok := owners.Load(id)
	if !ok {
		return nil
	}
	a, _ := v.(*人偶使)
	return a
}

func deal(ctx unit.Context, owner, to uint64, amt float64) {
	if amt <= 0 || to == 0 {
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: to, Amount: amt}
	noteDealt(owner)
}

func pushEnemy(ctx unit.Context, fromX, fromY float64, e unit.Snapshot, dist float64) {
	dx, dy := e.X-fromX, e.Y-fromY
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		dx, dy, n = 1, 0, 1
	}
	ux, uy := dx/n, dy/n
	nx, ny := clampHex(e.X+ux*dist, e.Y+uy*dist, e.Radius)
	ctx.Out <- unit.Teleport{UnitID: e.ID, X: nx, Y: ny}
	ctx.Out <- unit.SetVelocity{UnitID: e.ID, VX: ux * 220, VY: uy * 220}
}

func (a *人偶使) stun(ctx unit.Context, s unit.Sense, e *unit.Snapshot) {
	if e == nil || e.ID == 0 {
		return
	}
	a.stunID = e.ID
	a.stunUntil = s.Time + rectStun
	a.stunVX, a.stunVY = e.VX, e.VY
	a.stunCruise = math.Hypot(e.VX, e.VY)
	if spec, ok := unit.Lookup(e.Kind); ok && spec.Speed > 0 {
		a.stunCruise = spec.Speed
	}
	a.stunning = true
	ctx.Out <- unit.Stun{UnitID: e.ID, Hold: true}
	ctx.Out <- unit.SetVelocity{UnitID: e.ID, VX: 0, VY: 0}
	ctx.Out <- unit.SetCruise{UnitID: e.ID, Speed: 0}
}

func (a *人偶使) holdStun(ctx unit.Context, s unit.Sense) {
	if !a.stunning || a.stunID == 0 {
		return
	}
	alive := false
	for i := range s.Nearby {
		if s.Nearby[i].ID == a.stunID {
			alive = true
			break
		}
	}
	if !alive || s.Time+1e-9 >= a.stunUntil {
		if alive {
			ctx.Out <- unit.Stun{UnitID: a.stunID, Hold: false}
			ctx.Out <- unit.SetCruise{UnitID: a.stunID, Speed: a.stunCruise}
			vx, vy := a.stunVX, a.stunVY
			if math.Hypot(vx, vy) < 1e-6 {
				vx, vy = a.stunCruise, 0
			}
			ctx.Out <- unit.SetVelocity{UnitID: a.stunID, VX: vx, VY: vy}
		}
		a.stunning = false
		a.stunID = 0
		return
	}
	ctx.Out <- unit.Stun{UnitID: a.stunID, Hold: true}
	ctx.Out <- unit.SetVelocity{UnitID: a.stunID, VX: 0, VY: 0}
	ctx.Out <- unit.SetCruise{UnitID: a.stunID, Speed: 0}
}

func faceOf(a *人偶使, s unit.Sense, enemy unit.Snapshot) (float64, float64) {
	if enemy.ID != 0 {
		return toward(s.Self, enemy)
	}
	if n := math.Hypot(a.hx, a.hy); n > 1e-6 {
		return a.hx, a.hy
	}
	return 1, 0
}

func randDir(a *人偶使) (float64, float64) {
	ang := a.rng.Float64() * 2 * math.Pi
	return math.Cos(ang), math.Sin(ang)
}
