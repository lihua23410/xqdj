package 观察者

import (
	"math"
	"testing"

	tanyongchen "xqdj/character/tanyongchen"
	"xqdj/internal/unit"
)

func TestObserverSkipsMeleeArcsAndRecordsTanKunaiProxy(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	o := &observer{slot: 0, x: 0, y: 0, projectileSeen: map[uint64]bool{}}
	arc := unit.Snapshot{ID: 9, Slot: 1, Role: unit.RoleProjectile, Kind: "小骑士弧", ArcSpan: unit.Deg(120), X: 40, Radius: 20}
	kunai := unit.Snapshot{ID: 10, Slot: 1, Role: unit.RoleProjectile, Kind: tanyongchen.KindTanKunaiShot, X: 60, Radius: 1}
	o.recordProjectiles(unit.Context{ID: 1, Kind: KindObserver, Out: out}, unit.Sense{Self: unit.Snapshot{ID: 1, Slot: 0}, Nearby: []unit.Snapshot{arc, kunai}})
	if len(o.records) != 1 || o.records[0] != tanyongchen.KindTanKunaiShot {
		t.Fatalf("records=%v, want only Tan kunai proxy", o.records)
	}
}

func TestObserverReplaysTanKunaiAsDamageCopy(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	o := &observer{
		rng: unit.NewRNG(1), gauge: gaugeMax, targetID: 2,
		records: []string{tanyongchen.KindTanKunaiShot},
	}
	ctx := unit.Context{ID: 1, Kind: KindObserver, Out: out}
	self := unit.Snapshot{ID: 1, Slot: 0, Role: unit.RoleFighter, Radius: observerRadius}
	target := unit.Snapshot{ID: 2, Slot: 1, Role: unit.RoleFighter, X: 100, Radius: 18, AimPriority: 15}
	o.Handle(ctx, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{target}})
	for _, cmd := range drain(out) {
		if spawn, ok := cmd.(unit.Spawn); ok && spawn.Kind == KindObserverKunaiCopy {
			if math.Abs(spawn.DamageScale-replayScale) > 1e-9 {
				t.Fatalf("kunai copy damage scale=%v, want %v", spawn.DamageScale, replayScale)
			}
			return
		}
	}
	t.Fatal("Tan kunai was not mapped to observer kunai copy")
}

func TestObserverCanObserveMortalFighterProxyAndKeepsLock(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	o := &observer{rng: unit.NewRNG(1)}
	ctx := unit.Context{ID: 1, Kind: KindObserver, Out: out}
	self := unit.Snapshot{ID: 1, Slot: 0, Role: unit.RoleFighter, Radius: observerRadius}
	red := unit.Snapshot{ID: 2, OwnerID: 99, Slot: 1, Role: unit.RoleMinion, Mortal: true, AimPriority: 15, X: 100, Radius: 18}
	blue := unit.Snapshot{ID: 3, OwnerID: 99, Slot: 1, Role: unit.RoleMinion, Mortal: true, AimPriority: 15, X: 180, Radius: 18}
	o.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{red, blue}})
	_ = drain(out)
	if o.targetID != 2 {
		t.Fatalf("initial target=%d, want red half 2", o.targetID)
	}
	red.X, blue.X = 180, 100
	o.Handle(ctx, unit.Sense{Time: 0.5, Self: self, Nearby: []unit.Snapshot{red, blue}})
	if o.targetID != 2 {
		t.Fatalf("target switched to closer enemy %d, want stable lock 2", o.targetID)
	}
	red.ID = 4
	o.targetID = 4
	o.Handle(ctx, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{blue}})
	if o.targetID != 3 || o.gauge == 0 {
		t.Fatalf("same-owner half switch should keep observation, target=%d gauge=%v", o.targetID, o.gauge)
	}
}

