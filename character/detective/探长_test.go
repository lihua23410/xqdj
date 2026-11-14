package 探长

import (
	"math"
	"testing"

	"xqdj/internal/unit"
	_ "xqdj/map/hex"
)

func TestOdds(t *testing.T) {
	if math.Abs(odds(0)-0.10) > 1e-9 {
		t.Fatalf("0 层应是 10%%，实际 %v", odds(0))
	}
	if math.Abs(odds(7)-0.50) > 1e-9 {
		t.Fatalf("7 层应是 50%%，实际 %v", odds(7))
	}
	if odds(6) >= 0.50 {
		t.Fatalf("6 层应低于 50%%，实际 %v", odds(6))
	}
	if odds(8) <= 0.50 || odds(8) >= 1 {
		t.Fatalf("8 层应高于 50%% 且未封顶，实际 %v", odds(8))
	}
	if odds(11) != 1 {
		t.Fatalf("11 层应封顶为必定，实际 %v", odds(11))
	}
}

func TestSuspectKeepsTie(t *testing.T) {
	b := newBook()
	b.addSuspicion(4)
	b.addSuspicion(9)
	b.addSuspicion(4)
	if b.snapshot().suspect != 4 {
		t.Fatalf("疑点并列应留下原来的嫌疑人，实际 %d", b.snapshot().suspect)
	}
	b.addSuspicion(9)
	b.addSuspicion(9)
	if b.snapshot().suspect != 9 {
		t.Fatalf("疑点被超过后应换人，实际 %d", b.snapshot().suspect)
	}
}

func TestSceneTouchOnceUntilLeave(t *testing.T) {
	st := &unit.MatchState{}
	m := &标记{owner: 1, slot: 0, mode: modeScene}
	ctx, out := pipe(st, 8)
	self := unit.Snapshot{ID: 8, X: 0, Y: 0, Radius: sceneRadius, Slot: 0}
	foe := unit.Snapshot{ID: 2, X: 10, Y: 0, Radius: 18, Role: unit.RoleFighter, Slot: 1}
	m.Handle(ctx, unit.Sense{Self: self, Nearby: []unit.Snapshot{foe}})
	m.Handle(ctx, unit.Sense{Self: self, Nearby: []unit.Snapshot{foe}})
	if got := bookOf(st, 1).snapshot().suspicion[2]; got != 1 {
		t.Fatalf("站在案发现场里只记一次疑点，实际 %d", got)
	}
	far := foe
	far.X = 400
	m.Handle(ctx, unit.Sense{Self: self, Nearby: []unit.Snapshot{far}})
	m.Handle(ctx, unit.Sense{Self: self, Nearby: []unit.Snapshot{foe}})
	if got := bookOf(st, 1).snapshot().suspicion[2]; got != 2 {
		t.Fatalf("离开再进来应再记一次，实际 %d", got)
	}
	if len(drain(out)) != 0 {
		t.Fatal("案发现场不该消失")
	}
}

