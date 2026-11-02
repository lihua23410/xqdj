package sim

import (
	"testing"

	"xqdj/character"
	"xqdj/character/acausal"
	unitpkg "xqdj/internal/unit"
)

func TestHeadlessMatchKeepsDamageAndOmitsLogsAcrossRestarts(t *testing.T) {
	m := NewHeadlessMatchSeeded(7)
	defer m.End()
	for i := 0; i < 2; i++ {
		m.SetSlot(0, character.KindDummy)
		m.SetSlot(1, character.KindDummy)
		m.Start()
		m.mu.Lock()
		left, right := fighterSlot(m, 0), fighterSlot(m, 1)
		before := right.hp
		confirmHit(m, left, right, 10)
		after := right.hp
		confirmHit(m, left, right, after)
		m.checkWinLocked()
		winnerID := m.winnerID
		m.mu.Unlock()
		if after != before-10 {
			t.Fatalf("damage lost: before=%v after=%v", before, after)
		}
		if winner, ticks := m.Play(5); winner != character.KindDummy || ticks != 1 || winnerID != left.id {
			t.Fatalf("headless match lost win: %q tick=%d id=%d", winner, ticks, winnerID)
		}
		m.End()
		if recap, trace := m.LogText(); recap != "" || trace != "" {
			t.Fatalf("headless match kept logs: %q %q", recap, trace)
		}
		if len(m.logNames) != 0 || len(m.logLeapAt) != 0 || len(m.logLeaps) != 0 {
			t.Fatal("headless match kept logging metadata")
		}
	}
}

func TestHeadlessAcausalKeepsProbeTapeWithoutFilm(t *testing.T) {
	m := NewHeadlessMatchSeeded(9)
	m.SetSlot(0, 非因果.KindAcausal)
	m.SetSlot(1, character.KindKnight)
	m.Start()
	defer m.End()
	tape := m.probeTape
	if tape == nil || len(tape.Drops) == 0 || len(tape.Self) < 2 || len(tape.Bins) == 0 || len(tape.Fighters) != 1 || len(tape.Fighters[0].Samples) < 2 {
		t.Fatal("headless match lost combat probe data")
	}
	if len(m.activeOpening) != 0 || len(m.pendingOpening) != 0 || len(m.recordPresentations) != 0 || len(m.recordInitial) != 0 {
		t.Fatal("headless match kept rewind presentation")
	}
}

func TestNearbySnapshotsFilteringAndOwnership(t *testing.T) {
	snaps := []unitpkg.Snapshot{
		{ID: 1, X: 1, Y: 1},
		{ID: 2, X: 4, Y: 5}, // 视野边界。
		{ID: 3, X: 100, OwnerID: 1},
		{ID: 4, X: 10},
		{ID: 5, X: 2},
	}
	u := &unit{id: 1, p: vec{1, 1}, vision: 5}
	got := nearbySnapshots(snaps, u)
	if len(got) != 3 || got[0].ID != 2 || got[1].ID != 3 || got[2].ID != 5 {
		t.Fatalf("nearby %+v", got)
	}
	got[0].X = -1
	if snaps[1].X != 4 {
		t.Fatal("nearby list aliases the shared snapshot array")
	}
	if next := nearbySnapshots(snaps, u); next[0].X != 4 || got[0].X != -1 {
		t.Fatal("new sense overwrote a previously retained list")
	}
	if got := nearbySnapshots(snaps[:1], u); got != nil {
		t.Fatalf("self-only sense has nearby %+v", got)
	}
}

func BenchmarkNearbySnapshots(b *testing.B) {
	snaps := make([]unitpkg.Snapshot, 64)
	for i := range snaps {
		snaps[i].ID = uint64(i + 2)
	}
	u := &unit{id: 1, vision: 100}
	b.Run("preallocated", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = nearbySnapshots(snaps, u)
		}
	})
	b.Run("appendBaseline", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var nearby []unitpkg.Snapshot
			for _, o := range snaps {
				dx, dy := o.X-u.p.X, o.Y-u.p.Y
				if o.ID != u.id && (dx*dx+dy*dy <= u.vision*u.vision || o.OwnerID == u.id) {
					nearby = append(nearby, o)
				}
			}
		}
	})
}
