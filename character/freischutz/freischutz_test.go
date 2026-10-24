package Freischutz

import (
	"math"
	"testing"

	"xqdj/internal/unit"
	_ "xqdj/map/hex"
)

func TestBodyIsDarkBlue(t *testing.T) {
	spec, ok := unit.Lookup(KindFrei)
	if !ok || spec.Look.Color != bodyColor {
		t.Fatalf("color=%v ok=%v", spec.Look.Color, ok)
	}
	shot, ok := unit.Lookup(KindFreiShot)
	if !ok || shot.Look.Trail {
		t.Fatalf("shot should not use the ghost trail, trail=%v ok=%v", shot.Look.Trail, ok)
	}
}

func TestShotAimsAtAPerson(t *testing.T) {
	resetLive()
	out := make(chan unit.Cmd, 16)
	a := &中转站{rng: unit.NewRNG(1)}
	self := unit.Snapshot{
		ID: 1, Kind: KindFrei, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: bodyRadius,
	}
	person := unit.Snapshot{
		ID: 2, Role: unit.RoleFighter, Slot: 1, AimPriority: 15,
		X: 0, Y: 100, Radius: 18,
	}
	a.Handle(ctx(out), unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{person}})
	if shotOf(drain(out)) != nil {
		t.Fatal("放站那一拍不该开枪")
	}
	a.Handle(ctx(out), unit.Sense{Time: fireDelay, Self: self, Nearby: []unit.Snapshot{person}})
	wind := drain(out)
	if shotOf(wind) != nil || !stood(wind) || !facing(wind, 0, 1) {
		t.Fatalf("windup should stand and aim, cmds=%+v", wind)
	}
	person.X, person.Y = 100, 0
	a.Handle(ctx(out), unit.Sense{Time: fireDelay + aimWind, Self: self, Nearby: []unit.Snapshot{person}})
	sp := shotOf(drain(out))
	if sp == nil || sp.X != self.X || sp.Y != self.Y || sp.VX <= 0 || math.Abs(sp.VY) > sp.VX {
		t.Fatalf("shot should leave the body toward the aimed person, spawn=%+v", sp)
	}
}

func TestOpeningScattersSixNodes(t *testing.T) {
	resetLive()
	out := make(chan unit.Cmd, 16)
	a := &中转站{rng: unit.NewRNG(7)}
	self := unit.Snapshot{
		ID: 1, Kind: KindFrei, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: bodyRadius,
	}
	a.Handle(ctx(out), unit.Sense{Time: 0, Self: self})
	var nodes []unit.Spawn
	for _, c := range drain(out) {
		sp, ok := c.(unit.Spawn)
		if ok && sp.Kind == KindFreiNode {
			nodes = append(nodes, sp)
		}
	}
	if len(nodes) != openingNodes {
		t.Fatalf("nodes=%d", len(nodes))
	}
	f := unit.HexField()
	for _, n := range nodes {
		if !f.Walkable(n.X, n.Y, nodeRadius) {
			t.Fatalf("node outside the field: %+v", n)
		}
	}
}

func TestMissReprisesOnceFromANode(t *testing.T) {
	resetLive()
	out := make(chan unit.Cmd, 8)
	b := &弹{owner: 1, slot: 0, damage: 8, used: map[uint64]bool{}, seen: true, rng: unit.NewRNG(1)}
	node := nodeAt(4, 10, 0)
	person := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 10, Y: 80, Radius: 18}
	b.Handle(ctx(out), unit.Sense{
		Self:   unit.Snapshot{X: 0, Y: -640, VX: 0, VY: 1040, Radius: shotRadius},
		Nearby: []unit.Snapshot{node, person},
	})
	sp := shotOf(drain(out))
	if sp == nil || sp.X != 10 || sp.Y != 0 {
		t.Fatalf("reprise spawn=%+v", sp)
	}
	if math.Abs(math.Hypot(sp.VX, sp.VY)-1040) > 1e-6 || sp.VY <= 0 {
		t.Fatalf("inherited velocity=%+v", sp)
	}
	b2 := &弹{owner: 1, slot: 0, damage: shotDamage, used: map[uint64]bool{}, rng: unit.NewRNG(2)}
	b2.Handle(ctx(out), unit.Sense{
		Self:   unit.Snapshot{X: sp.X, Y: sp.Y, VX: sp.VX, VY: sp.VY, Radius: shotRadius},
		Nearby: []unit.Snapshot{node, person},
	})
	if b2.damage != 8 || !b2.spent {
		t.Fatalf("damage=%v spent=%v", b2.damage, b2.spent)
	}
	drain(out)
	b2.Handle(ctx(out), unit.Sense{
		Self:   unit.Snapshot{X: 0, Y: -640, VX: sp.VX, VY: sp.VY, Radius: shotRadius},
		Nearby: []unit.Snapshot{node, person},
	})
	if shotOf(drain(out)) != nil {
		t.Fatal("second miss should not reprise")
	}
}

