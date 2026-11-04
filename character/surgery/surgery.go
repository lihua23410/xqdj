// Package 外科手术 owns the operation, wounds and blood of each surgeon in one match.
package 外科手术

import (
	"embed"
	"math"
	"xqdj/internal/unit"
)

const (
	KindSurgery       = "外科手术"
	KindSurgeryField  = "外科手术场地伤口"
	KindScalpel       = "外科手术手术刀"
	KindSutureNeedle  = "外科手术缝合针"
	KindBloodBullet   = "外科手术血液子弹"
	bodyRadius        = 18.0
	cruise            = 170.0
	knifeSpeed        = 1000.0
	knifeWindup       = 0.1
	stitchSpeed       = 360.0
	approachSpeed     = 420.0
	stepTime          = 1.0 / 60
	maxWounds         = 6
	burstCharge       = 200.0
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindSurgery, assets)
	p.Register(unit.Spec{Kind: KindSurgery, Role: unit.RoleFighter, Fighter: true,
		Radius: bodyRadius, MaxHP: 100, Speed: cruise, Vision: 9999,
		Look: unit.Look{Color: "#76b5ab", FX: []string{"surgery"}, Ghost: 280}},
		func(unit.SpawnInfo) unit.Actor { return &surgeon{} })
	p.Register(unit.Spec{Kind: KindSurgeryField, Role: unit.RoleHelper, Nonsolid: true,
		Vision: 9999, MaxHP: 1, RequiresOwner: true},
		func(info unit.SpawnInfo) unit.Actor { return &woundField{owner: info.OwnerID} })
	p.Register(unit.Spec{Kind: KindScalpel, Role: unit.RoleProjectile,
		Radius: 6, MaxHP: 1, Speed: knifeSpeed, Pass: true, BodiesOnly: true,
		PassInnerWalls: true, Vision: 9999, RequiresOwner: true,
		Look: unit.Look{Color: "#d9e7e8", Overlay: true, FX: []string{"surgery-scalpel"}}},
		func(info unit.SpawnInfo) unit.Actor {
			plan, planned := info.Data.(knifePlan)
			return &scalpel{owner: info.OwnerID, slot: info.Slot, data: plan, copied: !planned, hits: map[uint64]bool{}}
		})
	p.Register(unit.Spec{Kind: KindSutureNeedle, Role: unit.RoleProjectile,
		Radius: 29, ArcInner: 18, ArcSpan: unit.Deg(50), Attach: true, Pass: true,
		BodiesOnly: true, MaxHP: 1, RequiresOwner: true,
		Look: unit.Look{Color: "#dce9e8", Overlay: true, FX: []string{"surgery-needle"}}},
		func(info unit.SpawnInfo) unit.Actor {
			wound, _ := info.Data.(uint64)
			return &needle{owner: info.OwnerID, slot: info.Slot, wound: wound}
		})
	p.Register(unit.Spec{Kind: KindBloodBullet, Role: unit.RoleProjectile,
		Radius: 3, MaxHP: 1, Speed: 240, Vision: 0, AttachFriends: true,
		BodiesOnly: true, RequiresOwner: true,
		Look: unit.Look{Color: "#a71931", FX: []string{"surgery-blood"}}},
		func(info unit.SpawnInfo) unit.Actor { return &bloodBullet{} })
}

const (
	idle = iota
	winding
	cutting
	approaching
	stitching
)

type surgeon struct{ booted bool }

