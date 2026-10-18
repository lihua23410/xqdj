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
	if sa, sb := fighterStreams(a), fighterStreams(b); len(sa) != len(sb) || len(sa) < 2 {
		t.Fatalf("streams %v vs %v", sa, sb)
	} else {
		for i := range sa {
			if sa[i] != sb[i] {
				t.Fatalf("stream %d %d vs %d", i, sa[i], sb[i])
			}
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

func fighterStreams(m *Match) []uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []uint64
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.role != "fighter" {
			continue
		}
		out = append(out, u.stream)
	}
	return out
}
