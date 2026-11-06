package 泵之心

import (
	"math"
	"testing"
	modules "xqdj/character/engine_heart/parts"
	"xqdj/character/engine_heart/parts/api"
	"xqdj/character/engine_heart/parts/arc"
	"xqdj/character/engine_heart/parts/ram"
	"xqdj/internal/unit"
)

func testWeapon(t *testing.T, id int) (unit.Actor, unit.Context, chan unit.Cmd, unit.Sense) {
	t.Helper()
	var actor unit.Actor
	for _, d := range modules.Catalog() {
		if d.ID == id && d.Weapon != nil {
			actor = d.Weapon()
		}
	}
	if actor == nil {
		t.Fatalf("missing weapon %d", id)
	}
	out := make(chan unit.Cmd, 512)
	ctx := unit.Context{ID: 1, Kind: KindEngineHeart, Out: out, State: &unit.MatchState{}}
	s := unit.Sense{Field: unit.HexField(), Self: unit.Snapshot{ID: 1, Role: unit.RoleFighter, Radius: 18, HP: 100, Slot: 0, VX: 150}, Nearby: []unit.Snapshot{{ID: 9, Role: unit.RoleFighter, Slot: 1, HP: 100, X: 60, Radius: 18}}}
	return actor, ctx, out, s
}
func drainDamage(out chan unit.Cmd) map[uint64]float64 {
	result := map[uint64]float64{}
	for len(out) > 0 {
		if d, ok := (<-out).(unit.Damage); ok {
			result[d.To] += d.Amount
		}
	}
	return result
}
func TestEverySelectionIncludesAnAdditionalWeapon(t *testing.T) {
	weapons := map[int]bool{}
	for _, d := range modules.Catalog() {
		weapons[d.ID] = d.Weapon != nil
	}
	for seed := uint64(1); seed <= 500; seed++ {
		w := &world{}
		w.initialize(seed, unit.Sense{Field: unit.HexField(), Self: unit.Snapshot{X: -100, Radius: 18, HP: 100}})
		count := 0
		seen := map[int]bool{}
		for _, p := range w.parts {
			if seen[p.Type] {
				t.Fatal("duplicate component")
			}
			seen[p.Type] = true
			if weapons[p.Type] {
				count++
			}
			if !w.field.Walkable(p.X, p.Y, partClearance) {
				t.Fatal("full component footprint crosses field or wall")
			}
		}
		if len(w.parts) != 7 || count == 0 {
			t.Fatalf("seed %d has no guaranteed attack", seed)
		}
	}
}
func TestLaserWarnsThenAppliesExposureDamageAndStopsAtWalls(t *testing.T) {
	a, ctx, out, s := testWeapon(t, 11)
	s.Nearby[0].X = 100
	s.Nearby = append(s.Nearby, unit.Snapshot{ID: 10, Role: unit.RoleFighter, Slot: 1, HP: 100, X: 180, Radius: 18}, unit.Snapshot{ID: 11, Role: unit.RoleFighter, Slot: 0, HP: 100, X: 50, Radius: 18})
	s.Field.Walls = []unit.FieldWall{{Kind: unit.WallCapsule, X1: 120, Y1: -200, X2: 120, Y2: 200, Radius: 3}}
	a.Handle(ctx, api.Pulse{Sense: s})
	if len(drainDamage(out)) != 0 {
		t.Fatal("uncharged laser fired")
	}
	a.Handle(ctx, api.Pulse{Sense: s, Echo: true})
	a.Handle(ctx, api.Pulse{Sense: s})
	if d := drainDamage(out); len(d) != 0 {
		t.Fatalf("laser damaged during its warning: %v", d)
	}
	total := map[uint64]float64{}
	for _, now := range []float64{.28, .4, .52, .64, .76} {
		s.Time = now
		a.Handle(ctx, s)
		for id, d := range drainDamage(out) {
			total[id] += d
		}
	}
	if math.Abs(total[9]-8) > 1e-8 || total[10] != 0 || total[11] != 0 {
		t.Fatalf("sustained beam crossed wall, hit ally or failed exposure: %v", total)
	}
	s.Time = .9
	a.Handle(ctx, s)
	drainDamage(out)
	s.Field.Walls = nil
	s.Time = 2
	a.Handle(ctx, api.Pulse{Sense: s})
	a.Handle(ctx, api.Pulse{Sense: s})
	drainDamage(out)
	s.Time = 2.28
	a.Handle(ctx, s)
	if d := drainDamage(out); d[9] != 1.6 || d[10] != 1.6 || d[11] != 0 {
		t.Fatalf("continuous beam failed to pierce aligned enemies: %v", d)
	}
}

