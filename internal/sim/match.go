package sim

import (
	"encoding/json"
	"math"
	"math/rand/v2"
	"runtime"
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
	id             uint64
	kind           string
	role           string
	slot           int
	owner          uint64
	p, v           vec
	radius         float64
	hp             float64
	maxHP          float64
	actor          unitpkg.Actor
	inbox          chan unitpkg.Event
	stop           chan struct{}
	stopped        bool
	solid          bool
	vision         float64
	cruise         float64
	decelT         float64
	semi           bool
	face           vec
	passWalls      bool
	shell          bool
	faction        string
	factionCycle   bool
	factionAmpOut  float64
	factionAmpIn   float64
	factionCollect bool
	factionSeen    map[string]bool
	factionNext    float64
	factionBarrage []string
	marks          map[string]*stackMark
}

type spawnSpot struct {
	p vec
	r float64
}

type barrier struct {
	id     uint64
	owner  uint64
	slot   int
	kind   string
	a, b   vec
	radius float64
	until  float64
	amount float64
	hitAt  map[uint64]float64
}

type wallSnap struct {
	ID     uint64  `json:"id"`
	X1     float64 `json:"x1"`
	Y1     float64 `json:"y1"`
	X2     float64 `json:"x2"`
	Y2     float64 `json:"y2"`
	Radius float64 `json:"radius"`
	Kind   string  `json:"kind"`
	Slot   int     `json:"slot"`
}

type Match struct {
	mu         sync.Mutex
	phase      Phase
	slots      [2]string
	units      map[uint64]*unit
	order      []uint64
	cmds       chan unitpkg.Cmd
	nextID     uint64
	time       float64
	hex        hexagon
	winner     string
	winnerID   uint64
	seq        uint64
	rng        *rand.Rand
	fx         []unitpkg.FX
	hitStop    int
	walls      []*barrier
	pending    []unitpkg.Damage
	dmgSeq     uint64
	pendingDmg map[uint64]dmgOffer
	wardAbsorb map[uint64]bool
}

type dmgOffer struct {
	from      uint64
	to        uint64
	amount    float64
	absorb    bool
	markKind  string
	markDelta int
	markIcon  string
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
		phase:      PhaseSelect,
		slots:      slots,
		units:      make(map[uint64]*unit),
		cmds:       make(chan unitpkg.Cmd, 512),
		hex:        newHexagon(HexRadius),
		rng:        rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)),
		pendingDmg: make(map[uint64]dmgOffer),
		wardAbsorb: make(map[uint64]bool),
	}
}

func (m *Match) SnapshotJSON() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	type ujson struct {
		unitpkg.Snapshot
	}
	msg := struct {
		Type     string                  `json:"type"`
		Phase    Phase                   `json:"phase"`
		Slots    [2]string               `json:"slots"`
		Kinds    []string                `json:"kinds"`
		Looks    map[string]unitpkg.Look `json:"looks"`
		Winner   string                  `json:"winner"`
		WinnerID uint64                  `json:"winnerId"`
		Time     float64                 `json:"time"`
		Seq      uint64                  `json:"seq"`
		HexR     float64                 `json:"hexRadius"`
		HitStop  int                     `json:"hitStop"`
		Units    []unitpkg.Snapshot      `json:"units"`
		Walls    []wallSnap              `json:"walls"`
		Effects  []unitpkg.FX            `json:"effects"`
	}{
		Type:     "state",
		Phase:    m.phase,
		Slots:    m.slots,
		Kinds:    unitpkg.FighterKinds(),
		Looks:    unitpkg.Looks(),
		Winner:   m.winner,
		WinnerID: m.winnerID,
		Time:     m.time,
		Seq:      m.seq,
		HexR:     HexRadius,
		HitStop:  m.hitStop,
		Units:    make([]unitpkg.Snapshot, 0, len(m.order)),
		Walls:    make([]wallSnap, 0, len(m.walls)),
		Effects:  append([]unitpkg.FX(nil), m.fx...),
	}
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.stopped || !u.inSnapshot() {
			continue
		}
		msg.Units = append(msg.Units, u.snap())
	}
	for _, w := range m.walls {
		msg.Walls = append(msg.Walls, wallSnap{
			ID: w.id, X1: w.a.X, Y1: w.a.Y, X2: w.b.X, Y2: w.b.Y,
			Radius: w.radius, Kind: w.kind, Slot: w.slot,
		})
	}
	b, _ := json.Marshal(msg)
	return b
}

