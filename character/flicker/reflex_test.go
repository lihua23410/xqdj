package 闪烁

import (
	"math"
	"testing"
	"xqdj/internal/unit"
)

func foe(id uint64, x, y float64) unit.Snapshot {
	return unit.Snapshot{ID: id, Slot: 1, Role: unit.RoleFighter, X: x, Y: y, Radius: 18, HP: 100, AimPriority: 15}
}
func approaching() unit.Snapshot { b := shot(2, 70, 0, -400, 0); b.OwnerID = 8; return b }

func plannedThreat(s unit.Sense) *unit.Threat {
	_, _, d, ok := planDodge(nil, s, 0)
	if !ok {
		return nil
	}
	return &d
}

func TestThreatPredictionAndEarliestImpact(t *testing.T) {
	me := self()
	for _, b := range []unit.Snapshot{shot(2, 70, 0, 400, 0), shot(2, 70, 35, -400, 0), shot(2, 180, 0, -400, 0)} {
		if plannedThreat(sense(0, me, b)) != nil {
			t.Fatalf("not an imminent threat: %+v", b)
		}
	}
	near := shot(3, 45, 0, -150, 0)
	fast := shot(9, 90, 0, -900, 0)
	if b := plannedThreat(sense(0, me, near, fast)); b == nil || b.From != 9 {
		t.Fatalf("earliest threat=%+v", b)
	}
	tie := fast
	tie.ID = 4
	if b := plannedThreat(sense(0, me, fast, tie)); b == nil || b.From != 4 {
		t.Fatal("ties must be stable")
	}
	me.VX = 200
	if plannedThreat(sense(0, me, shot(2, 100, 0, -400, 0))) == nil {
		t.Fatal("must include self velocity")
	}
}

func TestThreatFiltersAndWallProtection(t *testing.T) {
	for _, change := range []func(*unit.Snapshot){func(b *unit.Snapshot) { b.Slot = 0 }, func(b *unit.Snapshot) { b.Role = unit.RoleHelper }, func(b *unit.Snapshot) { b.VX = 0 }} {
		b := approaching()
		change(&b)
		if plannedThreat(sense(0, self(), b)) != nil {
			t.Fatalf("filtered threat=%+v", b)
		}
	}
	s := sense(0, self(), approaching())
	s.Walls = []unit.WallView{{X1: 40, Y1: -100, X2: 40, Y2: 100, Radius: 3}}
	if plannedThreat(s) != nil {
		t.Fatal("wall will intercept the bullet")
	}
	s.Nearby[0].PassWalls = true
	if plannedThreat(s) == nil {
		t.Fatal("wall-passing bullet is dangerous")
	}
}

func TestSideChoosesClearanceAndCrossfire(t *testing.T) {
	for _, walls := range []bool{true, false} {
		a := &闪烁{}
		ctx, out := newCtx()
		s := sense(0, self(), approaching())
		if walls {
			s.Walls = []unit.WallView{{X1: -100, Y1: -42, X2: 100, Y2: -42, Radius: 4}}
		} else {
			s.Nearby = append(s.Nearby, shot(3, 0, -75, 0, 400))
		}
		a.Handle(ctx, s)
		cmds := drain(out)
		v := velocity(cmds)
		if v == nil || v.VY <= 0 || math.Abs(v.VX) > 1e-9 {
			t.Fatalf("must dodge upward: walls=%v velocity=%+v", walls, v)
		}
		if p := lastPass(cmds); p == nil || !p.Hold {
			t.Fatal("side must pass units")
		}
	}
}

