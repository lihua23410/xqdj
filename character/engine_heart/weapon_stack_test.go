package 泵之心

import (
	"slices"
	"testing"
	modules "xqdj/character/engine_heart/parts"
	"xqdj/internal/unit"
)

func TestWeaponPickupAndRecoveryEnterBottomThroughTheField(t *testing.T) {
	state := &unit.MatchState{}
	out := make(chan unit.Cmd, 256)
	ctx := unit.Context{State: state, ID: 2, Out: out}
	w := worldFor(state, 1)
	w.parts = []part{{Type: 0, X: 100}, {Type: 11, X: 120}, {Type: 1, X: 140}, {Type: 12, X: 160}, {Type: 13, X: 180}}
	f := &floor{owner: 1}
	s := unit.Sense{Field: unit.HexField(), Nearby: []unit.Snapshot{{ID: 1, Role: unit.RoleFighter, Radius: 18, HP: 100}}}
	for i := range w.parts {
		w.parts[i].X, w.parts[i].Y = 0, 0
		s.Time += .01
		f.Handle(ctx, s)
	}
	if want := []int{4, 3, 1, 0, 2}; !slices.Equal(w.stack, want) {
		t.Fatalf("weapons must enter below existing parts: got %v want %v", w.stack, want)
	}
	w.hurt(24, s.Time)
	if !slices.Equal(w.stack, []int{4, 3, 1}) || w.has(0) || w.has(1) {
		t.Fatalf("ordinary parts did not shed before the weapons: %v", w.stack)
	}
	w.hurt(12, s.Time)
	if w.has(11) || !w.has(12) || !w.has(13) {
		t.Fatal("after ordinary parts, the oldest bottom-inserted weapon should fall first")
	}
	// The same dropped weapon returns to the bottom, never to the ordinary top.
	w.parts[1].Flying = false
	w.parts[1].VX, w.parts[1].VY = 0, 0
	w.parts[1].X, w.parts[1].Y = 0, 0
	s.Time += .01
	f.Handle(ctx, s)
	if !slices.Equal(w.stack, []int{1, 4, 3}) || len(w.parts) != 5 {
		t.Fatalf("recovered weapon did not reuse its object at the bottom: %v", w.stack)
	}
}

func TestEveryRegisteredWeaponUsesBottomInsertion(t *testing.T) {
	for _, d := range modules.Catalog() {
		if d.Weapon == nil {
			continue
		}
		t.Run(d.Key, func(t *testing.T) {
			w := &world{parts: []part{{Type: 0, Equipped: true}, {Type: 1, Equipped: true}, {Type: d.ID}}, stack: []int{0, 1}}
			if !w.installPart(2) || !slices.Equal(w.stack, []int{2, 0, 1}) {
				t.Fatal("registered weapon did not enter the bottom")
			}
			if w.installPart(2) || len(w.stack) != 3 {
				t.Fatal("installed weapon was duplicated")
			}
			w.parts = append(w.parts, part{Type: d.ID})
			if w.installPart(3) {
				t.Fatal("a second component of the same weapon type was installed")
			}
		})
	}
}

func TestFlyingWeaponCannotBeAutoInserted(t *testing.T) {
	w := &world{parts: []part{{Type: 11, Flying: true}}}
	if w.installPart(0) || len(w.stack) != 0 || w.parts[0].Equipped {
		t.Fatal("bottom insertion bypassed the flight pickup lock")
	}
}
