package 阈值

import (
	"math"
	"testing"

	"xqdj/internal/unit"

	// 场地外形判定由 map 包在 init 里登记；随机落点要用 Field.RandomWalkable。
	_ "xqdj/map/hex"
)

func TestSmallHitLeavesLayers(t *testing.T) {
	a := &阈值{layers: layerMin}
	ctx, out := newCtx()
	a.Handle(ctx, unit.IncomingDamage{Token: 1, Amount: 1, Time: 0.2})
	cmds := drain(out)
	if got := confirmed(cmds); got != 1 {
		t.Fatalf("confirmed %v", got)
	}
	if a.layers != layerMin {
		t.Fatalf("layers=%v", a.layers)
	}
}

func TestEqualHitDoesNotRaise(t *testing.T) {
	a := &阈值{layers: layerMin}
	ctx, out := newCtx()
	a.Handle(ctx, unit.IncomingDamage{Token: 1, Amount: layerMin})
	if confirmed(drain(out)) != layerMin || a.layers != layerMin {
		t.Fatalf("layers=%v", a.layers)
	}
}

func TestOverflowCapsAndBanksHalf(t *testing.T) {
	a := &阈值{layers: layerMin}
	ctx, out := newCtx()
	a.Handle(ctx, unit.IncomingDamage{Token: 1, Amount: 15})
	if confirmed(drain(out)) != layerMin {
		t.Fatal("did not cap at the current layers")
	}
	// 3 层吃 15：扣 3，层数 = 3 + (15-3)/2 = 9。
	if math.Abs(a.layers-9) > 1e-9 {
		t.Fatalf("layers=%v", a.layers)
	}
	a.Handle(ctx, unit.IncomingDamage{Token: 2, Amount: 30})
	if math.Abs(confirmed(drain(out))-9) > 1e-9 {
		t.Fatal("second hit did not cap at the raised layers")
	}
	// 9 + (30-9)/2 = 19.5。
	if math.Abs(a.layers-19.5) > 1e-9 {
		t.Fatalf("layers=%v", a.layers)
	}
}

func TestShotPlacesSealAndSpends(t *testing.T) {
	a := &阈值{layers: 20}
	ctx, out := newCtx()
	a.Handle(ctx, senseAt(0))
	cmds := drain(out)
	sp, g, ok := sealOf(cmds)
	if !ok {
		t.Fatal("no seal")
	}
	// 20 层外接圆半径 320，六边形里放不下，落到场心，不跟在 (100, 0) 的目标上。
	if sp.X != 0 || sp.Y != 0 {
		t.Fatalf("placed at %+v", sp)
	}
	if g.sides != 20 || math.Abs(g.damage-shotDamage(20)) > 1e-9 || math.Abs(g.reach-sealReach(20)) > 1e-9 || math.Abs(g.life-sealLife(20)) > 1e-9 {
		t.Fatalf("seal=%+v", g)
	}
	if math.Abs(a.fireReady-fireInterval(20)) > 1e-9 {
		t.Fatalf("gap=%v", a.fireReady)
	}
	// 20 层出手，扣掉 (20-3)/2 = 8.5，剩下 11.5。
	if math.Abs(a.layers-11.5) > 1e-9 {
		t.Fatalf("layers=%v", a.layers)
	}
	if velocity(cmds) != nil {
		t.Fatal("drawing should not change own velocity")
	}
}

func TestReachGrowsWithLayers(t *testing.T) {
	if sealReach(layerMin) != sealRadiusAtMin {
		t.Fatalf("min reach=%v", sealReach(layerMin))
	}
	if sealLife(layerMin) != sealLifeAtMin || sealLife(6) != sealLifeAtMin*2 {
		t.Fatalf("life 3=%v 6=%v", sealLife(layerMin), sealLife(6))
	}
	if sealReach(6) != sealRadiusAtMin*2 || sealReach(20) <= sealReach(6) {
		t.Fatalf("3=%v 6=%v 20=%v", sealReach(layerMin), sealReach(6), sealReach(20))
	}
}

func TestSealLandsAwayFromTarget(t *testing.T) {
	a := &阈值{layers: layerMin}
	ctx, out := newCtx()
	ctx.Stream = 7
	a.Handle(ctx, senseAt(0))
	sp, g, ok := sealOf(drain(out))
	x, y := placeSeal(unit.NewRNG(7), unit.Field{}, sealReach(layerMin))
	if !ok || sp.X != x || sp.Y != y {
		t.Fatalf("got %v,%v want %v,%v ok=%v", sp.X, sp.Y, x, y, ok)
	}
	if sp.X == 100 && sp.Y == 0 {
		t.Fatal("landed on the target")
	}
	if math.Abs(g.reach-sealReach(layerMin)) > 1e-9 {
		t.Fatalf("reach=%v", g.reach)
	}
}

