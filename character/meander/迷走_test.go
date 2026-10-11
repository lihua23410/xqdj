package 迷走

import (
	"math"
	"math/rand/v2"
	"testing"

	"xqdj/internal/unit"
	_ "xqdj/map/circle"
	_ "xqdj/map/hex"
	_ "xqdj/map/octagon"
)

func TestCircleCrossStrict(t *testing.T) {
	a := ring{0, 0, 100}
	b := ring{120, 0, 100}
	p1, p2, ok := circleCross(a, b)
	if !ok {
		t.Fatal("expected cross")
	}
	if math.Abs(p1.y+p2.y) > 1e-6 {
		t.Fatalf("symmetric y: %v %v", p1, p2)
	}
	c := ring{200, 0, 100}
	if _, _, ok := circleCross(a, c); ok {
		t.Fatal("tangent should not count")
	}
}

func TestConnectedMazeFields(t *testing.T) {
	names := []string{"六边形", "圆", "八边形"}
	for _, name := range names {
		f, ok := unit.LookupField(name)
		if !ok {
			// 六边形可能没 RegisterField 名字，用 HexField
			if name == "六边形" {
				f = unit.HexField()
			} else {
				t.Fatalf("missing field %s", name)
			}
		}
		for seed := uint64(0); seed < 10; seed++ {
			rng := rand.New(rand.NewPCG(seed+1, seed+9))
			m, ok := buildMaze(f, rng)
			if !ok || len(m.rings) != ringCount {
				t.Fatalf("%s seed %d fail ok=%v n=%d", name, seed, ok, len(m.rings))
			}
			if !connected(len(m.rings), m.junc) {
				t.Fatalf("%s seed %d not connected junc=%d", name, seed, len(m.junc))
			}
			for _, j := range m.junc {
				if !f.OutlineContains(j.x, j.y, 0) {
					t.Fatalf("%s seed %d junction outside", name, seed)
				}
			}
		}
	}
}

func TestSpreadCentersNotClustered(t *testing.T) {
	f := unit.HexField()
	minPair := math.Inf(1)
	for seed := uint64(0); seed < 16; seed++ {
		rng := rand.New(rand.NewPCG(seed+1, seed+99))
		m, ok := buildMaze(f, rng)
		if !ok {
			t.Fatalf("seed %d build fail", seed)
		}
		for i := 0; i < len(m.rings); i++ {
			for j := i + 1; j < len(m.rings); j++ {
				d := math.Hypot(m.rings[i].x-m.rings[j].x, m.rings[i].y-m.rings[j].y)
				if d < minPair {
					minPair = d
				}
			}
		}
	}
	if minPair < 40 {
		t.Fatalf("centers too clustered: %v", minPair)
	}
}

func TestLegalExitsForbidBacktrack(t *testing.T) {
	a := &迷走{ring: 0, spin: 1}
	j := junction{a: 0, b: 1}
	ex := a.legalExits(j)
	for _, e := range ex {
		if e.ring == 0 && e.spin == -1 {
			t.Fatal("backtrack allowed")
		}
	}
	if len(ex) != 3 {
		t.Fatalf("want 3 exits, got %d", len(ex))
	}
}

func TestTangentOnCircle(t *testing.T) {
	r := ring{0, 0, 100}
	vx, vy := tangentVel(r, 0, 1, 170)
	if math.Abs(vx) > 1e-6 || math.Abs(vy-170) > 1e-6 {
		t.Fatalf("tangent %v %v", vx, vy)
	}
}

func TestBumpOpposesThenFlips(t *testing.T) {
	a := &迷走{
		booted: true,
		ring:   0,
		spin:   1,
		x:      100, y: 0,
		maze: maze{rings: []ring{{0, 0, 100}}},
	}
	// 在 (100,0) 逆时针前进切向是 +Y；法线朝 -Y 为相逆
	a.onBump(unit.Collision{Time: 1, NX: 0, NY: -1, Other: unit.Snapshot{Role: unit.RoleFighter}})
	if a.spin != -1 {
		t.Fatalf("want flip to -1, got %v", a.spin)
	}
}

func TestBumpAlongKeepsSpin(t *testing.T) {
	a := &迷走{
		booted: true,
		ring:   0,
		spin:   1,
		x:      100, y: 0,
		maze: maze{rings: []ring{{0, 0, 100}}},
	}
	a.onBump(unit.Collision{Time: 1, NX: 0, NY: 1, Other: unit.Snapshot{Role: unit.RoleFighter}})
	if a.spin != 1 {
		t.Fatalf("want keep 1, got %v", a.spin)
	}
}

func TestBumpRadialKeepsSpin(t *testing.T) {
	a := &迷走{
		booted: true,
		ring:   0,
		spin:   1,
		x:      100, y: 0,
		maze: maze{rings: []ring{{0, 0, 100}}},
	}
	a.onBump(unit.Collision{Time: 1, NX: 1, NY: 0, Other: unit.Snapshot{Role: unit.RoleFighter}})
	if a.spin != 1 {
		t.Fatalf("want keep 1 on radial, got %v", a.spin)
	}
}
