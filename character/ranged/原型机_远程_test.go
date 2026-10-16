package 原型机_远程

import (
	"math"
	"testing"

	"xqdj/internal/unit"
)

func TestBulletAcceleratesAlongLeg(t *testing.T) {
	b := &远程子弹{owner: 9, slot: 0}
	ctx, out := newCtx()
	b.Handle(ctx, bulletSense(0, bulletMuzzle, 0))
	v := velocity(drain(out))
	if v == nil || math.Abs(v.VX-bulletMuzzle) > 1e-6 || math.Abs(v.VY) > 1e-6 {
		t.Fatalf("muzzle=%+v", v)
	}
	b.Handle(ctx, bulletSense(1, bulletMuzzle, 0))
	v = velocity(drain(out))
	want := bulletMuzzle + bulletAccel
	if v == nil || math.Abs(v.VX-want) > 1e-6 || math.Abs(v.VY) > 1e-6 {
		t.Fatalf("after 1s=%+v want %v", v, want)
	}
}

func TestBounceResetsThenThirdWallDespawns(t *testing.T) {
	b := &远程子弹{owner: 9, slot: 0}
	ctx, out := newCtx()
	b.Handle(ctx, bulletSense(1, bulletMuzzle+bulletAccel, 0))
	_ = drain(out)

	b.Handle(ctx, unit.WallHit{Time: 1, NX: -1, NY: 0})
	cmds := drain(out)
	if despawned(cmds) {
		t.Fatal("first wall despawned")
	}
	if z := velocity(cmds); z == nil || z.VX != 0 || z.VY != 0 {
		t.Fatalf("reset=%+v", z)
	}

	dt := 1.0 / 60
	b.Handle(ctx, bulletSense(1+dt, bulletMuzzle+bulletAccel, 0))
	v := velocity(drain(out))
	want := bulletAccel * dt
	if v == nil || math.Abs(v.VX-(-want)) > 1e-6 || math.Abs(v.VY) > 1e-6 {
		t.Fatalf("reaccel=%+v want %v", v, -want)
	}

	b.Handle(ctx, unit.WallHit{Time: 2, NX: 1, NY: 0})
	if despawned(drain(out)) {
		t.Fatal("second wall despawned")
	}
	b.Handle(ctx, unit.WallHit{Time: 3, NX: -1, NY: 0})
	if !despawned(drain(out)) {
		t.Fatal("third wall kept the bullet")
	}
	b.Handle(ctx, bulletSense(4, 0, 0))
	if len(drain(out)) != 0 {
		t.Fatal("flew after the third wall")
	}
}

func TestHitDealsNine(t *testing.T) {
	b := &远程子弹{owner: 9, slot: 0}
	ctx, out := newCtx()
	b.Handle(ctx, unit.Collision{Other: unit.Snapshot{ID: 2, Slot: 1, Role: unit.RoleFighter, HP: 100}})
	cmds := drain(out)
	var dmg *unit.Damage
	for i := range cmds {
		d, ok := cmds[i].(unit.Damage)
		if ok {
			dmg = &d
		}
	}
	if dmg == nil || dmg.Amount != bulletDamage || !despawned(cmds) {
		t.Fatalf("hit=%+v", cmds)
	}
	b.Handle(ctx, bulletSense(1, bulletMuzzle, 0))
	if len(drain(out)) != 0 {
		t.Fatal("flew after hit")
	}
}

func TestBulletLooksLikeRocket(t *testing.T) {
	spec, ok := unit.Lookup(kindBullet)
	if !ok || spec.Look.Trail {
		t.Fatalf("trail=%v ok=%v", spec.Look.Trail, ok)
	}
	if len(spec.Look.FX) != 1 || spec.Look.FX[0] != "rocket" {
		t.Fatalf("fx=%v", spec.Look.FX)
	}
}

func TestShotLeavesAtMuzzle(t *testing.T) {
	r := &原型机_远程{}
	ctx, out := newCtx()
	r.Handle(ctx, unit.Sense{
		Self: unit.Snapshot{ID: 1, Slot: 0, Radius: rangedRadius, HP: 100, AimPriority: 15},
		Nearby: []unit.Snapshot{{
			ID: 2, Slot: 1, Role: unit.RoleFighter, X: 100, HP: 100, AimPriority: 15,
		}},
	})
	var sp *unit.Spawn
	for _, c := range drain(out) {
		s, ok := c.(unit.Spawn)
		if ok && s.Kind == kindBullet {
			sp = &s
		}
	}
	if sp == nil || math.Abs(sp.VX-bulletMuzzle) > 1e-6 || math.Abs(sp.VY) > 1e-6 {
		t.Fatalf("spawn=%+v", sp)
	}
}

func newCtx() (unit.Context, chan unit.Cmd) {
	out := make(chan unit.Cmd, 8)
	return unit.Context{ID: 7, Kind: kindBullet, Out: out}, out
}

func bulletSense(t, vx, vy float64) unit.Sense {
	return unit.Sense{
		Time: t,
		Self: unit.Snapshot{ID: 7, Kind: kindBullet, VX: vx, VY: vy, Radius: bulletRadius},
	}
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
	for i := range cmds {
		v, ok := cmds[i].(unit.SetVelocity)
		if ok {
			return &v
		}
	}
	return nil
}

func despawned(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if _, ok := c.(unit.Despawn); ok {
			return true
		}
	}
	return false
}
