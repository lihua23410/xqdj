package 筑墙者_百战

import (
	"math"
	"math/rand/v2"
	"testing"

	"xqdj/internal/unit"
)

func TestShortestMeetsInTheMiddle(t *testing.T) {
	d := shortest(0, 20*math.Pi/180)
	if d <= 0 || math.Abs(d-20*math.Pi/180) > 1e-9 {
		t.Fatalf("delta=%v", d)
	}
}

func TestShortestTieBreaksClockwise(t *testing.T) {
	d := shortest(0, math.Pi/2)
	if math.Abs(d-math.Pi/2) > 1e-9 {
		t.Fatalf("delta=%v", d)
	}
}

func TestBuildsOnBeat(t *testing.T) {
	a := newVeteran()
	out := make(chan unit.Cmd, 4)
	ctx := unit.Context{ID: 7, Kind: KindVeteran, Out: out}
	a.Handle(ctx, unit.Sense{Time: wallEvery - 0.1, Self: self(), Field: unit.HexField()})
	if len(drain(out)) != 0 {
		t.Fatal("built early")
	}
	a.Handle(ctx, unit.Sense{Time: wallEvery, Self: self(), Field: unit.HexField()})
	w := place(drain(out))
	if w == nil {
		t.Fatal("missing wall")
	}
	if math.Abs(math.Hypot(w.X2-w.X1, w.Y2-w.Y1)-wallLen) > 1e-6 {
		t.Fatalf("len=%v", math.Hypot(w.X2-w.X1, w.Y2-w.Y1))
	}
	if w.Radius != wallRadius || w.Amount != wallScrape || w.HitGap != wallGap || w.Life >= 0 || !w.WithOwner {
		t.Fatalf("wall=%+v", w)
	}
	if !unit.HexField().OutlineContains((w.X1+w.X2)/2, (w.Y1+w.Y2)/2, 0) {
		t.Fatal("center outside field")
	}
}

func TestSecondWallSitsBehind(t *testing.T) {
	a := newVeteran()
	a.nextBuild = 0
	out := make(chan unit.Cmd, 4)
	a.Handle(unit.Context{ID: 7, Kind: KindVeteran, Out: out}, unit.Sense{
		Time: 2, Self: self(), Field: unit.HexField(),
		Walls:  []unit.WallView{seg(1, 0, -100)},
		Nearby: []unit.Snapshot{{ID: 2, Slot: 1, X: 0, Y: 0, AimPriority: 15, Radius: 18}},
	})
	w := place(drain(out))
	if w == nil {
		t.Fatal("missing wall")
	}
	mx := (w.X1 + w.X2) / 2
	my := (w.Y1 + w.Y2) / 2
	if math.Abs(mx) > 1e-6 || math.Abs(my-wallLen/2) > 1e-6 {
		t.Fatalf("center=%v,%v", mx, my)
	}
}

func TestBehindOutsideWaitsReady(t *testing.T) {
	a := newVeteran()
	a.nextBuild = 0
	out := make(chan unit.Cmd, 2)
	sense := unit.Sense{
		Time: 2, Self: self(), Field: unit.HexField(),
		Walls:  []unit.WallView{seg(1, 0, 0)},
		Nearby: []unit.Snapshot{{ID: 2, Slot: 1, X: 0, Y: 200, AimPriority: 15}},
	}
	a.Handle(unit.Context{ID: 7, Kind: KindVeteran, Out: out}, sense)
	if len(drain(out)) != 0 || a.nextBuild != 0 {
		t.Fatal("placed or reset clock while behind is outside")
	}
	sense.Nearby[0].Y = 40
	a.Handle(unit.Context{ID: 7, Kind: KindVeteran, Out: out}, sense)
	w := place(drain(out))
	if w == nil {
		t.Fatal("should place once the point is inside")
	}
	my := (w.Y1 + w.Y2) / 2
	if math.Abs(my-(40+wallLen/2)) > 1e-6 {
		t.Fatalf("center y=%v", my)
	}
}

func TestWaitsUntilNormalsCoincide(t *testing.T) {
	a := newVeteran()
	a.phase = phasePair
	out := make(chan unit.Cmd, 4)
	// 两堵都横着，但左右分开。法线是竖的，墙心连线是横的，还不能撞。
	a.Handle(unit.Context{ID: 7, Kind: KindVeteran, Out: out}, unit.Sense{
		Time:  6,
		Self:  self(),
		Walls: []unit.WallView{segAt(1, 0, 0, 0), segAt(2, 0, 120, 0)},
	})
	s1, s2 := motions(drain(out))
	if s1 == nil || s2 == nil {
		t.Fatal("missing spin")
	}
	if s1.Ram != 0 || s2.Ram != 0 {
		t.Fatal("charged before normals coincide")
	}
	if s1.Spin >= 0 || s2.Spin >= 0 {
		t.Fatalf("spin=%v %v", s1.Spin, s2.Spin)
	}
}

