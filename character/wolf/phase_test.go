package 狼人

import (
	"math"
	"testing"
	"xqdj/internal/unit"
	_ "xqdj/map/hex"
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
	if sp.Kind != KindMoon || sp.OwnerID != 1 {
		t.Fatalf("spawn=%+v", sp)
	}
	if !unit.HexField().Walkable(sp.X, sp.Y, moonRadius) {
		t.Fatalf("moon (%v,%v) not walkable", sp.X, sp.Y)
	}
	ph := mustPhase(t, cmds)
	if ph.Amount != 0 || ph.UnitID != 1 {
		t.Fatalf("initial phase=%+v", ph)
	}
}

func TestMoonSpawnVariesAcrossBoots(t *testing.T) {
	f := unit.HexField()
	seen := map[[2]int]bool{}
	for i := 0; i < 8; i++ {
		out := make(chan unit.Cmd, 16)
		w := &狼人{}
		w.Handle(unit.Context{ID: 1, Kind: KindWolf, Out: out, Stream: uint64(i + 1)}, unit.Sense{
			Time: 0, Field: f, Self: selfAt(80, 0),
		})
		sp := mustSpawn(t, drain(out))
		if !f.Walkable(sp.X, sp.Y, moonRadius) {
			t.Fatalf("moon (%v,%v) not walkable", sp.X, sp.Y)
		}
		seen[[2]int{int(sp.X), int(sp.Y)}] = true
	}
	if len(seen) < 2 {
		t.Fatal("moon position pinned to wolf id")
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
	enemy := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 8, Y: 0, Radius: 18, AimPriority: unit.DefaultFighterAim}
	w.Handle(ctx, unit.Sense{Time: 1, Self: selfAt(120, 0), Nearby: []unit.Snapshot{moon, enemy}})
	if got := lastPhase(drain(out)); got != 1 {
		t.Fatalf("enemy touch phase=%v", got)
	}
}

func TestMinionTouchAdvancesPhase(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	w := bootedWolf()
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	moon := moonSnap(1)
	fighter := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 200, Y: 0, Radius: 18, AimPriority: unit.DefaultFighterAim}
	minion := unit.Snapshot{ID: 3, Role: unit.RoleMinion, Mortal: true, Slot: 1, X: 8, Y: 0, Radius: 14, AimPriority: unit.DefaultMortalAim}
	w.Handle(ctx, unit.Sense{Time: 1, Self: selfAt(120, 0), Nearby: []unit.Snapshot{moon, fighter, minion}})
	if got := lastPhase(drain(out)); got != 1 {
		t.Fatalf("minion touch with fighter on field phase=%v", got)
	}

	w.Handle(ctx, unit.Sense{Time: 1.1, Self: selfAt(120, 0), Nearby: []unit.Snapshot{moon, fighter, minion}})
	if cmds := drain(out); hasPhase(cmds) {
		t.Fatalf("minion stay should not advance: %v", cmds)
	}
}

func TestFighterAndMinionEachCount(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	w := bootedWolf()
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	moon := moonSnap(1)
	fighter := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 8, Y: 0, Radius: 18, AimPriority: unit.DefaultFighterAim}
	minion := unit.Snapshot{ID: 3, Role: unit.RoleMinion, Mortal: true, Slot: 1, X: 200, Y: 0, Radius: 14, AimPriority: unit.DefaultMortalAim}
	w.Handle(ctx, unit.Sense{Time: 1, Self: selfAt(120, 0), Nearby: []unit.Snapshot{moon, fighter, minion}})
	if got := lastPhase(drain(out)); got != 1 {
		t.Fatalf("fighter enter phase=%v", got)
	}
	minion.X = 8
	w.Handle(ctx, unit.Sense{Time: 1.1, Self: selfAt(120, 0), Nearby: []unit.Snapshot{moon, fighter, minion}})
	if got := lastPhase(drain(out)); got != 2 {
		t.Fatalf("minion enter while fighter inside phase=%v", got)
	}
}

func TestCloneDoesNotAdvancePhase(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	w := bootedWolf()
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	clone := unit.Snapshot{ID: 3, Role: unit.RoleClone, Slot: 1, X: 8, Y: 0, Radius: 18}
	w.Handle(ctx, unit.Sense{Time: 1, Self: selfAt(120, 0), Nearby: []unit.Snapshot{moonSnap(1), clone}})
	if cmds := drain(out); hasPhase(cmds) {
		t.Fatalf("clone should not count: %v", cmds)
	}
}

