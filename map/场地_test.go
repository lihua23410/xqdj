package 场地

import (
	"testing"
	"xqdj/internal/unit"
)

func TestNamesOrder(t *testing.T) {
	got := unit.FieldNames()
	if len(got) < 2 || got[0] != unit.NameCircle || got[1] != unit.NameHex {
		t.Fatalf("names=%v", got)
	}
}

func TestLookupUnknownMisses(t *testing.T) {
	if _, ok := unit.LookupField("没有这份"); ok {
		t.Fatal("unknown field should miss")
	}
}

func TestCircleHasHardBar(t *testing.T) {
	f, ok := unit.LookupField(unit.NameCircle)
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
