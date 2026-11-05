package 分身者_百战

import (
	"testing"
	"xqdj/internal/unit"
)

func TestShadowGuardsAndDashesStayInTheirMatch(t *testing.T) {
	a, b := &unit.MatchState{}, &unit.MatchState{}
	setStackGuard(a, 42, true)
	startDash(a, 42, 1, 0, 640, 100, 8, true, true, 0)
	startDash(b, 42, 0, 1, 320, 50, 3, false, false, 1)
	if getDash(a, 42).dmg != 8 || getDash(b, 42).dmg != 3 {
		t.Fatal("same-ID dash jobs crossed matches")
	}
	out := make(chan unit.Cmd, 8)
	shadow := &战影{}
	shadow.Handle(unit.Context{State: b, ID: 42, Out: out}, unit.IncomingDamage{Token: 1, Amount: 10})
	if !hasDamageConfirmation(drain(out)) {
		t.Fatal("A's guard or dash blocked damage in B")
	}
	(&百战{}).beginReform(unit.Context{State: b, ID: 1, Out: out}, 3, []unit.Snapshot{{ID: 42, Kind: KindShadow, OwnerID: 1}})
	if !isStackGuarded(a, 42) {
		t.Fatal("B's reform cleared A's guard")
	}
	clearDash(b, 42)
	if getDash(a, 42) == nil {
		t.Fatal("B's dash cleanup removed A's dash")
	}
}

func hasDamageConfirmation(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if _, ok := c.(unit.ConfirmDamage); ok {
			return true
		}
	}
	return false
}
