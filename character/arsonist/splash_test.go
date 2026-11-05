package 纵火者

import (
	"bytes"
	"math"
	"testing"

	"xqdj/internal/unit"
)

func TestSplashIsIrregularStableAndKeepsArea(t *testing.T) {
	var previous []byte
	for seed := uint64(0); seed < 64; seed++ {
		g := oilSurface{}
		g.addOilSplash(0, 0, oilRadius, seed, 0)
		area := float64(countState(&g, oil)) * surfaceStep * surfaceStep
		if relative := area / (math.Pi * oilRadius * oilRadius); relative < .94 || relative > 1.06 {
			t.Fatalf("seed %d: area ratio %.3f", seed, relative)
		}
		shape := makeOilSplash(oilRadius, seed)
		min, max := math.Inf(1), 0.0
		for _, r := range shape.radius {
			min = math.Min(min, r)
			max = math.Max(max, r)
		}
		if max-min < oilRadius*.18 || max > oilRadius*1.45 || min < oilRadius*.6 {
			t.Fatalf("seed %d: invalid contour %.1f..%.1f", seed, min, max)
		}
		for _, d := range shape.drops {
			if math.Hypot(d.x, d.y)+d.r > oilRadius*1.6 {
				t.Fatal("splash grew too far")
			}
		}
		for y := -24.0; y <= 24; y += surfaceStep {
			for x := -24.0; x <= 24; x += surfaceStep {
				if stateAt(&g, x, y) != oil {
					t.Fatalf("seed %d: hole in the thick center", seed)
				}
			}
		}
		packet := g.payload()
		rev := g.revision
		g.addOilSplash(0, 0, oilRadius, seed, 1)
		if g.revision != rev || !bytes.Equal(packet, g.payload()) {
			t.Fatal("splash changed shape with time")
		}
		again := oilSurface{}
		again.addOilSplash(0, 0, oilRadius, seed, 0)
		if !bytes.Equal(packet, again.payload()) {
			t.Fatal("same seed is not deterministic")
		}
		if seed > 0 && bytes.Equal(packet, previous) {
			t.Fatal("different bottles reuse the same contour")
		}
		previous = packet
	}
}
func TestSplashUsesSharedFireAndCollisionCoverage(t *testing.T) {
	g := oilSurface{}
	g.addOilSplash(0, 0, 54, 37, 0)
	shape := makeOilSplash(54, 37)
	for _, d := range shape.drops {
		body := unit.Snapshot{X: d.x, Y: d.y, Radius: 2}
		if !g.touches(body, oil) {
			t.Fatal("visible droplets have no oil contact")
		}
	}
	// Igniting the main puddle must advance through that same irregular mask.
	g.addFire(0, 0, 6, 0)
	g.advance(1)
	if stateAt(&g, 0, 0) != flame {
		t.Fatal("splash did not ignite")
	}
	if countState(&g, oil) == 0 {
		t.Fatal("detached splashes must not all ignite instantly")
	}
	g.advance(5)
	if stateAt(&g, 0, 0) != dry {
		t.Fatal("burned puddle returned to oil")
	}
}
func TestBottleBreakSoundIsEmittedExactlyOnce(t *testing.T) {
	for _, fire := range []bool{false, true} {
		b := newBottle(fire)(unit.SpawnInfo{OwnerID: 1}).(*bottle)
		out := make(chan unit.Cmd, 16)
		ctx := unit.Context{ID: 44, Kind: KindArsonOil, Out: out}
		b.Handle(ctx, unit.WallHit{X: 30, Y: 50})
		b.Handle(ctx, unit.WallHit{X: 30, Y: 50})
		count := 0
		for _, c := range drain(out) {
			if fx, ok := c.(unit.FX); ok {
				count++
				want := "arson-oil-break"
				if fire {
					want = "arson-fire-break"
				}
				if fx.Name != want || fx.UnitID != 44 || fx.X != 30 || fx.Y != 50 {
					t.Fatalf("wrong sound event: %+v", fx)
				}
			}
		}
		if count != 1 {
			t.Fatalf("got %d break sounds", count)
		}
	}
}
