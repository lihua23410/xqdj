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
	if heat := lastHeat(cmds); heat == nil || heat.Amount != 1 {
		t.Fatalf("heat=%v", heat)
	}
}

func TestWallHitRaisesGearAndStats(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	e := booted(out)
	e.Handle(unit.Context{ID: 1, Kind: KindEngine, Out: out}, unit.WallHit{
		Time: 1, NX: -1, NY: 0,
	})
	cmds := drain(out)
	if e.gear != 2 {
		t.Fatalf("gear=%d", e.gear)
	}
	if !hasCruise(cmds, 156) || !hasVision(cmds, 120) {
		t.Fatalf("cmds=%v", cmds)
	}
	vel := lastVel(cmds)
	if vel == nil || math.Abs(math.Hypot(vel.VX, vel.VY)-156) > 1e-6 {
		t.Fatalf("vel=%v", vel)
	}
}

func TestWallHitCapsAtSix(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	e := booted(out)
	ctx := unit.Context{ID: 1, Kind: KindEngine, Out: out}
	for i := 0; i < 8; i++ {
		e.Handle(ctx, unit.WallHit{Time: float64(i) + 1, NX: -1, NY: 0})
		_ = drain(out)
	}
	if e.gear != 6 {
		t.Fatalf("gear=%d", e.gear)
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

func TestWeaknessBreakStallsThenBlastsThenResets(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	e := booted(out)
	ctx := unit.Context{ID: 1, Kind: KindEngine, Out: out}
	e.gear = 3
	e.hx, e.hy = 1, 0
	e.Handle(ctx, unit.IncomingDamage{Token: 1, Amount: 40, Time: 2})
	cmds := drain(out)
	if !e.frail || e.gear != 3 || e.weakness != 0 {
		t.Fatalf("frail=%v gear=%d weak=%v", e.frail, e.gear, e.weakness)
	}
	if vel := lastVel(cmds); vel == nil || vel.VX != 0 || vel.VY != 0 {
		t.Fatalf("stall vel=%v", vel)
	}
	e.Handle(ctx, unit.Sense{
		Time:   2.5,
		Self:   selfAt(0, 0, 0, 0),
		Nearby: []unit.Snapshot{enemyAt(40, 0)},
	})
	if lastDamage(drain(out)) != nil {
		t.Fatal("虚弱 should not 普通攻击")
	}
	e.Handle(ctx, unit.Sense{
		Time:   3.0,
		Self:   selfAt(0, 0, 0, 0),
		Nearby: []unit.Snapshot{enemyAt(40, 0)},
	})
	cmds = drain(out)
	d := lastDamage(cmds)
	if d == nil || d.Amount != 15 {
		t.Fatalf("blast=%v", d)
	}
	if lastNamed(cmds, "blast") == nil {
		t.Fatal("missing blast fx")
	}
	e.Handle(ctx, unit.Sense{Time: 4.0, Self: selfAt(0, 0, 0, 0)})
	cmds = drain(out)
	if e.frail || e.gear != 1 {
		t.Fatalf("after frail gear=%d frail=%v", e.gear, e.frail)
	}
	if !hasCruise(cmds, 120) || !hasVision(cmds, 96) {
		t.Fatalf("reset cmds=%v", cmds)
	}
	vel := lastVel(cmds)
	if vel == nil || math.Abs(vel.VX-120) > 1e-6 || math.Abs(vel.VY) > 1e-6 {
		t.Fatalf("resume vel=%v", vel)
	}
}

func TestGearUpOverflowBreaks(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	e := booted(out)
	e.gear = 3
	e.weakness = 40
	e.hx, e.hy = 1, 0
	e.Handle(unit.Context{ID: 1, Kind: KindEngine, Out: out}, unit.WallHit{
		Time: 1, NX: -1, NY: 0,
	})
	if e.gear != 4 || !e.frail {
		t.Fatalf("gear=%d frail=%v", e.gear, e.frail)
	}
}

func TestFrailIncomingDoesNotFillWeakness(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	e := booted(out)
	e.frail = true
	e.Handle(unit.Context{ID: 1, Kind: KindEngine, Out: out}, unit.IncomingDamage{
		Token: 9, Amount: 12, Time: 1,
	})
	if e.weakness != 0 {
		t.Fatalf("weakness=%v", e.weakness)
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
