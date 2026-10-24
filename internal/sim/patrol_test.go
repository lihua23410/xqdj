package sim

import (
	"math"
	"testing"

	"xqdj/character"
)

func patrolMarkerOf(m *Match) *unit {
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.kind == character.KindPatrolMarker {
			return u
		}
	}
	return nil
}

func patrolUnits(m *Match) (*unit, *unit) {
	return fighterByKind(m, character.KindPatrol), fighterByKind(m, character.KindDummy)
}

// TestPatrolMarkerCycle 走完一整圈：丢出后站定 → 标识物加速 → 撞中出伤 →
// 飞回主人消失 → 恢复移动 → 5 秒后再次丢出。随机路径不可控的环节用轮询等相位，
// 命中用把标识物挪到靶子身上强制触发（锁内改状态有 glitch 测试先例）。
func TestPatrolMarkerCycle(t *testing.T) {
	m := NewMatchSeeded(7)
	m.SetSlot(0, character.KindPatrol)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()

	// 相位一：等到标识物出手，本体应当站定。
	deadline := 5.0
	for {
		m.mu.Lock()
		patrol, _ := patrolUnits(m)
		mk := patrolMarkerOf(m)
		if patrol != nil && mk != nil {
			if got := math.Hypot(mk.v.X, mk.v.Y); got < 1 {
				m.mu.Unlock()
				t.Fatalf("marker thrown with zero speed")
			}
			if math.Hypot(patrol.v.X, patrol.v.Y) > 1e-6 {
				m.mu.Unlock()
				t.Fatalf("patrol still moving after throw: v=%v", patrol.v)
			}
			m.mu.Unlock()
			break
		}
		tm := m.time
		m.mu.Unlock()
		if tm > deadline {
			t.Fatalf("marker never thrown by t=%v", tm)
		}
		waitTicks(m, 3)
	}

	// 相位二：去程持续加速。撞墙只改方向不改速率，两拍速率必须升高。
	var sp1, sp2 float64
	for i := 0; i < 30; i++ {
		m.mu.Lock()
		mk := patrolMarkerOf(m)
		if mk != nil && i == 0 {
			sp1 = math.Hypot(mk.v.X, mk.v.Y)
		}
		m.mu.Unlock()
		waitTicks(m, 1)
	}
	m.mu.Lock()
	if mk := patrolMarkerOf(m); mk != nil {
		sp2 = math.Hypot(mk.v.X, mk.v.Y)
	}
	m.mu.Unlock()
	if sp1 <= 0 || sp2 <= sp1 {
		t.Fatalf("marker not accelerating: sp1=%v sp2=%v", sp1, sp2)
	}

	// 相位三：强制命中。靶子可能已经被自然命中打掉血，那也照样算。
	m.mu.Lock()
	_, dummy := patrolUnits(m)
	hp0 := dummy.hp
	if mk := patrolMarkerOf(m); mk != nil && dummy.hp == hp0 {
		mk.p = dummy.p
	}
	m.mu.Unlock()

	hit := false
	for i := 0; i < 12*60 && !hit; i++ {
		m.mu.Lock()
		_, dummy := patrolUnits(m)
		h := dummy.hp
		m.mu.Unlock()
		if h < hp0 {
			hit = true
		}
		waitTicks(m, 1)
	}
	if !hit {
		t.Fatal("marker never damaged the dummy")
	}

	// 相位四：标识物回家消失，本体恢复移动。记下收回时刻。
	catchT := -1.0
	for i := 0; i < 6*60; i++ {
		m.mu.Lock()
		patrol, _ := patrolUnits(m)
		mk := patrolMarkerOf(m)
		moving := math.Hypot(patrol.v.X, patrol.v.Y) > 1
		tm := m.time
		m.mu.Unlock()
		if mk == nil && moving {
			catchT = tm
			break
		}
		waitTicks(m, 1)
	}
	if catchT < 0 {
		t.Fatal("marker never came home / patrol never resumed")
	}

	// 相位五：收回后至少 5 秒才会再次丢出；再丢时又要站定。
	rethrowT := -1.0
	for i := 0; i < 8*60; i++ {
		m.mu.Lock()
		patrol, _ := patrolUnits(m)
		mk := patrolMarkerOf(m)
		tm := m.time
		m.mu.Unlock()
		if mk != nil {
			if math.Hypot(patrol.v.X, patrol.v.Y) > 1e-6 {
				t.Fatalf("patrol moving at rethrow: v=%v", patrol.v)
			}
			rethrowT = tm
			break
		}
		waitTicks(m, 1)
	}
	if rethrowT < 0 {
		t.Fatal("patrol never rethrew")
	}
	if gap := rethrowT - catchT; gap < 4.9 {
		t.Fatalf("rethrow too soon after catch: gap=%v want >= 5", gap)
	}
}
