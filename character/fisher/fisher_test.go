package 钓鱼佬

import (
	"math"
	"math/rand/v2"
	"testing"
	"xqdj/internal/unit"
)

func TestCatchWeightEmptyAndPositive(t *testing.T) {
	if y := catchWeight(0, 0); y > 0 {
		t.Fatalf("x=0 y=%v want 空军", y)
	}
	if y := catchWeight(0.19, 0); y > 0 {
		t.Fatalf("x=0.19 y=%v want 空军", y)
	}
	if y := catchWeight(0.5, 0); y <= 0 {
		t.Fatalf("x=0.5 y=%v want 有鱼", y)
	}
	got := catchWeight(0.5, 0)
	want := 100*(1-math.Sqrt(0.5)) - 10
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("y=%v want %v", got, want)
	}
}

func TestCatchBonusAtMisses(t *testing.T) {
	if catchBonus(0, false) != 0 || catchBonus(1, false) != 0 {
		t.Fatal("before 2 misses should be +0")
	}
	if catchBonus(2, false) != missBonus2 {
		t.Fatal("2 misses should be +5")
	}
	if catchBonus(3, false) != missBonus3 || catchBonus(3, true) != missBonus3 {
		t.Fatal("3 misses / rod should be +10")
	}
}

func TestFishAndRamDamageFromWeight(t *testing.T) {
	if math.Abs(fishDamage(20)-10) > 1e-9 {
		t.Fatalf("20kg fish=%v want 10", fishDamage(20))
	}
	if fishDamage(0.5) != 1 {
		t.Fatal("fish floor 1")
	}
	if fishDamage(100) != 26 {
		t.Fatal("fish cap 26")
	}
	if math.Abs(ramDamage(20)-10) > 1e-9 {
		t.Fatalf("20kg ram=%v want 10", ramDamage(20))
	}
	if ramDamage(1) != 4 {
		t.Fatal("ram floor 4")
	}
	if ramDamage(100) != 22 {
		t.Fatal("ram cap 22")
	}
}

func TestSmallMidFishSpeedDoubled(t *testing.T) {
	base := func(y float64) float64 { return 160 + y*1.6 }
	if math.Abs(fishSpeed(10)-base(10)*2) > 1e-9 {
		t.Fatalf("小鱼 speed=%v", fishSpeed(10))
	}
	if math.Abs(fishSpeed(30)-base(30)*2) > 1e-9 {
		t.Fatalf("中鱼 speed=%v", fishSpeed(30))
	}
	if math.Abs(fishSpeed(60)-base(60)) > 1e-9 {
		t.Fatalf("大鱼 should not double, speed=%v", fishSpeed(60))
	}
}

func TestBootSpawnsThreeRimPonds(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &钓鱼佬{rng: rand.New(rand.NewPCG(1, 2))}
	a.Handle(unit.Context{ID: 1, Kind: KindFisher, Out: out}, unit.Sense{
		Time: 0,
		Self: selfAt(0, 0),
	})
	cmds := drain(out)
	var ponds []unit.Spawn
	for _, c := range cmds {
		s, ok := c.(unit.Spawn)
		if ok && s.Kind == KindPond {
			ponds = append(ponds, s)
		}
	}
	if len(ponds) != pondN {
		t.Fatalf("ponds=%d want %d cmds=%v", len(ponds), pondN, cmds)
	}
	for i, p := range ponds {
		if p.OwnerID != 1 {
			t.Fatalf("pond %d owner=%d", i, p.OwnerID)
		}
		if math.Hypot(p.X, p.Y) < 80 {
			t.Fatalf("pond %d too central (%v,%v)", i, p.X, p.Y)
		}
		if !unit.HexContains(p.X, p.Y, pondRadius) {
			t.Fatalf("pond %d outside hex (%v,%v)", i, p.X, p.Y)
		}
	}
}

