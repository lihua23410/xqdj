package sim

import (
	"encoding/json"
	"math"
	"math/rand/v2"
	"runtime"
	"strings"
	"sync"
	"time"

	"xqdj/internal/replay"
	unitpkg "xqdj/internal/unit"
	奇美拉 "xqdj/map/chimera"
)

type Phase string

const (
	PhaseSelect  Phase = "select"
	PhaseRunning Phase = "running"
	PhasePaused  Phase = "paused"
	PhaseEnded   Phase = "ended"
)

type unit struct {
	id              uint64
	stream          uint64
	kind            string
	role            string
	slot            int
	seat            int
	owner           uint64
	p, v            vec
	radius          float64
	hp              float64
	maxHP           float64
	actor           unitpkg.Actor
	inbox           chan unitpkg.Event
	stop            chan struct{}
	exited          chan struct{} // 角色协程退出时关闭，停场要等它
	stopped         bool
	solid           bool
	vision          float64
	cruise          float64
	cruiseFS        *cruiseFS
	velM            float64 // 已经乘进当前速度的巡航 M，出生为 1
	fsList          []impulseFS
	decelT          float64
	stunUntil       float64
	semi            bool
	face            vec
	passWalls       bool
	breakWalls      bool
	mortal          bool
	aimPriority     uint8
	pass            bool
	stand           bool
	stun            bool
	held            bool
	holdUntil       float64
	holdVel         vec
	noFrameFreeze   bool
	noHealthNumbers bool
	visionBlock     bool
	shell           bool
	attach          bool
	arcSpan         float64
	arcInner        float64
	faction         string
	factionCycle    bool
	factionAmpOut   float64
	factionAmpIn    float64
	factionCollect  bool
	factionSeen     map[string]bool
	factionNext     float64
	factionBarrage  []string
	factionBlastR   float64
	factionBlastD   float64
	marks           map[string]*stackMark
	tap             func(unitpkg.Event)
	nailWall        uint64
	nailLocal       vec
}

type spawnSpot struct {
	p    vec
	r    float64
	side int
}

const (
	minSideN = 1
	maxSideN = 3
	teamSep  = 220.0
	sepStep  = 40.0
	spawnPad = 8.0
)

type barrier struct {
	id           uint64
	owner        uint64
	slot         int
	kind         string
	a, b         vec
	prevA, prevB vec
	radius       float64
	until        float64
	amount       float64
	hitAt        map[uint64]float64
	hard         bool
	square       bool
	visionBlock  bool
	field        bool
	spin         float64 // 弧度每秒，逆时针为正。0 表示不转。
	ang          float64
	base         float64
	pivot        vec
	halfLen      float64
	vx, vy       float64
	hitGap       float64
	withOwner    bool
	ram          float64
	rammed       map[uint64]struct{}
	stunR        float64
	stunDur      float64
	riding       map[uint64]struct{}
	passOwner    bool
}

type wallSnap struct {
	ID          uint64  `json:"id"`
	X1          float64 `json:"x1"`
	Y1          float64 `json:"y1"`
	X2          float64 `json:"x2"`
	Y2          float64 `json:"y2"`
	Radius      float64 `json:"radius"`
	Kind        string  `json:"kind"`
	Slot        int     `json:"slot"`
	Hard        bool    `json:"hard,omitempty"`
	Square      bool    `json:"square,omitempty"`
	Field       bool    `json:"field,omitempty"`
	VisionBlock bool    `json:"visionBlock,omitempty"`
}

type Match struct {
	mu          sync.Mutex
	phase       Phase
	slots       [2][]string
	teamPick    [2][]string // 团队赛人选。当场名单会清空，这一份留下来
	lineup      [2][]string // 开局名单，胜负文案和血条用
	team        bool
	units       map[uint64]*unit
	order       []uint64
	actors      []*unit // 本场开过的所有角色（离场的也留着，停场要等它们的协程真退出）
	cmds        chan unitpkg.Cmd
	nextID      uint64
	time        float64
	spec        fieldSpec
	winner      string
	winnerID    uint64
	seq         uint64
	rng         *rand.Rand
	seed        uint64
	pinned      bool
	skipProbe   bool
	spawnSilent bool
	rootN       uint64
	probeTape   *unitpkg.ProbeTape
	recap       strings.Builder
	trace       strings.Builder
	logReady    bool
	logMute     bool
	logNames    map[uint64]string
	logLeapAt   map[uint64]float64 // 上一拍瞬移的时刻。连续每拍瞬移不当成一下子挪走
	logLeaps    []logLeap
	record      []json.RawMessage // 本场观众快照，按拍记；导出视频用
	recordDone  bool
	fx          []unitpkg.FX
	hitStop     int
	walls       []*barrier
	pending     []unitpkg.Damage
	dmgSeq      uint64
	pendingDmg  map[uint64]dmgOffer
	wardAbsorb  map[uint64]bool

	// 开场演出与真局初态单独归档，不跟随直播演出回执清除。
	recordInitial       json.RawMessage
	recordPresentations []replay.Presentation
	pendingOpening      []replay.Presentation
	activeOpening       []replay.Presentation
	openingUntil        time.Time
	presentationN       uint64
}

type dmgOffer struct {
	from      uint64
	to        uint64
	amount    float64
	absorb    bool
	noFreeze  bool
	markKind  string
	markDelta int
	markIcon  string
	// credit 是「这笔伤害记谁的功」，在发起那一刻就定下来。
	// 不能等确认时才去找发起者：剑弧、子弹这类发起者往往发完伤害就自己消失了，
	// 到确认那一拍 m.units[from] 已经是 nil，击杀就记丢了。
	credit uint64
	// 提出那一刻双方的速率，和还挂在身上的瞬时推动。确认时再抄就晚了。
	atkSpeed, defSpeed float64
	atkImpulse         []float64
	defImpulse         []float64
}

func NewMatch() *Match {
	return newMatch(0, false)
}

func NewMatchSeeded(seed uint64) *Match {
	return newMatch(seed, true)
}

func newMatch(seed uint64, pinned bool) *Match {
	kinds := unitpkg.FighterKinds()
	var slots [2][]string
	if len(kinds) >= 2 {
		slots = [2][]string{{kinds[0]}, {kinds[1]}}
	}
	spec := defaultField()
	return &Match{
		phase:      PhaseSelect,
		slots:      slots,
		units:      make(map[uint64]*unit),
		cmds:       make(chan unitpkg.Cmd, 512),
		spec:       spec,
		rng:        unitpkg.EngineRNG(seed),
		seed:       seed,
		pinned:     pinned,
		logNames:   map[uint64]string{},
		pendingDmg: make(map[uint64]dmgOffer),
		wardAbsorb: make(map[uint64]bool),
	}
}

func (m *Match) SnapshotJSON() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snapshotJSONLocked(true)
}

