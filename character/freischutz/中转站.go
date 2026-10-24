// 中转站[Freischutz] 从自己的位置朝人开枪，和放站错开。开枪前先站定瞄准。子弹不改方向。
// 开局在场上随机放一批站。经过别的站时速率和伤害各乘二。放下的站各 3 次，用完消失。本体不耗耐久。
// 没打中人的子弹只再来一次：继承速率和伤害，从随机节点朝人重射。重射再打空就停。
package Freischutz

import (
	"embed"
	"math"
	"math/rand/v2"
	"sync"

	"xqdj/internal/unit"
)

const KindFrei = "中转站[Freischutz]"
const KindFreiNode = "中转站[Freischutz]随从"
const KindFreiShot = "中转站[Freischutz]弹"

const (
	bodyRadius = 18.0
	bodyHP     = 100.0
	cruise     = 160.0
	bodyColor  = "#0e2a5c"

	dropGap      = 2.0
	openingNodes = 6
	fireDelay    = 1.0 // 放下之后再隔这么久才开始瞄准，不和放站叠在同一拍
	aimWind      = 0.5 // 站定瞄准，到点才出弹
	samePoint    = 1.0

	nodeRadius = 16.0
	nodeVision = 9999.0
	nodeColor  = "#071833"
	durability = 3

	shotRadius = 5.0
	shotSpeed  = 520.0
	shotDamage = 2.0
	shotHitCD  = 0.1
	shotCap    = 10
	shotColor  = "#8ec5ff"
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindFrei, assets)
	p.Register(unit.Spec{
		Kind: KindFrei, Role: unit.RoleFighter, Radius: bodyRadius, MaxHP: bodyHP,
		Speed: cruise, Vision: nodeVision, Fighter: true,
		Look: unit.Look{Color: bodyColor, Glow: true, FX: []string{"freischutz"}},
	}, func(info unit.SpawnInfo) unit.Actor { return &中转站{rng: unit.NewRNG(info.Stream)} })
	p.Register(unit.Spec{
		Kind: KindFreiNode, Role: unit.RoleHelper, Radius: nodeRadius, MaxHP: 1,
		Speed: 0, Vision: nodeVision, Fighter: false,
		Look: unit.Look{Color: nodeColor, FX: []string{"freischutz-node"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &站{gate: newGate(info.OwnerID), slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind: KindFreiShot, Role: unit.RoleProjectile, Radius: shotRadius, MaxHP: 1,
		Speed: shotSpeed, Vision: nodeVision, Fighter: false, PassWalls: true,
		Look: unit.Look{Color: shotColor, Glow: true, FX: []string{"freischutz-shot"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &弹{
			owner: info.OwnerID, slot: info.Slot, damage: shotDamage,
			used: map[uint64]bool{}, rng: unit.NewRNG(info.Stream),
		}
	})
}

type 中转站 struct {
	booted   bool
	nextDrop float64
	nextFire float64
	aiming   bool
	aimUntil float64
	slot     int
	rng      *rand.Rand
}

func (a *中转站) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	a.slot = s.Self.Slot
	if !a.booted {
		a.booted = true
		a.nextDrop = s.Time + dropGap
		a.nextFire = s.Time + fireDelay
		clearLive(s.Self.Slot)
		a.scatter(ctx, s)
	}
	if s.Time+1e-9 >= a.nextDrop {
		a.nextDrop += dropGap
		a.leave(ctx, s)
	}
	person := unit.SeekIf(s, func(o unit.Snapshot) bool {
		return o.Role == unit.RoleFighter
	})
	if a.aiming {
		if person == nil {
			a.stopAim(ctx, s)
			return
		}
		a.face(ctx, s, person)
		if s.Time+1e-9 >= a.aimUntil {
			fire(ctx, s, ctx.ID, a.slot)
			a.stopAim(ctx, s)
			a.nextFire = s.Time + dropGap
		}
		return
	}
	if s.Time+1e-9 < a.nextFire || person == nil {
		return
	}
	if live(a.slot) >= shotCap {
		a.nextFire += dropGap
		return
	}
	a.aiming = true
	a.aimUntil = s.Time + aimWind
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	ctx.Out <- unit.FX{
		Name: "aim", Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot,
		X: s.Self.X, Y: s.Self.Y, Amount: aimWind,
	}
	a.face(ctx, s, person)
}

func (a *中转站) stopAim(ctx unit.Context, s unit.Sense) {
	if !a.aiming {
		return
	}
	a.aiming = false
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}
	ctx.Out <- unit.FX{
		Name: "aim", Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot,
		X: s.Self.X, Y: s.Self.Y,
	}
}

func (a *中转站) face(ctx unit.Context, s unit.Sense, target *unit.Snapshot) {
	dx, dy := target.X-s.Self.X, target.Y-s.Self.Y
	dist := math.Hypot(dx, dy)
	if dist < samePoint {
		return
	}
	ctx.Out <- unit.FX{
		Name: "face", Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot,
		X: s.Self.X, Y: s.Self.Y, VX: dx / dist, VY: dy / dist,
	}
}

func (a *中转站) scatter(ctx unit.Context, s unit.Sense) {
	if a.rng == nil {
		a.rng = unit.NewRNG(ctx.Stream)
	}
	for i := 0; i < openingNodes; i++ {
		x, y, ok := s.Field.RandomWalkable(a.rng, nodeRadius)
		if !ok {
			continue
		}
		a.place(ctx, s, x, y)
	}
}

func (a *中转站) leave(ctx unit.Context, s unit.Sense) {
	a.place(ctx, s, s.Self.X, s.Self.Y)
}

func (a *中转站) place(ctx unit.Context, s unit.Sense, x, y float64) {
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind:    KindFreiNode,
		X:       x,
		Y:       y,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	})
}

