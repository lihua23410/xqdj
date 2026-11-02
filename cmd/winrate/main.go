package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"xqdj/internal/sim"
	"xqdj/internal/unit"

	_ "xqdj/character"
)

type pairResult struct {
	a, b           string
	winsA, winsB   int
	draws, timeout int
}

type tally struct {
	win, lose, draw int
}

func main() {
	nGames := flag.Int("n", 100, "每组对战场次")
	maxSec := flag.Float64("timeout", 90, "单场模拟超时秒数，超时计平")
	outPath := flag.String("o", "winrate.md", "输出文件")
	jobs := flag.Int("j", runtime.NumCPU()*2, "同时进行的场次数")
	only := flag.String("only", "", "只测该战斗机并写回全表（不含自己打自己）")
	flag.Parse()
	if *nGames < 1 {
		fmt.Fprintln(os.Stderr, "每组场次 -n 必须大于 0")
		os.Exit(1)
	}

	kinds := rankedKinds(unit.FighterKinds())
	if len(kinds) < 2 {
		fmt.Fprintln(os.Stderr, "没有足够的战斗机")
		os.Exit(1)
	}
	if *only != "" {
		found := false
		for _, k := range kinds {
			if k == *only {
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "没有战斗机 %q，已注册：%s\n", *only, strings.Join(kinds, " "))
			os.Exit(1)
		}
	}
	maxTicks := int(*maxSec * sim.TickHz)
	if maxTicks < 1 {
		maxTicks = 1
	}
	if *jobs < 1 {
		*jobs = 1
	}

	var prior []pairResult
	if *only != "" {
		var err error
		prior, err = readPrior(*outPath, kinds, *only, *nGames, *maxSec)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	var pairs []pair
	if *only != "" {
		for _, k := range kinds {
			if k == *only {
				continue
			}
			pairs = append(pairs, pair{*only, k})
		}
	} else {
		for i := range kinds {
			for j := i + 1; j < len(kinds); j++ {
				pairs = append(pairs, pair{kinds[i], kinds[j]})
			}
		}
	}

	total := len(pairs) * *nGames
	fmt.Fprintf(os.Stderr, "组合 %d，每组 %d 场，共 %d 场，并发 %d\n", len(pairs), *nGames, total, *jobs)
	t0 := time.Now()
	results, done := runPairs(pairs, *nGames, maxTicks, *jobs, playMatch)
	elapsed := time.Since(t0)
	fmt.Fprintf(os.Stderr, "打完 %d 场，用时 %s\n", done, elapsed.Round(time.Millisecond))

	updated := ""
	merged := results
	if *only != "" {
		var err error
		merged, err = mergeLedger(kinds, *only, prior, results)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		updated = *only
	} else {
		var err error
		merged, err = orientAll(kinds, results)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	text := render(kinds, merged, *nGames, *maxSec, elapsed, *jobs, updated)
	if err := writeReport(*outPath, text); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "写入", *outPath)
}

func mixSeed(a, b string, n uint64) uint64 {
	h := uint64(14695981039346656037)
	for _, c := range a + "\x00" + b {
		h ^= uint64(c)
		h *= 1099511628211
	}
	h ^= n
	h *= 1099511628211
	return h
}

func render(kinds []string, results []pairResult, nGames int, maxSec float64, elapsed time.Duration, jobs int, updated string) string {
	solo, byPair := tallies(kinds, results)

	var b strings.Builder
	fmt.Fprintf(&b, "# 小球胜率\n\n")
	fmt.Fprintf(&b, "- 时间：%s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "- 每组对战：%d 场（左右槽各一半）\n", nGames)
	fmt.Fprintf(&b, "- 超时：%.0f 秒模拟时间，超时计平\n", maxSec)
	fmt.Fprintf(&b, "- 本次并发：%d 场\n", jobs)
	fmt.Fprintf(&b, "- 事件处理：可靠投递、保留逐拍感知、阶段背压；场内角色仍异步\n")
	fmt.Fprintf(&b, "- 组合数：%d，本次用时：%s\n", len(results), elapsed.Round(time.Millisecond))
	if updated != "" {
		fmt.Fprintf(&b, "- 上次增量：%s\n", updated)
	}
	fmt.Fprintf(&b, "\n")

	fmt.Fprintf(&b, "## 单球胜率\n\n")
	fmt.Fprintf(&b, "胜率 = 胜 /（胜+负），平不计入。胜平率 =（胜+平）/（胜+负+平）。积分 = 胜×2 + 平，负不计分。排名按积分，积分相同则并列，下一名跳号。不含自己打自己。\n\n")
	fmt.Fprintf(&b, "| 排名 | 战斗机 | 积分 | 场次 | 胜 | 负 | 平 | 胜率 | 胜平率 |\n")
	fmt.Fprintf(&b, "| ---: | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	type row struct {
		kind string
		t    tally
	}
	var rows []row
	for _, k := range kinds {
		rows = append(rows, row{k, solo[k]})
	}
	sort.Slice(rows, func(i, j int) bool {
		pi, pj := points(rows[i].t), points(rows[j].t)
		if pi != pj {
			return pi > pj
		}
		if rows[i].t.win != rows[j].t.win {
			return rows[i].t.win > rows[j].t.win
		}
		return rows[i].kind < rows[j].kind
	})
	rank := 1
	for i, r := range rows {
		if i > 0 && points(r.t) != points(rows[i-1].t) {
			rank = i + 1
		}
		g := games(r.t)
		fmt.Fprintf(&b, "| %d | %s | %d | %d | %d | %d | %d | %s | %s |\n",
			rank, r.kind, points(r.t), g, r.t.win, r.t.lose, r.t.draw, pct(rate(r.t)), pct(winDraw(r.t)))
	}

	fmt.Fprintf(&b, "\n## 一对一胜率\n\n")
	fmt.Fprintf(&b, "单元格为行对列的胜率（行赢的场次 / %d）。对角线是自己打自己，不测。\n\n", nGames)
	writeMatrix(&b, kinds, byPair, nGames, false)

	fmt.Fprintf(&b, "\n## 一对一胜平率\n\n")
	fmt.Fprintf(&b, "单元格为行对列的胜平率（行赢+平 / %d）。对角线是自己打自己，不测。\n\n", nGames)
	writeMatrix(&b, kinds, byPair, nGames, true)

	fmt.Fprintf(&b, "\n## 一对一明细\n\n")
	fmt.Fprintf(&b, "| 对阵 | 场次 | 前者胜 | 后者胜 | 平 | 超时平 | 前者胜平率 | 后者胜平率 |\n")
	fmt.Fprintf(&b, "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	sorted := append([]pairResult(nil), results...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].a != sorted[j].a {
			return sorted[i].a < sorted[j].a
		}
		return sorted[i].b < sorted[j].b
	})
	for _, r := range sorted {
		label := r.a + " vs " + r.b
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d | %d | %s | %s |\n",
			label, nGames, r.winsA, r.winsB, r.draws, r.timeout,
			pct(float64(r.winsA+r.draws)/float64(nGames)),
			pct(float64(r.winsB+r.draws)/float64(nGames)))
	}
	fmt.Fprintf(&b, "\n")
	return b.String()
}

