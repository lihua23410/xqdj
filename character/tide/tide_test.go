package tide

import (
	"encoding/json"
	"math"
	"testing"

	"xqdj/internal/unit"
)

func drainCmds(out chan unit.Cmd) []unit.Cmd {
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

func TestTideHitsEnemiesOnBothSidesAndSharesCooldown(t *testing.T) {
	f := unit.NewActor(KindTide, unit.SpawnInfo{Stream: 13})
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Out: out}
	sense := unit.Sense{
		Self: unit.Snapshot{ID: 1, Radius: bodyRadius, VX: 160},
		Nearby: []unit.Snapshot{
			{ID: 2, Slot: 1, Y: 35, Radius: 6, Role: unit.RoleFighter},
			{ID: 3, Slot: 1, Y: -35, Radius: 6, Mortal: true},
			{ID: 4, Slot: 0, Y: 35, Radius: 6, Role: unit.RoleFighter},
			{ID: 5, Slot: 1, Y: 35, Radius: 6, Role: unit.RoleProjectile},
		},
	}
	f.Handle(ctx, sense)
	cmds := drainCmds(out)
	wantDmg := clamp(1.7+160.0/175.0, 1.7, 3.6)

	checkTarget := func(at float64, id uint64, side float64, list []unit.Cmd) {
		t.Helper()
		var gotDmg *unit.Damage
		var gotSlow *unit.AddFSComponent
		var gotForce *unit.Force
		for _, c := range list {
			switch v := c.(type) {
			case unit.Damage:
				if v.To == id {
					vv := v
					gotDmg = &vv
				}
			case unit.AddFSComponent:
				if v.UnitID == id {
					vv := v
					gotSlow = &vv
				}
			case unit.Force:
				if v.UnitID == id {
					vv := v
					gotForce = &vv
				}
			}
		}
		if gotDmg == nil || math.Abs(gotDmg.Amount-wantDmg) > 1e-9 || !gotDmg.NoFreeze {
			t.Fatalf("target %d wrong damage: %+v (want %.4f)", id, gotDmg, wantDmg)
		}
		if gotSlow == nil || gotSlow.Zone != unit.FSZoneM || gotSlow.Value != slowFactor || math.Abs(gotSlow.ExpiresAt-(at+slowDuration)) > 1e-9 {
			t.Fatalf("target %d wrong slow: %+v", id, gotSlow)
		}
		if gotForce == nil || math.Abs(gotForce.AX-pushFwdForce) > 1e-9 || math.Abs(gotForce.AY-side*pushOutForce) > 1e-9 {
			t.Fatalf("target %d wrong force: %+v", id, gotForce)
		}
	}

	checkTarget(0, 2, 1, cmds)
	checkTarget(0, 3, -1, cmds)
	for _, c := range cmds {
		if d, ok := c.(unit.Damage); ok && (d.To == 4 || d.To == 5) {
			t.Fatalf("water hit an ally or projectile: %+v", d)
		}
	}

	sense.Nearby = sense.Nearby[:1]
	sense.Nearby[0].Y = -35
	sense.Time = .37
	f.Handle(ctx, sense)
	for _, c := range drainCmds(out) {
		if _, ok := c.(unit.Damage); ok {
			t.Fatal("switching sides bypassed the hit interval")
		}
	}

	sense.Time = .38
	f.Handle(ctx, sense)
	checkTarget(.38, 2, -1, drainCmds(out))
}

func TestTideWallShadowStopsOnlyTheExposedSide(t *testing.T) {
	frame := waterFrame{forward: point{x: 1}}
	water := newWater(18, 60)
	wall := unit.WallView{X1: -60, X2: 60, Y1: 36, Y2: 36, Radius: 3, Square: true}
	water.clip(frame, wall)
	if water.hitSide(point{0, 52}, 5) != 0 {
		t.Fatal("water hit an enemy behind a square wall")
	}
	if water.hitSide(point{0, 30}, 2) != 1 || water.hitSide(point{0, -52}, 5) != -1 {
		t.Fatal("wall should leave water in front of it and on the other side")
	}

	wall.X2, wall.Radius = -19, 5
	water = newWater(18, 60)
	water.clip(frame, wall)
	if water.hitSide(point{-17, 52}, .5) != 1 {
		t.Fatal("water failed to pass the square wall's endpoint")
	}
	wall.Square = false
	water.clip(frame, wall)
	if water.hitSide(point{-17, 52}, .5) != 0 {
		t.Fatal("the rounded wall cap should block water")
	}
}