func TestEachSidestepCanCounterWithoutAffectingNormalFire(t *testing.T) {
	a := &闪烁{}
	ctx, out := newCtx()
	for _, step := range []struct {
		time                         float64
		threat, dodge, fire, counter bool
	}{
		{time: 0, dodge: true, fire: true},
		{time: sideDuration, counter: true},
		{time: .12},
		{time: .19, threat: true},
		{time: .2, dodge: true},
		{time: .2 + sideDuration, counter: true},
		{time: .55},
		{time: .9, dodge: true, fire: true},
		{time: .9 + sideDuration, counter: true},
	} {
		nearby := []unit.Snapshot{foe(8, 150, 0), foe(9, 0, 60)}
		if step.threat || step.dodge {
			nearby = append(nearby, approaching())
		}
		a.Handle(ctx, sense(step.time, self(), nearby...))
		cmds := drain(out)
		if got := len(fxNamed(cmds, "reflex")); got > 1 || (got == 1) != step.dodge {
			t.Fatalf("t=%.2f: got %d sidesteps, want dodge=%v", step.time, got, step.dodge)
		}
		if step.time == sideDuration {
			v := velocity(cmds)
			p := lastPass(cmds)
			if p == nil || p.Hold || v == nil || math.Abs(math.Hypot(v.VX, v.VY)-cruise) > 1e-9 {
				t.Fatalf("short sidestep must end and return to cruise at the end of the segment: %+v", cmds)
			}
		}
		shots, counters := 0, 0
		for _, c := range cmds {
			if d, ok := launched(c); ok {
				switch c.(unit.Spawn).Kind {
				case kindSplit:
					shots++
					if d.To != 9 || d.Amount != 2 || !d.NoFreeze {
						t.Fatalf("t=%.2f: unexpected shot: %+v", step.time, c)
					}
				case kindSmart:
					counters++
					if d.To != 8 || d.Amount != 1 || !d.NoFreeze {
						t.Fatalf("t=%.2f: unexpected counter: %+v", step.time, c)
					}
				default:
					t.Fatalf("t=%.2f: unexpected projectile: %+v", step.time, c)
				}
			}
		}
		if shots > 1 || (shots == 1) != step.fire {
			t.Fatalf("t=%.2f: got %d shots, want fire=%v", step.time, shots, step.fire)
		}
		if counters > 1 || (counters == 1) != step.counter {
			t.Fatalf("t=%.2f: got %d counters, want counter=%v", step.time, counters, step.counter)
		}
	}
}

func TestSidestepChainsWhenTheRemainingRouteWouldBeHit(t *testing.T) {
	a := &闪烁{}
	ctx, out := newCtx()
	me := self()
	targets := []unit.Snapshot{foe(8, 120, 0), foe(9, 110, 0), foe(10, 140, 0)}
	const tick = 1.0 / 60
	step := func(at float64, nearby []unit.Snapshot, threats ...unit.Threat) []unit.Cmd {
		t.Helper()
		s := sense(at, me, nearby...)
		s.Threats = threats
		a.Handle(ctx, s)
		cmds := drain(out)
		v := velocity(cmds)
		if v == nil {
			t.Fatalf("t=%g: expected a velocity command", at)
		}
		me.VX, me.VY = v.VX, v.VY
		me.X, me.Y = me.X+v.VX*tick, me.Y+v.VY*tick
		return cmds
	}
	first := step(0, append(targets, approaching()))
	if p := lastPass(first); p == nil || !p.Hold || me.VY >= 0 {
		t.Fatalf("expected the initial downward sidestep: %+v", first)
	}

	// Continuing downward at 720 would clear the circle. The current
	// segment only has one tick left, after which cruise would get hit.
	second := step(tick, targets, unit.Threat{
		From: 9, Slot: 1, Y: -15, Radius: 3, ActiveAt: .1, Until: .1,
	})
	if len(fxNamed(second, "reflex")) != 1 || math.Abs(me.VX-sideSpeed) > 1e-9 || lastPass(second) != nil {
		t.Fatalf("expected a chained horizontal sidestep without releasing pass: %+v", second)
	}

	beam := unit.Threat{
		Shape: unit.ThreatCapsule, From: 10, Slot: 1,
		X: -70, Y: -12, X2: 70, Y2: -12, Radius: 2, ActiveAt: .1, Until: .1,
	}
	third := step(2*tick, targets, beam)
	if len(fxNamed(third, "reflex")) != 1 || math.Abs(me.VY-sideSpeed) > 1e-9 || lastPass(third) != nil {
		t.Fatalf("expected a third sidestep away from the horizontal beam: %+v", third)
	}
	continued := step(3*tick, targets, beam)
	if len(fxNamed(continued, "reflex")) != 0 || a.phase != phSide || math.Abs(a.readyAt-(2*tick+reflexCooldown)) > 1e-9 {
		t.Fatalf("safe current route should run to completion: %+v", continued)
	}
	finished := step(4*tick, targets)
	if p := lastPass(finished); p == nil || p.Hold || a.phase != phIdle || math.Abs(me.VY-cruise) > 1e-9 {
		t.Fatalf("last segment must release pass and restore cruise: %+v", finished)
	}
	if a.visual.Echo == nil || a.visual.Echo.X != 0 || a.visual.Echo.Y != 0 {
		t.Fatalf("echo must stay at the beginning of the chain: %+v", a.visual.Echo)
	}
	for tick, tc := range []struct {
		cmds []unit.Cmd
		want uint64
	}{
		{first, 0}, {second, 8}, {third, 9}, {continued, 0}, {finished, 10},
	} {
		var targets []uint64
		for _, c := range tc.cmds {
			if round, ok := c.(unit.Spawn); ok && round.Kind == kindSmart {
				targets = append(targets, round.Data.(bulletData).Target)
			}
		}
		if tc.want == 0 {
			if len(targets) != 0 {
				t.Fatalf("tick %d: unexpected counters to %v", tick, targets)
			}
		} else if len(targets) != 1 || targets[0] != tc.want {
			t.Fatalf("tick %d: expected one counter to %d, got %v", tick, tc.want, targets)
		}
	}
}

