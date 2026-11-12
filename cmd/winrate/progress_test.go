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

func TestProgressBarShowsFocusWinRate(t *testing.T) {
	var buf bytes.Buffer
	bar := newProgress(&buf, 8)
	bar.interval = 0
	bar.watch("时计")

	bar.note("时计", "雷达", "平局")
	bar.report("draw")
	if !strings.Contains(buf.String(), "时计 胜率 —") {
		t.Fatalf("undecided rate missing: %q", buf.String())
	}

	buf.Reset()
	bar.note("时计", "雷达", "时计")
	bar.report("win")
	if !strings.Contains(buf.String(), "时计 胜率 100.0%") {
		t.Fatalf("win not counted: %q", buf.String())
	}

	buf.Reset()
	bar.note("雷达", "时计", "雷达")
	bar.note("甲", "乙", "甲")
	bar.report("lose")
	if !strings.Contains(buf.String(), "时计 胜率 50.0%") {
		t.Fatalf("loss or outsider mishandled: %q", buf.String())
	}

	buf.Reset()
	bar.note("时计", "雷达", "时计")
	bar.report("win again")
	if !strings.Contains(buf.String(), "时计 胜率 66.7%") {
		t.Fatalf("rate did not refresh: %q", buf.String())
	}
}

func TestProgressBarOmitsRateWithoutFocus(t *testing.T) {
	var buf bytes.Buffer
	bar := newProgress(&buf, 2)
	bar.note("时计", "雷达", "时计")
	bar.report("x")
	if strings.Contains(buf.String(), "胜率") {
		t.Fatalf("full run showed a fighter rate: %q", buf.String())
	}
}

func TestProgressBarShrinksForFocus(t *testing.T) {
	var buf bytes.Buffer
	bar := newProgress(&buf, 10)
	bar.cols = 80
	bar.fitWidth()
	full := bar.width
	bar.watch("粉红色迷人的pink")
	if bar.width >= full {
		t.Fatalf("width %d did not shrink from %d", bar.width, full)
	}
	if bar.width < 4 {
		t.Fatalf("width collapsed to %d", bar.width)
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
