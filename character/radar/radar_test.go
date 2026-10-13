package 雷达

import (
	"math"
	"testing"
	"xqdj/internal/unit"
	_ "xqdj/map/hex"
)

func TestLaserStartsNorth(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	e := &雷达{}
	e.Handle(unit.Context{ID: 1, Kind: KindRadar, Out: out}, unit.Sense{
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

func TestLaserQuarterTurnIsEast(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	e := &雷达{}
	e.Handle(unit.Context{ID: 1, Kind: KindRadar, Out: out}, unit.Sense{
		Time:  5.5 / 4, // 1/4 圈 = radarSpin/4
		Self:  selfAt(0, 0),
		Field: unit.HexField(),
	})
	beam := lastNamed(drain(out), "beam")
	if beam == nil || math.Abs(beam.VX-1) > 1e-9 || math.Abs(beam.VY) > 1e-9 {
		t.Fatalf("dir=%v want east", beam)
	}
}

func TestLaserHitsEnemyOnRay(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	e := &雷达{}
	e.Handle(unit.Context{ID: 1, Kind: KindRadar, Out: out}, unit.Sense{
		Time:   0,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{enemyAt(0, 80)},
		Field:  unit.HexField(),
	})
	d := lastDamage(drain(out))
	if d == nil || d.Amount != radarDamage || d.To != 2 {
		t.Fatalf("damage=%v", d)
	}
}

func TestLaserMissesBehindAndOffAxis(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	e := &雷达{}
	ctx := unit.Context{ID: 1, Kind: KindRadar, Out: out}
	e.Handle(ctx, unit.Sense{
		Time:   0,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{enemyAt(0, -80)},
		Field:  unit.HexField(),
	})
	if lastDamage(drain(out)) != nil {
		t.Fatal("behind the ray should miss")
	}
	e.Handle(ctx, unit.Sense{
		Time:   0,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{enemyAt(80, 0)},
		Field:  unit.HexField(),
	})
	if lastDamage(drain(out)) != nil {
		t.Fatal("east at t=0 should miss")
	}
}

func TestLaserHitCooldown(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	e := &雷达{}
	ctx := unit.Context{ID: 1, Kind: KindRadar, Out: out}
	e.Handle(ctx, unit.Sense{
		Time:   0,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{enemyAt(0, 60)},
		Field:  unit.HexField(),
	})
	if lastDamage(drain(out)) == nil {
		t.Fatal("first hit")
	}
	dx, dy := laserDir(0.15)
	e.Handle(ctx, unit.Sense{
		Time:   0.15,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{enemyAt(dx*60, dy*60)},
		Field:  unit.HexField(),
	})
	if lastDamage(drain(out)) != nil {
		t.Fatal("0.15s still in CD")
	}
	dx, dy = laserDir(0.3)
	e.Handle(ctx, unit.Sense{
		Time:   0.3,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{enemyAt(dx*60, dy*60)},
		Field:  unit.HexField(),
	})
	d := lastDamage(drain(out))
	if d == nil || d.Amount != radarDamage {
		t.Fatalf("after CD damage=%v", d)
	}
}

func TestLaserIgnoresNonFighter(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	e := &雷达{}
	clone := unit.Snapshot{ID: 3, Role: unit.RoleClone, Slot: 1, X: 0, Y: 80, Radius: 18}
	e.Handle(unit.Context{ID: 1, Kind: KindRadar, Out: out}, unit.Sense{
		Time:   0,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{clone},
		Field:  unit.HexField(),
	})
	if lastDamage(drain(out)) != nil {
		t.Fatal("clone should not take 激光")
	}
}

func TestWallClipsLaser(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	e := &雷达{}
	field := unit.HexField()
	field.Walls = []unit.FieldWall{{
		Kind: unit.WallHard,
		X1:   -40, Y1: 40, X2: 40, Y2: 40,
		Radius: 6,
	}}
	e.Handle(unit.Context{ID: 1, Kind: KindRadar, Out: out}, unit.Sense{
		Time:   0,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{enemyAt(0, 120)},
		Field:  field,
	})
	cmds := drain(out)
	if lastDamage(cmds) != nil {
		t.Fatal("enemy behind hard wall should not take 激光")
	}
	beam := lastNamed(cmds, "beam")
	if beam == nil || beam.Amount > 50 {
		t.Fatalf("beam len=%v want clipped near wall", beam)
	}
}

func TestActorWallClipsLaser(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	e := &雷达{}
	e.Handle(unit.Context{ID: 1, Kind: KindRadar, Out: out}, unit.Sense{
		Time:   0,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{enemyAt(0, 120)},
		Field:  unit.HexField(),
		Walls: []unit.WallView{{
			X1: -40, Y1: 40, X2: 40, Y2: 40, Radius: 8,
		}},
	})
	cmds := drain(out)
	if lastDamage(cmds) != nil {
		t.Fatal("enemy behind actor wall should not take 激光")
	}
	beam := lastNamed(cmds, "beam")
	if beam == nil || beam.Amount > 55 {
		t.Fatalf("beam len=%v want clipped by Sense.Walls", beam)
	}
}

func TestLaserHitsEnemyInFrontOfWall(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	e := &雷达{}
	e.Handle(unit.Context{ID: 1, Kind: KindRadar, Out: out}, unit.Sense{
		Time:   0,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{enemyAt(0, 40)},
		Field:  unit.HexField(),
		Walls: []unit.WallView{{
			X1: -40, Y1: 100, X2: 40, Y2: 100, Radius: 8,
		}},
	})
	d := lastDamage(drain(out))
	if d == nil || d.To != 2 {
		t.Fatalf("enemy in front of wall damage=%v", d)
	}
}

func TestAcceptsIncomingDamage(t *testing.T) {
	out := make(chan unit.Cmd, 4)
	e := &雷达{}
	e.Handle(unit.Context{ID: 1, Kind: KindRadar, Out: out}, unit.IncomingDamage{
		Token: 4, Amount: 10, Time: 1,
	})
	if _, ok := drain(out)[0].(unit.ConfirmDamage); !ok {
		t.Fatal("should confirm")
	}
}

func selfAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindRadar, Role: unit.RoleFighter, Slot: 0,
		X: x, Y: y, Radius: radarRadius, Vision: radarVision,
	}
}

func enemyAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: x, Y: y, Radius: 18}
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
