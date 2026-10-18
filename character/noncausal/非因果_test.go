package 非因果

import (
	"encoding/json"
	"math"
	"testing"

	_ "xqdj/character/dummy"
	"xqdj/internal/sim"
	"xqdj/internal/unit"
)

func TestSpectrumTable(t *testing.T) {
	cases := []struct {
		name    string
		in      []pulse
		cadence float64
		bins    []bin
	}{
		{
			name:    "mean interval and histogram",
			in:      []pulse{{0, 4}, {2, 4}, {3, 8}},
			cadence: 1.5,
			bins:    []bin{{4, 2}, {8, 1}},
		},
		{
			name:    "unsorted pulses",
			in:      []pulse{{3, 8}, {0, 4}, {2, 4}},
			cadence: 1.5,
			bins:    []bin{{4, 2}, {8, 1}},
		},
		{
			name:    "single pulse uses fallback cadence",
			in:      []pulse{{1, 3}},
			cadence: fallbackCadence,
			bins:    []bin{{3, 1}},
		},
		{
			name:    "empty probe",
			in:      nil,
			cadence: fallbackCadence,
			bins:    []bin{{fallbackDamage, 1}},
		},
		{
			name:    "simultaneous hits clamp to one tick",
			in:      []pulse{{0, 5}, {0, 5}},
			cadence: tickDT,
			bins:    []bin{{5, 2}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildSpectrum(tc.in)
			if math.Abs(got.cadence-tc.cadence) > 1e-9 {
				t.Fatalf("cadence %v want %v", got.cadence, tc.cadence)
			}
			if len(got.bins) != len(tc.bins) {
				t.Fatalf("bins %v want %v", got.bins, tc.bins)
			}
			for i := range tc.bins {
				if got.bins[i] != tc.bins[i] {
					t.Fatalf("bin %d %v want %v", i, got.bins[i], tc.bins[i])
				}
			}
		})
	}
}

func TestDrawStaysInHistogramAndReplays(t *testing.T) {
	sp := buildSpectrum([]pulse{{0, 5}, {1, 5}, {2, 5}, {3, 9}})
	heavy, light := 0, 0
	rng := unit.NewRNG(12345)
	for i := 0; i < 4000; i++ {
		_, amt := sp.draw(rng)
		switch amt {
		case 5:
			heavy++
		case 9:
			light++
		default:
			t.Fatalf("draw %v outside histogram", amt)
		}
	}
	if heavy < light*2 {
		t.Fatalf("weight 3:1 not visible heavy=%d light=%d", heavy, light)
	}
	a := unit.NewRNG(7)
	b := unit.NewRNG(7)
	for i := 0; i < 24; i++ {
		ia, aa := sp.draw(a)
		ib, ab := sp.draw(b)
		if ia != ib || aa != ab {
			t.Fatalf("stream 7 diverged at %d: (%d,%v) (%d,%v)", i, ia, aa, ib, ab)
		}
	}
	c := unit.NewRNG(8)
	same := true
	a = unit.NewRNG(7)
	for i := 0; i < 24; i++ {
		_, aa := sp.draw(a)
		_, cc := sp.draw(c)
		if aa != cc {
			same = false
			break
		}
	}
	if same {
		t.Fatal("different streams stayed in lockstep")
	}
}

func TestReplayIndex(t *testing.T) {
	path := []sample{{t: 0, x: 0}, {t: 1, x: 10}, {t: 2, x: 20}}
	cases := []struct {
		key float64
		i   int
	}{
		{0, 0},
		{0.5, 0},
		{1, 1},
		{1.5, 1},
		{2, 2},
		{9, 2},
	}
	for _, tc := range cases {
		if got := atIndex(path, 0, tc.key); got != tc.i {
			t.Fatalf("key %v index %d want %d", tc.key, got, tc.i)
		}
	}
}

