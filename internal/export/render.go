package export

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type chromeProc struct {
	cmd       *exec.Cmd
	page      *cdpConn
	browserWS string
	port      int
	profile   string
	log       *os.File
}

func (m *Manager) run(j *Job) {
	defer j.cancel()
	pageURL := m.baseURL + "/?export=" + j.ID
	if err := m.render(j, pageURL); err != nil {
		j.fail(err)
		return
	}
}

func (m *Manager) render(j *Job, pageURL string) error {
	ch, err := launchChrome(j.ctx, m.chrome, j.dir)
	if err != nil {
		return err
	}
	defer ch.close()

	if err := ch.page.setViewport(j.ctx, 1080, 1920); err != nil {
		return err
	}
	if err := ch.page.call(j.ctx, "Page.enable", nil, nil); err != nil {
		return err
	}
	if err := ch.page.call(j.ctx, "Runtime.enable", nil, nil); err != nil {
		return err
	}
	j.setPhase("preparing")
	if err := ch.page.navigate(j.ctx, pageURL); err != nil {
		return err
	}
	if err := ch.page.waitFor(j.ctx, "window.__exportReady === true", 45*time.Second); err != nil {
		msg, _ := ch.page.evalString(j.ctx, "String(window.__exportError || '')")
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}

	video := filepath.Join(j.dir, "video.mp4")
	ff, err := startFFmpeg(j.ctx, m.ffmpeg, video)
	if err != nil {
		return err
	}
	j.setPhase("rendering")
	total := j.total
	for i := 0; i < total; i++ {
		if j.cancelled() {
			ff.kill()
			return j.ctx.Err()
		}
		if _, err := ch.page.eval(j.ctx, fmt.Sprintf("window.__exportStep(%d)", i)); err != nil {
			ff.kill()
			return fmt.Errorf("第 %d 帧: %w", i, err)
		}
		png, err := ch.page.screenshot(j.ctx)
		if err != nil {
			ff.kill()
			return fmt.Errorf("第 %d 帧截图: %w", i, err)
		}
		if err := ff.write(png); err != nil {
			ff.kill()
			return fmt.Errorf("第 %d 帧写编码器: %w", i, err)
		}
		j.setDone(i + 1)
	}
	if err := ff.close(); err != nil {
		return err
	}

	audio, err := ch.page.evalString(j.ctx, "window.__exportAudio()")
	if err != nil {
		return fmt.Errorf("取音轨事件: %w", err)
	}
	if err := os.WriteFile(filepath.Join(j.dir, "audio.json"), []byte(audio), 0o644); err != nil {
		return err
	}

	j.setPhase("muxing")
	out := filepath.Join(j.dir, "out.mp4")
	mixed := filepath.Join(j.dir, "mix.wav")
	if err := mixAudio(j.ctx, m.ffmpeg, audio, video, mixed, j.total); err != nil {
		return fmt.Errorf("混音: %w", err)
	}
	if err := mux(j.ctx, m.ffmpeg, video, mixed, out); err != nil {
		return fmt.Errorf("封装: %w", err)
	}
	j.mu.Lock()
	j.file = out
	j.phase = "done"
	j.mu.Unlock()
	return nil
}

func launchChrome(ctx context.Context, bin, jobDir string) (*chromeProc, error) {
	port, err := freePort()
	if err != nil {
		return nil, err
	}
	profile, err := os.MkdirTemp("", "xqdj-chrome-*")
	if err != nil {
		return nil, err
	}
	logFile, err := os.Create(filepath.Join(jobDir, "chrome.log"))
	if err != nil {
		os.RemoveAll(profile)
		return nil, err
	}
	args := []string{
		"--headless=new",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--hide-scrollbars",
		"--mute-audio",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-timer-throttling",
		"--disable-renderer-backgrounding",
		"--disable-backgrounding-occluded-windows",
		"--force-device-scale-factor=1",
		"--window-size=1080,1920",
		"--user-data-dir=" + profile,
		fmt.Sprintf("--remote-debugging-port=%d", port),
		"about:blank",
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		logFile.Close()
		os.RemoveAll(profile)
		return nil, err
	}
	ch := &chromeProc{cmd: cmd, port: port, profile: profile, log: logFile}
	if err := waitDevTools(ctx, port); err != nil {
		ch.close()
		return nil, err
	}
	browserWS, pageWS, err := devtoolsTargets(ctx, port)
	if err != nil {
		ch.close()
		return nil, err
	}
	ch.browserWS = browserWS
	page, err := dialCDP(pageWS)
	if err != nil {
		ch.close()
		return nil, err
	}
	ch.page = page
	return ch, nil
}