func TestRepriseDamageIsReadyAtBirth(t *testing.T) {
	resetLive()
	out := make(chan unit.Cmd, 4)
	parent := unit.Context{ID: 1, Kind: KindFrei, Out: out, Stream: 9}.WithBirths()
	b := &弹{owner: 1, slot: 0, damage: 8, used: map[uint64]bool{}, seen: true, rng: unit.NewRNG(1)}
	node := nodeAt(4, 10, 0)
	person := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 10, Y: 80, Radius: 18}
	b.Handle(parent, unit.Sense{
		Self:   unit.Snapshot{X: 0, Y: -640, VX: 0, VY: 1040, Radius: shotRadius},
		Nearby: []unit.Snapshot{node, person},
	})
	sp := shotOf(drain(out))
	if sp == nil || sp.Stream == 0 {
		t.Fatalf("spawn=%+v", sp)
	}
	born := unit.NewActor(KindFreiShot, unit.SpawnInfo{OwnerID: 1, Slot: 0, Stream: sp.Stream})
	shot, ok := born.(*弹)
	if !ok || shot.damage != 8 || !shot.spent {
		t.Fatalf("birth damage=%v spent=%v ok=%v", shot.damage, shot.spent, ok)
	}
}

func TestHitPersonDoesNotReprise(t *testing.T) {
	resetLive()
	out := make(chan unit.Cmd, 4)
	b := &弹{owner: 1, slot: 0, damage: 8, used: map[uint64]bool{}, seen: true, hitPerson: true, rng: unit.NewRNG(1)}
	node := nodeAt(4, 10, 0)
	person := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 10, Y: 80, Radius: 18}
	b.Handle(ctx(out), unit.Sense{
		Self:   unit.Snapshot{X: 0, Y: -640, VX: 0, VY: 1040},
		Nearby: []unit.Snapshot{node, person},
	})
	if shotOf(drain(out)) != nil {
		t.Fatal("a hit should not reprise")
	}
}

func TestPassDoublesSpeedAndDamageWithoutTurning(t *testing.T) {
	resetLive()
	out := make(chan unit.Cmd, 4)
	b := &弹{owner: 1, slot: 0, damage: shotDamage, used: map[uint64]bool{}, seen: true, lx: -40, ly: 0}
	node := unit.Snapshot{ID: 4, Kind: KindFreiNode, OwnerID: 1, X: 0, Y: 0, Radius: nodeRadius}
	self := unit.Snapshot{ID: 9, Kind: KindFreiShot, X: 40, Y: 0, VX: shotSpeed, VY: 0, Radius: shotRadius}
	b.Handle(ctx(out), unit.Sense{Self: self, Nearby: []unit.Snapshot{node}})
	v := onlyVel(t, drain(out))
	if math.Abs(v.VY) > 1e-9 || v.VX <= shotSpeed {
		t.Fatalf("velocity=%+v", v)
	}
	if math.Abs(math.Hypot(v.VX, v.VY)-shotSpeed*2) > 1e-6 {
		t.Fatalf("speed=%v", math.Hypot(v.VX, v.VY))
	}
	if b.damage != shotDamage*2 {
		t.Fatalf("damage=%v", b.damage)
	}
}

