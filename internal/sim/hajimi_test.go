package sim

import (
	"math"
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestStationHoldsAndDropsWhenSlowed(t *testing.T) {
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

	runArcher(m, 0.8)
	r0 := radiusOf(m, cat)
	m.mu.Lock()
	cat.setVel(cat.v.mul(0.45))
	cat.cruise = cat.v.len()
	m.mu.Unlock()
	runArcher(m, 0.7)
	r1 := radiusOf(m, cat)
	if r0 < 90 {
		t.Fatalf("没减速时不该掉出轨道，半径 %v", r0)
	}
	if r1 > r0-12 {
		t.Fatalf("减速后轨道该往下掉: %v → %v", r0, r1)
	}
}

func radiusOf(m *Match, u *unit) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return math.Hypot(u.p.X, u.p.Y)
}
