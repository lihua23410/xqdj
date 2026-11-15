package sim

import (
	"encoding/json"
	"maps"
	"math"
	"os"
	"time"

	unitpkg "xqdj/internal/unit"
)

func agentLog(hyp, loc, msg string, data map[string]any) {
	// #region agent log
	b, err := json.Marshal(map[string]any{
		"sessionId": "7e8d9b", "runId": "hp1", "hypothesisId": hyp,
		"location": loc, "message": msg, "data": data, "timestamp": time.Now().UnixMilli(),
	})
	if err != nil {
		return
	}
	f, err := os.OpenFile("/home/lihua/xqdj/.cursor/debug-7e8d9b.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	_, _ = f.Write(append(b, '\n'))
	_ = f.Close()
	// #endregion
}

const (
	kindHFF  = "大肥鱼(幻方量化)"
	kindRice = "大肥鱼白饭"
	kindOrca = "大肥鱼"
	paperSec = 10.0
	paperCap = 10 * TickHz * 8
)

// paperRun 是一场丢掉的子对局。真实场地不改，结束只留下假血掉量。
type paperRun struct {
	child      *Match
	host       uint64
	long       bool
	until      float64
	frames     int
	openFrames int
	closing    bool
	rows       []paperRow
	view       *paperView
}

type paperRow struct {
	id     uint64
	slot   int
	role   string
	mortal bool
}

type paperLedger struct {
	born map[uint64]bool
	loss map[uint64]float64
}

func (m *Match) stepSandboxLocked() bool {
	if m.paper != nil && m.paper.closing {
		m.closePaperLocked()
	}
	if m.paper != nil {
		m.paper.advance()
		return true
	}
	if m.paperClose != nil && m.seq-m.paperCloseSeq < paperFreezeTicks+paperFlightTicks {
		return true
	}
	if m.hitStop == 0 && len(m.settleQ) > 0 {
		q := m.settleQ[0]
		hp := -1.0
		if u := m.units[q.To]; u != nil {
			hp = u.hp
		}
		agentLog("B", "paper.go:stepSandbox", "offer settle quote", map[string]any{
			"from": q.From, "to": q.To, "amount": q.Amount, "hp": hp, "left": len(m.settleQ),
		})
		token := m.offerDamageLocked(q)
		m.linkPaperImpactLocked(q, token)
		m.settleQ = m.settleQ[1:]
		if len(m.settleQ) == 0 {
			m.releaseSettleLocked()
		}
	}
	return false
}

func (m *Match) maybeArmPaperLocked() bool {
	if m.paper != nil || m.settling || len(m.ready) == 0 {
		return false
	}
	id := m.popReadyLocked()
	u := m.units[id]
	if u == nil || u.stopped {
		delete(m.paperHold, id)
		return false
	}
	m.arming = true
	m.pendingPaper = unitpkg.Paper{}
	m.send(u, unitpkg.Stare{Time: m.time, Self: u.snap(), Field: m.liveFieldLocked()})
	m.drainCmdsLocked()
	m.arming = false
	p := m.pendingPaper
	if p.UnitID != u.id || p.Stock == "" || !stockKnown(p.Stock) {
		m.send(u, unitpkg.PaperDone{Time: m.time})
		m.paperRelease = u.id
		m.drainCmdsLocked()
		return false
	}
	m.paper = m.forkPaperLocked(u, p)
	if m.paper == nil {
		m.send(u, unitpkg.PaperDone{Time: m.time})
		m.paperRelease = u.id
		m.drainCmdsLocked()
		return false
	}
	m.paper.advance()
	return true
}

func stockKnown(kind string) bool {
	spec, ok := unitpkg.Lookup(kind)
	return ok && spec.Fighter && kind != kindHFF
}

func (m *Match) queueReadyLocked(id uint64) {
	if m.sandbox {
		return
	}
	u := m.units[id]
	if u == nil || u.stopped || u.kind != kindHFF {
		return
	}
	if m.paperHold[id] {
		return
	}
	if m.paperHold == nil {
		m.paperHold = map[uint64]bool{}
	}
	m.paperHold[id] = true
	m.ready = append(m.ready, id)
}

func (m *Match) popReadyLocked() uint64 {
	sortReady(m, m.ready)
	id := m.ready[0]
	m.ready = append([]uint64(nil), m.ready[1:]...)
	return id
}

func sortReady(m *Match, ids []uint64) {
	for i := 1; i < len(ids); i++ {
		j := i
		for j > 0 && readyBefore(m, ids[j], ids[j-1]) {
			ids[j], ids[j-1] = ids[j-1], ids[j]
			j--
		}
	}
}

func readyBefore(m *Match, a, b uint64) bool {
	ua, ub := m.units[a], m.units[b]
	if ua == nil || ub == nil {
		return ua != nil
	}
	if ua.slot != ub.slot {
		return ua.slot < ub.slot
	}
	if ua.seat != ub.seat {
		return ua.seat < ub.seat
	}
	return ua.id < ub.id
}

func (m *Match) forkPaperLocked(host *unit, p unitpkg.Paper) *paperRun {
	g := unitpkg.NewFork()
	shared := m.shared.ForkState(g)
	child := &Match{
		phase:      PhaseRunning,
		slots:      m.slots,
		lineup:     m.lineup,
		team:       m.team,
		units:      map[uint64]*unit{},
		cmds:       make(chan unitpkg.Cmd, 512),
		actorWork:  newActorWork(),
		shared:     shared,
		time:       m.time,
		spec:       m.spec,
		seed:       m.seed,
		headless:   true,
		skipProbe:  true,
		logMute:    true,
		nextID:     m.nextID,
		sandbox:    true,
		pendingDmg: map[uint64]dmgOffer{},
		wardAbsorb: map[uint64]bool{},
		logNames:   map[uint64]string{},
		ledger:     &paperLedger{born: map[uint64]bool{}, loss: map[uint64]float64{}},
	}
	if rng := unitpkg.ForkRNG(m.rng, g); rng != nil {
		child.rng = rng
	}
	if child.rng == nil {
		child.rng = unitpkg.EngineRNG(m.seed ^ 0x51ed)
	}
	rows := make([]paperRow, 0, len(m.order))
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.stopped {
			continue
		}
		cu := cloneUnitShell(u)
		child.ledger.born[cu.id] = true
		if u.id == host.id {
			cu.embodyLocked(p.Stock, child.shared, child.seed)
		} else if cu.actor != nil {
			cu.actor = unitpkg.ForkActor(u.actor, g)
		}
		if cu.actor == nil {
			continue
		}
		if u.id != host.id {
			cu.hp = cu.maxHP
		}
		cu.inbox = newActorInbox(child.actorWork)
		cu.stop = make(chan struct{})
		cu.exited = make(chan struct{})
		cu.stopped = false
		child.units[cu.id] = cu
		child.order = append(child.order, cu.id)
		child.actors = append(child.actors, cu)
		rows = append(rows, paperRow{id: u.id, slot: u.slot, role: u.role, mortal: u.mortal})
		ctx := unitpkg.Context{
			State: child.shared, ID: cu.id, Kind: cu.kind, Out: child.cmds,
			Stream: cu.stream, Field: child.liveFieldLocked(),
		}.WithBirths()
		go runActor(cu, ctx)
	}
	for _, w := range m.walls {
		child.walls = append(child.walls, cloneBarrier(w))
	}
	if child.units[host.id] == nil {
		child.stopAllLocked()
		return nil
	}
	cu := child.units[host.id]
	agentLog("A", "paper.go:fork", "forked host", map[string]any{
		"samePtr": cu == host, "parentHP": host.hp, "childHP": cu.hp,
		"parentKind": host.kind, "childKind": cu.kind, "long": p.Long, "stock": p.Stock,
	})
	run := &paperRun{
		child: child,
		host:  host.id,
		long:  p.Long,
		until: m.time + paperSec,
		rows:  rows,
	}
	if !m.headless {
		preview := cu.snap()
		run.view = &paperView{
			Stock: p.Stock, Long: p.Long, Line: p.Line,
			HostID: host.id, HostX: host.p.X, HostY: host.p.Y,
			Preview: &preview, Duration: paperSec,
		}
	}
	return run
}

