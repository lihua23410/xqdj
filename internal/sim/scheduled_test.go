package sim

import (
	"errors"
	"slices"
	"testing"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
	六边形 "xqdj/map/hex"
	大六边形 "xqdj/map/widehex"
)

func TestScheduledBattleKeepsSidesAndRecordsResult(t *testing.T) {
	m := NewMatchSeeded(31)
	t.Cleanup(m.End)
	sides := [][]string{
		{character.KindArcher, character.KindAxe},
		{character.KindMelee, character.KindRanged},
	}
	battle, err := m.StartScheduled("first", sides)
	if err != nil || battle.Status != "running" || (battle.Field != 六边形.Name && battle.Field != 大六边形.Name) {
		t.Fatalf("start = %+v, %v", battle, err)
	}
	if retry, err := m.StartScheduled("first", sides); err != nil || retry != battle {
		t.Fatalf("retry = %+v, %v; want %+v", retry, err, battle)
	}
	if _, err := m.StartScheduled("second", sides); !errors.Is(err, ErrBattleInProgress) {
		t.Fatalf("second start should be blocked, got %v", err)
	}
	m.mu.Lock()
	if !m.team || !slices.Equal(m.lineup[0], sides[0]) || !slices.Equal(m.lineup[1], sides[1]) {
		got := m.lineup
		m.mu.Unlock()
		t.Fatalf("unexpected team lineup: %v", got)
	}
	for _, id := range slices.Clone(m.order) {
		u := m.units[id]
		if u != nil && u.role == unitpkg.RoleFighter && u.slot == 0 {
			m.removeLocked(u)
		}
	}
	m.checkWinLocked()
	m.mu.Unlock()
	finished, ok := m.ScheduledStatus("first")
	if !ok || finished.Field != battle.Field || finished.Status != "finished" || finished.Result != "right" {
		t.Fatalf("unexpected result: %+v, %v", finished, ok)
	}
	if _, err := m.StartScheduled("second", sides); err != nil {
		t.Fatal(err)
	}
	m.End()
	if stopped, ok := m.ScheduledStatus("second"); !ok || stopped.Status != "aborted" || stopped.Result != "" {
		t.Fatalf("unexpected stopped result: %+v, %v", stopped, ok)
	}
	if got, ok := m.ScheduledStatus("first"); !ok || got != finished {
		t.Fatalf("first result was lost: %+v, %v", got, ok)
	}
}