func TestObserverGainsGaugeAndRecordsEnemyProjectile(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	o := &observer{rng: unit.NewRNG(1)}
	ctx := unit.Context{ID: 1, Kind: KindObserver, Out: out}
	self := unit.Snapshot{ID: 1, Slot: 0, Role: unit.RoleFighter, X: 0, Y: 0, Radius: observerRadius}
	target := unit.Snapshot{ID: 2, Slot: 1, Role: unit.RoleFighter, X: 120, Y: 0, Radius: 18, AimPriority: 15}
	o.Handle(ctx, unit.Sense{Time: 0.9, Self: self, Nearby: []unit.Snapshot{target}})
	_ = drain(out)
	proj := unit.Snapshot{ID: 9, Slot: 1, Role: unit.RoleProjectile, Kind: "靶子子弹", X: 80, Y: 20, Radius: 5, VX: 100}
	o.Handle(ctx, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{target, proj}})
	cmds := drain(out)
	if math.Abs(o.gauge-6) > 1e-6 {
		t.Fatalf("gauge=%v, want 0.1s of observation plus 5 for the recorded shot", o.gauge)
	}
	if len(o.records) != 1 || o.records[0] != "靶子子弹" {
		t.Fatalf("records=%v, want recorded projectile", o.records)
	}
	if !hasFX(cmds, "observer-record") {
		t.Fatal("observer-record FX missing")
	}
}

func TestObserverFiresAfterSevenAndHalfSecondsOfObservation(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	o := &observer{}
	ctx := unit.Context{ID: 1, Kind: KindObserver, Out: out}
	self := unit.Snapshot{ID: 1, Slot: 0, Role: unit.RoleFighter, Radius: observerRadius}
	target := unit.Snapshot{ID: 2, Slot: 1, Role: unit.RoleFighter, X: 100, AimPriority: 15}
	for step := 0; step <= 60; step++ {
		now := float64(step) / 8 // 0.125 秒一拍，避免浮点累计影响临界时刻。
		o.Handle(ctx, unit.Sense{Time: now, Self: self, Nearby: []unit.Snapshot{target}})
		cmds := drain(out)
		if fired := hasSpawn(cmds, KindObserverBeam); fired != (step == 60) {
			t.Fatalf("time=%v fired=%v gauge=%v, want first beam at 7.5s", now, fired, o.gauge)
		}
	}
	if o.gauge != 0 {
		t.Fatalf("gauge=%v, want reset after firing", o.gauge)
	}
}

func TestObserverStateReportsGaugePercentage(t *testing.T) {
	for _, tc := range []struct{ gauge, percent float64 }{{0, 0}, {37.5, 50}, {75, 100}} {
		out := make(chan unit.Cmd, 2)
		o := &observer{gauge: tc.gauge}
		o.emitState(unit.Context{ID: 1, Kind: KindObserver, Out: out}, stateGap)
		cmds := drain(out)
		if len(cmds) != 2 {
			t.Fatalf("gauge=%v commands=%v, want gauge and film states", tc.gauge, cmds)
		}
		fx, ok := cmds[0].(unit.FX)
		if !ok || fx.Name != "observer-state" || math.Abs(fx.Amount-tc.percent) > 1e-9 {
			t.Fatalf("gauge=%v state=%+v, want %v%%", tc.gauge, fx, tc.percent)
		}
	}
}

func TestObserverFilmStateReportsOnlyArchivedShotsAndPendingReplay(t *testing.T) {
	out := make(chan unit.Cmd, 2)
	o := &observer{
		film:   []shotNote{{KindObserverBeam, 1}, {KindObserverKunaiCopy, .85}, {KindObserverBeam, .85}},
		fired:  []shotNote{{KindObserverBeam, .72}},
		replay: []shotNote{{KindObserverBeam, .61}, {KindObserverKunaiCopy, .72}},
	}
	o.emitState(unit.Context{ID: 7, Kind: KindObserver, Out: out}, stateGap)
	for _, cmd := range drain(out) {
		if fx, ok := cmd.(unit.FX); ok && fx.Name == "observer-film-state" {
			if fx.UnitID != 7 || fx.Amount != 3 || fx.VX != 4 || fx.VY != 2 {
				t.Fatalf("film state=%+v, want 3 archived beams, 4 archived shots and 2 pending shots", fx)
			}
			return
		}
	}
	t.Fatal("missing film state")
}

