package sim

import (
	"math"
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestCatClimbsOutOfTheCenter(t *testing.T) {
	m := NewMatchSeeded(61)
	m.SetSlot(0, character.KindHajimi)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	t.Cleanup(m.End)
	time.Sleep(20 * time.Millisecond)

	var cat, foe *unit
	m.mu.Lock()
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.role != unitpkg.RoleFighter {
			continue
		}
		if u.slot == 0 {
			cat = u
		} else {
			foe = u
		}
	}
	if cat == nil || foe == nil {
		m.mu.Unlock()
		t.Fatal("missing fighters")
	}
	foe.p = vec{-220, 180}
	foe.setVel(vec{})
	foe.stand = true
	foe.cruise = 0
	cat.p = vec{140, 0}
	cat.setVel(vec{0, 178})
	cat.cruise = 178
	m.mu.Unlock()

	maxR := 0.0
	for i := 0; i < 32; i++ {
		runArcher(m, 0.25)
		if r := radiusOf(m, cat); r > maxR {
			maxR = r
		}
	}
	if maxR < 165 || maxR > 260 {
		t.Fatalf("猫该慢慢离开场心，最远到 %v", maxR)
	}
}

func radiusOf(m *Match, u *unit) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return math.Hypot(u.p.X, u.p.Y)
}
