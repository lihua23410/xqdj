package 我与周旋久

import (
	"testing"

	"xqdj/internal/unit"
)

func TestMonsterDetoursAroundWall(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	body := &我{mode: modeMonster, boot: true, slot: 0}
	ctx := unit.Context{ID: 1, Kind: KindMe, Out: out}
	self := unit.Snapshot{ID: 1, Kind: KindMe, Slot: 0, Radius: meRadius, HP: 100}
	foe := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 200, Y: 30, Radius: 18}
	wall := unit.WallView{ID: 9, X1: 50, Y1: -80, X2: 50, Y2: 40, Radius: 6}
	body.Handle(ctx, unit.Sense{
		Time: 1, Self: self, Nearby: []unit.Snapshot{foe}, Walls: []unit.WallView{wall},
	})
	v := lastMeVel(drainMe(out))
	if v == nil || v.VY <= v.VX {
		t.Fatalf("墙挡在正前方时该朝上端绕，不该直顶: %+v", v)
	}
}

func drainMe(out chan unit.Cmd) []unit.Cmd {
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

func lastMeVel(cmds []unit.Cmd) *unit.SetVelocity {
	var v *unit.SetVelocity
	for i := range cmds {
		s, ok := cmds[i].(unit.SetVelocity)
		if ok && s.UnitID == 1 {
			cp := s
			v = &cp
		}
	}
	return v
}
