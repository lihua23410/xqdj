package 原型机_整合

import (
	"math"
	"testing"

	"xqdj/internal/unit"
)

func TestTrackSetsAimAndRestores(t *testing.T) {
	a := fighter()
	ctx, out := newCtx()
	a.Handle(ctx, sense(0, 0, 0, []unit.Snapshot{aim(3, 40, 0, 15)}))
	if got := aimOf(drain(out), 3); got == nil || got.Value != 1 || got.From != 1 {
		t.Fatalf("lock aim=%+v", got)
	}
	a.Handle(ctx, sense(5, 0, 0, []unit.Snapshot{aim(3, 40, 0, 15)}))
	if got := aimOf(drain(out), 3); got == nil || got.Value != 15 {
		t.Fatalf("restore aim=%+v", got)
	}
}

func TestOpeningInsideLocksLowerAim(t *testing.T) {
	a := fighter()
	ctx, out := newCtx()
	a.Handle(ctx, sense(0, 0, 0, []unit.Snapshot{
		aim(2, 100, 0, 60),
		aim(3, 40, 0, 15),
	}))
	if a.trackID != 3 {
		t.Fatalf("track=%d want 3", a.trackID)
	}
	if fx := named(drain(out), "track"); fx == nil || fx.Amount != 1 {
		t.Fatal("missing track on")
	}
}

func TestSpawnInsideDoesNotLock(t *testing.T) {
	a := fighter()
	ctx, out := newCtx()
	a.Handle(ctx, sense(0, 0, 0, []unit.Snapshot{aim(2, 400, 0, 15)}))
	_ = drain(out)
	a.Handle(ctx, sense(0.02, 0, 0, []unit.Snapshot{
		aim(2, 400, 0, 15),
		aim(9, 20, 0, 15),
	}))
	if a.trackID != 0 {
		t.Fatalf("spawn inside locked %d", a.trackID)
	}
}

func TestEnterShovesAway(t *testing.T) {
	a := fighter()
	ctx, out := newCtx()
	self := func(vx, vy float64) unit.Sense {
		return unit.Sense{
			Self:   unit.Snapshot{ID: 1, Slot: 0, VX: vx, VY: vy, Radius: bodyRadius, HP: 100, AimPriority: 15},
			Nearby: []unit.Snapshot{aim(2, 400, 0, 15)},
		}
	}
	a.Handle(ctx, self(80, 0))
	_ = drain(out)
	a.Handle(ctx, unit.Sense{
		Time:   0.02,
		Self:   unit.Snapshot{ID: 1, Slot: 0, VX: 80, VY: 0, Radius: bodyRadius, HP: 100, AimPriority: 15},
		Nearby: []unit.Snapshot{aim(2, 100, 0, 15)},
	})
	v := velocity(drain(out), 1)
	if v == nil || math.Abs(v.VX-(-150)) > 1e-6 || math.Abs(v.VY) > 1e-6 {
		t.Fatalf("shove=%+v", v)
	}
	a.Handle(ctx, unit.Sense{
		Time:   0.04,
		Self:   unit.Snapshot{ID: 1, Slot: 0, VX: -80, VY: 0, Radius: bodyRadius, HP: 100, AimPriority: 15},
		Nearby: []unit.Snapshot{aim(2, 100, 0, 15)},
	})
	if velocity(drain(out), 1) != nil {
		t.Fatal("shoved again while still inside")
	}
}

func TestFleeExtraCapsAt300(t *testing.T) {
	a := fighter()
	ctx, out := newCtx()
	far := aim(2, 400, 0, 15)
	a.Handle(ctx, unit.Sense{
		Self:   unit.Snapshot{ID: 1, Slot: 0, VX: cruise + fleeExtraMax, VY: 0, Radius: bodyRadius, HP: 100, AimPriority: 15},
		Nearby: []unit.Snapshot{far},
	})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{
		Time:   0.02,
		Self:   unit.Snapshot{ID: 1, Slot: 0, VX: cruise + fleeExtraMax, VY: 0, Radius: bodyRadius, HP: 100, AimPriority: 15},
		Nearby: []unit.Snapshot{aim(2, 100, 0, 15)},
	})
	v := velocity(drain(out), 1)
	if v == nil || math.Abs(v.VX-(-(cruise+fleeExtraMax))) > 1e-6 || math.Abs(v.VY) > 1e-6 {
		t.Fatalf("capped shove=%+v", v)
	}
}

