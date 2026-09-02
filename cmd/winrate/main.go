package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
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
	jobs := flag.Int("j", runtime.NumCPU()*8, "同时进行的场次数")
	flag.Parse()

	kinds := unit.FighterKinds()
	if len(kinds) < 1 {
		fmt.Fprintln(os.Stderr, "没有已注册的战斗机")
		os.Exit(1)
	}
	maxTicks := int(*maxSec * sim.TickHz)
	if maxTicks < 1 {
		maxTicks = 1
	}
	if *jobs < 1 {
		*jobs = 1
	}

	type pair struct{ a, b string }
	var pairs []pair
	for i := range kinds {
		for j := i; j < len(kinds); j++ {
			pairs = append(pairs, pair{kinds[i], kinds[j]})
		}
	}

	gate := make(chan struct{}, *jobs)
	var done atomic.Int64
	total := int64(len(pairs) * *nGames)
	fmt.Fprintf(os.Stderr, "组合 %d，每组 %d 场，共 %d 场，并发 %d\n", len(pairs), *nGames, total, *jobs)

	results := make([]pairResult, len(pairs))
	var wgPairs sync.WaitGroup
	t0 := time.Now()
	for pi, p := range pairs {
		wgPairs.Add(1)
		go func(pi int, p pair) {
			defer wgPairs.Done()
			var wg sync.WaitGroup
			wA := make([]int, *nGames)
			wB := make([]int, *nGames)
			wD := make([]int, *nGames)
			wT := make([]int, *nGames)
			for n := 0; n < *nGames; n++ {
				wg.Add(1)
				go func(n int) {
					defer wg.Done()
					gate <- struct{}{}
					defer func() { <-gate }()
					left, right := p.a, p.b
					if p.a != p.b && n%2 == 1 {
						left, right = p.b, p.a
					}
					seed := mixSeed(left, right, uint64(n)+1)
					m := sim.NewMatchSeeded(seed)
					m.SetSlot(0, left)
					m.SetSlot(1, right)
					m.Start()
					winner, ticks := m.Play(maxTicks)
					m.End()
					if ticks >= maxTicks && winner == "平局" {
						wT[n] = 1
					}
					switch {
					case winner == "平局" || winner == "":
						wD[n] = 1
					case winner == p.a && winner == p.b:
						wA[n] = 1
					case winner == p.a:
						wA[n] = 1
					case winner == p.b:
						wB[n] = 1
					default:
						wD[n] = 1
					}
					done.Add(1)
				}(n)
			}
			wg.Wait()
			r := pairResult{a: p.a, b: p.b}
			for n := 0; n < *nGames; n++ {
				r.winsA += wA[n]
				r.winsB += wB[n]
				r.draws += wD[n]
				r.timeout += wT[n]
			}
			results[pi] = r
		}(pi, p)
	}
	wgPairs.Wait()
	elapsed := time.Since(t0)
	fmt.Fprintf(os.Stderr, "打完 %d 场，用时 %s\n", done.Load(), elapsed.Round(time.Millisecond))

	text := render(kinds, results, *nGames, *maxSec, elapsed)
	if err := os.WriteFile(*outPath, []byte(text), 0o644); err != nil {
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

func render(kinds []string, results []pairResult, nGames int, maxSec float64, elapsed time.Duration) string {
	byPair := map[[2]string]pairResult{}
	solo := map[string]*tally{}
	for _, k := range kinds {
		solo[k] = &tally{}
	}
	for _, r := range results {
		byPair[[2]string{r.a, r.b}] = r
		if r.a == r.b {
			continue
		}
		solo[r.a].win += r.winsA
		solo[r.a].lose += r.winsB
		solo[r.a].draw += r.draws
		solo[r.b].win += r.winsB
		solo[r.b].lose += r.winsA
		solo[r.b].draw += r.draws
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# 小球胜率\n\n")
	fmt.Fprintf(&b, "- 时间：%s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "- 每组对战：%d 场（异名组合左右槽各一半）\n", nGames)
	fmt.Fprintf(&b, "- 超时：%.0f 秒模拟时间，超时计平\n", maxSec)
	fmt.Fprintf(&b, "- 组合数：%d，用时：%s\n\n", len(results), elapsed.Round(time.Millisecond))

	fmt.Fprintf(&b, "## 单球胜率\n\n")
	fmt.Fprintf(&b, "不含同名对打。胜率 = 胜 /（胜+负+平）。\n\n")
	fmt.Fprintf(&b, "| 角色 | 场次 | 胜 | 负 | 平 | 胜率 |\n")
	fmt.Fprintf(&b, "| --- | ---: | ---: | ---: | ---: | ---: |\n")
	type row struct {
		kind string
		t    tally
	}
	var rows []row
	for _, k := range kinds {
		rows = append(rows, row{k, *solo[k]})
	}
	sort.Slice(rows, func(i, j int) bool {
		gi, gj := games(rows[i].t), games(rows[j].t)
		ri, rj := rate(rows[i].t), rate(rows[j].t)
		if ri != rj {
			return ri > rj
		}
		if gi != gj {
			return gi > gj
		}
		return rows[i].kind < rows[j].kind
	})
	for _, r := range rows {
		g := games(r.t)
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d | %s |\n", r.kind, g, r.t.win, r.t.lose, r.t.draw, pct(rate(r.t)))
	}

	fmt.Fprintf(&b, "\n## 一对一胜率\n\n")
	fmt.Fprintf(&b, "单元格为行对列的胜率（行赢的场次 / %d）。对角线为同名对打中有胜负的比例（不是某侧胜率）。\n\n", nGames)
	fmt.Fprintf(&b, "|")
	for _, k := range kinds {
		fmt.Fprintf(&b, " | %s", k)
	}
	fmt.Fprintf(&b, " |\n| ---")
	for range kinds {
		fmt.Fprintf(&b, " | ---:")
	}
	fmt.Fprintf(&b, " |\n")
	for _, rowK := range kinds {
		fmt.Fprintf(&b, "| %s", rowK)
		for _, colK := range kinds {
			fmt.Fprintf(&b, " | %s", cell(byPair, rowK, colK, nGames))
		}
		fmt.Fprintf(&b, " |\n")
	}

	fmt.Fprintf(&b, "\n## 一对一明细\n\n")
	fmt.Fprintf(&b, "| 对阵 | 场次 | 前者胜 | 后者胜 | 平 | 超时平 |\n")
	fmt.Fprintf(&b, "| --- | ---: | ---: | ---: | ---: | ---: |\n")
	sorted := append([]pairResult(nil), results...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].a != sorted[j].a {
			return sorted[i].a < sorted[j].a
		}
		return sorted[i].b < sorted[j].b
	})
	for _, r := range sorted {
		label := r.a + " vs " + r.b
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d | %d |\n", label, nGames, r.winsA, r.winsB, r.draws, r.timeout)
	}
	fmt.Fprintf(&b, "\n")
	return b.String()
}

func games(t tally) int { return t.win + t.lose + t.draw }

func rate(t tally) float64 {
	g := games(t)
	if g == 0 {
		return 0
	}
	return float64(t.win) / float64(g)
}

func pct(r float64) string { return fmt.Sprintf("%.1f%%", r*100) }

func cell(byPair map[[2]string]pairResult, row, col string, n int) string {
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
	if row == col {
		return pct(float64(r.winsA) / float64(n))
	}
	w := r.winsA
	if row == r.b {
		w = r.winsB
	} else if row != r.a {
		return "—"
	}
	return pct(float64(w) / float64(n))
}
