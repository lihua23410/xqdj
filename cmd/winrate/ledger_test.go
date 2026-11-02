package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestMergeUpdatesBothSides(t *testing.T) {
	kinds := []string{"甲", "乙", "丙"}
	text := oldReport(10, 90, `
| 甲 vs 乙 | 10 | 10 | 0 | 0 | 0 | 100.0% | 0.0% |
| 甲 vs 丙 | 10 | 0 | 10 | 0 | 0 | 0.0% | 100.0% |
| 乙 vs 丙 | 10 | 5 | 5 | 0 | 0 | 50.0% | 50.0% |
`)
	prior, err := loadLedger(kinds, "甲", 10, 90, text)
	if err != nil {
		t.Fatal(err)
	}
	fresh := []pairResult{
		{"甲", "乙", 1, 9, 0, 0},
		{"甲", "丙", 4, 6, 0, 0},
	}
	merged, err := mergeLedger(kinds, "甲", prior, fresh)
	if err != nil {
		t.Fatal(err)
	}
	got := pairMap(merged)
	if got[[2]string{"甲", "乙"}] != fresh[0] || got[[2]string{"甲", "丙"}] != fresh[1] {
		t.Fatalf("replaced pairs %+v", got)
	}
	if got[[2]string{"乙", "丙"}] != (pairResult{"乙", "丙", 5, 5, 0, 0}) {
		t.Fatalf("untouched pair %+v", got[[2]string{"乙", "丙"}])
	}
	solo, _ := tallies(kinds, merged)
	if solo["甲"] != (tally{win: 5, lose: 15}) || solo["乙"] != (tally{win: 14, lose: 6}) {
		t.Fatalf("tallies 甲=%+v 乙=%+v", solo["甲"], solo["乙"])
	}
	report := render(kinds, merged, 10, 90, time.Second, 4, "甲")
	if !strings.Contains(report, "| 1 | 乙 | 28 | 20 | 14 | 6 | 0 | 70.0% | 70.0% |") {
		t.Fatalf("solo row for 乙 missing:\n%s", report)
	}
	if strings.Contains(report, "范围：仅") {
		t.Fatal("incremental report still marked partial")
	}
	if !strings.Contains(report, "- 本次并发：4 场\n") || !strings.Contains(report, "本次用时：") || !strings.Contains(report, "- 上次增量：甲\n") {
		t.Fatalf("header missing fields:\n%s", report)
	}
}

func TestMergeOrientsLaterFighter(t *testing.T) {
	kinds := []string{"甲", "乙"}
	prior, err := loadLedger(kinds, "乙", 10, 90, oldReport(10, 90, `
| 甲 vs 乙 | 10 | 10 | 0 | 0 | 0 | 100.0% | 0.0% |
`))
	if err != nil {
		t.Fatal(err)
	}
	// -only 把指定球放在前者；名册里甲更靠前，写回时要换位并交换胜场。
	merged, err := mergeLedger(kinds, "乙", prior, []pairResult{{"乙", "甲", 3, 7, 0, 0}})
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 1 || merged[0] != (pairResult{"甲", "乙", 7, 3, 0, 0}) {
		t.Fatalf("oriented %+v", merged)
	}
}

