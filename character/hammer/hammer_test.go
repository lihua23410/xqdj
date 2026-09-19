package 钉与锤

import (
	"math"
	"testing"
	"xqdj/internal/unit"
)

func TestHammerSmashStunsInMelee(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	h := &钉与锤{hammerArmed: true, nailReadyAt: 999, ritualReadyAt: 999, dollsSpawned: true}
	ctx := unit.Context{ID: 1, Kind: KindHammer, Out: out}
	self := me(0, 0)
	enemy := foe(30, 0)
	enemy.VX, enemy.VY = 90, 10
	h.Handle(ctx, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	st := lastStun(cmds)
	if st == nil || st.UnitID != 2 || !st.Hold {
		t.Fatalf("stun=%v", cmds)
	}
	v := lastVel(cmds, 2)
	if v == nil || v.VX <= 0 {
		t.Fatalf("knockback=%v", cmds)
	}
	if lastNamed(cmds, "hammer") == nil {
		t.Fatalf("missing hammer fx: %v", cmds)
	}
	if math.Abs(h.hammerReadyAt-(1+hammerCD)) > 1e-9 {
		t.Fatalf("hammer CD=%v want %v", h.hammerReadyAt, 1+hammerCD)
	}
}

func TestHammerStunRestoresCruise(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	h := &钉与锤{hammerArmed: true, nailReadyAt: 999, ritualReadyAt: 999, dollsSpawned: true}
	ctx := unit.Context{ID: 1, Kind: KindHammer, Out: out}
	self := me(0, 0)
	enemy := foe(30, 0)
	h.Handle(ctx, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{enemy}})
	_ = drain(out)

	h.Handle(ctx, unit.Sense{Time: 1 + hammerStunDur, Self: self, Nearby: []unit.Snapshot{enemy}})
	end := drain(out)
	st := lastStun(end)
	if st == nil || st.Hold || st.UnitID != 2 {
		t.Fatalf("release stun=%v cmds=%v", st, end)
	}
	v := lastVel(end, 2)
	if v == nil || math.Hypot(v.VX, v.VY) < 1 {
		t.Fatalf("release should restore motion, vel=%v cmds=%v", v, end)
	}
}

func TestHammerSmashHitsMinion(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	h := &钉与锤{hammerArmed: true, nailReadyAt: 999, ritualReadyAt: 999, dollsSpawned: true}
	ctx := unit.Context{ID: 1, Kind: KindHammer, Out: out}
	minion := unit.Snapshot{
		ID: 3, Kind: "活随从", Role: unit.RoleMinion, Slot: 1, Mortal: true,
		X: 28, Y: 0, Radius: 12, OwnerID: 2,
	}
	h.Handle(ctx, unit.Sense{Time: 1, Self: me(0, 0), Nearby: []unit.Snapshot{minion}})
	cmds := drain(out)
	st := lastStun(cmds)
	if st == nil || st.UnitID != 3 || !st.Hold {
		t.Fatalf("minion stun=%v cmds=%v", st, cmds)
	}
}

func TestHammerSmashDoesNotMoveDoll(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	h := &钉与锤{hammerArmed: true, nailReadyAt: 999, ritualReadyAt: 999, dollsSpawned: true}
	ctx := unit.Context{ID: 1, Kind: KindHammer, Out: out}
	doll := unit.Snapshot{
		ID: 10, Kind: KindDoll, Role: unit.RoleMinion, Slot: 1, Mortal: true,
		X: 26, Y: 0, Radius: dollRadius, OwnerID: 1, HP: 40,
	}
	h.Handle(ctx, unit.Sense{Time: 1, Self: me(0, 0), Nearby: []unit.Snapshot{doll}})
	cmds := drain(out)
	if lastStun(cmds) != nil {
		t.Fatalf("doll should not be displaced: %v", cmds)
	}
	if lastVel(cmds, 10) != nil {
		t.Fatalf("doll should not be knocked: %v", cmds)
	}
	if d := damageTo(cmds, 10); d == nil || d.Amount != swingDamage {
		t.Fatalf("doll should still take swing, dmg=%v cmds=%v", d, cmds)
	}
}

