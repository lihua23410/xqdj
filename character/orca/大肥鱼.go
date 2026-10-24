// 大肥鱼在场上碰到随机出现的白饭就变成虎鲸。
// 虎鲸潜水，再跃到跃起那一瞬敌人所在的位置，入水时溅起水花。
package 大肥鱼

import (
	"embed"
	"math"
	"math/rand/v2"
	"xqdj/internal/unit"
)

const KindOrca = "大肥鱼"
const KindRice = "大肥鱼白饭"

const (
	maidRadius    = 18.0
	maidHP        = 100.0
	maidCruise    = 130.0
	maidVision    = 0.0
	orcaRadius    = 34.0
	orcaVision    = 9999.0
	surfaceCruise = 160.0
	diveCruise    = 40.0
	formSec       = 6.0
	diveSec       = 0.45
	leapSec       = 0.5
	surfaceSec    = 1.1
	leapSpeed     = 640.0
	splashRadius  = 56.0 * 1.3
	splashDamage  = 12.0 * 0.75
	leapAhead     = 140.0
	bowlRadius    = 14.0
	bowlClear     = 16.0
	bowlCap       = 5
	riceEvery     = 3.0
	orcaDR        = 0.5
	riceKinds     = 7
	placeTries    = 40
	maidColor     = "#243456"
	riceColor     = "#f3efe4"

	phaseDive    = 1
	phaseLeap    = 2
	phaseSurface = 3
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindOrca, assets)
	p.Register(unit.Spec{
		Kind:    KindOrca,
		Role:    unit.RoleFighter,
		Radius:  maidRadius,
		MaxHP:   maidHP,
		Speed:   maidCruise,
		Vision:  maidVision,
		Fighter: true,
		Look:    unit.Look{Color: maidColor, FX: []string{"orca"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &大肥鱼{}
	})
	p.Register(unit.Spec{
		Kind:     KindRice,
		Role:     unit.RoleHelper,
		Radius:   bowlRadius,
		MaxHP:    1,
		Speed:    0,
		Vision:   0,
		Nonsolid: true,
		Look:     unit.Look{Color: riceColor, FX: []string{"orca-rice"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &白饭{}
	})
}

type 大肥鱼 struct {
	rng          *rand.Rand
	seen         bool
	hp           float64
	form         bool
	until        float64
	phase        int
	phaseUntil   float64
	aimX, aimY   float64
	fromX, fromY float64
	haveAim      bool
	nextRice     float64
}

func (a *大肥鱼) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		a.onHit(ctx, e)
	case unit.Sense:
		a.onSense(ctx, e)
	}
}

func (a *大肥鱼) onHit(ctx unit.Context, d unit.IncomingDamage) {
	hp := a.hp
	if !a.seen {
		hp = maidHP
	}
	amt := d.Amount
	if a.form {
		amt *= orcaDR
	}
	if amt+1e-9 >= hp {
		ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID}
		a.form = false
		a.hp = 0
		ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: false}
	} else if a.seen {
		a.hp -= amt
	}
	ctx.Out <- unit.ConfirmDamage{Token: d.Token, UnitID: ctx.ID, Amount: amt}
}

func (a *大肥鱼) onSense(ctx unit.Context, s unit.Sense) {
	a.ensure(ctx.Stream)
	a.seen = true
	a.hp = s.Self.HP
	if a.form && s.Time+1e-9 >= a.until {
		a.leave(ctx)
	}
	a.stock(ctx, s)
	if a.form {
		a.whale(ctx, s)
	} else {
		a.maid(ctx, s)
	}
	a.paint(ctx, s)
}

func (a *大肥鱼) ensure(stream uint64) {
	if a.rng != nil {
		return
	}
	a.rng = unit.NewRNG(stream)
}

func (a *大肥鱼) maid(ctx unit.Context, s unit.Sense) {
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: maidCruise}
	if bowl := overlappingBowl(s); bowl != nil {
		a.eat(ctx, s, bowl)
	}
}

