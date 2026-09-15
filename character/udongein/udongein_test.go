package 优昙华院

import (
	"math"
	"testing"
	"xqdj/internal/unit"
)

func TestPublicAndSkillCooldown(t *testing.T) {
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

	a.locked = true
	a.force = SkillLaser
	a.Handle(ctx, unit.Sense{Time: 1.4, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds = drain(out)
	if hasSpawn(cmds, KindLaser) {
		t.Fatalf("public cd 1.5s should block laser at 1.4s")
	}

	a.Handle(ctx, unit.Sense{Time: 1.5, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds = drain(out)
	if !hasSpawn(cmds, KindLaser) {
		t.Fatalf("t=1.5 should allow other skill after public cd: %v", cmds)
	}
}

func TestSkillCooldownTwoSeconds(t *testing.T) {
	a := fighter(SkillMind)
	out := make(chan unit.Cmd, 64)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	enemy := foe(80, 0)

	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	if !hasSpawn(drain(out), KindMindShot) {
		t.Fatal("t=0 mind")
	}

	a.Handle(ctx, unit.Sense{Time: 1.5, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	if hasSpawn(drain(out), KindMindShot) {
		t.Fatal("skill cd 2s should block mind at t=1.5 even though public cd is up")
	}

	a.Handle(ctx, unit.Sense{Time: 2.0, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	if !hasSpawn(drain(out), KindMindShot) {
		t.Fatal("t=2 should free mind")
	}
}

func TestSpellEnergyCostAndRegen(t *testing.T) {
	a := fighter(SkillCrown)
	a.energy = 0.9
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	enemy := foe(120, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	if hasSpawn(drain(out), KindCrown) {
		t.Fatal("crown needs 1 energy")
	}
	a.energy = 1
	a.Handle(ctx, unit.Sense{Time: 0.05, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	if !hasSpawn(drain(out), KindCrown) {
		t.Fatal("crown at 1 energy")
	}
	if a.energy > 0.05 {
		t.Fatalf("crown should spend 1, energy=%v", a.energy)
	}

	a.locked = true
	a.force = SkillSteer
	a.Handle(ctx, unit.Sense{Time: 1.05, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	_ = drain(out)
	want := energyRegen
	if a.energy < want-1e-6 || a.energy > want+0.05 {
		t.Fatalf("regen %v/s, energy=%v", energyRegen, a.energy)
	}
}

func TestNoEnergyNoCast(t *testing.T) {
	a := &优昙华院{energy: 0.05}
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	cmds := drain(out)
	if hasSpawn(cmds, KindMindShot) || hasSpawn(cmds, KindLaser) || hasSpawn(cmds, KindCrown) {
		t.Fatalf("no energy must not cast: %v", cmds)
	}
	if a.energy > 0.05+1e-6 {
		t.Fatalf("energy=%v", a.energy)
	}
}

func TestRandomCastSpendsEnergy(t *testing.T) {
	a := &优昙华院{energy: energyMax}
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
		a := &优昙华院{energy: energyMax}
		out := make(chan unit.Cmd, 32)
		ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
		a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
		tag := "none"
		for _, c := range drain(out) {
			switch x := c.(type) {
			case unit.Spawn:
				tag = x.Kind
			case unit.FX:
				if x.Name == "dose" {
					tag = "dose"
				}
			case unit.Damage:
				if tag == "none" {
					tag = "break"
				}
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

func TestDoseFourKillsSelf(t *testing.T) {
	a := fighter(SkillDose)
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	enemy := foe(200, 0)
	t0 := 0.0
	for i := 0; i < 4; i++ {
		a.energy = energyMax
		a.gcdUntil = 0
		a.skillCD[SkillDose] = 0
		a.Handle(ctx, unit.Sense{Time: t0, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
		cmds := drain(out)
		suicided := hasDamageTo(cmds, 1, doseSuicide)
		if i < 3 && suicided {
			t.Fatalf("suicide at dose %d", i+1)
		}
		if i == 3 && !suicided {
			t.Fatalf("4th dose should suicide: %v", cmds)
		}
		t0 += 3
	}
	if a.doses != 4 {
		t.Fatalf("doses=%d", a.doses)
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
	a.Handle(ctx, unit.Sense{Time: 0.5, Self: self, Nearby: []unit.Snapshot{enemy}})
	if hasVelocity(drain(out), 1, 40, 80) {
		t.Fatal("must stay locked during laser")
	}
	a.locked = true
	a.force = SkillSteer
	a.Handle(ctx, unit.Sense{Time: 1.0, Self: self, Nearby: []unit.Snapshot{enemy}})
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

func TestNoZeroFrameCast(t *testing.T) {
	a := &优昙华院{energy: energyMax, locked: true, force: SkillMind, gcdUntil: publicCD}
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindUdongein, Out: out}
	enemy := foe(80, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	if hasSpawn(drain(out), KindMindShot) {
		t.Fatal("opening public cd must block t=0")
	}
	a.Handle(ctx, unit.Sense{Time: publicCD, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	if !hasSpawn(drain(out), KindMindShot) {
		t.Fatal("should fire after opening public cd")
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
	return &优昙华院{energy: energyMax, locked: true, force: skill}
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
