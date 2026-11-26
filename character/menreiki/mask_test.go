package 面灵气

import (
	"math"
	"testing"
	"xqdj/internal/unit"
)

func TestMaskHitRateLimitedPerTarget(t *testing.T) {
	sc := &unit.MatchState{}
	if !maskHitReady(sc, 1, 7, 5) {
		t.Fatal("第一下该放行")
	}
	for _, now := range []float64{5.1, 5.2, 5.39} {
		if maskHitReady(sc, 1, 7, now) {
			t.Fatalf("%v 还在 CD 里，不该打中同一目标", now)
		}
	}
	if !maskHitReady(sc, 1, 7, 5.4) {
		t.Fatalf("过了 %v 秒该再打一下", maskHitCD)
	}
	if !maskHitReady(sc, 1, 8, 5.45) {
		t.Fatal("另一个目标各算各的 CD")
	}
}

func TestMaskSnapsBackWhenOwnerLeaves(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	ctx := unit.Context{State: &unit.MatchState{}, ID: 9, Kind: KindMenreikiMask1, Out: out}
	a := &面具{owner: 1}
	self := unit.Snapshot{
		ID: 9, Kind: KindMenreikiMask1, Role: unit.RoleHelper, OwnerID: 1, Slot: 0,
		X: 40, Y: 0, Radius: maskRadius,
	}
	owner := unit.Snapshot{
		ID: 1, Kind: KindMenreiki, Role: unit.RoleFighter, Slot: 0,
		X: 120, Y: 0, Radius: menreikiRadius,
	}
	a.Handle(ctx, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{owner}})
	var tp *unit.Teleport
	for _, c := range drain(out) {
		if v, ok := c.(unit.Teleport); ok {
			tp = &v
		}
	}
	if tp == nil {
		t.Fatal("主人甩开 80，面具该自己追回来")
	}
	want := owner.Radius + maskRadius + maskOrbitPad
	if d := math.Hypot(tp.X-owner.X, tp.Y-owner.Y); math.Abs(d-want) > 1e-6 {
		t.Fatalf("追回来该落在轨道圈上：d=%v want=%v", d, want)
	}
	if tp.X >= owner.X {
		t.Fatalf("该留在原来那一侧被拉回轨道，tp=%v", tp)
	}

	// 已经在轨道上就什么都别发，免得和主人每帧的 Teleport 打架。
	self.X = owner.X + want
	a.Handle(ctx, unit.Sense{Time: 1.1, Self: self, Nearby: []unit.Snapshot{owner}})
	for _, c := range drain(out) {
		if _, ok := c.(unit.Teleport); ok {
			t.Fatalf("没掉队不该自发瞬移: %v", c)
		}
	}
}

func TestGrantMarksAllEnemyFighters(t *testing.T) {
	matchState := &unit.MatchState{}

	out := make(chan unit.Cmd, 32)
	m := &面灵气{hook: 1}
	ctx := unit.Context{State: matchState, ID: 1, Kind: KindMenreiki, Out: out}
	self := unit.Snapshot{
		ID: 1, Kind: KindMenreiki, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: menreikiRadius,
	}
	near := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 40, Y: 0, Radius: 18, AimPriority: unit.DefaultFighterAim}
	mid := unit.Snapshot{ID: 3, Role: unit.RoleFighter, Slot: 1, X: 120, Y: 0, Radius: 18, AimPriority: unit.DefaultFighterAim}
	far := unit.Snapshot{ID: 4, Role: unit.RoleFighter, Slot: 1, X: 200, Y: 0, Radius: 18, AimPriority: unit.DefaultFighterAim}
	m.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{near, mid, far}})
	got := markFactionIDs(drain(out))
	for _, id := range []uint64{1, 2, 3, 4} {
		if !got[id] {
			t.Fatalf("missing mark on %d: %v", id, got)
		}
	}
}

