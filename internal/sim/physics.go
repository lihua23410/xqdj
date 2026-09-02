package sim

import "math"

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
	HexRadius     = 280.0
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
)

type ccdHit struct {
	kind hitKind
	t    float64
	a, b uint64
	n    vec
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
