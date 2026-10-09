package 火山

import (
	"math"
	"testing"

	"xqdj/internal/unit"
)

func TestVolcanoFieldRegistered(t *testing.T) {
	f, ok := unit.LookupField(Name)
	if !ok {
		t.Fatal("missing 火山")
	}
	if f.Shape != unit.ShapeHex || f.Extent != unit.HexRadius || len(f.Walls) != 0 {
		t.Fatalf("volcano=%+v", f)
	}
	if f.BootKind != KindVolcano {
		t.Fatalf("boot=%s", f.BootKind)
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

func TestBootReadyWithoutExtraSpawn(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &火山{nextErupt: eruptFirst}
	ctx := unit.Context{ID: 1, Kind: KindVolcano, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0, Self: unit.Snapshot{ID: 1, Kind: KindVolcano, Slot: 0, Radius: craterR}})
	cmds := drain(out)
	for _, c := range cmds {
		if _, ok := c.(unit.Spawn); ok {
			t.Fatalf("crater is boot unit itself, unexpected spawn: %v", cmds)
		}
	}
	if !a.booted {
		t.Fatal("should boot")
	}
}

func TestEruptPlansThreeSpotsAndLands(t *testing.T) {
	unit.SetLiveField(Field())
	out := make(chan unit.Cmd, 64)
	a := &火山{nextErupt: 0}
	ctx := unit.Context{ID: 2, Kind: KindVolcano, Out: out}
	self := unit.Snapshot{ID: 2, Kind: KindVolcano, Slot: 0}
	a.Handle(ctx, unit.Sense{Time: 0, Self: self})
	cmds := drain(out)
	var warns, erupts int
	for _, c := range cmds {
		fx, ok := c.(unit.FX)
		if !ok {
			continue
		}
		switch fx.Name {
		case "warn":
			warns++
		case "erupt":
			erupts++
		}
	}
	if erupts < 1 || warns != eruptN {
		t.Fatalf("erupt=%d warn=%d cmds=%v", erupts, warns, cmds)
	}
	if !a.winding {
		t.Fatal("should be winding")
	}
	a.Handle(ctx, unit.Sense{Time: 0.2, Self: self})
	cmds = drain(out)
	erupts = 0
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == "erupt" {
			erupts++
		}
	}
	if erupts < 1 {
		t.Fatal("erupt should keep pulsing while winding")
	}

	a.Handle(ctx, unit.Sense{Time: windSec, Self: self})
	cmds = drain(out)
	var lobs int
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == "lob" {
			lobs++
		}
		if _, ok := c.(unit.Spawn); ok {
			t.Fatal("lava should not spawn before lob lands")
		}
	}
	if lobs != eruptN {
		t.Fatalf("lob=%d", lobs)
	}
	if len(a.lobs) != eruptN || len(a.puddles) != 0 {
		t.Fatalf("lobs=%d puddles=%d", len(a.lobs), len(a.puddles))
	}

	a.Handle(ctx, unit.Sense{Time: windSec + lobFlight, Self: self})
	cmds = drain(out)
	var lands, spawns int
	for _, c := range cmds {
		switch v := c.(type) {
		case unit.FX:
			if v.Name == "land" {
				lands++
			}
		case unit.Spawn:
			if v.Kind == KindLava {
				spawns++
			}
		}
	}
	if lands != eruptN || spawns != eruptN {
		t.Fatalf("land=%d spawn=%d", lands, spawns)
	}
	if len(a.puddles) != eruptN || len(a.lobs) != 0 {
		t.Fatalf("puddles=%d lobs=%d", len(a.puddles), len(a.lobs))
	}
	for i := 0; i < eruptN; i++ {
		if math.Hypot(a.spots[i].x, a.spots[i].y) < craterGap-1e-6 {
			t.Fatalf("spot overlaps crater: %v", a.spots[i])
		}
		for j := i + 1; j < eruptN; j++ {
			d := math.Hypot(a.spots[i].x-a.spots[j].x, a.spots[i].y-a.spots[j].y)
			if d < puddleGap-1e-6 {
				t.Fatalf("spots overlap: %v gap=%v", a.spots, d)
			}
		}
	}
}

func TestLavaDamagesAndSlowsOnce(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &火山{
		booted:    true,
		nextErupt: 999,
		hitAt:     map[uint64]float64{},
		slowTok:   map[uint64]uint64{},
	}
	ctx := unit.Context{ID: 3, Kind: KindVolcano, Out: out}
	foe := unit.Snapshot{
		ID: 10, Kind: "对手", Role: unit.RoleFighter, Slot: 1,
		X: 0, Y: 0, Radius: 18,
	}
	a.Handle(ctx, unit.Sense{
		Time: 1, Self: unit.Snapshot{ID: 3, Kind: KindVolcano, Slot: 0},
		Nearby: []unit.Snapshot{foe},
	})
	cmds := drain(out)
	var dmg, slow int
	for _, c := range cmds {
		switch v := c.(type) {
		case unit.Damage:
			if v.To == 10 && v.Amount == dmgAmt && v.NoFreeze && v.From == 0 {
				dmg++
			}
		case unit.AddFSComponent:
			if v.UnitID == 10 && v.Zone == unit.FSZoneM && v.Value == lavaMul {
				slow++
			}
		}
	}
	if dmg != 1 || slow != 1 {
		t.Fatalf("dmg=%d slow=%d cmds=%v", dmg, slow, cmds)
	}

	a.Handle(ctx, unit.Sense{
		Time: 1.2, Self: unit.Snapshot{ID: 3, Kind: KindVolcano, Slot: 0},
		Nearby: []unit.Snapshot{foe},
	})
	cmds = drain(out)
	for _, c := range cmds {
		if _, ok := c.(unit.Damage); ok {
			t.Fatal("should not damage again within gap")
		}
	}

	a.Handle(ctx, unit.Sense{
		Time: 1.5, Self: unit.Snapshot{ID: 3, Kind: KindVolcano, Slot: 0},
		Nearby: []unit.Snapshot{foe},
	})
	cmds = drain(out)
	dmg = 0
	for _, c := range cmds {
		if _, ok := c.(unit.Damage); ok {
			dmg++
		}
	}
	if dmg != 1 {
		t.Fatalf("second tick dmg=%d", dmg)
	}
}

func TestOverlappingPuddlesSingleTick(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &火山{
		booted:    true,
		nextErupt: 999,
		hitAt:     map[uint64]float64{},
		slowTok:   map[uint64]uint64{},
		puddles:   []puddle{{x: 10, y: 0, until: 99}, {x: 20, y: 0, until: 99}},
	}
	ctx := unit.Context{ID: 4, Kind: KindVolcano, Out: out}
	foe := unit.Snapshot{
		ID: 11, Kind: "对手", Role: unit.RoleFighter, Slot: 0,
		X: 15, Y: 0, Radius: 18,
	}
	a.Handle(ctx, unit.Sense{
		Time: 2, Self: unit.Snapshot{ID: 4, Kind: KindVolcano, Slot: 0},
		Nearby: []unit.Snapshot{foe},
	})
	cmds := drain(out)
	n := 0
	for _, c := range cmds {
		if _, ok := c.(unit.Damage); ok {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("want 1 damage, got %d", n)
	}
}
