package 囚徒

import (
	"math"
	"math/rand/v2"
	"testing"
	"xqdj/internal/unit"
)

func TestNoHookNoChain(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &囚徒{nextDrop: dropFirst}
	a.Handle(unit.Context{ID: 1, Kind: KindPrisoner, Out: out}, unit.Sense{
		Time:   0,
		Self:   selfAt(0, 0, 180, 0),
		Nearby: []unit.Snapshot{enemyAt(40, 0)},
	})
	cmds := drain(out)
	if lastDamage(cmds) != nil {
		t.Fatal("no 钩爪, no 锁链 damage")
	}
	if lastNamed(cmds, "chain") != nil {
		t.Fatal("no 钩爪, no chain fx")
	}
}

func TestCapsuleWallDoesNotPlant(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := &囚徒{nextDrop: dropFirst}
	ctx := unit.Context{ID: 1, Kind: KindPrisoner, Out: out}
	a.Handle(ctx, unit.Sense{Self: selfAt(0, 0, 180, 0)})
	_ = drain(out)
	a.Handle(ctx, unit.WallHit{NX: 1, NY: 0, Kind: unit.WallCapsule})
	a.Handle(ctx, unit.Sense{Self: selfAt(0, 0, 180, 0)})
	if lastNamed(drain(out), "chain") != nil {
		t.Fatal("capsule 墙 should not plant 钩爪")
	}
	if a.hooked {
		t.Fatal("hooked")
	}
}

func TestHexEdgePlantsOutward(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := &囚徒{nextDrop: dropFirst}
	ctx := unit.Context{ID: 1, Kind: KindPrisoner, Out: out}
	nx, ny := hexNormal(0)
	ap := unit.HexRadius * math.Sqrt(3) / 2
	x, y := nx*(ap-prisonerRadius), ny*(ap-prisonerRadius)
	a.Handle(ctx, unit.Sense{Self: selfAt(x, y, 180, 0)})
	_ = drain(out)
	a.Handle(ctx, unit.WallHit{NX: nx, NY: ny})
	a.Handle(ctx, unit.Sense{Self: selfAt(x, y, 180, 0)})
	fx := lastNamed(drain(out), "chain")
	if fx == nil {
		t.Fatal("missing chain")
	}
	wantX, wantY := x+nx*prisonerRadius, y+ny*prisonerRadius
	if math.Abs(fx.VX-wantX) > 1e-6 || math.Abs(fx.VY-wantY) > 1e-6 {
		t.Fatalf("hook=(%v,%v) want (%v,%v)", fx.VX, fx.VY, wantX, wantY)
	}
}

func TestSecondHexEdgeMovesHook(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := &囚徒{nextDrop: dropFirst}
	ctx := unit.Context{ID: 1, Kind: KindPrisoner, Out: out}
	nx, ny := hexNormal(0)
	ap := unit.HexRadius * math.Sqrt(3) / 2
	x, y := nx*(ap-prisonerRadius), ny*(ap-prisonerRadius)
	a.Handle(ctx, unit.Sense{Self: selfAt(x, y, 180, 0)})
	a.Handle(ctx, unit.WallHit{NX: nx, NY: ny})
	_ = drain(out)
	nx2, ny2 := hexNormal(2)
	x2, y2 := nx2*(ap-prisonerRadius), ny2*(ap-prisonerRadius)
	a.Handle(ctx, unit.Sense{Self: selfAt(x2, y2, 180, 0)})
	a.Handle(ctx, unit.WallHit{NX: nx2, NY: ny2})
	a.Handle(ctx, unit.Sense{Self: selfAt(x2, y2, 180, 0)})
	fx := lastNamed(drain(out), "chain")
	wantX, wantY := x2+nx2*prisonerRadius, y2+ny2*prisonerRadius
	if fx == nil || math.Abs(fx.VX-wantX) > 1e-6 || math.Abs(fx.VY-wantY) > 1e-6 {
		t.Fatalf("refreshed hook=%v want (%v,%v)", fx, wantX, wantY)
	}
}

