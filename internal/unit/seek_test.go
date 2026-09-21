package unit

import "testing"

func TestDefaultAimPriority(t *testing.T) {
	if g := DefaultAimPriority(Spec{Role: RoleFighter}); g != DefaultFighterAim {
		t.Fatalf("fighter=%d", g)
	}
	if g := DefaultAimPriority(Spec{Mortal: true}); g != DefaultMortalAim {
		t.Fatalf("mortal=%d", g)
	}
	if g := DefaultAimPriority(Spec{Role: RoleProjectile}); g != 0 {
		t.Fatalf("proj=%d", g)
	}
	if g := DefaultAimPriority(Spec{Role: RoleFighter, AimPriority: 3}); g != 3 {
		t.Fatalf("written=%d", g)
	}
}

func TestSeekPrefersLowerAimPriority(t *testing.T) {
	s := Sense{
		Self: Snapshot{ID: 1, Slot: 0, X: 0, Y: 0},
		Nearby: []Snapshot{
			{ID: 2, Slot: 1, X: 10, Y: 0, AimPriority: 60},
			{ID: 3, Slot: 1, X: 100, Y: 0, AimPriority: 15},
		},
	}
	got := Seek(s)
	if got == nil || got.ID != 3 {
		t.Fatalf("got %#v", got)
	}
}

func TestSeekSamePriorityPicksNearest(t *testing.T) {
	s := Sense{
		Self: Snapshot{ID: 1, Slot: 0, X: 0, Y: 0},
		Nearby: []Snapshot{
			{ID: 2, Slot: 1, X: 80, Y: 0, AimPriority: 60},
			{ID: 3, Slot: 1, X: 20, Y: 0, AimPriority: 60},
		},
	}
	got := Seek(s)
	if got == nil || got.ID != 3 {
		t.Fatalf("got %#v", got)
	}
}

func TestSeekSkipsZeroAndSameSlot(t *testing.T) {
	s := Sense{
		Self: Snapshot{ID: 1, Slot: 0, X: 0, Y: 0},
		Nearby: []Snapshot{
			{ID: 2, Slot: 0, X: 5, Y: 0, AimPriority: 1},
			{ID: 3, Slot: 1, X: 8, Y: 0, AimPriority: 0},
			{ID: 4, Slot: 1, X: 40, Y: 0, AimPriority: 15},
		},
	}
	got := Seek(s)
	if got == nil || got.ID != 4 {
		t.Fatalf("got %#v", got)
	}
}