func TestHammerSmashHitsOwnDoll(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	h := &钉与锤{hammerArmed: true, nailReadyAt: 999, ritualReadyAt: 999, dollsSpawned: true}
	ctx := unit.Context{ID: 1, Kind: KindHammer, Out: out}
	doll := unit.Snapshot{
		ID: 10, Kind: KindDoll, Role: unit.RoleMinion, Slot: 1, Mortal: true,
		X: 26, Y: 0, Radius: dollRadius, OwnerID: 1,
	}
	h.Handle(ctx, unit.Sense{Time: 1, Self: me(0, 0), Nearby: []unit.Snapshot{doll}})
	cmds := drain(out)
	if lastNamed(cmds, "hammer") == nil {
		t.Fatalf("own doll should still take swing: %v", cmds)
	}
	if lastStun(cmds) != nil {
		t.Fatalf("own doll should not be displaced: %v", cmds)
	}
}

func TestArcCopiesDollHitImmediately(t *testing.T) {
	rememberFoe(1, 2)
	t.Cleanup(func() { rememberFoe(1, 0) })
	out := make(chan unit.Cmd, 16)
	a := &弧{owner: 1, slot: 0}
	ctx := unit.Context{ID: 8, Kind: KindHammerArc, Out: out}
	doll := unit.Snapshot{
		ID: 10, Kind: KindDoll, Role: unit.RoleMinion, Slot: 1, Mortal: true,
		OwnerID: 1, HP: 40, Radius: dollRadius,
	}
	a.Handle(ctx, unit.Collision{Time: 1, Other: doll})
	cmds := drain(out)
	if d := damageTo(cmds, 10); d == nil || d.Amount != arcDamage {
		t.Fatalf("doll dmg=%v cmds=%v", d, cmds)
	}
	if d := damageTo(cmds, 2); d == nil || d.Amount != arcDamage || d.From != 1 {
		t.Fatalf("copy dmg=%v cmds=%v", d, cmds)
	}
}

func TestNailHitPinsThenReleases(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	n := &钉{owner: 1, slot: 0, spawnTime: 0}
	ctx := unit.Context{ID: 9, Kind: KindNail, Out: out}
	enemy := foe(200, 0)
	enemy.VX, enemy.VY = 80, 0
	n.Handle(ctx, unit.Collision{Time: 1, Other: enemy})
	hit := drain(out)
	st := lastStun(hit)
	if st == nil || st.UnitID != 2 || !st.Hold {
		t.Fatalf("hit stun=%v", hit)
	}
	if lastNamed(hit, "nail-hit") == nil {
		t.Fatalf("missing nail-hit fx: %v", hit)
	}
	if lastTeleport(hit) == nil {
		t.Fatalf("nail should stick to target: %v", hit)
	}

	n.Handle(ctx, unit.Sense{
		Time: 1 + nailPushWait + 0.05,
		Self: unit.Snapshot{ID: 9, Kind: KindNail, Role: unit.RoleProjectile, Slot: 0, X: 200, Y: 0},
		Nearby: []unit.Snapshot{
			{ID: 2, Kind: KindHammer, Role: unit.RoleFighter, Slot: 1, X: 240, Y: 0, Radius: 18, VX: 80, VY: 0},
		},
	})
	lock := drain(out)
	if !n.locked {
		t.Fatalf("should lock after push wait, cmds=%v", lock)
	}
	if v := lastVel(lock, 2); v == nil || math.Hypot(v.VX, v.VY) > 1e-6 {
		t.Fatalf("lock should zero target vel, vel=%v", v)
	}

	n.Handle(ctx, unit.Sense{
		Time: n.lockTime + nailLockDur,
		Self: unit.Snapshot{ID: 9, Kind: KindNail, Role: unit.RoleProjectile, Slot: 0, X: 240, Y: 0},
		Nearby: []unit.Snapshot{
			{ID: 2, Kind: KindHammer, Role: unit.RoleFighter, Slot: 1, X: 240, Y: 0, Radius: 18, VX: 0, VY: 0},
		},
	})
	end := drain(out)
	rel := lastStun(end)
	if rel == nil || rel.Hold {
		t.Fatalf("release stun=%v cmds=%v", rel, end)
	}
	v := lastVel(end, 2)
	if v == nil || math.Hypot(v.VX, v.VY) < 1 {
		t.Fatalf("nailed target must move again, vel=%v cmds=%v", v, end)
	}
	if !hasDespawn(end, 9) {
		t.Fatalf("nail should despawn: %v", end)
	}
}

