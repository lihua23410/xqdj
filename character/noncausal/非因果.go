// 非因果：探针里自己是纯靶，记下打在身上的 (t, amount)，不让这局因此分出胜负。
// 真身阶段把航向反过来，用残影沿探针轨迹重放敌人战斗机的残差。
// 唯一的攻击是锁影弹，只磨残影上的 GhostHP。打光则 Despawn 那只敌人战斗机
// （不走伤害管线：真身挡不住，也不扣真身血）。自己的真血空了，或真身阶段的钟走到 T†，则摘掉自己。
//
// 事件都走现成的 unit.FX，字段见 fxName 常量上面的对照。
package 非因果

import (
	"embed"
	"math"
	"math/rand/v2"
	"sort"
	"sync"

	"xqdj/internal/unit"
)

const (
	KindNoncausal = "非因果"
	KindResidual  = "残影"
	KindLock      = "锁影弹"
)

const (
	bodyRadius = 18.0
	bodyHP     = 100.0
	bodySpeed  = 160.0
	bodyVision = 9999.0
	bodyColor  = "#7ec8e3"

	residualRadius = 12.0
	residualColor  = "#dff6ff"

	// missileRadius 锁影弹的碰撞半径。越大越容易被实心单位和墙白嫖挡掉。
	missileRadius = 5.0
	lockColor     = "#f4efc2"
	// missileSpeed 直线速度。寻的的「慢」不在这里，在 seekTurn。
	missileSpeed = 210.0
	// seekTurn 每秒最多转过的弧度。
	// 小：拐不过墙和真身，容易被挡到 T†。大：贴着残影绕，挡的密度要更高才挡得住。
	seekTurn  = 1.7
	muzzleGap = bodyRadius + missileRadius + 4

	// probeCap 探针最久多久（秒）。伤害先把假想生命磨光则提前结束，T† 取那个时长。
	probeCap = 12.0
	// fallbackCadence 探针里少于两下命中时的射速（秒）。
	fallbackCadence = 0.8
	// fallbackDamage 没有命中样本时的唯一直方图柱。
	fallbackDamage = 6.0
	// fallbackGhost 探针没看见敌人最大生命时的 GhostHP 上限。
	fallbackGhost = 100.0

	// tickDT 与比赛 60Hz 对齐。间隔平均为 0 时射速不会快过一拍。
	tickDT = 1.0 / 60.0
)

