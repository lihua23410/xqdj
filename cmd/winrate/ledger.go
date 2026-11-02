package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"xqdj/internal/unit"
)

var (
	reGames   = regexp.MustCompile(`每组对战：(\d+) 场`)
	reTimeout = regexp.MustCompile(`超时：([0-9]+(?:\.[0-9]+)?) 秒`)
	reJobs    = regexp.MustCompile(`(?:本次并发|并发上限)：(\d+)`)
	reElapsed = regexp.MustCompile(`用时：(\S+)`)
	reUpdated = regexp.MustCompile(`上次增量：(.+)`)
	reWhen    = regexp.MustCompile(`时间：(.+)`)
)

var detailHeader = []string{"对阵", "场次", "前者胜", "后者胜", "平", "超时平", "前者胜平率", "后者胜平率"}

// readPrior 在开打前读入全表明细。对不上就返回错误，调用方不得开打，也不得改文件。
func readPrior(path string, kinds []string, only string, nGames int, maxSec float64) ([]pairResult, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("没有可合并的全表 %s，先做一次不带 -only 的全量", path)
		}
		return nil, err
	}
	return loadLedger(kinds, only, nGames, maxSec, string(b))
}

func loadLedger(kinds []string, only string, nGames int, maxSec float64, text string) ([]pairResult, error) {
	head := preamble(text)
	if strings.Contains(head, "范围：仅") {
		return nil, fmt.Errorf("报告是残缺的单只结果，不能当作全表合并")
	}
	n, sec, err := parseHead(head)
	if err != nil {
		return nil, err
	}
	if n != nGames {
		return nil, fmt.Errorf("-n %d 与报告头的每组对战 %d 场不一致，改样本量要全量重跑", nGames, n)
	}
	if sec != maxSec {
		return nil, fmt.Errorf("-timeout %v 与报告头的超时 %v 秒不一致，改超时要全量重跑", maxSec, sec)
	}
	raw, err := parseDetail(text, n)
	if err != nil {
		return nil, err
	}
	order := kindOrder(kinds)
	seen := map[[2]string]bool{}
	kept := make([]pairResult, 0, len(raw))
	for _, r := range raw {
		_, oka := order[r.a]
		_, okb := order[r.b]
		if !oka || !okb {
			continue
		}
		c, err := orient(r, order)
		if err != nil {
			return nil, err
		}
		key := [2]string{c.a, c.b}
		if seen[key] {
			return nil, fmt.Errorf("解不开报告：对阵 %s vs %s 重复", c.a, c.b)
		}
		seen[key] = true
		kept = append(kept, c)
	}
	if err := complementComplete(kinds, only, kept); err != nil {
		return nil, err
	}
	return kept, nil
}

func preamble(text string) string {
	if i := strings.Index(text, "\n## "); i >= 0 {
		return text[:i]
	}
	return text
}

func parseHead(head string) (int, float64, error) {
	gm := reGames.FindStringSubmatch(head)
	if gm == nil {
		return 0, 0, fmt.Errorf("解不开报告：没有每组对战场次")
	}
	n, err := strconv.Atoi(gm[1])
	if err != nil {
		return 0, 0, fmt.Errorf("解不开报告：没有每组对战场次")
	}
	tm := reTimeout.FindStringSubmatch(head)
	if tm == nil {
		return 0, 0, fmt.Errorf("解不开报告：没有超时")
	}
	sec, err := strconv.ParseFloat(tm[1], 64)
	if err != nil {
		return 0, 0, fmt.Errorf("解不开报告：没有超时")
	}
	return n, sec, nil
}

