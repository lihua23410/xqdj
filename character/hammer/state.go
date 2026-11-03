package 钉与锤

import (
	"sync"

	"xqdj/internal/unit"
)

type matchState struct {
	once    sync.Once
	foeMu   sync.Mutex
	foeOf   map[uint64]uint64
	dollHit map[uint64]map[uint64]bool // 这次贴住已经打过的人偶
	xfer    map[uint64]float64         // 等本体在感知里转出的伤
	xferAt  map[uint64]float64         // 下次允许转伤的时刻（替身转伤有 CD）
}

func stateFor(scope *unit.MatchState) *matchState {
	s := unit.StateFor[matchState](scope)
	s.once.Do(func() {
		s.foeOf = map[uint64]uint64{}
		s.dollHit = map[uint64]map[uint64]bool{}
		s.xfer = map[uint64]float64{}
		s.xferAt = map[uint64]float64{}
	})
	return s
}
