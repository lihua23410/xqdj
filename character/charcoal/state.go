package 炭翁

import (
	"sync"

	"xqdj/internal/unit"
)

type matchState struct {
	orphans sync.Map
}

func stateFor(scope *unit.MatchState) *matchState {
	s := unit.StateFor[matchState](scope)
	return s
}
