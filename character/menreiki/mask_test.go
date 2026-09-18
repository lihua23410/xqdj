package 面灵气

import (
	"testing"
	"xqdj/internal/unit"
)

func TestMaskOrbitHitsMortalMinion(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	m := &面灵气{hook: 1}
	ctx := unit.Context{ID: 1, Kind: KindMenreiki, Out: out}
	orbit := menreikiRadius + maskRadius + maskOrbitPad
	self := unit.Snapshot{
		ID: 1, Kind: KindMenreiki, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: menreikiRadius, Faction: unit.FactionCyan,
	}
	minion := unit.Snapshot{
		ID: 8, Kind: "教父暗杀者", Role: unit.RoleMinion, Mortal: true, Slot: 1,
		X: orbit, Y: 0, Radius: 14, HP: 30, MaxHP: 30,
	}
	m.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{minion}})
	if !hasDamageTo(drain(out), 8) {
		t.Fatal("mask overlap should damage mortal minion")
	}
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

func hasDamageTo(cmds []unit.Cmd, to uint64) bool {
	for _, c := range cmds {
		if d, ok := c.(unit.Damage); ok && d.To == to {
			return true
		}
	}
	return false
}

func TestMaskHitGateIndependent(t *testing.T) {
	const owner uint64 = 7
	if spendMask(owner, 0) {
		t.Fatal("unarmed should not hit")
	}
	armMask(owner, 0)
	if !spendMask(owner, 0) {
		t.Fatal("armed mask 0 should hit")
	}
	if spendMask(owner, 0) {
		t.Fatal("spent mask 0 should not hit again")
	}
	if spendMask(owner, 1) {
		t.Fatal("mask 1 should stay independent")
	}
	armAllMasks(owner)
	for i := 0; i < 3; i++ {
		if !spendMask(owner, i) {
			t.Fatalf("arm all: mask %d", i)
		}
	}
}