func TestCollectPaysEachEnemy(t *testing.T) {
	matchState := &unit.MatchState{}

	out := make(chan unit.Cmd, 64)
	m := &面灵气{hook: 2, marked: true}
	ctx := unit.Context{State: matchState, ID: 1, Kind: KindMenreiki, Out: out}
	self := unit.Snapshot{
		ID: 1, Kind: KindMenreiki, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: menreikiRadius,
	}
	seek := unit.Snapshot{
		ID: 2, Role: unit.RoleFighter, Slot: 1, X: 40, Y: 0, Radius: 18,
		Faction: unit.FactionRed, Seen: unit.AllFactions(), AimPriority: unit.DefaultFighterAim,
	}
	other := unit.Snapshot{
		ID: 3, Role: unit.RoleFighter, Slot: 1, X: 200, Y: 0, Radius: 18,
		Faction: unit.FactionPale, Seen: unit.AllFactions(), AimPriority: unit.DefaultFighterAim,
	}
	m.Handle(ctx, unit.Sense{Time: 10, Self: self, Nearby: []unit.Snapshot{seek, other}})
	cmds := drain(out)
	if !hasDamageTo(cmds, 2) {
		t.Fatalf("seek target red collect should strike: %v", cmds)
	}
	if lastStackMark(cmds, 3) == nil {
		t.Fatalf("other pale collect should break: %v", cmds)
	}
}

