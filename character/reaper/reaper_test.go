package 收割者

import (
	"math"
	"testing"
	"xqdj/internal/unit"
)

func TestHexHitIgnoresInterior(t *testing.T) {
	if _, ok := hexHit(0, 0, 1, 0, 18); ok {
		t.Fatal("center should not be 场边")
	}
}

func TestHexHitAcceptsFace(t *testing.T) {
	nx, ny := hexNormal(0)
	ap := unit.HexRadius * math.Sqrt(3) / 2
	x, y := nx*(ap-18), ny*(ap-18)
	side, ok := hexHit(x, y, nx, ny, 18)
	if !ok || side != 0 {
		t.Fatalf("side=%d ok=%v", side, ok)
	}
}

func TestWallHitPlantsSickleOnHexEdge(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	e := &收割者{}
	nx, ny := hexNormal(2)
	ap := unit.HexRadius * math.Sqrt(3) / 2
	e.x, e.y = nx*(ap-18), ny*(ap-18)
	e.Handle(unit.Context{ID: 1, Kind: KindReaper, Out: out}, unit.WallHit{
		Time: 1, NX: nx, NY: ny,
	})
	cmds := drain(out)
	sp := lastSpawn(cmds)
	if sp == nil || sp.Kind != KindSickle {
		t.Fatalf("spawn=%v", cmds)
	}
	if e.sides[2] != 1 {
		t.Fatalf("sides=%v", e.sides)
	}
}

func TestCapsuleWallDoesNotPlant(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	e := &收割者{x: 10, y: 10}
	e.Handle(unit.Context{ID: 1, Kind: KindReaper, Out: out}, unit.WallHit{
		Time: 1, NX: 1, NY: 0,
	})
	if lastSpawn(drain(out)) != nil || e.sides != [6]int{} {
		t.Fatal("capsule must not plant")
	}
}

func TestSameSideStacks(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	e := &收割者{}
	nx, ny := hexNormal(1)
	ap := unit.HexRadius * math.Sqrt(3) / 2
	e.x, e.y = nx*(ap-18), ny*(ap-18)
	ctx := unit.Context{ID: 1, Kind: KindReaper, Out: out}
	e.Handle(ctx, unit.WallHit{Time: 1, NX: nx, NY: ny})
	_ = drain(out)
	e.Handle(ctx, unit.WallHit{Time: 2, NX: nx, NY: ny})
	_ = drain(out)
	if e.sides[1] != 2 {
		t.Fatalf("stack=%d", e.sides[1])
	}
}

func TestSixSidesRecall(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	e := &收割者{sides: [6]int{1, 1, 1, 2, 1, 1}}
	nearby := make([]unit.Snapshot, 0, 7)
	for i := 0; i < 6; i++ {
		nx, ny := hexNormal(i)
		nearby = append(nearby, unit.Snapshot{
			ID: uint64(10 + i), Kind: KindSickle, OwnerID: 1, Slot: 0,
			X: nx * 200, Y: ny * 200, Radius: sickleRadius,
		})
	}
	e.Handle(unit.Context{ID: 1, Kind: KindReaper, Out: out}, unit.Sense{
		Time:   3,
		Self:   unit.Snapshot{ID: 1, Kind: KindReaper, Role: unit.RoleFighter, Slot: 0},
		Nearby: nearby,
	})
	if len(drain(out)) != 0 {
		t.Fatal("收割者 itself should not spawn")
	}
	if e.sides != [6]int{} {
		t.Fatalf("sides after reap=%v", e.sides)
	}
	nReap := 0
	for i := 0; i < 6; i++ {
		kout := make(chan unit.Cmd, 8)
		k := &镰刀{owner: 1, slot: 0}
		k.Handle(unit.Context{ID: uint64(10 + i), Kind: KindSickle, Out: kout}, unit.Sense{
			Time: 3,
			Self: nearby[i],
		})
		cmds := drain(kout)
		sp := lastSpawn(cmds)
		if sp == nil || sp.Kind != KindReap {
			t.Fatalf("sickle %d spawn=%v", i, cmds)
		}
		if !hasDespawn(cmds, uint64(10+i)) {
			t.Fatalf("sickle %d should despawn", i)
		}
		nReap++
	}
	if nReap != 6 {
		t.Fatalf("reap=%d", nReap)
	}
}

