package sim

import (
	"time"

	"xqdj/character/acausal"
	unitpkg "xqdj/internal/unit"
)

const probeMaxTicks = 180 * TickHz

// closeUnfinished 探针被截断、身上的血还没变成掉血记录时，补上剩下的血。
// 否则频谱是空的，真局每一发伤害都是 0。按两秒一口来记，避免被前面的空采样稀释掉。
func (st *probeState) closeUnfinished() {
	var lost float64
	for _, d := range st.drops {
		lost += d
	}
	if lost > 0.5 {
		return
	}
	hp := st.hp
	if hp < 1 {
		hp = 100
	}
	drops := make([]float64, 120)
	drops[len(drops)-1] = hp
	st.drops = drops
}

type introBody struct {
	K string  `json:"k"`
	S int     `json:"s"`
	X float64 `json:"x"`
	Y float64 `json:"y"`
	R float64 `json:"r"`
}

type introFrame struct {
	T float64     `json:"t"`
	B []introBody `json:"b"`
}

type introFilm struct {
	Death  float64      `json:"death"`
	Frames []introFrame `json:"frames"`
}

func acausalKind() string { return 非因果.KindAcausal }

func (m *Match) runAcausalProbeLocked() {
	n := 0
	slot := -1
	for side := 0; side < 2; side++ {
		for _, k := range m.slots[side] {
			if k == 非因果.KindAcausal {
				n++
				slot = side
			}
		}
	}
	if n >= 2 {
		panic("两只非因果")
	}
	if n == 0 {
		非因果.Install(nil)
		m.intro = nil
		m.introSent = false
		m.introUntil = time.Time{}
		return
	}
	probe := newMatch(m.seed, true)
	probe.skipProbe = true
	probe.spawnSilent = true
	probe.team = m.team
	probe.spec = m.spec
	probe.slots[0] = append([]string{}, m.slots[0]...)
	probe.slots[1] = append([]string{}, m.slots[1]...)
	probe.Start()
	tape, film := captureProbe(probe, slot)
	probe.End()
	非因果.Install(tape)
	m.intro = film
	m.introSent = false
	m.introUntil = time.Time{}
	if film != nil && film.Death >= 0 {
		dur := 1 + film.Death/10
		m.introUntil = time.Now().Add(time.Duration((dur + 2) * float64(time.Second)))
	}
}

func (m *Match) flipAcausalLocked() {
	tape := 非因果.Current()
	if tape == nil {
		return
	}
	for _, u := range m.units {
		if u == nil || u.stopped || u.kind != 非因果.KindAcausal {
			continue
		}
		u.setVel(vec{-tape.SpawnVX, -tape.SpawnVY})
		if u.cruiseFS != nil && u.v.len2() > 1e-12 {
			u.cruiseFS.dir = u.v.norm()
		}
	}
}

func (m *Match) takeIntroLocked() *introFilm {
	if m.intro == nil || m.introUntil.IsZero() {
		return nil
	}
	return m.intro
}

func (m *Match) holdingIntro() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return !m.introUntil.IsZero() && time.Now().Before(m.introUntil)
}

// FinishIntro 让开场停住的真局开始走。
func (m *Match) FinishIntro() {
	m.mu.Lock()
	m.introUntil = time.Time{}
	m.mu.Unlock()
}

type minionBuild struct {
	track 非因果.MinionTrack
}

type probeKey struct {
	slot, seat int
	kind       string
}

type probeState struct {
	slot       int
	hp         float64
	have       bool
	death      float64
	drops      []float64
	spawnVX    float64
	spawnVY    float64
	self       []非因果.Sample
	selfHP     []float64
	fighters   map[uint64]*非因果.FighterTrack
	fightOrder []uint64
	minions    map[uint64]*minionBuild
	minOrder   []uint64
	nextMinion map[probeKey]int
	frames     []introFrame
	ticks      int
	justDied   bool
}

func captureProbe(p *Match, acausalSlot int) (*非因果.Tape, *introFilm) {
	st := &probeState{
		slot:       acausalSlot,
		death:      -1,
		fighters:   map[uint64]*非因果.FighterTrack{},
		minions:    map[uint64]*minionBuild{},
		nextMinion: map[probeKey]int{},
	}
	st.sample(p, true)
	finished := false
	for i := 0; i < probeMaxTicks; i++ {
		if p.phaseNow() != PhaseRunning {
			finished = true
			break
		}
		p.Tick()
		st.sample(p, false)
	}
	if !finished && p.phaseNow() == PhaseRunning {
		st.closeUnfinished()
	}
	tape := st.finish()
	var film *introFilm
	if st.death >= 0 {
		film = &introFilm{Death: st.death, Frames: st.frames}
	}
	return tape, film
}