func TestSteerTurnBudget(t *testing.T) {
	const speed = 50.0
	cases := []struct {
		name     string
		vx, vy   float64
		dx, dy   float64
		turn, dt float64
		want     float64
	}{
		{"clamp", 1, 0, 0, 1, 1, 0.2, 0.2},
		{"snap", 1, 0, 0, 1, 10, 1, math.Pi / 2},
		{"hold", 1, 0, 3, 0, 2, 0.5, 0},
		{"other way", 1, 0, 0, -1, 1, 0.2, -0.2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vx, vy := steer(tc.vx, tc.vy, tc.dx, tc.dy, tc.turn, speed, tc.dt)
			if math.Abs(math.Hypot(vx, vy)-speed) > 1e-6 {
				t.Fatalf("speed %v", math.Hypot(vx, vy))
			}
			if d := wrapPi(math.Atan2(vy, vx) - tc.want); math.Abs(d) > 1e-6 {
				t.Fatalf("angle %v want %v", math.Atan2(vy, vx), tc.want)
			}
		})
	}
}

func TestProbeRecordsThenOpensSpectrum(t *testing.T) {
	f, ctx, out := newFighter(1, 11)
	enemy := fighterSnap(2, 1, 200, 0, 80)
	f.Handle(ctx, senseAt(0, selfSnap(1, 0, 0, 0, 100, 90, 0), enemy))
	f.Handle(ctx, unit.IncomingDamage{Token: 1, Amount: 20, Time: 0.5})
	f.Handle(ctx, unit.IncomingDamage{Token: 2, Amount: 20, Time: 1})
	mid := drain(out)
	if !allBlock(mid) || hasConfirm(mid) || hasFX(mid, fxLockShot) || hasFX(mid, fxSpectrum) {
		t.Fatalf("probe should only block and record: %v", mid)
	}
	if len(f.b.pulses) != 2 || f.b.sum != 40 {
		t.Fatalf("pulses %+v sum %v", f.b.pulses, f.b.sum)
	}
	f.Handle(ctx, unit.IncomingDamage{Token: 3, Amount: 60, Time: 1.5})
	f.Handle(ctx, senseAt(1.5, selfSnap(1, 0, 0, 0, 100, 90, 0), fighterSnap(2, 1, 150, 40, 80)))
	cmds := drain(out)
	if !hasFX(cmds, fxSpectrum) || !hasFX(cmds, fxDoomAt) {
		t.Fatalf("real phase did not sync spectrum: %v", cmds)
	}
	spec := fxByName(cmds, fxSpectrum)
	if math.Abs(spec.Amount-0.5) > 1e-9 || spec.VX != 2 || spec.VY != 3 {
		t.Fatalf("spectrum fx %+v", spec)
	}
	bins := fxsByName(cmds, fxBin)
	if len(bins) != 2 || bins[0].Slot != 0 || bins[0].Amount != 20 || bins[0].VX != 2 ||
		bins[1].Slot != 1 || bins[1].Amount != 60 || bins[1].VX != 1 {
		t.Fatalf("bins %+v", bins)
	}
	doom := fxByName(cmds, fxDoomAt)
	if math.Abs(doom.Amount-1.5) > 1e-9 || doom.VX != 80 {
		t.Fatalf("doom-at %+v", doom)
	}
	if !hasSpawn(cmds, KindResidual) || !hasSpawn(cmds, KindLock) {
		t.Fatalf("missing residual or first shot: %v", cmds)
	}
	shot := fxByName(cmds, fxLockShot)
	if shot.Amount != 20 && shot.Amount != 60 {
		t.Fatalf("drawn %v not in histogram", shot.Amount)
	}
	if shot.Slot != 0 && shot.Slot != 1 {
		t.Fatalf("binIndex %d", shot.Slot)
	}
	vx, vy := reversed(cmds)
	if math.Abs(vx-(-90)) > 1e-9 || math.Abs(vy) > 1e-9 {
		t.Fatalf("heading (%v,%v) want (-90,0)", vx, vy)
	}
	if f.b.ghostMax != 80 || f.b.ghostHP != 80 || math.Abs(f.b.spec.cadence-0.5) > 1e-9 {
		t.Fatalf("ghost %v/%v cadence %v", f.b.ghostHP, f.b.ghostMax, f.b.spec.cadence)
	}
	f.Handle(ctx, unit.IncomingDamage{Token: 4, Amount: 5, Time: 1.7})
	if !hasConfirm(drain(out)) || len(f.b.pulses) != 3 {
		t.Fatal("real hit should confirm and not extend the probe histogram")
	}
}

