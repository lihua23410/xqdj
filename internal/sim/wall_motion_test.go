package sim

import (
	"math"
	"testing"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestPersistentWallOutlivesClock(t *testing.T) {
	m := newWallMatch(t)
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	owner := fighterByKind(m, character.KindWaller)
	m.placeWallLocked(unitpkg.PlaceWall{
		OwnerID: owner.id, Slot: owner.slot, Kind: owner.kind,
		X1: -20, Y1: 0, X2: 20, Y2: 0, Radius: 6, Life: -1, Amount: 1,
	})
	m.time = 100
	m.expireWallsLocked()
	if len(m.walls) != 1 {
		t.Fatalf("walls=%d", len(m.walls))
	}
}

func TestScrapeGap(t *testing.T) {
	m := newWallMatch(t)
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	owner := fighterByKind(m, character.KindWaller)
	enemy := fighterByKind(m, character.KindRanged)
	m.placeWallLocked(unitpkg.PlaceWall{
		OwnerID: owner.id, Slot: owner.slot, Kind: owner.kind,
		X1: -20, Y1: 0, X2: 20, Y2: 0, Radius: 6, Life: -1, Amount: 3, HitGap: 0.3,
	})
	w := m.walls[0]
	m.time = 1
	m.scrapeWallLocked(enemy, w)
	m.scrapeWallLocked(enemy, w)
	if len(m.pending) != 1 {
		t.Fatalf("pending=%d", len(m.pending))
	}
	m.time = 1.29
	m.scrapeWallLocked(enemy, w)
	if len(m.pending) != 1 {
		t.Fatalf("early pending=%d", len(m.pending))
	}
	m.time = 1.3
	m.scrapeWallLocked(enemy, w)
	if len(m.pending) != 2 {
		t.Fatalf("later pending=%d", len(m.pending))
	}
}

func TestHoldStillRestoresVelocity(t *testing.T) {
	m := newWallMatch(t)
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	u := fighterByKind(m, character.KindWaller)
	u.setVel(vec{155, 0})
	m.holdStillLocked(u.id, 1)
	if u.v.len() != 0 || math.Abs(u.holdVel.X-155) > 1e-9 || !u.stun {
		t.Fatalf("held vel=%v saved=%v stun=%v", u.v, u.holdVel, u.stun)
	}
	m.pinHeldLocked()
	m.decelerateLocked(1)
	if u.v.len() != 0 {
		t.Fatalf("cruise snapped held unit to %v", u.v.len())
	}
	m.time = 1
	m.releaseHoldLocked()
	if u.held || u.stun || math.Abs(u.v.X-155) > 1e-9 {
		t.Fatalf("wake vel=%v held=%v stun=%v", u.v, u.held, u.stun)
	}
}

func TestHoldStillZeroWakesIntoCruise(t *testing.T) {
	m := newWallMatch(t)
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	u := fighterByKind(m, character.KindWaller)
	u.setVel(vec{})
	u.cruise = 155
	u.cruiseFS.dir = vec{0, 1}
	m.holdStillLocked(u.id, 1)
	m.time = 1
	m.releaseHoldLocked()
	m.decelerateLocked(1)
	if math.Abs(u.v.X) > 1e-6 || math.Abs(u.v.Y-155) > 1e-6 {
		t.Fatalf("wake vel=%v", u.v)
	}
}

func TestOwnerDeathDropsWalls(t *testing.T) {
	m := newWallMatch(t)
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	owner := fighterByKind(m, character.KindWaller)
	m.placeWallLocked(unitpkg.PlaceWall{
		OwnerID: owner.id, Slot: owner.slot, Kind: owner.kind,
		X1: -20, Y1: 0, X2: 20, Y2: 0, Radius: 6, Life: -1, WithOwner: true,
	})
	m.removeLocked(owner)
	if len(m.walls) != 0 {
		t.Fatalf("walls=%d", len(m.walls))
	}
}

func TestRamOncePerWall(t *testing.T) {
	m := newWallMatch(t)
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	owner := fighterByKind(m, character.KindWaller)
	enemy := fighterByKind(m, character.KindRanged)
	owner.p = vec{80, 80}
	enemy.p = vec{0, 4}
	enemy.radius = 18
	m.placeWallLocked(unitpkg.PlaceWall{
		OwnerID: owner.id, Slot: owner.slot, Kind: owner.kind,
		X1: -40, Y1: 0, X2: 40, Y2: 0, Radius: 6, Life: -1,
	})
	m.placeWallLocked(unitpkg.PlaceWall{
		OwnerID: owner.id, Slot: owner.slot, Kind: owner.kind,
		X1: -40, Y1: 8, X2: 40, Y2: 8, Radius: 6, Life: -1,
	})
	m.setWallMotionLocked(unitpkg.SetWallMotion{WallID: m.walls[0].id, Ram: 8})
	m.setWallMotionLocked(unitpkg.SetWallMotion{WallID: m.walls[1].id, Ram: 8})
	m.ramWallLocked(m.walls[0])
	m.ramWallLocked(m.walls[1])
	m.ramWallLocked(m.walls[0])
	n := 0
	for _, d := range m.pending {
		if d.To == enemy.id && d.Amount == 8 {
			n++
		}
		if d.To == owner.id {
			t.Fatal("owner took ram")
		}
	}
	if n != 2 {
		t.Fatalf("rams=%d", n)
	}
}

func TestOverlappingWallsSlam(t *testing.T) {
	m := newWallMatch(t)
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	owner := fighterByKind(m, character.KindWaller)
	enemy := fighterByKind(m, character.KindRanged)
	owner.p = vec{0, 0}
	owner.setVel(vec{40, 0})
	enemy.p = vec{0, 400}
	m.placeWallLocked(unitpkg.PlaceWall{
		OwnerID: owner.id, Slot: owner.slot, Kind: owner.kind,
		X1: -40, Y1: 0, X2: 40, Y2: 0, Radius: 6, Life: -1,
	})
	m.placeWallLocked(unitpkg.PlaceWall{
		OwnerID: owner.id, Slot: owner.slot, Kind: owner.kind,
		X1: -40, Y1: 8, X2: 40, Y2: 8, Radius: 6, Life: -1,
	})
	for _, w := range m.walls {
		m.setWallMotionLocked(unitpkg.SetWallMotion{
			WallID: w.id, StunRadius: 24, StunDur: 1,
		})
	}
	m.spinWallsLocked(DT)
	if len(m.walls) != 0 {
		t.Fatalf("walls=%d", len(m.walls))
	}
	if owner.held || owner.stun || math.Abs(owner.v.X-40) > 1e-9 {
		t.Fatalf("owner held=%v stun=%v vel=%v", owner.held, owner.stun, owner.v)
	}
	if enemy.held {
		t.Fatal("far enemy was stunned")
	}
}

func newWallMatch(t *testing.T) *Match {
	t.Helper()
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindWaller)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	return m
}
