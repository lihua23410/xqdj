package 分身者

import (
	"math"
	"xqdj/internal/unit"
)

const KindDoppel = "分身者"
const KindClone = "分身"
const KindCloneArc = "分身弧"

const (
	doppelRadius  = 18.0
	doppelSpeed   = 175.0
	doppelHP      = 100.0
	cloneKind     = KindClone
	cloneDamage   = 6.0
	cloneHitCD    = 0.1
	cloneCount    = 3
	cloneGap      = 48.0
	cloneArcInner = doppelRadius
	cloneArcOuter = doppelRadius + 2
	cloneColor    = "#7a9bb8"
)

func init() {
	p := unit.NewPack(KindDoppel, nil)
	p.Register(unit.Spec{
		Kind:    KindDoppel,
		Role:    unit.RoleFighter,
		Radius:  doppelRadius,
		MaxHP:   doppelHP,
		Speed:   doppelSpeed,
		Vision:  0,
		Fighter: true,
		Look:    unit.Look{Color: cloneColor},
	}, func(unit.SpawnInfo) unit.Actor {
		return &分身者{}
	})
	p.Register(unit.Spec{
		Kind:    cloneKind,
		Role:    unit.RoleClone,
		Radius:  doppelRadius,
		MaxHP:   1,
		Speed:   doppelSpeed,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: cloneColor},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &分身{slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:     KindCloneArc,
		Role:     unit.RoleProjectile,
		Radius:   cloneArcOuter,
		MaxHP:    1,
		Speed:    doppelSpeed,
		Vision:   0,
		Fighter:  false,
		Attach:   true,
		ArcSpan:  2 * math.Pi,
		ArcInner: cloneArcInner,
		Look:     unit.Look{Color: cloneColor, Overlay: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &分身弧{slot: info.Slot}
	})
}

type 分身者 struct {
	spawned bool
}

func (d *分身者) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if d.spawned {
		return
	}
	d.spawned = true
	d.unleash(ctx, s)
}

func (d *分身者) unleash(ctx unit.Context, s unit.Sense) {
	sp := math.Hypot(s.Self.VX, s.Self.VY)
	vx, vy := s.Self.VX, s.Self.VY
	if sp < 1e-6 {
		sp = doppelSpeed
		vx, vy = doppelSpeed, 0
	}
	px, py := vx/sp, vy/sp
	qx, qy := -py, px
	for i := 0; i < cloneCount; i++ {
		ang := float64(i) * 2 * math.Pi / float64(cloneCount)
		cs, sn := math.Cos(ang), math.Sin(ang)
		ox, oy := (px*cs+qx*sn)*cloneGap, (py*cs+qy*sn)*cloneGap
		rvx, rvy := rotate2(vx, vy, ang)
		ctx.Out <- unit.Spawn{
			Kind:    cloneKind,
			X:       s.Self.X + ox,
			Y:       s.Self.Y + oy,
			VX:      rvx,
			VY:      rvy,
			OwnerID: ctx.ID,
			Slot:    s.Self.Slot,
		}
	}
}

func rotate2(vx, vy, ang float64) (float64, float64) {
	c, s := math.Cos(ang), math.Sin(ang)
	return vx*c - vy*s, vx*s + vy*c
}

type 分身 struct {
	slot int
	arc  unit.AttachState
}

func (c *分身) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if unit.RearmAttach(s, ctx.ID, KindCloneArc, cloneHitCD, &c.arc) {
		unit.SpawnAttach(ctx, s, KindCloneArc)
	}
}

type 分身弧 struct {
	slot int
}

func (a *分身弧) Handle(ctx unit.Context, ev unit.Event) {
	e, ok := ev.(unit.Collision)
	if !ok || !unit.EnemyFighter(e, a.slot) {
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: cloneDamage}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}