type 站 struct {
	gate   relayGate
	slot   int
	booted bool
}

func (n *站) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !n.booted {
		n.booted = true
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	}
	n.gate.touch(ctx, s)
}

// relayGate 记住每发弹上一拍的位置，用线段判断这一拍有没有穿身而过。
type relayGate struct {
	owner uint64
	left  int
	seen  map[uint64][2]float64
	used  map[uint64]bool
}

func newGate(owner uint64) relayGate {
	return relayGate{
		owner: owner,
		left:  durability,
		seen:  map[uint64][2]float64{},
		used:  map[uint64]bool{},
	}
}

// touch 线性扫描自己的弹。穿过一次，耐久减一；归零则这站消失。
func (g *relayGate) touch(ctx unit.Context, s unit.Sense) bool {
	if g.left <= 0 {
		return true
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind != KindFreiShot || o.OwnerID != g.owner {
			continue
		}
		prev, ok := g.seen[o.ID]
		g.seen[o.ID] = [2]float64{o.X, o.Y}
		if !ok || g.used[o.ID] {
			continue
		}
		reach := s.Self.Radius + o.Radius
		// 从站自己的位置打出去，这一段还在身子里，不算穿过。
		if covers(prev[0], prev[1], s.Self.X, s.Self.Y, reach) {
			continue
		}
		if !crossed(prev[0], prev[1], o.X, o.Y, s.Self.X, s.Self.Y, reach) {
			continue
		}
		g.used[o.ID] = true
		g.left--
		if g.left <= 0 {
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			return true
		}
	}
	return false
}

type 弹 struct {
	owner     uint64
	slot      int
	damage    float64
	used      map[uint64]bool
	seen      bool
	lx, ly    float64
	hitAt     float64
	hitPerson bool
	spent     bool
	rng       *rand.Rand
}

