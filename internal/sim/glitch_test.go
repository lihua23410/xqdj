package sim

import (
	"math"
	"testing"
	"time"
	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func waitTicks(m *Match, n int) {
	for i := 0; i < n; i++ {
		m.Tick()
		time.Sleep(2 * time.Millisecond)
	}
}

func parkGlitch(m *Match) (*unit, *unit) {
	g := fighterByKind(m, character.KindGlitch)
	o := fighterByKind(m, character.KindWaller)
	if g == nil || o == nil {
		return nil, nil
	}
	g.p = vec{-120, 0}
	o.p = vec{120, 0}
	g.setVel(vec{0, glitchCruise(g)})
	o.setVel(vec{0, -o.cruise})
	return g, o
}

func glitchCruise(u *unit) float64 {
	if u == nil || u.cruise == 0 {
		return 165
	}
	return u.cruise
}

func ownedKind(m *Match, owner uint64, kind string) *unit {
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.kind == kind && u.owner == owner {
			return u
		}
	}
	return nil
}

func TestStackMarkAccumulatesAndClears(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindGlitch)
	m.SetSlot(1, character.KindWaller)
	m.Start()
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	_, o := parkGlitch(m)
	if o == nil {
		t.Fatal("missing fighters")
	}
	for i := 0; i < 3; i++ {
		m.stackMarkLocked(unitpkg.StackMark{UnitID: o.id, Kind: "剑痕", Delta: 1, Icon: "/ball/地慧星/status/jianhen.png"})
	}
	got := o.markList()
	if len(got) != 1 || got[0].Stacks != 3 || got[0].Kind != "剑痕" {
		t.Fatalf("marks=%+v", got)
	}
	m.clearMarksLocked(unitpkg.ClearMarks{UnitID: o.id, Kind: "剑痕"})
	if o.markList() != nil {
		t.Fatalf("cleared marks=%+v", o.markList())
	}
}

func TestGlitchCollisionAppliesSwordMark(t *testing.T) {
	m := NewMatchSeeded(2)
	m.SetSlot(0, character.KindGlitch)
	m.SetSlot(1, character.KindWaller)
	m.Start()
	defer m.End()
	waitTicks(m, 4)
	m.mu.Lock()
	g, o := parkGlitch(m)
	if g == nil || o == nil {
		m.mu.Unlock()
		t.Fatal("missing fighters")
	}
	arc := ownedKind(m, g.id, character.KindGlitchArc)
	if arc == nil {
		m.mu.Unlock()
		t.Fatal("missing glitch arc")
	}
	m.send(arc, unitpkg.Collision{Time: m.time, Other: o.snap(), NX: 1, NY: 0})
	m.mu.Unlock()
	time.Sleep(4 * time.Millisecond)
	m.mu.Lock()
	m.drainCmdsLocked()
	m.settleHitsLocked()
	m.mu.Unlock()
	time.Sleep(4 * time.Millisecond)
	m.mu.Lock()
	m.drainCmdsLocked()
	m.settleHitsLocked()
	marks := o.markList()
	m.mu.Unlock()
	if len(marks) != 1 || marks[0].Kind != "剑痕" || marks[0].Stacks != 1 {
		t.Fatalf("marks=%+v", marks)
	}
}

func TestGlitchMarkSkippedWhenHitBlocked(t *testing.T) {
	m := NewMatchSeeded(21)
	m.SetSlot(0, character.KindGlitch)
	m.SetSlot(1, character.KindGlitch)
	m.Start()
	defer m.End()
	waitTicks(m, 4)
	m.mu.Lock()
	var atk, def *unit
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.role != unitpkg.RoleFighter {
			continue
		}
		if u.slot == 0 {
			atk = u
		} else {
			def = u
		}
	}
	if atk == nil || def == nil {
		m.mu.Unlock()
		t.Fatal("missing fighters")
	}
	atk.p = vec{-120, 0}
	def.p = vec{120, 0}
	arc := ownedKind(m, atk.id, character.KindGlitchArc)
	if arc == nil {
		m.mu.Unlock()
		t.Fatal("missing glitch arc")
	}
	m.send(arc, unitpkg.Collision{Time: m.time, Other: def.snap(), NX: 1, NY: 0})
	m.mu.Unlock()
	time.Sleep(4 * time.Millisecond)
	m.mu.Lock()
	m.drainCmdsLocked()
	m.settleHitsLocked()
	m.drainCmdsLocked()
	m.settleHitsLocked()
	marks := def.markList()
	m.mu.Unlock()
	if len(marks) != 0 {
		t.Fatalf("blocked hit still marked %+v", marks)
	}
}

