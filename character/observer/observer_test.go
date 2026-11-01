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
		rng: unit.NewRNG(1), gauge: 100, targetID: 2,
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
	if math.Abs(o.gauge-21) > 1e-6 {
		t.Fatalf("gauge=%v, want 10/s observation + 20 record", o.gauge)
	}
	if len(o.records) != 1 || o.records[0] != "靶子子弹" {
		t.Fatalf("records=%v, want recorded projectile", o.records)
	}
	if !hasFX(cmds, "observer-record") {
		t.Fatal("observer-record FX missing")
	}
}

func TestObserverReplaysTenRecordedProjectilesAtScaledDamage(t *testing.T) {
	out := make(chan unit.Cmd, 256)
	o := &observer{
		rng: unit.NewRNG(1), gauge: 100, targetID: 2,
		records: []string{KindObserverBeam},
	}
	ctx := unit.Context{ID: 1, Kind: KindObserver, Out: out}
	self := unit.Snapshot{ID: 1, Slot: 0, Role: unit.RoleFighter, X: 0, Y: 0, Radius: observerRadius}
	target := unit.Snapshot{ID: 2, Slot: 1, Role: unit.RoleFighter, X: 100, Y: 0, Radius: 18, AimPriority: 15}
	var replay unit.Spawn
	count := 0
	for i := 0; i < replayShots; i++ {
		o.Handle(ctx, unit.Sense{Time: 1 + float64(i)*replayGap, Self: self, Nearby: []unit.Snapshot{target}})
		for _, cmd := range drain(out) {
			if spawn, ok := cmd.(unit.Spawn); ok && spawn.Kind == KindObserverBeam {
				replay = spawn
				count++
			}
		}
	}
	if count != replayShots {
		t.Fatalf("replayed bullets=%d, want %d", count, replayShots)
	}
	if math.Abs(replay.DamageScale-replayScale) > 1e-9 {
		t.Fatalf("replay damage scale=%v, want %v", replay.DamageScale, replayScale)
	}
}

func TestObservationArmorReducesDamage(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	o := &observer{gauge: observeArmorAt}
	o.Handle(unit.Context{ID: 1, Kind: KindObserver, Out: out}, unit.IncomingDamage{Token: 7, Amount: 20, Time: 1})
	for _, cmd := range drain(out) {
		if hit, ok := cmd.(unit.ConfirmDamage); ok && hit.Token == 7 {
			if math.Abs(hit.Amount-16) > 1e-6 {
				t.Fatalf("armored damage=%v, want 16", hit.Amount)
			}
			return
		}
	}
	t.Fatal("damage confirmation missing")
}

func TestObserverUsesBeamWhenNoSamples(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	o := &observer{rng: unit.NewRNG(1), gauge: 100, targetID: 2}
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
