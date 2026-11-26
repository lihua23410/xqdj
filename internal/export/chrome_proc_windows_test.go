//go:build windows

package export

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	switch os.Getenv("XQDJ_PROC_HELPER") {
	case "parent":
		runProcParent()
		os.Exit(0)
	case "child":
		runProcChild()
		os.Exit(0)
	default:
		os.Exit(m.Run())
	}
}

func runProcParent() {
	lock := os.Getenv("XQDJ_LOCK_FILE")
	child := exec.Command(os.Args[0])
	child.Env = helperEnv("child", lock)
	out, err := child.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "stdout: %v\n", err)
		os.Exit(1)
	}
	// 子进程不能继承测试管道，否则父进程死后 Wait 会一直等这个写端。
	dontInherit(os.Stdout)
	dontInherit(os.Stderr)
	if err := child.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "start child: %v\n", err)
		os.Exit(1)
	}
	buf := make([]byte, 32)
	n, err := io.ReadAtLeast(out, buf, len("child"))
	if err != nil || !bytes.Contains(buf[:n], []byte("child")) {
		fmt.Fprintf(os.Stderr, "child not ready: %v %q\n", err, buf[:n])
		os.Exit(1)
	}
	fmt.Printf("ready %d\n", child.Process.Pid)
	os.Stdout.Sync()
	go io.Copy(io.Discard, out)
	select {}
}

func runProcChild() {
	f, err := os.OpenFile(os.Getenv("XQDJ_LOCK_FILE"), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lock: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("child")
	os.Stdout.Sync()
	select {}
	f.Close()
}

func dontInherit(f *os.File) {
	if f == nil {
		return
	}
	_ = syscall.SetHandleInformation(syscall.Handle(f.Fd()), syscall.HANDLE_FLAG_INHERIT, 0)
}

func helperEnv(role, lock string) []string {
	env := make([]string, 0, len(os.Environ())+2)
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "XQDJ_PROC_HELPER=") || strings.HasPrefix(e, "XQDJ_LOCK_FILE=") {
			continue
		}
		env = append(env, e)
	}
	return append(env, "XQDJ_PROC_HELPER="+role, "XQDJ_LOCK_FILE="+lock)
}

type safeBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *safeBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *safeBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func waitHelper(t *testing.T, buf *safeBuf) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, line := range strings.Split(buf.String(), "\n") {
			var pid int
			if _, err := fmt.Sscanf(strings.TrimSpace(line), "ready %d", &pid); err == nil && pid > 0 {
				return pid
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("helper output: %q", buf.String())
	return 0
}

func startLockTree(t *testing.T) (cmd *exec.Cmd, child int, lock string) {
	t.Helper()
	dir := t.TempDir()
	lock = filepath.Join(dir, "lock")
	cmd = exec.Command(os.Args[0])
	cmd.Env = helperEnv("parent", lock)
	buf := &safeBuf{}
	cmd.Stdout = buf
	cmd.Stderr = buf
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			finishChromeTree(cmd.Process.Pid, true)
		}
		_ = cmd.Wait()
	})
	child = waitHelper(t, buf)
	t.Cleanup(func() { finishChromeTree(child, true) })
	return cmd, child, lock
}

func waitDead(pid int) bool {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return true
		}
		time.Sleep(30 * time.Millisecond)
	}
	return !processAlive(pid)
}

func removeRetry(path string) error {
	var err error
	for i := 0; i < 30; i++ {
		err = os.Remove(path)
		if err == nil || os.IsNotExist(err) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return err
}

func TestParentKillLeavesChildHoldingFile(t *testing.T) {
	cmd, child, lock := startLockTree(t)
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if !waitDead(cmd.Process.Pid) {
		t.Fatal("parent still alive")
	}
	if !processAlive(child) {
		t.Fatal("killing the parent also killed the child")
	}
	if err := os.Remove(lock); err == nil {
		t.Fatal("orphan child did not keep the file lock")
	}
	finishChromeTree(cmd.Process.Pid, false)
	if !waitDead(child) {
		t.Fatalf("orphan %d still alive", child)
	}
	if err := removeRetry(lock); err != nil {
		t.Fatal(err)
	}
}

func TestFinishChromeTreeKillsLiveChildren(t *testing.T) {
	cmd, child, lock := startLockTree(t)
	finishChromeTree(cmd.Process.Pid, true)
	if !waitDead(cmd.Process.Pid) || !waitDead(child) {
		t.Fatalf("tree still alive parent=%v child=%v", processAlive(cmd.Process.Pid), processAlive(child))
	}
	if err := removeRetry(lock); err != nil {
		t.Fatal(err)
	}
}

func processesHolding(profile string) []int {
	quoted := strings.ReplaceAll(profile, "'", "''")
	script := fmt.Sprintf(`Get-CimInstance Win32_Process | Where-Object { $_.Name -match 'chrome|msedge|chromium' -and $_.CommandLine -and $_.CommandLine.Contains('%s') } | ForEach-Object { $_.ProcessId }`, quoted)
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return nil
	}
	var pids []int
	for _, f := range strings.Fields(string(out)) {
		n, err := strconv.Atoi(f)
		if err == nil && n > 0 {
			pids = append(pids, n)
		}
	}
	return pids
}