func tallies(kinds []string, results []pairResult) (map[string]tally, map[[2]string]pairResult) {
	byPair := map[[2]string]pairResult{}
	solo := map[string]tally{}
	for _, k := range kinds {
		solo[k] = tally{}
	}
	for _, r := range results {
		byPair[[2]string{r.a, r.b}] = r
		a := solo[r.a]
		a.win += r.winsA
		a.lose += r.winsB
		a.draw += r.draws
		solo[r.a] = a
		b := solo[r.b]
		b.win += r.winsB
		b.lose += r.winsA
		b.draw += r.draws
		solo[r.b] = b
	}
	return solo, byPair
}

func games(t tally) int { return t.win + t.lose + t.draw }

func points(t tally) int { return t.win*2 + t.draw }

// 靶子是沙包，胜率名册里当它不存在：不对打、不进总榜、不进矩阵、不进明细。
const benchKind = "靶子"

func rankedKinds(all []string) []string {
	out := make([]string, 0, len(all))
	for _, k := range all {
		if k != benchKind {
			out = append(out, k)
		}
	}
	return out
}

func rate(t tally) float64 {
	decided := t.win + t.lose
	if decided == 0 {
		return 0
	}
	return float64(t.win) / float64(decided)
}

func winDraw(t tally) float64 {
	g := games(t)
	if g == 0 {
		return 0
	}
	return float64(t.win+t.draw) / float64(g)
}

func pct(r float64) string { return fmt.Sprintf("%.1f%%", r*100) }

func writeMatrix(b *strings.Builder, kinds []string, byPair map[[2]string]pairResult, nGames int, withDraw bool) {
	fmt.Fprintf(b, "|")
	for _, k := range kinds {
		fmt.Fprintf(b, " | %s", k)
	}
	fmt.Fprintf(b, " |\n| ---")
	for range kinds {
		fmt.Fprintf(b, " | ---:")
	}
	fmt.Fprintf(b, " |\n")
	for _, rowK := range kinds {
		fmt.Fprintf(b, "| %s", rowK)
		for _, colK := range kinds {
			fmt.Fprintf(b, " | %s", cell(byPair, rowK, colK, nGames, withDraw))
		}
		fmt.Fprintf(b, " |\n")
	}
}

func cell(byPair map[[2]string]pairResult, row, col string, n int, withDraw bool) string {
	if n == 0 {
		return "—"
	}
	r, ok := byPair[[2]string{row, col}]
	if !ok {
		r, ok = byPair[[2]string{col, row}]
	}
	if !ok {
		return "—"
	}
	w := r.winsA
	if row == r.b {
		w = r.winsB
	} else if row != r.a {
		return "—"
	}
	if withDraw {
		w += r.draws
	}
	return pct(float64(w) / float64(n))
}
