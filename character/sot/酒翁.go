// 酒翁自己不出伤。战斗状态围绕酒坛组织：定时从敌人头顶砸酒，碎了留场，踩场叠醉意。
package 酒翁

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

const KindSot = "酒翁"
const KindSotWine = "酒翁酒"
const KindSotStain = "酒翁酒场"

const (
	bodyRadius = 18.0
	bodyHP     = 100.0
	cruise     = 165.0
	vision     = 9999.0
	bodyColor  = "#b1a78d"

	smashFirst = 1.0 // 开局第一轮砸酒
	smashGap   = 3.0 // 之后每隔这么久再砸
	fillGap    = 10.0
	fillStacks = 3

	gap12 = 0.8 // 第 1、2 坛间隔
	gapN  = 0.2 // 再往后每坛间隔

	drunkKind = "醉意"
	drunkIcon = "/ball/酒翁/status/xiaojiu.png"
	drunkMul  = 0.8
)

//go:embed fx status
var assets embed.FS

func init() {
	p := unit.NewPack(KindSot, assets)
	p.Register(unit.Spec{
		Kind: KindSot, Role: unit.RoleFighter, Radius: bodyRadius, MaxHP: bodyHP,
		Speed: cruise, Vision: vision, Fighter: true,
		Look: unit.Look{Color: bodyColor, Glow: true, FX: []string{"sot"}},
	}, func(unit.SpawnInfo) unit.Actor { return &酒翁{} })
	p.Register(unit.Spec{
		Kind: KindSotWine, Role: unit.RoleProjectile, Radius: wineRadius, MaxHP: 1,
		Speed: wineFall, Vision: 0, Fighter: false,
		Look: unit.Look{Color: bodyColor, Overlay: true, FX: []string{"sot-wine"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &酒{owner: info.OwnerID, slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind: KindSotStain, Role: unit.RoleHelper, Radius: stainRadius, MaxHP: 1,
		Speed: 0, Vision: stainRadius + 80, Fighter: false,
		Look: unit.Look{Color: "#7a3b28", Glow: true, FX: []string{"sot-stain"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &酒场{owner: info.OwnerID, slot: info.Slot}
	})
}

type drop struct {
	at float64
	x  float64
}

type 酒翁 struct {
	booted    bool
	first     bool
	slot      int
	rng       *rand.Rand
	nextSmash float64
	nextFill  float64
	drops     []drop
	slowTok   map[uint64]uint64
	nextTok   uint64
}

func (a *酒翁) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	a.onSense(ctx, s)
}

func (a *酒翁) onSense(ctx unit.Context, s unit.Sense) {
	a.slot = s.Self.Slot
	if !a.booted {
		a.booted = true
		a.first = true
		a.rng = rand.New(rand.NewPCG(ctx.ID, ctx.ID))
		a.nextSmash = s.Time + smashFirst
		a.nextFill = s.Time + fillGap
		a.slowTok = map[uint64]uint64{}
	}
	n := markStacks(s.Self, drunkKind)
	if s.Time+1e-9 >= a.nextFill {
		a.nextFill += fillGap
		if n <= 0 {
			n = fillStacks
			ctx.Out <- unit.StackMark{UnitID: ctx.ID, Kind: drunkKind, Delta: fillStacks, Icon: drunkIcon}
		}
	}
	if s.Time+1e-9 >= a.nextSmash {
		a.nextSmash += smashGap
		a.queueSmash(ctx, s, n)
	}
	a.flushDrops(ctx, s)
	a.syncDrunk(ctx, s)
}

func (a *酒翁) queueSmash(ctx unit.Context, s unit.Sense, have int) {
	t := unit.Seek(s)
	if t == nil {
		return
	}
	n := 1
	if a.first {
		a.first = false
		n = a.rng.IntN(3) + 1
	} else if have <= 0 {
		n = 1
	} else {
		n = a.rng.IntN(have) + 1
		ctx.Out <- unit.StackMark{UnitID: ctx.ID, Kind: drunkKind, Delta: -n, Icon: drunkIcon}
	}
	x := t.X
	for i := 0; i < n; i++ {
		a.drops = append(a.drops, drop{at: s.Time + dropDelay(i), x: x})
	}
}

func dropDelay(i int) float64 {
	if i <= 0 {
		return 0
	}
	return gap12 + float64(i-1)*gapN
}

func (a *酒翁) flushDrops(ctx unit.Context, s unit.Sense) {
	keep := a.drops[:0]
	for _, d := range a.drops {
		if s.Time+1e-9 < d.at {
			keep = append(keep, d)
			continue
		}
		a.spawnWine(ctx, s, d.x)
	}
	a.drops = keep
}

func (a *酒翁) spawnWine(ctx unit.Context, s unit.Sense, x float64) {
	px, py := dropPoint(s.Field, x, wineRadius)
	px, py = insetFromOutline(s.Field, px, py, wineRadius, wineSpawnPad)
	vx := (a.rng.Float64()*2 - 1) * wineToss
	vy := -wineDrop
	vx, vy = clipOutward(px, py, vx, vy)
	ctx.Out <- unit.Spawn{
		Kind: KindSotWine, OwnerID: ctx.ID, Slot: a.slot,
		X: px, Y: py, VX: vx, VY: vy,
	}
	ctx.Out <- unit.FX{Name: "sot-drop", Kind: ctx.Kind, X: px, Y: py, Slot: a.slot}
}

// dropPoint 敌人当下 x 的场顶，只认外轮廓，不躲墙、不抄 280。
func dropPoint(f unit.Field, x, radius float64) (float64, float64) {
	x = clampXOnOutline(f, x, radius)
	lo, hi := 0.0, f.Extent
	if hi < unit.MinExtent {
		hi = unit.HexRadius
	}
	if !f.OutlineContains(x, lo, radius) {
		x = 0
	}
	if f.OutlineContains(x, hi, radius) {
		return x, hi
	}
	for i := 0; i < 28; i++ {
		mid := 0.5 * (lo + hi)
		if f.OutlineContains(x, mid, radius) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return x, lo
}

func clampXOnOutline(f unit.Field, x, radius float64) float64 {
	if f.OutlineContains(x, 0, radius) {
		return x
	}
	lo, hi := 0.0, x
	if x < 0 {
		lo, hi = x, 0.0
	}
	for i := 0; i < 24; i++ {
		mid := 0.5 * (lo + hi)
		if f.OutlineContains(mid, 0, radius) {
			if x >= 0 {
				lo = mid
			} else {
				hi = mid
			}
		} else if x >= 0 {
			hi = mid
		} else {
			lo = mid
		}
	}
	if x >= 0 {
		return lo
	}
	return hi
}

func insetFromOutline(f unit.Field, x, y, radius, pad float64) (float64, float64) {
	need := radius + pad
	if f.OutlineContains(x, y, need) {
		return x, y
	}
	lo, hi := 0.0, 1.0
	for i := 0; i < 24; i++ {
		mid := 0.5 * (lo + hi)
		if f.OutlineContains(x*mid, y*mid, need) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return x * lo, y * lo
}

func clipOutward(x, y, vx, vy float64) (float64, float64) {
	n2 := x*x + y*y
	if n2 < 1e-8 {
		return vx, vy
	}
	radial := (x*vx + y*vy) / n2
	if radial <= 0 {
		return vx, vy
	}
	return vx - x*radial, vy - y*radial
}

func (a *酒翁) syncDrunk(ctx unit.Context, s unit.Sense) {
	seen := map[uint64]struct{}{}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		seen[o.ID] = struct{}{}
		n := markStacks(*o, drunkKind)
		if n <= 0 {
			if tok, ok := a.slowTok[o.ID]; ok {
				ctx.Out <- unit.RemoveFSComponent{UnitID: o.ID, Token: tok}
				delete(a.slowTok, o.ID)
			}
			continue
		}
		ctx.Out <- unit.AddFSComponent{
			UnitID: o.ID, Zone: unit.FSZoneM, Token: a.mint(o.ID),
			Value: math.Pow(drunkMul, float64(n)),
		}
	}
	for id, tok := range a.slowTok {
		if _, ok := seen[id]; !ok {
			ctx.Out <- unit.RemoveFSComponent{UnitID: id, Token: tok}
			delete(a.slowTok, id)
		}
	}
}

func (a *酒翁) mint(id uint64) uint64 {
	if a.slowTok == nil {
		a.slowTok = map[uint64]uint64{}
	}
	if tok, ok := a.slowTok[id]; ok {
		return tok
	}
	a.nextTok++
	a.slowTok[id] = a.nextTok
	return a.nextTok
}

func markStacks(u unit.Snapshot, kind string) int {
	for _, m := range u.Marks {
		if m.Kind == kind {
			return m.Stacks
		}
	}
	return 0
}
