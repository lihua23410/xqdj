package 内燃机

import (
	"math"
	"testing"
	"xqdj/internal/unit"
)

func TestBootAppliesGearOneStats(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	e := &内燃机{gear: 1}
	e.Handle(unit.Context{ID: 1, Kind: KindEngine, Out: out}, unit.Sense{
		Time: 0,
		Self: selfAt(0, 0, 120, 0),
	})
	cmds := drain(out)
	if !hasCruise(cmds, 120) || !hasVision(cmds, 96) {
		t.Fatalf("boot cmds=%v", cmds)
	}
	if !hasNoFrameFreeze(cmds) {
		t.Fatalf("boot missing NoFrameFreeze: %v", cmds)
	}
	if heat := lastHeat(cmds); heat == nil || heat.Amount != 1 || heat.VX != 0 {
		t.Fatalf("heat=%v", heat)
	}
}

func TestWallHitDoesNotRaiseGear(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	e := booted(out)
	e.Handle(unit.Context{ID: 1, Kind: KindEngine, Out: out}, unit.WallHit{
		Time: 1, NX: -1, NY: 0,
	})
	cmds := drain(out)
	if e.gear != 1 {
		t.Fatalf("gear=%d", e.gear)
	}
	if hasCruise(cmds, 156) {
		t.Fatalf("wall should not change cruise: %v", cmds)
	}
}

func TestHeatFillsAndRaisesGearWithoutSnappingVelocity(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	e := booted(out)
	ctx := unit.Context{ID: 1, Kind: KindEngine, Out: out}
	e.Handle(ctx, unit.Sense{Time: 2, Self: selfAt(0, 0, 120, 0)})
	cmds := drain(out)
	if e.gear != 2 || e.heat != 0 {
		t.Fatalf("gear=%d heat=%v", e.gear, e.heat)
	}
	if !hasCruise(cmds, 156) || !hasVision(cmds, 120) {
		t.Fatalf("cmds=%v", cmds)
	}
	if lastVel(cmds) != nil {
		t.Fatalf("升档 must not snap velocity: %v", cmds)
	}
}

func TestHeatFillTimes(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	e := booted(out)
	ctx := unit.Context{ID: 1, Kind: KindEngine, Out: out}

	e.Handle(ctx, unit.Sense{Time: 1.9, Self: selfAt(0, 0, 120, 0)})
	_ = drain(out)
	if e.gear != 1 {
		t.Fatalf("before 2s gear=%d", e.gear)
	}

	e.Handle(ctx, unit.Sense{Time: 2, Self: selfAt(0, 0, 120, 0)})
	_ = drain(out)
	if e.gear != 2 {
		t.Fatalf("t=2 gear=%d", e.gear)
	}

	e.Handle(ctx, unit.Sense{Time: 4, Self: selfAt(0, 0, 120, 0)})
	_ = drain(out)
	e.Handle(ctx, unit.Sense{Time: 6, Self: selfAt(0, 0, 120, 0)})
	_ = drain(out)
	e.Handle(ctx, unit.Sense{Time: 8, Self: selfAt(0, 0, 120, 0)})
	_ = drain(out)
	if e.gear != 5 {
		t.Fatalf("t=8 gear=%d want 5", e.gear)
	}

	e.Handle(ctx, unit.Sense{Time: 11.9, Self: selfAt(0, 0, 120, 0)})
	_ = drain(out)
	if e.gear != 5 {
		t.Fatalf("5档 4秒未满 gear=%d", e.gear)
	}
	e.Handle(ctx, unit.Sense{Time: 12, Self: selfAt(0, 0, 120, 0)})
	_ = drain(out)
	if e.gear != 6 {
		t.Fatalf("t=12 gear=%d want 6", e.gear)
	}
}

func TestIncomingDamageDoesNotFillHeat(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	e := booted(out)
	e.heat = 0.4
	e.Handle(unit.Context{ID: 1, Kind: KindEngine, Out: out}, unit.IncomingDamage{
		Token: 1, Amount: 30, Time: 1,
	})
	if e.heat != 0.4 || e.gear != 1 || e.frail {
		t.Fatalf("heat=%v gear=%d frail=%v", e.heat, e.gear, e.frail)
	}
	if _, ok := drain(out)[0].(unit.ConfirmDamage); !ok {
		t.Fatal("should still confirm hp loss")
	}
}

func TestMetronomeTicksWithoutTarget(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	e := booted(out)
	ctx := unit.Context{ID: 1, Kind: KindEngine, Out: out}
	e.gear = 3
	e.Handle(ctx, unit.Sense{Time: 0.5, Self: selfAt(0, 0, 192, 0)})
	if hasDamage(drain(out)) {
		t.Fatal("empty vision should not damage")
	}
	e.Handle(ctx, unit.Sense{
		Time:   0.6,
		Self:   selfAt(0, 0, 192, 0),
		Nearby: []unit.Snapshot{enemyAt(10, 0)},
	})
	if hasDamage(drain(out)) {
		t.Fatal("still in CD")
	}
	e.Handle(ctx, unit.Sense{
		Time:   1.0,
		Self:   selfAt(0, 0, 192, 0),
		Nearby: []unit.Snapshot{enemyAt(10, 0)},
	})
	d := lastDamage(drain(out))
	if d == nil || d.Amount != 1 || d.To != 2 {
		t.Fatalf("damage=%v", d)
	}
}

