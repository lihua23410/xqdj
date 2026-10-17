package 谭泳辰

import (
	"testing"

	"xqdj/internal/unit"
)

func tanSelf(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindTan, Role: unit.RoleFighter, Slot: 0,
		X: x, Y: y, Radius: tanRadius, Vision: tanVision, AimPriority: unit.DefaultFighterAim,
	}
}

func drain(out <-chan unit.Cmd) []unit.Cmd {
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

func hasDespawn(cmds []unit.Cmd, id uint64) bool {
	for _, c := range cmds {
		if d, ok := c.(unit.Despawn); ok && d.UnitID == id {
			return true
		}
	}
	return false
}

func sensePickup(t *谭泳辰老先生, self unit.Snapshot, near []unit.Snapshot) []unit.Cmd {
	out := make(chan unit.Cmd, 16)
	ctx := unit.Context{ID: 1, Kind: KindTan, Out: out}
	t.Handle(ctx, unit.Sense{Time: 1, Self: self, Nearby: near})
	return drain(out)
}

func TestSelfPicksKunai(t *testing.T) {
	a := &谭泳辰老先生{boot: true}
	item := unit.Snapshot{ID: 9, Kind: KindKunai, Role: unit.RoleHelper, OwnerID: 1, Slot: 0, X: 5, Y: 0, Radius: kunaiRadius}
	cmds := sensePickup(a, tanSelf(0, 0), []unit.Snapshot{item})
	if !a.hasKunai {
		t.Fatal("自己碰到苦无该捡到")
	}
	if !hasDespawn(cmds, 9) {
		t.Fatal("捡到后该收走苦无")
	}
}

func TestAllyFighterPicksBellForOwner(t *testing.T) {
	a := &谭泳辰老先生{boot: true}
	ally := unit.Snapshot{
		ID: 3, Kind: "酒翁", Role: unit.RoleFighter, Slot: 0,
		X: 80, Y: 0, Radius: 18, AimPriority: unit.DefaultFighterAim,
	}
	item := unit.Snapshot{ID: 8, Kind: KindBell, Role: unit.RoleHelper, OwnerID: 1, Slot: 0, X: 80, Y: 0, Radius: bellRadius}
	cmds := sensePickup(a, tanSelf(0, 0), []unit.Snapshot{ally, item})
	if !a.hasBell {
		t.Fatal("同边战斗机碰到铃铛该给主人捡到")
	}
	if a.hasKunai {
		t.Fatal("不该误捡苦无")
	}
	if !hasDespawn(cmds, 8) {
		t.Fatal("同边捡到后该收走铃铛")
	}
}

func TestEnemyFighterDoesNotPick(t *testing.T) {
	a := &谭泳辰老先生{boot: true}
	foe := unit.Snapshot{
		ID: 4, Kind: "靶子", Role: unit.RoleFighter, Slot: 1,
		X: 80, Y: 0, Radius: 18, AimPriority: unit.DefaultFighterAim,
	}
	item := unit.Snapshot{ID: 8, Kind: KindBell, Role: unit.RoleHelper, OwnerID: 1, Slot: 0, X: 80, Y: 0, Radius: bellRadius}
	cmds := sensePickup(a, tanSelf(0, 0), []unit.Snapshot{foe, item})
	if a.hasBell {
		t.Fatal("敌方碰到不该给主人捡到")
	}
	if hasDespawn(cmds, 8) {
		t.Fatal("敌方碰到不该收走铃铛")
	}
}

func TestMinionDoesNotPick(t *testing.T) {
	a := &谭泳辰老先生{boot: true}
	minion := unit.Snapshot{
		ID: 5, Kind: "蛆宝宝", Role: unit.RoleMinion, Slot: 0, Mortal: true,
		X: 80, Y: 0, Radius: 12,
	}
	item := unit.Snapshot{ID: 9, Kind: KindKunai, Role: unit.RoleHelper, OwnerID: 1, Slot: 0, X: 80, Y: 0, Radius: kunaiRadius}
	cmds := sensePickup(a, tanSelf(0, 0), []unit.Snapshot{minion, item})
	if a.hasKunai {
		t.Fatal("同边活随从碰到不该捡")
	}
	if hasDespawn(cmds, 9) {
		t.Fatal("活随从碰到不该收走苦无")
	}
}
func TestPurifyUsesUpdatedTiming(t *testing.T) {
	if kunaiFlight != 0.55 || purifyInvuln != 1.5 {
		t.Fatalf("kunai flight/purify invuln=%v/%v, want 0.55/1.5", kunaiFlight, purifyInvuln)
	}
	out := make(chan unit.Cmd, 8)
	a := &谭泳辰老先生{maxHP: tanHP, hp: 30, slot: 0}
	a.purify(unit.Context{ID: 1, Kind: KindTan, Out: out}, unit.Sense{Time: 10, Self: tanSelf(0, 0)})
	if a.invulnUntil != 11.5 {
		t.Fatalf("purify invuln until=%v, want 11.5", a.invulnUntil)
	}
}
