package 教父

import (
	"math"
	"testing"
	"xqdj/internal/unit"
)

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

func selfAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindGodfather, Role: unit.RoleFighter, Slot: 0,
		X: x, Y: y, Radius: godfatherRadius, HP: godfatherHP, MaxHP: godfatherHP,
	}
}

func enemyAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: x, Y: y, Radius: 18, AimPriority: unit.DefaultFighterAim}
}

func TestOpensFirstMinionAtOneSecond(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := &教父{}
	ctx := unit.Context{ID: 1, Kind: KindGodfather, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0.5, Self: selfAt(0, 0)})
	if cmds := drain(out); hasSpawn(cmds) {
		t.Fatalf("too early: %v", cmds)
	}
	a.Handle(ctx, unit.Sense{Time: 1, Self: selfAt(0, 0)})
	sp := lastSpawn(drain(out))
	if sp == nil || !isMinionKind(sp.Kind) || sp.OwnerID != 1 {
		t.Fatalf("spawn=%v", sp)
	}
}

func TestOpensSecondMinionAtTwoSecondsDifferentKind(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := &教父{}
	ctx := unit.Context{ID: 1, Kind: KindGodfather, Out: out}
	a.Handle(ctx, unit.Sense{Time: 1, Self: selfAt(0, 0)})
	first := lastSpawn(drain(out))
	if first == nil {
		t.Fatal("missing first")
	}
	a.Handle(ctx, unit.Sense{Time: 2, Self: selfAt(0, 0), Nearby: []unit.Snapshot{{
		ID: 9, Kind: first.Kind, OwnerID: 1, Role: unit.RoleMinion, Slot: 0,
	}}})
	second := lastSpawn(drain(out))
	if second == nil || second.Kind == first.Kind {
		t.Fatalf("first=%v second=%v", first, second)
	}
}

func TestOpensThirdMinionAtThreeSeconds(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := &教父{}
	ctx := unit.Context{ID: 1, Kind: KindGodfather, Out: out}
	a.Handle(ctx, unit.Sense{Time: 1, Self: selfAt(0, 0)})
	first := lastSpawn(drain(out))
	a.Handle(ctx, unit.Sense{Time: 2, Self: selfAt(0, 0), Nearby: []unit.Snapshot{{
		ID: 9, Kind: first.Kind, OwnerID: 1, Role: unit.RoleMinion, Slot: 0,
	}}})
	second := lastSpawn(drain(out))
	a.Handle(ctx, unit.Sense{Time: 2.5, Self: selfAt(0, 0), Nearby: []unit.Snapshot{
		{ID: 9, Kind: first.Kind, OwnerID: 1, Role: unit.RoleMinion, Slot: 0},
		{ID: 10, Kind: second.Kind, OwnerID: 1, Role: unit.RoleMinion, Slot: 0},
	}})
	if lastSpawn(drain(out)) != nil {
		t.Fatal("third should wait until 3s")
	}
	a.Handle(ctx, unit.Sense{Time: 3, Self: selfAt(0, 0), Nearby: []unit.Snapshot{
		{ID: 9, Kind: first.Kind, OwnerID: 1, Role: unit.RoleMinion, Slot: 0},
		{ID: 10, Kind: second.Kind, OwnerID: 1, Role: unit.RoleMinion, Slot: 0},
	}})
	third := lastSpawn(drain(out))
	if third == nil || third.Kind == first.Kind || third.Kind == second.Kind {
		t.Fatalf("first=%v second=%v third=%v", first, second, third)
	}
}

func TestRefillWaitsTwoPointFiveAfterLeave(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := &教父{}
	ctx := unit.Context{ID: 1, Kind: KindGodfather, Out: out}
	full := []unit.Snapshot{
		{ID: 9, Kind: KindAssassin, OwnerID: 1, Role: unit.RoleMinion, Slot: 0},
		{ID: 10, Kind: KindSniper, OwnerID: 1, Role: unit.RoleMinion, Slot: 0},
		{ID: 11, Kind: KindDealer, OwnerID: 1, Role: unit.RoleMinion, Slot: 0},
	}
	a.Handle(ctx, unit.Sense{Time: 3, Self: selfAt(0, 0), Nearby: full})
	_ = drain(out)
	left := full[:2]
	a.Handle(ctx, unit.Sense{Time: 4, Self: selfAt(0, 0), Nearby: left})
	if lastSpawn(drain(out)) != nil {
		t.Fatal("refill should wait")
	}
	a.Handle(ctx, unit.Sense{Time: 6.4, Self: selfAt(0, 0), Nearby: left})
	if lastSpawn(drain(out)) != nil {
		t.Fatal("too early for 2.5s refill")
	}
	a.Handle(ctx, unit.Sense{Time: 6.5, Self: selfAt(0, 0), Nearby: left})
	sp := lastSpawn(drain(out))
	if sp == nil || sp.Kind != KindDealer {
		t.Fatalf("refill=%v", sp)
	}
}

