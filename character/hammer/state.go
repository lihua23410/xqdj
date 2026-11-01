package 钉与锤

import (
	"sync"

	"xqdj/internal/unit"
)

type matchState struct {
	once  sync.Once
	foeMu sync.Mutex
	foeOf map[uint64]uint64
}

func stateFor(scope *unit.MatchState) *matchState {
	s := unit.StateFor[matchState](scope)
	s.once.Do(func() {
		s.foeOf = map[uint64]uint64{}
	})
	return s
}
