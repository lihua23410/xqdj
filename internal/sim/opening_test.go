package sim

import (
	"bytes"
	"encoding/json"
	"sync"
	"testing"
	"time"
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

func TestLiveBeatHoldsOpeningAcrossStart(t *testing.T) {
	opening := replay.Presentation{Pack: character.KindKnight, Type: "banner", Duration: 30, Data: json.RawMessage(`{"message":"hold"}`)}
	for trial := 0; trial < 30; trial++ {
		m := NewMatchSeeded(uint64(trial + 1))
		m.SetSlot(0, character.KindDummy)
		m.SetSlot(1, character.KindDummy)
		if _, err := m.QueueOpening(opening); err != nil {
			t.Fatal(err)
		}
		stop := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					m.liveBeat()
				}
			}
		}()
		time.Sleep(2 * time.Millisecond)
		m.Start()
		time.Sleep(2 * time.Millisecond)
		close(stop)
		wg.Wait()

		m.mu.Lock()
		held := !m.openingUntil.IsZero() && time.Now().Before(m.openingUntil)
		tm := m.time
		frames := len(m.record)
		m.mu.Unlock()
		m.End()
		if !held || tm != 0 || frames != 0 {
			t.Fatalf("trial %d held=%v time=%v frames=%d", trial, held, tm, frames)
		}
	}
}

func TestLiveBeatRecordsOnlyAfterOpening(t *testing.T) {
	m := NewMatchSeeded(4)
	m.SetSlot(0, character.KindDummy)
	m.SetSlot(1, character.KindDummy)
	id, err := m.QueueOpening(replay.Presentation{Pack: character.KindKnight, Type: "banner", Duration: 30, Data: json.RawMessage(`{"message":"hold"}`)})
	if err != nil {
		t.Fatal(err)
	}
	m.Start()
	defer m.End()
	snap := m.liveBeat()
	var live struct {
		Time          float64
		Presentations []replay.Presentation
	}
	if err = json.Unmarshal(snap, &live); err != nil {
		t.Fatal(err)
	}
	if live.Time != 0 || len(live.Presentations) != 1 || live.Presentations[0].ID != id {
		t.Fatalf("opening snap time=%v presentations=%+v", live.Time, live.Presentations)
	}
	if m.time != 0 || len(m.RecordFrames()) != 0 {
		t.Fatal("beat during opening recorded a frame")
	}
	m.FinishOpening(id)
	m.liveBeat()
	frames := m.RecordFrames()
	if m.time != DT || len(frames) != 1 {
		t.Fatalf("first beat after opening time=%v frames=%d", m.time, len(frames))
	}
	if bytes.Contains(frames[0], []byte(`"presentations"`)) {
		t.Fatal("opening leaked into the first battle frame")
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
