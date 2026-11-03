package ospf

import (
	"fmt"
	"math"
	"sync"

	"xqdj/internal/unit"
)

type point struct{ x, y float64 }

func (p point) distance(q point) float64 { return math.Hypot(p.x-q.x, p.y-q.y) }

type router struct {
	point
	id           uint64
	area         int
	db           database
	imported     map[lsaKey]summaryImport
	summarySince uint64
}

// Routers are appended in scheduled installation order. For simultaneous
// installations this is also spawn command order, hence ascending unit ID.
type network struct {
	mu         sync.Mutex
	owner      uint64
	slot       int
	coreR      float64 // backbone circle radius, same cut the skin draws
	fieldR     float64 // field apothem
	areas      [4][]*router
	bridges    [4]bridge
	sequence   uint64
	routers    uint64
	translated map[lsaKey]uint64
	// 本轮装机后已经通知过皮肤的航线。每条边一轮只发一次 FX，否则每个报文
	// 都发一遍会把前端压垮。
	sent map[segKey]bool
}

type bridge struct{ abr, br *router }

type summaryImport struct {
	sequence uint64
	pending  bool
}

// segKey 是一条航线段的身份：两端路由器 id（0 表示本体自己）。节点一直在
// 漂，用坐标当 key 会每帧都算新边，FX 能把前端淹掉。
type segKey struct{ from, to uint64 }

func (n *network) elect() {
	for area := 1; area < len(n.areas); area++ {
		var best bridge
		bestDistance := math.Inf(1)
		for _, abr := range n.areas[area] {
			for _, br := range n.areas[0] {
				if d := abr.distance(br.point); d < bestDistance {
					best = bridge{abr, br}
					bestDistance = d
				}
			}
		}
		n.bridges[area] = best
	}
	// 新一轮装机，航线要重新告诉皮肤一遍。
	n.sent = nil
}

func (n *network) nearestASBR(from point) *router {
	var best *router
	bestDistance := math.Inf(1)
	for _, area := range []int{0, 1, 3} {
		for _, r := range n.areas[area] {
			if d := from.distance(r.point); d < bestDistance {
				best, bestDistance = r, d
			}
		}
	}
	return best
}

type flood struct {
	announcement
	entered       uint8
	importSummary bool
}

func round(a announcement, area int) flood {
	return flood{announcement: a, entered: 1 << area}
}

// Each physical hop carries an announcement and the areas already entered.
// path holds the remaining turn points when the cable bends around the
// backbone; the packet steers to each of them in order.
type flight struct {
	network *network
	target  *router
	flood   flood
	path    []point
	// cable is the whole drawn route, start included, so the packet can
	// prove it never leaves the line the skin draws for it.
	cable []point
	// lastTime 只用来算每帧步长，喂给偏离检查当容差。
	lastTime float64
}

// assertOnCable panics when a packet drifts off its own cable. The skin draws
// exactly the points route() returns, so any divergence here means the packet
// and the line no longer agree — a bug worth crashing on while it is new.
//
// tol is how far one integration step can carry the packet (shotSpeed*dt), not
// a licence to wander: past a corner the packet is still on the incoming leg
// until it teleports onto the corner itself, so the residual is bounded by one
// step. A genuine route/flight disagreement grows without bound and still trips
// the gate on the very next step.
func (f *flight) assertOnCable(pos point, tol float64) {
	if len(f.cable) < 2 {
		return
	}
	best := math.Inf(1)
	for i := 0; i+1 < len(f.cable); i++ {
		if d := segmentDistance(pos, f.cable[i], f.cable[i+1]); d < best {
			best = d
		}
	}
	if limit := 2.5 + tol; best > limit {
		panic(fmt.Sprintf("ospf: 报文偏离网线 %.2f 单位，容差 %.2f（pos=%v cable=%v path=%d target=%v lastTime=%.4f）",
			best, limit, pos, f.cable, len(f.path), f.target.point, f.lastTime))
	}
}

