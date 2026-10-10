package 场地

import (
	"testing"

	"xqdj/internal/unit"
	"xqdj/map/chamber"
	"xqdj/map/circle"
	"xqdj/map/hex"
	"xqdj/map/volcano"
)

func TestNamesOrder(t *testing.T) {
	got := unit.FieldNames()
	// 目录名排序：chamber, circle, hex, volcano
	if len(got) < 4 || got[0] != 机关房.Name || got[1] != 圆.Name || got[2] != 六边形.Name || got[3] != 火山.Name {
		t.Fatalf("names=%v", got)
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