func TestShotFromFloorStaysAtFloor(t *testing.T) {
	a := &阈值{layers: layerMin}
	ctx, out := newCtx()
	a.Handle(ctx, senseAt(0))
	_, g, ok := sealOf(drain(out))
	if !ok || g.sides != 3 || math.Abs(g.damage-layerMin) > 1e-9 {
		t.Fatalf("seal=%+v ok=%v", g, ok)
	}
	if a.layers != layerMin {
		t.Fatalf("layers=%v", a.layers)
	}
}

func TestFractionalLayersFloorTheSides(t *testing.T) {
	a := &阈值{layers: 4.2}
	ctx, out := newCtx()
	a.Handle(ctx, senseAt(0))
	_, g, ok := sealOf(drain(out))
	if !ok || g.sides != 4 || math.Abs(g.damage-4.2) > 1e-9 {
		t.Fatalf("seal=%+v ok=%v", g, ok)
	}
}

func TestLowerLayersFireFaster(t *testing.T) {
	low := &阈值{layers: layerMin}
	ctx, out := newCtx()
	low.Handle(ctx, senseAt(0))
	if _, _, ok := sealOf(drain(out)); !ok {
		t.Fatal("opening seal missing")
	}
	low.Handle(ctx, senseAt(fireAtMin/2))
	if _, _, ok := sealOf(drain(out)); ok {
		t.Fatal("seal inside the opening gap")
	}
	low.Handle(ctx, senseAt(fireAtMin))
	if _, _, ok := sealOf(drain(out)); !ok {
		t.Fatal("did not draw when the opening gap ended")
	}

	high := &阈值{layers: 20}
	high.Handle(ctx, senseAt(0))
	if _, _, ok := sealOf(drain(out)); !ok {
		t.Fatal("heavy seal missing")
	}
	high.Handle(ctx, senseAt(fireAtMin))
	if _, _, ok := sealOf(drain(out)); ok {
		t.Fatal("20 layers drew on the opening gap")
	}
	high.Handle(ctx, senseAt(fireInterval(20)))
	if _, _, ok := sealOf(drain(out)); !ok {
		t.Fatal("20 layers did not draw when its own gap ended")
	}
}

func TestNoTargetDoesNotSpendLayers(t *testing.T) {
	a := &阈值{layers: 12}
	ctx, out := newCtx()
	a.Handle(ctx, unit.Sense{Time: 0, Self: unit.Snapshot{ID: 1, Slot: 0, Radius: bodyRadius}})
	if _, _, ok := sealOf(drain(out)); ok || a.layers != 12 || a.fireReady != 0 {
		t.Fatalf("spent without a target: layers=%v ready=%v", a.layers, a.fireReady)
	}
}

func TestTriangleIgnoresCircumcircleCorner(t *testing.T) {
	// 等边三角形内切圆半径是外接圆的一半。沿一条边的外法线走到 48，
	// 人还在外接圆里，但已经离开三角形。半径 1 碰不到那条边。
	const reach = 64.0
	ang := -math.Pi / 6
	x := math.Cos(ang) * 48
	y := math.Sin(ang) * 48
	if sealTouches(0, 0, 3, reach, 0, x, y, 1) {
		t.Fatal("point inside the circumcircle but outside the triangle was hit")
	}
	if !sealTouches(0, 0, 3, reach, 0, 0, 0, 18) {
		t.Fatal("center was missed")
	}
	// 边在距圆心 32 处。圆心再往外 17，半径 18，擦到边。
	tx := math.Cos(ang) * (32 + 17)
	ty := math.Sin(ang) * (32 + 17)
	if !sealTouches(0, 0, 3, reach, 0, tx, ty, 18) {
		t.Fatal("body touching the edge was missed")
	}
	mx := math.Cos(ang) * (32 + 18 + 4)
	my := math.Sin(ang) * (32 + 18 + 4)
	if sealTouches(0, 0, 3, reach, 0, mx, my, 18) {
		t.Fatal("body clear of the edge was hit")
	}
}

