package pink

import (
	"testing"

	"xqdj/internal/unit"
)

func TestRequestedWeaponDamageValues(t *testing.T) {
	if pistolDmg != 5 {
		t.Fatalf("pistol damage=%v, want 5", pistolDmg)
	}
	if bottleImpactDamage != 5 {
		t.Fatalf("bottle impact damage=%v, want 5", bottleImpactDamage)
	}
	if shotgunDmg != 5 {
		t.Fatalf("shotgun pellet damage=%v, want 5", shotgunDmg)
	}
	if gatlingDmg != 4 {
		t.Fatalf("gatling damage=%v, want 4", gatlingDmg)
	}
	if pistolCD != 0.5 || pistolShots != 20 {
		t.Fatalf("pistol config=%v/%d, want 0.5/20", pistolCD, pistolShots)
	}
	if rifleCD != 2 || shotgunCD != 2 {
		t.Fatalf("rifle/shotgun cooldowns=%v/%v, want 2/2", rifleCD, shotgunCD)
	}
	if shotgunUses != 7 || rifleShots != 7 {
		t.Fatalf("shotgun/rifle uses=%d/%d, want 7/7", shotgunUses, rifleShots)
	}
	if rocketCD != 6 || rocketShots != 5 {
		t.Fatalf("rocket config=%v/%d, want 6/5", rocketCD, rocketShots)
	}
	if gatlingMag != 60 {
		t.Fatalf("gatling magazine=%d, want 60", gatlingMag)
	}
	if dropChance != 0.50 {
		t.Fatalf("drop chance=%v, want 0.50", dropChance)
	}
	if crowdMax != 6 {
		t.Fatalf("crowd max=%d, want 6", crowdMax)
	}
	if crowdKillHeal != 3 || fireDamage != 4 {
		t.Fatalf("crowd heal/fire damage=%v/%v, want 3/4", crowdKillHeal, fireDamage)
	}
}

func TestCrowdKillGrantsHalfSecondInvulnerability(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	body := &pink{now: 5, slot: 0}
	ctx := unit.Context{ID: 1, Kind: KindPink, Out: out}
	body.Handle(ctx, unit.Kill{VictimID: 10, VictimKind: KindCrowd, VictimRole: unit.RoleMinion, Mortal: true})
	if body.invulnUntil != 5.5 {
		t.Fatalf("invulnerability until=%v, want 5.5", body.invulnUntil)
	}
	body.Handle(ctx, unit.IncomingDamage{Token: 3, Amount: 10, Time: 5.25})
	blocked := false
	healed := false
	for _, cmd := range drainPink(out) {
		if heal, ok := cmd.(unit.Heal); ok && heal.UnitID == 1 && heal.Amount == crowdKillHeal {
			healed = true
		}
		if b, ok := cmd.(unit.BlockDamage); ok && b.Token == 3 {
			blocked = true
		}
	}
	if !healed {
		t.Fatal("crowd kill should heal Pink by three health")
	}
	if !blocked {
		t.Fatal("damage within half a second of a crowd kill should be blocked")
	}
}

func TestCrowdKillsAdvanceEveryTwoAndResetWeapon(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	body := &pink{form: formPlain, weapon: weaponPistol, ammo: 7, rng: unit.NewRNG(1)}
	ctx := unit.Context{ID: 1, Kind: KindPink, Out: out}
	body.Handle(ctx, unit.Kill{VictimID: 10, VictimKind: KindCrowd, VictimRole: unit.RoleMinion, Mortal: true})
	if body.form != formPlain {
		t.Fatalf("form after one kill = %d, want plain", body.form)
	}
	body.Handle(ctx, unit.Kill{VictimID: 11, VictimKind: KindCrowd, VictimRole: unit.RoleMinion, Mortal: true})
	if body.form != formLaugh {
		t.Fatalf("form after two kills = %d, want laugh", body.form)
	}
	if body.weapon != weaponDefault || body.ammo != 0 {
		t.Fatalf("weapon should reset on form change, got weapon=%d ammo=%d", body.weapon, body.ammo)
	}
}