func TestNailIgnoresSecondHit(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	n := &钉{owner: 1, slot: 0, spawnTime: 0, hit: true, enemyID: 2}
	ctx := unit.Context{ID: 9, Kind: KindNail, Out: out}
	n.Handle(ctx, unit.Collision{Time: 2, Other: foe(10, 0)})
	if cmds := drain(out); len(cmds) != 0 {
		t.Fatalf("second collision should be ignored: %v", cmds)
	}
}

func TestNailWallHitComputesBounce(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	n := &钉{owner: 1, slot: 0, spawnTime: 0, phase: nailFly, flyX: nailSpeed, flyY: 0}
	ctx := unit.Context{ID: 9, Kind: KindNail, Out: out}
	n.Handle(ctx, unit.WallHit{Time: 1, NX: 1, NY: 0})
	cmds := drain(out)
	v := lastVel(cmds, 9)
	if v == nil || v.VX >= 0 || math.Abs(v.VY) > 1 {
		t.Fatalf("nail should bounce by computed dir, vel=%v cmds=%v", v, cmds)
	}
	if math.Abs(math.Hypot(v.VX, v.VY)-nailSpeed) > 1 {
		t.Fatalf("bounce speed=%v", math.Hypot(v.VX, v.VY))
	}

	n.Handle(ctx, unit.WallHit{Time: 2, NX: 1, NY: 0})
	n.Handle(ctx, unit.WallHit{Time: 3, NX: 1, NY: 0})
	end := drain(out)
	if !hasDespawn(end, 9) {
		t.Fatalf("third bounce should despawn: %v", end)
	}
}

func TestNailHitsLivingMinion(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	n := &钉{owner: 1, slot: 0, spawnTime: 0}
	ctx := unit.Context{ID: 9, Kind: KindNail, Out: out}
	minion := unit.Snapshot{
		ID: 4, Kind: "活随从", Role: unit.RoleMinion, Slot: 1, Mortal: true,
		X: 40, Y: 0, Radius: 12, HP: 30, VX: 80,
	}
	n.Handle(ctx, unit.Collision{Time: 1, Other: minion})
	cmds := drain(out)
	if d := damageTo(cmds, 4); d == nil || d.Amount != nailDamage {
		t.Fatalf("minion dmg=%v cmds=%v", d, cmds)
	}
	if m := lastStack(cmds, 4); m == nil || m.Kind != curseKind || m.Delta != 1 {
		t.Fatalf("minion curse=%v cmds=%v", m, cmds)
	}
	st := lastStun(cmds)
	if st == nil || st.UnitID != 4 || !st.Hold {
		t.Fatalf("minion should be pinned, stun=%v cmds=%v", st, cmds)
	}
}

