package 闪烁

import (
	"xqdj/internal/unit"
	_ "xqdj/map/hex"
)

func newCtx() (unit.Context, chan unit.Cmd) {
	out := make(chan unit.Cmd, 256)
	return unit.Context{ID: 1, Kind: KindFlicker, Out: out}, out
}

func self() unit.Snapshot {
	return unit.Snapshot{ID: 1, Slot: 0, Role: unit.RoleFighter, X: 0, Y: 0, HP: 100, Radius: bodyRadius}
}

func shot(id uint64, x, y, vx, vy float64) unit.Snapshot {
	return unit.Snapshot{ID: id, Slot: 1, Role: unit.RoleProjectile, Kind: "手枪弹", X: x, Y: y, VX: vx, VY: vy, Radius: 4}
}

func sense(t float64, me unit.Snapshot, nearby ...unit.Snapshot) unit.Sense {
	return unit.Sense{Time: t, Self: me, Nearby: nearby}
}

func senseKnock(t float64, me unit.Snapshot, nearby ...unit.Snapshot) unit.Sense {
	s := sense(t, me, nearby...)
	s.Knock = true
	return s
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

func velocity(cmds []unit.Cmd) *unit.SetVelocity {
	var v *unit.SetVelocity
	for _, c := range cmds {
		if x, ok := c.(unit.SetVelocity); ok {
			v = &x
		}
	}
	return v
}

func velocities(cmds []unit.Cmd) []unit.SetVelocity {
	var out []unit.SetVelocity
	for _, c := range cmds {
		if x, ok := c.(unit.SetVelocity); ok {
			out = append(out, x)
		}
	}
	return out
}

func lastPass(cmds []unit.Cmd) *unit.Pass {
	var p *unit.Pass
	for _, c := range cmds {
		if x, ok := c.(unit.Pass); ok {
			p = &x
		}
	}
	return p
}

func confirmed(cmds []unit.Cmd) float64 {
	for _, c := range cmds {
		if x, ok := c.(unit.ConfirmDamage); ok {
			return x.Amount
		}
	}
	return -1
}

func fxNamed(cmds []unit.Cmd, name string) []unit.FX {
	var out []unit.FX
	for _, c := range cmds {
		if x, ok := c.(unit.FX); ok && x.Name == name {
			out = append(out, x)
		}
	}
	return out
}
