package 纵火者

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"

	"xqdj/internal/unit"
)

func stateAt(g *oilSurface, x, y float64) uint8 {
	if c := g.cell(point{gridCoord(x), gridCoord(y)}); c != nil {
		return c.state
	}
	return dry
}
func countState(g *oilSurface, state uint8) int {
	n := 0
	for _, chunk := range g.chunks {
		for _, c := range chunk.cells {
			if c.state == state {
				n++
			}
		}
	}
	return n
}
func TestSurfaceContinuousCoverageAndCache(t *testing.T) {
	g := &oilSurface{}
	g.addOil(0, 0, 0, 0, oilRadius, 0)
	if n := countState(g, oil); math.Abs(float64(n)*surfaceStep*surfaceStep-math.Pi*oilRadius*oilRadius) > 200 {
		t.Fatalf("oil coverage %d", n)
	}
	g.addOil(0, 0, 90, 0, 18, 0)
	for x := 0.0; x <= 105; x += 3 {
		if stateAt(g, x, 0) != oil {
			t.Fatalf("hole in continuous trail at %v", x)
		}
	}
	if stateAt(g, 111, 0) != dry {
		t.Fatal("trail extends beyond its capsule")
	}
	data := g.payload()
	original := append([]byte(nil), data...)
	rev := g.revision
	g.addOil(0, 0, 90, 0, 18, 0)
	if g.revision != rev || &g.payload()[0] != &data[0] {
		t.Fatal("overlap must reuse coverage and cached packet")
	}
	g.addFire(0, 0, fireRadius, 0)
	if !bytes.Equal(data, original) {
		t.Fatal("mutated an already emitted frame")
	}
	var view surfaceView
	if err := json.Unmarshal(g.payload(), &view); err != nil {
		t.Fatal(err)
	}
	decoded := map[point]uint8{}
	for _, ch := range view.Chunks {
		for state, runs := range map[uint8][]int{oil: ch.Oil, flame: ch.Fire} {
			for i := 0; i < len(runs); i += 2 {
				start, n := runs[i], runs[i+1]
				if n <= 0 || start/chunkSide != (start+n-1)/chunkSide {
					t.Fatal("run crossed row")
				}
				for j := start; j < start+n; j++ {
					decoded[point{ch.X*chunkSide + j%chunkSide, ch.Y*chunkSide + j/chunkSide}] = state
				}
			}
		}
	}
	for k, ch := range g.chunks {
		for i, c := range ch.cells {
			p := point{k.x*chunkSide + i%chunkSide, k.y*chunkSide + i/chunkSide}
			if decoded[p] != c.state {
				t.Fatalf("visual and collision masks disagree at %v", p)
			}
		}
	}
}
func TestSeparatePoursConnectAndFrontUsesTime(t *testing.T) {
	g := &oilSurface{}
	g.addOil(0, 0, 90, 0, 12, 0)
	g.addOil(90, 0, 240, 0, 12, .1) // A different pour touching the first one.
	g.addFire(0, 0, 6, .2)
	g.advance(.2)
	if stateAt(g, 180, 0) != oil {
		t.Fatal("connected pool ignited instantly")
	}
	g.advance(1.39)
	if stateAt(g, 90, 0) != flame || stateAt(g, 120, 0) != oil {
		t.Fatal("front did not travel at 80 world units/s")
	}
	g.advance(2.55)
	if stateAt(g, 180, 0) != flame {
		t.Fatal("different pours blocked propagation")
	}
	other := &oilSurface{}
	other.addOil(90, 0, 240, 0, 12, 0)
	other.advance(4)
	if countState(other, flame) != 0 {
		t.Fatal("another owner's oil was lit")
	}
	g.advance(8)
	if len(g.chunks) != 0 {
		t.Fatal("burned surface must disappear, not return to oil")
	}
}
func TestSpreadCannotJumpDryGap(t *testing.T) {
	g := &oilSurface{}
	g.set(point{0, 0}, oil)
	g.set(point{2, 1}, oil)
	g.burn(point{0, 0}, 0)
	g.spread(point{0, 0}, 0)
	g.advance(.5)
	if g.cell(point{2, 1}).state != oil {
		t.Fatal("long direction jumped dry cells")
	}
	g.set(point{1, 0}, oil)
	g.set(point{1, 1}, oil)
	g.spread(point{0, 0}, .5)
	g.advance(.6)
	if g.cell(point{2, 1}).state != flame {
		t.Fatal("connected bridge did not carry fire")
	}
}
func TestNewBridgeCarriesExistingFire(t *testing.T) {
	g := &oilSurface{}
	g.addFire(0, 0, 6, 0)
	g.addOil(90, 0, 180, 0, 9, 0)
	g.advance(.2)
	g.addOil(0, 0, 95, 0, 9, .2)
	g.advance(.2)
	if stateAt(g, 90, 0) != oil {
		t.Fatal("new bridge ignited all at once")
	}
	g.advance(1.4)
	if stateAt(g, 90, 0) != flame {
		t.Fatal("new bridge did not join old oil to the live fire")
	}
}
func TestFireExpiryRefuelAndLargeTimeStep(t *testing.T) {
	g := &oilSurface{}
	g.addFire(0, 0, 9, 0)
	g.advance(.7)
	g.addOil(0, 0, 0, 0, 9, .7)
	g.advance(.7)
	g.advance(1.01)
	if stateAt(g, 0, 0) != flame {
		t.Fatal("stale expiry killed refuelled fire")
	}
	g.advance(2.71)
	if stateAt(g, 0, 0) != dry {
		t.Fatal("refuelled fire did not expire")
	}
	a, b := &oilSurface{}, &oilSurface{}
	for _, s := range []*oilSurface{a, b} {
		s.addOil(0, 0, 420, 0, 12, 0)
		s.addFire(0, 0, 9, 0)
	}
	for tm := .01; tm <= 8.01; tm += .01 {
		a.advance(tm)
	}
	b.advance(8.01)
	if countState(a, oil) != 0 || countState(b, oil) != 0 || len(a.chunks) != 0 || len(b.chunks) != 0 {
		t.Fatal("large timestep lost propagation or expiry events")
	}
}
func TestMultipleIgnitionsDoNotAccelerateFront(t *testing.T) {
	a, b := &oilSurface{}, &oilSurface{}
	for _, s := range []*oilSurface{a, b} {
		s.addOil(0, 0, 300, 0, 12, 0)
		s.addFire(0, 0, 6, 0)
	}
	b.addFire(0, 0, 6, 0)
	a.advance(.4)
	b.advance(.4)
	if countState(a, flame) != countState(b, flame) {
		t.Fatal("overlapping ignition doubled propagation speed")
	}
}

