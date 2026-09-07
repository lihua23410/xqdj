package 狼人

import (
	"math"
	"testing"
	"xqdj/internal/unit"
)

func TestBootSpawnsMoonAtCenter(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	w := &狼人{}
	w.Handle(unit.Context{ID: 1, Kind: KindWolf, Out: out}, unit.Sense{
		Time: 0,
		Self: selfAt(80, 0),
	})
	cmds := drain(out)
	sp := mustSpawn(t, cmds)
	if sp.Kind != KindMoon || sp.X != 0 || sp.Y != 0 || sp.OwnerID != 1 {
		t.Fatalf("spawn=%+v", sp)
	}
	ph := mustPhase(t, cmds)
	if ph.Amount != 0 {
		t.Fatalf("initial phase=%v", ph.Amount)
	}
}

func TestTouchAdvancesPhaseOncePerEnter(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	w := bootedWolf()
	moon := moonSnap(1)
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}

	w.Handle(ctx, unit.Sense{Time: 1, Self: selfAt(10, 0), Nearby: []unit.Snapshot{moon}})
	if got := lastPhase(drain(out)); got != 1 {
		t.Fatalf("enter phase=%v", got)
	}

	w.Handle(ctx, unit.Sense{Time: 1.1, Self: selfAt(10, 0), Nearby: []unit.Snapshot{moon}})
	if cmds := drain(out); hasPhase(cmds) {
		t.Fatalf("stay should not advance: %v", cmds)
	}

	w.Handle(ctx, unit.Sense{Time: 1.2, Self: selfAt(120, 0), Nearby: []unit.Snapshot{moon}})
	_ = drain(out)
	w.Handle(ctx, unit.Sense{Time: 1.3, Self: selfAt(10, 0), Nearby: []unit.Snapshot{moon}})
	if got := lastPhase(drain(out)); got != 2 {
		t.Fatalf("reenter phase=%v", got)
	}
}

func TestEnemyTouchAdvancesPhase(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	w := bootedWolf()
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	moon := moonSnap(1)
	enemy := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 8, Y: 0, Radius: 18}
	w.Handle(ctx, unit.Sense{Time: 1, Self: selfAt(120, 0), Nearby: []unit.Snapshot{moon, enemy}})
	if got := lastPhase(drain(out)); got != 1 {
		t.Fatalf("enemy touch phase=%v", got)
	}
}

func TestFullMoonStartsLockThenDash(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	w := bootedWolf()
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	moon := moonSnap(1)
	enemy := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 90, Y: 0, Radius: 18}

	for i := 0; i < fullMoon-1; i++ {
		touch(w, ctx, float64(i)+1, true)
		_ = drain(out)
		touch(w, ctx, float64(i)+1.5, false)
		_ = drain(out)
	}
	touch(w, ctx, 4, true)
	if w.phase != fullMoon || w.bite != biteLock || w.dashesLeft != wolfDashCount {
		t.Fatalf("phase=%d bite=%d left=%d", w.phase, w.bite, w.dashesLeft)
	}
	_ = drain(out)

	touch(w, ctx, 4.5, false)
	if w.bite != biteDash {
		t.Fatalf("after lock window bite=%d want dash", w.bite)
	}
	_ = drain(out)

	w.Handle(ctx, unit.Sense{Time: 4.6, Self: selfAt(10, 0), Nearby: []unit.Snapshot{moon, enemy}})
	if cmds := drain(out); hasPhase(cmds) {
		t.Fatalf("dash should lock phase: %v", cmds)
	}

	w.Handle(ctx, unit.Sense{Time: 4.7, Self: selfAt(40, 0), Nearby: []unit.Snapshot{moon, enemy}})
	cmds := drain(out)
	vel := lastVel(cmds)
	if vel == nil {
		t.Fatal("dash should set velocity")
	}
	if vel.VX <= 0 {
		t.Fatalf("dash vx=%v want locked +x", vel.VX)
	}
	if got := math.Hypot(vel.VX, vel.VY); abs(got-wolfDashSpeed) > 1e-6 {
		t.Fatalf("dash speed=%v want %v", got, wolfDashSpeed)
	}
}

