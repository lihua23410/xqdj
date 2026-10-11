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

func TestHubAndTrackLayout(t *testing.T) {
	f := unit.HexField()
	for seed := uint64(0); seed < 16; seed++ {
		rng := rand.New(rand.NewPCG(seed+1, seed+99))
		m, ok := buildMaze(f, rng)
		if !ok {
			t.Fatalf("seed %d build fail", seed)
		}
		if len(m.rings) != ringCount || m.guideR <= 0 {
			t.Fatalf("seed %d bad maze n=%d guideR=%v", seed, len(m.rings), m.guideR)
		}
		ext := f.Extent
		if m.guideR > ext*guideFrac+1e-6 || m.guideR < ext*guideFracMin-1e-6 {
			t.Fatalf("seed %d guideR %v out of range", seed, m.guideR)
		}
		// 心环近场心
		if math.Hypot(m.rings[0].x, m.rings[0].y) > ext*0.25 {
			t.Fatalf("seed %d hub too far: %v,%v", seed, m.rings[0].x, m.rings[0].y)
		}
		for i := 1; i < len(m.rings); i++ {
			d := math.Hypot(m.rings[i].x, m.rings[i].y)
			if math.Abs(d-m.guideR) > 1e-3 {
				t.Fatalf("seed %d track %d not on guide: d=%v R=%v", seed, i, d, m.guideR)
			}
		}
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

func TestDriftStaysInLeashAndWalkable(t *testing.T) {
	f := unit.HexField()
	rng := rand.New(rand.NewPCG(7, 11))
	m, ok := buildMaze(f, rng)
	if !ok {
		t.Fatal("build maze")
	}
	drifts := initDrifts(m, rng, 0)
	hubAx, hubAy := drifts[0].ax, drifts[0].ay
	now := 0.0
	dt := 1.0 / 60
	for step := 0; step < 60*20; step++ {
		now += dt
		driftCenters(m.rings, drifts, f, rng, now, dt)
		// 心环锚点钉死
		if math.Hypot(drifts[0].ax-hubAx, drifts[0].ay-hubAy) > 1e-6 {
			t.Fatalf("hub anchor moved")
		}
		for i, r := range m.rings {
			d := math.Hypot(r.x-drifts[i].ax, r.y-drifts[i].ay)
			if d > driftLeash+1e-3 {
				t.Fatalf("ring %d outside leash: %v", i, d)
			}
			if !f.Walkable(r.x, r.y, centerPad) {
				t.Fatalf("ring %d center not walkable at %v,%v", i, r.x, r.y)
			}
			if drifts[i].track {
				ad := math.Hypot(drifts[i].ax, drifts[i].ay)
				if math.Abs(ad-m.guideR) > 1e-3 {
					t.Fatalf("track %d anchor left guide: %v vs %v", i, ad, m.guideR)
				}
			}
		}
	}
}

func TestDriftRecomputesJunctions(t *testing.T) {
	f := unit.HexField()
	rings := []ring{
		{x: 0, y: 0, r: 100},
		{x: 120, y: 0, r: 100},
	}
	junc := fieldJunctions(f, rings)
	if len(junc) == 0 {
		t.Fatal("expected junctions before drift")
	}
	rings[1].x = 400
	junc = fieldJunctions(f, rings)
	if len(junc) != 0 {
		t.Fatalf("want no junctions after separate, got %d", len(junc))
	}
}

func TestZoneSampleInsideAndRim(t *testing.T) {
	// 环内：采样点就是敌人位置
	px, py := zoneSample(0, 0, 100, 30, 40, 30, 40, 50)
	if px != 30 || py != 40 {
		t.Fatalf("inside sample %v %v", px, py)
	}
	// 环外蹭边：投影到圆周
	px, py = zoneSample(0, 0, 100, 200, 0, 200, 0, 200)
	if math.Abs(px-100) > 1e-6 || math.Abs(py) > 1e-6 {
		t.Fatalf("rim sample %v %v", px, py)
	}
}

func TestTryJunctionUsesPositionNotIndex(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	a := &迷走{
		ring:     0,
		spin:     1,
		rng:      rng,
		hasLastJ: true,
		lastJX:   60,
		lastJY:   80,
		maze: maze{
			rings: []ring{{0, 0, 100}, {120, 0, 100}},
			junc:  []junction{{x: 60, y: 80, a: 0, b: 1}},
		},
	}
	s := unit.Sense{Time: 10}
	if _, _, ok := a.tryJunction(s, 60, 80); ok {
		t.Fatal("same position should be suppressed")
	}
}