func TestCollectPaysCompletingColorNotCurrent(t *testing.T) {
	matchState := &unit.MatchState{}

	out := make(chan unit.Cmd, 64)
	m := &面灵气{hook: 2, marked: true}
	ctx := unit.Context{State: matchState, ID: 1, Kind: KindMenreiki, Out: out}
	self := unit.Snapshot{
		ID: 1, Kind: KindMenreiki, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: menreikiRadius,
	}
	three := []string{unit.FactionCyan, unit.FactionRed, unit.FactionPurple}
	enemy := unit.Snapshot{
		ID: 2, Role: unit.RoleFighter, Slot: 1, X: 80, Y: 0, Radius: 18,
		Faction: unit.FactionPurple, Seen: three, AimPriority: unit.DefaultFighterAim,
	}
	m.Handle(ctx, unit.Sense{Time: 10, Self: self, Nearby: []unit.Snapshot{enemy}})
	if cmds := drain(out); hasDamageTo(cmds, 2) || lastStackMark(cmds, 2) != nil || lastStun(cmds) != nil {
		t.Fatalf("three colors should wait: %v", cmds)
	}
	enemy.Seen = unit.AllFactions()
	enemy.Faction = unit.FactionRed
	m.Handle(ctx, unit.Sense{Time: 10.2, Self: self, Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	if lastStackMark(cmds, 2) == nil {
		t.Fatalf("completing 苍 should break, not follow current 红: %v", cmds)
	}
	if hasDamageTo(cmds, 2) {
		t.Fatalf("current 红 should not strike: %v", cmds)
	}
}

func TestCollectDoesNotRepayWhileComplete(t *testing.T) {
	matchState := &unit.MatchState{}

	out := make(chan unit.Cmd, 64)
	m := &面灵气{hook: 2, marked: true}
	ctx := unit.Context{State: matchState, ID: 1, Kind: KindMenreiki, Out: out}
	self := unit.Snapshot{
		ID: 1, Kind: KindMenreiki, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: menreikiRadius,
	}
	enemy := unit.Snapshot{
		ID: 2, Role: unit.RoleFighter, Slot: 1, X: 80, Y: 0, Radius: 18,
		Faction: unit.FactionCyan, Seen: unit.AllFactions(), AimPriority: unit.DefaultFighterAim,
	}
	m.Handle(ctx, unit.Sense{Time: 10, Self: self, Nearby: []unit.Snapshot{enemy}})
	if lastStun(drain(out)) == nil {
		t.Fatal("first complete should stun")
	}
	enemy.Faction = unit.FactionRed
	m.Handle(ctx, unit.Sense{Time: 10.2, Self: self, Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	if lastStun(cmds) != nil && lastStun(cmds).Hold && lastStun(cmds).Until > 10.2 {
		t.Fatalf("still-complete should not settle again: %v", cmds)
	}
	if hasDamageTo(cmds, 2) {
		t.Fatalf("later 红 should not strike: %v", cmds)
	}
}

func TestShareStaysOnCollectorNotSeek(t *testing.T) {
	matchState := &unit.MatchState{}

	out := make(chan unit.Cmd, 64)
	m := &面灵气{hook: 2, marked: true}
	ctx := unit.Context{State: matchState, ID: 1, Kind: KindMenreiki, Out: out}
	self := unit.Snapshot{
		ID: 1, Kind: KindMenreiki, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: menreikiRadius,
	}
	seek := unit.Snapshot{
		ID: 2, Role: unit.RoleFighter, Slot: 1, X: 40, Y: 0, Radius: 18,
		Faction: unit.FactionCyan, AimPriority: unit.DefaultFighterAim,
	}
	collector := unit.Snapshot{
		ID: 3, Role: unit.RoleFighter, Slot: 1, X: 200, Y: 0, Radius: 18,
		Faction: unit.FactionPurple, Seen: unit.AllFactions(), AimPriority: unit.DefaultFighterAim,
	}
	m.Handle(ctx, unit.Sense{Time: 10, Self: self, Nearby: []unit.Snapshot{seek, collector}})
	_ = drain(out)
	m.Handle(ctx, unit.IncomingDamage{Token: 9, Amount: 10, Time: 10.1, From: 2})
	cmds := drain(out)
	if hasDamageTo(cmds, 2) {
		t.Fatalf("share must not follow Seek: %v", cmds)
	}
	if !hasDamageTo(cmds, 3) {
		t.Fatalf("share should hit collector: %v", cmds)
	}
}

func TestMaskOrbitHitsMortalMinion(t *testing.T) {
	matchState := &unit.MatchState{}

	out := make(chan unit.Cmd, 32)
	m := &面灵气{hook: 1}
	ctx := unit.Context{State: matchState, ID: 1, Kind: KindMenreiki, Out: out}
	orbit := menreikiRadius + maskRadius + maskOrbitPad
	self := unit.Snapshot{
		ID: 1, Kind: KindMenreiki, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: menreikiRadius, Faction: unit.FactionCyan,
	}
	minion := unit.Snapshot{
		ID: 8, Kind: "教父暗杀者", Role: unit.RoleMinion, Mortal: true, Slot: 1,
		X: orbit, Y: 0, Radius: 14, HP: 30, MaxHP: 30,
	}
	m.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{minion}})
	if !hasDamageTo(drain(out), 8) {
		t.Fatal("mask overlap should damage mortal minion")
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

func hasDamageTo(cmds []unit.Cmd, to uint64) bool {
	for _, c := range cmds {
		if d, ok := c.(unit.Damage); ok && d.To == to {
			return true
		}
	}
	return false
}

func TestMaskHitGateIndependent(t *testing.T) {
	matchState := &unit.MatchState{}

	const owner uint64 = 7
	if spendMask(matchState, owner, 0) {
		t.Fatal("unarmed should not hit")
	}
	armMask(matchState, owner, 0)
	if !spendMask(matchState, owner, 0) {
		t.Fatal("armed mask 0 should hit")
	}
	if spendMask(matchState, owner, 0) {
		t.Fatal("spent mask 0 should not hit again")
	}
	if spendMask(matchState, owner, 1) {
		t.Fatal("mask 1 should stay independent")
	}
	armAllMasks(matchState, owner)
	for i := 0; i < 3; i++ {
		if !spendMask(matchState, owner, i) {
			t.Fatalf("arm all: mask %d", i)
		}
	}
}

func TestCyanCollectPinsWithoutWritingCruise(t *testing.T) {
	matchState := &unit.MatchState{}

	out := make(chan unit.Cmd, 64)
	m := &面灵气{hook: 2}
	ctx := unit.Context{State: matchState, ID: 1, Kind: KindMenreiki, Out: out}
	self := unit.Snapshot{
		ID: 1, Kind: KindMenreiki, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: menreikiRadius,
	}
	enemy := unit.Snapshot{
		ID: 2, Kind: "筑墙者", Role: unit.RoleFighter, Slot: 1,
		X: 80, Y: 0, Radius: 18, Faction: unit.FactionCyan,
		Seen: unit.AllFactions(), AimPriority: unit.DefaultFighterAim,
	}
	m.Handle(ctx, unit.Sense{Time: 10, Self: self, Nearby: []unit.Snapshot{enemy}})
	hit := drain(out)
	until := 10 + stunSecs
	st := lastStun(hit)
	if st == nil || st.UnitID != 2 || !st.Hold || math.Abs(st.Until-until) > 1e-9 {
		t.Fatalf("stun=%v cmds=%v", st, hit)
	}
	pin := lastAddFSComponent(hit, 2)
	if pin == nil || pin.Zone != unit.FSZoneM || pin.Value != 0 || math.Abs(pin.ExpiresAt-until) > 1e-9 {
		t.Fatalf("pin=%v cmds=%v", pin, hit)
	}
	if lastCruise(hit, 2) != nil || lastVel(hit, 2) != nil {
		t.Fatalf("must not write target cruise/vel: %v", hit)
	}
	if lastNamed(hit, "stun") == nil {
		t.Fatalf("missing stun fx: %v", hit)
	}

	m.Handle(ctx, unit.Sense{Time: 10.2, Self: self, Nearby: []unit.Snapshot{enemy}})
	mid := drain(out)
	if lastStun(mid) != nil || lastCruise(mid, 2) != nil || lastVel(mid, 2) != nil {
		t.Fatalf("must not babysit pin: %v", mid)
	}
	if lastNamed(mid, "stun") == nil {
		t.Fatalf("stun overlay should keep ticking: %v", mid)
	}

	m.Handle(ctx, unit.Sense{Time: until, Self: self, Nearby: []unit.Snapshot{enemy}})
	end := drain(out)
	if lastStun(end) != nil || lastCruise(end, 2) != nil || lastVel(end, 2) != nil {
		t.Fatalf("engine owns release, cmds=%v", end)
	}
}

func TestCyanLostTargetClearsPin(t *testing.T) {
	matchState := &unit.MatchState{}

	out := make(chan unit.Cmd, 64)
	m := &面灵气{hook: 2}
	ctx := unit.Context{State: matchState, ID: 1, Kind: KindMenreiki, Out: out}
	self := unit.Snapshot{
		ID: 1, Kind: KindMenreiki, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: menreikiRadius,
	}
	enemy := unit.Snapshot{
		ID: 2, Kind: "筑墙者", Role: unit.RoleFighter, Slot: 1,
		X: 80, Y: 0, Radius: 18, Faction: unit.FactionCyan,
		Seen: unit.AllFactions(), AimPriority: unit.DefaultFighterAim,
	}
	m.Handle(ctx, unit.Sense{Time: 10, Self: self, Nearby: []unit.Snapshot{enemy}})
	_ = drain(out)
	m.Handle(ctx, unit.Sense{Time: 10.2, Self: self})
	end := drain(out)
	if lastRemoveFSComponent(end, 2) == nil {
		t.Fatalf("lost target should lift pin: %v", end)
	}
	st := lastStun(end)
	if st == nil || st.Hold {
		t.Fatalf("lost target should drop stun: %v", end)
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

func lastCruise(cmds []unit.Cmd, id uint64) *unit.SetCruise {
	var v *unit.SetCruise
	for _, c := range cmds {
		if x, ok := c.(unit.SetCruise); ok && x.UnitID == id {
			cp := x
			v = &cp
		}
	}
	return v
}

func lastAddFSComponent(cmds []unit.Cmd, id uint64) *unit.AddFSComponent {
	var v *unit.AddFSComponent
	for _, c := range cmds {
		if x, ok := c.(unit.AddFSComponent); ok && x.UnitID == id {
			cp := x
			v = &cp
		}
	}
	return v
}

func lastRemoveFSComponent(cmds []unit.Cmd, id uint64) *unit.RemoveFSComponent {
	var v *unit.RemoveFSComponent
	for _, c := range cmds {
		if x, ok := c.(unit.RemoveFSComponent); ok && x.UnitID == id {
			cp := x
			v = &cp
		}
	}
	return v
}

func markFactionIDs(cmds []unit.Cmd) map[uint64]bool {
	got := map[uint64]bool{}
	for _, c := range cmds {
		if v, ok := c.(unit.MarkFaction); ok {
			got[v.UnitID] = true
		}
	}
	return got
}

func lastStackMark(cmds []unit.Cmd, id uint64) *unit.StackMark {
	var mk *unit.StackMark
	for _, c := range cmds {
		if v, ok := c.(unit.StackMark); ok && v.UnitID == id {
			cp := v
			mk = &cp
		}
	}
	return mk
}

func TestPaleStopRestoresCruise(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	m := &面灵气{}
	ctx := unit.Context{State: &unit.MatchState{}, ID: 1, Kind: KindMenreiki, Out: out}
	self := unit.Snapshot{
		ID: 1, Kind: KindMenreiki, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: menreikiRadius, Faction: unit.FactionPale,
		VX: paleCruise, VY: 0,
	}
	enemy := unit.Snapshot{
		ID: 2, Role: unit.RoleFighter, Slot: 1, X: 0, Y: 200, Radius: 18,
		AimPriority: unit.DefaultFighterAim,
	}
	m.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{enemy}})
	_ = drain(out)

	m.Handle(ctx, unit.WallHit{Time: 0.05, NX: -1, NY: 0})
	self.VX, self.VY = 0, 0
	m.Handle(ctx, unit.Sense{Time: 0.2, Self: self, Nearby: []unit.Snapshot{enemy}})
	v := lastVel(drain(out), 1)
	if v == nil {
		t.Fatal("stopped pale should steer again")
	}
	sp := math.Hypot(v.VX, v.VY)
	if math.Abs(sp-paleCruise) > 1e-6 || v.VY <= 0 || math.Abs(v.VX) > 1e-6 {
		t.Fatalf("zero speed must resume cruise toward the enemy, vel=%v", v)
	}
}

func TestPaleSlowedSteerKeepsCruise(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	m := &面灵气{}
	ctx := unit.Context{State: &unit.MatchState{}, ID: 1, Kind: KindMenreiki, Out: out}
	self := unit.Snapshot{
		ID: 1, Kind: KindMenreiki, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: menreikiRadius, Faction: unit.FactionPale,
		VX: paleCruise, VY: 0,
	}
	enemy := unit.Snapshot{
		ID: 2, Role: unit.RoleFighter, Slot: 1, X: 200, Y: 0, Radius: 18,
		AimPriority: unit.DefaultFighterAim,
	}
	m.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{enemy}})
	_ = drain(out)

	m.Handle(ctx, unit.Collision{Time: 0.05, Other: enemy})
	self.VX, self.VY = 30, 0
	m.Handle(ctx, unit.Sense{Time: 0.2, Self: self, Nearby: []unit.Snapshot{enemy}})
	v := lastVel(drain(out), 1)
	if v == nil {
		t.Fatal("slowed pale should steer again")
	}
	sp := math.Hypot(v.VX, v.VY)
	if math.Abs(sp-paleCruise) > 1e-6 || v.VX <= 0 {
		t.Fatalf("steer must restore pale cruise, not the slowed speed, vel=%v", v)
	}
}

func TestPurpleMaskSkipsAllyShot(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	m := &面灵气{}
	ctx := unit.Context{State: &unit.MatchState{}, ID: 1, Kind: KindMenreiki, Out: out}
	self := unit.Snapshot{
		ID: 1, Kind: KindMenreiki, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: menreikiRadius, Faction: unit.FactionPurple,
	}
	orbit := menreikiRadius + maskRadius + maskOrbitPad
	ally := unit.Snapshot{
		ID: 8, Role: unit.RoleProjectile, Slot: 0, OwnerID: 4,
		X: orbit, Y: 0, Radius: 6, Kind: "火箭",
	}
	foe := unit.Snapshot{
		ID: 9, Role: unit.RoleProjectile, Slot: 1, OwnerID: 2,
		X: orbit, Y: 0, Radius: 6, Kind: "火箭",
	}
	own := unit.Snapshot{
		ID: 10, Role: unit.RoleProjectile, Slot: 0, OwnerID: 1,
		X: orbit, Y: 0, Radius: 6, Kind: "苦无",
	}
	m.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{ally, foe, own}})
	gone := despawnIDs(drain(out))
	if !gone[9] {
		t.Fatal("purple mask should clear the enemy shot")
	}
	if gone[8] || gone[10] {
		t.Fatalf("ally and own shots must stay, despawned=%v", gone)
	}
}

