package export

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"xqdj/internal/replay"
)

func TestExportTimelineWithOptionalProbe(t *testing.T) {
	for _, tc := range []struct {
		name  string
		film  *replay.IntroFilm
		extra int
	}{
		{"ordinary", nil, 0},
		{"rewind", &replay.IntroFilm{Death: 2.25, Frames: []replay.IntroFrame{{T: 0}}}, 74},
		{"no death", &replay.IntroFilm{Death: -1, Frames: []replay.IntroFrame{{}}}, 0},
		{"empty", &replay.IntroFilm{Death: 1}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tl := newTimeline(replay.Recording{Intro: tc.film, Frames: make([]json.RawMessage, 120)})
			if tl.ProbeFrames != tc.extra || tl.TotalFrames != 300+tc.extra+60+120+300 {
				t.Fatalf("timeline=%+v", tl)
			}
		})
	}
}

func TestExportMetadataIsSeparateFromBattleFrames(t *testing.T) {
	j := &Job{ID: "test", Initial: json.RawMessage(`{"time":0}`), Intro: &replay.IntroFilm{Death: 1, Frames: []replay.IntroFrame{{T: 0}}}, Frames: []json.RawMessage{json.RawMessage(`{"time":1}`)}}
	j.Timeline = newTimeline(replay.Recording{Intro: j.Intro, Frames: j.Frames})
	m := &Manager{jobs: map[string]*Job{"test": j}}
	mux := http.NewServeMux()
	m.Handle(mux)
	meta := httptest.NewRecorder()
	mux.ServeHTTP(meta, httptest.NewRequest("GET", "/api/export/test/meta", nil))
	var got struct {
		Initial  json.RawMessage
		Intro    *replay.IntroFilm
		Timeline timeline
	}
	if err := json.Unmarshal(meta.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Intro == nil || got.Timeline.ProbeFrames != 66 || string(got.Initial) != string(j.Initial) {
		t.Fatalf("metadata=%s", meta.Body.String())
	}
	frames := httptest.NewRecorder()
	mux.ServeHTTP(frames, httptest.NewRequest("GET", "/api/export/test/frames", nil))
	if frames.Body.String() != string(j.Frames[0])+"\n" {
		t.Fatalf("battle frames contain extra data: %s", frames.Body.String())
	}
}
