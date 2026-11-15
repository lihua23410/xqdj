package hff

import (
	"math"
	"strings"
	"testing"

	弓箭 "xqdj/character/archer"
	"xqdj/internal/unit"
	六边形 "xqdj/map/hex"
)

func TestParseSoloExcludesDraws(t *testing.T) {
	text := strings.Join([]string{
		"## 单球胜率",
		"| 排名 | t度 | 战斗机 | 积分 | 场次 | 胜 | 负 | 平 | 胜率 | 胜平率 |",
		"| ---: | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |",
		"| 1 | t0上 | 无下限术士 | 7809 | 6200 | 3894 | 2285 | 21 | 63.0% | 63.1% |",
		"| 2 | t0上 | 弓箭 | 1 | 2 | 1 | 1 | 0 | 50.0% | 50.0% |",
		"| 57 | t2下 | 非因果 | 5141 | 6200 | 2550 | 3609 | 41 | 41.4% | 41.8% |",
		"| 99 | t2下 | 大肥鱼(幻方量化) | 1 | 1 | 1 | 1 | 0 | 50.0% | 50.0% |",
		"## 一对一胜率",
		"| 1 | t0上 | 泡泡球 | 1 | 1 | 9 | 1 | 0 | 90.0% | 90.0% |",
	}, "\n")
	got := ParseSolo(text)
	want := 3894.0 / (3894.0 + 2285.0)
	if math.Abs(got["无下限术士"]-want) > 1e-12 {
		t.Fatalf("rate %v", got["无下限术士"])
	}
	if _, ok := got["泡泡球"]; ok {
		t.Fatal("parsed past the solo table")
	}
	if math.Abs(got["弓箭"]-0.5) > 1e-12 {
		t.Fatalf("archer %v", got["弓箭"])
	}
	SetRatesForTest(got)
	defer SetRatesForTest(nil)
	var sawArcher bool
	for _, s := range Pool() {
		if s.Name == KindHFF || s.Name == "非因果" {
			t.Fatalf("pool has %s", s.Name)
		}
		if s.Name == 弓箭.KindArcher {
			sawArcher = true
		}
	}
	if !sawArcher {
		t.Fatal("archer missing from the pool")
	}
}

func TestRepoSoloTable(t *testing.T) {
	got := ParseSolo(string(readWinrate()))
	want := 3894.0 / (3894.0 + 2285.0)
	if math.Abs(got["无下限术士"]-want) > 1e-9 {
		t.Fatalf("无下限术士 %v", got["无下限术士"])
	}
	if _, ok := got["非因果"]; !ok {
		t.Fatal("非因果 should stay in the table")
	}
	if _, ok := got[KindHFF]; ok {
		t.Fatal("she is not in the published table")
	}
}

func TestSideOnTheLineIsLong(t *testing.T) {
	if 六边形.Name == "" {
		t.Fatal("hex")
	}
	f := unit.Field{Shape: unit.ShapeHex, Extent: 280}
	lo, hi := SpanX(f)
	if !(hi > lo) || math.Abs(lo+hi) > 1 {
		t.Fatalf("span %v %v", lo, hi)
	}
	line, long := Side(0, lo, hi, 0.5)
	if math.Abs(line) > 1 || !long {
		t.Fatalf("mid line %v long %v", line, long)
	}
	if _, long := Side(line+1, lo, hi, 0.5); long {
		t.Fatal("right of the line should be short")
	}
	if _, long := Side(line, lo, hi, 0.5); !long {
		t.Fatal("on the line is long")
	}
}
