package sim

import (
	"fmt"
	"math"
	"testing"
	"time"
	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestHPChangesAreMultiplesOfHitDamage(t *testing.T) {
	for seed := uint64(1); seed <= 16; seed++ {
		m := NewMatchSeeded(seed)
		m.SetSlot(0, character.KindMelee)
		m.SetSlot(1, character.KindRanged)
		m.Start()
		prev := map[uint64]float64{}
		var log []string
		maxDrop := 0.0
		ended := false
		for i := 0; i < 240; i++ {
			m.Tick()
			time.Sleep(time.Millisecond)
			m.mu.Lock()
			seen := map[uint64]bool{}
			for _, id := range m.order {
				u := m.units[id]
				if u == nil || u.role != unitpkg.RoleFighter {
					continue
				}
				seen[id] = true
				if p, ok := prev[id]; ok && u.hp < p {
					d := p - u.hp
					log = append(log, fmt.Sprintf("t=%.3f id=%d %s drop=%.1f hp=%.1f->%.1f", m.time, id, u.kind, d, p, u.hp))
					if d > maxDrop {
						maxDrop = d
					}
				}
				prev[id] = u.hp
			}
			for id, p := range prev {
				if !seen[id] && p > 0 {
					log = append(log, fmt.Sprintf("t=%.3f id=%d removed drop=%.1f", m.time, id, p))
					if p > maxDrop {
						maxDrop = p
					}
					prev[id] = 0
				}
			}
			ended = m.phase == PhaseEnded
			m.mu.Unlock()
			if ended {
				break
			}
		}
		m.End()
		if maxDrop > 15.01 {
			t.Fatalf("seed %d HP dropped %.1f in one tick", seed, maxDrop)
		}
		if len(log) > 0 {
			t.Logf("seed=%d drops=%d maxDrop=%.1f\n%s", seed, len(log), maxDrop, joinLines(log, 20))
			return
		}
	}
	t.Fatal("no HP changes across seeds")
}

func TestRandomSpawnInsideHex(t *testing.T) {
	for seed := uint64(1); seed <= 40; seed++ {
		m := NewMatchSeeded(seed)
		m.Start()
		m.mu.Lock()
		var fighters []*unit
		for _, id := range m.order {
			u := m.units[id]
			if u != nil && u.role == unitpkg.RoleFighter {
				fighters = append(fighters, u)
			}
		}
		if len(fighters) != 2 {
			m.mu.Unlock()
			m.End()
			t.Fatalf("seed %d fighters=%d", seed, len(fighters))
		}
		if fighters[0].p.sub(fighters[1].p).len() < 1 {
			m.mu.Unlock()
			m.End()
			t.Fatalf("seed %d same position", seed)
		}
		for _, u := range fighters {
			if !m.hex.containsCenter(u.p, u.radius) {
				m.mu.Unlock()
				m.End()
				t.Fatalf("seed %d escaped spawn %+v", seed, u.p)
			}
		}
		d := fighters[0].p.sub(fighters[1].p).len()
		if d < fighters[0].radius+fighters[1].radius {
			m.mu.Unlock()
			m.End()
			t.Fatalf("seed %d overlap dist=%v", seed, d)
		}
		m.mu.Unlock()
		m.End()
	}
}

func TestHitStopFreezesMotionForThreeFrames(t *testing.T) {
	type pos struct{ x, y float64 }
	snapshot := func(m *Match) map[uint64]pos {
		out := map[uint64]pos{}
		m.mu.Lock()
		defer m.mu.Unlock()
		for _, id := range m.order {
			u := m.units[id]
			if u != nil && u.solid {
				out[id] = pos{u.p.X, u.p.Y}
			}
		}
		return out
	}
	assertFrozen := func(m *Match, before map[uint64]pos, label string) {
		t.Helper()
		m.mu.Lock()
		defer m.mu.Unlock()
		for id, p0 := range before {
			u := m.units[id]
			if u == nil || !u.solid {
				continue
			}
			if u.p.X != p0.x || u.p.Y != p0.y {
				t.Fatalf("%s: unit %d moved", label, id)
			}
		}
	}
	for seed := uint64(1); seed <= 32; seed++ {
		m := NewMatchSeeded(seed)
		m.SetSlot(0, character.KindMelee)
		m.SetSlot(1, character.KindRanged)
		m.Start()
		prevHP := map[uint64]float64{}
		found := false
		for i := 0; i < 360; i++ {
			before := snapshot(m)
			m.Tick()
			time.Sleep(time.Millisecond)
			m.mu.Lock()
			dropped := false
			stopLeft := m.hitStop
			for _, id := range m.order {
				u := m.units[id]
				if u == nil || u.role != unitpkg.RoleFighter {
					continue
				}
				if p, ok := prevHP[id]; ok && u.hp < p-0.5 {
					dropped = true
				}
				prevHP[id] = u.hp
			}
			m.mu.Unlock()
			if !dropped {
				continue
			}
			found = true
			if stopLeft != HitStopFrames-1 {
				m.End()
				t.Fatalf("seed %d hitStop after damage = %d, want %d", seed, stopLeft, HitStopFrames-1)
			}
			assertFrozen(m, before, fmt.Sprintf("seed %d damage tick", seed))
			for f := 0; f < HitStopFrames-1; f++ {
				m.Tick()
				time.Sleep(time.Millisecond)
				assertFrozen(m, before, fmt.Sprintf("seed %d freeze %d", seed, f))
			}
			m.End()
			return
		}
		m.End()
		if found {
			return
		}
	}
	t.Fatal("no damage across seeds")
}

func TestDoppelgangerSpawnsThreeClones(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindDoppel)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	defer m.End()
	for i := 0; i < 8; i++ {
		m.Tick()
		time.Sleep(2 * time.Millisecond)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	fighters, clones := 0, 0
	var bodyID uint64
	for _, id := range m.order {
		u := m.units[id]
		if u == nil {
			continue
		}
		switch u.role {
		case unitpkg.RoleFighter:
			if u.kind == character.KindDoppel {
				fighters++
				bodyID = u.id
			}
		case unitpkg.RoleClone:
			clones++
			if u.owner != bodyID && bodyID != 0 {
				t.Fatalf("clone owner=%d want body=%d", u.owner, bodyID)
			}
		}
	}
	if fighters != 1 || clones != 3 {
		t.Fatalf("doppel fighters=%d clones=%d", fighters, clones)
	}
}

func TestTwinSplitsIntoTwoSharingFighter(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindTwin)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	defer m.End()
	for i := 0; i < 8; i++ {
		m.Tick()
		time.Sleep(2 * time.Millisecond)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	fighters, twins := 0, 0
	var bodyID uint64
	for _, id := range m.order {
		u := m.units[id]
		if u == nil {
			continue
		}
		switch u.role {
		case unitpkg.RoleFighter:
			if u.kind == character.KindTwin {
				fighters++
				bodyID = u.id
				if !u.semi {
					t.Fatal("body should be a semicircle")
				}
			}
		case unitpkg.RoleTwin:
			twins++
			if u.owner != bodyID && bodyID != 0 {
				t.Fatalf("twin owner=%d want body=%d", u.owner, bodyID)
			}
			if u.kind != "无下限" {
				t.Fatalf("twin kind=%s", u.kind)
			}
			if !u.semi {
				t.Fatal("half should be a semicircle")
			}
		}
	}
	if fighters != 1 || twins != 1 {
		t.Fatalf("twin fighters=%d halves=%d", fighters, twins)
	}
}

func TestForceAddsAcceleration(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindMelee)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	var u *unit
	for _, id := range m.order {
		cand := m.units[id]
		if cand != nil && cand.role == unitpkg.RoleFighter {
			u = cand
			break
		}
	}
	if u == nil {
		t.Fatal("no fighter")
	}
	before := u.v
	m.applyCmdLocked(unitpkg.Force{UnitID: u.id, AX: 60, AY: 0})
	got := u.v.sub(before)
	want := 60 * DT
	if math.Abs(got.X-want) > 1e-9 || math.Abs(got.Y) > 1e-9 {
		t.Fatalf("dv=%+v want x=%v", got, want)
	}
}

func TestKnightStartsAt75HP(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindKnight)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.kind != character.KindKnight {
			continue
		}
		if u.maxHP != 100 || math.Abs(u.hp-75) > 1e-6 {
			t.Fatalf("hp=%v/%v want 75/100", u.hp, u.maxHP)
		}
		return
	}
	t.Fatal("no knight")
}

func TestWallerPlacesWallAfterThreeSeconds(t *testing.T) {
	m := NewMatchSeeded(3)
	m.SetSlot(0, character.KindWaller)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	defer m.End()
	for i := 0; i < 200; i++ {
		m.Tick()
		time.Sleep(time.Millisecond)
		m.mu.Lock()
		n := len(m.walls)
		tm := m.time
		m.mu.Unlock()
		if n > 0 {
			if tm < 2.9 {
				t.Fatalf("wall at t=%.3f too early", tm)
			}
			return
		}
	}
	t.Fatal("no wall after 200 ticks")
}

func joinLines(lines []string, n int) string {
	if len(lines) > n {
		lines = lines[:n]
	}
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return out
}