// 前端联调用的 FX 名。一次真实阶段开头同步频谱，之后每发、挡、磨各一条。
//
//	spectrum      真实阶段开头一次。amount=射速秒，vx=柱数，vy=脉冲数，slot=槽。
//	spectrum-bin  每个柱一条。slot=binIndex，amount=该柱伤害，vx=权重（次数），vy=射速秒。
//	doom-at       同一次。amount=T† 秒，vx=GhostHP 上限，vy=射速秒。
//	probe-hit     探针每一记。amount=伤害，vx=绝对时间 t。
//	lock-shot     开火。slot=binIndex，amount=drawnAmount，vx/vy=初速，x/y=枪口。unitId=战斗机。
//	lock-armed    弹体第一拍。slot=binIndex，amount=drawnAmount。unitId=弹。
//	lock-block    被实心单位或墙挡掉。slot=binIndex，vx=挡住者 id（墙为 0），vy=0 墙 / 1 单位。
//	ghost-hit     打中残影。slot=binIndex，amount=drawnAmount，vx=剩余 GhostHP。unitId=残影。
//	ghost-down    GhostHP 归零。amount=最后一发伤害，vx=被摘的敌人 id，vy=GhostHP 上限。
//	doom          钟走到 T†，自己离场。amount=T†。
const (
	fxSpectrum  = "spectrum"
	fxBin       = "spectrum-bin"
	fxDoomAt    = "doom-at"
	fxProbe     = "probe-hit"
	fxLockShot  = "lock-shot"
	fxLockArm   = "lock-armed"
	fxBlock     = "lock-block"
	fxGhostHit  = "ghost-hit"
	fxGhostDown = "ghost-down"
	fxDoom      = "doom"
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindNoncausal, assets)
	p.Register(unit.Spec{
		Kind:    KindNoncausal,
		Role:    unit.RoleFighter,
		Radius:  bodyRadius,
		MaxHP:   bodyHP,
		Speed:   bodySpeed,
		Vision:  bodyVision,
		Fighter: true,
		Look:    unit.Look{Color: bodyColor, FX: []string{"noncausal"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &非因果{stream: info.Stream}
	})

	// 残影照勇者同伴：空逻辑，位置全由主人 Teleport。RoleHelper 恒非实心，不挡锁影弹。
	p.Register(unit.Spec{
		Kind:   KindResidual,
		Role:   unit.RoleHelper,
		Radius: residualRadius,
		MaxHP:  1,
		Speed:  0,
		Vision: 0,
		Look:   unit.Look{Color: residualColor, FX: []string{"residual"}},
	}, func(unit.SpawnInfo) unit.Actor { return 残影{} })

	p.Register(unit.Spec{
		Kind:    KindLock,
		Role:    unit.RoleProjectile,
		Radius:  missileRadius,
		MaxHP:   1,
		Speed:   missileSpeed,
		Vision:  bodyVision,
		Fighter: false,
		Look:    unit.Look{Color: lockColor, Trail: true, FX: []string{"lock"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &锁影弹{owner: info.OwnerID}
	})
}

// 残影的位置由非因果每帧 Teleport 决定，自己不需要任何逻辑。
type 残影 struct{}

func (残影) Handle(unit.Context, unit.Event) {}

type pulse struct {
	t      float64
	amount float64
}

type bin struct {
	amount float64
	weight int
}

type spectrum struct {
	bins    []bin
	cadence float64
	pulses  int
}

type sample struct {
	t, x, y float64
}

type shot struct {
	bin    int
	amount float64
}

type book struct {
	mu       sync.Mutex
	id       uint64
	real     bool
	selfMax  float64
	haveT0   bool
	t0       float64
	pulses   []pulse
	sum      float64
	ready    bool
	doomAt   float64
	path     []sample
	enemyID  uint64
	enemyMax float64
	realAt   float64
	ghostHP  float64
	ghostMax float64
	spec     spectrum
	nextFire float64
	live     bool
	queue    []shot
	won      bool
	ended    bool
}

var books sync.Map

func newBook(id uint64, maxHP float64) *book {
	b := &book{id: id, selfMax: maxHP}
	books.Store(id, b)
	return b
}

func loadBook(id uint64) *book {
	v, ok := books.Load(id)
	if !ok {
		return nil
	}
	return v.(*book)
}

type 非因果 struct {
	b          *book
	stream     uint64
	rng        *rand.Rand
	x, y       float64
	slot       int
	residualID uint64
}

func (f *非因果) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		f.onHit(ctx, e)
	case unit.Sense:
		f.onSense(ctx, e)
	}
}

func (f *非因果) ensure(ctx unit.Context, maxHP float64) {
	if f.b != nil {
		return
	}
	if ctx.Stream != 0 {
		f.stream = ctx.Stream
	}
	if f.rng == nil {
		f.rng = unit.NewRNG(f.stream)
	}
	hp := maxHP
	if hp <= 0 {
		hp = bodyHP
	}
	f.b = newBook(ctx.ID, hp)
}

func (f *非因果) onHit(ctx unit.Context, e unit.IncomingDamage) {
	f.ensure(ctx, 0)
	if !f.b.notePulse(e.Time, e.Amount) {
		ctx.Out <- unit.ConfirmDamage{Token: e.Token, UnitID: ctx.ID, Amount: e.Amount}
		return
	}
	ctx.Out <- unit.BlockDamage{Token: e.Token, UnitID: ctx.ID}
	ctx.Out <- unit.FX{
		Name: fxProbe, UnitID: ctx.ID, Kind: KindNoncausal,
		X: f.x, Y: f.y, Amount: e.Amount, VX: e.Time, Slot: f.slot,
	}
}

func (b *book) notePulse(t, amount float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.real {
		return false
	}
	if !b.haveT0 {
		b.t0 = t
		b.haveT0 = true
	}
	if amount > 0 {
		b.pulses = append(b.pulses, pulse{t: t, amount: amount})
		b.sum += amount
	}
	if b.selfMax > 0 && b.sum+1e-9 >= b.selfMax {
		b.ready = true
		b.doomAt = t - b.t0
		if b.doomAt < 0 {
			b.doomAt = 0
		}
	}
	return true
}

func (f *非因果) onSense(ctx unit.Context, s unit.Sense) {
	f.ensure(ctx, s.Self.MaxHP)
	f.x, f.y = s.Self.X, s.Self.Y
	f.slot = s.Self.Slot
	if opened, ok := f.stepProbe(s); ok {
		f.emitOpen(ctx, s, opened)
	}
	if !f.b.isReal() {
		return
	}
	f.trackResidual(s)
	f.replay(ctx, s)
	if f.b.wonAlready() {
		return
	}
	if f.b.clockUp(s.Time) {
		f.doom(ctx, s)
		return
	}
	f.tryFire(ctx, s)
}

type opened struct {
	spec           spectrum
	doom           float64
	ghost          float64
	spawnX, spawnY float64
	haveSpawn      bool
}

func (f *非因果) stepProbe(s unit.Sense) (opened, bool) {
	b := f.b
	b.mu.Lock()
	defer b.mu.Unlock()
	if s.Self.MaxHP > 0 && !b.real {
		b.selfMax = s.Self.MaxHP
		if b.sum+1e-9 >= b.selfMax && b.selfMax > 0 {
			b.ready = true
			if b.doomAt == 0 && b.haveT0 {
				b.doomAt = s.Time - b.t0
			}
		}
	}
	if !b.haveT0 {
		b.t0 = s.Time
		b.haveT0 = true
	}
	if !b.real {
		if en := enemyOf(s); en != nil {
			b.enemyID = en.ID
			if en.MaxHP > b.enemyMax {
				b.enemyMax = en.MaxHP
			}
			b.path = append(b.path, sample{t: s.Time, x: en.X, y: en.Y})
		} else if n := len(b.path); n > 0 {
			last := b.path[n-1]
			b.path = append(b.path, sample{t: s.Time, x: last.x, y: last.y})
		}
		elapsed := s.Time - b.t0
		if !(b.ready || elapsed+1e-9 >= probeCap) {
			return opened{}, false
		}
		if !b.ready {
			b.doomAt = probeCap
		}
		if b.doomAt < 0 {
			b.doomAt = 0
		}
		b.real = true
		b.realAt = s.Time
		b.spec = buildSpectrum(b.pulses)
		b.ghostMax = b.enemyMax
		if b.ghostMax <= 0 {
			b.ghostMax = fallbackGhost
		}
		b.ghostHP = b.ghostMax
		b.nextFire = s.Time
		op := opened{spec: b.spec, doom: b.doomAt, ghost: b.ghostMax}
		if len(b.path) > 0 {
			op.haveSpawn = true
			op.spawnX, op.spawnY = b.path[0].x, b.path[0].y
		}
		return op, true
	}
	return opened{}, false
}

func (f *非因果) emitOpen(ctx unit.Context, s unit.Sense, op opened) {
	vx, vy := -s.Self.VX, -s.Self.VY
	if math.Hypot(vx, vy) < 1e-6 {
		vx, vy = -s.Self.FaceX, -s.Self.FaceY
	}
	if math.Hypot(vx, vy) < 1e-6 {
		vx, vy = -1, 0
	}
	// 反航向：只改自己。残影从轨迹起点重放，真身留在探针结束时的位置，两下就分开了。
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: vx, VY: vy}
	ctx.Out <- unit.SetFSDirection{UnitID: ctx.ID, VX: vx, VY: vy}
	ctx.Out <- unit.FX{
		Name: fxSpectrum, UnitID: ctx.ID, Kind: KindNoncausal,
		X: s.Self.X, Y: s.Self.Y, Slot: f.slot,
		Amount: op.spec.cadence, VX: float64(len(op.spec.bins)), VY: float64(op.spec.pulses),
	}
	for i, bin := range op.spec.bins {
		ctx.Out <- unit.FX{
			Name: fxBin, UnitID: ctx.ID, Kind: KindNoncausal,
			X: s.Self.X, Y: s.Self.Y, Slot: i,
			Amount: bin.amount, VX: float64(bin.weight), VY: op.spec.cadence,
		}
	}
	ctx.Out <- unit.FX{
		Name: fxDoomAt, UnitID: ctx.ID, Kind: KindNoncausal,
		X: s.Self.X, Y: s.Self.Y, Slot: f.slot,
		Amount: op.doom, VX: op.ghost, VY: op.spec.cadence,
	}
	if op.haveSpawn {
		ctx.Out <- ctx.Born(unit.Spawn{
			Kind: KindResidual, X: op.spawnX, Y: op.spawnY,
			OwnerID: ctx.ID, Slot: f.slot,
		})
	}
}