func TestFishAfterThreeSeconds(t *testing.T) {
	resetFishQ()
	out := make(chan unit.Cmd, 32)
	a := &钓鱼佬{booted: true, draw: seq(0.5)}
	ctx := unit.Context{ID: 1, Kind: KindFisher, Out: out}
	pond := pondAt(40, 0)
	a.Handle(ctx, unit.Sense{
		Time:   0,
		Self:   selfAt(40, 0),
		Nearby: []unit.Snapshot{pond, enemyAt(120, 0)},
	})
	cmds := drain(out)
	if lastNamed(cmds, "cast") == nil {
		t.Fatalf("should start 钓鱼: %v", cmds)
	}
	if !a.fishing {
		t.Fatal("fishing flag")
	}
	a.Handle(ctx, unit.Sense{
		Time:   3,
		Self:   selfAt(40, 0),
		Nearby: []unit.Snapshot{pond, enemyAt(120, 0)},
	})
	cmds = drain(out)
	if a.fishing {
		t.Fatal("should reel after 3s")
	}
	if lastNamed(cmds, "reel") == nil {
		t.Fatalf("missing reel: %v", cmds)
	}
	sp := lastSpawn(cmds)
	if sp == nil || (sp.Kind != KindFish && sp.Kind != KindFishMid && sp.Kind != KindFishBig) {
		t.Fatalf("spawn=%v", cmds)
	}
	y := catchWeight(0.5, 0)
	want := fishSpeed(y)
	got := math.Hypot(sp.VX, sp.VY)
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("launch speed=%v want %v kind=%s y=%v", got, want, sp.Kind, y)
	}
}

func TestMissVoidsPondAndCounts(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &钓鱼佬{booted: true, draw: seq(0)}
	ctx := unit.Context{ID: 1, Kind: KindFisher, Out: out}
	pond := pondAt(0, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: selfAt(0, 0), Nearby: []unit.Snapshot{pond}})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{Time: 3, Self: selfAt(0, 0), Nearby: []unit.Snapshot{pond}})
	cmds := drain(out)
	if a.misses != 1 {
		t.Fatalf("misses=%d", a.misses)
	}
	if lastNamed(cmds, "miss") == nil {
		t.Fatalf("missing miss fx: %v", cmds)
	}
	if !hasDespawn(cmds, pond.ID) {
		t.Fatalf("should void pond: %v", cmds)
	}
	if lastSpawnKind(cmds, KindFish) != nil {
		t.Fatal("空军 should not spawn 鱼")
	}
	if lastCruise(cmds) != fisherCruise*1.5 {
		t.Fatalf("1 空军 cruise=%v want %v", lastCruise(cmds), fisherCruise*1.5)
	}
}

func TestMissMarksStackWalkSpeed(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := &钓鱼佬{booted: true, misses: 2}
	a.Handle(unit.Context{ID: 1, Kind: KindFisher, Out: out}, unit.Sense{
		Time: 1,
		Self: selfAt(40, 0),
	})
	if lastCruise(drain(out)) != fisherCruise*2 {
		t.Fatalf("2 空军 should be 200%% cruise")
	}
}

func TestSecondMissAddsFiveNextCatch(t *testing.T) {
	resetFishQ()
	out := make(chan unit.Cmd, 16)
	a := &钓鱼佬{booted: true, misses: 2, draw: seq(0.15)}
	ctx := unit.Context{ID: 1, Kind: KindFisher, Out: out}
	pond := pondAt(0, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: selfAt(0, 0), Nearby: []unit.Snapshot{pond}})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{Time: 3, Self: selfAt(0, 0), Nearby: []unit.Snapshot{pond}})
	cmds := drain(out)
	if lastSpawn(cmds) == nil {
		t.Fatalf("with +5, x=0.15 should catch: %v y=%v", cmds, catchWeight(0.15, 5))
	}
	if catchWeight(0.15, 0) > 0 {
		t.Fatal("control: x=0.15 without bonus should still be 空军")
	}
}

