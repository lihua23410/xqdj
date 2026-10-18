package 非因果

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestPresentationHooks(t *testing.T) {
	css, err := os.ReadFile("fx/noncausal.css")
	if err != nil {
		t.Fatal(err)
	}
	sheet := string(css)
	if !strings.Contains(sheet, "rgba(223, 246, 255, 0.3)") {
		t.Fatal("residual opacity should sit in the 25%–35% band")
	}
	js, err := os.ReadFile("fx/noncausal.js")
	if err != nil {
		t.Fatal(err)
	}
	text := string(js)
	for _, needle := range []string{
		"打的是对面那条死掉的世界线",
		"GHOST_OPACITY = 0.3",
		"REWIND_MAX_SEC = 3",
		"lookFX.residual",
		"lookFX.lock",
		"spectrum-bin",
		"lock-block",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("missing %s", needle)
		}
	}
	if strings.Contains(text, "requestAnimationFrame") || strings.Contains(text, "setInterval") {
		t.Fatal("histogram must not scan")
	}
	shot, err := os.ReadFile("fx/shot.js")
	if err != nil {
		t.Fatal(err)
	}
	shotText := string(shot)
	for _, name := range []string{
		`"spectrum"`, `"spectrum-bin"`, `"doom-at"`, `"probe-hit"`,
		`"lock-shot"`, `"lock-armed"`, `"lock-block"`, `"ghost-hit"`,
		`"ghost-down"`, `"doom"`, "因果被截",
	} {
		if !strings.Contains(shotText, name) {
			t.Fatalf("shot hook %s", name)
		}
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed")
	}
	cmd := exec.Command("node", "--test", "present_test.mjs")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("present tests: %v\n%s", err, out)
	}
}
