package sim

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func fighterSlot(m *Match, slot int) *unit {
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && !u.stopped && u.role == unitpkg.RoleFighter && u.slot == slot {
			return u
		}
	}
	return nil
}

func newDummyLog(t *testing.T) *Match {
	t.Helper()
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindDummy)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	return m
}

func confirmHit(m *Match, from, to *unit, amount float64) {
	m.offerDamageLocked(unitpkg.Damage{From: from.id, To: to.id, Amount: amount})
	var token uint64
	for tok, off := range m.pendingDmg {
		if off.to == to.id && off.from == from.id {
			token = tok
		}
	}
	m.confirmDamageLocked(unitpkg.ConfirmDamage{Token: token, UnitID: to.id, Amount: amount})
}

func TestRecapConfirmedHit(t *testing.T) {
	m := newDummyLog(t)
	defer m.End()
	m.mu.Lock()
	left, right := fighterSlot(m, 0), fighterSlot(m, 1)
	if left == nil || right == nil {
		m.mu.Unlock()
		t.Fatal("missing fighters")
	}
	confirmHit(m, left, right, 10)
	recap, trace := m.recap.String(), m.trace.String()
	ready := m.logReady
	m.mu.Unlock()
	if ready {
		t.Fatal("log ready while the fight is running")
	}
	want := "0.00  左1 靶子 → 右1 靶子  扣 10  剩余 99989"
	if !strings.Contains(recap, want) {
		t.Fatalf("recap missing hit:\n%s", recap)
	}
	if !strings.Contains(recap, "六边形 · 单挑\n左1 靶子\n右1 靶子\n") {
		t.Fatalf("header:\n%s", recap)
	}
	if !strings.Contains(trace, "seed 1\n") || !strings.Contains(trace, "提出") || !strings.Contains(trace, "确认") {
		t.Fatalf("trace:\n%s", trace)
	}
	if strings.Contains(recap, "提出") || strings.Contains(recap, "出生") {
		t.Fatalf("recap has trace detail:\n%s", recap)
	}
}

func TestRecapFieldSource(t *testing.T) {
	m := newDummyLog(t)
	defer m.End()
	m.mu.Lock()
	right := fighterSlot(m, 1)
	if right == nil {
		m.mu.Unlock()
		t.Fatal("missing fighter")
	}
	m.offerDamageLocked(unitpkg.Damage{From: 0, To: right.id, Amount: 1})
	var token uint64
	for tok, off := range m.pendingDmg {
		if off.to == right.id && off.from == 0 {
			token = tok
		}
	}
	m.confirmDamageLocked(unitpkg.ConfirmDamage{Token: token, UnitID: right.id, Amount: 1})
	recap := m.recap.String()
	trace := m.trace.String()
	m.mu.Unlock()
	if !strings.Contains(recap, "0.00  场地 → 右1 靶子  扣 1  剩余 99998") {
		t.Fatalf("recap:\n%s", recap)
	}
	if strings.Contains(trace, "记功") {
		t.Fatalf("field damage took credit:\n%s", trace)
	}
}

func TestBlockStaysOutOfRecap(t *testing.T) {
	m := newDummyLog(t)
	defer m.End()
	m.mu.Lock()
	left, right := fighterSlot(m, 0), fighterSlot(m, 1)
	if left == nil || right == nil {
		m.mu.Unlock()
		t.Fatal("missing fighters")
	}
	m.offerDamageLocked(unitpkg.Damage{From: left.id, To: right.id, Amount: 10})
	var token uint64
	for tok, off := range m.pendingDmg {
		if off.to == right.id {
			token = tok
		}
	}
	m.applyCmdLocked(unitpkg.BlockDamage{Token: token, UnitID: right.id})
	recap, trace := m.recap.String(), m.trace.String()
	m.mu.Unlock()
	if strings.Contains(recap, "扣 ") {
		t.Fatalf("blocked hit entered recap:\n%s", recap)
	}
	if !strings.Contains(trace, "挡住") || !strings.Contains(trace, "提出") {
		t.Fatalf("trace:\n%s", trace)
	}
}