func (m *Match) snapshotJSONLocked(withOpening bool) []byte {
	type ujson struct {
		unitpkg.Snapshot
	}
	msg := struct {
		Type          string                      `json:"type"`
		Phase         Phase                       `json:"phase"`
		Slots         [2][]string                 `json:"slots"`
		Roster        [2][]string                 `json:"roster"`
		Team          bool                        `json:"team"`
		Kinds         []string                    `json:"kinds"`
		Looks         map[string]unitpkg.Look     `json:"looks"`
		Packs         map[string]unitpkg.PackInfo `json:"packs"`
		Winner        string                      `json:"winner"`
		WinnerID      uint64                      `json:"winnerId"`
		Time          float64                     `json:"time"`
		Seq           uint64                      `json:"seq"`
		HexR          float64                     `json:"hexRadius"`
		Fields        []string                    `json:"fields"`
		Field         string                      `json:"field"`
		Shape         string                      `json:"fieldShape"`
		HitStop       int                         `json:"hitStop"`
		LogReady      bool                        `json:"logReady"`
		Units         []unitpkg.Snapshot          `json:"units"`
		Walls         []wallSnap                  `json:"walls"`
		Effects       []unitpkg.FX                `json:"effects"`
		Presentations []replay.Presentation       `json:"presentations,omitempty"`
	}{
		Type:     "state",
		Phase:    m.phase,
		Slots:    sidesOrEmpty(m.slots),
		Roster:   sidesOrEmpty(m.lineup),
		Team:     m.team,
		Kinds:    unitpkg.FighterKinds(),
		Looks:    unitpkg.Looks(),
		Packs:    unitpkg.Packs(),
		Winner:   m.winner,
		WinnerID: m.winnerID,
		Time:     m.time,
		Seq:      m.seq,
		HexR:     m.spec.extent,
		Fields:   m.listFieldsLocked(),
		Field:    m.spec.name,
		Shape:    m.spec.shape,
		HitStop:  m.hitStop,
		LogReady: m.logReady,
		Units:    make([]unitpkg.Snapshot, 0, len(m.order)),
		Walls:    make([]wallSnap, 0, len(m.walls)),
		Effects:  append([]unitpkg.FX(nil), m.fx...),
	}
	if withOpening {
		msg.Presentations = m.takeOpeningLocked()
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
			Hard: w.hard, Square: w.square, Field: w.field,
			VisionBlock: w.visionBlock,
		})
	}
	if m.phase == PhaseSelect {
		for i, h := range m.spec.hard {
			msg.Walls = append(msg.Walls, wallSnap{
				ID: uint64(i + 1), X1: h.x1, Y1: h.y1, X2: h.x2, Y2: h.y2,
				Radius: h.halfW, Hard: true, Square: true, Field: true,
			})
		}
		for i, c := range m.spec.caps {
			msg.Walls = append(msg.Walls, wallSnap{
				ID: uint64(len(m.spec.hard) + i + 1),
				X1: c.x1, Y1: c.y1, X2: c.x2, Y2: c.y2,
				Radius: c.radius, Field: true,
			})
		}
	}
	b, _ := json.Marshal(msg)
	return b
}

func (u *unit) snap() unitpkg.Snapshot {
	return unitpkg.Snapshot{
		ID:              u.id,
		Kind:            u.kind,
		Role:            u.role,
		X:               u.p.X,
		Y:               u.p.Y,
		VX:              u.v.X,
		VY:              u.v.Y,
		Radius:          u.radius,
		HP:              u.hp,
		MaxHP:           u.maxHP,
		Vision:          u.vision,
		OwnerID:         u.owner,
		Slot:            u.slot,
		Seat:            u.seat,
		Semi:            u.semi,
		FaceX:           u.face.X,
		FaceY:           u.face.Y,
		PassWalls:       u.passWalls,
		Mortal:          u.mortal,
		BreakWalls:      u.breakWalls,
		ArcSpan:         u.arcSpan,
		ArcInner:        u.arcInner,
		Faction:         u.faction,
		Seen:            u.seenList(),
		Marks:           u.markList(),
		AimPriority:     u.aimPriority,
		Nonsolid:        !u.solid,
		VisionBlock:     u.visionBlock,
		NoHealthNumbers: u.noHealthNumbers,
	}
}

func (u *unit) shape() ccdShape {
	return ccdShape{
		p: u.p, v: u.v, face: u.face,
		r: u.radius, semi: u.semi,
		arcSpan: u.arcSpan, arcInner: u.arcInner,
	}
}

