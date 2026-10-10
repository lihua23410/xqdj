package 盾斧

import (
	"math"
	"testing"
	"xqdj/internal/unit"
)

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

func me(vx, vy float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindAxe, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, VX: vx, VY: vy, Radius: axeRadius,
	}
}

func foeAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 2, Kind: "筑墙者", Role: unit.RoleFighter, Slot: 1,
		X: x, Y: y, Radius: 18, AimPriority: unit.DefaultFighterAim,
	}
}

func lastDmg(cmds []unit.Cmd, to uint64) *unit.Damage {
	var d *unit.Damage
	for _, c := range cmds {
		if v, ok := c.(unit.Damage); ok && v.To == to {
			cp := v
			d = &cp
		}
	}
	return d
}

func countDmg(cmds []unit.Cmd, to uint64) (n int, sum float64) {
	for _, c := range cmds {
		if v, ok := c.(unit.Damage); ok && v.To == to {
			n++
			sum += v.Amount
		}
	}
	return
}

func TestFanSeekStartsSlashWithoutContactDamage(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &盾斧{form: formShield, hx: 0, hy: 1}
	ctx := unit.Context{ID: 1, Kind: KindAxe, Out: out}
	enemy := foeAt(0, 28)
	a.Handle(ctx, unit.Sense{Time: 1, Self: me(0, 165), Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	if a.step != stepSlash1 {
		t.Fatalf("step=%d want slash", a.step)
	}
	if a.form != formSword {
		t.Fatalf("form=%d want sword", a.form)
	}
	if n, _ := countDmg(cmds, 2); n != 0 {
		t.Fatalf("trigger must not damage: %v", cmds)
	}
}

func TestSlashHitsFighterForFiveThenEnergy(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &盾斧{form: formSword, hx: 0, hy: 1, step: stepSlash1, until: slashLife, slashN: 0}
	ctx := unit.Context{ID: 1, Kind: KindAxe, Out: out}
	enemy := foeAt(0, 28)
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 165), Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	n, sum := countDmg(cmds, 2)
	if n != 1 || math.Abs(sum-slashDmg) > 1e-9 {
		t.Fatalf("first tick dmg n=%d sum=%v cmds=%v", n, sum, cmds)
	}
	if a.energy != 1 {
		t.Fatalf("energy=%d want 1", a.energy)
	}
	a.Handle(ctx, unit.Sense{Time: slashGap, Self: me(0, 165), Nearby: []unit.Snapshot{enemy}})
	cmds = drain(out)
	n, sum = countDmg(cmds, 2)
	if n != 1 || math.Abs(sum-slashDmg) > 1e-9 {
		t.Fatalf("second tick dmg n=%d sum=%v", n, sum)
	}
	if a.energy != 1 {
		t.Fatalf("one slash max +1 energy, got %d", a.energy)
	}
}

func TestMinionAlsoGrantsEnergy(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &盾斧{form: formSword, hx: 0, hy: 1, step: stepSlash1, until: slashLife}
	ctx := unit.Context{ID: 1, Kind: KindAxe, Out: out}
	minion := unit.Snapshot{
		ID: 4, Role: unit.RoleMinion, Mortal: true, Slot: 1,
		X: 0, Y: 28, Radius: 14,
	}
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 165), Nearby: []unit.Snapshot{minion}})
	n, sum := countDmg(drain(out), 4)
	if n != 1 || math.Abs(sum-slashDmg) > 1e-9 {
		t.Fatalf("minion dmg n=%d sum=%v", n, sum)
	}
	if a.energy != 1 {
		t.Fatalf("只要造成伤害就该回能量，minion got %d", a.energy)
	}
}

func TestRedSwordAddsTwo(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &盾斧{form: formSword, redSword: true, hx: 0, hy: 1, step: stepSlash1, until: slashLife}
	ctx := unit.Context{ID: 1, Kind: KindAxe, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0, Self: me(0, 165), Nearby: []unit.Snapshot{foeAt(0, 28)}})
	d := lastDmg(drain(out), 2)
	if d == nil || math.Abs(d.Amount-(slashDmg+redBonus)) > 1e-9 {
		t.Fatalf("red sword dmg=%v", d)
	}
}