func TestChainDoesNotDamage(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := planted(-prisonerRadius, 0)
	a.Handle(unit.Context{ID: 1, Kind: KindPrisoner, Out: out}, unit.Sense{
		Time:   1,
		Self:   selfAt(0, 0, 180, 0),
		Nearby: []unit.Snapshot{enemyAt(-8, 0)},
	})
	if lastDamage(drain(out)) != nil {
		t.Fatal("锁链 itself should not damage")
	}
}

func TestBodyCollisionDoesNotDamage(t *testing.T) {
	out := make(chan unit.Cmd, 4)
	a := planted(-prisonerRadius, 0)
	a.Handle(unit.Context{ID: 1, Kind: KindPrisoner, Out: out}, unit.Collision{
		Time:  2,
		Other: enemyAt(0, 0),
	})
	if lastDamage(drain(out)) != nil {
		t.Fatal("球撞人 should not damage")
	}
}

func TestSlackDoesNotConstrain(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := planted(-prisonerRadius, 0)
	a.Handle(unit.Context{ID: 1, Kind: KindPrisoner, Out: out}, unit.Sense{
		Self: selfAt(50, 0, 180, 0),
	})
	cmds := drain(out)
	if lastTeleport(cmds) != nil || lastVel(cmds) != nil {
		t.Fatalf("slack should cruise, cmds=%v", cmds)
	}
}

func TestTautReversesIncomingTangent(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := planted(-prisonerRadius, 0)
	self := selfAt(-prisonerRadius+chainLen+10, 0, 10, 180)
	a.Handle(unit.Context{ID: 1, Kind: KindPrisoner, Out: out}, unit.Sense{Self: self})
	cmds := drain(out)
	tp := lastTeleport(cmds)
	if tp == nil || math.Abs(tp.X-(-prisonerRadius+chainLen)) > 1e-6 || math.Abs(tp.Y) > 1e-6 {
		t.Fatalf("teleport=%v", tp)
	}
	v := lastVel(cmds)
	if v == nil {
		t.Fatal("missing SetVelocity")
	}
	speed := math.Hypot(10, 180)
	if math.Abs(math.Hypot(v.VX, v.VY)-speed) > 1e-6 {
		t.Fatalf("speed=%v want %v", math.Hypot(v.VX, v.VY), speed)
	}
	if math.Abs(v.VX) > 1e-6 || v.VY >= 0 {
		t.Fatalf("want −Y tangent (reversed), got (%v,%v)", v.VX, v.VY)
	}
}

func TestTautDoesNotFlipEveryTick(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := planted(-prisonerRadius, 0)
	ctx := unit.Context{ID: 1, Kind: KindPrisoner, Out: out}
	self := selfAt(-prisonerRadius+chainLen, 0, 0, 180)
	a.Handle(ctx, unit.Sense{Self: self})
	first := lastVel(drain(out))
	if first == nil || first.VY >= 0 {
		t.Fatalf("first=%v", first)
	}
	self.VX, self.VY = first.VX, first.VY
	a.Handle(ctx, unit.Sense{Self: self})
	second := lastVel(drain(out))
	if second == nil || second.VY >= 0 {
		t.Fatalf("second tick flipped back: %v", second)
	}
}

func TestCapsuleBounceFollowsTangent(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := planted(-prisonerRadius, 0)
	ctx := unit.Context{ID: 1, Kind: KindPrisoner, Out: out}
	self := selfAt(-prisonerRadius+chainLen, 0, 0, 180)
	a.Handle(ctx, unit.Sense{Self: self})
	first := lastVel(drain(out))
	if first == nil || first.VY >= 0 {
		t.Fatalf("first=%v", first)
	}
	a.Handle(ctx, unit.WallHit{NX: 0, NY: -1, Kind: unit.WallCapsule})
	if a.hx != -prisonerRadius || a.hy != 0 {
		t.Fatal("capsule should not move 钩爪")
	}
	self.VX, self.VY = 0, 180
	a.Handle(ctx, unit.Sense{Self: self})
	v := lastVel(drain(out))
	if v == nil || v.VY <= 0 {
		t.Fatalf("capsule bounce should follow bounced tangent, got %v", v)
	}
}

