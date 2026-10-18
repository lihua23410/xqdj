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

func TestGravityFieldConvertsEnemyProjectilesForThreeSeconds(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	cat := &哈基米{booted: true, next: 100, nextField: 10, rng: unit.NewRNG(1)}
	ctx := unit.Context{ID: 1, Kind: KindHajimi, Out: out}
	self := unit.Snapshot{ID: 1, Slot: 0, X: orbitR, Y: 0, VX: 0, VY: hajimiSpeed}
	enemy := unit.Snapshot{ID: 2, Role: unit.RoleProjectile, Slot: 1, X: 100, Y: 0, VX: 120, VY: 0, Radius: 5}
	outside := unit.Snapshot{ID: 3, Role: unit.RoleProjectile, Slot: 1, X: 700, Y: 0, VX: 120, VY: 0, Radius: 5}
	clone := unit.Snapshot{ID: 4, Role: unit.RoleClone, Slot: 1, X: 80, Y: 20, Radius: 18}
	ally := unit.Snapshot{ID: 5, Role: unit.RoleProjectile, Slot: 0, X: 90, Y: 10, VX: 120, VY: 0, Radius: 5}
	catBullet := unit.Snapshot{ID: 8, Kind: KindHajimiBullet, Role: unit.RoleProjectile, OwnerID: 1, Slot: 0, X: 260, Y: 0, VX: 100, VY: 0, Radius: 5}
	cat.Handle(ctx, unit.Sense{Time: 10, Self: self, Nearby: []unit.Snapshot{enemy, outside, clone, ally, catBullet}})
	cmds := drain(out)
	if _, ok := teleportOf(cmds, 8); !ok {
		t.Fatal("own bullet should be captured into an orbit")
	}
	if _, ok := velocityOf(cmds, 8); !ok {
		t.Fatal("orbiting bullet should receive tangent velocity")
	}
	field := false
	for _, cmd := range cmds {
		if fx, ok := cmd.(unit.FX); ok && fx.Name == "gravity-field" && fx.Amount == gravityFieldRadius && fx.VX == gravityFieldDuration {
			field = true
		}
	}
	if !field || cat.fieldUntil != 13 {
		t.Fatalf("field fx=%v until=%v, want duration through 13", field, cat.fieldUntil)
	}
	if !hasDespawn(cmds, 2) || !hasConvertedSpawn(cmds, 100, 0, 1) {
		t.Fatal("enemy projectile inside field should be converted to Hajimi bullet")
	}
	if _, ok := velocityOf(cmds, 2); ok {
		t.Fatal("converted enemy projectile should not receive a twist velocity")
	}
	v5, ok5 := velocityOf(cmds, 5)
	v4, ok4 := velocityOf(cmds, 4)
	if !ok5 || !ok4 {
		t.Fatalf("ally projectile/clone should be twisted, ally=%v clone=%v", ok5, ok4)
	}
	if speed := math.Hypot(v5.VX, v5.VY); math.Abs(speed-120) > 1e-6 {
		t.Fatalf("ally projectile speed=%v, want preserved 120", speed)
	}
	if speed := math.Hypot(v4.VX, v4.VY); math.Abs(speed-gravityFieldMinSpeed) > 1e-6 {
		t.Fatalf("clone speed=%v, want %v", speed, gravityFieldMinSpeed)
	}
	if hasDespawn(cmds, 3) {
		t.Fatal("out-of-range projectile should not be converted")
	}
	late := unit.Snapshot{ID: 6, Role: unit.RoleProjectile, Slot: 1, X: 120, Y: 10, Radius: 5}
	cat.Handle(ctx, unit.Sense{Time: 12.9, Self: self, Nearby: []unit.Snapshot{late}})
	if !hasDespawn(drain(out), 6) {
		t.Fatal("projectile entering during the three-second field should be converted")
	}
	expired := unit.Snapshot{ID: 7, Role: unit.RoleProjectile, Slot: 1, X: 120, Y: 10, Radius: 5}
	cat.Handle(ctx, unit.Sense{Time: 13.1, Self: self, Nearby: []unit.Snapshot{expired}})
	releaseCmds := drain(out)
	if hasDespawn(releaseCmds, 7) || cat.fieldUntil != 0 {
		t.Fatal("field should stop converting after three seconds")
	}
	releaseVel, ok := velocityOf(releaseCmds, 8)
	if !ok {
		t.Fatal("orbiting bullet should be released when the field ends")
	}
	wantRelease := gravityReleaseSpeed(120)
	if speed := math.Hypot(releaseVel.VX, releaseVel.VY); math.Abs(speed-wantRelease) > 1e-6 {
		t.Fatalf("release speed=%v, want potential-energy speed %v", speed, wantRelease)
	}
	if cat.orbiters != nil {
		t.Fatal("orbit list should clear after release")
	}
	if cat.nextField != 20 {
		t.Fatalf("next field=%v, want 20", cat.nextField)
	}
}

func hasDespawn(cmds []unit.Cmd, id uint64) bool {
	for _, cmd := range cmds {
		if d, ok := cmd.(unit.Despawn); ok && d.UnitID == id {
			return true
		}
	}
	return false
}

func hasConvertedSpawn(cmds []unit.Cmd, x, y float64, owner uint64) bool {
	for _, cmd := range cmds {
		if spawn, ok := cmd.(unit.Spawn); ok && spawn.Kind == KindHajimiBullet && spawn.OwnerID == owner && spawn.X == x && spawn.Y == y {
			return true
		}
	}
	return false
}

func teleportOf(cmds []unit.Cmd, id uint64) (unit.Teleport, bool) {
	for _, cmd := range cmds {
		if t, ok := cmd.(unit.Teleport); ok && t.UnitID == id {
			return t, true
		}
	}
	return unit.Teleport{}, false
}

func velocityOf(cmds []unit.Cmd, id uint64) (unit.SetVelocity, bool) {
	for _, cmd := range cmds {
		if v, ok := cmd.(unit.SetVelocity); ok && v.UnitID == id {
			return v, true
		}
	}
	return unit.SetVelocity{}, false
}

func TestFasterBulletHitsHarder(t *testing.T) {
	slow := hitFor(80)
	fast := hitFor(240)
	if fast <= slow {
		t.Fatalf("更快的子弹该更疼: 慢 %v 快 %v", slow, fast)
	}
	if fast != 8.4 {
		t.Fatalf("240px/s bullet damage=%v, want 8.4", fast)
	}
	if got := bulletDamage(1000); got != dmgMax {
		t.Fatalf("max bullet damage=%v, want %v", got, dmgMax)
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
