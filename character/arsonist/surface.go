package 纵火者

import (
	"encoding/json"
	"math"
	"sort"

	"xqdj/internal/unit"
)

// One shared coverage grid per arsonist: touching oil connects regardless of
// which bottle or walker laid it. Three world units are small relative to bodies.
const surfaceStep = 3.0
const chunkSide = 32
const (
	dry uint8 = iota
	oil
	flame
)

type point struct{ x, y int }
type surfaceCell struct {
	version   uint64
	due       float64 // oil: earliest ignition; flame: expiry
	state     uint8
	scheduled bool
}
type surfaceChunk struct {
	cells [chunkSide * chunkSide]surfaceCell
	count int
	dirty bool
	view  chunkView
}
type chunkView struct {
	X    int   `json:"x"`
	Y    int   `json:"y"`
	Oil  []int `json:"o,omitempty"`
	Fire []int `json:"f,omitempty"`
}
type surfaceView struct {
	Revision uint64      `json:"revision"`
	Step     float64     `json:"step"`
	Side     int         `json:"side"`
	Chunks   []chunkView `json:"chunks"`
}

type fireEvent struct {
	at      float64
	p       point
	version uint64
	expire  bool
}
type fireQueue []fireEvent

func before(a, b fireEvent) bool {
	if a.at != b.at {
		return a.at < b.at
	}
	if a.expire != b.expire {
		return a.expire
	}
	if a.p.y != b.p.y {
		return a.p.y < b.p.y
	}
	return a.p.x < b.p.x
}
func (q *fireQueue) push(e fireEvent) {
	*q = append(*q, e)
	i := len(*q) - 1
	for i > 0 {
		p := (i - 1) / 2
		if !before(e, (*q)[p]) {
			break
		}
		(*q)[i] = (*q)[p]
		i = p
	}
	(*q)[i] = e
}
func (q *fireQueue) pop() fireEvent {
	a := *q
	out := a[0]
	last := a[len(a)-1]
	a = a[:len(a)-1]
	if len(a) > 0 {
		i := 0
		for {
			left := i*2 + 1
			if left >= len(a) {
				break
			}
			best := left
			if right := left + 1; right < len(a) && before(a[right], a[left]) {
				best = right
			}
			if !before(a[best], last) {
				break
			}
			a[i] = a[best]
			i = best
		}
		a[i] = last
	}
	*q = a
	return out
}

type oilSurface struct {
	chunks           map[point]*surfaceChunk
	events           fireQueue
	serial, revision uint64
	viewRevision     uint64
	view             json.RawMessage
}

func chunkKey(p point) point { return point{p.x >> 5, p.y >> 5} }
func cellIndex(p point) int  { return (p.y&31)*chunkSide + (p.x & 31) }
func (g *oilSurface) cell(p point) *surfaceCell {
	if c := g.chunks[chunkKey(p)]; c != nil {
		return &c.cells[cellIndex(p)]
	}
	return nil
}
func (g *oilSurface) set(p point, state uint8) *surfaceCell {
	if g.chunks == nil {
		g.chunks = map[point]*surfaceChunk{}
	}
	k := chunkKey(p)
	chunk := g.chunks[k]
	if chunk == nil {
		chunk = &surfaceChunk{}
		g.chunks[k] = chunk
	}
	c := &chunk.cells[cellIndex(p)]
	if c.state == dry && state != dry {
		chunk.count++
	}
	if c.state != dry && state == dry {
		chunk.count--
	}
	g.serial++
	g.revision++
	chunk.dirty = true
	*c = surfaceCell{state: state, version: g.serial}
	if chunk.count == 0 {
		delete(g.chunks, k)
	}
	return c
}

func (g *oilSurface) wet(p point) bool { c := g.cell(p); return c != nil && c.state != dry }

type spreadStep struct {
	p     point
	delay float64
}