func (b *弹) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		b.claim(e.Self)
		if !e.Field.OutlineContains(e.Self.X, e.Self.Y, 0) {
			b.end(ctx, e)
			return
		}
		b.boost(ctx, e)
	case unit.Collision:
		if e.Other.Role == unit.RoleFighter && e.Other.Slot == b.slot && e.Other.ID != b.owner {
			dropLive(b.slot)
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			return
		}
		if !unit.Hittable(e.Other, b.slot) || e.Time < b.hitAt {
			return
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: b.damage}
		b.hitAt = e.Time + shotHitCD
		if e.Other.Role == unit.RoleFighter {
			b.hitPerson = true
		}
	}
}

// claim 接上重射交过来的伤害。这一发已经用过唯一的一次重射。
func (b *弹) claim(self unit.Snapshot) {
	if b.seen {
		return
	}
	if dmg, ok := takeReprise(self.X, self.Y, math.Hypot(self.VX, self.VY)); ok {
		b.damage = dmg
		b.spent = true
	}
}

// end 飞出场。没打中人、而且这发还不是重射，就从随机节点再来一次。
func (b *弹) end(ctx unit.Context, s unit.Sense) {
	if !b.hitPerson && !b.spent {
		b.reprise(ctx, s)
	}
	dropLive(b.slot)
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

func (b *弹) reprise(ctx unit.Context, s unit.Sense) {
	var nodes []unit.Snapshot
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind == KindFreiNode && o.OwnerID == b.owner {
			nodes = append(nodes, *o)
		}
	}
	if len(nodes) == 0 {
		return
	}
	if b.rng == nil {
		b.rng = unit.NewRNG(ctx.ID)
	}
	node := nodes[b.rng.IntN(len(nodes))]
	person := nearestPerson(s.Nearby, b.slot, node.X, node.Y)
	if person == nil {
		return
	}
	dx, dy := person.X-node.X, person.Y-node.Y
	dist := math.Hypot(dx, dy)
	if dist < samePoint {
		return
	}
	sp := math.Hypot(s.Self.VX, s.Self.VY)
	if sp < 1 {
		sp = shotSpeed
	}
	ux, uy := dx/dist, dy/dist
	queueReprise(node.X, node.Y, b.damage)
	addLive(b.slot)
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind:    KindFreiShot,
		X:       node.X,
		Y:       node.Y,
		VX:      ux * sp,
		VY:      uy * sp,
		OwnerID: b.owner,
		Slot:    b.slot,
	})
}

func nearestPerson(nearby []unit.Snapshot, slot int, x, y float64) *unit.Snapshot {
	var best *unit.Snapshot
	bestD := 0.0
	for i := range nearby {
		o := &nearby[i]
		if o.Role != unit.RoleFighter || o.Slot == slot {
			continue
		}
		d := math.Hypot(o.X-x, o.Y-y)
		if best == nil || d < bestD {
			cp := *o
			best = &cp
			bestD = d
		}
	}
	return best
}

// boost 沿上一拍到这一拍的线段找站。穿过几站，速率和伤害就乘几个二，方向不动。
func (b *弹) boost(ctx unit.Context, s unit.Sense) {
	ax, ay := b.lx, b.ly
	bx, by := s.Self.X, s.Self.Y
	if !b.seen {
		b.seen = true
		b.lx, b.ly = bx, by
		return
	}
	n := 0
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !isStation(*o, b.owner) || b.used[o.ID] {
			continue
		}
		reach := o.Radius + s.Self.Radius
		if covers(ax, ay, o.X, o.Y, reach) {
			continue
		}
		if !crossed(ax, ay, bx, by, o.X, o.Y, reach) {
			continue
		}
		b.used[o.ID] = true
		n++
	}
	b.lx, b.ly = bx, by
	if n == 0 {
		return
	}
	sp := math.Hypot(s.Self.VX, s.Self.VY)
	if sp < 1e-6 {
		return
	}
	mul := math.Pow(2, float64(n))
	b.damage *= mul
	ctx.Out <- unit.SetVelocity{
		UnitID: ctx.ID,
		VX:     s.Self.VX * mul,
		VY:     s.Self.VY * mul,
	}
}

