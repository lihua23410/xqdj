package sim

import (
	"math"
	"testing"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func patrolLockMatch(t *testing.T) (*Match, *unit, *unit) {
	t.Helper()
	m := NewMatchSeeded(7)
	m.SetSlot(0, character.KindPatrol)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	t.Cleanup(m.End)
	m.mu.Lock()
	p, enemy := patrolUnits(m)
	p.p, enemy.p = vec{-120, 0}, vec{140, 120}
	m.mu.Unlock()
	return m, p, enemy
}

func TestPatrolOwnedVelocityLockRejectsSpeedForceAndImpulse(t *testing.T) {
	m, p, _ := patrolLockMatch(t)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.applyCmdLocked(unitpkg.AddFS{UnitID: p.id, Token: 7, DX: 1, BaseSpeed: 800, ExpiresAt: 100})
	marker := m.addUnitLocked(character.KindPatrolMarker, vec{-70, 80}, vec{}, p.id, p.slot)
	if !p.velocityLocked || p.v.len2() != 0 || len(p.fsList) != 0 {
		t.Fatal("existing impulse was not cleared on marker birth")
	}
	for _, cmd := range []unitpkg.Cmd{
		unitpkg.SetVelocity{UnitID: p.id, VX: 900, VY: 400},
		unitpkg.Force{UnitID: p.id, AX: 90000, AY: 40000},
		unitpkg.AddFS{UnitID: p.id, Token: 8, DX: 1, DY: 1, BaseSpeed: 900, ExpiresAt: 100},
	} {
		m.applyCmdLocked(cmd)
		m.applyImpulseLocked()
		if p.v.len2() != 0 || len(p.fsList) != 0 {
			t.Fatalf("movement command bypassed lock: %T v=%v fs=%v", cmd, p.v, p.fsList)
		}
	}
	// 眩晕不发 Sense，约束仍然必须拒绝速度；巡航恢复也不能顶开它。
	m.applyCmdLocked(unitpkg.Stun{UnitID: p.id, Hold: true, Until: 100})
	m.applyCmdLocked(unitpkg.SetVelocity{UnitID: p.id, VX: 500})
	m.decelerateLocked(DT)
	p.setVel(vec{900, 100})
	if p.v.len2() != 0 {
		t.Fatal("stun or cruise bypassed lock")
	}
	// 最后一颗离场才解除约束；被丢弃的击退不再复活。
	m.removeLocked(marker)
	p.setVel(vec{155, 0})
	m.applyImpulseLocked()
	if p.velocityLocked || p.v.X != 155 || p.v.Y != 0 || len(p.fsList) != 0 {
		t.Fatalf("discarded impulse returned after unlock: v=%v fs=%v", p.v, p.fsList)
	}
}

func TestPatrolVelocityLockAllowsTeleportAndWallDisplacement(t *testing.T) {
	m, p, enemy := patrolLockMatch(t)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.addUnitLocked(character.KindPatrolMarker, vec{-70, 80}, vec{}, p.id, p.slot)
	m.applyCmdLocked(unitpkg.Teleport{UnitID: p.id, X: -20, Y: -40})
	if p.p != (vec{-20, -40}) || p.v.len2() != 0 {
		t.Fatalf("teleport blocked or added speed: p=%v v=%v", p.p, p.v)
	}
	before := p.p
	m.placeWallLocked(unitpkg.PlaceWall{OwnerID: enemy.id, Kind: enemy.kind, X1: -90, Y1: -40, X2: 50, Y2: -40, Radius: 6, Life: 20})
	m.constrainUnitLocked(p)
	if p.p == before || p.v.len2() != 0 {
		t.Fatalf("wall displacement blocked or added speed: before=%v p=%v v=%v", before, p.p, p.v)
	}
}

func TestPatrolVelocityLockTracksAllCurrentOwnedMarkers(t *testing.T) {
	m, p, enemy := patrolLockMatch(t)
	m.mu.Lock()
	defer m.mu.Unlock()
	foreign := m.addUnitLocked(character.KindPatrolMarker, vec{80, 80}, vec{}, enemy.id, p.slot)
	if p.velocityLocked {
		t.Fatal("same-side foreign marker locked patrol")
	}
	first := m.addUnitLocked(character.KindPatrolMarker, vec{-70, 80}, vec{}, p.id, p.slot)
	second := m.addUnitLocked(character.KindPatrolMarker, vec{-70, -80}, vec{}, p.id, p.slot)
	m.removeLocked(first)
	if !p.velocityLocked {
		t.Fatal("removing one of two markers unlocked patrol")
	}
	m.removeLocked(foreign)
	if !p.velocityLocked {
		t.Fatal("foreign marker removal changed lock")
	}
	m.removeLocked(second)
	if p.velocityLocked {
		t.Fatal("last marker removal did not unlock patrol")
	}
	p.setVel(vec{155, 0})
	m.addUnitLocked(character.KindPatrolMarker, vec{-70, 80}, vec{}, p.id, p.slot)
	if !p.velocityLocked || p.v.len2() != 0 {
		t.Fatal("returned stolen marker did not lock moving patrol again")
	}
}

func TestPatrolLastMarkerLeavesDuringStun(t *testing.T) {
	m, p, _ := patrolLockMatch(t)
	m.mu.Lock()
	marker := m.addUnitLocked(character.KindPatrolMarker, vec{-70, 80}, vec{}, p.id, p.slot)
	m.settleHitsLocked()
	m.holdStillLocked(p.id, m.time+1)
	m.removeLocked(marker)
	m.mu.Unlock()
	waitTicks(m, 15)
	m.mu.Lock()
	if p.velocityLocked || !p.stun || p.v.len2() != 0 {
		m.mu.Unlock()
		t.Fatalf("last marker changed stun or retained lock: locked=%v stun=%v v=%v", p.velocityLocked, p.stun, p.v)
	}
	m.mu.Unlock()
	waitTicks(m, 50)
	m.mu.Lock()
	defer m.mu.Unlock()
	if p.stun || math.Abs(p.v.len()-155) > 1e-6 || p.stand {
		t.Fatalf("did not resume after existing stun: stun=%v stand=%v speed=%v", p.stun, p.stand, p.v.len())
	}
}
