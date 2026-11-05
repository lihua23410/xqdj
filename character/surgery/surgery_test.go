package 外科手术

import (
	"math"
	"testing"
	"xqdj/internal/unit"
	_ "xqdj/map"
)

func testContext(id uint64, state *unit.MatchState) (unit.Context, chan unit.Cmd) {
	out := make(chan unit.Cmd, 4096)
	return unit.Context{ID: id, State: state, Out: out, Field: unit.HexField(), Stream: id}.WithBirths(), out
}

func take(out chan unit.Cmd) []unit.Cmd {
	var got []unit.Cmd
	for len(out) > 0 {
		got = append(got, <-out)
	}
	return got
}

func finishedWound(id uint64) *wound {
	w := makeWound(id, point{-120, 0}, point{120, 0}, unit.HexField())
	w.Drawn = 1
	for i := range w.Closed {
		w.Closed[i] = true
	}
	return w
}

func TestMoreWoundsAccelerateBurstAndPartialSutureIsPreserved(t *testing.T) {
	for count, duration := range map[int]float64{1: 20, 3: 200.0 / 14, 6: 10} {
		o := &operation{}
		for i := 0; i < count; i++ {
			w := finishedWound(uint64(i + 1))
			w.Closed[0] = false
			o.wounds = append(o.wounds, w)
		}
		o.advance(0)
		if o.advance(duration - .01) {
			t.Fatal("burst before full")
		}
		if o.advance(duration) || math.Abs(o.charge-200) > 1e-6 {
			t.Fatalf("%d open wounds did not fill 200 at %v: %v", count, duration, o.charge)
		}
		o.wounds[0].Closed[0] = true
		if !o.advance(duration+stepTime) || o.wounds[0].complete() {
			t.Fatal("burst must reopen the completed wound")
		}
		for _, w := range o.wounds[1:] {
			if w.Closed[0] {
				t.Fatal("burst closed a wound that was still open")
			}
		}
	}
	partial := finishedWound(2)
	partial.Closed[3] = false
	o := &operation{wounds: []*wound{finishedWound(1), partial}, charge: burstCharge}
	if !o.advance(0) || !partial.Closed[0] || partial.Closed[3] {
		t.Fatal("burst changed a partial suture")
	}
}

func TestFullySuturedWoundsDoNotChargeAndRetainProgress(t *testing.T) {
	o := &operation{wounds: []*wound{finishedWound(1), finishedWound(2), finishedWound(3)}, charge: 37}
	o.advance(0)
	o.advance(30)
	if o.charge != 37 {
		t.Fatalf("closed wounds increased progress to %v", o.charge)
	}
	o.wounds[1].Closed[0] = false
	o.advance(31)
	if o.charge != 47 {
		t.Fatalf("only one open wound should contribute: %v", o.charge)
	}
	o.wounds[1].Closed[0] = true
	o.advance(50)
	if o.charge != 47 {
		t.Fatal("sealing the last wound lost or advanced progress")
	}
	o.wounds[1].Closed[0] = false
	o.advance(51)
	if o.charge != 57 {
		t.Fatal("reopened wound did not resume charging")
	}
}

func TestFullChargeWaitsForFirstCompletedSuture(t *testing.T) {
	w := finishedWound(1)
	w.Closed[0] = false
	o := &operation{wounds: []*wound{w}, charge: burstCharge}
	if o.advance(0) || o.charge != burstCharge {
		t.Fatal("empty burst consumed full charge")
	}
	w.Closed[0] = true
	if !o.advance(.1) || w.complete() {
		t.Fatal("first completed wound did not release held charge")
	}
}

func finishSutureAtLastPuncture(o *operation, w *wound, now float64) {
	last := len(w.Points) - 1
	for i := range w.Closed {
		w.Closed[i] = i != last-1
	}
	p := w.Points[last]
	o.stage, o.current, o.direction, o.next, o.lastBody = stitching, w.ID, 1, last, p
	var cmds commands
	o.sew(1, unit.Sense{Time: now, Self: unit.Snapshot{ID: 1, X: p.X, Y: p.Y}, Field: unit.HexField()}, &cmds)
}

