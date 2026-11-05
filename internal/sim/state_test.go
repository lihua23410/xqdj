package sim

import (
	"testing"
	"time"

	unitpkg "xqdj/internal/unit"
)

const stateProbeKind = "test-match-state-probe"

type stateProbe struct{ birth unitpkg.SpawnInfo }
type inspectState struct{ reply chan unitpkg.Context }

func (a *stateProbe) Handle(ctx unitpkg.Context, ev unitpkg.Event) {
	if inspect, ok := ev.(inspectState); ok {
		inspect.reply <- ctx
	}
}

func init() {
	unitpkg.Register(unitpkg.Spec{Kind: stateProbeKind, Role: unitpkg.RoleHelper}, func(info unitpkg.SpawnInfo) unitpkg.Actor {
		return &stateProbe{birth: info}
	})
}

func TestMatchStateFollowsActorsAndResetsOnlyOwnMatch(t *testing.T) {
	type ledger struct{ value int }
	a, b := NewMatchSeeded(42), NewMatchSeeded(42)
	t.Cleanup(a.End)
	t.Cleanup(b.End)
	var scopes []*unitpkg.MatchState
	for i, m := range []*Match{a, b} {
		m.mu.Lock()
		m.applyCmdLocked(unitpkg.Spawn{Kind: stateProbeKind, Stream: 77, Data: i + 1})
		u := m.units[1]
		if u == nil {
			m.mu.Unlock()
			t.Fatal("probe did not spawn")
		}
		birth := u.actor.(*stateProbe).birth
		reply := make(chan unitpkg.Context, 1)
		m.send(u, inspectState{reply})
		m.mu.Unlock()
		select {
		case ctx := <-reply:
			if ctx.State == nil || ctx.State != birth.State || ctx.State != m.shared || birth.Data != i+1 || ctx.Stream != 77 {
				t.Fatalf("birth/context did not receive this match's state and data: birth=%+v context=%+v", birth, ctx)
			}
			unitpkg.StateFor[ledger](ctx.State).value = i + 1
			scopes = append(scopes, ctx.State)
		case <-time.After(time.Second):
			t.Fatal("actor did not receive inspection")
		}
	}
	if scopes[0] == scopes[1] {
		t.Fatal("same-seed matches shared state")
	}
	a.Start()
	if a.shared == scopes[0] || unitpkg.StateFor[ledger](a.shared).value != 0 {
		t.Fatal("starting a new run retained the old state")
	}
	a.End()
	if b.shared != scopes[1] || unitpkg.StateFor[ledger](b.shared).value != 2 {
		t.Fatal("resetting or ending A changed B")
	}
}
