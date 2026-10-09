package unit

import (
	"testing"
)

func TestHexWalkableCenter(t *testing.T) {
	SetLiveField(HexField())
	if !HexContains(0, 0, 18) {
		t.Fatal("hex center should be walkable")
	}
}