func TestFullMoonStartsLockThenDash(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	w := bootedWolf()
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	moon := moonSnap(1)
	enemy := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 90, Y: 0, Radius: 18, AimPriority: unit.DefaultFighterAim}

	for i := 0; i < fullMoon-1; i++ {
		touch(w, ctx, float64(i)+1, true)
		_ = drain(out)
		touch(w, ctx, float64(i)+1.5, false)
		_ = drain(out)
	}
	touch(w, ctx, 4, true)
	if w.phase != fullMoon || w.bite != biteLock || abs(w.biteUntil-(4+wolfBiteTime)) > 1e-9 {
		t.Fatalf("phase=%d bite=%d until=%v", w.phase, w.bite, w.biteUntil)
	}
	_ = drain(out)

	touch(w, ctx, 4.3, false)
	if w.bite != biteDash {
		t.Fatalf("after 0.3s lock bite=%d want dash", w.bite)
	}
	_ = drain(out)

	w.Handle(ctx, unit.Sense{Time: 4.4, Self: selfAt(10, 0), Nearby: []unit.Snapshot{moon, enemy}})
	if cmds := drain(out); hasPhase(cmds) {
		t.Fatalf("dash should lock phase: %v", cmds)
	}

	w.Handle(ctx, unit.Sense{Time: 4.5, Self: selfAt(40, 0), Nearby: []unit.Snapshot{moon, enemy}})
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
	w.biteUntil = 100
	w.lockUntil = 1.1
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}

	w.Handle(ctx, unit.Sense{
		Time:   1.05,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 0, Y: 80, Radius: 18, AimPriority: unit.DefaultFighterAim}},
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
		Nearby: []unit.Snapshot{{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 0, Y: 80, Radius: 18, AimPriority: unit.DefaultFighterAim}},
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
		Nearby: []unit.Snapshot{{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 80, Y: 0, Radius: 18, AimPriority: unit.DefaultFighterAim}},
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
	out := make(chan unit.Cmd, 32)
	w := dashingWolf()
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	w.Handle(ctx, unit.Collision{
		Time:  2,
		Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 40, Y: 0, Radius: 18},
		NX:    1,
	})
	if w.bite != biteDash {
		t.Fatalf("person should not end dash bite=%d", w.bite)
	}
	if cmds := drain(out); len(cmds) != 0 {
		t.Fatalf("collision should be ignored: %v", cmds)
	}
}

func TestNoTargetDashesLastAim(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	w := bootedWolf()
	w.bite = biteLock
	w.biteUntil = 100
	w.lockUntil = 1
	w.aimX = 1
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	w.Handle(ctx, unit.Sense{Time: 1, Self: selfAt(0, 0)})
	if w.bite != biteDash || w.aimX != 1 {
		t.Fatalf("bite=%d aimX=%v", w.bite, w.aimX)
	}
	vel := lastVel(drain(out))
	if vel == nil || vel.VX <= 0 {
		t.Fatalf("should dash last aim: %+v", vel)
	}
}

func TestWallEndsDashAndRelocks(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	w := dashingWolf()
	w.x, w.y = 200, 0
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	w.Handle(ctx, unit.WallHit{Time: 3, NX: 1, NY: 0})
	if w.bite != biteLock {
		t.Fatalf("bite=%d", w.bite)
	}
	cmds := drain(out)
	if p := lastPass(cmds); p == nil || !p.Hold {
		t.Fatalf("wall should keep pass for the rest of the bite: %+v", p)
	}
	vel := lastVel(cmds)
	if vel == nil || vel.VX != 0 || vel.VY != 0 {
		t.Fatalf("wall should stop for next lock: %+v", vel)
	}
	if abs(w.lockUntil-(3+wolfDashLock)) > 1e-9 {
		t.Fatalf("relock until=%v want %v", w.lockUntil, 3+wolfDashLock)
	}

	w.Handle(ctx, unit.WallHit{Time: 3, NX: 0, NY: 1})
	if w.bite != biteLock {
		t.Fatalf("second wall while locking should not dash: bite=%d", w.bite)
	}

	enemy := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 90, Y: 0, Radius: 18, AimPriority: unit.DefaultFighterAim}
	w.Handle(ctx, unit.Sense{Time: 3.2, Self: selfAt(200, 0), Nearby: []unit.Snapshot{enemy}})
	if w.bite != biteLock {
		t.Fatalf("still locking at 0.2s bite=%d", w.bite)
	}
	_ = drain(out)

	w.Handle(ctx, unit.Sense{Time: 3.3, Self: selfAt(200, 0), Nearby: []unit.Snapshot{enemy}})
	if w.bite != biteDash {
		t.Fatalf("dash after 0.3s lock bite=%d", w.bite)
	}
}

