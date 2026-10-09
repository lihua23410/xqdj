package 场地

import (
	"testing"
	"xqdj/internal/unit"
	"xqdj/map/circle"
	"xqdj/map/hex"
	"xqdj/map/volcano"
)

func TestNamesOrder(t *testing.T) {
	got := unit.FieldNames()
	if len(got) < 3 || got[0] != 圆.Name || got[1] != 六边形.Name || got[2] != 火山.Name {
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