func TestLockTracksThenDashFreezesAim(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	w := bootedWolf()
	w.phase = fullMoon
	w.bite = biteLock
	w.dashesLeft = wolfDashCount
	w.lockUntil = 1.1
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}

	w.Handle(ctx, unit.Sense{
		Time:   1.05,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 0, Y: 80, Radius: 18}},
	})
	if w.aimY <= 0 {
		t.Fatalf("lock should track +y aimY=%v", w.aimY)
	}
	if vel := lastVel(drain(out)); vel == nil || vel.VX != 0 || vel.VY != 0 {
		t.Fatalf("lock should stand still: %+v", vel)
	}

	w.Handle(ctx, unit.Sense{
		Time:   1.1,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 0, Y: 80, Radius: 18}},
	})
	if w.bite != biteDash {
		t.Fatalf("bite=%d want dash", w.bite)
	}
	cmds := drain(out)
	if p := lastPass(cmds); p == nil || !p.Hold {
		t.Fatalf("dash should take pass: %+v", p)
	}
	launch := lastVel(cmds)
	if launch == nil || launch.VY <= 0 || abs(launch.VX) > 50 {
		t.Fatalf("launch vel should follow lock +y: %+v", launch)
	}

	w.Handle(ctx, unit.Sense{
		Time:   1.2,
		Self:   selfAt(0, 20),
		Nearby: []unit.Snapshot{{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 80, Y: 0, Radius: 18}},
	})
	if abs(w.aimX) > 0.2 || w.aimY <= 0 {
		t.Fatalf("dash must not retarget, aim=(%v,%v)", w.aimX, w.aimY)
	}
	vel := lastVel(drain(out))
	if vel == nil || vel.VY <= 0 || abs(vel.VX) > 50 {
		t.Fatalf("dash vel should stay +y: %+v", vel)
	}
}

func TestDashIgnoresPersonKeepsGoing(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	w := dashingWolf()
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	w.Handle(ctx, unit.Collision{
		Time:  2,
		Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 40, Y: 0, Radius: 18},
		NX:    1,
	})
	if w.bite != biteDash || w.dashesLeft != wolfDashCount {
		t.Fatalf("person should not end dash bite=%d left=%d", w.bite, w.dashesLeft)
	}
	if cmds := drain(out); len(cmds) != 0 {
		t.Fatalf("collision should be ignored: %v", cmds)
	}
}

func TestWallEndsDashAndRelocks(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	w := dashingWolf()
	w.x, w.y = 200, 0
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	w.Handle(ctx, unit.WallHit{Time: 3, NX: 1, NY: 0})
	if w.bite != biteLock || w.dashesLeft != wolfDashCount-1 {
		t.Fatalf("bite=%d left=%d", w.bite, w.dashesLeft)
	}
	cmds := drain(out)
	if p := lastPass(cmds); p == nil || p.Hold {
		t.Fatalf("wall should drop pass: %+v", p)
	}
	vel := lastVel(cmds)
	if vel == nil || vel.VX != 0 || vel.VY != 0 {
		t.Fatalf("wall should stop for next lock: %+v", vel)
	}
	if abs(w.lockUntil-(3+wolfDashSeek)) > 1e-9 {
		t.Fatalf("relock until=%v want %v", w.lockUntil, 3+wolfDashSeek)
	}

	w.Handle(ctx, unit.WallHit{Time: 3, NX: 0, NY: 1})
	if w.dashesLeft != wolfDashCount-1 {
		t.Fatalf("second wall while locking should not count: left=%d", w.dashesLeft)
	}

	enemy := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 90, Y: 0, Radius: 18}
	w.Handle(ctx, unit.Sense{Time: 3.3, Self: selfAt(200, 0), Nearby: []unit.Snapshot{enemy}})
	if w.bite != biteLock {
		t.Fatalf("still seeking at 0.3s bite=%d", w.bite)
	}
	_ = drain(out)

	w.Handle(ctx, unit.Sense{Time: 3.35, Self: selfAt(200, 0), Nearby: []unit.Snapshot{enemy}})
	if w.bite != biteDash {
		t.Fatalf("dash after 0.35s seek bite=%d", w.bite)
	}
}

