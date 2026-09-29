package sim

import (
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestCharcoalIncomingDoesNotSpendHP(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindCharcoal)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()
	waitTicks(m, 2)

	m.mu.Lock()
	self := fighterByKind(m, character.KindCharcoal)
	other := fighterByKind(m, character.KindDummy)
	if self == nil || other == nil {
		m.mu.Unlock()
		t.Fatal("missing fighters")
	}
	hp := self.hp
	id := self.id
	m.applyCmdLocked(unitpkg.Damage{From: other.id, To: id, Amount: 10})
	m.mu.Unlock()
	time.Sleep(20 * time.Millisecond)
	m.mu.Lock()
	m.settleHitsLocked()
	u := m.units[id]
	if u == nil || u.hp != hp {
		got := 0.0
		if u != nil {
			got = u.hp
		}
		m.mu.Unlock()
		t.Fatalf("hp=%v want %v", got, hp)
	}
	m.mu.Unlock()
}

func TestCharcoalBounceHandsOffCoal(t *testing.T) {
	m := NewMatchSeeded(3)
	m.SetSlot(0, character.KindCharcoal)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()
	waitTicks(m, 2)

	for {
		m.mu.Lock()
		self := fighterByKind(m, character.KindCharcoal)
		other := fighterByKind(m, character.KindDummy)
		if self == nil || other == nil {
			m.mu.Unlock()
			t.Fatal("missing fighters")
		}
		self.p = vec{-160, 0}
		other.p = vec{160, 0}
		self.setVel(vec{})
		other.setVel(vec{})
		now := m.time
		m.mu.Unlock()
		if now >= 2.5 {
			break
		}
		m.Tick()
		time.Sleep(2 * time.Millisecond)
	}

	m.mu.Lock()
	self := fighterByKind(m, character.KindCharcoal)
	other := fighterByKind(m, character.KindDummy)
	self.p = vec{0, 0}
	other.p = vec{40, 0}
	self.setVel(vec{240, 0})
	other.setVel(vec{-165, 0})
	m.mu.Unlock()
	waitTicks(m, 6)

	for {
		m.mu.Lock()
		now := m.time
		m.mu.Unlock()
		if now >= 14.7 {
			break
		}
		m.Tick()
		time.Sleep(2 * time.Millisecond)
	}

	m.mu.Lock()
	self = fighterByKind(m, character.KindCharcoal)
	other = fighterByKind(m, character.KindDummy)
	shp, ohp := self.hp, other.hp
	m.mu.Unlock()
	if ohp > 99999-5 {
		t.Fatalf("dummy hp=%v charcoal hp=%v; the bounce did not hand off the coal", ohp, shp)
	}
}

func TestNoFreezeSkipsHitStop(t *testing.T) {
	m := NewMatchSeeded(2)
	m.SetSlot(0, character.KindDummy)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()
	waitTicks(m, 1)

	m.mu.Lock()
	var ids []uint64
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.role == unitpkg.RoleFighter {
			ids = append(ids, id)
		}
	}
	if len(ids) < 2 {
		m.mu.Unlock()
		t.Fatal("need two fighters")
	}
	m.applyCmdLocked(unitpkg.Damage{From: ids[0], To: ids[1], Amount: 1, NoFreeze: true})
	m.mu.Unlock()
	time.Sleep(20 * time.Millisecond)
	m.mu.Lock()
	m.settleHitsLocked()
	stopped := m.hitStop
	m.mu.Unlock()
	if stopped != 0 {
		t.Fatalf("hitStop=%d", stopped)
	}
}