func TestDistanceEndsDashAndRelocks(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	w := dashingWolf()
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	w.Handle(ctx, unit.Sense{Time: 2, Self: selfAt(wolfDashLen, 0)})
	if w.bite != biteLock {
		t.Fatalf("full distance should relock bite=%d", w.bite)
	}
	if w.phase != fullMoon {
		t.Fatalf("phase=%d", w.phase)
	}
}

func TestClockDuringLockEndsBite(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	w := bootedWolf()
	w.bite = biteLock
	w.phase = fullMoon
	w.biteUntil = 1
	w.lockUntil = 5
	w.aimX = 1
	w.biteDmg = 9
	w.biteHits = 1
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	w.Handle(ctx, unit.Sense{Time: 1, Self: selfAt(0, 0)})
	if w.biting() || w.phase != 0 {
		t.Fatalf("bite=%d phase=%d", w.bite, w.phase)
	}
	if w.biteDmg != 9 || w.biteHits != 1 {
		t.Fatalf("dmg=%v hits=%d", w.biteDmg, w.biteHits)
	}
	vel := lastVel(drain(out))
	if vel == nil || vel.VX <= 0 {
		t.Fatalf("cruise along aim: %+v", vel)
	}
}

func TestClockDuringDashFinishesSegment(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	w := dashingWolf()
	w.biteUntil = 1
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	w.Handle(ctx, unit.Sense{Time: 2, Self: selfAt(40, 0)})
	if w.bite != biteDash || w.phase != fullMoon {
		t.Fatalf("tail should keep dashing bite=%d phase=%d", w.bite, w.phase)
	}
	_ = drain(out)
	w.Handle(ctx, unit.Sense{Time: 2.1, Self: selfAt(40+wolfDashLen, 0)})
	if w.biting() || w.phase != 0 {
		t.Fatalf("after the tail bite=%d phase=%d", w.bite, w.phase)
	}
	vel := lastVel(drain(out))
	if vel == nil || vel.VX <= 0 {
		t.Fatalf("distance end cruises along aim: %+v", vel)
	}
}

