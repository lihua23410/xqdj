package main

import (
	"sync"

	"xqdj/internal/sim"
)

type pair struct{ a, b string }

type gameRunner func(left, right string, seed uint64, maxTicks int) (winner string, ticks int)

// runPairs 只启动固定数量的工作协程；排队场次不持有协程或 Match。
// 角色仍由 Match 内原有的异步 actor 执行，工作池只限制同时进行的场次数。
func runPairs(pairs []pair, nGames, maxTicks, jobs int, play gameRunner) ([]pairResult, int) {
	results := make([]pairResult, len(pairs))
	for i, p := range pairs {
		results[i] = pairResult{a: p.a, b: p.b}
	}
	total := len(pairs) * nGames
	if total == 0 {
		return results, 0
	}
	if jobs < 1 {
		jobs = 1
	}
	if jobs > total {
		jobs = total
	}
	type task struct{ pair, round int }
	type outcome struct {
		pair   int
		winner string
		ticks  int
	}
	queue := make(chan task, jobs)
	completed := make(chan outcome, jobs)
	var workers sync.WaitGroup
	workers.Add(jobs)
	for i := 0; i < jobs; i++ {
		go func() {
			defer workers.Done()
			for task := range queue {
				p := pairs[task.pair]
				left, right := p.a, p.b
				if task.round%2 == 1 {
					left, right = right, left
				}
				seed := mixSeed(left, right, uint64(task.round)+1)
				winner, ticks := play(left, right, seed, maxTicks)
				completed <- outcome{task.pair, winner, ticks}
			}
		}()
	}
	go func() {
		// 按轮次交错对阵，避免 -n 较大时工作池同时挤满某一组重对局。
		for n := 0; n < nGames; n++ {
			for pi := range pairs {
				queue <- task{pi, n}
			}
		}
		close(queue)
		workers.Wait()
		close(completed)
	}()
	done := 0
	for game := range completed {
		r := &results[game.pair]
		if game.ticks >= maxTicks && game.winner == "平局" {
			r.timeout++
		}
		switch game.winner {
		case r.a:
			r.winsA++
		case r.b:
			r.winsB++
		default:
			r.draws++
		}
		done++
	}
	return results, done
}

func playMatch(left, right string, seed uint64, maxTicks int) (string, int) {
	m := sim.NewHeadlessMatchSeeded(seed)
	m.SetSlot(0, left)
	m.SetSlot(1, right)
	m.Start()
	winner, ticks := m.Play(maxTicks)
	m.End()
	return winner, ticks
}
