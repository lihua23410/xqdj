package 优昙华院

import (
	"math"
	"testing"
	"xqdj/internal/unit"
)

func TestAnimLockBlocksNextSkill(t *testing.T) {
	a := fighter(SkillMind)
	out := make(chan unit.Cmd, 64)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	enemy := foe(80, 0)

	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	if !hasSpawn(cmds, KindMindShot) {
		t.Fatalf("t=0 should fire mind: %v", cmds)
	}
	if a.energy < energyMax-energyCost-1e-6 || a.energy > energyMax-energyCost+1e-6 {
		t.Fatalf("energy after mind=%v", a.energy)
	}

	a.force = SkillLaser
	a.Handle(ctx, unit.Sense{Time: 0.2, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds = drain(out)
	if hasSpawn(cmds, KindLaser) || hasFX(cmds, "laser-warn") {
		t.Fatalf("mind fire window should block laser at 0.2s")
	}

	a.Handle(ctx, unit.Sense{Time: mindFire, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds = drain(out)
	if hasSpawn(cmds, KindLaser) || hasFX(cmds, "laser-warn") {
		t.Fatalf("mind recovery should still block laser at t=%v", mindFire)
	}

	done := skillAnim(SkillMind)
	a.Handle(ctx, unit.Sense{Time: done, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds = drain(out)
	if !hasFX(cmds, "laser-warn") {
		t.Fatalf("t=%v should allow laser after mind recovery: %v", done, cmds)
	}
}

func TestEnergyDoesNotRegen(t *testing.T) {
	a := fighter(SkillCrown)
	a.energy = 1
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	enemy := foe(120, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	if !hasSpawn(drain(out), KindCrown) {
		t.Fatal("crown at 1 energy")
	}
	if a.energy > 0.05 {
		t.Fatalf("crown should spend 1, energy=%v", a.energy)
	}
	a.locked = true
	a.force = SkillSteer
	a.Handle(ctx, unit.Sense{Time: 2, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	_ = drain(out)
	if a.energy > 0.05 {
		t.Fatalf("energy should not regen, energy=%v", a.energy)
	}
}

func TestWallRefillsEnergy(t *testing.T) {
	a := fighter(SkillSteer)
	a.energy = 1.2
	out := make(chan unit.Cmd, 16)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	a.Handle(ctx, unit.WallHit{Time: 1, NX: 1, NY: 0})
	if math.Abs(a.energy-(1.2+energyWall)) > 1e-6 {
		t.Fatalf("wall should restore %v, energy=%v", energyWall, a.energy)
	}
	a.energy = energyMax - 0.1
	a.Handle(ctx, unit.WallHit{Time: 2, NX: 1, NY: 0})
	if a.energy != energyMax {
		t.Fatalf("wall should clamp to cap, energy=%v", a.energy)
	}
	_ = drain(out)
}

func TestNoEnergyNoCast(t *testing.T) {
	a := &优昙华院{energy: 0.05, energyCap: energyMax, deck: []uint8{}}
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	cmds := drain(out)
	if hasSpawn(cmds, KindMindShot) || hasSpawn(cmds, KindLaser) || hasSpawn(cmds, KindCrown) || hasSpawn(cmds, KindSeekShot) {
		t.Fatalf("no energy must not cast: %v", cmds)
	}
	if a.energy > 0.05+1e-6 {
		t.Fatalf("energy=%v", a.energy)
	}
}

func TestRandomCastSpendsEnergy(t *testing.T) {
	a := &优昙华院{energy: energyMax, energyCap: energyMax}
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	if a.energy > energyMax-energyCost+1e-6 {
		t.Fatalf("a ready skill should spend energy, energy=%v", a.energy)
	}
}

func TestRandomOpeningVaries(t *testing.T) {
	seen := map[string]int{}
	for i := 0; i < 50; i++ {
		a := &优昙华院{energy: energyMax, energyCap: energyMax}
		out := make(chan unit.Cmd, 32)
		ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
		a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
		tag := "none"
		for _, c := range drain(out) {
			switch x := c.(type) {
			case unit.Spawn:
				tag = x.Kind
			}
		}
		seen[tag]++
	}
	if len(seen) < 3 {
		t.Fatalf("opening skill should vary, got %v", seen)
	}
}

func TestMindSplitsOnWall(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 9, Kind: KindMindShot, Out: out}
	b := &心弹{owner: 1, slot: 1, x: 10, y: 12, booted: true}
	b.Handle(ctx, unit.WallHit{Time: 1, NX: 1, NY: 0})
	cmds := drain(out)
	n := 0
	died := false
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == KindMindShard {
			n++
			if s.OwnerID != 1 || s.Slot != 1 {
				t.Fatalf("shard owner/slot %+v", s)
			}
		}
		if _, ok := c.(unit.Despawn); ok {
			died = true
		}
	}
	if n != shardCount || !died {
		t.Fatalf("split shards=%d despawn=%v cmds=%v", n, died, cmds)
	}
}

func TestDoseFourBlastsAndResets(t *testing.T) {
	a := fighter(SkillDose)
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	enemy := foe(200, 0)
	t0 := 0.0
	for i := 0; i < 4; i++ {
		a.energy = energyMax
		a.animUntil = 0
		a.Handle(ctx, unit.Sense{Time: t0, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
		cmds := drain(out)
		blasted := hasSpawn(cmds, KindBlast) || hasFX(cmds, "blast")
		if i < 3 && blasted {
			t.Fatalf("blast at dose %d", i+1)
		}
		if i < 3 && hasDamageTo(cmds, 1, 1) {
			t.Fatalf("should not hurt self at dose %d: %v", i+1, cmds)
		}
		if i == 3 && !blasted {
			t.Fatalf("4th dose should blast: %v", cmds)
		}
		if i == 3 && hasDamageTo(cmds, 1, 1) {
			t.Fatalf("blast must not hurt self: %v", cmds)
		}
		t0 += 3
	}
	if a.doses != 0 {
		t.Fatalf("doses should reset after blast, doses=%d", a.doses)
	}
}

func TestLaserWindupGivesDodgeWindow(t *testing.T) {
	a := fighter(SkillLaser)
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	enemy := foe(90, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	if hasSpawn(cmds, KindLaser) {
		t.Fatalf("windup should not spawn beam yet: %v", cmds)
	}
	if !hasFX(cmds, "laser-warn") {
		t.Fatalf("windup should telegraph: %v", cmds)
	}
	if hasDamageTo(cmds, 2, 0) {
		t.Fatalf("windup must not damage: %v", cmds)
	}
	a.Handle(ctx, unit.Sense{Time: laserWind - 0.05, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds = drain(out)
	if hasSpawn(cmds, KindLaser) || hasDamageTo(cmds, 2, 0) {
		t.Fatalf("still in windup: %v", cmds)
	}
	a.Handle(ctx, unit.Sense{Time: laserWind, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds = drain(out)
	if !hasSpawn(cmds, KindLaser) {
		t.Fatalf("beam should fire after windup: %v", cmds)
	}
	if !hasDamageTo(cmds, 2, laserDamage) {
		t.Fatalf("first tick after windup should damage: %v", cmds)
	}
}

func TestLaserInterruptedByHit(t *testing.T) {
	a := fighter(SkillLaser)
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	enemy := foe(90, 0)
	self := me(0, 0)
	self.VX, self.VY = 152, 0
	a.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{enemy}})
	_ = drain(out)
	if a.laserUntil <= 0 {
		t.Fatal("laser not running")
	}
	a.Handle(ctx, unit.IncomingDamage{Token: 7, Amount: 10, Time: 0.4})
	cmds := drain(out)
	if a.laserUntil != 0 {
		t.Fatal("hit should break laser")
	}
	if !hasDespawnKind(cmds, KindLaser) {
		t.Fatalf("should despawn laser: %v", cmds)
	}
	if !hasVelocity(cmds, 1, 152, 0) {
		t.Fatalf("interrupt should restore motion: %v", cmds)
	}
}

func TestLaserRestoresVelocityAfterEnd(t *testing.T) {
	a := fighter(SkillLaser)
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	enemy := foe(90, 0)
	self := me(0, 0)
	self.VX, self.VY = 40, 80
	a.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{enemy}})
	_ = drain(out)
	self.VX, self.VY = 0, 0
	a.Handle(ctx, unit.Sense{Time: 0.25, Self: self, Nearby: []unit.Snapshot{enemy}})
	if hasVelocity(drain(out), 1, 40, 80) {
		t.Fatal("must stay locked during laser")
	}
	a.locked = true
	a.force = SkillSteer
	a.Handle(ctx, unit.Sense{Time: laserWind + laserLife, Self: self, Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	if a.laserUntil != 0 {
		t.Fatal("laser should end")
	}
	if !hasVelocity(cmds, 1, 40, 80) {
		t.Fatalf("after laser should move again: %v", cmds)
	}
}

func TestBreakMissDoesNotSpend(t *testing.T) {
	a := fighter(SkillBreak)
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	enemy := foe(220, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	if hasDamageTo(cmds, 2, 1) {
		t.Fatalf("out of range break should miss: %v", cmds)
	}
	if a.energy != energyMax {
		t.Fatalf("miss must not spend energy, energy=%v", a.energy)
	}
}

func TestShardSlowsAndFadesOut(t *testing.T) {
	if shardSpeedAt(1) >= shardSpeedAt(0)-1e-6 {
		t.Fatalf("shard speed should fall inverse with time: t0=%v t1=%v", shardSpeedAt(0), shardSpeedAt(1))
	}
	if shardAlpha(0) < 0.99 {
		t.Fatalf("shard alpha at t=0 = %v", shardAlpha(0))
	}
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 3, Kind: KindMindShard, Out: out}
	b := &碎弹{owner: 1}
	self := unit.Snapshot{
		ID: 3, Kind: KindMindShard, Role: unit.RoleProjectile,
		X: 0, Y: 0, VX: shardSpeed, Radius: shardRadius, Slot: 0,
	}
	b.Handle(ctx, unit.Sense{Time: 0, Self: self})
	cmds := drain(out)
	v0 := velOf(cmds, 3)
	if v0 < shardSpeed*0.9 {
		t.Fatalf("t=0 speed=%v", v0)
	}
	self.VX = v0
	b.Handle(ctx, unit.Sense{Time: 0.6, Self: self})
	cmds = drain(out)
	v1 := velOf(cmds, 3)
	if v1 >= v0-1e-6 {
		t.Fatalf("should slow: t0=%v t0.6=%v", v0, v1)
	}
	want := shardSpeedAt(0.6)
	if math.Abs(v1-want) > 1e-6 {
		t.Fatalf("speed %v want inverse %v", v1, want)
	}
	end := shardTau*(1/shardFadeCut-1) + 0.05
	b.Handle(ctx, unit.Sense{Time: end, Self: self})
	if !hasDespawnID(drain(out), 3) {
		t.Fatal("alpha 0 should despawn shard")
	}
}

func TestMindStartsNearRangedSpeed(t *testing.T) {
	if mindSpeed(0) < 150 {
		t.Fatalf("mind t=0 speed=%v, should not crawl", mindSpeed(0))
	}
	if mindSpeed(1)-mindSpeed(0) < 400 {
		t.Fatalf("mind speed ramp too small: t0=%v t1=%v", mindSpeed(0), mindSpeed(1))
	}
}

func TestShardRadiusIsSphere(t *testing.T) {
	spec, ok := unit.Lookup(KindMindShard)
	if !ok {
		t.Fatal("missing shard spec")
	}
	want := 32.0
	if spec.Radius < want-1e-6 || spec.Radius > want+1e-6 {
		t.Fatalf("shard radius=%v want %v", spec.Radius, want)
	}
	mind, ok := unit.Lookup(KindMindShot)
	if !ok {
		t.Fatal("missing mind spec")
	}
	if mind.Radius < 10 {
		t.Fatalf("mind shot too small: %v", mind.Radius)
	}
}

func TestCrownPiercesOrdinaryShot(t *testing.T) {
	spec, ok := unit.Lookup(KindCrown)
	if !ok {
		t.Fatal("missing crown spec")
	}
	if !spec.PassWalls {
		t.Fatal("crown should pass walls like 紫弹")
	}
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 9, Kind: KindCrown, Out: out}
	b := &花冠{owner: 1, slot: 0}
	b.Handle(ctx, unit.Collision{
		Other: unit.Snapshot{ID: 40, Role: unit.RoleProjectile, Slot: 1, OwnerID: 2, Radius: 6},
	})
	cmds := drain(out)
	if hasDespawnID(cmds, 9) {
		t.Fatalf("crown must keep flying: %v", cmds)
	}
	if hasDespawnID(cmds, 40) {
		t.Fatalf("crown should not delete the other shot: %v", cmds)
	}
}

func TestCrownGrowsRadius(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 9, Kind: KindCrown, Out: out}
	b := &花冠{owner: 1, slot: 0}
	self := unit.Snapshot{ID: 9, X: 0, Y: 0, Radius: crownBaseR, Slot: 0}
	b.Handle(ctx, unit.Sense{Self: self})
	_ = drain(out)
	self.X = 100
	b.Handle(ctx, unit.Sense{Self: self})
	cmds := drain(out)
	want := crownRadius(100)
	if !hasSetRadius(cmds, 9, want) {
		t.Fatalf("want radius %v after 100 travel, cmds=%v", want, cmds)
	}
	self.X = 230
	b.Handle(ctx, unit.Sense{Self: self})
	if !hasSetRadius(drain(out), 9, crownMaxR) {
		t.Fatal("should clamp to crownMaxR")
	}
}

func TestMindClearsProjectileOnPath(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 9, Kind: KindMindShot, Out: out}
	b := &心弹{owner: 1, slot: 0, booted: true, passed: true, ux: 1}
	b.Handle(ctx, unit.Collision{
		Other: unit.Snapshot{ID: 40, Role: unit.RoleProjectile, Slot: 1, OwnerID: 2, Radius: 6},
	})
	cmds := drain(out)
	if !hasDespawnID(cmds, 40) {
		t.Fatalf("should delete the other shot: %v", cmds)
	}
	if hasDespawnID(cmds, 9) {
		t.Fatalf("mind shot must keep flying: %v", cmds)
	}
}

func TestCanCastAtZero(t *testing.T) {
	a := fighter(SkillMind)
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	enemy := foe(80, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	if !hasSpawn(drain(out), KindMindShot) {
		t.Fatal("should fire at t=0")
	}
}

func TestAspectSpawnsOnEnemyAndLocksDir(t *testing.T) {
	a := fighter(SkillAspect)
	out := make(chan unit.Cmd, 64)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	enemy := foe(80, 0)
	self := me(0, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	var cx, cy float64
	clone := false
	var vx, vy float64
	gotShot := false
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == KindAspect {
			clone = true
			cx, cy = s.X, s.Y
		}
		if s, ok := c.(unit.Spawn); ok && s.Kind == KindAspectShot {
			gotShot = true
			vx, vy = s.VX, s.VY
		}
	}
	if !clone {
		t.Fatalf("missing clone: %v", cmds)
	}
	if math.Hypot(cx-self.X, cy-self.Y) > aspectMax+1e-6 {
		t.Fatalf("clone leash %v > %v at (%v,%v)", math.Hypot(cx-self.X, cy-self.Y), aspectMax, cx, cy)
	}
	if math.Hypot(cx-enemy.X, cy-enemy.Y) < 8 {
		t.Fatalf("clone should not spawn on enemy (%v,%v)", cx, cy)
	}
	if !gotShot {
		t.Fatal("clone should fire")
	}
	moved := foe(80, 40)
	a.Handle(ctx, unit.Sense{Time: aspectGap, Self: self, Nearby: []unit.Snapshot{moved}})
	cmds = drain(out)
	for _, c := range cmds {
		s, ok := c.(unit.Spawn)
		if !ok || s.Kind != KindAspectShot {
			continue
		}
		if math.Abs(s.VX-vx) > 1e-6 || math.Abs(s.VY-vy) > 1e-6 {
			t.Fatalf("aim must stay locked, first=(%v,%v) next=(%v,%v)", vx, vy, s.VX, s.VY)
		}
	}
}

func TestGasCostsOneAndSpawnsOnEnemy(t *testing.T) {
	a := fighter(SkillGas)
	a.energy = 0.9
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	enemy := foe(80, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	if hasSpawn(drain(out), KindGas) {
		t.Fatal("gas needs 1 energy")
	}
	a.energy = 1
	a.Handle(ctx, unit.Sense{Time: 0.05, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	if !hasSpawn(cmds, KindGas) {
		t.Fatalf("should spawn gas: %v", cmds)
	}
	if a.energy > 0.05 {
		t.Fatalf("gas should spend 1, energy=%v", a.energy)
	}
	for _, c := range cmds {
		s, ok := c.(unit.Spawn)
		if !ok || s.Kind != KindGas {
			continue
		}
		if math.Hypot(s.X-enemy.X, s.Y-enemy.Y) > 1 {
			t.Fatalf("dbin 0 should drop on enemy, got (%v,%v)", s.X, s.Y)
		}
	}
}

func TestGasDamagesAndSlows(t *testing.T) {
	g := &瓦斯{owner: 1, slot: 0}
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 9, Kind: KindGas, Out: out}
	self := unit.Snapshot{
		ID: 9, Kind: KindGas, Role: unit.RoleHelper,
		X: 0, Y: 0, Radius: gasRadius, Slot: 0, OwnerID: 1,
	}
	enemy := foe(10, 0)
	enemy.VX = 150
	g.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	if !hasDamageTo(cmds, 2, gasDamage) {
		t.Fatalf("first tick should damage: %v", cmds)
	}
	if !hasCruise(cmds, 2, 150*gasSlow) {
		t.Fatalf("should slow cruise: %v", cmds)
	}
	g.Handle(ctx, unit.Sense{Time: 0.4, Self: self, Nearby: []unit.Snapshot{enemy}})
	if hasDamageTo(drain(out), 2, gasDamage) {
		t.Fatal("no extra tick before 0.5s")
	}
	g.Handle(ctx, unit.Sense{Time: 0.5, Self: self, Nearby: []unit.Snapshot{enemy}})
	if !hasDamageTo(drain(out), 2, gasDamage) {
		t.Fatal("0.5s tick")
	}
	g.Handle(ctx, unit.Sense{Time: 10, Self: self, Nearby: []unit.Snapshot{enemy}})
	cmds = drain(out)
	if !hasDespawnID(cmds, 9) {
		t.Fatalf("gas should end at 10s: %v", cmds)
	}
}

func TestAspectOnlyCloneShoots(t *testing.T) {
	a := fighter(SkillAspect)
	out := make(chan unit.Cmd, 64)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	enemy := foe(80, 0)
	self := me(0, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	shots := 0
	fromSelf := 0
	for _, c := range cmds {
		s, ok := c.(unit.Spawn)
		if !ok || s.Kind != KindAspectShot {
			continue
		}
		shots++
		if math.Hypot(s.X-self.X, s.Y-self.Y) < self.Radius+shotRadius+8 {
			fromSelf++
		}
	}
	if shots != 1 {
		t.Fatalf("first volley shots=%d want 1 from clone", shots)
	}
	if fromSelf != 0 {
		t.Fatalf("body must not fire, fromSelf=%d cmds=%v", fromSelf, cmds)
	}
	for i := 1; i < aspectShots; i++ {
		a.Handle(ctx, unit.Sense{Time: float64(i) * aspectGap, Self: self, Nearby: []unit.Snapshot{enemy}})
		cmds = drain(out)
		n := 0
		for _, c := range cmds {
			if s, ok := c.(unit.Spawn); ok && s.Kind == KindAspectShot {
				n++
				if math.Hypot(s.X-self.X, s.Y-self.Y) < self.Radius+shotRadius+8 {
					t.Fatalf("body fired on shot %d", i+1)
				}
			}
		}
		if n != 1 {
			t.Fatalf("shot %d count=%d", i+1, n)
		}
	}
}

func fighter(skill uint8) *优昙华院 {
	return &优昙华院{
		energy:    energyMax,
		energyCap: energyMax,
		locked:    true,
		force:     skill,
		deck:      []uint8{},
	}
}

func me(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindUdongein, Role: unit.RoleFighter,
		X: x, Y: y, Radius: udongeinRadius, HP: 100, MaxHP: 100, Slot: 0,
	}
}

func foe(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 2, Kind: "原型机_远程", Role: unit.RoleFighter,
		X: x, Y: y, Radius: 18, HP: 100, MaxHP: 100, Slot: 1,
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

func hasSpawn(cmds []unit.Cmd, kind string) bool {
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == kind {
			return true
		}
	}
	return false
}

func hasDamageTo(cmds []unit.Cmd, id uint64, amt float64) bool {
	for _, c := range cmds {
		if d, ok := c.(unit.Damage); ok && d.To == id && d.Amount >= amt-1e-6 {
			return true
		}
	}
	return false
}

func hasDespawnID(cmds []unit.Cmd, id uint64) bool {
	for _, c := range cmds {
		if d, ok := c.(unit.Despawn); ok && d.UnitID == id {
			return true
		}
	}
	return false
}

func hasSetRadius(cmds []unit.Cmd, id uint64, r float64) bool {
	for _, c := range cmds {
		s, ok := c.(unit.SetRadius)
		if !ok || s.UnitID != id {
			continue
		}
		if math.Abs(s.Radius-r) < 1e-6 {
			return true
		}
	}
	return false
}

func velOf(cmds []unit.Cmd, id uint64) float64 {
	for _, c := range cmds {
		v, ok := c.(unit.SetVelocity)
		if !ok || v.UnitID != id {
			continue
		}
		return math.Hypot(v.VX, v.VY)
	}
	return 0
}

func hasDespawnKind(cmds []unit.Cmd, kind string) bool {
	for _, c := range cmds {
		if d, ok := c.(unit.DespawnOwned); ok && d.Kind == kind {
			return true
		}
	}
	return false
}

func hasSelfVelocity(cmds []unit.Cmd, id uint64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.SetVelocity); ok && v.UnitID == id {
			return true
		}
	}
	return false
}

func hasVelocity(cmds []unit.Cmd, id uint64, vx, vy float64) bool {
	for _, c := range cmds {
		v, ok := c.(unit.SetVelocity)
		if !ok || v.UnitID != id {
			continue
		}
		if math.Abs(v.VX-vx) < 1e-6 && math.Abs(v.VY-vy) < 1e-6 {
			return true
		}
	}
	return false
}

func hasCruise(cmds []unit.Cmd, id uint64, speed float64) bool {
	for _, c := range cmds {
		v, ok := c.(unit.SetCruise)
		if !ok || v.UnitID != id {
			continue
		}
		if math.Abs(v.Speed-speed) < 1e-6 {
			return true
		}
	}
	return false
}

func hasFX(cmds []unit.Cmd, name string) bool {
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == name {
			return true
		}
	}
	return false
}

func TestVolleyFiresThreeSeekers(t *testing.T) {
	a := fighter(SkillVolley)
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	enemy := foe(80, 0)
	n := 0
	for i := 0; i < volleyCount; i++ {
		a.Handle(ctx, unit.Sense{Time: float64(i) * volleyGap, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
		cmds := drain(out)
		got := 0
		for _, c := range cmds {
			if s, ok := c.(unit.Spawn); ok && s.Kind == KindSeekShot {
				got++
			}
		}
		if got != 1 {
			t.Fatalf("shot %d count=%d cmds=%v", i+1, got, cmds)
		}
		n += got
	}
	if n != volleyCount {
		t.Fatalf("volley shots=%d want %d", n, volleyCount)
	}
	if a.energy != energyMax-energyCost {
		t.Fatalf("volley should spend 1, energy=%v", a.energy)
	}
}

func TestVolleyAnimLastsUntilLastShot(t *testing.T) {
	a := fighter(SkillVolley)
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	enemy := foe(80, 0)
	for i := 0; i < volleyCount; i++ {
		a.Handle(ctx, unit.Sense{Time: float64(i) * volleyGap, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
		_ = drain(out)
	}
	a.force = SkillLaser
	last := float64(volleyCount-1) * volleyGap
	a.Handle(ctx, unit.Sense{Time: last, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	if hasSpawn(cmds, KindLaser) || hasFX(cmds, "laser-warn") {
		t.Fatalf("last volley shot still in fire window, should not laser: %v", cmds)
	}
	a.Handle(ctx, unit.Sense{Time: last + volleyGap, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds = drain(out)
	if hasSpawn(cmds, KindLaser) || hasFX(cmds, "laser-warn") {
		t.Fatalf("volley recovery should still block laser: %v", cmds)
	}
	done := skillAnim(SkillVolley)
	a.Handle(ctx, unit.Sense{Time: done, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds = drain(out)
	if !hasFX(cmds, "laser-warn") {
		t.Fatalf("after volley recovery should laser: %v", cmds)
	}
}

func TestVolleyLocksAimAtCastNotTrack(t *testing.T) {
	a := fighter(SkillVolley)
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	cmds := drain(out)
	var firstVX, firstVY float64
	got := false
	for _, c := range cmds {
		s, ok := c.(unit.Spawn)
		if !ok || s.Kind != KindSeekShot {
			continue
		}
		got = true
		firstVX, firstVY = s.VX, s.VY
	}
	if !got {
		t.Fatalf("missing first shot: %v", cmds)
	}
	moved := foe(80, 80)
	a.Handle(ctx, unit.Sense{Time: volleyGap, Self: me(0, 0), Nearby: []unit.Snapshot{moved}})
	cmds = drain(out)
	for _, c := range cmds {
		s, ok := c.(unit.Spawn)
		if !ok || s.Kind != KindSeekShot {
			continue
		}
		dot := firstVX*s.VX + firstVY*s.VY
		if dot < 0 {
			t.Fatalf("later shot tracked the moved enemy, first=(%v,%v) next=(%v,%v)", firstVX, firstVY, s.VX, s.VY)
		}
		base := math.Atan2(firstVY, firstVX)
		gotAng := math.Atan2(s.VY, s.VX)
		if math.Abs(gotAng-base) > volleySpread+1e-3 {
			t.Fatalf("aim must stay locked at cast, first=%v next=%v", base, gotAng)
		}
	}
	b := &索敌弹{owner: 1}
	self := unit.Snapshot{
		ID: 9, Kind: KindSeekShot, Role: unit.RoleProjectile,
		X: 0, Y: 0, VX: volleySpeed, Radius: volleyRadius, Slot: 0,
	}
	b.Handle(unit.Context{ID: 9, Kind: KindSeekShot, Out: out}, unit.Sense{
		Time: 0, Self: self, Nearby: []unit.Snapshot{foe(40, 30)},
	})
	if hasSelfVelocity(drain(out), 9) {
		t.Fatal("in-flight shot must not track")
	}
}

func TestFrontHitSpendsEnergyAndGuards(t *testing.T) {
	a := fighter(SkillSteer)
	out := make(chan unit.Cmd, 16)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	self := me(0, 0)
	self.VX, self.VY = 152, 0
	enemy := foe(80, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{enemy}})
	_ = drain(out)
	a.Handle(ctx, unit.IncomingDamage{Token: 3, From: 2, Amount: 10, Time: 0.1})
	cmds := drain(out)
	if math.Abs(a.energy-(energyMax-10/frontCostDiv)) > 1e-6 {
		t.Fatalf("front hit should spend initial 10/%v energy, energy=%v", frontCostDiv, a.energy)
	}
	if !hasConfirm(cmds, 3, 10*frontDR) {
		t.Fatalf("front hit should keep %v of 10: %v", 10*frontDR, cmds)
	}
}

func TestGuardFailDumpsEnergyAndBreaks(t *testing.T) {
	a := fighter(SkillSteer)
	a.energy = 1
	out := make(chan unit.Cmd, 16)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	self := me(0, 0)
	self.VX, self.VY = 152, 0
	a.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{foe(80, 0)}})
	_ = drain(out)
	a.Handle(ctx, unit.IncomingDamage{Token: 8, From: 2, Amount: 10, Time: 0.1})
	cmds := drain(out)
	if a.energy != 0 {
		t.Fatalf("unpaid guard should dump energy, energy=%v", a.energy)
	}
	if !a.drained || a.energyCap != energyMax-1 {
		t.Fatalf("should enter break, drained=%v cap=%v", a.drained, a.energyCap)
	}
	if !hasConfirm(cmds, 8, 10*emptyHurt) {
		t.Fatalf("unpaid guard should take 150%% of raw 10: %v", cmds)
	}
}

func TestRearHitDoesNotGuard(t *testing.T) {
	a := fighter(SkillSteer)
	out := make(chan unit.Cmd, 16)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	self := me(0, 0)
	self.VX, self.VY = 152, 0
	enemy := foe(-80, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{enemy}})
	_ = drain(out)
	a.Handle(ctx, unit.IncomingDamage{Token: 4, From: 2, Amount: 10, Time: 0.1})
	cmds := drain(out)
	if a.energy != energyMax {
		t.Fatalf("rear hit should not spend energy, energy=%v", a.energy)
	}
	if !hasConfirm(cmds, 4, 10) {
		t.Fatalf("rear hit should take full 10: %v", cmds)
	}
}

func TestEmptyEnergyHurtsMoreAndCutsCap(t *testing.T) {
	a := fighter(SkillSteer)
	a.energy = 0
	out := make(chan unit.Cmd, 16)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	self := me(0, 0)
	self.VX = 152
	a.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{foe(80, 0)}})
	_ = drain(out)
	a.Handle(ctx, unit.IncomingDamage{Token: 5, From: 2, Amount: 10, Time: 0.1})
	cmds := drain(out)
	if !a.drained || a.energyCap != energyMax-1 {
		t.Fatalf("empty hit should cut cap, drained=%v cap=%v", a.drained, a.energyCap)
	}
	if !hasConfirm(cmds, 5, 15) {
		t.Fatalf("empty energy should take 150%%: %v", cmds)
	}
	a.Handle(ctx, unit.WallHit{Time: 1, NX: 1, NY: 0})
	if math.Abs(a.energy-energyWall) > 1e-6 {
		t.Fatalf("wall should restore %v, energy=%v", energyWall, a.energy)
	}
	if !a.drained || a.energyCap != energyMax-1 {
		t.Fatalf("drain lasts until energy reaches cap, drained=%v cap=%v", a.drained, a.energyCap)
	}
}

func TestCardChargeDrawsAndCosts(t *testing.T) {
	a := fighter(SkillSteer)
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	_ = drain(out)
	a.cardMu.Lock()
	a.deck = []uint8{SkillMind, SkillDose, SkillLaser, SkillAspect}
	a.hand = nil
	a.progress = 0
	a.cardMu.Unlock()
	for i := 0; i < 10; i++ {
		a.charge()
	}
	if len(a.hand) != 1 {
		t.Fatalf("10 charges should draw one card, hand=%v", a.hand)
	}
	sk := a.hand[0]
	if sk != SkillMind && sk != SkillDose && sk != SkillLaser && sk != SkillAspect {
		t.Fatalf("drawn card %v not from deck", sk)
	}
	a.locked = false
	a.animUntil = 0
	a.energy = 0
	a.hand = []uint8{SkillDose, SkillMind, SkillLaser}
	a.Handle(ctx, unit.Sense{Time: 1, Self: me(0, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	cmds := drain(out)
	if a.doses != 1 {
		t.Fatalf("cost3 dose should cast, doses=%d cmds=%v", a.doses, cmds)
	}
	if len(a.hand) != 0 {
		t.Fatalf("cost3 should eat this card and the next two, hand=%v", a.hand)
	}
	if a.energy != 0 {
		t.Fatalf("spell card must not spend energy, energy=%v", a.energy)
	}
}

func TestAttackCardCastsAndUpgrades(t *testing.T) {
	a := fighter(SkillSteer)
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	_ = drain(out)
	a.locked = false
	a.animUntil = 0
	a.energy = 0
	a.hand = []uint8{SkillMind}
	a.Handle(ctx, unit.Sense{Time: 1, Self: me(0, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	cmds := drain(out)
	if !hasSpawn(cmds, KindMindShot) {
		t.Fatalf("attack card should fire mind: %v", cmds)
	}
	if a.upgrades[SkillMind] != 1 {
		t.Fatalf("attack card should upgrade, upgrades=%v", a.upgrades[SkillMind])
	}
	if math.Abs(ownerSkill(1, SkillMind)-1.2) > 1e-6 {
		t.Fatalf("mind mul=%v want 1.2", ownerSkill(1, SkillMind))
	}
	if a.energy != 0 {
		t.Fatalf("attack card should not spend energy, energy=%v", a.energy)
	}
}

func TestBlastHurtsEnemyOnContact(t *testing.T) {
	b := &爆药{owner: 1, slot: 0}
	out := make(chan unit.Cmd, 16)
	ctx := unit.Context{ID: 9, Kind: KindBlast, Out: out}
	self := unit.Snapshot{
		ID: 9, Kind: KindBlast, Role: unit.RoleHelper,
		X: 0, Y: 0, Radius: blastRadius, Slot: 0, OwnerID: 1,
	}
	enemy := foe(10, 0)
	b.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	if !hasDamageTo(cmds, 2, blastDamage) {
		t.Fatalf("blast should hit enemy: %v", cmds)
	}
	if hasDamageTo(cmds, 1, 1) {
		t.Fatalf("blast must not hit owner: %v", cmds)
	}
	b.Handle(ctx, unit.Sense{Time: 0.2, Self: self, Nearby: []unit.Snapshot{enemy}})
	if hasDamageTo(drain(out), 2, blastDamage) {
		t.Fatal("same enemy should only be hit once")
	}
}

func hasConfirm(cmds []unit.Cmd, token uint64, amt float64) bool {
	for _, c := range cmds {
		d, ok := c.(unit.ConfirmDamage)
		if !ok || d.Token != token {
			continue
		}
		if math.Abs(d.Amount-amt) < 1e-6 {
			return true
		}
	}
	return false
}
