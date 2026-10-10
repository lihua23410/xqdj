package 雷达_百战

import (
	"math"
	"testing"

	"xqdj/internal/unit"
	_ "xqdj/map/hex"
)

func TestSweepStartsNorth(t *testing.T) {
	clearArmWaits()
	out := make(chan unit.Cmd, 8)
	a := &百战{mode: modeBig}
	a.Handle(unit.Context{ID: 1, Kind: KindRadarVeteran, Out: out}, unit.Sense{
		Time:  0,
		Self:  selfAt(0, 0),
		Field: unit.HexField(),
	})
	beam := lastNamed(drain(out), "beam")
	if beam == nil {
		t.Fatal("missing beam")
	}
	if math.Abs(beam.VX) > 1e-9 || math.Abs(beam.VY-1) > 1e-9 {
		t.Fatalf("dir=(%v,%v) want north", beam.VX, beam.VY)
	}
}

func TestSweepQuarterTurnIsEast(t *testing.T) {
	clearArmWaits()
	out := make(chan unit.Cmd, 8)
	a := &百战{mode: modeBig}
	a.Handle(unit.Context{ID: 1, Kind: KindRadarVeteran, Out: out}, unit.Sense{
		Time:  sweepSpin / 4,
		Self:  selfAt(0, 0),
		Field: unit.HexField(),
	})
	beam := lastNamed(drain(out), "beam")
	if beam == nil || math.Abs(beam.VX-1) > 1e-9 || math.Abs(beam.VY) > 1e-9 {
		t.Fatalf("dir=%v want east", beam)
	}
}

func TestSweepSpawnsBigBarrageOnHit(t *testing.T) {
	clearArmWaits()
	out := make(chan unit.Cmd, 8)
	a := &百战{mode: modeBig}
	a.Handle(unit.Context{ID: 1, Kind: KindRadarVeteran, Out: out}, unit.Sense{
		Time:   0,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{enemyAt(0, 80)},
		Field:  unit.HexField(),
	})
	spawns := allSpawns(drain(out))
	if len(spawns) != 1 || spawns[0].Kind != KindBarrage {
		t.Fatalf("spawns=%v", spawns)
	}
	if math.Abs(spawns[0].X) > 1e-6 || math.Abs(spawns[0].Y-(80-18)) > 1e-6 {
		t.Fatalf("mark=(%v,%v) want on circle surface", spawns[0].X, spawns[0].Y)
	}
}

func TestSweepSpawnsThreeSmallBarrages(t *testing.T) {
	clearArmWaits()
	out := make(chan unit.Cmd, 16)
	a := &百战{mode: modeSmall}
	a.Handle(unit.Context{ID: 1, Kind: KindRadarVeteran, Out: out}, unit.Sense{
		Time:   0,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{enemyAt(0, 80)},
		Field:  unit.HexField(),
	})
	spawns := allSpawns(drain(out))
	if len(spawns) != 3 {
		t.Fatalf("want 3 small, got %v", spawns)
	}
	mx, my := 0.0, 80-18.0
	for _, sp := range spawns {
		if sp.Kind != KindSmallBarrage {
			t.Fatalf("kind=%q", sp.Kind)
		}
		if math.Hypot(sp.X-mx, sp.Y-my) > smallScatter+1e-6 {
			t.Fatalf("spot=(%v,%v) outside scatter", sp.X, sp.Y)
		}
	}
	waits := drainArmWaits()
	if len(waits) != 3 || waits[0] != 0 {
		t.Fatalf("arm waits=%v want [0, …]", waits)
	}
	if waits[1] < 0 || waits[1] > smallGapMax+1e-9 {
		t.Fatalf("gap1=%v", waits[1])
	}
	if waits[2] < waits[1]-1e-9 || waits[2] > waits[1]+smallGapMax+1e-9 {
		t.Fatalf("arm2=%v not chained from %v", waits[2], waits[1])
	}
}

func TestSweepNoDirectDamage(t *testing.T) {
	clearArmWaits()
	out := make(chan unit.Cmd, 8)
	a := &百战{mode: modeBig}
	a.Handle(unit.Context{ID: 1, Kind: KindRadarVeteran, Out: out}, unit.Sense{
		Time:   0,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{enemyAt(0, 80)},
		Field:  unit.HexField(),
	})
	if lastDamage(drain(out)) != nil {
		t.Fatal("sweep must not deal damage")
	}
}

