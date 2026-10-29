package sim

import (
	"encoding/json"
	"testing"
	"xqdj/character"
	"xqdj/internal/replay"
)

func TestQueueOpeningIsGenericAndOwnsPayload(t *testing.T) {
	m := NewMatchSeeded(4)
	m.SetSlot(0, character.KindDummy)
	m.SetSlot(1, character.KindDummy)
	p := replay.Presentation{Pack: character.KindKnight, Type: "banner", Duration: 0.12, Data: json.RawMessage(`{"message":"hello"}`)}
	first, err := m.QueueOpening(p)
	if err != nil {
		t.Fatal(err)
	}
	p.Data[2] = 'X'
	second, err := m.QueueOpening(replay.Presentation{Pack: character.KindKnight, Type: "meter", Duration: 0.25, Data: json.RawMessage(`[1,2,3]`)})
	if err != nil {
		t.Fatal(err)
	}
	m.Start()
	defer m.End()
	r := m.Recording()
	if len(r.Presentations) != 2 || r.Presentations[0].Type != "banner" || string(r.Presentations[0].Data) != `{"message":"hello"}` || string(r.Presentations[1].Data) != `[1,2,3]` {
		t.Fatalf("recording=%+v", r)
	}
	var live struct{ Presentations []replay.Presentation }
	if err = json.Unmarshal(m.SnapshotJSON(), &live); err != nil {
		t.Fatal(err)
	}
	if len(live.Presentations) != 2 || live.Presentations[1].ID != second {
		t.Fatal("live and export have different presentations")
	}
	m.FinishOpening(first)
	if !m.holdingOpening() {
		t.Fatal("early segment receipt skipped the queue")
	}
	m.FinishOpening(second)
	if m.holdingOpening() {
		t.Fatal("last receipt did not release match")
	}
	if _, err = m.QueueOpening(replay.Presentation{}); err == nil {
		t.Fatal("accepted a mid-match presentation")
	}
	m.End()
	third, err := m.QueueOpening(replay.Presentation{Pack: character.KindKnight, Type: "banner", Duration: 1, Data: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	if third == first || third == second {
		t.Fatal("presentation IDs reused across matches")
	}
	m.Start()
	m.FinishOpening(second)
	if !m.holdingOpening() {
		t.Fatal("stale receipt skipped next match")
	}
	m.FinishOpening(third)
	if len(r.Presentations) != 2 || r.Presentations[0].ID != first {
		t.Fatal("new match mutated archived presentations")
	}
}

func TestQueueOpeningRejectsInvalidDescriptors(t *testing.T) {
	m := NewMatchSeeded(1)
	for _, p := range []replay.Presentation{
		{Pack: "missing", Type: "text", Duration: 1, Data: json.RawMessage(`{}`)},
		{Pack: character.KindKnight, Duration: 1, Data: json.RawMessage(`{}`)},
		{Pack: character.KindKnight, Type: "text", Duration: -1, Data: json.RawMessage(`{}`)},
		{Pack: character.KindKnight, Type: "text", Duration: 1, Data: json.RawMessage(`invalid`)},
	} {
		if _, err := m.QueueOpening(p); err == nil {
			t.Fatalf("accepted %+v", p)
		}
	}
	if len(m.pendingOpening) != 0 {
		t.Fatal("invalid presentation left pending data")
	}
}
