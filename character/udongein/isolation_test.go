package 人偶使

import (
	"sync"
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

func TestAbortGenAndRNGDoNotRace(t *testing.T) {
	scope := &unit.MatchState{}
	ownerOut := make(chan unit.Cmd, 512)
	a := fighter(SkillNone)
	ctx := unit.Context{State: scope, ID: 1, Kind: KindNingyushi, Out: ownerOut, Stream: 3}
	a.ensureRNG(ctx)
	a.cardMu.Lock()
	a.initDeckLocked()
	a.cardMu.Unlock()
	stateFor(scope).owners.Store(uint64(1), a)

	dollOut := make(chan unit.Cmd, 1024)
	d := newDoll(unit.SpawnInfo{OwnerID: 1, Slot: 0, Data: dollSpec{mode: dollStrike, follow: true, bornGen: 0}})
	dctx := unit.Context{State: scope, ID: 9, Kind: KindNingyushiDoll, Out: dollOut}
	ownerSnap := unit.Snapshot{ID: 1, X: 0, Y: 0}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 300; i++ {
			a.job.kind = SkillN24
			a.endRush(ctx, float64(i))
			_ = a.pickCast(unit.Sense{Time: float64(i), Self: me(0, 0)}, nil)
			_, _ = randDir(a)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 300; i++ {
			d.Handle(dctx, unit.Sense{
				Time:   float64(i) / 60,
				Self:   unit.Snapshot{ID: 9, X: 30, Y: 0, Radius: dollRadius},
				Nearby: []unit.Snapshot{ownerSnap},
			})
			noteDealt(scope, 1)
		}
	}()
	wg.Wait()
	if a.abortGen.Load() == 0 {
		t.Fatal("rush should advance the abort generation")
	}
}