func TestTideBlockedByWall(t *testing.T) {
	sq := []unit.WallView{{X1: 0, Y1: -50, X2: 0, Y2: 50, Radius: 4, Square: true}}
	if !blockedByWall(-20, 0, 20, 0, sq) {
		t.Fatal("expected square wall to block cross segment")
	}
	if blockedByWall(-20, 60, 20, 60, sq) {
		t.Fatal("did not expect segment past wall endpoint to be blocked")
	}
	cap := []unit.WallView{{X1: 0, Y1: -50, X2: 0, Y2: 50, Radius: 6, Square: false}}
	if !blockedByWall(-20, 54, 20, 54, cap) {
		t.Fatal("expected capsule endcap to block segment within radius")
	}
	if blockedByWall(-20, 0, -5, 0, sq) {
		t.Fatal("open space on same side should not be blocked")
	}
}

func TestTideWakeDropSurfAndUndertow(t *testing.T) {
	a := unit.NewActor(KindTide, unit.SpawnInfo{Stream: 21}).(*fighter)
	out := make(chan unit.Cmd, 64)
	ctx := unit.Context{ID: 1, Out: out}

	// Step 1: initialize position at t=0.
	a.Handle(ctx, unit.Sense{Time: 0, Self: unit.Snapshot{ID: 1, Radius: bodyRadius, X: 0, Y: 0, VX: 200}})
	drainCmds(out)

	// Step 2: move 30 px -> drops 1 wake segment.
	a.Handle(ctx, unit.Sense{Time: 0.15, Self: unit.Snapshot{ID: 1, Radius: bodyRadius, X: 30, Y: 0, VX: 200}})
	drainCmds(out)
	if len(a.wake) != 1 {
		t.Fatalf("expected 1 wake segment, got %d", len(a.wake))
	}

	// Step 3: teleport jump (>90 px) -> resets origin without bridging wake.
	a.Handle(ctx, unit.Sense{Time: 0.20, Self: unit.Snapshot{ID: 1, Radius: bodyRadius, X: 200, Y: 0, VX: 200}})
	drainCmds(out)
	if len(a.wake) != 1 {
		t.Fatalf("expected teleport jump not to create wake, got %d", len(a.wake))
	}

	// Step 4: fresh wake (<0.55s) should NOT trigger surf even if overlapping.
	a.Handle(ctx, unit.Sense{Time: 0.30, Self: unit.Snapshot{ID: 1, Radius: bodyRadius, X: 15, Y: 0, VX: 200}})
	for _, c := range drainCmds(out) {
		if fx, ok := c.(unit.FX); ok && fx.Name == "tide-surf" {
			t.Fatal("fresh wake triggered surf before surfMinAge")
		}
	}

	// Step 5: enemy overlapping wake at t=0.50 receives undertow + wake damage.
	a.Handle(ctx, unit.Sense{
		Time: 0.50,
		Self: unit.Snapshot{ID: 1, Radius: bodyRadius, X: 200, Y: 0, VX: 200},
		Nearby: []unit.Snapshot{
			{ID: 2, Slot: 1, X: 15, Y: 32, Radius: 8, Role: unit.RoleFighter},
		},
	})
	cmds := drainCmds(out)
	var wakeDmg, undertow bool
	for _, c := range cmds {
		if d, ok := c.(unit.Damage); ok && d.To == 2 && math.Abs(d.Amount-wakeDamage) < 1e-9 {
			wakeDmg = true
		}
		if f, ok := c.(unit.Force); ok && f.UnitID == 2 && f.AY < 0 {
			undertow = true
		}
	}
	if !wakeDmg || !undertow {
		t.Fatalf("expected wake damage (%v) and inward undertow (%v)", wakeDmg, undertow)
	}

	// Step 6: self overlaps old wake (>=0.55s) -> triggers surf + splashes enemy on that segment.
	a.Handle(ctx, unit.Sense{
		Time: 0.75,
		Self: unit.Snapshot{ID: 1, Radius: bodyRadius, X: 15, Y: 0, VX: 200},
		Nearby: []unit.Snapshot{
			{ID: 4, Slot: 1, X: 15, Y: 30, Radius: 10, Role: unit.RoleFighter},
		},
	})
	cmds = drainCmds(out)
	var surfFX, surfKick, surfSplash bool
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == "tide-surf" {
			surfFX = true
		}
		if fc, ok := c.(unit.Force); ok && fc.UnitID == 1 && fc.AX > 0 {
			surfKick = true
		}
		if d, ok := c.(unit.Damage); ok && d.To == 4 && math.Abs(d.Amount-surfSplashDmg) < 1e-9 {
			surfSplash = true
		}
	}
	if !surfFX || !surfKick || !surfSplash {
		t.Fatalf("expected surfFX=%v surfKick=%v surfSplash=%v", surfFX, surfKick, surfSplash)
	}
}