func TestMissileBlockedRefiresOnCadence(t *testing.T) {
	f, ctx, out := newFighter(3, 5)
	self := selfSnap(3, 0, 0, 0, 100, 80, 0)
	f.Handle(ctx, senseAt(0, self, fighterSnap(4, 1, 220, 0, 100)))
	f.Handle(ctx, unit.IncomingDamage{Token: 1, Amount: 20, Time: 0.5})
	f.Handle(ctx, unit.IncomingDamage{Token: 2, Amount: 20, Time: 1})
	f.Handle(ctx, unit.IncomingDamage{Token: 3, Amount: 60, Time: 1.5})
	f.Handle(ctx, senseAt(1.5, self, fighterSnap(4, 1, 180, 10, 100)))
	if !hasSpawn(drain(out), KindLock) {
		t.Fatal("expected the opening shot")
	}
	if math.Abs(f.b.spec.cadence-0.5) > 1e-9 {
		t.Fatalf("cadence %v", f.b.spec.cadence)
	}
	mctx, mout := newCtx(30, KindLock)
	miss := &锁影弹{owner: 3}
	miss.Handle(mctx, unit.Collision{
		Time: 1.75,
		Other: unit.Snapshot{
			ID: 8, Role: unit.RoleMinion, Slot: 1, Mortal: true,
			X: 40, Y: 0, Radius: 10,
		},
	})
	blocked := drain(mout)
	if !hasFX(blocked, fxBlock) || hasFX(blocked, fxGhostHit) || !hasDespawn(blocked, 30) {
		t.Fatalf("block should destroy the missile: %v", blocked)
	}
	block := fxByName(blocked, fxBlock)
	if block.VX != 8 || block.VY != 1 || (block.Slot != 0 && block.Slot != 1) {
		t.Fatalf("block fx %+v", block)
	}
	if f.b.ghostHP != 100 || f.b.live || math.Abs(f.b.nextFire-2.25) > 1e-9 {
		t.Fatalf("after block hp %v live %v next %v", f.b.ghostHP, f.b.live, f.b.nextFire)
	}
	f.Handle(ctx, senseAt(2, self, fighterSnap(4, 1, 160, 10, 100)))
	if hasSpawn(drain(out), KindLock) {
		t.Fatal("re-fired before the cadence measured from the block")
	}
	f.Handle(ctx, senseAt(2.25, self, fighterSnap(4, 1, 140, 10, 100)))
	again := drain(out)
	if !hasSpawn(again, KindLock) {
		t.Fatalf("expected re-fire on cadence: %v", again)
	}
	shot := fxByName(again, fxLockShot)
	if shot.Amount != 20 && shot.Amount != 60 {
		t.Fatalf("re-fire draw %v", shot.Amount)
	}
	if math.Abs(f.b.spec.cadence-0.5) > 1e-9 {
		t.Fatal("cadence changed between shots")
	}
}

func TestGhostHitDepletesAndBodyDoesNot(t *testing.T) {
	f, ctx, out := newFighter(5, 9)
	self := selfSnap(5, 0, 0, 0, 10, 0, 0)
	enemy := fighterSnap(7, 1, 120, 0, 10)
	f.Handle(ctx, senseAt(0, self, enemy))
	f.Handle(ctx, unit.IncomingDamage{Token: 1, Amount: 10, Time: 1})
	f.Handle(ctx, senseAt(1, self, enemy))
	if !hasSpawn(drain(out), KindLock) || f.b.ghostHP != 10 {
		t.Fatalf("ghost hp %v", f.b.ghostHP)
	}

	body, bout := newCtx(40, KindLock)
	miss := &锁影弹{owner: 5}
	miss.Handle(body, unit.Collision{
		Time: 1.1,
		Other: unit.Snapshot{
			ID: 7, Role: unit.RoleFighter, Slot: 1, X: 30, Y: 0, Radius: 18,
		},
	})
	got := drain(bout)
	if hasFX(got, fxGhostHit) || hasDespawn(got, 7) || hasDamage(got) {
		t.Fatalf("real body must not spend GhostHP: %v", got)
	}
	if f.b.ghostHP != 10 {
		t.Fatalf("ghost hp after body %v", f.b.ghostHP)
	}

	// 上一发被真身挡掉，等射速再开一发，把残影打光。
	f.Handle(ctx, senseAt(1.1+fallbackCadence, self, enemy))
	if !hasSpawn(drain(out), KindLock) {
		t.Fatal("expected the shot that reaches the residual")
	}
	gctx, gout := newCtx(41, KindLock)
	ghost := &锁影弹{owner: 5}
	ghost.Handle(gctx, unit.Sense{
		Time: 1.1 + fallbackCadence,
		Self: unit.Snapshot{ID: 41, X: 120, Y: 0, Radius: missileRadius, VX: missileSpeed},
		Nearby: []unit.Snapshot{{
			ID: 9, Kind: KindResidual, OwnerID: 5, Role: unit.RoleHelper,
			X: 120, Y: 0, Radius: residualRadius, Nonsolid: true,
		}},
	})
	hit := drain(gout)
	gh := fxByName(hit, fxGhostHit)
	if gh.Amount != 10 || gh.Slot != 0 || gh.VX != 0 || gh.UnitID != 9 {
		t.Fatalf("ghost-hit %+v", gh)
	}
	down := fxByName(hit, fxGhostDown)
	if down.VX != 7 || down.VY != 10 || down.Amount != 10 {
		t.Fatalf("ghost-down %+v", down)
	}
	if !hasDespawn(hit, 7) || hasDamage(hit) || f.b.ghostHP != 0 || !f.b.won {
		t.Fatalf("win settle hp %v won %v cmds %v", f.b.ghostHP, f.b.won, hit)
	}
	f.Handle(ctx, senseAt(1+10, self, enemy))
	late := drain(out)
	if hasDespawn(late, 5) || hasSpawn(late, KindLock) {
		t.Fatalf("a ghost win must not suicide on the clock or keep firing: %v", late)
	}
}