func TestThirdMissUnlocksTenRolls(t *testing.T) {
	resetFishQ()
	out := make(chan unit.Cmd, 64)
	a := &钓鱼佬{booted: true, misses: 3, rod: true, draw: seq(
		0.4, 0, 0.4, 0, 0.4, 0, 0.4, 0, 0.4, 0,
	)}
	ctx := unit.Context{ID: 1, Kind: KindFisher, Out: out}
	pond := pondAt(0, 0)
	a.Handle(ctx, unit.Sense{
		Time: 0, Self: selfAt(0, 0),
		Nearby: []unit.Snapshot{pond, enemyAt(80, 0)},
	})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{
		Time: 3, Self: selfAt(0, 0),
		Nearby: []unit.Snapshot{pond, enemyAt(80, 0)},
	})
	cmds := drain(out)
	n := 0
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && isFishKind(s.Kind) {
			n++
		}
	}
	if n != 5 {
		t.Fatalf("10 rolls with 5 hits spawned %d: %v", n, cmds)
	}
	if a.misses != 3 {
		t.Fatalf("partial 空军 should not add misses, got %d", a.misses)
	}
}

func TestTenRollAllMissVoidsOnce(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &钓鱼佬{booted: true, misses: 3, rod: true, draw: seq(0, 0, 0, 0, 0, 0, 0, 0, 0, 0)}
	ctx := unit.Context{ID: 1, Kind: KindFisher, Out: out}
	pond := pondAt(0, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: selfAt(0, 0), Nearby: []unit.Snapshot{pond}})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{Time: 3, Self: selfAt(0, 0), Nearby: []unit.Snapshot{pond}})
	cmds := drain(out)
	if a.misses != 4 {
		t.Fatalf("session miss should +1, got %d", a.misses)
	}
	if hasDespawn(cmds, pond.ID) || hasDespawnOwned(cmds, KindPond) {
		t.Fatalf("场鱼塘 must stay: %v", cmds)
	}
	if lastSpawnKind(cmds, KindFish) != nil {
		t.Fatal("all empty should not spawn")
	}
}

func TestLeaveAndReenterBeforeNextFish(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &钓鱼佬{booted: true, draw: seq(0.5, 0.5)}
	ctx := unit.Context{ID: 1, Kind: KindFisher, Out: out}
	pond := pondAt(0, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: selfAt(0, 0), Nearby: []unit.Snapshot{pond}})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{Time: 3, Self: selfAt(0, 0), Nearby: []unit.Snapshot{pond}})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{Time: 3.1, Self: selfAt(0, 0), Nearby: []unit.Snapshot{pond}})
	cmds := drain(out)
	if lastNamed(cmds, "cast") != nil || a.fishing {
		t.Fatal("still overlapping should not 再钓")
	}
	a.Handle(ctx, unit.Sense{Time: 3.2, Self: selfAt(200, 0), Nearby: nil})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{Time: 3.3, Self: selfAt(0, 0), Nearby: []unit.Snapshot{pond}})
	cmds = drain(out)
	if lastNamed(cmds, "cast") == nil {
		t.Fatal("re-enter should 再钓")
	}
}

func TestSecondMissFloodsArena(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &钓鱼佬{booted: true, misses: 1, draw: seq(0)}
	ctx := unit.Context{ID: 1, Kind: KindFisher, Out: out}
	pond := pondAt(0, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: selfAt(0, 0), Nearby: []unit.Snapshot{pond}})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{Time: 3, Self: selfAt(0, 0), Nearby: []unit.Snapshot{pond}})
	cmds := drain(out)
	if a.misses != 2 {
		t.Fatalf("misses=%d", a.misses)
	}
	if !hasDespawnOwned(cmds, KindPond) {
		t.Fatalf("should clear rim 鱼塘: %v", cmds)
	}
	sp := lastSpawnKind(cmds, KindFlood)
	if sp == nil || sp.X != 0 || sp.Y != 0 {
		t.Fatalf("场鱼塘 spawn=%v cmds=%v", sp, cmds)
	}
}