func TestLaserLocksDirectionAndCanBeDodgedOrEnteredLate(t *testing.T) {
	a, ctx, out, s := testWeapon(t, 11)
	s.Nearby[0].X = 160
	a.Handle(ctx, api.Pulse{Sense: s})
	a.Handle(ctx, api.Pulse{Sense: s})
	drainDamage(out)
	s.Nearby[0].Y = 80
	s.Time = .28
	a.Handle(ctx, s)
	for len(out) > 0 {
		switch c := (<-out).(type) {
		case unit.Damage:
			t.Fatal("laser tracked a target that dodged its warning")
		case unit.FX:
			if c.Name == "heart-laser" && math.Abs(c.VY-c.Y) > 1e-8 {
				t.Fatal("beam axis turned after lock")
			}
		}
	}
	// Someone who was not on the initial aiming line can enter the ongoing beam.
	s.Nearby = append(s.Nearby, unit.Snapshot{ID: 20, Role: unit.RoleFighter, Slot: 1, HP: 100, X: 220, Radius: 18})
	s.Time = .4
	a.Handle(ctx, s)
	if d := drainDamage(out); d[9] != 0 || d[20] != 1.6 {
		t.Fatalf("exposure did not use current positions: %v", d)
	}
	for _, now := range []float64{.41, .42, .43, .44} {
		s.Time = now
		a.Handle(ctx, s)
	}
	if len(drainDamage(out)) != 0 {
		t.Fatal("continuous damage multiplied with Sense frequency")
	}
	s.Nearby[1].Y = 80
	s.Time = .52
	a.Handle(ctx, s)
	if len(drainDamage(out)) != 0 {
		t.Fatal("leaving the beam did not stop damage")
	}
}

func TestLaserHoldsChargeAtCloseRangeAndNeverFiresOnContact(t *testing.T) {
	a, ctx, out, s := testWeapon(t, 11)
	s.Nearby[0].X = 36
	for i := 0; i < 3; i++ {
		a.Handle(ctx, api.Pulse{Sense: s})
	}
	a.Handle(ctx, unit.Collision{Other: s.Nearby[0]})
	if len(out) != 0 {
		t.Fatal("long-range laser discharged as a contact weapon")
	}
	s.Nearby[0].X = 160
	a.Handle(ctx, api.Pulse{Sense: s})
	found := false
	for len(out) > 0 {
		if f, ok := (<-out).(unit.FX); ok && f.Name == "heart-laser-aim" {
			found = true
		}
	}
	if !found {
		t.Fatal("laser lost stored pressure while waiting for a distant firing opportunity")
	}
}
func TestRamRequiresActualTravelAndHitsOnceThenDetaches(t *testing.T) {
	a, ctx, out, s := testWeapon(t, 12)
	for i := 0; i < 3; i++ {
		a.Handle(ctx, api.Pulse{Sense: s})
	}
	if len(drainDamage(out)) != 0 {
		t.Fatal("ram damaged a remote target before moving")
	}
	s.Time = .1
	s.Self.X = 50
	a.Handle(ctx, s)
	deliverContact(t, a, ctx, out, s, ram.KindContact, unit.Collision{Time: .11, Other: s.Nearby[0]})
	deliverContact(t, a, ctx, out, s, ram.KindContact, unit.Collision{Time: .11, Other: s.Nearby[0]})
	if d := drainDamage(out); d[9] != 8 {
		t.Fatalf("ram repeated or missed collision: %v", d)
	}
	a.Handle(ctx, api.Detached{})
	drainDamage(out)
	deliverContact(t, a, ctx, out, s, ram.KindContact, unit.Collision{Time: .12, Other: unit.Snapshot{ID: 12, Role: unit.RoleFighter, Slot: 1, HP: 100}})
	if len(drainDamage(out)) != 0 {
		t.Fatal("removed ram kept causing impacts")
	}
}
func TestSteamStaysAtReleasePointAndEndsAfterThreeTicks(t *testing.T) {
	a, ctx, out, s := testWeapon(t, 13)
	a.Handle(ctx, api.Pulse{Sense: s})
	s.Self.X = 200
	s.Time = .12
	a.Handle(ctx, s)
	s.Time = .24
	a.Handle(ctx, s)
	s.Time = .4
	a.Handle(ctx, s)
	if d := drainDamage(out); d[9] != 4.5 {
		t.Fatalf("cloud followed owner or missed its fixed area: %v", d)
	}
	a.Handle(ctx, api.Pulse{Sense: s})
	a.Handle(ctx, api.Detached{})
	drainDamage(out)
	s.Time = .52
	a.Handle(ctx, s)
	if len(drainDamage(out)) != 0 {
		t.Fatal("removed cylinder continued emitting damage")
	}
}
func TestArcRequiresNearBeatsAndActualContactThenChains(t *testing.T) {
	a, ctx, out, s := testWeapon(t, 14)
	s.Nearby = append(s.Nearby, unit.Snapshot{ID: 10, Role: unit.RoleFighter, Slot: 1, HP: 100, X: 145, Radius: 18}, unit.Snapshot{ID: 11, Role: unit.RoleFighter, Slot: 1, HP: 100, X: 230, Radius: 18}, unit.Snapshot{ID: 12, Role: unit.RoleFighter, Slot: 0, HP: 100, X: 120, Radius: 18})
	a.Handle(ctx, api.Pulse{Sense: s})
	deliverContact(t, a, ctx, out, s, arc.KindContact, unit.Collision{Time: .1, Other: s.Nearby[0]})
	if len(drainDamage(out)) != 0 {
		t.Fatal("one proximity beat fully charged the coil")
	}
	a.Handle(ctx, api.Pulse{Sense: s})
	if len(drainDamage(out)) != 0 {
		t.Fatal("coil attacked remotely instead of waiting for contact")
	}
	deliverContact(t, a, ctx, out, s, arc.KindContact, unit.Collision{Time: .5, Other: s.Nearby[3]})
	if len(drainDamage(out)) != 0 {
		t.Fatal("ally contact consumed stored electricity")
	}
	victim := s.Nearby[0]
	victim.X = 36
	deliverContact(t, a, ctx, out, s, arc.KindContact, unit.Collision{Time: .5, Other: victim})
	d := map[uint64]float64{}
	stuns := 0
	for len(out) > 0 {
		switch c := (<-out).(type) {
		case unit.Damage:
			d[c.To] += c.Amount
		case unit.Stun:
			if c.UnitID != 9 || !c.Hold || math.Abs(c.Until-.68) > 1e-8 {
				t.Fatal("contact stun was not attached to the first victim")
			}
			stuns++
		}
	}
	if len(d) != 3 || d[9] != 6 || d[10] != 3 || d[11] != 2 || d[12] != 0 || stuns != 1 {
		t.Fatalf("invalid contact discharge or chain: %v stuns=%d", d, stuns)
	}
	deliverContact(t, a, ctx, out, s, arc.KindContact, unit.Collision{Time: .51, Other: victim})
	if len(drainDamage(out)) != 0 {
		t.Fatal("one stored charge discharged repeatedly")
	}
}

