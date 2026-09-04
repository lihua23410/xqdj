package sim

import (
	"math"
	unitpkg "xqdj/internal/unit"
)

type vec struct {
	X, Y float64
}

func (v vec) add(o vec) vec     { return vec{v.X + o.X, v.Y + o.Y} }
func (v vec) sub(o vec) vec     { return vec{v.X - o.X, v.Y - o.Y} }
func (v vec) mul(s float64) vec { return vec{v.X * s, v.Y * s} }
func (v vec) dot(o vec) float64 { return v.X*o.X + v.Y*o.Y }
func (v vec) len() float64      { return math.Hypot(v.X, v.Y) }
func (v vec) len2() float64     { return v.X*v.X + v.Y*v.Y }

func (v vec) norm() vec {
	l := v.len()
	if l < 1e-12 {
		return vec{1, 0}
	}
	return vec{v.X / l, v.Y / l}
}

const (
	HexRadius     = unitpkg.HexRadius
	TickHz        = 60
	DT            = 1.0 / TickHz
	HitStopFrames = 3
	skin          = 0.08
)

type hexagon struct {
	n [6]vec
	d [6]float64
}

func newHexagon(circum float64) hexagon {
	// Flat-top: vertices at k * 60°. Outward normals = vertex directions at
	// k * 60° + 30° (toward each flat).
	apothem := circum * math.Sqrt(3) / 2
	var h hexagon
	for i := 0; i < 6; i++ {
		a := (float64(i) + 0.5) * math.Pi / 3
		h.n[i] = vec{math.Cos(a), math.Sin(a)}
		h.d[i] = apothem
	}
	return h
}

func (h hexagon) containsCenter(p vec, radius float64) bool {
	limit := h.d[0] - radius
	for i := 0; i < 6; i++ {
		if h.n[i].dot(p) > limit+1e-6 {
			return false
		}
	}
	return true
}

func reflectVelocity(v, outward vec) vec {
	n := outward.norm()
	return v.sub(n.mul(2 * v.dot(n)))
}

type hitKind int

const (
	hitNone hitKind = iota
	hitWall
	hitPair
	hitBarrier
)

type ccdHit struct {
	kind hitKind
	t    float64
	a, b uint64
	w    uint64
	n    vec
}

func perp(v vec) vec { return vec{-v.Y, v.X} }

func closestOnSeg(p, a, b vec) vec {
	ab := b.sub(a)
	ab2 := ab.len2()
	if ab2 < 1e-12 {
		return a
	}
	t := p.sub(a).dot(ab) / ab2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return a.add(ab.mul(t))
}

func sweptPointVsCapsule(p, vel vec, dt float64, a, b vec, R float64) (float64, vec, bool) {
	q := closestOnSeg(p, a, b)
	d := p.sub(q)
	dist := d.len()
	if dist < R-1e-8 {
		n := q.sub(p)
		if n.len2() < 1e-12 {
			n = perp(b.sub(a))
		}
		return 0, n.norm(), true
	}
	bestT := dt + 1
	var bestN vec
	found := false
	consider := func(t float64, at vec) {
		if t < -1e-9 || t > dt {
			return
		}
		if t < 0 {
			t = 0
		}
		qq := closestOnSeg(at, a, b)
		n := qq.sub(at)
		if n.len2() < 1e-12 {
			return
		}
		if t < bestT {
			bestT = t
			bestN = n.norm()
			found = true
		}
	}
	if t, _, ok := sweptCircles(p, vel, 0, a, vec{}, R, dt); ok {
		consider(t, p.add(vel.mul(t)))
	}
	if t, _, ok := sweptCircles(p, vel, 0, b, vec{}, R, dt); ok {
		consider(t, p.add(vel.mul(t)))
	}
	ab := b.sub(a)
	alen := ab.len()
	if alen > 1e-9 {
		u := ab.norm()
		n0 := perp(u)
		for _, s := range []vec{n0, n0.mul(-1)} {
			limit := a.dot(s) + R
			vn := s.dot(vel)
			if vn >= -1e-9 {
				continue
			}
			t := (limit - s.dot(p)) / vn
			if t < -1e-9 || t > dt {
				continue
			}
			if t < 0 {
				t = 0
			}
			at := p.add(vel.mul(t))
			pr := at.sub(a).dot(u)
			if pr < 0 || pr > alen {
				continue
			}
			consider(t, at)
		}
	}
	if !found {
		return 0, vec{}, false
	}
	return bestT, bestN, true
}