func TestCompletingSutureAddsNoCharge(t *testing.T) {
	w := finishedWound(1)
	o := &operation{wounds: []*wound{w}, charge: 20}
	finishSutureAtLastPuncture(o, w, 1)
	if o.charge != 20 {
		t.Fatalf("first closure charge=%v want 20", o.charge)
	}
	var cmds commands
	o.sew(1, unit.Sense{Time: 1.1, Field: unit.HexField()}, &cmds)
	if o.charge != 20 {
		t.Fatal("completed wound was rewarded again")
	}
	o.charge = 200
	if !o.advance(2) || w.complete() {
		t.Fatal("wound did not reopen")
	}
	finishSutureAtLastPuncture(o, w, 3)
	if o.charge != 0 {
		t.Fatalf("reclosed wound added charge: %v", o.charge)
	}
}

func TestScalpelStartsAfterPointOneSecondWindup(t *testing.T) {
	scope := new(unit.MatchState)
	ctx, out := testContext(1, scope)
	a := &surgeon{booted: true}
	s := unit.Sense{Time: 1, Field: unit.HexField(), Self: unit.Snapshot{ID: 1, Slot: 0, Role: unit.RoleFighter, HP: 100, X: -80, VX: 170},
		Nearby: []unit.Snapshot{{ID: 2, Slot: 1, Role: unit.RoleFighter, HP: 100, X: 80, AimPriority: 15}}}
	a.Handle(ctx, s)
	take(out)
	s.Time = 1.099
	a.Handle(ctx, s)
	for _, cmd := range take(out) {
		if spawn, ok := cmd.(unit.Spawn); ok && spawn.Kind == KindScalpel {
			t.Fatal("knife started before 0.1s")
		}
	}
	s.Time = 1.1
	a.Handle(ctx, s)
	knives := 0
	for _, cmd := range take(out) {
		if spawn, ok := cmd.(unit.Spawn); ok && spawn.Kind == KindScalpel {
			knives++
		}
	}
	if knives != 1 {
		t.Fatalf("knives at 0.1s=%d want 1", knives)
	}
}

func TestCrossCutIsPerpendicularEqualLengthAndCentered(t *testing.T) {
	a, b := point{-100, 0}, point{100, 0}
	plan, ok := planCrossKnife(unit.HexField(), a, b)
	if !ok {
		t.Fatal("missing cross cut")
	}
	center := lerp(a, b, .5)
	if lerp(plan.A, plan.B, .5).sub(center).length() > 1e-6 {
		t.Fatal("cross cut is not centered on first wound")
	}
	if math.Abs(plan.B.sub(plan.A).length()-b.sub(a).length()) > 1e-6 {
		t.Fatal("unclipped cross cut length changed")
	}
	d, other := b.sub(a), plan.B.sub(plan.A)
	if math.Abs(d.X*other.X+d.Y*other.Y) > 1e-6 {
		t.Fatal("cross cut is not perpendicular")
	}
}

func TestCrossCutStopsEachArmAtItsOwnFieldEdge(t *testing.T) {
	a, b := point{220, -60}, point{220, 60}
	plan, ok := planCrossKnife(unit.HexField(), a, b)
	if !ok {
		t.Fatal("missing edge-clipped cross cut")
	}
	center := lerp(a, b, .5)
	if math.Abs(plan.Start.sub(center).length()-60) > 1e-6 {
		t.Fatal("clear arm was shortened with the blocked arm")
	}
	if plan.End.sub(center).length() >= 60 {
		t.Fatal("blocked arm did not stop at the edge")
	}
	for _, p := range []point{plan.Start, plan.End} {
		if !unit.HexField().OutlineContains(p.X, p.Y, 6) {
			t.Fatal("knife escaped field")
		}
	}
	w := makeWound(2, plan.A, plan.B, unit.HexField())
	for _, p := range w.Points {
		if !unit.HexField().OutlineContains(p.X, p.Y, bodyRadius) {
			t.Fatal("suture point escaped field")
		}
	}
}

