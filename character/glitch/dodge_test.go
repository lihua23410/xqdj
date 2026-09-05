package 地慧星

import (
	"math"
	"testing"
	"xqdj/internal/unit"
)

func TestFarthestCageSpotOppositeEnemy(t *testing.T) {
	x, y := farthestCageSpot(120, 0)
	if x >= 0 || math.Abs(y) > 1 {
		t.Fatalf("from +x enemy got (%v,%v), want left cage corner", x, y)
	}
	if !unit.HexContains(x, y, glitchCagePad) {
		t.Fatalf("spot (%v,%v) outside cage", x, y)
	}
	ox, oy := farthestCageSpot(-120, 40)
	if ox <= 0 {
		t.Fatalf("from -x enemy got (%v,%v), want right side", ox, oy)
	}
}

func TestDodgeTeleportsFarthestAndShoots(t *testing.T) {
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
	wantX, wantY := farthestCageSpot(120, 0)
	if !hasTeleport(cmds, wantX, wantY) {
		t.Fatalf("teleport missing or wrong: %v want %v,%v", cmds, wantX, wantY)
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
