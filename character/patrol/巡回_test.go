package 巡回

import (
	"math"
	"strconv"
	"testing"

	"xqdj/internal/unit"
)

func patrolCommands(out chan unit.Cmd) []unit.Cmd {
	var cmds []unit.Cmd
	for len(out) > 0 {
		cmds = append(cmds, <-out)
	}
	return cmds
}

func markerFixture() (*标识物, unit.Context, unit.Sense, chan unit.Cmd) {
	out := make(chan unit.Cmd, 16)
	ctx := unit.Context{ID: 2, Kind: KindPatrolMarker, Out: out}
	b := unit.NewActor(KindPatrolMarker, unit.SpawnInfo{OwnerID: 1}).(*标识物)
	s := unit.Sense{
		Time:   10,
		Self:   unit.Snapshot{ID: 2, Kind: KindPatrolMarker, X: 31, Radius: 11},
		Nearby: []unit.Snapshot{{ID: 1, Kind: KindPatrol, Radius: 18}},
	}
	b.Handle(ctx, s)
	patrolCommands(out)
	return b, ctx, s, out
}

func TestMarkerStartsAtRestAndAcceleratesAlongThrowDirection(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	ctx := unit.Context{ID: 1, Kind: KindPatrol, Out: out, Stream: 7}.WithBirths()
	a := unit.NewActor(KindPatrol, unit.SpawnInfo{Stream: 7})
	a.Handle(ctx, unit.Sense{})
	a.Handle(ctx, unit.Sense{Time: 0.8})
	var spawn *unit.Spawn
	for _, cmd := range patrolCommands(out) {
		if c, ok := cmd.(unit.Spawn); ok {
			spawn = &c
		}
	}
	if spawn == nil {
		t.Fatal("marker not thrown")
	}
	if spawn.VX != 0 || spawn.VY != 0 {
		t.Fatalf("initial velocity=(%v,%v), want rest", spawn.VX, spawn.VY)
	}
	b := unit.NewActor(spawn.Kind, unit.SpawnInfo{OwnerID: spawn.OwnerID, Slot: spawn.Slot})
	ctx.ID, ctx.Kind = 2, KindPatrolMarker
	s := unit.Sense{
		Time:   1,
		Self:   unit.Snapshot{ID: 2, X: spawn.X, Y: spawn.Y, Radius: 11},
		Nearby: []unit.Snapshot{{ID: 1, Radius: 18}},
	}
	b.Handle(ctx, s)
	patrolCommands(out)
	s.Time += 1.0 / 60
	b.Handle(ctx, s)
	for _, cmd := range patrolCommands(out) {
		if v, ok := cmd.(unit.SetVelocity); ok {
			speed := math.Hypot(v.VX, v.VY)
			if math.Abs(speed-26/0.014/4/60) > 1e-8 || v.VX*spawn.X+v.VY*spawn.Y <= 0 {
				t.Fatalf("first acceleration=(%v,%v), launch offset=(%v,%v)", v.VX, v.VY, spawn.X, spawn.Y)
			}
			return
		}
	}
	t.Fatal("stationary marker never started moving")
}

func TestMarkerDamageWindowAndFourSecondSpeed(t *testing.T) {
	for _, tc := range []struct{ seconds, damage float64 }{{0, 8}, {1, 8}, {2, 13}, {3, 19.5}, {4, 26}, {5, 26}} {
		t.Run(strconv.Itoa(int(tc.seconds))+" seconds", func(t *testing.T) {
			b, ctx, s, out := markerFixture()
			for frame := 1; frame <= int(tc.seconds*60); frame++ {
				s.Time = 10 + float64(frame)/60
				b.Handle(ctx, s)
				for _, cmd := range patrolCommands(out) {
					if v, ok := cmd.(unit.SetVelocity); ok {
						s.Self.VX, s.Self.VY = v.VX, v.VY
					}
				}
			}
			if tc.seconds == 4 && math.Abs(math.Hypot(s.Self.VX, s.Self.VY)-26/0.014) > 1e-6 {
				t.Fatalf("4-second speed=(%v,%v), want %v", s.Self.VX, s.Self.VY, 26/0.014)
			}
			// 5 秒已经折返，等过命中间隔后仍应保持 26 封顶。
			b.Handle(ctx, unit.Collision{Time: s.Time + 0.3, Other: unit.Snapshot{ID: 9, Role: unit.RoleFighter, Slot: 1}})
			for _, cmd := range patrolCommands(out) {
				if damage, ok := cmd.(unit.Damage); ok {
					if math.Abs(damage.Amount-tc.damage) > 1e-6 {
						t.Fatalf("damage=%v, want %v", damage.Amount, tc.damage)
					}
					return
				}
			}
			t.Fatal("no collision damage")
		})
	}
}