func TestThrustTimeoutGoesToFill(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &盾斧{form: formSword, hx: 0, hy: 1, step: stepThrust, until: 1.5, thrustDir: [2]float64{0, 1}}
	ctx := unit.Context{ID: 1, Kind: KindAxe, Out: out}
	a.Handle(ctx, unit.Sense{Time: 1.5, Self: me(0, 350)})
	if a.step != stepFill {
		t.Fatalf("step=%d want fill", a.step)
	}
	if a.form != formShield {
		t.Fatalf("back to shield, form=%d", a.form)
	}
}

func TestThrustHitStartsSecondSlash(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &盾斧{form: formSword, hx: 0, hy: 1, step: stepThrust, slot: 0}
	ctx := unit.Context{ID: 1, Kind: KindAxe, Out: out}
	a.Handle(ctx, unit.Collision{
		Time:  1,
		Other: foeAt(10, 40),
	})
	cmds := drain(out)
	d := lastDmg(cmds, 2)
	if d == nil || math.Abs(d.Amount-thrustDmg) > 1e-9 {
		t.Fatalf("thrust dmg=%v", d)
	}
	if a.step != stepSlash2 {
		t.Fatalf("step=%d want slash2", a.step)
	}
}

func TestThrustAimsWhenSlashEndsNotAtSeek(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &盾斧{
		form: formSword, hx: 0, hy: 1, step: stepSlash1,
		until: slashLife, slashN: 2, lockID: 2, thrustDir: [2]float64{0, 1},
	}
	ctx := unit.Context{ID: 1, Kind: KindAxe, Out: out}
	a.Handle(ctx, unit.Sense{Time: slashLife, Self: me(0, 0), Nearby: []unit.Snapshot{foeAt(40, 10)}})
	if a.step != stepThrust {
		t.Fatalf("step=%d want thrust", a.step)
	}
	n := math.Hypot(40, 10)
	if math.Abs(a.thrustDir[0]-40/n) > 1e-6 || math.Abs(a.thrustDir[1]-10/n) > 1e-6 {
		t.Fatalf("thrustDir=%v want toward (40,10)", a.thrustDir)
	}
}

func TestFrontShieldHalvesDamage(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := &盾斧{form: formShield, hx: 0, hy: 1, x: 0, y: 0, seen: map[uint64][2]float64{9: {0, 80}}}
	ctx := unit.Context{ID: 1, Kind: KindAxe, Out: out}
	a.Handle(ctx, unit.IncomingDamage{Token: 3, From: 9, Amount: 10})
	cmds := drain(out)
	ok := false
	for _, c := range cmds {
		if v, okc := c.(unit.ConfirmDamage); okc && math.Abs(v.Amount-5) < 1e-9 {
			ok = true
		}
	}
	if !ok {
		t.Fatalf("want 5, cmds=%v", cmds)
	}
}

func TestEnergyOneFillsWhiteThenKeepsIt(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &盾斧{form: formShield, energy: 1, phial: phialEmpty, step: stepFill, until: 1}
	ctx := unit.Context{ID: 1, Kind: KindAxe, Out: out}
	a.Handle(ctx, unit.Sense{Time: 1, Self: me(0, 0)})
	if a.phial != phialWhite || a.step != stepBottle {
		t.Fatalf("phial=%d step=%d", a.phial, a.step)
	}
	a.until = 1.5
	a.Handle(ctx, unit.Sense{Time: 1.5, Self: me(0, 0)})
	if a.phial != phialWhite || a.redShield || a.step != stepIdle {
		t.Fatalf("white should stay, phial=%d red=%v step=%d", a.phial, a.redShield, a.step)
	}
}

func TestGoldPhialGrantsRedShield(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &盾斧{form: formShield, energy: 2, phial: phialEmpty, step: stepFill, until: 1}
	ctx := unit.Context{ID: 1, Kind: KindAxe, Out: out}
	a.Handle(ctx, unit.Sense{Time: 1, Self: me(0, 0)})
	a.until = 1.5
	a.Handle(ctx, unit.Sense{Time: 1.5, Self: me(0, 0)})
	if !a.redShield || a.phial != phialEmpty {
		t.Fatalf("redShield=%v phial=%d", a.redShield, a.phial)
	}
}

