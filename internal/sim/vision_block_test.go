package sim

import (
	"testing"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

type visionNop struct{}

func (visionNop) Handle(unitpkg.Context, unitpkg.Event) {}

const visionBlockStubKind = "视线测试桩"

func registerVisionBlockStub(t *testing.T) {
	t.Helper()
	if _, ok := unitpkg.Lookup(visionBlockStubKind); ok {
		return
	}
	unitpkg.Register(unitpkg.Spec{
		Kind: visionBlockStubKind, Role: unitpkg.RoleHelper,
		Radius: 10, MaxHP: 1, Vision: 0, VisionBlock: true,
	}, func(unitpkg.SpawnInfo) unitpkg.Actor { return visionNop{} })
}

func TestVisionBlockSpecFlowsIntoSnapshot(t *testing.T) {
	registerVisionBlockStub(t)
	m := newWallMatch(t)
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	u := m.addUnitLocked(visionBlockStubKind, vec{0, 0}, vec{0, 0}, 0, 0)
	if u == nil {
		t.Fatal("stub spawn failed")
	}
	if !u.visionBlock {
		t.Fatal("unit.visionBlock not set from spec")
	}
	if !u.snap().VisionBlock {
		t.Fatal("snapshot.VisionBlock not set")
	}
	m.applyCmdLocked(unitpkg.VisionBlock{UnitID: u.id, Hold: false})
	if u.visionBlock {
		t.Fatal("token did not clear")
	}
	if u.snap().VisionBlock {
		t.Fatal("snapshot still carries VisionBlock after token clear")
	}
	m.applyCmdLocked(unitpkg.VisionBlock{UnitID: u.id, Hold: true})
	if !u.visionBlock {
		t.Fatal("token did not set")
	}
}

func TestVisionBlockPlaceWallFlowsIntoWallView(t *testing.T) {
	m := newWallMatch(t)
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	owner := fighterByKind(m, character.KindWaller)
	m.placeWallLocked(unitpkg.PlaceWall{
		OwnerID: owner.id, Slot: owner.slot, Kind: owner.kind,
		X1: -20, Y1: 0, X2: 20, Y2: 0, Radius: 6, Life: -1,
		VisionBlock: true,
	})
	if len(m.walls) != 1 {
		t.Fatalf("walls=%d", len(m.walls))
	}
	if !m.walls[0].visionBlock {
		t.Fatal("barrier.visionBlock not set")
	}
	views := m.wallViewsLocked()
	if len(views) != 1 || !views[0].VisionBlock {
		t.Fatal("wall view does not carry VisionBlock")
	}
}

func TestOctagonFieldBootsVisionBlockWalls(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetField("八边形")
	m.SetSlot(0, character.KindMelee)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	defer m.End()
	for i := 0; i < 4; i++ {
		m.Tick()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.walls) != 3 {
		t.Fatalf("walls=%d want 3", len(m.walls))
	}
	for _, w := range m.walls {
		if !w.hard || !w.visionBlock {
			t.Fatalf("octagon wall %+v must be hard vision-block", w)
		}
	}
	views := m.wallViewsLocked()
	if len(views) != 3 {
		t.Fatalf("views=%d want 3", len(views))
	}
	for _, v := range views {
		if !v.VisionBlock {
			t.Fatal("octagon wall view missing VisionBlock")
		}
	}
}