func TestObserverFilmStateIncludesOrdinaryShotsWithoutBeams(t *testing.T) {
	out := make(chan unit.Cmd, 2)
	o := &observer{
		film:   []shotNote{{KindObserverKunaiCopy, .85}, {KindObserverKunaiCopy, .72}, {KindObserverKunaiCopy, .61}},
		fired:  []shotNote{{KindObserverKunaiCopy, .85}, {KindObserverKunaiCopy, .85}},
		replay: []shotNote{{KindObserverKunaiCopy, .85}},
	}
	o.emitState(unit.Context{ID: 7, Kind: KindObserver, Out: out}, stateGap)
	for _, cmd := range drain(out) {
		if fx, ok := cmd.(unit.FX); ok && fx.Name == "observer-film-state" {
			if fx.Amount != 0 || fx.VX != 5 || fx.VY != 1 {
				t.Fatalf("film state=%+v, want 0 beams, 5 archived ordinary shots and 1 pending shot", fx)
			}
			return
		}
	}
	t.Fatal("missing film state")
}

func TestObserverReplaysRecordedSequenceAtScaledDamage(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	want := []string{KindObserverBeam, KindObserverKunaiCopy, KindObserverBeam}
	o := &observer{
		rng: unit.NewRNG(1), gauge: gaugeMax, targetID: 2,
		kinds:   []string{KindObserverBeam, tanyongchen.KindTanKunaiShot},
		records: []string{KindObserverBeam, tanyongchen.KindTanKunaiShot, KindObserverBeam},
	}
	ctx := unit.Context{ID: 1, Kind: KindObserver, Out: out}
	self := unit.Snapshot{ID: 1, Slot: 0, Role: unit.RoleFighter, X: 0, Y: 0, Radius: observerRadius}
	target := unit.Snapshot{ID: 2, Slot: 1, Role: unit.RoleFighter, X: 100, Y: 0, Radius: 18, AimPriority: 15}
	var got []string
	for i := 0; i < len(want); i++ {
		o.Handle(ctx, unit.Sense{Time: 1 + float64(i)*replayGap, Self: self, Nearby: []unit.Snapshot{target}})
		for _, cmd := range drain(out) {
			spawn, ok := cmd.(unit.Spawn)
			if !ok || (spawn.Kind != KindObserverBeam && spawn.Kind != KindObserverKunaiCopy) {
				continue
			}
			got = append(got, spawn.Kind)
			if math.Abs(spawn.DamageScale-replayScale) > 1e-9 {
				t.Fatalf("replay damage scale=%v, want %v", spawn.DamageScale, replayScale)
			}
		}
	}
	if len(got) != len(want) {
		t.Fatalf("replayed %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("replayed %v, want %v", got, want)
		}
	}
	if len(o.records) != 0 || len(o.kinds) != 0 {
		t.Fatalf("bank kinds=%v records=%v, want cleared after firing", o.kinds, o.records)
	}
}

func TestObserverConfirmsDamageWithoutArmor(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	o := &observer{gauge: gaugeMax}
	o.Handle(unit.Context{ID: 1, Kind: KindObserver, Out: out}, unit.IncomingDamage{Token: 7, Amount: 20, Time: 1})
	for _, cmd := range drain(out) {
		if hit, ok := cmd.(unit.ConfirmDamage); ok && hit.Token == 7 {
			if math.Abs(hit.Amount-20) > 1e-6 {
				t.Fatalf("confirmed damage=%v, want 20", hit.Amount)
			}
			return
		}
	}
	t.Fatal("damage confirmation missing")
}

func TestObserverLocksFirstThreeKindsAndKeepsRepeats(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	o := &observer{slot: 0, x: 0, y: 0, gauge: 40, projectileSeen: map[uint64]bool{}}
	ctx := unit.Context{ID: 1, Kind: KindObserver, Out: out}
	shots := []unit.Snapshot{
		{ID: 1, Slot: 1, Role: unit.RoleProjectile, Kind: "火箭", X: 10, Radius: 5},
		{ID: 2, Slot: 1, Role: unit.RoleProjectile, Kind: "火箭", X: 20, Radius: 5},
		{ID: 3, Slot: 1, Role: unit.RoleProjectile, Kind: "苦无", X: 30, Radius: 5},
		{ID: 4, Slot: 1, Role: unit.RoleProjectile, Kind: "面灵气弹", X: 40, Radius: 5},
		{ID: 5, Slot: 1, Role: unit.RoleProjectile, Kind: "酒坛", X: 50, Radius: 5},
		{ID: 6, Slot: 1, Role: unit.RoleProjectile, Kind: "火箭", X: 60, Radius: 5},
	}
	o.recordProjectiles(ctx, unit.Sense{Self: unit.Snapshot{ID: 1, Slot: 0}, Nearby: shots})
	want := []string{"火箭", "火箭", "苦无", "面灵气弹", "火箭"}
	if len(o.records) != len(want) {
		t.Fatalf("records=%v, want %v", o.records, want)
	}
	for i := range want {
		if o.records[i] != want[i] {
			t.Fatalf("records=%v, want %v", o.records, want)
		}
	}
	if len(o.kinds) != 3 || o.kinds[0] != "火箭" || o.kinds[2] != "面灵气弹" {
		t.Fatalf("kinds=%v, want the first three", o.kinds)
	}
	if o.gauge != 65 {
		t.Fatalf("gauge=%v, want 40 plus 5 for each of the five recorded shots", o.gauge)
	}
	if !o.projectileSeen[5] {
		t.Fatal("rejected fourth kind should stay seen")
	}
	recordFX := 0
	for _, cmd := range drain(out) {
		if fx, ok := cmd.(unit.FX); ok && fx.Name == "observer-record" {
			recordFX++
		}
	}
	if recordFX != 3 {
		t.Fatalf("new-kind films=%d, want 3", recordFX)
	}
}

