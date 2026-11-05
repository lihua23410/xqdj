package 炭翁

import (
	"testing"
	"xqdj/internal/unit"
)

func TestEmberLedgersAreMatchLocal(t *testing.T) {
	a, b := &unit.MatchState{}, &unit.MatchState{}
	for i, scope := range []*unit.MatchState{a, b} {
		brain := newBrain()
		brain.owner = 7
		brain.coals = []placed{{holder: 2, value: float64(i + 3), due: 9}}
		stateFor(scope).orphans.Store(uint64(7), brain)
	}
	for i, scope := range []*unit.MatchState{a, b} {
		ctx, out := pipe(scope, 99)
		(&烬{token: 7}).Handle(ctx, sense(9, unit.Snapshot{ID: 99, Role: unit.RoleHelper}, foeAt(0, 0)))
		ds := damages(drain(out))
		if len(ds) != 1 || ds[0].Amount != float64(i+3) {
			t.Fatalf("match %d used another match's ledger: %+v", i, ds)
		}
	}
}