func TestSameTargetCooldown(t *testing.T) {
	clearArmWaits()
	out := make(chan unit.Cmd, 8)
	a := &百战{mode: modeBig}
	ctx := unit.Context{ID: 1, Kind: KindRadarVeteran, Out: out}
	markAlong := func(t float64) unit.Snapshot {
		dx, dy := sweepDir(t)
		return enemyAt(dx*60, dy*60)
	}
	a.Handle(ctx, unit.Sense{
		Time: 0, Self: selfAt(0, 0), Nearby: []unit.Snapshot{markAlong(0)}, Field: unit.HexField(),
	})
	if lastSpawn(drain(out)) == nil {
		t.Fatal("first mark")
	}
	a.Handle(ctx, unit.Sense{
		Time: 1.0, Self: selfAt(0, 0), Nearby: []unit.Snapshot{markAlong(1.0)}, Field: unit.HexField(),
	})
	if lastSpawn(drain(out)) != nil {
		t.Fatal("1.0s still in CD")
	}
	a.Handle(ctx, unit.Sense{
		Time: 1.2, Self: selfAt(0, 0), Nearby: []unit.Snapshot{markAlong(1.2)}, Field: unit.HexField(),
	})
	if lastSpawn(drain(out)) == nil {
		t.Fatal("after 1.2s should mark again")
	}
}

func TestDifferentTargetsNoSharedCD(t *testing.T) {
	clearArmWaits()
	out := make(chan unit.Cmd, 8)
	a := &百战{mode: modeBig}
	ctx := unit.Context{ID: 1, Kind: KindRadarVeteran, Out: out}
	a.Handle(ctx, unit.Sense{
		Time: 0, Self: selfAt(0, 0),
		Nearby: []unit.Snapshot{enemyAt(0, 60)},
		Field:  unit.HexField(),
	})
	if lastSpawn(drain(out)) == nil {
		t.Fatal("first")
	}
	other := enemyAt(0, 100)
	other.ID = 3
	a.Handle(ctx, unit.Sense{
		Time: 0.05, Self: selfAt(0, 0),
		Nearby: []unit.Snapshot{enemyAt(0, 60), other},
		Field:  unit.HexField(),
	})
	if lastSpawn(drain(out)) == nil {
		t.Fatal("second target should mark without shared CD")
	}
}

func TestWallClipsSweep(t *testing.T) {
	clearArmWaits()
	out := make(chan unit.Cmd, 8)
	a := &百战{mode: modeBig}
	field := unit.HexField()
	field.Walls = []unit.FieldWall{{
		Kind: unit.WallHard,
		X1:   -40, Y1: 40, X2: 40, Y2: 40,
		Radius: 6,
	}}
	a.Handle(unit.Context{ID: 1, Kind: KindRadarVeteran, Out: out}, unit.Sense{
		Time:   0,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{enemyAt(0, 120)},
		Field:  field,
	})
	cmds := drain(out)
	if lastSpawn(cmds) != nil {
		t.Fatal("enemy behind hard wall should not be marked")
	}
	beam := lastNamed(cmds, "beam")
	if beam == nil || beam.Amount > 50 {
		t.Fatalf("beam len=%v want clipped near wall", beam)
	}
}

func TestActorHardWallClipsSweep(t *testing.T) {
	clearArmWaits()
	out := make(chan unit.Cmd, 8)
	a := &百战{mode: modeBig}
	a.Handle(unit.Context{ID: 1, Kind: KindRadarVeteran, Out: out}, unit.Sense{
		Time:   0,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{enemyAt(0, 120)},
		Field:  unit.HexField(),
		Walls: []unit.WallView{{
			X1: -40, Y1: 40, X2: 40, Y2: 40, Radius: 8,
		}},
	})
	cmds := drain(out)
	if lastSpawn(cmds) != nil {
		t.Fatal("enemy behind actor hard wall should not be marked")
	}
	beam := lastNamed(cmds, "beam")
	if beam == nil || beam.Amount > 55 {
		t.Fatalf("beam len=%v want clipped by Sense.Walls", beam)
	}
}

func TestBigBarrageExplodesAfterDelay(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	b := newShell(unit.SpawnInfo{OwnerID: 1, Slot: 0}, bigDelay, bigDamage, bigRadius, bigKnockSp, bigKnockDur, 0)
	ctx := unit.Context{ID: 9, Kind: KindBarrage, Out: out}
	b.Handle(ctx, unit.Sense{
		Time: 0, Self: shellAt(0, 50, KindBarrage, bigRadius),
		Nearby: []unit.Snapshot{enemyAt(10, 50)},
	})
	cmds := drain(out)
	if lastDamage(cmds) != nil {
		t.Fatal("too early")
	}
	if lastNamed(cmds, "mark") == nil {
		t.Fatal("missing mark fx")
	}
	b.Handle(ctx, unit.Sense{
		Time: 0.5, Self: shellAt(0, 50, KindBarrage, bigRadius),
		Nearby: []unit.Snapshot{enemyAt(10, 50)},
	})
	if lastDamage(drain(out)) != nil {
		t.Fatal("0.5s still winding up")
	}
	b.Handle(ctx, unit.Sense{
		Time: bigDelay, Self: shellAt(0, 50, KindBarrage, bigRadius),
		Nearby: []unit.Snapshot{enemyAt(10, 50)},
	})
	cmds = drain(out)
	d := lastDamage(cmds)
	if d == nil || d.Amount != bigDamage || d.From != 1 || d.To != 2 {
		t.Fatalf("damage=%v", d)
	}
	fs := lastFS(cmds)
	if fs == nil || fs.BaseSpeed != bigKnockSp || math.Abs(fs.ExpiresAt-(bigDelay+bigKnockDur)) > 1e-9 {
		t.Fatalf("knock=%v", fs)
	}
}

