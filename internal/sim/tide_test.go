package sim

import (
	"math"
	"testing"
	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestTideAcceleratesAndKeepsNewDirectionAfterStopping(t *testing.T) {
	m := NewMatchSeeded(71)
	m.SetSlot(0, character.KindTide)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()

	u := fighterByKind(m, character.KindTide)
	if u == nil || u.v != (vec{}) || math.Abs(u.accelerationDir.len()-1) > 1e-9 {
		t.Fatal("tide should spawn at rest with a starting direction")
	}
	dir := u.accelerationDir
	for i := 0; i < 4*TickHz; i++ {
		m.decelerateLocked(DT)
	}
	assertVelocity(t, u.v, dir.mul(320))

	turned := perp(dir)
	m.resolveLocked(ccdHit{kind: hitWall, a: u.id, n: dir.sub(turned)})
	assertVelocity(t, u.v, turned.mul(160))
	m.resolveLocked(ccdHit{kind: hitWall, a: u.id, n: turned})
	assertVelocity(t, u.v, vec{})
	m.decelerateLocked(DT)
	assertVelocity(t, u.v, turned.mul(-80*DT))

	m.applyCmdLocked(unitpkg.AddFS{UnitID: u.id, DX: dir.X, DY: dir.Y, BaseSpeed: 220, Token: 1})
	m.decelerateLocked(DT)
	assertVelocity(t, u.v, dir.mul(220))
	m.applyCmdLocked(unitpkg.RemoveFS{UnitID: u.id, Token: 1})
	m.decelerateLocked(DT)
	assertVelocity(t, u.v, dir.mul(220+80*DT))
}

func TestTideWaterHitsThroughNormalMatchTicks(t *testing.T) {
	m := NewMatchSeeded(91)
	m.SetSlot(0, character.KindTide)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()
	m.mu.Lock()
	tide := fighterByKind(m, character.KindTide)
	target := fighterByKind(m, character.KindDummy)
	tide.p, target.p = vec{}, vec{12, 40}
	tide.setVel(vec{160, 0})
	target.stand = true
	target.setVel(vec{})
	m.mu.Unlock()

	for i := 0; i < 4; i++ {
		m.Tick()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if target.hp >= target.maxHP-1.5 || target.v.Y <= 0 || math.Abs(target.cruiseMul()-0.76) > 1e-9 || m.hitStop != 0 {
		t.Fatalf("target: hp=%v/%v velocity=%v multiplier=%v hitStop=%d", target.hp, target.maxHP, target.v, target.cruiseMul(), m.hitStop)
	}
}

func TestTideForceDoesNotTriggerTurnSpeedLoss(t *testing.T) {
	m := NewMatchSeeded(103)
	m.SetSlot(0, character.KindTide)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()

	u := fighterByKind(m, character.KindTide)
	u.setVelNoTurnLoss(vec{200, 0})
	m.applyCmdLocked(unitpkg.Force{UnitID: u.id, AX: 0, AY: 120 * TickHz})
	assertVelocity(t, u.v, vec{200, 120})
}

func TestTideEbbAndMomentumShieldInMatch(t *testing.T) {
	m := NewMatchSeeded(115)
	m.SetSlot(0, character.KindTide)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()

	m.mu.Lock()
	tide := fighterByKind(m, character.KindTide)
	target := fighterByKind(m, character.KindDummy)
	tide.p = vec{240, 0}
	tide.setVelNoTurnLoss(vec{260, 0})
	target.p = vec{140, 20}
	target.setVel(vec{})
	// Test high-speed Momentum Shield damage reduction.
	m.offerDamageLocked(unitpkg.Damage{From: target.id, To: tide.id, Amount: 20, NoFreeze: true})
	m.mu.Unlock()

	m.Tick()
	m.mu.Lock()
	if tide.hp <= tide.maxHP-19 || tide.hp >= tide.maxHP-15 {
		// Expect ~17.4 HP lost after 260 speed mitigation (13% reduction)
		t.Fatalf("expected mitigated hp loss, got hp=%v/%v", tide.hp, tide.maxHP)
	}
	// Now drive tide straight into wall at high speed to trigger Ebb pull + launch.
	tide.p = vec{255, 0}
	tide.setVelNoTurnLoss(vec{280, 0})
	target.p = vec{170, 0}
	target.setVel(vec{})
	m.mu.Unlock()

	pulled := false
	launched := false
	for i := 0; i < TickHz; i++ {
		m.Tick()
		m.mu.Lock()
		if target.v.X > 10 {
			pulled = true
		}
		if i > int(0.50*TickHz) && tide.v.len() >= 130 {
			launched = true
		}
		m.mu.Unlock()
	}
	if !pulled {
		t.Fatal("expected Ebb to pull nearby target toward tide")
	}
	if !launched {
		t.Fatal("expected Ebb launch to restore high speed after 0.55s")
	}
}

func assertVelocity(t *testing.T, got, want vec) {
	t.Helper()
	if got.sub(want).len() > 1e-8 {
		t.Fatalf("velocity = %v, want %v", got, want)
	}
}
