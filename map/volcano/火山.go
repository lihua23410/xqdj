// 火山：六边形外形，场心常驻熔岩；定时喷发在可进入区域落下三摊有寿命的熔岩。
package 火山

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
	_ "xqdj/map/hex" // ShapeHex 判定在六边形包
)

const (
	Name = "火山"

	KindVolcano = "火山"
	KindLava    = "熔岩"

	craterR    = 48.0
	puddleR    = 22.0
	puddleLife = 6.0
	lavaMul    = 0.5
	dmgGap     = 0.5
	dmgAmt     = 1.0
	eruptFirst = 4.0
	eruptEvery = 10.0
	windSec    = 1.0
	lobFlight = 0.55 // 从场心抛到落点的飞行秒数
	// 圆心距：小摊互不重叠、也不压火山口
	puddleGap = puddleR*2 + 8 // 52
	craterGap = craterR + puddleR + 8 // 78
	eruptN    = 3

	magmaColor = "#c43c12"
)

//go:embed fx
var assets embed.FS

func Field() unit.Field {
	return unit.Field{
		Name: Name, Shape: unit.ShapeHex, Extent: unit.HexRadius,
		BootKind: KindVolcano,
	}
}

func init() {
	unit.RegisterField(Field())

	p := unit.NewPack(KindVolcano, assets)
	// 开打冒出的就是火山口本体（半径 craterR），不再另 Spawn 一层「火山口」。
	p.Register(unit.Spec{
		Kind: KindVolcano, Role: unit.RoleHelper, Radius: craterR, MaxHP: 1,
		Speed: 0, Vision: 9999, Fighter: false,
		Look: unit.Look{Color: magmaColor, Glow: true, FX: []string{"volcano-crater"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &火山{nextErupt: eruptFirst}
	})
	p.Register(unit.Spec{
		Kind: KindLava, Role: unit.RoleHelper, Radius: puddleR, MaxHP: 1,
		Speed: 0, Vision: 0, Fighter: false,
		Look: unit.Look{Color: magmaColor, Glow: true, FX: []string{"volcano-lava"}},
	}, func(unit.SpawnInfo) unit.Actor { return lavaMark{} })
}

type lavaMark struct{}

func (lavaMark) Handle(unit.Context, unit.Event) {}

type puddle struct {
	id    uint64
	x, y  float64
	until float64
}

type lob struct {
	x, y float64
	at   float64
}

type 火山 struct {
	booted    bool
	rng       *rand.Rand
	nextErupt float64
	winding   bool
	windUntil float64
	spots     [eruptN]struct{ x, y float64 }
	lobs      []lob
	puddles   []puddle
	hitAt     map[uint64]float64
	slowTok   map[uint64]uint64
	nextTok   uint64
}

func (a *火山) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !a.booted {
		a.boot(ctx, s)
	}
	a.tickErupt(ctx, s)
	a.flushLobs(ctx, s)
	a.expirePuddles(ctx, s)
	a.tickLava(ctx, s)
}

func (a *火山) boot(ctx unit.Context, s unit.Sense) {
	a.booted = true
	a.ensureRNG(ctx.ID, s.Time)
	a.hitAt = map[uint64]float64{}
	a.slowTok = map[uint64]uint64{}
}

func (a *火山) ensureRNG(id uint64, t float64) {
	if a.rng != nil {
		return
	}
	a.rng = rand.New(rand.NewPCG(id, uint64(t*1e6)+1))
}

func (a *火山) tickErupt(ctx unit.Context, s unit.Sense) {
	if !a.winding && s.Time+1e-9 >= a.nextErupt {
		a.planSpots()
		a.winding = true
		a.windUntil = s.Time + windSec
	}
	if !a.winding {
		return
	}
	// 预警全程每帧推喷发/落点预兆，和刑具 drop 一样靠客户端续命。
	ctx.Out <- unit.FX{
		Name: "erupt", Kind: KindVolcano, UnitID: ctx.ID,
		X: 0, Y: 0, Amount: craterR,
	}
	for i := 0; i < eruptN; i++ {
		ctx.Out <- unit.FX{
			Name: "warn", Kind: KindVolcano, UnitID: ctx.ID,
			X: a.spots[i].x, Y: a.spots[i].y, Amount: puddleR,
		}
	}
	if s.Time+1e-9 < a.windUntil {
		return
	}
	a.winding = false
	a.nextErupt += eruptEvery
	// 先抛投：客户端从场心飞到落点；落地后再 Spawn。
	for i := 0; i < eruptN; i++ {
		a.lobs = append(a.lobs, lob{
			x: a.spots[i].x, y: a.spots[i].y, at: s.Time + lobFlight,
		})
	}
}

func (a *火山) flushLobs(ctx unit.Context, s unit.Sense) {
	if len(a.lobs) == 0 {
		return
	}
	keep := a.lobs[:0]
	for _, L := range a.lobs {
		// 飞行全程每帧续推 lob，避免单帧 effects 被 WS 挤掉。
		ctx.Out <- unit.FX{
			Name: "lob", Kind: KindVolcano, UnitID: ctx.ID,
			X: L.x, Y: L.y, Amount: puddleR,
		}
		if s.Time+1e-9 < L.at {
			keep = append(keep, L)
			continue
		}
		ctx.Out <- unit.Spawn{
			Kind: KindLava, X: L.x, Y: L.y,
			OwnerID: ctx.ID, Slot: s.Self.Slot,
		}
		ctx.Out <- unit.FX{
			Name: "land", Kind: KindVolcano, UnitID: ctx.ID,
			X: L.x, Y: L.y, Amount: puddleR,
		}
		a.puddles = append(a.puddles, puddle{
			x: L.x, y: L.y, until: s.Time + puddleLife,
		})
	}
	a.lobs = keep
}

func (a *火山) planSpots() {
	used := make([][2]float64, 0, eruptN)
	for i := 0; i < eruptN; i++ {
		x, y := a.randSpot(used)
		used = append(used, [2]float64{x, y})
		a.spots[i].x, a.spots[i].y = x, y
	}
}

func (a *火山) randSpot(used [][2]float64) (float64, float64) {
	a.ensureRNG(1, 0)
	f := unit.LiveField()
	rng := a.rng
	for n := 0; n < 120; n++ {
		x, y, ok := f.RandomWalkable(rng, puddleR)
		if !ok {
			break
		}
		if !a.lavaClear(x, y, used) {
			continue
		}
		return x, y
	}
	// 兜底：在圆环上找空位，仍避开口子与已有摊
	for n := 0; n < 48; n++ {
		ang := rng.Float64() * 2 * math.Pi
		rad := craterGap + rng.Float64()*(f.Extent*0.55)
		x, y := f.Clamp(math.Cos(ang)*rad, math.Sin(ang)*rad, puddleR)
		if a.lavaClear(x, y, used) {
			return x, y
		}
	}
	x, y, ok := f.RandomWalkable(rng, puddleR)
	if ok {
		return x, y
	}
	return f.Clamp(0, craterGap+20, puddleR)
}

// lavaClear：不与火山口、本轮已定点、在飞/在场的小摊重叠。
func (a *火山) lavaClear(x, y float64, used [][2]float64) bool {
	if math.Hypot(x, y) < craterGap {
		return false
	}
	for _, p := range used {
		if math.Hypot(x-p[0], y-p[1]) < puddleGap {
			return false
		}
	}
	for _, p := range a.puddles {
		if math.Hypot(x-p.x, y-p.y) < puddleGap {
			return false
		}
	}
	for _, L := range a.lobs {
		if math.Hypot(x-L.x, y-L.y) < puddleGap {
			return false
		}
	}
	return true
}

func (a *火山) expirePuddles(ctx unit.Context, s unit.Sense) {
	a.bindPuddles(s)
	keep := a.puddles[:0]
	for _, p := range a.puddles {
		if s.Time+1e-9 < p.until {
			keep = append(keep, p)
			continue
		}
		id := p.id
		if id == 0 {
			id = a.findLavaAt(s, p.x, p.y)
		}
		if id != 0 {
			ctx.Out <- unit.Despawn{UnitID: id}
		}
	}
	a.puddles = keep
}

func (a *火山) findLavaAt(s unit.Sense, x, y float64) uint64 {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind != KindLava || o.OwnerID != s.Self.ID {
			continue
		}
		if math.Hypot(o.X-x, o.Y-y) <= 1 {
			return o.ID
		}
	}
	return 0
}