func (a *大肥鱼) stock(ctx unit.Context, s unit.Sense) {
	n := bowlCount(s)
	if n >= bowlCap {
		if s.Time+1e-9 >= a.nextRice {
			a.nextRice = s.Time + riceEvery
		}
		return
	}
	if s.Time+1e-9 < a.nextRice {
		return
	}
	x, y, ok := a.place(s)
	if !ok {
		return
	}
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: KindRice, X: x, Y: y,
		OwnerID: ctx.ID, Slot: s.Self.Slot,
	})
	a.nextRice = s.Time + riceEvery
}

func (a *大肥鱼) place(s unit.Sense) (float64, float64, bool) {
	if a.rng == nil {
		return 0, 0, false
	}
	for i := 0; i < placeTries; i++ {
		x, y, ok := s.Field.RandomWalkable(a.rng, bowlRadius)
		if !ok {
			return 0, 0, false
		}
		if math.Hypot(x-s.Self.X, y-s.Self.Y) < s.Self.Radius+bowlRadius+bowlClear {
			continue
		}
		return x, y, true
	}
	return 0, 0, false
}

func (a *大肥鱼) eat(ctx unit.Context, s unit.Sense, bowl *unit.Snapshot) {
	ctx.Out <- unit.Despawn{UnitID: bowl.ID}
	a.form = true
	a.until = s.Time + formSec
	a.phase = phaseDive
	a.phaseUntil = s.Time + diveSec
	a.haveAim = false
	ctx.Out <- unit.SetRadius{UnitID: ctx.ID, Radius: orcaRadius}
	ctx.Out <- unit.SetVision{UnitID: ctx.ID, Vision: orcaVision}
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: diveCruise}
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
	if x, y, moved := fit(s.Field, s.Self.X, s.Self.Y, orcaRadius); moved {
		ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: x, Y: y}
	}
}

func (a *大肥鱼) leave(ctx unit.Context) {
	a.form = false
	a.phase = 0
	a.haveAim = false
	ctx.Out <- unit.SetRadius{UnitID: ctx.ID, Radius: maidRadius}
	ctx.Out <- unit.SetVision{UnitID: ctx.ID, Vision: maidVision}
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: maidCruise}
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: false}
}

func (a *大肥鱼) whale(ctx unit.Context, s unit.Sense) {
	switch a.phase {
	case phaseDive:
		a.dive(ctx, s)
	case phaseLeap:
		a.leap(ctx, s)
	default:
		a.surface(ctx, s)
	}
}

func (a *大肥鱼) dive(ctx unit.Context, s unit.Sense) {
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: diveCruise}
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
	if s.Time+1e-9 < a.phaseUntil {
		return
	}
	a.fromX, a.fromY = s.Self.X, s.Self.Y
	a.aimX, a.aimY = a.mark(s)
	a.haveAim = true
	a.phase = phaseLeap
	a.phaseUntil = s.Time + leapSec
	ctx.Out <- unit.FX{
		Name: "leap", Kind: ctx.Kind, UnitID: ctx.ID, Slot: s.Self.Slot,
		X: s.Self.X, Y: s.Self.Y, VX: a.aimX, VY: a.aimY,
	}
}

func (a *大肥鱼) leap(ctx unit.Context, s unit.Sense) {
	if !a.haveAim {
		a.land(ctx, s)
		return
	}
	dx, dy := a.aimX-s.Self.X, a.aimY-s.Self.Y
	if math.Hypot(dx, dy) <= orcaRadius || s.Time+1e-9 >= a.phaseUntil {
		a.land(ctx, s)
		return
	}
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: leapSpeed}
	a.steer(ctx, dx, dy, leapSpeed)
}

func (a *大肥鱼) land(ctx unit.Context, s unit.Sense) {
	x, y := a.aimX, a.aimY
	if !a.haveAim {
		x, y = s.Self.X, s.Self.Y
	}
	x, y = s.Field.Clamp(x, y, orcaRadius)
	ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: x, Y: y}
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: false}
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: surfaceCruise}
	a.steer(ctx, x-a.fromX, y-a.fromY, surfaceCruise)
	a.splash(ctx, s, x, y)
	a.phase = phaseSurface
	a.phaseUntil = s.Time + surfaceSec
	a.haveAim = false
}

