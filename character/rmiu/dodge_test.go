package r缪

import (
	"testing"
	"xqdj/internal/unit"
)

func TestDodgeChance(t *testing.T) {
	if dodgeChance(170, 170) != 0 {
		t.Fatal("equal speed must not dodge")
	}
	if dodgeChance(170, 220) != 0 {
		t.Fatal("slower must not dodge")
	}
	p := dodgeChance(220, 170)
	want := (50.0 / dodgeSpeedDiv) / 100.0
	if mathAbs(p-want) > 1e-12 {
		t.Fatalf("p=%v want %v", p, want)
	}
}

func TestDodgeBlocksHit(t *testing.T) {
	old := miuDodgeRoll
	miuDodgeRoll = func() float64 { return 0 }
	defer func() { miuDodgeRoll = old }()

	out := make(chan unit.Cmd, 8)
	m := meleeMiu()
	m.seenSp = map[uint64]float64{99: 170}
	m.x, m.y = 1, 2
	ctx := unit.Context{ID: 10, Kind: KindMiu, Out: out}
	m.Handle(ctx, unit.IncomingDamage{Token: 4, From: 99, Amount: 9, Speed: 270})
	cmds := drain(out)
	blocked := false
	for _, c := range cmds {
		if v, ok := c.(unit.BlockDamage); ok && v.Token == 4 {
			blocked = true
		}
		if _, ok := c.(unit.ConfirmDamage); ok {
			t.Fatal("dodge must not confirm")
		}
	}
	if !blocked {
		t.Fatal("expected BlockHit")
	}
}

func TestNoDodgeWhenSlower(t *testing.T) {
	old := miuDodgeRoll
	miuDodgeRoll = func() float64 { return 0 }
	defer func() { miuDodgeRoll = old }()

	out := make(chan unit.Cmd, 8)
	m := meleeMiu()
	m.seenSp = map[uint64]float64{99: 400}
	ctx := unit.Context{ID: 10, Kind: KindMiu, Out: out}
	m.Handle(ctx, unit.IncomingDamage{Token: 5, From: 99, Amount: 9, Speed: 170})
	ok := false
	for _, c := range drain(out) {
		if v, okc := c.(unit.ConfirmDamage); okc && v.Token == 5 {
			ok = true
		}
		if _, isB := c.(unit.BlockDamage); isB {
			t.Fatal("slower cannot dodge")
		}
	}
	if !ok {
		t.Fatal("should ConfirmHit")
	}
}