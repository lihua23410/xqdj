package 机关房

import (
	"math"
	"testing"

	"xqdj/internal/unit"
)

func TestFieldSquareBoot(t *testing.T) {
	f := Field()
	if f.Shape != unit.ShapeSquare || f.Extent != unit.HexRadius || f.BootKind != KindChamber {
		t.Fatalf("field=%+v", f)
	}
	if !f.OutlineContains(0, 0, 18) {
		t.Fatal("center")
	}
	half := squareHalf()
	if f.OutlineContains(half+1, 0, 0) {
		t.Fatal("outside square")
	}
}

func TestSquareHalfMatchesOutline(t *testing.T) {
	half := squareHalf()
	want := unit.HexRadius * (3 - math.Sqrt(3)) / 2
	if math.Abs(half-want) > 1e-9 {
		t.Fatalf("half=%v want=%v", half, want)
	}
	f := Field()
	if !f.OutlineContains(half-1e-6, half-1e-6, 0) {
		t.Fatal("square corner inside")
	}
	if f.OutlineContains(half+1, 0, 0) {
		t.Fatal("outside square")
	}
}

func TestPickToxicDistinct(t *testing.T) {
	a := &机关房{}
	a.ensureRNG(1, 0)
	for i := 0; i < 40; i++ {
		a.pickToxic()
		if a.toxic[0] == a.toxic[1] {
			t.Fatalf("same %v", a.toxic)
		}
	}
}

func TestCellInnerCenterInset(t *testing.T) {
	half := squareHalf()
	c := cellInner(half, 4)
	cell := (2 * half) / 3
	if math.Abs(c.x0-(-half+cell+wallHalf)) > 1e-9 {
		t.Fatalf("x0=%v", c.x0)
	}
}