// segmentDistance returns how far p sits from the segment a-b.
func segmentDistance(p, a, b point) float64 {
	dx, dy := b.x-a.x, b.y-a.y
	length2 := dx*dx + dy*dy
	if length2 == 0 {
		return p.distance(a)
	}
	t := ((p.x-a.x)*dx + (p.y-a.y)*dy) / length2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return p.distance(point{a.x + dx*t, a.y + dy*t})
}

type commands []unit.Cmd

func (cs commands) send(ctx unit.Context) {
	for _, c := range cs {
		if s, ok := c.(unit.Spawn); ok {
			c = ctx.Born(s)
		}
		ctx.Out <- c
	}
}

func (n *network) install(at point, area int, out *commands) {
	n.routers++
	r := &router{point: at, id: n.routers, area: area}
	n.areas[area] = append(n.areas[area], r)
	*out = append(*out, unit.Spawn{Kind: KindOSPFNode, X: at.x, Y: at.y,
		OwnerID: n.owner, Slot: n.slot, Data: nodeBirth{n, r}})

	nodes := n.areas[area]
	dr := nodes[0]
	if r != dr {
		// Database exchange is control state only. Historical LSAs don't
		// turn into new attacks when a neighbor joins the broadcast network.
		for _, known := range dr.db {
			r.learn(known)
		}
	}

	old := n.bridges
	n.elect()
	for area, previous := range old {
		if previous.abr != nil && previous.abr != n.bridges[area].abr {
			n.withdrawSummaries(previous.abr)
		}
	}
	n.publish(r, routerLSA, destination{node: r}, out)
	if r != dr {
		n.publish(dr, networkLSA, destination{area: area}, out)
	}

	for a := 1; a < len(n.bridges); a++ {
		link := n.bridges[a]
		if link.abr == nil {
			continue
		}
		changed := old[a].abr != link.abr
		if changed || a == area {
			n.advertise(link, summaryLSA, destination{area: a}, out)
		}
		switch a {
		case 1:
			if changed {
				for _, asbr := range n.areas[1] {
					n.advertise(link, asbrLSA, destination{node: asbr}, out)
				}
			} else if area == 1 {
				n.advertise(link, asbrLSA, destination{node: r}, out)
			}
			if changed || area == 0 {
				n.publish(link.abr, summaryLSA, destination{area: 0}, out)
			}
			if changed {
				for _, asbr := range n.areas[0] {
					n.publish(link.abr, asbrLSA, destination{node: asbr}, out)
				}
			} else if area == 0 {
				n.publish(link.abr, asbrLSA, destination{node: r}, out)
			}
			if old[a] != link {
				n.syncTransitDatabases(link, out)
			}
		case 2, 3:
			if changed {
				n.publish(link.abr, summaryLSA, destination{isDefault: true}, out)
			}
		}
	}
	if link := n.bridges[3]; link.abr != nil && old[3] != link {
		for _, known := range link.abr.db.ordered() {
			n.translate(link, known, out)
		}
	}
}

func (n *network) publish(from *router, kind lsa, dest destination, out *commands) {
	n.deliver(from, round(n.originate(kind, from, dest), from.area), out)
}

func (n *network) advertise(link bridge, kind lsa, dest destination, out *commands) {
	f := round(n.originate(kind, link.abr, dest), 0)
	f.entered |= 1 << link.abr.area
	n.transmit(link.abr.point, link.br, f, out)
}

func (n *network) syncTransitDatabases(link bridge, out *commands) {
	// A newly attached ABR exchanges historical type-5 state silently.
	// Freshly originated type-5 LSAs still use the normal physical flood.
	for _, source := range []*router{link.br, link.abr} {
		for _, a := range source.db {
			if a.kind != externalLSA {
				continue
			}
			for _, r := range n.areas[1-source.area] {
				r.learn(a)
			}
		}
	}
	for _, a := range link.br.db.ordered() {
		if isSummary(a.kind) && a.advertiser.area != 1 && n.activeSummary(a) &&
			link.abr.imported[a.lsaKey].sequence < a.sequence {
			n.publishImportedSummary(link.abr, a, out)
		}
	}
}

