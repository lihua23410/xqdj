package 地慧星

import (
	"math"
	"testing"
	"xqdj/internal/unit"
)

func TestDodgeTeleportsWalkableAndShoots(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	g := &地慧星{slashReadyAt: glitchSlashCD}
	ctx := unit.Context{ID: 1, Kind: KindGlitch, Out: out}
	enemy := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 120, Y: 0, Radius: 18}
	g.Handle(ctx, unit.Sense{
		Time:   1,
		Self:   unit.Snapshot{ID: 1, Kind: KindGlitch, Role: unit.RoleFighter, Slot: 0, X: -120, Y: 0, VX: 0, VY: 165, Radius: glitchRadius},
		Nearby: []unit.Snapshot{enemy},
	})
	_ = drainDodge(out)

	g.Handle(ctx, unit.IncomingDamage{Token: 7, From: 2, Amount: 14, Time: 1.2, Speed: 165})
	cmds := drainDodge(out)
	var tp *unit.Teleport
	for _, c := range cmds {
		if t, ok := c.(unit.Teleport); ok {
			cp := t
			tp = &cp
		}
	}
	if tp == nil {
		t.Fatalf("teleport missing: %v", cmds)
	}
	if !unit.HexContains(tp.X, tp.Y, glitchCagePad) {
		t.Fatalf("teleport (%v,%v) not walkable", tp.X, tp.Y)
	}
	shot := mustKind(t, cmds, KindGlitchShot)
	if shot.VX <= 0 {
		t.Fatalf("shot vx=%v want toward enemy +x", shot.VX)
	}
	if math.Abs(math.Hypot(shot.VX, shot.VY)-glitchShotSpeed) > 1e-6 {
		t.Fatalf("shot speed=%v want %v", math.Hypot(shot.VX, shot.VY), glitchShotSpeed)
	}
	if !hasBlock(cmds) {
		t.Fatal("dodge should block the hit")
	}
}

func TestWallBoostSnapsCruiseAndSpeed(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	g := &地慧星{slashReadyAt: glitchSlashCD}
	ctx := unit.Context{ID: 1, Kind: KindGlitch, Out: out}
	self := unit.Snapshot{
		ID: 1, Kind: KindGlitch, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, VX: 0, VY: glitchSpeed, Radius: glitchRadius,
	}
	g.Handle(ctx, unit.WallHit{Time: 1, NX: 1, NY: 0})
	g.Handle(ctx, unit.Sense{Time: 1.01, Self: self})
	cmds := drainDodge(out)
	want := glitchSpeed + glitchWallBoost
	if !hasCruiseSpeed(cmds, want) {
		t.Fatalf("boost should raise cruise to %v: %v", want, cmds)
	}
	if !hasVel(cmds, 0, want) {
		t.Fatalf("boost should snap vel to %v: %v", want, cmds)
	}

	g.Handle(ctx, unit.IncomingDamage{Token: 1, From: 2, Amount: 10, Time: 1.2})
	if !hasBlock(drainDodge(out)) {
		t.Fatal("first hit should dodge")
	}
	g.Handle(ctx, unit.IncomingDamage{Token: 2, From: 2, Amount: 10, Time: 1.3})
	end := drainDodge(out)
	if !hasCruiseSpeed(end, glitchSpeed) {
		t.Fatalf("real damage should snap cruise back: %v", end)
	}
	if !hasVel(end, 0, glitchSpeed) {
		t.Fatalf("real damage should snap vel back: %v", end)
	}
}

func TestCapsuleWallDoesNotBoost(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	g := &地慧星{slashReadyAt: glitchSlashCD}
	ctx := unit.Context{ID: 1, Kind: KindGlitch, Out: out}
	self := unit.Snapshot{
		ID: 1, Kind: KindGlitch, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, VX: 0, VY: glitchSpeed, Radius: glitchRadius,
	}
	g.Handle(ctx, unit.WallHit{Time: 1, NX: 1, NY: 0, Kind: unit.WallCapsule})
	g.Handle(ctx, unit.Sense{Time: 1.01, Self: self})
	if hasCruiseSpeed(drainDodge(out), glitchSpeed+glitchWallBoost) {
		t.Fatal("胶囊墙 must not boost")
	}
}

func drainDodge(out <-chan unit.Cmd) []unit.Cmd {
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

func mustKind(t *testing.T, cmds []unit.Cmd, kind string) unit.Spawn {
	t.Helper()
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == kind {
			return s
		}
	}
	t.Fatalf("no spawn %s in %v", kind, cmds)
	return unit.Spawn{}
}

func hasBlock(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if _, ok := c.(unit.BlockDamage); ok {
			return true
		}
	}
	return false
}

func hasCruiseSpeed(cmds []unit.Cmd, speed float64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.SetCruise); ok && math.Abs(v.Speed-speed) < 1e-9 {
			return true
		}
	}
	return false
}

func hasVel(cmds []unit.Cmd, vx, vy float64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.SetVelocity); ok && math.Abs(v.VX-vx) < 1e-9 && math.Abs(v.VY-vy) < 1e-9 {
			return true
		}
	}
	return false
}
