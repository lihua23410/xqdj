package 非因果

import (
	"math"
	"testing"

	"xqdj/internal/unit"
)

func TestBurstSpectrumIsHigh(t *testing.T) {
	const n = 120 // 2 秒
	drops := make([]float64, n)
	drops[n-1] = 100
	bins := Magnitudes(drops)
	if len(bins) != n/2+1 {
		t.Fatalf("bins %d", len(bins))
	}
	for i, h := range bins {
		if math.Abs(h-50) > 0.05 {
			t.Fatalf("bin %d height %v want 50", i, h)
		}
	}
	_, dmg, speed := Draw(bins, 0)
	if math.Abs(dmg-150) > 0.05 {
		t.Fatalf("dmg %v", dmg)
	}
	if speed < speedMin-1e-6 || speed > speedMax+1e-6 {
		t.Fatalf("speed %v", speed)
	}
}

func TestGrindSpectrumIsLow(t *testing.T) {
	const n = 2400 // 40 秒
	drops := make([]float64, n)
	for i := range drops {
		drops[i] = 100 / float64(n)
	}
	bins := Magnitudes(drops)
	if math.Abs(bins[0]-2.5) > 0.05 {
		t.Fatalf("dc %v", bins[0])
	}
	for i := 1; i < len(bins); i++ {
		if bins[i] > 0.05 {
			t.Fatalf("bin %d %v", i, bins[i])
		}
	}
	_, dmg, speed := Draw(bins, 0.5)
	if math.Abs(dmg-7.5) > 0.05 {
		t.Fatalf("dmg %v", dmg)
	}
	if math.Abs(speed-speedMin) > 1e-6 {
		t.Fatalf("speed %v", speed)
	}
}

func TestDrawPrefersTallColumn(t *testing.T) {
	bins := make([]float64, 2000)
	for i := range bins {
		bins[i] = 1
	}
	bins[0] = 100
	tall := 0
	for i := 0; i < 200; i++ {
		_, dmg, _ := Draw(bins, float64(i)/199)
		if dmg > 100 {
			tall++
		}
	}
	if tall < 150 {
		t.Fatalf("tall draws %d/200", tall)
	}
}

func TestHealDoesNotEnterDrops(t *testing.T) {
	bins := Magnitudes([]float64{0, 0, 0})
	_, dmg, _ := Draw(bins, 0.5)
	if dmg != 0 {
		t.Fatalf("dmg %v", dmg)
	}
}

func TestWallStopsBeforeGhost(t *testing.T) {
	kind, id := stepShot(0, 0, 40, 0, 5,
		[]circle{{id: 9, x: 40, y: 0, r: 18, kind: hitFighterGhost}},
		[]wallSeg{{x1: 10, y1: -30, x2: 10, y2: 30, r: 2}},
		nil)
	if kind != hitWall || id != 0 {
		t.Fatalf("kind %d id %d", kind, id)
	}
}

func TestGhostHitSpendsShadowNotBody(t *testing.T) {
	kind, id := stepShot(0, 0, 40, 0, 5,
		[]circle{
			{id: 3, x: 80, y: 0, r: 18, kind: hitBody},
			{id: 9, x: 20, y: 0, r: 18, kind: hitFighterGhost},
		},
		nil, nil)
	if kind != hitFighterGhost || id != 9 {
		t.Fatalf("kind %d id %d", kind, id)
	}
}

func TestDoomStopsAfterDeath(t *testing.T) {
	tape := &Tape{
		DeathTime: 5,
		Bins:      []float64{1},
		Fighters: []FighterTrack{{
			Slot: 1, Seat: 0, Kind: "靶子", MaxHP: 10, Radius: 18,
			Samples: []Sample{{X: 80, Y: 0}},
		}},
	}
	a := &非因果{tape: tape}
	out := make(chan unit.Cmd, 128)
	ctx := unit.Context{ID: 7, Kind: KindAcausal, Out: out, Stream: 4}.WithBirths()
	sense := func(now float64) unit.Sense {
		return unit.Sense{
			Time: now,
			Self: unit.Snapshot{ID: 7, HP: 80, MaxHP: 100, Slot: 0, Role: unit.RoleFighter},
			Nearby: []unit.Snapshot{{
				ID: 99, Role: unit.RoleFighter, Slot: 1, Seat: 0, HP: 100, Radius: 18, Kind: "靶子",
			}},
			Field: unit.Field{Shape: unit.ShapeHex, Extent: unit.HexRadius},
		}
	}
	a.Handle(ctx, sense(0.1))
	drain(out)
	if len(a.shadow) != 1 || a.shadow[0] != 10 {
		t.Fatalf("shadow %v", a.shadow)
	}
	a.shadow[0] = 0
	a.Handle(ctx, sense(0.2))
	if !drainHas(out, 99, doomDamage) {
		t.Fatal("want 9999 while alive")
	}
	a.Handle(ctx, sense(5))
	sawDoom := false
	sawSuicide := false
	for {
		select {
		case c := <-out:
			d, ok := c.(unit.Damage)
			if !ok {
				continue
			}
			if d.To == 7 && d.Amount == 80 {
				sawSuicide = true
			}
			if d.To == 99 && d.Amount == doomDamage {
				sawDoom = true
			}
		default:
			if !sawSuicide {
				t.Fatal("want lethal self damage at the death time")
			}
			if !sawDoom {
				t.Fatal("9999 still goes out on the death frame")
			}
			return
		}
	}
}

func drain(out <-chan unit.Cmd) {
	for {
		select {
		case <-out:
		default:
			return
		}
	}
}

func drainHas(out <-chan unit.Cmd, to uint64, amount float64) bool {
	found := false
	for {
		select {
		case c := <-out:
			if d, ok := c.(unit.Damage); ok && d.To == to && d.Amount == amount {
				found = true
			}
		default:
			return found
		}
	}
}

func TestClockSuicides(t *testing.T) {
	tape := &Tape{DeathTime: 0.2, Bins: []float64{10}}
	a := &非因果{tape: tape}
	out := make(chan unit.Cmd, 128)
	ctx := unit.Context{ID: 7, Kind: KindAcausal, Out: out, Stream: 3}.WithBirths()
	a.Handle(ctx, unit.Sense{
		Time:  1,
		Self:  unit.Snapshot{ID: 7, HP: 40, MaxHP: 100, Slot: 0, Role: unit.RoleFighter},
		Field: unit.Field{Shape: unit.ShapeHex, Extent: unit.HexRadius},
	})
	if !drainHas(out, 7, 40) {
		t.Fatal("want current HP as self damage")
	}
}