func TestMarkerForcesReturnAtFiveSeconds(t *testing.T) {
	b, ctx, s, out := markerFixture()
	s.Self.X = 200
	s.Time = 14.99
	b.Handle(ctx, s)
	patrolCommands(out)
	if b.returning {
		t.Fatal("returned before 5 seconds")
	}
	s.Time = 15
	b.Handle(ctx, s)
	patrolCommands(out)
	if !b.returning {
		t.Fatal("did not return at 5 seconds")
	}
	s.Time += 1.0 / 60
	b.Handle(ctx, s)
	for _, cmd := range patrolCommands(out) {
		if v, ok := cmd.(unit.SetVelocity); ok && v.VX < 0 {
			return
		}
	}
	t.Fatal("forced return did not head toward owner")
}

func TestMarkerReturnHasNoTimeoutAndMustActuallyReachOwner(t *testing.T) {
	b, ctx, s, out := markerFixture()
	b.flip(s.Time)
	b.speed = 100000 // 理论单拍行程已经超过敌我距离，也不能隔墙接回。
	s.Self.X = 200
	s.Nearby[0].X = -200
	s.Walls = []unit.WallView{{ID: 7, X1: 0, Y1: -100, X2: 0, Y2: 100, Radius: 6}}
	for frame := 1; frame <= 8*60; frame++ {
		s.Time = 10 + float64(frame)/60
		b.Handle(ctx, s)
		for _, cmd := range patrolCommands(out) {
			if _, ok := cmd.(unit.Despawn); ok {
				t.Fatalf("marker disappeared while still blocked at return age %v", s.Time-10)
			}
		}
	}
	// 真正回到主人身边（包括完全重合）仍然接回。
	s.Self.X = s.Nearby[0].X
	s.Time += 1.0 / 60
	b.Handle(ctx, s)
	for _, cmd := range patrolCommands(out) {
		if c, ok := cmd.(unit.Despawn); ok && c.UnitID == ctx.ID {
			return
		}
	}
	t.Fatal("marker at owner was not caught")
}

func TestPatrolOwnedPresenceControlsWaitingAndCooldownDuringStun(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	ctx := unit.Context{ID: 1, Kind: KindPatrol, Out: out}
	a := &巡回{rng: unit.NewRNG(7)}
	a.Handle(ctx, unit.OwnedPresenceChanged{Time: 1, Kind: KindPatrolMarker, Present: true})
	patrolCommands(out)
	a.Handle(ctx, unit.OwnedPresenceChanged{Time: 2, Kind: KindPatrolMarker, SelfStunned: true})
	if a.waiting || !a.resumePending || a.nextThrowAt != 7 {
		t.Fatalf("stunned removal state: %+v", a)
	}
	for _, cmd := range patrolCommands(out) {
		if _, ok := cmd.(unit.SetVelocity); ok {
			t.Fatal("resumed movement before stun ended")
		}
	}
	// 恢复 Sense 代表眩晕已结束；冷却仍从第 2 秒离场时刻算，不从第 4 秒算。
	a.Handle(ctx, unit.Sense{Time: 4})
	moving := false
	for _, cmd := range patrolCommands(out) {
		if v, ok := cmd.(unit.SetVelocity); ok && math.Abs(math.Hypot(v.VX, v.VY)-155) < 1e-6 {
			moving = true
		}
	}
	if !moving {
		t.Fatal("did not resume cruising after stun")
	}
	a.Handle(ctx, unit.Sense{Time: 6.99})
	if cmds := patrolCommands(out); len(cmds) != 0 {
		t.Fatalf("threw before 5-second cooldown: %v", cmds)
	}
	a.Handle(ctx, unit.Sense{Time: 7})
	for _, cmd := range patrolCommands(out) {
		if _, ok := cmd.(unit.Spawn); ok {
			return
		}
	}
	t.Fatal("did not throw when cooldown ended")
}

func TestPatrolReappearingOwnedMarkerRestartsWaiting(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	ctx := unit.Context{ID: 1, Kind: KindPatrol, Out: out}
	a := &巡回{rng: unit.NewRNG(7)}
	a.Handle(ctx, unit.OwnedPresenceChanged{Time: 1, Kind: KindPatrolMarker, Present: true})
	patrolCommands(out)
	a.Handle(ctx, unit.OwnedPresenceChanged{Time: 2, Kind: KindPatrolMarker})
	patrolCommands(out)
	a.Handle(ctx, unit.OwnedPresenceChanged{Time: 4, Kind: KindPatrolMarker, Present: true})
	patrolCommands(out)
	a.Handle(ctx, unit.Sense{Time: 20})
	if !a.waiting || len(patrolCommands(out)) != 0 {
		t.Fatal("reappearing marker did not prevent another throw")
	}
	a.Handle(ctx, unit.OwnedPresenceChanged{Time: 20, Kind: KindPatrolMarker})
	if a.waiting || a.nextThrowAt != 25 {
		t.Fatalf("cooldown did not restart from last removal: %+v", a)
	}
}

func TestMarkerAccelerationPreservesWallReflection(t *testing.T) {
	b, ctx, s, out := markerFixture()
	s.Time = 11
	s.Self.VX = -100
	b.Handle(ctx, s)
	for _, cmd := range patrolCommands(out) {
		if v, ok := cmd.(unit.SetVelocity); ok && v.VX < 0 && math.Abs(v.VY) < 1e-9 {
			return
		}
	}
	t.Fatal("acceleration lost reflected direction")
}
