package 地慧星

import (
	"testing"
	"xqdj/internal/unit"
)

func TestHitQuotesSwordMark(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	g := &地慧星弧{slot: 0, booted: true}
	g.Handle(unit.Context{ID: 1, Kind: KindGlitchArc, Out: out}, unit.Collision{
		Time:  1,
		Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1},
	})
	select {
	case cmd := <-out:
		d, ok := cmd.(unit.Damage)
		if !ok {
			t.Fatalf("cmd=%T", cmd)
		}
		if d.To != 2 || d.MarkKind != glitchMarkKind || d.Amount != glitchDamage {
			t.Fatalf("damage=%+v", d)
		}
	default:
		t.Fatal("no damage cmd")
	}
}

func TestGhostMarksEnemyOncePerPass(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	g := &地慧星残影{slot: 0}
	ctx := unit.Context{ID: 9, Kind: KindGlitchGhost, Out: out}
	self := unit.Snapshot{ID: 9, Kind: KindGlitchGhost, Role: unit.RoleHelper, Slot: 0, X: 0, Y: 0, Radius: glitchRadius}
	enemy := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 10, Y: 0, Radius: 18}

	g.Handle(ctx, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{enemy}})
	m := mustStackMark(t, drainGhost(out))
	if m.UnitID != 2 || m.Kind != glitchMarkKind || m.Delta != 1 || m.Icon != glitchMarkIcon {
		t.Fatalf("mark=%+v", m)
	}

	g.Handle(ctx, unit.Sense{Time: 1.1, Self: self, Nearby: []unit.Snapshot{enemy}})
	if cmds := drainGhost(out); len(cmds) != 0 {
		t.Fatalf("stay should not restack: %v", cmds)
	}

	away := enemy
	away.X = 80
	g.Handle(ctx, unit.Sense{Time: 1.2, Self: self, Nearby: []unit.Snapshot{away}})
	if cmds := drainGhost(out); len(cmds) != 0 {
		t.Fatalf("leave should be quiet: %v", cmds)
	}

	g.Handle(ctx, unit.Sense{Time: 1.3, Self: self, Nearby: []unit.Snapshot{enemy}})
	if mustStackMark(t, drainGhost(out)).Delta != 1 {
		t.Fatal("reenter should mark again")
	}

	ally := unit.Snapshot{ID: 1, Role: unit.RoleFighter, Slot: 0, X: 0, Y: 0, Radius: 18}
	g.inside = false
	g.Handle(ctx, unit.Sense{Time: 2, Self: self, Nearby: []unit.Snapshot{ally}})
	if cmds := drainGhost(out); len(cmds) != 0 {
		t.Fatalf("owner should not be marked: %v", cmds)
	}
}

func drainGhost(out <-chan unit.Cmd) []unit.Cmd {
	var cmds []unit.Cmd
	for {
		select {
		case c := <-out:
			cmds = append(cmds, c)
		default:
			return cmds
		}
	}
}

func mustStackMark(t *testing.T, cmds []unit.Cmd) unit.StackMark {
	t.Helper()
	for _, c := range cmds {
		if m, ok := c.(unit.StackMark); ok {
			return m
		}
	}
	t.Fatalf("no stack mark in %v", cmds)
	return unit.StackMark{}
}