var spreadSteps = func() []spreadStep {
	var out []spreadStep
	for y := -2; y <= 2; y++ {
		for x := -2; x <= 2; x++ {
			ax, ay := abs(x), abs(y)
			if (ax == 0 && ay == 0) || (ax == 2 && ay == 0) || (ax == 0 && ay == 2) || (ax == 2 && ay == 2) {
				continue
			}
			out = append(out, spreadStep{point{x, y}, math.Hypot(float64(x), float64(y)) * surfaceStep / spreadSpeed})
		}
	}
	return out
}()

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// Longer directions reduce grid-direction bias. They may not jump dry cells.
func (g *oilSurface) connected(a, b point) bool {
	dx, dy := b.x-a.x, b.y-a.y
	if abs(dx) == 2 {
		return g.wet(point{a.x + dx/2, a.y}) && g.wet(point{a.x + dx/2, a.y + dy})
	}
	if abs(dy) == 2 {
		return g.wet(point{a.x, a.y + dy/2}) && g.wet(point{a.x + dx, a.y + dy/2})
	}
	return true
}
func (g *oilSurface) schedule(p point, at float64) {
	c := g.cell(p)
	if c == nil || c.state != oil || (c.scheduled && c.due <= at+1e-9) {
		return
	}
	c.scheduled, c.due = true, at
	g.events.push(fireEvent{p: p, at: at, version: c.version})
}
func (g *oilSurface) spread(p point, at float64) {
	for _, step := range spreadSteps {
		n := point{p.x + step.p.x, p.y + step.p.y}
		if c := g.cell(n); c != nil && c.state == oil && g.connected(p, n) {
			g.schedule(n, at+step.delay)
		}
	}
}
func (g *oilSurface) burn(p point, at float64) { g.burnFor(p, at, oilFireLife) }
func (g *oilSurface) burnFor(p point, at, life float64) {
	expires := at + life
	if c := g.cell(p); c != nil && c.state == flame && c.due >= expires {
		return
	}
	c := g.set(p, flame)
	c.due = expires
	g.events.push(fireEvent{p: p, at: expires, version: c.version, expire: true})
}
func (g *oilSurface) advance(now float64) {
	for len(g.events) > 0 && g.events[0].at <= now+1e-9 {
		e := g.events.pop()
		c := g.cell(e.p)
		if c == nil || c.version != e.version {
			continue
		}
		if e.expire {
			if c.state == flame {
				g.set(e.p, dry)
			}
			continue
		}
		if c.state != oil || !c.scheduled || math.Abs(c.due-e.at) > 1e-9 {
			continue
		}
		g.burn(e.p, e.at)
		g.spread(e.p, e.at)
	}
}

func gridCoord(v float64) int { return int(math.Floor(v / surfaceStep)) }

// rasterTaper 沿线段把半径从 r0 线性收到 r1。拖尾用它把粗壮的根部接回油滩，
// 否则等半径的细胶囊切在大圆盘边上会留下一个细腰和两处凹角。
func rasterTaper(x0, y0, x1, y1, r0, r1 float64, visit func(point)) {
	// Sub-texel bodies still paint a continuous path at this grid resolution.
	floor := surfaceStep/math.Sqrt2 + 1e-9
	r0, r1 = math.Max(floor, r0), math.Max(floor, r1)
	r := math.Max(r0, r1)
	dx, dy := x1-x0, y1-y0
	length2 := dx*dx + dy*dy
	covered := false
	for y := gridCoord(math.Min(y0, y1) - r); y <= gridCoord(math.Max(y0, y1)+r); y++ {
		for x := gridCoord(math.Min(x0, x1) - r); x <= gridCoord(math.Max(x0, x1)+r); x++ {
			px, py := (float64(x)+.5)*surfaceStep, (float64(y)+.5)*surfaceStep
			t := 0.0
			if length2 > 1e-12 {
				t = math.Max(0, math.Min(1, ((px-x0)*dx+(py-y0)*dy)/length2))
			}
			ox, oy := px-x0-t*dx, py-y0-t*dy
			rr := r0 + (r1-r0)*t
			if ox*ox+oy*oy <= rr*rr {
				covered = true
				visit(point{x, y})
			}
		}
	}
	// Very small bodies still leave a visible, hittable minimum of one texel.
	if !covered {
		visit(point{gridCoord(x0), gridCoord(y0)})
	}
}

func rasterStroke(x0, y0, x1, y1, r float64, visit func(point)) {
	rasterTaper(x0, y0, x1, y1, r, r, visit)
}