func TestExactlyTwoCutsFinishBeforeSuturingAndEachCanDamage(t *testing.T) {
	scope := new(unit.MatchState)
	ctx, out := testContext(1, scope)
	a := &surgeon{booted: true}
	s := unit.Sense{Time: 1, Field: unit.HexField(), Self: unit.Snapshot{ID: 1, Slot: 0, Role: unit.RoleFighter, HP: 100, X: -80},
		Nearby: []unit.Snapshot{{ID: 2, Slot: 1, Role: unit.RoleFighter, HP: 100, X: 80, AimPriority: 15}}}
	a.Handle(ctx, s)
	take(out)
	spawnAt := func(now float64) unit.Spawn {
		s.Time = now
		a.Handle(ctx, s)
		for _, cmd := range take(out) {
			if born, ok := cmd.(unit.Spawn); ok && born.Kind == KindScalpel {
				return born
			}
		}
		t.Fatalf("no scalpel at %v", now)
		return unit.Spawn{}
	}
	endKnife := func(spawn unit.Spawn, id uint64, now float64) {
		child, childOut := testContext(id, scope)
		knife := unit.NewActor(spawn.Kind, unit.SpawnInfo{OwnerID: 1, Slot: 0, State: scope, Data: spawn.Data})
		contact := unit.Collision{Time: now, Other: unit.Snapshot{ID: 99, Slot: 1, Role: unit.RoleFighter, HP: 100}}
		knife.Handle(child, contact)
		knife.Handle(child, contact)
		hits := 0
		for _, cmd := range take(childOut) {
			if d, ok := cmd.(unit.Damage); ok {
				hits++
				if d.Amount != 8 || d.From != id {
					t.Fatal("wrong per-stroke damage")
				}
			}
		}
		if hits != 1 {
			t.Fatalf("stroke quoted %d hits, want one", hits)
		}
		plan := spawn.Data.(knifePlan)
		knife.Handle(child, unit.Sense{Time: now, Self: unit.Snapshot{ID: id, X: plan.End.X, Y: plan.End.Y}})
		take(childOut)
	}
	first := spawnAt(1.1)
	endKnife(first, 11, 1.2)
	s.Time = 1.21
	a.Handle(ctx, s)
	take(out)
	o := unit.StateFor[clinic](scope).operation(1)
	if o.stage != winding || len(o.wounds) != 1 {
		t.Fatal("suturing began before second cut")
	}
	s.Time = 1.309
	a.Handle(ctx, s)
	for _, cmd := range take(out) {
		if born, ok := cmd.(unit.Spawn); ok && born.Kind == KindScalpel {
			t.Fatal("second cut skipped its 0.1s windup")
		}
	}
	second := spawnAt(1.31)
	if len(o.wounds) != 2 || o.wounds[0].ID == o.wounds[1].ID {
		t.Fatal("cross was not recorded as two separate wounds")
	}
	endKnife(second, 12, 1.4)
	s.Time = 1.41
	a.Handle(ctx, s)
	for _, cmd := range take(out) {
		if born, ok := cmd.(unit.Spawn); ok && born.Kind == KindScalpel {
			t.Fatal("second cut recursively created a third cut")
		}
	}
	if o.stage < approaching {
		t.Fatal("did not enter suturing after both cuts")
	}
	o.charge = 0
	finishSutureAtLastPuncture(o, o.wounds[0], 2)
	finishSutureAtLastPuncture(o, o.wounds[1], 3)
	if o.charge != 0 {
		t.Fatalf("completing sutures added charge: %v", o.charge)
	}
}