func TestChargeTouchAddsSuspicionAndDespawns(t *testing.T) {
	st := &unit.MatchState{}
	m := &标记{owner: 1, slot: 0, mode: modeCharge}
	ctx, out := pipe(st, 8)
	self := unit.Snapshot{ID: 8, X: 0, Y: 0, Radius: chargeRadius, Slot: 0}
	foe := unit.Snapshot{ID: 3, X: 0, Y: 0, Radius: 18, Role: unit.RoleFighter, Slot: 1}
	m.Handle(ctx, unit.Sense{Self: self, Nearby: []unit.Snapshot{foe}})
	m.Handle(ctx, unit.Sense{Self: self, Nearby: []unit.Snapshot{foe}})
	if got := bookOf(st, 1).snapshot().suspicion[3]; got != 1 {
		t.Fatalf("指控只记一次疑点，实际 %d", got)
	}
	cmds := drain(out)
	n := 0
	for _, c := range cmds {
		if _, ok := c.(unit.Despawn); ok {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("碰到指控应消失一次，实际 %d", n)
	}
}

func TestForgeryTouchAddsSuspicion(t *testing.T) {
	st := &unit.MatchState{}
	m := &标记{owner: 1, slot: 0, mode: modeForgery}
	ctx, _ := pipe(st, 8)
	self := unit.Snapshot{ID: 8, Radius: forgeryRadius, Slot: 0}
	foe := unit.Snapshot{ID: 3, Radius: 18, Role: unit.RoleFighter, Slot: 1}
	m.Handle(ctx, unit.Sense{Self: self, Nearby: []unit.Snapshot{foe}})
	view := bookOf(st, 1).snapshot()
	if view.suspicion[3] != 1 || view.admonition != 0 {
		t.Fatalf("伪证只记疑点，不给申诫，实际 %+v", view)
	}
}

func TestOpeningPlacesSceneAndCharges(t *testing.T) {
	st := &unit.MatchState{}
	a := &探长{}
	ctx, out := pipe(st, 1)
	ctx.Stream = 7
	foe := unit.Snapshot{ID: 2, X: 200, Y: 0, Radius: 18, Role: unit.RoleFighter, Slot: 1, Seat: 0}
	a.Handle(ctx, unit.Sense{
		Time: 0, Field: unit.HexField(),
		Self:   unit.Snapshot{ID: 1, Slot: 0, Radius: bodyRadius, Role: unit.RoleFighter},
		Nearby: []unit.Snapshot{foe},
	})
	cmds := drain(out)
	scene, charge := 0, 0
	for _, c := range cmds {
		sp, ok := c.(unit.Spawn)
		if !ok {
			continue
		}
		switch sp.Kind {
		case KindScene:
			scene++
		case KindCharge:
			charge++
		}
	}
	if scene != 1 || charge != 5 {
		t.Fatalf("一个敌方槽位应铺 1 个案发现场和 5 个指控，实际现场 %d 指控 %d", scene, charge)
	}
}

func TestFourContactsEnterJudgment(t *testing.T) {
	st := &unit.MatchState{}
	ctx, out := pipe(st, 1)
	a := &探长{ready: true, slot: 0, boot: 0, touching: map[contact]struct{}{}, born: ctx}
	self := unit.Snapshot{ID: 1, X: 0, Y: 0, Radius: 18, Role: unit.RoleFighter, Slot: 0}
	foe := unit.Snapshot{ID: 2, X: 20, Y: 0, Radius: 18, Role: unit.RoleFighter, Slot: 1}
	far := foe
	far.X = 200
	for i := 0; i < 4; i++ {
		a.Handle(ctx, unit.Sense{Time: float64(i), Self: self, Nearby: []unit.Snapshot{far}})
		drain(out)
		a.Handle(ctx, unit.Sense{Time: float64(i) + 0.1, Self: self, Nearby: []unit.Snapshot{foe}})
		if i < 3 {
			drain(out)
		}
	}
	if a.phase != phaseJudgment {
		t.Fatalf("调查满 4 次应进入审判，实际 %d", a.phase)
	}
	if !bookOf(st, 1).snapshot().guilty[2] {
		t.Fatal("被撞满 4 次的单位应有罪")
	}
	if countKind(drain(out), KindBullet) != 1 {
		t.Fatal("进入审判后应立刻打出一发")
	}
}

func TestBumpCountsInvestigation(t *testing.T) {
	st := &unit.MatchState{}
	ctx, out := pipe(st, 1)
	a := &探长{ready: true, slot: 0, born: ctx, touching: map[contact]struct{}{}}
	self := unit.Snapshot{ID: 1, X: 0, Y: 0, Radius: 18, Role: unit.RoleFighter, Slot: 0}
	foe := unit.Snapshot{ID: 2, X: 80, Y: 0, Radius: 18, Role: unit.RoleFighter, Slot: 1, VX: -160}
	a.Handle(ctx, unit.Collision{Time: 1, Other: foe})
	if bookOf(st, 1).snapshot().investigation[2] != 1 {
		t.Fatal("撞上应记一次调查")
	}
	a.Handle(ctx, unit.Collision{Time: 1.02, Other: foe})
	if bookOf(st, 1).snapshot().investigation[2] != 1 {
		t.Fatal("还没分开时再撞不应再记")
	}
	far := foe
	far.X = 400
	a.Handle(ctx, unit.Sense{Time: 1.1, Self: self, Nearby: []unit.Snapshot{far}})
	drain(out)
	a.Handle(ctx, unit.Collision{Time: 2, Other: foe})
	if got := bookOf(st, 1).snapshot().investigation[2]; got != 2 {
		t.Fatalf("分开后再撞应再记一次，实际 %d", got)
	}
}

func TestAssistantBounceStillInvestigates(t *testing.T) {
	st := &unit.MatchState{}
	ctx, _ := pipe(st, 1)
	a := &探长{ready: true, slot: 0, born: ctx, touching: map[contact]struct{}{}}
	self := unit.Snapshot{ID: 1, X: 0, Y: 0, Radius: 18, Role: unit.RoleFighter, Slot: 0}
	ally := unit.Snapshot{ID: 8, X: 0, Y: 0, Radius: 18, Role: unit.RoleFighter, Slot: 0, VX: 160}
	foe := unit.Snapshot{ID: 2, X: 18 + 18 + 4, Y: 0, Radius: 18, Role: unit.RoleFighter, Slot: 1, VX: -160}
	a.Handle(ctx, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{ally, foe}})
	if bookOf(st, 1).snapshot().investigation[2] != 1 {
		t.Fatal("助手弹开的这一拍仍应记调查")
	}
}

