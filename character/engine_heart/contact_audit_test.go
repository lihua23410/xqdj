package 泵之心

import (
	"testing"
	"xqdj/character/engine_heart/parts/api"
	"xqdj/character/engine_heart/parts/arc"
	"xqdj/internal/audit"
	"xqdj/internal/unit"
)

// Exercise the existing engine's fan -> owner receipt -> propagation actor flow.
func deliverContact(t *testing.T, a unit.Actor, ctx unit.Context, out chan unit.Cmd, s unit.Sense, kind string, hit unit.Collision) {
	t.Helper()
	var saved []unit.Cmd
	for len(out) > 0 {
		saved = append(saved, <-out)
	}
	fan := unit.NewActor(kind, unit.SpawnInfo{State: ctx.State, OwnerID: ctx.ID, Slot: s.Self.Slot})
	if fan == nil {
		t.Fatalf("missing contact actor %s", kind)
	}
	fctx := ctx
	fctx.ID = 101
	fctx.Kind = kind
	fan.Handle(fctx, hit)
	var contacts []unit.FanContact
	for len(out) > 0 {
		c := <-out
		audit.CheckCommand(audit.Origin{ID: 101, Kind: kind, Role: unit.RoleProjectile, Collision: true, Fan: true}, c)
		if contact, ok := c.(unit.FanContact); ok {
			contacts = append(contacts, contact)
		}
		saved = append(saved, c)
	}
	for _, c := range contacts {
		a.Handle(ctx, unit.FanHit{From: 101, Kind: kind, Time: c.Collision.Time, Other: c.Collision.Other})
		var births []unit.Spawn
		for len(out) > 0 {
			cmd := <-out
			audit.CheckCommand(audit.Origin{ID: ctx.ID, Kind: KindEngineHeart, Role: unit.RoleFighter, Collision: true}, cmd)
			saved = append(saved, cmd)
			if birth, ok := cmd.(unit.Spawn); ok && birth.Kind == arc.KindChain {
				births = append(births, birth)
			}
		}
		for _, birth := range births {
			child := unit.NewActor(birth.Kind, unit.SpawnInfo{State: ctx.State, OwnerID: ctx.ID, Slot: birth.Slot, Data: birth.Data})
			if child == nil {
				t.Fatal("missing propagation actor")
			}
			cctx := ctx
			cctx.ID = 102
			cctx.Kind = birth.Kind
			view := s
			view.Time = hit.Time
			view.Self = unit.Snapshot{ID: 102, OwnerID: ctx.ID, Slot: birth.Slot, X: birth.X, Y: birth.Y}
			child.Handle(cctx, view)
			for len(out) > 0 {
				cmd := <-out
				audit.CheckCommand(audit.Origin{ID: 102, Kind: birth.Kind, Role: unit.RoleHelper}, cmd)
				saved = append(saved, cmd)
			}
		}
	}
	for _, cmd := range saved {
		out <- cmd
	}
}

// Isolated damage tests miss the engine's event-origin check. Exercise that exact seam.
func TestContactWeaponCommandsPassRuntimeAudit(t *testing.T) {
	for _, id := range []int{12, 14} {
		a, ctx, out, s := testWeapon(t, id)
		for i := 0; i < 3; i++ {
			a.Handle(ctx, api.Pulse{Sense: s})
		}
		for len(out) > 0 {
			<-out
		}
		a.Handle(ctx, unit.Collision{Time: .1, Other: s.Nearby[0]})
		for len(out) > 0 {
			audit.CheckCommand(audit.Origin{ID: ctx.ID, Kind: KindEngineHeart, Role: unit.RoleFighter, Collision: true}, <-out)
		}
	}
}