func TestObserverKeepsRecordsWhenGaugeDrainsOrTargetSwitches(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	o := &observer{
		booted: true, lastTime: 0.9,
		rng: unit.NewRNG(1), gauge: 40, targetID: 2, targetGroup: 2,
		kinds: []string{"火箭"}, records: []string{"火箭", "火箭"},
	}
	ctx := unit.Context{ID: 1, Kind: KindObserver, Out: out}
	self := unit.Snapshot{ID: 1, Slot: 0, Role: unit.RoleFighter, Radius: observerRadius}
	o.Handle(ctx, unit.Sense{Time: 1, Self: self})
	_ = drain(out)
	if len(o.records) != 2 || o.gauge >= 40 {
		t.Fatalf("after drain gauge=%v records=%v, want a lower gauge and the same bank", o.gauge, o.records)
	}
	o.gauge = 40
	o.targetID = 2
	o.targetGroup = 2
	other := unit.Snapshot{ID: 3, Slot: 1, Role: unit.RoleFighter, X: 100, Radius: 18, AimPriority: 15}
	o.Handle(ctx, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{other}})
	if o.gauge != 0 || len(o.records) != 2 {
		t.Fatalf("after switch gauge=%v records=%v, want gauge cleared and bank kept", o.gauge, o.records)
	}
}

func TestObserverDropsRemainingReplayWhenTargetIsGone(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	o := &observer{
		rng: unit.NewRNG(1), replaying: true, replay: []shotNote{{"火箭", 1}, {"苦无", 1}, {"火箭", 1}},
		kinds: []string{"酒坛"}, records: []string{"酒坛"},
	}
	ctx := unit.Context{ID: 1, Kind: KindObserver, Out: out}
	self := unit.Snapshot{ID: 1, Slot: 0, Role: unit.RoleFighter, Radius: observerRadius}
	o.Handle(ctx, unit.Sense{Time: 1, Self: self})
	if o.replaying || len(o.replay) != 0 {
		t.Fatalf("replay still queued: replaying=%v replay=%v", o.replaying, o.replay)
	}
	if len(o.records) != 1 || o.records[0] != "酒坛" {
		t.Fatalf("next bank=%v, want the unfired wine kept", o.records)
	}
	for _, cmd := range drain(out) {
		if _, ok := cmd.(unit.Spawn); ok {
			t.Fatal("dropped replay should not spawn")
		}
	}
}

