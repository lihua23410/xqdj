package export

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"xqdj/internal/replay"
)

func TestExportTimelineWithGenericPresentations(t *testing.T) {
	for _, durations := range [][]float64{nil, {1.225}, {0.12, 1.1}} {
		var list []replay.Presentation
		want := 0
		for i, d := range durations {
			list = append(list, replay.Presentation{Pack: "test", Type: "banner", Duration: d, Data: json.RawMessage(`{"message":"hello"}`)})
			if i == 0 && d == 1.225 {
				want += 74
			} else if d == 0.12 {
				want += 8
			} else {
				want += 66
			}
		}
		tl := newTimeline(replay.Recording{Presentations: list, Frames: make([]json.RawMessage, 120)})
		if tl.OpeningFrames != want || tl.TotalFrames != 300+want+60+120+300 || len(tl.PresentationFrames) != len(list) {
			t.Fatalf("timeline=%+v", tl)
		}
	}
}

func TestExportMetadataIsSeparateFromBattleFrames(t *testing.T) {
	j := &Job{ID: "test", Initial: json.RawMessage(`{"time":0}`), Presentations: []replay.Presentation{{ID: "1", Pack: "test", Type: "banner", Duration: 1.1, Data: json.RawMessage(`{"message":"hello"}`)}}, Frames: []json.RawMessage{json.RawMessage(`{"time":1}`)}}
	j.Timeline = newTimeline(replay.Recording{Presentations: j.Presentations, Frames: j.Frames})
	m := &Manager{jobs: map[string]*Job{"test": j}}
	mux := http.NewServeMux()
	m.Handle(mux)
	meta := httptest.NewRecorder()
	mux.ServeHTTP(meta, httptest.NewRequest("GET", "/api/export/test/meta", nil))
	var got struct {
		Initial       json.RawMessage
		Presentations []replay.Presentation
		Timeline      timeline
	}
	if err := json.Unmarshal(meta.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Presentations) != 1 || got.Timeline.OpeningFrames != 66 || string(got.Initial) != string(j.Initial) || string(got.Presentations[0].Data) != `{"message":"hello"}` {
		t.Fatalf("metadata=%s", meta.Body.String())
	}
	frames := httptest.NewRecorder()
	mux.ServeHTTP(frames, httptest.NewRequest("GET", "/api/export/test/frames", nil))
	if frames.Body.String() != string(j.Frames[0])+"\n" {
		t.Fatalf("battle frames contain extra data: %s", frames.Body.String())
	}
}