func (f *非因果) trackResidual(s unit.Sense) {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind == KindResidual && o.OwnerID == f.b.id {
			f.residualID = o.ID
			return
		}
	}
}

func (f *非因果) replay(ctx unit.Context, s unit.Sense) {
	if f.residualID == 0 {
		return
	}
	x, y, ok := f.b.replayAt(s.Time)
	if !ok {
		return
	}
	ctx.Out <- unit.Teleport{UnitID: f.residualID, X: x, Y: y}
}

func (f *非因果) doom(ctx unit.Context, s unit.Sense) {
	ctx.Out <- unit.FX{
		Name: fxDoom, UnitID: ctx.ID, Kind: KindNoncausal,
		X: s.Self.X, Y: s.Self.Y, Slot: f.slot, Amount: f.b.doomDuration(),
	}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

func (f *非因果) tryFire(ctx unit.Context, s unit.Sense) {
	tx, ty, ok := f.b.replayAt(s.Time)
	if !ok {
		return
	}
	bin, amt, ok := f.b.armFire(s.Time, f.rng)
	if !ok {
		return
	}
	mx, my, vx, vy := muzzle(s.Self.X, s.Self.Y, tx, ty, s.Self.FaceX, s.Self.FaceY)
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: KindLock, X: mx, Y: my, VX: vx, VY: vy,
		OwnerID: ctx.ID, Slot: f.slot,
	})
	ctx.Out <- unit.FX{
		Name: fxLockShot, UnitID: ctx.ID, Kind: KindLock,
		X: mx, Y: my, VX: vx, VY: vy, Amount: amt, Slot: bin,
	}
}

