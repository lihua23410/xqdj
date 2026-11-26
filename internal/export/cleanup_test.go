package export

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"xqdj/internal/replay"
)

func TestExportDownloadDropsFramesAndTempDir(t *testing.T) {
	dir := makeExportDir(t)
	out := filepath.Join(dir, "out.mp4")
	payload := []byte("mp4-bytes")
	if err := os.WriteFile(out, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	j := &Job{
		ID:            "job1",
		Frames:        []json.RawMessage{json.RawMessage(`{"time":1}`), json.RawMessage(`{"time":2}`)},
		Initial:       json.RawMessage(`{"time":0}`),
		Presentations: []replay.Presentation{{ID: "p", Pack: "test", Type: "banner", Duration: 0.1, Data: json.RawMessage(`{}`)}},
		phase:         "done",
		dir:           dir,
		file:          out,
		dropNow:       make(chan struct{}),
	}
	m := &Manager{jobs: map[string]*Job{j.ID: j}}
	m.finish(j)

	j.mu.Lock()
	released := j.Frames == nil && j.Initial == nil && j.Presentations == nil
	j.mu.Unlock()
	if !released {
		t.Fatal("frame payload still retained after render")
	}
	for _, name := range []string{"video.mp4", "mix.wav", "audio.json", "chrome.log"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("%s still in temp dir", name)
		}
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatal("final mp4 removed before download")
	}

	mux := http.NewServeMux()
	m.Handle(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/export/job1/file", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != string(payload) {
		t.Fatalf("download %d %q", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("export temp dir remains after download")
	}
	m.mu.Lock()
	_, ok := m.jobs[j.ID]
	m.mu.Unlock()
	if ok {
		t.Fatal("job still holds export state")
	}
}

func TestExportFailureDeletesTempDirAndForgetsJob(t *testing.T) {
	prev := exportStatusKeep
	exportStatusKeep = 200 * time.Millisecond
	t.Cleanup(func() { exportStatusKeep = prev })

	dir := makeExportDir(t)
	j := &Job{
		ID:      "bad",
		Frames:  []json.RawMessage{json.RawMessage(`{"time":1}`)},
		Initial: json.RawMessage(`{"time":0}`),
		phase:   "error",
		err:     "渲染失败",
		dir:     dir,
		dropNow: make(chan struct{}),
	}
	m := &Manager{jobs: map[string]*Job{j.ID: j}}
	m.finish(j)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("failed export left temp dir")
	}
	j.mu.Lock()
	released := j.Frames == nil && j.Initial == nil
	j.mu.Unlock()
	if !released {
		t.Fatal("failed export kept frame payload")
	}

	mux := http.NewServeMux()
	m.Handle(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/export/bad", nil))
	var status map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status["phase"] != "error" || status["error"] != "渲染失败" {
		t.Fatalf("status=%v", status)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		_, ok := m.jobs[j.ID]
		m.mu.Unlock()
		if !ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("failed job was not dropped")
}

func TestExportDoneWithoutDownloadExpires(t *testing.T) {
	prev := exportFileKeep
	exportFileKeep = 200 * time.Millisecond
	t.Cleanup(func() { exportFileKeep = prev })

	dir := makeExportDir(t)
	out := filepath.Join(dir, "out.mp4")
	if err := os.WriteFile(out, []byte("mp4"), 0o644); err != nil {
		t.Fatal(err)
	}
	j := &Job{
		ID:      "stale",
		Frames:  []json.RawMessage{json.RawMessage(`{"time":1}`)},
		phase:   "done",
		dir:     dir,
		file:    out,
		dropNow: make(chan struct{}),
	}
	m := &Manager{jobs: map[string]*Job{j.ID: j}}
	m.finish(j)
	if _, err := os.Stat(out); err != nil {
		t.Fatal("final mp4 removed before the keep window")
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, dirErr := os.Stat(dir)
		m.mu.Lock()
		_, ok := m.jobs[j.ID]
		m.mu.Unlock()
		if os.IsNotExist(dirErr) && !ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("undownloaded export was kept")
}

func makeExportDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "xqdj-export-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	for _, name := range []string{"video.mp4", "mix.wav", "audio.json", "chrome.log"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}
