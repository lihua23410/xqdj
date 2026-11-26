package 勇者

import (
	"testing"

	"xqdj/internal/unit"
	_ "xqdj/map/hex"
)

func TestPickupLeavesEnemyRelics(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	h := &勇者{x: 0, y: 0, r: heroRadius, slot: 0}
	ctx := unit.Context{ID: 1, Kind: KindHero, Out: out}
	own := unit.Snapshot{ID: 8, Kind: KindSwordPick, OwnerID: 1, Slot: 0, X: 0, Y: 0, Radius: relicRadius}
	foe := unit.Snapshot{ID: 9, Kind: KindWarriorPick, OwnerID: 2, Slot: 1, X: 0, Y: 0, Radius: relicRadius}
	h.pickup(ctx, unit.Sense{Self: unit.Snapshot{ID: 1, Slot: 0}, Nearby: []unit.Snapshot{foe, own}})
	if !h.hasSword || h.hasWarrior {
		t.Fatalf("sword=%v warrior=%v", h.hasSword, h.hasWarrior)
	}
	gone := despawned(drain(out))
	if !gone[8] || gone[9] {
		t.Fatalf("despawned=%v", gone)
	}
}

func TestSlimeCapCountsOnlyOwn(t *testing.T) {
	field := unit.HexField()
	foe := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 200, Y: 0, Radius: 18}
	var theirs []unit.Snapshot
	for id := uint64(10); id < 14; id++ {
		theirs = append(theirs, unit.Snapshot{ID: id, Kind: KindSlime, OwnerID: 2, Slot: 0, Role: unit.RoleMinion})
	}
	out := make(chan unit.Cmd, 8)
	h := &勇者{x: 10000, y: 10000, r: heroRadius, slot: 0, rng: unit.NewRNG(1)}
	ctx := unit.Context{ID: 1, Kind: KindHero, Stream: 1, Out: out}
	h.tickSlimes(ctx, unit.Sense{Time: 1, Field: field, Nearby: append([]unit.Snapshot{foe}, theirs...)})
	if bornSlimes(drain(out)) != 1 {
		t.Fatal("对方满 4 只时自己仍该能召")
	}

	var mine []unit.Snapshot
	for id := uint64(20); id < 24; id++ {
		mine = append(mine, unit.Snapshot{ID: id, Kind: KindSlime, OwnerID: 1, Slot: 1, Role: unit.RoleMinion})
	}
	h.slimeReadyAt = 0
	h.tickSlimes(ctx, unit.Sense{Time: 10, Field: field, Nearby: append([]unit.Snapshot{foe}, mine...)})
	if bornSlimes(drain(out)) != 0 {
		t.Fatal("自己满 4 只不该再召")
	}
}

func despawned(cmds []unit.Cmd) map[uint64]bool {
	got := map[uint64]bool{}
	for _, c := range cmds {
		if d, ok := c.(unit.Despawn); ok {
			got[d.UnitID] = true
		}
	}
	return got
}

func bornSlimes(cmds []unit.Cmd) int {
	n := 0
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == KindSlime {
			n++
		}
	}
	return n
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
