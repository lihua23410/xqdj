package 血契

import (
	"math"
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

func TestSameFrameReelAndBatsStayAboveFuse(t *testing.T) {
	// 血量落在 (10, 11]：收线 8 记账之后，召蝠不能再按原始血量烧 3。
	cmds := sensePact(t, 11, true)
	hp, ok := setHPOf(cmds)
	if !ok || math.Abs(hp-(11-reelCost)) > 1e-9 {
		t.Fatalf("应只收线，HP=%v ok=%v", hp, ok)
	}
	if bornBats(cmds) != 0 {
		t.Fatal("熔断后不该同帧召蝠")
	}

	// 扣完收线仍高于熔断线，两笔都该烧。
	cmds = sensePact(t, blackoutHP+reelCost+batCost+1, true)
	hp, ok = setHPOf(cmds)
	want := blackoutHP + 1
	if !ok || math.Abs(hp-want) > 1e-9 || bornBats(cmds) != 1 {
		t.Fatalf("收线后仍够血，应再召蝠：HP=%v bats=%d", hp, bornBats(cmds))
	}

	cmds = sensePact(t, blackoutHP, true)
	if _, ok := setHPOf(cmds); ok || bornBats(cmds) != 0 {
		t.Fatal("已在熔断线上不该烧血")
	}
}

func sensePact(t *testing.T, hp float64, lined bool) []unit.Cmd {
	t.Helper()
	out := make(chan unit.Cmd, 32)
	b := &血契{booted: true, lines: map[uint64]*bloodLine{}}
	if lined {
		b.lines[9] = &bloodLine{lastHP: 80, lastMax: 100}
	}
	ctx := unit.Context{ID: 3, Kind: KindBloodPact, Stream: 1, Out: out}
	self := unit.Snapshot{
		ID: 3, Kind: KindBloodPact, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: pactRadius, HP: hp, MaxHP: 100,
	}
	foe := unit.Snapshot{
		ID: 9, Role: unit.RoleFighter, Slot: 1, HP: 80, MaxHP: 100,
		X: reelSaveFrom + 40, Y: 0, Radius: 18, AimPriority: unit.DefaultFighterAim,
	}
	b.Handle(ctx, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{foe}})
	return drainBat(out)
}

func setHPOf(cmds []unit.Cmd) (float64, bool) {
	for i := len(cmds) - 1; i >= 0; i-- {
		if c, ok := cmds[i].(unit.SetHP); ok && c.UnitID == 3 {
			return c.HP, true
		}
	}
	return 0, false
}

func bornBats(cmds []unit.Cmd) int {
	n := 0
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == KindBloodBat {
			n++
		}
	}
	return n
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