func TestFirstTrialCanConvictThenFire(t *testing.T) {
	st := &unit.MatchState{}
	ctx, out := pipe(st, 1)
	a := &探长{
		ready: true, slot: 0, boot: 0, firstTrial: false,
		touching: map[contact]struct{}{}, born: ctx,
		roll: func() float64 { return 0 },
	}
	self := unit.Snapshot{ID: 1, X: 0, Y: 0, Radius: 18, Role: unit.RoleFighter, Slot: 0}
	foe := unit.Snapshot{ID: 5, X: 80, Y: 0, Radius: 18, Role: unit.RoleFighter, Slot: 1}
	bookOf(st, 1).addSuspicion(5)
	a.Handle(ctx, unit.Sense{Time: 20, Self: self, Nearby: []unit.Snapshot{foe}})
	cmds := drain(out)
	if !hasStand(cmds, 1, true) || !hasStun(cmds, 5) || !hasFX(cmds, "judge") {
		t.Fatal("一审应站住、眩晕嫌疑人，并在场心喊 JUDGE")
	}
	a.Handle(ctx, unit.Sense{Time: 23, Self: self, Nearby: []unit.Snapshot{foe}})
	if !bookOf(st, 1).snapshot().guilty[5] {
		t.Fatal("检定成功应把嫌疑人标为有罪")
	}
	cmds = drain(out)
	if countKind(cmds, KindBullet) != 1 {
		t.Fatal("有罪后应开枪")
	}
	if !spawnDamage(cmds, judgmentDamage) {
		t.Fatal("有罪的子弹应是 11 点")
	}
	if !hasVerdict(cmds, verdictGuilty) {
		t.Fatal("有罪应在场心写出")
	}
}

func TestFailedTrialCanEnterForgery(t *testing.T) {
	st := &unit.MatchState{}
	ctx, out := pipe(st, 1)
	rolls := []float64{0.99, 0, 0.9}
	a := &探长{
		ready: true, slot: 0, boot: 0, enemies: 1,
		touching: map[contact]struct{}{}, born: ctx.WithBirths(),
		rng: unit.NewRNG(1),
		roll: func() float64 {
			n := rolls[0]
			rolls = rolls[1:]
			return n
		},
	}
	a.born.Out = ctx.Out
	self := unit.Snapshot{ID: 1, Radius: 18, Role: unit.RoleFighter, Slot: 0}
	foe := unit.Snapshot{ID: 5, X: 80, Radius: 18, Role: unit.RoleFighter, Slot: 1}
	bookOf(st, 1).addSuspicion(5)
	a.Handle(ctx, unit.Sense{Time: 20, Field: unit.HexField(), Self: self, Nearby: []unit.Snapshot{foe}})
	drain(out)
	a.Handle(ctx, unit.Sense{Time: 23, Field: unit.HexField(), Self: self, Nearby: []unit.Snapshot{foe}})
	if a.phase != phaseForgery {
		t.Fatalf("检定失败且分支掷到伪证，实际阶段 %d", a.phase)
	}
	cmds := drain(out)
	if countKind(cmds, KindForgery) != 5 {
		t.Fatal("伪证数量应等于敌方槽位 ×4 + 1")
	}
	if !hasVerdict(cmds, verdictClear) {
		t.Fatal("走进伪证时场心仍写无罪")
	}
	if a.secondTrialAt != 23+forgeryWait {
		t.Fatalf("伪证后应再等 20 秒进二审，实际 %v", a.secondTrialAt)
	}
}

