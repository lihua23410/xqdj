package sim

import (
	"encoding/json"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	unitpkg "xqdj/internal/unit"
)

type Phase string

const (
	PhaseSelect  Phase = "select"
	PhaseRunning Phase = "running"
	PhasePaused  Phase = "paused"
	PhaseEnded   Phase = "ended"
)

type unit struct {
	id      uint64
	kind    string
	role    string
	slot    int
	owner   uint64
	p, v    vec
	radius  float64
	hp      float64
	maxHP   float64
	actor   unitpkg.Actor
	inbox   chan unitpkg.Event
	stop    chan struct{}
	stopped bool
	solid   bool
	vision  float64
	cruise  float64
	decelT  float64
}

type spawnSpot struct {
	p vec
	r float64
}

type Match struct {
	mu       sync.Mutex
	phase    Phase
	slots    [2]string
	units    map[uint64]*unit
	order    []uint64
	cmds     chan unitpkg.Cmd
	nextID   uint64
	time     float64
	hex      hexagon
	winner   string
	winnerID uint64
	seq      uint64
	rng      *rand.Rand
	fx       []unitpkg.FX
	hitStop  int
}

func NewMatch() *Match {
	return NewMatchSeeded(uint64(time.Now().UnixNano()))
}

func NewMatchSeeded(seed uint64) *Match {
	kinds := unitpkg.FighterKinds()
	var slots [2]string
	if len(kinds) >= 2 {
		slots = [2]string{kinds[0], kinds[1]}
	}
	return &Match{
		phase: PhaseSelect,
		slots: slots,
		units: make(map[uint64]*unit),
		cmds:  make(chan unitpkg.Cmd, 512),
		hex:   newHexagon(HexRadius),
		rng:   rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)),
	}
}

func (m *Match) SnapshotJSON() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	type ujson struct {
		unitpkg.Snapshot
	}
	msg := struct {
		Type     string             `json:"type"`
		Phase    Phase              `json:"phase"`
		Slots    [2]string          `json:"slots"`
		Kinds    []string           `json:"kinds"`
		Winner   string             `json:"winner"`
		WinnerID uint64             `json:"winnerId"`
		Time     float64            `json:"time"`
		Seq      uint64             `json:"seq"`
		HexR     float64            `json:"hexRadius"`
		HitStop  int                `json:"hitStop"`
		Units    []unitpkg.Snapshot `json:"units"`
		Effects  []unitpkg.FX       `json:"effects"`
	}{
		Type:     "state",
		Phase:    m.phase,
		Slots:    m.slots,
		Kinds:    unitpkg.FighterKinds(),
		Winner:   m.winner,
		WinnerID: m.winnerID,
		Time:     m.time,
		Seq:      m.seq,
		HexR:     HexRadius,
		HitStop:  m.hitStop,
		Units:    make([]unitpkg.Snapshot, 0, len(m.order)),
		Effects:  append([]unitpkg.FX(nil), m.fx...),
	}
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.stopped || !u.solid {
			continue
		}
		msg.Units = append(msg.Units, u.snap())
	}
	b, _ := json.Marshal(msg)
	return b
}

func (u *unit) snap() unitpkg.Snapshot {
	return unitpkg.Snapshot{
		ID:      u.id,
		Kind:    u.kind,
		Role:    u.role,
		X:       u.p.X,
		Y:       u.p.Y,
		VX:      u.v.X,
		VY:      u.v.Y,
		Radius:  u.radius,
		HP:      u.hp,
		MaxHP:   u.maxHP,
		Vision:  u.vision,
		OwnerID: u.owner,
		Slot:    u.slot,
	}
}

func (m *Match) SetSlot(slot int, kind string) {
	if slot < 0 || slot > 1 {
		return
	}
	if !unitpkg.IsFighterKind(kind) {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.phase != PhaseSelect {
		return
	}
	m.slots[slot] = kind
}

func (m *Match) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.phase == PhasePaused {
		m.phase = PhaseRunning
		return
	}
	if m.phase != PhaseSelect && m.phase != PhaseEnded {
		return
	}
	m.resetLocked()
	var spots []spawnSpot
	for slot := 0; slot < 2; slot++ {
		kind := m.slots[slot]
		spec, ok := unitpkg.Lookup(kind)
		if !ok {
			continue
		}
		p := m.randomSpawnLocked(spec.Radius, spots)
		ang := m.rng.Float64() * 2 * math.Pi
		v := vec{math.Cos(ang) * spec.Speed, math.Sin(ang) * spec.Speed}
		m.spawnFighterLocked(kind, slot, p, v)
		spots = append(spots, spawnSpot{p: p, r: spec.Radius})
	}
	m.phase = PhaseRunning
}

