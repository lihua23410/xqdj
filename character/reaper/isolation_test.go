package 收割者

import (
	"testing"
	"xqdj/internal/unit"
)

func TestRecallMarkIsMatchLocal(t *testing.T) {
	a, b := &unit.MatchState{}, &unit.MatchState{}
	stateFor(a).recallingIDs.Store(uint64(9), struct{}{})
	if _, ok := stateFor(b).recallingIDs.LoadAndDelete(uint64(9)); ok {
		t.Fatal("B consumed A's recall mark")
	}
	if _, ok := stateFor(a).recallingIDs.LoadAndDelete(uint64(9)); !ok {
		t.Fatal("A lost its recall mark")
	}
}
