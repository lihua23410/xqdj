package unit

import (
	"math"
	"testing"
)

func TestDetourStraightWhenClear(t *testing.T) {
	wx, wy, hold := Detour(0, 0, 9, 80, 0, nil, DetourHold{}, 1)
	if wx != 80 || wy != 0 || hold.ID != 0 {
		t.Fatalf("clear=(%v,%v) hold=%+v", wx, wy, hold)
	}
}

func TestDetourGoesPastNearerEnd(t *testing.T) {
	// 竖墙在 x=50，目标偏上，上端更近。
	wall := WallView{ID: 9, X1: 50, Y1: -80, X2: 50, Y2: 40, Radius: 6}
	wx, wy, hold := Detour(0, 0, 9, 200, 30, []WallView{wall}, DetourHold{}, 1)
	wantY := 40 + 9 + 6 + detourPad
	if math.Abs(wx-50) > 1e-6 || math.Abs(wy-wantY) > 1e-6 {
		t.Fatalf("waypoint=(%v,%v) want (50,%v)", wx, wy, wantY)
	}
	if hold.ID != 9 || hold.End != 2 || hold.Until != 1+detourHold {
		t.Fatalf("hold=%+v", hold)
	}
}

func TestDetourIgnoresVisionBlockFlag(t *testing.T) {
	wall := WallView{ID: 3, X1: 40, Y1: -20, X2: 40, Y2: 20, Radius: 6}
	wx, wy, _ := Detour(0, 0, 9, 100, 0, []WallView{wall}, DetourHold{}, 0)
	if math.Abs(wy) < 1 {
		t.Fatalf("physical wall should detour, got (%v,%v)", wx, wy)
	}
}

func TestDetourKeepsSideWhileHeld(t *testing.T) {
	wall := WallView{ID: 9, X1: 50, Y1: -80, X2: 50, Y2: 40, Radius: 6}
	hold := DetourHold{Until: 2, ID: 9, End: 1}
	wx, wy, next := Detour(0, 0, 9, 200, 30, []WallView{wall}, hold, 1.5)
	wantY := -80 - (9 + 6 + detourPad)
	if math.Abs(wy-wantY) > 1e-6 || math.Abs(wx-50) > 1e-6 {
		t.Fatalf("held side=(%v,%v) want (50,%v)", wx, wy, wantY)
	}
	if next != hold {
		t.Fatalf("hold refreshed: %+v", next)
	}
}
