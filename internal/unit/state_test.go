package unit

import (
	"sync"
	"testing"
)

func TestStateForIsLocalToMatchAndType(t *testing.T) {
	type first struct{ value int }
	type second struct{ value int }
	a, b := &MatchState{}, &MatchState{}
	StateFor[first](a).value = 7
	StateFor[first](b).value = 9
	StateFor[second](a).value = 11
	if StateFor[first](a).value != 7 || StateFor[first](b).value != 9 || StateFor[second](a).value != 11 {
		t.Fatal("state crossed match or pack type boundaries")
	}
	if StateFor[first](a) != StateFor[first](a) {
		t.Fatal("actors in one match did not share their pack state")
	}
}

func TestStateForConcurrentInitialization(t *testing.T) {
	type ledger struct {
		mu    sync.Mutex
		count int
	}
	scope := &MatchState{}
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s := StateFor[ledger](scope)
			s.mu.Lock()
			s.count++
			s.mu.Unlock()
		}()
	}
	wg.Wait()
	if got := StateFor[ledger](scope).count; got != 32 {
		t.Fatalf("concurrent initialization lost writes: %d", got)
	}
}
