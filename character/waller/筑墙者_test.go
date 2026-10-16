package 筑墙者

import (
	"math"
	"testing"
	"xqdj/internal/unit"
)

func drain(out chan unit.Cmd) []unit.Cmd {
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

func wallMid(t *testing.T, nearby []unit.Snapshot) (mx, my float64) {
	t.Helper()
	out := make(chan unit.Cmd, 4)
	ctx := unit.Context{ID: 1, Kind: KindWaller, Out: out}
	w := &筑墙者{nextBuild: 0}
	w.Handle(ctx, unit.Sense{
		Time:   wallEvery,
		Self:   unit.Snapshot{ID: 1, Slot: 0, X: 0, Y: 0, AimPriority: unit.DefaultFighterAim},
		Nearby: nearby,
	})
	for _, c := range drain(out) {
		pw, ok := c.(unit.PlaceWall)
		if !ok {
			continue
		}
		return (pw.X1 + pw.X2) / 2, (pw.Y1 + pw.Y2) / 2
	}
	t.Fatal("no PlaceWall")
	return 0, 0
}

func TestPlaceWallUsesSeekNotFirstNearby(t *testing.T) {
	far := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 200, Y: 0, AimPriority: unit.DefaultFighterAim}
	near := unit.Snapshot{ID: 3, Role: unit.RoleFighter, Slot: 1, X: 80, Y: 0, AimPriority: unit.DefaultFighterAim}
	mx, my := wallMid(t, []unit.Snapshot{far, near})
	if math.Abs(mx-40) > 1e-6 || math.Abs(my) > 1e-6 {
		t.Fatalf("mid=(%v,%v) want (40,0) Seek nearest, not first Nearby", mx, my)
	}
}

func TestPlaceWallSkipsSameSlot(t *testing.T) {
	ally := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 0, X: 40, Y: 0, AimPriority: unit.DefaultFighterAim}
	enemy := unit.Snapshot{ID: 3, Role: unit.RoleFighter, Slot: 1, X: 100, Y: 0, AimPriority: unit.DefaultFighterAim}
	mx, my := wallMid(t, []unit.Snapshot{ally, enemy})
	if math.Abs(mx-50) > 1e-6 || math.Abs(my) > 1e-6 {
		t.Fatalf("mid=(%v,%v) want (50,0) toward enemy, not same-slot", mx, my)
	}
}
