package r缪

import (
	"testing"
	"xqdj/internal/unit"
)

func shotSelf(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 9, Kind: KindMiuShot, Role: unit.RoleProjectile, Slot: 0,
		X: x, Y: y, Radius: miuShotRadius,
	}
}

func allyMiu(id uint64, x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: id, Kind: KindMiu, Role: unit.RoleMinion, Slot: 0, Mortal: true,
		X: x, Y: y, Radius: miuRadius,
	}
}

func hasDespawn(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if d, ok := c.(unit.Despawn); ok && d.UnitID == 9 {
			return true
		}
	}
	return false
}

func hasDmgTo(cmds []unit.Cmd, to uint64) bool {
	for _, c := range cmds {
		if d, ok := c.(unit.Damage); ok && d.To == to {
			return true
		}
	}
	return false
}

func TestShotHitsAllyAndDespawns(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	b := &缪弹{owner: 1, slot: 0}
	ctx := unit.Context{ID: 9, Kind: KindMiuShot, Out: out}
	b.Handle(ctx, unit.Sense{
		Time:   1,
		Self:   shotSelf(0, 0),
		Nearby: []unit.Snapshot{allyMiu(2, 10, 0)},
	})
	cmds := drain(out)
	if !hasDmgTo(cmds, 2) {
		t.Fatalf("ally should take damage: %v", cmds)
	}
	if !hasDespawn(cmds) {
		t.Fatal("shot should despawn on ally")
	}
}

func TestShotSweepsThroughAlly(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	b := &缪弹{owner: 1, slot: 0, booted: true, px: -40, py: 0}
	ctx := unit.Context{ID: 9, Kind: KindMiuShot, Out: out}
	b.Handle(ctx, unit.Sense{
		Time:   1,
		Self:   shotSelf(40, 0),
		Nearby: []unit.Snapshot{allyMiu(2, 0, 0)},
	})
	cmds := drain(out)
	if !hasDmgTo(cmds, 2) || !hasDespawn(cmds) {
		t.Fatalf("sweep should hit ally: %v", cmds)
	}
}

func TestShotDoesNotHitShooter(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	b := &缪弹{owner: 1, slot: 0}
	ctx := unit.Context{ID: 9, Kind: KindMiuShot, Out: out}
	b.Handle(ctx, unit.Sense{
		Time:   1,
		Self:   shotSelf(0, 0),
		Nearby: []unit.Snapshot{allyMiu(1, 0, 0)},
	})
	cmds := drain(out)
	if hasDmgTo(cmds, 1) || hasDespawn(cmds) {
		t.Fatalf("must not hit shooter: %v", cmds)
	}
}

func TestShotHitsEnemyCollision(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	b := &缪弹{owner: 1, slot: 0}
	ctx := unit.Context{ID: 9, Kind: KindMiuShot, Out: out}
	b.Handle(ctx, unit.Collision{
		Other: unit.Snapshot{ID: 8, Kind: "靶子", Role: unit.RoleFighter, Slot: 1},
	})
	cmds := drain(out)
	if !hasDmgTo(cmds, 8) || !hasDespawn(cmds) {
		t.Fatalf("enemy collision should despawn: %v", cmds)
	}
}
