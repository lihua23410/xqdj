// 雷达(百战)不转向。没有点名时扫掠线被墙裁断，本身不出伤。
// 扫到的第一个人被点名：线消失，立炮 1 秒后满 1、2、3、4 秒各挂一轮，第四轮挂出或人没了再收炮 1 秒，线回来。
package 雷达_百战

import (
	"embed"
	"math"
	"math/rand/v2"
	"sync"

	"xqdj/internal/unit"
)

const (
	KindRadarVeteran = "雷达(百战)"
	KindBarrage      = "大炮击"
	KindSmallBarrage = "小炮击"
)

const (
	vetRadius = 18.0
	vetHP     = 100.0
	vetCruise = 160.0
	vetVision = 9999.0
	vetColor  = "#1a6b6b"

	sweepWidth = 4.0
	sweepSpin  = 6.0 // 顺时针转一圈的秒数
	nameGap    = 1.0
	nameShots  = 4
	nameAim    = 1
	morphDur   = 1.0

	bigDelay    = 0.64
	bigDamage   = 12.0
	bigRadius   = 80.0
	bigKnockSp  = 280.0
	bigKnockDur = 0.25
	bigRise     = 0.36
	blastHold   = 0.75 // 爆炸后炮弹还留着，落弹影和爆炸特效能熬过被丢掉的帧

	smallCount    = 3
	smallScatter  = 35.0
	smallGapMax   = 0.15
	smallDelay    = 0.4
	smallDamage   = 5.0
	smallRadius   = 55.0
	smallKnockSp  = 200.0
	smallKnockDur = 0.15
	smallRise     = 0.20

	modeRandom = 0
	modeBig    = 1
	modeSmall  = 2
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindRadarVeteran, assets)
	p.Register(unit.Spec{
		Kind:    KindRadarVeteran,
		Role:    unit.RoleFighter,
		Radius:  vetRadius,
		MaxHP:   vetHP,
		Speed:   vetCruise,
		Vision:  vetVision,
		Fighter: true,
		Look: unit.Look{
			Color: vetColor,
			FX:    []string{"radar-vet"},
		},
	}, func(unit.SpawnInfo) unit.Actor {
		return &百战{}
	})
	p.Register(unit.Spec{
		Kind:     KindBarrage,
		Role:     unit.RoleHelper,
		Radius:   1,
		MaxHP:    1,
		Speed:    0,
		Vision:   bigRadius + 80,
		Nonsolid: true,
		Look: unit.Look{
			Color:   "#d4a04a",
			Overlay: true,
			FX:      []string{"radar-shell"},
		},
	}, func(info unit.SpawnInfo) unit.Actor {
		return newShell(info, bigDelay, bigDamage, bigRadius, bigKnockSp, bigKnockDur, 0)
	})
	p.Register(unit.Spec{
		Kind:     KindSmallBarrage,
		Role:     unit.RoleHelper,
		Radius:   1,
		MaxHP:    1,
		Speed:    0,
		Vision:   smallRadius + 80,
		Nonsolid: true,
		Look: unit.Look{
			Color:   "#c89850",
			Overlay: true,
			FX:      []string{"radar-shell-sm"},
		},
	}, func(info unit.SpawnInfo) unit.Actor {
		return newShell(info, smallDelay, smallDamage, smallRadius, smallKnockSp, smallKnockDur, popArmWait())
	})
}

// SpawnInfo 不带自定义字段；小炮击起前摇延迟经队列在 Spawn 前推入、工厂弹出。
var (
	armMu    sync.Mutex
	armWaits []float64
)

func pushArmWait(v float64) {
	armMu.Lock()
	armWaits = append(armWaits, v)
	armMu.Unlock()
}

func popArmWait() float64 {
	armMu.Lock()
	defer armMu.Unlock()
	if len(armWaits) == 0 {
		return 0
	}
	v := armWaits[0]
	armWaits = armWaits[1:]
	return v
}

const (
	phaseSweep = iota
	phaseRaise
	phaseBarrage
	phaseLower
)

type 百战 struct {
	rng      *rand.Rand
	mode     int // 0 随机；测试可钉死大/小
	phase    int
	morphAt  float64
	nameID   uint64
	nameAim  uint8
	shots    int
	nextShot float64
	cues     []launchCue
}

// launchCue 升起这一段里每帧都再发一次发射，避免单帧特效被挤掉。
type launchCue struct {
	until float64
	big   bool
	delay float64
	side  float64
}

