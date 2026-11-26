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
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
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

// 终态还要留一会儿给页面读错误；成片等下载，超时后同样收掉，避免没人来取就一直占着。
var (
	exportStatusKeep = 2 * time.Minute
	exportFileKeep   = 15 * time.Minute
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
	mu            sync.Mutex
	ID            string
	Frames        []json.RawMessage
	Initial       json.RawMessage
	Presentations []replay.Presentation
	Timeline      timeline
	phase         string
	done          int
	total         int
	err           string
	dir           string
	file          string
	cancel        context.CancelFunc
	ctx           context.Context
	dropNow       chan struct{}
	dropOnce      sync.Once
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
		ID:            id,
		Frames:        frames,
		Initial:       recording.Initial,
		Presentations: recording.Presentations,
		Timeline:      tl,
		phase:         "preparing",
		total:         tl.TotalFrames,
		dir:           dir,
		ctx:           ctx,
		cancel:        cancel,
		dropNow:       make(chan struct{}),
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
	j.mu.Lock()
	frames := j.Frames
	j.mu.Unlock()
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	for _, f := range frames {
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
	f, err := os.Open(file)
	if err != nil {
		http.Error(w, "成片不在了", http.StatusNotFound)
		return
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		http.Error(w, "成片不在了", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Content-Length", strconv.FormatInt(st.Size(), 10))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="xqdj-%s.mp4"`, j.ID))
	n, err := io.Copy(w, f)
	f.Close()
	if err != nil || n != st.Size() {
		return
	}
	m.removeJob(j)
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
	FPS                int   `json:"fps"`
	TitleFrames        int   `json:"titleFrames"`
	OpeningFrames      int   `json:"openingFrames"`
	PresentationFrames []int `json:"presentationFrames"`
	HoldStartFrames    int   `json:"holdStartFrames"`
	EndFrames          int   `json:"endFrames"`
	BattleFrames       int   `json:"battleFrames"`
	TotalFrames        int   `json:"totalFrames"`
}

func newTimeline(r replay.Recording) timeline {
	opening := 0
	segments := make([]int, len(r.Presentations))
	for i, p := range r.Presentations {
		segments[i] = int(math.Ceil(p.Duration * fps))
		opening += segments[i]
	}
	return timeline{FPS: fps, TitleFrames: introFrames, OpeningFrames: opening, PresentationFrames: segments, HoldStartFrames: holdStart, EndFrames: holdEnd, BattleFrames: len(r.Frames), TotalFrames: introFrames + opening + holdStart + len(r.Frames) + holdEnd}
}

func (m *Manager) handleMeta(w http.ResponseWriter, r *http.Request) {
	j := m.job(r)
	if j == nil {
		http.NotFound(w, r)
		return
	}
	j.mu.Lock()
	initial, presentations, tl := j.Initial, j.Presentations, j.Timeline
	j.mu.Unlock()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(struct {
		Initial       json.RawMessage       `json:"initial,omitempty"`
		Presentations []replay.Presentation `json:"presentations"`
		Timeline      timeline              `json:"timeline"`
	}{initial, presentations, tl})
}

// finish 在渲染协程退出时调用。帧数据此刻不再被导出页读取，先放开；
// 失败和取消立刻删掉临时目录，成片则保留到下载或超时。
func (m *Manager) finish(j *Job) {
	j.releasePayload()
	j.mu.Lock()
	phase, dir, file := j.phase, j.dir, j.file
	j.mu.Unlock()
	if phase != "done" {
		if dir != "" {
			_ = os.RemoveAll(dir)
			j.mu.Lock()
			if j.dir == dir {
				j.dir = ""
				j.file = ""
			}
			j.mu.Unlock()
		}
		m.armDrop(j, exportStatusKeep)
		return
	}
	if dir != "" && file != "" {
		removeIntermediates(dir, file)
	}
	m.armDrop(j, exportFileKeep)
}

func (j *Job) releasePayload() {
	j.mu.Lock()
	j.Frames = nil
	j.Initial = nil
	j.Presentations = nil
	j.mu.Unlock()
}

func removeIntermediates(dir, keep string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	base := filepath.Base(keep)
	for _, e := range entries {
		if e.Name() == base {
			continue
		}
		_ = os.RemoveAll(filepath.Join(dir, e.Name()))
	}
}

func (m *Manager) armDrop(j *Job, after time.Duration) {
	j.mu.Lock()
	if j.dropNow == nil {
		j.dropNow = make(chan struct{})
	}
	ch := j.dropNow
	j.mu.Unlock()
	go func() {
		timer := time.NewTimer(after)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ch:
		}
		m.removeJob(j)
	}()
}

func (m *Manager) removeJob(j *Job) {
	j.dropOnce.Do(func() {
		j.mu.Lock()
		if j.dropNow != nil {
			close(j.dropNow)
		}
		j.mu.Unlock()
	})
	j.mu.Lock()
	dir := j.dir
	j.dir = ""
	j.file = ""
	j.Frames = nil
	j.Initial = nil
	j.Presentations = nil
	j.mu.Unlock()
	if dir != "" {
		_ = os.RemoveAll(dir)
	}
	m.mu.Lock()
	if m.jobs[j.ID] == j {
		delete(m.jobs, j.ID)
	}
	m.mu.Unlock()
}