func TestLinearScanCatchesAJumpOverTheStation(t *testing.T) {
	resetLive()
	if !crossed(-80, 0, 80, 0, 0, 0, nodeRadius+shotRadius) {
		t.Fatal("segment through the station should hit")
	}
	if crossed(-80, 40, 80, 40, 0, 0, nodeRadius+shotRadius) {
		t.Fatal("segment that misses the station should not hit")
	}
	out := make(chan unit.Cmd, 4)
	n := &站{gate: newGate(1), slot: 0, booted: true}
	c := ctx(out)
	far := shotAt(20, -80, 0)
	n.Handle(c, unit.Sense{Time: 1, Self: nodeAt(4, 0, 0), Nearby: []unit.Snapshot{far}})
	if n.gate.left != durability || hasDespawn(drain(out)) {
		t.Fatalf("first sight should only record, left=%d", n.gate.left)
	}
	past := shotAt(20, 80, 0)
	n.Handle(c, unit.Sense{Time: 2, Self: nodeAt(4, 0, 0), Nearby: []unit.Snapshot{past}})
	if n.gate.left != durability-1 {
		t.Fatalf("jumped-over shot should spend one use, left=%d", n.gate.left)
	}
	if hasDespawn(drain(out)) {
		t.Fatal("one use should not remove the station")
	}
}

func TestStationVanishesAfterThreeShots(t *testing.T) {
	resetLive()
	out := make(chan unit.Cmd, 4)
	n := &站{gate: newGate(1), slot: 0, booted: true}
	c := ctx(out)
	self := nodeAt(4, 0, 0)
	for i := 0; i < durability; i++ {
		id := uint64(30 + i)
		n.Handle(c, unit.Sense{Time: float64(i), Self: self, Nearby: []unit.Snapshot{shotAt(id, -80, 0)}})
		drain(out)
		n.Handle(c, unit.Sense{Time: float64(i) + 0.5, Self: self, Nearby: []unit.Snapshot{shotAt(id, 80, 0)}})
		cmds := drain(out)
		if i < durability-1 {
			if hasDespawn(cmds) {
				t.Fatalf("use %d should keep the station", i+1)
			}
			continue
		}
		if !hasDespawn(cmds) {
			t.Fatal("third use should remove the station")
		}
	}
	if n.gate.left != 0 {
		t.Fatalf("left=%d", n.gate.left)
	}
}

func TestSameShotSpendsOneUse(t *testing.T) {
	resetLive()
	out := make(chan unit.Cmd, 4)
	n := &站{gate: newGate(1), slot: 0, booted: true}
	c := ctx(out)
	self := nodeAt(4, 0, 0)
	n.Handle(c, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{shotAt(20, -30, 0)}})
	drain(out)
	n.Handle(c, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{shotAt(20, 0, 0)}})
	drain(out)
	n.Handle(c, unit.Sense{Time: 2, Self: self, Nearby: []unit.Snapshot{shotAt(20, 4, 0)}})
	if hasDespawn(drain(out)) || n.gate.left != durability-1 {
		t.Fatalf("overlapping follow-up should not spend again, left=%d", n.gate.left)
	}
}

func ctx(out chan unit.Cmd) unit.Context {
	return unit.Context{ID: 1, Kind: KindFrei, Out: out}
}

func nodeAt(id uint64, x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: id, Kind: KindFreiNode, Role: unit.RoleHelper, OwnerID: 1,
		X: x, Y: y, Radius: nodeRadius,
	}
}

func shotAt(id uint64, x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: id, Kind: KindFreiShot, Role: unit.RoleProjectile, OwnerID: 1, Slot: 0,
		X: x, Y: y, Radius: shotRadius, VX: shotSpeed,
	}
}

func stood(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		v, ok := c.(unit.Stand)
		if ok && v.Hold {
			return true
		}
	}
	return false
}

func facing(cmds []unit.Cmd, ux, uy float64) bool {
	for _, c := range cmds {
		v, ok := c.(unit.FX)
		if !ok || v.Name != "face" {
			continue
		}
		return v.VX*uy-v.VY*ux == 0 && v.VX*ux+v.VY*uy > 0
	}
	return false
}

func shotOf(cmds []unit.Cmd) *unit.Spawn {
	for _, c := range cmds {
		v, ok := c.(unit.Spawn)
		if ok && v.Kind == KindFreiShot {
			return &v
		}
	}
	return nil
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

func onlyVel(t *testing.T, cmds []unit.Cmd) unit.SetVelocity {
	t.Helper()
	var v *unit.SetVelocity
	for _, c := range cmds {
		sv, ok := c.(unit.SetVelocity)
		if !ok {
			continue
		}
		if v != nil {
			t.Fatalf("two velocities: %v", cmds)
		}
		cp := sv
		v = &cp
	}
	if v == nil {
		t.Fatalf("missing velocity in %v", cmds)
	}
	return *v
}

func hasDespawn(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if _, ok := c.(unit.Despawn); ok {
			return true
		}
	}
	return false
}
