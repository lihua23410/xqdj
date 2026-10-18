package 反重力猫咪

import (
	"math"
	"testing"

	"xqdj/internal/unit"
)

func TestBootPutsCatOnOrbit(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	cat := &哈基米{}
	ctx := unit.Context{ID: 1, Kind: KindHajimi, Out: out}
	self := unit.Snapshot{ID: 1, X: orbitR, Y: 0, VX: 10, VY: 0}
	cat.Handle(ctx, unit.Sense{Time: 0, Self: self})
	var vel *unit.SetVelocity
	var force *unit.Force
	for _, c := range drain(out) {
		switch v := c.(type) {
		case unit.SetVelocity:
			cp := v
			vel = &cp
		case unit.Force:
			if v.UnitID == 1 {
				cp := v
				force = &cp
			}
		}
	}
	if vel == nil || vel.VY < 100 || math.Abs(vel.VX) > 30 {
		t.Fatalf("开局该沿逆时针切线飞: %+v", vel)
	}
	if force == nil || force.AX >= 0 {
		t.Fatalf("引力该指向场心: %+v", force)
	}
}

func TestBulletFallsAndVanishes(t *testing.T) {
	out := make(chan unit.Cmd, 4)
	b := &落体弹{owner: 1, slot: 0, speed: 100}
	ctx := unit.Context{ID: 9, Kind: KindHajimiBullet, Out: out}
	b.Handle(ctx, unit.Sense{Time: 0.2, Self: unit.Snapshot{X: 80, Y: 0, VX: -40, VY: 30}})
	var inward bool
	for _, c := range drain(out) {
		f, ok := c.(unit.Force)
		if ok && f.AX < 0 {
			inward = true
		}
	}
	if !inward {
		t.Fatal("子弹该被场心拉过去")
	}
	b.Handle(ctx, unit.Sense{Time: 0.4, Self: unit.Snapshot{X: 4, Y: 0, VX: -200, VY: 0}})
	var gone bool
	for _, c := range drain(out) {
		if _, ok := c.(unit.Despawn); ok {
			gone = true
		}
	}
	if !gone {
		t.Fatal("碰到场心该消失")
	}
}

func TestFasterBulletHitsHarder(t *testing.T) {
	slow := hitFor(80)
	fast := hitFor(240)
	if fast <= slow {
		t.Fatalf("更快的子弹该更疼: 慢 %v 快 %v", slow, fast)
	}
}

func hitFor(speed float64) float64 {
	out := make(chan unit.Cmd, 4)
	b := &落体弹{owner: 1, slot: 0, speed: speed}
	ctx := unit.Context{ID: 9, Kind: KindHajimiBullet, Out: out}
	b.Handle(ctx, unit.Collision{Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1}})
	for _, c := range drain(out) {
		d, ok := c.(unit.Damage)
		if ok {
			return d.Amount
		}
	}
	return 0
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