func TestCrossInLocksAndDoesNotSwitch(t *testing.T) {
	a := fighter()
	ctx, _ := newCtx()
	far := aim(2, 400, 0, 15)
	a.Handle(ctx, sense(0, 0, 0, []unit.Snapshot{far}))
	near := aim(2, 100, 0, 15)
	a.Handle(ctx, sense(0.02, 0, 0, []unit.Snapshot{near}))
	if a.trackID != 2 {
		t.Fatalf("track=%d", a.trackID)
	}
	other := aim(4, 30, 0, 1)
	a.Handle(ctx, sense(0.04, 0, 0, []unit.Snapshot{near, other}))
	if a.trackID != 2 {
		t.Fatalf("switched to %d", a.trackID)
	}
}

func TestLeaveDoesNotDropTrack(t *testing.T) {
	a := fighter()
	ctx, _ := newCtx()
	a.Handle(ctx, sense(0, 0, 0, []unit.Snapshot{aim(2, 40, 0, 15)}))
	a.Handle(ctx, sense(1, 0, 0, []unit.Snapshot{aim(2, 400, 0, 15)}))
	if a.trackID != 2 || a.trackUntil != 5 {
		t.Fatalf("id=%d until=%v", a.trackID, a.trackUntil)
	}
}

func TestExpiryDoesNotRefreshWhileInside(t *testing.T) {
	a := fighter()
	ctx, out := newCtx()
	a.Handle(ctx, sense(0, 0, 0, []unit.Snapshot{aim(2, 40, 0, 15)}))
	_ = drain(out)
	a.Handle(ctx, sense(5, 0, 0, []unit.Snapshot{aim(2, 40, 0, 15)}))
	if a.trackID != 0 {
		t.Fatal("still tracking")
	}
	if fx := named(drain(out), "track"); fx == nil || fx.Amount != 0 {
		t.Fatal("missing track off")
	}
	a.Handle(ctx, sense(5.02, 0, 0, []unit.Snapshot{aim(2, 40, 0, 15)}))
	if a.trackID != 0 {
		t.Fatal("refreshed while inside")
	}
}

func TestReenterAfterLeave(t *testing.T) {
	a := fighter()
	ctx, _ := newCtx()
	a.Handle(ctx, sense(0, 0, 0, []unit.Snapshot{aim(2, 40, 0, 15)}))
	a.Handle(ctx, sense(5, 0, 0, []unit.Snapshot{aim(2, 40, 0, 15)}))
	a.Handle(ctx, sense(5.02, 0, 0, []unit.Snapshot{aim(2, 400, 0, 15)}))
	a.Handle(ctx, sense(5.04, 0, 0, []unit.Snapshot{aim(2, 40, 0, 15)}))
	if a.trackID != 2 {
		t.Fatalf("track=%d", a.trackID)
	}
}

func TestDeathClears(t *testing.T) {
	a := fighter()
	ctx, _ := newCtx()
	a.Handle(ctx, sense(0, 0, 0, []unit.Snapshot{aim(2, 40, 0, 15)}))
	a.Handle(ctx, sense(1, 0, 0, nil))
	if a.trackID != 0 {
		t.Fatal("still tracking a gone unit")
	}
}

func TestShotConeNoRecoil(t *testing.T) {
	a := fighter()
	a.roll = func() float64 { return 1 }
	ctx, out := newCtx()
	self := unit.Snapshot{ID: 1, Slot: 0, X: 0, Y: 0, VX: 100, VY: 0, Radius: bodyRadius, HP: 100, AimPriority: 15}
	a.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{aim(2, 300, 0, 15)}})
	cmds := drain(out)
	sp := spawned(cmds)
	if sp == nil {
		t.Fatal("no shot")
	}
	ang := math.Atan2(sp.VY, sp.VX)
	if math.Abs(ang-cone) > 1e-6 {
		t.Fatalf("ang=%v want %v", ang, cone)
	}
	if math.Abs(math.Hypot(sp.VX, sp.VY)-roundSpeed) > 1e-6 {
		t.Fatalf("speed=%v", math.Hypot(sp.VX, sp.VY))
	}
	if rv := velocity(cmds, 1); rv != nil {
		t.Fatalf("recoil=%+v", rv)
	}
}

