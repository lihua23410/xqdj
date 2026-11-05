package 血契

import (
	"testing"

	"xqdj/internal/unit"
)

func TestPounceStrikeOnceAndTethers(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	b := &血契{
		slot:        0,
		x:           0,
		y:           0,
		pounceUntil: 1,
		lines:       map[uint64]*bloodLine{},
	}
	ctx := unit.Context{ID: 3, Kind: KindBloodPact, Out: out}
	foe := unit.Snapshot{ID: 9, Role: unit.RoleFighter, Slot: 1, HP: 80, MaxHP: 100, X: 40, Y: 0}
	b.Handle(ctx, unit.FanHit{Kind: KindPounceArc, Time: 0.2, Other: foe})
	b.Handle(ctx, unit.FanHit{Kind: KindPounceArc, Time: 0.3, Other: foe})

	dmg, bond := 0, 0
	for _, c := range drainBat(out) {
		if d, ok := c.(unit.Damage); ok && d.To == foe.ID {
			dmg++
			if d.Amount != biteDamage {
				t.Fatalf("扑中该结咬合伤害 %v，实际 %v", biteDamage, d.Amount)
			}
		}
		if fx, ok := c.(unit.FX); ok && fx.Name == "blood-bond" {
			bond++
		}
	}
	if dmg != 0 {
		t.Fatalf("主人不应重复发出饿扑伤害，实际 %d", dmg)
	}
	if bond != 1 {
		t.Fatalf("扑中该当拍建线，实际 %d", bond)
	}
	if b.lines[foe.ID] == nil {
		t.Fatal("血线该记在这一扑上")
	}
}

func TestPounceIgnoresHitAfterDash(t *testing.T) {
	out := make(chan unit.Cmd, 4)
	b := &血契{slot: 0, pounceUntil: 0.5, lines: map[uint64]*bloodLine{}}
	ctx := unit.Context{ID: 3, Kind: KindBloodPact, Out: out}
	foe := unit.Snapshot{ID: 9, Role: unit.RoleFighter, Slot: 1, HP: 80, MaxHP: 100}
	b.Handle(ctx, unit.FanHit{Kind: KindPounceArc, Time: 0.8, Other: foe})
	for _, c := range drainBat(out) {
		if _, ok := c.(unit.Damage); ok {
			t.Fatal("扑完之后的碰撞不该再结饿扑的伤")
		}
	}
}