func (b *book) isReal() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.real
}

func (b *book) wonAlready() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.won
}

func (b *book) doomDuration() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.doomAt
}

func (b *book) clockUp(now float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.ended || b.won || !b.real {
		return false
	}
	if now+1e-9 < b.realAt+b.doomAt {
		return false
	}
	b.ended = true
	b.live = false
	return true
}

func (b *book) replayAt(now float64) (float64, float64, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.real || len(b.path) == 0 {
		return 0, 0, false
	}
	i := atIndex(b.path, b.t0, now-b.realAt)
	p := b.path[i]
	return p.x, p.y, true
}

func (b *book) armFire(now float64, rng *rand.Rand) (bin int, amount float64, ok bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.real || b.live || b.won || b.ended || now+1e-9 < b.nextFire {
		return 0, 0, false
	}
	bin, amount = b.spec.draw(rng)
	b.live = true
	b.nextFire = now + b.spec.cadence
	b.queue = append(b.queue, shot{bin: bin, amount: amount})
	return bin, amount, true
}

func (b *book) takeShot() (shot, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.queue) == 0 {
		return shot{}, false
	}
	s := b.queue[0]
	b.queue = b.queue[1:]
	return s, true
}

type ghostEnd struct {
	hit     bool
	remain  float64
	down    bool
	enemy   uint64
	max     float64
	cadence float64
}

