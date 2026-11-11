package 闪烁

import (
	"math"
	"math/rand/v2"
	"testing"
	"xqdj/internal/unit"
)

func foe(id uint64, x, y float64) unit.Snapshot {
	return unit.Snapshot{ID: id, Slot: 1, Role: unit.RoleFighter, X: x, Y: y, Radius: 18, HP: 100, AimPriority: 15}
}
func approaching() unit.Snapshot { b := shot(2, 70, 0, -400, 0); b.OwnerID = 8; return b }

func TestThreatPredictionAndEarliestImpact(t *testing.T) {
	me := self()
	for _, b := range []unit.Snapshot{shot(2, 70, 0, 400, 0), shot(2, 70, 35, -400, 0), shot(2, 180, 0, -400, 0)} {
		if imminent(sense(0, me, b)) != nil {
			t.Fatalf("not an imminent threat: %+v", b)
		}
	}
	near := shot(3, 45, 0, -150, 0)
	fast := shot(9, 90, 0, -900, 0)
	if b := imminent(sense(0, me, near, fast)); b == nil || b.ID != 9 {
		t.Fatalf("earliest threat=%+v", b)
	}
	tie := fast
	tie.ID = 4
	if b := imminent(sense(0, me, fast, tie)); b == nil || b.ID != 4 {
		t.Fatal("ties must be stable")
	}
	me.VX = 200
	if imminent(sense(0, me, shot(2, 100, 0, -400, 0))) == nil {
		t.Fatal("must include self velocity")
	}
}

