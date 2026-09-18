package 人偶使

import (
	"math"
	"testing"
	"xqdj/internal/unit"
)

func TestLockoutBlocksOtherSkillsAndRegen(t *testing.T) {
	a := fighter(SkillAtkFan)
	a.energy = 2
	out := make(chan unit.Cmd, 64)
	ctx := unit.Context{ID: 1, Kind: KindNingyushi, Out: out}
	enemy := foe(80, 0)

	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	if !hasSpawn(cmds, KindNingyushiDoll) {
		t.Fatalf("t=0 should spawn doll: %v", cmds)
	}
	if math.Abs(a.energy-2) > 1e-6 {
		t.Fatalf("auto should not spend energy, energy=%v", a.energy)
	}

	a.force = SkillPlace
	a.Handle(ctx, unit.Sense{Time: 0.3, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds = drain(out)
	if countSpawn(cmds, KindNingyushiDoll) != 0 {
		t.Fatalf("lockout should block place: %v", cmds)
	}
	if a.energy > 2+1e-6 {
		t.Fatalf("lockout should block regen, energy=%v", a.energy)
	}

	a.Handle(ctx, unit.Sense{Time: 1.05, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	_ = drain(out)
	if a.energy > 2+1e-6 {
		t.Fatalf("still locked, energy=%v", a.energy)
	}

	done := lockSpan(SkillAtkFan)
	a.Handle(ctx, unit.Sense{Time: done, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds = drain(out)
	if !hasSpawn(cmds, KindNingyushiDoll) {
		t.Fatalf("t=%v should allow place after lock: %v", done, cmds)
	}
	if math.Abs(a.energy-1) > 1e-6 {
		t.Fatalf("place should spend 1, energy=%v", a.energy)
	}
}

func TestEnergyRegensAfterLock(t *testing.T) {
	a := fighter(SkillNone)
	a.energy = 1
	out := make(chan unit.Cmd, 16)
	ctx := unit.Context{ID: 1, Kind: KindNingyushi, Out: out}
	enemy := foe(200, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{Time: 0.8, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	_ = drain(out)
	if math.Abs(a.energy-2) > 1e-6 {
		t.Fatalf("should regen +1 per 0.8s, energy=%v", a.energy)
	}
	a.Handle(ctx, unit.Sense{Time: 1.6, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	_ = drain(out)
	if math.Abs(a.energy-3) > 1e-6 {
		t.Fatalf("second tick +1, energy=%v", a.energy)
	}
}

func TestRegenRestartsAfterSkillLock(t *testing.T) {
	a := fighter(SkillNone)
	a.energy = 3
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindNingyushi, Out: out}
	far := foe(200, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{far}})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{Time: 0.5, Self: me(0, 0), Nearby: []unit.Snapshot{far}})
	_ = drain(out)
	a.force = SkillPlace
	a.Handle(ctx, unit.Sense{Time: 0.5, Self: me(0, 0), Nearby: []unit.Snapshot{far}})
	_ = drain(out)
	if math.Abs(a.energy-2) > 1e-6 {
		t.Fatalf("place should spend 1, energy=%v", a.energy)
	}
	a.force = SkillNone
	unlock := 0.5 + lockSpan(SkillPlace)
	a.Handle(ctx, unit.Sense{Time: unlock + 0.79, Self: me(0, 0), Nearby: []unit.Snapshot{far}})
	_ = drain(out)
	if math.Abs(a.energy-2) > 1e-6 {
		t.Fatalf("need a fresh 0.8s after lock, energy=%v", a.energy)
	}
	a.Handle(ctx, unit.Sense{Time: unlock + 0.8, Self: me(0, 0), Nearby: []unit.Snapshot{far}})
	_ = drain(out)
	if math.Abs(a.energy-3) > 1e-6 {
		t.Fatalf("regen +1 after idle 0.8s, energy=%v", a.energy)
	}
}

func TestWallDoesNotRefillEnergy(t *testing.T) {
	a := fighter(SkillNone)
	a.energy = 1.2
	out := make(chan unit.Cmd, 16)
	ctx := unit.Context{ID: 1, Kind: KindNingyushi, Out: out}
	a.Handle(ctx, unit.WallHit{Time: 1, NX: 1, NY: 0})
	if math.Abs(a.energy-1.2) > 1e-6 {
		t.Fatalf("wall should not restore energy, energy=%v", a.energy)
	}
}

func TestNoEnergyNoSkillCast(t *testing.T) {
	a := fighter(SkillFan)
	a.energy = 0.05
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindNingyushi, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	cmds := drain(out)
	if hasSpawn(cmds, KindNingyushiDoll) {
		t.Fatalf("no energy must not cast skill: %v", cmds)
	}
}

func TestFanHitsThenMisses(t *testing.T) {
	a := fighter(SkillAtkFan)
	out := make(chan unit.Cmd, 64)
	ctx := unit.Context{ID: 1, Kind: KindNingyushi, Out: out}
	enemy := foe(80, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	fx := lastFX(cmds, "fan")
	if fx == nil || math.Abs(fx.Amount-unit.Deg(atkFanDeg)) > 1e-6 {
		t.Fatalf("auto fan should show 120° zone: %v", cmds)
	}
	if math.Abs(fx.X-dollReach) > 1 || math.Abs(fx.Y) > 1 {
		t.Fatalf("fan origin should sit 54 toward enemy, got (%v,%v)", fx.X, fx.Y)
	}
	sp := lastSpawn(cmds, KindNingyushiDoll)
	if sp == nil || math.Hypot(sp.X, sp.Y) > 1 {
		t.Fatalf("doll should spawn on self, got %+v", sp)
	}
	a.Handle(ctx, unit.Sense{Time: 0.1, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds = drain(out)
	if !hasDamage(cmds, 2, atkFanDmg) {
		t.Fatalf("first fan tick should hit: %v", cmds)
	}
	far := foe(220, 0)
	a.Handle(ctx, unit.Sense{Time: 0.1 + 1.0/6, Self: me(0, 0), Nearby: []unit.Snapshot{far}})
	cmds = drain(out)
	if hasDamageTo(cmds, 2) {
		t.Fatalf("far enemy should miss remaining fan: %v", cmds)
	}
}

func TestRectKnockback(t *testing.T) {
	a := fighter(SkillAtkRect)
	out := make(chan unit.Cmd, 64)
	ctx := unit.Context{ID: 1, Kind: KindNingyushi, Out: out}
	enemy := foe(70, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{Time: 0.2, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	if !hasDamage(cmds, 2, atkRectDmg) {
		t.Fatalf("rect should deal 9.8: %v", cmds)
	}
	if !hasTeleportAway(cmds, 2, 70, knockDist) {
		t.Fatalf("rect should push ~108: %v", cmds)
	}
	st := lastStun(cmds)
	if st == nil || st.UnitID != 2 || !st.Hold {
		t.Fatalf("rect should stun 1s: %+v", st)
	}
	if lastFX(cmds, "stun") != nil {
		t.Fatalf("rect stun should have no vfx: %v", cmds)
	}
	a.Handle(ctx, unit.Sense{Time: 0.2 + rectStun, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	end := drain(out)
	rel := lastStun(end)
	if rel == nil || rel.Hold {
		t.Fatalf("stun should end at 1s: %+v", rel)
	}
}

func TestSpecial22NeedsIdleDoll(t *testing.T) {
	a := fighter(SkillN22)
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindNingyushi, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	if math.Abs(a.energy-energyMax) > 1e-6 {
		t.Fatalf("empty special 22 must not spend, energy=%v", a.energy)
	}
	doll := unit.Snapshot{ID: 9, Kind: KindNingyushiDoll, Role: unit.RoleHelper, OwnerID: 1, X: 40, Y: 0, Radius: dollRadius, Slot: 0}
	setState(9, dollIdle)
	t.Cleanup(func() { dropState(9) })
	a.Handle(ctx, unit.Sense{Time: 0.05, Self: me(0, 0), Nearby: []unit.Snapshot{foe(80, 0), doll}})
	if math.Abs(a.energy-(energyMax-1)) > 1e-6 {
		t.Fatalf("idle doll should allow special 22, energy=%v", a.energy)
	}
}

func TestNumberCardUpgradesSlot(t *testing.T) {
	a := fighter(SkillNone)
	a.hand = []uint8{SkillN22}
	a.deck = []uint8{}
	a.hx, a.hy = 1, 0
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindNingyushi, Out: out}
	a.locked = false
	a.Handle(ctx, unit.Sense{Time: 0, Self: meVX(0, 0, 80, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	_ = drain(out)
	if a.level[SkillN22] != 1 {
		t.Fatalf("first [22] should unlock level 1, lv=%v", a.level[SkillN22])
	}
	a.lockUntil = 0
	a.hand = []uint8{SkillN22}
	a.energy = energyMax
	a.Handle(ctx, unit.Sense{Time: 2, Self: meVX(0, 0, 80, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	_ = drain(out)
	if a.level[SkillN22] != 2 {
		t.Fatalf("second [22] should upgrade, lv=%v", a.level[SkillN22])
	}
}

func TestCardPickNotAlwaysFirst(t *testing.T) {
	seen := map[uint8]int{}
	for i := 0; i < 40; i++ {
		a := fighter(SkillNone)
		a.locked = false
		a.hand = []uint8{CardSpirit, CardDemon}
		a.deck = []uint8{}
		a.hx, a.hy = 1, 0
		out := make(chan unit.Cmd, 32)
		ctx := unit.Context{ID: 1, Kind: KindNingyushi, Out: out}
		a.Handle(ctx, unit.Sense{Time: 0, Self: meVX(0, 0, 40, 0), Nearby: []unit.Snapshot{foe(60, 0)}})
		_ = drain(out)
		if len(a.hand) != 1 {
			t.Fatalf("should consume 1 card, hand=%v", a.hand)
		}
		seen[a.hand[0]]++
	}
	if seen[CardSpirit] == 0 || seen[CardDemon] == 0 {
		t.Fatalf("should pick among usable cards, leftover=%v", seen)
	}
}

func TestChargeOnHitAndSpend(t *testing.T) {
	a := fighter(SkillPlace)
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindNingyushi, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{foe(200, 0)}})
	_ = drain(out)
	if math.Abs(a.progress-cardFill) > 1e-6 {
		t.Fatalf("spend 1 energy should charge 0.1, prog=%v", a.progress)
	}
	a.Handle(ctx, unit.IncomingDamage{Token: 1, From: 2, Amount: 10, Time: 0.01})
	_ = drain(out)
	if math.Abs(a.progress-3*cardFill) > 1e-6 {
		t.Fatalf("hit should charge 0.1 and guard spend 1 energy another 0.1, prog=%v", a.progress)
	}
}

func TestGuardIgnoresFacing(t *testing.T) {
	a := fighter(SkillNone)
	a.energy = 5
	a.hx, a.hy = 1, 0
	a.x, a.y = 0, 0
	a.nearby = []unit.Snapshot{foe(-80, 0)}
	out := make(chan unit.Cmd, 16)
	ctx := unit.Context{ID: 1, Kind: KindNingyushi, Out: out}
	a.Handle(ctx, unit.IncomingDamage{Token: 7, From: 2, Amount: 10, Time: 1})
	cmds := drain(out)
	if !hasConfirm(cmds, 7, 10*frontDR) {
		t.Fatalf("energy should guard from behind, got %v", cmds)
	}
	if math.Abs(a.energy-(5-10/frontCostDiv)) > 1e-6 {
		t.Fatalf("guard should spend energy, energy=%v", a.energy)
	}
	a.energy = 0.4
	a.Handle(ctx, unit.IncomingDamage{Token: 8, From: 2, Amount: 10, Time: 2})
	cmds = drain(out)
	if !hasConfirm(cmds, 8, 10*emptyHurt) {
		t.Fatalf("not enough energy should break: %v", cmds)
	}
	if a.energy > 1e-9 {
		t.Fatalf("break should dump leftover, energy=%v", a.energy)
	}
	a.Handle(ctx, unit.IncomingDamage{Token: 9, From: 2, Amount: 10, Time: 3})
	cmds = drain(out)
	if !hasConfirm(cmds, 9, 10*emptyHurt) {
		t.Fatalf("empty should keep vulnerability: %v", cmds)
	}
}

func TestDeckCounts(t *testing.T) {
	a := fighter(SkillNone)
	a.ensureRNG(unit.Context{ID: 1})
	a.initDeckLocked()
	if len(a.deck) != 20 {
		t.Fatalf("deck size=%d", len(a.deck))
	}
	n := map[uint8]int{}
	for _, sk := range a.deck {
		n[sk]++
	}
	want := map[uint8]int{SkillN26: 4, SkillN62: 2, SkillN24: 4, SkillN22: 2, CardDemon: 1, CardBattle: 2, CardHourai: 1, CardSpirit: 4}
	for sk, c := range want {
		if n[sk] != c {
			t.Fatalf("card %d count=%d want %d", sk, n[sk], c)
		}
	}
}

func TestN24N62LockedUntilCard(t *testing.T) {
	a := fighter(SkillN24)
	a.hx, a.hy = 1, 0
	out := make(chan unit.Cmd, 16)
	ctx := unit.Context{ID: 1, Kind: KindNingyushi, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0, Self: meVX(0, 0, 80, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	if hasSpawn(drain(out), KindNingyushiDoll) {
		t.Fatal("[24] should be empty before card")
	}
	a.level[SkillN24] = 1
	a.Handle(ctx, unit.Sense{Time: 0.2, Self: meVX(0, 0, 80, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	if !hasSpawn(drain(out), KindNingyushiDoll) {
		t.Fatal("unlocked [24] should spawn dolls")
	}
}

func TestSpell24RushesTowardEnemy(t *testing.T) {
	a := fighter(SkillN24)
	a.level[SkillN24] = 1
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindNingyushi, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	cmds := drain(out)
	if countSpawn(cmds, KindNingyushiDoll) != spell24N {
		t.Fatalf("[24] should gather 6 dolls, got %v", cmds)
	}
	vx := lastVel(cmds, 1)
	if vx == nil || vx.VX <= 0 {
		t.Fatalf("[24] should rush toward enemy, vel=%+v", vx)
	}
	a.Handle(ctx, unit.Sense{Time: 0.2, Self: me(20, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	cmds = drain(out)
	if !hasDamage(cmds, 2, float64(1+2)*1.4) {
		t.Fatalf("thrust in front should hit: %v", cmds)
	}
}

func TestSpell24CollisionDumpsRemaining(t *testing.T) {
	a := fighter(SkillN24)
	a.level[SkillN24] = 1
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindNingyushi, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	_ = drain(out)
	a.force = SkillNone
	full := float64(spell24Hits) * float64(1+2) * 1.4
	a.Handle(ctx, unit.Collision{Time: 0.1, Other: foe(36, 0), NX: 1, NY: 0})
	cmds := drain(out)
	if !hasDamage(cmds, 2, full) {
		t.Fatalf("body hit should dump remaining [24], want %v got %v", full, cmds)
	}
	if a.job.kind != 0 {
		t.Fatal("[24] should end after body hit")
	}
}

func TestSpell24EndsOnWall(t *testing.T) {
	a := fighter(SkillN24)
	a.level[SkillN24] = 1
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindNingyushi, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	_ = drain(out)
	a.force = SkillNone
	a.Handle(ctx, unit.WallHit{Time: 0.5, NX: 1, NY: 0})
	cmds := drain(out)
	if a.job.kind != 0 {
		t.Fatal("[24] should drop on wall")
	}
	if a.lockedOut(0.5) {
		t.Fatal("[24] lock should end on wall")
	}
	cr := lastCruise(cmds, 1)
	if cr == nil || math.Abs(cr.Speed-ningyushiSpeed) > 1e-6 {
		t.Fatalf("[24] should restore cruise on wall, cruise=%+v", cr)
	}
	a.Handle(ctx, unit.Sense{Time: 0.5, Self: me(200, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	cmds = drain(out)
	if v := lastVel(cmds, 1); v != nil && v.VX > ningyushiSpeed {
		t.Fatalf("[24] must not keep rushing after wall, vel=%+v", v)
	}
}

func TestCallOnSpecialAndSpell(t *testing.T) {
	a := fighter(SkillPlace)
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindNingyushi, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{foe(200, 0)}})
	if lastNamed(drain(out), "call") != nil {
		t.Fatal("unnamed skills should not shout")
	}
	a.force = SkillN26
	a.lockUntil = 0
	a.Handle(ctx, unit.Sense{Time: 2, Self: me(0, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	fx := lastNamed(drain(out), "call")
	if fx == nil || math.Abs(fx.Amount-float64(SkillN26)) > 1e-6 || fx.VY > 0 {
		t.Fatalf("special [26] should shout 人偶操创, fx=%+v", fx)
	}
	a.level[SkillN26] = 1
	a.lockUntil = 0
	a.Handle(ctx, unit.Sense{Time: 4, Self: me(0, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	fx = lastNamed(drain(out), "call")
	if fx == nil || math.Abs(fx.Amount-float64(SkillN26)) > 1e-6 || fx.VY < 1 {
		t.Fatalf("replaced [26] should shout 人偶归巢, fx=%+v", fx)
	}
}

func TestIdleDollHasNoRangeAndDoesNotRecall(t *testing.T) {
	resetQueues()
	pushDoll(dollSpec{mode: dollIdle})
	d := newDoll(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 9, Kind: KindNingyushiDoll, Out: out}
	owner := me(0, 0)
	d.Handle(ctx, unit.Sense{Time: 0, Self: dollSnap(9, 80, 0), Nearby: []unit.Snapshot{owner, foe(120, 0)}})
	cmds := drain(out)
	if rangeOn(cmds) {
		t.Fatalf("idle doll should have no range: %v", cmds)
	}
	d.Handle(ctx, unit.Sense{Time: 1, Self: dollSnap(9, 80, 0), Nearby: []unit.Snapshot{owner, foe(120, 0)}})
	cmds = drain(out)
	if hasTeleport(cmds, 9) {
		t.Fatalf("待命人偶 should stay put: %v", cmds)
	}
	dropState(9)
}

func TestStrikeDollRecallsAfterAttack(t *testing.T) {
	resetQueues()
	pushDoll(dollSpec{mode: dollStrike, recallAt: 1.1})
	d := newDoll(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 9, Kind: KindNingyushiDoll, Out: out}
	owner := me(0, 0)
	self := dollSnap(9, 108, 0)
	d.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{owner}})
	_ = drain(out)
	d.Handle(ctx, unit.Sense{Time: 1.1, Self: self, Nearby: []unit.Snapshot{owner}})
	d.Handle(ctx, unit.Sense{Time: 1.1 + 1.0/60, Self: self, Nearby: []unit.Snapshot{owner}})
	cmds := drain(out)
	tp := lastTeleport(cmds, 9)
	if tp == nil || tp.X >= 108 {
		t.Fatalf("strike doll should start recalling after attack: %v", cmds)
	}
	d.Handle(ctx, unit.Sense{Time: 1.1 + recallLife, Self: self, Nearby: []unit.Snapshot{owner}})
	if !hasDespawn(drain(out), 9) {
		t.Fatal("strike recall should finish in about 0.4s")
	}
	dropState(9)
}

func TestDollDeploysInPointTwo(t *testing.T) {
	resetQueues()
	pushDoll(dollSpec{mode: dollStrike, pose: poseAd, x: dollReach, y: 0, arriveAt: deployLife})
	d := newDoll(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 9, Kind: KindNingyushiDoll, Out: out}
	owner := me(0, 0)
	d.Handle(ctx, unit.Sense{Time: 0, Self: dollSnap(9, 0, 0), Nearby: []unit.Snapshot{owner}})
	tp := lastTeleport(drain(out), 9)
	if tp == nil || tp.X <= 0 || tp.X >= dollReach {
		t.Fatalf("doll should start flying toward dest, got %+v", tp)
	}
	d.Handle(ctx, unit.Sense{Time: deployLife, Self: dollSnap(9, tp.X, tp.Y), Nearby: []unit.Snapshot{owner}})
	tp = lastTeleport(drain(out), 9)
	if tp == nil || math.Abs(tp.X-dollReach) > 1 || math.Abs(tp.Y) > 1 {
		t.Fatalf("doll should arrive at dest in 0.2s, got %+v", tp)
	}
	dropState(9)
}

func TestEllipseDollShowsRange(t *testing.T) {
	resetQueues()
	pushDoll(dollSpec{mode: dollEllipse, armAt: 0.8, until: 3.8, rangeOn: true})
	d := newDoll(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 9, Kind: KindNingyushiDoll, Out: out}
	d.Handle(ctx, unit.Sense{Time: 0, Self: dollSnap(9, 54, 0), Nearby: []unit.Snapshot{me(0, 0), foe(80, 0)}})
	if rangeOn(drain(out)) {
		t.Fatal("ellipse should wait for windup")
	}
	d.Handle(ctx, unit.Sense{Time: 0.8, Self: dollSnap(9, 54, 0), Nearby: []unit.Snapshot{me(0, 0), foe(80, 0)}})
	if !rangeOn(drain(out)) {
		t.Fatal("armed ellipse doll should show range")
	}
	dropState(9)
}

func dollSnap(id uint64, x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: id, Kind: KindNingyushiDoll, Role: unit.RoleHelper, OwnerID: 1,
		X: x, Y: y, Radius: dollRadius, Slot: 0,
	}
}

func TestEllipseAxesStayHorizontal(t *testing.T) {
	resetQueues()
	pushDoll(dollSpec{mode: dollEllipse, ux: 0, uy: 1, armAt: 0, until: 4, rangeOn: true})
	d := newDoll(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 9, Kind: KindNingyushiDoll, Out: out}
	d.Handle(ctx, unit.Sense{Time: 0, Self: dollSnap(9, 54, 0), Nearby: []unit.Snapshot{me(0, 0), foe(80, 80)}})
	a1 := rangeAxes(drain(out))
	if a1 == nil || math.Abs(a1[0]-1) > 1e-6 || math.Abs(a1[1]) > 1e-6 {
		t.Fatalf("ellipse must stay screen-horizontal, axes=%v", a1)
	}
	dropState(9)
}

func TestEllipseHitsMortalMinion(t *testing.T) {
	resetQueues()
	pushDoll(dollSpec{mode: dollOrbit, ang: 0, until: 3, rangeOn: true})
	d := newDoll(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 9, Kind: KindNingyushiDoll, Out: out}
	self := dollSnap(9, 200, 0)
	d.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{me(0, 0), mortalMinion(230, 0)}})
	cmds := drain(out)
	if !hasDamage(cmds, 8, burstDmg) {
		t.Fatalf("ellipse should hit mortal minion: %v", cmds)
	}
	dropState(9)
}

func TestBattleAxeUsesEllipse(t *testing.T) {
	resetQueues()
	pushDoll(dollSpec{mode: dollOrbit, ang: 0, until: 3, rangeOn: true})
	d := newDoll(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 9, Kind: KindNingyushiDoll, Out: out}
	self := dollSnap(9, 200, 0)
	d.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{me(0, 0), foe(230, 0)}})
	cmds := drain(out)
	if !rangeOn(cmds) {
		t.Fatal("battle axe doll should show ellipse")
	}
	if !hasDamage(cmds, 2, burstDmg) {
		t.Fatalf("horizontal ellipse should hit: %v", cmds)
	}
	d.Handle(ctx, unit.Sense{Time: 0.2, Self: self, Nearby: []unit.Snapshot{me(0, 0), foe(200, 50)}})
	if hasDamageTo(drain(out), 2) {
		t.Fatal("vertical miss should stay outside horizontal ellipse")
	}
	dropState(9)
}

func TestBattleExpandsWithHit(t *testing.T) {
	resetQueues()
	pushDoll(dollSpec{
		mode: dollOrbit, ang: 0, x: orbitR, y: 0,
		arriveAt: orbitExpand, until: orbitExpand + orbitLife, rangeOn: true,
	})
	d := newDoll(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 9, Kind: KindNingyushiDoll, Out: out}
	d.Handle(ctx, unit.Sense{Time: 0, Self: dollSnap(9, 0, 0), Nearby: []unit.Snapshot{me(0, 0), foe(36, 0)}})
	cmds := drain(out)
	tp := lastTeleport(cmds, 9)
	if tp == nil || tp.X <= 0 || tp.X >= orbitR {
		t.Fatalf("battle dolls should fly out slowly, got %+v", tp)
	}
	if !rangeOn(cmds) {
		t.Fatal("expanding battle doll should already hit")
	}
	if !hasDamage(cmds, 2, burstDmg) {
		t.Fatalf("expanding battle doll should damage on the way: %v", cmds)
	}
	d.Handle(ctx, unit.Sense{Time: orbitExpand, Self: dollSnap(9, tp.X, 0), Nearby: []unit.Snapshot{me(0, 0), foe(230, 0)}})
	tp = lastTeleport(drain(out), 9)
	if tp == nil || math.Abs(tp.X-orbitR) > 1 {
		t.Fatalf("should reach the ring at 1.2s, got %+v", tp)
	}
	d.Handle(ctx, unit.Sense{Time: orbitExpand + orbitLife, Self: dollSnap(9, orbitR, 0), Nearby: []unit.Snapshot{me(0, 0)}})
	if !hasDespawn(drain(out), 9) {
		t.Fatal("battle duration should cover expand + orbit")
	}
	dropState(9)
}

func TestChaseGoesToLockedPoint(t *testing.T) {
	resetQueues()
	pushDoll(dollSpec{mode: dollIdle})
	d := newDoll(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 9, Kind: KindNingyushiDoll, Out: out}
	self := dollSnap(9, 0, 0)
	orderChase(9, 0.8, 3.8, 100, 0)
	t.Cleanup(func() { dropState(9) })
	d.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{me(0, 0), foe(100, 0)}})
	_ = drain(out)
	d.Handle(ctx, unit.Sense{Time: 0.8, Self: self, Nearby: []unit.Snapshot{me(0, 0), foe(100, 80)}})
	tp := lastTeleport(drain(out), 9)
	if tp == nil || tp.X <= 0 {
		t.Fatalf("chase should start toward locked point: %v", tp)
	}
	if math.Abs(tp.Y) > 1e-6 {
		t.Fatalf("chase must not follow live enemy, y=%v", tp.Y)
	}
}

func TestSpell26EllipseOnThrowAndReturn(t *testing.T) {
	resetQueues()
	pushDoll(dollSpec{
		mode: dollEllipseRecall, x: dollReach, y: 0, arriveAt: deployLife,
		rangeOn: true, slow: true, once: true, dmg: 8.4,
	})
	d := newDoll(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 9, Kind: KindNingyushiDoll, Out: out}
	self := dollSnap(9, 0, 0)
	d.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{me(0, 0), foe(20, 0)}})
	cmds := drain(out)
	if !rangeOn(cmds) {
		t.Fatal("boomerang should show ellipse while thrown")
	}
	if !hasDamage(cmds, 2, 8.4) {
		t.Fatalf("throw should hit with ellipse: %v", cmds)
	}
	d.Handle(ctx, unit.Sense{Time: deployLife, Self: dollSnap(9, dollReach, 0), Nearby: []unit.Snapshot{me(0, 0), foe(20, 0)}})
	cmds = drain(out)
	tp := lastTeleport(cmds, 9)
	if tp == nil || tp.X >= dollReach {
		t.Fatalf("boomerang should start returning after arriving: %v", cmds)
	}
	if !rangeOn(cmds) {
		t.Fatal("boomerang should keep ellipse while returning")
	}
	d.Handle(ctx, unit.Sense{Time: deployLife + recallLife, Self: dollSnap(9, 20, 0), Nearby: []unit.Snapshot{me(0, 0), foe(20, 0)}})
	if !hasDespawn(drain(out), 9) {
		t.Fatal("spell 26 recall should finish in about 0.4s")
	}
	dropState(9)
}

func rangeOn(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == "range" && fx.Amount > 0 {
			return true
		}
	}
	return false
}

func rangeAxes(cmds []unit.Cmd) []float64 {
	for i := len(cmds) - 1; i >= 0; i-- {
		if fx, ok := cmds[i].(unit.FX); ok && fx.Name == "range" && fx.Amount > 0 {
			return []float64{fx.VX, fx.VY}
		}
	}
	return nil
}

func hasTeleport(cmds []unit.Cmd, id uint64) bool {
	return lastTeleport(cmds, id) != nil
}

func hasDespawn(cmds []unit.Cmd, id uint64) bool {
	for _, c := range cmds {
		if d, ok := c.(unit.Despawn); ok && d.UnitID == id {
			return true
		}
	}
	return false
}

func lastTeleport(cmds []unit.Cmd, id uint64) *unit.Teleport {
	var tp *unit.Teleport
	for _, c := range cmds {
		if v, ok := c.(unit.Teleport); ok && v.UnitID == id {
			x := v
			tp = &x
		}
	}
	return tp
}

func fighter(skill uint8) *人偶使 {
	resetQueues()
	return &人偶使{
		energy:    energyMax,
		energyCap: energyMax,
		locked:    true,
		force:     skill,
		deck:      []uint8{},
		hand:      []uint8{},
	}
}

func me(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindNingyushi, Role: unit.RoleFighter,
		X: x, Y: y, Radius: ningyushiRadius, HP: 100, MaxHP: 100, Slot: 0,
	}
}

func meVX(x, y, vx, vy float64) unit.Snapshot {
	s := me(x, y)
	s.VX, s.VY = vx, vy
	return s
}

func foe(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 2, Kind: "原型机_远程", Role: unit.RoleFighter,
		X: x, Y: y, Radius: 18, HP: 100, MaxHP: 100, Slot: 1,
	}
}

func mortalMinion(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 8, Kind: "教父暗杀者", Role: unit.RoleMinion, Mortal: true,
		X: x, Y: y, Radius: 14, HP: 30, MaxHP: 30, Slot: 1,
	}
}

func TestChainLaserPolylineThenClears(t *testing.T) {
	resetQueues()
	a := fighter(SkillN22)
	a.level[SkillN22] = 1
	out := make(chan unit.Cmd, 64)
	ctx := unit.Context{ID: 1, Kind: KindNingyushi, Out: out}
	d1 := unit.Snapshot{ID: 9, Kind: KindNingyushiDoll, Role: unit.RoleHelper, OwnerID: 1, X: 40, Y: 0, Radius: dollRadius, Slot: 0}
	d2 := unit.Snapshot{ID: 10, Kind: KindNingyushiDoll, Role: unit.RoleHelper, OwnerID: 1, X: 60, Y: 0, Radius: dollRadius, Slot: 0}
	setState(9, dollIdle)
	setState(10, dollIdle)
	t.Cleanup(func() { dropState(9); dropState(10) })
	enemy := foe(80, 0)
	near := []unit.Snapshot{enemy, d1, d2}

	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: near})
	cmds := drain(out)
	if hasSpawn(cmds, KindNingyushiBeam) {
		t.Fatalf("lasers should wait for windup: %v", cmds)
	}

	a.Handle(ctx, unit.Sense{Time: chainWind, Self: me(0, 0), Nearby: near})
	cmds = drain(out)
	segs := beamSegs(cmds)
	if len(segs) != 3 {
		t.Fatalf("two idle dolls should make 3 lasers at once, got %d: %v", len(segs), segs)
	}
	if math.Abs(segs[0][0]) > 1e-6 || math.Abs(segs[0][2]-40) > 1e-6 {
		t.Fatalf("first laser self->doll: %v", segs[0])
	}
	if math.Abs(segs[1][0]-40) > 1e-6 || math.Abs(segs[1][2]-60) > 1e-6 {
		t.Fatalf("second laser doll->doll: %v", segs[1])
	}
	if math.Abs(segs[2][0]-60) > 1e-6 || math.Abs(segs[2][2]-80) > 1e-6 {
		t.Fatalf("third laser doll->enemy: %v", segs[2])
	}
	if !hasDamage(cmds, 2, 8.4) {
		t.Fatalf("path should hit enemy for (level+5)*1.4: %v", cmds)
	}

	a.Handle(ctx, unit.Sense{Time: chainWind + laserHold, Self: me(0, 0), Nearby: near})
	cmds = drain(out)
	if !hasDespawnOwned(cmds, KindNingyushiBeam) {
		t.Fatalf("finished lasers should despawn: %v", cmds)
	}
}

func TestHouraiBeamDespawns(t *testing.T) {
	b := 激光{}
	out := make(chan unit.Cmd, 8)
	ctx := unit.Context{ID: 11, Kind: KindNingyushiBeam, Out: out}
	b.Handle(ctx, unit.Sense{Time: 1, Self: unit.Snapshot{ID: 11, Kind: KindNingyushiBeam}})
	if hasDespawn(drain(out), 11) {
		t.Fatal("beam should last through hourai")
	}
	b.Handle(ctx, unit.Sense{Time: 1 + houraiLife, Self: unit.Snapshot{ID: 11, Kind: KindNingyushiBeam}})
	if !hasDespawn(drain(out), 11) {
		t.Fatal("beam must despawn after hourai life")
	}
}

func laserSegs(cmds []unit.Cmd) [][4]float64 {
	return beamSegs(cmds)
}

func beamSegs(cmds []unit.Cmd) [][4]float64 {
	var segs [][4]float64
	for _, c := range cmds {
		sp, ok := c.(unit.Spawn)
		if !ok || sp.Kind != KindNingyushiBeam {
			continue
		}
		segs = append(segs, [4]float64{sp.X, sp.Y, sp.X + sp.VX, sp.Y + sp.VY})
	}
	return segs
}

func hasDespawnOwned(cmds []unit.Cmd, kind string) bool {
	for _, c := range cmds {
		if d, ok := c.(unit.DespawnOwned); ok && d.Kind == kind {
			return true
		}
	}
	return false
}

func lastFX(cmds []unit.Cmd, name string) *unit.FX {
	var fx *unit.FX
	for _, c := range cmds {
		if v, ok := c.(unit.FX); ok && v.Name == name {
			x := v
			fx = &x
		}
	}
	return fx
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

func hasSpawn(cmds []unit.Cmd, kind string) bool {
	return countSpawn(cmds, kind) > 0
}

func lastSpawn(cmds []unit.Cmd, kind string) *unit.Spawn {
	var sp *unit.Spawn
	for _, c := range cmds {
		if v, ok := c.(unit.Spawn); ok && v.Kind == kind {
			x := v
			sp = &x
		}
	}
	return sp
}

func countSpawn(cmds []unit.Cmd, kind string) int {
	n := 0
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == kind {
			n++
		}
	}
	return n
}

func hasDamage(cmds []unit.Cmd, to uint64, amt float64) bool {
	for _, c := range cmds {
		if d, ok := c.(unit.Damage); ok && d.To == to && math.Abs(d.Amount-amt) < 1e-6 {
			return true
		}
	}
	return false
}

func hasDamageTo(cmds []unit.Cmd, to uint64) bool {
	for _, c := range cmds {
		if d, ok := c.(unit.Damage); ok && d.To == to {
			return true
		}
	}
	return false
}

func hasTeleportAway(cmds []unit.Cmd, id uint64, fromX, dist float64) bool {
	for _, c := range cmds {
		tp, ok := c.(unit.Teleport)
		if !ok || tp.UnitID != id {
			continue
		}
		if math.Abs(tp.X-fromX) > dist*0.5 {
			return true
		}
	}
	return false
}

func lastNamed(cmds []unit.Cmd, name string) *unit.FX {
	var fx *unit.FX
	for _, c := range cmds {
		if v, ok := c.(unit.FX); ok && v.Name == name {
			x := v
			fx = &x
		}
	}
	return fx
}

func lastCruise(cmds []unit.Cmd, id uint64) *unit.SetCruise {
	var v *unit.SetCruise
	for _, c := range cmds {
		if x, ok := c.(unit.SetCruise); ok && x.UnitID == id {
			y := x
			v = &y
		}
	}
	return v
}

func lastVel(cmds []unit.Cmd, id uint64) *unit.SetVelocity {
	var v *unit.SetVelocity
	for _, c := range cmds {
		if x, ok := c.(unit.SetVelocity); ok && x.UnitID == id {
			y := x
			v = &y
		}
	}
	return v
}

func lastStun(cmds []unit.Cmd) *unit.Stun {
	var st *unit.Stun
	for _, c := range cmds {
		if v, ok := c.(unit.Stun); ok {
			x := v
			st = &x
		}
	}
	return st
}

func hasConfirm(cmds []unit.Cmd, token uint64, amt float64) bool {
	for _, c := range cmds {
		if d, ok := c.(unit.ConfirmDamage); ok && d.Token == token && math.Abs(d.Amount-amt) < 1e-6 {
			return true
		}
	}
	return false
}
