package sim

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	unitpkg "xqdj/internal/unit"
)

// LogText 返回这场的战报和排查。真正重开之后是新的一场。
func (m *Match) LogText() (recap, trace string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.recap.String(), m.trace.String()
}

func (m *Match) writeLogHeaderLocked() {
	m.recap.Reset()
	m.trace.Reset()
	m.logReady = false
	m.logNames = map[uint64]string{}
	m.logLeapAt = map[uint64]float64{}
	m.logLeaps = nil
	mode := "单挑"
	if m.team {
		mode = "团队赛"
	}
	var roster strings.Builder
	for side := 0; side < 2; side++ {
		edge := "左"
		if side == 1 {
			edge = "右"
		}
		for seat, kind := range m.lineup[side] {
			fmt.Fprintf(&roster, "%s%d %s\n", edge, seat+1, kind)
		}
	}
	head := m.spec.name + " · " + mode + "\n" + roster.String() + "\n"
	m.recap.WriteString(head)
	fmt.Fprintf(&m.trace, "seed %d\n", m.seed)
	m.trace.WriteString(head)
}

func (m *Match) closeLogLocked(line string) {
	if m.logReady || m.logMute {
		return
	}
	m.noteRosterStatusLocked()
	m.noteRecap(line)
	m.noteTrace(line)
	m.logReady = true
}

// noteRosterStatusLocked 收尾前把两边每一只都写上。只写胜的一边时，另一边会从战报里消失。
func (m *Match) noteRosterStatusLocked() {
	for side := 0; side < 2; side++ {
		edge := "左"
		if side == 1 {
			edge = "右"
		}
		for seat, kind := range m.lineup[side] {
			label := fmt.Sprintf("%s%d %s", edge, seat+1, kind)
			var hp float64
			alive := false
			for _, id := range m.order {
				u := m.units[id]
				if u == nil || u.stopped || u.role != unitpkg.RoleFighter || u.slot != side || u.seat != seat {
					continue
				}
				alive = true
				hp = u.hp
				break
			}
			line := label + "  倒下"
			if alive {
				line = fmt.Sprintf("%s  剩余 %s", label, logNum(hp))
			}
			m.noteRecap(line)
			m.noteTrace(line)
		}
	}
}

func (m *Match) rosterLine(side int) string {
	edge := "左"
	if side == 1 {
		edge = "右"
	}
	parts := make([]string, 0, len(m.lineup[side]))
	for seat, kind := range m.lineup[side] {
		parts = append(parts, fmt.Sprintf("%s%d %s", edge, seat+1, kind))
	}
	return strings.Join(parts, "、")
}

func (m *Match) noteRecap(line string) {
	if m.logMute {
		return
	}
	m.recap.WriteString(line)
	m.recap.WriteByte('\n')
}

func (m *Match) noteTrace(line string) {
	if m.logMute {
		return
	}
	m.trace.WriteString(line)
	m.trace.WriteByte('\n')
}

