package sim

import (
	"testing"
	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestFlickerDamageSourceSurvivesProjectileRemoval(t *testing.T) {
	m := NewHeadlessMatchSeeded(2077)
	m.SetSlot(0, character.KindFlicker)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	f := fighterSlot(m, 0)
	enemy := fighterSlot(m, 1)
	got := tapEvents(f)
	bullet := m.addUnitLocked("子弹", vec{}, vec{}, enemy.id, 1)
	if bullet == nil {
		t.Fatal("missing projectile")
	}
	m.offerDamageLocked(unitpkg.Damage{From: bullet.id, To: f.id, Amount: 10})
	m.removeLocked(bullet)
	hits := eventsOf[unitpkg.IncomingDamage](got)
	if len(hits) != 1 || hits[0].SourceRole != unitpkg.RoleProjectile || hits[0].SourceKind != "子弹" || hits[0].SourceOwnerID != enemy.id {
		t.Fatalf("source metadata=%+v", hits)
	}
}

func TestFlickerCanDamageWithoutEnemyProjectiles(t *testing.T) {
	m := NewHeadlessMatchSeeded(2077)
	m.SetSlot(0, character.KindFlicker)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()
	for i := 0; i < 60*12; i++ {
		m.Tick()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	target := fighterSlot(m, 1)
	if target != nil && target.hp >= target.maxHP {
		t.Fatal("flicker never attacks a non-shooting enemy")
	}
}
