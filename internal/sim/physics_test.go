package sim

import (
	"math"
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

func TestSweptPointVsCapsuleHeadOn(t *testing.T) {
	t0, n, ok := sweptPointVsCapsule(vec{-20, 0}, vec{10, 0}, 2, vec{0, -100}, vec{0, 100}, 6)
	if !ok {
		t.Fatal("expected hit")
	}
	if t0 < 1.3 || t0 > 1.5 {
		t.Fatalf("toi=%v", t0)
	}
	if n.X < 0.9 {
		t.Fatalf("normal %+v", n)
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

func TestSweptPairSemiHitsFlatFace(t *testing.T) {
	// Semi at origin, bulge +x. Circle coming from the empty side (-x).
	t0, n, ok := sweptPairShapes(
		ccdShape{p: vec{0, 0}, v: vec{0, 0}, r: 18, face: vec{1, 0}, semi: true},
		ccdShape{p: vec{-50, 0}, v: vec{20, 0}, r: 18, face: vec{1, 0}},
		4,
	)
	if !ok {
		t.Fatal("expected hit on flat face")
	}
	if t0 < 1.5 || t0 > 1.7 {
		t.Fatalf("toi=%v", t0)
	}
	if n.X < 0.9 {
		t.Fatalf("normal %+v want from circle toward semi (+x)", n)
	}
}

func TestSweptPairComplementarySemisSeparate(t *testing.T) {
	// Two halves of one circle, already moving apart.
	_, _, ok := sweptPairShapes(
		ccdShape{p: vec{0, 0}, v: vec{40, 0}, r: 18, face: vec{1, 0}, semi: true},
		ccdShape{p: vec{-2, 0}, v: vec{-40, 0}, r: 18, face: vec{-1, 0}, semi: true},
		0.05,
	)
	if ok {
		t.Fatal("splitting halves should not collide")
	}
}

func TestSweptArcHitsFront(t *testing.T) {
	arc := ccdShape{
		p: vec{0, 0}, v: vec{0, 0}, face: vec{1, 0},
		r: 20, arcInner: 18, arcSpan: math.Pi / 2,
	}
	ball := ccdShape{p: vec{50, 0}, v: vec{-20, 0}, r: 18}
	t0, n, ok := sweptPairShapes(arc, ball, 2)
	if !ok {
		t.Fatal("expected frontal arc hit")
	}
	if t0 < 0.5 || t0 > 0.7 {
		t.Fatalf("toi=%v", t0)
	}
	if n.X > -0.9 {
		t.Fatalf("normal %+v want toward ball (+x from arc, so n from ball to arc is -x)", n)
	}
}

func TestSweptArcMissesBack(t *testing.T) {
	arc := ccdShape{
		p: vec{0, 0}, v: vec{0, 0}, face: vec{1, 0},
		r: 20, arcInner: 18, arcSpan: math.Pi / 2,
	}
	ball := ccdShape{p: vec{-50, 0}, v: vec{20, 0}, r: 18}
	if _, _, ok := sweptPairShapes(arc, ball, 2); ok {
		t.Fatal("90° arc should not hit from behind")
	}
}

func TestSweptArcFullRingHitsBack(t *testing.T) {
	arc := ccdShape{
		p: vec{0, 0}, v: vec{0, 0}, face: vec{1, 0},
		r: 20, arcInner: 18, arcSpan: 2 * math.Pi,
	}
	ball := ccdShape{p: vec{-50, 0}, v: vec{20, 0}, r: 18}
	t0, _, ok := sweptPairShapes(arc, ball, 2)
	if !ok {
		t.Fatal("360° ring should hit from behind")
	}
	if t0 < 0.5 || t0 > 0.7 {
		t.Fatalf("toi=%v", t0)
	}
}

func TestSweptArcIgnoresSemiEmptyHalf(t *testing.T) {
	arc := ccdShape{
		p: vec{-30, 0}, v: vec{0, 0}, face: vec{1, 0},
		r: 20, arcInner: 18, arcSpan: 2 * math.Pi,
	}
	semi := ccdShape{p: vec{0, 0}, v: vec{0, 0}, face: vec{1, 0}, r: 18, semi: true}
	if _, _, ok := sweptPairShapes(arc, semi, 0.05); ok {
		t.Fatal("arc in the empty half of a semi should not hit a full disk")
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
			if ua.passWalls || ua.attach {
				continue
			}
			if ua.semi {
				if !m.hex.containsSemi(ua.p, ua.face, ua.radius) {
					m.mu.Unlock()
					t.Fatalf("unit %d escaped hex at t=%v p=%+v", ua.id, m.time, ua.p)
				}
			} else if !m.hex.containsCenter(ua.p, ua.radius) {
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
				if ua.passWalls || ub.passWalls || ua.attach || ub.attach {
					continue
				}
				if !ua.solid || !ub.solid {
					continue
				}
				ca, ra := colOf(ua.p, ua.face, ua.radius, ua.semi)
				cb, rb := colOf(ub.p, ub.face, ub.radius, ub.semi)
				d := ca.sub(cb).len()
				if d+1e-3 < ra+rb {
					m.mu.Unlock()
					t.Fatalf("overlap %d/%d dist=%v rsum=%v", ua.id, ub.id, d, ra+rb)
				}
			}
		}
		m.mu.Unlock()
	}
}