func TestGearSixTickDamagesThree(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	e := booted(out)
	ctx := unit.Context{ID: 1, Kind: KindEngine, Out: out}
	e.gear = 6
	e.Handle(ctx, unit.Sense{
		Time:   0.5,
		Self:   selfAt(0, 0, 300, 0),
		Nearby: []unit.Snapshot{enemyAt(10, 0)},
	})
	d := lastDamage(drain(out))
	if d == nil || d.Amount != 3 {
		t.Fatalf("damage=%v", d)
	}
}

func TestGearSixHeatBreaksThenBlastsThenResets(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	e := booted(out)
	ctx := unit.Context{ID: 1, Kind: KindEngine, Out: out}
	e.gear = 6
	e.hx, e.hy = 1, 0
	e.Handle(ctx, unit.Sense{Time: 6, Self: selfAt(0, 0, 300, 0)})
	cmds := drain(out)
	if !e.frail || e.gear != 6 || e.heat != 0 {
		t.Fatalf("frail=%v gear=%d heat=%v", e.frail, e.gear, e.heat)
	}
	if vel := lastVel(cmds); vel == nil || vel.VX != 0 || vel.VY != 0 {
		t.Fatalf("stall vel=%v", vel)
	}
	e.Handle(ctx, unit.Sense{
		Time:   6.5,
		Self:   selfAt(0, 0, 0, 0),
		Nearby: []unit.Snapshot{enemyAt(40, 0)},
	})
	if lastDamage(drain(out)) != nil {
		t.Fatal("虚弱 should not 普通攻击")
	}
	if e.heat != 0 {
		t.Fatalf("虚弱 heat=%v", e.heat)
	}
	e.Handle(ctx, unit.Sense{
		Time:   7.0,
		Self:   selfAt(0, 0, 0, 0),
		Nearby: []unit.Snapshot{enemyAt(40, 0)},
	})
	cmds = drain(out)
	d := lastDamage(cmds)
	if d == nil || d.Amount != 30 {
		t.Fatalf("blast=%v", d)
	}
	if lastNamed(cmds, "blast") == nil {
		t.Fatal("missing blast fx")
	}
	e.Handle(ctx, unit.Sense{Time: 8.0, Self: selfAt(0, 0, 0, 0)})
	cmds = drain(out)
	if e.frail || e.gear != 1 || e.heat != 0 {
		t.Fatalf("after frail gear=%d frail=%v heat=%v", e.gear, e.frail, e.heat)
	}
	if !hasCruise(cmds, 120) || !hasVision(cmds, 96) {
		t.Fatalf("reset cmds=%v", cmds)
	}
	vel := lastVel(cmds)
	if vel == nil || math.Abs(vel.VX-120) > 1e-6 || math.Abs(vel.VY) > 1e-6 {
		t.Fatalf("resume vel=%v", vel)
	}
}

func TestFrailIncomingStillConfirms(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	e := booted(out)
	e.frail = true
	e.Handle(unit.Context{ID: 1, Kind: KindEngine, Out: out}, unit.IncomingDamage{
		Token: 9, Amount: 12, Time: 1,
	})
	if e.heat != 0 {
		t.Fatalf("heat=%v", e.heat)
	}
	if _, ok := drain(out)[0].(unit.ConfirmDamage); !ok {
		t.Fatal("should still confirm hp loss")
	}
}

func booted(out chan unit.Cmd) *内燃机 {
	e := &内燃机{gear: 1}
	e.Handle(unit.Context{ID: 1, Kind: KindEngine, Out: out}, unit.Sense{
		Time: 0,
		Self: selfAt(0, 0, 120, 0),
	})
	_ = drain(out)
	return e
}

func selfAt(x, y, vx, vy float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindEngine, Role: unit.RoleFighter, Slot: 0,
		X: x, Y: y, VX: vx, VY: vy, Radius: engineRadius, Vision: 96,
	}
}

func enemyAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: x, Y: y, Radius: 18, AimPriority: unit.DefaultFighterAim}
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

func hasCruise(cmds []unit.Cmd, speed float64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.SetCruise); ok && math.Abs(v.Speed-speed) < 1e-9 {
			return true
		}
	}
	return false
}

func hasVision(cmds []unit.Cmd, vis float64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.SetVision); ok && math.Abs(v.Vision-vis) < 1e-9 {
			return true
		}
	}
	return false
}

func hasNoFrameFreeze(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.NoFrameFreeze); ok && v.Hold {
			return true
		}
	}
	return false
}

func hasDamage(cmds []unit.Cmd) bool {
	return lastDamage(cmds) != nil
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

func lastVel(cmds []unit.Cmd) *unit.SetVelocity {
	var vel *unit.SetVelocity
	for _, c := range cmds {
		if v, ok := c.(unit.SetVelocity); ok {
			cp := v
			vel = &cp
		}
	}
	return vel
}

func lastHeat(cmds []unit.Cmd) *unit.FX {
	return lastNamed(cmds, "heat")
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