func TestNailHitsDollWithoutDisplace(t *testing.T) {
	rememberFoe(1, 2)
	t.Cleanup(func() { rememberFoe(1, 0) })
	out := make(chan unit.Cmd, 32)
	n := &钉{owner: 1, slot: 0, spawnTime: 0}
	ctx := unit.Context{ID: 9, Kind: KindNail, Out: out}
	doll := unit.Snapshot{
		ID: 10, Kind: KindDoll, Role: unit.RoleMinion, Slot: 1, Mortal: true,
		OwnerID: 1, HP: 40, X: 20, Y: 0, Radius: dollRadius,
	}
	n.Handle(ctx, unit.Collision{Time: 1, Other: doll})
	cmds := drain(out)
	if d := damageTo(cmds, 10); d == nil || d.Amount != nailDamage {
		t.Fatalf("doll dmg=%v cmds=%v", d, cmds)
	}
	if d := damageTo(cmds, 2); d == nil || d.Amount != nailDamage || d.From != 1 {
		t.Fatalf("copy dmg=%v cmds=%v", d, cmds)
	}
	if m := lastStack(cmds, 10); m == nil || m.Kind != curseKind {
		t.Fatalf("doll curse=%v cmds=%v", m, cmds)
	}
	if lastStun(cmds) != nil {
		t.Fatalf("doll should not be displaced: %v", cmds)
	}
	if lastVel(cmds, 10) != nil {
		t.Fatalf("doll should keep velocity: %v", cmds)
	}
	if n.hit {
		t.Fatalf("nail should pierce doll")
	}

	n.Handle(ctx, unit.Collision{Time: 1.02, Other: doll})
	if cmds := drain(out); len(cmds) != 0 {
		t.Fatalf("same doll should not be hit twice: %v", cmds)
	}
}

func TestCurseSevenBurstsAndClears(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	n := &钉{owner: 1, slot: 0, spawnTime: 0}
	ctx := unit.Context{ID: 9, Kind: KindNail, Out: out}
	enemy := foe(40, 0)
	enemy.HP = 100
	enemy.Marks = []unit.Mark{{Kind: curseKind, Stacks: 6, Icon: curseIcon}}
	n.Handle(ctx, unit.Collision{Time: 1, Other: enemy})
	cmds := drain(out)
	if m := lastStack(cmds, 2); m == nil || m.Delta != 1 {
		t.Fatalf("stack=%v cmds=%v", m, cmds)
	}
	if d := damageOf(cmds, 2, curseBurst); d == nil {
		t.Fatalf("burst missing: %v", cmds)
	}
	if lastClear(cmds, 2) == nil {
		t.Fatalf("curse should clear: %v", cmds)
	}
}

