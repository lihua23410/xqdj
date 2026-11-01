package 面灵气

import (
	"sync"

	"xqdj/internal/unit"
)

type matchState struct {
	once        sync.Once
	liveMu      sync.Mutex
	liveFaction map[uint64]string
	maskReady   map[uint64][3]bool
}

func stateFor(scope *unit.MatchState) *matchState {
	s := unit.StateFor[matchState](scope)
	s.once.Do(func() {
		s.liveFaction = map[uint64]string{}
		s.maskReady = map[uint64][3]bool{}
	})
	return s
}