func (m *Match) randomSpawnLocked(radius float64, others []spawnSpot) vec {
	R := HexRadius
	ap := R * math.Sqrt(3) / 2
	for try := 0; try < 128; try++ {
		p := vec{(m.rng.Float64()*2 - 1) * R, (m.rng.Float64()*2 - 1) * ap}
		if !m.hex.containsCenter(p, radius+4) {
			continue
		}
		ok := true
		for _, o := range others {
			if p.sub(o.p).len() < radius+o.r+8 {
				ok = false
				break
			}
		}
		if ok {
			return p
		}
	}
	if len(others) == 0 {
		return vec{}
	}
	return vec{120, 0}
}

func (m *Match) Pause() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.phase == PhaseRunning {
		m.phase = PhasePaused
	}
}

func (m *Match) End() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopAllLocked()
	m.phase = PhaseSelect
	m.winner = ""
	m.winnerID = 0
	m.time = 0
}

func (m *Match) Tick() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.phase != PhaseRunning {
		return
	}
	m.fx = m.fx[:0]
	if m.hitStop > 0 {
		m.drainCmdsLocked()
	} else {
		m.decelerateLocked(DT)
		m.drainCmdsLocked()
	}
	if m.hitStop > 0 {
		m.hitStop--
		m.seq++
		if m.hitStop == 0 {
			m.checkWinLocked()
		}
		return
	}
	m.physicsLocked(DT)
	m.time += DT
	m.seq++
	m.emitLocked()
	m.checkWinLocked()
}

func (m *Match) decelerateLocked(dt float64) {
	const step = 0.2
	const drop = 10.0
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || !u.solid || u.role != unitpkg.RoleFighter {
			continue
		}
		sp := u.v.len()
		if sp <= u.cruise+1e-6 {
			u.decelT = 0
			continue
		}
		u.decelT += dt
		for u.decelT >= step-1e-12 {
			u.decelT -= step
			sp = u.v.len()
			if sp <= u.cruise+1e-6 {
				break
			}
			ns := sp - drop
			if ns < u.cruise {
				ns = u.cruise
			}
			u.v = u.v.norm().mul(ns)
		}
	}
}

func (m *Match) resetLocked() {
	m.stopAllLocked()
	m.units = make(map[uint64]*unit)
	m.order = nil
	m.nextID = 0
	m.time = 0
	m.winner = ""
	m.winnerID = 0
	m.seq = 0
	m.hitStop = 0
}

func (m *Match) stopAllLocked() {
	for _, u := range m.units {
		m.killLocked(u)
	}
	m.units = make(map[uint64]*unit)
	m.order = nil
}

func (m *Match) killLocked(u *unit) {
	if u == nil || u.stopped {
		return
	}
	u.stopped = true
	close(u.stop)
}

func (m *Match) spawnFighterLocked(kind string, slot int, p, v vec) {
	m.addUnitLocked(kind, p, v, 0, slot)
}

func (m *Match) addUnitLocked(kind string, p, v vec, owner uint64, slot int) *unit {
	spec, ok := unitpkg.Lookup(kind)
	if !ok {
		return nil
	}
	actor := unitpkg.NewActor(kind, unitpkg.SpawnInfo{OwnerID: owner, Slot: slot})
	if actor == nil {
		return nil
	}
	m.nextID++
	id := m.nextID
	u := &unit{
		id:     id,
		kind:   kind,
		role:   spec.Role,
		slot:   slot,
		owner:  owner,
		p:      p,
		v:      v,
		radius: spec.Radius,
		hp:     spec.MaxHP,
		maxHP:  spec.MaxHP,
		vision: spec.Vision,
		cruise: spec.Speed,
		actor:  actor,
		inbox:  make(chan unitpkg.Event, 64),
		stop:   make(chan struct{}),
		solid:  true,
	}
	m.units[id] = u
	m.order = append(m.order, id)
	ctx := unitpkg.Context{ID: id, Kind: kind, Out: m.cmds}
	go runActor(u, ctx)
	return u
}

func runActor(u *unit, ctx unitpkg.Context) {
	for {
		select {
		case <-u.stop:
			return
		case ev, ok := <-u.inbox:
			if !ok {
				return
			}
			u.actor.Handle(ctx, ev)
		}
	}
}

func (m *Match) drainCmdsLocked() {
	for i := 0; i < 512; i++ {
		select {
		case cmd := <-m.cmds:
			m.applyCmdLocked(cmd)
		default:
			return
		}
	}
}