func TestFlyFishKeepsMoving(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &钓鱼佬{booted: true, misses: 2}
	flood := floodAt()
	a.Handle(unit.Context{ID: 1, Kind: KindFisher, Out: out}, unit.Sense{
		Time:   1,
		Self:   selfAt(40, 0),
		Nearby: []unit.Snapshot{flood},
	})
	cmds := drain(out)
	if !a.fishing || lastNamed(cmds, "cast") == nil {
		t.Fatalf("should 飞着钓: %v", cmds)
	}
	if lastCruise(cmds) == 0 {
		t.Fatal("飞着钓 must not freeze")
	}
	if v := lastVel(cmds); v != nil && v.VX == 0 && v.VY == 0 {
		t.Fatal("飞着钓 must not zero velocity")
	}
}

func TestFlyFishAutoRecast(t *testing.T) {
	resetFishQ()
	out := make(chan unit.Cmd, 16)
	a := &钓鱼佬{booted: true, misses: 2, draw: seq(0.5)}
	ctx := unit.Context{ID: 1, Kind: KindFisher, Out: out}
	flood := floodAt()
	self := selfAt(40, 0)
	self.VX, self.VY = 80, 0
	a.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{flood, enemyAt(120, 0)}})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{Time: 3, Self: self, Nearby: []unit.Snapshot{flood, enemyAt(120, 0)}})
	_ = drain(out)
	if a.fishing {
		t.Fatal("just reeled")
	}
	a.Handle(ctx, unit.Sense{Time: 3.1, Self: self, Nearby: []unit.Snapshot{flood, enemyAt(120, 0)}})
	cmds := drain(out)
	if lastNamed(cmds, "cast") == nil || !a.fishing {
		t.Fatal("should auto 再钓")
	}
}