func (u *unit) snap() unitpkg.Snapshot {
	return unitpkg.Snapshot{
		ID:        u.id,
		Kind:      u.kind,
		Role:      u.role,
		X:         u.p.X,
		Y:         u.p.Y,
		VX:        u.v.X,
		VY:        u.v.Y,
		Radius:    u.radius,
		HP:        u.hp,
		MaxHP:     u.maxHP,
		Vision:    u.vision,
		OwnerID:   u.owner,
		Slot:      u.slot,
		Semi:      u.semi,
		FaceX:     u.face.X,
		FaceY:     u.face.Y,
		PassWalls: u.passWalls,
		Faction:   u.faction,
		Seen:      u.seenList(),
		Marks:     u.markList(),
	}
}

func (u *unit) inSnapshot() bool {
	if u == nil || u.stopped {
		return false
	}
	return u.solid || u.role == unitpkg.RoleHelper
}

func (u *unit) seenList() []string {
	if !u.factionCollect || len(u.factionSeen) == 0 {
		return nil
	}
	out := make([]string, 0, 4)
	for _, f := range unitpkg.AllFactions() {
		if u.factionSeen[f] {
			out = append(out, f)
		}
	}
	return out
}

func (u *unit) noteFaction(f string) {
	if !u.factionCollect || f == "" {
		return
	}
	if u.factionSeen == nil {
		u.factionSeen = map[string]bool{}
	}
	u.factionSeen[f] = true
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
	m.settleHitsLocked()
	if m.hitStop > 0 {
		m.hitStop--
		m.seq++
		if m.hitStop == 0 {
			m.checkWinLocked()
		}
		return
	}
	m.physicsLocked(DT)
	for _, d := range m.pending {
		m.applyCmdLocked(d)
	}
	m.pending = m.pending[:0]
	m.settleHitsLocked()
	m.stickShellsLocked()
	m.time += DT
	m.expireWallsLocked()
	m.seq++
	m.emitLocked()
	m.checkWinLocked()
}

func (m *Match) Play(maxTicks int) (winner string, ticks int) {
	for t := 0; t < maxTicks; t++ {
		m.Tick()
		m.mu.Lock()
		ended := m.phase == PhaseEnded
		w := m.winner
		m.mu.Unlock()
		if ended {
			if w == "" {
				w = "平局"
			}
			return w, t + 1
		}
		runtime.Gosched()
	}
	return "平局", maxTicks
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
	m.walls = nil
	m.pending = nil
	m.dmgSeq = 0
	m.pendingDmg = make(map[uint64]dmgOffer)
	m.wardAbsorb = make(map[uint64]bool)
}

func (m *Match) stopAllLocked() {
	for _, u := range m.units {
		m.killLocked(u)
	}
	m.units = make(map[uint64]*unit)
	m.order = nil
	m.walls = nil
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
		id:        id,
		kind:      kind,
		role:      spec.Role,
		slot:      slot,
		owner:     owner,
		p:         p,
		v:         v,
		radius:    spec.Radius,
		hp:        spec.MaxHP,
		maxHP:     spec.MaxHP,
		vision:    spec.Vision,
		cruise:    spec.Speed,
		actor:     actor,
		inbox:     make(chan unitpkg.Event, 64),
		stop:      make(chan struct{}),
		solid:     spec.Role != unitpkg.RoleHelper,
		semi:      spec.Semi,
		face:      vec{1, 0},
		passWalls: spec.PassWalls,
		shell:     spec.Shell,
	}
	if spec.StartHP > 0 && spec.StartHP < spec.MaxHP {
		u.hp = spec.StartHP
	}
	u.aimFace()
	m.units[id] = u
	m.order = append(m.order, id)
	ctx := unitpkg.Context{ID: id, Kind: kind, Out: m.cmds}
	go runActor(u, ctx)
	return u
}

