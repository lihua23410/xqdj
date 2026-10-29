package sim

import (
	"testing"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func markStacksOf(u *unit, kind string) int {
	if u == nil || u.marks[kind] == nil {
		return 0
	}
	return u.marks[kind].stacks
}

func markLifetimeOf(u *unit, kind string) float64 {
	if u == nil || u.marks[kind] == nil {
		return 0
	}
	return u.marks[kind].lifetime
}

func markNextOf(u *unit, kind string) float64 {
	if u == nil || u.marks[kind] == nil || len(u.marks[kind].hits) == 0 {
		return 0
	}
	return u.marks[kind].hits[0] + u.marks[kind].lifetime
}

func newMarkMatch(t *testing.T, seed uint64) (*Match, *unit) {
	t.Helper()
	m := NewMatchSeeded(seed)
	m.SetSlot(0, character.KindDummy)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	u := fighterByKind(m, character.KindDummy)
	if u == nil {
		m.End()
		t.Fatal("missing dummy")
	}
	return m, u
}

func TestStackLifetimeDropsOldest(t *testing.T) {
	m, u := newMarkMatch(t, 3)
	defer m.End()

	m.mu.Lock()
	defer m.mu.Unlock()
	m.time = 0
	m.stackMarkLocked(unitpkg.StackMark{UnitID: u.id, Kind: "扫描", Delta: 1, Lifetime: 4})
	m.time = 1
	m.stackMarkLocked(unitpkg.StackMark{UnitID: u.id, Kind: "扫描", Delta: 1, Lifetime: 4})
	m.time = 2
	m.stackMarkLocked(unitpkg.StackMark{UnitID: u.id, Kind: "扫描", Delta: 1, Lifetime: 4})
	if got := markStacksOf(u, "扫描"); got != 3 {
		t.Fatalf("stacks=%d", got)
	}
	if got := markLifetimeOf(u, "扫描"); got != 4 {
		t.Fatalf("lifetime=%v", got)
	}
	if got := markNextOf(u, "扫描"); got != 4 {
		t.Fatalf("最老一层的过期时间=%v", got)
	}

	m.time = 3.99
	m.decayMarksLocked()
	if got := markStacksOf(u, "扫描"); got != 3 {
		t.Fatalf("过期前就掉了：%d", got)
	}

	m.time = 4
	m.decayMarksLocked()
	if got := markStacksOf(u, "扫描"); got != 2 {
		t.Fatalf("t=0 那层过期后 stacks=%d", got)
	}
	if got := markNextOf(u, "扫描"); got != 5 {
		t.Fatalf("下一层过期时间=%v", got)
	}

	m.time = 5
	m.decayMarksLocked()
	if got := markStacksOf(u, "扫描"); got != 1 {
		t.Fatalf("t=1 那层过期后 stacks=%d", got)
	}

	m.time = 6
	m.decayMarksLocked()
	if got := markStacksOf(u, "扫描"); got != 0 {
		t.Fatalf("最后一层过期后还有 %d 层", got)
	}
}

func TestLifetimeNotExtendedByNewStack(t *testing.T) {
	m, u := newMarkMatch(t, 4)
	defer m.End()

	m.mu.Lock()
	defer m.mu.Unlock()
	m.time = 0
	m.stackMarkLocked(unitpkg.StackMark{UnitID: u.id, Kind: "扫描", Delta: 1, Lifetime: 4})
	m.time = 1
	m.stackMarkLocked(unitpkg.StackMark{UnitID: u.id, Kind: "扫描", Delta: 1, Lifetime: 4})
	if got := markNextOf(u, "扫描"); got != 4 {
		t.Fatalf("第二层把第一层的过期时间推后了：%v", got)
	}

	m.time = 4
	m.decayMarksLocked()
	if got := markStacksOf(u, "扫描"); got != 1 {
		t.Fatalf("t=0 那层过期后 stacks=%d", got)
	}
	if got := markNextOf(u, "扫描"); got != 5 {
		t.Fatalf("下一层过期时间=%v", got)
	}

	m.time = 5
	m.decayMarksLocked()
	if got := markStacksOf(u, "扫描"); got != 0 {
		t.Fatalf("t=1 那层过期后还有 %d 层", got)
	}
}

func TestNonLifetimeMarkStays(t *testing.T) {
	m, u := newMarkMatch(t, 5)
	defer m.End()

	m.mu.Lock()
	defer m.mu.Unlock()
	m.stackMarkLocked(unitpkg.StackMark{UnitID: u.id, Kind: "永久", Delta: 1})
	m.time = 999
	m.decayMarksLocked()
	if got := markStacksOf(u, "永久"); got != 1 {
		t.Fatalf("无寿命标记被清了：%d", got)
	}
}

func TestMarkSnapshotCarriesLifetime(t *testing.T) {
	m, u := newMarkMatch(t, 6)
	defer m.End()

	m.mu.Lock()
	defer m.mu.Unlock()
	m.time = 0
	m.stackMarkLocked(unitpkg.StackMark{UnitID: u.id, Kind: "扫描", Delta: 2, Lifetime: 4})
	got := u.markList()
	if len(got) != 1 || got[0].Stacks != 2 || got[0].Lifetime != 4 || got[0].NextDecayAt != 4 {
		t.Fatalf("marks=%+v", got)
	}
}

func TestHiddenMarkStaysInSnapshot(t *testing.T) {
	m, u := newMarkMatch(t, 7)
	defer m.End()

	m.mu.Lock()
	defer m.mu.Unlock()
	m.time = 0
	m.stackMarkLocked(unitpkg.StackMark{UnitID: u.id, Kind: "扫描", Delta: 2, Lifetime: 4, Hidden: true})
	got := u.markList()
	if len(got) != 1 || got[0].Stacks != 2 || !got[0].Hidden {
		t.Fatalf("marks=%+v", got)
	}
}