func TestFishSeeksOnlyOnSpawnAndBounce(t *testing.T) {
	resetFishQ()
	pushFish(fishJob{y: 20, dmg: 4, speed: 180})
	f := newFish(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	out := make(chan unit.Cmd, 8)
	ctx := unit.Context{ID: 9, Kind: KindFish, Out: out}
	self := unit.Snapshot{ID: 9, Kind: KindFish, Role: unit.RoleProjectile, Slot: 0, X: 0, Y: 0, VX: 180, Radius: 10}
	f.Handle(ctx, unit.Sense{
		Time:   1,
		Self:   self,
		Nearby: []unit.Snapshot{enemyAt(40, 0)},
	})
	if lastVel(drain(out)) == nil {
		t.Fatal("出现时应索敌")
	}
	moved := self
	f.Handle(ctx, unit.Sense{
		Time:   1.1,
		Self:   moved,
		Nearby: []unit.Snapshot{enemyAt(40, 80)},
	})
	if lastVel(drain(out)) != nil {
		t.Fatal("飞行中不应一直索敌")
	}
	f.Handle(ctx, unit.WallHit{Time: 1.2, NX: 1, NY: 0})
	v := lastVel(drain(out))
	if v == nil {
		t.Fatal("反弹时应再索敌")
	}
}

func TestFishLastBounceCoastsToStop(t *testing.T) {
	resetFishQ()
	pushFish(fishJob{y: 20, dmg: 4, speed: 180})
	f := newFish(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	f.steady = true
	f.drag = 400
	out := make(chan unit.Cmd, 8)
	ctx := unit.Context{ID: 9, Kind: KindFish, Out: out}
	enemy := enemyAt(40, 0)
	self := unit.Snapshot{ID: 9, Kind: KindFish, Role: unit.RoleProjectile, Slot: 0, X: 0, Y: 0, VX: 180, VY: 0, Radius: 10}
	f.Handle(ctx, unit.Sense{
		Time: 1,
		Self: self,
		Nearby: []unit.Snapshot{
			{ID: 1, Kind: KindFisher, Role: unit.RoleFighter, Slot: 0, X: -30, Y: 0, Radius: 18},
			enemy,
		},
	})
	_ = drain(out)
	for i := 1; i <= maxBounce; i++ {
		f.Handle(ctx, unit.WallHit{Time: float64(i), NX: 1, NY: 0})
		cmds := drain(out)
		if hasDespawn(cmds, 9) {
			t.Fatalf("bounce %d should stay: %v", i, cmds)
		}
		if lastVel(cmds) == nil {
			t.Fatalf("bounce %d should retarget: %v", i, cmds)
		}
	}
	if !f.spent {
		t.Fatal("last bounce should start 减速")
	}
	f.Handle(ctx, unit.WallHit{Time: 4, NX: 1, NY: 0})
	if hasDespawn(drain(out), 9) {
		t.Fatal("last bounce leftover should not despawn")
	}
	self.VX, self.VY = 180, 0
	f.Handle(ctx, unit.Sense{Time: 1.5, Self: self, Nearby: []unit.Snapshot{enemy}})
	v := lastVel(drain(out))
	if v == nil {
		t.Fatal("coast should rewrite speed")
	}
	sp := math.Hypot(v.VX, v.VY)
	if sp >= 180-1e-6 {
		t.Fatalf("speed should drop, got %v", sp)
	}
	self.VX, self.VY = v.VX, v.VY
	stopped := false
	for tnow := 1.7; tnow <= 4; tnow += 0.2 {
		f.Handle(ctx, unit.Sense{Time: tnow, Self: self, Nearby: []unit.Snapshot{enemy}})
		cmds := drain(out)
		if vv := lastVel(cmds); vv != nil {
			self.VX, self.VY = vv.VX, vv.VY
			if math.Hypot(vv.VX, vv.VY) < 1e-6 {
				stopped = true
				break
			}
		}
	}
	if !stopped {
		t.Fatalf("should rest at 0, vel=(%v,%v)", self.VX, self.VY)
	}
}

func TestSpentFishDoesNotDamage(t *testing.T) {
	resetFishQ()
	pushFish(fishJob{y: 20, dmg: fishDamage(20), speed: 180})
	f := newFish(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	f.spent = true
	out := make(chan unit.Cmd, 8)
	f.Handle(unit.Context{ID: 9, Kind: KindFish, Out: out}, unit.Collision{
		Time:  1,
		Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, Radius: 18},
	})
	cmds := drain(out)
	if lastDamage(cmds) != nil || hasDespawn(cmds, 9) {
		t.Fatalf("spent leftover should be harmless: %v", cmds)
	}
}

func TestFishHitsEnemyGoesInert(t *testing.T) {
	resetFishQ()
	pushFish(fishJob{y: 20, dmg: fishDamage(20), speed: 180})
	f := newFish(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	out := make(chan unit.Cmd, 8)
	ctx := unit.Context{ID: 9, Kind: KindFish, Out: out}
	f.Handle(ctx, unit.Collision{
		Time:  1,
		Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, Radius: 18},
	})
	cmds := drain(out)
	d := lastDamage(cmds)
	if d == nil || d.To != 2 || math.Abs(d.Amount-fishDamage(20)) > 1e-9 {
		t.Fatalf("damage=%v", cmds)
	}
	if hasDespawn(cmds, 9) || !f.spent {
		t.Fatal("hit should 惰性, not despawn")
	}
	f.Handle(ctx, unit.Collision{
		Time:  1.1,
		Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, Radius: 18},
	})
	if lastDamage(drain(out)) != nil {
		t.Fatal("inert fish should not damage again")
	}
}

func TestBigCatchStartsRamUntilAllFaces(t *testing.T) {
	resetFishQ()
	out := make(chan unit.Cmd, 32)
	a := &钓鱼佬{booted: true, draw: seq(0.9), misses: 2}
	ctx := unit.Context{ID: 1, Kind: KindFisher, Out: out}
	pond := pondAt(0, 0)
	a.Handle(ctx, unit.Sense{
		Time: 0, Self: selfAt(0, 0),
		Nearby: []unit.Snapshot{pond, enemyAt(80, 0)},
	})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{
		Time: 3, Self: selfAt(0, 0),
		Nearby: []unit.Snapshot{pond, enemyAt(80, 0)},
	})
	cmds := drain(out)
	if !a.ram {
		t.Fatalf("y=%v should ram", catchWeight(0.9, 0))
	}
	if lastNamed(cmds, "ram") == nil {
		t.Fatalf("missing ram fx: %v", cmds)
	}
	if lastCruise(cmds) != ramCruise {
		t.Fatalf("cruise=%v", lastCruise(cmds))
	}
	if lastSpawnKind(cmds, KindFishBig) != nil || lastSpawnKind(cmds, KindFish) != nil || lastSpawnKind(cmds, KindFishMid) != nil {
		t.Fatal("50kg 鱼 should wait until ram ends")
	}

	ap := unit.HexRadius * math.Sqrt(3) / 2
	for i := 0; i < 6; i++ {
		nx, ny := hexNormal(i)
		a.x, a.y = nx*(ap-fisherRadius), ny*(ap-fisherRadius)
		a.Handle(ctx, unit.WallHit{Time: float64(10 + i), NX: nx, NY: ny})
		cmds = drain(out)
	}
	if a.ram {
		t.Fatal("all faces should end ram")
	}
	if lastCruise(cmds) != a.walkSpeed() {
		t.Fatalf("restore cruise=%v want %v", lastCruise(cmds), a.walkSpeed())
	}
	if a.misses != 2 {
		t.Fatalf("ram end should keep 空军, misses=%d", a.misses)
	}
	sp := lastSpawn(cmds)
	if sp == nil || !isFishKind(sp.Kind) {
		t.Fatalf("ram end should spit leftover 鱼: %v", cmds)
	}
	f := newFish(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	if !f.spent {
		t.Fatal("spit 鱼 should already be 惰性")
	}
}

func TestRamSkipsPond(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &钓鱼佬{booted: true, ram: true, ramY: 60}
	a.Handle(unit.Context{ID: 1, Kind: KindFisher, Out: out}, unit.Sense{
		Time:   1,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{pondAt(0, 0), enemyAt(80, 0)},
	})
	cmds := drain(out)
	if a.fishing || lastNamed(cmds, "cast") != nil {
		t.Fatal("高速碰撞期间碰到鱼塘不应钓鱼")
	}
}

func TestRamHitsEnemyNormalDoesNot(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := &钓鱼佬{booted: true, ram: true, ramY: 60, slot: 0}
	ctx := unit.Context{ID: 1, Kind: KindFisher, Out: out}
	hit := unit.Collision{
		Time:  1,
		Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, Radius: 18},
	}
	a.Handle(ctx, hit)
	d := lastDamage(drain(out))
	if d == nil || d.To != 2 {
		t.Fatalf("ram should damage: %v", d)
	}
	a.ram = false
	a.Handle(ctx, unit.Collision{
		Time:  2,
		Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, Radius: 18},
	})
	if lastDamage(drain(out)) != nil {
		t.Fatal("常态无伤害")
	}
}

