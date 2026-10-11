package 无下限术士

import (
	"math"
	"testing"
	"xqdj/internal/unit"
)

func TestBootHidesAndSplits(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &无下限术士{slot: 0}
	a.Handle(unit.Context{ID: 1, Kind: KindTwin, Out: out}, unit.Sense{
		Time: 0,
		Self: unit.Snapshot{ID: 1, Slot: 0, X: 10, Y: 0, VX: 200, VY: 0},
	})
	cmds := drain(out)
	if !hasNoHealthNumbers(cmds, 1) {
		t.Fatalf("missing NoHealthNumbers: %v", cmds)
	}
	if !hasAim(cmds, 1, 0) {
		t.Fatalf("missing SetAim 0: %v", cmds)
	}
	if !hasSpawn(cmds, KindRed) || !hasSpawn(cmds, KindBlue) {
		t.Fatalf("missing halves: %v", cmds)
	}
}

func TestHalfForwardsDamageToOwner(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	h := &半{owner: 1, slot: 0, red: true}
	h.Handle(unit.Context{ID: 2, Kind: KindRed, Out: out}, unit.IncomingDamage{
		Token: 9, From: 99, Amount: 8,
	})
	cmds := drain(out)
	if !hasConfirm(cmds, 9, 2, 8) {
		t.Fatalf("missing confirm: %v", cmds)
	}
	if !hasDamage(cmds, 99, 1, 8) {
		t.Fatalf("forward From should stay attacker: %v", cmds)
	}
}

func drain(out chan unit.Cmd) []unit.Cmd {
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

func hasNoHealthNumbers(cmds []unit.Cmd, id uint64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.NoHealthNumbers); ok && v.UnitID == id && v.Hold {
			return true
		}
	}
	return false
}

func hasAim(cmds []unit.Cmd, id uint64, v uint8) bool {
	for _, c := range cmds {
		if s, ok := c.(unit.SetAimPriority); ok && s.UnitID == id && s.Value == v {
			return true
		}
	}
	return false
}

func hasSpawn(cmds []unit.Cmd, kind string) bool {
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == kind {
			return true
		}
	}
	return false
}

func hasConfirm(cmds []unit.Cmd, token, id uint64, amt float64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.ConfirmDamage); ok && v.Token == token && v.UnitID == id && v.Amount == amt {
			return true
		}
	}
	return false
}

func hasDamage(cmds []unit.Cmd, from, to uint64, amt float64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.Damage); ok && v.From == from && v.To == to && v.Amount == amt {
			return true
		}
	}
	return false
}

func halfSense(id uint64, red bool, x, y float64, sibX, sibY float64, t float64) (unit.Context, *半, chan unit.Cmd) {
	return halfSenseOpt(id, red, x, y, sibX, sibY, t, true)
}

// bootPullDone=true：跳过开局那一次无视距离互吸，只测距离带互吸。
func halfSenseOpt(id uint64, red bool, x, y float64, sibX, sibY float64, t float64, bootPullDone bool) (unit.Context, *半, chan unit.Cmd) {
	out := make(chan unit.Cmd, 32)
	kind := KindBlue
	h := &半{owner: 1, slot: 0, red: red, bootPullDone: bootPullDone}
	if red {
		kind = KindRed
	}
	sibKind := KindBlue
	if !red {
		sibKind = KindRed
	}
	ctx := unit.Context{ID: id, Kind: kind, Out: out}
	h.Handle(ctx, unit.Sense{
		Time: t,
		Self: unit.Snapshot{ID: id, Kind: kind, Slot: 0, X: x, Y: y, VX: twinSpeed, VY: 0, Radius: twinRadius},
		Nearby: []unit.Snapshot{{
			ID: id + 1, Kind: sibKind, Slot: 0, X: sibX, Y: sibY, Radius: twinRadius,
		}},
	})
	return ctx, h, out
}

func TestAttractDisabledDuringBootGrace(t *testing.T) {
	_, h, out := halfSenseOpt(2, true, 0, 0, 50, 0, 0, false)
	cmds := drain(out)
	if h.attracting || lastVel(cmds, 2) != nil {
		t.Fatal("boot grace: must not attract")
	}
	if !hasChantFX(cmds, 2, 0) {
		t.Fatalf("red should chant 【九纲】 at t=0: %v", cmds)
	}
}