func TestTrailAndCooldown(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	f := &油火{owner: 1}
	f.sense(out, 0, []unit.Snapshot{ownerAt(), {ID: 9, Kind: KindArsonOilMark, OwnerID: 1}, foeAt(2, 0, 0)})
	f.sense(out, .1, []unit.Snapshot{ownerAt(), foeAt(2, 100, 0)})
	if f.walk[2].phase != phaseCool || f.walk[2].left != 0 {
		t.Fatal("walking 100 should spend exactly 90 and start cooldown")
	}
	if stateAt(&f.surface, 105, 0) != oil || stateAt(&f.surface, 111, 0) != dry {
		t.Fatal("incorrect trail end")
	}
	rev := f.surface.revision
	f.sense(out, .15, []unit.Snapshot{ownerAt(), foeAt(2, 140, 0)})
	f.sense(out, .19, []unit.Snapshot{ownerAt(), foeAt(2, 0, 0)})
	if f.surface.revision != rev {
		t.Fatal("painted during cooldown")
	}
	f.sense(out, .7, []unit.Snapshot{ownerAt(), foeAt(2, 0, 0)})
	f.sense(out, 1.0, []unit.Snapshot{ownerAt(), foeAt(2, 0, 120)})
	if stateAt(&f.surface, 0, 100) != oil {
		t.Fatal("did not restart trail after cooldown")
	}
}
func TestBlinkDoesNotPaintGapOrSpendTrail(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	f := &油火{owner: 1}
	f.sense(out, 0, []unit.Snapshot{ownerAt(), {ID: 9, Kind: KindArsonOilMark, OwnerID: 1}, foeAt(2, 0, 0)})
	f.sense(out, .1, []unit.Snapshot{ownerAt(), foeAt(2, 300, 0)})
	if stateAt(&f.surface, 150, 0) != dry || stateAt(&f.surface, 300, 0) != oil || f.walk[2].left != trailLen {
		t.Fatal("blink painted empty gap or spent its trail")
	}
	f.sense(out, .2, []unit.Snapshot{ownerAt(), foeAt(2, 340, 0)})
	if f.walk[2].left != 50 || stateAt(&f.surface, 330, 0) != oil {
		t.Fatal("landing movement did not spend remaining trail")
	}
	f.surface.addFire(0, 0, fireRadius, .2)
	f.surface.advance(1)
	if stateAt(&f.surface, 300, 0) != oil {
		t.Fatal("fire jumped teleport gap")
	}
}
func TestBurnedOilDoesNotTriggerTrail(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	f := &油火{owner: 1}
	f.surface.addOil(0, 0, 0, 0, 30, 0)
	f.sense(out, 0, []unit.Snapshot{ownerAt(), foeAt(2, 0, 0)})
	f.surface.addFire(0, 0, 36, .1)
	f.sense(out, .1, []unit.Snapshot{ownerAt(), foeAt(2, 1, 0)})
	if f.walk[2].phase != phaseIdle || countState(&f.surface, oil) != 0 {
		t.Fatal("burned oil incorrectly created a fresh trail")
	}
}
func TestBurnTickDoesNotResetAndAlliesDoNotBurn(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	f := &油火{owner: 1}
	f.surface.addFire(0, 0, fireRadius, 0)
	ally := foeAt(3, 0, 0)
	ally.Slot = 0
	near := []unit.Snapshot{ownerAt(), foeAt(2, 0, 0), ally}
	if len(damages(f.sense(out, 0, near))) != 0 {
		t.Fatal("immediate damage")
	}
	if len(damages(f.sense(out, .1, near))) != 0 || f.burn[2].next != burnTick {
		t.Fatal("refresh reset burn cadence")
	}
	got := damages(f.sense(out, burnTick, near))
	if len(got) != 1 || got[0].To != 2 || got[0].From != 7 || got[0].Amount != burnTickDmg || !got[0].NoFreeze {
		t.Fatalf("incorrect burn %+v", got)
	}
	far := []unit.Snapshot{ownerAt(), foeAt(2, 400, 0), ally}
	f.sense(out, burnTick+0.05, far)
	got = damages(f.sense(out, burnTick+burnLife, far))
	if len(got) != 5 {
		t.Fatalf("burn after leaving must finish its second, got %d ticks", len(got))
	}
}
func TestOwnerIsolationDeathAndSingleFullPacket(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	f := &油火{owner: 1}
	cmds := f.sense(out, 0, []unit.Snapshot{ownerAt(), {ID: 9, Kind: KindArsonOilMark, OwnerID: 1}, {ID: 10, Kind: KindArsonFireMark, OwnerID: 2}})
	if countState(&f.surface, flame) != 0 {
		t.Fatal("foreign marker ignited own oil")
	}
	packets := 0
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok {
			if fx.Name == "arson-burning" {
				continue
			}
			packets++
			if fx.Name != "arson-surface" || len(fx.Data) == 0 {
				t.Fatal("missing full surface")
			}
		}
	}
	if packets != 1 {
		t.Fatalf("want one surface packet, got %d", packets)
	}
	f.surface.addFire(0, 0, 36, 0)
	f.sense(out, .1, []unit.Snapshot{ownerAt(), foeAt(2, 0, 0)})
	cmds = f.sense(out, .3, []unit.Snapshot{foeAt(2, 0, 0)})
	if len(f.surface.chunks) != 0 || len(f.burn) != 0 || len(damages(cmds)) != 0 {
		t.Fatal("owner death left oil/fire/burn alive")
	}
	var empty surfaceView
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == "arson-surface" {
			if err := json.Unmarshal(fx.Data, &empty); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(empty.Chunks) != 0 {
		t.Fatal("owner death must clear the rendered surface")
	}
}

