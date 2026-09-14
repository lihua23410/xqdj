package sim

import (
	"testing"
	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestNoFrameFreezeSkipsHitStop(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindWaller)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	src := fighterByKind(m, character.KindWaller)
	dst := fighterByKind(m, character.KindRanged)
	if src == nil || dst == nil {
		t.Fatal("missing fighters")
	}
	m.applyCmdLocked(unitpkg.NoFrameFreeze{UnitID: src.id, Hold: true})
	deal(m, src.id, dst.id, 6)
	if m.hitStop != 0 {
		t.Fatalf("hitStop=%d want 0", m.hitStop)
	}
}

func TestDamageWithoutTokenStillHitStops(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindWaller)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	src := fighterByKind(m, character.KindWaller)
	dst := fighterByKind(m, character.KindRanged)
	deal(m, src.id, dst.id, 6)
	if m.hitStop != HitStopFrames {
		t.Fatalf("hitStop=%d want %d", m.hitStop, HitStopFrames)
	}
}

func TestNoFrameFreezeFollowsOwner(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindWaller)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	src := fighterByKind(m, character.KindWaller)
	dst := fighterByKind(m, character.KindRanged)
	m.applyCmdLocked(unitpkg.NoFrameFreeze{UnitID: src.id, Hold: true})
	shot := m.addUnitLocked("子弹", src.p, vec{}, src.id, src.slot)
	if shot == nil {
		t.Fatal("missing shot")
	}
	deal(m, shot.id, dst.id, 6)
	if m.hitStop != 0 {
		t.Fatalf("owned shot hitStop=%d want 0", m.hitStop)
	}
}

func deal(m *Match, from, to uint64, amt float64) {
	m.dmgSeq++
	token := m.dmgSeq
	m.pendingDmg[token] = dmgOffer{from: from, to: to, amount: amt}
	m.confirmDamageLocked(unitpkg.ConfirmDamage{Token: token, UnitID: to, Amount: amt})
}
