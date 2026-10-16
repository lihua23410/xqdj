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

func TestNameMarksNearestAndHidesBeam(t *testing.T) {
	clearArmWaits()
	out := make(chan unit.Cmd, 8)
	a := &百战{mode: modeBig}
	ctx := unit.Context{ID: 1, Kind: KindRadarVeteran, Out: out}
	far := enemyAt(0, 100)
	far.ID = 3
	a.Handle(ctx, unit.Sense{
		Time: 0, Self: selfAt(0, 0),
		Nearby: []unit.Snapshot{far, enemyAt(0, 60)},
		Field:  unit.HexField(),
	})
	cmds := drain(out)
	if lastSpawn(cmds) != nil || !beamHidden(cmds) {
		t.Fatal("naming frame should not fire, and the beam should be cleared")
	}
	name := lastNamed(cmds, "name")
	if name == nil || name.Amount <= 0 || math.Abs(name.Y-60) > 1e-6 {
		t.Fatalf("name=%v", name)
	}
	aim := lastAim(cmds)
	if aim == nil || aim.UnitID != 2 || aim.Value != nameAim {
		t.Fatalf("aim=%v", aim)
	}
}

func TestNameFiresBigAtCenter(t *testing.T) {
	clearArmWaits()
	out := make(chan unit.Cmd, 8)
	a := &百战{mode: modeBig}
	ctx := unit.Context{ID: 1, Kind: KindRadarVeteran, Out: out}
	sense := func(t float64, e unit.Snapshot) {
		a.Handle(ctx, unit.Sense{
			Time: t, Self: selfAt(0, 0), Nearby: []unit.Snapshot{e}, Field: unit.HexField(),
		})
	}
	e := enemyAt(0, 80)
	sense(0, e)
	drain(out)
	e.X, e.Y = 12, 40
	sense(morphDur+nameGap, e)
	cmds := drain(out)
	spawns := allSpawns(cmds)
	if len(spawns) != 1 || spawns[0].Kind != KindBarrage {
		t.Fatalf("spawns=%v", spawns)
	}
	if math.Abs(spawns[0].X-12) > 1e-6 || math.Abs(spawns[0].Y-40) > 1e-6 {
		t.Fatalf("pin=(%v,%v) want current center", spawns[0].X, spawns[0].Y)
	}
	launches := allNamed(cmds, "launch")
	if len(launches) != 1 || launches[0].Amount != 1 || launches[0].X != 0 || launches[0].Y != 0 {
		t.Fatalf("launch=%v", launches)
	}
}

func TestNameFiresThreeSmallThenRestores(t *testing.T) {
	clearArmWaits()
	out := make(chan unit.Cmd, 16)
	a := &百战{mode: modeSmall}
	ctx := unit.Context{ID: 1, Kind: KindRadarVeteran, Out: out}
	e := enemyAt(0, 80)
	first := morphDur + nameGap
	for _, t := range []float64{0, first, first + nameGap, first + 2*nameGap} {
		a.Handle(ctx, unit.Sense{
			Time: t, Self: selfAt(0, 0), Nearby: []unit.Snapshot{e}, Field: unit.HexField(),
		})
		drain(out)
	}
	a.Handle(ctx, unit.Sense{
		Time: first + 3*nameGap, Self: selfAt(0, 0), Nearby: []unit.Snapshot{e}, Field: unit.HexField(),
	})
	cmds := drain(out)
	spawns := allSpawns(cmds)
	if len(spawns) != 3 {
		t.Fatalf("want 3 small, got %v", spawns)
	}
	for _, sp := range spawns {
		if sp.Kind != KindSmallBarrage || math.Hypot(sp.X, sp.Y-80) > smallScatter+1e-6 {
			t.Fatalf("spot=%+v", sp)
		}
	}
	launches := allNamed(cmds, "launch")
	if len(launches) != 3 {
		t.Fatalf("launches=%v", launches)
	}
	for i, fx := range launches {
		if fx.Amount != 0 || math.Abs(fx.VX-float64(i-1)*10) > 1e-9 {
			t.Fatalf("launch %d=%+v", i, fx)
		}
		if i > 0 && fx.VY < launches[i-1].VY-1e-9 {
			t.Fatalf("delay not chained: %v", launches)
		}
	}
	aim := lastAim(cmds)
	if aim == nil || aim.Value != 15 {
		t.Fatalf("restore=%v", aim)
	}
	if lastNamed(cmds, "beam") == nil || lastNamed(cmds, "beam").Amount > 0 {
		t.Fatal("sweep stays down while the barrel retracts")
	}
	if name := lastNamed(cmds, "name"); name == nil || name.Amount > 0 {
		t.Fatal("mark should clear with the name")
	}
	a.Handle(ctx, unit.Sense{
		Time: first + 3*nameGap + morphDur, Self: selfAt(0, 0), Field: unit.HexField(),
	})
	back := drain(out)
	if lastNamed(back, "beam") == nil || lastNamed(back, "beam").Amount <= 0 {
		t.Fatal("sweep should return after the barrel retracts")
	}
	if gun := lastNamed(back, "gun"); gun == nil || gun.Amount >= 0 {
		t.Fatal("barrel should be gone")
	}
}