func TestGoldBothRedsEntersAxeAfterWindAndStand(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &盾斧{
		form: formShield, redShield: true, redSword: true,
		energy: 2, phial: phialEmpty, step: stepFill, until: 1,
		hx: 0, hy: 1,
	}
	ctx := unit.Context{ID: 1, Kind: KindAxe, Out: out}
	a.Handle(ctx, unit.Sense{Time: 1, Self: me(0, 0)})
	a.Handle(ctx, unit.Sense{Time: 1.5, Self: me(0, 0)})
	if a.step != stepAxeIn || a.form != formShield {
		t.Fatalf("after gold want axe wind, step=%d form=%d", a.step, a.form)
	}
	a.Handle(ctx, unit.Sense{Time: 1.7, Self: me(0, 0)})
	if a.step != stepAxeHold {
		t.Fatalf("step=%d want axe hold", a.step)
	}
	a.Handle(ctx, unit.Sense{Time: 2.7, Self: me(0, 0)})
	if a.form != formAxe || a.step != stepAxeIdle {
		t.Fatalf("form=%d step=%d want axe idle", a.form, a.step)
	}
	found := false
	for _, c := range drain(out) {
		if v, ok := c.(unit.SetCruise); ok && math.Abs(v.Speed-axeSlow) < 1e-9 {
			found = true
		}
	}
	if !found {
		t.Fatal("axe form should drop cruise")
	}
}

func TestAxeSegTurnsTowardEnemy(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &盾斧{form: formAxe, hx: 0, hy: 1, step: stepDai, until: 0.35, lockID: 2}
	ctx := unit.Context{ID: 1, Kind: KindAxe, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0.35, Self: me(0, 0), Nearby: []unit.Snapshot{foeAt(40, 10)}})
	n := math.Hypot(40, 10)
	if math.Abs(a.hx-40/n) > 1e-6 || math.Abs(a.hy-10/n) > 1e-6 {
		t.Fatalf("facing=%v,%v want toward (40,10)", a.hx, a.hy)
	}
	if a.step != stepTsui1 {
		t.Fatalf("step=%d want tsui1", a.step)
	}
}

func TestShieldCovers300ButNotBack(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := &盾斧{form: formShield, hx: 0, hy: 1, x: 0, y: 0, seen: map[uint64][2]float64{
		8: {80, 0},
		9: {0, -80},
	}}
	ctx := unit.Context{ID: 1, Kind: KindAxe, Out: out}
	a.Handle(ctx, unit.IncomingDamage{Token: 1, From: 8, Amount: 10})
	side := lastConfirm(drain(out))
	if side == nil || math.Abs(side.Amount-5) > 1e-9 {
		t.Fatalf("300° side want 5 got %v", side)
	}
	a.Handle(ctx, unit.IncomingDamage{Token: 2, From: 9, Amount: 10})
	back := lastConfirm(drain(out))
	if back == nil || math.Abs(back.Amount-10) > 1e-9 {
		t.Fatalf("behind want 10 got %v", back)
	}
}

func lastConfirm(cmds []unit.Cmd) *unit.ConfirmDamage {
	var d *unit.ConfirmDamage
	for _, c := range cmds {
		if v, ok := c.(unit.ConfirmDamage); ok {
			cp := v
			d = &cp
		}
	}
	return d
}

func TestInFanGeometry(t *testing.T) {
	self := me(0, 165)
	if !inFan(self, 0, 1, foeAt(0, 28), seekR, 120) {
		t.Fatal("ahead should hit")
	}
	if inFan(self, 0, 1, foeAt(80, 0), seekR, 120) {
		t.Fatal("side should miss")
	}
}