func TestFailedTrialCanStayClear(t *testing.T) {
	st := &unit.MatchState{}
	ctx, out := pipe(st, 1)
	rolls := []float64{0.99, 0.99}
	a := &探长{
		ready: true, slot: 0, boot: 0,
		touching: map[contact]struct{}{}, born: ctx,
		roll: func() float64 {
			n := rolls[0]
			rolls = rolls[1:]
			return n
		},
	}
	self := unit.Snapshot{ID: 1, Radius: 18, Role: unit.RoleFighter, Slot: 0}
	foe := unit.Snapshot{ID: 5, X: 80, Radius: 18, Role: unit.RoleFighter, Slot: 1}
	bookOf(st, 1).addSuspicion(5)
	a.Handle(ctx, unit.Sense{Time: 20, Self: self, Nearby: []unit.Snapshot{foe}})
	drain(out)
	a.Handle(ctx, unit.Sense{Time: 23, Self: self, Nearby: []unit.Snapshot{foe}})
	if a.phase != phaseInvestigate {
		t.Fatalf("两次检定都失败应回到调查，实际 %d", a.phase)
	}
	cmds := drain(out)
	if !hasVerdict(cmds, verdictClear) {
		t.Fatal("未定罪应在场心写出无罪")
	}
	if countKind(cmds, KindForgery) != 0 {
		t.Fatal("无罪不该铺伪证")
	}
	if !a.bareRetrial || a.secondTrialAt != 23+forgeryWait {
		t.Fatalf("无罪后应再等 20 秒进二审，实际 %v bare %v", a.secondTrialAt, a.bareRetrial)
	}
}

func TestSecondAcquittalKills(t *testing.T) {
	st := &unit.MatchState{}
	ctx, out := pipe(st, 1)
	rolls := []float64{0.99, 0.99, 0.99}
	a := &探长{
		ready: true, slot: 0, boot: 0,
		touching: map[contact]struct{}{}, born: ctx,
		roll: func() float64 {
			n := rolls[0]
			rolls = rolls[1:]
			return n
		},
	}
	self := unit.Snapshot{ID: 1, Radius: 18, Role: unit.RoleFighter, Slot: 0}
	foe := unit.Snapshot{ID: 5, X: 80, Radius: 18, Role: unit.RoleFighter, Slot: 1}
	bookOf(st, 1).addSuspicion(5)
	a.Handle(ctx, unit.Sense{Time: 20, Self: self, Nearby: []unit.Snapshot{foe}})
	drain(out)
	a.Handle(ctx, unit.Sense{Time: 23, Self: self, Nearby: []unit.Snapshot{foe}})
	drain(out)
	a.Handle(ctx, unit.Sense{Time: 43, Self: self, Nearby: []unit.Snapshot{foe}})
	if !hasFX(drain(out), "judge") {
		t.Fatal("无罪后的二审也应在场心写出 JUDGE")
	}
	a.Handle(ctx, unit.Sense{Time: 46, Self: self, Nearby: []unit.Snapshot{foe}})
	cmds := drain(out)
	if !hasVerdict(cmds, verdictClear) {
		t.Fatal("二审未定罪应写出无罪")
	}
	dmg, ok := cmds[0].(unit.Damage)
	if !ok || dmg.To != 1 || dmg.From != 1 {
		found := false
		for _, c := range cmds {
			dmg, ok = c.(unit.Damage)
			if ok && dmg.To == 1 && dmg.From == 1 {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("再次无罪应把自己打死，实际 %#v", cmds)
		}
	}
}

func TestTrialImplicatesAccomplices(t *testing.T) {
	st := &unit.MatchState{}
	ctx, out := pipe(st, 1)
	a := &探长{
		ready: true, slot: 0, boot: 0, firstTrial: false,
		touching: map[contact]struct{}{}, born: ctx,
		roll: func() float64 { return 0 },
	}
	self := unit.Snapshot{ID: 1, X: 0, Y: 0, Radius: 18, Role: unit.RoleFighter, Slot: 0}
	lead := unit.Snapshot{ID: 5, X: 80, Y: 0, Radius: 18, Role: unit.RoleFighter, Slot: 1}
	hand := unit.Snapshot{ID: 6, X: -80, Y: 0, Radius: 18, Role: unit.RoleFighter, Slot: 1}
	clean := unit.Snapshot{ID: 7, X: 0, Y: 90, Radius: 18, Role: unit.RoleFighter, Slot: 1}
	bookOf(st, 1).addSuspicion(5)
	bookOf(st, 1).addSuspicion(5)
	bookOf(st, 1).addSuspicion(6)
	a.Handle(ctx, unit.Sense{Time: 20, Self: self, Nearby: []unit.Snapshot{lead, hand, clean}})
	if !hasMark(drain(out), 6, markAccomplice) {
		t.Fatal("有疑点又不是嫌疑人的应标成共犯")
	}
	a.Handle(ctx, unit.Sense{Time: 23, Self: self, Nearby: []unit.Snapshot{lead, hand, clean}})
	view := bookOf(st, 1).snapshot()
	if !view.guilty[5] || !view.guilty[6] {
		t.Fatalf("检定通过应连共犯一起有罪，实际 %+v", view.guilty)
	}
	if view.guilty[7] {
		t.Fatal("没有疑点的人不是共犯")
	}
}

func TestBumpGuiltStaysPersonal(t *testing.T) {
	b := newBook()
	b.addSuspicion(9)
	for i := 0; i < investigateNeed; i++ {
		b.addInvestigation(4)
	}
	view := b.snapshot()
	if !view.guilty[4] || view.guilty[9] {
		t.Fatalf("调查满 4 次只定被撞的人，实际 %+v", view.guilty)
	}
}

func TestAdmonitionCanCondemnSelf(t *testing.T) {
	st := &unit.MatchState{}
	ctx, out := pipe(st, 1)
	a := &探长{ready: true, born: ctx, roll: func() float64 { return 0 }}
	bookOf(st, 1).noteAdmonition()
	a.rollAdmonitions(ctx)
	cmds := drain(out)
	dmg, ok := cmds[0].(unit.Damage)
	if !ok || dmg.To != 1 || dmg.From != 1 {
		t.Fatalf("申诫掷中应把自己打死，实际 %#v", cmds)
	}
}

func TestBulletCarriesItsDamage(t *testing.T) {
	for _, want := range []float64{judgmentDamage, obsessionDamage} {
		b := &子弹{owner: 1, slot: 0, damage: want}
		ctx, out := pipe(&unit.MatchState{}, 9)
		b.Handle(ctx, unit.Collision{Other: unit.Snapshot{ID: 2, Slot: 1, Role: unit.RoleFighter, Radius: 18}})
		cmds := drain(out)
		dmg, ok := cmds[0].(unit.Damage)
		if !ok || dmg.Amount != want || dmg.To != 2 {
			t.Fatalf("子弹应造成 %v 点，实际 %#v", want, cmds)
		}
	}
}

func TestObsessionBulletIsTwenty(t *testing.T) {
	st := &unit.MatchState{}
	ctx, out := pipe(st, 1)
	a := &探长{
		ready: true, slot: 0, phase: phaseObsession, shotsLeft: 1, nextShot: 0,
		born: ctx, roll: func() float64 { return 1 },
	}
	self := unit.Snapshot{ID: 1, X: 0, Y: 0, Radius: 18, Role: unit.RoleFighter, Slot: 0}
	foe := unit.Snapshot{ID: 5, X: 80, Y: 0, Radius: 18, Role: unit.RoleFighter, Slot: 1}
	a.Handle(ctx, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{foe}})
	if !spawnDamage(drain(out), obsessionDamage) {
		t.Fatal("我执的子弹应是 20 点")
	}
}

