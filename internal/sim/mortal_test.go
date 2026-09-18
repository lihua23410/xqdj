package sim

import (
	"math"
	"testing"
	"time"
	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestMortalMinionTakesDamageWithoutHitStop(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindGodfather)
	m.SetSlot(1, character.KindMelee)
	m.Start()
	defer m.End()
	time.Sleep(20 * time.Millisecond)
	m.mu.Lock()
	defer m.mu.Unlock()
	boss := fighterByKind(m, character.KindGodfather)
	if boss == nil {
		t.Fatal("missing godfather")
	}
	min := m.addUnitLocked(character.KindAssassin, vec{0, 0}, vec{}, boss.id, 0)
	if min == nil || !min.mortal {
		t.Fatalf("minion=%v", min)
	}
	before := min.hp
	m.offerDamageLocked(unitpkg.Damage{From: boss.id, To: min.id, Amount: 12})
	if math.Abs(min.hp-(before-12)) > 1e-6 {
		t.Fatalf("hp=%v want %v", min.hp, before-12)
	}
	if m.hitStop != 0 {
		t.Fatalf("hitstop=%d", m.hitStop)
	}
}

func TestHealMortalClampsToMaxHP(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindGodfather)
	m.SetSlot(1, character.KindMelee)
	m.Start()
	defer m.End()
	time.Sleep(20 * time.Millisecond)
	m.mu.Lock()
	defer m.mu.Unlock()
	boss := fighterByKind(m, character.KindGodfather)
	min := m.addUnitLocked(character.KindAssassin, vec{0, 0}, vec{}, boss.id, 0)
	min.hp = min.maxHP - 4
	m.applyCmdLocked(unitpkg.Heal{UnitID: min.id, Amount: 10})
	if math.Abs(min.hp-min.maxHP) > 1e-6 {
		t.Fatalf("hp=%v want %v", min.hp, min.maxHP)
	}
}

func TestBreakWallsRemovesPlacedWall(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindGodfather)
	m.SetSlot(1, character.KindMelee)
	m.Start()
	defer m.End()
	time.Sleep(20 * time.Millisecond)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.placeWallLocked(unitpkg.PlaceWall{
		OwnerID: 1, Slot: 1, Kind: character.KindWaller,
		X1: -40, Y1: 0, X2: 40, Y2: 0, Radius: 6, Life: 8, Amount: 1,
	})
	if len(m.walls) != 1 {
		t.Fatalf("walls=%d", len(m.walls))
	}
	id := m.walls[0].id
	m.breakWallLocked(id)
	if len(m.walls) != 0 {
		t.Fatalf("wall still there: %d", len(m.walls))
	}
}
