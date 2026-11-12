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
	for i := 0; i < 2; i++ {
		a.rings[i].active = true
		a.ensureOmega(i)
		turn := 2 * math.Pi / a.omegas[i]
		if turn < ringTurnMin-1e-9 || turn > ringTurnMax+1e-9 {
			t.Fatalf("ring %d turn %v", i, turn)
		}
	}
	if a.omegas[0] == a.omegas[1] {
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

func TestFirstReleasedVerseStaysUntilTheNext(t *testing.T) {
	a := &老牧师{known: map[uint64]struct{}{7: {}, 8: {}}}
	a.rings[1] = newRing(0)
	a.rings[1].slots[0].id = 7
	a.rings[1].slots[1].id = 8
	a.rings[1].spin = 0
	a.omegas[1] = 2
	out := make(chan unit.Cmd, 16)
	ctx := unit.Context{Out: out}
	a.releaseOuter(ctx, unit.Sense{Self: unit.Snapshot{ID: 1, Slot: 0}})
	for len(out) > 0 {
		c := <-out
		v, ok := c.(unit.SetVelocity)
		if !ok || v.VX != 0 || v.VY != 0 {
			t.Fatalf("first release should hold still: %v", c)
		}
	}
	for _, f := range a.loose {
		if !f.hold {
			t.Fatalf("first verse already flying: %+v", f)
		}
	}
	a.sync(unit.Sense{
		Self: unit.Snapshot{ID: 1},
		Nearby: []unit.Snapshot{
			{ID: 7, Kind: KindGlyph, OwnerID: 1, X: 10, Y: 20},
			{ID: 8, Kind: KindGlyph, OwnerID: 1, X: 0, Y: 0},
		},
	})
	a.flyLoose(ctx, unit.HexField(), 0.5)
	if len(out) != 0 {
		t.Fatalf("held verse moved: %v", <-out)
	}

	a.rings[1] = newRing(1)
	a.rings[1].slots[0].id = 9
	a.known[9] = struct{}{}
	a.releaseOuter(ctx, unit.Sense{Self: unit.Snapshot{ID: 1, Slot: 0}})
	for _, f := range a.loose {
		if !f.hold {
			t.Fatalf("no target should keep the verse parked: %+v", f)
		}
	}
	for len(out) > 0 {
		<-out
	}

	aim := unit.Sense{
		Self: unit.Snapshot{ID: 1, Slot: 0},
		Nearby: []unit.Snapshot{
			{ID: 7, Kind: KindGlyph, OwnerID: 1, X: 10, Y: 20},
			{ID: 8, Kind: KindGlyph, OwnerID: 1, X: 0, Y: 0},
			{ID: 9, Kind: KindGlyph, OwnerID: 1, X: 3, Y: 4},
			{ID: 3, Slot: 1, X: 10, Y: 120, AimPriority: unit.DefaultFighterAim},
		},
	}
	a.sync(aim)
	a.rings[1] = newRing(2)
	a.rings[1].slots[0].id = 11
	a.known[11] = struct{}{}
	a.releaseOuter(ctx, aim)
	got := map[uint64]unit.SetVelocity{}
	for len(out) > 0 {
		c := <-out
		v, ok := c.(unit.SetVelocity)
		if !ok {
			t.Fatalf("cmd %T", c)
		}
		got[v.UnitID] = v
	}
	speed := 2 * ringRadius[1]
	if math.Abs(got[7].VX) > 1e-6 || math.Abs(got[7].VY-speed) > 1e-6 {
		t.Fatalf("glyph 7 should aim straight at the target: %+v", got[7])
	}
	dx, dy := 10.0, 120.0
	nlen := math.Hypot(dx, dy)
	if math.Abs(got[8].VX-dx/nlen*speed) > 1e-6 || math.Abs(got[8].VY-dy/nlen*speed) > 1e-6 {
		t.Fatalf("glyph 8 should aim at the target once: %+v", got[8])
	}
	if got[11].VX != 0 || got[11].VY != 0 {
		t.Fatalf("newer verse should stay: %+v", got[11])
	}
	aim.Nearby = append(aim.Nearby, unit.Snapshot{ID: 11, Kind: KindGlyph, OwnerID: 1, X: 1, Y: 1})
	a.sync(aim)
	a.flyLoose(ctx, unit.HexField(), 0.5)
	moved := map[uint64]unit.Teleport{}
	for len(out) > 0 {
		c := <-out
		tp, ok := c.(unit.Teleport)
		if !ok {
			continue
		}
		moved[tp.UnitID] = tp
	}
	if _, ok := moved[11]; ok {
		t.Fatalf("newer verse flew: %+v", moved[11])
	}
	if math.Abs(moved[7].X-10) > 1e-6 || math.Abs(moved[7].Y-(20+speed*0.5)) > 1e-6 {
		t.Fatalf("older verse did not fly toward the target: %+v", moved[7])
	}
}

func TestOnlyIrrecoverableLooseGlyphsDespawn(t *testing.T) {
	a := &老牧师{
		known: map[uint64]struct{}{7: {}, 8: {}, 9: {}, 10: {}},
		loose: []looseGlyph{
			{id: 7, vx: 100},
			{id: 8, vx: -100},
			{id: 9, vx: 100, hold: true},
			{id: 10, vx: 100},
		},
	}
	a.sync(unit.Sense{
		Self: unit.Snapshot{ID: 1},
		Nearby: []unit.Snapshot{
			{ID: 7, Kind: KindGlyph, OwnerID: 1, X: 1000},
			{ID: 8, Kind: KindGlyph, OwnerID: 1, X: 1000},
			{ID: 9, Kind: KindGlyph, OwnerID: 1, X: 1000},
			{ID: 10, Kind: KindGlyph, OwnerID: 1, X: 900},
		},
	})
	out := make(chan unit.Cmd, 8)
	a.flyLoose(unit.Context{Out: out}, unit.Field{Extent: 850}, 0.5)
	var gone []uint64
	moved := make(map[uint64]float64)
	for len(out) > 0 {
		switch c := (<-out).(type) {
		case unit.Despawn:
			gone = append(gone, c.UnitID)
		case unit.Teleport:
			moved[c.UnitID] = c.X
		}
	}
	if len(gone) != 1 || gone[0] != 7 || len(a.loose) != 3 {
		t.Fatalf("only the distant outbound glyph should despawn: gone=%v loose=%+v", gone, a.loose)
	}
	if len(moved) != 2 || moved[8] != 950 || moved[10] != 950 {
		t.Fatalf("returning and nearer glyphs should keep flying: %v", moved)
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