func (m *Match) applyCmdLocked(cmd unitpkg.Cmd) {
	switch c := cmd.(type) {
	case unitpkg.SetVelocity:
		u := m.units[c.UnitID]
		if u == nil || u.stopped {
			return
		}
		u.v = vec{c.VX, c.VY}
	case unitpkg.Damage:
		u := m.units[c.To]
		if u == nil || u.stopped || u.role != unitpkg.RoleFighter {
			return
		}
		m.fx = append(m.fx, unitpkg.FX{
			Name:   "hurt",
			UnitID: u.id,
			Kind:   u.kind,
			X:      u.p.X,
			Y:      u.p.Y,
			Slot:   u.slot,
			Amount: c.Amount,
		})
		u.hp -= c.Amount
		m.hitStop = HitStopFrames
		if u.hp <= 0 {
			u.hp = 0
			m.removeLocked(u)
		} else {
			m.swapOwnedLocked(u.id)
		}
	case unitpkg.Spawn:
		spec, ok := unitpkg.Lookup(c.Kind)
		if !ok || spec.Fighter {
			return
		}
		m.addUnitLocked(c.Kind, vec{c.X, c.Y}, vec{c.VX, c.VY}, c.OwnerID, c.Slot)
	case unitpkg.Despawn:
		u := m.units[c.UnitID]
		if u == nil {
			return
		}
		m.removeLocked(u)
	case unitpkg.SwapOwned:
		m.swapOwnedLocked(c.UnitID)
	case unitpkg.FX:
		m.fx = append(m.fx, c)
	}
}

func (m *Match) swapOwnedLocked(bodyID uint64) {
	body := m.units[bodyID]
	if body == nil || body.stopped || body.role != unitpkg.RoleFighter {
		return
	}
	var clones []*unit
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.stopped || u.owner != bodyID || u.role != unitpkg.RoleClone {
			continue
		}
		clones = append(clones, u)
	}
	if len(clones) == 0 {
		return
	}
	other := clones[m.rng.IntN(len(clones))]
	m.fx = append(m.fx,
		unitpkg.FX{Name: "swap", UnitID: body.id, Kind: body.kind, X: body.p.X, Y: body.p.Y, Slot: body.slot},
		unitpkg.FX{Name: "swap", UnitID: other.id, Kind: body.kind, X: other.p.X, Y: other.p.Y, Slot: other.slot},
	)
	body.p, other.p = other.p, body.p
	body.v, other.v = other.v, body.v
}

