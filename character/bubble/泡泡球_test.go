package 泡泡球

import (
	"math"
	"testing"

	"xqdj/internal/unit"
)

func TestEnemyBubbleDoesNotMerge(t *testing.T) {
	st := &unit.MatchState{}
	out := make(chan unit.Cmd, 8)
	me := &泡泡{owner: 1, slot: 0, radius: 20, finalRadius: 20, booted: true, vx: 1}
	ctx := unit.Context{State: st, ID: 10, Kind: KindBubble, Out: out}
	me.remember(ctx)
	otherCtx := unit.Context{State: st, ID: 11, Kind: KindBubble, Out: make(chan unit.Cmd, 1)}
	(&泡泡{owner: 2, slot: 1, radius: 20, finalRadius: 20, booted: true}).remember(otherCtx)

	me.Handle(ctx, unit.Collision{Time: 1, Other: unit.Snapshot{
		ID: 11, Kind: KindBubble, Slot: 1, Radius: 20, OwnerID: 2,
	}})
	if id, ok := despawned(drain(out)); ok {
		t.Fatalf("enemy bubble swallowed %d", id)
	}
}

func TestExactQuarterRadiusOnlyLargerMerges(t *testing.T) {
	st := &unit.MatchState{}
	outSmall := make(chan unit.Cmd, 8)
	outLarge := make(chan unit.Cmd, 16)
	small := &泡泡{owner: 1, slot: 0, radius: 20, finalRadius: 20, booted: true, vx: 1}
	large := &泡泡{owner: 1, slot: 0, radius: 20.25, finalRadius: 20.25, booted: true, vx: 1}
	ctxS := unit.Context{State: st, ID: 11, Kind: KindBubble, Out: outSmall}
	ctxL := unit.Context{State: st, ID: 10, Kind: KindBubble, Out: outLarge}
	small.remember(ctxS)
	large.remember(ctxL)

	small.Handle(ctxS, unit.Collision{Time: 1, Other: unit.Snapshot{
		ID: 10, Kind: KindBubble, Slot: 0, Radius: 20.25, OwnerID: 1,
	}})
	large.Handle(ctxL, unit.Collision{Time: 1, Other: unit.Snapshot{
		ID: 11, Kind: KindBubble, Slot: 0, Radius: 20, OwnerID: 1,
	}})
	if _, ok := despawned(drain(outSmall)); ok {
		t.Fatal("smaller bubble must not swallow the one exactly 0.25 larger")
	}
	if id, ok := despawned(drain(outLarge)); !ok || id != 11 {
		t.Fatalf("larger bubble should swallow the smaller, despawn=%d ok=%v", id, ok)
	}
}

func TestPopOnCenterDoesNotEmitNaN(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	foe := unit.Snapshot{ID: 5, Role: unit.RoleFighter, Slot: 1, X: 40, Y: 40, Radius: 18}
	b := &泡泡{
		owner: 1, slot: 0, radius: 18, x: 40, y: 40,
		ammo:   []bubbleAmmo{{Kind: "火箭", Damage: 3, VX: 10, VY: 0}},
		nearby: []unit.Snapshot{foe},
	}
	ctx := unit.Context{State: &unit.MatchState{}, ID: 10, Kind: KindBubble, Out: out}
	b.hitEnemy(ctx, foe)
	shots := 0
	for _, c := range drain(out) {
		sp, ok := c.(unit.Spawn)
		if !ok || sp.Kind != KindBubbleShot {
			continue
		}
		shots++
		if math.IsNaN(sp.VX) || math.IsNaN(sp.VY) || math.Hypot(sp.VX, sp.VY) < 1 {
			t.Fatalf("shot velocity %+v", sp)
		}
	}
	if shots != 1 {
		t.Fatalf("shots=%d", shots)
	}
}

func despawned(cmds []unit.Cmd) (uint64, bool) {
	for _, c := range cmds {
		if d, ok := c.(unit.Despawn); ok {
			return d.UnitID, true
		}
	}
	return 0, false
}

func drain(out chan unit.Cmd) []unit.Cmd {
	var cmds []unit.Cmd
	for {
		select {
		case c := <-out:
			cmds = append(cmds, c)
		default:
			return cmds
		}
	}
}