func (u *unit) aimFace() {
	if !u.semi {
		return
	}
	if u.v.len2() > 1e-8 {
		u.face = u.v.norm()
	}
	if u.face.len2() < 1e-12 {
		u.face = vec{1, 0}
	}
}

func (u *unit) setVel(v vec) {
	u.v = v
	u.aimFace()
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
		u.setVel(vec{c.VX, c.VY})
	case unitpkg.Damage:
		m.offerDamageLocked(c)
	case unitpkg.ConfirmDamage:
		m.confirmDamageLocked(c)
	case unitpkg.BlockDamage:
		off, ok := m.pendingDmg[c.Token]
		if !ok || c.UnitID != off.to {
			return
		}
		delete(m.pendingDmg, c.Token)
		delete(m.wardAbsorb, off.to)
	case unitpkg.StackMark:
		m.stackMarkLocked(c)
	case unitpkg.ClearMarks:
		m.clearMarksLocked(c)
	case unitpkg.Heal:
		u := m.units[c.UnitID]
		if u == nil || u.stopped || u.role != unitpkg.RoleFighter || c.Amount <= 0 {
			return
		}
		before := u.hp
		u.hp += c.Amount
		if u.hp > u.maxHP {
			u.hp = u.maxHP
		}
		got := u.hp - before
		if got >= 0.5 {
			m.fx = append(m.fx, unitpkg.FX{
				Name: "heal", UnitID: u.id, Kind: u.kind,
				X: u.p.X, Y: u.p.Y, Slot: u.slot, Amount: got,
			})
		}
	case unitpkg.MarkFaction:
		m.markFactionLocked(c)
	case unitpkg.ClearFactionSeen:
		u := m.units[c.UnitID]
		if u == nil {
			return
		}
		u.factionSeen = map[string]bool{}
		u.noteFaction(u.faction)
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
	case unitpkg.DespawnOwned:
		m.despawnOwnedLocked(c.OwnerID, c.Kind)
	case unitpkg.SwapOwned:
		m.swapOwnedLocked(c.UnitID)
	case unitpkg.PlaceWall:
		m.placeWallLocked(c)
	case unitpkg.FX:
		m.fx = append(m.fx, c)
	case unitpkg.Force:
		u := m.units[c.UnitID]
		if u == nil || u.stopped || !u.solid {
			return
		}
		u.setVel(u.v.add(vec{c.AX, c.AY}.mul(DT)))
	case unitpkg.Teleport:
		u := m.units[c.UnitID]
		if u == nil || u.stopped || !u.solid {
			return
		}
		u.p = vec{c.X, c.Y}
		m.constrainUnitLocked(u)
	}
}

func (m *Match) settleHitsLocked() {
	runtime.Gosched()
	m.drainCmdsLocked()
}

func (m *Match) offerDamageLocked(c unitpkg.Damage) {
	if c.Amount <= 0 {
		return
	}
	u := m.units[c.To]
	if u == nil || u.stopped || u.role != unitpkg.RoleFighter {
		return
	}
	m.dmgSeq++
	token := m.dmgSeq
	absorb := m.shellOfLocked(u.id) != nil || m.wardAbsorb[u.id]
	m.pendingDmg[token] = dmgOffer{
		from:      c.From,
		to:        c.To,
		amount:    c.Amount,
		absorb:    absorb,
		markKind:  c.MarkKind,
		markDelta: c.MarkDelta,
		markIcon:  c.MarkIcon,
	}
	m.send(u, unitpkg.IncomingDamage{
		Token:  token,
		From:   c.From,
		Amount: c.Amount,
		Time:   m.time,
		Speed:  u.v.len(),
	})
}