func TestAssassinWaitsOneSecondThenDashes(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &暗杀者{owner: 1, slot: 0, attacks: 2}
	ctx := unit.Context{ID: 3, Kind: KindAssassin, Out: out}
	self := unit.Snapshot{ID: 3, Role: unit.RoleMinion, Slot: 0, X: 0, Y: 0, Radius: minionRadius, Mortal: true}
	en := enemyAt(80, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{en}})
	if hasPass(drain(out)) {
		t.Fatal("should wait cd")
	}
	a.Handle(ctx, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{en}})
	cmds := drain(out)
	if !hasPass(cmds) {
		t.Fatalf("want pass: %v", cmds)
	}
	v := lastVel(cmds)
	if v == nil || math.Abs(v.VX-assassinSpeed) > 1e-6 {
		t.Fatalf("vel=%v", v)
	}
	if a.attacks != 1 {
		t.Fatalf("attacks=%d", a.attacks)
	}
}

func TestAssassinWallEndsDashAtCruise(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &暗杀者{
		owner: 1, slot: 0, attacks: 1,
		dashing: true, dashLeft: 100, aimX: 1, aimY: 0,
		hit: map[uint64]bool{}, booted: true,
	}
	ctx := unit.Context{ID: 3, Kind: KindAssassin, Out: out}
	a.Handle(ctx, unit.WallHit{Time: 1.5, NX: 1, NY: 0})
	v := lastVel(drain(out))
	if v == nil {
		t.Fatal("wall should put assassin back on cruise")
	}
	sp := math.Hypot(v.VX, v.VY)
	if math.Abs(sp-minionCruise) > 1e-6 {
		t.Fatalf("speed=%v want cruise %v vel=%+v", sp, minionCruise, v)
	}
	if v.VX >= 0 {
		t.Fatalf("want bounce away from wall, vel=%+v", v)
	}
}

func TestAssassinDashHitsOnce(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := &暗杀者{owner: 1, slot: 0, attacks: 2, dashing: true, dashLeft: 100, hit: map[uint64]bool{}}
	ctx := unit.Context{ID: 3, Kind: KindAssassin, Out: out}
	hit := unit.Collision{Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1}}
	a.Handle(ctx, hit)
	a.Handle(ctx, hit)
	n := 0
	for _, c := range drain(out) {
		if _, ok := c.(unit.Damage); ok {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("hits=%d", n)
	}
}

func TestSniperFiresAfterSevenSeconds(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &狙击者{owner: 1, slot: 0, shots: 1}
	ctx := unit.Context{ID: 4, Kind: KindSniper, Out: out}
	self := unit.Snapshot{ID: 4, Role: unit.RoleMinion, Slot: 0, X: 0, Y: 0, Radius: minionRadius, Mortal: true}
	en := enemyAt(100, 0)
	a.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{en}})
	if lastSpawn(drain(out)) != nil {
		t.Fatal("fired too early")
	}
	a.Handle(ctx, unit.Sense{Time: 7, Self: self, Nearby: []unit.Snapshot{en}})
	cmds := drain(out)
	sp := lastSpawn(cmds)
	if sp == nil || sp.Kind != KindGodfatherShot {
		t.Fatalf("shot=%v", sp)
	}
	if math.Abs(sp.VX-shotSpeed) > 1 {
		t.Fatalf("shot vx=%v", sp.VX)
	}
	if lastDespawn(cmds) != nil {
		t.Fatal("sniper should pause then dash out, not vanish on the shot")
	}
	if v := lastVel(cmds); v == nil || math.Hypot(v.VX, v.VY) > 1e-6 {
		t.Fatalf("want pause: %v", v)
	}
}

func TestDealerDropsAtFiveThenLeavesAfterTwo(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := &贩卖者{owner: 1, slot: 0, left: 2}
	ctx := unit.Context{ID: 5, Kind: KindDealer, Out: out}
	self := unit.Snapshot{ID: 5, Role: unit.RoleMinion, Slot: 0, X: 10, Y: 4, Radius: minionRadius, Mortal: true}
	a.Handle(ctx, unit.Sense{Time: 0, Self: self})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{Time: 5, Self: self})
	sp := lastSpawn(drain(out))
	if sp == nil || sp.Kind != KindDrug || sp.OwnerID != 1 {
		t.Fatalf("drug=%v", sp)
	}
	a.Handle(ctx, unit.Sense{Time: 10, Self: self})
	cmds := drain(out)
	if lastSpawn(cmds) == nil {
		t.Fatalf("second drop: %v", cmds)
	}
	if lastDespawn(cmds) != nil {
		t.Fatalf("should pause then dash out: %v", cmds)
	}
	if v := lastVel(cmds); v == nil || math.Hypot(v.VX, v.VY) > 1e-6 {
		t.Fatalf("want pause: %v", v)
	}
}

func TestShotHitsEachTargetOnce(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	b := &狙击弹{slot: 0, hit: map[uint64]bool{}}
	ctx := unit.Context{ID: 8, Kind: KindGodfatherShot, Out: out}
	c := unit.Collision{Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, Mortal: false}}
	b.Handle(ctx, c)
	b.Handle(ctx, c)
	n := 0
	for _, cmd := range drain(out) {
		if _, ok := cmd.(unit.Damage); ok {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("hits=%d", n)
	}
}

