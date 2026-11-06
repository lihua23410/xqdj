package 泵之心

import (
	"math"
	"testing"
	"xqdj/character/engine_heart/parts/api"
	"xqdj/internal/unit"
)

func TestRoleGuardHasAGapAndSweepsFastProjectiles(t *testing.T) {
	g := api.Guard{Inner: 25, Outer: 31, Span: math.Pi * .6}
	if !crossesGuard(g, 100, 0, 0, 0, 4) {
		t.Fatal("fast projectile tunneled through plate")
	}
	if crossesGuard(g, -100, 0, 0, 0, 4) {
		t.Fatal("plate protected its empty back")
	}
	fan := unit.Snapshot{X: 50, Radius: 30, ArcInner: 18, ArcSpan: math.Pi / 2, VX: -100}
	if !fanTouches(g, fan, unit.Snapshot{}) {
		t.Fatal("incoming fan missed plate")
	}
	fan.VX = 100
	if fanTouches(g, fan, unit.Snapshot{}) {
		t.Fatal("empty back of fan became a complete circle")
	}
}

func TestPackageGuardDoesNotTurnAreaDamageIntoImmunity(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Out: out}
	w := &world{self: unit.Snapshot{ID: 1}, parts: []part{{Type: partPlate, Equipped: true}}, seen: map[uint64]seenAttack{}}
	w.seen[9] = seenAttack{snapshot: unit.Snapshot{ID: 9, Kind: KindEngineHeart, Role: unit.RoleFighter, X: 60}}
	if w.blockOffer(ctx, unit.IncomingDamage{From: 9, Amount: 20}) {
		t.Fatal("direct area damage was blocked")
	}
	w.seen[10] = seenAttack{snapshot: unit.Snapshot{ID: 10, Kind: KindHeartSlag, Role: unit.RoleProjectile, X: 60, Radius: 5}}
	if !w.blockOffer(ctx, unit.IncomingDamage{From: 10, Amount: 5}) {
		t.Fatal("known front projectile was not blocked")
	}
	w.seen[11] = seenAttack{snapshot: unit.Snapshot{ID: 11, Kind: KindHeartSlag, Role: unit.RoleProjectile, X: -60, Radius: 5}}
	if w.blockOffer(ctx, unit.IncomingDamage{From: 11, Amount: 5}) {
		t.Fatal("known rear projectile was blocked")
	}
	w.seen[12] = seenAttack{snapshot: unit.Snapshot{ID: 12, Kind: KindHeartSlag, Role: unit.RoleProjectile, X: -45, Radius: 100, ArcSpan: math.Pi / 2}}
	if w.blockOffer(ctx, unit.IncomingDamage{From: 12, Amount: 5, ArcSpan: math.Pi / 2}) {
		t.Fatal("fan reach incorrectly inflated the defensive arc")
	}
}

func TestNetHPLedgerKeepsDropProgressThroughHealing(t *testing.T) {
	state := &unit.MatchState{}
	out := make(chan unit.Cmd, 128)
	ctx := unit.Context{State: state, ID: 2, Out: out}
	w := worldFor(state, 1)
	w.parts = []part{{Type: partPiston, Equipped: true}, {Type: partPlate, Equipped: true}}
	w.stack = []int{0, 1}
	f := &floor{owner: 1}
	s := unit.Sense{Field: unit.HexField(), Nearby: []unit.Snapshot{{ID: 1, HP: 100, Radius: 18}}}
	f.Handle(ctx, s)
	s.Time += 1.0 / 60
	f.Handle(ctx, s)
	if len(w.stack) != 2 {
		t.Fatal("unchanged HP consumed equipment")
	}
	s.Nearby[0].HP = 87
	s.Time += 1.0 / 60
	f.Handle(ctx, s)
	if w.has(partPlate) || !w.has(partPiston) || math.Abs(w.wear-1) > 1e-9 {
		t.Fatal("confirmed net loss did not pop top component")
	}
	s.Nearby[0].HP = 100
	s.Time += 1.0 / 60
	f.Handle(ctx, s)
	s.Nearby[0].HP = 89
	s.Time += 1.0 / 60
	f.Handle(ctx, s)
	if len(w.stack) != 0 || w.wear != 0 {
		t.Fatal("healing erased accumulated damage progress")
	}
}