func (a *surgeon) Handle(ctx unit.Context, ev unit.Event) {
	world := unit.StateFor[clinic](ctx.State)
	o := world.operation(ctx.ID)
	var cmds commands
	o.Lock()
	switch e := ev.(type) {
	case unit.IncomingDamage:
		if world.isBlood(e.From, e.Time) && o.immune(e.Time) {
			cmds.add(unit.BlockDamage{Token: e.Token, UnitID: ctx.ID})
		} else {
			cmds.add(unit.ConfirmDamage{Token: e.Token, UnitID: ctx.ID, Amount: e.Amount})
		}
	case unit.Stunned:
		o.stunUntil = e.Until
		o.stunned = true
		o.interrupt(ctx.ID, e.Time, &cmds)
	case unit.Sense:
		o.self = e.Self
		if !a.booted {
			a.booted = true
			o.readyAt = e.Time + 1
			cmds.add(unit.Spawn{Kind: KindSurgeryField, OwnerID: ctx.ID, Slot: e.Self.Slot})
		}
		o.onSense(ctx.ID, e, &cmds)
	}
	o.Unlock()
	cmds.send(ctx)
}

func (o *operation) immune(now float64) bool {
	return o.stage >= approaching && !o.knocked && !o.stunned && now+1e-9 >= o.stunUntil
}

func (o *operation) stopNeedle(id uint64, cmds *commands) {
	if o.needleOn {
		cmds.add(unit.DespawnOwned{OwnerID: id, Kind: KindSutureNeedle})
		o.needleOn = false
	}
}

func (o *operation) interrupt(id uint64, now float64, cmds *commands) {
	o.stopNeedle(id, cmds)
	if o.stage >= approaching {
		o.stage = approaching
		o.route = nil
	}
	if o.stage == winding {
		o.stage, o.readyAt = idle, now+1.5
		o.crossPending = false
	}
	cmds.add(unit.Stand{UnitID: id}, unit.SetVelocity{UnitID: id, VX: cruise * o.heading.X, VY: cruise * o.heading.Y})
}

func (o *operation) onSense(id uint64, s unit.Sense, cmds *commands) {
	if math.Hypot(s.Self.VX, s.Self.VY) > 1 && o.stage == idle {
		o.heading = norm(point{s.Self.VX, s.Self.VY})
	}
	if o.heading == (point{}) {
		o.heading.X = 1
	}
	if s.Knock {
		if !o.knocked {
			o.interrupt(id, s.Time, cmds)
		}
		o.knocked = true
		return
	}
	o.knocked = false
	// Sense is only delivered after both timed and indefinite stuns release.
	o.stunned, o.stunUntil = false, 0
	if s.Time < o.stunUntil {
		return
	}
	if o.stage == cutting {
		if o.knifeOn {
			return
		}
		if o.crossPending {
			o.crossPending = false
			if w := o.wound(o.plan.Wound); w != nil {
				if plan, ok := planCrossKnife(s.Field, w.A, w.B); ok {
					o.plan, o.stage, o.windUntil = plan, winding, s.Time+knifeWindup
					cmds.add(unit.Stand{UnitID: id, Hold: true}, unit.SetVelocity{UnitID: id})
					return
				}
			}
		}
		o.stage = idle
	}
	if o.stage == winding {
		if s.Time+1e-9 >= o.windUntil {
			o.serial++
			w := makeWound(o.serial, o.plan.A, o.plan.B, s.Field)
			o.wounds = append(o.wounds, w)
			if len(o.wounds) > maxWounds {
				o.wounds = o.wounds[1:]
			}
			o.plan.Wound = w.ID
			o.knifeOn, o.stage = true, cutting
			v := norm(o.plan.End.sub(o.plan.Start)).mul(knifeSpeed)
			cmds.add(unit.Spawn{Kind: KindScalpel, OwnerID: id, Slot: s.Self.Slot,
				X: o.plan.Start.X, Y: o.plan.Start.Y, VX: v.X, VY: v.Y, Data: o.plan},
				unit.FX{Name: "surgery-cut", Kind: KindSurgery, UnitID: id, X: o.plan.Start.X, Y: o.plan.Start.Y})
		}
		return
	}
	if o.stage == idle && o.hasOpen() {
		o.stage = approaching
		cmds.add(unit.Pass{UnitID: id, Hold: true}, unit.PassInnerWalls{UnitID: id, Hold: true}, unit.Stand{UnitID: id})
	}
	if o.stage >= approaching {
		o.sew(id, s, cmds)
		return
	}
	if s.Time+1e-9 < o.readyAt {
		return
	}
	target := unit.Seek(s)
	if target == nil {
		return
	}
	plan, ok := planKnife(s.Field, point{s.Self.X, s.Self.Y}, point{target.X, target.Y})
	if !ok {
		o.readyAt = s.Time + .25
		return
	}
	o.plan, o.stage, o.windUntil = plan, winding, s.Time+knifeWindup
	o.crossPending = true
	o.heading = norm(plan.Start.sub(plan.End))
	cmds.add(unit.Stand{UnitID: id, Hold: true}, unit.SetVelocity{UnitID: id})
}

