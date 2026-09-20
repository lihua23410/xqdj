package r缪

import (
	"math"
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

func thrustingMiu() *缪 {
	return &缪{
		slot:        0,
		booted:      true,
		speed:       miuBaseSpeed,
		vx:          miuThrustSpeed,
		vy:          0,
		thrusting:   true,
		thrustToken: 9,
		thrustUntil: 1,
		thrustDX:    1,
		thrustDY:    0,
	}
}

func velCmd(cmds []unit.Cmd) *unit.SetVelocity {
	var v *unit.SetVelocity
	for _, c := range cmds {
		if sv, ok := c.(unit.SetVelocity); ok {
			cp := sv
			v = &cp
		}
	}
	return v
}

func TestThrustSnapsSpeedOnWall(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	m := thrustingMiu()
	ctx := unit.Context{ID: 1, Kind: KindMiu, Out: out}
	m.Handle(ctx, unit.WallHit{Time: 0.2, NX: 1, NY: 0})
	if m.thrusting {
		t.Fatal("thrust should end on wall")
	}
	v := velCmd(drain(out))
	if v == nil {
		t.Fatal("missing SetVelocity")
	}
	if math.Abs(v.VX+miuBaseSpeed) > 1e-6 || math.Abs(v.VY) > 1e-6 {
		t.Fatalf("vel=%v,%v want -%v,0", v.VX, v.VY, miuBaseSpeed)
	}
}

func TestThrustSnapsSpeedOnEnemy(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	m := thrustingMiu()
	ctx := unit.Context{ID: 1, Kind: KindMiu, Out: out}
	m.Handle(ctx, unit.Collision{
		Time:  0.2,
		Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, Mortal: true},
		NX:    1, NY: 0,
	})
	if m.thrusting {
		t.Fatal("thrust should end on enemy")
	}
	v := velCmd(drain(out))
	if v == nil {
		t.Fatal("missing SetVelocity")
	}
	sp := math.Hypot(v.VX, v.VY)
	if math.Abs(sp-miuBaseSpeed) > 1e-6 {
		t.Fatalf("speed=%v want %v", sp, miuBaseSpeed)
	}
}

func TestThrustSnapsSpeedOnFriendly(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	m := thrustingMiu()
	ctx := unit.Context{ID: 1, Kind: KindMiu, Out: out}
	m.Handle(ctx, unit.Collision{
		Time:  0.2,
		Other: unit.Snapshot{ID: 3, Kind: KindMiu, Role: unit.RoleMinion, Slot: 0, Mortal: true},
		NX:    1, NY: 0,
	})
	if m.thrusting {
		t.Fatal("thrust should end on friendly")
	}
	v := velCmd(drain(out))
	if v == nil {
		t.Fatal("missing SetVelocity")
	}
	sp := math.Hypot(v.VX, v.VY)
	if math.Abs(sp-miuBaseSpeed) > 1e-6 {
		t.Fatalf("speed=%v want %v", sp, miuBaseSpeed)
	}
}

func TestThrustTimeoutSnapsSpeed(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	m := thrustingMiu()
	m.thrustUntil = 0.4
	ctx := unit.Context{ID: 1, Kind: KindMiu, Out: out}
	m.Handle(ctx, unit.Sense{
		Time: 0.4,
		Self: unit.Snapshot{
			ID: 1, Kind: KindMiu, Role: unit.RoleMinion, Slot: 0,
			X: 0, Y: 0, VX: miuThrustSpeed, VY: 0, Radius: miuRadius,
			HP: miuHP, MaxHP: miuHP,
		},
	})
	if m.thrusting {
		t.Fatal("thrust should end on timeout")
	}
	v := velCmd(drain(out))
	if v == nil {
		t.Fatal("missing SetVelocity")
	}
	if math.Abs(v.VX-miuBaseSpeed) > 1e-6 || math.Abs(v.VY) > 1e-6 {
		t.Fatalf("vel=%v,%v want %v,0", v.VX, v.VY, miuBaseSpeed)
	}
}