func TestBasicFireWithoutProjectilesAndAimPriority(t *testing.T) {
	a := &闪烁{}
	ctx, out := newCtx()
	target := foe(8, 160, 0)
	near := foe(9, 0, 50)
	near.AimPriority = 60
	for _, tm := range []float64{0, .2, .9} {
		a.Handle(ctx, sense(tm, self(), near, target))
		n := 0
		for _, c := range drain(out) {
			if d, ok := launched(c); ok {
				n++
				if d.To != 8 || d.Amount != splitDamage {
					t.Fatalf("basic shot=%+v", d)
				}
			}
		}
		want := 1
		if tm == .2 {
			want = 0
		}
		if n != want {
			t.Fatalf("at %v: %d shots", tm, n)
		}
	}
}

func TestCompletedSidestepAddsOneShotFromItsOrigin(t *testing.T) {
	a := &闪烁{}
	ctx, out := newCtx()
	target := foe(8, 160, 0)
	a.Handle(ctx, sense(0, self(), target, approaching()))
	drain(out)

	me := self()
	me.Y, me.VY = -sideDist, -sideSpeed
	a.Handle(ctx, sense(sideDuration, me, target))
	drain(out)
	if a.visual.Echo == nil {
		t.Fatal("completed sidestep should leave a visible firing echo")
	}

	me.VY = 0
	a.Handle(ctx, sense(.9, me, target))
	var rounds []unit.Spawn
	for _, cmd := range drain(out) {
		if round, ok := cmd.(unit.Spawn); ok && round.Kind == kindSplit {
			if data := round.Data.(bulletData); data.Damage != 2 {
				t.Fatalf("each round must deal 2 damage: %+v", data)
			}
			rounds = append(rounds, round)
		}
	}
	if len(rounds) != 2 {
		t.Fatalf("expected two rounds, got %d", len(rounds))
	}
	if echo, body := rounds[0], rounds[1]; echo.Y != 0 || body.Y != me.Y || echo.VY <= 0 || body.VY >= 0 {
		t.Fatalf("expected shots spreading outward from the start and end of the sidestep: %+v", rounds)
	}
	if a.visual.Echo != nil {
		t.Fatal("the firing echo is consumed by the volley")
	}
}

func TestShotRespectsAimabilityAndSight(t *testing.T) {
	a := &闪烁{}
	ctx, out := newCtx()
	hidden := foe(8, 150, 0)
	hidden.AimPriority = 0
	s := sense(0, self(), hidden, foe(9, -150, 0))
	s.Walls = []unit.WallView{{X1: -60, Y1: -100, X2: -60, Y2: 100, Radius: 4, VisionBlock: true}}
	a.Handle(ctx, s)
	for _, c := range drain(out) {
		if _, ok := launched(c); ok {
			t.Fatal("must not shoot at a hidden or obstructed target")
		}
	}
}

func TestDamageConfirmedInFullWhileIdleAndDodging(t *testing.T) {
	a := &闪烁{}
	ctx, out := newCtx()
	for _, dodging := range []bool{false, true} {
		if dodging {
			a.Handle(ctx, sense(0, self(), approaching()))
			if velocity(drain(out)) == nil {
				t.Fatal("expected a sidestep")
			}
		}
		for i, role := range []string{unit.RoleProjectile, unit.RoleHelper, unit.RoleFighter} {
			a.Handle(ctx, offer(uint64(i+1), .02, 2, 8, role, "", 0))
			cmds := drain(out)
			if len(cmds) != 1 || confirmed(cmds) != 10 {
				t.Fatalf("dodging=%v role=%s: wanted only full confirmation, got %+v", dodging, role, cmds)
			}
		}
	}
}

