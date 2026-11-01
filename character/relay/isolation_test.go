package 中转站

import (
	"testing"
	"xqdj/internal/unit"
)

func TestPassAndShotCapAreMatchLocal(t *testing.T) {
	a, b := &unit.MatchState{}, &unit.MatchState{}
	setLive(a, 0, shotCap)
	setLive(b, 0, 1)
	queuePass(a, passInfo{sender: 9, target: 4, slot: 0, track: newTrack()})
	if sender, _ := takeShot(b, unit.Snapshot{OwnerID: 4}); sender != 0 {
		t.Fatal("B received A's pass")
	}
	if live(a, 0) != shotCap || live(b, 0) != 1 {
		t.Fatal("same-side shot caps crossed matches")
	}
	resetPass(b)
	if sender, _ := takeShot(a, unit.Snapshot{OwnerID: 4}); sender != 9 {
		t.Fatal("resetting B discarded A's pass")
	}
}
