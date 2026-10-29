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
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
	"xqdj/internal/replay"
)

const (
	// 帧数账本通过 /meta 下发，导出页不再另算一份总时长。
	introFrames = 300 // 片头：逐个出现 60 + 移到顶栏 180 + 元素浮现 60
	holdStart   = 60  // 首帧静止
	holdEnd     = 300 // 胜者定格：0.6 秒淡入 + 4.4 秒静止
	fps         = 60
)

var errNoFrames = errors.New("这一场还没有可导出的帧")

type Manager struct {
	baseURL   string
	recording func() replay.Recording
	chrome    string
	ffmpeg    string

	mu   sync.Mutex
	jobs map[string]*Job
}

type Job struct {
	mu       sync.Mutex
	ID       string
	Frames   []json.RawMessage
	Initial  json.RawMessage
	Intro    *replay.IntroFilm
	Timeline timeline
	phase    string
	done     int
	total    int
	err      string
	dir      string
	file     string
	cancel   context.CancelFunc
	ctx      context.Context
}

func NewManager(baseURL string, recording func() replay.Recording) *Manager {
	m := &Manager{
		baseURL:   baseURL,
		recording: recording,
		chrome:    findChrome(),
		ffmpeg:    findFFmpeg(),
		jobs:      make(map[string]*Job),
	}
	m.logEnv()
	return m
}

// logEnv 在启动时把选中的浏览器和 ffmpeg 打出来。导出出问题时第一眼看这行就知道
// 是环境没配好还是渲染本身有问题。
func (m *Manager) logEnv() {
	if m.chrome == "" {
		log.Printf("导出：找不到 Chromium 浏览器，用 XQDJ_CHROME 指定 chrome.exe / msedge.exe 的完整路径")
	} else {
		log.Printf("导出：浏览器 %s", m.chrome)
	}
	if m.ffmpeg == "" {
		log.Printf("导出：找不到 ffmpeg，用 XQDJ_FFMPEG 指定 ffmpeg.exe 的完整路径")
	} else {
		log.Printf("导出：ffmpeg %s", m.ffmpeg)
	}
}

// findChrome 找一个能跑无头截图的 Chromium：先看 XQDJ_CHROME，再看 PATH，
// 最后翻 Windows 的常见安装位置（本机往往既没装 Chrome，也没把 msedge 加进 PATH，
// 但 Edge 就是 Chromium，导出照样能用）。
func findChrome() string {
	if p := os.Getenv("XQDJ_CHROME"); p != "" {
		return p
	}
	for _, name := range []string{
		"google-chrome-stable", "google-chrome", "chromium", "chromium-browser",
		"chrome", "chrome.exe", "msedge", "msedge.exe",
	} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	for _, p := range browserCandidates() {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func browserCandidates() []string {
	if runtime.GOOS != "windows" {
		return nil
	}
	var out []string
	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)", "LocalAppData"} {
		root := os.Getenv(env)
		if root == "" {
			continue
		}
		out = append(out,
			filepath.Join(root, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(root, "Microsoft", "Edge", "Application", "msedge.exe"),
		)
	}
	return out
}

func findFFmpeg() string {
	if p := os.Getenv("XQDJ_FFMPEG"); p != "" {
		return p
	}
	for _, name := range []string{"ffmpeg", "ffmpeg.exe"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	for _, p := range ffmpegCandidates() {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func ffmpegCandidates() []string {
	if runtime.GOOS != "windows" {
		return nil
	}
	var out []string
	if root := os.Getenv("ProgramData"); root != "" {
		out = append(out, filepath.Join(root, "chocolatey", "bin", "ffmpeg.exe"))
	}
	if root := os.Getenv("LocalAppData"); root != "" {
		out = append(out, filepath.Join(root, "Microsoft", "WinGet", "Links", "ffmpeg.exe"))
	}
	if root := os.Getenv("ProgramFiles"); root != "" {
		out = append(out, filepath.Join(root, "ffmpeg", "bin", "ffmpeg.exe"))
	}
	out = append(out, `C:\ffmpeg\bin\ffmpeg.exe`)
	return out
}

func (m *Manager) Handle(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/export", m.handleStart)
	mux.HandleFunc("GET /api/export/{id}", m.handleStatus)
	mux.HandleFunc("GET /api/export/{id}/frames", m.handleFrames)
	mux.HandleFunc("GET /api/export/{id}/meta", m.handleMeta)
	mux.HandleFunc("POST /api/export/{id}/cancel", m.handleCancel)
	mux.HandleFunc("GET /api/export/{id}/file", m.handleFile)
}

func (m *Manager) handleStart(w http.ResponseWriter, r *http.Request) {
	if m.chrome == "" || m.ffmpeg == "" {
		http.Error(w, "导出需要本机 Chrome 和 ffmpeg", http.StatusServiceUnavailable)
		return
	}
	recording := m.recording()
	frames := recording.Frames
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
	tl := newTimeline(recording)
	j := &Job{
		ID:       id,
		Frames:   frames,
		Initial:  recording.Initial,
		Intro:    recording.Intro,
		Timeline: tl,
		phase:    "preparing",
		total:    tl.TotalFrames,
		dir:      dir,
		ctx:      ctx,
		cancel:   cancel,
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

// timeline 是服务端与导出页面共用的帧数账本。
type timeline struct {
	FPS             int `json:"fps"`
	TitleFrames     int `json:"titleFrames"`
	ProbeFrames     int `json:"probeFrames"`
	HoldStartFrames int `json:"holdStartFrames"`
	EndFrames       int `json:"endFrames"`
	BattleFrames    int `json:"battleFrames"`
	TotalFrames     int `json:"totalFrames"`
}

func newTimeline(r replay.Recording) timeline {
	probe := int(math.Ceil(r.Intro.Duration() * fps))
	return timeline{FPS: fps, TitleFrames: introFrames, ProbeFrames: probe, HoldStartFrames: holdStart, EndFrames: holdEnd, BattleFrames: len(r.Frames), TotalFrames: introFrames + probe + holdStart + len(r.Frames) + holdEnd}
}

func (m *Manager) handleMeta(w http.ResponseWriter, r *http.Request) {
	j := m.job(r)
	if j == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(struct {
		Initial  json.RawMessage   `json:"initial,omitempty"`
		Intro    *replay.IntroFilm `json:"intro,omitempty"`
		Timeline timeline          `json:"timeline"`
	}{j.Initial, j.Intro, j.Timeline})
}