func TestLoadKeepsTimeoutAndBracketNames(t *testing.T) {
	kinds := []string{"中转站[Freischutz]", "分身者(百战)"}
	got, err := loadLedger(kinds, "分身者(百战)", 10, 90, oldReport(10, 90, `
| 中转站[Freischutz] vs 分身者(百战) | 10 | 3 | 3 | 4 | 1 | 70.0% | 70.0% |
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != (pairResult{"中转站[Freischutz]", "分身者(百战)", 3, 3, 4, 1}) {
		t.Fatalf("parsed %+v", got)
	}
}

func TestDroppedFighterLeavesTheRest(t *testing.T) {
	kinds := []string{"甲", "乙"}
	got, err := loadLedger(kinds, "甲", 10, 90, oldReport(10, 90, `
| 甲 vs 乙 | 10 | 4 | 4 | 2 | 2 | 60.0% | 60.0% |
| 甲 vs 旧 | 10 | 1 | 2 | 7 | 0 | 80.0% | 90.0% |
| 乙 vs 旧 | 10 | 0 | 0 | 10 | 10 | 100.0% | 100.0% |
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].a != "甲" || got[0].b != "乙" || got[0].timeout != 2 {
		t.Fatalf("kept %+v", got)
	}
}

func TestOnlyNewFighterCompletesRoster(t *testing.T) {
	kinds := []string{"甲", "乙", "丙"}
	text := oldReport(10, 90, `
| 甲 vs 乙 | 10 | 2 | 2 | 6 | 0 | 80.0% | 80.0% |
`)
	if _, err := loadLedger(kinds, "丙", 10, 90, text); err != nil {
		t.Fatal(err)
	}
	prior, err := loadLedger(kinds, "丙", 10, 90, text)
	if err != nil {
		t.Fatal(err)
	}
	merged, err := mergeLedger(kinds, "丙", prior, []pairResult{
		{"丙", "甲", 8, 1, 1, 1},
		{"丙", "乙", 0, 9, 1, 0},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := pairMap(merged)
	if got[[2]string{"甲", "丙"}] != (pairResult{"甲", "丙", 1, 8, 1, 1}) {
		t.Fatalf("new pair oriented %+v", got[[2]string{"甲", "丙"}])
	}
	if _, ok := got[[2]string{"甲", "乙"}]; !ok || len(got) != 3 {
		t.Fatalf("roster %+v", got)
	}
}

func TestRefusePartialSampleRosterAndMissingFile(t *testing.T) {
	// 这些拒绝都发生在 readPrior / loadLedger，不会调用对局。
	kinds := []string{"甲", "乙", "丙"}
	complete := oldReport(10, 90, `
| 甲 vs 乙 | 10 | 1 | 1 | 8 | 0 | 90.0% | 90.0% |
| 甲 vs 丙 | 10 | 1 | 1 | 8 | 0 | 90.0% | 90.0% |
| 乙 vs 丙 | 10 | 1 | 1 | 8 | 0 | 90.0% | 90.0% |
`)
	partial := strings.Replace(complete, "- 组合数：1，用时：1s", "- 组合数：1，用时：1s\n- 范围：仅 甲 vs 其余战斗机；其他球的单球胜率是对阵 甲 的成绩", 1)
	if err := mustRefuse(loadLedger(kinds, "甲", 10, 90, partial)); !strings.Contains(err, "残缺") {
		t.Fatalf("partial: %v", err)
	}
	if err := mustRefuse(loadLedger(kinds, "甲", 50, 90, complete)); !strings.Contains(err, "不一致") || !strings.Contains(err, "-n") {
		t.Fatalf("n: %v", err)
	}
	if err := mustRefuse(loadLedger(kinds, "甲", 10, 60, complete)); !strings.Contains(err, "不一致") || !strings.Contains(err, "-timeout") {
		t.Fatalf("timeout: %v", err)
	}
	broken := oldReport(10, 90, "| 甲 vs 乙 | 10 | 1 | 1 | 1 | 0 | 20.0% | 20.0% |\n")
	if err := mustRefuse(loadLedger(kinds, "甲", 10, 90, broken)); !strings.Contains(err, "胜负平") {
		t.Fatalf("counts: %v", err)
	}

	oneNew := oldReport(10, 90, `
| 甲 vs 乙 | 10 | 1 | 1 | 8 | 0 | 90.0% | 90.0% |
`)
	if err := mustRefuse(loadLedger(kinds, "甲", 10, 90, oneNew)); !strings.Contains(err, "只有 -only 丙") {
		t.Fatalf("old while new missing: %v", err)
	}

	twoNew := []string{"甲", "乙", "丙", "丁"}
	if err := mustRefuse(loadLedger(twoNew, "丙", 10, 90, oneNew)); !strings.Contains(err, "丙") || !strings.Contains(err, "丁") || !strings.Contains(err, "全量重跑") {
		t.Fatalf("two new: %v", err)
	}
	if err := mustRefuse(loadLedger(twoNew, "甲", 10, 90, oneNew)); !strings.Contains(err, "全量重跑") {
		t.Fatalf("two new via old fighter: %v", err)
	}

	dir := t.TempDir()
	if _, err := readPrior(filepath.Join(dir, "missing.md"), kinds, "甲", 10, 90); err == nil || !strings.Contains(err.Error(), "没有可合并的全表") {
		t.Fatalf("missing file: %v", err)
	}
}

func TestRoundTripAndAtomicWrite(t *testing.T) {
	kinds := []string{"甲", "乙", "丙"}
	results := []pairResult{
		{"甲", "乙", 6, 3, 1, 1},
		{"甲", "丙", 0, 0, 10, 4},
		{"乙", "丙", 2, 8, 0, 0},
	}
	text := render(kinds, results, 10, 90, 1500*time.Millisecond, 8, "乙")
	got, err := loadLedger(kinds, "甲", 10, 90, text)
	if err != nil {
		t.Fatal(err)
	}
	if !samePairs(results, got) {
		t.Fatalf("round trip %+v", got)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "winrate.md")
	if err := os.WriteFile(path, []byte("旧账"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeReport(path, text); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != text {
		t.Fatal("replaced file does not match")
	}
	left, err := filepath.Glob(filepath.Join(dir, ".winrate-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Fatalf("temp files left behind: %v", left)
	}
}

func TestTiedPointsShareRank(t *testing.T) {
	kinds := []string{"甲", "乙", "丙"}
	results := []pairResult{
		{"甲", "乙", 0, 0, 10, 0},
		{"甲", "丙", 10, 0, 0, 0},
		{"乙", "丙", 10, 0, 0, 0},
	}
	report := render(kinds, results, 10, 90, time.Second, 1, "")
	if !strings.Contains(report, "| 1 | 甲 | 30 | 20 | 10 | 0 | 10 | 100.0% | 100.0% |") ||
		!strings.Contains(report, "| 1 | 乙 | 30 | 20 | 10 | 0 | 10 | 100.0% | 100.0% |") ||
		!strings.Contains(report, "| 3 | 丙 | 0 | 20 | 0 | 20 | 0 | 0.0% | 0.0% |") {
		t.Fatalf("shared rank:\n%s", report)
	}
}

func TestSamePointsOrdersByWins(t *testing.T) {
	kinds := []string{"甲", "乙", "丙"}
	results := []pairResult{
		{"甲", "乙", 6, 1, 3, 0},
		{"甲", "丙", 0, 10, 0, 0},
		{"乙", "丙", 2, 2, 6, 0},
	}
	report := render(kinds, results, 10, 90, time.Second, 1, "")
	yi := strings.Index(report, "| 2 | 甲 |")
	bing := strings.Index(report, "| 2 | 乙 |")
	if yi < 0 || bing < 0 || yi > bing {
		t.Fatalf("more wins should list first:\n%s", report)
	}
}

func TestWinRateIgnoresDraws(t *testing.T) {
	if rate(tally{win: 1, lose: 1, draw: 8}) != 0.5 {
		t.Fatalf("rate=%v", rate(tally{win: 1, lose: 1, draw: 8}))
	}
	if rate(tally{draw: 4}) != 0 {
		t.Fatal("no decisive games")
	}
}

func TestBenchKindDropsOut(t *testing.T) {
	if got := rankedKinds([]string{"甲", benchKind, "乙"}); len(got) != 2 || got[0] != "甲" || got[1] != "乙" {
		t.Fatalf("ranked %v", got)
	}
	kinds := []string{"甲", "乙"}
	text := oldReport(10, 90, `
| 甲 vs 乙 | 10 | 4 | 4 | 2 | 2 | 60.0% | 60.0% |
| 甲 vs 靶子 | 10 | 1 | 0 | 9 | 9 | 100.0% | 90.0% |
| 乙 vs 靶子 | 10 | 0 | 10 | 0 | 0 | 0.0% | 100.0% |
`)
	got, err := loadLedger(kinds, "甲", 10, 90, text)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].a != "甲" || got[0].b != "乙" {
		t.Fatalf("kept %+v", got)
	}
	report := render(kinds, got, 10, 90, time.Second, 1, "")
	if strings.Contains(report, benchKind) {
		t.Fatalf("bench kind still in report:\n%s", report)
	}
}

func TestRebuildDropsBench(t *testing.T) {
	src, err := os.ReadFile("../../winrate.md")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "winrate.md")
	if err := os.WriteFile(path, src, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := rebuildReport(path); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if strings.Contains(text, benchKind) {
		t.Fatal("rebuild kept 靶子")
	}
	if !strings.Contains(text, "胜率 = 胜 /（胜+负），平不计入。") {
		t.Fatal("win rate formula missing")
	}
	if !strings.Contains(text, "| 排名 | 战斗机 | 积分 |") {
		t.Fatal("rank table missing")
	}
}

func TestCommittedReportStillParses(t *testing.T) {
	b, err := os.ReadFile("../../winrate.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	n, sec, err := parseHead(preamble(text))
	if err != nil {
		t.Fatal(err)
	}
	if n != 100 || sec != 90 {
		t.Fatalf("header n=%d timeout=%v", n, sec)
	}
	rows, err := parseDetail(text, n)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("detail table is empty")
	}
	kinds := map[string]struct{}{}
	for _, r := range rows {
		kinds[r.a] = struct{}{}
		kinds[r.b] = struct{}{}
	}
	if _, err := loadLedger(rosterOf(kinds), rosterOf(kinds)[0], n, sec, text); err != nil {
		t.Fatal(err)
	}
}

func rosterOf(set map[string]struct{}) []string {
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func oldReport(n int, sec float64, rows string) string {
	return fmt.Sprintf(`# 小球胜率

- 时间：2026-11-02 09:54:25
- 每组对战：%d 场（左右槽各一半）
- 超时：%.0f 秒模拟时间，超时计平
- 并发上限：32 场
- 事件处理：可靠投递、保留逐拍感知、阶段背压；场内角色仍异步
- 组合数：1，用时：1s

## 一对一明细

| 对阵 | 场次 | 前者胜 | 后者胜 | 平 | 超时平 | 前者胜平率 | 后者胜平率 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
%s`, n, sec, rows)
}

func mustRefuse(_ []pairResult, err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func pairMap(rows []pairResult) map[[2]string]pairResult {
	out := map[[2]string]pairResult{}
	for _, r := range rows {
		out[[2]string{r.a, r.b}] = r
	}
	return out
}

func samePairs(a, b []pairResult) bool {
	if len(a) != len(b) {
		return false
	}
	got := pairMap(b)
	for _, r := range a {
		if got[[2]string{r.a, r.b}] != r {
			return false
		}
	}
	return true
}