func TestNameClearsWhenTargetGone(t *testing.T) {
	clearArmWaits()
	out := make(chan unit.Cmd, 8)
	a := &百战{mode: modeBig}
	ctx := unit.Context{ID: 1, Kind: KindRadarVeteran, Out: out}
	a.Handle(ctx, unit.Sense{
		Time: 0, Self: selfAt(0, 0), Nearby: []unit.Snapshot{enemyAt(0, 80)}, Field: unit.HexField(),
	})
	drain(out)
	a.Handle(ctx, unit.Sense{
		Time: 0.4, Self: selfAt(0, 0), Field: unit.HexField(),
	})
	cmds := drain(out)
	if lastSpawn(cmds) != nil || lastAim(cmds) != nil || !beamHidden(cmds) {
		t.Fatal("dead target should not be shot, rewritten, or bring the sweep back yet")
	}
	if name := lastNamed(cmds, "name"); name == nil || name.Amount > 0 {
		t.Fatal("mark should clear")
	}
	a.Handle(ctx, unit.Sense{
		Time: 0.4 + morphDur, Self: selfAt(0, 0), Field: unit.HexField(),
	})
	back := drain(out)
	beam := lastNamed(back, "beam")
	if beam == nil || beam.Amount <= 0 {
		t.Fatal("sweep should return after the barrel retracts")
	}
}

func TestNameLeavesZeroAim(t *testing.T) {
	clearArmWaits()
	out := make(chan unit.Cmd, 8)
	a := &百战{mode: modeBig}
	e := enemyAt(0, 80)
	e.AimPriority = 0
	a.Handle(unit.Context{ID: 1, Kind: KindRadarVeteran, Out: out}, unit.Sense{
		Time: 0, Self: selfAt(0, 0), Nearby: []unit.Snapshot{e}, Field: unit.HexField(),
	})
	cmds := drain(out)
	if lastAim(cmds) != nil || !beamHidden(cmds) {
		t.Fatal("zero aim stays, beam hides")
	}
	if name := lastNamed(cmds, "name"); name == nil || name.Amount <= 0 {
		t.Fatal("named target should be marked")
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

func TestNameDoesNotFireEarly(t *testing.T) {
	clearArmWaits()
	out := make(chan unit.Cmd, 8)
	a := &百战{mode: modeBig}
	ctx := unit.Context{ID: 1, Kind: KindRadarVeteran, Out: out}
	e := enemyAt(0, 80)
	a.Handle(ctx, unit.Sense{
		Time: 0, Self: selfAt(0, 0), Nearby: []unit.Snapshot{e}, Field: unit.HexField(),
	})
	if lastSpawn(drain(out)) != nil {
		t.Fatal("mark frame must not fire")
	}
	a.Handle(ctx, unit.Sense{
		Time: 0.9, Self: selfAt(0, 0), Nearby: []unit.Snapshot{e}, Field: unit.HexField(),
	})
	cmds := drain(out)
	if lastSpawn(cmds) != nil || !beamHidden(cmds) {
		t.Fatal("still waiting for the first second")
	}
}

func TestNameIgnoresOthersWhileHeld(t *testing.T) {
	clearArmWaits()
	out := make(chan unit.Cmd, 8)
	a := &百战{mode: modeBig}
	ctx := unit.Context{ID: 1, Kind: KindRadarVeteran, Out: out}
	a.Handle(ctx, unit.Sense{
		Time: 0, Self: selfAt(0, 0),
		Nearby: []unit.Snapshot{enemyAt(0, 60)},
		Field:  unit.HexField(),
	})
	drain(out)
	other := enemyAt(0, 100)
	other.ID = 3
	a.Handle(ctx, unit.Sense{
		Time: 0.05, Self: selfAt(0, 0),
		Nearby: []unit.Snapshot{enemyAt(0, 60), other},
		Field:  unit.HexField(),
	})
	cmds := drain(out)
	if lastSpawn(cmds) != nil || lastAim(cmds) != nil || !beamHidden(cmds) {
		t.Fatal("a held name ignores everyone else")
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
	if lastNamed(cmds, "mark") != nil {
		t.Fatal("shadow before the shell has risen")
	}
	b.Handle(ctx, unit.Sense{
		Time: bigRise, Self: shellAt(0, 50, KindBarrage, bigRadius),
		Nearby: []unit.Snapshot{enemyAt(10, 50)},
	})
	if lastNamed(drain(out), "mark") == nil {
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
	if lastNamed(drain(out), "mark") != nil {
		t.Fatal("shadow before the shell has risen")
	}
	b.Handle(ctx, unit.Sense{
		Time: 0.1 + smallRise, Self: shellAt(0, 50, KindSmallBarrage, smallRadius),
		Nearby: []unit.Snapshot{enemyAt(10, 50)},
	})
	if lastNamed(drain(out), "mark") == nil {
		t.Fatal("mark should start after the rise")
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
	return unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: x, Y: y, Radius: 18, AimPriority: 15, HP: 100}
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

func lastAim(cmds []unit.Cmd) *unit.SetAimPriority {
	var aim *unit.SetAimPriority
	for _, c := range cmds {
		if v, ok := c.(unit.SetAimPriority); ok {
			cp := v
			aim = &cp
		}
	}
	return aim
}

func beamHidden(cmds []unit.Cmd) bool {
	beam := lastNamed(cmds, "beam")
	return beam != nil && beam.Amount <= 0
}

func allNamed(cmds []unit.Cmd, name string) []unit.FX {
	var out []unit.FX
	for _, c := range cmds {
		if v, ok := c.(unit.FX); ok && v.Name == name {
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