func TestWallStunAndKnockback(t *testing.T) {
	t.Run("wall", func(t *testing.T) {
		a := &闪烁{}
		ctx, out := newCtx()
		a.Handle(ctx, sense(0, self(), approaching()))
		drain(out)
		a.Handle(ctx, unit.WallHit{Time: .02, NX: 0, NY: 1})
		drain(out)
		a.Handle(ctx, sense(.05, self()))
		cmds := drain(out)
		v := velocity(cmds)
		if a.phase != phIdle || v == nil || v.VY <= 0 || math.Abs(math.Hypot(v.VX, v.VY)-cruise) > 1e-9 {
			t.Fatalf("wall recovery=%+v phase=%v", v, a.phase)
		}
	})
	t.Run("stun", func(t *testing.T) {
		a := &闪烁{}
		ctx, out := newCtx()
		a.Handle(ctx, sense(0, self(), approaching()))
		drain(out)
		a.Handle(ctx, unit.Stunned{Time: .03, Until: .8})
		cmds := drain(out)
		if p := lastPass(cmds); p == nil || p.Hold {
			t.Fatal("stun must release pass")
		}
		a.Handle(ctx, sense(.8, self(), foe(8, 150, 0)))
		shots := 0
		for _, c := range drain(out) {
			if round, ok := c.(unit.Spawn); ok {
				shots++
				if round.Kind != kindSplit {
					t.Fatalf("stun must cancel the counter: %+v", round)
				}
			}
		}
		if shots != 1 {
			t.Fatalf("expected only the normal shot after stun, got %d", shots)
		}
	})
	t.Run("knock", func(t *testing.T) {
		a := &闪烁{}
		ctx, out := newCtx()
		a.Handle(ctx, sense(0, self(), approaching()))
		drain(out)
		me := self()
		me.VX = 460
		for _, tm := range []float64{.05, .2} {
			a.Handle(ctx, senseKnock(tm, me))
			if velocity(drain(out)) != nil {
				t.Fatal("must yield to external push")
			}
		}
		a.Handle(ctx, sense(.25, self()))
		if v := velocity(drain(out)); v == nil || math.Abs(math.Hypot(v.VX, v.VY)-cruise) > 1e-9 {
			t.Fatalf("no cruise recovery: %+v", v)
		}
	})
}

func TestClipReachChecksFinalPartialStep(t *testing.T) {
	walls := []unit.WallView{{X1: 7, Y1: -10, X2: 7, Y2: 10, Radius: 1}}
	if reach := clipReach(0, 0, 1, 0, 7, 1, unit.Field{}, walls); reach > 5.01 {
		t.Fatalf("unchecked tail crossed wall: %v", reach)
	}
}

func TestWallRecoveryWithReflectedSideVelocity(t *testing.T) {
	for _, tc := range []struct {
		name         string
		hits         int
		extraTargets bool
	}{
		{name: "one bullet without damage"},
		{name: "repeated damage", hits: 4},
		{name: "multiple targets", extraTargets: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &闪烁{}
			ctx, out := newCtx()
			nearby := []unit.Snapshot{approaching()}
			if tc.extraTargets {
				nearby = append(nearby, foe(8, 150, 0), foe(9, 0, 60))
			}
			a.Handle(ctx, sense(0, self(), nearby...))
			v := velocity(drain(out))
			if v == nil {
				t.Fatal("missing sidestep")
			}
			me := self()
			me.VX, me.VY = v.VX, -v.VY
			a.Handle(ctx, unit.WallHit{Time: .02, NX: 0, NY: 1})
			drain(out)
			for i := 0; i < tc.hits; i++ {
				a.Handle(ctx, offer(uint64(i+1), .02, 2, 8, unit.RoleProjectile, "原型机_整合弹", 0))
				drain(out)
				me.HP -= 10
			}
			a.Handle(ctx, sense(.05, me, nearby[1:]...))
			v = velocity(drain(out))
			if v == nil || math.Abs(v.VX) > 1e-9 || math.Abs(v.VY-cruise) > 1e-9 {
				t.Fatalf("reflected sidestep must recover to cruise: velocity=%+v phase=%v pending=%v", v, a.phase, a.wantCruise)
			}
		})
	}
}

