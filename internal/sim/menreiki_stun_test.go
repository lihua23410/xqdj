package sim

import (
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

// 面灵气被眩晕时面具不能脱手。眩晕只拦 Sense，不拦物理：本体照旧靠惯性滑走，
// 而面具是非实心 helper，位置全靠主人每帧 Teleport —— 主人一断 Sense，
// 面具就定在世界里被落在后面。这里照弓箭/钉与锤那种 Stun 命令打一下，
// 量两秒里面具离主人的最远距离。
func TestStunnedMenreikiKeepsMasks(t *testing.T) {
	m := NewMatchSeeded(7)
	m.SetSlot(0, character.KindMenreiki)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()
	time.Sleep(30 * time.Millisecond)

	step := func() {
		m.Tick()
		time.Sleep(200 * time.Microsecond)
	}
	for i := 0; i < 60; i++ {
		step()
	}

	m.mu.Lock()
	me := fighterByKind(m, character.KindMenreiki)
	if me == nil {
		m.mu.Unlock()
		t.Fatal("missing menreiki")
	}
	id, orbit := me.id, me.radius+15+6
	if n := countOwnedMasksLocked(m, id); n != 3 {
		m.mu.Unlock()
		t.Fatalf("面具只有 %d 只", n)
	}
	m.applyCmdLocked(unitpkg.Stun{UnitID: id, Hold: true, Until: m.time + 2})
	m.mu.Unlock()

	worst := 0.0
	for i := 0; i < 120; i++ {
		step()
		m.mu.Lock()
		if owner := m.units[id]; owner != nil {
			for _, oid := range m.order {
				h := m.units[oid]
				if h == nil || h.owner != id || h.role != unitpkg.RoleHelper {
					continue
				}
				if d := h.p.sub(owner.p).len(); d > worst {
					worst = d
				}
			}
		}
		m.mu.Unlock()
	}
	if worst > orbit+8 {
		t.Fatalf("眩晕期间面具掉到了 %v（轨道半径 %v）", worst, orbit)
	}
}

func countOwnedMasksLocked(m *Match, owner uint64) int {
	n := 0
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.owner != owner || u.role != unitpkg.RoleHelper {
			continue
		}
		n++
	}
	return n
}