func TestFifthWallResetsNewMoon(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	w := dashingWolf()
	w.dashesLeft = 1
	w.phase = fullMoon
	w.slot = 0
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	w.Handle(ctx, unit.WallHit{Time: 8, NX: 1, NY: 0})
	if w.biting() || w.phase != 0 || w.dashesLeft != 0 {
		t.Fatalf("after fifth wall bite=%d phase=%d left=%d", w.bite, w.phase, w.dashesLeft)
	}
	cmds := drain(out)
	if got := lastPhase(cmds); got != 0 {
		t.Fatalf("phase fx=%v", got)
	}
	if p := lastPass(cmds); p == nil || p.Hold {
		t.Fatalf("reset should drop pass: %+v", p)
	}
	vel := lastVel(cmds)
	if vel == nil {
		t.Fatal("fifth wall should return to cruise speed")
	}
	if got := math.Hypot(vel.VX, vel.VY); abs(got-wolfSpeed) > 1e-6 {
		t.Fatalf("after bite speed=%v want cruise %v", got, wolfSpeed)
	}
	if vel.VX >= 0 {
		t.Fatalf("should bounce off +x wall, vx=%v", vel.VX)
	}
	rage := lastRage(cmds)
	if rage == nil || rage.Amount != 0 {
		t.Fatalf("rage end fx=%+v", rage)
	}
}

func TestMoonTouchIgnoredDuringBite(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	w := dashingWolf()
	w.phase = fullMoon
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	w.Handle(ctx, unit.Sense{Time: 2, Self: selfAt(10, 0), Nearby: []unit.Snapshot{moonSnap(1)}})
	if w.phase != fullMoon {
		t.Fatalf("phase=%d", w.phase)
	}
	if cmds := drain(out); hasPhase(cmds) {
		t.Fatalf("bite should lock phase: %v", cmds)
	}
}

func bootedWolf() *狼人 {
	return &狼人{booted: true}
}

func dashingWolf() *狼人 {
	return &狼人{booted: true, bite: biteDash, dashesLeft: wolfDashCount, aimX: 1, phase: fullMoon}
}

func selfAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindWolf, Role: unit.RoleFighter, Slot: 0,
		X: x, Y: y, Radius: wolfRadius,
	}
}

func moonSnap(owner uint64) unit.Snapshot {
	return unit.Snapshot{
		ID: 99, Kind: KindMoon, Role: unit.RoleHelper, OwnerID: owner, Slot: 0,
		X: 0, Y: 0, Radius: moonRadius,
	}
}

func touch(w *狼人, ctx unit.Context, t float64, inside bool) {
	x := 120.0
	if inside {
		x = 10
	}
	w.Handle(ctx, unit.Sense{Time: t, Self: selfAt(x, 0), Nearby: []unit.Snapshot{moonSnap(1)}})
}

func drain(out <-chan unit.Cmd) []unit.Cmd {
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

func mustSpawn(t *testing.T, cmds []unit.Cmd) unit.Spawn {
	t.Helper()
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == KindMoon {
			return s
		}
	}
	t.Fatalf("no moon spawn in %v", cmds)
	return unit.Spawn{}
}

func mustPhase(t *testing.T, cmds []unit.Cmd) unit.FX {
	t.Helper()
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == "phase" {
			return fx
		}
	}
	t.Fatalf("no phase fx in %v", cmds)
	return unit.FX{}
}

func lastPhase(cmds []unit.Cmd) float64 {
	n := -1.0
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == "phase" {
			n = fx.Amount
		}
	}
	return n
}

func lastRage(cmds []unit.Cmd) *unit.FX {
	var fx *unit.FX
	for _, c := range cmds {
		if f, ok := c.(unit.FX); ok && f.Name == "rage" {
			cp := f
			fx = &cp
		}
	}
	return fx
}

func lastVel(cmds []unit.Cmd) *unit.SetVelocity {
	var vel *unit.SetVelocity
	for _, c := range cmds {
		if v, ok := c.(unit.SetVelocity); ok {
			cp := v
			vel = &cp
		}
	}
	return vel
}

func lastPass(cmds []unit.Cmd) *unit.Pass {
	var p *unit.Pass
	for _, c := range cmds {
		if v, ok := c.(unit.Pass); ok {
			cp := v
			p = &cp
		}
	}
	return p
}

func hasPhase(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == "phase" {
			return true
		}
	}
	return false
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