// 拖尾等粗地切在油瓶大圆盘边上会留下细腰和两个凹角，根部必须比末端粗、
// 并且一路不回头地收到身体半径，接缝才连续。
func TestTrailRootTapersIntoPoolInsteadOfPinching(t *testing.T) {
	out := make(chan unit.Cmd, 512)
	f := &油火{owner: 1}
	f.sense(out, 0, []unit.Snapshot{ownerAt(), {ID: 9, Kind: KindArsonOilMark, OwnerID: 1}, foeAt(2, 0, 0)})
	x := 0.0
	start := math.NaN()
	for i := 1; i <= 120; i++ {
		priorX, priorPhase := f.walk[2].x, f.walk[2].phase
		x += 165.0 / 60
		f.sense(out, float64(i)/60, []unit.Snapshot{ownerAt(), foeAt(2, x, 0)})
		if priorPhase == phaseIdle && f.walk[2].phase == phasePaint && math.IsNaN(start) {
			start = priorX
		}
	}
	if math.IsNaN(start) {
		t.Fatal("never left the oil")
	}
	width := func(wx float64) int {
		n := 0
		for y := -40; y <= 40; y++ {
			if c := f.surface.cell(point{gridCoord(wx), y}); c != nil && c.state == oil {
				n++
			}
		}
		return n * int(surfaceStep)
	}
	root := width(start + 3)
	if root < 3*int(bodyRadius) {
		t.Fatalf("拖尾根部只有 %d 宽，接回油滩处会塌成细腰", root)
	}
	prev := root
	for wx := start + 6; wx <= start+trailLen-3; wx += surfaceStep {
		w := width(wx)
		if w > prev+int(surfaceStep) {
			t.Fatalf("拖尾在 %v 处反而变宽到 %d，根部没有收窄", wx, w)
		}
		prev = w
	}
	if end := width(start + trailLen - 3); end > 2*int(bodyRadius)+int(surfaceStep) {
		t.Fatalf("拖尾末端 %d 宽，应该收到身体直径", end)
	}
}