func TestTautZeroSpeedRestarts(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := planted(-prisonerRadius, 0)
	ctx := unit.Context{ID: 1, Kind: KindPrisoner, Out: out}
	self := selfAt(-prisonerRadius+chainLen, 0, 0, 180)
	a.Handle(ctx, unit.Sense{Self: self})
	_ = drain(out)
	self.VX, self.VY = 0, 0
	a.Handle(ctx, unit.Sense{Self: self})
	v := lastVel(drain(out))
	if v == nil || math.Abs(math.Hypot(v.VX, v.VY)-prisonerCruise) > 1e-6 {
		t.Fatalf("stuck taut should resume cruise, got %v", v)
	}
}

func TestAcceptsIncomingDamage(t *testing.T) {
	out := make(chan unit.Cmd, 4)
	a := &囚徒{nextDrop: dropFirst}
	a.Handle(unit.Context{ID: 1, Kind: KindPrisoner, Out: out}, unit.IncomingDamage{
		Token: 4, Amount: 10, Time: 1,
	})
	if _, ok := drain(out)[0].(unit.ConfirmDamage); !ok {
		t.Fatal("should confirm")
	}
}

func TestDropWindsThenLands(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &囚徒{nextDrop: dropFirst, rng: rand.New(rand.NewPCG(1, 2))}
	ctx := unit.Context{ID: 1, Kind: KindPrisoner, Out: out}
	enemy := enemyAt(0, 0)
	a.Handle(ctx, unit.Sense{Time: 4.9, Self: selfAt(0, 0, 180, 0), Nearby: []unit.Snapshot{enemy}})
	if nSpawn(drain(out)) != 0 {
		t.Fatal("too early")
	}
	a.Handle(ctx, unit.Sense{Time: 5, Self: selfAt(0, 0, 180, 0), Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	if nNamed(cmds, "drop") != 3 {
		t.Fatalf("windup drops=%d", nNamed(cmds, "drop"))
	}
	if nSpawn(cmds) != 0 {
		t.Fatal("should not land during windup")
	}
	a.Handle(ctx, unit.Sense{Time: 6, Self: selfAt(0, 0, 180, 0), Nearby: []unit.Snapshot{enemy}})
	land := drain(out)
	kinds := spawnKinds(land)
	if len(kinds) != 3 {
		t.Fatalf("landed=%v", kinds)
	}
	want := map[string]int{KindCage: 1, KindGallows: 1, KindChair: 1}
	for _, k := range kinds {
		want[k]--
	}
	for k, n := range want {
		if n != 0 {
			t.Fatalf("kind %s leftover %d in %v", k, n, kinds)
		}
	}
}

func TestDropSpotsKeepGap(t *testing.T) {
	a := &囚徒{nextDrop: dropFirst, rng: rand.New(rand.NewPCG(3, 4))}
	a.planDrop(unit.Sense{
		Self:   selfAt(0, 0, 180, 0),
		Nearby: []unit.Snapshot{enemyAt(40, 20)},
	})
	if math.Hypot(a.spots[0].x-40, a.spots[0].y-20) > 1 {
		t.Fatalf("enemy spot=%v", a.spots[0])
	}
	for i := 0; i < 3; i++ {
		if !unit.HexContains(a.spots[i].x, a.spots[i].y, cageFit) {
			t.Fatalf("spot %d out of cage fit: %v", i, a.spots[i])
		}
		for j := i + 1; j < 3; j++ {
			d := math.Hypot(a.spots[i].x-a.spots[j].x, a.spots[i].y-a.spots[j].y)
			if d < spotGap {
				t.Fatalf("spots %d/%d gap %v", i, j, d)
			}
		}
	}
}

func TestShockSetsCruiseAndRefreshes(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := &囚徒{nextDrop: dropFirst}
	ctx := unit.Context{ID: 1, Kind: KindPrisoner, Out: out}
	self := selfAt(0, 0, 180, 0)
	self.Marks = []unit.Mark{{Kind: shockKind, Stacks: 1}}
	a.Handle(ctx, unit.Sense{Time: 2, Self: self})
	cmds := drain(out)
	if lastCruise(cmds) == nil || lastCruise(cmds).Speed != shockCruise {
		t.Fatalf("cruise=%v", lastCruise(cmds))
	}
	if lastNamed(cmds, "scream") == nil {
		t.Fatal("missing scream")
	}
	self.Marks = nil
	a.Handle(ctx, unit.Sense{Time: 4, Self: self})
	_ = drain(out)
	self.Marks = []unit.Mark{{Kind: shockKind, Stacks: 1}}
	a.Handle(ctx, unit.Sense{Time: 4, Self: self})
	_ = drain(out)
	self.Marks = nil
	a.Handle(ctx, unit.Sense{Time: 8.9, Self: self})
	if lastCruise(drain(out)) != nil {
		t.Fatal("should still be 触电")
	}
	a.Handle(ctx, unit.Sense{Time: 9, Self: self})
	c := lastCruise(drain(out))
	if c == nil || c.Speed != prisonerCruise {
		t.Fatalf("restore=%v", c)
	}
}

func TestCagePlacesEightWallsThenDespawns(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &囚笼{owner: 1, slot: 0}
	ctx := unit.Context{ID: 9, Kind: KindCage, Out: out}
	a.Handle(ctx, unit.Sense{Time: 6, Self: unit.Snapshot{ID: 9, X: 10, Y: 20, Slot: 0}})
	walls := wallsOf(drain(out))
	if len(walls) != 8 {
		t.Fatalf("walls=%d", len(walls))
	}
	for _, w := range walls {
		if w.Radius != cageWall || w.Amount != cageDamage || w.Life != gearLife || w.OwnerID != 1 {
			t.Fatalf("wall=%+v", w)
		}
		d1 := math.Hypot(w.X1-10, w.Y1-20)
		d2 := math.Hypot(w.X2-10, w.Y2-20)
		if math.Abs(d1-cageRing) > 1e-6 || math.Abs(d2-cageRing) > 1e-6 {
			t.Fatalf("ring dist %v %v", d1, d2)
		}
	}
	a.Handle(ctx, unit.Sense{Time: 16, Self: unit.Snapshot{ID: 9, X: 10, Y: 20}})
	if !hasDespawn(drain(out)) {
		t.Fatal("cage should despawn at 10s")
	}
}

func TestGallowsLocksOnTouch(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &绞刑架{owner: 1, slot: 0}
	ctx := unit.Context{ID: 9, Kind: KindGallows, Out: out}
	self := unit.Snapshot{ID: 9, Kind: KindGallows, Role: unit.RoleHelper, Slot: 0, X: 0, Y: 0, Radius: gallowsR}
	enemy := enemyAt(0, 0)
	enemy.VX, enemy.VY = 90, 40
	hit := unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{enemy}}
	a.Handle(ctx, hit)
	cmds := drain(out)
	if lastStun(cmds) == nil || !lastStun(cmds).Hold || lastStun(cmds).UnitID != 2 {
		t.Fatalf("stun=%v", lastStun(cmds))
	}
	if lastVel(cmds) == nil || lastVel(cmds).VX != 0 || lastVel(cmds).VY != 0 {
		t.Fatal("velocity should be 0")
	}
	if lastDamage(cmds) == nil || lastDamage(cmds).Amount != gallowsDmg || lastDamage(cmds).To != 2 {
		t.Fatalf("dmg=%v", lastDamage(cmds))
	}
	if p := lastPass(cmds); p == nil || !p.Hold || p.UnitID != 9 {
		t.Fatalf("pass=%v", p)
	}
	hit.Time = 1.2
	a.Handle(ctx, hit)
	if lastDamage(drain(out)) != nil {
		t.Fatal("still in 0.6s")
	}
	hit.Time = 1.6
	a.Handle(ctx, hit)
	if lastDamage(drain(out)) == nil {
		t.Fatal("tick after CD")
	}
	hit.Time = 11
	a.Handle(ctx, hit)
	end := drain(out)
	st := lastStun(end)
	if st == nil || st.Hold {
		t.Fatalf("release stun=%v", st)
	}
	v := lastVel(end)
	if v == nil || math.Abs(v.VX-90) > 1e-9 || math.Abs(v.VY-40) > 1e-9 {
		t.Fatalf("release vel=%v want (90,40)", v)
	}
	if c := lastCruise(end); c != nil && c.Speed == 0 {
		t.Fatal("release should not leave cruise 0")
	}
	if !hasDespawn(end) {
		t.Fatal("gallows should despawn")
	}
}

func TestChairHoldsPass(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	a := &电椅{owner: 1, slot: 0, rng: rand.New(rand.NewPCG(1, 2))}
	ctx := unit.Context{ID: 9, Kind: KindChair, Out: out}
	self := unit.Snapshot{ID: 9, Kind: KindChair, Role: unit.RoleHelper, Slot: 0, X: 0, Y: 0, Radius: chairR}
	a.Handle(ctx, unit.Sense{Time: 1, Self: self})
	if p := lastPass(drain(out)); p == nil || !p.Hold || p.UnitID != 9 {
		t.Fatalf("pass=%v", p)
	}
}

func TestChairBoltsHitEnemyNotSelf(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	a := &电椅{owner: 1, slot: 0}
	ctx := unit.Context{ID: 9, Kind: KindChair, Out: out}
	self := unit.Snapshot{ID: 9, Kind: KindChair, Role: unit.RoleHelper, Slot: 0, X: 0, Y: 0, Radius: chairR}
	owner := unit.Snapshot{ID: 1, Kind: KindPrisoner, Role: unit.RoleFighter, Slot: 0, X: 0, Y: 0, Radius: prisonerRadius}
	enemy := enemyAt(40, 0)
	bolts := [][]vec{{{0, 0}, {80, 0}}}
	a.strike(ctx, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{owner, enemy}}, bolts)
	cmds := drain(out)
	if lastDamage(cmds) == nil || lastDamage(cmds).To != 2 || lastDamage(cmds).Amount != currentDmg {
		t.Fatalf("enemy dmg=%v", lastDamage(cmds))
	}
	for _, c := range cmds {
		if d, ok := c.(unit.Damage); ok && d.To == 1 {
			t.Fatal("current should not hit 囚徒球")
		}
	}
	if nNamed(cmds, "bolt") != 1 {
		t.Fatalf("bolt fx=%d", nNamed(cmds, "bolt"))
	}
}

