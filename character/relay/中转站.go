// 中转站不转向、不出伤。开局留下随从，之后每 2 秒再留一只。
// 随从和中转站本体都传球：下一手按这一趟的传球优先度挑，不算发出者。
package 中转站

import (
	"embed"
	"math"
	"sync"

	"xqdj/internal/unit"
)

const KindRelay = "中转站"
const KindRelayNode = "中转站随从"
const KindRelayShot = "中转弹"

const (
	bodyRadius = 18.0
	bodyHP     = 100.0
	cruise     = 160.0
	bodyColor  = "#c4843a"

	dropGap   = 2.0
	samePoint = 1.0 // 中心相距小于这，算同一点，没有方向

	nodeRadius = 10.0
	nodeVision = 9999.0
	nodeColor  = "#6e4a28"

	shotRadius = 5.0
	shotSpeed  = 520.0
	shotDamage = 2.0
	shotHitCD  = 0.1
	shotOff   = 640.0
	shotCap   = 10
	shotColor = "#ffc14a"
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindRelay, assets)
	p.Register(unit.Spec{
		Kind: KindRelay, Role: unit.RoleFighter, Radius: bodyRadius, MaxHP: bodyHP,
		Speed: cruise, Vision: 0, Fighter: true,
		Look: unit.Look{Color: bodyColor, Glow: true, FX: []string{"relay"}},
	}, func(unit.SpawnInfo) unit.Actor { return &中转站{} })
	p.Register(unit.Spec{
		// 不实心才不挡路。画面只收实心、战斗机和 helper，所以用 helper，语义仍是随从。
		Kind: KindRelayNode, Role: unit.RoleHelper, Radius: nodeRadius, MaxHP: 1,
		Speed: 0, Vision: nodeVision, Fighter: false,
		Look: unit.Look{Color: nodeColor, FX: []string{"relay-node"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &随从{owner: info.OwnerID, slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind: KindRelayShot, Role: unit.RoleProjectile, Radius: shotRadius, MaxHP: 1,
		Speed: shotSpeed, Vision: 9999, Fighter: false, PassWalls: true,
		Look: unit.Look{Color: shotColor, Glow: true, Trail: true, FX: []string{"relay-shot"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &中转弹{slot: info.Slot, target: info.OwnerID}
	})
}

type 中转站 struct {
	booted   bool
	nextDrop float64
	slot     int
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
		a.nextDrop = s.Time
		clearLive(s.Self.Slot)
	}
	if s.Time+1e-9 >= a.nextDrop {
		a.nextDrop += dropGap
		a.leave(ctx, s)
	}
	a.relay(ctx, s)
}

func (a *中转站) leave(ctx unit.Context, s unit.Sense) {
	ctx.Out <- unit.Spawn{
		Kind:    KindRelayNode,
		X:       s.Self.X,
		Y:       s.Self.Y,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	}
}

func (a *中转站) relay(ctx unit.Context, s unit.Sense) {
	catch(ctx, s, ctx.ID, a.slot)
}

type 随从 struct {
	owner uint64
	slot  int
	ready bool
}

func (n *随从) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !n.ready {
		n.ready = true
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
		pass(ctx, s, n.owner, n.slot, 0, newTrack())
	}
	catch(ctx, s, n.owner, n.slot)
}

func catch(ctx unit.Context, s unit.Sense, owner uint64, slot int) bool {
	caught := false
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind != KindRelayShot || o.OwnerID != s.Self.ID {
			continue
		}
		if math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) > s.Self.Radius+o.Radius {
			continue
		}
		sender, tr := takeShot(*o)
		if tr == nil {
			tr = newTrack()
		}
		tr.mark(s.Self.ID)
		ctx.Out <- unit.Despawn{UnitID: o.ID}
		pass(ctx, s, owner, slot, sender, tr)
		caught = true
	}
	return caught
}