func TestClockDespawnsSelf(t *testing.T) {
	f, ctx, out := newFighter(6, 1)
	self := selfSnap(6, 0, 10, 0, 10, 0, 0)
	enemy := fighterSnap(8, 1, 40, 0, 50)
	f.Handle(ctx, senseAt(0, self, enemy))
	f.Handle(ctx, unit.IncomingDamage{Token: 1, Amount: 10, Time: 2})
	f.Handle(ctx, senseAt(2, self, enemy))
	drain(out)
	f.Handle(ctx, senseAt(3.9, self, enemy))
	if hasDespawn(drain(out), 6) {
		t.Fatal("clock early")
	}
	f.Handle(ctx, senseAt(4, self, enemy))
	cmds := drain(out)
	doom := fxByName(cmds, fxDoom)
	if math.Abs(doom.Amount-2) > 1e-9 || !hasDespawn(cmds, 6) {
		t.Fatalf("clock loss cmds %v", cmds)
	}
}

func TestMatchOpensRealPhase(t *testing.T) {
	m := sim.NewMatchSeeded(11)
	m.SetSlot(0, KindNoncausal)
	m.SetSlot(1, "靶子")
	m.Start()
	t.Cleanup(m.End)
	var sawSpectrum, sawBin, sawResidual bool
	var cadence, binAmt, doomAt, ghostMax float64
	limit := int(probeCap*60) + 40
	for i := 0; i < limit; i++ {
		m.Tick()
		var msg struct {
			Phase   string    `json:"phase"`
			Effects []unit.FX `json:"effects"`
			Units   []struct {
				Kind string `json:"kind"`
			} `json:"units"`
		}
		if err := json.Unmarshal(m.SnapshotJSON(), &msg); err != nil {
			t.Fatal(err)
		}
		if msg.Phase != "running" {
			t.Fatalf("phase %s at tick %d", msg.Phase, i)
		}
		for _, u := range msg.Units {
			if u.Kind == KindResidual {
				sawResidual = true
			}
		}
		for _, fx := range msg.Effects {
			if fx.Name == fxSpectrum {
				sawSpectrum = true
				cadence = fx.Amount
			}
			if fx.Name == fxBin {
				sawBin = true
				binAmt = fx.Amount
			}
			if fx.Name == fxDoomAt {
				doomAt = fx.Amount
				ghostMax = fx.VX
			}
		}
	}
	if !sawSpectrum || !sawBin || !sawResidual {
		t.Fatalf("spectrum %v bin %v residual %v", sawSpectrum, sawBin, sawResidual)
	}
	if math.Abs(cadence-fallbackCadence) > 1e-9 || binAmt != fallbackDamage {
		t.Fatalf("cadence %v bin %v", cadence, binAmt)
	}
	if math.Abs(doomAt-probeCap) > 1e-6 || ghostMax != 99999 {
		t.Fatalf("doom %v ghostMax %v", doomAt, ghostMax)
	}
}

