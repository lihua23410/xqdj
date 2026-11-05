package 人偶使

import (
	"sync"

	"xqdj/internal/unit"
)

type matchState struct {
	once     sync.Once
	stateMu  sync.Mutex
	dollSt   map[uint64]uint8
	chaseAt  map[uint64]chaseMark
	recallID map[uint64]bool
	owners   sync.Map
}

func stateFor(scope *unit.MatchState) *matchState {
	s := unit.StateFor[matchState](scope)
	s.once.Do(func() {
		s.dollSt = map[uint64]uint8{}
		s.chaseAt = map[uint64]chaseMark{}
		s.recallID = map[uint64]bool{}
	})
	return s
}
