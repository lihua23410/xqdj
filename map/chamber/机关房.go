// 机关房：正方形判定与场皮、九宫/毒气/上锁全在本包。
package 机关房

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

const (
	Name = "机关房"

	KindChamber = "机关房"

	waitSec  = 15.0
	lockSec  = 7.0
	windSec  = 1.0
	wallHalf = 6.0
	gasMul   = 0.5
	dmgGap   = 0.5
	dmgAmt   = 1.0
	gridN    = 3

	lookColor = "#3d5c4a"
)

//go:embed fx
var assets embed.FS

// squareHalf 六边形内最大轴对齐正方形的半边长（场心到边）。
func squareHalf() float64 {
	return unit.HexRadius * (3 - math.Sqrt(3)) / 2
}

func Field() unit.Field {
	return unit.Field{
		Name: Name, Shape: unit.ShapeSquare, Extent: unit.HexRadius,
		BootKind: KindChamber,
	}
}

func init() {
	unit.RegisterField(Field())

	p := unit.NewPack(KindChamber, assets)
	p.Register(unit.Spec{
		Kind: KindChamber, Role: unit.RoleHelper, Radius: 1, MaxHP: 1,
		Speed: 0, Vision: 9999, Fighter: false, Nonsolid: true,
		Look: unit.Look{Color: lookColor, FX: []string{"chamber-lock"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &机关房{nextLock: waitSec}
	})
}

type cellRect struct {
	x0, y0, x1, y1 float64
}

type 机关房 struct {
	booted    bool
	rng       *rand.Rand
	nextLock  float64
	winding   bool
	windUntil float64
	locking   bool
	lockUntil float64
	toxic     [2]int
	gas       [2]cellRect
	hitAt     map[uint64]float64
	slowTok   map[uint64]uint64
	nextTok   uint64
}

func (a *机关房) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !a.booted {
		a.boot(ctx, s)
	}
	a.tickCycle(ctx, s)
	a.tickGas(ctx, s)
}

func (a *机关房) boot(ctx unit.Context, s unit.Sense) {
	a.booted = true
	a.ensureRNG(ctx.ID, s.Time)
	a.hitAt = map[uint64]float64{}
	a.slowTok = map[uint64]uint64{}
}

func (a *机关房) ensureRNG(id uint64, t float64) {
	if a.rng != nil {
		return
	}
	a.rng = rand.New(rand.NewPCG(id, uint64(t*1e6)+1))
}

func (a *机关房) tickCycle(ctx unit.Context, s unit.Sense) {
	if a.locking {
		a.emitLockFX(ctx)
		if s.Time+1e-9 >= a.lockUntil {
			a.locking = false
			a.nextLock = s.Time + waitSec
		}
		return
	}
	if !a.winding && s.Time+1e-9 >= a.nextLock-windSec {
		a.pickToxic()
		a.winding = true
		a.windUntil = a.nextLock
		if a.windUntil < s.Time {
			a.windUntil = s.Time
		}
	}
	if !a.winding {
		return
	}
	a.emitWarnFX(ctx)
	if s.Time+1e-9 < a.windUntil {
		return
	}
	a.winding = false
	a.beginLock(ctx, s)
}

func (a *机关房) beginLock(ctx unit.Context, s unit.Sense) {
	a.locking = true
	a.lockUntil = s.Time + lockSec
	half := squareHalf()
	a.placeGridWalls(ctx, s, half)
	for i := 0; i < 2; i++ {
		a.gas[i] = cellInner(half, a.toxic[i])
	}
	a.emitLockFX(ctx)
}

func (a *机关房) placeGridWalls(ctx unit.Context, s unit.Sense, half float64) {
	// 有寿胶囊墙：不改引擎硬墙到期逻辑；刮伤 0、方端。
	divs := []float64{-half / 3, half / 3}
	for _, x := range divs {
		ctx.Out <- unit.PlaceWall{
			OwnerID: ctx.ID, Slot: s.Self.Slot, Kind: KindChamber,
			X1: x, Y1: -half, X2: x, Y2: half,
			Radius: wallHalf, Life: lockSec, Amount: 0, Square: true,
		}
	}
	for _, y := range divs {
		ctx.Out <- unit.PlaceWall{
			OwnerID: ctx.ID, Slot: s.Self.Slot, Kind: KindChamber,
			X1: -half, Y1: y, X2: half, Y2: y,
			Radius: wallHalf, Life: lockSec, Amount: 0, Square: true,
		}
	}
}

func (a *机关房) pickToxic() {
	a.ensureRNG(1, 0)
	i0 := a.rng.IntN(gridN * gridN)
	i1 := a.rng.IntN(gridN*gridN - 1)
	if i1 >= i0 {
		i1++
	}
	a.toxic[0], a.toxic[1] = i0, i1
}

func cellInner(half float64, idx int) cellRect {
	ci := idx % gridN
	cj := idx / gridN
	cell := (2 * half) / gridN
	x0 := -half + float64(ci)*cell
	x1 := x0 + cell
	y0 := -half + float64(cj)*cell
	y1 := y0 + cell
	if ci > 0 {
		x0 += wallHalf
	}
	if ci < gridN-1 {
		x1 -= wallHalf
	}
	if cj > 0 {
		y0 += wallHalf
	}
	if cj < gridN-1 {
		y1 -= wallHalf
	}
	return cellRect{x0, y0, x1, y1}
}

func (a *机关房) emitWarnFX(ctx unit.Context) {
	half := squareHalf()
	ctx.Out <- unit.FX{
		Name: "grid", Kind: KindChamber, UnitID: ctx.ID, Amount: half,
	}
	for i := 0; i < 2; i++ {
		r := cellInner(half, a.toxic[i])
		ctx.Out <- unit.FX{
			Name: "warn", Kind: KindChamber, UnitID: ctx.ID,
			X: r.x0, Y: r.y0, VX: r.x1, VY: r.y1,
		}
	}
}

func (a *机关房) emitLockFX(ctx unit.Context) {
	half := squareHalf()
	ctx.Out <- unit.FX{
		Name: "grid", Kind: KindChamber, UnitID: ctx.ID, Amount: half,
	}
	for i := 0; i < 2; i++ {
		r := a.gas[i]
		ctx.Out <- unit.FX{
			Name: "gas", Kind: KindChamber, UnitID: ctx.ID,
			X: r.x0, Y: r.y0, VX: r.x1, VY: r.y1,
		}
	}
}

func (a *机关房) tickGas(ctx unit.Context, s unit.Sense) {
	if !a.locking {
		for id, tok := range a.slowTok {
			ctx.Out <- unit.RemoveFSComponent{UnitID: id, Token: tok}
			delete(a.slowTok, id)
		}
		return
	}
	inGas := map[uint64]struct{}{}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !gasTarget(*o) {
			continue
		}
		if !a.covers(o.X, o.Y, o.Radius) {
			continue
		}
		inGas[o.ID] = struct{}{}
		a.applySlow(ctx, o.ID)
		if last, ok := a.hitAt[o.ID]; ok && s.Time+1e-9 < last+dmgGap {
			continue
		}
		a.hitAt[o.ID] = s.Time
		ctx.Out <- unit.Damage{
			From: 0, To: o.ID, Amount: dmgAmt, NoFreeze: true,
		}
	}
	for id, tok := range a.slowTok {
		if _, ok := inGas[id]; ok {
			continue
		}
		ctx.Out <- unit.RemoveFSComponent{UnitID: id, Token: tok}
		delete(a.slowTok, id)
	}
}

func (a *机关房) covers(x, y, r float64) bool {
	for i := 0; i < 2; i++ {
		if circleHitsRect(x, y, r, a.gas[i]) {
			return true
		}
	}
	return false
}

func circleHitsRect(cx, cy, r float64, g cellRect) bool {
	qx := clamp(cx, g.x0, g.x1)
	qy := clamp(cy, g.y0, g.y1)
	return math.Hypot(cx-qx, cy-qy) <= r+1e-9
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (a *机关房) applySlow(ctx unit.Context, id uint64) {
	ctx.Out <- unit.AddFSComponent{
		UnitID: id, Zone: unit.FSZoneM, Token: a.mint(id),
		Value: gasMul,
	}
}

func (a *机关房) mint(id uint64) uint64 {
	if tok, ok := a.slowTok[id]; ok {
		return tok
	}
	a.nextTok++
	a.slowTok[id] = a.nextTok
	return a.nextTok
}

func gasTarget(o unit.Snapshot) bool {
	if o.Nonsolid {
		return false
	}
	return o.Role == unit.RoleFighter || o.Mortal
}