func cloneUnitShell(u *unit) *unit {
	c := *u
	c.inbox = nil
	c.stop = nil
	c.exited = nil
	c.stopped = false
	if u.marks != nil {
		c.marks = make(map[string]*stackMark, len(u.marks))
		for k, mark := range u.marks {
			cp := *mark
			if mark.hits != nil {
				cp.hits = append([]float64(nil), mark.hits...)
			}
			c.marks[k] = &cp
		}
	}
	if u.threats != nil {
		c.threats = append([]unitpkg.Threat(nil), u.threats...)
	}
	if u.fsList != nil {
		c.fsList = append([]impulseFS(nil), u.fsList...)
	}
	if u.cruiseFS != nil {
		cp := *u.cruiseFS
		if u.cruiseFS.components != nil {
			cp.components = maps.Clone(u.cruiseFS.components)
		}
		c.cruiseFS = &cp
	}
	if u.factionSeen != nil {
		c.factionSeen = maps.Clone(u.factionSeen)
	}
	if u.factionBarrage != nil {
		c.factionBarrage = append([]string(nil), u.factionBarrage...)
	}
	return &c
}

func cloneBarrier(w *barrier) *barrier {
	if w == nil {
		return nil
	}
	c := *w
	if w.hitAt != nil {
		c.hitAt = maps.Clone(w.hitAt)
	}
	if w.rammed != nil {
		c.rammed = maps.Clone(w.rammed)
	}
	if w.riding != nil {
		c.riding = maps.Clone(w.riding)
	}
	return &c
}