func TestWallAfterClockResetsNewMoon(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	w := dashingWolf()
	w.biteUntil = 1
	w.phase = fullMoon
	w.slot = 0
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	w.Handle(ctx, unit.WallHit{Time: 8, NX: 1, NY: 0})
	if w.biting() || w.phase != 0 {
		t.Fatalf("after wall bite=%d phase=%d", w.bite, w.phase)
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
		t.Fatal("wall should return to cruise speed")
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

func TestFanKeepOnlyWhileDashing(t *testing.T) {
	out := make(chan unit.Cmd, 4)
	w := dashingWolf()
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	w.Handle(ctx, unit.IncomingDamage{Token: 1, Amount: 10, ArcSpan: 1})
	if got := lastConfirm(drain(out)); got == nil || abs(got.Amount-4) > 1e-9 {
		t.Fatalf("dash fan=%+v", got)
	}
	w.Handle(ctx, unit.IncomingDamage{Token: 2, Amount: 10})
	if got := lastConfirm(drain(out)); got == nil || got.Amount != 10 {
		t.Fatalf("dash bullet=%+v", got)
	}
	w.bite = biteLock
	w.Handle(ctx, unit.IncomingDamage{Token: 3, Amount: 10, ArcSpan: 1})
	if got := lastConfirm(drain(out)); got == nil || got.Amount != 10 {
		t.Fatalf("lock fan=%+v", got)
	}
}

func TestBiteDamageStepsAndCarries(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}.WithBirths()
	w := bootedWolf()
	w.phase = fullMoon
	w.bite = biteLock
	w.biteUntil = 100
	w.lockUntil = 1
	w.biteDmg = wolfBiteBase
	hold := selfAt(0, 0)

	w.Handle(ctx, unit.Sense{Time: 1, Self: hold})
	sp := lastBiteSpawn(drain(out))
	if sp == nil || w.bite != biteDash {
		t.Fatalf("dash should spawn one bite arc, bite=%d spawn=%v", w.bite, sp != nil)
	}
	if got := newBiteArc(unit.SpawnInfo{Stream: sp.Stream, Data: sp.Data}); got.dmg != wolfBiteBase {
		t.Fatalf("first arc dmg=%v", got.dmg)
	}
	w.Handle(ctx, unit.Sense{Time: 1.05, Self: hold, Nearby: []unit.Snapshot{biteArcSnap()}})
	if lastBiteSpawn(drain(out)) != nil {
		t.Fatal("one dash should not spawn a second arc")
	}
	w.Handle(ctx, unit.Sense{Time: 1.1, Self: hold})
	if w.biteHits != 1 || w.biteDmg != wolfBiteBase {
		t.Fatalf("after first hit hits=%d dmg=%v", w.biteHits, w.biteDmg)
	}
	if lastBiteSpawn(drain(out)) != nil {
		t.Fatal("a hit should not rearm during the same dash")
	}
	w.Handle(ctx, unit.Sense{Time: 1.2, Self: selfAt(wolfDashLen, 0)})
	if w.bite != biteLock {
		t.Fatalf("distance should end the dash bite=%d", w.bite)
	}
	_ = drain(out)
	w.Handle(ctx, unit.Sense{Time: 1.5, Self: hold})
	sp = lastBiteSpawn(drain(out))
	if sp == nil {
		t.Fatal("next dash should spawn the second arc")
	}
	if got := newBiteArc(unit.SpawnInfo{Stream: sp.Stream, Data: sp.Data}); got.dmg != wolfBiteBase {
		t.Fatalf("second arc dmg=%v", got.dmg)
	}
	w.Handle(ctx, unit.Sense{Time: 1.55, Self: hold, Nearby: []unit.Snapshot{biteArcSnap()}})
	_ = drain(out)
	w.Handle(ctx, unit.Sense{Time: 1.6, Self: hold})
	if w.biteHits != 0 || w.biteDmg != wolfBiteBase+1 {
		t.Fatalf("after second hit hits=%d dmg=%v", w.biteHits, w.biteDmg)
	}
	_ = drain(out)

	w.biteUntil = 2
	w.bite = biteLock
	w.lockUntil = 9
	w.Handle(ctx, unit.Sense{Time: 2, Self: hold})
	if w.biting() || w.biteDmg != wolfBiteBase+1 || w.biteHits != 0 {
		t.Fatalf("finish dmg=%v hits=%d biting=%v", w.biteDmg, w.biteHits, w.biting())
	}
	_ = drain(out)

	w.biteHits = 1
	w.biteDmg = wolfBiteBase
	w.phase = fullMoon - 1
	touch(w, ctx, 3, true)
	if w.bite != biteLock || w.biteHits != 1 || w.biteDmg != wolfBiteBase {
		t.Fatalf("next round bite=%d hits=%d dmg=%v", w.bite, w.biteHits, w.biteDmg)
	}
	_ = drain(out)
	w.lockUntil = 3
	w.Handle(ctx, unit.Sense{Time: 3.3, Self: hold})
	_ = drain(out)
	w.Handle(ctx, unit.Sense{Time: 3.35, Self: hold, Nearby: []unit.Snapshot{biteArcSnap()}})
	_ = drain(out)
	w.Handle(ctx, unit.Sense{Time: 3.4, Self: hold})
	if w.biteHits != 0 || w.biteDmg != wolfBiteBase+1 {
		t.Fatalf("carried odd hit hits=%d dmg=%v", w.biteHits, w.biteDmg)
	}
}

func TestMoonTouchIgnoredDuringBite(t *testing.T) {
	out := make(chan unit.Cmd, 32)
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

func TestHowlAfterTenDrySeconds(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	w := bootedWolf()
	w.phaseAt = 0
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}

	w.Handle(ctx, unit.Sense{Time: 9.9, Self: selfAt(0, 0)})
	if cmds := drain(out); howlOf(cmds) != nil || w.howling {
		t.Fatal("未满 10 秒不该嗷唔")
	}

	w.Handle(ctx, unit.Sense{Time: 10, Self: selfAt(3, 4)})
	cmds := drain(out)
	fx := howlOf(cmds)
	if fx == nil || fx.X != 3 || fx.Y != 4 {
		t.Fatalf("howl=%+v", fx)
	}
	if w.phase != 0 || !w.howling {
		t.Fatalf("phase=%d howling=%v", w.phase, w.howling)
	}
	if vel := lastVel(cmds); vel == nil || vel.VX != 0 || vel.VY != 0 {
		t.Fatalf("howl should stand: %+v", vel)
	}
	if s := lastStand(cmds); s == nil || !s.Hold {
		t.Fatalf("stand=%+v", s)
	}

	w.Handle(ctx, unit.Sense{Time: 10.9, Self: selfAt(3, 4)})
	if cmds := drain(out); howlOf(cmds) != nil || w.phase != 0 {
		t.Fatal("这一秒里不该再嚎，也不该提前加月相")
	}

	w.Handle(ctx, unit.Sense{Time: 11, Self: selfAt(3, 4)})
	cmds = drain(out)
	if w.howling || w.phase != 1 {
		t.Fatalf("howling=%v phase=%d", w.howling, w.phase)
	}
	if howlOf(cmds) != nil {
		t.Fatal("加月相那拍不该再飘一次")
	}
	if s := lastStand(cmds); s == nil || s.Hold {
		t.Fatalf("howl should release stand: %+v", s)
	}
}

func TestHowlIntoFullMoonStartsBite(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	w := bootedWolf()
	w.phase = fullMoon - 1
	w.phaseAt = 0
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	w.Handle(ctx, unit.Sense{Time: 10, Self: selfAt(0, 0)})
	_ = drain(out)
	w.Handle(ctx, unit.Sense{Time: 11, Self: selfAt(0, 0)})
	if w.phase != fullMoon || w.bite != biteLock {
		t.Fatalf("phase=%d bite=%d", w.phase, w.bite)
	}
}

func TestTouchDuringHowlDoesNotDouble(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	w := bootedWolf()
	w.phaseAt = 0
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	w.Handle(ctx, unit.Sense{Time: 10, Self: selfAt(80, 0)})
	_ = drain(out)
	touch(w, ctx, 10.2, true)
	if w.howling || w.phase != 1 {
		t.Fatalf("howling=%v phase=%d", w.howling, w.phase)
	}
}

func TestNoHowlDuringBite(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	w := dashingWolf()
	w.phaseAt = 0
	ctx := unit.Context{ID: 1, Kind: KindWolf, Out: out}
	w.Handle(ctx, unit.Sense{Time: 20, Self: selfAt(0, 0)})
	if w.howling || w.phase != fullMoon {
		t.Fatalf("howling=%v phase=%d", w.howling, w.phase)
	}
	if howlOf(drain(out)) != nil {
		t.Fatal("撕咬里不该嗷唔")
	}
}

func bootedWolf() *狼人 {
	return &狼人{booted: true}
}

func dashingWolf() *狼人 {
	return &狼人{
		booted: true, bite: biteDash, aimX: 1, phase: fullMoon,
		biteUntil: 100, dashLeft: wolfDashLen,
	}
}

func biteArcSnap() unit.Snapshot {
	return unit.Snapshot{ID: 7, Kind: KindWolfBite, OwnerID: 1, Role: unit.RoleProjectile}
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

func lastConfirm(cmds []unit.Cmd) *unit.ConfirmDamage {
	var d *unit.ConfirmDamage
	for _, c := range cmds {
		if v, ok := c.(unit.ConfirmDamage); ok {
			cp := v
			d = &cp
		}
	}
	return d
}

func lastBiteSpawn(cmds []unit.Cmd) *unit.Spawn {
	var sp *unit.Spawn
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == KindWolfBite {
			cp := s
			sp = &cp
		}
	}
	return sp
}

func howlOf(cmds []unit.Cmd) *unit.FX {
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == "howl" {
			cp := fx
			return &cp
		}
	}
	return nil
}

func lastStand(cmds []unit.Cmd) *unit.Stand {
	var s *unit.Stand
	for _, c := range cmds {
		if v, ok := c.(unit.Stand); ok {
			cp := v
			s = &cp
		}
	}
	return s
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