func (a *百战) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if a.rng == nil {
		a.rng = unit.NewRNG(ctx.Stream)
	}
	started := a.phase
	if started == phaseRaise {
		if _, alive := namedOf(s, a.nameID); !alive {
			a.startLower(ctx, s, unit.Snapshot{}, false)
		} else if s.Time+1e-9 >= a.morphAt+morphDur {
			a.phase = phaseBarrage
			a.shots = 0
			a.nextShot = a.morphAt + morphDur + nameGap
		}
	}
	if a.phase == phaseBarrage {
		a.tickBarrage(ctx, s)
	}
	if started == phaseLower && s.Time+1e-9 >= a.morphAt+morphDur {
		a.phase = phaseSweep
	}
	dx, dy := sweepDir(s.Time)
	reach := clipReach(s.Self.X, s.Self.Y, dx, dy, visionOf(s), sweepWidth/2, s.Field, s.Walls)
	if a.phase == phaseSweep {
		if o, ok := a.nearest(s, dx, dy, reach); ok {
			a.beginName(ctx, s, o)
		}
	}
	a.emitPose(ctx, s, dx, dy, reach)
	a.flushLaunches(ctx, s)
}

func (a *百战) tickBarrage(ctx unit.Context, s unit.Sense) {
	o, alive := namedOf(s, a.nameID)
	if !alive {
		a.startLower(ctx, s, unit.Snapshot{}, false)
		return
	}
	if s.Time+1e-9 < a.nextShot {
		return
	}
	a.hangRound(ctx, s, o.X, o.Y)
	a.shots++
	a.nextShot += nameGap
	if a.shots >= nameShots {
		a.startLower(ctx, s, o, true)
	}
}

func (a *百战) beginName(ctx unit.Context, s unit.Sense, o unit.Snapshot) {
	a.phase = phaseRaise
	a.morphAt = s.Time
	a.nameID = o.ID
	a.nameAim = o.AimPriority
	a.shots = 0
	if o.AimPriority != 0 {
		ctx.Out <- unit.SetAimPriority{From: ctx.ID, UnitID: o.ID, Value: nameAim}
	}
}

func (a *百战) startLower(ctx unit.Context, s unit.Sense, o unit.Snapshot, restore bool) {
	a.phase = phaseLower
	a.morphAt = s.Time
	if restore && a.nameAim != 0 {
		ctx.Out <- unit.SetAimPriority{From: ctx.ID, UnitID: o.ID, Value: a.nameAim}
	}
}

func (a *百战) emitPose(ctx unit.Context, s unit.Sense, dx, dy, reach float64) {
	grow, gun := a.gunGrow(s.Time)
	a.emitGun(ctx, s, grow, gun)
	if a.phase == phaseRaise || a.phase == phaseBarrage {
		if o, ok := namedOf(s, a.nameID); ok {
			a.emitName(ctx, s, o)
		} else {
			a.emitName(ctx, s, unit.Snapshot{})
		}
		a.emitBeam(ctx, s, 0, 0, 0)
		return
	}
	a.emitName(ctx, s, unit.Snapshot{})
	if a.phase == phaseLower {
		a.emitBeam(ctx, s, 0, 0, 0)
		return
	}
	a.emitBeam(ctx, s, dx, dy, reach)
}

func (a *百战) gunGrow(t float64) (float64, bool) {
	switch a.phase {
	case phaseRaise:
		return morphProgress(t, a.morphAt), true
	case phaseBarrage:
		return 1, true
	case phaseLower:
		return 1 - morphProgress(t, a.morphAt), true
	default:
		return 0, false
	}
}

func morphProgress(t, start float64) float64 {
	p := (t - start) / morphDur
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}

func (a *百战) nearest(s unit.Sense, dx, dy, reach float64) (unit.Snapshot, bool) {
	half := sweepWidth / 2
	var best unit.Snapshot
	var bestAlong float64
	found := false
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		along, ok := sweepAlong(s.Self.X, s.Self.Y, dx, dy, reach, o.X, o.Y, o.Radius, half)
		if !ok {
			continue
		}
		if !found || along < bestAlong {
			best = *o
			bestAlong = along
			found = true
		}
	}
	return best, found
}

func namedOf(s unit.Sense, id uint64) (unit.Snapshot, bool) {
	for i := range s.Nearby {
		if s.Nearby[i].ID == id {
			return s.Nearby[i], true
		}
	}
	return unit.Snapshot{}, false
}

func (a *百战) pickBig() bool {
	switch a.mode {
	case modeBig:
		return true
	case modeSmall:
		return false
	default:
		return a.rng.Float64() < 0.5
	}
}

func (a *百战) hangRound(ctx unit.Context, s unit.Sense, x, y float64) {
	if a.pickBig() {
		a.noteLaunch(s, true, 0, 0)
		ctx.Out <- ctx.Born(unit.Spawn{
			Kind: KindBarrage, X: x, Y: y,
			OwnerID: ctx.ID, Slot: s.Self.Slot,
		})
		return
	}
	a.hangSmall(ctx, s, x, y)
}

