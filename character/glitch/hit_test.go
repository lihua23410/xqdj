package 地慧星

import (
	"testing"
	"xqdj/internal/unit"
)

func TestHitQuotesSwordMark(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	g := &地慧星{}
	g.Handle(unit.Context{ID: 1, Kind: KindGlitch, Out: out}, unit.Collision{
		Time:  1,
		Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter},
	})
	select {
	case cmd := <-out:
		d, ok := cmd.(unit.Damage)
		if !ok {
			t.Fatalf("cmd=%T", cmd)
		}
		if d.To != 2 || d.MarkKind != glitchMarkKind || d.Amount != glitchDamage {
			t.Fatalf("damage=%+v", d)
		}
	default:
		t.Fatal("no damage cmd")
	}
}
