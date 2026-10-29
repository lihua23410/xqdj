package sim

import (
	"bytes"
	"encoding/json"
	"testing"
	"xqdj/character"
)

func TestRecordingKeepsIntroAfterLiveIntroAndMatchEnd(t *testing.T) {
	m := NewMatchSeeded(9)
	m.SetSlot(0, character.KindAcausal)
	m.SetSlot(1, character.KindKnight)
	m.Start()
	defer m.End()
	initial := m.Recording()
	if initial.Intro.Duration() <= 0 || len(initial.Initial) == 0 || len(initial.Frames) != 0 {
		t.Fatal("missing separate initial state and intro")
	}
	var snap struct {
		Time  float64
		Intro json.RawMessage
		Units []json.RawMessage
	}
	if err := json.Unmarshal(initial.Initial, &snap); err != nil {
		t.Fatal(err)
	}
	if snap.Time != 0 || len(snap.Intro) != 0 || len(snap.Units) != 2 {
		t.Fatalf("initial=%s", initial.Initial)
	}
	m.FinishIntro()
	m.Tick()
	m.recordSnapshot(m.SnapshotJSON())
	m.End()
	finished := m.Recording()
	if finished.Intro != initial.Intro || !bytes.Equal(finished.Initial, initial.Initial) || len(finished.Frames) != 1 {
		t.Fatal("live intro completion or End lost recording")
	}
	if bytes.Contains(finished.Frames[0], []byte(`"intro"`)) {
		t.Fatal("intro duplicated into battle frame")
	}
	m.SetSlot(0, character.KindDummy)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	next := m.Recording()
	if next.Intro != nil || len(next.Initial) != 0 || len(next.Frames) != 0 {
		t.Fatal("next match inherited an intro")
	}
	if finished.Intro.Duration() <= 0 || len(finished.Frames) != 1 {
		t.Fatal("new match invalidated the old export")
	}
}
