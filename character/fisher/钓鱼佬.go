// 钓鱼佬不转向。开局在场边铺三个鱼塘，踩进去钓三秒，起竿把鱼甩向敌人。
package 钓鱼佬

import (
	"embed"
	"math"
	"math/rand/v2"
	"xqdj/internal/unit"
)

const KindFisher = "钓鱼佬"
const KindPond = "钓鱼佬鱼塘"
const KindFlood = "钓鱼佬场鱼塘"
const KindFish = "钓鱼佬鱼"
const KindFishMid = "钓鱼佬中鱼"
const KindFishBig = "钓鱼佬大鱼"

const (
	fisherRadius = 18.0
	fisherHP     = 100.0
	fisherCruise = 148.0
	fisherVision = 9999.0
	fisherColor  = "#e8b84a"
	fisherDR     = 0.5
	fishSec      = 3.0
	pondN        = 3
	pondRadius   = 42.0
	pondColor    = "#3d8fd4"
	floodRadius  = unit.MaxExtent
	ramSecs      = 8.0

	missBonus2 = 5.0
	missBonus3 = 10.0
	rodRolls   = 10
	rodUnlock  = 3
	ramNeed    = 50.0
	ramCruise  = 380.0
	ramHitCD   = 0.35
	maxBounce  = 3
)

//go:embed fx fish.png
var assets embed.FS

