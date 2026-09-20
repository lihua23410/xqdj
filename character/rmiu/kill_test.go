package r缪

import (
	"testing"
	"xqdj/internal/unit"
)

func TestClaimKillOnLastHit(t *testing.T) {
	resetMiuKillState()
	out := make(chan unit.Cmd, 16)
	m := meleeMiu()
	m.prevMiuIDs = map[uint64]bool{2: true}
	noteMiuHit(2, 10, KindMiu)
	ctx := unit.Context{ID: 10, Kind: KindMiu, Out: out}
	m.Handle(ctx, unit.Sense{Time: 1, Self: miuSelf(), Nearby: nil})
	if m.killBonus != killBonusPer {
		t.Fatalf("bonus=%v want %v", m.killBonus, killBonusPer)
	}
	if mathAbs(m.speed-(miuBaseSpeed+killBonusPer)) > 1e-9 {
		t.Fatalf("speed=%v", m.speed)
	}
}

func TestClaimKillIgnoresOtherHitter(t *testing.T) {
	resetMiuKillState()
	out := make(chan unit.Cmd, 16)
	m := meleeMiu()
	m.prevMiuIDs = map[uint64]bool{2: true}
	noteMiuHit(2, 99, KindMiu)
	ctx := unit.Context{ID: 10, Kind: KindMiu, Out: out}
	m.Handle(ctx, unit.Sense{Time: 1, Self: miuSelf(), Nearby: nil})
	if m.killBonus != 0 {
		t.Fatalf("bonus=%v, should not claim others' kills", m.killBonus)
	}
}

func TestBodySacrificeClaimsKill(t *testing.T) {
	resetMiuKillState()
	out := make(chan unit.Cmd, 32)
	r := &R缪{
		booted: true, hp: 8, x: 0, y: 0,
		hasBest: true, bestID: 2, bestHP: 90, bestX: 40, bestY: 0,
		prevMiuIDs:   map[uint64]bool{2: true},
		speed:        miuBaseSpeed,
		meleeReadyAt: 999, fireReadyAt: 999,
	}
	ctx := unit.Context{ID: 1, Kind: KindRMiu, Out: out}
	r.Handle(ctx, unit.IncomingDamage{Token: 7, From: 99, Amount: 20})
	_ = drain(out)
	r.Handle(ctx, unit.Sense{
		Time:   1,
		Self:   bodySelf(40, 0, 90, 0, 20),
		Nearby: nil,
	})
	if r.killBonus != killBonusPer {
		t.Fatalf("sacrifice should grant kill bonus, got %v", r.killBonus)
	}
}

func TestLowHPSpeedNotInherited(t *testing.T) {
	resetMiuKillState()
	out := make(chan unit.Cmd, 16)
	m := meleeMiu()
	ctx := unit.Context{ID: 10, Kind: KindMiu, Out: out}
	self := miuSelf()
	self.HP = 40
	m.Handle(ctx, unit.Sense{Time: 1, Self: self})
	if m.killBonus != 0 {
		t.Fatalf("low HP must not write kill bonus, got %v", m.killBonus)
	}
	want := miuBaseSpeed + lowHPBoost
	if mathAbs(m.speed-want) > 1e-9 {
		t.Fatalf("speed=%v want %v", m.speed, want)
	}
	miuMu.Lock()
	stored := miuBonus[10]
	miuMu.Unlock()
	if stored != 0 {
		t.Fatalf("inherited store=%v want 0", stored)
	}

	resetMiuKillState()
	m = meleeMiu()
	m.prevMiuIDs = map[uint64]bool{2: true}
	noteMiuHit(2, 10, KindMiu)
	miuMu.Lock()
	miuBonus[2] = killBonusPer
	miuMu.Unlock()
	full := miuSelf()
	m.Handle(ctx, unit.Sense{Time: 2, Self: full})
	if mathAbs(m.killBonus-(killBonusPer+killBonusPer)) > 1e-9 {
		t.Fatalf("inherit kill only, bonus=%v", m.killBonus)
	}
	if mathAbs(m.speed-(miuBaseSpeed+2*killBonusPer)) > 1e-9 {
		t.Fatalf("full HP speed=%v, victim low-HP boost must not inherit", m.speed)
	}
}

func mathAbs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
