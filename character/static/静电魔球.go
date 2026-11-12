// 静电魔球游走攒电。满电后沿预警过的固定线路放电，命中才续链。
package 静电魔球

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

const KindStaticBall = "静电魔球"

const (
	bodyRadius = 15.0
	bodyHP     = 95.0
	bodyCruise = 160.0
	bodyVision = 280.0
	bodyColor  = "#8fdcff"

	chargeMove   = 7.5 // 巡航摩擦生电，每秒；站定不生电
	chargeBump   = 8.0 // 撞上别的单位一次
	chargeScrape = 3.5 // 撞墙一次
	chargeHurt   = 7.0 // 挨一下
	chargeMax    = 100.0

	fullBoost    = 35.0  // 满电追击时的巡航加成
	approachDist = 170.0 // 远于这个就贴近
	retreatDist  = 110.0 // 近于这个就拉开，中间环绕走位
	movingSpeed  = 20.0  // 低于这个速度算站定，不摩擦

	markCharge = "静电充能" // 挂在自己身上的层数 = 电荷/10，对手看得见
	markSteps  = 10
	timeEps    = 1e-9
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindStaticBall, assets)
	p.Register(unit.Spec{
		Kind:    KindStaticBall,
		Role:    unit.RoleFighter,
		Radius:  bodyRadius,
		MaxHP:   bodyHP,
		Speed:   bodyCruise,
		Vision:  bodyVision,
		Fighter: true,
		Look: unit.Look{
			Color: bodyColor,
			Glow:  true,
			FX:    []string{"staticball"},
		},
	}, func(unit.SpawnInfo) unit.Actor {
		return &静电魔球{}
	})
}

type 静电魔球 struct {
	charge    float64
	applied   int  // 已经挂在身上的充能层数，免得每拍都发 StackMark
	boosted   bool // 巡航是否正加着满电追击
	chain     *discharge
	orbitSign float64 // 环绕方向，+1/-1，出生定一次
	lastTime  float64
	rng       *rand.Rand
}

func (a *静电魔球) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		unit.ConfirmHit(ctx, e)
		a.gain(chargeHurt)
	case unit.Collision:
		a.gain(chargeBump)
	case unit.WallHit:
		a.gain(chargeScrape)
	case unit.Stunned:
		a.stopChain(ctx, true)
	case unit.Sense:
		a.onSense(ctx, e)
	}
}

func (a *静电魔球) gain(n float64) {
	a.charge = math.Min(a.charge+n, chargeMax)
}

func (a *静电魔球) onSense(ctx unit.Context, s unit.Sense) {
	if a.rng == nil {
		a.rng = unit.NewRNG(ctx.Stream)
		a.orbitSign = 1
		if a.rng.Float64() < 0.5 {
			a.orbitSign = -1
		}
	}
	dt := s.Time - a.lastTime
	a.lastTime = s.Time
	if dt > 0 && math.Hypot(s.Self.VX, s.Self.VY) > movingSpeed {
		a.gain(chargeMove * dt)
	}
	a.syncMark(ctx)

	t := nearestHittable(s)
	if a.chain != nil {
		a.stepChain(ctx, s)
		a.steer(ctx, s, t, false)
		return
	}
	full := a.charge >= chargeMax-timeEps
	if full && t != nil && dist(s.Self, *t) <= jumpRange && a.startChain(ctx, s) {
		a.steer(ctx, s, t, false)
		return
	}
	a.setBoost(ctx, full)
	a.steer(ctx, s, t, full)
}

// syncMark 把电荷折成充能层数挂自己身上：对手看得见的预警，皮肤也靠它发光。
func (a *静电魔球) syncMark(ctx unit.Context) {
	want := int(a.charge/chargeMax*markSteps + timeEps)
	if want > markSteps {
		want = markSteps
	}
	if want == a.applied {
		return
	}
	ctx.Out <- unit.StackMark{UnitID: ctx.ID, Kind: markCharge, Delta: want - a.applied}
	a.applied = want
}

func (a *静电魔球) setBoost(ctx unit.Context, full bool) {
	if full == a.boosted {
		return
	}
	speed := bodyCruise
	if full {
		speed += fullBoost
	}
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: speed}
	a.boosted = full
}

// steer 走位：远就贴、近就退、中间环绕；满电时直奔目标凑放电。
func (a *静电魔球) steer(ctx unit.Context, s unit.Sense, t *unit.Snapshot, full bool) {
	if t == nil {
		return // 没人可追，保持原方向继续摩擦
	}
	dx, dy := t.X-s.Self.X, t.Y-s.Self.Y
	d := math.Hypot(dx, dy)
	if d < timeEps {
		return
	}
	ux, uy := dx/d, dy/d
	var wx, wy float64
	switch {
	case full || d > approachDist:
		wx, wy = ux, uy
	case d < retreatDist:
		wx, wy = -ux, -uy
	default:
		wx, wy = -uy*a.orbitSign, ux*a.orbitSign
	}
	ctx.Out <- unit.SetFSDirection{UnitID: ctx.ID, VX: wx, VY: wy}
}

// nearestHittable 走位用的索敌：最近的能挨电目标。
func nearestHittable(s unit.Sense) *unit.Snapshot {
	var best *unit.Snapshot
	bestD := math.Inf(1)
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		d := dist(s.Self, *o)
		if d < bestD {
			bestD = d
			best = o
		}
	}
	return best
}

func dist(a, b unit.Snapshot) float64 {
	return math.Hypot(a.X-b.X, a.Y-b.Y)
}