func TestBootChantAlternates(t *testing.T) {
	// step1：蓝半念【偏光】，红半应熄字（新半尚未点亮则无 clear FX）。
	_, hRed, outRed := halfSenseOpt(2, true, 0, 0, 50, 0, chantGap, false)
	cmdsRed := drain(outRed)
	if hRed.chantOn || hasChantFX(cmdsRed, 2, 1) {
		t.Fatalf("step1 is blue's turn; red must not chant: %v", cmdsRed)
	}
	_, hBlue, outBlue := halfSenseOpt(3, false, 50, 0, 0, 0, chantGap, false)
	cmdsBlue := drain(outBlue)
	if !hBlue.chantOn || !hasChantFX(cmdsBlue, 3, 1) {
		t.Fatalf("step1 blue should chant 【偏光】: %v", cmdsBlue)
	}
}

func TestBootPullIgnoresDistanceOnce(t *testing.T) {
	_, h, out := halfSenseOpt(2, true, 0, 0, 200, 0, bootGrace, false)
	cmds := drain(out)
	if !h.attracting || !h.bootPulling || h.bootPullDone {
		t.Fatal("after grace: once-only ignore-distance pull")
	}
	v := lastVel(cmds, 2)
	if v == nil || math.Abs(v.VX-twinSpeed) > 1e-6 {
		t.Fatalf("want vel toward far sibling: %+v cmds=%v", v, cmds)
	}
	h.Handle(unit.Context{ID: 2, Kind: KindRed, Out: out}, unit.Collision{
		Time:  bootGrace + 0.5,
		Other: unit.Snapshot{ID: 3, Kind: KindBlue, Slot: 0, X: 36.1, Y: 0},
	})
	drain(out)
	if h.attracting || h.bootPulling || !h.bootPullDone {
		t.Fatal("sibling collision ends opening pull")
	}

	_, hFar, outFar := halfSenseOpt(4, true, 0, 0, 200, 0, bootGrace+0.6, true)
	hFar.attractReadyAt = bootGrace + 0.5 + attractCD
	hFar.Handle(unit.Context{ID: 4, Kind: KindRed, Out: outFar}, unit.Sense{
		Time: bootGrace + 0.6,
		Self: unit.Snapshot{ID: 4, Kind: KindRed, Slot: 0, X: 0, Y: 0, Radius: twinRadius},
		Nearby: []unit.Snapshot{{
			ID: 5, Kind: KindBlue, Slot: 0, X: 200, Y: 0, Radius: twinRadius,
		}},
	})
	if hFar.attracting || lastVel(drain(outFar), 4) != nil {
		t.Fatal("after opening pull, far distance must use band rules")
	}
}

func TestAttractPullsWhenInBand(t *testing.T) {
	_, h, out := halfSense(2, true, 0, 0, 50, 0, bootGrace)
	cmds := drain(out)
	if !h.attracting {
		t.Fatal("should start attracting in (36,80]")
	}
	v := lastVel(cmds, 2)
	if v == nil {
		t.Fatalf("missing SetVelocity: %v", cmds)
	}
	if math.Abs(v.VX-twinSpeed) > 1e-6 || math.Abs(v.VY) > 1e-6 {
		t.Fatalf("want vel toward sibling (+x): %+v", v)
	}
	if !hasAttractFX(cmds, 2, 1) {
		t.Fatalf("missing twin-attract on: %v", cmds)
	}
}

func TestAttractSkipsWhenTooCloseOrFar(t *testing.T) {
	_, hClose, outClose := halfSense(2, true, 0, 0, 36, 0, bootGrace)
	if hClose.attracting || lastVel(drain(outClose), 2) != nil {
		t.Fatal("d=36 is not in (36,80]; must not attract")
	}
	_, hFar, outFar := halfSense(2, true, 0, 0, 81, 0, bootGrace)
	if hFar.attracting || lastVel(drain(outFar), 2) != nil {
		t.Fatal("d=81 is outside band; must not attract")
	}
}