func (m *Match) removeLocked(u *unit) {
	if u == nil || u.stopped {
		return
	}
	id := u.id
	role := u.role
	m.killLocked(u)
	delete(m.units, u.id)
	for i, oid := range m.order {
		if oid == u.id {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}
	if role != unitpkg.RoleFighter {
		return
	}
	var extras []*unit
	for _, oid := range m.order {
		o := m.units[oid]
		if o != nil && o.owner == id {
			extras = append(extras, o)
		}
	}
	for _, o := range extras {
		m.removeLocked(o)
	}
}

type pairID struct{ a, b uint64 }

func canonPair(a, b uint64) pairID {
	if a > b {
		a, b = b, a
	}
	return pairID{a, b}
}

func (m *Match) physicsLocked(dt float64) {
	const maxIter = 24
	remain := dt
	ignore := map[pairID]bool{}
	for iter := 0; iter < maxIter && remain > 1e-8; iter++ {
		hit, ok := m.earliestHitLocked(remain, ignore)
		if !ok {
			for _, id := range m.order {
				u := m.units[id]
				if u.solid {
					u.p = u.p.add(u.v.mul(remain))
				}
			}
			m.constrainAllLocked()
			return
		}
		if hit.t > 1e-10 {
			for _, id := range m.order {
				u := m.units[id]
				if u.solid {
					u.p = u.p.add(u.v.mul(hit.t))
				}
			}
		}
		m.resolveLocked(hit)
		if hit.kind == hitPair {
			ignore[canonPair(hit.a, hit.b)] = true
		}
		remain -= hit.t
		if remain < 0 {
			remain = 0
		}
	}
	m.constrainAllLocked()
}

func (m *Match) earliestHitLocked(dt float64, ignore map[pairID]bool) (ccdHit, bool) {
	best := ccdHit{t: dt + 1}
	found := false
	for _, id := range m.order {
		u := m.units[id]
		if !u.solid {
			continue
		}
		if h, ok := sweptPointVsHex(u.p, u.v, u.radius, dt, m.hex); ok {
			if h.t < best.t {
				h.a = u.id
				best = h
				found = true
			}
		}
	}
	n := len(m.order)
	for i := 0; i < n; i++ {
		a := m.units[m.order[i]]
		if !a.solid {
			continue
		}
		for j := i + 1; j < n; j++ {
			b := m.units[m.order[j]]
			if !b.solid {
				continue
			}
			if ignore[canonPair(a.id, b.id)] {
				continue
			}
			if a.role == unitpkg.RoleProjectile && a.owner != 0 && a.owner == b.id {
				continue
			}
			if b.role == unitpkg.RoleProjectile && b.owner != 0 && b.owner == a.id {
				continue
			}
			t, nrm, ok := sweptCircles(a.p, a.v, a.radius, b.p, b.v, b.radius, dt)
			if !ok {
				continue
			}
			if t < best.t {
				best = ccdHit{kind: hitPair, t: t, a: a.id, b: b.id, n: nrm}
				found = true
			}
		}
	}
	return best, found
}

func (m *Match) resolveLocked(h ccdHit) {
	switch h.kind {
	case hitWall:
		u := m.units[h.a]
		if u == nil {
			return
		}
		limit := m.hex.d[0] - u.radius - skin
		pen := u.p.dot(h.n) - limit
		if pen > 0 {
			u.p = u.p.sub(h.n.mul(pen))
		}
		m.send(u, unitpkg.WallHit{Time: m.time, NX: h.n.X, NY: h.n.Y})
		u.v = reflectVelocity(u.v, h.n)
	case hitPair:
		a := m.units[h.a]
		b := m.units[h.b]
		if a == nil || b == nil {
			return
		}
		delta := a.p.sub(b.p)
		dist := delta.len()
		n := h.n
		if dist > 1e-9 {
			n = delta.norm()
		}
		target := a.radius + b.radius + skin
		if dist < 1e-9 {
			dist = 0
		}
		push := (target - dist) / 2
		a.p = a.p.add(n.mul(push))
		b.p = b.p.sub(n.mul(push))
		m.send(a, unitpkg.Collision{Time: m.time, Other: b.snap(), NX: n.X, NY: n.Y})
		m.send(b, unitpkg.Collision{Time: m.time, Other: a.snap(), NX: -n.X, NY: -n.Y})
		mid := a.p.add(b.p).mul(0.5)
		name := "impact"
		if a.role == unitpkg.RoleProjectile || b.role == unitpkg.RoleProjectile {
			name = "hit"
		}
		slot := a.slot
		kind := a.kind
		if a.role == unitpkg.RoleProjectile {
			slot = b.slot
			kind = b.kind
		}
		m.fx = append(m.fx, unitpkg.FX{Name: name, Kind: kind, X: mid.X, Y: mid.Y, Slot: slot})
		projA := a.role == unitpkg.RoleProjectile
		projB := b.role == unitpkg.RoleProjectile
		if projA || projB {
			if projA {
				a.v = vec{0, 0}
				a.solid = false
			}
			if projB {
				b.v = vec{0, 0}
				b.solid = false
			}
		} else {
			sa, sb := a.v.len(), b.v.len()
			a.v = n.mul(sa)
			b.v = n.mul(-sb)
		}
	}
}

func (m *Match) constrainAllLocked() {
	for _, id := range m.order {
		u := m.units[id]
		if !u.solid {
			continue
		}
		limit := m.hex.d[0] - u.radius - skin
		for i := 0; i < 6; i++ {
			n := m.hex.n[i]
			pen := u.p.dot(n) - limit
			if pen > 0 {
				u.p = u.p.sub(n.mul(pen))
			}
		}
	}
}

func (m *Match) emitLocked() {
	snaps := make([]unitpkg.Snapshot, 0, len(m.order))
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && !u.stopped {
			snaps = append(snaps, u.snap())
		}
	}
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.stopped {
			continue
		}
		sense := unitpkg.Sense{Time: m.time, Self: u.snap()}
		vr2 := u.vision * u.vision
		for _, o := range snaps {
			if o.ID == u.id {
				continue
			}
			dx := o.X - u.p.X
			dy := o.Y - u.p.Y
			if dx*dx+dy*dy <= vr2 {
				sense.Nearby = append(sense.Nearby, o)
			}
		}
		m.send(u, sense)
	}
}

func (m *Match) send(u *unit, ev unitpkg.Event) {
	if u == nil || u.stopped {
		return
	}
	select {
	case u.inbox <- ev:
	case <-u.stop:
	default:
	}
}

func (m *Match) checkWinLocked() {
	var fighters []*unit
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && !u.stopped && u.role == unitpkg.RoleFighter {
			fighters = append(fighters, u)
		}
	}
	if len(fighters) >= 2 {
		return
	}
	m.phase = PhaseEnded
	if len(fighters) == 1 {
		m.winner = fighters[0].kind
		m.winnerID = fighters[0].id
	} else {
		m.winner = "平局"
	}
}

func (m *Match) Loop(broadcast func([]byte), halt <-chan struct{}) {
	ticker := time.NewTicker(time.Second / TickHz)
	defer ticker.Stop()
	for {
		select {
		case <-halt:
			return
		case <-ticker.C:
			m.Tick()
			broadcast(m.SnapshotJSON())
		}
	}
}