func (a *百战) hangSmall(ctx unit.Context, s unit.Sense, mx, my float64) {
	arm := 0.0
	for i := 0; i < smallCount; i++ {
		if i > 0 {
			arm += a.rng.Float64() * smallGapMax
		}
		x, y := scatterNear(mx, my, smallScatter, a.rng)
		pushArmWait(arm)
		a.noteLaunch(s, false, arm, float64(i-1)*10)
		ctx.Out <- ctx.Born(unit.Spawn{
			Kind: KindSmallBarrage, X: x, Y: y,
			OwnerID: ctx.ID, Slot: s.Self.Slot,
		})
	}
}

func (a *百战) noteLaunch(s unit.Sense, big bool, delay, side float64) {
	dur := smallRise
	if big {
		dur = bigRise
	}
	a.cues = append(a.cues, launchCue{
		until: s.Time + dur, big: big, delay: delay, side: side,
	})
}

func (a *百战) flushLaunches(ctx unit.Context, s unit.Sense) {
	if len(a.cues) == 0 {
		return
	}
	keep := make([]launchCue, 0, len(a.cues))
	for _, c := range a.cues {
		if s.Time > c.until+1e-9 {
			continue
		}
		a.emitLaunch(ctx, s, c.big, c.delay, c.side)
		if s.Time+1e-9 < c.until {
			keep = append(keep, c)
		}
	}
	a.cues = keep
}

func (a *百战) emitLaunch(ctx unit.Context, s unit.Sense, big bool, delay, side float64) {
	amt := 0.0
	if big {
		amt = 1
	}
	ctx.Out <- unit.FX{
		Name:   "launch",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		Slot:   s.Self.Slot,
		X:      s.Self.X,
		Y:      s.Self.Y,
		VX:     side,
		VY:     delay,
		Amount: amt,
	}
}

func (a *百战) emitGun(ctx unit.Context, s unit.Sense, grow float64, on bool) {
	amt := -1.0
	if on {
		amt = grow
	}
	ctx.Out <- unit.FX{
		Name:   "gun",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		Slot:   s.Self.Slot,
		X:      s.Self.X,
		Y:      s.Self.Y,
		Amount: amt,
	}
}

func (a *百战) emitBeam(ctx unit.Context, s unit.Sense, dx, dy, reach float64) {
	ctx.Out <- unit.FX{
		Name:   "beam",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		Slot:   s.Self.Slot,
		X:      s.Self.X,
		Y:      s.Self.Y,
		VX:     dx,
		VY:     dy,
		Amount: reach,
	}
}

func (a *百战) emitName(ctx unit.Context, s unit.Sense, o unit.Snapshot) {
	fx := unit.FX{
		Name:   "name",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		Slot:   s.Self.Slot,
	}
	if o.ID != 0 {
		fx.X, fx.Y = o.X, o.Y
		fx.Amount = o.Radius
	}
	ctx.Out <- fx
}

type 炮击 struct {
	owner    uint64
	slot     int
	delay    float64
	damage   float64
	radius   float64
	knockSp  float64
	knockDur float64
	armWait  float64
	booted   bool
	pass     bool
	armAt    float64
	showAt   float64
	boomAt   float64
	boomed   bool
	dead     bool
}

func newShell(info unit.SpawnInfo, delay, damage, radius, knockSp, knockDur, armWait float64) *炮击 {
	if armWait < 0 {
		armWait = 0
	}
	return &炮击{
		owner: info.OwnerID, slot: info.Slot,
		delay: delay, damage: damage, radius: radius,
		knockSp: knockSp, knockDur: knockDur, armWait: armWait,
	}
}

func (b *炮击) rise() float64 {
	lead := smallRise
	if b.delay+1e-9 >= bigDelay {
		lead = bigRise
	}
	if lead >= b.delay {
		return 0
	}
	return lead
}

