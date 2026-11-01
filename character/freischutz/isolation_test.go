package Freischutz

import (
	"testing"
	"xqdj/internal/unit"
)

func TestShotCapAndRepriseAreMatchLocal(t *testing.T) {
	a, b := &unit.MatchState{}, &unit.MatchState{}
	setLive(a, 0, 7)
	setLive(b, 0, 2)
	queueReprise(a, 77, 0, 0, 8)
	queueReprise(b, 77, 0, 0, 3)
	if dmg, ok := takeRepriseStream(b, 77); !ok || dmg != 3 {
		t.Fatalf("B's reprise = %v, %v", dmg, ok)
	}
	resetLive(b)
	if dmg, ok := takeRepriseStream(a, 77); !ok || dmg != 8 {
		t.Fatalf("B's cleanup removed A's reprise: %v, %v", dmg, ok)
	}
	if live(a, 0) != 7 {
		t.Fatal("B's cleanup changed A's shot count")
	}
}