func (a *大肥鱼) surface(ctx unit.Context, s unit.Sense) {
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: surfaceCruise}
	if s.Time+1e-9 < a.phaseUntil {
		return
	}
	if a.until-s.Time <= diveSec+leapSec {
		return
	}
	a.phase = phaseDive
	a.phaseUntil = s.Time + diveSec
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: diveCruise}
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
}

func (a *大肥鱼) mark(s unit.Sense) (float64, float64) {
	if e := nearestEnemy(s); e != nil {
		return s.Field.Clamp(e.X, e.Y, orcaRadius)
	}
	vx, vy := unitVec(s.Self.VX, s.Self.VY, leapAhead)
	return s.Field.Clamp(s.Self.X+vx, s.Self.Y+vy, orcaRadius)
}

func (a *大肥鱼) splash(ctx unit.Context, s unit.Sense, x, y float64) {
	ctx.Out <- unit.FX{
		Name: "splash", Kind: ctx.Kind, UnitID: ctx.ID, Slot: s.Self.Slot,
		X: x, Y: y, Amount: splashRadius,
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.ID == s.Self.ID || o.HP <= 0 || !enemyBody(s.Self.Slot, *o) {
			continue
		}
		if math.Hypot(o.X-x, o.Y-y) > splashRadius {
			continue
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: splashDamage}
	}
}

func (a *大肥鱼) steer(ctx unit.Context, dx, dy, speed float64) {
	vx, vy := unitVec(dx, dy, speed)
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: vx, VY: vy}
}

func (a *大肥鱼) paint(ctx unit.Context, s unit.Sense) {
	amt := 0.0
	if a.form {
		switch a.phase {
		case phaseDive:
			amt = 2
		case phaseLeap:
			amt = 3
		default:
			amt = 1
		}
	}
	ctx.Out <- unit.FX{
		Name: "form", Kind: ctx.Kind, UnitID: ctx.ID, Slot: s.Self.Slot,
		X: s.Self.X, Y: s.Self.Y, Amount: amt,
	}
}

func bowlCount(s unit.Sense) int {
	n := 0
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID == s.Self.ID && o.Kind == KindRice {
			n++
		}
	}
	return n
}

func overlappingBowl(s unit.Sense) *unit.Snapshot {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID != s.Self.ID || o.Kind != KindRice {
			continue
		}
		if math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) <= s.Self.Radius+o.Radius {
			return o
		}
	}
	return nil
}

func nearestEnemy(s unit.Sense) *unit.Snapshot {
	var best *unit.Snapshot
	bestD := math.MaxFloat64
	for pass := 0; pass < 2; pass++ {
		for i := range s.Nearby {
			o := &s.Nearby[i]
			fighter := o.Role == unit.RoleFighter
			if pass == 0 && !fighter {
				continue
			}
			if pass == 1 && (fighter || !o.Mortal) {
				continue
			}
			if !enemyBody(s.Self.Slot, *o) || o.HP <= 0 {
				continue
			}
			d := math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y)
			if d < bestD {
				bestD = d
				best = o
			}
		}
		if best != nil {
			return best
		}
	}
	return nil
}

func enemyBody(slot int, o unit.Snapshot) bool {
	if o.Slot == slot {
		return false
	}
	return o.Role == unit.RoleFighter || o.Mortal
}

func unitVec(dx, dy, speed float64) (float64, float64) {
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return speed, 0
	}
	return dx / n * speed, dy / n * speed
}

func fit(f unit.Field, x, y, radius float64) (float64, float64, bool) {
	nx, ny := f.Clamp(x, y, radius)
	if math.Hypot(nx-x, ny-y) < 0.5 {
		return x, y, false
	}
	return nx, ny, true
}

type 白饭 struct {
	skin   int
	rolled bool
}

func (b *白饭) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !b.rolled {
		b.rolled = true
		b.skin = unit.NewRNG(ctx.Stream).IntN(riceKinds)
	}
	ctx.Out <- unit.FX{
		Name: "rice", Kind: ctx.Kind, UnitID: ctx.ID, Slot: s.Self.Slot,
		X: s.Self.X, Y: s.Self.Y, Amount: float64(b.skin),
	}
}