func (g *oilSurface) addOil(x0, y0, x1, y1, r, now float64) {
	g.addOilTaper(x0, y0, x1, y1, r, r, now)
}
func (g *oilSurface) addOilTaper(x0, y0, x1, y1, r0, r1, now float64) {
	type freshCell struct {
		p       point
		relight bool
	}
	var fresh []freshCell
	rasterTaper(x0, y0, x1, y1, r0, r1, func(p point) {
		c := g.cell(p)
		if c != nil && c.state == oil {
			return
		}
		relight := c != nil && c.state == flame
		g.set(p, oil)
		fresh = append(fresh, freshCell{p, relight})
	})
	// Seed from pre-existing flame, before advancing any of the new oil. Otherwise
	// sequential rasterization would ignite the entire new pool in one tick.
	for _, f := range fresh {
		lit := f.relight
		if !lit {
			for y := -1; y <= 1 && !lit; y++ {
				for x := -1; x <= 1; x++ {
					if c := g.cell(point{f.p.x + x, f.p.y + y}); c != nil && c.state == flame {
						lit = true
						break
					}
				}
			}
		}
		if lit {
			g.schedule(f.p, now)
		}
	}
}
func (g *oilSurface) addFire(x, y, r, now float64) {
	var lit []point
	rasterStroke(x, y, x, y, r, func(p point) {
		life := bottleFireLife
		if c := g.cell(p); c != nil && c.state == oil {
			life = oilFireLife
		}
		g.burnFor(p, now, life)
		lit = append(lit, p)
	})
	for _, p := range lit {
		g.spread(p, now)
	}
}

func bodyHitsCell(o unit.Snapshot, p point) bool {
	x0, y0 := float64(p.x)*surfaceStep, float64(p.y)*surfaceStep
	dx := o.X - math.Max(x0, math.Min(x0+surfaceStep, o.X))
	dy := o.Y - math.Max(y0, math.Min(y0+surfaceStep, o.Y))
	return dx*dx+dy*dy <= o.Radius*o.Radius+1e-9
}

func (g *oilSurface) touches(o unit.Snapshot, state uint8) bool {
	for y := gridCoord(o.Y - o.Radius); y <= gridCoord(o.Y+o.Radius); y++ {
		for x := gridCoord(o.X - o.Radius); x <= gridCoord(o.X+o.Radius); x++ {
			p := point{x, y}
			c := g.cell(p)
			if c != nil && c.state == state && bodyHitsCell(o, p) {
				return true
			}
		}
	}
	return false
}

// igniteBody 把身体碰上的油当拍点着，再按火头速度往外烧。不把整片相连的油一次点完。
func (g *oilSurface) igniteBody(o unit.Snapshot, now float64) {
	var lit []point
	for y := gridCoord(o.Y - o.Radius); y <= gridCoord(o.Y+o.Radius); y++ {
		for x := gridCoord(o.X - o.Radius); x <= gridCoord(o.X+o.Radius); x++ {
			p := point{x, y}
			c := g.cell(p)
			if c == nil || c.state != oil || !bodyHitsCell(o, p) {
				continue
			}
			g.burn(p, now)
			lit = append(lit, p)
		}
	}
	for _, p := range lit {
		g.spread(p, now)
	}
}

func (g *oilSurface) payload() json.RawMessage {
	if g.view != nil && g.viewRevision == g.revision {
		return g.view
	}
	keys := make([]point, 0, len(g.chunks))
	for k := range g.chunks {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].y != keys[j].y {
			return keys[i].y < keys[j].y
		}
		return keys[i].x < keys[j].x
	})
	view := surfaceView{Revision: g.revision, Step: surfaceStep, Side: chunkSide, Chunks: make([]chunkView, 0, len(keys))}
	for _, k := range keys {
		chunk := g.chunks[k]
		if chunk.dirty {
			v := chunkView{X: k.x, Y: k.y}
			for row := 0; row < chunkSide; row++ {
				for x := 0; x < chunkSide; {
					start := row*chunkSide + x
					state := chunk.cells[start].state
					end := x + 1
					for end < chunkSide && chunk.cells[row*chunkSide+end].state == state {
						end++
					}
					if state == oil {
						v.Oil = append(v.Oil, start, end-x)
					} else if state == flame {
						v.Fire = append(v.Fire, start, end-x)
					}
					x = end
				}
			}
			chunk.view = v
			chunk.dirty = false
		}
		view.Chunks = append(view.Chunks, chunk.view)
	}
	data, err := json.Marshal(view)
	if err != nil {
		panic(err)
	}
	g.view, g.viewRevision = data, g.revision
	return data
}
