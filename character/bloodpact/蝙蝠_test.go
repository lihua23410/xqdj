package 血契

import (
	"testing"

	"xqdj/internal/unit"
)

func TestBatDetoursAroundWall(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	bat := &血契蝠{owner: 1, slot: 0}
	ctx := unit.Context{ID: 7, Kind: KindBloodBat, Out: out}
	self := unit.Snapshot{ID: 7, Kind: KindBloodBat, Slot: 0, Radius: batRadius}
	foe := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 200, Y: 30, Radius: 18}
	wall := unit.WallView{ID: 9, X1: 50, Y1: -80, X2: 50, Y2: 40, Radius: 6}
	bat.Handle(ctx, unit.Sense{Time: 0, Self: self})
	drainBat(out)
	bat.nextBite = 100 // 冷却中才绕环；冷却一到会直线穿过去
	bat.Handle(ctx, unit.Sense{Time: 0.02, Self: self, Nearby: []unit.Snapshot{foe}, Walls: []unit.WallView{wall}})
	v := lastBatVel(drainBat(out))
	if v == nil || v.VY <= v.VX {
		t.Fatalf("墙挡在正前方时该朝上端绕，不该直顶: %+v", v)
	}
}

func TestBatDashPassesThrough(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	bat := &血契蝠{owner: 1, slot: 0}
	ctx := unit.Context{ID: 7, Kind: KindBloodBat, Out: out}
	self := unit.Snapshot{ID: 7, Kind: KindBloodBat, Slot: 0, Radius: batRadius}
	foe := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 100, Y: 0, Radius: 18}
	bat.Handle(ctx, unit.Sense{Time: 0, Self: self})
	drainBat(out)
	bat.Handle(ctx, unit.Sense{Time: 0.02, Self: self, Nearby: []unit.Snapshot{foe}})
	v := lastBatVel(drainBat(out))
	if !bat.dashing || v == nil || v.VX < batDash-1 || v.VY > 1 || v.VY < -1 {
		t.Fatalf("冷却到了该锁死朝敌人穿，速率 %v: dashing=%v vel=%+v", batDash, bat.dashing, v)
	}
	if bat.dashLeft < 100+18+batRadius+batRing-1 {
		t.Fatalf("穿过距离该够到对面环上: %v", bat.dashLeft)
	}
	bat.Handle(ctx, unit.Collision{Time: 0.05, Other: foe})
	dmg := 0
	for _, c := range drainBat(out) {
		if d, ok := c.(unit.Damage); ok && d.To == foe.ID {
			dmg++
			if d.Amount != batDamage {
				t.Fatalf("一口该是 %v，实际 %v", batDamage, d.Amount)
			}
		}
	}
	if dmg != 1 {
		t.Fatalf("穿过时咬一口，实际 %d", dmg)
	}
	bat.Handle(ctx, unit.Collision{Time: 0.06, Other: foe})
	for _, c := range drainBat(out) {
		if _, ok := c.(unit.Damage); ok {
			t.Fatal("同一次穿过不该再咬")
		}
	}
	if !bat.dashing {
		t.Fatal("咬到了不该停下")
	}
	bat.Handle(ctx, unit.WallHit{Time: 0.07})
	if bat.dashing {
		t.Fatal("撞墙这次穿该停")
	}
}

func drainBat(out chan unit.Cmd) []unit.Cmd {
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

func lastBatVel(cmds []unit.Cmd) *unit.SetVelocity {
	var v *unit.SetVelocity
	for i := range cmds {
		s, ok := cmds[i].(unit.SetVelocity)
		if ok && s.UnitID == 7 {
			cp := s
			v = &cp
		}
	}
	return v
}
