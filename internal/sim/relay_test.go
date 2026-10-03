package sim

import (
	"testing"
	"time"

	"xqdj/character"
)

func TestRelayLeavesNodeThenShot(t *testing.T) {
	m := NewMatchSeeded(3)
	m.SetSlot(0, character.KindRelay)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	for i := 0; i < 3; i++ {
		m.Tick()
		time.Sleep(time.Millisecond)
	}
	if n := countKind(m, character.KindRelayNode); n != 1 {
		t.Fatalf("opening nodes=%d", n)
	}
	for i := 0; i < 130; i++ {
		m.Tick()
		time.Sleep(time.Millisecond)
	}
	if n := countKind(m, character.KindRelayNode); n < 2 {
		t.Fatalf("nodes=%d", n)
	}
	seen := len(unitsOf(m, character.KindRelayShot)) > 0
	for i := 0; i < 8 && !seen; i++ {
		m.Tick()
		time.Sleep(time.Millisecond)
		if len(unitsOf(m, character.KindRelayShot)) > 0 {
			seen = true
		}
	}
	if !seen {
		t.Fatal("second node should start a pass")
	}
	m.End()
}

func countKind(m *Match, kind string) int {
	return len(unitsOf(m, kind))
}

func unitsOf(m *Match, kind string) []*unit {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*unit
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && !u.stopped && u.kind == kind {
			out = append(out, u)
		}
	}
	return out
}
