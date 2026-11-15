package hff

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"xqdj/internal/unit"
)

// Stock 是胜率表里的一只可操盘战斗机。Rate 是胜 /（胜+负）。
type Stock struct {
	Name string
	Rate float64
}

var (
	ratesMu       sync.Mutex
	ratesOnce     sync.Once
	rates         map[string]float64
	ratesOverride map[string]float64
)

// SetRatesForTest 用一张表代替开局读到的单球胜率。nil 恢复读文件。
func SetRatesForTest(m map[string]float64) {
	ratesMu.Lock()
	ratesOverride = m
	ratesMu.Unlock()
}

func ratesTable() map[string]float64 {
	ratesMu.Lock()
	ov := ratesOverride
	ratesMu.Unlock()
	if ov != nil {
		return ov
	}
	ratesOnce.Do(func() {
		rates = ParseSolo(string(readWinrate()))
	})
	return rates
}

// Pool 是盯盘能抽到的股票。排除她自己、非因果，以及表里有但这场没有的名字。
func Pool() []Stock {
	table := ratesTable()
	out := make([]Stock, 0, len(table))
	for name, rate := range table {
		if name == KindHFF || name == "非因果" {
			continue
		}
		spec, ok := unit.Lookup(name)
		if !ok || !spec.Fighter {
			continue
		}
		out = append(out, Stock{Name: name, Rate: rate})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ParseSolo 读单球胜率表。胜率取胜和负两列，平不计入。
func ParseSolo(text string) map[string]float64 {
	out := map[string]float64{}
	in := false
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "## ") {
			if in {
				break
			}
			if strings.Contains(line, "单球胜率") {
				in = true
			}
			continue
		}
		if !in || !strings.HasPrefix(line, "|") {
			continue
		}
		cols := strings.Split(line, "|")
		if len(cols) < 9 {
			continue
		}
		name := strings.TrimSpace(cols[3])
		if name == "" || name == "战斗机" || strings.Contains(cols[1], "---") {
			continue
		}
		wins, err1 := strconv.Atoi(strings.TrimSpace(cols[6]))
		losses, err2 := strconv.Atoi(strings.TrimSpace(cols[7]))
		if err1 != nil || err2 != nil || wins+losses == 0 {
			continue
		}
		out[name] = float64(wins) / float64(wins+losses)
	}
	return out
}

func readWinrate() []byte {
	dir, err := os.Getwd()
	if err != nil {
		return nil
	}
	for i := 0; i < 8; i++ {
		b, err := os.ReadFile(filepath.Join(dir, "winrate.md"))
		if err == nil {
			return b
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil
}
