package 地慧星

import (
	"math"
	"testing"
	"xqdj/internal/unit"
)

func TestSlashWindupLocksThenCutsOldAngle(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	g := &地慧星{slashReadyAt: glitchSlashCD}
	ctx := unit.Context{ID: 1, Kind: KindGlitch, Out: out}
	self := unit.Snapshot{
		ID: 1, Kind: KindGlitch, Role: unit.RoleFighter, Slot: 0,
		X: -80, Y: 10, VX: 40, VY: 30, Radius: glitchRadius,
	}
	enemy := unit.Snapshot{
		ID: 2, Role: unit.RoleFighter, Slot: 1,
		X: 40, Y: 10, Radius: 18,
		Marks: []unit.Mark{{Kind: glitchMarkKind, Stacks: 3}},
	}
	ghost := unit.Snapshot{ID: 9, Kind: KindGlitchGhost, Role: unit.RoleHelper, OwnerID: 1, X: -40, Y: 0}

	g.Handle(ctx, unit.Sense{Time: 20, Self: self, Nearby: []unit.Snapshot{enemy, ghost}})
	cmds := drainCmds(out)
	if hasDamage(cmds) {
		t.Fatalf("charge should not damage yet: %v", cmds)
	}
	if !hasZeroVel(cmds) || !hasTeleport(cmds, -80, 10) {
		t.Fatalf("windup should lock pose: %v", cmds)
	}
	if !hasIaiFX(cmds) {
		t.Fatalf("windup should play iai: %v", cmds)
	}
	sp := mustSlash(t, cmds)
	if math.Abs(sp.X-(-80)) > 1e-9 || math.Abs(sp.Y-10) > 1e-9 {
		t.Fatalf("slash origin=%v,%v want locked pose", sp.X, sp.Y)
	}
	if math.Abs(sp.VX-1) > 1e-9 || math.Abs(sp.VY) > 1e-9 {
		t.Fatalf("slash dir=%v,%v want aim at charge start +x", sp.VX, sp.VY)
	}

	moved := self
	moved.X, moved.Y = -10, 80
	moved.VX, moved.VY = 200, -50
	g.Handle(ctx, unit.Sense{Time: 21, Self: moved, Nearby: []unit.Snapshot{enemy, ghost}})
	cmds = drainCmds(out)
	if hasDamage(cmds) {
		t.Fatalf("charge should not damage yet: %v", cmds)
	}
	if !hasZeroVel(cmds) || !hasTeleport(cmds, -80, 10) {
		t.Fatalf("should keep lock: %v", cmds)
	}

	slid := enemy
	slid.X, slid.Y = 40, 220
	g.Handle(ctx, unit.Sense{Time: 22, Self: moved, Nearby: []unit.Snapshot{slid, ghost}})
	cmds = drainCmds(out)
	if hasDamage(cmds) {
		t.Fatal("off-line enemy should miss")
	}
	if hasClearMarks(cmds) || hasDespawnGhosts(cmds) {
		t.Fatalf("miss should keep marks/ghosts: %v", cmds)
	}
	if !hasRestoreVel(cmds, 40, 30) {
		t.Fatalf("should restore pre-lock velocity: %v", cmds)
	}

	g.Handle(ctx, unit.Sense{Time: 31.9, Self: self, Nearby: []unit.Snapshot{slid, ghost}})
	cmds = drainCmds(out)
	if hasSlash(cmds) {
		t.Fatalf("miss CD 10s should still block: %v", cmds)
	}
	g.Handle(ctx, unit.Sense{Time: 32, Self: self, Nearby: []unit.Snapshot{slid, ghost}})
	cmds = drainCmds(out)
	if !hasSlash(cmds) {
		t.Fatalf("miss CD 10s should be ready: %v", cmds)
	}
}

