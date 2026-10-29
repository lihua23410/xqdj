package 六边形

import (
	"testing"

	"xqdj/internal/unit"
)

func TestHexOutlineWalkable(t *testing.T) {
	if !Field().Walkable(0, 0, 18) {
		t.Fatal("center")
	}
	if Field().Walkable(unit.HexRadius, 0, 18) {
		t.Fatal("edge")
	}
}
