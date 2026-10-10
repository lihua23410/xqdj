// 雷达(百战)不转向。扫掠线被墙裁断，本身不出伤；扫到敌方战斗机或活随从则挂大炮击或小炮击。
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
	sweepSpin  = 3.0 // 顺时针转一圈的秒数
	sweepHitCD = 1.2 // 同一目标两次挂炮击的间隔

	bigDelay    = 0.64
	bigDamage   = 12.0
	bigRadius   = 80.0
	bigKnockSp  = 280.0
	bigKnockDur = 0.25

	smallCount    = 3
	smallScatter  = 35.0
	smallGapMax   = 0.15
	smallDelay    = 0.4
	smallDamage   = 5.0
	smallRadius   = 55.0
	smallKnockSp  = 200.0
	smallKnockDur = 0.15

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
	}, func(info unit.SpawnInfo) unit.Actor {
		return &百战{
			hitAt: map[uint64]float64{},
			rng:   rand.New(rand.NewPCG(uint64(info.Slot+1)*0x9E3779B97F4A7C15, 0xB7A11)),
		}
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

type 百战 struct {
	hitAt map[uint64]float64
	rng   *rand.Rand
	mode  int // 0 随机；测试可钉死大/小
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
		a.rng = rand.New(rand.NewPCG(ctx.ID, ctx.ID^0xB7A11))
	}
	dx, dy := sweepDir(s.Time)
	reach := clipReach(s.Self.X, s.Self.Y, dx, dy, visionOf(s), sweepWidth/2, s.Field, s.Walls)
	a.sweep(ctx, s, dx, dy, reach)
	a.emitBeam(ctx, s, dx, dy, reach)
}

func (a *百战) sweep(ctx unit.Context, s unit.Sense, dx, dy, reach float64) {
	if a.hitAt == nil {
		a.hitAt = map[uint64]float64{}
	}
	half := sweepWidth / 2
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		mx, my, ok := markPoint(s.Self.X, s.Self.Y, dx, dy, reach, o.X, o.Y, o.Radius, half)
		if !ok {
			continue
		}
		if at, seen := a.hitAt[o.ID]; seen && s.Time+1e-9 < at+sweepHitCD {
			continue
		}
		a.hitAt[o.ID] = s.Time
		if a.pickBig() {
			ctx.Out <- unit.Spawn{
				Kind: KindBarrage, X: mx, Y: my,
				OwnerID: ctx.ID, Slot: s.Self.Slot,
			}
			continue
		}
		a.hangSmall(ctx, s, mx, my)
	}
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

func (a *百战) hangSmall(ctx unit.Context, s unit.Sense, mx, my float64) {
	arm := 0.0
	for i := 0; i < smallCount; i++ {
		if i > 0 {
			arm += a.rng.Float64() * smallGapMax
		}
		x, y := scatterNear(mx, my, smallScatter, a.rng)
		pushArmWait(arm)
		ctx.Out <- unit.Spawn{
			Kind: KindSmallBarrage, X: x, Y: y,
			OwnerID: ctx.ID, Slot: s.Self.Slot,
		}
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
	boomAt   float64
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
		b.boomAt = b.armAt + b.delay
	}
	if s.Time+1e-9 < b.armAt {
		return
	}
	prog := (s.Time - b.armAt) / b.delay
	if prog < 0 {
		prog = 0
	}
	if prog > 1 {
		prog = 1
	}
	ctx.Out <- unit.FX{
		Name:   "mark",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		Slot:   b.slot,
		X:      s.Self.X,
		Y:      s.Self.Y,
		Amount: b.radius * prog,
	}
	if s.Time+1e-9 < b.boomAt {
		return
	}
	b.explode(ctx, s)
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
	ctx.Out <- unit.FX{
		Name:   "blast",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		Slot:   b.slot,
		X:      x,
		Y:      y,
		Amount: b.radius,
	}
	b.dead = true
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
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

// markPoint：射线扫到目标圆时的扫掠地点（与圆的最近点；相交取先碰到的点）。
func markPoint(px, py, ux, uy, reach, hx, hy, hr, halfW float64) (mx, my float64, ok bool) {
	dx, dy := hx-px, hy-py
	along := dx*ux + dy*uy
	if along < 0 || along > reach {
		return 0, 0, false
	}
	cx, cy := px+ux*along, py+uy*along
	perp := math.Hypot(hx-cx, hy-cy)
	if perp > hr+halfW {
		return 0, 0, false
	}
	dist2 := dx*dx + dy*dy
	disc := along*along - (dist2 - hr*hr)
	if disc >= 0 {
		root := math.Sqrt(disc)
		t0 := along - root
		if t0 < 0 {
			t0 = along + root
		}
		if t0 >= 0 && t0 <= reach {
			return px + ux*t0, py + uy*t0, true
		}
	}
	return cx, cy, true
}
