package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestProgressBarRewritesRate(t *testing.T) {
	var buf bytes.Buffer
	bar := newProgress(&buf, 10)
	// 模拟每场 10ms：第一场立刻打点，之后按 5 场/秒节流。
	for i := 0; i < 10; i++ {
		bar.step()
		time.Sleep(10 * time.Millisecond)
	}
	bar.finish()
	out := buf.String()
	if !strings.Contains(out, "10/10") {
		t.Fatalf("no completion count in %q", out)
	}
	if !strings.Contains(out, "100.0%") {
		t.Fatalf("no full percentage in %q", out)
	}
	lines := strings.Count(out, "\n")
	if lines > 21 {
		t.Fatalf("progress flooded the log: %d lines", lines)
	}
}

func TestProgressBarShowsRemaining(t *testing.T) {
	var buf bytes.Buffer
	bar := newProgress(&buf, 100)
	bar.step()
	time.Sleep(3 * time.Millisecond)
	bar.step()
	out := buf.String()
	if !strings.Contains(out, "1/100") {
		t.Fatalf("first step not reported: %q", out)
	}
	if !strings.Contains(out, "剩") {
		t.Fatalf("no estimate line: %q", out)
	}
}

func TestHumanDur(t *testing.T) {
	cases := map[time.Duration]string{
		900 * time.Millisecond:                      "900ms",
		42 * time.Second:                            "42s",
		90 * time.Second:                            "1m30s",
		3*time.Hour + 4*time.Minute + 5*time.Second: "3h04m05s",
	}
	for in, want := range cases {
		if got := humanDur(in); got != want {
			t.Fatalf("humanDur(%v) = %q, want %q", in, got, want)
		}
	}
}
