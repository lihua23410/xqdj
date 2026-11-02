package sim

import (
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestOfferDamageCarriesFanSpan(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindWolf)
	m.SetSlot(1, character.KindMelee)
	m.Start()
	defer m.End()
	time.Sleep(20 * time.Millisecond)

	m.mu.Lock()
	defer m.mu.Unlock()
	wolf := fighterByKind(m, character.KindWolf)
	foe := fighterByKind(m, character.KindMelee)
	if wolf == nil || foe == nil {
		t.Fatal("missing fighters")
	}
	arc := m.addUnitLocked(character.KindWolfBite, vec{}, vec{}, wolf.id, 0)
	if arc == nil || arc.arcSpan == 0 {
		t.Fatalf("bite arc span=%v", arc)
	}
	got := tapEvents(wolf)
	m.offerDamageLocked(unitpkg.Damage{From: arc.id, To: wolf.id, Amount: 10})
	m.offerDamageLocked(unitpkg.Damage{From: foe.id, To: wolf.id, Amount: 8})
	hits := eventsOf[unitpkg.IncomingDamage](got)
	if len(hits) != 2 {
		t.Fatalf("hits=%d", len(hits))
	}
	if !hits[0].FromFan() || hits[0].ArcSpan != arc.arcSpan || hits[0].Amount != 10 {
		t.Fatalf("fan hit=%+v span=%v", hits[0], arc.arcSpan)
	}
	if hits[0].SourceOwnerID != wolf.id {
		t.Fatalf("source owner=%d want %d", hits[0].SourceOwnerID, wolf.id)
	}
	if hits[1].FromFan() || hits[1].Amount != 8 {
		t.Fatalf("body hit=%+v", hits[1])
	}
}