func TestHammerSmashCursesAlreadyCursed(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	h := &钉与锤{hammerArmed: true, nailReadyAt: 999, ritualReadyAt: 999, dollsSpawned: true}
	ctx := unit.Context{ID: 1, Kind: KindHammer, Out: out}
	enemy := foe(30, 0)
	enemy.Marks = []unit.Mark{{Kind: curseKind, Stacks: 2, Icon: curseIcon}}
	h.Handle(ctx, unit.Sense{Time: 1, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	if d := damageTo(cmds, 2); d == nil || d.Amount != swingDamage {
		t.Fatalf("swing dmg=%v cmds=%v", d, cmds)
	}
	if m := lastStack(cmds, 2); m == nil || m.Kind != curseKind {
		t.Fatalf("hammer should curse marked target: %v", cmds)
	}
}

func TestFireNailSwingsAround(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	h := &钉与锤{nailReadyAt: 0, ritualReadyAt: 999, dollsSpawned: true}
	ctx := unit.Context{ID: 1, Kind: KindHammer, Out: out}
	near := unit.Snapshot{
		ID: 4, Kind: "活随从", Role: unit.RoleMinion, Slot: 1, Mortal: true,
		X: 20, Y: 0, Radius: 12, HP: 30,
	}
	far := foe(160, 0)
	h.Handle(ctx, unit.Sense{Time: 1, Self: me(0, 0), Nearby: []unit.Snapshot{near, far}})
	cmds := drain(out)
	if lastNamed(cmds, "hammer") == nil {
		t.Fatalf("nail fire should swing hammer: %v", cmds)
	}
	if d := damageTo(cmds, 4); d == nil || d.Amount != swingDamage {
		t.Fatalf("nearby swing dmg=%v cmds=%v", d, cmds)
	}
	if damageTo(cmds, 2) != nil {
		t.Fatalf("far enemy should not take swing: %v", cmds)
	}
	if countSpawn(cmds, KindNail) != 1 {
		t.Fatalf("want 1 nail, cmds=%v", cmds)
	}
}

func TestRitualNailsFollowThenShoot(t *testing.T) {
	resetNailJobs()
	t.Cleanup(resetNailJobs)
	out := make(chan unit.Cmd, 32)
	h := &钉与锤{started: true, nailReadyAt: 999, dollsSpawned: true}
	ctx := unit.Context{ID: 1, Kind: KindHammer, Out: out}
	enemy := foe(80, 0)
	h.Handle(ctx, unit.Sense{Time: 1, Self: me(0, 0), Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	if countSpawn(cmds, KindNail) != 3 {
		t.Fatalf("want 3 ritual nails, cmds=%v", cmds)
	}
	if math.Abs(h.ritualReadyAt-(1+ritualCD)) > 1e-9 {
		t.Fatalf("ritual CD=%v", h.ritualReadyAt)
	}

	n := &钉{owner: 1, slot: 0, spawnTime: -1}
	nctx := unit.Context{ID: 21, Kind: KindNail, Out: out}
	n.Handle(nctx, unit.Sense{
		Time:   1,
		Self:   unit.Snapshot{ID: 21, Kind: KindNail, Role: unit.RoleProjectile, Slot: 0, X: 80, Y: 56},
		Nearby: []unit.Snapshot{enemy},
	})
	boot := drain(out)
	if lastPass(boot) == nil || !lastPass(boot).Hold {
		t.Fatalf("orbit nail should pass, cmds=%v", boot)
	}
	tp := lastTeleport(boot)
	if tp == nil {
		t.Fatalf("orbit nail should sit beside enemy: %v", boot)
	}
	wantX, wantY := enemy.X, enemy.Y+ritualOrbit
	if math.Hypot(tp.X-wantX, tp.Y-wantY) > 1 {
		t.Fatalf("orbit pos=(%v,%v) want=(%v,%v)", tp.X, tp.Y, wantX, wantY)
	}

	moved := foe(40, 30)
	n.Handle(nctx, unit.Sense{
		Time:   2,
		Self:   unit.Snapshot{ID: 21, Kind: KindNail, Role: unit.RoleProjectile, Slot: 0, X: tp.X, Y: tp.Y},
		Nearby: []unit.Snapshot{moved},
	})
	follow := drain(out)
	ftp := lastTeleport(follow)
	if ftp == nil {
		t.Fatalf("should follow enemy: %v", follow)
	}
	fx, fy := moved.X, moved.Y+ritualOrbit
	if math.Hypot(ftp.X-fx, ftp.Y-fy) > 1 {
		t.Fatalf("follow pos=(%v,%v) want=(%v,%v)", ftp.X, ftp.Y, fx, fy)
	}

	n.Handle(nctx, unit.Sense{
		Time:   1 + ritualFollow,
		Self:   unit.Snapshot{ID: 21, Kind: KindNail, Role: unit.RoleProjectile, Slot: 0, X: ftp.X, Y: ftp.Y},
		Nearby: []unit.Snapshot{moved},
	})
	lock := drain(out)
	if n.phase != nailWind {
		t.Fatalf("should lock after follow, phase=%d cmds=%v", n.phase, lock)
	}

	n.Handle(nctx, unit.Sense{
		Time: 1 + ritualFollow + ritualWind,
		Self: unit.Snapshot{ID: 21, Kind: KindNail, Role: unit.RoleProjectile, Slot: 0, X: ftp.X, Y: ftp.Y},
	})
	shot := drain(out)
	v := lastVel(shot, 21)
	if v == nil || math.Hypot(v.VX, v.VY) < nailSpeed-1 {
		t.Fatalf("should shoot after windup, vel=%v cmds=%v", v, shot)
	}
	if n.lockX != moved.X || n.lockY != moved.Y {
		t.Fatalf("lock=(%v,%v) want enemy (%v,%v)", n.lockX, n.lockY, moved.X, moved.Y)
	}
	dx, dy := n.lockX-ftp.X, n.lockY-ftp.Y
	d := math.Hypot(dx, dy)
	wantVX, wantVY := nailSpeed*dx/d, nailSpeed*dy/d
	if math.Hypot(v.VX-wantVX, v.VY-wantVY) > 1 {
		t.Fatalf("shot dir vel=(%v,%v) want=(%v,%v)", v.VX, v.VY, wantVX, wantVY)
	}
	if lastPass(shot) == nil || lastPass(shot).Hold {
		t.Fatalf("shot nails should collide, cmds=%v", shot)
	}
}

func TestRitualStartsOnCooldown(t *testing.T) {
	resetNailJobs()
	t.Cleanup(resetNailJobs)
	out := make(chan unit.Cmd, 32)
	h := &钉与锤{nailReadyAt: 999, dollsSpawned: true}
	ctx := unit.Context{ID: 1, Kind: KindHammer, Out: out}
	h.Handle(ctx, unit.Sense{Time: 1, Self: me(0, 0), Nearby: []unit.Snapshot{foe(80, 0)}})
	cmds := drain(out)
	if countSpawn(cmds, KindNail) != 0 {
		t.Fatalf("ritual should start on CD, cmds=%v", cmds)
	}
	if math.Abs(h.ritualReadyAt-(1+ritualCD)) > 1e-9 {
		t.Fatalf("opening CD=%v want %v", h.ritualReadyAt, 1+ritualCD)
	}
}

func me(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindHammer, Role: unit.RoleFighter, Slot: 0,
		X: x, Y: y, Radius: fighterRadius, VX: fighterSpeed, VY: 0,
	}
}

func foe(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 2, Kind: KindHammer, Role: unit.RoleFighter, Slot: 1,
		X: x, Y: y, Radius: fighterRadius,
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

func lastStun(cmds []unit.Cmd) *unit.Stun {
	var st *unit.Stun
	for _, c := range cmds {
		if v, ok := c.(unit.Stun); ok {
			cp := v
			st = &cp
		}
	}
	return st
}

func lastVel(cmds []unit.Cmd, id uint64) *unit.SetVelocity {
	var v *unit.SetVelocity
	for _, c := range cmds {
		if x, ok := c.(unit.SetVelocity); ok && x.UnitID == id {
			cp := x
			v = &cp
		}
	}
	return v
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

func lastTeleport(cmds []unit.Cmd) *unit.Teleport {
	var tp *unit.Teleport
	for _, c := range cmds {
		if v, ok := c.(unit.Teleport); ok {
			cp := v
			tp = &cp
		}
	}
	return tp
}

func hasDespawn(cmds []unit.Cmd, id uint64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.Despawn); ok && v.UnitID == id {
			return true
		}
	}
	return false
}

func damageTo(cmds []unit.Cmd, id uint64) *unit.Damage {
	var d *unit.Damage
	for _, c := range cmds {
		if v, ok := c.(unit.Damage); ok && v.To == id {
			cp := v
			d = &cp
		}
	}
	return d
}

func damageOf(cmds []unit.Cmd, id uint64, amount float64) *unit.Damage {
	for _, c := range cmds {
		if v, ok := c.(unit.Damage); ok && v.To == id && math.Abs(v.Amount-amount) < 1e-9 {
			cp := v
			return &cp
		}
	}
	return nil
}

func lastStack(cmds []unit.Cmd, id uint64) *unit.StackMark {
	var m *unit.StackMark
	for _, c := range cmds {
		if v, ok := c.(unit.StackMark); ok && v.UnitID == id {
			cp := v
			m = &cp
		}
	}
	return m
}

func lastClear(cmds []unit.Cmd, id uint64) *unit.ClearMarks {
	var m *unit.ClearMarks
	for _, c := range cmds {
		if v, ok := c.(unit.ClearMarks); ok && v.UnitID == id {
			cp := v
			m = &cp
		}
	}
	return m
}

func countSpawn(cmds []unit.Cmd, kind string) int {
	n := 0
	for _, c := range cmds {
		if v, ok := c.(unit.Spawn); ok && v.Kind == kind {
			n++
		}
	}
	return n
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
