package 中转站

import (
	"sync"

	"xqdj/internal/unit"
)

type matchState struct {
	passMu      sync.Mutex
	passPending []passInfo
	liveShots   map[int]int
}

func stateFor(scope *unit.MatchState) *matchState {
	s := unit.StateFor[matchState](scope)
	return s
}