func pipe(st *unit.MatchState, id uint64) (unit.Context, chan unit.Cmd) {
	out := make(chan unit.Cmd, 128)
	return unit.Context{State: st, ID: id, Kind: KindDetective, Out: out, Stream: 3}, out
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

func countKind(cmds []unit.Cmd, kind string) int {
	n := 0
	for _, c := range cmds {
		if sp, ok := c.(unit.Spawn); ok && sp.Kind == kind {
			n++
		}
	}
	return n
}

func spawnDamage(cmds []unit.Cmd, dmg float64) bool {
	for _, c := range cmds {
		sp, ok := c.(unit.Spawn)
		if ok && sp.Kind == KindBullet && sp.Data == dmg {
			return true
		}
	}
	return false
}

func hasStand(cmds []unit.Cmd, id uint64, hold bool) bool {
	for _, c := range cmds {
		if s, ok := c.(unit.Stand); ok && s.UnitID == id && s.Hold == hold {
			return true
		}
	}
	return false
}

func hasStun(cmds []unit.Cmd, id uint64) bool {
	for _, c := range cmds {
		if s, ok := c.(unit.Stun); ok && s.UnitID == id && s.Hold {
			return true
		}
	}
	return false
}

func hasFX(cmds []unit.Cmd, name string) bool {
	for _, c := range cmds {
		if f, ok := c.(unit.FX); ok && f.Name == name && f.X == 0 && f.Y == 0 {
			return true
		}
	}
	return false
}

func hasVerdict(cmds []unit.Cmd, amount float64) bool {
	for _, c := range cmds {
		if f, ok := c.(unit.FX); ok && f.Name == "verdict" && f.Amount == amount && f.X == 0 && f.Y == 0 {
			return true
		}
	}
	return false
}

func hasMark(cmds []unit.Cmd, id uint64, kind string) bool {
	for _, c := range cmds {
		m, ok := c.(unit.StackMark)
		if ok && m.UnitID == id && m.Kind == kind && m.Delta > 0 {
			return true
		}
	}
	return false
}