func isStation(o unit.Snapshot, owner uint64) bool {
	if o.Kind == KindFreiNode && o.OwnerID == owner {
		return true
	}
	return o.Kind == KindFrei && o.ID == owner
}

// covers 点是否已经在半径里。出生在站心时，飞出去的第一段不算穿过。
func covers(x, y, px, py, radius float64) bool {
	return math.Hypot(x-px, y-py) <= radius
}

// crossed 线性扫描：线段到圆心的最近距离不超过半径，算穿过去了。
func crossed(ax, ay, bx, by, px, py, radius float64) bool {
	dx, dy := bx-ax, by-ay
	l2 := dx*dx + dy*dy
	if l2 < 1e-12 {
		return math.Hypot(px-ax, py-ay) <= radius
	}
	t := ((px-ax)*dx + (py-ay)*dy) / l2
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	qx, qy := ax+t*dx, ay+t*dy
	return math.Hypot(px-qx, py-qy) <= radius
}

func fire(ctx unit.Context, s unit.Sense, owner uint64, slot int) {
	if live(slot) >= shotCap {
		return
	}
	target := unit.SeekIf(s, func(o unit.Snapshot) bool {
		return o.Role == unit.RoleFighter
	})
	if target == nil {
		return
	}
	dx, dy := target.X-s.Self.X, target.Y-s.Self.Y
	dist := math.Hypot(dx, dy)
	if dist < samePoint {
		return
	}
	ux, uy := dx/dist, dy/dist
	addLive(slot)
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind:    KindFreiShot,
		X:       s.Self.X,
		Y:       s.Self.Y,
		VX:      ux * shotSpeed,
		VY:      uy * shotSpeed,
		OwnerID: owner,
		Slot:    slot,
	})
}

var (
	liveMu    sync.Mutex
	liveShots map[int]int
)

func resetLive() {
	liveMu.Lock()
	liveShots = nil
	reprises = nil
	liveMu.Unlock()
}

func live(slot int) int {
	liveMu.Lock()
	defer liveMu.Unlock()
	if liveShots == nil {
		return 0
	}
	return liveShots[slot]
}

func addLive(slot int) {
	liveMu.Lock()
	defer liveMu.Unlock()
	if liveShots == nil {
		liveShots = map[int]int{}
	}
	liveShots[slot]++
}

func dropLive(slot int) {
	liveMu.Lock()
	defer liveMu.Unlock()
	if liveShots == nil {
		return
	}
	if liveShots[slot] > 0 {
		liveShots[slot]--
	}
}

func clearLive(slot int) {
	liveMu.Lock()
	defer liveMu.Unlock()
	if liveShots != nil {
		delete(liveShots, slot)
	}
}

func setLive(slot, n int) {
	liveMu.Lock()
	defer liveMu.Unlock()
	if liveShots == nil {
		liveShots = map[int]int{}
	}
	liveShots[slot] = n
}

type reprise struct {
	x, y   float64
	damage float64
}

var reprises []reprise

func queueReprise(x, y, damage float64) {
	liveMu.Lock()
	reprises = append(reprises, reprise{x: x, y: y, damage: damage})
	liveMu.Unlock()
}

func takeReprise(x, y, speed float64) (float64, bool) {
	liveMu.Lock()
	defer liveMu.Unlock()
	best := -1
	bestD := math.MaxFloat64
	limit := speed/60 + 24
	if limit < 24 {
		limit = 24
	}
	for i, r := range reprises {
		d := math.Hypot(x-r.x, y-r.y)
		if d < bestD {
			bestD = d
			best = i
		}
	}
	if best < 0 || bestD > limit {
		return 0, false
	}
	dmg := reprises[best].damage
	reprises = append(reprises[:best], reprises[best+1:]...)
	return dmg, true
}