func TestNoTargetNoShot(t *testing.T) {
	a := fighter()
	ctx, out := newCtx()
	a.Handle(ctx, sense(0, 0, 0, nil))
	if spawned(drain(out)) != nil {
		t.Fatal("shot without target")
	}
}

func TestSteerOneDegree(t *testing.T) {
	a := fighter()
	ctx, out := newCtx()
	a.Handle(ctx, sense(0, 0, 0, []unit.Snapshot{aim(2, 40, 0, 15)}))
	_ = drain(out)
	bullet := unit.Snapshot{ID: 8, Kind: KindIntegratedRound, OwnerID: 1, Slot: 0, X: 0, Y: 10, VX: roundSpeed, VY: 0}
	target := aim(2, 0, 80, 15)
	a.Handle(ctx, sense(0.02, 0, 0, []unit.Snapshot{target, bullet}))
	v := velocity(drain(out), 8)
	if v == nil {
		t.Fatal("no steer")
	}
	got := math.Atan2(v.VY, v.VX)
	if math.Abs(got-turnStep) > 1e-6 {
		t.Fatalf("ang=%v want %v", got, turnStep)
	}
	if math.Abs(math.Hypot(v.VX, v.VY)-roundSpeed) > 1e-6 {
		t.Fatalf("speed changed %v", math.Hypot(v.VX, v.VY))
	}
}

func TestRoundHitsAndSecondWall(t *testing.T) {
	b := &整合弹{owner: 1, slot: 0}
	ctx, out := newCtx()
	b.Handle(ctx, unit.WallHit{})
	if len(drain(out)) != 0 {
		t.Fatal("first wall despawned")
	}
	b.Handle(ctx, unit.WallHit{})
	if _, ok := drain(out)[0].(unit.Despawn); !ok {
		t.Fatal("second wall kept the round")
	}
	ctx, out = newCtx()
	b.Handle(ctx, unit.Collision{Other: unit.Snapshot{ID: 2, Slot: 1, Role: unit.RoleFighter, HP: 100}})
	cmds := drain(out)
	d, ok := cmds[0].(unit.Damage)
	if !ok || d.Amount != roundDamage {
		t.Fatalf("damage=%+v", cmds)
	}
}

func fighter() *原型机_整合 {
	return &原型机_整合{roll: func() float64 { return 0.5 }}
}

func newCtx() (unit.Context, chan unit.Cmd) {
	out := make(chan unit.Cmd, 16)
	return unit.Context{ID: 1, Kind: KindIntegrated, Out: out}, out
}

func sense(t, vx, vy float64, nearby []unit.Snapshot) unit.Sense {
	return unit.Sense{
		Time: t,
		Self: unit.Snapshot{ID: 1, Slot: 0, VX: vx, VY: vy, Radius: bodyRadius, HP: 100, AimPriority: 15},
		Nearby: nearby,
	}
}

func aim(id uint64, x, y float64, pri uint8) unit.Snapshot {
	return unit.Snapshot{ID: id, Slot: 1, Role: unit.RoleFighter, X: x, Y: y, HP: 100, AimPriority: pri}
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

func named(cmds []unit.Cmd, name string) *unit.FX {
	for i := range cmds {
		fx, ok := cmds[i].(unit.FX)
		if ok && fx.Name == name {
			return &fx
		}
	}
	return nil
}

func spawned(cmds []unit.Cmd) *unit.Spawn {
	for i := range cmds {
		sp, ok := cmds[i].(unit.Spawn)
		if ok && sp.Kind == KindIntegratedRound {
			return &sp
		}
	}
	return nil
}

func aimOf(cmds []unit.Cmd, id uint64) *unit.SetAimPriority {
	for i := range cmds {
		v, ok := cmds[i].(unit.SetAimPriority)
		if ok && v.UnitID == id {
			return &v
		}
	}
	return nil
}

func velocity(cmds []unit.Cmd, id uint64) *unit.SetVelocity {
	for i := range cmds {
		v, ok := cmds[i].(unit.SetVelocity)
		if ok && v.UnitID == id {
			return &v
		}
	}
	return nil
}
