// 静电魔球：摩擦起电的闪电球。平时游走攒电荷，满了就在敌人之间放链式闪电。
// 链条每跳挑当前节点射程内最近的敌方（战斗机或活随从，同槽友军永远不挨）；
// 单人局没有第二个目标可弹时，把自己当中继点（不出伤），四跳照样完整砸在唯一的对手身上。
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

	chargeMove   = 4.5 // 巡航摩擦生电，每秒；站定不生电
	chargeBump   = 8.0 // 撞上别的单位一次
	chargeScrape = 3.5 // 撞墙一次
	chargeHurt   = 7.0 // 挨一下
	chargeMax    = 100.0

	jumpRange = 250.0 // 一跳的最远距离
	chainHits = 4     // 最多命中敌人几跳
	hopCap    = 16    // 保险丝：含中继的最多跳数
	hopEvery  = 0.25  // 跳与跳的间隔：链条慢慢爬，观众看得清闪电怎么走

	fullBoost    = 35.0  // 满电追击时的巡航加成
	approachDist = 170.0 // 远于这个就贴近
	retreatDist  = 110.0 // 近于这个就拉开，中间环绕走位
	movingSpeed  = 20.0  // 低于这个速度算站定，不摩擦

	markCharge = "静电充能" // 挂在自己身上的层数 = 电荷/10，对手看得见
	markSteps  = 10
	timeEps    = 1e-9
)

// chainDamage 逐跳衰减：第几下打多疼。中继跳（弹过自己）不出伤。
var chainDamage = [chainHits]float64{11, 9, 7, 5}

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
	chain     *链条  // 非空 = 正在放电，一跳一跳推进
	nextHopAt float64
	orbitSign float64 // 环绕方向，+1/-1，出生定一次
	lastTime  float64
	rng       *rand.Rand
}

// 链条是进行中的放电：上一跳的落点（ID 和坐标）是下一跳的起点。
type 链条 struct {
	fromID       uint64
	fromX, fromY float64
	hits, hops   int
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
	if full && t != nil && dist(s.Self, *t) <= jumpRange {
		a.startChain(ctx, s)
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

// startChain/stepChain 见上。整个角色只做一件事：电人。

// startChain 满电开链：电荷当场清空（对手看得见计量归零），从自己出发第一跳立刻打。
func (a *静电魔球) startChain(ctx unit.Context, s unit.Sense) {
	a.charge = 0
	a.syncMark(ctx)
	a.setBoost(ctx, false)
	a.chain = &链条{fromID: s.Self.ID, fromX: s.Self.X, fromY: s.Self.Y}
	a.nextHopAt = s.Time
	a.stepChain(ctx, s)
}

// stepChain 推进一跳：上一跳的落点是这一跳的起点，目标按当前感知重新选，
// 敌人跑出射程链条就断在半路（电荷已经清了，得重新充）。
func (a *静电魔球) stepChain(ctx unit.Context, s unit.Sense) {
	c := a.chain
	if s.Time+timeEps < a.nextHopAt {
		return
	}
	fx, fy := c.fromX, c.fromY
	if c.fromID == s.Self.ID {
		fx, fy = s.Self.X, s.Self.Y
	} else {
		for i := range s.Nearby {
			if s.Nearby[i].ID == c.fromID {
				fx, fy = s.Nearby[i].X, s.Nearby[i].Y
				break
			}
		}
	}
	from := unit.Snapshot{ID: c.fromID, X: fx, Y: fy}
	node, ok := nextHop(from, s.Self, hittables(s), jumpRange)
	if !ok {
		a.chain = nil
		return
	}
	amount := 0.0
	if node.ID != s.Self.ID {
		amount = chainDamage[c.hits]
		ctx.Out <- unit.Damage{
			From: ctx.ID, To: node.ID, Amount: amount,
			NoFreeze: c.hits > 0, // 一条链只有第一下停帧
		}
		c.hits++
	}
	// UnitID 带起点单位、Slot 带落点单位：电弧要描两三百毫秒，期间人都在动，
	// 前端整段追着这两个球的实时位置画，雷落在身上而不是落在人刚才站的地方。
	ctx.Out <- unit.FX{
		Name: "static-bolt", Kind: ctx.Kind, UnitID: from.ID,
		X: fx, Y: fy,
		VX: node.X, VY: node.Y,
		Slot: int(node.ID), Amount: amount,
	}
	c.fromID, c.fromX, c.fromY = node.ID, node.X, node.Y
	c.hops++
	a.nextHopAt = s.Time + hopEvery
	if c.hits >= chainHits || c.hops >= hopCap {
		a.chain = nil
	}
}

// nextHop 选下一跳：射程内优先最近的敌方战斗机，没有才轮到最近的活随从，
// 最后回退自己中继（from 本身就是自己时没有退路，返回 false）。
// 候选已在 hittables 里滤过敌我，这里只管几何和不去撞当前节点自己。
func nextHop(from, self unit.Snapshot, cands []unit.Snapshot, jumpRange float64) (unit.Snapshot, bool) {
	minion, fighter := -1, -1
	minionD, fighterD := math.Inf(1), math.Inf(1)
	for i := range cands {
		o := &cands[i]
		if o.ID == from.ID {
			continue
		}
		d := dist(from, *o)
		if d > jumpRange+timeEps {
			continue
		}
		if o.Role == unit.RoleFighter {
			if d < fighterD {
				fighterD, fighter = d, i
			}
			continue
		}
		if d < minionD {
			minionD, minion = d, i
		}
	}
	if fighter >= 0 {
		return cands[fighter], true
	}
	if minion >= 0 {
		return cands[minion], true
	}
	if self.ID != from.ID && dist(from, self) <= jumpRange+timeEps {
		return self, true
	}
	return unit.Snapshot{}, false
}

// hittables 视野内所有能挨电的敌方：战斗机或活随从，不含同槽友军。
func hittables(s unit.Sense) []unit.Snapshot {
	var out []unit.Snapshot
	for i := range s.Nearby {
		if unit.Hittable(s.Nearby[i], s.Self.Slot) {
			out = append(out, s.Nearby[i])
		}
	}
	return out
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