func TestTideEbbPullLaunchAndStunCancel(t *testing.T) {
	a := unit.NewActor(KindTide, unit.SpawnInfo{Stream: 31}).(*fighter)
	out := make(chan unit.Cmd, 64)
	ctx := unit.Context{ID: 1, Out: out}

	a.Handle(ctx, unit.Sense{Time: 0, Self: unit.Snapshot{ID: 1, Radius: bodyRadius, X: 0, Y: 0, VX: 220}})
	drainCmds(out)

	// Sharp speed drop from 220 to 80 triggers Ebb.
	a.Handle(ctx, unit.Sense{
		Time: 0.05,
		Self: unit.Snapshot{ID: 1, Radius: bodyRadius, X: 5, Y: 0, VX: 80},
		Nearby: []unit.Snapshot{
			{ID: 2, Slot: 1, X: 70, Y: 0, Radius: 18, Role: unit.RoleFighter},
			{ID: 3, Slot: 1, X: -70, Y: 0, Radius: 18, Role: unit.RoleFighter},
		},
		Walls: []unit.WallView{
			{X1: -35, Y1: -40, X2: -35, Y2: 40, Radius: 4, Square: true},
		},
	})
	cmds := drainCmds(out)
	var ebbFX, initDmg2, pull2, hit3 bool
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == "tide-ebb" {
			ebbFX = true
		}
		if d, ok := c.(unit.Damage); ok && d.To == 2 {
			initDmg2 = true
		}
		if d, ok := c.(unit.Damage); ok && d.To == 3 {
			hit3 = true
		}
		if fc, ok := c.(unit.Force); ok && fc.UnitID == 2 && fc.AX < 0 {
			pull2 = true
		}
	}
	if !ebbFX || !initDmg2 || !pull2 || hit3 {
		t.Fatalf("ebbFX=%v initDmg2=%v pull2=%v hit3=%v", ebbFX, initDmg2, pull2, hit3)
	}

	// Finish Ebb at t=0.60 -> launches forward.
	a.Handle(ctx, unit.Sense{Time: 0.60, Self: unit.Snapshot{ID: 1, Radius: bodyRadius, X: 10, Y: 0, VX: 100}})
	cmds = drainCmds(out)
	var launchFX, launchForce bool
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == "tide-launch" {
			launchFX = true
		}
		if fc, ok := c.(unit.Force); ok && fc.UnitID == 1 && fc.AX > 0 {
			launchForce = true
		}
	}
	if !launchFX || !launchForce {
		t.Fatalf("expected launchFX=%v and launchForce=%v", launchFX, launchForce)
	}

	// Stunned cancels pending ebb.
	a.Handle(ctx, unit.Sense{Time: 2.00, Self: unit.Snapshot{ID: 1, Radius: bodyRadius, X: 10, Y: 0, VX: 220}})
	drainCmds(out)
	a.Handle(ctx, unit.Sense{Time: 2.05, Self: unit.Snapshot{ID: 1, Radius: bodyRadius, X: 12, Y: 0, VX: 80}})
	drainCmds(out)
	if !a.ebbActive {
		t.Fatal("expected ebbActive after second drop")
	}
	a.Handle(ctx, unit.Stunned{Time: 2.10, Until: 2.70})
	if a.ebbActive {
		t.Fatal("expected Stunned to cancel active ebb")
	}
}