func pass(ctx unit.Context, s unit.Sense, owner uint64, slot int, sender uint64, tr *hopTrack) {
	if live(slot) >= shotCap {
		return
	}
	target := choose(s.Self, s.Nearby, owner, sender, tr)
	if target == nil {
		return
	}
	dx, dy := target.X-s.Self.X, target.Y-s.Self.Y
	dist := math.Hypot(dx, dy)
	if dist < samePoint {
		return
	}
	ux, uy := dx/dist, dy/dist
	place := s.Self.Radius + shotRadius + 1
	if place > dist*0.5 {
		place = dist * 0.5
	}
	x := s.Self.X + ux*place
	y := s.Self.Y + uy*place
	vx, vy := ux*shotSpeed, uy*shotSpeed
	addLive(slot)
	queuePass(passInfo{sender: s.Self.ID, target: target.ID, x: x, y: y, vx: vx, vy: vy, track: tr, slot: slot})
	ctx.Out <- unit.Spawn{
		Kind:    KindRelayShot,
		X:       x,
		Y:       y,
		VX:      vx,
		VY:      vy,
		OwnerID: target.ID,
		Slot:    slot,
	}
}

func choose(self unit.Snapshot, nearby []unit.Snapshot, owner, sender uint64, tr *hopTrack) *unit.Snapshot {
	var best *unit.Snapshot
	bestD := 0.0
	for i := range nearby {
		o := &nearby[i]
		if !mate(*o, self, owner) || o.ID == sender {
			continue
		}
		d := math.Hypot(o.X-self.X, o.Y-self.Y)
		if d < samePoint {
			continue
		}
		if best == nil || prefer(*o, d, *best, bestD, tr) {
			cp := *o
			best = &cp
			bestD = d
		}
	}
	return best
}

func prefer(a unit.Snapshot, da float64, b unit.Snapshot, db float64, tr *hopTrack) bool {
	if tr != nil {
		fa, fb := tr.ahead(a.ID), tr.ahead(b.ID)
		if fa != fb {
			return fa < fb
		}
	}
	if da < db-1e-6 {
		return true
	}
	if db < da-1e-6 {
		return false
	}
	return earlier(a, b)
}

func mate(o, self unit.Snapshot, owner uint64) bool {
	if o.ID == self.ID {
		return false
	}
	if o.Kind == KindRelayNode && o.OwnerID == owner {
		return true
	}
	return o.Kind == KindRelay && o.ID == owner
}

func earlier(a, b unit.Snapshot) bool {
	aF := a.Kind == KindRelay
	bF := b.Kind == KindRelay
	if aF != bF {
		return aF
	}
	return a.ID < b.ID
}

type hopTrack struct {
	seq  int
	last map[uint64]int
}

func newTrack() *hopTrack {
	return &hopTrack{last: map[uint64]int{}}
}

func (t *hopTrack) mark(id uint64) {
	if t == nil {
		return
	}
	t.seq++
	if t.last == nil {
		t.last = map[uint64]int{}
	}
	t.last[id] = t.seq
}

// ahead：还没接到过的最小；接过的按上一次接到的手次，越小越久。
func (t *hopTrack) ahead(id uint64) int {
	if t == nil || t.last == nil {
		return 0
	}
	n, ok := t.last[id]
	if !ok {
		return 0
	}
	return n
}

type passInfo struct {
	sender uint64
	target uint64
	x, y   float64
	vx, vy float64
	track  *hopTrack
	slot   int
}

var (
	passMu      sync.Mutex
	passPending []passInfo
	liveShots   map[int]int
)

func resetPass() {
	passMu.Lock()
	passPending = nil
	liveShots = nil
	passMu.Unlock()
}

func live(slot int) int {
	passMu.Lock()
	defer passMu.Unlock()
	if liveShots == nil {
		return 0
	}
	return liveShots[slot]
}

func addLive(slot int) {
	passMu.Lock()
	defer passMu.Unlock()
	if liveShots == nil {
		liveShots = map[int]int{}
	}
	liveShots[slot]++
}

