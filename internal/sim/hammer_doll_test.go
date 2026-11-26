package sim

import (
	"math"
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestHammerDollsLoseHPAndDoNotRefill(t *testing.T) {
	m := NewHeadlessMatchSeeded(3)
	m.skipProbe = true
	m.SetSlot(0, character.KindHammer)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()

	m.mu.Lock()
	hammer := fighterByKind(m, character.KindHammer)
	dummy := fighterByKind(m, character.KindDummy)
	if hammer == nil || dummy == nil {
		m.mu.Unlock()
		t.Fatal("missing fighters")
	}
	hammer.p, hammer.v = vec{0, 0}, vec{}
	dummy.p, dummy.v = vec{80, 0}, vec{}
	m.time = 30 - DT
	m.mu.Unlock()

	dolls := waitHammerDolls(t, m, 40)
	if len(dolls) != 3 {
		t.Fatalf("dolls=%d", len(dolls))
	}

	m.mu.Lock()
	hammer.stun = true
	hammer.stunUntil = 1e9
	var extra []*unit
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && !u.stopped && u.role != unitpkg.RoleFighter && u.kind != character.KindDoll {
			extra = append(extra, u)
		}
	}
	for _, u := range extra {
		m.removeLocked(u)
	}
	m.mu.Unlock()

	const hit = 15.0
	before := map[uint64]float64{}
	m.mu.Lock()
	for _, d := range dolls {
		if math.Abs(d.maxHP-100) < 1e-6 || d.hp <= 100 {
			m.mu.Unlock()
			t.Fatalf("doll %d still on placeholder hp=%v max=%v", d.id, d.hp, d.maxHP)
		}
		before[d.id] = d.hp
		m.offerDamageLocked(unitpkg.Damage{From: dummy.id, To: d.id, Amount: hit})
	}
	m.mu.Unlock()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
		m.Tick()
		m.mu.Lock()
		settled := true
		for _, d := range dolls {
			if d.stopped || math.Abs(d.hp-(before[d.id]-hit)) > 1e-6 || pendingToward(m, d.id) != 0 {
				settled = false
			}
		}
		m.mu.Unlock()
		if settled {
			break
		}
	}
	m.mu.Lock()
	for _, d := range dolls {
		want := before[d.id] - hit
		if d.stopped || math.Abs(d.hp-want) > 1e-6 {
			m.mu.Unlock()
			t.Fatalf("doll %d hp=%v want %v stopped=%v", d.id, d.hp, want, d.stopped)
		}
		if n := pendingToward(m, d.id); n != 0 {
			m.mu.Unlock()
			t.Fatalf("doll %d left %d pending hits", d.id, n)
		}
	}
	hurt := map[uint64]float64{}
	for _, d := range dolls {
		hurt[d.id] = d.hp
	}
	hammer.stun = false
	hammer.stunUntil = 0
	hammer.p = vec{500, 400}
	m.mu.Unlock()

	for i := 0; i < 20; i++ {
		m.Tick()
	}
	m.mu.Lock()
	hammer.p = vec{140, 0}
	m.mu.Unlock()
	for i := 0; i < 20; i++ {
		m.Tick()
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	for _, d := range dolls {
		if d.stopped {
			continue
		}
		if d.hp > hurt[d.id]+1e-6 {
			t.Fatalf("doll %d refilled %v -> %v (max %v)", d.id, hurt[d.id], d.hp, d.maxHP)
		}
	}
}

func waitHammerDolls(t *testing.T, m *Match, frames int) []*unit {
	t.Helper()
	var dolls []*unit
	for i := 0; i < frames; i++ {
		m.Tick()
		m.mu.Lock()
		dolls = dolls[:0]
		ready := 0
		for _, id := range m.order {
			u := m.units[id]
			if u == nil || u.stopped || u.kind != character.KindDoll {
				continue
			}
			dolls = append(dolls, u)
			if math.Abs(u.maxHP-100) > 1e-6 && math.Abs(u.hp-u.maxHP) < 1e-6 {
				ready++
			}
		}
		m.mu.Unlock()
		if ready == 3 {
			return append([]*unit(nil), dolls...)
		}
	}
	t.Fatalf("dolls did not take half HP, saw %d", len(dolls))
	return nil
}

func pendingToward(m *Match, id uint64) int {
	n := 0
	for _, off := range m.pendingDmg {
		if off.to == id {
			n++
		}
	}
	return n
}
