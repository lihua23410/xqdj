package 无下限术士

import (
	"testing"
	"xqdj/internal/unit"
)

func TestBootHidesAndSplits(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &无下限术士{slot: 0}
	a.Handle(unit.Context{ID: 1, Kind: KindTwin, Out: out}, unit.Sense{
		Time: 0,
		Self: unit.Snapshot{ID: 1, Slot: 0, X: 10, Y: 0, VX: 200, VY: 0},
	})
	cmds := drain(out)
	if !hasNoHealthNumbers(cmds, 1) {
		t.Fatalf("missing NoHealthNumbers: %v", cmds)
	}
	if !hasAim(cmds, 1, 0) {
		t.Fatalf("missing SetAim 0: %v", cmds)
	}
	if !hasSpawn(cmds, KindRed) || !hasSpawn(cmds, KindBlue) {
		t.Fatalf("missing halves: %v", cmds)
	}
}

func TestHalfForwardsDamageToOwner(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	h := &半{owner: 1, slot: 0, red: true}
	h.Handle(unit.Context{ID: 2, Kind: KindRed, Out: out}, unit.IncomingDamage{
		Token: 9, From: 99, Amount: 8,
	})
	cmds := drain(out)
	if !hasConfirm(cmds, 9, 2, 8) {
		t.Fatalf("missing confirm: %v", cmds)
	}
	if !hasDamage(cmds, 99, 1, 8) {
		t.Fatalf("forward From should stay attacker: %v", cmds)
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

func hasNoHealthNumbers(cmds []unit.Cmd, id uint64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.NoHealthNumbers); ok && v.UnitID == id && v.Hold {
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

func hasConfirm(cmds []unit.Cmd, token, id uint64, amt float64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.ConfirmDamage); ok && v.Token == token && v.UnitID == id && v.Amount == amt {
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