func TestGlitchDodgeBlocksAndLeavesGhost(t *testing.T) {
	m := NewMatchSeeded(3)
	m.SetSlot(0, character.KindGlitch)
	m.SetSlot(1, character.KindWaller)
	m.Start()
	defer m.End()
	waitTicks(m, 4)
	m.mu.Lock()
	g, o := parkGlitch(m)
	if g == nil || o == nil {
		m.mu.Unlock()
		t.Fatal("missing fighters")
	}
	hp0 := g.hp
	x0, y0 := g.p.X, g.p.Y
	id := g.id
	m.applyCmdLocked(unitpkg.Damage{From: o.id, To: id, Amount: 14})
	m.settleHitsLocked()
	m.mu.Unlock()
	time.Sleep(4 * time.Millisecond)
	m.mu.Lock()
	m.drainCmdsLocked()
	m.settleHitsLocked()
	u := m.units[id]
	if u == nil {
		m.mu.Unlock()
		t.Fatal("glitch gone")
	}
	if u.hp != hp0 {
		m.mu.Unlock()
		t.Fatalf("dodged hit still dropped hp %.1f -> %.1f", hp0, u.hp)
	}
	moved := math.Hypot(u.p.X-x0, u.p.Y-y0) > 1
	ghosts := 0
	for _, oid := range m.order {
		h := m.units[oid]
		if h != nil && h.kind == character.KindGlitchGhost && h.owner == id {
			ghosts++
			if h.solid {
				m.mu.Unlock()
				t.Fatal("ghost should not be solid")
			}
		}
	}
	m.mu.Unlock()
	if !moved {
		t.Fatal("dodge did not teleport")
	}
	if ghosts != 1 {
		t.Fatalf("ghosts=%d want 1", ghosts)
	}
}

func TestGlitchWallSpeedUntilRealDamage(t *testing.T) {
	m := NewMatchSeeded(4)
	m.SetSlot(0, character.KindGlitch)
	m.SetSlot(1, character.KindWaller)
	m.Start()
	defer m.End()
	waitTicks(m, 4)
	m.mu.Lock()
	g, o := parkGlitch(m)
	if g == nil || o == nil {
		m.mu.Unlock()
		t.Fatal("missing fighters")
	}
	id := g.id
	m.send(g, unitpkg.WallHit{Time: m.time, NX: 1, NY: 0})
	m.send(g, unitpkg.WallHit{Time: m.time, NX: 1, NY: 0})
	m.mu.Unlock()
	time.Sleep(4 * time.Millisecond)
	m.mu.Lock()
	m.drainCmdsLocked()
	m.send(g, unitpkg.Sense{Time: m.time, Self: g.snap()})
	m.mu.Unlock()
	time.Sleep(4 * time.Millisecond)
	m.mu.Lock()
	m.drainCmdsLocked()
	u := m.units[id]
	if u.v.len() <= glitchCruise(u)+1e-3 {
		m.mu.Unlock()
		t.Fatalf("speed after walls %v want > cruise %v", u.v.len(), glitchCruise(u))
	}
	boosted := u.v.len()
	hp0 := u.hp
	m.applyCmdLocked(unitpkg.Damage{From: o.id, To: id, Amount: 10})
	m.settleHitsLocked()
	m.mu.Unlock()
	time.Sleep(4 * time.Millisecond)
	m.mu.Lock()
	m.drainCmdsLocked()
	m.settleHitsLocked()
	u = m.units[id]
	if u.hp != hp0 {
		m.mu.Unlock()
		t.Fatalf("first hit should dodge, hp %.1f -> %.1f", hp0, u.hp)
	}
	if math.Abs(u.v.len()-boosted) > 1e-3 {
		m.mu.Unlock()
		t.Fatalf("blocked hit reset speed %v was %v", u.v.len(), boosted)
	}
	m.applyCmdLocked(unitpkg.Damage{From: o.id, To: id, Amount: 10})
	m.settleHitsLocked()
	m.mu.Unlock()
	time.Sleep(4 * time.Millisecond)
	m.mu.Lock()
	m.drainCmdsLocked()
	m.settleHitsLocked()
	u = m.units[id]
	if u == nil || u.hp >= hp0 {
		m.mu.Unlock()
		t.Fatalf("second hit should land hp=%v", u)
	}
	if math.Abs(u.v.len()-glitchCruise(u)) > 1e-3 {
		m.mu.Unlock()
		t.Fatalf("real damage should drop speed to cruise, got %v", u.v.len())
	}
	m.mu.Unlock()
}

func TestGlitchWallBounceKeepsReflection(t *testing.T) {
	m := NewMatchSeeded(6)
	m.SetSlot(0, character.KindGlitch)
	m.SetSlot(1, character.KindWaller)
	m.Start()
	defer m.End()
	waitTicks(m, 2)
	m.mu.Lock()
	g, o := parkGlitch(m)
	if g == nil || o == nil {
		m.mu.Unlock()
		t.Fatal("missing fighters")
	}
	o.p = vec{-80, 160}
	o.setVel(vec{0, 0})
	g.p = vec{200, 0}
	g.setVel(vec{165, 0})
	id := g.id
	m.mu.Unlock()
	waitTicks(m, 40)
	m.mu.Lock()
	defer m.mu.Unlock()
	u := m.units[id]
	if u == nil {
		t.Fatal("glitch gone")
	}
	if u.v.X >= 0 {
		t.Fatalf("should bounce left, v=%+v p=%+v", u.v, u.p)
	}
}

