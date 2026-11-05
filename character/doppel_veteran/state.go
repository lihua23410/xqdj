package 分身者_百战

import (
	"sync"

	"xqdj/internal/unit"
)

type matchState struct {
	dashJobs   sync.Map
	stackGuard sync.Map
}

func stateFor(scope *unit.MatchState) *matchState {
	s := unit.StateFor[matchState](scope)
	return s
}