func (m *Match) confirmDamageLocked(c unitpkg.ConfirmDamage) {
	off, ok := m.pendingDmg[c.Token]
	if !ok || c.UnitID != off.to {
		return
	}
	delete(m.pendingDmg, c.Token)
	u := m.units[off.to]
	if u == nil || u.stopped || u.role != unitpkg.RoleFighter {
		return
	}
	if off.absorb || m.shellOfLocked(u.id) != nil || m.wardAbsorb[u.id] {
		delete(m.wardAbsorb, u.id)
		if sh := m.shellOfLocked(u.id); sh != nil {
			m.popShellLocked(sh, off.from)
			delete(m.wardAbsorb, u.id)
		}
		return
	}
	amt := off.amount
	if c.Amount > 0 && c.Amount < amt {
		amt = c.Amount
	}
	from := m.units[off.from]
	amt = m.scaleDamage(m.factionBearer(from), u, amt)
	m.fx = append(m.fx, unitpkg.FX{
		Name:   "hurt",
		UnitID: u.id,
		Kind:   u.kind,
		X:      u.p.X,
		Y:      u.p.Y,
		Slot:   u.slot,
		Amount: amt,
	})
	u.hp -= amt
	if off.markKind != "" && amt > 0 {
		delta := off.markDelta
		if delta == 0 {
			delta = 1
		}
		m.stackMarkLocked(unitpkg.StackMark{
			UnitID: u.id,
			Kind:   off.markKind,
			Delta:  delta,
			Icon:   off.markIcon,
		})
	}
	m.hitStop = HitStopFrames
	if u.hp <= 0 {
		u.hp = 0
		m.removeLocked(u)
	} else {
		m.swapOwnedLocked(u.id)
	}
}

func (m *Match) despawnOwnedLocked(owner uint64, kind string) {
	var dead []*unit
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.stopped || u.owner != owner {
			continue
		}
		if kind != "" && u.kind != kind {
			continue
		}
		if u.role == unitpkg.RoleFighter {
			continue
		}
		dead = append(dead, u)
	}
	for _, u := range dead {
		m.removeLocked(u)
	}
}

func (m *Match) stickShellsLocked() {
	var dead []*unit
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || !u.shell {
			continue
		}
		owner := m.units[u.owner]
		if owner == nil || owner.stopped {
			dead = append(dead, u)
			continue
		}
		u.p = owner.p
		u.v = owner.v
	}
	for _, u := range dead {
		m.removeLocked(u)
	}
}

func (m *Match) shellOfLocked(owner uint64) *unit {
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.shell && u.owner == owner && !u.stopped {
			return u
		}
	}
	return nil
}

func (m *Match) popShellLocked(shell *unit, from uint64) {
	if shell == nil || shell.stopped {
		return
	}
	owner := m.units[shell.owner]
	t := m.time
	m.removeLocked(shell)
	if owner != nil && !owner.stopped {
		m.wardAbsorb[owner.id] = true
		m.send(owner, unitpkg.GuardBreak{Time: t, From: from})
	}
}

func (m *Match) placeWallLocked(c unitpkg.PlaceWall) {
	if c.Life <= 0 || c.Radius <= 0 {
		return
	}
	a, b := vec{c.X1, c.Y1}, vec{c.X2, c.Y2}
	if a.sub(b).len2() < 1 {
		return
	}
	m.nextID++
	m.fx = append(m.fx, unitpkg.FX{
		Name: "wall-spawn", Kind: c.Kind, Slot: c.Slot,
		X: a.X, Y: a.Y, VX: b.X, VY: b.Y,
	})
	m.walls = append(m.walls, &barrier{
		id:     m.nextID,
		owner:  c.OwnerID,
		slot:   c.Slot,
		kind:   c.Kind,
		a:      a,
		b:      b,
		radius: c.Radius,
		until:  m.time + c.Life,
		amount: c.Amount,
		hitAt:  map[uint64]float64{},
	})
}

func (m *Match) expireWallsLocked() {
	n := 0
	for _, w := range m.walls {
		if m.time < w.until {
			m.walls[n] = w
			n++
			continue
		}
		m.fx = append(m.fx, unitpkg.FX{
			Name: "wall-fade", Kind: w.kind, Slot: w.slot,
			X: w.a.X, Y: w.a.Y, VX: w.b.X, VY: w.b.Y,
		})
	}
	m.walls = m.walls[:n]
}