func TestSickleCutsWithCD(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	k := &镰刀{owner: 1, slot: 0}
	ctx := unit.Context{ID: 9, Kind: KindSickle, Out: out}
	enemy := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 10, Y: 0, Radius: 18}
	k.Handle(ctx, unit.Sense{
		Time: 0,
		Self: unit.Snapshot{ID: 9, X: 0, Y: 0, Radius: 18, Slot: 0},
		Nearby: []unit.Snapshot{
			{ID: 1, Role: unit.RoleFighter, Slot: 0, X: 80, Y: 0, Radius: 18},
			enemy,
		},
	})
	d := lastDamage(drain(out))
	if d == nil || d.Amount != 6 || d.To != 2 {
		t.Fatalf("damage=%v", d)
	}
	k.Handle(ctx, unit.Sense{
		Time:   0.3,
		Self:   unit.Snapshot{ID: 9, X: 0, Y: 0, Radius: 18, Slot: 0},
		Nearby: []unit.Snapshot{enemy},
	})
	if lastDamage(drain(out)) != nil {
		t.Fatal("0.3s still in CD")
	}
	k.Handle(ctx, unit.Sense{
		Time:   1.0,
		Self:   unit.Snapshot{ID: 9, X: 0, Y: 0, Radius: 18, Slot: 0},
		Nearby: []unit.Snapshot{enemy},
	})
	if lastDamage(drain(out)) == nil {
		t.Fatal("1.0s should cut")
	}
}

func TestCutTurnsBlood(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	k := &镰刀{owner: 1, slot: 0}
	ctx := unit.Context{ID: 9, Kind: KindSickle, Out: out}
	k.Handle(ctx, unit.Sense{
		Time: 0,
		Self: unit.Snapshot{ID: 9, X: 0, Y: 0, Radius: 18, Slot: 0},
	})
	fx := lastBlood(drain(out))
	if fx == nil || fx.Amount != 0 {
		t.Fatalf("clean blood=%v", fx)
	}
	k.Handle(ctx, unit.Sense{
		Time: 0,
		Self: unit.Snapshot{ID: 9, X: 0, Y: 0, Radius: 18, Slot: 0},
		Nearby: []unit.Snapshot{{
			ID: 2, Role: unit.RoleFighter, Slot: 1, X: 10, Y: 0, Radius: 18,
		}},
	})
	fx = lastBlood(drain(out))
	if fx == nil || fx.Amount != 1 || fx.UnitID != 9 {
		t.Fatalf("hit blood=%v", fx)
	}
}

func TestReapDespawnsOnOwner(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	k := &镰刀{owner: 1, slot: 0, recalling: true}
	k.Handle(unit.Context{ID: 9, Kind: KindReap, Out: out}, unit.Sense{
		Time: 1,
		Self: unit.Snapshot{ID: 9, X: 0, Y: 0, Radius: 36, Slot: 0},
		Nearby: []unit.Snapshot{{
			ID: 1, Role: unit.RoleFighter, Slot: 0, X: 10, Y: 0, Radius: 18,
		}},
	})
	cmds := drain(out)
	if !hasDespawn(cmds, 9) {
		t.Fatal("should vanish on 收割者")
	}
	if lastHeal(cmds) != nil {
		t.Fatal("clean sickle should not heal")
	}
}

func TestParkedDoesNotDespawnOnOwner(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	k := &镰刀{owner: 1, slot: 0}
	k.Handle(unit.Context{ID: 9, Kind: KindSickle, Out: out}, unit.Sense{
		Time: 1,
		Self: unit.Snapshot{ID: 9, X: 0, Y: 0, Radius: 18, Slot: 0},
		Nearby: []unit.Snapshot{{
			ID: 1, Role: unit.RoleFighter, Slot: 0, X: 5, Y: 0, Radius: 18,
		}},
	})
	if hasDespawn(drain(out), 9) {
		t.Fatal("parked sickle must stay")
	}
}