func (o *operation) sew(id uint64, s unit.Sense, cmds *commands) {
	p := point{s.Self.X, s.Self.Y}
	w := o.wound(o.current)
	if w == nil || w.complete() {
		o.stopNeedle(id, cmds)
		w, o.direction = o.nearest(p)
		if w == nil {
			o.current, o.stage, o.readyAt = 0, idle, s.Time+1.5
			cmds.add(unit.Pass{UnitID: id}, unit.PassInnerWalls{UnitID: id}, unit.Stand{UnitID: id},
				unit.SetVelocity{UnitID: id, VX: o.heading.X * cruise, VY: o.heading.Y * cruise})
			return
		}
		o.current, o.stage = w.ID, approaching
		o.route = nil
		o.next = w.firstUnsewn(o.direction)
	}
	// Resume from the last finished puncture after control or a forced displacement.
	start := o.next - o.direction
	if start < 0 {
		start = 0
	}
	if start >= len(w.Points) {
		start = len(w.Points) - 1
	}
	if o.stage == stitching && p.sub(o.lastBody).length() > 45 {
		o.stage = approaching
		o.route = nil
		o.stopNeedle(id, cmds)
	}
	o.lastBody = p
	target := w.Points[o.next]
	speed := stitchSpeed
	if o.stage == approaching {
		target, speed = w.Points[start], approachSpeed
		if p.sub(target).length() < 2 {
			o.stage = stitching
			target, speed = w.Points[o.next], stitchSpeed
		}
	}
	if o.stage == stitching && !o.needleOn {
		o.needleOn = true
		cmds.add(unit.Spawn{Kind: KindSutureNeedle, OwnerID: id, Slot: s.Self.Slot, X: p.X, Y: p.Y, Data: w.ID})
	}
	if o.stage == stitching && p.sub(target).length() < 2 {
		segment := min(o.next, o.next-o.direction)
		w.Closed[segment] = true
		cmds.add(unit.FX{Name: "surgery-stitch", Kind: KindSurgery, UnitID: id, X: target.X, Y: target.Y})
		if w.complete() {
			o.stopNeedle(id, cmds)
			o.current = 0
			cmds.add(unit.SetVelocity{UnitID: id})
			return
		}
		o.next += o.direction
		target = w.Points[o.next]
	}
	if o.stage == approaching {
		if o.routeGoal != target || len(o.route) == 0 {
			o.route, o.routeGoal = fieldRoute(s.Field, p, target, bodyRadius+.5), target
		}
		for len(o.route) > 1 && o.route[0].sub(p).length() < 2 {
			o.route = o.route[1:]
		}
		if len(o.route) > 0 {
			target = o.route[0]
		} else {
			// Never cut across a concave gap if forced against an edge.
			cmds.add(unit.SetVelocity{UnitID: id})
			return
		}
	}
	delta := target.sub(p)
	v := norm(delta).mul(math.Min(speed, delta.length()/stepTime))
	if v.length() > 1 {
		o.heading = norm(v)
	}
	cmds.add(unit.SetVelocity{UnitID: id, VX: v.X, VY: v.Y}, unit.SetFSDirection{UnitID: id, VX: v.X, VY: v.Y})
}
