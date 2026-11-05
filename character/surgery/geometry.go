package 外科手术

import (
	"container/heap"
	"math"
	"xqdj/internal/unit"
)

type point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func (p point) add(q point) point   { return point{p.X + q.X, p.Y + q.Y} }
func (p point) sub(q point) point   { return point{p.X - q.X, p.Y - q.Y} }
func (p point) mul(v float64) point { return point{p.X * v, p.Y * v} }
func (p point) length() float64     { return math.Hypot(p.X, p.Y) }
func norm(p point) point {
	if d := p.length(); d > 1e-9 {
		return p.mul(1 / d)
	}
	return point{}
}
func lerp(a, b point, t float64) point { return a.add(b.sub(a).mul(t)) }

type knifePlan struct {
	Start, End, A, B point
	Wound            uint64
}

func outline(field unit.Field) unit.Outline {
	shape := field.Shape
	if shape == "" {
		shape = unit.ShapeHex
	}
	return unit.MakeOutline(shape, field.Extent)
}

func planKnife(field unit.Field, self, target point) (knifePlan, bool) {
	d := norm(target.sub(self))
	if d == (point{}) {
		return knifePlan{}, false
	}
	far, ok := rayEdge(field, self, d, 6)
	if !ok || far < 16 {
		return knifePlan{}, false
	}
	safe, ok := rayEdge(field, self, d, bodyRadius+.5)
	if !ok || safe < 14 {
		return knifePlan{}, false
	}
	end := math.Min(bodyRadius+8, safe-13)
	return knifePlan{Start: self.add(d.mul(far - .6)), End: self.add(d.mul(end)),
		A: self.add(d.mul(safe - 1)), B: self.add(d.mul(end))}, true
}

// The transverse stroke crosses the original wound's midpoint. Each arm is
// independently clipped at its first field edge; clipping one never shortens
// the opposite arm. Internal walls do not participate in either stroke.
func planCrossKnife(field unit.Field, a, b point) (knifePlan, bool) {
	length := b.sub(a).length()
	if length < 1 {
		return knifePlan{}, false
	}
	center := lerp(a, b, .5)
	d := norm(b.sub(a))
	normal := point{-d.Y, d.X}
	distances := [2]float64{}
	safeDistances := [2]float64{}
	for i, direction := range []point{normal, normal.mul(-1)} {
		far, ok := rayEdge(field, center, direction, 6)
		if !ok {
			return knifePlan{}, false
		}
		safe, ok := rayEdge(field, center, direction, bodyRadius+.5)
		if !ok {
			return knifePlan{}, false
		}
		distances[i] = math.Max(0, math.Min(length/2, far-.6))
		safeDistances[i] = math.Max(0, math.Min(length/2, safe-1))
	}
	if safeDistances[0]+safeDistances[1] < 1 {
		return knifePlan{}, false
	}
	return knifePlan{
		Start: center.add(normal.mul(distances[0])), End: center.sub(normal.mul(distances[1])),
		A: center.add(normal.mul(safeDistances[0])), B: center.sub(normal.mul(safeDistances[1])),
	}, true
}

// Long ray queries cannot use the short physics sweep: some outlines suppress
// an inward-facing sweep. Walk to the first exit, then refine that boundary.
func rayEdge(field unit.Field, from, direction point, radius float64) (float64, bool) {
	edge := outline(field)
	if !edge.Contains(from.X, from.Y, radius) {
		return 0, false
	}
	for distance := 4.0; distance <= 10000; distance += 4 {
		p := from.add(direction.mul(distance))
		if edge.Contains(p.X, p.Y, radius) {
			continue
		}
		lo, hi := distance-4, distance
		for i := 0; i < 24; i++ {
			mid := (lo + hi) / 2
			p = from.add(direction.mul(mid))
			if edge.Contains(p.X, p.Y, radius) {
				lo = mid
			} else {
				hi = mid
			}
		}
		return lo, true
	}
	return 0, false
}

func makeWound(id uint64, a, b point, field unit.Field) *wound {
	n := max(1, int(math.Ceil(b.sub(a).length()/24)))
	w := &wound{ID: id, A: a, B: b, Points: make([]point, n+1), Closed: make([]bool, n), hits: map[uint64]punctureHit{}}
	d := norm(b.sub(a))
	normal := point{-d.Y, d.X}
	for i := range w.Points {
		center := lerp(a, b, float64(i)/float64(n))
		offset := 14.0
		if i%2 != 0 {
			offset = -offset
		}
		for trial := 0; trial < 10; trial++ {
			p := center.add(normal.mul(offset))
			if field.OutlineContains(p.X, p.Y, bodyRadius+.5) &&
				(i == 0 || clearLine(field, w.Points[i-1], p, bodyRadius+.5)) {
				w.Points[i] = p
				break
			}
			offset *= .5
			w.Points[i] = center
		}
	}
	return w
}

