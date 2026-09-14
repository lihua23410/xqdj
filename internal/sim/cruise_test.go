package sim

import (
	"math"
	"testing"
	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestSetCruiseDoesNotChangeVelocity(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindEngine)
	m.SetSlot(1, character.KindWaller)
	m.Start()
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	u := fighterByKind(m, character.KindEngine)
	if u == nil {
		t.Fatal("missing 内燃机")
	}
	u.setVel(vec{80, 0})
	m.applyCmdLocked(unitpkg.SetCruise{UnitID: u.id, Speed: 300})
	if u.cruise != 300 {
		t.Fatalf("cruise=%v", u.cruise)
	}
	if math.Abs(u.v.X-80) > 1e-9 || u.v.Y != 0 {
		t.Fatalf("vel=%+v", u.v)
	}
}

func TestSetVisionChangesSenseRadius(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindEngine)
	m.SetSlot(1, character.KindWaller)
	m.Start()
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	u := fighterByKind(m, character.KindEngine)
	if u == nil {
		t.Fatal("missing 内燃机")
	}
	m.applyCmdLocked(unitpkg.SetVision{UnitID: u.id, Vision: 40})
	if u.vision != 40 {
		t.Fatalf("vision=%v", u.vision)
	}
}

func TestCruisePullsDownAndPushesUp(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindWaller)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	defer m.End()
	m.mu.Lock()
	u := fighterByKind(m, character.KindWaller)
	if u == nil {
		t.Fatal("missing 筑墙者")
	}
	u.cruise = 120
	u.decelT = 0
	u.setVel(vec{200, 0})
	m.decelerateLocked(0.2)
	if math.Abs(u.v.len()-190) > 1e-6 {
		t.Fatalf("overspeed after 0.2s = %v", u.v.len())
	}
	u.decelT = 0
	u.setVel(vec{50, 0})
	m.decelerateLocked(0.2)
	if math.Abs(u.v.len()-55) > 1e-6 {
		t.Fatalf("underspeed after 0.2s = %v", u.v.len())
	}
	m.mu.Unlock()
}

func TestCruiseZeroSpeedDoesNotPush(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindWaller)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	u := fighterByKind(m, character.KindWaller)
	u.cruise = 120
	u.decelT = 0
	u.setVel(vec{})
	m.decelerateLocked(1)
	if u.v.len() > 1e-9 {
		t.Fatalf("zero speed was pushed to %v", u.v.len())
	}
}
