package sim

import (
	"math"
	"testing"
	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestPassKeepsBothVelocitiesOnPairHit(t *testing.T) {
	a, b, m := parkedRangedPair(t)
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()

	a.p, b.p = vec{-20, 0}, vec{20, 0}
	a.setVel(vec{240, 0})
	b.setVel(vec{-80, 0})
	va, vb := a.v, b.v
	m.applyCmdLocked(unitpkg.Pass{UnitID: a.id, Hold: true})
	m.physicsLocked(DT)

	if math.Abs(a.v.X-va.X) > 1e-9 || math.Abs(a.v.Y-va.Y) > 1e-9 {
		t.Fatalf("holder vel %+v -> %+v", va, a.v)
	}
	if math.Abs(b.v.X-vb.X) > 1e-9 || math.Abs(b.v.Y-vb.Y) > 1e-9 {
		t.Fatalf("other vel %+v -> %+v", vb, b.v)
	}
}

func TestPassDroppedRestoresBounce(t *testing.T) {
	a, b, m := parkedRangedPair(t)
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()

	a.p, b.p = vec{-20, 0}, vec{20, 0}
	a.setVel(vec{240, 0})
	b.setVel(vec{-80, 0})
	m.applyCmdLocked(unitpkg.Pass{UnitID: a.id, Hold: true})
	m.applyCmdLocked(unitpkg.Pass{UnitID: a.id, Hold: false})
	m.physicsLocked(DT)

	if a.v.X > 0 {
		t.Fatalf("holder should bounce, vx=%v", a.v.X)
	}
	if b.v.X < 0 {
		t.Fatalf("other should bounce, vx=%v", b.v.X)
	}
}

func TestPassStillBouncesOnHexWall(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindRanged)
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
	u.p = vec{220, 0}
	u.setVel(vec{400, 0})
	m.applyCmdLocked(unitpkg.Pass{UnitID: u.id, Hold: true})
	hit := false
	for i := 0; i < 40; i++ {
		m.physicsLocked(DT)
		if u.v.X < 0 {
			hit = true
			break
		}
	}
	if !hit {
		t.Fatalf("pass must not skip walls, vx=%v p=%+v", u.v.X, u.p)
	}
}

func parkedRangedPair(t *testing.T) (*unit, *unit, *Match) {
	t.Helper()
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindRanged)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	m.mu.Lock()
	defer m.mu.Unlock()
	var fighters []*unit
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.role == unitpkg.RoleFighter {
			fighters = append(fighters, u)
		}
	}
	if len(fighters) < 2 {
		t.Fatal("need two fighters")
	}
	return fighters[0], fighters[1], m
}
