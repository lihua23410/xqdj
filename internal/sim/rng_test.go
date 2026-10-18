package sim

import "testing"

func TestSameSeedSameSpawn(t *testing.T) {
	a := NewMatchSeeded(42)
	b := NewMatchSeeded(42)
	a.Start()
	b.Start()
	defer a.End()
	defer b.End()
	pa := fighterPlace(a)
	pb := fighterPlace(b)
	if len(pa) != len(pb) || len(pa) < 2 {
		t.Fatalf("fighters %v vs %v", pa, pb)
	}
	for i := range pa {
		if pa[i] != pb[i] {
			t.Fatalf("spawn %d %v vs %v", i, pa[i], pb[i])
		}
	}
}

type place struct {
	kind   string
	x, y   float64
	vx, vy float64
}

func fighterPlace(m *Match) []place {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []place
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.role != "fighter" {
			continue
		}
		out = append(out, place{u.kind, u.p.X, u.p.Y, u.v.X, u.v.Y})
	}
	return out
}