func TestObserverEchoesAllBeamsAtDecayedDamage(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	o := &observer{
		rng: unit.NewRNG(1), booted: true, lastTime: 1,
		gauge: gaugeMax, targetID: 2,
		kinds:   []string{KindObserverBeam},
		records: []string{KindObserverBeam, KindObserverBeam},
		film: []shotNote{
			{KindObserverBeam, replayScale},
			{KindObserverBeam, replayScale},
			{KindObserverBeam, replayScale},
			{KindObserverBeam, replayScale},
		},
	}
	ctx := unit.Context{ID: 1, Kind: KindObserver, Out: out}
	self := unit.Snapshot{ID: 1, Slot: 0, Role: unit.RoleFighter, Radius: observerRadius}
	target := unit.Snapshot{ID: 2, Slot: 1, Role: unit.RoleFighter, X: 100, Radius: 18, AimPriority: 15}
	var scales []float64
	for i := 0; i < 6; i++ {
		o.Handle(ctx, unit.Sense{Time: 1 + float64(i)*replayGap, Self: self, Nearby: []unit.Snapshot{target}})
		for _, cmd := range drain(out) {
			if spawn, ok := cmd.(unit.Spawn); ok && spawn.Kind == KindObserverBeam {
				scales = append(scales, spawn.DamageScale)
			}
		}
	}
	want := []float64{replayScale, replayScale, replayScale * replayScale, replayScale * replayScale, replayScale * replayScale, replayScale * replayScale}
	if len(scales) != len(want) {
		t.Fatalf("scales=%v, want %v", scales, want)
	}
	for i := range want {
		if math.Abs(scales[i]-want[i]) > 1e-9 {
			t.Fatalf("scales=%v, want %v", scales, want)
		}
	}
	if len(o.film) != 6 || o.replaying {
		t.Fatalf("film=%v replaying=%v, want the six shots that flew", o.film, o.replaying)
	}
}

func TestObserverEchoesOtherProjectilesAt51PercentAndKeepsBeamOrder(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	o := &observer{
		records: []string{KindObserverKunaiCopy},
		film: []shotNote{
			{KindObserverKunaiCopy, 1},
			{KindObserverBeam, 0.9},
			{KindObserverKunaiCopy, 0.8},
			{KindObserverKunaiCopy, 0.7},
			{KindObserverKunaiCopy, 0.6},
			{KindObserverBeam, 0.5},
		},
	}
	ctx := unit.Context{ID: 1, Kind: KindObserver, Out: out}
	target := &unit.Snapshot{ID: 2, Slot: 1, X: 100}
	want := []shotNote{
		{KindObserverKunaiCopy, replayScale},
		{KindObserverKunaiCopy, replayScale},
		{KindObserverBeam, 0.9 * replayScale},
		{KindObserverKunaiCopy, 0.8 * replayScale},
		{KindObserverKunaiCopy, 0.7 * replayScale},
		{KindObserverBeam, 0.5 * replayScale},
	}
	o.beginVolley(1)
	var got []shotNote
	for i := 0; i < len(want); i++ {
		o.tickReplay(ctx, unit.Sense{Time: 1 + float64(i)*replayGap}, target)
		for _, cmd := range drain(out) {
			if spawn, ok := cmd.(unit.Spawn); ok {
				got = append(got, shotNote{spawn.Kind, spawn.DamageScale})
			}
		}
	}
	if len(got) != len(want) || o.replaying || len(o.film) != len(want) {
		t.Fatalf("shots=%v film=%v replaying=%v, want %v", got, o.film, o.replaying, want)
	}
	for i := range want {
		if got[i].kind != want[i].kind || math.Abs(got[i].scale-want[i].scale) > 1e-9 {
			t.Fatalf("shot %d=%v, want %v", i, got[i], want[i])
		}
	}
}

func TestObserverDryRoundsAccumulateAllBeamsAndDecayEachGeneration(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Kind: KindObserver, Out: out}
	o := &observer{}
	target := &unit.Snapshot{ID: 2, Slot: 1, X: 100}
	for round := 1; round <= 10; round++ {
		now := float64(round) * 10
		o.beginVolley(now)
		var scales []float64
		for i := 0; i < round; i++ {
			o.tickReplay(ctx, unit.Sense{Time: now + float64(i)*replayGap}, target)
			for _, cmd := range drain(out) {
				if spawn, ok := cmd.(unit.Spawn); ok {
					if spawn.Kind != KindObserverBeam {
						t.Fatalf("round %d spawned %s, want beam", round, spawn.Kind)
					}
					scales = append(scales, spawn.DamageScale)
				}
			}
		}
		if len(scales) != round || len(o.film) != round || o.replaying {
			t.Fatalf("round %d: scales=%v film=%v replaying=%v", round, scales, o.film, o.replaying)
		}
		for i, scale := range scales {
			if want := math.Pow(replayScale, float64(i)); math.Abs(scale-want) > 1e-9 {
				t.Fatalf("round %d beam %d scale=%v, want %v", round, i, scale, want)
			}
		}
	}
}