func TestUnsewnWoundsSprayUntilEverySegmentIsClosed(t *testing.T) {
	scope := new(unit.MatchState)
	ctx, out := testContext(9, scope)
	o := unit.StateFor[clinic](scope).operation(1)
	w := finishedWound(1)
	for i := range w.Closed {
		w.Closed[i] = false
	}
	o.wounds = []*wound{w}
	f := &woundField{owner: 1}
	count := func(from, to float64) int {
		n := 0
		for now := from; now < to-1e-8; now += stepTime {
			f.Handle(ctx, unit.Sense{Time: now, Field: unit.HexField()})
			for _, cmd := range take(out) {
				if s, ok := cmd.(unit.Spawn); ok && s.Kind == KindBloodBullet {
					n++
				}
			}
		}
		return n
	}
	// No 大出血: an unsewn wound bleeds by itself.
	first := count(0, 1)
	second := count(1, 2)
	if first != 12 || second != 12 {
		t.Fatalf("continuous blood: first=%d second=%d", first, second)
	}
	for i := 0; i < len(w.Closed)/2; i++ {
		w.Closed[i] = true
	}
	partial := count(2, 3)
	if partial >= second || partial == 0 {
		t.Fatalf("partial closure must stop some jets, got %d", partial)
	}
	for i := range w.Closed {
		w.Closed[i] = true
	}
	if got := count(3, 4); got != 0 {
		t.Fatalf("closed wound still sprayed %d bullets", got)
	}
	if w.Bleeding {
		t.Fatal("a fully sutured wound still reports itself as bleeding")
	}
}

func TestBloodImmunityRecognizesAnotherSurgeonAfterBulletDisappears(t *testing.T) {
	scope := new(unit.MatchState)
	world := unit.StateFor[clinic](scope)
	bulletCtx, bulletOut := testContext(88, scope)
	b := &bloodBullet{}
	b.Handle(bulletCtx, unit.Collision{Time: 3, Other: unit.Snapshot{ID: 1, Kind: KindSurgery, Role: unit.RoleFighter}})
	got := take(bulletOut)
	if len(got) != 2 {
		t.Fatal("blood must quote and despawn on body contact")
	}
	a := &surgeon{booted: true}
	ctx, out := testContext(1, scope)
	o := world.operation(1)
	o.stage = approaching
	a.Handle(ctx, unit.IncomingDamage{From: 88, Token: 4, Time: 3, Amount: 2})
	if _, ok := (<-out).(unit.BlockDamage); !ok {
		t.Fatal("stitching surgeon did not block blood")
	}
	a.Handle(ctx, unit.Stunned{Time: 3, Until: 5})
	take(out)
	a.Handle(ctx, unit.IncomingDamage{From: 88, Token: 5, Time: 3.1, Amount: 2})
	if _, ok := (<-out).(unit.ConfirmDamage); !ok {
		t.Fatal("stunned surgeon kept blood immunity")
	}
	a.Handle(ctx, unit.IncomingDamage{From: 123, Token: 6, Time: 5, Amount: 8})
	if _, ok := (<-out).(unit.ConfirmDamage); !ok {
		t.Fatal("ordinary damage was incorrectly blocked")
	}
	if unit.StateFor[clinic](new(unit.MatchState)).isBlood(88, 3) {
		t.Fatal("blood source leaked to another match")
	}
}

func TestNeedleDamageBelongsToActiveNeedleAndIsLimitedPerWound(t *testing.T) {
	scope := new(unit.MatchState)
	ctx, out := testContext(7, scope)
	o := unit.StateFor[clinic](scope).operation(1)
	w := finishedWound(2)
	w.Closed[0] = false
	o.wounds = []*wound{w}
	o.stage = stitching
	o.current = 2
	a := &needle{owner: 1, wound: 2, slot: 0}
	for _, now := range []float64{0, .1, .2, .4, .6} {
		a.Handle(ctx, unit.Collision{Time: now, Other: unit.Snapshot{ID: 9, Role: unit.RoleFighter, Slot: 1}})
	}
	cmds := take(out)
	if len(cmds) != 3 {
		t.Fatalf("needle hit count=%d want 3", len(cmds))
	}
	for _, cmd := range cmds {
		if d := cmd.(unit.Damage); d.From != 7 || d.Amount != 2 {
			t.Fatal("needle damage has wrong source")
		}
	}
	o.stage = approaching
	a.Handle(ctx, unit.Collision{Time: 2, Other: unit.Snapshot{ID: 10, Role: unit.RoleFighter, Slot: 1}})
	if len(out) != 0 {
		t.Fatal("transfer created needle damage")
	}
}