func parseDetail(text string, nGames int) ([]pairResult, error) {
	const marker = "## 一对一明细"
	i := strings.Index(text, marker)
	if i < 0 {
		return nil, fmt.Errorf("解不开报告：没有一对一明细")
	}
	body := text[i+len(marker):]
	if j := strings.Index(body, "\n## "); j >= 0 {
		body = body[:j]
	}
	var header []string
	var rows []pairResult
	for _, line := range strings.Split(body, "\n") {
		cells := splitRow(line)
		if cells == nil {
			if header != nil && strings.TrimSpace(line) != "" {
				break
			}
			continue
		}
		if header == nil {
			if isSeparator(cells) {
				continue
			}
			header = cells
			if strings.Join(header, "\t") != strings.Join(detailHeader, "\t") {
				return nil, fmt.Errorf("解不开报告：一对一明细的表头变了")
			}
			continue
		}
		if isSeparator(cells) {
			continue
		}
		row, err := parseDetailRow(cells, nGames)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	if header == nil {
		return nil, fmt.Errorf("解不开报告：没有一对一明细")
	}
	return rows, nil
}

func parseDetailRow(cells []string, nGames int) (pairResult, error) {
	if len(cells) != len(detailHeader) {
		return pairResult{}, fmt.Errorf("解不开报告：对阵列数不对：%s", strings.Join(cells, " | "))
	}
	a, b, err := splitMatchup(cells[0])
	if err != nil {
		return pairResult{}, err
	}
	nums := make([]int, 5)
	for i := 0; i < 5; i++ {
		nums[i], err = strconv.Atoi(cells[i+1])
		if err != nil {
			return pairResult{}, fmt.Errorf("解不开报告：对阵 %s vs %s 的场次解不开", a, b)
		}
	}
	label := a + " vs " + b
	if nums[0] != nGames {
		return pairResult{}, fmt.Errorf("解不开报告：对阵 %s 的场次是 %d，报告头是 %d", label, nums[0], nGames)
	}
	if err := checkCounts(label, nums[0], nums[1], nums[2], nums[3], nums[4]); err != nil {
		return pairResult{}, err
	}
	return pairResult{a: a, b: b, winsA: nums[1], winsB: nums[2], draws: nums[3], timeout: nums[4]}, nil
}

func checkCounts(label string, games, winsA, winsB, draws, timeout int) error {
	if games < 0 || winsA < 0 || winsB < 0 || draws < 0 || timeout < 0 {
		return fmt.Errorf("解不开报告：对阵 %s 有负数", label)
	}
	if winsA+winsB+draws != games {
		return fmt.Errorf("解不开报告：对阵 %s 的胜负平加起来是 %d，不是 %d 场", label, winsA+winsB+draws, games)
	}
	if timeout > draws {
		return fmt.Errorf("解不开报告：对阵 %s 的超时平 %d 多于平 %d", label, timeout, draws)
	}
	return nil
}

func splitMatchup(s string) (string, string, error) {
	const sep = " vs "
	if strings.Count(s, sep) != 1 {
		return "", "", fmt.Errorf("解不开报告：对阵列解不开：%q", s)
	}
	a, b, _ := strings.Cut(s, sep)
	if a == "" || b == "" {
		return "", "", fmt.Errorf("解不开报告：对阵列解不开：%q", s)
	}
	return a, b, nil
}

func splitRow(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") {
		return nil
	}
	parts := strings.Split(line, "|")
	if len(parts) > 0 && strings.TrimSpace(parts[0]) == "" {
		parts = parts[1:]
	}
	if len(parts) > 0 && strings.TrimSpace(parts[len(parts)-1]) == "" {
		parts = parts[:len(parts)-1]
	}
	if len(parts) == 0 {
		return nil
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func isSeparator(cells []string) bool {
	for _, c := range cells {
		if c == "" {
			return false
		}
		for _, r := range c {
			if r != '-' && r != ':' {
				return false
			}
		}
	}
	return true
}

func kindOrder(kinds []string) map[string]int {
	order := make(map[string]int, len(kinds))
	for i, k := range kinds {
		order[k] = i
	}
	return order
}

// orient 按名册顺序固定前者、后者。名册里较后的名字若写在前者，胜场对调。
func orient(r pairResult, order map[string]int) (pairResult, error) {
	ia, oka := order[r.a]
	ib, okb := order[r.b]
	if !oka || !okb {
		return pairResult{}, fmt.Errorf("对阵 %s vs %s 不在当前名册", r.a, r.b)
	}
	if ia == ib || r.a == r.b {
		return pairResult{}, fmt.Errorf("解不开报告：明细里有自己打自己：%s", r.a)
	}
	if ia > ib {
		r.a, r.b = r.b, r.a
		r.winsA, r.winsB = r.winsB, r.winsA
	}
	return r, nil
}

func orientAll(kinds []string, results []pairResult) ([]pairResult, error) {
	order := kindOrder(kinds)
	out := make([]pairResult, 0, len(results))
	seen := map[[2]string]bool{}
	for _, r := range results {
		c, err := orient(r, order)
		if err != nil {
			return nil, err
		}
		key := [2]string{c.a, c.b}
		if seen[key] {
			return nil, fmt.Errorf("对阵 %s vs %s 重复", c.a, c.b)
		}
		seen[key] = true
		out = append(out, c)
	}
	if err := rosterComplete(kinds, out); err != nil {
		return nil, err
	}
	return out, nil
}

// mergeLedger 丢掉退册的球和这只的旧对阵，换上新结果。写回前全表必须每一对都在。
func mergeLedger(kinds []string, only string, prior, fresh []pairResult) ([]pairResult, error) {
	order := kindOrder(kinds)
	if _, ok := order[only]; !ok {
		return nil, fmt.Errorf("没有战斗机 %q", only)
	}
	out := make([]pairResult, 0, len(kinds)*(len(kinds)-1)/2)
	seen := map[[2]string]bool{}
	for _, r := range prior {
		if r.a == only || r.b == only {
			continue
		}
		if _, ok := order[r.a]; !ok {
			continue
		}
		if _, ok := order[r.b]; !ok {
			continue
		}
		key := [2]string{r.a, r.b}
		if seen[key] {
			return nil, fmt.Errorf("对阵 %s vs %s 重复", r.a, r.b)
		}
		seen[key] = true
		out = append(out, r)
	}
	for _, r := range fresh {
		c, err := orient(r, order)
		if err != nil {
			return nil, err
		}
		if c.a != only && c.b != only {
			return nil, fmt.Errorf("新对阵 %s vs %s 与 %s 无关", c.a, c.b, only)
		}
		key := [2]string{c.a, c.b}
		if seen[key] {
			return nil, fmt.Errorf("新对阵 %s vs %s 重复", c.a, c.b)
		}
		seen[key] = true
		out = append(out, c)
	}
	if err := rosterComplete(kinds, out); err != nil {
		return nil, err
	}
	return out, nil
}

func complementComplete(kinds []string, only string, pairs []pairResult) error {
	have := map[[2]string]bool{}
	seen := map[string]bool{}
	for _, r := range pairs {
		have[[2]string{r.a, r.b}] = true
		seen[r.a] = true
		seen[r.b] = true
	}
	var noLedger []string
	for _, k := range kinds {
		if !seen[k] {
			noLedger = append(noLedger, k)
		}
	}
	missing := 0
	for i := range kinds {
		if kinds[i] == only {
			continue
		}
		for j := i + 1; j < len(kinds); j++ {
			if kinds[j] == only {
				continue
			}
			if !have[[2]string{kinds[i], kinds[j]}] {
				missing++
			}
		}
	}
	if missing == 0 {
		return nil
	}
	if len(noLedger) >= 2 {
		return fmt.Errorf("名册里这些战斗机还没有旧账：%s。一次多了两只以上，请全量重跑", strings.Join(noLedger, "、"))
	}
	if len(noLedger) == 1 && noLedger[0] != only {
		return fmt.Errorf("只有 -only %s 能补齐这只新球的对阵", noLedger[0])
	}
	return fmt.Errorf("现有明细不是完整全表，缺 %d 对，请全量重跑", missing)
}

func rosterComplete(kinds []string, pairs []pairResult) error {
	have := map[[2]string]bool{}
	for _, r := range pairs {
		have[[2]string{r.a, r.b}] = true
	}
	for i := range kinds {
		for j := i + 1; j < len(kinds); j++ {
			if !have[[2]string{kinds[i], kinds[j]}] {
				return fmt.Errorf("合并后仍缺 %s vs %s，没有写回", kinds[i], kinds[j])
			}
		}
	}
	want := len(kinds) * (len(kinds) - 1) / 2
	if len(have) != want {
		return fmt.Errorf("合并后对阵数是 %d，不是完整全表", len(have))
	}
	return nil
}

// rebuildReport 用现有明细重算总榜和矩阵，丢掉靶子，不重跑对局。报告头的时间、用时和上次增量留着。
func rebuildReport(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(b)
	head := preamble(text)
	n, sec, err := parseHead(head)
	if err != nil {
		return err
	}
	rows, err := parseDetail(text, n)
	if err != nil {
		return err
	}
	kinds := rankedKinds(unit.FighterKinds())
	order := kindOrder(kinds)
	kept := make([]pairResult, 0, len(rows))
	seen := map[[2]string]bool{}
	for _, r := range rows {
		if _, ok := order[r.a]; !ok {
			continue
		}
		if _, ok := order[r.b]; !ok {
			continue
		}
		c, err := orient(r, order)
		if err != nil {
			return err
		}
		key := [2]string{c.a, c.b}
		if seen[key] {
			return fmt.Errorf("对阵 %s vs %s 重复", c.a, c.b)
		}
		seen[key] = true
		kept = append(kept, c)
	}
	if err := rosterComplete(kinds, kept); err != nil {
		return err
	}
	jobs := 1
	if m := reJobs.FindStringSubmatch(head); m != nil {
		jobs, err = strconv.Atoi(m[1])
		if err != nil {
			return err
		}
	}
	var elapsed time.Duration
	if m := reElapsed.FindStringSubmatch(head); m != nil {
		elapsed, err = time.ParseDuration(m[1])
		if err != nil {
			return fmt.Errorf("解不开报告用时 %q", m[1])
		}
	}
	updated := ""
	if m := reUpdated.FindStringSubmatch(head); m != nil {
		updated = strings.TrimSpace(m[1])
	}
	when := ""
	if m := reWhen.FindStringSubmatch(head); m != nil {
		when = strings.TrimSpace(m[1])
	}
	body := render(kinds, kept, n, sec, elapsed, jobs, updated)
	if when != "" {
		lines := strings.Split(body, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "- 时间：") {
				lines[i] = "- 时间：" + when
				break
			}
		}
		body = strings.Join(lines, "\n")
	}
	return writeReport(path, body)
}

func writeReport(path, text string) error {
	dir := filepath.Dir(path)
	if dir == "" {
		dir = "."
	}
	f, err := os.CreateTemp(dir, ".winrate-*.md")
	if err != nil {
		return err
	}
	name := f.Name()
	ok := false
	defer func() {
		if !ok {
			os.Remove(name)
		}
	}()
	if _, err := f.WriteString(text); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, 0o644); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	ok = true
	return nil
}