func (c *chromeProc) close() {
	if c == nil {
		return
	}
	if c.page != nil {
		c.page.close()
	}
	if c.browserWS != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if b, err := dialCDP(c.browserWS); err == nil {
			_ = b.call(ctx, "Browser.close", nil, nil)
			b.close()
		}
		cancel()
	}
	done := make(chan struct{})
	go func() {
		if c.cmd != nil {
			_ = c.cmd.Wait()
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		if c.cmd != nil && c.cmd.Process != nil {
			_ = c.cmd.Process.Kill()
		}
		<-done
	}
	if c.log != nil {
		_ = c.log.Close()
	}
	if c.profile != "" {
		_ = os.RemoveAll(c.profile)
	}
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func waitDevTools(ctx context.Context, port int) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/json/version", port)
	deadline := time.Now().Add(20 * time.Second)
	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if resp, err := http.DefaultClient.Do(req); err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("Chrome 调试端口没起来")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func devtoolsTargets(ctx context.Context, port int) (browserWS, pageWS string, err error) {
	var version struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := getJSON(ctx, fmt.Sprintf("http://127.0.0.1:%d/json/version", port), &version); err != nil {
		return "", "", err
	}
	browserWS = version.WebSocketDebuggerURL
	var list []struct {
		Type                 string `json:"type"`
		URL                  string `json:"url"`
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := getJSON(ctx, fmt.Sprintf("http://127.0.0.1:%d/json/list", port), &list); err != nil {
		return "", "", err
	}
	for _, t := range list {
		if t.Type == "page" && t.WebSocketDebuggerURL != "" {
			pageWS = t.WebSocketDebuggerURL
			break
		}
	}
	if pageWS == "" {
		var page struct {
			WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("http://127.0.0.1:%d/json/new?about:blank", port), nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", "", err
		}
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			return "", "", err
		}
		pageWS = page.WebSocketDebuggerURL
	}
	if pageWS == "" {
		return "", "", fmt.Errorf("Chrome 没有可用的页面")
	}
	return browserWS, pageWS, nil
}

func getJSON(ctx context.Context, url string, out any) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

type ffmpegPipe struct {
	cmd    *exec.Cmd
	in     io.WriteCloser
	errBuf *bytes.Buffer
}

func startFFmpeg(ctx context.Context, bin, out string) (*ffmpegPipe, error) {
	args := []string{
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "image2pipe", "-vcodec", "png", "-framerate", "60", "-i", "-",
		"-an", "-c:v", "libx264", "-preset", "veryfast", "-crf", "18",
		"-pix_fmt", "yuv420p", "-r", "60", out,
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &ffmpegPipe{cmd: cmd, in: in, errBuf: &errBuf}, nil
}

func (f *ffmpegPipe) write(p []byte) error {
	for len(p) > 0 {
		n, err := f.in.Write(p)
		if err != nil {
			return err
		}
		p = p[n:]
	}
	return nil
}

func (f *ffmpegPipe) close() error {
	_ = f.in.Close()
	if err := f.cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg: %w: %s", err, strings.TrimSpace(f.errBuf.String()))
	}
	return nil
}

func (f *ffmpegPipe) kill() {
	_ = f.in.Close()
	if f.cmd.Process != nil {
		_ = f.cmd.Process.Kill()
	}
	_ = f.cmd.Wait()
}

func mux(ctx context.Context, bin, video, audio, out string) error {
	args := []string{
		"-hide_banner", "-loglevel", "error", "-y",
		"-i", video, "-i", audio,
		"-c:v", "copy", "-c:a", "aac", "-b:a", "192k", "-shortest", out,
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(errBuf.String()))
	}
	return nil
}