func (u *unit) inSnapshot() bool {
	if u == nil || u.stopped {
		return false
	}
	return u.solid || u.role == unitpkg.RoleHelper || u.role == unitpkg.RoleFighter
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

func (m *Match) SetField(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.phase != PhaseSelect {
		return
	}
	if !m.team && name == 奇美拉.Name {
		return
	}
	spec, ok := lookupField(name)
	if !ok {
		return
	}
	m.spec = spec
}

func sidesOrEmpty(in [2][]string) [2][]string {
	var out [2][]string
	for i := 0; i < 2; i++ {
		out[i] = append([]string{}, in[i]...)
		if out[i] == nil {
			out[i] = []string{}
		}
	}
	return out
}

func (m *Match) SetSlot(slot int, kind string) {
	if slot < 0 || slot > 1 || !unitpkg.IsFighterKind(kind) {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.phase != PhaseSelect {
		return
	}
	if m.team {
		if len(m.slots[slot]) >= maxSideN {
			return
		}
		m.slots[slot] = append(m.slots[slot], kind)
		m.rememberTeamLocked()
		return
	}
	m.slots[slot] = []string{kind}
}

func (m *Match) RemoveSeat(side, index int) {
	if side < 0 || side > 1 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.phase != PhaseSelect || !m.team {
		return
	}
	cur := m.slots[side]
	if index < 0 || index >= len(cur) {
		return
	}
	m.slots[side] = append(cur[:index:index], cur[index+1:]...)
	m.rememberTeamLocked()
}

func (m *Match) SetSeat(side, index int, kind string) {
	if side < 0 || side > 1 || !unitpkg.IsFighterKind(kind) {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.phase != PhaseSelect {
		return
	}
	if index < 0 || index >= len(m.slots[side]) {
		return
	}
	m.slots[side][index] = kind
	m.rememberTeamLocked()
}

func (m *Match) SetSideLen(side, n int) {
	if side < 0 || side > 1 {
		return
	}
	if n < minSideN {
		n = minSideN
	}
	if n > maxSideN {
		n = maxSideN
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.phase != PhaseSelect || !m.team {
		return
	}
	cur := append([]string{}, m.slots[side]...)
	kinds := unitpkg.FighterKinds()
	for len(cur) < n {
		cur = append(cur, nextSideKind(kinds, cur))
	}
	m.slots[side] = cur[:n]
	m.rememberTeamLocked()
}

func (m *Match) UseTeam() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.phase != PhaseSelect {
		return
	}
	m.team = true
	if m.spec.name == "" || m.spec.name == "六边形" {
		if spec, ok := lookupField(奇美拉.Name); ok {
			m.spec = spec
		}
	}
	m.slots[0] = []string{}
	m.slots[1] = []string{}
	m.restoreTeamLocked()
}

func (m *Match) rememberTeamLocked() {
	if !m.team {
		return
	}
	for i := 0; i < 2; i++ {
		m.teamPick[i] = append([]string{}, m.slots[i]...)
	}
}

func (m *Match) restoreTeamLocked() {
	if len(m.teamPick[0]) == 0 && len(m.teamPick[1]) == 0 {
		return
	}
	for i := 0; i < 2; i++ {
		m.slots[i] = append([]string{}, m.teamPick[i]...)
	}
}

func (m *Match) UseDuel() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.phase != PhaseSelect {
		return
	}
	m.team = false
	if m.spec.name == 奇美拉.Name {
		if spec, ok := lookupField("六边形"); ok {
			m.spec = spec
		}
	}
	kinds := unitpkg.FighterKinds()
	for side := 0; side < 2; side++ {
		k := ""
		if len(m.slots[side]) > 0 && unitpkg.IsFighterKind(m.slots[side][0]) {
			k = m.slots[side][0]
		} else if side < len(kinds) {
			k = kinds[side]
		} else if len(kinds) > 0 {
			k = kinds[0]
		}
		if k == "" {
			m.slots[side] = []string{}
			continue
		}
		m.slots[side] = []string{k}
	}
}

func takeKinds(kinds []string, start, n int) []string {
	out := make([]string, 0, n)
	if len(kinds) == 0 || n <= 0 {
		return out
	}
	for i := 0; i < n; i++ {
		out = append(out, kinds[(start+i)%len(kinds)])
	}
	return out
}

func nextSideKind(kinds, cur []string) string {
	if len(kinds) == 0 {
		return ""
	}
	start := 0
	if len(cur) > 0 {
		last := cur[len(cur)-1]
		for i, k := range kinds {
			if k == last {
				start = i + 1
				break
			}
		}
	}
	for step := 0; step < len(kinds); step++ {
		k := kinds[(start+step)%len(kinds)]
		seen := false
		for _, c := range cur {
			if c == k {
				seen = true
				break
			}
		}
		if !seen {
			return k
		}
	}
	return kinds[start%len(kinds)]
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
	if m.team && (len(m.slots[0]) == 0 || len(m.slots[1]) == 0) {
		return
	}
	if !m.pinned {
		m.seed = unitpkg.NextStream()
	}
	m.rng = unitpkg.EngineRNG(m.seed)
	if !m.skipProbe {
		m.runAcausalProbeLocked()
	}
	m.logMute = true
	m.resetLocked()
	m.installFieldLocked()
	m.lineup[0] = append([]string{}, m.slots[0]...)
	m.lineup[1] = append([]string{}, m.slots[1]...)
	m.writeLogHeaderLocked()
	m.logMute = false
	m.noteWallsBuiltLocked()
	sep := 0.0
	if m.team {
		sep = teamSep
	}
	var spots []spawnSpot
	for side := 0; side < 2; side++ {
		for seat, kind := range m.slots[side] {
			spec, ok := unitpkg.Lookup(kind)
			if !ok {
				continue
			}
			p := m.placeLocked(spec.Radius, side, spots, sep)
			ang := m.rng.Float64() * 2 * math.Pi
			v := vec{math.Cos(ang) * spec.Speed, math.Sin(ang) * spec.Speed}
			m.spawnFighterLocked(kind, side, seat, p, v)
			spots = append(spots, spawnSpot{p: p, r: spec.Radius, side: side})
		}
	}
	m.flipAcausalLocked()
	if m.spec.bootKind != "" {
		m.addUnitLocked(m.spec.bootKind, vec{}, vec{}, 0, 0)
	}
	m.phase = PhaseRunning
	// 演出只存一次；真局初态在第一拍推进之前固定下来。
	if !m.skipProbe && len(m.pendingOpening) > 0 {
		m.activeOpening = m.pendingOpening
		m.pendingOpening = nil
		m.recordPresentations = append([]replay.Presentation(nil), m.activeOpening...)
		m.recordInitial = m.snapshotJSONLocked(false)
		duration := 0.0
		for _, p := range m.activeOpening {
			duration += p.Duration
		}
		m.openingUntil = time.Now().Add(time.Duration((duration + 2) * float64(time.Second)))
	}
}

func (m *Match) listFieldsLocked() []string {
	names := fieldNames()
	if m.team {
		return names
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		if n == 奇美拉.Name {
			continue
		}
		out = append(out, n)
	}
	return out
}

func (m *Match) chimeraSides() bool {
	return m.team && m.spec.name == 奇美拉.Name
}

func (m *Match) placeLocked(radius float64, side int, others []spawnSpot, minSep float64) vec {
	gap := minSep
	for {
		if p, ok := m.tryPlaceLocked(radius, side, others, gap); ok {
			return p
		}
		if gap <= 0 {
			break
		}
		gap -= sepStep
		if gap < 0 {
			gap = 0
		}
	}
	if m.chimeraSides() {
		cx, cy := 奇美拉.East()
		if side == 0 {
			cx, cy = 奇美拉.West()
		}
		x, y := m.liveFieldLocked().Clamp(cx, cy, radius+4)
		return vec{x, y}
	}
	x, y := m.liveFieldLocked().Clamp(120, 0, radius+4)
	return vec{x, y}
}

func (m *Match) tryPlaceLocked(radius float64, side int, others []spawnSpot, gap float64) (vec, bool) {
	f := m.liveFieldLocked()
	half := unitpkg.HexRadius
	cx, cy := 0.0, 0.0
	regional := m.chimeraSides()
	if regional {
		if side == 0 {
			cx, cy = 奇美拉.West()
		} else {
			cx, cy = 奇美拉.East()
		}
	}
	tries := 128
	if regional {
		tries = 256
	}
	for try := 0; try < tries; try++ {
		var x, y float64
		if regional {
			x = cx + (m.rng.Float64()*2-1)*half
			y = cy + (m.rng.Float64()*2-1)*half
			if !奇美拉.RegionContains(side, x, y, radius+4) || !f.Walkable(x, y, radius+4) {
				continue
			}
		} else {
			var ok bool
			x, y, ok = f.RandomWalkable(m.rng, radius+4)
			if !ok {
				break
			}
		}
		p := vec{x, y}
		good := true
		for _, o := range others {
			need := radius + o.r + spawnPad
			if o.side != side && gap > need {
				need = gap
			}
			if p.sub(o.p).len() < need {
				good = false
				break
			}
		}
		if good {
			return p, true
		}
	}
	return vec{}, false
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
	if m.phase == PhaseRunning || m.phase == PhasePaused {
		m.closeLogLocked("中止")
	}
	m.openingUntil = time.Time{}
	m.activeOpening = nil
	m.pendingOpening = nil
	m.logMute = true
	m.stopAllLocked()
	m.logMute = false
	m.phase = PhaseSelect
	m.winner = ""
	m.winnerID = 0
	m.time = 0
	m.lineup = [2][]string{}
	m.probeTape = nil
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
		m.pinHeldLocked()
		m.expireFSLocked()
		m.syncCruiseLocked()
		m.decelerateLocked(DT)
		m.drainCmdsLocked()
		m.applyImpulseLocked()
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
	m.flushPendingLocked()
	m.settleHitsLocked()
	m.reapAttachLocked()
	m.stickFollowersLocked()
	m.spinWallsLocked(DT)
	m.flushPendingLocked()
	m.pinHeldLocked()
	m.time += DT
	m.expireWallsLocked()
	m.releaseHoldLocked()
	m.expireStunLocked()
	m.decayMarksLocked()
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
	const rise = drop / 2
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || !u.solid || u.cruiseFS == nil {
			continue
		}
		if u.skipDecel() {
			u.decelT = 0
			continue
		}
		sp := u.v.len()
		if sp < 1e-6 {
			if u.stand || u.cruise < 1e-6 {
				u.decelT = 0
				continue
			}
			u.setVel(u.cruiseDir().mul(u.cruise))
			u.decelT = 0
			continue
		}
		if math.Abs(sp-u.cruise) <= 1e-6 {
			u.decelT = 0
			continue
		}
		u.decelT += dt
		for u.decelT >= step-1e-12 {
			u.decelT -= step
			sp = u.v.len()
			if sp < 1e-6 {
				break
			}
			if sp > u.cruise+1e-6 {
				ns := sp - drop
				if ns < u.cruise {
					ns = u.cruise
				}
				u.v = u.v.norm().mul(ns)
				continue
			}
			if sp < u.cruise-1e-6 {
				ns := sp + rise
				if ns > u.cruise {
					ns = u.cruise
				}
				u.v = u.v.norm().mul(ns)
				continue
			}
			break
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
	m.rootN = 0
	m.record = nil
	m.recordDone = false
	m.recordInitial = nil
	m.recordPresentations = nil
	m.activeOpening = nil
	m.openingUntil = time.Time{}
}

func (m *Match) installFieldLocked() {
	for _, h := range m.spec.hard {
		m.nextID++
		w := &barrier{
			id:     m.nextID,
			a:      vec{h.x1, h.y1},
			b:      vec{h.x2, h.y2},
			radius: h.halfW,
			until:  math.Inf(1),
			hard:   true,
			square: true,
			field:  true,
			hitAt:  map[uint64]float64{},
		}
		if h.period > 0 {
			dx, dy := h.x2-h.x1, h.y2-h.y1
			w.halfLen = math.Hypot(dx, dy) / 2
			w.pivot = vec{(h.x1 + h.x2) / 2, (h.y1 + h.y2) / 2}
			w.base = math.Atan2(dy, dx)
			w.spin = -2 * math.Pi / h.period
			w.riding = map[uint64]struct{}{}
			w.pose()
		}
		m.walls = append(m.walls, w)
	}
	for _, c := range m.spec.caps {
		until := math.Inf(1)
		if c.life > 0 {
			until = m.time + c.life
		}
		m.nextID++
		m.walls = append(m.walls, &barrier{
			id:     m.nextID,
			a:      vec{c.x1, c.y1},
			b:      vec{c.x2, c.y2},
			radius: c.radius,
			until:  until,
			amount: c.amount,
			field:  true,
			hitAt:  map[uint64]float64{},
		})
	}
}

func (m *Match) liveFieldLocked() unitpkg.Field {
	f := m.spec.toUnitField()
	hard := make([]*barrier, 0, 1)
	for _, w := range m.walls {
		if w.hard && w.field {
			hard = append(hard, w)
		}
	}
	hi := 0
	for i := range f.Walls {
		if !f.Walls[i].Kind.Hard() || hi >= len(hard) {
			continue
		}
		w := hard[hi]
		hi++
		f.Walls[i].X1, f.Walls[i].Y1 = w.a.X, w.a.Y
		f.Walls[i].X2, f.Walls[i].Y2 = w.b.X, w.b.Y
	}
	for _, w := range m.walls {
		if w.hard {
			continue
		}
		f.Walls = append(f.Walls, unitpkg.FieldWall{
			Kind: unitpkg.WallCapsule,
			X1:   w.a.X, Y1: w.a.Y, X2: w.b.X, Y2: w.b.Y,
			Radius: w.radius,
		})
	}
	return f
}

func (m *Match) stopAllLocked() {
	us := m.actors
	for _, u := range us {
		m.killLocked(u)
	}
	m.units = make(map[uint64]*unit)
	m.order = nil
	m.walls = nil
	m.actors = nil
	// 停场不等于收工：角色协程手上那一次 Handle 可能还没写完，而 ctx.Out 就是 m.cmds。
	// 这些指令要是留到下一场，就会打在重新从 1 开始的 ID 上——火山的减速（没寿命，谁也摘不掉）
	// 和熔岩 Spawn 都是这么漏进下一场的。所以这里等它们真退出，边等边把队列读空。
	// 名单要用 actors（含已离场的）：子弹、熔岩这些小东西整场都在 despawn，只等 m.units 会漏。
	// 角色只会在 ctx.Out 上阻塞（不会卡在别处），边读边等不会死锁。
	for _, u := range us {
		for done := false; !done; {
			select {
			case <-u.exited:
				done = true
			case <-m.cmds:
			}
		}
	}
	for {
		select {
		case <-m.cmds:
		default:
			// 上一场的特效和顿帧一起清掉：快照是每帧原样推 effects 的，留着就会在
			// 选人界面和下一场开头把上一场的喷发/熔岩再演一遍。（读空之后才清，免得又把残留指令里的 FX 写回来。）
			m.fx = m.fx[:0]
			m.hitStop = 0
			return
		}
	}
}

func (m *Match) killLocked(u *unit) {
	if u == nil || u.stopped {
		return
	}
	u.stopped = true
	close(u.stop)
}

func (m *Match) spawnFighterLocked(kind string, slot, seat int, p, v vec) {
	u := m.addUnitLocked(kind, p, v, 0, slot)
	if u != nil {
		u.seat = seat
		m.noteBirthLocked(u)
	}
}

func (m *Match) rootStream() uint64 {
	m.rootN++
	return unitpkg.DeriveStream(m.seed, m.rootN, unitpkg.StreamRoot)
}

func (m *Match) addUnitLocked(kind string, p, v vec, owner uint64, slot int) *unit {
	return m.addUnitStreamLocked(kind, p, v, owner, slot, m.rootStream())
}

func (m *Match) addUnitStreamLocked(kind string, p, v vec, owner uint64, slot int, stream uint64) *unit {
	spec, ok := unitpkg.Lookup(kind)
	if !ok {
		return nil
	}
	actor := unitpkg.NewActor(kind, unitpkg.SpawnInfo{
		OwnerID: owner, Slot: slot, Stream: stream,
		Field: m.liveFieldLocked(), Probe: m.probeTape,
		Silent: m.spawnSilent && kind == acausalKind(),
	})
	if actor == nil {
		return nil
	}
	m.nextID++
	id := m.nextID
	u := &unit{
		id:          id,
		kind:        kind,
		role:        spec.Role,
		slot:        slot,
		owner:       owner,
		p:           p,
		v:           v,
		radius:      spec.Radius,
		hp:          spec.MaxHP,
		maxHP:       spec.MaxHP,
		vision:      spec.Vision,
		cruise:      spec.Speed,
		velM:        1,
		actor:       actor,
		inbox:       make(chan unitpkg.Event, 64),
		stop:        make(chan struct{}),
		exited:      make(chan struct{}),
		solid:       spec.Role != unitpkg.RoleHelper && !spec.Nonsolid,
		semi:        spec.Semi,
		face:        vec{1, 0},
		passWalls:   spec.PassWalls,
		breakWalls:  spec.BreakWalls,
		mortal:      spec.Mortal,
		aimPriority: unitpkg.DefaultAimPriority(spec),
		shell:       spec.Shell,
		attach:      spec.Attach,
		arcSpan:     spec.ArcSpan,
		arcInner:    spec.ArcInner,
		visionBlock: spec.VisionBlock,
	}
	if spec.StartHP > 0 && spec.StartHP < spec.MaxHP {
		u.hp = spec.StartHP
	}
	if spec.Role == unitpkg.RoleFighter || spec.Cruise {
		u.cruiseFS = newCruiseFS(v, spec.Speed)
		u.syncCruise()
	}
	u.aimFace()
	m.units[id] = u
	m.order = append(m.order, id)
	m.actors = append(m.actors, u)
	if spec.Role != unitpkg.RoleFighter {
		m.noteBirthLocked(u)
	}
	u.stream = stream
	ctx := unitpkg.Context{ID: id, Kind: kind, Out: m.cmds, Stream: stream, Field: m.liveFieldLocked()}.WithBirths()
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
	defer close(u.exited)
	for {
		select {
		case <-u.stop:
			return
		case ev, ok := <-u.inbox:
			if !ok {
				return
			}
			if sense, ok := ev.(unitpkg.Sense); ok {
				ctx.Field = sense.Field
			}
			u.actor.Handle(ctx, ev)
		}
	}
}

func (m *Match) drainCmdsLocked() {
	defer m.flushLeapsLocked()
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
		mul := u.cruiseMul()
		u.setVel(vec{c.VX, c.VY}.mul(mul))
		u.velM = mul
		if u.held {
			u.holdVel = u.v
		}
	case unitpkg.SetCruise:
		m.applySetCruiseLocked(c)
	case unitpkg.AddFS:
		m.applyAddFSLocked(c)
	case unitpkg.RemoveFS:
		m.applyRemoveFSLocked(c)
	case unitpkg.AddFSComponent:
		m.applyAddFSComponentLocked(c)
	case unitpkg.RemoveFSComponent:
		m.applyRemoveFSComponentLocked(c)
	case unitpkg.SetFSDirection:
		m.applySetFSDirectionLocked(c)
	case unitpkg.SetVision:
		u := m.units[c.UnitID]
		if u == nil || u.stopped {
			return
		}
		next := c.Vision
		if next < 0 {
			next = 0
		}
		prev := u.vision
		u.vision = next
		m.noteSpanLocked(u, "视野", prev, next)
	case unitpkg.SetArcSpan:
		u := m.units[c.UnitID]
		if u == nil || u.stopped || !u.attach {
			return
		}
		span := c.Span
		if span < 0 {
			span = 0
		}
		prev := u.arcSpan
		u.arcSpan = span
		m.noteSpanLocked(u, "扇环", prev, span)
	case unitpkg.SetRadius:
		u := m.units[c.UnitID]
		if u == nil || u.stopped || c.Radius <= 0 {
			return
		}
		prev := u.radius
		u.radius = c.Radius
		m.noteSpanLocked(u, "半径", prev, u.radius)
	case unitpkg.SetHP:
		u := m.units[c.UnitID]
		if u == nil || u.stopped {
			return
		}
		prevHP, prevMax := u.hp, u.maxHP
		if c.MaxHP > 0 {
			u.maxHP = c.MaxHP
		}
		if c.HP >= 0 {
			u.hp = c.HP
			if u.hp > u.maxHP {
				u.hp = u.maxHP
			}
		}
		if !logSame(prevHP, u.hp) || !logSame(prevMax, u.maxHP) {
			m.noteSetHPLocked(u)
		}
	case unitpkg.SetAimPriority:
		u := m.units[c.UnitID]
		if u == nil || u.stopped {
			return
		}
		if u.role != unitpkg.RoleFighter && !u.mortal {
			return
		}
		if u.aimPriority == 0 && c.From != u.id {
			return
		}
		if u.aimPriority == c.Value {
			return
		}
		u.aimPriority = c.Value
		m.noteAimLocked(c.From, u, c.Value)
	case unitpkg.Damage:
		m.offerDamageLocked(c)
	case unitpkg.ConfirmDamage:
		m.confirmDamageLocked(c)
	case unitpkg.BlockDamage:
		off, ok := m.pendingDmg[c.Token]
		if !ok || c.UnitID != off.to {
			return
		}
		m.noteBlockLocked(off)
		delete(m.pendingDmg, c.Token)
		delete(m.wardAbsorb, off.to)
	case unitpkg.StackMark:
		m.stackMarkLocked(c)
	case unitpkg.ClearMarks:
		m.clearMarksLocked(c)
	case unitpkg.Heal:
		u := m.units[c.UnitID]
		if u == nil || u.stopped || !u.takesHit() || c.Amount <= 0 {
			return
		}
		before := u.hp
		u.hp += c.Amount
		if u.hp > u.maxHP {
			u.hp = u.maxHP
		}
		got := u.hp - before
		m.noteHealLocked(u, got)
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
		u := m.addUnitStreamLocked(c.Kind, vec{c.X, c.Y}, vec{c.VX, c.VY}, c.OwnerID, c.Slot, c.Stream)
		m.nailToWallLocked(u, c)
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
	case unitpkg.SetWallMotion:
		m.setWallMotionLocked(c)
	case unitpkg.HoldStill:
		m.holdStillLocked(c.UnitID, c.Until)
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
		if u == nil || u.stopped {
			return
		}
		if !u.solid && u.role != unitpkg.RoleHelper {
			return
		}
		from := u.p
		dest := vec{c.X, c.Y}
		u.p = dest
		m.constrainUnitLocked(u)
		m.recordLeapLocked(u, from, dest, u.p)
	case unitpkg.Pass:
		u := m.units[c.UnitID]
		if u == nil || u.stopped || u.pass == c.Hold {
			return
		}
		u.pass = c.Hold
		m.noteHoldLocked(u, "穿过", c.Hold)
	case unitpkg.Stand:
		u := m.units[c.UnitID]
		if u == nil || u.stopped || u.stand == c.Hold {
			return
		}
		u.stand = c.Hold
		m.noteHoldLocked(u, "站住", c.Hold)
	case unitpkg.NoFrameFreeze:
		u := m.units[c.UnitID]
		if u == nil || u.stopped || u.noFrameFreeze == c.Hold {
			return
		}
		u.noFrameFreeze = c.Hold
		m.noteHoldLocked(u, "不停帧", c.Hold)
	case unitpkg.NoHealthNumbers:
		u := m.units[c.UnitID]
		if u == nil || u.stopped {
			return
		}
		u.noHealthNumbers = c.Hold
	case unitpkg.Stun:
		u := m.units[c.UnitID]
		if u == nil || u.stopped {
			return
		}
		u.stun = c.Hold
		if !c.Hold {
			u.stunUntil = 0
			m.noteStunLocked(u, 0, true)
		} else {
			u.stunUntil = c.Until
			m.noteStunLocked(u, c.Until, false)
			m.send(u, unitpkg.Stunned{Time: m.time, Until: c.Until})
		}
	case unitpkg.VisionBlock:
		u := m.units[c.UnitID]
		if u == nil || u.stopped {
			return
		}
		if u.visionBlock == c.Hold {
			return
		}
		u.visionBlock = c.Hold
		m.noteHoldLocked(u, "挡视线", c.Hold)
	}
}

func (m *Match) settleHitsLocked() {
	runtime.Gosched()
	m.drainCmdsLocked()
}

func (u *unit) takesHit() bool {
	return u != nil && !u.stopped && (u.role == unitpkg.RoleFighter || u.mortal)
}

func (m *Match) offerDamageLocked(c unitpkg.Damage) {
	if c.Amount <= 0 {
		return
	}
	u := m.units[c.To]
	if !u.takesHit() {
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
		noFreeze:  c.NoFreeze,
		markKind:  c.MarkKind,
		markDelta: c.MarkDelta,
		markIcon:  c.MarkIcon,
		credit:    m.creditOfLocked(c.From),
	}
	m.noteOfferLocked(c)
	if from := m.units[c.From]; from != nil {
		off := m.pendingDmg[token]
		off.atkSpeed = from.v.len()
		off.atkImpulse = impulseSpeeds(from)
		m.pendingDmg[token] = off
	}
	{
		off := m.pendingDmg[token]
		off.defSpeed = u.v.len()
		off.defImpulse = impulseSpeeds(u)
		m.pendingDmg[token] = off
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
	if u == nil || u.stopped || !u.takesHit() {
		return
	}
	// 用发起那一刻记下的记功对象；找不到才退回原始的 from（比如记功对象也同拍没了）。
	from := m.units[off.credit]
	if from == nil {
		from = m.units[off.from]
	}
	if u.role == unitpkg.RoleFighter && (off.absorb || m.shellOfLocked(u.id) != nil || m.wardAbsorb[u.id]) {
		delete(m.wardAbsorb, u.id)
		m.noteAbsorbLocked(off)
		if sh := m.shellOfLocked(u.id); sh != nil {
			m.popShellLocked(sh, off.from)
			delete(m.wardAbsorb, u.id)
		}
		return
	}
	pre := off.amount
	if c.Amount > 0 && c.Amount < pre {
		pre = c.Amount
	}
	amt := m.scaleDamage(m.factionBearer(from), u, pre)
	if u.role == unitpkg.RoleFighter && off.markKind != "" && amt > 0 {
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
	remain := u.hp
	if amt > 0 {
		remain = u.hp - amt
		if remain < 0 {
			remain = 0
		}
	}
	m.noteConfirmLocked(off, amt, remain)
	if u.role != unitpkg.RoleFighter {
		m.applyHurtLocked(from, u, amt, false, false)
		return
	}
	m.applyHurtLocked(from, u, amt, !off.noFreeze, true)
}

func (m *Match) applyHurtLocked(from, u *unit, amt float64, freeze, swap bool) {
	if u == nil || amt <= 0 {
		return
	}
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
	if u.hp < 0 {
		u.hp = 0
	}
	m.noteHurtLocked(from, u, amt)
	if freeze {
		m.hitStopIfNeeded(from)
	}
	if u.hp <= 0 {
		m.noteDownLocked(u, true)
		m.creditKillLocked(from, u)
		m.removeLocked(u)
	} else if swap {
		m.swapOwnedLocked(u.id)
	}
}

// creditOfLocked 算「这笔伤害记谁的功」：一般是发起者本人；发起者是别人的弹或随从时
// 记到它的主人头上（owner 链只走一层，和 holdsNoFrameFreeze 同一套口径）。
func (m *Match) creditOfLocked(fromID uint64) uint64 {
	f := m.units[fromID]
	if f == nil {
		return fromID
	}
	if f.owner != 0 {
		if o := m.units[f.owner]; o != nil && !o.stopped {
			return o.id
		}
	}
	return f.id
}

// creditKillLocked 把「谁打死了谁」记到该记的人头上。
// from 传的已经是记功对象（见 dmgOffer.credit），所以这里一般直接用。
func (m *Match) creditKillLocked(from, victim *unit) {
	if from == nil || victim == nil {
		return
	}
	killer := from
	if victim.kind == "史莱姆" && victim.owner != 0 {
		if o := m.units[victim.owner]; o != nil && !o.stopped {
			killer = o
		}
	} else if from.owner != 0 {
		if o := m.units[from.owner]; o != nil && !o.stopped {
			killer = o
		}
	}
	if killer == nil || killer.stopped || killer.id == victim.id {
		return
	}
	m.noteCreditLocked(killer, victim)
	m.send(killer, unitpkg.Kill{
		VictimID:   victim.id,
		VictimKind: victim.kind,
		VictimRole: victim.role,
		Mortal:     victim.mortal,
	})
}

func (m *Match) hitStopIfNeeded(from *unit) {
	if m.holdsNoFrameFreeze(from) {
		return
	}
	m.hitStop = HitStopFrames
}

func (m *Match) holdsNoFrameFreeze(u *unit) bool {
	if u == nil {
		return false
	}
	if u.noFrameFreeze {
		return true
	}
	if u.owner == 0 {
		return false
	}
	o := m.units[u.owner]
	return o != nil && o.noFrameFreeze
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

func (m *Match) stickFollowersLocked() {
	var dead []*unit
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || (!u.shell && !u.attach) {
			continue
		}
		owner := m.units[u.owner]
		if owner == nil || owner.stopped {
			dead = append(dead, u)
			continue
		}
		u.p = owner.p
		u.v = owner.v
		if u.attach {
			if u.v.len2() > 1e-8 {
				u.face = u.v.norm()
			} else if owner.face.len2() > 1e-12 {
				u.face = owner.face
			} else if u.face.len2() < 1e-12 {
				u.face = vec{1, 0}
			}
		}
	}
	for _, u := range dead {
		m.removeLocked(u)
	}
}

func (m *Match) reapAttachLocked() {
	var dead []*unit
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.attach && !u.solid && !u.stopped {
			dead = append(dead, u)
		}
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
		m.send(owner, unitpkg.GuardBreak{Time: t, From: from})
	}
}

func (m *Match) placeWallLocked(c unitpkg.PlaceWall) {
	if c.Radius <= 0 {
		return
	}
	if !c.Hard && c.Life == 0 {
		return
	}
	a, b := vec{c.X1, c.Y1}, vec{c.X2, c.Y2}
	if a.sub(b).len2() < 1 {
		return
	}
	until := math.Inf(1)
	if c.Life > 0 {
		until = m.time + c.Life
	}
	m.nextID++
	m.fx = append(m.fx, unitpkg.FX{
		Name: "wall-spawn", Kind: c.Kind, Slot: c.Slot,
		X: a.X, Y: a.Y, VX: b.X, VY: b.Y,
	})
	w := &barrier{
		id:          m.nextID,
		owner:       c.OwnerID,
		slot:        c.Slot,
		kind:        c.Kind,
		a:           a,
		b:           b,
		radius:      c.Radius,
		until:       until,
		amount:      c.Amount,
		hitGap:      c.HitGap,
		withOwner:   c.WithOwner,
		hard:        c.Hard,
		square:      c.Square || c.Hard,
		visionBlock: c.VisionBlock,
		hitAt:       map[uint64]float64{},
	}
	w.bindPose()
	w.prevA, w.prevB = w.a, w.b
	m.walls = append(m.walls, w)
	m.noteWallLocked("砌墙", w)
}

func (m *Match) expireWallsLocked() {
	n := 0
	for _, w := range m.walls {
		if w.hard || m.time < w.until {
			m.walls[n] = w
			n++
			continue
		}
		m.noteWallLocked("拆墙", w)
		m.fx = append(m.fx, unitpkg.FX{
			Name: "wall-fade", Kind: w.kind, Slot: w.slot,
			X: w.a.X, Y: w.a.Y, VX: w.b.X, VY: w.b.Y,
		})
	}
	m.walls = m.walls[:n]
}

func (m *Match) breakWallLocked(id uint64, fade bool) {
	n := 0
	for _, w := range m.walls {
		if w.id == id {
			m.noteWallLocked("拆墙", w)
			if fade {
				m.fx = append(m.fx, unitpkg.FX{
					Name: "wall-fade", Kind: w.kind, Slot: w.slot,
					X: w.a.X, Y: w.a.Y, VX: w.b.X, VY: w.b.Y,
				})
			}
			continue
		}
		m.walls[n] = w
		n++
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
	keepVel := body.stun
	body.p, other.p = other.p, body.p
	if !keepVel {
		body.v, other.v = other.v, body.v
	}
	m.noteSwapLocked(body, other, keepVel)
}

func (m *Match) removeLocked(u *unit) {
	if u == nil || u.stopped {
		return
	}
	m.noteGoneLocked(u)
	id := u.id
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
	m.dropOwnerWallsLocked(id)
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

func (w *barrier) theta() float64 { return w.base + w.ang }

func (w *barrier) pose() {
	if w.halfLen < 1e-9 {
		return
	}
	th := w.theta()
	dir := vec{math.Cos(th), math.Sin(th)}
	w.a = w.pivot.sub(dir.mul(w.halfLen))
	w.b = w.pivot.add(dir.mul(w.halfLen))
}

func (w *barrier) worldToLocal(p vec) vec {
	d := p.sub(w.pivot)
	th := w.theta()
	c, s := math.Cos(th), math.Sin(th)
	return vec{c*d.X + s*d.Y, -s*d.X + c*d.Y}
}

func (w *barrier) localToWorld(l vec) vec {
	th := w.theta()
	c, s := math.Cos(th), math.Sin(th)
	return w.pivot.add(vec{c*l.X - s*l.Y, s*l.X + c*l.Y})
}

func (m *Match) nailToWallLocked(u *unit, c unitpkg.Spawn) {
	if u == nil || !c.HardNail {
		return
	}
	var wall *barrier
	for _, w := range m.walls {
		if w.spin != 0 {
			wall = w
			break
		}
	}
	if wall == nil {
		return
	}
	u.nailWall = wall.id
	u.nailLocal = wall.worldToLocal(u.p)
}

func (m *Match) spinWallsLocked(dt float64) {
	for _, w := range m.walls {
		w.prevA, w.prevB = w.a, w.b
		moved := false
		if w.spin != 0 {
			w.ang += w.spin * dt
			moved = true
		}
		if w.vx != 0 || w.vy != 0 {
			w.pivot = w.pivot.add(vec{w.vx, w.vy}.mul(dt))
			moved = true
		}
		if !moved {
			continue
		}
		w.pose()
		m.carryNailsLocked(w)
		m.shoveWallLocked(w)
		m.ramWallLocked(w)
	}
	m.slamWallsLocked()
}

func (m *Match) flushPendingLocked() {
	for _, d := range m.pending {
		m.applyCmdLocked(d)
	}
	m.pending = m.pending[:0]
}

func (m *Match) pinHeldLocked() {
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.stopped || !u.held {
			continue
		}
		u.setVel(vec{})
	}
}

func (m *Match) releaseHoldLocked() {
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.stopped || !u.held {
			continue
		}
		if m.time+1e-9 < u.holdUntil {
			continue
		}
		u.setVel(u.holdVel)
		u.held = false
		if u.stun && u.stunUntil > 0 && u.stunUntil <= u.holdUntil+1e-6 {
			u.stun = false
			u.stunUntil = 0
			m.noteStunLocked(u, 0, true)
		}
	}
}

func (m *Match) holdStillLocked(id uint64, until float64) {
	u := m.units[id]
	if u == nil || u.stopped {
		return
	}
	u.holdVel = u.v
	u.held = true
	u.holdUntil = until
	u.setVel(vec{})
	u.stun = true
	u.stunUntil = until
	m.noteStunLocked(u, until, false)
	m.send(u, unitpkg.Stunned{Time: m.time, Until: until})
}

func (m *Match) dropOwnerWallsLocked(id uint64) {
	n := 0
	for _, w := range m.walls {
		if w.withOwner && w.owner == id {
			m.noteWallLocked("拆墙", w)
			m.fx = append(m.fx, unitpkg.FX{
				Name: "wall-fade", Kind: w.kind, Slot: w.slot,
				X: w.a.X, Y: w.a.Y, VX: w.b.X, VY: w.b.Y,
			})
			continue
		}
		m.walls[n] = w
		n++
	}
	m.walls = m.walls[:n]
}

func (m *Match) setWallMotionLocked(c unitpkg.SetWallMotion) {
	w := m.wallByID(c.WallID)
	if w == nil {
		return
	}
	w.bindPose()
	if c.Pin {
		w.pivot = vec{c.X, c.Y}
		w.base = c.Ang
		w.ang = 0
		w.pose()
	}
	same := logSame(w.spin, c.Spin) && logSame(w.vx, c.VX) && logSame(w.vy, c.VY) &&
		logSame(w.ram, c.Ram) && logSame(w.stunR, c.StunRadius) && logSame(w.stunDur, c.StunDur) &&
		w.passOwner == c.PassOwner
	wasRam := w.ram > 0
	w.spin = c.Spin
	w.vx, w.vy = c.VX, c.VY
	w.ram = c.Ram
	w.stunR = c.StunRadius
	w.stunDur = c.StunDur
	w.passOwner = c.PassOwner
	if !same {
		m.noteWallMotionLocked(w)
	}
	if c.Ram > 0 && !wasRam {
		w.rammed = map[uint64]struct{}{}
	}
}

func (w *barrier) bindPose() {
	if w.halfLen >= 1e-6 {
		return
	}
	d := w.b.sub(w.a)
	w.halfLen = d.len() / 2
	if w.halfLen < 1e-6 {
		return
	}
	w.pivot = vec{(w.a.X + w.b.X) / 2, (w.a.Y + w.b.Y) / 2}
	w.base = math.Atan2(d.Y, d.X)
	w.ang = 0
}

func (m *Match) scrapeWallLocked(u *unit, w *barrier) {
	if w.amount <= 0 || !u.takesHit() || u.slot == w.slot || u.id == w.owner {
		return
	}
	gap := w.hitGap
	if gap <= 0 {
		gap = 0.1
	}
	if at, ok := w.hitAt[u.id]; ok && m.time < at {
		return
	}
	if w.hitAt == nil {
		w.hitAt = map[uint64]float64{}
	}
	w.hitAt[u.id] = m.time + gap
	m.pending = append(m.pending, unitpkg.Damage{From: w.owner, To: u.id, Amount: w.amount})
}

func (m *Match) ramWallLocked(w *barrier) {
	if w.ram <= 0 {
		return
	}
	if w.rammed == nil {
		w.rammed = map[uint64]struct{}{}
	}
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.stopped || u.id == w.owner || u.slot == w.slot || !u.takesHit() {
			continue
		}
		if _, ok := w.rammed[u.id]; ok {
			continue
		}
		cc, cr := colOf(u.p, u.face, u.radius, u.semi)
		if distPointOBB(cc, w.a, w.b, w.radius) >= cr {
			continue
		}
		w.rammed[u.id] = struct{}{}
		m.pending = append(m.pending, unitpkg.Damage{From: w.owner, To: u.id, Amount: w.ram})
	}
}

// slamWallsLocked 用扫描重叠，避免冲撞墙单帧跨过 2*radius 判定窗口。
func (m *Match) slamWallsLocked() {
	for i := 0; i < len(m.walls); i++ {
		a := m.walls[i]
		if a.stunR <= 0 || a.owner == 0 {
			continue
		}
		for j := i + 1; j < len(m.walls); j++ {
			b := m.walls[j]
			if b.owner != a.owner || b.stunR <= 0 {
				continue
			}
			if !capsulesSwept(a.prevA, a.prevB, a.a, a.b, a.radius, b.prevA, b.prevB, b.a, b.b, b.radius) {
				continue
			}
			m.resolveSlamLocked(a, b)
			return
		}
	}
}

func (m *Match) resolveSlamLocked(a, b *barrier) {
	c := a.pivot.add(b.pivot).mul(0.5)
	dur := a.stunDur
	if dur <= 0 {
		dur = b.stunDur
	}
	r := a.stunR
	until := m.time + dur
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.stopped || u.id == a.owner {
			continue
		}
		if u.p.sub(c).len() > u.radius+r {
			continue
		}
		if u.takesHit() && u.slot != a.slot {
			m.holdStillLocked(u.id, until)
		}
	}
	owner := m.units[a.owner]
	m.breakWallLocked(a.id, false)
	m.breakWallLocked(b.id, false)
	m.fx = append(m.fx, unitpkg.FX{
		Name: "slam", Kind: a.kind, Slot: a.slot,
		X: c.X, Y: c.Y, Amount: r,
	})
	if owner != nil && !owner.stopped {
		m.send(owner, unitpkg.WallSlam{
			Time: m.time, X: c.X, Y: c.Y,
		})
	}
}

func (m *Match) wallViewsLocked() []unitpkg.WallView {
	if len(m.walls) == 0 {
		return nil
	}
	out := make([]unitpkg.WallView, 0, len(m.walls))
	for _, w := range m.walls {
		out = append(out, unitpkg.WallView{
			ID: w.id, OwnerID: w.owner, Slot: w.slot,
			X1: w.a.X, Y1: w.a.Y, X2: w.b.X, Y2: w.b.Y,
			Radius: w.radius, VisionBlock: w.visionBlock,
		})
	}
	return out
}

func segDist(p1, q1, p2, q2 vec) float64 {
	u := q1.sub(p1)
	v := q2.sub(p2)
	w := p1.sub(p2)
	a := u.dot(u)
	b := u.dot(v)
	c := v.dot(v)
	d := u.dot(w)
	e := v.dot(w)
	D := a*c - b*b
	sN, sD := D, D
	tN, tD := D, D
	if D < 1e-12 {
		sN = 0
		sD = 1
		tN = e
		tD = c
	} else {
		sN = b*e - c*d
		tN = a*e - b*d
		if sN < 0 {
			sN = 0
			tN = e
			tD = c
		} else if sN > sD {
			sN = sD
			tN = e + b
			tD = c
		}
	}
	if tN < 0 {
		tN = 0
		if -d < 0 {
			sN = 0
		} else if -d > a {
			sN = sD
		} else {
			sN = -d
			sD = a
		}
	} else if tN > tD {
		tN = tD
		if -d+b < 0 {
			sN = 0
		} else if -d+b > a {
			sN = sD
		} else {
			sN = -d + b
			sD = a
		}
	}
	sc, tc := 0.0, 0.0
	if math.Abs(sN) > 1e-12 {
		sc = sN / sD
	}
	if math.Abs(tN) > 1e-12 {
		tc = tN / tD
	}
	return w.add(u.mul(sc)).sub(v.mul(tc)).len()
}

func (m *Match) carryNailsLocked(w *barrier) {
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.stopped || u.nailWall != w.id {
			continue
		}
		u.p = w.localToWorld(u.nailLocal)
	}
}

