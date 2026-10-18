package sim

import (
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestHajimiStillThereAfterASecond(t *testing.T) {
	m := NewMatchSeeded(90)
	m.SetSlot(0, character.KindHajimi)
	m.SetSlot(1, "原型机_整合")
	m.Start()
	t.Cleanup(m.End)
	time.Sleep(20 * time.Millisecond)
	runArcher(m, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	var cat *unit
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.slot == 0 && u.role == unitpkg.RoleFighter && !u.stopped {
			cat = u
		}
	}
	if cat == nil {
		t.Fatal("猫不见了")
	}
}