func (m *Match) wallByID(id uint64) *barrier {
	for _, w := range m.walls {
		if w.id == id {
			return w
		}
	}
	return nil
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
	for tok, off := range m.pendingDmg {
		if off.to == id {
			delete(m.pendingDmg, tok)
		}
	}
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
	ignoreUW := map[uwID]bool{}
	for iter := 0; iter < maxIter && remain > 1e-8; iter++ {
		hit, ok := m.earliestHitLocked(remain, ignore, ignoreUW)
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
			a, b := m.units[hit.a], m.units[hit.b]
			if a != nil && a.shell {
				ignore[canonPair(a.owner, hit.b)] = true
				m.popShellLocked(a, hit.b)
			}
			if b != nil && b.shell {
				ignore[canonPair(b.owner, hit.a)] = true
				m.popShellLocked(b, hit.a)
			}
		}
		if hit.kind == hitBarrier {
			ignoreUW[uwID{hit.a, hit.w}] = true
		}
		remain -= hit.t
		if remain < 0 {
			remain = 0
		}
	}
	m.constrainAllLocked()
}

type uwID struct{ u, w uint64 }

func (m *Match) earliestHitLocked(dt float64, ignore map[pairID]bool, ignoreUW map[uwID]bool) (ccdHit, bool) {
	best := ccdHit{t: dt + 1}
	found := false
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || !u.solid || u.role == unitpkg.RoleHelper {
			continue
		}
		if !u.passWalls && !u.shell {
			if h, ok := sweptShapeVsHex(u.p, u.v, u.face, u.radius, dt, m.hex, u.semi); ok {
				if h.t < best.t {
					h.a = u.id
					best = h
					found = true
				}
			}
			cc, cr := colOf(u.p, u.face, u.radius, u.semi)
			for _, w := range m.walls {
				if ignoreUW[uwID{u.id, w.id}] {
					continue
				}
				R := cr + w.radius + skin
				t, nrm, ok := sweptPointVsCapsule(cc, u.v, dt, w.a, w.b, R)
				if !ok {
					continue
				}
				if t < best.t {
					best = ccdHit{kind: hitBarrier, t: t, a: u.id, w: w.id, n: nrm}
					found = true
				}
			}
		}
	}
	n := len(m.order)
	for i := 0; i < n; i++ {
		a := m.units[m.order[i]]
		if a == nil || !a.solid || a.role == unitpkg.RoleHelper {
			continue
		}
		for j := i + 1; j < n; j++ {
			b := m.units[m.order[j]]
			if b == nil || !b.solid || b.role == unitpkg.RoleHelper {
				continue
			}
			if ignore[canonPair(a.id, b.id)] {
				continue
			}
			if a.role == unitpkg.RoleProjectile && (a.slot == b.slot || (a.owner != 0 && a.owner == b.id)) {
				continue
			}
			if b.role == unitpkg.RoleProjectile && (b.slot == a.slot || (b.owner != 0 && b.owner == a.id)) {
				continue
			}
			if a.role == unitpkg.RoleFighter && b.role == unitpkg.RoleFighter {
				if m.shellOfLocked(a.id) != nil || m.shellOfLocked(b.id) != nil {
					continue
				}
			}
			t, nrm, ok := sweptPairShapes(
				a.p, a.v, a.radius, a.face, a.semi,
				b.p, b.v, b.radius, b.face, b.semi,
				dt,
			)
			if !ok {
				continue
			}
			use := !found || t < best.t-1e-12
			if !use && t <= best.t+1e-12 && (a.shell || b.shell) {
				aa, bb := m.units[best.a], m.units[best.b]
				if aa == nil || bb == nil || (!aa.shell && !bb.shell) {
					use = true
				}
			}
			if use {
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
		if u.semi {
			limit = m.hex.d[0] - semiExtent(u.face, u.radius, h.n) - skin
		}
		pen := u.p.dot(h.n) - limit
		if pen > 0 {
			u.p = u.p.sub(h.n.mul(pen))
		}
		m.send(u, unitpkg.WallHit{Time: m.time, NX: h.n.X, NY: h.n.Y})
		u.setVel(reflectVelocity(u.v, h.n))
		m.cycleFactionLocked(u)
	case hitBarrier:
		u := m.units[h.a]
		w := m.wallByID(h.w)
		if u == nil || w == nil {
			return
		}
		cc, cr := colOf(u.p, u.face, u.radius, u.semi)
		R := cr + w.radius + skin
		q := closestOnSeg(cc, w.a, w.b)
		d := cc.sub(q)
		dist := d.len()
		if dist < 1e-9 {
			d = perp(w.b.sub(w.a))
			dist = d.len()
		}
		if dist > 1e-9 && dist < R {
			corr := q.add(d.norm().mul(R)).sub(cc)
			u.p = u.p.add(corr)
		}
		m.send(u, unitpkg.WallHit{Time: m.time, NX: h.n.X, NY: h.n.Y})
		u.setVel(reflectVelocity(u.v, h.n))
		m.cycleFactionLocked(u)
		if u.role == unitpkg.RoleFighter && u.slot != w.slot && u.id != w.owner {
			if at, ok := w.hitAt[u.id]; !ok || m.time >= at {
				w.hitAt[u.id] = m.time + 0.1
				m.pending = append(m.pending, unitpkg.Damage{From: w.owner, To: u.id, Amount: w.amount})
			}
		}
	case hitPair:
		a := m.units[h.a]
		b := m.units[h.b]
		if a == nil || b == nil {
			return
		}
		n := h.n
		if n.len2() < 1e-12 {
			n = vec{1, 0}
		} else {
			n = n.norm()
		}
		ca, ra := colOf(a.p, a.face, a.radius, a.semi)
		cb, rb := colOf(b.p, b.face, b.radius, b.semi)
		delta := ca.sub(cb)
		dist := delta.len()
		target := ra + rb + skin
		pierce := a.passWalls || b.passWalls
		if !pierce && dist > 1e-9 && dist < target {
			pn := delta.norm()
			push := (target - dist) / 2
			a.p = a.p.add(pn.mul(push))
			b.p = b.p.sub(pn.mul(push))
		}
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
		if !pierce {
			m.fx = append(m.fx, unitpkg.FX{Name: name, Kind: kind, X: mid.X, Y: mid.Y, Slot: slot})
		}
		projA := a.role == unitpkg.RoleProjectile
		projB := b.role == unitpkg.RoleProjectile
		if projA || projB {
			if projA && !a.passWalls {
				a.v = vec{0, 0}
				a.solid = false
			}
			if projB && !b.passWalls {
				b.v = vec{0, 0}
				b.solid = false
			}
			if a.passWalls && !projB {
				b.setVel(n.mul(-b.v.len()))
			} else if b.passWalls && !projA {
				a.setVel(n.mul(a.v.len()))
			}
		} else {
			sa, sb := a.v.len(), b.v.len()
			a.setVel(n.mul(sa))
			b.setVel(n.mul(-sb))
		}
	}
}

func (m *Match) constrainAllLocked() {
	for _, id := range m.order {
		m.constrainUnitLocked(m.units[id])
	}
}

func (m *Match) constrainUnitLocked(u *unit) {
	if u == nil || !u.solid || u.passWalls || u.shell {
		return
	}
	for i := 0; i < 6; i++ {
		n := m.hex.n[i]
		ext := u.radius
		if u.semi {
			ext = semiExtent(u.face, u.radius, n)
		}
		limit := m.hex.d[0] - ext - skin
		pen := u.p.dot(n) - limit
		if pen > 0 {
			u.p = u.p.sub(n.mul(pen))
		}
	}
	cc, cr := colOf(u.p, u.face, u.radius, u.semi)
	for _, w := range m.walls {
		R := cr + w.radius + skin
		q := closestOnSeg(cc, w.a, w.b)
		d := cc.sub(q)
		dist := d.len()
		if dist >= R {
			continue
		}
		n := d
		if n.len2() < 1e-12 {
			n = perp(w.b.sub(w.a))
		}
		if n.len2() < 1e-12 {
			continue
		}
		corr := q.add(n.norm().mul(R)).sub(cc)
		u.p = u.p.add(corr)
		cc, cr = colOf(u.p, u.face, u.radius, u.semi)
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
