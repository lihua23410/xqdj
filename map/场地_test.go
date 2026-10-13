package 场地

import (
	"testing"

	"xqdj/internal/unit"
	"xqdj/map/chamber"
	奇美拉 "xqdj/map/chimera"
	"xqdj/map/circle"
	"xqdj/map/hex"
	八边形 "xqdj/map/octagon"
	"xqdj/map/volcano"
	大六边形 "xqdj/map/widehex"
)

func TestNamesOrder(t *testing.T) {
	got := unit.FieldNames()
	want := map[string]bool{
		机关房.Name:  true,
		圆.Name:    true,
		六边形.Name:  true,
		八边形.Name:  true,
		火山.Name:   true,
		大六边形.Name: true,
		奇美拉.Name:  true,
	}
	if len(got) != len(want) {
		t.Fatalf("names=%v", got)
	}
	for _, n := range got {
		if !want[n] {
			t.Fatalf("unexpected %s in %v", n, got)
		}
	}
}

func TestLookupUnknownMisses(t *testing.T) {
	if _, ok := unit.LookupField("没有这份"); ok {
		t.Fatal("unknown field should miss")
	}
}

func TestCircleHasHardBar(t *testing.T) {
	f, ok := unit.LookupField(圆.Name)
	if !ok {
		t.Fatal("missing 圆")
	}
	if f.Shape != unit.ShapeCircle || len(f.Walls) != 1 || !f.Walls[0].Kind.Hard() || f.Walls[0].Period != 8 {
		t.Fatalf("circle=%+v", f)
	}
	if f.Walkable(0, 0, 18) {
		t.Fatal("场心硬墙 should block")
	}
}

func TestVolcanoHasBootKind(t *testing.T) {
	f, ok := unit.LookupField(火山.Name)
	if !ok || f.BootKind != 火山.KindVolcano {
		t.Fatalf("volcano=%+v ok=%v", f, ok)
	}
}

func TestChamberSquareBoot(t *testing.T) {
	f, ok := unit.LookupField(机关房.Name)
	if !ok || f.Shape != unit.ShapeSquare || f.BootKind != 机关房.KindChamber {
		t.Fatalf("chamber=%+v ok=%v", f, ok)
	}
}