func init() {
	p := unit.NewPack(KindFisher, assets)
	p.Register(unit.Spec{
		Kind:    KindFisher,
		Role:    unit.RoleFighter,
		Radius:  fisherRadius,
		MaxHP:   fisherHP,
		Speed:   fisherCruise,
		Vision:  fisherVision,
		Fighter: true,
		Look:    unit.Look{Color: fisherColor, Glow: true, FX: []string{"fisher"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &钓鱼佬{}
	})
	p.Register(unit.Spec{
		Kind:    KindPond,
		Role:    unit.RoleHelper,
		Radius:  pondRadius,
		MaxHP:   1,
		Speed:   0,
		Vision:  pondRadius + 40,
		Fighter: false,
		Look:    unit.Look{Color: pondColor, Glow: true, Overlay: true, FX: []string{"fisher-pond"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return 鱼塘{}
	})
	p.Register(unit.Spec{
		Kind:    KindFlood,
		Role:    unit.RoleHelper,
		Radius:  floodRadius,
		MaxHP:   1,
		Speed:   0,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: pondColor, FX: []string{"fisher-flood"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return 鱼塘{}
	})
	registerFish(p, KindFish, 10)
	registerFish(p, KindFishMid, 18)
	registerFish(p, KindFishBig, 28)
}

func registerFish(p *unit.Pack, kind string, r float64) {
	p.Register(unit.Spec{
		Kind:    kind,
		Role:    unit.RoleProjectile,
		Radius:  r,
		MaxHP:   1,
		Speed:   220,
		Vision:  9999,
		Fighter: false,
		Look:    unit.Look{Color: "#8ec8e8", Overlay: true, FX: []string{"fisher-fish"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return newFish(info)
	})
}

type 钓鱼佬 struct {
	rng      *rand.Rand
	draw     func() float64
	booted   bool
	fishing  bool
	until    float64
	pondID   uint64
	misses   int
	rod      bool
	inside   bool
	ram      bool
	ramY     float64
	ramUntil float64
	x, y     float64
	slot     int
	ramHitAt float64
	holdVX   float64
	holdVY   float64
	vx, vy   float64
	held     []float64
}

func (a *钓鱼佬) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		a.onHit(ctx, e)
	case unit.Collision:
		a.onBump(ctx, e)
	case unit.WallHit:
	case unit.Sense:
		a.onSense(ctx, e)
	}
}

func (a *钓鱼佬) roll() float64 {
	if a.draw != nil {
		return a.draw()
	}
	a.ensureRNG(0)
	return a.rng.Float64()
}

func (a *钓鱼佬) ensureRNG(id uint64) {
	if a.rng != nil {
		return
	}
	a.rng = rand.New(rand.NewPCG(rand.Uint64()^id, rand.Uint64()))
}

func (a *钓鱼佬) onHit(ctx unit.Context, d unit.IncomingDamage) {
	amt := d.Amount
	if a.fishing {
		amt *= fisherDR
	}
	ctx.Out <- unit.ConfirmDamage{Token: d.Token, UnitID: ctx.ID, Amount: amt}
}

func (a *钓鱼佬) onBump(ctx unit.Context, c unit.Collision) {
	if !a.ram {
		return
	}
	if !unit.EnemyTarget(c, a.slot) {
		return
	}
	if c.Time+1e-9 < a.ramHitAt {
		return
	}
	a.ramHitAt = c.Time + ramHitCD
	ctx.Out <- unit.Damage{From: ctx.ID, To: c.Other.ID, Amount: ramDamage(a.ramY)}
}

func (a *钓鱼佬) onSense(ctx unit.Context, s unit.Sense) {
	a.x, a.y = s.Self.X, s.Self.Y
	a.vx, a.vy = s.Self.VX, s.Self.VY
	a.slot = s.Self.Slot
	a.ensureRNG(ctx.ID)
	if !a.booted {
		a.booted = true
		a.plantPonds(ctx)
	}
	if a.ram && a.ramUntil > 0 && s.Time+1e-9 >= a.ramUntil {
		a.endRam(ctx)
	}
	a.emitHUD(ctx, s)

	if a.fishing {
		if a.flyFish() {
			if !a.ram {
				ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: a.walkSpeed()}
			}
		} else {
			ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: 0}
			ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
		}
		if s.Time+1e-9 >= a.until {
			a.finish(ctx, s)
		}
		return
	}
	if !a.ram {
		ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: a.walkSpeed()}
	}

	pond := overlappingPond(s)
	if a.flyFish() {
		if pond != nil && !a.ram {
			a.startFish(ctx, s, *pond)
		}
		a.inside = pond != nil
		return
	}
	if pond != nil {
		if !a.inside && !a.ram {
			a.startFish(ctx, s, *pond)
		}
		a.inside = true
		return
	}
	a.inside = false
}

func (a *钓鱼佬) startFish(ctx unit.Context, s unit.Sense, pond unit.Snapshot) {
	a.fishing = true
	a.until = s.Time + fishSec
	a.pondID = pond.ID
	a.holdVX, a.holdVY = s.Self.VX, s.Self.VY
	if !a.flyFish() {
		ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: 0}
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	}
	ctx.Out <- unit.FX{
		Name: "cast", Kind: ctx.Kind, UnitID: ctx.ID,
		X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot,
	}
}

func (a *钓鱼佬) finish(ctx unit.Context, s unit.Sense) {
	stood := !a.flyFish()
	a.fishing = false
	n := 1
	if a.rod || a.misses >= rodUnlock {
		n = rodRolls
	}
	bonus := catchBonus(a.misses, a.rod)
	got := a.rolls(n, bonus)
	enemy := enemyOf(s)
	if len(got) == 0 {
		a.misses++
		if a.misses >= rodUnlock && !a.rod {
			a.rod = true
			got = a.rolls(rodRolls, missBonus3)
		}
	}
	if len(got) == 0 {
		if a.misses == 2 {
			ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: KindPond}
			ctx.Out <- unit.Spawn{
				Kind: KindFlood, X: 0, Y: 0,
				OwnerID: ctx.ID, Slot: a.slot,
			}
		} else if a.misses < 2 && a.pondID != 0 {
			ctx.Out <- unit.Despawn{UnitID: a.pondID}
		}
		ctx.Out <- unit.FX{
			Name: "miss", Kind: ctx.Kind, UnitID: ctx.ID,
			X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot, Amount: float64(a.misses),
		}
	} else {
		a.reel(ctx, s, enemy, got)
	}
	a.pondID = 0
	if !a.ram {
		if !stood {
			ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: a.walkSpeed()}
		} else {
			a.setWalk(ctx, a.holdVX, a.holdVY)
		}
	}
}

func (a *钓鱼佬) rolls(n int, bonus float64) []float64 {
	var got []float64
	for i := 0; i < n; i++ {
		y := catchWeight(a.roll(), bonus)
		if y > 0 {
			got = append(got, y)
		}
	}
	return got
}