func TestAttractEndsOnSiblingCollision(t *testing.T) {
	_, h, out := halfSense(2, true, 0, 0, 50, 0, bootGrace)
	drain(out)
	if !h.attracting {
		t.Fatal("setup: should be attracting")
	}
	hitAt := bootGrace + 0.05
	h.Handle(unit.Context{ID: 2, Kind: KindRed, Out: out}, unit.Collision{
		Time:  hitAt,
		Other: unit.Snapshot{ID: 3, Kind: KindBlue, Slot: 0, X: 36.1, Y: 0},
	})
	cmds := drain(out)
	if h.attracting {
		t.Fatal("sibling collision must end attract (semi rest gap stays >36)")
	}
	if math.Abs(h.attractReadyAt-(hitAt+attractCD)) > 1e-9 {
		t.Fatalf("readyAt=%v want %v", h.attractReadyAt, hitAt+attractCD)
	}
	if !hasAttractFX(cmds, 2, 0) {
		t.Fatalf("missing twin-attract off: %v", cmds)
	}
}

func TestAttractEndsIntoCooldown(t *testing.T) {
	t0 := bootGrace
	_, h, out := halfSense(2, true, 0, 0, 50, 0, t0)
	drain(out)
	if !h.attracting {
		t.Fatal("setup: should be attracting")
	}
	endAt := t0 + 0.2
	h.Handle(unit.Context{ID: 2, Kind: KindRed, Out: out}, unit.Sense{
		Time: endAt,
		Self: unit.Snapshot{ID: 2, Kind: KindRed, Slot: 0, X: 0, Y: 0, Radius: twinRadius},
		Nearby: []unit.Snapshot{{
			ID: 3, Kind: KindBlue, Slot: 0, X: 30, Y: 0, Radius: twinRadius,
		}},
	})
	cmds := drain(out)
	if h.attracting {
		t.Fatal("d<=36 should end attract")
	}
	if math.Abs(h.attractReadyAt-(endAt+attractCD)) > 1e-9 {
		t.Fatalf("readyAt=%v want %v", h.attractReadyAt, endAt+attractCD)
	}
	if !hasAttractFX(cmds, 2, 0) {
		t.Fatalf("missing twin-attract off: %v", cmds)
	}
	if lastVel(cmds, 2) != nil {
		t.Fatalf("must not rewrite velocity after end: %v", cmds)
	}

	h.Handle(unit.Context{ID: 2, Kind: KindRed, Out: out}, unit.Sense{
		Time: endAt + attractCD - 0.01,
		Self: unit.Snapshot{ID: 2, Kind: KindRed, Slot: 0, X: 0, Y: 0, Radius: twinRadius},
		Nearby: []unit.Snapshot{{
			ID: 3, Kind: KindBlue, Slot: 0, X: 50, Y: 0, Radius: twinRadius,
		}},
	})
	drain(out)
	if h.attracting {
		t.Fatal("still on cooldown; must not attract")
	}

	h.Handle(unit.Context{ID: 2, Kind: KindRed, Out: out}, unit.Sense{
		Time: endAt + attractCD,
		Self: unit.Snapshot{ID: 2, Kind: KindRed, Slot: 0, X: 0, Y: 0, Radius: twinRadius},
		Nearby: []unit.Snapshot{{
			ID: 3, Kind: KindBlue, Slot: 0, X: 50, Y: 0, Radius: twinRadius,
		}},
	})
	cmds = drain(out)
	if !h.attracting {
		t.Fatal("cooldown done; should attract again")
	}
	if lastVel(cmds, 2) == nil {
		t.Fatalf("missing SetVelocity after cooldown: %v", cmds)
	}
}

func lastVel(cmds []unit.Cmd, id uint64) *unit.SetVelocity {
	var v *unit.SetVelocity
	for i := range cmds {
		if s, ok := cmds[i].(unit.SetVelocity); ok && s.UnitID == id {
			cp := s
			v = &cp
		}
	}
	return v
}

func hasAttractFX(cmds []unit.Cmd, id uint64, amount float64) bool {
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == "twin-attract" && fx.UnitID == id && fx.Amount == amount {
			return true
		}
	}
	return false
}

func hasChantFX(cmds []unit.Cmd, id uint64, amount float64) bool {
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == "twin-chant" && fx.UnitID == id && fx.Amount == amount {
			return true
		}
	}
	return false
}