func TestCrowdOnlyTakesDamageFromPink(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	body := &crowd{
		owner: 1,
		seen: map[uint64]unit.Snapshot{
			8: {OwnerID: 1},
			9: {OwnerID: 2},
		},
	}
	ctx := unit.Context{ID: 3, Kind: KindCrowd, Out: out}
	body.Handle(ctx, unit.IncomingDamage{Token: 1, From: 2, Amount: 6})
	body.Handle(ctx, unit.IncomingDamage{Token: 2, From: 1, Amount: 6})
	body.Handle(ctx, unit.IncomingDamage{Token: 3, From: 8, Amount: 6})
	confirmed := map[uint64]bool{}
	blocked := map[uint64]bool{}
	for _, cmd := range drainPink(out) {
		switch hit := cmd.(type) {
		case unit.ConfirmDamage:
			confirmed[hit.Token] = true
		case unit.BlockDamage:
			blocked[hit.Token] = true
		}
	}
	if !blocked[1] || !confirmed[2] || !confirmed[3] {
		t.Fatalf("crowd damage gates blocked=%v confirmed=%v", blocked, confirmed)
	}
}

func TestPlainCollisionDealsSix(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	body := &pink{form: formPlain, weapon: weaponDefault, slot: 0}
	body.Handle(unit.Context{ID: 1, Kind: KindPink, Out: out}, unit.Collision{
		Time:  1,
		Other: unit.Snapshot{ID: 2, Slot: 1, Role: unit.RoleFighter},
	})
	cmds := drainPink(out)
	if d := firstDamage(cmds, 2); d == nil || d.Amount != 6 {
		t.Fatalf("plain collision damage = %+v, want 6", d)
	}
}

func TestShotgunFiresSixPelletsAndExpiresAfterConfiguredUses(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	body := &pink{
		form: formLaugh, weapon: weaponShotgun, ammo: shotgunUses, slot: 0,
		x: 0, y: 0, seen: map[uint64]unit.Snapshot{},
	}
	target := unit.Snapshot{
		ID: 2, Kind: "靶子", Role: unit.RoleFighter, Slot: 1, AimPriority: 15,
		X: 120, Y: 0, Radius: 18,
	}
	body.tickWeapon(unit.Context{ID: 1, Kind: KindPink, Out: out}, unit.Sense{
		Time: 1, Self: unit.Snapshot{ID: 1, Kind: KindPink, Slot: 0},
		Nearby: []unit.Snapshot{target},
	})
	n := countSpawns(drainPink(out), KindShotgunShot)
	if n != shotgunPellets {
		t.Fatalf("shotgun pellets = %d, want %d", n, shotgunPellets)
	}
	if body.weapon != weaponShotgun || body.ammo != shotgunUses-1 {
		t.Fatalf("after first shotgun use got weapon=%d ammo=%d", body.weapon, body.ammo)
	}
	if body.nextAttackAt != 1+shotgunCD {
		t.Fatalf("shotgun cooldown nextAttackAt=%v, want %v", body.nextAttackAt, 1+shotgunCD)
	}
	for shot := 1; shot < shotgunUses; shot++ {
		body.tickWeapon(unit.Context{ID: 1, Kind: KindPink, Out: out}, unit.Sense{
			Time:   1 + float64(shot)*shotgunCD,
			Self:   unit.Snapshot{ID: 1, Kind: KindPink, Slot: 0},
			Nearby: []unit.Snapshot{target},
		})
		n += countSpawns(drainPink(out), KindShotgunShot)
	}
	if n != shotgunPellets*shotgunUses {
		t.Fatalf("total shotgun pellets = %d, want %d", n, shotgunPellets*shotgunUses)
	}
	if body.weapon != weaponDefault || body.ammo != 0 {
		t.Fatalf("shotgun should expire after three uses, got weapon=%d ammo=%d", body.weapon, body.ammo)
	}
}