func TestChairChainMarksShock(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &电椅{owner: 1, slot: 0}
	ctx := unit.Context{ID: 9, Kind: KindChair, Out: out}
	self := unit.Snapshot{ID: 9, Kind: KindChair, Role: unit.RoleHelper, Slot: 0, X: 0, Y: 0, Radius: chairR}
	body := unit.Snapshot{ID: 1, Kind: KindPrisoner, Role: unit.RoleFighter, Slot: 0, OwnerID: 0, X: 80, Y: 0, Radius: prisonerRadius}
	hook := unit.Snapshot{ID: 8, Kind: KindHook, Role: unit.RoleHelper, OwnerID: 1, X: 0, Y: 0, Radius: 8}
	bolts := [][]vec{{{0, 2}, {80, 2}}}
	a.strike(ctx, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{body, hook}}, bolts)
	cmds := drain(out)
	m := lastMark(cmds)
	if m == nil || m.UnitID != 1 || m.Kind != shockKind {
		t.Fatalf("mark=%v", m)
	}
	if lastDamage(cmds) != nil {
		t.Fatal("chain hit should not also damage the ball")
	}
}

func TestBoltsDoNotCross(t *testing.T) {
	rng := rand.New(rand.NewPCG(9, 8))
	for n := 0; n < 20; n++ {
		bolts := genBolts(0, 0, rng)
		if len(bolts) < currentNMin || len(bolts) > currentNMax {
			t.Fatalf("n=%d", len(bolts))
		}
		var used [][4]float64
		for _, b := range bolts {
			if len(b) < 3 || len(b) > 5 {
				t.Fatalf("pts=%d want 2–4 segs (1–3 bends)", len(b))
			}
			if boltCrosses(b, used) {
				t.Fatal("crossed")
			}
			L := 0.0
			for i := 0; i+1 < len(b); i++ {
				L += math.Hypot(b[i+1].x-b[i].x, b[i+1].y-b[i].y)
				used = append(used, [4]float64{b[i].x, b[i].y, b[i+1].x, b[i+1].y})
			}
			if L < currentMinL-1e-6 || L > currentMaxL+1e-6 {
				t.Fatalf("len=%v", L)
			}
		}
	}
}

