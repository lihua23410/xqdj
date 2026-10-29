package 大六边形

import (
	"math"
	"testing"

	"xqdj/internal/unit"
)

func TestWideHexField(t *testing.T) {
	f, ok := unit.LookupField(Name)
	if !ok {
		t.Fatal("missing 大六边形")
	}
	if f.Shape != unit.ShapeHex || f.Extent != Extent || len(f.Walls) != 0 || f.BootKind != KindWidehex {
		t.Fatalf("field=%+v", f)
	}
	if Extent != 27*unit.HexRadius/16 {
		t.Fatalf("extent=%v", Extent)
	}
}

func TestWideHexWalkable(t *testing.T) {
	if !Field().Walkable(0, 0, 18) {
		t.Fatal("center")
	}
	if Field().Walkable(Extent, 0, 18) {
		t.Fatal("vertex")
	}
	if !Field().Walkable(unit.HexRadius, 0, 18) {
		t.Fatal("default vertex should be inside")
	}
	apo := Extent * math.Sqrt(3) / 2
	if Field().Walkable(0, apo, 18) {
		t.Fatal("flat")
	}
	if !Field().Walkable(0, apo/2, 18) {
		t.Fatal("mid apo")
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

func TestBootSpawnsOppositeVortices(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := &大六边形{}
	ctx := unit.Context{ID: 1, Kind: KindWidehex, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0, Self: unit.Snapshot{ID: 1, Kind: KindWidehex, Slot: 0}})
	cmds := drain(out)
	var sp []unit.Spawn
	var bar *unit.PlaceWall
	for _, c := range cmds {
		switch v := c.(type) {
		case unit.Spawn:
			if v.Kind == KindVortex {
				sp = append(sp, v)
			}
		case unit.PlaceWall:
			cp := v
			bar = &cp
		}
	}
	if len(sp) != 2 {
		t.Fatalf("spawn=%d cmds=%v", len(sp), cmds)
	}
	if math.Abs(math.Hypot(sp[0].X, sp[0].Y)-orbitR) > 1e-6 {
		t.Fatalf("r=%v want %v", math.Hypot(sp[0].X, sp[0].Y), orbitR)
	}
	if math.Abs(sp[0].X+sp[1].X) > 1e-6 || math.Abs(sp[0].Y+sp[1].Y) > 1e-6 {
		t.Fatalf("not opposite %+v %+v", sp[0], sp[1])
	}
	if bar == nil || !bar.Hard || !bar.VisionBlock || bar.Radius != barR {
		t.Fatalf("bar=%+v", bar)
	}
	if bar.X1 != -barHalf || bar.X2 != barHalf || bar.Y1 != 0 || bar.Y2 != 0 {
		t.Fatalf("bar ends=%+v", bar)
	}
}

func TestBarSpinIsClockwise(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := &大六边形{booted: true, inside: map[uint64]uint64{}, cool: map[uint64]float64{}}
	ctx := unit.Context{ID: 1, Kind: KindWidehex, Out: out}
	a.Handle(ctx, unit.Sense{
		Time: 1, Self: unit.Snapshot{ID: 1, Kind: KindWidehex},
		Walls: []unit.WallView{{ID: 9, OwnerID: 1, VisionBlock: true, X1: -barHalf, X2: barHalf, Radius: barR}},
	})
	var mot *unit.SetWallMotion
	for _, c := range drain(out) {
		if v, ok := c.(unit.SetWallMotion); ok {
			cp := v
			mot = &cp
		}
	}
	want := -2 * math.Pi / barPeriod
	if mot == nil || mot.WallID != 9 || math.Abs(mot.Spin-want) > 1e-9 {
		t.Fatalf("motion=%+v want spin %v", mot, want)
	}
}

func TestSpinIsCounterclockwise(t *testing.T) {
	x1, y1, x2, y2 := poseAt(spinPeriod / 4)
	if math.Abs(x1) > 1e-6 || math.Abs(y1-orbitR) > 1e-6 {
		t.Fatalf("quarter=(%v,%v) want (0,%v)", x1, y1, orbitR)
	}
	if math.Abs(x2) > 1e-6 || math.Abs(y2+orbitR) > 1e-6 {
		t.Fatalf("opposite=%v,%v", x2, y2)
	}
}

func TestWarpSendsInwardAndFlips(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &大六边形{booted: true, a: 2, b: 3, inside: map[uint64]uint64{}, cool: map[uint64]float64{}}
	ctx := unit.Context{ID: 1, Kind: KindWidehex, Out: out}
	va := unit.Snapshot{ID: 2, Kind: KindVortex, OwnerID: 1, X: orbitR, Y: 0, Radius: vortexR}
	vb := unit.Snapshot{ID: 3, Kind: KindVortex, OwnerID: 1, X: -orbitR, Y: 0, Radius: vortexR}
	foe := unit.Snapshot{
		ID: 10, Role: unit.RoleFighter, X: orbitR, Y: 0, Radius: 18,
		VX: 120, VY: 0, FaceX: 1, FaceY: 0,
	}
	a.Handle(ctx, unit.Sense{
		Time: 1, Self: unit.Snapshot{ID: 1, Kind: KindWidehex},
		Nearby: []unit.Snapshot{va, vb, foe},
	})
	wantX := -orbitR + (vortexR + 18 + warpPad)
	var tp *unit.Teleport
	var vel *unit.SetVelocity
	var dir *unit.SetFSDirection
	for _, c := range drain(out) {
		switch v := c.(type) {
		case unit.Teleport:
			if v.UnitID == foe.ID {
				cp := v
				tp = &cp
			}
		case unit.SetVelocity:
			if v.UnitID == foe.ID {
				cp := v
				vel = &cp
			}
		case unit.SetFSDirection:
			if v.UnitID == foe.ID {
				cp := v
				dir = &cp
			}
		}
	}
	if tp == nil {
		t.Fatal("missing Teleport")
	}
	if math.Abs(tp.X-wantX) > 1e-6 || math.Abs(tp.Y) > 1e-6 {
		t.Fatalf("dest=(%v,%v) want (%v,0)", tp.X, tp.Y, wantX)
	}
	if vel == nil || math.Abs(vel.VX+120) > 1e-6 || math.Abs(vel.VY) > 1e-6 {
		t.Fatalf("vel=%+v", vel)
	}
	if dir == nil || dir.VX >= 0 {
		t.Fatalf("dir=%+v", dir)
	}
}

func TestWarpSkipsHelper(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &大六边形{booted: true, a: 2, b: 3, inside: map[uint64]uint64{}, cool: map[uint64]float64{}}
	ctx := unit.Context{ID: 1, Kind: KindWidehex, Out: out}
	va := unit.Snapshot{ID: 2, Kind: KindVortex, OwnerID: 1, X: orbitR, Y: 0, Radius: vortexR}
	vb := unit.Snapshot{ID: 3, Kind: KindVortex, OwnerID: 1, X: -orbitR, Y: 0, Radius: vortexR}
	moon := unit.Snapshot{ID: 11, Kind: "狼人月亮", Role: unit.RoleHelper, X: orbitR, Y: 0, Radius: 34}
	a.Handle(ctx, unit.Sense{
		Time: 1, Self: unit.Snapshot{ID: 1, Kind: KindWidehex},
		Nearby: []unit.Snapshot{va, vb, moon},
	})
	for _, c := range drain(out) {
		if tp, ok := c.(unit.Teleport); ok && tp.UnitID == moon.ID {
			t.Fatalf("helper should not warp: %+v", tp)
		}
	}
}

func TestWarpSkipsVortexAndStayingInside(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &大六边形{booted: true, a: 2, b: 3, inside: map[uint64]uint64{10: 2}, cool: map[uint64]float64{}}
	ctx := unit.Context{ID: 1, Kind: KindWidehex, Out: out}
	va := unit.Snapshot{ID: 2, Kind: KindVortex, OwnerID: 1, X: orbitR, Y: 0, Radius: vortexR}
	vb := unit.Snapshot{ID: 3, Kind: KindVortex, OwnerID: 1, X: -orbitR, Y: 0, Radius: vortexR}
	foe := unit.Snapshot{ID: 10, Role: unit.RoleFighter, X: orbitR, Y: 0, Radius: 18, VX: 1}
	a.Handle(ctx, unit.Sense{
		Time: 1, Self: unit.Snapshot{ID: 1, Kind: KindWidehex},
		Nearby: []unit.Snapshot{va, vb, foe},
	})
	for _, c := range drain(out) {
		if tp, ok := c.(unit.Teleport); ok && tp.UnitID == foe.ID {
			t.Fatalf("staying overlap should not warp: %+v", tp)
		}
	}
}