func (u *unit) embodyLocked(stock string, state *unitpkg.MatchState, seed uint64) {
	spec, ok := unitpkg.Lookup(stock)
	if !ok {
		return
	}
	u.actor = unitpkg.NewActor(stock, unitpkg.SpawnInfo{
		State: state, MatchSeed: seed, X: u.p.X, Y: u.p.Y,
		OwnerID: u.owner, Slot: u.slot, Stream: u.stream,
	})
	u.kind = stock
	u.role = spec.Role
	u.radius = spec.Radius
	u.maxHP = spec.MaxHP
	u.hp = spec.MaxHP
	u.vision = spec.Vision
	u.observeThreats = spec.ObserveThreats
	u.cruise = spec.Speed
	u.velM = 1
	u.solid = spec.Role != unitpkg.RoleHelper && !spec.Nonsolid
	u.hidden = spec.Hidden
	u.semi = spec.Semi
	u.passWalls = spec.PassWalls
	u.passInnerWalls = spec.PassInnerWalls
	u.bodiesOnly = spec.BodiesOnly
	u.breakWalls = spec.BreakWalls
	u.mortal = spec.Mortal
	u.aimPriority = unitpkg.DefaultAimPriority(spec)
	u.shell = spec.Shell
	u.attach = spec.Attach
	u.attachFriends = spec.AttachFriends
	u.pass = spec.Pass
	u.arcSpan = spec.ArcSpan
	u.arcInner = spec.ArcInner
	u.visionBlock = spec.VisionBlock
	u.zeroOwnedKind = spec.ZeroVelocityWhileOwned
	if spec.Role == unitpkg.RoleFighter || spec.Cruise {
		u.cruiseFS = newCruiseFS(u.v, spec.Speed)
		u.syncCruise()
	} else {
		u.cruiseFS = nil
	}
	u.aimFace()
}

func (m *Match) closePaperLocked() {
	run := m.paper
	m.paper = nil
	if run == nil || run.child == nil {
		return
	}
	loss := map[uint64]float64{}
	if run.child.ledger != nil {
		maps.Copy(loss, run.child.ledger.loss)
	}
	run.child.stopAllLocked()
	m.settleQ = settleQuotes(run.host, run.rows, loss, run.long)
	if run.view != nil {
		m.preparePaperCloseLocked(run)
	}
	m.settleHost = run.host
	parentHP := -1.0
	if u := m.units[run.host]; u != nil {
		parentHP = u.hp
	}
	quotes := make([]map[string]any, 0, len(m.settleQ))
	for _, q := range m.settleQ {
		quotes = append(quotes, map[string]any{"from": q.From, "to": q.To, "amount": q.Amount})
	}
	agentLog("C", "paper.go:close", "sandbox closed", map[string]any{
		"loss": loss, "long": run.long, "parentHP": parentHP, "quotes": quotes,
	})
	if len(m.settleQ) == 0 {
		m.releaseSettleLocked()
		return
	}
	m.settling = true
}