func logTime(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

func logNum(v float64) string {
	r := math.Round(v)
	if math.Abs(v-r) < 1e-6 {
		return strconv.FormatInt(int64(r), 10)
	}
	s := strconv.FormatFloat(v, 'f', 2, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

func (m *Match) rememberLabel(u *unit) string {
	if u == nil {
		return "场地"
	}
	if m.logNames == nil {
		m.logNames = map[uint64]string{}
	}
	if s, ok := m.logNames[u.id]; ok {
		return s
	}
	var s string
	switch {
	case u.role == unitpkg.RoleFighter:
		edge := "左"
		if u.slot == 1 {
			edge = "右"
		}
		s = fmt.Sprintf("%s%d %s", edge, u.seat+1, u.kind)
	case u.owner != 0:
		owner := m.labelByID(u.owner)
		if owner != "" && !strings.HasPrefix(owner, "#") {
			s = owner + "的" + u.kind
		} else {
			s = u.kind
		}
	default:
		s = u.kind
	}
	m.logNames[u.id] = s
	return s
}

func (m *Match) labelByID(id uint64) string {
	if id == 0 {
		return "场地"
	}
	if m.logNames != nil {
		if s, ok := m.logNames[id]; ok {
			return s
		}
	}
	if u := m.units[id]; u != nil {
		return m.rememberLabel(u)
	}
	return "#" + strconv.FormatUint(id, 10)
}

func (m *Match) idLabel(id uint64) string {
	if id == 0 {
		return "场地"
	}
	return fmt.Sprintf("id=%d %s", id, m.labelByID(id))
}

func (m *Match) sourceLabel(from *unit) string {
	if from == nil {
		return "场地"
	}
	return m.rememberLabel(from)
}

func (m *Match) noteBirthLocked(u *unit) {
	if u == nil {
		return
	}
	m.rememberLabel(u)
	m.noteTrace(fmt.Sprintf("%s  出生  %s  %s", logTime(m.time), m.idLabel(u.id), logPoint(u.p)))
}

func (m *Match) noteWallsBuiltLocked() {
	for _, w := range m.walls {
		m.noteWallLocked("砌墙", w)
	}
}

func (m *Match) noteWallLocked(verb string, w *barrier) {
	if w == nil {
		return
	}
	if verb == "砌墙" {
		owner := "场地"
		if w.owner != 0 {
			owner = m.idLabel(w.owner)
		}
		m.noteTrace(fmt.Sprintf("%s  砌墙  id=%d  主人 %s", logTime(m.time), w.id, owner))
		return
	}
	m.noteTrace(fmt.Sprintf("%s  拆墙  id=%d", logTime(m.time), w.id))
}

func (m *Match) noteOfferLocked(c unitpkg.Damage) {
	m.noteTrace(fmt.Sprintf("%s  提出  %s → %s  %s",
		logTime(m.time), m.idLabel(c.From), m.idLabel(c.To), logNum(c.Amount)))
}

func (m *Match) noteBlockLocked(off dmgOffer) {
	m.noteTrace(fmt.Sprintf("%s  挡住  %s  提出 %s",
		logTime(m.time), m.idLabel(off.to), logNum(off.amount)))
}

func (m *Match) noteAbsorbLocked(off dmgOffer) {
	m.noteTrace(fmt.Sprintf("%s  吸收  %s  提出 %s",
		logTime(m.time), m.idLabel(off.to), logNum(off.amount)))
}

func (m *Match) noteConfirmLocked(off dmgOffer, applied, hp float64) {
	line := fmt.Sprintf("%s  确认  %s → %s  提出 %s  扣 %s  剩余 %s  攻速 %s  受速 %s",
		logTime(m.time), m.idLabel(off.from), m.idLabel(off.to), logNum(off.amount), logNum(applied), logNum(hp),
		logNum(off.atkSpeed), logNum(off.defSpeed))
	for _, s := range off.atkImpulse {
		line += "  攻瞬时 " + logNum(s)
	}
	for _, s := range off.defImpulse {
		line += "  受瞬时 " + logNum(s)
	}
	m.noteTrace(line)
}

func (m *Match) noteHurtLocked(from, u *unit, amt float64) {
	if u == nil {
		return
	}
	m.noteRecap(fmt.Sprintf("%s  %s → %s  扣 %s  剩余 %s",
		logTime(m.time), m.sourceLabel(from), m.rememberLabel(u), logNum(amt), logNum(u.hp)))
}

func (m *Match) noteDownLocked(u *unit, recap bool) {
	if u == nil {
		return
	}
	label := m.rememberLabel(u)
	if recap && u.role == unitpkg.RoleFighter {
		m.noteRecap(fmt.Sprintf("%s  %s 倒下", logTime(m.time), label))
	}
	m.noteTrace(fmt.Sprintf("%s  倒下  %s", logTime(m.time), m.idLabel(u.id)))
}

func (m *Match) noteGoneLocked(u *unit) {
	if u == nil || u.hp <= 0 {
		return
	}
	m.noteTrace(fmt.Sprintf("%s  消失  %s", logTime(m.time), m.idLabel(u.id)))
}

func (m *Match) noteHealLocked(u *unit, got float64) {
	if u == nil || got <= 0 {
		return
	}
	label := m.rememberLabel(u)
	m.noteRecap(fmt.Sprintf("%s  %s 回复 %s  剩余 %s",
		logTime(m.time), label, logNum(got), logNum(u.hp)))
	m.noteTrace(fmt.Sprintf("%s  回复  %s  %s  剩余 %s",
		logTime(m.time), m.idLabel(u.id), logNum(got), logNum(u.hp)))
}

func (m *Match) noteSetHPLocked(u *unit) {
	if u == nil {
		return
	}
	m.noteTrace(fmt.Sprintf("%s  改血  %s  %s/%s",
		logTime(m.time), m.idLabel(u.id), logNum(u.hp), logNum(u.maxHP)))
}

func (m *Match) noteStunLocked(u *unit, until float64, release bool) {
	if u == nil {
		return
	}
	who := m.idLabel(u.id)
	if release {
		m.noteTrace(fmt.Sprintf("%s  解除  %s", logTime(m.time), who))
		return
	}
	if until > 0 {
		m.noteTrace(fmt.Sprintf("%s  眩晕  %s  至 %s", logTime(m.time), who, logTime(until)))
		return
	}
	m.noteTrace(fmt.Sprintf("%s  眩晕  %s", logTime(m.time), who))
}

func (m *Match) noteCreditLocked(killer, victim *unit) {
	if killer == nil || victim == nil {
		return
	}
	role := "战斗机"
	if victim.mortal || victim.role == unitpkg.RoleMinion {
		role = "活随从"
	} else if victim.role != unitpkg.RoleFighter {
		role = victim.role
	}
	m.noteTrace(fmt.Sprintf("%s  记功  %s  打死  %s  %s",
		logTime(m.time), m.idLabel(killer.id), m.idLabel(victim.id), role))
}

func logPoint(p vec) string {
	return "(" + logNum(p.X) + "," + logNum(p.Y) + ")"
}

func logSame(a, b float64) bool {
	return math.Abs(a-b) < 1e-6
}

func (m *Match) noteHoldLocked(u *unit, verb string, hold bool) {
	if u == nil {
		return
	}
	if !hold {
		verb = "放下" + verb
	}
	m.noteTrace(fmt.Sprintf("%s  %s  %s", logTime(m.time), verb, m.idLabel(u.id)))
}

func (m *Match) noteSpanLocked(u *unit, verb string, prev, next float64) {
	if u == nil || logSame(prev, next) {
		return
	}
	m.noteTrace(fmt.Sprintf("%s  %s  %s  %s → %s",
		logTime(m.time), verb, m.idLabel(u.id), logNum(prev), logNum(next)))
}

func (m *Match) noteAimLocked(from uint64, u *unit, next uint8) {
	if u == nil {
		return
	}
	m.noteTrace(fmt.Sprintf("%s  瞄准  %s  把  %s  改成 %d",
		logTime(m.time), m.idLabel(from), m.idLabel(u.id), next))
}

func (m *Match) noteMarkLocked(u *unit, kind string, prev, next int) {
	if u == nil || kind == "" || prev == next {
		return
	}
	if next <= 0 {
		m.noteTrace(fmt.Sprintf("%s  标记  %s  %s  清掉", logTime(m.time), m.idLabel(u.id), kind))
		return
	}
	m.noteTrace(fmt.Sprintf("%s  标记  %s  %s  %d → %d",
		logTime(m.time), m.idLabel(u.id), kind, prev, next))
}

func (m *Match) noteFactionNameLocked(u *unit, name string) {
	if u == nil || name == "" {
		return
	}
	m.noteTrace(fmt.Sprintf("%s  派系  %s  %s", logTime(m.time), m.idLabel(u.id), name))
}

func (m *Match) notePushLocked(u *unit, kind string, speed float64, remove bool) {
	if u == nil {
		return
	}
	verb := "推动"
	if remove {
		verb = "撤下推动"
	}
	m.noteTrace(fmt.Sprintf("%s  %s  %s  %s %s",
		logTime(m.time), verb, m.idLabel(u.id), kind, logNum(speed)))
}

func (m *Match) noteWallMotionLocked(w *barrier) {
	if w == nil {
		return
	}
	line := fmt.Sprintf("%s  墙动  id=%d  转 %s  移 (%s,%s)",
		logTime(m.time), w.id, logNum(w.spin), logNum(w.vx), logNum(w.vy))
	if !logSame(w.ram, 0) || !logSame(w.stunR, 0) || !logSame(w.stunDur, 0) {
		line += fmt.Sprintf("  冲 %s  晕 %s %s", logNum(w.ram), logNum(w.stunR), logNum(w.stunDur))
	}
	m.noteTrace(line)
}

func (m *Match) noteSwapLocked(body, other *unit, keepVel bool) {
	if body == nil || other == nil {
		return
	}
	line := fmt.Sprintf("%s  换位  %s  ↔  %s  %s ↔ %s",
		logTime(m.time), m.idLabel(body.id), m.idLabel(other.id), logPoint(body.p), logPoint(other.p))
	if keepVel {
		line += "  速度不换"
	}
	m.noteTrace(line)
}

type logLeap struct {
	id         uint64
	from, dest vec
	land       vec
	drop       bool
}

func nearPt(a, b vec) bool {
	return a.sub(b).len2() < 1e-4
}

// recordLeapLocked 先攒着。同一拍里对位互换的两下，以及连续小步，都不进排查。
func (m *Match) recordLeapLocked(u *unit, from, dest, land vec) {
	if u == nil {
		return
	}
	if m.logLeapAt == nil {
		m.logLeapAt = map[uint64]float64{}
	}
	prev, seen := m.logLeapAt[u.id]
	m.logLeapAt[u.id] = m.time
	step := u.cruise * DT
	dist := land.sub(from).len()
	chained := seen && prev < m.time-1e-12 && m.time-prev <= DT+1e-6
	if chained || dist <= step+1e-4 {
		return
	}
	leap := logLeap{id: u.id, from: from, dest: dest, land: land}
	for i := range m.logLeaps {
		o := &m.logLeaps[i]
		if o.drop || o.id == u.id {
			continue
		}
		if nearPt(leap.dest, o.from) && nearPt(o.dest, leap.from) {
			o.drop = true
			leap.drop = true
			break
		}
	}
	m.logLeaps = append(m.logLeaps, leap)
}

func (m *Match) flushLeapsLocked() {
	for _, lp := range m.logLeaps {
		if lp.drop {
			continue
		}
		m.noteTrace(fmt.Sprintf("%s  瞬移  %s  %s → %s",
			logTime(m.time), m.idLabel(lp.id), logPoint(lp.from), logPoint(lp.land)))
	}
	m.logLeaps = m.logLeaps[:0]
}

func impulseSpeeds(u *unit) []float64 {
	if u == nil || len(u.fsList) == 0 {
		return nil
	}
	out := make([]float64, len(u.fsList))
	for i, fs := range u.fsList {
		out[i] = fs.baseSpeed
	}
	return out
}
