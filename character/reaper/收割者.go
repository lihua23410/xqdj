// 收割者不转向。撞场边钉镰刀，六条场边都有至少一把才收回。
package 收割者

import (
	"embed"
	"math"
	"sync"
	"xqdj/internal/unit"
)

const KindReaper = "收割者"
const KindSickle = "收割者镰刀"
const KindReap = "收割者收回镰刀"

const (
	reaperRadius = 18.0
	reaperHP     = 100.0
	reaperCruise = 175.0
	reaperVision = 0.0
	reaperColor  = "#5a2a4a"

	sickleRadius = 18.0
	reapRadius   = 36.0
	sickleDamage = 6.0
	sickleHitCD  = 0.4
	sickleAccel  = 40.0
	sickleColor  = "#c43d6e"
	sickleVision = 9999.0
	sickleHeal   = 3.0
	harvestCarry = 1.0
)

// recallingIDs：主人判定收回时记下当时场上镰刀的 id。新钉的刀 id 不在里面，途中仍能钉。
var recallingIDs sync.Map

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindReaper, assets)
	p.Register(unit.Spec{
		Kind:    KindReaper,
		Role:    unit.RoleFighter,
		Radius:  reaperRadius,
		MaxHP:   reaperHP,
		Speed:   reaperCruise,
		Vision:  reaperVision,
		Fighter: true,
		Look:    unit.Look{Color: reaperColor, FX: []string{"reaper"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &收割者{}
	})
	p.Register(unit.Spec{
		Kind:      KindSickle,
		Role:      unit.RoleProjectile,
		Radius:    sickleRadius,
		MaxHP:     1,
		Speed:     0,
		Vision:    sickleVision,
		Fighter:   false,
		PassWalls: true,
		Look:      unit.Look{Color: sickleColor, Overlay: true, FX: []string{"sickle"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &镰刀{owner: info.OwnerID, slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:      KindReap,
		Role:      unit.RoleProjectile,
		Radius:    reapRadius,
		MaxHP:     1,
		Speed:     0,
		Vision:    sickleVision,
		Fighter:   false,
		PassWalls: true,
		Look:      unit.Look{Color: sickleColor, Overlay: true, FX: []string{"sickle"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &镰刀{owner: info.OwnerID, slot: info.Slot, recalling: true}
	})
}

type 收割者 struct {
	sides [6]int
	x, y  float64
	slot  int
}

func (r *收割者) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		r.x, r.y = e.Self.X, e.Self.Y
		r.slot = e.Self.Slot
		r.maybeReap(ctx, e)
	case unit.WallHit:
		r.onWall(ctx, e)
	}
}

func (r *收割者) onWall(ctx unit.Context, w unit.WallHit) {
	side, ok := hexHit(r.x, r.y, w.NX, w.NY, reaperRadius)
	if !ok {
		return
	}
	nx, ny := hexNormal(side)
	ctx.Out <- unit.Spawn{
		Kind:    KindSickle,
		X:       r.x + nx*reaperRadius,
		Y:       r.y + ny*reaperRadius,
		OwnerID: ctx.ID,
		Slot:    r.slot,
	}
	r.sides[side]++
}

func (r *收割者) maybeReap(ctx unit.Context, s unit.Sense) {
	for _, n := range r.sides {
		if n <= 0 {
			return
		}
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind != KindSickle || o.OwnerID != ctx.ID {
			continue
		}
		recallingIDs.Store(o.ID, struct{}{})
	}
	r.sides = [6]int{}
}

type 镰刀 struct {
	owner      uint64
	slot       int
	recalling  bool
	drew       bool
	speed      float64
	lastT      float64
	hitReadyAt float64
	booted     bool
}

func (k *镰刀) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		k.onSense(ctx, e)
	}
}

func (k *镰刀) onSense(ctx unit.Context, s unit.Sense) {
	if !k.recalling {
		if _, ok := recallingIDs.LoadAndDelete(ctx.ID); ok {
			vx := 0.0
			if k.drew {
				vx = harvestCarry
			}
			ctx.Out <- unit.Spawn{
				Kind:    KindReap,
				X:       s.Self.X,
				Y:       s.Self.Y,
				VX:      vx,
				OwnerID: k.owner,
				Slot:    k.slot,
			}
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			return
		}
	}
	if !k.booted {
		k.booted = true
		k.lastT = s.Time
		if k.recalling && math.Abs(s.Self.VX) >= harvestCarry*0.5 {
			k.drew = true
		}
		ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
	}
	dt := s.Time - k.lastT
	if dt < 0 {
		dt = 0
	}
	k.lastT = s.Time
	owner := ownerOf(s, k.owner)
	if k.recalling {
		k.chase(ctx, s, owner, dt)
		if owner != nil && overlap(s.Self.X, s.Self.Y, s.Self.Radius, owner.X, owner.Y, owner.Radius) {
			k.showBlood(ctx)
			if k.drew {
				ctx.Out <- unit.Heal{UnitID: k.owner, Amount: sickleHeal}
			}
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			return
		}
	} else {
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	}
	k.cut(ctx, s)
	k.showBlood(ctx)
}

func (k *镰刀) showBlood(ctx unit.Context) {
	amt := 0.0
	if k.drew {
		amt = 1
	}
	ctx.Out <- unit.FX{Name: "blood", Kind: ctx.Kind, UnitID: ctx.ID, Amount: amt}
}

func (k *镰刀) chase(ctx unit.Context, s unit.Sense, owner *unit.Snapshot, dt float64) {
	k.speed += sickleAccel * dt
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: k.speed + 40}
	if owner == nil || k.speed <= 0 {
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
		return
	}
	dx, dy := owner.X-s.Self.X, owner.Y-s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
		return
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: dx / n * k.speed, VY: dy / n * k.speed}
}

func (k *镰刀) cut(ctx unit.Context, s unit.Sense) {
	if s.Time+1e-9 < k.hitReadyAt {
		return
	}
	r := s.Self.Radius
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role != unit.RoleFighter || o.Slot == k.slot {
			continue
		}
		if !overlap(s.Self.X, s.Self.Y, r, o.X, o.Y, o.Radius) {
			continue
		}
		k.hitReadyAt = s.Time + sickleHitCD
		k.drew = true
		ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: sickleDamage}
		return
	}
}

func ownerOf(s unit.Sense, id uint64) *unit.Snapshot {
	if s.Self.ID == id {
		return &s.Self
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.ID == id {
			return o
		}
	}
	return nil
}

func overlap(ax, ay, ar, bx, by, br float64) bool {
	return math.Hypot(ax-bx, ay-by) <= ar+br+1e-9
}

func hexNormal(i int) (float64, float64) {
	a := (float64(i) + 0.5) * math.Pi / 3
	return math.Cos(a), math.Sin(a)
}

// hexHit：只有贴在六边形场边上的撞墙才算。砌在场上的墙法线对不上或人还在场内深处。
func hexHit(x, y, nx, ny, radius float64) (int, bool) {
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