func dropLive(slot int) {
	passMu.Lock()
	defer passMu.Unlock()
	if liveShots == nil {
		return
	}
	if liveShots[slot] > 0 {
		liveShots[slot]--
	}
}

func clearLive(slot int) {
	passMu.Lock()
	defer passMu.Unlock()
	if liveShots != nil {
		delete(liveShots, slot)
	}
}

func setLive(slot, n int) {
	passMu.Lock()
	defer passMu.Unlock()
	if liveShots == nil {
		liveShots = map[int]int{}
	}
	liveShots[slot] = n
}

func queuePass(p passInfo) {
	passMu.Lock()
	passPending = append(passPending, p)
	passMu.Unlock()
}

func takeShot(b unit.Snapshot) (uint64, *hopTrack) {
	passMu.Lock()
	defer passMu.Unlock()
	best := -1
	bestD := math.MaxFloat64
	for i, p := range passPending {
		if p.target != 0 && b.OwnerID != 0 && p.target != b.OwnerID {
			continue
		}
		d := math.Hypot(b.X-p.x, b.Y-p.y)
		if d < bestD {
			bestD = d
			best = i
		}
	}
	if best < 0 {
		return 0, nil
	}
	p := passPending[best]
	passPending = append(passPending[:best], passPending[best+1:]...)
	if liveShots != nil && liveShots[p.slot] > 0 {
		liveShots[p.slot]--
	}
	return p.sender, p.track
}

func pokePass(b unit.Snapshot) {
	passMu.Lock()
	defer passMu.Unlock()
	best := -1
	bestD := math.MaxFloat64
	for i, p := range passPending {
		if p.target != 0 && b.OwnerID != 0 && p.target != b.OwnerID {
			continue
		}
		d := math.Hypot(b.X-p.x, b.Y-p.y)
		if d < bestD {
			bestD = d
			best = i
		}
	}
	if best < 0 {
		return
	}
	passPending[best].x = b.X
	passPending[best].y = b.Y
	passPending[best].vx = b.VX
	passPending[best].vy = b.VY
}

func dropPending(b unit.Snapshot) {
	passMu.Lock()
	defer passMu.Unlock()
	best := -1
	bestD := math.MaxFloat64
	for i, p := range passPending {
		if p.target != 0 && b.OwnerID != 0 && p.target != b.OwnerID {
			continue
		}
		d := math.Hypot(b.X-p.x, b.Y-p.y)
		if d < bestD {
			bestD = d
			best = i
		}
	}
	if best < 0 {
		return
	}
	p := passPending[best]
	passPending = append(passPending[:best], passPending[best+1:]...)
	if liveShots != nil && liveShots[p.slot] > 0 {
		liveShots[p.slot]--
	}
}

type 中转弹 struct {
	slot       int
	target     uint64
	hitReadyAt float64
}

func (b *中转弹) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		pokePass(e.Self)
		if math.Abs(e.Self.X) > shotOff || math.Abs(e.Self.Y) > shotOff {
			dropPending(e.Self)
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			return
		}
		b.home(ctx, e)
	case unit.Collision:
		if !unit.Hittable(e.Other, b.slot) || e.Time < b.hitReadyAt {
			return
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: shotDamage}
		b.hitReadyAt = e.Time + shotHitCD
	}
}

func (b *中转弹) home(ctx unit.Context, s unit.Sense) {
	if b.target == 0 {
		return
	}
	var body *unit.Snapshot
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.ID != b.target || o.Kind != KindRelay {
			continue
		}
		cp := *o
		body = &cp
		break
	}
	if body == nil {
		return
	}
	dx, dy := body.X-s.Self.X, body.Y-s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	ctx.Out <- unit.SetVelocity{
		UnitID: ctx.ID,
		VX:     dx / n * shotSpeed,
		VY:     dy / n * shotSpeed,
	}
}