func barrierNormal(cc vec, w *barrier) vec {
	q := closestOnOBB(cc, w.a, w.b, w.radius)
	n := q.sub(cc)
	if n.len2() < 1e-12 {
		n = perp(w.b.sub(w.a))
	}
	if n.len2() < 1e-12 {
		return vec{0, 1}
	}
	return n.norm()
}

func (m *Match) shoveWallLocked(w *barrier) {
	const sep = 2.0
	if w.riding == nil {
		w.riding = map[uint64]struct{}{}
	}
	still := map[uint64]struct{}{}
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.stopped || !u.solid || u.passWalls || u.shell || u.attach || u.role == unitpkg.RoleHelper {
			continue
		}
		if u.breakWalls && w.hard {
			continue
		}
		if w.passOwner && u.id == w.owner {
			continue
		}
		cc, cr := colOf(u.p, u.face, u.radius, u.semi)
		d := distPointOBB(cc, w.a, w.b, w.radius)
		need := cr + skin
		if d >= need+sep {
			continue
		}
		_, was := w.riding[u.id]
		if d < need {
			m.pushOutOBB(u, w, cc, cr)
			if !was {
				cc, _ = colOf(u.p, u.face, u.radius, u.semi)
				m.bounceBarrierLocked(u, w, barrierNormal(cc, w))
			} else {
				m.scrapeWallLocked(u, w)
			}
		}
		if was || d < need {
			still[u.id] = struct{}{}
		}
	}
	w.riding = still
}

