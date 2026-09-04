package 狼人

import (
	"testing"
	"xqdj/internal/unit"
)

func TestBootSpawnsMoonAtCenter(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	w := &狼人{}
	w.Handle(unit.Context{ID: 1, Kind: KindWolf, Out: out}, unit.Sense{
		Time: 0,
		Self: selfAt(80, 0),
	})
	cmds := drain(out)
	sp := mustSpawn(t, cmds)
	if sp.Kind != KindMoon || sp.X != 0 || sp.Y != 0 || sp.OwnerID != 1 {
		t.Fatalf("spawn=%+v", sp)
	}
	ph := mustPhase(t, cmds)
	if ph.Amount != 0 {
		t.Fatalf("initial phase=%v", ph.Amount)
	}
}

func TestTouchAdvancesPhaseOncePerEnter(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	w := bootedWolf()
	moon := moonSnap(1)
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}

	w.Handle(ctx, unit.Sense{Time: 1, Self: selfAt(10, 0), Nearby: []unit.Snapshot{moon}})
	if got := lastPhase(drain(out)); got != 1 {
		t.Fatalf("enter phase=%v", got)
	}

	w.Handle(ctx, unit.Sense{Time: 1.1, Self: selfAt(10, 0), Nearby: []unit.Snapshot{moon}})
	if cmds := drain(out); hasPhase(cmds) {
		t.Fatalf("stay should not advance: %v", cmds)
	}

	w.Handle(ctx, unit.Sense{Time: 1.2, Self: selfAt(120, 0), Nearby: []unit.Snapshot{moon}})
	_ = drain(out)
	w.Handle(ctx, unit.Sense{Time: 1.3, Self: selfAt(10, 0), Nearby: []unit.Snapshot{moon}})
	if got := lastPhase(drain(out)); got != 2 {
		t.Fatalf("reenter phase=%v", got)
	}
}

func TestEnemyTouchAdvancesPhase(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	w := bootedWolf()
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	moon := moonSnap(1)
	enemy := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 8, Y: 0, Radius: 18}
	w.Handle(ctx, unit.Sense{Time: 1, Self: selfAt(120, 0), Nearby: []unit.Snapshot{moon, enemy}})
	if got := lastPhase(drain(out)); got != 1 {
		t.Fatalf("enemy touch phase=%v", got)
	}
}

func TestFullMoonStartsRageThenLocksPhase(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	w := bootedWolf()
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	moon := moonSnap(1)

	for i := 0; i < fullMoon; i++ {
		touch(w, ctx, float64(i)+1, true)
		_ = drain(out)
		touch(w, ctx, float64(i)+1.5, false)
		_ = drain(out)
	}
	if w.phase != fullMoon || !w.raging {
		t.Fatalf("phase=%d raging=%v", w.phase, w.raging)
	}

	w.Handle(ctx, unit.Sense{Time: 4.2, Self: selfAt(10, 0), Nearby: []unit.Snapshot{moon}})
	if cmds := drain(out); hasPhase(cmds) {
		t.Fatalf("rage should lock phase: %v", cmds)
	}

	w.Handle(ctx, unit.Sense{
		Time:   4.3,
		Self:   selfAt(40, 30),
		Nearby: []unit.Snapshot{moon, unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 90, Y: 30, Radius: 18}},
	})
	cmds := drain(out)
	var vel *unit.SetVelocity
	for _, c := range cmds {
		if v, ok := c.(unit.SetVelocity); ok {
			vel = &v
		}
	}
	if vel == nil {
		t.Fatal("rage should chase")
	}
	if vel.VX <= 0 {
		t.Fatalf("chase vx=%v want toward enemy +x", vel.VX)
	}
}

func TestRageEndsAdvancesPhase(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	w := bootedWolf()
	w.phase = fullMoon
	w.raging = true
	w.rageUntil = 5
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	w.Handle(ctx, unit.Sense{Time: 5, Self: selfAt(120, 0), Nearby: []unit.Snapshot{moonSnap(1)}})
	if w.raging {
		t.Fatal("rage should end")
	}
	if w.phase != fullMoon+1 {
		t.Fatalf("phase after rage=%d", w.phase)
	}
	if got := lastPhase(drain(out)); got != float64(fullMoon+1) {
		t.Fatalf("phase fx=%v", got)
	}
}

func bootedWolf() *狼人 {
	return &狼人{booted: true}
}

func selfAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindWolf, Role: unit.RoleFighter, Slot: 0,
		X: x, Y: y, Radius: wolfRadius,
	}
}

func moonSnap(owner uint64) unit.Snapshot {
	return unit.Snapshot{
		ID: 99, Kind: KindMoon, Role: unit.RoleHelper, OwnerID: owner, Slot: 0,
		X: 0, Y: 0, Radius: moonRadius,
	}
}

func touch(w *狼人, ctx unit.Context, t float64, inside bool) {
	x := 120.0
	if inside {
		x = 10
	}
	w.Handle(ctx, unit.Sense{Time: t, Self: selfAt(x, 0), Nearby: []unit.Snapshot{moonSnap(1)}})
}

func drain(out <-chan unit.Cmd) []unit.Cmd {
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

func mustSpawn(t *testing.T, cmds []unit.Cmd) unit.Spawn {
	t.Helper()
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == KindMoon {
			return s
		}
	}
	t.Fatalf("no moon spawn in %v", cmds)
	return unit.Spawn{}
}

func mustPhase(t *testing.T, cmds []unit.Cmd) unit.FX {
	t.Helper()
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == "phase" {
			return fx
		}
	}
	t.Fatalf("no phase fx in %v", cmds)
	return unit.FX{}
}

func lastPhase(cmds []unit.Cmd) float64 {
	n := -1.0
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == "phase" {
			n = fx.Amount
		}
	}
	return n
}

func hasPhase(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == "phase" {
			return true
		}
	}
	return false
}