func TestAlignedNormalCharges(t *testing.T) {
	a := newVeteran()
	a.phase = phasePair
	out := make(chan unit.Cmd, 4)
	// 上下叠着的两堵横墙，法线已经是同一条竖线。
	a.Handle(unit.Context{ID: 7, Kind: KindVeteran, Out: out}, unit.Sense{
		Time:  8,
		Self:  self(),
		Walls: []unit.WallView{seg(1, 0, 0), seg(2, 0, 100)},
	})
	s1, s2 := motions(drain(out))
	if s1 == nil || s2 == nil {
		t.Fatal("missing charge")
	}
	if math.Abs(s1.VY-chargeV0) > 1e-6 || math.Abs(s2.VY+chargeV0) > 1e-6 || s1.VX != 0 || s2.VX != 0 {
		t.Fatalf("vel=(%v,%v) (%v,%v)", s1.VX, s1.VY, s2.VX, s2.VY)
	}
	if s1.Ram != ramDamage || s1.StunRadius != stunRadius || s1.StunDur != stunDur {
		t.Fatalf("ram=%+v", s1)
	}
}

func TestChargeAcceleratesAlongNormal(t *testing.T) {
	a := newVeteran()
	a.phase = phasePair
	a.locked = true
	a.chargeFrom = 8
	a.axisX, a.axisY = 0, 1
	out := make(chan unit.Cmd, 4)
	a.Handle(unit.Context{ID: 7, Kind: KindVeteran, Out: out}, unit.Sense{
		Time:  9,
		Self:  self(),
		Walls: []unit.WallView{seg(1, 0, 0), seg(2, 0, 100)},
	})
	s1, _ := motions(drain(out))
	if s1 == nil || math.Abs(s1.VY-(chargeV0+chargeAcc)) > 1e-6 {
		t.Fatalf("accel vel=%v", s1)
	}
}

func TestSlamRestartsClockAfterStun(t *testing.T) {
	a := newVeteran()
	a.phase = phasePair
	a.locked = true
	a.Handle(unit.Context{ID: 7, Kind: KindVeteran}, unit.WallSlam{Time: 10, SelfStunned: true})
	if a.phase != phaseIdle || a.locked {
		t.Fatal("phase")
	}
	if math.Abs(a.nextBuild-(10+wallEvery)) > 1e-9 {
		t.Fatalf("next=%v", a.nextBuild)
	}
}

func TestBreakResumesSavedInterval(t *testing.T) {
	a := newVeteran()
	a.phase = phasePair
	a.savedLeft = wallEvery
	out := make(chan unit.Cmd, 4)
	a.Handle(unit.Context{ID: 7, Kind: KindVeteran, Out: out}, unit.Sense{
		Time:  9,
		Self:  self(),
		Walls: []unit.WallView{seg(1, 0, 0)},
	})
	got := drain(out)
	if len(got) != 1 {
		t.Fatalf("cmds=%d", len(got))
	}
	if _, ok := got[0].(unit.SetWallMotion); !ok {
		t.Fatalf("cmd=%T", got[0])
	}
	if math.Abs(a.nextBuild-(9+wallEvery)) > 1e-9 {
		t.Fatalf("next=%v", a.nextBuild)
	}
}

func newVeteran() *百战 {
	return &百战{
		nextBuild: wallEvery,
		savedLeft: wallEvery,
		rng:       rand.New(rand.NewPCG(1, 2)),
	}
}

func self() unit.Snapshot {
	return unit.Snapshot{ID: 7, Slot: 0, Radius: vetRadius}
}

func seg(id uint64, ang, y float64) unit.WallView {
	return segAt(id, ang, 0, y)
}

func segAt(id uint64, ang, x, y float64) unit.WallView {
	dx, dy := math.Cos(ang)*wallLen/2, math.Sin(ang)*wallLen/2
	return unit.WallView{
		ID: id, OwnerID: 7, Slot: 0,
		X1: x - dx, Y1: y - dy, X2: x + dx, Y2: y + dy,
		Radius: wallRadius,
	}
}

func place(cmds []unit.Cmd) *unit.PlaceWall {
	for _, c := range cmds {
		if w, ok := c.(unit.PlaceWall); ok {
			return &w
		}
	}
	return nil
}

func motions(cmds []unit.Cmd) (a, b *unit.SetWallMotion) {
	var got []unit.SetWallMotion
	for _, c := range cmds {
		if m, ok := c.(unit.SetWallMotion); ok {
			got = append(got, m)
		}
	}
	if len(got) < 2 {
		return nil, nil
	}
	return &got[0], &got[1]
}

func drain(ch <-chan unit.Cmd) []unit.Cmd {
	var out []unit.Cmd
	for {
		select {
		case c := <-ch:
			out = append(out, c)
		default:
			return out
		}
	}
}