func TestSmallBarrageWaitsArmThenDelay(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	b := newShell(unit.SpawnInfo{OwnerID: 1, Slot: 0}, smallDelay, smallDamage, smallRadius, smallKnockSp, smallKnockDur, 0.1)
	ctx := unit.Context{ID: 9, Kind: KindSmallBarrage, Out: out}
	b.Handle(ctx, unit.Sense{
		Time: 0, Self: shellAt(0, 50, KindSmallBarrage, smallRadius),
		Nearby: []unit.Snapshot{enemyAt(10, 50)},
	})
	if lastNamed(drain(out), "mark") != nil {
		t.Fatal("mark before arm")
	}
	b.Handle(ctx, unit.Sense{
		Time: 0.1, Self: shellAt(0, 50, KindSmallBarrage, smallRadius),
		Nearby: []unit.Snapshot{enemyAt(10, 50)},
	})
	if lastNamed(drain(out), "mark") == nil {
		t.Fatal("mark should start at arm")
	}
	b.Handle(ctx, unit.Sense{
		Time: 0.1 + smallDelay - 0.01, Self: shellAt(0, 50, KindSmallBarrage, smallRadius),
		Nearby: []unit.Snapshot{enemyAt(10, 50)},
	})
	if lastDamage(drain(out)) != nil {
		t.Fatal("still winding")
	}
	b.Handle(ctx, unit.Sense{
		Time: 0.1 + smallDelay, Self: shellAt(0, 50, KindSmallBarrage, smallRadius),
		Nearby: []unit.Snapshot{enemyAt(10, 50)},
	})
	cmds := drain(out)
	d := lastDamage(cmds)
	if d == nil || d.Amount != smallDamage {
		t.Fatalf("damage=%v", d)
	}
	fs := lastFS(cmds)
	if fs == nil || fs.BaseSpeed != smallKnockSp {
		t.Fatalf("knock=%v", fs)
	}
}

func TestKnockDirAtCenterIsUnit(t *testing.T) {
	ux, uy := knockDir(0, 0, 0, 0, 7, 1.25)
	if math.Abs(math.Hypot(ux, uy)-1) > 1e-9 {
		t.Fatalf("dir=(%v,%v) not unit", ux, uy)
	}
}

func TestAcceptsIncomingDamage(t *testing.T) {
	out := make(chan unit.Cmd, 4)
	a := &百战{}
	a.Handle(unit.Context{ID: 1, Kind: KindRadarVeteran, Out: out}, unit.IncomingDamage{
		Token: 4, Amount: 10, Time: 1,
	})
	if _, ok := drain(out)[0].(unit.ConfirmDamage); !ok {
		t.Fatal("should confirm")
	}
}

func selfAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindRadarVeteran, Role: unit.RoleFighter, Slot: 0,
		X: x, Y: y, Radius: vetRadius, Vision: vetVision,
	}
}

func enemyAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: x, Y: y, Radius: 18}
}

func shellAt(x, y float64, kind string, radius float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 9, Kind: kind, Role: unit.RoleHelper, Slot: 0,
		X: x, Y: y, Radius: 1, Vision: radius + 80,
	}
}

func clearArmWaits() {
	armMu.Lock()
	armWaits = nil
	armMu.Unlock()
}

func drainArmWaits() []float64 {
	armMu.Lock()
	defer armMu.Unlock()
	out := append([]float64(nil), armWaits...)
	armWaits = nil
	return out
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

func lastDamage(cmds []unit.Cmd) *unit.Damage {
	var d *unit.Damage
	for _, c := range cmds {
		if v, ok := c.(unit.Damage); ok {
			cp := v
			d = &cp
		}
	}
	return d
}

func lastSpawn(cmds []unit.Cmd) *unit.Spawn {
	var sp *unit.Spawn
	for _, c := range cmds {
		if v, ok := c.(unit.Spawn); ok {
			cp := v
			sp = &cp
		}
	}
	return sp
}

func allSpawns(cmds []unit.Cmd) []unit.Spawn {
	var out []unit.Spawn
	for _, c := range cmds {
		if v, ok := c.(unit.Spawn); ok {
			out = append(out, v)
		}
	}
	return out
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

func lastFS(cmds []unit.Cmd) *unit.AddFS {
	var fs *unit.AddFS
	for _, c := range cmds {
		if v, ok := c.(unit.AddFS); ok {
			cp := v
			fs = &cp
		}
	}
	return fs
}