func TestSlashHitsIfEnemyStaysOnFrozenLine(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	g := &地慧星{slashReadyAt: glitchSlashCD}
	ctx := unit.Context{ID: 1, Kind: KindGlitch, Out: out}
	self := unit.Snapshot{
		ID: 1, Kind: KindGlitch, Role: unit.RoleFighter, Slot: 0,
		X: -80, Y: 0, VX: 165, VY: 0, Radius: glitchRadius,
	}
	enemy := unit.Snapshot{
		ID: 2, Role: unit.RoleFighter, Slot: 1,
		X: 90, Y: 0, Radius: 18,
		Marks: []unit.Mark{{Kind: glitchMarkKind, Stacks: 3}},
	}
	ghosts := []unit.Snapshot{
		{ID: 8, Kind: KindGlitchGhost, Role: unit.RoleHelper, OwnerID: 1},
		{ID: 9, Kind: KindGlitchGhost, Role: unit.RoleHelper, OwnerID: 1},
	}
	g.Handle(ctx, unit.Sense{Time: 20, Self: self, Nearby: append([]unit.Snapshot{enemy}, ghosts...)})
	_ = drainCmds(out)

	along := enemy
	along.X = 140
	along.Y = 4
	g.Handle(ctx, unit.Sense{Time: 22, Self: self, Nearby: append([]unit.Snapshot{along}, ghosts...)})
	cmds := drainCmds(out)
	d := mustDamage(t, cmds)
	want := (6.0 + 3.0*2.0) * (1.0 + 2.0)
	if math.Abs(d.Amount-want) > 1e-9 || d.To != 2 {
		t.Fatalf("damage=%+v want %v to enemy", d, want)
	}
	if !hasClearMarks(cmds) || !hasDespawnGhosts(cmds) {
		t.Fatalf("hit should spend marks/ghosts: %v", cmds)
	}

	g.Handle(ctx, unit.Sense{Time: 41.9, Self: self, Nearby: append([]unit.Snapshot{along}, ghosts...)})
	cmds = drainCmds(out)
	if hasSlash(cmds) {
		t.Fatalf("hit CD 20s should still block: %v", cmds)
	}
	g.Handle(ctx, unit.Sense{Time: 42, Self: self, Nearby: append([]unit.Snapshot{along}, ghosts...)})
	cmds = drainCmds(out)
	if !hasSlash(cmds) {
		t.Fatalf("hit CD 20s should be ready: %v", cmds)
	}
}

func drainCmds(out <-chan unit.Cmd) []unit.Cmd {
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

func hasDamage(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if _, ok := c.(unit.Damage); ok {
			return true
		}
	}
	return false
}

func hasZeroVel(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.SetVelocity); ok && v.VX == 0 && v.VY == 0 {
			return true
		}
	}
	return false
}

func hasIaiFX(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == "iai" {
			return true
		}
	}
	return false
}

func hasRestoreVel(cmds []unit.Cmd, vx, vy float64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.SetVelocity); ok && math.Abs(v.VX-vx) < 1e-9 && math.Abs(v.VY-vy) < 1e-9 {
			return true
		}
	}
	return false
}

func hasTeleport(cmds []unit.Cmd, x, y float64) bool {
	for _, c := range cmds {
		if tp, ok := c.(unit.Teleport); ok && math.Abs(tp.X-x) < 1e-9 && math.Abs(tp.Y-y) < 1e-9 {
			return true
		}
	}
	return false
}

func hasClearMarks(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if _, ok := c.(unit.ClearMarks); ok {
			return true
		}
	}
	return false
}

func hasDespawnGhosts(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if d, ok := c.(unit.DespawnOwned); ok && d.Kind == KindGlitchGhost {
			return true
		}
	}
	return false
}

func hasSlash(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == KindGlitchSlash {
			return true
		}
	}
	return false
}

func mustSlash(t *testing.T, cmds []unit.Cmd) unit.Spawn {
	t.Helper()
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == KindGlitchSlash {
			return s
		}
	}
	t.Fatalf("no slash in %v", cmds)
	return unit.Spawn{}
}

func mustDamage(t *testing.T, cmds []unit.Cmd) unit.Damage {
	t.Helper()
	for _, c := range cmds {
		if d, ok := c.(unit.Damage); ok {
			return d
		}
	}
	t.Fatalf("no damage in %v", cmds)
	return unit.Damage{}
}