func (m *Match) phaseNow() Phase {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.phase
}

func (st *probeState) sample(p *Match, initial bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	st.justDied = false
	var me *unit
	for _, id := range p.order {
		u := p.units[id]
		if u != nil && !u.stopped && u.kind == 非因果.KindAcausal {
			me = u
		}
	}
	if initial {
		if me != nil {
			st.hp = me.hp
			st.have = true
			st.spawnVX = me.v.X
			st.spawnVY = me.v.Y
		}
	} else if st.have {
		if me == nil {
			if st.hp > 0 && st.death < 0 {
				st.drops = append(st.drops, st.hp)
				st.death = p.time
				st.justDied = true
				if n := len(st.self); n > 0 {
					last := st.self[n-1]
					st.self = append(st.self, last)
					st.selfHP = append(st.selfHP, 0)
				}
			}
			st.have = false
		} else {
			d := st.hp - me.hp
			if d < 0 {
				d = 0
			}
			st.drops = append(st.drops, d)
			st.hp = me.hp
		}
	}
	for _, id := range p.order {
		u := p.units[id]
		if u == nil || u.stopped {
			continue
		}
		if u.kind == 非因果.KindAcausal {
			st.self = append(st.self, 非因果.Sample{X: u.p.X, Y: u.p.Y})
			st.selfHP = append(st.selfHP, u.hp)
		}
		if u.slot == st.slot {
			continue
		}
		if u.role == unitpkg.RoleFighter && u.slot != st.slot {
			ft := st.fighters[u.id]
			if ft == nil {
				cp := 非因果.FighterTrack{
					Slot: u.slot, Seat: u.seat, Kind: u.kind,
					MaxHP: u.maxHP, Radius: u.radius,
				}
				st.fighters[u.id] = &cp
				st.fightOrder = append(st.fightOrder, u.id)
				ft = &cp
			}
			ft.Samples = append(ft.Samples, 非因果.Sample{X: u.p.X, Y: u.p.Y})
		}
		if u.role == unitpkg.RoleMinion && u.mortal && u.slot != st.slot {
			mb := st.minions[u.id]
			if mb == nil {
				oslot, oseat := u.slot, 0
				if o := p.units[u.owner]; o != nil {
					oslot, oseat = o.slot, o.seat
				}
				key := probeKey{oslot, oseat, u.kind}
				idx := st.nextMinion[key]
				st.nextMinion[key] = idx + 1
				mb = &minionBuild{track: 非因果.MinionTrack{
					OwnerSlot: oslot, OwnerSeat: oseat, Kind: u.kind,
					Index: idx, MaxHP: u.maxHP, Radius: u.radius,
				}}
				st.minions[u.id] = mb
				st.minOrder = append(st.minOrder, u.id)
			}
			mb.track.Samples = append(mb.track.Samples, 非因果.Sample{X: u.p.X, Y: u.p.Y})
		}
	}
	if st.death < 0 || st.justDied {
		if initial || st.ticks%6 == 0 || st.justDied {
			var bodies []introBody
			for _, id := range p.order {
				u := p.units[id]
				if u == nil || u.stopped {
					continue
				}
				if u.role != unitpkg.RoleFighter && !u.mortal {
					continue
				}
				bodies = append(bodies, introBody{K: u.kind, S: u.slot, X: u.p.X, Y: u.p.Y, R: u.radius})
			}
			st.frames = append(st.frames, introFrame{T: p.time, B: bodies})
		}
	}
	st.ticks++
}

func (st *probeState) finish() *非因果.Tape {
	tape := &非因果.Tape{
		DeathTime: st.death,
		SpawnVX:   st.spawnVX,
		SpawnVY:   st.spawnVY,
		Drops:     st.drops,
		Self:      st.self,
		SelfHP:    st.selfHP,
	}
	for _, id := range st.fightOrder {
		tape.Fighters = append(tape.Fighters, *st.fighters[id])
	}
	for _, id := range st.minOrder {
		tape.Minions = append(tape.Minions, st.minions[id].track)
	}
	return tape
}
