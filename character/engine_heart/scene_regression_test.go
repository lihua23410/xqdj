package 泵之心

import (
	"math"
	"testing"
	"xqdj/internal/unit"
)

func TestDroppedPartStartsInsideTheField(t *testing.T) {
	f := unit.HexField()
	x, y := f.Clamp(280, 0, 18)
	w := &world{}
	w.initialize(3, unit.Sense{Field: f, Self: unit.Snapshot{X: x, Y: y, Radius: 18, HP: 100}})
	w.parts = []part{{Type: 0, Equipped: true}}
	w.stack = []int{0}
	w.hurt(12, 0)
	p := w.parts[0]
	if !f.Walkable(p.X, p.Y, 7) {
		t.Fatalf("part spawned beyond the boundary: (%f,%f)", p.X, p.Y)
	}
}

func TestUncollectedPartsDoNotFollowTheOwner(t *testing.T) {
	f := unit.HexField()
	w := &world{parts: []part{{Type: 0, X: -100, Y: -60}}}
	for beat := 0; beat < 100; beat++ {
		w.beats = beat
		w.nudge(f, 100, 100)
	}
	p := w.parts[0]
	if math.Hypot(p.X+100, p.Y+60) > 1 {
		t.Fatalf("uncollected part followed the owner to (%f,%f)", p.X, p.Y)
	}
}
