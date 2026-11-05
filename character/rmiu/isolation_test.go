package r缪

import (
	"testing"
	"xqdj/internal/unit"
)

func TestMiuKillCreditIsMatchLocal(t *testing.T) {
	a, b := &unit.MatchState{}, &unit.MatchState{}
	noteMiuHit(a, 2, 10, KindMiu)
	noteMiuHit(b, 2, 99, KindMiu)
	for i, scope := range []*unit.MatchState{a, b} {
		m := meleeMiu()
		m.prevMiuIDs = map[uint64]bool{2: true}
		m.Handle(unit.Context{State: scope, ID: 10, Kind: KindMiu, Out: make(chan unit.Cmd, 32)}, unit.Sense{Time: 1, Self: miuSelf()})
		want := 0.0
		if i == 0 {
			want = killBonusPer
		}
		if m.killBonus != want {
			t.Fatalf("match %d got bonus %v, want %v", i, m.killBonus, want)
		}
	}
}