func TestArcCannotChargeAtRangeAndChainsCannotCrossWalls(t *testing.T) {
	a, ctx, out, s := testWeapon(t, 14)
	s.Nearby[0].X = 180
	for i := 0; i < 10; i++ {
		a.Handle(ctx, api.Pulse{Sense: s})
	}
	victim := s.Nearby[0]
	victim.X = 36
	deliverContact(t, a, ctx, out, s, arc.KindContact, unit.Collision{Time: .1, Other: victim})
	if len(drainDamage(out)) != 0 {
		t.Fatal("ranged beats charged a melee coil")
	}
	s.Nearby[0].X = 60
	s.Nearby = append(s.Nearby, unit.Snapshot{ID: 10, Role: unit.RoleFighter, Slot: 1, HP: 100, X: 145, Radius: 18})
	s.Field.Walls = []unit.FieldWall{{Kind: unit.WallCapsule, X1: 100, Y1: -200, X2: 100, Y2: 200, Radius: 3}}
	a.Handle(ctx, api.Pulse{Sense: s})
	a.Handle(ctx, api.Pulse{Sense: s})
	drainDamage(out)
	deliverContact(t, a, ctx, out, s, arc.KindContact, unit.Collision{Time: .5, Other: victim})
	if d := drainDamage(out); len(d) != 1 || d[9] != 6 {
		t.Fatalf("contact chain crossed a solid wall: %v", d)
	}
}

func TestLaserAndCoilAreInterruptedByStunOrDetachment(t *testing.T) {
	for _, id := range []int{11, 14} {
		for _, interrupt := range []unit.Event{unit.Stunned{}, api.Detached{}} {
			a, ctx, out, s := testWeapon(t, id)
			if id == 11 {
				s.Nearby[0].X = 160
			}
			a.Handle(ctx, api.Pulse{Sense: s})
			a.Handle(ctx, api.Pulse{Sense: s})
			drainDamage(out)
			a.Handle(ctx, interrupt)
			drainDamage(out)
			s.Time = .4
			a.Handle(ctx, s)
			if id == 14 {
				deliverContact(t, a, ctx, out, s, arc.KindContact, unit.Collision{Time: .4, Other: s.Nearby[0]})
			} else {
				a.Handle(ctx, unit.Collision{Time: .4, Other: s.Nearby[0]})
			}
			if len(drainDamage(out)) != 0 {
				t.Fatalf("weapon %d kept its interrupted attack", id)
			}
		}
	}
}
func TestDroppedWeaponLosesItsPendingCharge(t *testing.T) {
	state := &unit.MatchState{}
	out := make(chan unit.Cmd, 512)
	ctx := unit.Context{State: state, ID: 1, Kind: KindEngineHeart, Out: out}
	w := worldFor(state, 1)
	w.parts = []part{{Type: 11, Equipped: true}}
	w.stack = []int{0}
	h := &heart{}
	h.syncWeapons(ctx, w)
	h.weaponsHandle(ctx, api.Pulse{})
	w.parts[0].Equipped = false
	h.syncWeapons(ctx, w)
	w.parts[0].Equipped = true
	h.syncWeapons(ctx, w)
	s := unit.Sense{Field: unit.HexField(), Self: unit.Snapshot{ID: 1, Slot: 0}, Nearby: []unit.Snapshot{{ID: 9, Role: unit.RoleFighter, Slot: 1, HP: 100, X: 60, Radius: 18}}}
	h.weaponsHandle(ctx, api.Pulse{Sense: s})
	if len(drainDamage(out)) != 0 {
		t.Fatal("recovered laser retained its removed charge")
	}
}