func isSummary(kind lsa) bool { return kind == summaryLSA || kind == asbrLSA }

func (n *network) sendSummary(link bridge, a announcement, out *commands) {
	if link.abr.imported[a.lsaKey].sequence >= a.sequence {
		return
	}
	link.abr.setImport(a, true)
	f := round(a, 1)
	f.importSummary = true
	n.transmit(link.br.point, link.abr, f, out)
}

func (r *router) setImport(a announcement, pending bool) {
	if r.imported == nil {
		r.imported = make(map[lsaKey]summaryImport)
	}
	r.imported[a.lsaKey] = summaryImport{sequence: a.sequence, pending: pending}
}

func (n *network) publishImportedSummary(at *router, a announcement, out *commands) {
	at.setImport(a, false)
	n.publish(at, a.kind, a.destination, out)
}

func (n *network) translate(link bridge, a announcement, out *commands) {
	if a.kind != nssaLSA || !a.propagate || a.forwarding == nil || n.translated[a.lsaKey] >= a.sequence {
		return
	}
	if n.translated == nil {
		n.translated = make(map[lsaKey]uint64)
	}
	n.translated[a.lsaKey] = a.sequence
	translated := n.originate(externalLSA, link.abr, a.destination)
	translated.forwarding = a.forwarding
	f := round(translated, 0)
	f.entered |= 1 << 3
	n.transmit(link.abr.point, link.br, f, out)
}

func (n *network) transmit(from point, to *router, f flood, out *commands) {
	if !n.launch(0, from, to, f, out) {
		n.receive(to, f, out)
	}
}

// route bends a cable around the backbone so a length never cuts across
// another area. The bend sits right at the circle's shoulder, so the detour
// stays as short as a two-leg run can be. Backbone-to-backbone links are free
// to cut straight across: that circle is their own area.
func (n *network) route(from, to point) []point {
	if n.coreR <= 0 {
		return nil
	}
	fa, ta := math.Hypot(from.x, from.y), math.Hypot(to.x, to.y)
	if fa < n.coreR*1.02 && ta < n.coreR*1.02 {
		return nil
	}
	if segmentClears(from, to, n.coreR*1.04) {
		return nil
	}
	dx, dy := to.x-from.x, to.y-from.y
	length2 := dx*dx + dy*dy
	if length2 == 0 {
		return nil
	}
	t := -(from.x*dx + from.y*dy) / length2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	mx, my := from.x+dx*t, from.y+dy*t
	near := math.Hypot(mx, my)
	var nx, ny float64
	if near > 1e-6 {
		nx, ny = mx/near, my/near
	} else {
		length := math.Sqrt(length2)
		nx, ny = -dy/length, dx/length
	}
	for push := n.coreR*1.12 - near; push <= n.coreR*3.5; push += n.coreR*0.25 {
		q := point{mx + nx*push, my + ny*push}
		if n.fieldR > 0 && math.Hypot(q.x, q.y) > n.fieldR*0.94 {
			break
		}
		if segmentClears(from, q, n.coreR*1.02) && segmentClears(q, to, n.coreR*1.02) {
			return []point{q, to}
		}
	}
	return nil
}

// segmentClears reports whether the segment from a to b stays outside a circle
// of radius r centred on the origin.
func segmentClears(a, b point, r float64) bool {
	dx, dy := b.x-a.x, b.y-a.y
	length2 := dx*dx + dy*dy
	if length2 == 0 {
		return math.Hypot(a.x, a.y) > r
	}
	t := -(a.x*dx + a.y*dy) / length2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return math.Hypot(a.x+dx*t, a.y+dy*t) > r
}