// 超解前先接一个突刺：追解收完进 stepChouThrust，以 350 速朝锁定目标顶过去；
// 撞到人结 3 伤，然后接超解旋（超时也接）。
func TestChouThrustBeforeSuperDischarge(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &盾斧{form: formAxe, hx: 0, hy: 1, step: stepTsui2, until: 1, lockID: 2}
	ctx := unit.Context{ID: 1, Kind: KindAxe, Out: out}

	a.Handle(ctx, unit.Sense{Time: 1, Self: me(0, 0), Nearby: []unit.Snapshot{foeAt(0, 60)}})
	if a.step != stepChouThrust {
		t.Fatalf("追解收完该先突刺，step=%d", a.step)
	}
	dashed := false
	for _, c := range drain(out) {
		if v, ok := c.(unit.SetVelocity); ok && math.Abs(math.Hypot(v.VX, v.VY)-thrustSp) < 1e-6 {
			dashed = true
		}
	}
	if !dashed {
		t.Fatal("突刺该以 350 速顶出去")
	}

	// 撞上人：3 伤 + 接超解旋
	a.Handle(ctx, unit.Collision{Time: 1.1, Other: foeAt(0, 40)})
	cmds := drain(out)
	if d := lastDmg(cmds, 2); d == nil || math.Abs(d.Amount-thrustDmg) > 1e-9 {
		t.Fatalf("突刺命中该结 %v，得到 %v", thrustDmg, d)
	}
	if a.step != stepChouSpin {
		t.Fatalf("命中后该接超解旋，step=%d", a.step)
	}

	// 另一条路：没撞到，走完 chouThrustLife 也接超解旋
	out2 := make(chan unit.Cmd, 32)
	b := &盾斧{form: formAxe, hx: 0, hy: 1, step: stepChouThrust, until: 0.5, lockID: 2}
	ctx2 := unit.Context{ID: 1, Kind: KindAxe, Out: out2}
	b.Handle(ctx2, unit.Sense{Time: 0.5, Self: me(0, 0), Nearby: []unit.Snapshot{foeAt(0, 200)}})
	if b.step != stepChouSpin {
		t.Fatalf("突刺超时也该接超解旋，step=%d", b.step)
	}
}

// 超解砸地：三处圆摊开、两两不重叠，最远落点比原来那套分环（104 以内）更远。
func TestChouSlamCirclesSpread(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	a := &盾斧{form: formAxe, hx: 0, hy: 1, step: stepChouWait, until: 1, lockID: 2}
	ctx := unit.Context{ID: 1, Kind: KindAxe, Out: out}

	var cx, cy, rr []float64
	for k := 0; k < 3; k++ {
		at := 1.0 + float64(k)*waveGap
		a.Handle(ctx, unit.Sense{Time: at, Self: me(0, 0), Nearby: []unit.Snapshot{foeAt(0, 200)}})
		for _, c := range drain(out) {
			fx, ok := c.(unit.FX)
			if !ok || fx.Name != "wave" {
				continue
			}
			cx = append(cx, fx.X)
			cy = append(cy, fx.Y)
			rr = append(rr, math.Hypot(fx.VX, fx.VY))
		}
	}
	if len(cx) != 3 {
		t.Fatalf("超解该落三处砸地，实际 %d 处", len(cx))
	}

	far := 0.0
	for i := 0; i < 3; i++ {
		if math.Abs(rr[i]-slamR) > 1e-9 {
			t.Fatalf("第 %d 处半径该是 %v，得到 %v", i+1, slamR, rr[i])
		}
		if d := math.Hypot(cx[i], cy[i]) + rr[i]; d > far {
			far = d
		}
	}
	// 两两圆心距离 ≥ 两半径之和 → 圆不重叠
	for i := 0; i < 3; i++ {
		for j := i + 1; j < 3; j++ {
			d := math.Hypot(cx[i]-cx[j], cy[i]-cy[j])
			if d < rr[i]+rr[j]-1e-9 {
				t.Fatalf("第 %d、%d 处砸地圆重叠了：圆心距 %.1f < %.1f", i+1, j+1, d, rr[i]+rr[j])
			}
		}
	}
	if far <= 104 {
		t.Fatalf("砸地范围该比原来的分环（104）更远，实际最远只到 %.1f", far)
	}
}
