package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// progressBar 把"已完成多少场"画成一行进度条。一次全量要跑十几分钟，默认
// 就打开——没有进度看不出是在跑还是卡死了。
type progressBar struct {
	out      io.Writer
	total    int
	done     int
	start    time.Time
	last     time.Time
	interval time.Duration
	live     bool // 终端上原地刷新，重定向到文件时按百分比打点
	width    int
	lineGap  time.Duration
	lastLine time.Time
	rows     int // 终端行数；0 表示拿不到，退化成单行原地刷新
}

func newProgress(out io.Writer, total int) *progressBar {
	p := &progressBar{out: out, total: total, start: time.Now(),
		last: time.Now(), interval: 200 * time.Millisecond, width: 28,
		lineGap: 40 * time.Millisecond}
	if f, ok := out.(*os.File); ok {
		if st, err := f.Stat(); err == nil && st.Mode()&os.ModeCharDevice != 0 {
			p.live = true
			if rows, cols := termSize(f); rows >= 4 {
				p.rows = rows
				if cols > 40 && p.width > cols-34 {
					p.width = cols - 34
				}
				// 底部一行固定给进度条，上面 (rows-1) 行做滚动区：日志在
				// 里面自然上滚，抢不到进度条那一行。
				fmt.Fprintf(out, "\x1b[1;%dr", rows-1)
				fmt.Fprintf(out, "\x1b[%d;1H\n", rows-1)
			}
		}
	}
	return p
}

// step 记一场。节流到 5 场/秒刷新一次，免得进度比模拟本身还贵。
func (p *progressBar) step() {
	if p == nil {
		return
	}
	p.done++
	// 第一场也立刻画：不然长时间跑的第一屏是空的。
	if p.done < p.total && p.done > 1 && time.Since(p.last) < p.interval {
		return
	}
	p.draw(false)
}

// report 记一场，并把这一场作为一行滚动日志打在进度条上方。进度条钉在
// 最后一行不跟着滚，终端上看上去就是"上面跑马、底下定住"。
// 一局几百毫秒、32 路并发时每秒能完成几十场，全打会刷爆终端，所以按
// lineGap 抽稀；抽掉的场次不算丢，done 照样涨。
func (p *progressBar) report(text string) {
	if p == nil {
		return
	}
	p.done++
	if !p.live {
		p.draw(false)
		return
	}
	if time.Since(p.lastLine) >= p.lineGap {
		p.lastLine = time.Now()
		fmt.Fprintln(p.out, text)
	}
	p.draw(false)
}

// finish 收尾：补一次满格，并结束原地刷新的那一行。
func (p *progressBar) finish() {
	if p == nil {
		return
	}
	p.draw(true)
}

func (p *progressBar) draw(done bool) {
	if p.total <= 0 {
		return
	}
	// 进度条 5 次/秒就够，每场都跳一次光标反而闪。
	if !done && p.done > 1 && time.Since(p.last) < p.interval {
		return
	}
	p.last = time.Now()
	frac := float64(p.done) / float64(p.total)
	if frac > 1 {
		frac = 1
	}
	fill := int(frac*float64(p.width) + .5)
	bar := strings.Repeat("=", fill) + strings.Repeat("-", p.width-fill)
	elapsed := time.Since(p.start)
	line := fmt.Sprintf("[%s] %5.1f%% %d/%d  %s", bar, frac*100, p.done, p.total, humanDur(elapsed))
	if p.done > 0 && p.done < p.total {
		per := elapsed / time.Duration(p.done)
		line += "  剩 " + humanDur(per*time.Duration(p.total-p.done))
	}
	if p.live {
		if p.rows >= 4 {
			// 存光标 → 跳到最底一行重画 → 把光标还回去，滚动区里的日志
			// 就不会被这一行挤走。
			fmt.Fprintf(p.out, "\x1b7\x1b[%d;1H\r%s\x1b[K\x1b8", p.rows, line)
		} else {
			fmt.Fprint(p.out, "\r"+line+"\x1b[K")
		}
		if done {
			p.closeScreen()
		}
		return
	}
	// 重定向到文件时每 5% 打一行，日志不会被刷爆。
	stride := p.total / 20
	if stride < 1 {
		stride = 1
	}
	if done || p.done == 1 || p.done%stride == 0 {
		fmt.Fprintln(p.out, line)
	}
}

// closeScreen 收尾：把进度条留在最后一行的位置，恢复滚动区并换行，
// 之后的正常输出不会踩到它。
func (p *progressBar) closeScreen() {
	if p.rows >= 4 {
		fmt.Fprintf(p.out, "\x1b[%d;1H\x1b[K\x1b[r\n", p.rows)
		return
	}
	fmt.Fprintln(p.out)
}

func humanDur(d time.Duration) string {
	if d < time.Second {
		return d.Round(time.Millisecond).String()
	}
	d = d.Round(time.Second)
	h := d / time.Hour
	m := (d % time.Hour) / time.Minute
	s := (d % time.Minute) / time.Second
	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	}
	if m == 0 {
		return fmt.Sprintf("%ds", s)
	}
	return fmt.Sprintf("%dm%02ds", m, s)
}