func (n *network) launch(fromID uint64, from point, to *router, f flood, out *commands) bool {
	dist := from.distance(to.point)
	if dist < 1 {
		return false
	}
	path := n.route(from, to.point)
	next := to.point
	if len(path) > 0 {
		next = path[0]
	}
	leg := from.distance(next)
	ux, uy := (next.x-from.x)/leg, (next.y-from.y)/leg
	offset := math.Min(nodeRadius+shotRadius+1, dist/2)
	cable := append([]point{from}, path...)
	cable = append(cable, to.point)
	// 把这条报文的实际航线整段发给皮肤。皮肤照着它画网线，报文的每一次
	// 位移都落在画出来的线上，不会再出现"报文走在没画的线上"。
	if n.sent == nil {
		n.sent = make(map[segKey]bool)
	}
	for i := 0; i+1 < len(cable); i++ {
		k := segKey{fromID, to.id}
		if n.sent[k] {
			continue
		}
		n.sent[k] = true
		*out = append(*out, unit.FX{Kind: KindOSPF, Name: "ospf-cable", UnitID: n.owner,
			X: cable[i].x, Y: cable[i].y, VX: cable[i+1].x, VY: cable[i+1].y})
	}
	*out = append(*out, unit.Spawn{
		Kind: shotKind(f.kind), X: from.x + ux*offset, Y: from.y + uy*offset,
		VX: ux * shotSpeed, VY: uy * shotSpeed, OwnerID: n.owner, Slot: n.slot,
		Data: flight{network: n, target: to, flood: f, path: path, cable: cable},
	})
	return true
}

func (n *network) queueBlast(at *router, out *commands) {
	*out = append(*out, unit.Spawn{Kind: arrivalBlastKind, X: at.x, Y: at.y, OwnerID: n.owner, Slot: n.slot})
}

func (n *network) receive(at *router, f flood, out *commands) {
	n.queueBlast(at, out)
	n.deliver(at, f, out)
}

func (n *network) deliver(at *router, f flood, out *commands) {
	if isSummary(f.kind) && !n.activeSummary(f.announcement) {
		return
	}
	if f.importSummary {
		entry := at.imported[f.lsaKey]
		if entry.sequence != f.sequence || !entry.pending {
			return
		}
		if n.bridges[1].abr != at {
			delete(at.imported, f.lsaKey)
			return
		}
		n.publishImportedSummary(at, f.announcement, out)
		return
	}
	for {
		if !at.learn(f.announcement) {
			return
		}
		n.cross(at, &f, out)
		next := n.nextLocal(at, f)
		if next == nil || n.launch(at.id, at.point, next, f, out) {
			return
		}
		n.queueBlast(next, out)
		at = next
	}
}

func (n *network) nextLocal(at *router, f flood) *router {
	nodes := n.areas[at.area]
	eligible := func(r *router) bool { return r.db[f.lsaKey].sequence < f.sequence }
	if dr := nodes[0]; eligible(dr) {
		return dr
	}
	var best *router
	bestDistance := math.Inf(1)
	for _, r := range nodes {
		if d := at.distance(r.point); eligible(r) && d < bestDistance {
			best, bestDistance = r, d
		}
	}
	return best
}

func (n *network) cross(at *router, f *flood, out *commands) {
	if at.area == 3 && n.bridges[3].abr == at {
		n.translate(n.bridges[3], f.announcement, out)
		return
	}

	link := n.bridges[1]
	if at.area == 0 && at == link.br && f.entered&(1<<1) == 0 && isSummary(f.kind) {
		f.entered |= 1 << 1
		n.sendSummary(link, f.announcement, out)
		return
	}
	if f.kind != externalLSA {
		return
	}
	var to *router
	switch {
	case at.area == 0 && at == link.br:
		to = link.abr
	case at.area == 1 && at == link.abr:
		to = link.br
	}
	if to == nil || f.entered&(1<<to.area) != 0 {
		return
	}
	f.entered |= 1 << to.area
	crossing := round(f.announcement, to.area)
	crossing.entered = f.entered
	n.transmit(at.point, to, crossing, out)
}