func sweptPointVsHex(p, vel vec, radius, dt float64, hex hexagon) (ccdHit, bool) {
	limit := hex.d[0] - radius
	bestT := dt + 1
	var bestN vec
	hit := false
	for i := 0; i < 6; i++ {
		n := hex.n[i]
		vn := n.dot(vel)
		if vn <= 1e-9 {
			continue
		}
		dist := limit - n.dot(p)
		t := dist / vn
		if t < -1e-9 || t > dt {
			continue
		}
		if t < 0 {
			t = 0
		}
		at := p.add(vel.mul(t))
		ok := true
		for j := 0; j < 6; j++ {
			if j == i {
				continue
			}
			if hex.n[j].dot(at) > limit+1e-4 {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		if t < bestT {
			bestT = t
			bestN = n
			hit = true
		}
	}
	if !hit {
		return ccdHit{}, false
	}
	return ccdHit{kind: hitWall, t: bestT, n: bestN}, true
}

func semiExtent(face vec, radius float64, n vec) float64 {
	if face.len2() < 1e-12 {
		return radius
	}
	f := face.norm()
	if n.dot(f) >= -1e-9 {
		return radius
	}
	return radius * math.Abs(perp(f).dot(n))
}

func colOf(p, face vec, r float64, semi bool) (vec, float64) {
	if !semi {
		return p, r
	}
	f := face
	if f.len2() < 1e-12 {
		f = vec{1, 0}
	} else {
		f = f.norm()
	}
	return p.add(f.mul(r * 0.5)), r * 0.5
}

func diameterOf(p, face vec, r float64) (vec, vec) {
	f := face
	if f.len2() < 1e-12 {
		f = vec{1, 0}
	} else {
		f = f.norm()
	}
	q := perp(f).mul(r)
	return p.add(q), p.sub(q)
}

func (h hexagon) containsSemi(p, face vec, radius float64) bool {
	for i := 0; i < 6; i++ {
		ext := semiExtent(face, radius, h.n[i])
		if h.n[i].dot(p) > h.d[0]-ext+1e-6 {
			return false
		}
	}
	return true
}

func sweptShapeVsHex(p, vel, face vec, radius, dt float64, hex hexagon, semi bool) (ccdHit, bool) {
	bestT := dt + 1
	var bestN vec
	hit := false
	for i := 0; i < 6; i++ {
		n := hex.n[i]
		ext := radius
		if semi {
			ext = semiExtent(face, radius, n)
		}
		limit := hex.d[0] - ext
		vn := n.dot(vel)
		if vn <= 1e-9 {
			continue
		}
		dist := limit - n.dot(p)
		t := dist / vn
		if t < -1e-9 || t > dt {
			continue
		}
		if t < 0 {
			t = 0
		}
		at := p.add(vel.mul(t))
		ok := true
		for j := 0; j < 6; j++ {
			if j == i {
				continue
			}
			extj := radius
			if semi {
				extj = semiExtent(face, radius, hex.n[j])
			}
			if hex.n[j].dot(at) > hex.d[0]-extj+1e-4 {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		if t < bestT {
			bestT = t
			bestN = n
			hit = true
		}
	}
	if !hit {
		return ccdHit{}, false
	}
	return ccdHit{kind: hitWall, t: bestT, n: bestN}, true
}

type ccdShape struct {
	p, v, face vec
	r          float64
	semi       bool
	arcSpan    float64
	arcInner   float64
}

func (s ccdShape) isArc() bool { return s.arcSpan > 1e-9 }

func rotateVec(v vec, ang float64) vec {
	c, s := math.Cos(ang), math.Sin(ang)
	return vec{v.X*c - v.Y*s, v.X*s + v.Y*c}
}

func inArcSpan(dir, face vec, half float64) bool {
	if dir.len2() < 1e-16 {
		return true
	}
	if half >= math.Pi-1e-9 {
		return true
	}
	d := dir.norm()
	f := face
	if f.len2() < 1e-12 {
		f = vec{1, 0}
	} else {
		f = f.norm()
	}
	return d.dot(f) >= math.Cos(half)-1e-8
}

func sweptArcVsCircle(arc, ball ccdShape, dt float64) (float64, vec, bool) {
	half := arc.arcSpan / 2
	if half < 0 {
		half = 0
	}
	full := arc.arcSpan >= 2*math.Pi-1e-6
	face := arc.face
	if face.len2() < 1e-12 {
		face = vec{1, 0}
	} else {
		face = face.norm()
	}
	ri, ro := arc.arcInner, arc.r
	if ri < 0 {
		ri = 0
	}
	if ri > ro {
		ri = ro
	}
	bp, br := ball.p, ball.r
	if ball.semi {
		bp, br = colOf(ball.p, ball.face, ball.r, true)
	}
	bestT := dt + 1
	var bestN vec
	found := false
	consider := func(t float64, n vec) {
		if t < -1e-9 || t > dt || n.len2() < 1e-12 {
			return
		}
		if t < 0 {
			t = 0
		}
		if t < bestT {
			bestT = t
			bestN = n.norm()
			found = true
		}
	}
	if t, n, ok := sweptCircles(arc.p, arc.v, ro, bp, ball.v, br, dt); ok {
		ap := arc.p.add(arc.v.mul(t))
		at := bp.add(ball.v.mul(t))
		if full || inArcSpan(at.sub(ap), face, half) {
			consider(t, n)
		}
	}
	if !full {
		relV := ball.v.sub(arc.v)
		R := br + skin
		for _, ang := range []float64{half, -half} {
			u := rotateVec(face, ang)
			a := arc.p.add(u.mul(ri))
			b := arc.p.add(u.mul(ro))
			if t, n, ok := sweptPointVsCapsule(bp, relV, dt, a, b, R); ok {
				consider(t, n.mul(-1))
			}
		}
	}
	if !found {
		return 0, vec{}, false
	}
	return bestT, bestN, true
}

func sweptPairShapes(a, b ccdShape, dt float64) (float64, vec, bool) {
	if a.isArc() && !b.isArc() {
		return sweptArcVsCircle(a, b, dt)
	}
	if b.isArc() && !a.isArc() {
		t, n, ok := sweptArcVsCircle(b, a, dt)
		if !ok {
			return 0, vec{}, false
		}
		return t, n.mul(-1), true
	}
	return sweptPairCircles(a.p, a.v, a.r, a.face, a.semi, b.p, b.v, b.r, b.face, b.semi, dt)
}

func sweptPairCircles(
	pa, va vec, ra float64, fa vec, semiA bool,
	pb, vb vec, rb float64, fb vec, semiB bool,
	dt float64,
) (float64, vec, bool) {
	ca, cra := colOf(pa, fa, ra, semiA)
	cb, crb := colOf(pb, fb, rb, semiB)
	bestT := dt + 1
	var bestN vec
	found := false
	consider := func(t float64, n vec) {
		if t < -1e-9 || t > dt {
			return
		}
		if t < 0 {
			t = 0
		}
		if n.len2() < 1e-12 {
			return
		}
		if t < bestT {
			bestT = t
			bestN = n.norm()
			found = true
		}
	}
	if t, n, ok := sweptCircles(ca, va, cra, cb, vb, crb, dt); ok {
		consider(t, n)
	}
	if semiA {
		d1, d2 := diameterOf(pa, fa, ra)
		if t, n, ok := sweptPointVsCapsule(cb, vb.sub(va), dt, d1, d2, crb+skin); ok {
			at := cb.add(vb.sub(va).mul(t))
			f := fa
			if f.len2() < 1e-12 {
				f = vec{1, 0}
			}
			if at.sub(pa).dot(f) <= 1e-4 {
				consider(t, n)
			}
		}
	}
	if semiB {
		d1, d2 := diameterOf(pb, fb, rb)
		if t, n, ok := sweptPointVsCapsule(ca, va.sub(vb), dt, d1, d2, cra+skin); ok {
			at := ca.add(va.sub(vb).mul(t))
			f := fb
			if f.len2() < 1e-12 {
				f = vec{1, 0}
			}
			if at.sub(pb).dot(f) <= 1e-4 {
				consider(t, n.mul(-1))
			}
		}
	}
	if !found {
		return 0, vec{}, false
	}
	return bestT, bestN, true
}

func sweptCircles(pa, va vec, ra float64, pb, vb vec, rb, dt float64) (float64, vec, bool) {
	relP := pa.sub(pb)
	relV := va.sub(vb)
	r := ra + rb
	r2 := r * r
	a := relV.len2()
	b := 2 * relP.dot(relV)
	c := relP.len2() - r2
	if c > 0 && b >= 0 {
		return 0, vec{}, false
	}
	if a < 1e-16 {
		if c <= 0 {
			n := relP.norm()
			return 0, n, true
		}
		return 0, vec{}, false
	}
	disc := b*b - 4*a*c
	if disc < 0 {
		return 0, vec{}, false
	}
	t := (-b - math.Sqrt(disc)) / (2 * a)
	if t < -1e-9 || t > dt {
		if c <= 1e-8 {
			return 0, relP.norm(), true
		}
		return 0, vec{}, false
	}
	if t < 0 {
		t = 0
	}
	at := relP.add(relV.mul(t))
	return t, at.norm(), true
}
