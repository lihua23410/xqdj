package 蛆

import (
	"testing"

	"xqdj/internal/unit"
)

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

func TestBootHidesAimAndSpawnsBabies(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &蛆{}
	a.Handle(unit.Context{ID: 1, Kind: KindMaggot, Out: out}, unit.Sense{
		Time: 0,
		Self: unit.Snapshot{ID: 1, Slot: 0, X: 0, Y: 0, HP: bodyHP, MaxHP: bodyHP},
	})
	cmds := drain(out)
	if !hasNoHealthNumbers(cmds, 1) {
		t.Fatalf("missing NoHealthNumbers: %v", cmds)
	}
	if !hasNoFrameFreeze(cmds, 1) {
		t.Fatalf("missing NoFrameFreeze: %v", cmds)
	}
	if !hasAim(cmds, 1, 0) {
		t.Fatalf("missing SetAim 0: %v", cmds)
	}
	n := 0
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == KindBaby {
			n++
		}
	}
	if n != bootBabies {
		t.Fatalf("want %d babies, got %d: %v", bootBabies, n, cmds)
	}
}

func TestKeepAliveKillsAfterGrace(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &蛆{
		booted:     true,
		awaitSpawn: false,
		orphanFrom: -1,
	}
	ctx := unit.Context{ID: 1, Kind: KindMaggot, Out: out}
	a.Handle(ctx, unit.Sense{
		Time: 1,
		Self: unit.Snapshot{ID: 1, Slot: 0, HP: bodyHP, MaxHP: bodyHP},
	})
	_ = drain(out)
	if a.orphanFrom < 0 {
		t.Fatalf("orphan timer should start")
	}
	a.Handle(ctx, unit.Sense{
		Time: a.orphanFrom + keepAliveGrace,
		Self: unit.Snapshot{ID: 1, Slot: 0, HP: bodyHP, MaxHP: bodyHP},
	})
	cmds := drain(out)
	if !hasDamage(cmds, 1, 1, bodyHP) {
		t.Fatalf("want self lethal damage: %v", cmds)
	}
}

func TestAuraWaitsThenTicks(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &蛆{
		booted:     true,
		awaitSpawn: false,
		orphanFrom: -1,
	}
	flies := []unit.Snapshot{
		{ID: 10, Kind: KindFly, OwnerID: 1, Slot: 0, X: 0, Y: 0, Radius: flyRadius, Mortal: true},
		{ID: 11, Kind: KindFly, OwnerID: 1, Slot: 0, X: 10, Y: 0, Radius: flyRadius, Mortal: true},
		{ID: 12, Kind: KindFly, OwnerID: 1, Slot: 0, X: 20, Y: 0, Radius: flyRadius, Mortal: true},
	}
	enemy := unit.Snapshot{ID: 99, Kind: "敌", Role: unit.RoleFighter, Slot: 1, X: 50, Y: 0, Radius: 18}
	nearby := append(flies, enemy)
	ctx := unit.Context{ID: 1, Kind: KindMaggot, Out: out}
	a.Handle(ctx, unit.Sense{
		Time: 0,
		Self: unit.Snapshot{ID: 1, Slot: 0, HP: bodyHP, MaxHP: bodyHP},
		Nearby: nearby,
	})
	cmds := drain(out)
	if hasDamageTo(cmds, 99) {
		t.Fatalf("aura should wait first gap: %v", cmds)
	}
	a.Handle(ctx, unit.Sense{
		Time: 0.5,
		Self: unit.Snapshot{ID: 1, Slot: 0, HP: bodyHP, MaxHP: bodyHP},
		Nearby: nearby,
	})
	cmds = drain(out)
	if hasDamageTo(cmds, 99) {
		t.Fatalf("aura should still be waiting at 0.5s: %v", cmds)
	}
	a.Handle(ctx, unit.Sense{
		Time: auraGap,
		Self: unit.Snapshot{ID: 1, Slot: 0, HP: bodyHP, MaxHP: bodyHP},
		Nearby: nearby,
	})
	cmds = drain(out)
	if !hasDamage(cmds, 1, 99, auraPerLayer) {
		t.Fatalf("want 1 aura damage: %v", cmds)
	}
}

func TestFlyPairLaysEggAfterOverlap(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &蛆{
		booted:      true,
		awaitSpawn:  false,
		orphanFrom:  -1,
		breedUntil:  map[uint64]float64{},
		overlapFrom: map[flyPair]float64{},
		flyPassing:  map[uint64]bool{},
	}
	ctx := unit.Context{ID: 1, Kind: KindMaggot, Out: out}
	nearby := []unit.Snapshot{
		{ID: 10, Kind: KindFly, OwnerID: 1, Slot: 0, X: 0, Y: 0, Radius: flyRadius, Mortal: true},
		{ID: 11, Kind: KindFly, OwnerID: 1, Slot: 0, X: 10, Y: 0, Radius: flyRadius, Mortal: true},
	}
	a.Handle(ctx, unit.Sense{
		Time: 0,
		Self: unit.Snapshot{ID: 1, Slot: 0, HP: bodyHP, MaxHP: bodyHP},
		Nearby: nearby,
	})
	_ = drain(out)
	if _, ok := a.overlapFrom[canonPair(10, 11)]; !ok {
		t.Fatalf("overlap should start when dist<=24")
	}
	a.Handle(ctx, unit.Sense{
		Time: overlapNeed,
		Self: unit.Snapshot{ID: 1, Slot: 0, HP: bodyHP, MaxHP: bodyHP},
		Nearby: nearby,
	})
	cmds := drain(out)
	if !hasSpawn(cmds, KindEgg) {
		t.Fatalf("missing egg spawn: %v", cmds)
	}
	if a.breedUntil[10] <= 0 || a.breedUntil[11] <= 0 {
		t.Fatalf("both flies should enter breed CD: %v", a.breedUntil)
	}
}

func hasNoHealthNumbers(cmds []unit.Cmd, id uint64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.NoHealthNumbers); ok && v.UnitID == id && v.Hold {
			return true
		}
	}
	return false
}

func hasNoFrameFreeze(cmds []unit.Cmd, id uint64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.NoFrameFreeze); ok && v.UnitID == id && v.Hold {
			return true
		}
	}
	return false
}

func hasAim(cmds []unit.Cmd, id uint64, v uint8) bool {
	for _, c := range cmds {
		if s, ok := c.(unit.SetAimPriority); ok && s.UnitID == id && s.Value == v {
			return true
		}
	}
	return false
}

func hasSpawn(cmds []unit.Cmd, kind string) bool {
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == kind {
			return true
		}
	}
	return false
}

func hasDamage(cmds []unit.Cmd, from, to uint64, amt float64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.Damage); ok && v.From == from && v.To == to && v.Amount == amt {
			return true
		}
	}
	return false
}

func hasDamageTo(cmds []unit.Cmd, to uint64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.Damage); ok && v.To == to {
			return true
		}
	}
	return false
}
