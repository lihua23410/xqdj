// 囚徒不转向。钩爪只钉场边。锁链不出伤。刑具三种各一落下。
package 囚徒

import (
	"embed"
	"math"
	"math/rand/v2"
	"xqdj/internal/unit"
)

const KindPrisoner = "囚徒"
const KindHook = "囚徒钩爪"
const KindCage = "囚徒囚笼"
const KindGallows = "囚徒绞刑架"
const KindChair = "囚徒电椅"

const (
	prisonerRadius = 18.0
	prisonerHP     = 100.0
	prisonerCruise = 180.0
	prisonerVision = 9999.0
	prisonerColor  = "#1a1a1a"

	chainLen = 350.0

	dropFirst   = 5.0
	dropEvery   = 12.0
	dropWind    = 0.3
	gearLife    = 10.0
	spotGap     = 110.0
	shockCruise = 360.0
	shockSec    = 5.0
	shockKind   = "触电"
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindPrisoner, assets)
	p.Register(unit.Spec{
		Kind:    KindPrisoner,
		Role:    unit.RoleFighter,
		Radius:  prisonerRadius,
		MaxHP:   prisonerHP,
		Speed:   prisonerCruise,
		Vision:  prisonerVision,
		Fighter: true,
		Look:    unit.Look{Color: prisonerColor, FX: []string{"prisoner"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &囚徒{nextDrop: dropFirst}
	})
	p.Register(unit.Spec{
		Kind:    KindHook,
		Role:    unit.RoleHelper,
		Radius:  8,
		MaxHP:   1,
		Speed:   0,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: "#b56a32", Overlay: true},
	}, func(unit.SpawnInfo) unit.Actor {
		return idle{}
	})
	p.Register(unit.Spec{
		Kind:    KindCage,
		Role:    unit.RoleHelper,
		Radius:  cageOuter,
		MaxHP:   1,
		Speed:   0,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: cageColor, Overlay: true, FX: []string{"cage"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &囚笼{owner: info.OwnerID, slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:    KindGallows,
		Role:    unit.RoleHelper,
		Radius:  gallowsR,
		MaxHP:   1,
		Speed:   0,
		Vision:  80,
		Fighter: false,
		Look:    unit.Look{Color: "#ad6327", Overlay: true, FX: []string{"gallows"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &绞刑架{owner: info.OwnerID, slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:    KindChair,
		Role:    unit.RoleHelper,
		Radius:  chairR,
		MaxHP:   1,
		Speed:   0,
		Vision:  9999,
		Fighter: false,
		Look:    unit.Look{Color: "#d3843d", Overlay: true, FX: []string{"chair"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &电椅{owner: info.OwnerID, slot: info.Slot}
	})
}

type idle struct{}

func (idle) Handle(unit.Context, unit.Event) {}

type 囚徒 struct {
	x, y       float64
	r          float64
	slot       int
	hooked     bool
	hx, hy     float64
	spin       float64
	nextDrop   float64
	winding    bool
	windUntil  float64
	spots      [3]vec
	kinds      [3]string
	shockUntil float64
	matchSpin  bool
	rng        *rand.Rand
}

type vec struct{ x, y float64 }

func (a *囚徒) Handle(ctx unit.Context, ev unit.Event) {
	if a.nextDrop == 0 {
		a.nextDrop = dropFirst
	}
	if unit.AcceptHit(ctx, ev) {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		a.x, a.y = e.Self.X, e.Self.Y
		a.slot = e.Self.Slot
		if e.Self.Radius > 0 {
			a.r = e.Self.Radius
		}
		a.tickShock(ctx, e)
		a.tickDrop(ctx, e)
		if !a.hooked {
			return
		}
		a.constrain(ctx, e)
		a.emitChain(ctx, e)
	case unit.WallHit:
		a.plant(ctx, e.NX, e.NY)
	}
}

func (a *囚徒) plant(ctx unit.Context, nx, ny float64) {
	r := a.r
	if r <= 0 {
		r = prisonerRadius
	}
	if _, ok := hexEdge(a.x, a.y, nx, ny, r); !ok {
		a.spin = 0
		a.matchSpin = true
		return
	}
	n := math.Hypot(nx, ny)
	if n < 1e-9 {
		return
	}
	nx, ny = nx/n, ny/n
	a.hx, a.hy = a.x+nx*r, a.y+ny*r
	a.hooked = true
	a.spin = 0
	a.matchSpin = false
	ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: KindHook}
	ctx.Out <- unit.Spawn{
		Kind: KindHook, X: a.hx, Y: a.hy,
		OwnerID: ctx.ID, Slot: a.slot,
	}
}

func (a *囚徒) constrain(ctx unit.Context, s unit.Sense) {
	dx := s.Self.X - a.hx
	dy := s.Self.Y - a.hy
	dist := math.Hypot(dx, dy)
	if dist+1e-9 < chainLen {
		a.spin = 0
		a.matchSpin = false
		return
	}
	if dist < 1e-9 {
		return
	}
	urx, ury := dx/dist, dy/dist
	if dist > chainLen {
		ctx.Out <- unit.Teleport{
			UnitID: ctx.ID,
			X:      a.hx + urx*chainLen,
			Y:      a.hy + ury*chainLen,
		}
	}
	vx, vy := s.Self.VX, s.Self.VY
	speed := math.Hypot(vx, vy)
	if s.Time < a.shockUntil && speed < shockCruise {
		speed = shockCruise
	}
	if speed < 1e-9 {
		speed = prisonerCruise
	}
	if a.spin == 0 {
		ccw := dx*vy-dy*vx > 0
		if a.matchSpin {
			if ccw {
				a.spin = 1
			} else {
				a.spin = -1
			}
			a.matchSpin = false
		} else if ccw {
			a.spin = -1
		} else {
			a.spin = 1
		}
	}
	ctx.Out <- unit.SetVelocity{
		UnitID: ctx.ID,
		VX:     -ury * a.spin * speed,
		VY:     urx * a.spin * speed,
	}
}

func (a *囚徒) tickShock(ctx unit.Context, s unit.Sense) {
	if hasMark(s.Self, shockKind) {
		a.shockUntil = s.Time + shockSec
		ctx.Out <- unit.ClearMarks{UnitID: ctx.ID, Kind: shockKind}
		ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: shockCruise}
		ctx.Out <- unit.FX{Name: "scream", Kind: ctx.Kind, UnitID: ctx.ID, Slot: s.Self.Slot, X: s.Self.X, Y: s.Self.Y}
	}
	if a.shockUntil > 0 && s.Time+1e-9 >= a.shockUntil {
		a.shockUntil = 0
		ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: prisonerCruise}
	}
}

func (a *囚徒) tickDrop(ctx unit.Context, s unit.Sense) {
	if !a.winding && s.Time+1e-9 >= a.nextDrop {
		a.planDrop(s)
		a.winding = true
		a.windUntil = s.Time + dropWind
	}
	if !a.winding {
		return
	}
	for i := 0; i < 3; i++ {
		ctx.Out <- unit.FX{
			Name: "drop", Kind: a.kinds[i], UnitID: ctx.ID, Slot: s.Self.Slot,
			X: a.spots[i].x, Y: a.spots[i].y,
		}
	}
	if s.Time+1e-9 < a.windUntil {
		return
	}
	a.winding = false
	a.nextDrop += dropEvery
	for i := 0; i < 3; i++ {
		ctx.Out <- unit.Spawn{
			Kind: a.kinds[i], X: a.spots[i].x, Y: a.spots[i].y,
			OwnerID: ctx.ID, Slot: s.Self.Slot,
		}
	}
}

func (a *囚徒) planDrop(s unit.Sense) {
	kinds := []string{KindCage, KindGallows, KindChair}
	rng := a.rng
	if rng == nil {
		rng = rand.New(rand.NewPCG(uint64(s.Time*1e6)+s.Self.ID, 1))
	}
	rng.Shuffle(3, func(i, j int) { kinds[i], kinds[j] = kinds[j], kinds[i] })
	copy(a.kinds[:], kinds)
	ex, ey := 0.0, 0.0
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role == unit.RoleFighter && o.Slot != s.Self.Slot {
			ex, ey = o.X, o.Y
			break
		}
	}
	a.spots[0] = clampFit(ex, ey)
	a.spots[1] = a.randSpot(rng, a.spots[0])
	a.spots[2] = a.randSpot(rng, a.spots[0], a.spots[1])
}

func (a *囚徒) randSpot(rng *rand.Rand, avoid ...vec) vec {
	reach := unit.HexRadius*math.Sqrt(3)/2 - cageFit
	if reach < 20 {
		reach = 20
	}
	for n := 0; n < 80; n++ {
		ang := rng.Float64() * 2 * math.Pi
		rad := math.Sqrt(rng.Float64()) * reach
		p := clampFit(math.Cos(ang)*rad, math.Sin(ang)*rad)
		ok := true
		for _, o := range avoid {
			if math.Hypot(p.x-o.x, p.y-o.y) < spotGap {
				ok = false
				break
			}
		}
		if ok {
			return p
		}
	}
	return clampFit(0, 0)
}

func (a *囚徒) emitChain(ctx unit.Context, s unit.Sense) {
	ctx.Out <- unit.FX{
		Name:   "chain",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		Slot:   s.Self.Slot,
		X:      s.Self.X,
		Y:      s.Self.Y,
		VX:     a.hx,
		VY:     a.hy,
		Amount: math.Hypot(s.Self.X-a.hx, s.Self.Y-a.hy),
	}
}

func clampFit(x, y float64) vec {
	if unit.HexContains(x, y, cageFit) {
		return vec{x, y}
	}
	lo, hi := 0.0, 1.0
	for i := 0; i < 24; i++ {
		mid := (lo + hi) / 2
		if unit.HexContains(x*mid, y*mid, cageFit) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return vec{x * lo, y * lo}
}

func hasMark(s unit.Snapshot, kind string) bool {
	for _, m := range s.Marks {
		if m.Kind == kind && m.Stacks > 0 {
			return true
		}
	}
	return false
}

func hexNormal(i int) (float64, float64) {
	a := (float64(i) + 0.5) * math.Pi / 3
	return math.Cos(a), math.Sin(a)
}

func hexEdge(x, y, nx, ny, radius float64) (int, bool) {
	n := math.Hypot(nx, ny)
	if n < 1e-9 {
		return 0, false
	}
	nx, ny = nx/n, ny/n
	ap := unit.HexRadius * math.Sqrt(3) / 2
	best := -1
	bestDot := 0.92
	for i := 0; i < 6; i++ {
		hx, hy := hexNormal(i)
		d := hx*nx + hy*ny
		if d > bestDot {
			bestDot = d
			best = i
		}
	}
	if best < 0 {
		return 0, false
	}
	hx, hy := hexNormal(best)
	if hx*x+hy*y <= ap-radius-8 {
		return 0, false
	}
	return best, true
}

func segHits(ax, ay, bx, by, px, py, pr, halfW float64) bool {
	dx, dy := bx-ax, by-ay
	l2 := dx*dx + dy*dy
	if l2 < 1e-12 {
		return math.Hypot(px-ax, py-ay) <= pr+halfW
	}
	t := ((px-ax)*dx + (py-ay)*dy) / l2
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	qx, qy := ax+t*dx, ay+t*dy
	return math.Hypot(px-qx, py-qy) <= pr+halfW
}