func TestOilBottleLeaksThinTrailAsItFlies(t *testing.T) {
	out := make(chan unit.Cmd, 512)
	f := &油火{owner: 1}
	bottle := func(x float64) unit.Snapshot {
		return unit.Snapshot{ID: 7, Kind: KindArsonOil, Role: unit.RoleProjectile,
			Slot: 0, OwnerID: 1, HP: 1, MaxHP: 1, Radius: bottleRadius, X: x}
	}
	for i := 0; i <= 20; i++ {
		f.sense(out, float64(i)/60, []unit.Snapshot{ownerAt(), bottle(float64(i) * 5)})
	}
	for x := 5.0; x <= 100; x += surfaceStep {
		if stateAt(&f.surface, x, 0) != oil {
			t.Fatalf("油瓶飞过的路径在 x=%v 处断了", x)
		}
	}
	for _, y := range []float64{12, -12} {
		if stateAt(&f.surface, 60, y) == oil {
			t.Fatalf("y=%v 也有油，说明瓶迹宽得和单位拖尾一样", y)
		}
	}
}

func TestFireBottleLeavesNoOil(t *testing.T) {
	out := make(chan unit.Cmd, 512)
	f := &油火{owner: 1}
	for i := 0; i <= 20; i++ {
		f.sense(out, float64(i)/60, []unit.Snapshot{ownerAt(), {
			ID: 7, Kind: KindArsonFire, Role: unit.RoleProjectile, Slot: 0, OwnerID: 1,
			HP: 1, MaxHP: 1, Radius: bottleRadius, X: float64(i) * 5,
		}})
	}
	if n := countState(&f.surface, oil); n != 0 {
		t.Fatalf("火瓶飞过的路径上留下了 %d 格油", n)
	}
}

func TestTinyBodyStillPaintsCoverage(t *testing.T) {
	g := oilSurface{}
	g.addOil(0, 0, 90, 0, 1, 0)
	for x := 0.0; x <= 90; x += surfaceStep {
		if !g.touches(unit.Snapshot{X: x, Y: 0, Radius: 1}, oil) {
			t.Fatalf("tiny trail has a gap at %v", x)
		}
	}
	if !g.touches(unit.Snapshot{X: 0, Y: 0, Radius: 1}, oil) {
		t.Fatal("tiny body left no oil")
	}
}

func TestOilBurnsTwoSecondsBottleFireOneSecond(t *testing.T) {
	g := oilSurface{}
	g.set(point{1, 0}, oil)
	g.addFire(1.5, 1.5, 1, 0)
	g.advance(.04)
	if g.cell(point{1, 0}).state != flame {
		t.Fatal("oil did not catch fire")
	}
	g.advance(1.01)
	if stateAt(&g, 1.5, 1.5) != dry || stateAt(&g, 4.5, 1.5) != flame {
		t.Fatal("bottle fire and burning oil must have independent lifetimes")
	}
	g.advance(2.04)
	if len(g.chunks) != 0 {
		t.Fatal("burning oil did not expire after two seconds")
	}
}
func TestBurningVisualFollowsDamageLifetime(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	f := &油火{owner: 1}
	f.surface.addFire(0, 0, 36, 0)
	near := []unit.Snapshot{ownerAt(), foeAt(2, 0, 0)}
	f.sense(out, 0, near)
	near[1].X = 300
	cmds := f.sense(out, .25, near)
	read := func(cmds []unit.Cmd) []burningView {
		t.Helper()
		for _, c := range cmds {
			if fx, ok := c.(unit.FX); ok && fx.Name == "arson-burning" {
				var got []burningView
				if err := json.Unmarshal(fx.Data, &got); err != nil {
					t.Fatal(err)
				}
				return got
			}
		}
		t.Fatal("missing full burning visual packet")
		return nil
	}
	got := read(cmds)
	if len(got) != 1 || got[0].ID != 2 || got[0].Until != 1 {
		t.Fatalf("burn visual after leaving: %+v", got)
	}
	if got = read(f.sense(out, 1.1, near)); len(got) != 0 {
		t.Fatalf("expired burn visuals: %+v", got)
	}
}