func clearLine(field unit.Field, a, b point, radius float64) bool {
	steps := max(1, int(math.Ceil(b.sub(a).length()/4)))
	for i := 0; i <= steps; i++ {
		p := lerp(a, b, float64(i)/float64(steps))
		if !field.OutlineContains(p.X, p.Y, radius) {
			return false
		}
	}
	return true
}

// A small visibility-checked grid routes transfers around concave field edges.
// Internal walls intentionally do not participate. Only transfers use this;
// actual suturing always follows the wound's prescribed alternating punctures.
func fieldRoute(field unit.Field, from, to point, radius float64) []point {
	if clearLine(field, from, to, radius) {
		return []point{to}
	}
	lo, hi := point{1e9, 1e9}, point{-1e9, -1e9}
	for i := 0; i < 192; i++ {
		x, y, _, _ := field.SampleEdge(float64(i)/192, 0)
		lo.X, lo.Y = math.Min(lo.X, x), math.Min(lo.Y, y)
		hi.X, hi.Y = math.Max(hi.X, x), math.Max(hi.Y, y)
	}
	const cell = 32.0
	nx, ny := int(math.Ceil((hi.X-lo.X)/cell))+1, int(math.Ceil((hi.Y-lo.Y)/cell))+1
	valid := make([]bool, nx*ny)
	pos := func(i int) point { return point{lo.X + float64(i%nx)*cell, lo.Y + float64(i/nx)*cell} }
	start, goal := -1, -1
	sd, gd := math.Inf(1), math.Inf(1)
	for i := range valid {
		p := pos(i)
		valid[i] = field.OutlineContains(p.X, p.Y, radius)
		if !valid[i] {
			continue
		}
		d := p.sub(from).length()
		if d < sd && clearLine(field, from, p, radius) {
			start, sd = i, d
		}
		d = p.sub(to).length()
		if d < gd && clearLine(field, p, to, radius) {
			goal, gd = i, d
		}
	}
	if start < 0 || goal < 0 {
		return nil
	}
	cost := make([]float64, len(valid))
	parent := make([]int, len(valid))
	for i := range cost {
		cost[i] = math.Inf(1)
		parent[i] = -1
	}
	cost[start] = 0
	queue := routeQueue{{node: start, priority: pos(start).sub(to).length()}}
	heap.Init(&queue)
	for queue.Len() > 0 {
		u := heap.Pop(&queue).(routeNode).node
		if u == goal {
			break
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				x, y := u%nx+dx, u/nx+dy
				if x < 0 || y < 0 || x >= nx || y >= ny {
					continue
				}
				v := y*nx + x
				if !valid[v] {
					continue
				}
				next := cost[u] + pos(v).sub(pos(u)).length()
				if next >= cost[v] || !clearLine(field, pos(u), pos(v), radius) {
					continue
				}
				cost[v], parent[v] = next, u
				heap.Push(&queue, routeNode{v, next + pos(v).sub(to).length()})
			}
		}
	}
	if math.IsInf(cost[goal], 1) {
		return nil
	}
	path := []point{to}
	for n := goal; n != -1; n = parent[n] {
		path = append(path, pos(n))
	}
	for l, r := 0, len(path)-1; l < r; l, r = l+1, r-1 {
		path[l], path[r] = path[r], path[l]
	}
	// Remove grid stair steps whenever a longer visible leg fits inside the field.
	var result []point
	p := from
	for len(path) > 0 {
		n := len(path) - 1
		for n > 0 && !clearLine(field, p, path[n], radius) {
			n--
		}
		result = append(result, path[n])
		p = path[n]
		path = path[n+1:]
	}
	return result
}

type routeNode struct {
	node     int
	priority float64
}
type routeQueue []routeNode

func (q routeQueue) Len() int           { return len(q) }
func (q routeQueue) Less(i, j int) bool { return q[i].priority < q[j].priority }
func (q routeQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *routeQueue) Push(v any)        { *q = append(*q, v.(routeNode)) }
func (q *routeQueue) Pop() any          { n := len(*q) - 1; v := (*q)[n]; *q = (*q)[:n]; return v }
