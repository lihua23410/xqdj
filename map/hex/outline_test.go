package 六边形

import (
	"testing"

	"xqdj/internal/unit"
)

func TestHexOutlineWalkable(t *testing.T) {
	unit.SetLiveField(Field())
	if !unit.HexContains(0, 0, 18) {
		t.Fatal("center")
	}
	if unit.HexContains(unit.HexRadius, 0, 18) {
		t.Fatal("edge")
	}
}