func TestKnifeAndZigzagStayInsideEveryField(t *testing.T) {
	for _, name := range unit.FieldNames() {
		field, _ := unit.LookupField(name)
		for i := 0; i < 12; i++ {
			a := float64(i) * math.Pi / 6
			plan, ok := planKnife(field, point{}, point{math.Cos(a) * 100, math.Sin(a) * 100})
			if !ok {
				t.Fatalf("%s direction %d has no cut", name, i)
			}
			if !field.OutlineContains(plan.Start.X, plan.Start.Y, 6) {
				t.Fatalf("knife outside %s", name)
			}
			w := makeWound(1, plan.A, plan.B, field)
			for n, p := range w.Points {
				if !field.OutlineContains(p.X, p.Y, bodyRadius) {
					t.Fatalf("puncture %d outside %s", n, name)
				}
				if n > 0 && !clearLine(field, w.Points[n-1], p, bodyRadius) {
					t.Fatalf("stitch cuts outside %s", name)
				}
			}
		}
	}
}

func TestTransfersRouteAroundChimeraExterior(t *testing.T) {
	field, ok := unit.LookupField("奇美拉")
	if !ok {
		t.Fatal("missing chimera")
	}
	from, to := point{-700, 0}, point{0, 700}
	// Use guaranteed valid interior points from the field's region footprint.
	from.X = -600
	to.Y = 600
	if !field.OutlineContains(from.X, from.Y, 19) || !field.OutlineContains(to.X, to.Y, 19) {
		t.Fatal("invalid test endpoints")
	}
	if clearLine(field, from, to, 19) {
		t.Fatal("test must cross an exterior gap")
	}
	path := fieldRoute(field, from, to, 19)
	if len(path) < 2 {
		t.Fatal("no transfer route through the hub")
	}
	p := from
	for _, q := range path {
		if !clearLine(field, p, q, 19) {
			t.Fatal("route leaves field")
		}
		p = q
	}
	if p.sub(to).length() > 1 {
		t.Fatal("route did not reach wound")
	}
}

// Observer stores only the kind and recreates a projectile without private Data.
func TestCopiedScalpelNeedsNoPrivateOperationPlan(t *testing.T) {
	scope := new(unit.MatchState)
	ctx, out := testContext(77, scope)
	a := unit.NewActor(KindScalpel, unit.SpawnInfo{OwnerID: 99, Slot: 1, State: scope})
	a.Handle(ctx, unit.Sense{Time: 1, Self: unit.Snapshot{ID: 77, VX: 1000}})
	contact := unit.Collision{Time: 1.1, Other: unit.Snapshot{ID: 3, Role: unit.RoleFighter, Slot: 0}}
	a.Handle(ctx, contact)
	a.Handle(ctx, contact)
	hits := 0
	for _, cmd := range take(out) {
		if d, ok := cmd.(unit.Damage); ok {
			hits++
			if d.From != 77 || d.To != 3 || d.Amount != 8 {
				t.Fatalf("invalid copied scalpel damage: %+v", d)
			}
		}
	}
	if hits != 1 {
		t.Fatalf("copied scalpel hit %d times, want once", hits)
	}
	if len(unit.StateFor[clinic](scope).operations) != 0 {
		t.Fatal("copied scalpel created a surgeon operation")
	}
	a.Handle(ctx, unit.WallHit{Time: 1.2, Kind: unit.WallEdge})
	if _, ok := (<-out).(unit.Despawn); !ok {
		t.Fatal("copied scalpel did not leave on field edge")
	}
}