func TestMatchBootsAgainstDummy(t *testing.T) {
	m := sim.NewMatchSeeded(7)
	m.SetSlot(0, KindNoncausal)
	m.SetSlot(1, "靶子")
	m.Start()
	t.Cleanup(m.End)
	for i := 0; i < 90; i++ {
		m.Tick()
	}
	var msg struct {
		Phase string `json:"phase"`
		Units []struct {
			Kind string  `json:"kind"`
			HP   float64 `json:"hp"`
			Role string  `json:"role"`
		} `json:"units"`
	}
	if err := json.Unmarshal(m.SnapshotJSON(), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Phase != "running" {
		t.Fatalf("phase %s", msg.Phase)
	}
	var seen bool
	for _, u := range msg.Units {
		if u.Kind == KindNoncausal && u.Role == "fighter" {
			seen = true
			if u.HP != bodyHP {
				t.Fatalf("probe dropped real hp to %v", u.HP)
			}
		}
	}
	if !seen {
		t.Fatal("非因果 missing from the match")
	}
}

func newFighter(id, stream uint64) (*非因果, unit.Context, <-chan unit.Cmd) {
	f := &非因果{stream: stream, rng: unit.NewRNG(stream)}
	ctx, out := newCtx(id, KindNoncausal)
	ctx.Stream = stream
	return f, ctx, out
}

func newCtx(id uint64, kind string) (unit.Context, <-chan unit.Cmd) {
	out := make(chan unit.Cmd, 64)
	return unit.Context{ID: id, Kind: kind, Out: out, Stream: 1}.WithBirths(), out
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

func selfSnap(id uint64, slot int, x, y, maxHP, vx, vy float64) unit.Snapshot {
	return unit.Snapshot{
		ID: id, Kind: KindNoncausal, Role: unit.RoleFighter, Slot: slot,
		X: x, Y: y, VX: vx, VY: vy, Radius: bodyRadius, HP: maxHP, MaxHP: maxHP,
	}
}

func fighterSnap(id uint64, slot int, x, y, maxHP float64) unit.Snapshot {
	return unit.Snapshot{
		ID: id, Role: unit.RoleFighter, Slot: slot,
		X: x, Y: y, Radius: 18, HP: maxHP, MaxHP: maxHP,
		AimPriority: unit.DefaultFighterAim,
	}
}

func senseAt(t float64, self unit.Snapshot, nearby ...unit.Snapshot) unit.Sense {
	return unit.Sense{Time: t, Self: self, Nearby: nearby}
}

func hasFX(cmds []unit.Cmd, name string) bool {
	return fxByName(cmds, name).Name == name
}

func fxByName(cmds []unit.Cmd, name string) unit.FX {
	all := fxsByName(cmds, name)
	if len(all) == 0 {
		return unit.FX{}
	}
	return all[0]
}

func fxsByName(cmds []unit.Cmd, name string) []unit.FX {
	var out []unit.FX
	for _, c := range cmds {
		fx, ok := c.(unit.FX)
		if ok && fx.Name == name {
			out = append(out, fx)
		}
	}
	return out
}

func hasSpawn(cmds []unit.Cmd, kind string) bool {
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == kind {
			return true
		}
	}
	return false
}

func hasDespawn(cmds []unit.Cmd, id uint64) bool {
	for _, c := range cmds {
		if d, ok := c.(unit.Despawn); ok && d.UnitID == id {
			return true
		}
	}
	return false
}

func hasDamage(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if _, ok := c.(unit.Damage); ok {
			return true
		}
	}
	return false
}

func hasConfirm(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if _, ok := c.(unit.ConfirmDamage); ok {
			return true
		}
	}
	return false
}

func allBlock(cmds []unit.Cmd) bool {
	n := 0
	for _, c := range cmds {
		if _, ok := c.(unit.BlockDamage); ok {
			n++
		}
	}
	return n > 0
}

func reversed(cmds []unit.Cmd) (float64, float64) {
	for _, c := range cmds {
		if v, ok := c.(unit.SetVelocity); ok {
			return v.VX, v.VY
		}
	}
	return 0, 0
}