func planted(hx, hy float64) *囚徒 {
	return &囚徒{hooked: true, hx: hx, hy: hy, r: prisonerRadius, nextDrop: dropFirst}
}

func selfAt(x, y, vx, vy float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindPrisoner, Role: unit.RoleFighter, Slot: 0,
		X: x, Y: y, VX: vx, VY: vy, Radius: prisonerRadius, Vision: prisonerVision,
	}
}

func enemyAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: x, Y: y, Radius: 18}
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

func lastNamed(cmds []unit.Cmd, name string) *unit.FX {
	var fx *unit.FX
	for _, c := range cmds {
		if v, ok := c.(unit.FX); ok && v.Name == name {
			cp := v
			fx = &cp
		}
	}
	return fx
}

func nNamed(cmds []unit.Cmd, name string) int {
	n := 0
	for _, c := range cmds {
		if v, ok := c.(unit.FX); ok && v.Name == name {
			n++
		}
	}
	return n
}

func nSpawn(cmds []unit.Cmd) int {
	n := 0
	for _, c := range cmds {
		if _, ok := c.(unit.Spawn); ok {
			n++
		}
	}
	return n
}

func spawnKinds(cmds []unit.Cmd) []string {
	var kinds []string
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok {
			kinds = append(kinds, s.Kind)
		}
	}
	return kinds
}