func TestPauseKeepsLog(t *testing.T) {
	m := newDummyLog(t)
	defer m.End()
	m.mu.Lock()
	left, right := fighterSlot(m, 0), fighterSlot(m, 1)
	confirmHit(m, left, right, 10)
	m.mu.Unlock()
	before, beforeTrace := m.LogText()
	m.Pause()
	m.Start()
	after, afterTrace := m.LogText()
	if after != before || afterTrace != beforeTrace {
		t.Fatalf("pause resume rewrote the log\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if m.phase != PhaseRunning {
		t.Fatalf("phase %s", m.phase)
	}
}

func TestRestartClearsLog(t *testing.T) {
	m := newDummyLog(t)
	defer m.End()
	m.mu.Lock()
	left, right := fighterSlot(m, 0), fighterSlot(m, 1)
	confirmHit(m, left, right, 10)
	m.mu.Unlock()
	m.End()
	recap, _ := m.LogText()
	if !strings.Contains(recap, "扣 10") || !strings.Contains(recap, "中止") {
		t.Fatalf("aborted recap:\n%s", recap)
	}
	m.Start()
	recap, trace := m.LogText()
	if strings.Contains(recap, "扣 10") || strings.Contains(recap, "中止") {
		t.Fatalf("old log survived restart:\n%s", recap)
	}
	if !strings.Contains(recap, "六边形 · 单挑") || !strings.Contains(trace, "seed 1\n") {
		t.Fatalf("new header missing\n%s\n%s", recap, trace)
	}
	if m.logReady {
		t.Fatal("new fight is already ready to copy")
	}
}

func TestAbortAndWinClosingLines(t *testing.T) {
	m := newDummyLog(t)
	m.End()
	recap, _ := m.LogText()
	if !strings.Contains(recap, "左1 靶子  剩余 99999\n右1 靶子  剩余 99999\n中止\n") {
		t.Fatalf("abort:\n%s", recap)
	}
	var snap struct {
		LogReady bool   `json:"logReady"`
		Phase    string `json:"phase"`
	}
	if json.Unmarshal(m.SnapshotJSON(), &snap) != nil || !snap.LogReady || snap.Phase != "select" {
		t.Fatalf("snapshot %+v", snap)
	}

	m = newDummyLog(t)
	m.mu.Lock()
	left, right := fighterSlot(m, 0), fighterSlot(m, 1)
	right.hp = 10
	confirmHit(m, left, right, 10)
	m.checkWinLocked()
	m.mu.Unlock()
	before, _ := m.LogText()
	if !strings.Contains(before, "0.00  右1 靶子 倒下\n左1 靶子  剩余 99999\n右1 靶子  倒下\n胜：左1 靶子\n") {
		t.Fatalf("win recap:\n%s", before)
	}
	m.End()
	after, _ := m.LogText()
	if after != before {
		t.Fatalf("ending after a win changed the log\n%s", after)
	}
	if strings.Contains(after, "中止") {
		t.Fatalf("win became abort:\n%s", after)
	}
}

func TestTraceStateChanges(t *testing.T) {
	m := newDummyLog(t)
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	left, right := fighterSlot(m, 0), fighterSlot(m, 1)
	if left == nil || right == nil {
		t.Fatal("missing fighters")
	}
	_, trace := m.recap.String(), m.trace.String()
	if !strings.Contains(trace, "出生  id=") || !strings.Contains(trace, "(") {
		t.Fatalf("birth has no landing:\n%s", trace)
	}

	left.v = vec{170, 0}
	right.v = vec{0, 40}
	left.fsList = []impulseFS{{token: 1, dir: vec{1, 0}, baseSpeed: 400}}
	confirmHit(m, left, right, 10)
	_, trace = m.recap.String(), m.trace.String()
	if !strings.Contains(trace, "确认") || !strings.Contains(trace, "攻速 170  受速 40  攻瞬时 400") {
		t.Fatalf("confirm speeds:\n%s", trace)
	}
	recap, _ := m.recap.String(), m.trace.String()
	if strings.Contains(recap, "攻速") || strings.Contains(recap, "瞬时") {
		t.Fatalf("speeds leaked into recap:\n%s", recap)
	}

	left.p = vec{120, 40}
	clone := m.addUnitLocked(character.KindClone, vec{40, -20}, vec{}, left.id, left.slot)
	if clone == nil {
		t.Fatal("clone was not born")
	}
	m.swapOwnedLocked(left.id)
	_, trace = m.recap.String(), m.trace.String()
	wantSwap := fmt.Sprintf("换位  id=%d 左1 靶子  ↔  id=%d 左1 靶子的分身  (40,-20) ↔ (120,40)", left.id, clone.id)
	if !strings.Contains(trace, wantSwap) {
		t.Fatalf("swap:\n%s\nwant %s", trace, wantSwap)
	}
	left.stun = true
	m.swapOwnedLocked(left.id)
	_, trace = m.recap.String(), m.trace.String()
	if !strings.Contains(trace, "速度不换") {
		t.Fatalf("stunned swap kept velocity:\n%s", trace)
	}

	left.p = vec{0, 0}
	m.applyCmdLocked(unitpkg.Teleport{UnitID: left.id, X: 80, Y: 0})
	m.flushLeapsLocked()
	_, trace = m.recap.String(), m.trace.String()
	if !strings.Contains(trace, fmt.Sprintf("瞬移  id=%d 左1 靶子  (0,0) → (80,0)", left.id)) {
		t.Fatalf("blink:\n%s", trace)
	}
	jumps := strings.Count(trace, "瞬移")
	m.applyCmdLocked(unitpkg.Teleport{UnitID: left.id, X: 81, Y: 0})
	m.flushLeapsLocked()
	m.time += DT
	m.applyCmdLocked(unitpkg.Teleport{UnitID: left.id, X: 160, Y: 0})
	m.flushLeapsLocked()
	_, trace = m.recap.String(), m.trace.String()
	if strings.Count(trace, "瞬移") != jumps {
		t.Fatalf("walk or chained blink was logged:\n%s", trace)
	}
	right.p = vec{50, 0}
	left.p = vec{0, 0}
	m.time += 1
	m.applyCmdLocked(unitpkg.Teleport{UnitID: left.id, X: 50, Y: 0})
	m.applyCmdLocked(unitpkg.Teleport{UnitID: right.id, X: 0, Y: 0})
	m.flushLeapsLocked()
	_, trace = m.recap.String(), m.trace.String()
	if strings.Count(trace, "瞬移") != jumps {
		t.Fatalf("paired teleport was logged:\n%s", trace)
	}

	m.applyCmdLocked(unitpkg.Pass{UnitID: left.id, Hold: true})
	m.applyCmdLocked(unitpkg.Pass{UnitID: left.id, Hold: true})
	m.applyCmdLocked(unitpkg.Pass{UnitID: left.id, Hold: false})
	_, trace = m.recap.String(), m.trace.String()
	if strings.Count(trace, "穿过") != 2 || !strings.Contains(trace, "放下穿过") {
		t.Fatalf("pass:\n%s", trace)
	}

	m.applySetCruiseLocked(unitpkg.SetCruise{UnitID: left.id, Speed: 360})
	m.applySetCruiseLocked(unitpkg.SetCruise{UnitID: left.id, Speed: 360})
	_, trace = m.recap.String(), m.trace.String()
	if strings.Count(trace, "巡航") != 1 || !strings.Contains(trace, "165 → 360") {
		t.Fatalf("cruise:\n%s", trace)
	}

	m.stackMarkLocked(unitpkg.StackMark{UnitID: right.id, Kind: "醉意", Delta: 1})
	m.stackMarkLocked(unitpkg.StackMark{UnitID: right.id, Kind: "醉意", Delta: 1})
	m.clearMarksLocked(unitpkg.ClearMarks{UnitID: right.id, Kind: "醉意"})
	_, trace = m.recap.String(), m.trace.String()
	if !strings.Contains(trace, "醉意  1 → 2") || !strings.Contains(trace, "醉意  清掉") {
		t.Fatalf("marks:\n%s", trace)
	}

	m.nextID++
	w := &barrier{id: m.nextID, a: vec{0, 0}, b: vec{10, 0}, radius: 4}
	m.walls = append(m.walls, w)
	m.setWallMotionLocked(unitpkg.SetWallMotion{WallID: w.id, Spin: 1.2, VX: 40})
	m.setWallMotionLocked(unitpkg.SetWallMotion{WallID: w.id, Spin: 1.2, VX: 40})
	_, trace = m.recap.String(), m.trace.String()
	if strings.Count(trace, "墙动") != 1 || !strings.Contains(trace, "转 1.2  移 (40,0)") {
		t.Fatalf("wall:\n%s", trace)
	}
}
