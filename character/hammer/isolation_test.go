package 钉与锤

import (
	"testing"
	"xqdj/internal/unit"
)

func TestMarkedFoeIsMatchLocal(t *testing.T) {
	a, b := &unit.MatchState{}, &unit.MatchState{}
	rememberFoe(a, 1, 2)
	rememberFoe(b, 1, 3)
	if foeOfOwner(a, 1) != 2 || foeOfOwner(b, 1) != 3 {
		t.Fatal("same-ID owners shared their marked foe")
	}
}