func (a *钓鱼佬) reel(ctx unit.Context, s unit.Sense, enemy *unit.Snapshot, got []float64) {
	best := 0.0
	var live []float64
	for _, y := range got {
		if y > best {
			best = y
		}
		if y >= ramNeed {
			a.held = append(a.held, y)
			continue
		}
		live = append(live, y)
	}
	for i, y := range live {
		a.toss(ctx, s, enemy, y, i, len(live), false)
	}
	if len(a.held) > 0 && !a.ram {
		a.beginRam(ctx, s, best)
	}
	ctx.Out <- unit.FX{
		Name: "reel", Kind: ctx.Kind, UnitID: ctx.ID,
		X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot, Amount: best,
	}
}

func (a *钓鱼佬) toss(ctx unit.Context, s unit.Sense, enemy *unit.Snapshot, y float64, i, n int, spent bool) {
	ux, uy := 1.0, 0.0
	if enemy != nil {
		dx, dy := enemy.X-s.Self.X, enemy.Y-s.Self.Y
		d := math.Hypot(dx, dy)
		if d > 1e-6 {
			ux, uy = dx/d, dy/d
		}
	} else if m := math.Hypot(a.holdVX, a.holdVY); m > 8 {
		ux, uy = a.holdVX/m, a.holdVY/m
	}
	a.launch(ctx, s.Self.X, s.Self.Y, ux, uy, y, i, n, spent)
}

func (a *钓鱼佬) launch(ctx unit.Context, x, y, ux, uy, weight float64, i, n int, spent bool) {
	kind, r := fishKind(weight)
	sp := fishSpeed(weight)
	if spent {
		sp *= 0.45
	}
	pushFish(fishJob{y: weight, dmg: fishDamage(weight), speed: sp, spent: spent})
	if n > 1 {
		span := 1.2
		off := -span/2 + span*float64(i)/float64(n-1)
		cs, sn := math.Cos(off), math.Sin(off)
		ux, uy = ux*cs-uy*sn, ux*sn+uy*cs
	}
	gap := fisherRadius + r + 2
	ctx.Out <- unit.Spawn{
		Kind: kind, OwnerID: ctx.ID, Slot: a.slot,
		X: x + ux*gap, Y: y + uy*gap,
		VX: ux * sp, VY: uy * sp,
	}
}

func (a *钓鱼佬) beginRam(ctx unit.Context, s unit.Sense, y float64) {
	a.ram = true
	a.ramY = y
	a.ramUntil = s.Time + ramSecs
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: ramCruise}
	ux, uy := 1.0, 0.0
	if n := math.Hypot(s.Self.VX, s.Self.VY); n > 8 {
		ux, uy = s.Self.VX/n, s.Self.VY/n
	} else if e := enemyOf(s); e != nil {
		dx, dy := e.X-s.Self.X, e.Y-s.Self.Y
		d := math.Hypot(dx, dy)
		if d > 1e-6 {
			ux, uy = dx/d, dy/d
		}
	} else if n := math.Hypot(a.holdVX, a.holdVY); n > 8 {
		ux, uy = a.holdVX/n, a.holdVY/n
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: ux * ramCruise, VY: uy * ramCruise}
	ctx.Out <- unit.FX{
		Name: "ram", Kind: ctx.Kind, UnitID: ctx.ID,
		X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot, Amount: y,
	}
}

func (a *钓鱼佬) endRam(ctx unit.Context) {
	held := a.held
	a.held = nil
	a.ram = false
	a.ramY = 0
	sp := a.walkSpeed()
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: sp}
	ux, uy := 1.0, 0.0
	n := math.Hypot(a.vx, a.vy)
	if n > 1e-6 {
		ux, uy = a.vx/n, a.vy/n
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: ux * sp, VY: uy * sp}
	}
	for i, y := range held {
		a.launch(ctx, a.x, a.y, ux, uy, y, i, len(held), true)
	}
}

func (a *钓鱼佬) walkSpeed() float64 {
	return fisherCruise * (1 + 0.5*float64(a.misses))
}

