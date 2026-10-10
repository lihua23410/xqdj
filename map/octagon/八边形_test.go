package 八边形

import (
	"math"
	"testing"

	"xqdj/internal/unit"
)

func TestFieldOctagonBoot(t *testing.T) {
	f := Field()
	if f.Shape != unit.ShapeOctagon || f.Extent != unit.HexRadius || f.BootKind != KindOctagon {
		t.Fatalf("field=%+v", f)
	}
	if !f.OutlineContains(0, 0, 18) {
		t.Fatal("center")
	}
	apo := unit.HexRadius * math.Cos(math.Pi/8)
	if f.OutlineContains(apo+1, 0, 0) {
		t.Fatal("outside apothem on axis")
	}
	// 顶点方向 22.5°：略超 apothem 仍在场内（截角处比内切圆宽）。
	if !f.OutlineContains((apo+4)*math.Cos(math.Pi/8), (apo+4)*math.Sin(math.Pi/8), 0) {
		t.Fatal("corner strip should be walkable")
	}
	if f.OutlineContains(unit.HexRadius*math.Cos(math.Pi/8), unit.HexRadius*math.Sin(math.Pi/8), 18) {
		t.Fatal("outside vertex")
	}
}

func TestControllerPlacesThreeVisionBlockWalls(t *testing.T) {
	a := &八边形{}
	out := make(chan unit.Cmd, 8)
	a.Handle(unit.Context{ID: 1, Kind: KindOctagon, Out: out}, unit.Sense{
		Self: unit.Snapshot{ID: 1, Slot: 0},
	})
	close(out)
	var walls []unit.PlaceWall
	for c := range out {
		if w, ok := c.(unit.PlaceWall); ok {
			walls = append(walls, w)
		}
	}
	if len(walls) != 3 {
		t.Fatalf("walls=%d want 3", len(walls))
	}
	seen := map[float64]bool{}
	for _, w := range walls {
		if !w.Hard || !w.VisionBlock {
			t.Fatalf("wall %+v must be hard vision-block", w)
		}
		mx, my := (w.X1+w.X2)/2, (w.Y1+w.Y2)/2
		if d := math.Hypot(mx, my); math.Abs(d-wallDist) > 1e-6 {
			t.Fatalf("mid dist=%v want %v", d, wallDist)
		}
		if l := math.Hypot(w.X2-w.X1, w.Y2-w.Y1); math.Abs(l-wallLen) > 1e-6 {
			t.Fatalf("len=%v want %v", l, wallLen)
		}
		ang := math.Atan2(my, mx)
		deg := math.Mod(math.Round(ang*180/math.Pi), 360)
		if deg < 0 {
			deg += 360
		}
		seen[deg] = true
	}
	for _, want := range []float64{90, 210, 330} {
		if !seen[want] {
			t.Fatalf("missing %v° among %v", want, seen)
		}
	}
}

func TestControllerBootsOnce(t *testing.T) {
	a := &八边形{}
	out := make(chan unit.Cmd, 16)
	for i := 0; i < 3; i++ {
		a.Handle(unit.Context{ID: 1, Kind: KindOctagon, Out: out}, unit.Sense{
			Self: unit.Snapshot{ID: 1, Slot: 0},
		})
	}
	close(out)
	n := 0
	for c := range out {
		if _, ok := c.(unit.PlaceWall); ok {
			n++
		}
	}
	if n != 3 {
		t.Fatalf("walls=%d want 3 (boot once)", n)
	}
}