func TestWallRecoveryWaitsForKnockbackAndTracksFurtherBounces(t *testing.T) {
	a := &闪烁{}
	ctx, out := newCtx()
	a.Handle(ctx, sense(0, self(), approaching()))
	drain(out)
	a.Handle(ctx, unit.WallHit{Time: .02, NX: 0, NY: 1})
	drain(out)
	me := self()
	me.VY = sideSpeed
	a.Handle(ctx, senseKnock(.05, me))
	if v := velocity(drain(out)); v != nil {
		t.Fatalf("wall recovery overwrote active knockback: %+v", v)
	}
	// A second wall is reached before the knockback releases the velocity.
	a.Handle(ctx, unit.WallHit{Time: .06, NX: math.Sqrt(.5), NY: math.Sqrt(.5)})
	drain(out)
	me.VX, me.VY = -sideSpeed, 0
	a.Handle(ctx, senseKnock(.07, me))
	if v := velocity(drain(out)); v != nil {
		t.Fatalf("second bounce overwrote active knockback: %+v", v)
	}
	a.Handle(ctx, sense(.08, me))
	v := velocity(drain(out))
	if v == nil || math.Abs(v.VX+cruise) > 1e-9 || math.Abs(v.VY) > 1e-9 {
		t.Fatalf("must recover along the twice-reflected direction: %+v", v)
	}
}

// 检查开火对象和弹体携带的伤害；实际碰撞另有测试。
func launched(c unit.Cmd) (unit.Damage, bool) {
	spawn, ok := c.(unit.Spawn)
	if !ok {
		return unit.Damage{}, false
	}
	data, ok := spawn.Data.(bulletData)
	return unit.Damage{To: data.Target, Amount: data.Damage, NoFreeze: !data.Boost}, ok
}
func TestBulletCollisionAndLimitedTracking(t *testing.T) {
	ctx, out := newCtx()
	for _, smart := range []bool{false, true} {
		b := &projectile{bulletData: bulletData{Target: 8, Smart: smart, Damage: 3, Until: 1}, owner: 1, slot: 0}
		me := self()
		me.ID = 20
		me.VX = bulletSpeed
		b.Handle(ctx, sense(.1, me, foe(8, 100, 50)))
		v := velocity(drain(out))
		if !smart && v != nil {
			t.Fatal("ordinary bullet steered")
		}
		if smart && (v == nil || v.VY <= 0 || math.Atan2(v.VY, v.VX) > turnRate/60+1e-9) {
			t.Fatalf("tracking exceeded limit: %+v", v)
		}
		b.Handle(ctx, unit.Collision{Other: foe(8, 100, 50)})
		cmds := drain(out)
		hit := false
		for _, c := range cmds {
			if d, ok := c.(unit.Damage); ok {
				hit = true
				if d.From != ctx.ID || d.Amount != 3 {
					t.Fatal(d)
				}
			}
		}
		if !hit {
			t.Fatal("collision did not damage")
		}
		b.Handle(ctx, unit.Collision{Other: foe(8, 100, 50)})
		if len(drain(out)) != 0 {
			t.Fatal("duplicate impact")
		}
	}
	b := &projectile{bulletData: bulletData{Smart: true, Target: 8, Until: 1}}
	b.Handle(ctx, sense(.1, self()))
	drain(out)
	if b.Target != 0 {
		t.Fatal("missing target must release lock")
	}
	b.Handle(ctx, unit.WallHit{})
	if !b.dead {
		t.Fatal("wall must stop bullet")
	}
}

func TestSplitRoundBendsOnce(t *testing.T) {
	ctx, out := newCtx()
	b := &projectile{bulletData: bulletData{Target: 8, Until: 1, BendAt: .1}}
	me := self()
	me.X, me.Y = 43, 39
	target := foe(8, 140, 39)
	target.VY = 150

	b.Handle(ctx, sense(.09, me, target))
	if v := velocity(drain(out)); v != nil {
		t.Fatalf("turned too early: %+v", v)
	}
	b.Handle(ctx, sense(.1, me, target))
	if v := velocity(drain(out)); v == nil || v.VX <= 0 || v.VY <= 0 || math.Abs(math.Hypot(v.VX, v.VY)-bulletSpeed) > 1e-9 {
		t.Fatalf("expected a single turn ahead of the moving target: %+v", v)
	}
	target.Y = -50
	b.Handle(ctx, sense(.11, me, target))
	if v := velocity(drain(out)); v != nil {
		t.Fatalf("round kept steering after its turn: %+v", v)
	}
}