func (a *火山) bindPuddles(s unit.Sense) {
	for i := range a.puddles {
		p := &a.puddles[i]
		if p.id != 0 {
			continue
		}
		for j := range s.Nearby {
			o := &s.Nearby[j]
			if o.Kind != KindLava || o.OwnerID != s.Self.ID {
				continue
			}
			if math.Hypot(o.X-p.x, o.Y-p.y) > 1 {
				continue
			}
			taken := false
			for _, q := range a.puddles {
				if q.id == o.ID {
					taken = true
					break
				}
			}
			if taken {
				continue
			}
			p.id = o.ID
			break
		}
	}
}

func (a *火山) tickLava(ctx unit.Context, s unit.Sense) {
	inLava := map[uint64]unit.Snapshot{}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !lavaTarget(*o) {
			continue
		}
		if a.covers(o.X, o.Y, o.Radius) {
			inLava[o.ID] = *o
		}
	}
	for id := range inLava {
		a.applySlow(ctx, id)
		if last, ok := a.hitAt[id]; ok && s.Time+1e-9 < last+dmgGap {
			continue
		}
		a.hitAt[id] = s.Time
		ctx.Out <- unit.Damage{
			From: 0, To: id, Amount: dmgAmt, NoFreeze: true,
		}
	}
	for id, tok := range a.slowTok {
		if _, ok := inLava[id]; ok {
			continue
		}
		ctx.Out <- unit.RemoveFSComponent{UnitID: id, Token: tok}
		delete(a.slowTok, id)
	}
}

func (a *火山) covers(x, y, r float64) bool {
	if math.Hypot(x, y) <= craterR+r {
		return true
	}
	for _, p := range a.puddles {
		if math.Hypot(x-p.x, y-p.y) <= puddleR+r {
			return true
		}
	}
	return false
}

func (a *火山) applySlow(ctx unit.Context, id uint64) {
	ctx.Out <- unit.AddFSComponent{
		UnitID: id, Zone: unit.FSZoneM, Token: a.mint(id),
		Value: lavaMul,
	}
}

func (a *火山) mint(id uint64) uint64 {
	if tok, ok := a.slowTok[id]; ok {
		return tok
	}
	a.nextTok++
	a.slowTok[id] = a.nextTok
	return a.nextTok
}

func lavaTarget(o unit.Snapshot) bool {
	if o.Nonsolid {
		return false
	}
	return o.Role == unit.RoleFighter || o.Mortal
}
