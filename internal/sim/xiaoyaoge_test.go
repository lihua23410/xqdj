package sim

import (
	"testing"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestTemporaryFighterIsSummonedWithoutKeepingTeamAlive(t *testing.T) {
	m := NewHeadlessMatchSeeded(17)
	m.SetSlot(0, character.KindXiaoyaoge)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	t.Cleanup(m.End)

	m.mu.Lock()
	defer m.mu.Unlock()

	owner := fighterSlot(m, 0)
	if owner == nil {
		t.Fatal("missing xiaoyaoge")
	}
	before := len(m.order)
	m.applyCmdLocked(unitpkg.Spawn{
		Kind:      character.KindArcher,
		X:         owner.p.X,
		Y:         owner.p.Y,
		OwnerID:   owner.id,
		Slot:      owner.slot,
		Temporary: true,
	})
	if len(m.order) != before+1 {
		t.Fatal("temporary fighter was not spawned")
	}

	var helper *unit
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.kind == character.KindArcher && u.owner == owner.id {
			helper = u
			break
		}
	}
	if helper == nil {
		t.Fatal("missing summoned fighter")
	}
	if !helper.summoned {
		t.Fatal("temporary fighter was not marked as summoned")
	}
	if got := m.livingFighterLocked(0); got != owner {
		t.Fatal("summoned fighter replaced the side's real fighter")
	}

	owner.stopped = true
	m.checkWinLocked()
	phase, winner := m.phase, m.winner
	owner.stopped = false
	if phase != PhaseEnded || winner != character.KindDummy {
		t.Fatalf("summoned fighter kept its side alive: phase=%s winner=%s", phase, winner)
	}
}
