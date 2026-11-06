package 泵之心

import (
	"math"
	"testing"
	"xqdj/internal/unit"
	_ "xqdj/map/hex"
)

const (
	partPiston = iota
	partManifold
	partFlywheel
	partCam
	partNozzle
	partPlate
	partDouble
	partTurbine
	partCage
	partSteer
	partWinch
	partCount
)

var partNames = [...]string{"增压活塞", "分流歧管", "惯性飞轮", "配气凸轮", "燃油喷嘴", "护心夹板", "双联阀组", "偏流叶轮", "捕弹阀笼", "换向喷口", "回收绞盘"}

func TestSevenPartsAreUniqueFixedAndMatchLocal(t *testing.T) {
	for seed := uint64(1); seed < 40; seed++ {
		w := &world{}
		s := unit.Sense{Field: unit.HexField(), Self: unit.Snapshot{ID: 1, X: -100, Radius: 18}}
		w.initialize(seed, s)
		if len(w.parts) != 7 {
			t.Fatalf("seed %d: got %d parts", seed, len(w.parts))
		}
		seen := map[int]bool{}
		for _, p := range w.parts {
			if seen[p.Type] || !s.Field.Walkable(p.X, p.Y, 7) {
				t.Fatalf("invalid initial part %+v", p)
			}
			seen[p.Type] = true
		}
		copyWorld := &world{}
		copyWorld.initialize(seed, s)
		for i, p := range w.parts {
			if p != copyWorld.parts[i] {
				t.Fatal("selection depends on unrelated actor order")
			}
		}
		w.initialize(seed+100, s)
		for _, p := range w.parts {
			if !seen[p.Type] {
				t.Fatal("selection rerolled inside a match")
			}
		}
	}
	a, b := &unit.MatchState{}, &unit.MatchState{}
	if worldFor(a, 1) == worldFor(a, 2) || worldFor(a, 1) == worldFor(b, 1) {
		t.Fatal("assembly leaked across owners or matches")
	}
}

func TestPickupOrderDropAndRecoveryPreserveTheFiveObjects(t *testing.T) {
	state := &unit.MatchState{}
	out := make(chan unit.Cmd, 128)
	ctx := unit.Context{State: state, ID: 2, Out: out}
	w := worldFor(state, 1)
	w.parts = []part{{Type: partCage, X: 8}, {Type: partPiston, X: -2}, {Type: partCam, X: 120}, {Type: partWinch, X: 160}, {Type: partNozzle, X: 200}}
	f := &floor{owner: 1}
	owner := unit.Snapshot{ID: 1, Role: unit.RoleFighter, Radius: 18, X: -30, HP: 100}
	s := unit.Sense{Field: unit.HexField(), Nearby: []unit.Snapshot{owner}}
	f.Handle(ctx, s)
	owner.X = 10
	s.Nearby[0] = owner
	s.Time = 1.0 / 60
	f.Handle(ctx, s)
	if len(w.stack) != 2 || w.parts[w.stack[0]].Type != partPiston || w.parts[w.stack[1]].Type != partCage {
		t.Fatalf("wrong pickup stack %v", w.stack)
	}
	w.hurt(13, s.Time)
	if w.has(partCage) || !w.has(partPiston) || !w.parts[0].Flying || math.Abs(w.wear-1) > 1e-9 {
		t.Fatal("damage did not remove the newest part with remainder")
	}
	// A flying component cannot be reattached even if the body overlaps it.
	w.parts[0].X, w.parts[0].Y = owner.X, owner.Y
	s.Time += .001
	f.Handle(ctx, s)
	if w.has(partCage) {
		t.Fatal("caught a flying component")
	}
	w.parts[0].Flying = false
	w.parts[0].VX, w.parts[0].VY = 0, 0
	w.parts[0].X, w.parts[0].Y = owner.X, owner.Y
	f.Handle(ctx, s)
	if !w.has(partCage) || w.stack[len(w.stack)-1] != 0 || len(w.parts) != 5 {
		t.Fatal("recovery must reuse the same object and append it at the top")
	}
	w.hurt(50, s.Time)
	if len(w.stack) != 0 || w.wear != 0 {
		t.Fatal("large hit did not empty the stack cleanly")
	}
	w.hurt(100, s.Time)
	if w.wear != 0 {
		t.Fatal("naked heart accumulated a debt against future equipment")
	}
}

func TestFlyingPartsStopExactlyAndCannotTunnelThroughThinWalls(t *testing.T) {
	f := unit.HexField()
	f.Walls = []unit.FieldWall{{Kind: unit.WallCapsule, X1: 0, Y1: -200, X2: 0, Y2: 200, Radius: 3}}
	p := part{X: -30, VX: 520, Flying: true}
	for i := 0; i < 48; i++ {
		advance(&p, f, 1.0/60)
		if p.X > -10+1e-6 || !f.Walkable(p.X, p.Y, 7) {
			t.Fatalf("part crossed the thin wall: %+v", p)
		}
	}
	if p.Flying || p.VX != 0 || p.VY != 0 {
		t.Fatalf("part never became collectible: %+v", p)
	}
}

func TestGameplayPartsWorkIndependently(t *testing.T) {
	for _, kind := range []int{partDouble, partTurbine, partCage, partSteer, partWinch, partPlate} {
		t.Run(partNames[kind], func(t *testing.T) {
			state := &unit.MatchState{}
			out := make(chan unit.Cmd, 128)
			w := worldFor(state, 1)
			w.initialized = true
			w.parts = []part{{Type: kind, Equipped: true}}
			w.stack = []int{0}
			h := &heart{boot: true, period: 1}
			s := unit.Sense{Time: .41, Field: unit.HexField(), Self: unit.Snapshot{ID: 1, Role: unit.RoleFighter, Slot: 0, Radius: 18, VX: 150, HP: 100}, Nearby: []unit.Snapshot{{ID: 9, Role: unit.RoleFighter, Slot: 1, X: 50, HP: 100}}}
			h.Handle(unit.Context{State: state, ID: 1, Kind: KindEngineHeart, Out: out}, s)
			if len(out) == 0 {
				t.Fatal("a single component disabled the core cycle")
			}
		})
	}
}