// endMissile 弹没了。hitGhost 为真才扣 GhostHP。下一发从这一拍再等一个射速，不连射。
func (b *book) endMissile(now float64, hitGhost bool, amount float64) ghostEnd {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.live = false
	var g ghostEnd
	g.remain = b.ghostHP
	g.enemy = b.enemyID
	g.max = b.ghostMax
	g.cadence = b.spec.cadence
	if hitGhost && amount > 0 && b.real && !b.won && !b.ended {
		b.ghostHP -= amount
		if b.ghostHP < 0 {
			b.ghostHP = 0
		}
		g.hit = true
		g.remain = b.ghostHP
		if b.ghostHP == 0 {
			b.won = true
			g.down = true
		}
	}
	if !b.won && !b.ended {
		cad := b.spec.cadence
		if cad <= 0 {
			cad = fallbackCadence
		}
		b.nextFire = now + cad
	}
	return g
}

func enemyOf(s unit.Sense) *unit.Snapshot {
	var best *unit.Snapshot
	bestD := math.MaxFloat64
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role != unit.RoleFighter || o.Slot == s.Self.Slot || o.ID == s.Self.ID {
			continue
		}
		d := math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y)
		if d < bestD {
			best = o
			bestD = d
		}
	}
	return best
}

func quant(a float64) float64 {
	return math.Round(a*1e6) / 1e6
}

// buildSpectrum 射速是命中间隔的平均（主频率的周期），整段真实阶段不再改。
// 伤害按 amount 的经验直方图抽样，每发各抽一次。
func buildSpectrum(pulses []pulse) spectrum {
	ps := append([]pulse(nil), pulses...)
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].t == ps[j].t {
			return ps[i].amount < ps[j].amount
		}
		return ps[i].t < ps[j].t
	})
	cadence := fallbackCadence
	if len(ps) >= 2 {
		sum := 0.0
		for i := 1; i < len(ps); i++ {
			dt := ps[i].t - ps[i-1].t
			if dt < 0 {
				dt = 0
			}
			sum += dt
		}
		cadence = sum / float64(len(ps)-1)
	}
	if cadence < tickDT {
		cadence = tickDT
	}
	counts := map[float64]int{}
	for _, p := range ps {
		counts[quant(p.amount)]++
	}
	keys := make([]float64, 0, len(counts))
	for a := range counts {
		keys = append(keys, a)
	}
	sort.Float64s(keys)
	bins := make([]bin, 0, len(keys))
	for _, a := range keys {
		bins = append(bins, bin{amount: a, weight: counts[a]})
	}
	if len(bins) == 0 {
		bins = []bin{{amount: fallbackDamage, weight: 1}}
	}
	return spectrum{bins: bins, cadence: cadence, pulses: len(ps)}
}

func (sp spectrum) draw(rng *rand.Rand) (int, float64) {
	total := 0
	for _, b := range sp.bins {
		total += b.weight
	}
	if total <= 0 || len(sp.bins) == 0 {
		return 0, fallbackDamage
	}
	if rng == nil {
		return 0, sp.bins[0].amount
	}
	r := rng.IntN(total)
	acc := 0
	for i, b := range sp.bins {
		acc += b.weight
		if r < acc {
			return i, b.amount
		}
	}
	last := len(sp.bins) - 1
	return last, sp.bins[last].amount
}

func atIndex(path []sample, t0, key float64) int {
	i := 0
	for i+1 < len(path) && path[i+1].t-t0 <= key+1e-9 {
		i++
	}
	return i
}

func muzzle(sx, sy, tx, ty, faceX, faceY float64) (x, y, vx, vy float64) {
	dx, dy := tx-sx, ty-sy
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		dx, dy = faceX, faceY
		n = math.Hypot(dx, dy)
	}
	if n < 1e-6 {
		dx, dy, n = 1, 0, 1
	}
	ux, uy := dx/n, dy/n
	return sx + ux*muzzleGap, sy + uy*muzzleGap, ux * missileSpeed, uy * missileSpeed
}