func TestSpinTurnsThePolygon(t *testing.T) {
	// 尖朝上时，这条边的外侧 40 处在三角形外。转半圈后尖转到这边，这个点进到里面。
	ang := -math.Pi / 6
	x := math.Cos(ang) * 40
	y := math.Sin(ang) * 40
	if sealTouches(0, 0, 3, 64, 0, x, y, 1) {
		t.Fatal("upright triangle should miss this point")
	}
	if !sealTouches(0, 0, 3, 64, math.Pi, x, y, 1) {
		t.Fatal("half turn should cover this point")
	}
}

func TestSealWaitsThenHurtsTouch(t *testing.T) {
	b := &阈值印{owner: 9, slot: 0, sides: 3, damage: 6, reach: sealRadiusAtMin, life: 2}
	ctx, out := newCtx()
	near := unit.Snapshot{ID: 2, Slot: 1, Role: unit.RoleFighter, X: 0, Y: 0, Radius: 18, HP: 100}
	far := unit.Snapshot{ID: 3, Slot: 1, Role: unit.RoleFighter, X: 400, Y: 0, Radius: 18, HP: 100}
	ally := unit.Snapshot{ID: 4, Slot: 0, Role: unit.RoleFighter, X: 0, Y: 0, Radius: 18, HP: 100}
	sense := func(t float64, xs ...unit.Snapshot) {
		b.Handle(ctx, unit.Sense{Time: t, Self: unit.Snapshot{X: 0, Y: 0}, Nearby: xs})
	}
	sense(0, near, far, ally)
	if got := damages(drain(out)); len(got) != 0 {
		t.Fatalf("settled during the telegraph: %+v", got)
	}
	sense(0.5, near, far, ally)
	if got := damages(drain(out)); len(got) != 0 {
		t.Fatalf("settled early: %+v", got)
	}
	sense(1, near, far, ally)
	cmds := drain(out)
	got := damages(cmds)
	if len(got) != 1 || got[0].To != near.ID || got[0].Amount != 6 {
		t.Fatalf("hits=%+v", got)
	}
	if despawned(cmds) {
		t.Fatal("seal vanished on the first check")
	}
	sense(1.2, near, far, ally)
	if got := damages(drain(out)); len(got) != 0 {
		t.Fatalf("same body was hit again: %+v", got)
	}
	walker := unit.Snapshot{ID: 5, Slot: 1, Role: unit.RoleFighter, X: 0, Y: 0, Radius: 18, HP: 100}
	sense(2, walker)
	got = damages(drain(out))
	if len(got) != 1 || got[0].To != walker.ID {
		t.Fatalf("walker=%+v", got)
	}
	sense(3, near, walker)
	cmds = drain(out)
	if got := damages(cmds); len(got) != 0 {
		t.Fatalf("hit while fading: %+v", got)
	}
	if !despawned(cmds) {
		t.Fatal("seal stayed after its life")
	}
}

func senseAt(t float64) unit.Sense {
	return unit.Sense{
		Time: t,
		Self: unit.Snapshot{ID: 1, Slot: 0, Radius: bodyRadius, HP: bodyHP, AimPriority: 15},
		Nearby: []unit.Snapshot{{
			ID: 2, Slot: 1, Role: unit.RoleFighter, X: 100, HP: 100, AimPriority: 15,
		}},
	}
}

func newCtx() (unit.Context, chan unit.Cmd) {
	out := make(chan unit.Cmd, 8)
	return unit.Context{ID: 7, Kind: KindThreshold, Out: out}.WithBirths(), out
}

func drain(out chan unit.Cmd) []unit.Cmd {
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

func confirmed(cmds []unit.Cmd) float64 {
	for _, c := range cmds {
		d, ok := c.(unit.ConfirmDamage)
		if ok {
			return d.Amount
		}
	}
	return -1
}

func sealOf(cmds []unit.Cmd) (unit.Spawn, seal, bool) {
	for i := range cmds {
		s, ok := cmds[i].(unit.Spawn)
		if ok && s.Kind == kindSeal {
			g, ok := takeSeal(s.Stream)
			return s, g, ok
		}
	}
	return unit.Spawn{}, seal{}, false
}

func velocity(cmds []unit.Cmd) *unit.SetVelocity {
	for i := range cmds {
		v, ok := cmds[i].(unit.SetVelocity)
		if ok {
			return &v
		}
	}
	return nil
}

func damages(cmds []unit.Cmd) []unit.Damage {
	var out []unit.Damage
	for _, c := range cmds {
		d, ok := c.(unit.Damage)
		if ok {
			out = append(out, d)
		}
	}
	return out
}

func despawned(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if _, ok := c.(unit.Despawn); ok {
			return true
		}
	}
	return false
}