func TestTideMomentumShieldAndShedSplash(t *testing.T) {
	a := unit.NewActor(KindTide, unit.SpawnInfo{Stream: 41}).(*fighter)
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Out: out}

	// Low speed <= 100: full confirm.
	a.Handle(ctx, unit.IncomingDamage{Token: 1, Amount: 20, Speed: 90})
	c := (<-out).(unit.ConfirmDamage)
	if math.Abs(c.Amount-20) > 1e-9 {
		t.Fatalf("low speed confirm = %v, want 20", c.Amount)
	}

	// Initialize speed before taking high-speed IncomingDamage.
	a.Handle(ctx, unit.Sense{Time: 0, Self: unit.Snapshot{ID: 1, Radius: bodyRadius, VX: 390}})
	drainCmds(out)

	// High speed 390: 26% reduction -> 74% taken (14.8).
	a.Handle(ctx, unit.IncomingDamage{Token: 2, Amount: 20, Speed: 390})
	c = (<-out).(unit.ConfirmDamage)
	if math.Abs(c.Amount-14.8) > 1e-9 {
		t.Fatalf("high speed confirm = %v, want 14.8", c.Amount)
	}

	// Sense applies braking force + shed splash, and expectedDrop prevents false Ebb.
	a.Handle(ctx, unit.Sense{
		Time: 0.05,
		Self: unit.Snapshot{ID: 1, Radius: bodyRadius, VX: 390},
		Nearby: []unit.Snapshot{
			{ID: 2, Slot: 1, X: 60, Y: 0, Radius: 12, Role: unit.RoleFighter},
		},
	})
	cmds := drainCmds(out)
	var brake, shedFX, splashDmg bool
	for _, cmd := range cmds {
		if fc, ok := cmd.(unit.Force); ok && fc.UnitID == 1 && fc.AX < 0 {
			brake = true
		}
		if fx, ok := cmd.(unit.FX); ok && fx.Name == "tide-shed" {
			shedFX = true
		}
		if d, ok := cmd.(unit.Damage); ok && d.To == 2 {
			splashDmg = true
		}
	}
	if !brake || !shedFX || !splashDmg {
		t.Fatalf("brake=%v shedFX=%v splashDmg=%v", brake, shedFX, splashDmg)
	}

	// Next tick after speed drops by expectedDrop should NOT trigger Ebb.
	nextSpeed := 390 - a.expectedDrop
	a.Handle(ctx, unit.Sense{Time: 0.10, Self: unit.Snapshot{ID: 1, Radius: bodyRadius, VX: nextSpeed}})
	for _, cmd := range drainCmds(out) {
		if fx, ok := cmd.(unit.FX); ok && fx.Name == "tide-ebb" {
			t.Fatal("momentum shedding must not trigger Ebb")
		}
	}
}

func TestTideStateFXPayloadAndRevisionCaching(t *testing.T) {
	a := unit.NewActor(KindTide, unit.SpawnInfo{Stream: 51}).(*fighter)
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Out: out}

	a.Handle(ctx, unit.Sense{Time: 0, Self: unit.Snapshot{ID: 1, Radius: bodyRadius, X: 0, VX: 150}})
	var first statePayload
	for _, cmd := range drainCmds(out) {
		if fx, ok := cmd.(unit.FX); ok && fx.Name == "tide-state" {
			if err := json.Unmarshal(fx.Data, &first); err != nil {
				t.Fatalf("unmarshal tide-state: %v", err)
			}
		}
	}
	if first.Rev != 1 {
		t.Fatalf("first rev = %d, want 1", first.Rev)
	}

	// Keyframe without state change keeps same rev.
	a.Handle(ctx, unit.Sense{Time: 0.25, Self: unit.Snapshot{ID: 1, Radius: bodyRadius, X: 10, VX: 150}})
	var second statePayload
	for _, cmd := range drainCmds(out) {
		if fx, ok := cmd.(unit.FX); ok && fx.Name == "tide-state" {
			_ = json.Unmarshal(fx.Data, &second)
		}
	}
	if second.Rev != 1 {
		t.Fatalf("unchanged keyframe bumped rev to %d", second.Rev)
	}

	// Dropping wake increments rev.
	a.Handle(ctx, unit.Sense{Time: 0.30, Self: unit.Snapshot{ID: 1, Radius: bodyRadius, X: 30, VX: 150}})
	var third statePayload
	for _, cmd := range drainCmds(out) {
		if fx, ok := cmd.(unit.FX); ok && fx.Name == "tide-state" {
			_ = json.Unmarshal(fx.Data, &third)
		}
	}
	if third.Rev != 2 || len(third.Segs) != 1 {
		t.Fatalf("expected rev=2 and 1 wake seg, got rev=%d segs=%d", third.Rev, len(third.Segs))
	}
}