func TestPickKindSkipsLive(t *testing.T) {
	k := pickKind(map[string]bool{KindAssassin: true, KindSniper: true}, nil)
	if k != KindDealer {
		t.Fatalf("got %q", k)
	}
}

func hasSpawn(cmds []unit.Cmd) bool { return lastSpawn(cmds) != nil }

func lastSpawn(cmds []unit.Cmd) *unit.Spawn {
	var s *unit.Spawn
	for i := range cmds {
		if v, ok := cmds[i].(unit.Spawn); ok {
			cp := v
			s = &cp
		}
	}
	return s
}

func lastDespawn(cmds []unit.Cmd) *unit.Despawn {
	var s *unit.Despawn
	for i := range cmds {
		if v, ok := cmds[i].(unit.Despawn); ok {
			cp := v
			s = &cp
		}
	}
	return s
}

func lastVel(cmds []unit.Cmd) *unit.SetVelocity {
	var s *unit.SetVelocity
	for i := range cmds {
		if v, ok := cmds[i].(unit.SetVelocity); ok {
			cp := v
			s = &cp
		}
	}
	return s
}

func TestLeavePauseThenDashToNearestEdge(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &暗杀者{owner: 1, slot: 0, attacks: 0}
	ctx := unit.Context{ID: 3, Kind: KindAssassin, Out: out}
	self := unit.Snapshot{ID: 3, Role: unit.RoleMinion, Slot: 0, X: 80, Y: 0, Radius: minionRadius, Mortal: true}
	a.Handle(ctx, unit.Sense{Time: 0, Self: self})
	if v := lastVel(drain(out)); v == nil || math.Hypot(v.VX, v.VY) > 1e-6 {
		t.Fatalf("pause vel=%v", v)
	}
	a.Handle(ctx, unit.Sense{Time: leavePause, Self: self})
	v := lastVel(drain(out))
	dx, dy := nearestEdgeDir(80, 0)
	if v == nil || math.Abs(v.VX-dx*leaveSpeed) > 1e-6 || math.Abs(v.VY-dy*leaveSpeed) > 1e-6 {
		t.Fatalf("dash vel=%v want %v,%v", v, dx*leaveSpeed, dy*leaveSpeed)
	}
}

func TestLeaveFadesOnFieldEdge(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &暗杀者{owner: 1, slot: 0, attacks: 0}
	ctx := unit.Context{ID: 3, Kind: KindAssassin, Out: out}
	self := unit.Snapshot{ID: 3, Role: unit.RoleMinion, Slot: 0, X: 80, Y: 0, Radius: minionRadius, Mortal: true}
	a.Handle(ctx, unit.Sense{Time: 0, Self: self})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{Time: leavePause, Self: self})
	_ = drain(out)
	dx, dy := nearestEdgeDir(80, 0)
	a.Handle(ctx, unit.WallHit{Time: leavePause + 0.01, NX: dx, NY: dy})
	cmds := drain(out)
	if lastFX(cmds, "leave") == nil {
		t.Fatalf("want leave fx: %v", cmds)
	}
	if lastDespawn(cmds) != nil {
		t.Fatal("fade first")
	}
	a.Handle(ctx, unit.Sense{Time: leavePause + 0.01 + leaveFade, Self: self})
	if lastDespawn(drain(out)) == nil {
		t.Fatal("want despawn after fade")
	}
}

func TestDealerScreamWhenOverlappingDrug(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := &贩卖者{owner: 1, slot: 0, left: 2}
	ctx := unit.Context{ID: 5, Kind: KindDealer, Out: out}
	self := unit.Snapshot{ID: 5, Role: unit.RoleMinion, Slot: 0, X: 0, Y: 0, Radius: minionRadius, Mortal: true}
	drug := unit.Snapshot{ID: 9, Kind: KindDrug, Role: unit.RoleHelper, Slot: 0, OwnerID: 1, X: 0, Y: 0, Radius: drugRadius}
	a.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{drug}})
	fx := lastFX(drain(out), "scream")
	if fx == nil || fx.Kind != KindGodfather {
		t.Fatalf("scream=%v", fx)
	}
}

func TestDealerVisionCoversDrugOverlap(t *testing.T) {
	s, ok := unit.Lookup(KindDealer)
	if !ok {
		t.Fatal("missing dealer")
	}
	if s.Vision < minionRadius+drugRadius {
		t.Fatalf("vision=%v want >= %v", s.Vision, minionRadius+drugRadius)
	}
}

func hasPass(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.Pass); ok && v.Hold {
			return true
		}
	}
	return false
}

func lastFX(cmds []unit.Cmd, name string) *unit.FX {
	var s *unit.FX
	for i := range cmds {
		if v, ok := cmds[i].(unit.FX); ok && (name == "" || v.Name == name) {
			cp := v
			s = &cp
		}
	}
	return s
}
