// Package export 把一场对局的观众快照逐帧渲染成 MP4。
// 页面仍用 internal/web 的 index.html，靠假时钟推进；这里只负责起浏览器、截图、编码。
package export

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

const (
	holdStart = 60
	holdEnd   = 120
	fps       = 60
)

var errNoFrames = errors.New("这一场还没有可导出的帧")

type Manager struct {
	baseURL string
	frames  func() []json.RawMessage
	chrome  string
	ffmpeg  string

	mu   sync.Mutex
	jobs map[string]*Job
}

type Job struct {
	mu     sync.Mutex
	ID     string
	Frames []json.RawMessage
	phase  string
	done   int
	total  int
	err    string
	dir    string
	file   string
	cancel context.CancelFunc
	ctx    context.Context
}

func NewManager(baseURL string, frames func() []json.RawMessage) *Manager {
	return &Manager{
		baseURL: baseURL,
		frames:  frames,
		chrome:  findChrome(),
		ffmpeg:  findFFmpeg(),
		jobs:    make(map[string]*Job),
	}
}

func findChrome() string {
	if p := os.Getenv("XQDJ_CHROME"); p != "" {
		return p
	}
	for _, name := range []string{"google-chrome-stable", "google-chrome", "chromium", "chromium-browser"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

func findFFmpeg() string {
	if p := os.Getenv("XQDJ_FFMPEG"); p != "" {
		return p
	}
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p
	}
	return ""
}

func (m *Manager) Handle(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/export", m.handleStart)
	mux.HandleFunc("GET /api/export/{id}", m.handleStatus)
	mux.HandleFunc("GET /api/export/{id}/frames", m.handleFrames)
	mux.HandleFunc("POST /api/export/{id}/cancel", m.handleCancel)
	mux.HandleFunc("GET /api/export/{id}/file", m.handleFile)
}

func (m *Manager) handleStart(w http.ResponseWriter, r *http.Request) {
	if m.chrome == "" || m.ffmpeg == "" {
		http.Error(w, "导出需要本机 Chrome 和 ffmpeg", http.StatusServiceUnavailable)
		return
	}
	frames := m.frames()
	if len(frames) < 2 {
		http.Error(w, errNoFrames.Error(), http.StatusConflict)
		return
	}
	id := newID()
	dir := filepath.Join(os.TempDir(), "xqdj-export-"+id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	j := &Job{
		ID:     id,
		Frames: frames,
		phase:  "preparing",
		total:  holdStart + len(frames) + holdEnd,
		dir:    dir,
		ctx:    ctx,
		cancel: cancel,
	}
	m.mu.Lock()
	m.jobs[id] = j
	m.mu.Unlock()
	go m.run(j)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "total": j.total})
}

func (m *Manager) job(r *http.Request) *Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.jobs[r.PathValue("id")]
}

func (m *Manager) handleStatus(w http.ResponseWriter, r *http.Request) {
	j := m.job(r)
	if j == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(j.snapshot())
}

func (m *Manager) handleFrames(w http.ResponseWriter, r *http.Request) {
	j := m.job(r)
	if j == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	for _, f := range j.Frames {
		if _, err := w.Write(f); err != nil {
			return
		}
		if _, err := w.Write([]byte{'\n'}); err != nil {
			return
		}
	}
}

func (m *Manager) handleCancel(w http.ResponseWriter, r *http.Request) {
	j := m.job(r)
	if j == nil {
		http.NotFound(w, r)
		return
	}
	j.cancel()
	j.mu.Lock()
	if j.phase != "done" {
		j.phase = "cancelled"
	}
	j.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (m *Manager) handleFile(w http.ResponseWriter, r *http.Request) {
	j := m.job(r)
	if j == nil {
		http.NotFound(w, r)
		return
	}
	j.mu.Lock()
	phase, file := j.phase, j.file
	j.mu.Unlock()
	if phase != "done" || file == "" {
		http.Error(w, "还没导出完", http.StatusConflict)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="xqdj-%s.mp4"`, j.ID))
	http.ServeFile(w, r, file)
}

func (j *Job) snapshot() map[string]any {
	j.mu.Lock()
	defer j.mu.Unlock()
	out := map[string]any{
		"id":    j.ID,
		"phase": j.phase,
		"done":  j.done,
		"total": j.total,
	}
	if j.err != "" {
		out["error"] = j.err
	}
	if j.phase == "done" {
		out["download"] = "/api/export/" + j.ID + "/file"
	}
	return out
}

func (j *Job) setPhase(phase string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.phase == "cancelled" {
		return
	}
	j.phase = phase
}

func (j *Job) setDone(n int) {
	j.mu.Lock()
	j.done = n
	j.mu.Unlock()
}

func (j *Job) fail(err error) {
	if err == nil {
		return
	}
	if j.ctx.Err() != nil {
		return
	}
	j.mu.Lock()
	if j.phase == "done" || j.phase == "cancelled" {
		j.mu.Unlock()
		return
	}
	j.phase = "error"
	j.err = err.Error()
	j.mu.Unlock()
}

func (j *Job) cancelled() bool {
	return j.ctx.Err() != nil
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