func TestFishingDamageReduction(t *testing.T) {
	out := make(chan unit.Cmd, 4)
	a := &钓鱼佬{fishing: true}
	a.Handle(unit.Context{ID: 1, Kind: KindFisher, Out: out}, unit.IncomingDamage{
		Token: 7, Amount: 10, Time: 1,
	})
	cmds := drain(out)
	if len(cmds) != 1 {
		t.Fatalf("cmds=%v", cmds)
	}
	c, ok := cmds[0].(unit.ConfirmDamage)
	if !ok || math.Abs(c.Amount-5) > 1e-9 {
		t.Fatalf("confirm=%v", cmds[0])
	}
}

func TestUnlockRodOnThirdMiss(t *testing.T) {
	resetFishQ()
	out := make(chan unit.Cmd, 64)
	a := &钓鱼佬{booted: true, misses: 2, draw: seq(0, 0.4, 0.4, 0.4, 0.4, 0.4, 0.4, 0.4, 0.4, 0.4, 0.4)}
	ctx := unit.Context{ID: 1, Kind: KindFisher, Out: out}
	pond := pondAt(0, 0)
	a.Handle(ctx, unit.Sense{
		Time: 0, Self: selfAt(0, 0),
		Nearby: []unit.Snapshot{pond, enemyAt(80, 0)},
	})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{
		Time: 3, Self: selfAt(0, 0),
		Nearby: []unit.Snapshot{pond, enemyAt(80, 0)},
	})
	cmds := drain(out)
	if a.misses != 3 || !a.rod {
		t.Fatalf("misses=%d rod=%v", a.misses, a.rod)
	}
	n := 0
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && isFishKind(s.Kind) {
			n++
		}
	}
	if n != 10 {
		t.Fatalf("第3次空军应当场10连, spawned %d: %v", n, cmds)
	}
	if hasDespawn(cmds, pond.ID) {
		t.Fatal("10连有鱼时不应作废鱼塘")
	}
}

func selfAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindFisher, Role: unit.RoleFighter, Slot: 0,
		X: x, Y: y, Radius: fisherRadius, Vision: fisherVision, VX: 40,
	}
}

func enemyAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: x, Y: y, Radius: 18}
}

func pondAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 40, Kind: KindPond, Role: unit.RoleHelper,
		X: x, Y: y, Radius: pondRadius, Slot: 0, OwnerID: 1,
	}
}

func floodAt() unit.Snapshot {
	return unit.Snapshot{
		ID: 41, Kind: KindFlood, Role: unit.RoleHelper,
		X: 0, Y: 0, Radius: floodRadius, Slot: 0, OwnerID: 1,
	}
}

func seq(xs ...float64) func() float64 {
	i := 0
	return func() float64 {
		if i >= len(xs) {
			return 0
		}
		v := xs[i]
		i++
		return v
	}
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

func lastNamed(cmds []unit.Cmd, name string) *unit.FX {
	var fx *unit.FX
	for _, c := range cmds {
		if v, ok := c.(unit.FX); ok && v.Name == name {
			cp := v
			fx = &cp
		}
	}
	return fx
}

func lastDamage(cmds []unit.Cmd) *unit.Damage {
	var d *unit.Damage
	for _, c := range cmds {
		if v, ok := c.(unit.Damage); ok {
			cp := v
			d = &cp
		}
	}
	return d
}

func lastSpawn(cmds []unit.Cmd) *unit.Spawn {
	var sp *unit.Spawn
	for _, c := range cmds {
		if v, ok := c.(unit.Spawn); ok {
			cp := v
			sp = &cp
		}
	}
	return sp
}

func lastSpawnKind(cmds []unit.Cmd, kind string) *unit.Spawn {
	var sp *unit.Spawn
	for _, c := range cmds {
		if v, ok := c.(unit.Spawn); ok && v.Kind == kind {
			cp := v
			sp = &cp
		}
	}
	return sp
}

func lastVel(cmds []unit.Cmd) *unit.SetVelocity {
	var v *unit.SetVelocity
	for _, c := range cmds {
		if s, ok := c.(unit.SetVelocity); ok {
			cp := s
			v = &cp
		}
	}
	return v
}

func lastCruise(cmds []unit.Cmd) float64 {
	s := math.NaN()
	for _, c := range cmds {
		if v, ok := c.(unit.SetCruise); ok {
			s = v.Speed
		}
	}
	return s
}

func hasDespawn(cmds []unit.Cmd, id uint64) bool {
	for _, c := range cmds {
		if d, ok := c.(unit.Despawn); ok && d.UnitID == id {
			return true
		}
	}
	return false
}

func hasDespawnOwned(cmds []unit.Cmd, kind string) bool {
	for _, c := range cmds {
		if d, ok := c.(unit.DespawnOwned); ok && d.Kind == kind {
			return true
		}
	}
	return false
}

func isFishKind(kind string) bool {
	return kind == KindFish || kind == KindFishMid || kind == KindFishBig
}
