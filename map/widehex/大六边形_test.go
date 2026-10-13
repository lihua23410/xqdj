package 大六边形

import (
	"math"
	"testing"

	"xqdj/internal/unit"
)

func TestWideHexField(t *testing.T) {
	f, ok := unit.LookupField(Name)
	if !ok {
		t.Fatal("missing 大六边形")
	}
	if f.Shape != unit.ShapeHex || f.Extent != Extent || len(f.Walls) != 0 || f.BootKind != "" {
		t.Fatalf("field=%+v", f)
	}
	if Extent != 3*unit.HexRadius {
		t.Fatalf("extent=%v", Extent)
	}
}

func TestWideHexWalkable(t *testing.T) {
	unit.SetLiveField(Field())
	if !unit.HexContains(0, 0, 18) {
		t.Fatal("center")
	}
	if unit.HexContains(Extent, 0, 18) {
		t.Fatal("vertex")
	}
	// 默认六边形的场边点，在这一份里是场内。
	if !unit.HexContains(unit.HexRadius, 0, 18) {
		t.Fatal("default vertex should be inside")
	}
	apo := Extent * math.Sqrt(3) / 2
	if unit.HexContains(0, apo, 18) {
		t.Fatal("flat")
	}
	if !unit.HexContains(0, apo/2, 18) {
		t.Fatal("mid apo")
	}
}
