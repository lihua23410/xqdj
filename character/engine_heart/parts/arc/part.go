// Package arc owns proximity charging and collision-triggered electrical retaliation.
package arc

import (
	"math"
	"xqdj/character/engine_heart/parts/api"
	"xqdj/internal/unit"
)

func Definition() api.Definition {
	return api.Definition{ID: 14, Key: "arc", Name: "接触电弧线圈", Install: func(*api.Build) {}, Weapon: func() unit.Actor { return &weapon{} }, Register: register}
}

const KindContact = "泵之心接触电弧"
const KindChain = "泵之心电弧传导"

func register(p *unit.Pack) {
	p.RegisterAttackFan(unit.Spec{Kind: KindContact, Role: unit.RoleProjectile, Attach: true, RequiresOwner: true, Radius: 20, ArcInner: 18, ArcSpan: 2 * math.Pi, Look: unit.Look{FX: []string{"pump-attack-contact"}}})
	p.Register(unit.Spec{Kind: KindChain, Role: unit.RoleHelper, Nonsolid: true, RequiresOwner: true, Vision: 9999}, func(info unit.SpawnInfo) unit.Actor {
		first, _ := info.Data.(uint64)
		return &chain{owner: info.OwnerID, first: first}
	})
}

type weapon struct {
	charge int
	last   unit.Sense
	epoch  float64
}

func (w *weapon) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case api.Pulse:
		w.last = e.Sense
		if e.Echo {
			return
		}
		if api.Target(e.Sense, 90) != nil {
			if w.charge == 1 {
				w.epoch++
			}
			w.charge = min(2, w.charge+1)
		}
		w.ensure(ctx)
		w.show(ctx)
	case unit.Sense:
		w.last = e
		w.ensure(ctx)
		w.show(ctx)
	case unit.FanHit:
		if e.Kind != KindContact || w.charge < 2 {
			return
		}
		w.charge = 0
		w.ensure(ctx)
		w.hide(ctx)
		ctx.Out <- unit.Stun{UnitID: e.Other.ID, Hold: true, Until: e.Time + .18}
		api.Flash(ctx, w.last, "heart-arc", w.last.Self.X, w.last.Self.Y, e.Other.X, e.Other.Y, 1)
		ctx.Out <- ctx.Born(unit.Spawn{Kind: KindChain, OwnerID: ctx.ID, Slot: w.last.Self.Slot, X: e.Other.X, Y: e.Other.Y, Data: e.Other.ID})
	case unit.Stunned, api.Detached:
		w.charge = 0
		w.ensure(ctx)
		w.hide(ctx)
	}
}
func (w *weapon) ensure(ctx unit.Context) {
	unit.SyncAttackFan(ctx, w.last, KindContact, unit.FanAttack{Active: w.charge >= 2, Epoch: w.epoch, Once: true, Damage: 6, NoFreeze: true})
}
func (w *weapon) show(ctx unit.Context) {
	if w.charge > 0 {
		api.Flash(ctx, w.last, "heart-arc-charge", w.last.Self.X, w.last.Self.Y, w.last.Self.X, w.last.Self.Y, float64(w.charge)/2)
	}
}
func (w *weapon) hide(ctx unit.Context) {
	ctx.Out <- unit.FX{Name: "heart-arc-off", Kind: ctx.Kind, UnitID: ctx.ID}
}

type chain struct {
	owner, first uint64
	done         bool
}

func (w *chain) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok || w.done {
		return
	}
	w.done = true
	defer func() { ctx.Out <- unit.Despawn{UnitID: ctx.ID} }()
	used := map[uint64]bool{w.first: true}
	x, y := s.Self.X, s.Self.Y
	reach := 105.0
	for hop := 1; hop < 3; hop++ {
		var target *unit.Snapshot
		best := math.MaxFloat64
		for i := range s.Nearby {
			o := &s.Nearby[i]
			d := math.Hypot(o.X-x, o.Y-y)
			if unit.Hittable(*o, s.Self.Slot) && o.HP > 0 && !used[o.ID] && d <= reach+o.Radius && d < best && api.Clear(s, x, y, *o) {
				target = o
				best = d
			}
		}
		if target == nil {
			break
		}
		used[target.ID] = true
		ctx.Out <- unit.Damage{From: ctx.ID, To: target.ID, Amount: float64(4 - hop), NoFreeze: true}
		api.FlashOwner(ctx, w.owner, s, "heart-arc", x, y, target.X, target.Y, float64(3-hop)/3)
		x, y, reach = target.X, target.Y, 105
	}
}
