// 迷走：开局铺心环+轨环；轨环锚点沿引导圆转、圆心栓绳抖；永在环线上切向移动；交点非回头随机改道；当前环线点亮刮伤。
package 迷走

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

const KindMeander = "迷走"
const KindRing = "迷走环线"
const KindLit = "迷走点亮"

const (
	bodyRadius = 18.0
	bodyHP     = 100.0
	cruise     = 170.0
	bodyColor  = "#5e6d78"

	ringColor = "#3d5c66"
	litColor  = "#1a6f82"

	litDamage = 3.0
	litHitCD  = 0.4

	arriveR     = 12.0
	junctionCD  = 0.18
	offRailSlop = 3.0
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindMeander, assets)
	p.Register(unit.Spec{
		Kind:    KindMeander,
		Role:    unit.RoleFighter,
		Radius:  bodyRadius,
		MaxHP:   bodyHP,
		Speed:   cruise,
		Vision:  0,
		Fighter: true,
		Look:    unit.Look{Color: bodyColor, FX: []string{"meander"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &迷走{}
	})
	p.Register(unit.Spec{
		Kind:    KindRing,
		Role:    unit.RoleHelper,
		Radius:  100,
		MaxHP:   1,
		Speed:   0,
		Vision:  0,
		Fighter: false,
		// 不进 #over：落在 #units，被场地 clip-path 裁掉场外弧，且 z-index 低于战斗机。
		Look: unit.Look{Color: ringColor, Ring: true, FX: []string{"meander-ring"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return idle{}
	})
	p.Register(unit.Spec{
		Kind:    KindLit,
		Role:    unit.RoleHelper,
		Radius:  100,
		MaxHP:   1,
		Speed:   0,
		Vision:  9999,
		Fighter: false,
		Look:    unit.Look{Color: litColor, Ring: true, FX: []string{"meander-lit"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &点亮{slot: info.Slot}
	})
}

type idle struct{}

func (idle) Handle(unit.Context, unit.Event) {}

type 迷走 struct {
	booted bool
	maze   maze
	ring   int
	spin   float64 // +1 逆时针，-1 顺时针
	slot   int
	x, y   float64

	hasLastJ  bool
	lastJX    float64
	lastJY    float64
	ignoreJ   float64
	litID     uint64
	ringIDs   []uint64
	drifts    []ringDrift
	lastT     float64
	rng       *rand.Rand
	flipUntil float64
}

func (a *迷走) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		a.slot = e.Self.Slot
		a.x, a.y = e.Self.X, e.Self.Y
		a.ensureRNG(ctx.Stream)
		if !a.booted {
			a.boot(ctx, e)
			return
		}
		a.stepDrift(e)
		a.syncHelpers(ctx, e)
		a.rail(ctx, e)
	case unit.WallHit:
		a.flipSpin(e.Time)
	case unit.Collision:
		a.onBump(e)
	}
}

// onBump：被其他球撞时，看法线在当前切向上的分量——相逆才反绕，同向或近乎径向不反。
func (a *迷走) onBump(e unit.Collision) {
	if !a.booted || a.spin == 0 {
		return
	}
	if e.Time+1e-9 < a.flipUntil {
		return
	}
	if a.ring < 0 || a.ring >= len(a.maze.rings) {
		return
	}
	// 弹/弧刮到不算「球撞球」
	if e.Other.Role == unit.RoleProjectile {
		return
	}
	nlen := math.Hypot(e.NX, e.NY)
	if nlen < 1e-9 {
		return
	}
	nx, ny := e.NX/nlen, e.NY/nlen
	r := a.maze.rings[a.ring]
	_, _, ang := pointOnRing(r, a.x, a.y)
	// 当前前进切向（与 tangentVel 同向）
	tx := -math.Sin(ang) * a.spin
	ty := math.Cos(ang) * a.spin
	// 引擎把被撞方速度写成沿法线；法线在切向上与前进相逆 → 反绕
	if nx*tx+ny*ty < -0.08 {
		a.flipSpin(e.Time)
	}
}

func (a *迷走) flipSpin(t float64) {
	if t+1e-9 < a.flipUntil {
		return
	}
	if a.spin == 0 {
		a.spin = 1
	}
	a.spin = -a.spin
	a.flipUntil = t + 0.08
	a.ignoreJ = t + junctionCD
}

func (a *迷走) ensureRNG(id uint64) {
	if a.rng != nil {
		return
	}
	a.rng = unit.NewRNG(id)
}

func (a *迷走) boot(ctx unit.Context, s unit.Sense) {
	m, ok := buildMaze(s.Field, a.rng)
	if !ok || len(m.rings) == 0 {
		return
	}
	a.maze = m
	a.booted = true
	a.hasLastJ = false
	a.ringIDs = make([]uint64, len(m.rings))
	a.drifts = initDrifts(m, a.rng, s.Time)
	a.lastT = s.Time

	for _, r := range m.rings {
		ctx.Out <- ctx.Born(unit.Spawn{
			Kind: KindRing, X: r.x, Y: r.y,
			OwnerID: ctx.ID, Slot: s.Self.Slot,
		})
	}
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: KindLit, X: m.rings[0].x, Y: m.rings[0].y,
		OwnerID: ctx.ID, Slot: s.Self.Slot,
	})

	a.ring = nearestRing(m.rings, s.Self.X, s.Self.Y)
	if a.rng.Float64() < 0.5 {
		a.spin = 1
	} else {
		a.spin = -1
	}
	r := m.rings[a.ring]
	px, py, ang := pointOnRing(r, s.Self.X, s.Self.Y)
	vx, vy := tangentVel(r, ang, a.spin, cruise)
	ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: px, Y: py}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: vx, VY: vy}
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: cruise}
}

