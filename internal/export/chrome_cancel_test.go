package export

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestCancelLeavesBrowserUntilClose(t *testing.T) {
	bin := findChrome()
	if bin == "" {
		t.Skip("no chrome")
	}
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := launchChrome(ctx, bin, dir)
	if err != nil {
		t.Fatal(err)
	}
	profile := ch.profile
	pid := ch.cmd.Process.Pid
	t.Cleanup(func() {
		ch.close()
		if treeAlive(pid) {
			finishChromeTree(pid, true)
		}
	})

	cancel()
	time.Sleep(500 * time.Millisecond)
	if !processAlive(pid) {
		t.Fatal("cancel killed the browser before Browser.close")
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/json/version", ch.port))
	if err != nil {
		t.Fatalf("devtools gone after cancel: %v", err)
	}
	resp.Body.Close()

	ch.close()
	if _, err := os.Stat(profile); !os.IsNotExist(err) {
		t.Fatalf("profile leaked: %s (%v)", profile, err)
	}
	if treeAlive(pid) {
		t.Fatal("chrome process tree still alive")
	}
	if pids := processesHolding(profile); len(pids) > 0 {
		t.Fatalf("processes still using profile: %v", pids)
	}
}
