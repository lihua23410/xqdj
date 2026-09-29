package 老牧师

import (
	"math"
	"math/rand/v2"
	"testing"

	"xqdj/internal/unit"
)

func TestSyncLeavesExtraSlotsWhenGlyphsAreMissing(t *testing.T) {
	a := &老牧师{known: map[uint64]struct{}{}}
	a.rings[0] = newRing(0)
	a.sync(unit.Sense{
		Self: unit.Snapshot{ID: 1},
		Nearby: []unit.Snapshot{
			{ID: 10, Kind: KindGlyph, OwnerID: 1},
			{ID: 11, Kind: KindGlyph, OwnerID: 1},
			{ID: 12, Kind: KindGlyph, OwnerID: 1},
		},
	})
	n := 0
	for _, sl := range a.rings[0].slots {
		if sl.id != 0 {
			n++
		}
	}
	if n != 3 || !a.rings[0].slots[3].live {
		t.Fatalf("assigned %d, last live %v", n, a.rings[0].slots[3].live)
	}
}

func TestEachRingRollsItsOwnTurn(t *testing.T) {
	a := &老牧师{rng: rand.New(rand.NewPCG(3, 9))}
	for i := 0; i < 3; i++ {
		a.rings[i].active = true
		a.ensureOmega(i)
		turn := 2 * math.Pi / a.omegas[i]
		if turn < ringTurnMin-1e-9 || turn > ringTurnMax+1e-9 {
			t.Fatalf("ring %d turn %v", i, turn)
		}
	}
	if a.omegas[0] == a.omegas[1] || a.omegas[1] == a.omegas[2] || a.omegas[0] == a.omegas[2] {
		t.Fatalf("omegas %v", a.omegas)
	}
	kept := a.omegas[0]
	a.ensureOmega(0)
	if a.omegas[0] != kept {
		t.Fatal("rerolled")
	}
}

func TestRingSpinUsesItsOwnOmega(t *testing.T) {
	a := &老牧师{haveT: true, lastT: 1}
	a.rings[0].active = true
	a.rings[1].active = true
	a.omegas[0] = 2
	a.omegas[1] = 4
	a.advance(1.5)
	if math.Abs(a.rings[0].spin-1) > 1e-9 || math.Abs(a.rings[1].spin-2) > 1e-9 {
		t.Fatalf("spin %v %v", a.rings[0].spin, a.rings[1].spin)
	}
}

func TestGlyphVanishesOnlyAfterHPLoss(t *testing.T) {
	still := unit.Sense{Nearby: []unit.Snapshot{{ID: 7, HP: 40}}}
	if glyphConfirmed(still, []pendingHit{{id: 7, hp: 40}}) {
		t.Fatal("unchanged hp should keep the glyph")
	}
	hurt := unit.Sense{Nearby: []unit.Snapshot{{ID: 7, HP: 38}}}
	if !glyphConfirmed(hurt, []pendingHit{{id: 7, hp: 40}}) {
		t.Fatal("lost hp should remove the glyph")
	}
	if !glyphConfirmed(unit.Sense{}, []pendingHit{{id: 7, hp: 40}}) {
		t.Fatal("missing target should remove the glyph")
	}
}
