package 收割者

import (
	"sync"

	"xqdj/internal/unit"
)

type matchState struct {
	recallingIDs sync.Map
}

func stateFor(scope *unit.MatchState) *matchState {
	s := unit.StateFor[matchState](scope)
	return s
}