func TestThreatFiltersAndWallProtection(t *testing.T) {
	for _, change := range []func(*unit.Snapshot){func(b *unit.Snapshot) { b.Slot = 0 }, func(b *unit.Snapshot) { b.Kind = bloodBullet }, func(b *unit.Snapshot) { b.ArcSpan = 1 }, func(b *unit.Snapshot) { b.Role = unit.RoleHelper }, func(b *unit.Snapshot) { b.VX = 0 }} {
		b := approaching()
		change(&b)
		if imminent(sense(0, self(), b)) != nil {
			t.Fatalf("filtered threat=%+v", b)
		}
	}
	s := sense(0, self(), approaching())
	s.Walls = []unit.WallView{{X1: 40, Y1: -100, X2: 40, Y2: 100, Radius: 3}}
	if imminent(s) != nil {
		t.Fatal("wall will intercept the bullet")
	}
	s.Nearby[0].PassWalls = true
	if imminent(s) == nil {
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
			s.Nearby = append(s.Nearby, shot(3, 0, -120, 0, 400))
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

func TestBurstAimsAtSourceAndDoesNotRequireNoDamage(t *testing.T) {
	a := &闪烁{}
	ctx, out := newCtx()
	me := self()
	target := foe(8, 150, 0)
	decoy := foe(9, 0, 60)
	a.Handle(ctx, sense(0, me, approaching(), target, decoy))
	drain(out)
	me.HP = 99
	for i, tm := range []float64{.16, .27, .38} {
		a.Handle(ctx, sense(tm, me, target, decoy))
		cmds := drain(out)
		n := 0
		for _, c := range cmds {
			if d, ok := launched(c); ok {
				if d.To != 8 || d.Amount != beamDamage {
					t.Fatalf("shot %d: %+v", i, d)
				}
				n++
			}
		}
		if n != 1 {
			t.Fatalf("shot %d: damage count=%d", i, n)
		}
	}
	a.Handle(ctx, sense(.42, me, target))
	for _, c := range drain(out) {
		if _, ok := launched(c); ok {
			t.Fatal("fourth burst shot")
		}
	}
}

func TestPerfectFinalShotAndFallbackTarget(t *testing.T) {
	a := &闪烁{}
	ctx, out := newCtx()
	target := foe(9, 140, 20)
	a.Handle(ctx, sense(0, self(), approaching()))
	drain(out)
	for i, tm := range []float64{.16, .27, .38} {
		a.Handle(ctx, sense(tm, self(), target))
		cmds := drain(out)
		want := beamDamage
		if i == 2 {
			want = perfectDamage
		}
		found := false
		for _, c := range cmds {
			if d, ok := launched(c); ok {
				found = true
				if d.To != 9 || d.Amount != want || d.NoFreeze != (i != 2) {
					t.Fatalf("shot %d: %+v", i, d)
				}
			}
		}
		if !found {
			t.Fatal("missing fallback counter")
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
				if d.To != 8 || d.Amount != shotDamage {
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

func TestCounterSpendsMissingTargetsAndRespectsAimability(t *testing.T) {
	a := &闪烁{}
	ctx, out := newCtx()
	hidden := foe(8, 150, 0)
	hidden.AimPriority = 0
	a.Handle(ctx, sense(0, self(), approaching()))
	drain(out)
	a.Handle(ctx, sense(.16, self(), hidden))
	for _, c := range drain(out) {
		if _, ok := launched(c); ok {
			t.Fatal("must not target aim=0")
		}
	}
	a.Handle(ctx, sense(.27, self(), foe(9, 150, 0)))
	drain(out)
	if a.burstIndex != 2 {
		t.Fatal("missed opportunity must not accumulate")
	}
	s := sense(.38, self(), foe(8, 150, 0))
	s.Walls = []unit.WallView{{X1: 60, Y1: -100, X2: 60, Y2: 100, Radius: 4, VisionBlock: true}}
	a.Handle(ctx, s)
	for _, c := range drain(out) {
		if _, ok := launched(c); ok {
			t.Fatal("wall blocks targeting")
		}
	}
}

func TestProjectileReductionUsesSourceAtOfferTime(t *testing.T) {
	for _, tc := range []struct {
		role, kind       string
		span, time, want float64
	}{
		{unit.RoleProjectile, "手枪弹", 0, .05, 3.5},
		{unit.RoleProjectile, "扇环", 1, .05, 10},
		{unit.RoleProjectile, bloodBullet, 0, .05, 10},
		{unit.RoleHelper, "熔岩", 0, .05, 10},
		{unit.RoleFighter, "射线", 0, .05, 10},
		{unit.RoleProjectile, "手枪弹", 0, .2, 10},
	} {
		a := &闪烁{}
		ctx, out := newCtx()
		a.Handle(ctx, sense(0, self(), approaching()))
		drain(out)
		a.Handle(ctx, unit.IncomingDamage{Token: 1, From: 999, Amount: 10, Time: tc.time, SourceRole: tc.role, SourceKind: tc.kind, ArcSpan: tc.span})
		if got := confirmed(drain(out)); math.Abs(got-tc.want) > 1e-9 {
			t.Fatalf("%+v: got %v", tc, got)
		}
	}
}

func TestCooldownWallStunAndKnockback(t *testing.T) {
	t.Run("cooldown", func(t *testing.T) {
		a := &闪烁{}
		ctx, out := newCtx()
		a.Handle(ctx, sense(0, self(), approaching()))
		drain(out)
		for _, tm := range []float64{.12, .5, 1.39} {
			a.Handle(ctx, sense(tm, self(), approaching()))
			if len(fxNamed(drain(out), "reflex")) != 0 {
				t.Fatal("retrigger before cooldown")
			}
		}
		a.Handle(ctx, sense(1.4, self(), approaching()))
		if len(fxNamed(drain(out), "reflex")) != 1 {
			t.Fatal("ready ability should react")
		}
	})
	t.Run("wall", func(t *testing.T) {
		a := &闪烁{}
		ctx, out := newCtx()
		a.Handle(ctx, sense(0, self(), approaching()))
		drain(out)
		a.Handle(ctx, unit.WallHit{Time: .04, NX: 0, NY: 1})
		drain(out)
		a.Handle(ctx, sense(.05, self()))
		cmds := drain(out)
		v := velocity(cmds)
		if a.phase != phFocus || v == nil || v.VY <= 0 || math.Abs(math.Hypot(v.VX, v.VY)-cruise) > 1e-9 {
			t.Fatalf("wall recovery=%+v phase=%v", v, a.phase)
		}
	})
	t.Run("stun", func(t *testing.T) {
		a := &闪烁{}
		ctx, out := newCtx()
		a.Handle(ctx, sense(0, self(), approaching()))
		drain(out)
		a.Handle(ctx, unit.Stunned{Time: .05, Until: .8})
		cmds := drain(out)
		if p := lastPass(cmds); p == nil || p.Hold {
			t.Fatal("stun must release pass")
		}
		a.Handle(ctx, sense(.8, self(), foe(8, 150, 0)))
		for _, c := range drain(out) {
			if d, ok := launched(c); ok && d.Amount != shotDamage {
				t.Fatal("stun must not replay burst")
			}
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

func TestNonMissileRejectSpendsSide(t *testing.T) {
	owner := foe(8, 0, 120)
	from := foe(9, 120, 0)

	t.Run("reject then slide", func(t *testing.T) {
		a := &闪烁{rng: rngHit(true)}
		ctx, out := newCtx()
		a.Handle(ctx, offer(1, 0, from.ID, owner.ID, unit.RoleProjectile, "扇环", 1))
		cmds := drain(out)
		if !hasBlock(cmds, 1) || confirmed(cmds) >= 0 {
			t.Fatal("must reject the whole packet")
		}
		if math.Abs(a.readyAt-reflexCooldown) > 1e-9 || !a.pending {
			t.Fatal("cooldown starts at the offer")
		}
		a.Handle(ctx, offer(2, 0, from.ID, owner.ID, unit.RoleHelper, "熔岩", 0))
		cmds = drain(out)
		if hasBlock(cmds, 2) || math.Abs(confirmed(cmds)-10) > 1e-9 {
			t.Fatal("a second packet in the same moment must land")
		}
		a.Handle(ctx, sense(0.02, self(), owner, from, approaching()))
		cmds = drain(out)
		v := velocity(cmds)
		if v == nil || math.Abs(v.VY) > 1e-6 || math.Abs(math.Abs(v.VX)-sideSpeed) > 1e-6 {
			t.Fatalf("queued side keeps its heading: %+v", v)
		}
		if p := lastPass(cmds); p == nil || !p.Hold {
			t.Fatal("side must pass")
		}
		if math.Abs(a.readyAt-reflexCooldown) > 1e-9 || a.sourceID != owner.ID {
			t.Fatalf("ready=%v source=%d", a.readyAt, a.sourceID)
		}
	})

	t.Run("miss", func(t *testing.T) {
		a := &闪烁{rng: rngHit(false)}
		ctx, out := newCtx()
		a.Handle(ctx, offer(1, 0, from.ID, owner.ID, unit.RoleProjectile, "扇环", 1))
		cmds := drain(out)
		if hasBlock(cmds, 1) || math.Abs(confirmed(cmds)-10) > 1e-9 || a.pending {
			t.Fatal("a missed roll pays the packet and does not side")
		}
		a.Handle(ctx, sense(0.02, self(), owner))
		if len(fxNamed(drain(out), "reflex")) != 0 {
			t.Fatal("miss must not side on the next sense")
		}
	})

	t.Run("missile", func(t *testing.T) {
		a := &闪烁{rng: rngHit(true)}
		ctx, out := newCtx()
		a.Handle(ctx, offer(1, 0, 2, 8, unit.RoleProjectile, "手枪弹", 0))
		cmds := drain(out)
		if hasBlock(cmds, 1) || math.Abs(confirmed(cmds)-10) > 1e-9 || a.pending {
			t.Fatal("flying shots do not roll")
		}
	})

	t.Run("blood", func(t *testing.T) {
		a := &闪烁{rng: rngHit(true)}
		ctx, out := newCtx()
		a.Handle(ctx, offer(1, 0, 2, 8, unit.RoleProjectile, bloodBullet, 0))
		if !hasBlock(drain(out), 1) {
			t.Fatal("blood is not a flying shot")
		}
	})

	t.Run("body", func(t *testing.T) {
		a := &闪烁{rng: rngHit(true)}
		ctx, out := newCtx()
		a.Handle(ctx, offer(1, 0, from.ID, 0, unit.RoleFighter, "相撞", 0))
		if !hasBlock(drain(out), 1) {
			t.Fatal("body impact must be able to reject")
		}
		a.Handle(ctx, sense(0.02, self(), from))
		v := velocity(drain(out))
		if v == nil || math.Abs(v.VX) > 1e-6 || math.Abs(math.Abs(v.VY)-sideSpeed) > 1e-6 || a.sourceID != from.ID {
			t.Fatalf("no owner uses From: %+v source=%d", v, a.sourceID)
		}
	})

	t.Run("unseen", func(t *testing.T) {
		a := &闪烁{rng: rngHit(true)}
		ctx, out := newCtx()
		a.Handle(ctx, offer(1, 0, from.ID, owner.ID, unit.RoleFighter, "相撞", 0))
		drain(out)
		me := self()
		me.VY = 200
		a.Handle(ctx, sense(0.02, me))
		v := velocity(drain(out))
		if v == nil || math.Abs(v.VY) > 1e-6 || math.Abs(math.Abs(v.VX)-sideSpeed) > 1e-6 || a.sourceID != owner.ID {
			t.Fatalf("unseen owner follows velocity: %+v source=%d", v, a.sourceID)
		}
	})

	t.Run("still", func(t *testing.T) {
		a := &闪烁{rng: rngHit(true)}
		ctx, out := newCtx()
		a.Handle(ctx, offer(1, 0, 0, 0, unit.RoleHelper, "熔岩", 0))
		drain(out)
		a.Handle(ctx, sense(0.02, self()))
		v := velocity(drain(out))
		if v == nil || math.Abs(v.VX) > 1e-6 || v.VY <= 0 {
			t.Fatalf("still body steps off +X: %+v", v)
		}
	})

	t.Run("cramped", func(t *testing.T) {
		a := &闪烁{rng: rngHit(true)}
		ctx, out := newCtx()
		a.Handle(ctx, offer(1, 0, 0, owner.ID, unit.RoleHelper, "熔岩", 0))
		drain(out)
		s := sense(0.02, self(), foe(owner.ID, 90, 0))
		s.Walls = []unit.WallView{
			{X1: -80, Y1: 22, X2: 80, Y2: 22, Radius: 1},
			{X1: -80, Y1: -22, X2: 80, Y2: -22, Radius: 1},
		}
		a.Handle(ctx, s)
		cmds := drain(out)
		if a.phase != phFocus || velocity(cmds) != nil || lastPass(cmds) != nil {
			t.Fatal("no room opens the counter without moving or passing")
		}
		a.Handle(ctx, sense(0.07, self(), foe(owner.ID, 90, 0)))
		found := false
		for _, c := range drain(out) {
			if d, ok := launched(c); ok {
				found = true
				if d.To != owner.ID || d.Amount != beamDamage {
					t.Fatalf("counter=%+v", d)
				}
			}
		}
		if !found {
			t.Fatal("cramped reject still fires")
		}
	})

	t.Run("stun before slide", func(t *testing.T) {
		a := &闪烁{rng: rngHit(true)}
		ctx, out := newCtx()
		a.Handle(ctx, offer(1, 0, from.ID, owner.ID, unit.RoleProjectile, "扇环", 1))
		drain(out)
		a.Handle(ctx, unit.Stunned{Time: 0.01, Until: 0.4})
		drain(out)
		a.Handle(ctx, sense(0.05, self(), owner))
		cmds := drain(out)
		if a.phase != phIdle || a.pending || len(fxNamed(cmds, "reflex")) != 0 || velocity(cmds) != nil {
			t.Fatal("stun before the slide cancels the move and the counter")
		}
		if math.Abs(a.readyAt-reflexCooldown) > 1e-9 {
			t.Fatal("cooldown stays spent")
		}
	})

	t.Run("death before slide", func(t *testing.T) {
		a := &闪烁{rng: rngHit(true)}
		ctx, out := newCtx()
		a.Handle(ctx, offer(1, 0, 0, owner.ID, unit.RoleHelper, "熔岩", 0))
		drain(out)
		me := self()
		me.HP = 0
		a.Handle(ctx, sense(0.02, me, owner))
		if a.phase != phIdle || a.pending || len(fxNamed(drain(out), "reflex")) != 0 || math.Abs(a.readyAt-reflexCooldown) > 1e-9 {
			t.Fatal("death before the slide cancels it and keeps the cooldown")
		}
	})

	t.Run("knock", func(t *testing.T) {
		a := &闪烁{rng: rngHit(true)}
		ctx, out := newCtx()
		a.Handle(ctx, offer(1, 0, 0, owner.ID, unit.RoleHelper, "熔岩", 0))
		drain(out)
		a.Handle(ctx, senseKnock(0.02, self(), owner))
		cmds := drain(out)
		if a.phase != phSide || velocity(cmds) != nil {
			t.Fatal("knock must not cancel or rewrite the slide")
		}
		if p := lastPass(cmds); p == nil || !p.Hold {
			t.Fatal("knock keeps the slide's pass")
		}
		a.Handle(ctx, sense(0.05, self(), owner))
		v := velocity(drain(out))
		if v == nil || math.Abs(math.Abs(v.VX)-sideSpeed) > 1e-6 {
			t.Fatalf("slide resumes after knock: %+v", v)
		}
	})

	t.Run("busy", func(t *testing.T) {
		a := &闪烁{rng: rngHit(true)}
		ctx, out := newCtx()
		a.Handle(ctx, sense(0, self(), approaching()))
		drain(out)
		a.Handle(ctx, offer(1, 0.05, from.ID, owner.ID, unit.RoleProjectile, "扇环", 1))
		cmds := drain(out)
		if hasBlock(cmds, 1) || math.Abs(confirmed(cmds)-10) > 1e-9 || len(fxNamed(cmds, "reflex")) != 0 {
			t.Fatal("side already running does not roll")
		}
		a.Handle(ctx, sense(0.12, self(), owner))
		drain(out)
		if a.phase != phFocus {
			t.Fatal("expected the counter window")
		}
		a.Handle(ctx, offer(2, 0.2, from.ID, owner.ID, unit.RoleProjectile, "扇环", 1))
		cmds = drain(out)
		if hasBlock(cmds, 2) || math.Abs(confirmed(cmds)-10) > 1e-9 {
			t.Fatal("counter window does not roll")
		}
	})

	t.Run("cooldown", func(t *testing.T) {
		a := &闪烁{rng: rngHit(true)}
		ctx, out := newCtx()
		a.Handle(ctx, offer(1, 0, 0, owner.ID, unit.RoleHelper, "熔岩", 0))
		drain(out)
		a.Handle(ctx, sense(0.02, self(), owner))
		drain(out)
		a.Handle(ctx, sense(0.6, self(), owner))
		drain(out)
		if a.phase != phIdle {
			t.Fatalf("phase=%v", a.phase)
		}
		a.rng = rngHit(true)
		a.Handle(ctx, offer(2, 1.0, 0, owner.ID, unit.RoleHelper, "熔岩", 0))
		cmds := drain(out)
		if hasBlock(cmds, 2) || math.Abs(confirmed(cmds)-10) > 1e-9 {
			t.Fatal("cooldown must still be running")
		}
		a.rng = rngHit(true)
		a.Handle(ctx, offer(3, 1.4, 0, owner.ID, unit.RoleHelper, "熔岩", 0))
		if !hasBlock(drain(out), 3) {
			t.Fatal("ready again at 1.4")
		}
	})

	t.Run("perfect", func(t *testing.T) {
		a := &闪烁{rng: rngHit(true)}
		ctx, out := newCtx()
		a.Handle(ctx, offer(1, 0, 0, owner.ID, unit.RoleHelper, "熔岩", 0))
		drain(out)
		var shots []unit.Damage
		for _, tm := range []float64{0.02, 0.14, 0.18, 0.28, 0.38} {
			a.Handle(ctx, sense(tm, self(), owner))
			for _, c := range drain(out) {
				if d, ok := launched(c); ok {
					shots = append(shots, d)
				}
			}
		}
		if len(shots) != 3 || shots[2].To != owner.ID || shots[2].Amount != perfectDamage {
			t.Fatalf("rejected packet keeps the clean counter: %+v", shots)
		}
	})
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

func hasBlock(cmds []unit.Cmd, token uint64) bool {
	for _, c := range cmds {
		if b, ok := c.(unit.BlockDamage); ok && b.Token == token {
			return true
		}
	}
	return false
}

func rngHit(hit bool) *rand.Rand {
	for stream := uint64(1); stream < 20000; stream++ {
		r := unit.NewRNG(stream)
		if (r.Float64() < nonMissileReject) == hit {
			return unit.NewRNG(stream)
		}
	}
	panic("no rng stream")
}