func (a *迷走) stepDrift(s unit.Sense) {
	dt := 0.0
	if a.lastT > 0 {
		dt = s.Time - a.lastT
	}
	a.lastT = s.Time
	if dt <= 0 || dt > 0.5 {
		return
	}
	driftCenters(a.maze.rings, a.drifts, s.Field, a.rng, s.Time, dt)
	a.maze.junc = fieldJunctions(s.Field, a.maze.rings)
}

func (a *迷走) syncHelpers(ctx unit.Context, s unit.Sense) {
	type cand struct {
		id uint64
		x  float64
		y  float64
		r  float64
	}
	var rings []cand
	var lit *cand
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID != ctx.ID {
			continue
		}
		switch o.Kind {
		case KindRing:
			rings = append(rings, cand{id: o.ID, x: o.X, y: o.Y, r: o.Radius})
		case KindLit:
			c := cand{id: o.ID, x: o.X, y: o.Y, r: o.Radius}
			lit = &c
			a.litID = o.ID
		}
	}
	used := make([]bool, len(a.maze.rings))
	for _, c := range rings {
		best := -1
		for i, id := range a.ringIDs {
			if id == c.id && !used[i] {
				best = i
				break
			}
		}
		if best < 0 {
			bestD := math.Inf(1)
			for i, r := range a.maze.rings {
				if used[i] {
					continue
				}
				d := math.Hypot(c.x-r.x, c.y-r.y)
				if d < bestD {
					best, bestD = i, d
				}
			}
		}
		if best < 0 {
			continue
		}
		used[best] = true
		a.ringIDs[best] = c.id
		want := a.maze.rings[best]
		if math.Hypot(c.x-want.x, c.y-want.y) > 0.5 {
			ctx.Out <- unit.Teleport{UnitID: c.id, X: want.x, Y: want.y}
		}
		if math.Abs(c.r-want.r) > 0.5 {
			ctx.Out <- unit.SetRadius{UnitID: c.id, Radius: want.r}
		}
	}
	if lit == nil || a.ring < 0 || a.ring >= len(a.maze.rings) {
		return
	}
	r := a.maze.rings[a.ring]
	if math.Hypot(lit.x-r.x, lit.y-r.y) > 0.5 {
		ctx.Out <- unit.Teleport{UnitID: lit.id, X: r.x, Y: r.y}
	}
	if math.Abs(lit.r-r.r) > 0.5 {
		ctx.Out <- unit.SetRadius{UnitID: lit.id, Radius: r.r}
	}
}

func (a *迷走) rail(ctx unit.Context, s unit.Sense) {
	if len(a.maze.rings) == 0 {
		return
	}
	if a.spin == 0 {
		a.spin = 1
	}
	if a.ring < 0 || a.ring >= len(a.maze.rings) {
		a.ring = nearestRing(a.maze.rings, s.Self.X, s.Self.Y)
	}
	r := a.maze.rings[a.ring]
	dist := math.Hypot(s.Self.X-r.x, s.Self.Y-r.y)
	if math.Abs(dist-r.r) > offRailSlop || dist < 1e-6 {
		a.ring = nearestRing(a.maze.rings, s.Self.X, s.Self.Y)
		r = a.maze.rings[a.ring]
	}
	px, py, ang := pointOnRing(r, s.Self.X, s.Self.Y)
	if jx, jy, hopped := a.tryJunction(s, px, py); hopped {
		px, py = jx, jy
		r = a.maze.rings[a.ring]
		px, py, ang = pointOnRing(r, px, py)
	}
	if math.Hypot(s.Self.X-px, s.Self.Y-py) > 0.5 {
		ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: px, Y: py}
	}
	speed := math.Hypot(s.Self.VX, s.Self.VY)
	if speed < cruise*0.5 {
		speed = cruise
	}
	if speed > cruise*1.5 {
		speed = cruise
	}
	vx, vy := tangentVel(r, ang, a.spin, speed)
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: vx, VY: vy}
}

func (a *迷走) tryJunction(s unit.Sense, x, y float64) (jx, jy float64, ok bool) {
	if s.Time+1e-9 < a.ignoreJ {
		return 0, 0, false
	}
	best := -1
	bestD := arriveR
	for i, j := range a.maze.junc {
		if j.a != a.ring && j.b != a.ring {
			continue
		}
		d := math.Hypot(x-j.x, y-j.y)
		if d < bestD {
			best, bestD = i, d
		}
	}
	if best < 0 {
		a.hasLastJ = false
		return 0, 0, false
	}
	j := a.maze.junc[best]
	// 交点表每拍重算，下标不稳；用位置认「刚走过的岔口」。
	if a.hasLastJ && math.Hypot(j.x-a.lastJX, j.y-a.lastJY) < arriveR {
		return 0, 0, false
	}
	exits := a.legalExits(j)
	if len(exits) == 0 {
		exits = []exit{{ring: a.ring, spin: -a.spin}}
	}
	pick := exits[a.rng.IntN(len(exits))]
	a.ring = pick.ring
	a.spin = pick.spin
	a.hasLastJ = true
	a.lastJX, a.lastJY = j.x, j.y
	a.ignoreJ = s.Time + junctionCD
	return j.x, j.y, true
}

type exit struct {
	ring int
	spin float64
}

func (a *迷走) legalExits(j junction) []exit {
	cands := []exit{
		{ring: j.a, spin: 1},
		{ring: j.a, spin: -1},
		{ring: j.b, spin: 1},
		{ring: j.b, spin: -1},
	}
	out := make([]exit, 0, 4)
	for _, e := range cands {
		// 禁原路折返：同一环线且反绕向
		if e.ring == a.ring && e.spin == -a.spin {
			continue
		}
		out = append(out, e)
	}
	return out
}