func (m *Match) releaseSettleLocked() {
	m.settling = false
	id := m.settleHost
	m.settleHost = 0
	if u := m.units[id]; u != nil && !u.stopped {
		m.send(u, unitpkg.PaperDone{Time: m.time})
	}
	m.paperRelease = id
}

// paperProfitScale 是她赚到的那一笔写回对手的倍数。她自己付的差额仍是一倍。
const paperProfitScale = 2

func settleQuotes(host uint64, rows []paperRow, loss map[uint64]float64, long bool) []unitpkg.Damage {
	var hostSlot int
	var have bool
	for _, row := range rows {
		if row.id == host {
			hostSlot = row.slot
			have = true
			break
		}
	}
	if !have {
		return nil
	}
	hers := loss[host]
	var self float64
	var out []unitpkg.Damage
	for _, row := range rows {
		if row.id == host || row.slot == hostSlot {
			continue
		}
		if row.role != unitpkg.RoleFighter && !row.mortal {
			continue
		}
		delta := hers - loss[row.id]
		if math.Abs(delta) < 1e-9 {
			continue
		}
		if long == (delta > 0) {
			self += math.Abs(delta)
			continue
		}
		out = append(out, unitpkg.Damage{From: host, To: row.id, Amount: math.Abs(delta) * paperProfitScale})
	}
	if self > 0 {
		out = append([]unitpkg.Damage{{From: host, To: host, Amount: self}}, out...)
	}
	return out
}

func (m *Match) notePaperLossLocked(id uint64, amt float64) {
	if m.ledger == nil || amt <= 0 {
		return
	}
	m.ledger.loss[id] += amt
}

func (m *Match) notePaperWipeLocked(u *unit, originID uint64) {
	if m.ledger == nil || u == nil || !m.ledger.born[u.id] || u.hp <= 0 || !u.takesHit() {
		return
	}
	if originID == 0 || originID == u.id {
		return
	}
	m.ledger.loss[u.id] += u.hp
}

func (m *Match) stopPaperLocked() {
	if m.paper != nil && m.paper.child != nil {
		m.paper.child.stopAllLocked()
	}
	m.paper = nil
	m.paperClose = nil
	m.ready = nil
	m.settleQ = nil
	m.settling = false
	m.paperHold = nil
}

func (m *Match) touchForHFFLocked(u *unit, snaps, nearby []unitpkg.Snapshot) []unitpkg.Snapshot {
	if u == nil || u.kind != kindHFF {
		return nearby
	}
	have := map[uint64]bool{}
	for _, o := range nearby {
		have[o.ID] = true
	}
	var bowls []unitpkg.Snapshot
	for _, o := range snaps {
		if o.Kind != kindRice || o.ID == u.id || have[o.ID] {
			continue
		}
		if math.Hypot(o.X-u.p.X, o.Y-u.p.Y) <= u.radius+o.Radius {
			bowls = append(bowls, o)
			nearby = append(nearby, o)
			have[o.ID] = true
		}
	}
	for _, o := range snaps {
		if o.Kind != kindOrca || o.ID == u.id || have[o.ID] {
			continue
		}
		for _, b := range bowls {
			if math.Hypot(o.X-b.X, o.Y-b.Y) <= o.Radius+b.Radius {
				nearby = append(nearby, o)
				have[o.ID] = true
				break
			}
		}
	}
	return nearby
}

func (m *Match) collectPaperLocked() (units []unitpkg.Snapshot, walls []wallSnap, fx []unitpkg.FX, hitStop int, sceneTime float64) {
	child := m.paper.child
	child.mu.Lock()
	units = make([]unitpkg.Snapshot, 0, len(child.order))
	for _, id := range child.order {
		u := child.units[id]
		if u == nil || u.stopped || !u.inSnapshot() {
			continue
		}
		units = append(units, u.snap())
	}
	fx = append([]unitpkg.FX(nil), child.fx...)
	walls = make([]wallSnap, 0, len(child.walls))
	for _, w := range child.walls {
		walls = append(walls, wallSnap{
			ID: w.id, X1: w.a.X, Y1: w.a.Y, X2: w.b.X, Y2: w.b.Y,
			Radius: w.radius, Kind: w.kind, Slot: w.slot,
			Hard: w.hard, Square: w.square, Field: w.field,
			VisionBlock: w.visionBlock,
		})
	}
	hitStop = child.hitStop
	sceneTime = child.time
	child.mu.Unlock()
	return
}
