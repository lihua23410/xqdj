package sim

import (
	"encoding/json"
	"math"
	"testing"
	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestSurgeryCutsStitchesAndContinuouslyBleedsInMatch(t *testing.T) {
	m := NewMatchSeeded(73)
	m.SetSlot(0, character.KindSurgery)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	t.Cleanup(m.End)
	m.mu.Lock()
	for _, u := range m.units {
		if u.role == unitpkg.RoleFighter {
			u.hp, u.maxHP = 10000, 10000
		}
	}
	m.mu.Unlock()
	seenCut, seenStitch, seenBurst := false, false, false
	bloodFrames := 0
	maxWounds := 0
	for tick := 0; tick < 60*90; tick++ {
		m.Tick()
		var snap struct {
			Units   []unitpkg.Snapshot `json:"units"`
			Effects []unitpkg.FX       `json:"effects"`
		}
		if err := json.Unmarshal(m.SnapshotJSON(), &snap); err != nil {
			t.Fatal(err)
		}
		for _, u := range snap.Units {
			if u.Kind == character.KindScalpel {
				seenCut = true
			}
			if u.Kind == character.KindSutureNeedle {
				seenStitch = true
			}
			if u.Kind == character.KindBloodBullet {
				bloodFrames++
				break
			}
		}
		for _, fx := range snap.Effects {
			if fx.Name == "surgery-burst" {
				seenBurst = true
			}
			if fx.Name == "surgery-state" {
				var view struct {
					Charge   float64 `json:"charge"`
					Capacity float64 `json:"capacity"`
					Wounds   []struct {
						Bleeding bool `json:"bleeding"`
					} `json:"wounds"`
				}
				if err := json.Unmarshal(fx.Data, &view); err != nil {
					t.Fatal(err)
				}
				maxWounds = max(maxWounds, len(view.Wounds))
				if len(view.Wounds) > 6 || view.Charge < 0 || view.Charge > view.Capacity || view.Capacity != 200 {
					t.Fatal("wound/charge bounds violated")
				}
			}
		}
		if seenCut && seenStitch && seenBurst && bloodFrames > 90 && maxWounds >= 3 {
			break
		}
	}
	if !seenCut || !seenStitch || !seenBurst || bloodFrames < 90 || maxWounds < 3 {
		t.Fatalf("cut=%v stitch=%v burst=%v blood frames=%d wounds=%d", seenCut, seenStitch, seenBurst, bloodFrames, maxWounds)
	}
	// Owner death recursively removes independent weapons, floor and blood.
	m.mu.Lock()
	owner := fighterSlot(m, 0).id
	m.removeLocked(m.units[owner])
	for _, u := range m.units {
		if u.owner == owner {
			m.mu.Unlock()
			t.Fatalf("owned %s survived surgeon", u.kind)
		}
	}
	m.mu.Unlock()
}

func TestBodiesOnlyBloodHitsOwnerAndFriendsButPassesOtherProjectiles(t *testing.T) {
	m := newDummyLog(t)
	t.Cleanup(m.End)
	m.mu.Lock()
	defer m.mu.Unlock()
	a, b := fighterSlot(m, 0), fighterSlot(m, 1)
	a.p, b.p = vec{0, 0}, vec{0, 150}
	a.setVel(vec{})
	b.setVel(vec{})
	blood := m.addUnitLocked(character.KindBloodBullet, vec{-60, 0}, vec{240, 0}, a.id, a.slot)
	other := m.addUnitLocked("子弹", vec{-35, 0}, vec{}, b.id, b.slot)
	if blood == nil || other == nil {
		t.Fatal("missing projectiles")
	}
	if _, hit := m.earliestHitLocked(.05, nil, nil); hit {
		t.Fatal("blood collided with another projectile")
	}
	other.p = vec{0, -150}
	hit, ok := m.earliestHitLocked(.3, nil, nil)
	if !ok || hit.kind != hitPair || !(hit.a == a.id && hit.b == blood.id || hit.b == a.id && hit.a == blood.id) {
		t.Fatal("blood cannot collide with its own creator")
	}
	// Also check an unrelated fighter sharing the source's side.
	a.p = vec{0, 200}
	b.slot = a.slot
	b.p = vec{0, 0}
	if _, ok = m.earliestHitLocked(.3, nil, nil); !ok {
		t.Fatal("blood cannot collide with same-side fighter")
	}
}

func TestPassInnerWallsStillRespectsFieldEdgeAndMovingWalls(t *testing.T) {
	m := newDummyLog(t)
	t.Cleanup(m.End)
	m.mu.Lock()
	defer m.mu.Unlock()
	a, b := fighterSlot(m, 0), fighterSlot(m, 1)
	b.p = vec{0, -180}
	a.p = vec{-40, 0}
	a.setVel(vec{240, 0})
	m.applyCmdLocked(unitpkg.PlaceWall{OwnerID: b.id, Slot: b.slot, Kind: "test", X1: 0, Y1: -90, X2: 0, Y2: 90, Radius: 6, Life: -1, Hard: true, Square: true})
	m.applyCmdLocked(unitpkg.PassInnerWalls{UnitID: a.id, Hold: true})
	for i := 0; i < 30; i++ {
		m.physicsLocked(DT)
	}
	if a.p.X < 50 || a.v.X <= 0 {
		t.Fatalf("internal wall blocked sewing: %+v", a.p)
	}
	for i := 0; i < 80; i++ {
		m.physicsLocked(DT)
	}
	if !m.spec.outline.Contains(a.p.X, a.p.Y, a.radius) {
		t.Fatalf("escaped outer field: %+v", a.p)
	}
	a.p = vec{0, 0}
	for _, w := range m.walls {
		m.shoveWallLocked(w)
	}
	if math.Abs(a.p.X) > 1e-8 {
		t.Fatal("moving wall shoved an inner-wall passer")
	}
	m.applyCmdLocked(unitpkg.PassInnerWalls{UnitID: a.id})
	if math.Abs(a.p.X) < a.radius+6 {
		t.Fatal("dropping inner-wall pass left the body embedded")
	}
}

func TestObserverReplaysSurgeryScalpelWithoutPrivatePlan(t *testing.T) {
	m := NewMatchSeeded(73)
	m.SetSlot(0, character.KindObserver)
	m.SetSlot(1, character.KindSurgery)
	m.Start()
	t.Cleanup(m.End)
	m.mu.Lock()
	observer, surgeon := fighterSlot(m, 0), fighterSlot(m, 1)
	observerID := observer.id
	observer.p, surgeon.p = vec{80, 0}, vec{-80, 0}
	for _, u := range []*unit{observer, surgeon} {
		u.hp, u.maxHP = 10000, 10000
		m.applyCmdLocked(unitpkg.SetCruise{UnitID: u.id, Speed: 0})
		u.setVel(vec{})
	}
	m.mu.Unlock()
	for tick := 0; tick < 60*20; tick++ {
		m.Tick()
		m.mu.Lock()
		copied := false
		for _, u := range m.units {
			if u.kind == character.KindScalpel && u.owner == observerID {
				copied = true
				if math.Abs(u.damageScale-.85) > 1e-9 {
					m.mu.Unlock()
					t.Fatalf("copy scale=%v want .85", u.damageScale)
				}
			}
		}
		m.mu.Unlock()
		if copied {
			return
		}
	}
	t.Fatal("observer never replayed the scalpel")
}