func lastVel(cmds []unit.Cmd) *unit.SetVelocity {
	var v *unit.SetVelocity
	for _, c := range cmds {
		if x, ok := c.(unit.SetVelocity); ok {
			cp := x
			v = &cp
		}
	}
	return v
}

func lastTeleport(cmds []unit.Cmd) *unit.Teleport {
	var tp *unit.Teleport
	for _, c := range cmds {
		if x, ok := c.(unit.Teleport); ok {
			cp := x
			tp = &cp
		}
	}
	return tp
}

func lastCruise(cmds []unit.Cmd) *unit.SetCruise {
	var v *unit.SetCruise
	for _, c := range cmds {
		if x, ok := c.(unit.SetCruise); ok {
			cp := x
			v = &cp
		}
	}
	return v
}

func lastStun(cmds []unit.Cmd) *unit.Stun {
	var v *unit.Stun
	for _, c := range cmds {
		if x, ok := c.(unit.Stun); ok {
			cp := x
			v = &cp
		}
	}
	return v
}

func lastPass(cmds []unit.Cmd) *unit.Pass {
	var v *unit.Pass
	for _, c := range cmds {
		if x, ok := c.(unit.Pass); ok {
			cp := x
			v = &cp
		}
	}
	return v
}

func lastMark(cmds []unit.Cmd) *unit.StackMark {
	var v *unit.StackMark
	for _, c := range cmds {
		if x, ok := c.(unit.StackMark); ok {
			cp := x
			v = &cp
		}
	}
	return v
}

func wallsOf(cmds []unit.Cmd) []unit.PlaceWall {
	var out []unit.PlaceWall
	for _, c := range cmds {
		if w, ok := c.(unit.PlaceWall); ok {
			out = append(out, w)
		}
	}
	return out
}

func hasDespawn(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if _, ok := c.(unit.Despawn); ok {
			return true
		}
	}
	return false
}
