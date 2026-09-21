package sim

import (
	"testing"
	"time"
	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestAimPriorityDefaults(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindGodfather)
	m.SetSlot(1, character.KindMelee)
	m.Start()
	defer m.End()
	time.Sleep(20 * time.Millisecond)
	m.mu.Lock()
	defer m.mu.Unlock()
	boss := fighterByKind(m, character.KindGodfather)
	melee := fighterByKind(m, character.KindMelee)
	if boss == nil || melee == nil {
		t.Fatal("missing fighters")
	}
	if boss.aimPriority != unitpkg.DefaultFighterAim {
		t.Fatalf("fighter aim=%d", boss.aimPriority)
	}
	min := m.addUnitLocked(character.KindAssassin, vec{0, 0}, vec{}, boss.id, 0)
	if min == nil || min.aimPriority != unitpkg.DefaultMortalAim {
		t.Fatalf("minion aim=%v", min)
	}
	nail := m.addUnitLocked(character.KindNail, vec{10, 0}, vec{}, melee.id, 1)
	if nail != nil && nail.aimPriority != 0 {
		t.Fatalf("proj aim=%d", nail.aimPriority)
	}
}

func TestSetAimPriorityAnyoneWhenArmed(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindGodfather)
	m.SetSlot(1, character.KindMelee)
	m.Start()
	defer m.End()
	time.Sleep(20 * time.Millisecond)
	m.mu.Lock()
	defer m.mu.Unlock()
	boss := fighterByKind(m, character.KindGodfather)
	melee := fighterByKind(m, character.KindMelee)
	m.applyCmdLocked(unitpkg.SetAimPriority{From: melee.id, UnitID: boss.id, Value: 1})
	if boss.aimPriority != 1 {
		t.Fatalf("after enemy set=%d", boss.aimPriority)
	}
	m.applyCmdLocked(unitpkg.SetAimPriority{From: melee.id, UnitID: boss.id, Value: 0})
	if boss.aimPriority != 0 {
		t.Fatalf("stripped=%d", boss.aimPriority)
	}
	m.applyCmdLocked(unitpkg.SetAimPriority{From: melee.id, UnitID: boss.id, Value: 15})
	if boss.aimPriority != 0 {
		t.Fatalf("enemy restored=%d", boss.aimPriority)
	}
	m.applyCmdLocked(unitpkg.SetAimPriority{From: boss.id, UnitID: boss.id, Value: 15})
	if boss.aimPriority != 15 {
		t.Fatalf("self restored=%d", boss.aimPriority)
	}
}

func TestSetAimPriorityRejectsProjectile(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindGodfather)
	m.SetSlot(1, character.KindMelee)
	m.Start()
	defer m.End()
	time.Sleep(20 * time.Millisecond)
	m.mu.Lock()
	defer m.mu.Unlock()
	melee := fighterByKind(m, character.KindMelee)
	nail := m.addUnitLocked(character.KindNail, vec{10, 0}, vec{}, melee.id, 1)
	if nail == nil {
		t.Fatal("missing nail")
	}
	m.applyCmdLocked(unitpkg.SetAimPriority{From: nail.id, UnitID: nail.id, Value: 15})
	if nail.aimPriority != 0 {
		t.Fatalf("self on proj=%d", nail.aimPriority)
	}
}