func TestGatlingReloadsOnceThenExpires(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	ctx := unit.Context{ID: 1, Kind: KindPink, Out: out}
	body := &pink{
		form: formManic, weapon: weaponGatling, ammo: 1, slot: 0,
		x: 0, y: 0, seen: map[uint64]unit.Snapshot{},
	}
	target := unit.Snapshot{
		ID: 2, Role: unit.RoleFighter, Slot: 1, AimPriority: 15,
		X: 100, Radius: 18,
	}
	s := unit.Sense{Time: 1, Self: unit.Snapshot{ID: 1, Slot: 0}, Nearby: []unit.Snapshot{target}}
	body.tickWeapon(ctx, s)
	if !body.reloading || body.reloads != 1 {
		t.Fatalf("first empty magazine should reload, reloading=%v reloads=%d", body.reloading, body.reloads)
	}
	body.reloading = false
	body.ammo = 1
	body.nextAttackAt = 0
	s.Time = 8
	body.tickWeapon(ctx, s)
	if body.weapon != weaponDefault || body.ammo != 0 {
		t.Fatalf("second empty magazine should expire, got weapon=%d ammo=%d", body.weapon, body.ammo)
	}
}

func TestFuelDashHitsThreeTimesAndBlocksDamage(t *testing.T) {
	out := make(chan unit.Cmd, 24)
	ctx := unit.Context{ID: 1, Kind: KindPink, Out: out}
	target := unit.Snapshot{
		ID: 2, Role: unit.RoleFighter, Slot: 1, AimPriority: 15,
		X: 30, Radius: 18,
	}
	body := &pink{
		form: formManic, weapon: weaponFuel, ammo: fuelDashes, slot: 0,
		x: 0, y: 0, hx: 1, seen: map[uint64]unit.Snapshot{2: target},
	}
	body.startFuelDash(ctx, 1)
	cmds := drainPink(out)
	total, hits := damageTotal(cmds, 2)
	if hits != sawHits || total != sawHits*sawHit {
		t.Fatalf("fuel hits=%d total=%v, want %d hits and %v", hits, total, sawHits, sawHits*sawHit)
	}
	body.Handle(ctx, unit.IncomingDamage{Token: 3, From: 2, Amount: 99, Time: 1.1})
	blocked := false
	for _, cmd := range drainPink(out) {
		if b, ok := cmd.(unit.BlockDamage); ok && b.Token == 3 {
			blocked = true
		}
	}
	if !blocked {
		t.Fatal("fuel dash should temporarily block incoming damage")
	}
}

func TestRocketExplosionDamagesEachTargetOnce(t *testing.T) {
	out := make(chan unit.Cmd, 24)
	body := &bullet{
		owner: 1, slot: 0, kind: KindRocketShot, damage: rocketDmg,
		lastNear: []unit.Snapshot{
			{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 8, Y: 0, Radius: 18},
			{ID: 3, Role: unit.RoleFighter, Slot: 1, X: 120, Y: 0, Radius: 18},
		},
	}
	impact := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 0, Y: 0, Radius: 18}
	body.Handle(unit.Context{ID: 9, Kind: KindRocketShot, Out: out}, unit.Collision{Time: 1, Other: impact})
	cmds := drainPink(out)
	if total, hits := damageTotal(cmds, 2); hits != 1 || total != rocketDmg {
		t.Fatalf("impact target hits=%d total=%v, want one %v hit", hits, total, rocketDmg)
	}
	if total, hits := damageTotal(cmds, 3); hits != 0 || total != 0 {
		t.Fatalf("out-of-radius target hits=%d total=%v, want none", hits, total)
	}
	if !hasDespawn(cmds, 9) {
		t.Fatal("rocket should despawn after exploding")
	}
}

func TestAllPinkBulletsDieOnWall(t *testing.T) {
	for _, kind := range []string{KindPistolShot, KindShotgunShot, KindRifleShot, KindGatlingShot} {
		out := make(chan unit.Cmd, 4)
		body := &bullet{owner: 1, slot: 0, kind: kind, damage: 3}
		body.Handle(unit.Context{ID: 8, Kind: kind, Out: out}, unit.WallHit{Time: 1, X: 50, Y: 20})
		if !hasDespawn(drainPink(out), 8) {
			t.Fatalf("%s did not despawn on wall", kind)
		}
	}
}

