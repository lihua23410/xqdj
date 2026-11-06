// Package ram owns locked charges with swept, once-per-victim impact damage.
package ram

import (
	"xqdj/character/engine_heart/parts/api"
	"xqdj/internal/unit"
)

func Definition() api.Definition {
	return api.Definition{ID: 12, Key: "ram", Name: "破阵撞角", Install: func(*api.Build) {}, Weapon: func() unit.Actor { return &weapon{} }, Register: func(p *unit.Pack) {
		p.RegisterAttackFan(unit.Spec{Kind: KindContact, Role: unit.RoleProjectile, Attach: true, Pass: true, RequiresOwner: true, Radius: 23, ArcInner: 18, ArcSpan: unit.Deg(120), Look: unit.Look{FX: []string{"pump-attack-contact"}}})
	}}
}

const KindContact = "泵之心撞角判定"

type weapon struct {
	charge int
	until  float64
	last   unit.Sense
}

func token(id uint64) uint64 { return unit.DeriveStream(id, 12, 311) }
func (w *weapon) stop(ctx unit.Context) {
	w.until = 0
	ctx.Out <- unit.RemoveFS{UnitID: ctx.ID, Token: token(ctx.ID)}
	unit.SyncAttackFan(ctx, w.last, KindContact, unit.FanAttack{})
}
func (w *weapon) hit(ctx unit.Context, o unit.Snapshot, now float64) {
	if !unit.Hittable(o, w.last.Self.Slot) {
		return
	}
	dx, dy := api.Direction(o.X-w.last.Self.X, o.Y-w.last.Self.Y)
	ctx.Out <- unit.AddFS{UnitID: o.ID, DX: dx, DY: dy, BaseSpeed: 420, OnWall: true, ExpiresAt: now + .18, Token: unit.DeriveStream(ctx.ID, o.ID, 312)}
}
func (w *weapon) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case api.Pulse:
		if e.Echo {
			return
		}
		w.charge = min(3, w.charge+1)
		if w.charge < 3 {
			return
		}
		if e.SeekingPart {
			return // Keep the charge for the next contraction without overriding pickup movement.
		}
		w.charge = 0
		s := e.Sense
		t := api.Target(s, 240)
		if t == nil || s.Knock {
			return
		}
		dx, dy := api.Direction(t.X-s.Self.X, t.Y-s.Self.Y)
		w.until = s.Time + .26
		w.last = s
		w.last.Self.VX, w.last.Self.VY = dx*150, dy*150
		ctx.Out <- unit.RemoveFS{UnitID: ctx.ID, Token: api.CoreDashToken(ctx.ID)}
		ctx.Out <- unit.SetFSDirection{UnitID: ctx.ID, VX: dx, VY: dy}
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: dx * 150, VY: dy * 150}
		ctx.Out <- unit.AddFS{UnitID: ctx.ID, DX: dx, DY: dy, BaseSpeed: 300, OnWall: true, ExpiresAt: w.until, Token: token(ctx.ID)}
		d := api.Ray(s, s.Self.X, s.Self.Y, dx, dy, 110)
		api.Flash(ctx, s, "heart-ram", s.Self.X, s.Self.Y, s.Self.X+dx*d, s.Self.Y+dy*d, 1)
		w.sync(ctx)
	case unit.Sense:
		w.last = e
		if w.until == 0 {
			return
		}
		if e.Time > w.until {
			w.stop(ctx)
			return
		}
		w.sync(ctx)
	case unit.FanHit:
		if e.Kind == KindContact && w.until > 0 && e.Time <= w.until {
			w.hit(ctx, e.Other, e.Time)
		}
	case unit.WallHit, unit.Stunned, api.Detached:
		w.stop(ctx)
	}
}

func (w *weapon) sync(ctx unit.Context) {
	unit.SyncAttackFan(ctx, w.last, KindContact, unit.FanAttack{Active: w.until > 0, Epoch: w.until, Until: w.until, Damage: 8, PerTarget: true, NoFreeze: true})
}
