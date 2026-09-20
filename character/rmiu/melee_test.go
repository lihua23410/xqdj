package r缪

import (
	"testing"
	"xqdj/internal/unit"
)

func meleeMiu() *缪 {
	return &缪{
		ownerID:       1,
		slot:          0,
		booted:        true,
		speed:         miuBaseSpeed,
		vx:            miuBaseSpeed,
		vy:            0,
		meleeReadyAt:  0,
		fireReadyAt:   999,
		thrustReadyAt: 999,
	}
}

func miuSelf() unit.Snapshot {
	return unit.Snapshot{
		ID: 10, Kind: KindMiu, Role: unit.RoleMinion, Slot: 0, OwnerID: 1,
		X: 0, Y: 0, VX: miuBaseSpeed, VY: 0, Radius: miuRadius,
	}
}

func foe(id uint64, x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: id, Kind: "靶子", Role: unit.RoleFighter, Slot: 1,
		X: x, Y: y, Radius: 18,
	}
}

func TestMeleeSpawnsRangeArc(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	m := meleeMiu()
	ctx := unit.Context{ID: 10, Kind: KindMiu, Out: out}
	m.Handle(ctx, unit.Sense{Time: 1, Self: miuSelf()})
	found := false
	for _, c := range drain(out) {
		if sp, ok := c.(unit.Spawn); ok && sp.Kind == KindMiuArc {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("missing melee range arc")
	}
}

func TestMeleeHitsFront(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	m := meleeMiu()
	ctx := unit.Context{ID: 10, Kind: KindMiu, Out: out}
	m.Handle(ctx, unit.Sense{Time: 1, Self: miuSelf(), Nearby: []unit.Snapshot{foe(2, 30, 0)}})
	if !hasDmgTo(drain(out), 2) {
		t.Fatal("front of 60° dagger fan should hit")
	}
}

func TestMeleeMissesBack(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	m := meleeMiu()
	ctx := unit.Context{ID: 10, Kind: KindMiu, Out: out}
	m.Handle(ctx, unit.Sense{Time: 1, Self: miuSelf(), Nearby: []unit.Snapshot{foe(3, -30, 0)}})
	if hasDmgTo(drain(out), 3) {
		t.Fatal("behind should not take melee")
	}
}

func TestMeleeMissesSideOutsideFan(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	m := meleeMiu()
	ctx := unit.Context{ID: 10, Kind: KindMiu, Out: out}
	m.Handle(ctx, unit.Sense{Time: 1, Self: miuSelf(), Nearby: []unit.Snapshot{foe(4, 0, 30)}})
	if hasDmgTo(drain(out), 4) {
		t.Fatal("90° is outside 60° dagger fan")
	}
}