func TestMaskOnCooldownKeepsCharge(t *testing.T) {
	st := &unit.MatchState{}
	out := make(chan unit.Cmd, 32)
	m := &面灵气{}
	ctx := unit.Context{State: st, ID: 1, Kind: KindMenreiki, Out: out}
	orbit := menreikiRadius + maskRadius + maskOrbitPad
	self := unit.Snapshot{
		ID: 1, Kind: KindMenreiki, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: menreikiRadius, Faction: unit.FactionCyan,
	}
	enemy := unit.Snapshot{
		ID: 2, Role: unit.RoleFighter, Slot: 1, X: orbit, Y: 0, Radius: 18,
	}
	if !maskHitReady(st, 1, 2, 0) {
		t.Fatal("seed cd")
	}
	m.Handle(ctx, unit.Sense{Time: 0.1, Self: self, Nearby: []unit.Snapshot{enemy}})
	if hasDamageTo(drain(out), 2) {
		t.Fatal("cooldown should block the hit")
	}
	if !maskCharged(st, 1, 0) {
		t.Fatal("cooldown ate mask 0 charge")
	}
}

func TestMaskCollisionOnCooldownKeepsCharge(t *testing.T) {
	st := &unit.MatchState{}
	out := make(chan unit.Cmd, 8)
	armMask(st, 1, 0)
	if !maskHitReady(st, 1, 2, 1) {
		t.Fatal("seed cd")
	}
	a := &面具{owner: 1, slot: 0, index: 0}
	ctx := unit.Context{State: st, ID: 9, Kind: KindMenreikiMask1, Out: out}
	enemy := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, Radius: 18}
	a.Handle(ctx, unit.Collision{Time: 1.1, Other: enemy})
	if hasDamageTo(drain(out), 2) {
		t.Fatal("cooldown collision should not hit")
	}
	if !maskCharged(st, 1, 0) {
		t.Fatal("cooldown collision ate the charge")
	}

	a.Handle(ctx, unit.Collision{Time: 1 + maskHitCD, Other: enemy})
	if !hasDamageTo(drain(out), 2) {
		t.Fatal("after cooldown the saved charge should hit")
	}
	if spendMask(st, 1, 0) {
		t.Fatal("the hit should have spent the charge")
	}
}

func maskCharged(scope *unit.MatchState, owner uint64, i int) bool {
	if i < 0 || i > 2 {
		return false
	}
	shared := stateFor(scope)
	shared.liveMu.Lock()
	defer shared.liveMu.Unlock()
	return shared.maskReady[owner][i]
}

func despawnIDs(cmds []unit.Cmd) map[uint64]bool {
	got := map[uint64]bool{}
	for _, c := range cmds {
		if d, ok := c.(unit.Despawn); ok {
			got[d.UnitID] = true
		}
	}
	return got
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
