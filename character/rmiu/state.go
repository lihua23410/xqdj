package r缪

import (
	"sync"

	"xqdj/internal/unit"
)

type matchState struct {
	once            sync.Once
	miuMu           sync.Mutex
	miuBonus        map[uint64]float64
	miuLastAttacker map[uint64]uint64
	miuClaimed      map[uint64]bool
}

func stateFor(scope *unit.MatchState) *matchState {
	s := unit.StateFor[matchState](scope)
	s.once.Do(func() {
		s.miuBonus = map[uint64]float64{}
		s.miuLastAttacker = map[uint64]uint64{}
		s.miuClaimed = map[uint64]bool{}
	})
	return s
}
