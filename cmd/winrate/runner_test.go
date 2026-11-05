package main

import (
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestRunPairsAccountsForEveryRound(t *testing.T) {
	pairs := []pair{{"甲", "乙"}, {"甲", "丙"}}
	type game struct {
		left, right, winner string
		ticks               int
	}
	want := map[uint64]game{}
	for _, p := range pairs {
		for n, winner := range []string{p.a, p.b, "平局", "", "未知"} {
			left, right := p.a, p.b
			if n%2 == 1 {
				left, right = right, left
			}
			want[mixSeed(left, right, uint64(n)+1)] = game{left, right, winner, 100}
		}
	}
	var mu sync.Mutex
	seen := map[uint64]int{}
	results, done := runPairs(pairs, 5, 100, 3, func(left, right string, seed uint64, maxTicks int) (string, int) {
		mu.Lock()
		defer mu.Unlock()
		g, ok := want[seed]
		if !ok || left != g.left || right != g.right || maxTicks != 100 {
			t.Errorf("unexpected game: %q %q seed=%d timeout=%d", left, right, seed, maxTicks)
		}
		seen[seed]++
		return g.winner, g.ticks
	}, nil)
	if done != 10 {
		t.Fatalf("completed %d games, want 10", done)
	}
	for seed := range want {
		if seen[seed] != 1 {
			t.Errorf("seed %d played %d times", seed, seen[seed])
		}
	}
	wantResults := []pairResult{{"甲", "乙", 1, 1, 3, 1}, {"甲", "丙", 1, 1, 3, 1}}
	if !reflect.DeepEqual(results, wantResults) {
		t.Fatalf("results %+v, want %+v", results, wantResults)
	}
}

func TestRunPairsBoundsActiveGames(t *testing.T) {
	const jobs = 3
	baseline := runtime.NumGoroutine()
	started := make(chan struct{}, jobs+1)
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	finished := make(chan int, 1)
	go func() {
		_, done := runPairs([]pair{{"甲", "乙"}}, 10000, 10, jobs, func(_, _ string, _ uint64, _ int) (string, int) {
			select {
			case started <- struct{}{}:
			default:
			}
			<-release
			return "甲", 1
		}, nil)
		finished <- done
	}()
	for i := 0; i < jobs; i++ {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("workers did not start")
		}
	}
	// 只限制活跃场次的信号量仍会为全部排队场次建协程。
	// 给生产者调度机会，再确认大量待测场次没有变成大量等待协程。
	for i := 0; i < 100; i++ {
		runtime.Gosched()
	}
	if extra := runtime.NumGoroutine() - baseline; extra > jobs+32 {
		t.Fatalf("queued games created %d extra goroutines", extra)
	}
	select {
	case <-started:
		t.Fatal("more games started than workers")
	default:
	}
	unblock()
	select {
	case done := <-finished:
		if done != 10000 {
			t.Fatalf("completed %d games", done)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("workers did not finish")
	}
}