func TestHitSickleHealsOnRecall(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	k := &镰刀{owner: 1, slot: 0, recalling: true, drew: true}
	k.Handle(unit.Context{ID: 9, Kind: KindReap, Out: out}, unit.Sense{
		Time: 1,
		Self: unit.Snapshot{ID: 9, X: 0, Y: 0, Radius: 36, Slot: 0},
		Nearby: []unit.Snapshot{{
			ID: 1, Role: unit.RoleFighter, Slot: 0, X: 10, Y: 0, Radius: 18,
		}},
	})
	h := lastHeal(drain(out))
	if h == nil || h.UnitID != 1 || h.Amount != 3 {
		t.Fatalf("heal=%v", h)
	}
}

func TestTwoCutsStillHeal3(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	k := &镰刀{owner: 1, slot: 0}
	ctx := unit.Context{ID: 9, Kind: KindSickle, Out: out}
	enemy := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 10, Y: 0, Radius: 18}
	self := unit.Snapshot{ID: 9, X: 0, Y: 0, Radius: 18, Slot: 0}
	k.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{enemy}})
	_ = drain(out)
	k.Handle(ctx, unit.Sense{Time: 0.4, Self: self, Nearby: []unit.Snapshot{enemy}})
	_ = drain(out)
	k.recalling = true
	k.Handle(unit.Context{ID: 9, Kind: KindReap, Out: out}, unit.Sense{
		Time: 1,
		Self: unit.Snapshot{ID: 9, X: 0, Y: 0, Radius: 36, Slot: 0},
		Nearby: []unit.Snapshot{{
			ID: 1, Role: unit.RoleFighter, Slot: 0, X: 10, Y: 0, Radius: 18,
		}},
	})
	heals := 0
	var amt float64
	for _, c := range drain(out) {
		if v, ok := c.(unit.Heal); ok {
			heals++
			amt += v.Amount
		}
	}
	if heals != 1 || amt != 3 {
		t.Fatalf("heals=%d amt=%v", heals, amt)
	}
}

func TestHitParkedCarriesHealOnTransform(t *testing.T) {
	recallingIDs.Store(uint64(21), struct{}{})
	out := make(chan unit.Cmd, 8)
	k := &镰刀{owner: 1, slot: 0, drew: true}
	k.Handle(unit.Context{ID: 21, Kind: KindSickle, Out: out}, unit.Sense{
		Time: 2,
		Self: unit.Snapshot{ID: 21, X: 40, Y: 0, Radius: 18, Slot: 0},
	})
	sp := lastSpawn(drain(out))
	if sp == nil || sp.Kind != KindReap || sp.VX != harvestCarry {
		t.Fatalf("carry spawn=%v", sp)
	}
	rout := make(chan unit.Cmd, 8)
	rk := &镰刀{owner: 1, slot: 0, recalling: true}
	rk.Handle(unit.Context{ID: 22, Kind: KindReap, Out: rout}, unit.Sense{
		Time: 2,
		Self: unit.Snapshot{ID: 22, X: 0, Y: 0, Radius: 36, Slot: 0, VX: harvestCarry},
		Nearby: []unit.Snapshot{{
			ID: 1, Role: unit.RoleFighter, Slot: 0, X: 10, Y: 0, Radius: 18,
		}},
	})
	h := lastHeal(drain(rout))
	if h == nil || h.Amount != 3 {
		t.Fatalf("carried heal=%v", h)
	}
}

func TestAcceptsIncomingDamage(t *testing.T) {
	out := make(chan unit.Cmd, 4)
	e := &收割者{}
	e.Handle(unit.Context{ID: 1, Kind: KindReaper, Out: out}, unit.IncomingDamage{
		Token: 3, Amount: 9, Time: 1,
	})
	if _, ok := drain(out)[0].(unit.ConfirmDamage); !ok {
		t.Fatal("should confirm")
	}
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

func lastHeal(cmds []unit.Cmd) *unit.Heal {
	var h *unit.Heal
	for _, c := range cmds {
		if v, ok := c.(unit.Heal); ok {
			cp := v
			h = &cp
		}
	}
	return h
}

func lastBlood(cmds []unit.Cmd) *unit.FX {
	var fx *unit.FX
	for _, c := range cmds {
		if v, ok := c.(unit.FX); ok && v.Name == "blood" {
			cp := v
			fx = &cp
		}
	}
	return fx
}

func hasDespawn(cmds []unit.Cmd, id uint64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.Despawn); ok && v.UnitID == id {
			return true
		}
	}
	return false
}