func TestBottleIgnitesOverlappingOilForTwoSeconds(t *testing.T) {
	g := oilSurface{}
	g.set(point{0, 0}, oil)
	g.addFire(1.5, 1.5, 1, 0)
	g.advance(.8)
	g.addFire(1.5, 1.5, 1, .8)
	g.advance(1.9)
	if stateAt(&g, 1.5, 1.5) != flame {
		t.Fatal("bottle overlap shortened the burning oil lifetime")
	}
	g.advance(2.01)
	if stateAt(&g, 1.5, 1.5) != dry {
		t.Fatal("overlapping oil must burn for two seconds from ignition")
	}
}

func TestBurningEnemyIgnitesOilAndOilSlowsOnlyEnemies(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	f := &油火{owner: 1, slot: 0}
	f.surface.addOil(200, 0, 400, 0, 12, 0)
	ally := foeAt(3, 200, 0)
	ally.Slot = 0
	near := []unit.Snapshot{ownerAt(), foeAt(2, 200, 0), ally}
	cmds := f.sense(out, 0, near)
	if stateAt(&f.surface, 200, 0) != oil {
		t.Fatal("enemy who is not on fire ignited the oil")
	}
	if v, ok := fsMul(cmds, 2); !ok || v != oilSlow {
		t.Fatalf("enemy in oil should move at 60%%, got %v %v", v, ok)
	}
	if _, ok := fsMul(cmds, 1); ok {
		t.Fatal("arsonist should not be slowed by his own oil")
	}
	if _, ok := fsMul(cmds, 3); ok {
		t.Fatal("ally in oil should not be slowed")
	}
	cmds = f.sense(out, 0.2, []unit.Snapshot{ownerAt(), foeAt(2, 0, 0), ally})
	if !fsRemoved(cmds, 2) {
		t.Fatal("leaving oil should restore speed")
	}
	f.burn = map[uint64]*scorch{2: {on: true, next: 100, until: 5}}
	f.sense(out, 0.4, []unit.Snapshot{ownerAt(), foeAt(2, 200, 0)})
	if stateAt(&f.surface, 200, 0) != flame {
		t.Fatal("burning enemy did not ignite the oil he touched")
	}
	if stateAt(&f.surface, 400, 0) != oil {
		t.Fatal("burning body ignited the whole connected pool at once")
	}
}

func TestFireSlowsEnemiesLikeOil(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	f := &油火{owner: 1, slot: 0}
	f.surface.addFire(0, 0, fireRadius, 0)
	ally := foeAt(3, 0, 0)
	ally.Slot = 0
	cmds := f.sense(out, 0, []unit.Snapshot{ownerAt(), foeAt(2, 0, 0), ally})
	if v, ok := fsMul(cmds, 2); !ok || v != oilSlow {
		t.Fatalf("enemy in fire should move at 60%%, got %v %v", v, ok)
	}
	if _, ok := fsMul(cmds, 3); ok {
		t.Fatal("ally in fire should not be slowed")
	}
	cmds = f.sense(out, 0.2, []unit.Snapshot{ownerAt(), foeAt(2, 400, 0)})
	if !fsRemoved(cmds, 2) {
		t.Fatal("leaving fire should restore speed")
	}
}

func fsMul(cmds []unit.Cmd, id uint64) (float64, bool) {
	for _, c := range cmds {
		if fs, ok := c.(unit.AddFSComponent); ok && fs.UnitID == id && fs.Zone == unit.FSZoneM {
			return fs.Value, true
		}
	}
	return 0, false
}

func fsRemoved(cmds []unit.Cmd, id uint64) bool {
	for _, c := range cmds {
		if fs, ok := c.(unit.RemoveFSComponent); ok && fs.UnitID == id {
			return true
		}
	}
	return false
}
