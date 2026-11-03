package ospf

import (
	"embed"
	"fmt"
	"math"

	"xqdj/internal/unit"
)

const KindOSPF = "中转站[ospf]"
const KindOSPFNode = "OSPF节点"
const arrivalBlastKind = "OSPF接报爆炸"

const (
	nodeRadius   = 10
	shotRadius   = 5
	shotSpeed    = 260
	blastRadius  = 24
	packetDamage = 2
	dropInterval = 2
	sendInterval = 1
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindOSPF, assets)
	p.Register(unit.Spec{
		Kind: KindOSPF, Role: unit.RoleFighter, Fighter: true,
		Radius: 18, MaxHP: 100, Speed: 160,
		Look: unit.Look{Color: "#54dde0", Glow: true, FX: []string{"ospf"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &fighter{layout: areasFor(info), birth: point{info.X, info.Y}, nextSend: sendInterval}
	})
	p.Register(unit.Spec{
		Kind: KindOSPFNode, Role: unit.RoleHelper, Radius: nodeRadius, MaxHP: 1,
		Vision: 9999, Nonsolid: true, RequiresOwner: true,
		Look: unit.Look{Color: "#858e99", FX: []string{"ospf-node"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		data, _ := info.Data.(nodeBirth)
		return &node{nodeBirth: data, lastRole: -1}
	})

	p.Register(unit.Spec{
		Kind: arrivalBlastKind, Role: unit.RoleHelper, MaxHP: 1,
		Vision: 9999, Nonsolid: true, RequiresOwner: true,
	}, func(info unit.SpawnInfo) unit.Actor {
		return &arrivalBlast{owner: info.OwnerID, slot: info.Slot}
	})

	for kind, color := range map[lsa]string{
		routerLSA: "#77e8fa", networkLSA: "#718bff", summaryLSA: "#ffdf93",
		asbrLSA: "#eea4ff", externalLSA: "#ff9379", nssaLSA: "#83f6a4",
	} {
		p.Register(unit.Spec{
			Kind: shotKind(kind), Role: unit.RoleProjectile, Radius: shotRadius, MaxHP: 1,
			Speed: shotSpeed, Pass: true, PassWalls: true, RequiresOwner: true,
			Look: unit.Look{Color: color, FX: []string{"ospf-shot"}},
		}, func(info unit.SpawnInfo) unit.Actor {
			data, _ := info.Data.(flight)
			return &packet{flight: data, slot: info.Slot}
		})
	}
}

func shotKind(kind lsa) string { return fmt.Sprintf("OSPF报文%d", kind) }

type fighter struct {
	layout             areaMap
	birth              point
	net                *network
	nextDrop, nextSend float64
}

func (a *fighter) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if a.net == nil {
		a.net = &network{owner: ctx.ID, slot: s.Self.Slot,
			coreR: a.layout.radius, fieldR: a.layout.radius * 2}
	}
	out := a.advance(s)
	out.send(ctx)
	// X/Y carry the backbone radius and the sector rotation so the skin can
	// draw the same area cut the simulation uses.
	// Amount packs the sector→area permutation as base-10 digits so the skin
	// colours each sector with the area that actually lives there.
	code := 0
	if len(a.layout.areas) == 3 {
		code = a.layout.areas[0] + a.layout.areas[1]*10 + a.layout.areas[2]*100
	}
	ctx.Out <- unit.FX{Kind: KindOSPF, Name: "ospf-clock", UnitID: ctx.ID,
		X: a.layout.radius * 2, Y: a.layout.angle, Amount: float64(code),
		VX: a.nextDrop, VY: a.nextSend}
}

func (a *fighter) advance(s unit.Sense) commands {
	n := a.net
	n.mu.Lock()
	defer n.mu.Unlock()
	var out commands
	from := point{s.Self.X, s.Self.Y}
	now := s.Time + 1e-9
	for {
		if a.nextDrop <= now && (a.nextDrop <= a.nextSend || n.nearestASBR(from) == nil) {
			at := from
			if a.nextDrop == 0 {
				at = a.birth
			}
			n.install(at, a.layout.at(at.x, at.y), &out)
			// A full clock releases one packet when an installation first
			// supplies an ASBR. Start the next interval at that release.
			if a.nextSend < a.nextDrop && n.nearestASBR(from) != nil {
				a.nextSend = a.nextDrop
			}
			a.nextDrop += dropInterval
			continue
		}
		if a.nextSend > now {
			break
		}
		target := n.nearestASBR(from)
		if target == nil {
			break
		}
		n.inject(from, target, &out)
		a.nextSend += sendInterval
	}
	return out
}

type nodeBirth struct {
	net    *network
	router *router
}

type node struct {
	nodeBirth
	lastRole  int
	refreshAt float64
}

func (n *node) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok || n.router == nil {
		return
	}
	n.net.mu.Lock()
	area := n.router.area
	role := 0
	if n.net.areas[area][0] == n.router {
		role |= 1
	}
	if n.net.bridges[area].abr == n.router {
		role |= 2
	}
	// 4 = 这是某个非零区的边界路由器（跨区干线的骨干那端）。皮肤要靠它
	// 把上行线画到真正收包的路由器上，而不是"最近的骨干节点"。
	for a := 1; a < len(n.net.bridges); a++ {
		if n.net.bridges[a].br == n.router {
			role |= 4
			break
		}
	}
	n.net.mu.Unlock()
	if role != n.lastRole || s.Time >= n.refreshAt {
		n.lastRole, n.refreshAt = role, s.Time+1
		ctx.Out <- unit.FX{Kind: KindOSPF, Name: "ospf-role", UnitID: ctx.ID, VX: float64(area), VY: float64(role)}
	}
}

type packet struct {
	flight
	slot       int
	done       bool
	hitReadyAt map[uint64]float64
}

// arrive 把报文交给 target，并把自己撤掉。
func (p *packet) arrive(ctx unit.Context) {
	if p.done {
		return
	}
	p.done = true
	var out commands
	p.network.mu.Lock()
	p.network.receive(p.target, p.flood, &out)
	p.network.mu.Unlock()
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	out.send(ctx)
}

func (p *packet) Handle(ctx unit.Context, ev unit.Event) {
	if p.done {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		if p.target == nil {
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			p.done = true
			return
		}
		p.lastTime = e.Time
		// 先判到达：进了接收半径的报文已经到站，位置本来就会越过终点一点，
		// 那一点不属于"在路上偏离网线"。
		if dx, dy := p.target.x-e.Self.X, p.target.y-e.Self.Y; math.Hypot(dx, dy) <= nodeRadius+shotRadius {
			p.arrive(ctx)
			return
		}
		// Debug gate: a packet that leaves its own cable means the route and
		// the flight disagree, which also puts it off the line the skin draws.
		p.assertOnCable(point{e.Self.X, e.Self.Y}, shotSpeed/30)
		// Fly the cable: every turn point is a real waypoint, so the packet
		// never cuts a corner the drawn line doesn't take.
		for len(p.path) > 0 {
			next := p.path[0]
			if math.Hypot(next.x-e.Self.X, next.y-e.Self.Y) > nodeRadius+shotRadius {
				return
			}
			// 落在折点上再转向：报文不会在拐角处抄近路，轨迹与画出来的
			// 折线重合，偏离检查才有意义。
			ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: next.x, Y: next.y}
			p.path = p.path[1:]
			if len(p.path) == 0 {
				break
			}
			tgt := p.path[0]
			d := math.Hypot(tgt.x-next.x, tgt.y-next.y)
			if d > 1 {
				ctx.Out <- unit.SetVelocity{UnitID: ctx.ID,
					VX: (tgt.x - next.x) / d * shotSpeed, VY: (tgt.y - next.y) / d * shotSpeed}
			}
		}
		dx, dy := p.target.x-e.Self.X, p.target.y-e.Self.Y
		if math.Hypot(dx, dy) > nodeRadius+shotRadius {
			return
		}
		p.arrive(ctx)
	case unit.Collision:
		if !unit.Hittable(e.Other, p.slot) || e.Time < p.hitReadyAt[e.Other.ID] {
			return
		}
		if p.hitReadyAt == nil {
			p.hitReadyAt = make(map[uint64]float64)
		}
		p.hitReadyAt[e.Other.ID] = e.Time + .1
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: packetDamage}
	}
}

type arrivalBlast struct {
	owner uint64
	slot  int
	done  bool
}

func (b *arrivalBlast) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok || b.done {
		return
	}
	b.done = true
	x, y := s.Self.X, s.Self.Y
	for _, o := range s.Nearby {
		if unit.Hittable(o, b.slot) && math.Hypot(o.X-x, o.Y-y) <= blastRadius+o.Radius {
			ctx.Out <- unit.Damage{From: b.owner, To: o.ID, Amount: packetDamage}
		}
	}
	ctx.Out <- unit.FX{Name: "ospf-arrival-blast", Kind: KindOSPF, UnitID: ctx.ID,
		X: x, Y: y, Slot: b.slot, Amount: blastRadius}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}
