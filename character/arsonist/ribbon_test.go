package 纵火者

import (
	"bytes"
	"math"
	"testing"

	"xqdj/internal/unit"
)

func TestOilRibbonFollowsDistanceInsteadOfFrameRate(t *testing.T) {
	for _, thin := range []bool{false, true} {
		whole, frames := oilSurface{}, oilSurface{}
		radius := func(d float64) float64 {
			if thin {
				return bottleTrailWidth
			}
			return trailRadius(bodyRadius, d)
		}
		whole.addOilRibbon(0, 0, 90, 0, radius(0), radius(90), 0, 123, 0)
		for i := 0; i < 60; i++ {
			a, b := float64(i)*1.5, float64(i+1)*1.5
			frames.addOilRibbon(a, 0, b, 0, radius(a), radius(b), a, 123, float64(i)/60)
		}
		if !bytes.Equal(whole.payload(), frames.payload()) {
			t.Fatalf("thin=%v: frame subdivision changed the oil ribbon", thin)
		}
		for x := 0.0; x <= 90; x += 1.5 {
			if !whole.touches(unit.Snapshot{X: x, Radius: 1}, oil) {
				t.Fatalf("thin=%v: gap at %v", thin, x)
			}
		}
	}
}
func TestFineOilRibbonHasUnevenButBoundedWidth(t *testing.T) {
	g := oilSurface{}
	g.addOilRibbon(0, 0, 240, 0, 8, 8, 0, 77, 0)
	min, max := math.Inf(1), 0.0
	for x := 24.0; x < 230; x += surfaceStep {
		width := 0.0
		for y := -7; y <= 7; y++ {
			if c := g.cell(point{gridCoord(x), y}); c != nil && c.state == oil {
				width += surfaceStep
			}
		}
		min = math.Min(min, width)
		max = math.Max(max, width)
	}
	if max-min < 3 || min < 9 || max > 30 {
		t.Fatalf("thin ribbon width %.1f..%.1f", min, max)
	}
}
func TestTinyBodyRibbonStaysConnected(t *testing.T) {
	g := oilSurface{}
	g.addOilRibbon(0, 0, 90, 0, 1, 1, 0, 58, 0)
	for x := 0.0; x <= 90; x++ {
		if !g.touches(unit.Snapshot{X: x, Radius: 1}, oil) {
			t.Fatalf("tiny body oil gap at %v", x)
		}
	}
}
