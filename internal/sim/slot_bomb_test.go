package sim

import (
	"math"
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestSlotBombWallBlastHitsNearby(t *testing.T) {
	m, owner := quietSlotMatch(t)
	m.mu.Lock()
	bystander := fighterSlot(m, 1)
	parkFighter(m, owner, vec{0, 200})
	parkFighter(m, bystander, vec{-40, 80})
	before := bystander.hp
	m.applyCmdLocked(unitpkg.PlaceWall{
		OwnerID: owner.id, Slot: owner.slot, Kind: "test",
		X1: 0, Y1: -120, X2: 0, Y2: 120, Radius: 8, Life: -1, Hard: true, Square: true,
	})
	shot := m.addUnitLocked(character.KindSlotBombShot, vec{-140, 0}, vec{380, 0}, owner.id, owner.slot)
	if shot == nil || shot.vision < 100 {
		vision := 0.0
		if shot != nil {
			vision = shot.vision
		}
		m.mu.Unlock()
		t.Fatalf("bomb shot vision=%v, nearby stays empty", vision)
	}
	m.mu.Unlock()

	drop := waitHPDrop(t, m, bystander, before)
	if drop <= 0 {
		t.Fatal("wall blast dealt no damage to a target inside the ring")
	}
}

func TestSlotBombDirectHitAlsoSplashes(t *testing.T) {
	m, owner := quietSlotMatch(t)
	m.mu.Lock()
	primary := fighterSlot(m, 1)
	parkFighter(m, owner, vec{0, 200})
	parkFighter(m, primary, vec{40, 0})
	splash := m.addUnitLocked(character.KindDummy, vec{40, 80}, vec{}, owner.id, primary.slot)
	if splash == nil {
		m.mu.Unlock()
		t.Fatal("missing splash target")
	}
	parkFighter(m, splash, vec{40, 80})
	beforePrimary, beforeSplash := primary.hp, splash.hp
	shot := m.addUnitLocked(character.KindSlotBombShot, vec{-120, 0}, vec{380, 0}, owner.id, owner.slot)
	if shot == nil {
		m.mu.Unlock()
		t.Fatal("bomb shot did not spawn")
	}
	m.mu.Unlock()

	primaryDrop := waitHPDrop(t, m, primary, beforePrimary)
	splashDrop := waitHPDrop(t, m, splash, beforeSplash)
	if primaryDrop <= 0 || splashDrop <= 0 {
		t.Fatalf("direct=%v splash=%v, both the impact and the ring should take damage", primaryDrop, splashDrop)
	}
	if math.Abs(primaryDrop-splashDrop) > 1e-6 {
		t.Fatalf("direct=%v splash=%v, splash should match the direct hit", primaryDrop, splashDrop)
	}
}

func quietSlotMatch(t *testing.T) (*Match, *unit) {
	t.Helper()
	m := NewHeadlessMatchSeeded(11)
	m.skipProbe = true
	m.SetSlot(0, character.KindSlot)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	t.Cleanup(m.End)
	m.mu.Lock()
	defer m.mu.Unlock()
	owner := fighterSlot(m, 0)
	if owner == nil || fighterSlot(m, 1) == nil {
		t.Fatal("missing fighters")
	}
	return m, owner
}

func parkFighter(m *Match, u *unit, p vec) {
	u.p = p
	u.setVel(vec{})
	m.applyCmdLocked(unitpkg.SetCruise{UnitID: u.id, Speed: 0})
}

func waitHPDrop(t *testing.T, m *Match, u *unit, before float64) float64 {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		m.Tick()
		m.mu.Lock()
		drop := before - u.hp
		pending := pendingToward(m, u.id)
		m.mu.Unlock()
		if drop > 1e-6 && pending == 0 {
			return drop
		}
	}
	m.mu.Lock()
	drop := before - u.hp
	m.mu.Unlock()
	return drop
}
