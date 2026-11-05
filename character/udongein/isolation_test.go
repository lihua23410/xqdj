package 人偶使

import (
	"testing"
	"xqdj/internal/unit"
)

func TestDollOrdersAndOwnersStayInTheirMatch(t *testing.T) {
	a, b := &unit.MatchState{}, &unit.MatchState{}
	ownerA, ownerB := &人偶使{}, &人偶使{}
	stateFor(a).owners.Store(uint64(1), ownerA)
	stateFor(b).owners.Store(uint64(1), ownerB)
	setState(a, 9, dollIdle)
	setState(b, 9, dollShoot)
	orderRecall(a, 9)
	orderChase(a, 9, 1, 3, 100, 200)
	if ownerOf(a, 1) != ownerA || ownerOf(b, 1) != ownerB || stateOf(a, 9) != dollIdle || stateOf(b, 9) != dollShoot {
		t.Fatal("same-ID owners or doll states crossed matches")
	}
	if takeRecall(b, 9) {
		t.Fatal("B took A's recall order")
	}
	if _, _, _, _, ok := chaseWindow(b, 9); ok {
		t.Fatal("B took A's chase order")
	}
	dropState(b, 9)
	if !takeRecall(a, 9) {
		t.Fatal("B's cleanup removed A's order")
	}
}

func TestDollBirthValuesFollowTheirOwnSpawn(t *testing.T) {
	makeSpawn := func(spec dollSpec) unit.Spawn {
		out := make(chan unit.Cmd, 1)
		spawnDollAt(unit.Context{ID: 1, Out: out}, unit.Sense{}, 0, 0, spec)
		return (<-out).(unit.Spawn)
	}
	a := makeSpawn(dollSpec{mode: dollIdle})
	b := makeSpawn(dollSpec{mode: dollStrike, shots: 7})
	dollB := newDoll(unit.SpawnInfo{Data: b.Data})
	dollA := newDoll(unit.SpawnInfo{Data: a.Data})
	if dollA.spec.mode != dollIdle || dollB.spec.mode != dollStrike || dollB.left != 7 {
		t.Fatalf("interleaved births mixed their payloads: A=%+v B=%+v", dollA.spec, dollB.spec)
	}
}
