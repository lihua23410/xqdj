package Freischutz

import (
	"sync"

	"xqdj/internal/unit"
)

type matchState struct {
	liveMu    sync.Mutex
	liveShots map[int]int
	reprises  []reprise
}

func stateFor(scope *unit.MatchState) *matchState {
	s := unit.StateFor[matchState](scope)
	return s
}
