package 面灵气

import (
	"testing"
	"xqdj/internal/unit"
)

func TestMaskOwnersAndChargesAreMatchLocal(t *testing.T) {
	a, b := &unit.MatchState{}, &unit.MatchState{}
	setLive(a, 1, "qing")
	setLive(b, 1, "hong")
	armAllMasks(a, 1)
	armAllMasks(b, 1)
	if ownerFaction(a, 1) != "qing" || ownerFaction(b, 1) != "hong" {
		t.Fatal("same-ID owners shared factions")
	}
	if !spendMask(a, 1, 0) || !spendMask(b, 1, 0) {
		t.Fatal("spending A's mask spent B's mask")
	}
	armMask(b, 1, 0)
	if spendMask(a, 1, 0) {
		t.Fatal("rearming B rearmed A")
	}
}