func TestObserverDryRoundFiresBeamThenEcho(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	o := &observer{
		rng: unit.NewRNG(1), booted: true, lastTime: 1,
		gauge: gaugeMax, targetID: 2,
		film: []shotNote{{KindObserverBeam, 1}},
	}
	ctx := unit.Context{ID: 1, Kind: KindObserver, Out: out}
	self := unit.Snapshot{ID: 1, Slot: 0, Role: unit.RoleFighter, Radius: observerRadius}
	target := unit.Snapshot{ID: 2, Slot: 1, Role: unit.RoleFighter, X: 100, Radius: 18, AimPriority: 15}
	var scales []float64
	for i := 0; i < 2; i++ {
		o.Handle(ctx, unit.Sense{Time: 1 + float64(i)*replayGap, Self: self, Nearby: []unit.Snapshot{target}})
		for _, cmd := range drain(out) {
			if spawn, ok := cmd.(unit.Spawn); ok && spawn.Kind == KindObserverBeam {
				scales = append(scales, spawn.DamageScale)
			}
		}
	}
	if len(scales) != 2 || math.Abs(scales[0]-1) > 1e-9 || math.Abs(scales[1]-replayScale) > 1e-9 {
		t.Fatalf("dry round scales=%v, want fresh beam then 85%% echo", scales)
	}
}

func TestObserverFilmKeepsOnlyShotsThatFlew(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	o := &observer{
		rng: unit.NewRNG(1), booted: true, lastTime: 1,
		gauge: gaugeMax, targetID: 2,
		kinds:   []string{KindObserverBeam},
		records: []string{KindObserverBeam, KindObserverBeam, KindObserverBeam},
	}
	ctx := unit.Context{ID: 1, Kind: KindObserver, Out: out}
	self := unit.Snapshot{ID: 1, Slot: 0, Role: unit.RoleFighter, Radius: observerRadius}
	target := unit.Snapshot{ID: 2, Slot: 1, Role: unit.RoleFighter, X: 100, Radius: 18, AimPriority: 15}
	o.Handle(ctx, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{target}})
	_ = drain(out)
	o.Handle(ctx, unit.Sense{Time: 1 + replayGap, Self: self})
	if len(o.film) != 1 || math.Abs(o.film[0].scale-replayScale) > 1e-9 {
		t.Fatalf("film=%v, want only the one shot that flew", o.film)
	}
	if o.replaying || len(o.replay) != 0 {
		t.Fatalf("replay still queued: %+v", o.replay)
	}
}

func TestObserverUsesBeamWhenNoSamples(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	o := &observer{rng: unit.NewRNG(1), gauge: gaugeMax, targetID: 2}
	ctx := unit.Context{ID: 1, Kind: KindObserver, Out: out}
	self := unit.Snapshot{ID: 1, Slot: 0, Role: unit.RoleFighter, Radius: observerRadius}
	target := unit.Snapshot{ID: 2, Slot: 1, Role: unit.RoleFighter, X: 100, Radius: 18, AimPriority: 15}
	o.Handle(ctx, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{target}})
	cmds := drain(out)
	n := 0
	for _, cmd := range cmds {
		if spawn, ok := cmd.(unit.Spawn); ok && spawn.Kind == KindObserverBeam {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("no-sample beam count=%d, want exactly one", n)
	}
}

func TestObserverBeamDamage(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	b := &beam{owner: 1, slot: 0}
	b.Handle(unit.Context{ID: 4, Kind: KindObserverBeam, Out: out}, unit.Collision{
		Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1},
	})
	for _, cmd := range drain(out) {
		if d, ok := cmd.(unit.Damage); ok {
			if d.Amount != beamDamage || d.To != 2 {
				t.Fatalf("beam damage=%+v", d)
			}
			return
		}
	}
	t.Fatal("beam did not deal damage")
}

func drain(out chan unit.Cmd) []unit.Cmd {
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

func hasFX(cmds []unit.Cmd, name string) bool {
	for _, cmd := range cmds {
		if fx, ok := cmd.(unit.FX); ok && fx.Name == name {
			return true
		}
	}
	return false
}

func hasSpawn(cmds []unit.Cmd, kind string) bool {
	for _, cmd := range cmds {
		if spawn, ok := cmd.(unit.Spawn); ok && spawn.Kind == kind {
			return true
		}
	}
	return false
}