func TestGlitchGhostDoesNotCollide(t *testing.T) {
	m := NewMatchSeeded(7)
	m.SetSlot(0, character.KindGlitch)
	m.SetSlot(1, character.KindWaller)
	m.Start()
	defer m.End()
	waitTicks(m, 2)
	m.mu.Lock()
	g, o := parkGlitch(m)
	if g == nil || o == nil {
		m.mu.Unlock()
		t.Fatal("missing fighters")
	}
	o.p = vec{0, 160}
	o.setVel(vec{0, 0})
	g.p = vec{0, 0}
	g.setVel(vec{120, 0})
	m.applyCmdLocked(unitpkg.Spawn{
		Kind: character.KindGlitchGhost, X: 20, Y: 0,
		OwnerID: g.id, Slot: 0,
	})
	id := g.id
	vx0 := g.v.X
	m.mu.Unlock()
	waitTicks(m, 8)
	m.mu.Lock()
	defer m.mu.Unlock()
	u := m.units[id]
	if u == nil {
		t.Fatal("glitch gone")
	}
	if u.v.X < vx0-1 {
		t.Fatalf("ghost bounced the fighter v.X=%v was %v", u.v.X, vx0)
	}
	ghosts := 0
	for _, oid := range m.order {
		h := m.units[oid]
		if h != nil && h.kind == character.KindGlitchGhost {
			ghosts++
			if h.solid {
				t.Fatal("ghost should not be solid")
			}
		}
	}
	if ghosts != 1 {
		t.Fatalf("ghosts=%d", ghosts)
	}
}

func TestGlitchSlashUsesMarksAndGhostsThenClears(t *testing.T) {
	m := NewMatchSeeded(5)
	m.SetSlot(0, character.KindGlitch)
	m.SetSlot(1, character.KindWaller)
	m.Start()
	defer m.End()
	waitTicks(m, 4)
	m.mu.Lock()
	g, o := parkGlitch(m)
	if g == nil || o == nil {
		m.mu.Unlock()
		t.Fatal("missing fighters")
	}
	m.despawnOwnedLocked(g.id, character.KindGlitchGhost)
	m.clearMarksLocked(unitpkg.ClearMarks{UnitID: o.id, Kind: "剑痕"})
	for i := 0; i < 3; i++ {
		m.stackMarkLocked(unitpkg.StackMark{UnitID: o.id, Kind: "剑痕", Delta: 1, Icon: "/ball/地慧星/status/jianhen.png"})
	}
	m.applyCmdLocked(unitpkg.Spawn{Kind: character.KindGlitchGhost, X: -40, Y: 10, OwnerID: g.id, Slot: 0})
	m.applyCmdLocked(unitpkg.Spawn{Kind: character.KindGlitchGhost, X: -30, Y: -8, OwnerID: g.id, Slot: 0})
	nearby := []unitpkg.Snapshot{o.snap()}
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.kind == character.KindGlitchGhost {
			nearby = append(nearby, u.snap())
		}
	}
	hp0 := o.hp
	oid := o.id
	m.send(g, unitpkg.Sense{Time: 20, Self: g.snap(), Nearby: nearby})
	m.mu.Unlock()
	time.Sleep(4 * time.Millisecond)
	m.mu.Lock()
	m.drainCmdsLocked()
	m.settleHitsLocked()
	found := false
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.kind == character.KindGlitchSlash {
			found = true
			break
		}
	}
	m.mu.Unlock()
	if !found {
		t.Fatal("missing slash helper")
	}
	time.Sleep(4 * time.Millisecond)
	m.mu.Lock()
	m.drainCmdsLocked()
	m.settleHitsLocked()
	defer m.mu.Unlock()
	dst := m.units[oid]
	if dst == nil {
		t.Fatal("enemy gone")
	}
	wantDrop := (6.0 + 3.0*2.0) * (1.0 + 2.0)
	if math.Abs((hp0-dst.hp)-wantDrop) > 1e-6 {
		t.Fatalf("hp drop %v want %v (hp %v -> %v)", hp0-dst.hp, wantDrop, hp0, dst.hp)
	}
	if dst.markList() != nil {
		t.Fatalf("marks leftover %+v", dst.markList())
	}
	ghosts := 0
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.kind == character.KindGlitchGhost {
			ghosts++
		}
	}
	if ghosts != 0 {
		t.Fatalf("ghosts leftover %d", ghosts)
	}
}