func TestForecastFanUsesItsAttackSide(t *testing.T) {
	fan := shot(2, 80, 0, -400, 0)
	fan.OwnerID, fan.Radius, fan.ArcInner, fan.ArcSpan = 8, 32, 15, math.Pi/3
	fan.FaceX = 1
	if plannedThreat(sense(0, self(), fan)) != nil {
		t.Fatal("approaching with the fan pointing away should not trigger")
	}
	fan.FaceX = -1
	if dx, dy, d, ok := planDodge(nil, sense(0, self(), fan), 0); !ok || d.From != 8 || math.Abs(dx) > 1e-9 || math.Abs(dy) < .99 {
		t.Fatalf("expected a lateral dodge from the front-facing fan, got (%g,%g), %+v", dx, dy, d)
	}
}

func TestBeamDodgeFollowsTheBeamNormal(t *testing.T) {
	me := self()
	me.X, me.Y, me.VY = 30, -40, 170
	for _, shape := range []unit.ThreatShape{unit.ThreatRay, unit.ThreatCapsule} {
		s := sense(0, me)
		s.Threats = []unit.Threat{{Shape: shape, Slot: 1, X: -40, X2: 100, Radius: 6, ActiveAt: .12, Until: .12}}
		if dx, dy, _, ok := planDodge(nil, s, 0); !ok || math.Abs(dx) > 1e-9 || dy > -.99 {
			t.Fatalf("expected to dodge perpendicular to the beam, got (%g,%g), ok=%v", dx, dy, ok)
		}
	}
}

func TestTimedTerrainDodgeLeavesEchoAndCounters(t *testing.T) {
	a := &闪烁{}
	ctx, out := newCtx()
	me, target := self(), foe(8, 140, 0)
	me.VX = cruise
	s := sense(0, me, target)
	s.Threats = []unit.Threat{{Slot: -1, X: 25, Radius: 20, ActiveAt: .1, Until: .1}}
	a.Handle(ctx, s)
	v := velocity(drain(out))
	if v == nil || math.Abs(v.VX) > 1e-9 || math.Abs(math.Abs(v.VY)-sideSpeed) > 1e-9 {
		t.Fatalf("expected a sidestep perpendicular to the approach: %+v", v)
	}
	me.X, me.Y, me.VX, me.VY = v.VX*sideDuration, v.VY*sideDuration, v.VX, v.VY
	a.Handle(ctx, sense(sideDuration, me, target))
	if a.visual.Echo == nil {
		t.Fatal("terrain sidestep must leave an echo")
	}
	for _, cmd := range drain(out) {
		if round, ok := cmd.(unit.Spawn); ok && round.Kind == kindSmart {
			if data := round.Data.(bulletData); data.Target != target.ID {
				t.Fatalf("terrain counter must use normal aim: %+v", data)
			}
			return
		}
	}
	t.Fatal("terrain sidestep must counter")
}

func TestNoDodgeWhenEveryPathStillGetsHit(t *testing.T) {
	s := sense(0, self())
	s.Threats = []unit.Threat{{Slot: 1, From: 8, Radius: 80, ActiveAt: .1, Until: .1}}
	if d := plannedThreat(s); d != nil {
		t.Fatalf("no reachable exit from a blast centered on the fighter: %+v", d)
	}
	s.Threats[0].ActiveAt, s.Threats[0].Until = .5, .5
	if d := plannedThreat(s); d == nil {
		t.Fatal("long warning allows the sidestep and following cruise to clear the blast")
	}
	s.Self.VX = cruise
	s.Threats[0] = unit.Threat{Slot: 1, From: 8, X: 25, Radius: 40, ActiveAt: .1, Until: .1}
	if d := plannedThreat(s); d != nil {
		t.Fatalf("both normals are unsafe, retreating must not spend a sidestep: %+v", d)
	}
}

func TestTrailSamplesTwiceAsOften(t *testing.T) {
	start := func() *闪烁 {
		a := &闪烁{}
		ctx, out := newCtx()
		a.Handle(ctx, sense(0, self(), approaching()))
		drain(out)
		return a
	}
	early := start()
	ctx, out := newCtx()
	early.Handle(ctx, sense(0.011, self()))
	drain(out)
	if len(early.visual.Trail) != 1 {
		t.Fatalf("trail=%d", len(early.visual.Trail))
	}
	later := start()
	later.Handle(ctx, sense(trailSample, self()))
	drain(out)
	if len(later.visual.Trail) < 2 {
		t.Fatalf("trail=%d", len(later.visual.Trail))
	}
}

func offer(token uint64, t float64, from, owner uint64, role, kind string, span float64) unit.IncomingDamage {
	return unit.IncomingDamage{
		Token: token, From: from, SourceOwnerID: owner, Amount: 10, Time: t,
		SourceRole: role, SourceKind: kind, ArcSpan: span,
	}
}
