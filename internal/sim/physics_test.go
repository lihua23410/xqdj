package sim

import (
	"testing"

	_ "xqdj/character"
)

func TestReflectIncidentEqualsReflected(t *testing.T) {
	n := vec{1, 0}
	v := vec{1, 1}
	out := reflectVelocity(v, n)
	if out.X > -0.99 || out.X < -1.01 || out.Y < 0.99 || out.Y > 1.01 {
		t.Fatalf("got %+v", out)
	}
}

func TestHexagonContainsCenter(t *testing.T) {
	h := newHexagon(HexRadius)
	if !h.containsCenter(vec{0, 0}, 18) {
		t.Fatal("origin should be inside")
	}
	if h.containsCenter(vec{HexRadius, 0}, 18) {
		t.Fatal("right vertex is outside for a circle of r=18")
	}
}

func TestSweptCirclesHeadOn(t *testing.T) {
	t0, _, ok := sweptCircles(vec{-20, 0}, vec{10, 0}, 5, vec{20, 0}, vec{-10, 0}, 5, 2)
	if !ok {
		t.Fatal("expected hit")
	}
	if t0 < 1.4 || t0 > 1.6 {
		t.Fatalf("toi=%v", t0)
	}
}

func TestMatchNoOverlap(t *testing.T) {
	m := NewMatchSeeded(7)
	m.Start()
	defer m.End()
	for i := 0; i < 180; i++ {
		m.Tick()
		m.mu.Lock()
		ids := append([]uint64(nil), m.order...)
		for a := 0; a < len(ids); a++ {
			ua := m.units[ids[a]]
			if ua == nil {
				continue
			}
			if !m.hex.containsCenter(ua.p, ua.radius) {
				m.mu.Unlock()
				t.Fatalf("unit %d escaped hex at t=%v p=%+v", ua.id, m.time, ua.p)
			}
			for b := a + 1; b < len(ids); b++ {
				ub := m.units[ids[b]]
				if ub == nil {
					continue
				}
				if ua.owner == ub.id || ub.owner == ua.id {
					continue
				}
				if !ua.solid || !ub.solid {
					continue
				}
				d := ua.p.sub(ub.p).len()
				if d+1e-3 < ua.radius+ub.radius {
					m.mu.Unlock()
					t.Fatalf("overlap %d/%d dist=%v rsum=%v", ua.id, ub.id, d, ua.radius+ub.radius)
				}
			}
		}
		m.mu.Unlock()
	}
}