func (m *Match) bounceBarrierLocked(u *unit, w *barrier, n vec) {
	kind := unitpkg.WallCapsule
	if w.hard {
		kind = unitpkg.WallHard
	}
	if w.spin != 0 || w.vx != 0 || w.vy != 0 {
		if w.riding == nil {
			w.riding = map[uint64]struct{}{}
		}
		w.riding[u.id] = struct{}{}
	}
	m.send(u, unitpkg.WallHit{
		Time: m.time, NX: n.X, NY: n.Y, Kind: kind,
		X: u.p.X, Y: u.p.Y,
	})
	u.setVel(reflectVelocity(u.v, n))
	m.cycleFactionLocked(u)
	u.expireOnWallFS()
	m.scrapeWallLocked(u, w)
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
		if !u.passWalls && !u.shell && !u.attach {
			if h, ok := sweptShapeVsOutline(u.p, u.v, u.face, u.radius, dt, m.spec, u.semi); ok {
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
				if w.passOwner && u.id == w.owner {
					continue
				}
				if u.breakWalls && w.hard {
					continue
				}
				var t float64
				var nrm vec
				var ok bool
				if w.square {
					t, nrm, ok = sweptPointVsOBB(cc, u.v, dt, w.a, w.b, w.radius, cr+skin)
				} else {
					R := cr + w.radius + skin
					t, nrm, ok = sweptPointVsCapsule(cc, u.v, dt, w.a, w.b, R)
				}
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
			if a.attach || b.attach {
				if a.attach && b.attach {
					continue
				}
				if a.attach && !attachHits(a, b) {
					continue
				}
				if b.attach && !attachHits(b, a) {
					continue
				}
			}
			if a.role == unitpkg.RoleFighter && b.role == unitpkg.RoleFighter {
				if m.shellOfLocked(a.id) != nil || m.shellOfLocked(b.id) != nil {
					continue
				}
			}
			t, nrm, ok := sweptPairShapes(a.shape(), b.shape(), dt)
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
		m.constrainOutlineLocked(u, h.n)
		m.send(u, unitpkg.WallHit{
			Time: m.time, NX: h.n.X, NY: h.n.Y, Kind: unitpkg.WallEdge,
			X: u.p.X, Y: u.p.Y,
		})
		u.setVel(reflectVelocity(u.v, h.n))
		m.cycleFactionLocked(u)
		u.expireOnWallFS()
	case hitBarrier:
		u := m.units[h.a]
		w := m.wallByID(h.w)
		if u == nil || w == nil {
			return
		}
		if u.breakWalls && !w.hard {
			m.breakWallLocked(w.id, true)
			return
		}
		cc, cr := colOf(u.p, u.face, u.radius, u.semi)
		if w.square {
			m.pushOutOBB(u, w, cc, cr)
		} else {
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
		}
		m.bounceBarrierLocked(u, w, h.n)
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
		pass := a.pass || b.pass
		pierce := a.passWalls || b.passWalls || a.breakWalls || b.breakWalls || pass
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
		if pass {
			return
		}
		projA := a.role == unitpkg.RoleProjectile
		projB := b.role == unitpkg.RoleProjectile
		if projA || projB {
			if projA && !a.passWalls && !a.breakWalls {
				a.v = vec{0, 0}
				a.solid = false
			}
			if projB && !b.passWalls && !b.breakWalls {
				b.v = vec{0, 0}
				b.solid = false
			}
			if a.passWalls && !a.breakWalls && !projB {
				b.setVel(n.mul(-b.v.len()))
			} else if b.passWalls && !b.breakWalls && !projA {
				a.setVel(n.mul(a.v.len()))
			}
		} else {
			sa, sb := a.v.len(), b.v.len()
			a.setVel(n.mul(sa))
			b.setVel(n.mul(-sb))
		}
	}
}

func attachHits(atk, target *unit) bool {
	if atk == nil || target == nil {
		return false
	}
	if target.slot == atk.slot || target.id == atk.owner {
		return false
	}
	return target.role == unitpkg.RoleFighter || target.mortal
}

func (m *Match) constrainAllLocked() {
	for _, id := range m.order {
		m.constrainUnitLocked(m.units[id])
	}
}

func (m *Match) constrainUnitLocked(u *unit) {
	if u == nil || !u.solid || u.passWalls || u.shell || u.attach {
		return
	}
	m.constrainOutlineLocked(u, vec{})
	cc, cr := colOf(u.p, u.face, u.radius, u.semi)
	for _, w := range m.walls {
		if u.breakWalls && w.hard {
			continue
		}
		if w.passOwner && u.id == w.owner {
			continue
		}
		if w.square {
			m.pushOutOBB(u, w, cc, cr)
			cc, cr = colOf(u.p, u.face, u.radius, u.semi)
			continue
		}
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

func (m *Match) constrainOutlineLocked(u *unit, n vec) {
	ox, oy := m.spec.outline.Constrain(
		u.p.X, u.p.Y, u.radius, n.X, n.Y, u.face.X, u.face.Y, skin, u.semi,
	)
	u.p = vec{ox, oy}
}

func (m *Match) pushOutOBB(u *unit, w *barrier, cc vec, cr float64) {
	need := cr + skin
	d := distPointOBB(cc, w.a, w.b, w.radius)
	if d >= need {
		return
	}
	q := closestOnOBB(cc, w.a, w.b, w.radius)
	n := cc.sub(q)
	if n.len2() < 1e-12 {
		n = perp(w.b.sub(w.a))
	}
	if n.len2() < 1e-12 {
		return
	}
	u.p = u.p.add(n.norm().mul(need - d))
}

func (m *Match) emitLocked() {
	field := m.liveFieldLocked()
	snaps := make([]unitpkg.Snapshot, 0, len(m.order))
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && !u.stopped {
			snaps = append(snaps, u.snap())
		}
	}
	views := m.wallViewsLocked()
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.stopped || u.stun {
			continue
		}
		sense := unitpkg.Sense{Time: m.time, Self: u.snap(), Field: field, Walls: views, Knock: len(u.fsList) > 0}
		vr2 := u.vision * u.vision
		for _, o := range snaps {
			if o.ID == u.id {
				continue
			}
			dx := o.X - u.p.X
			dy := o.Y - u.p.Y
			if dx*dx+dy*dy <= vr2 || o.OwnerID == u.id {
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
	if u.tap != nil {
		u.tap(ev)
	}
	select {
	case u.inbox <- ev:
	case <-u.stop:
	default:
	}
}

func (m *Match) checkWinLocked() {
	alive := [2]int{}
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.stopped || u.role != unitpkg.RoleFighter {
			continue
		}
		if u.slot != 0 && u.slot != 1 {
			continue
		}
		alive[u.slot]++
	}
	if alive[0] > 0 && alive[1] > 0 {
		return
	}
	m.phase = PhaseEnded
	m.winnerID = 0
	switch {
	case alive[0] > 0:
		m.winner = strings.Join(m.lineup[0], "、")
		if u := m.livingFighterLocked(0); u != nil {
			m.winnerID = u.id
		}
		m.closeLogLocked("胜：" + m.rosterLine(0))
	case alive[1] > 0:
		m.winner = strings.Join(m.lineup[1], "、")
		if u := m.livingFighterLocked(1); u != nil {
			m.winnerID = u.id
		}
		m.closeLogLocked("胜：" + m.rosterLine(1))
	default:
		m.winner = "平局"
		m.closeLogLocked("平局")
	}
}

func (m *Match) livingFighterLocked(slot int) *unit {
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && !u.stopped && u.role == unitpkg.RoleFighter && u.slot == slot {
			return u
		}
	}
	return nil
}

func (m *Match) Loop(broadcast func([]byte), halt <-chan struct{}) {
	ticker := time.NewTicker(time.Second / TickHz)
	defer ticker.Stop()
	for {
		select {
		case <-halt:
			return
		case <-ticker.C:
			if m.holdingOpening() {
				broadcast(m.SnapshotJSON())
				continue
			}
			m.Tick()
			m.mu.Lock()
			snap := m.snapshotJSONLocked(true)
			recorded := snap
			if m.takeOpeningLocked() != nil {
				recorded = m.snapshotJSONLocked(false)
			}
			m.recordSnapshotLocked(recorded)
			m.mu.Unlock()
			broadcast(snap)
		}
	}
}

// recordSnapshot 把本拍观众快照留在本场里。只从 Loop 调用，保证一场只有一条时间线。
func (m *Match) recordSnapshot(snap []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recordSnapshotLocked(snap)
}

func (m *Match) recordSnapshotLocked(snap []byte) {
	if m.recordDone {
		return
	}
	switch m.phase {
	case PhaseRunning, PhaseEnded:
		m.record = append(m.record, append(json.RawMessage(nil), snap...))
		if m.phase == PhaseEnded {
			m.recordDone = true
		}
	}
}

// RecordFrames 返回本场已记下的观众快照。下一场 Start 时清掉。
func (m *Match) RecordFrames() []json.RawMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]json.RawMessage, len(m.record))
	copy(out, m.record)
	return out
}

// Recording 原子取得同一场的帧、初态和演出。新一场不会修改旧导出任务持有的数据。
func (m *Match) Recording() replay.Recording {
	m.mu.Lock()
	defer m.mu.Unlock()
	return replay.Recording{Frames: append([]json.RawMessage(nil), m.record...), Initial: m.recordInitial, Presentations: append([]replay.Presentation(nil), m.recordPresentations...)}
}