func TestFireAreaCanSeeTargets(t *testing.T) {
	spec, ok := unit.Lookup(KindFire)
	if !ok {
		t.Fatal("fire area kind is not registered")
	}
	if spec.Vision < pinkVision {
		t.Fatalf("fire area vision=%v, want full-field vision", spec.Vision)
	}
}

func TestBottleImpactDealsDamageAndSpawnsFire(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	body := &bottleShot{owner: 1, slot: 0}
	body.Handle(unit.Context{ID: 8, Kind: KindBottleShot, Out: out}, unit.Collision{
		Time:  1,
		Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 30, Y: 20},
	})
	cmds := drainPink(out)
	if total, hits := damageTotal(cmds, 2); hits != 1 || total != bottleImpactDamage {
		t.Fatalf("bottle impact hits=%d total=%v, want one %v hit", hits, total, bottleImpactDamage)
	}
	if countSpawns(cmds, KindFire) != 1 {
		t.Fatal("bottle impact should spawn one fire area")
	}
	stunned := false
	for _, cmd := range cmds {
		if stun, ok := cmd.(unit.Stun); ok && stun.UnitID == 2 && stun.Hold && stun.Until == 1+bottleStun {
			stunned = true
		}
	}
	if !stunned {
		t.Fatal("bottle impact should stun the hit target")
	}
}

func TestFireAreaTicksAndExpires(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	ctx := unit.Context{ID: 7, Kind: KindFire, Out: out}
	body := &fireArea{}
	self := unit.Snapshot{ID: 7, Kind: KindFire, Role: unit.RoleHelper, OwnerID: 1, Slot: 0, X: 0, Y: 0, Radius: fireRadius}
	foe := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 90, Radius: 18}
	body.Handle(ctx, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{foe}})
	cmds := drainPink(out)
	if total, hits := damageTotal(cmds, 2); hits != 1 || total != fireDamage {
		t.Fatalf("first fire tick hits=%d total=%v", hits, total)
	}
	slowed := false
	for _, cmd := range cmds {
		if fs, ok := cmd.(unit.AddFSComponent); ok && fs.UnitID == 2 && fs.Zone == unit.FSZoneM && fs.Value == fireSlow {
			slowed = true
		}
	}
	if !slowed {
		t.Fatal("fire area should slow targets inside its enlarged radius")
	}
	body.Handle(ctx, unit.Sense{Time: 1.2, Self: self, Nearby: []unit.Snapshot{foe}})
	if cmds := drainPink(out); len(cmds) != 0 {
		t.Fatalf("fire ticked too early: %+v", cmds)
	}
	body.Handle(ctx, unit.Sense{Time: 6, Self: self, Nearby: []unit.Snapshot{foe}})
	if !hasDespawn(drainPink(out), 7) {
		t.Fatal("fire should disappear after five seconds")
	}
}

func drainPink(out chan unit.Cmd) []unit.Cmd {
	var cmds []unit.Cmd
	for {
		select {
		case cmd := <-out:
			cmds = append(cmds, cmd)
		default:
			return cmds
		}
	}
}

func firstDamage(cmds []unit.Cmd, to uint64) *unit.Damage {
	for _, cmd := range cmds {
		if d, ok := cmd.(unit.Damage); ok && d.To == to {
			cp := d
			return &cp
		}
	}
	return nil
}

func countSpawns(cmds []unit.Cmd, kind string) int {
	n := 0
	for _, cmd := range cmds {
		if spawn, ok := cmd.(unit.Spawn); ok && spawn.Kind == kind {
			n++
		}
	}
	return n
}

func damageTotal(cmds []unit.Cmd, to uint64) (float64, int) {
	total := 0.0
	hits := 0
	for _, cmd := range cmds {
		if d, ok := cmd.(unit.Damage); ok && d.To == to {
			total += d.Amount
			hits++
		}
	}
	return total, hits
}

func hasDespawn(cmds []unit.Cmd, id uint64) bool {
	for _, cmd := range cmds {
		if d, ok := cmd.(unit.Despawn); ok && d.UnitID == id {
			return true
		}
	}
	return false
}
