package sim

import (
	"testing"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestSpawnDamageScaleIsAppliedAtOffer(t *testing.T) {
	m := NewMatchSeeded(1)
	defer m.End()
	m.mu.Lock()
	target := m.addUnitLocked(character.KindDummy, vec{0, 0}, vec{0, 0}, 0, 1)
	shot := m.addUnitLocked(character.KindArrow, vec{-50, 0}, vec{100, 0}, 0, 0)
	if target == nil || shot == nil {
		m.mu.Unlock()
		t.Fatal("failed to create test units")
	}
	shot.damageScale = 0.7
	m.offerDamageLocked(unitpkg.Damage{From: shot.id, To: target.id, Amount: 10})
	got := 0.0
	for _, offer := range m.pendingDmg {
		got = offer.amount
	}
	m.mu.Unlock()
	if got != 7 {
		t.Fatalf("scaled damage=%v, want 7", got)
	}
}