func (b *炮击) Handle(ctx unit.Context, ev unit.Event) {
	if b.dead {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !b.pass {
		b.pass = true
		ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
	}
	if !b.booted {
		b.booted = true
		b.armAt = s.Time + b.armWait
		b.showAt = b.armAt + b.rise()
		b.boomAt = b.armAt + b.delay
	}
	if s.Time+1e-9 < b.showAt {
		return
	}
	if s.Time+1e-9 < b.boomAt {
		shadow := b.boomAt - b.showAt
		if shadow < 0.16 {
			shadow = 0.16
		}
		ctx.Out <- unit.FX{
			Name:   "mark",
			Kind:   ctx.Kind,
			UnitID: ctx.ID,
			Slot:   b.slot,
			X:      s.Self.X,
			Y:      s.Self.Y,
			Amount: b.radius,
			VY:     shadow,
		}
		return
	}
	if !b.boomed {
		b.boomed = true
		b.explode(ctx, s)
	}
	ctx.Out <- unit.FX{
		Name:   "blast",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		Slot:   b.slot,
		X:      s.Self.X,
		Y:      s.Self.Y,
		Amount: b.radius,
	}
	if s.Time+1e-9 >= b.boomAt+blastHold {
		b.dead = true
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

func (b *炮击) explode(ctx unit.Context, s unit.Sense) {
	x, y := s.Self.X, s.Self.Y
	from := b.owner
	if from == 0 {
		from = ctx.ID
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, b.slot) {
			continue
		}
		if math.Hypot(o.X-x, o.Y-y) > b.radius+o.Radius {
			continue
		}
		ctx.Out <- unit.Damage{From: from, To: o.ID, Amount: b.damage}
		if b.knockSp > 0 {
			ux, uy := knockDir(x, y, o.X, o.Y, o.ID, s.Time)
			ctx.Out <- unit.AddFS{
				UnitID: o.ID, DX: ux, DY: uy, BaseSpeed: b.knockSp, OnWall: true,
				ExpiresAt: s.Time + b.knockDur, Token: ctx.ID<<32 | o.ID,
			}
		}
	}
}

func scatterNear(cx, cy, r float64, rng *rand.Rand) (float64, float64) {
	ang := rng.Float64() * 2 * math.Pi
	rad := math.Sqrt(rng.Float64()) * r
	return cx + math.Cos(ang)*rad, cy + math.Sin(ang)*rad
}

// knockDir：从钉点指向目标；距离≈0 时用 id/time 定一个平面随机方向。
func knockDir(px, py, hx, hy float64, id uint64, t float64) (float64, float64) {
	dx, dy := hx-px, hy-py
	n := math.Hypot(dx, dy)
	if n > 1e-6 {
		return dx / n, dy / n
	}
	ang := math.Mod(float64(id)*2.399963229728653+t*17.13, 2*math.Pi)
	return math.Cos(ang), math.Sin(ang)
}

func sweepDir(t float64) (float64, float64) {
	ang := math.Pi/2 - (2*math.Pi/sweepSpin)*t
	return math.Cos(ang), math.Sin(ang)
}

func visionOf(s unit.Sense) float64 {
	if s.Self.Vision > 0 {
		return s.Self.Vision
	}
	return vetVision
}

// clipReach：沿射线找第一处被场边 / 硬墙 / 胶囊墙挡住的距离。
// Field.Walkable 含场边与场地预放墙；演员砌的硬墙（如八边形三段）只在 Sense.Walls 里，须另查。
// 按世界步长采样，避免视野 9999 时固定份数跨过薄墙。
func clipReach(px, py, ux, uy, maxLen, halfW float64, field unit.Field, walls []unit.WallView) float64 {
	if maxLen <= 0 {
		return 0
	}
	step := 4.0
	if halfW > step {
		step = halfW
	}
	prev := 0.0
	for t := step; t <= maxLen+1e-9; t += step {
		if t > maxLen {
			t = maxLen
		}
		if !blockedAt(px+ux*t, py+uy*t, halfW, field, walls) {
			prev = t
			if t >= maxLen {
				return maxLen
			}
			continue
		}
		lo, hi := prev, t
		for j := 0; j < 18; j++ {
			mid := 0.5 * (lo + hi)
			if !blockedAt(px+ux*mid, py+uy*mid, halfW, field, walls) {
				lo = mid
			} else {
				hi = mid
			}
		}
		return hi
	}
	return maxLen
}

func blockedAt(x, y, halfW float64, field unit.Field, walls []unit.WallView) bool {
	if !field.Walkable(x, y, halfW) {
		return true
	}
	for i := range walls {
		w := &walls[i]
		if segDist(x, y, w.X1, w.Y1, w.X2, w.Y2) < w.Radius+halfW {
			return true
		}
	}
	return false
}

func segDist(px, py, x1, y1, x2, y2 float64) float64 {
	dx, dy := x2-x1, y2-y1
	l2 := dx*dx + dy*dy
	if l2 < 1e-12 {
		return math.Hypot(px-x1, py-y1)
	}
	t := ((px-x1)*dx + (py-y1)*dy) / l2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return math.Hypot(px-(x1+dx*t), py-(y1+dy*t))
}

// sweepAlong 是目标圆心在扫掠线上的投影。线够到这个圆才算碰到。
func sweepAlong(px, py, ux, uy, reach, hx, hy, hr, halfW float64) (along float64, ok bool) {
	dx, dy := hx-px, hy-py
	along = dx*ux + dy*uy
	if along < 0 || along > reach {
		return 0, false
	}
	cx, cy := px+ux*along, py+uy*along
	if math.Hypot(hx-cx, hy-cy) > hr+halfW {
		return 0, false
	}
	return along, true
}