func (a *钓鱼佬) flyFish() bool {
	return a.misses >= 2
}

func (a *钓鱼佬) setWalk(ctx unit.Context, vx, vy float64) {
	sp := a.walkSpeed()
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: sp}
	n := math.Hypot(vx, vy)
	if n > 8 {
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: vx / n * sp, VY: vy / n * sp}
	}
}

func (a *钓鱼佬) plantPonds(ctx unit.Context) {
	used := make([][2]float64, 0, pondN)
	for i := 0; i < pondN; i++ {
		x, y := a.placePond(used)
		used = append(used, [2]float64{x, y})
		ctx.Out <- unit.Spawn{
			Kind: KindPond, X: x, Y: y,
			OwnerID: ctx.ID, Slot: a.slot,
		}
	}
}

func (a *钓鱼佬) placePond(used [][2]float64) (float64, float64) {
	minGap := pondRadius*2 + 24
	f := unit.LiveField()
	for n := 0; n < 40; n++ {
		x, y, ok := f.RandomWalkable(a.rng, pondRadius)
		if !ok {
			break
		}
		good := true
		for _, p := range used {
			if math.Hypot(x-p[0], y-p[1]) < minGap {
				good = false
				break
			}
		}
		if good {
			return x, y
		}
	}
	x, y, ok := f.RandomWalkable(a.rng, pondRadius)
	if ok {
		return x, y
	}
	return f.Clamp(0, 80, pondRadius)
}

func (a *钓鱼佬) emitHUD(ctx unit.Context, s unit.Sense) {
	prog := 0.0
	if a.fishing && fishSec > 0 {
		left := a.until - s.Time
		if left < 0 {
			left = 0
		}
		prog = 1 - left/fishSec
	}
	rod := 0.0
	if a.rod {
		rod = 1
	}
	ram := 0.0
	if a.ram {
		ram = 1
	}
	fish := 0.0
	if a.fishing {
		fish = 1
	}
	ctx.Out <- unit.FX{
		Name: "hud", Kind: ctx.Kind, UnitID: ctx.ID, Slot: s.Self.Slot,
		Amount: float64(a.misses), VX: prog, VY: rod + fish*2 + ram*4,
	}
	kg := 0.0
	if a.ram {
		kg = a.ramY
	}
	ctx.Out <- unit.FX{
		Name: "carry", Kind: ctx.Kind, UnitID: ctx.ID, Slot: s.Self.Slot,
		Amount: kg,
	}
}

func overlappingPond(s unit.Sense) *unit.Snapshot {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind != KindPond && o.Kind != KindFlood {
			continue
		}
		if o.OwnerID != s.Self.ID {
			continue
		}
		if math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) <= o.Radius+s.Self.Radius {
			return o
		}
	}
	return nil
}

func enemyOf(s unit.Sense) *unit.Snapshot {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role == unit.RoleFighter && o.Slot != s.Self.Slot {
			return o
		}
	}
	return nil
}

func catchWeight(x, bonus float64) float64 {
	if x < 0 {
		x = 0
	}
	if x > 1 {
		x = 1
	}
	return 100*(1-math.Sqrt(1-x)) - 10 + bonus
}

func catchBonus(misses int, rod bool) float64 {
	if rod || misses >= rodUnlock {
		return missBonus3
	}
	if misses >= 2 {
		return missBonus2
	}
	return 0
}

func fishDamage(y float64) float64 {
	d := y * 0.5
	if d < 1 {
		return 1
	}
	if d > 26 {
		return 26
	}
	return d
}

func fishSpeed(y float64) float64 {
	sp := 160 + y*1.6
	kind, _ := fishKind(y)
	if kind != KindFishBig {
		return sp * 2
	}
	return sp
}

func ramDamage(y float64) float64 {
	d := y * 0.5
	if d < 4 {
		return 4
	}
	if d > 22 {
		return 22
	}
	return d
}

func fishKind(y float64) (string, float64) {
	switch {
	case y < 22:
		return KindFish, 10
	case y < 48:
		return KindFishMid, 18
	default:
		return KindFishBig, 28
	}
}
