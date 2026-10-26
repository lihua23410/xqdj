package 大肥鱼

import (
	"math"
	"math/rand/v2"
	"testing"
	"xqdj/internal/unit"
	_ "xqdj/map/hex"
)

func TestRiceSpawnsAtRandomWalkable(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &大肥鱼{rng: rand.New(rand.NewPCG(1, 2))}
	a.Handle(unit.Context{ID: 1, Kind: KindOrca, Out: out, Stream: 1}, unit.Sense{
		Time:  0,
		Self:  selfAt(0, 0),
		Field: unit.HexField(),
	})
	cmds := drain(out)
	var bowl *unit.Spawn
	for i := range cmds {
		s, ok := cmds[i].(unit.Spawn)
		if ok && s.Kind == KindRice {
			bowl = &s
		}
		if _, ok := cmds[i].(unit.SetVelocity); ok {
			t.Fatal("原身不该朝白饭走")
		}
	}
	if bowl == nil {
		t.Fatal("场上该随机刷一碗白饭")
	}
	if !unit.HexField().Walkable(bowl.X, bowl.Y, bowlRadius) {
		t.Fatalf("碗不在可进入区域 (%v,%v)", bowl.X, bowl.Y)
	}
	if math.Hypot(bowl.X, bowl.Y) < maidRadius+bowlRadius+bowlClear-1e-6 {
		t.Fatalf("碗不该刷在她身上 (%v,%v)", bowl.X, bowl.Y)
	}
	if bowl.OwnerID != 1 {
		t.Fatalf("owner=%d", bowl.OwnerID)
	}
}

func TestRiceEveryThreeSecondsUpToThree(t *testing.T) {
	field := unit.HexField()
	held := unit.Snapshot{ID: 9, Kind: KindRice, OwnerID: 1, X: 80, Y: 0, Radius: bowlRadius, HP: 1}
	a := &大肥鱼{rng: rand.New(rand.NewPCG(1, 2)), nextRice: 3}
	out := make(chan unit.Cmd, 8)
	a.Handle(ctx(out), unit.Sense{
		Time: 1, Self: selfAt(0, 0), Nearby: []unit.Snapshot{held}, Field: field,
	})
	if spawned(drain(out), KindRice) {
		t.Fatal("没到 3 秒不该再刷")
	}

	out = make(chan unit.Cmd, 8)
	a.Handle(ctx(out), unit.Sense{
		Time: 3, Self: selfAt(0, 0), Nearby: []unit.Snapshot{held}, Field: field,
	})
	if !spawned(drain(out), KindRice) {
		t.Fatal("满 3 秒且不足 5 碗该再刷一碗")
	}

	full := []unit.Snapshot{
		{ID: 9, Kind: KindRice, OwnerID: 1, X: 80, Y: 0, Radius: bowlRadius, HP: 1},
		{ID: 10, Kind: KindRice, OwnerID: 1, X: -80, Y: 0, Radius: bowlRadius, HP: 1},
		{ID: 11, Kind: KindRice, OwnerID: 1, X: 0, Y: 80, Radius: bowlRadius, HP: 1},
		{ID: 12, Kind: KindRice, OwnerID: 1, X: 0, Y: -80, Radius: bowlRadius, HP: 1},
		{ID: 13, Kind: KindRice, OwnerID: 1, X: 60, Y: 60, Radius: bowlRadius, HP: 1},
	}
	a.nextRice = 0
	out = make(chan unit.Cmd, 8)
	a.Handle(ctx(out), unit.Sense{
		Time: 10, Self: selfAt(0, 0), Nearby: full, Field: field,
	})
	if spawned(drain(out), KindRice) {
		t.Fatal("已经 5 碗不该再刷")
	}
	if math.Abs(a.nextRice-(10+riceEvery)) > 1e-9 {
		t.Fatalf("满员时该把下一碗推到 3 秒后，next=%v", a.nextRice)
	}
}

func TestWhaleIgnoresRice(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &大肥鱼{form: true, until: 20, phase: phaseDive, phaseUntil: 50, nextRice: 100}
	bowl := unit.Snapshot{ID: 9, Kind: KindRice, OwnerID: 1, X: 10, Y: 0, Radius: bowlRadius, HP: 1}
	a.Handle(ctx(out), unit.Sense{
		Time:   2,
		Self:   unit.Snapshot{ID: 1, Slot: 0, X: 0, Y: 0, Radius: orcaRadius, HP: maidHP},
		Nearby: []unit.Snapshot{bowl},
		Field:  unit.HexField(),
	})
	cmds := drain(out)
	if despawned(cmds, 9) {
		t.Fatal("虎鲸不吃白饭")
	}
	if a.until != 20 {
		t.Fatal("重叠白饭不该重开虎鲸")
	}
}

func TestWhaleHalvesDamage(t *testing.T) {
	out := make(chan unit.Cmd, 4)
	a := &大肥鱼{seen: true, hp: 40, form: true}
	a.Handle(ctx(out), unit.IncomingDamage{Token: 7, Amount: 10, Time: 1})
	cmds := drain(out)
	ok := false
	for _, c := range cmds {
		cf, yes := c.(unit.ConfirmDamage)
		if yes && cf.Amount == 5 {
			ok = true
		}
	}
	if !ok {
		t.Fatalf("确认伤害=%v 该是 5", cmds)
	}
	if a.hp != 35 {
		t.Fatalf("hp=%v", a.hp)
	}
}

func TestEatOwnBowlStartsDive(t *testing.T) {
	bowl := unit.Snapshot{ID: 9, Kind: KindRice, OwnerID: 1, X: 20, Y: 0, Radius: bowlRadius, HP: 1}
	field := unit.HexField()
	a := &大肥鱼{}
	out := make(chan unit.Cmd, 16)
	a.Handle(ctx(out), unit.Sense{
		Time: 1, Self: selfAt(0, 0), Nearby: []unit.Snapshot{bowl}, Field: field,
	})
	cmds := drain(out)
	if hasRadius(cmds, orcaRadius) || despawned(cmds, 9) {
		t.Fatal("刚碰上该开始吃，不该立刻变身")
	}
	if !a.eating {
		t.Fatal("该在吃饭")
	}
	first := teleportOf(cmds, 1)
	if first == nil {
		t.Fatal("吃饭该坐到碗边")
	}
	if math.Hypot(first.X-bowl.X, first.Y-bowl.Y) > chewGap+chewAmp+3 {
		t.Fatalf("离碗太远 (%v,%v)", first.X, first.Y)
	}

	out = make(chan unit.Cmd, 16)
	a.Handle(ctx(out), unit.Sense{
		Time: 1.4, Self: selfAt(first.X, first.Y), Nearby: []unit.Snapshot{bowl}, Field: field,
	})
	cmds = drain(out)
	second := teleportOf(cmds, 1)
	if second == nil || (second.X == first.X && second.Y == first.Y) {
		t.Fatalf("吃饭该小幅抖动 first=%v second=%v", first, second)
	}
	if despawned(cmds, 9) {
		t.Fatal("1.5 秒没到，碗还在")
	}

	out = make(chan unit.Cmd, 16)
	a.Handle(ctx(out), unit.Sense{
		Time: 1 + eatSec, Self: selfAt(second.X, second.Y), Nearby: []unit.Snapshot{bowl}, Field: field,
	})
	cmds = drain(out)
	if !hasRadius(cmds, orcaRadius) {
		t.Fatal("吃完该变成半径 34")
	}
	if !despawned(cmds, 9) {
		t.Fatal("吃完碗该消失")
	}
	if a.phase != phaseDive {
		t.Fatalf("phase=%d 该潜水", a.phase)
	}
	if !passHeld(cmds, true) {
		t.Fatal("潜水该穿过去")
	}
}

func TestIgnoreOtherBowl(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &大肥鱼{rng: rand.New(rand.NewPCG(3, 4))}
	other := unit.Snapshot{ID: 9, Kind: KindRice, OwnerID: 2, X: 10, Y: 0, Radius: bowlRadius, HP: 1}
	a.Handle(ctx(out), unit.Sense{
		Time:   0,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{other},
		Field:  unit.HexField(),
	})
	cmds := drain(out)
	if hasRadius(cmds, orcaRadius) {
		t.Fatal("别人的白饭不能吃")
	}
	if despawned(cmds, 9) {
		t.Fatal("不该收掉别人的碗")
	}
}

func TestBodyTouchDealsNothing(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := &大肥鱼{form: true, until: 10, phase: phaseLeap}
	a.Handle(ctx(out), unit.Collision{
		Time:  1,
		Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, HP: 100},
	})
	if len(damages(drain(out))) != 0 {
		t.Fatal("贴上不出伤，水花才出伤")
	}
}

func TestLeapLandsWhereEnemyWas(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &大肥鱼{form: true, until: 20, phase: phaseDive, phaseUntil: 1}
	enemy := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 80, Y: 0, HP: 100, Radius: 18}
	a.Handle(ctx(out), unit.Sense{
		Time:   1,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{enemy},
		Field:  unit.HexField(),
	})
	if named(drain(out), "leap") == nil {
		t.Fatal("潜水结束该跃起")
	}
	if math.Abs(a.aimX-80) > 1e-6 || a.aimY != 0 {
		t.Fatalf("落点=%v,%v 该是跃起时的位置", a.aimX, a.aimY)
	}

	out = make(chan unit.Cmd, 16)
	enemy.X = 220
	a.Handle(ctx(out), unit.Sense{
		Time:   1 + leapSec,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{enemy},
		Field:  unit.HexField(),
	})
	cmds := drain(out)
	tp := teleportOf(cmds, 1)
	if tp == nil || math.Abs(tp.X-80) > 1e-6 || tp.Y != 0 {
		t.Fatalf("入水=%v 不该跟到人的新位置", tp)
	}
	if len(damages(cmds)) != 0 {
		t.Fatal("人已经离开落点，水花不该打到")
	}
	if named(cmds, "splash") == nil {
		t.Fatal("入水该溅水花")
	}
}

func TestLeapStopsAtReach(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &大肥鱼{form: true, until: 20, phase: phaseDive, phaseUntil: 1}
	enemy := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 240, Y: 0, HP: 100, Radius: 18}
	a.Handle(ctx(out), unit.Sense{
		Time:   1,
		Self:   selfAt(-200, 0),
		Nearby: []unit.Snapshot{enemy},
		Field:  unit.HexField(),
	})
	drain(out)
	want := -200 + leapReach
	if math.Abs(a.aimX-want) > 1e-6 || a.aimY != 0 {
		t.Fatalf("落点=%v,%v 该停在最远 %v，不该钉在敌人身上", a.aimX, a.aimY, want)
	}

	a.Handle(ctx(out), unit.Sense{
		Time:   1 + leapSec,
		Self:   selfAt(-200, 0),
		Nearby: []unit.Snapshot{enemy},
		Field:  unit.HexField(),
	})
	cmds := drain(out)
	tp := teleportOf(cmds, 1)
	if tp == nil || math.Abs(tp.X-want) > 1e-6 || tp.Y != 0 {
		t.Fatalf("入水=%v 不该闪到 %v", tp, enemy.X)
	}
	if len(damages(cmds)) != 0 {
		t.Fatal("够不着的敌人不该吃到水花")
	}
}

func TestSplashHitsWhoeverIsThere(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &大肥鱼{
		form: true, until: 20, phase: phaseLeap, phaseUntil: 5,
		haveAim: true, aimX: 80, aimY: 0, fromX: 0, fromY: 0,
	}
	enemy := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 80, Y: 0, HP: 100, Radius: 18}
	a.Handle(ctx(out), unit.Sense{
		Time:   5,
		Self:   selfAt(0, 0),
		Nearby: []unit.Snapshot{enemy},
		Field:  unit.HexField(),
	})
	hits := damages(drain(out))
	if len(hits) != 1 || hits[0].Amount != splashDamage || hits[0].To != 2 {
		t.Fatalf("水花=%v", hits)
	}
}

func TestFormEndsAtSix(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &大肥鱼{form: true, until: 6, phase: phaseDive}
	a.Handle(ctx(out), unit.Sense{
		Time:  6,
		Self:  unit.Snapshot{ID: 1, Slot: 0, X: 0, Y: 0, Radius: orcaRadius, HP: maidHP},
		Field: unit.HexField(),
	})
	cmds := drain(out)
	if a.form {
		t.Fatal("6 秒该回到原身")
	}
	if !hasRadius(cmds, maidRadius) {
		t.Fatal("半径该收回 18")
	}
	if !hasVision(cmds, maidVision) {
		t.Fatal("视野该收回 0")
	}
	if len(damages(cmds)) != 0 {
		t.Fatal("回到原身不该再溅水花")
	}
}

func TestGrowPushesOutOfWall(t *testing.T) {
	a := &大肥鱼{}
	field := unit.HexField()
	field.Walls = []unit.FieldWall{{
		Kind: unit.WallHard,
		X1:   -80, Y1: 0, X2: 80, Y2: 0,
		Radius: 50,
	}}
	bowl := unit.Snapshot{ID: 9, Kind: KindRice, OwnerID: 1, X: 16, Y: 0, Radius: bowlRadius, HP: 1}
	out := make(chan unit.Cmd, 16)
	a.Handle(ctx(out), unit.Sense{
		Time: 0, Self: selfAt(0, 0), Nearby: []unit.Snapshot{bowl}, Field: field,
	})
	drain(out)
	out = make(chan unit.Cmd, 16)
	a.Handle(ctx(out), unit.Sense{
		Time: eatSec, Self: selfAt(0, 0), Nearby: []unit.Snapshot{bowl}, Field: field,
	})
	tp := teleportOf(drain(out), 1)
	if tp == nil {
		t.Fatal("变大插进墙该先推出去")
	}
	if !field.Walkable(tp.X, tp.Y, orcaRadius) {
		t.Fatalf("推出后仍不可站 (%v,%v)", tp.X, tp.Y)
	}
}

func TestDeathClearsOwned(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := &大肥鱼{seen: true, hp: 12, form: true}
	a.Handle(ctx(out), unit.IncomingDamage{Token: 3, Amount: 24, Time: 1})
	cmds := drain(out)
	found := false
	for _, c := range cmds {
		d, ok := c.(unit.DespawnOwned)
		if ok && d.OwnerID == 1 && d.Kind == "" {
			found = true
		}
	}
	if !found {
		t.Fatal("倒下该清掉自己的白饭")
	}
}

func TestRiceSkinSticks(t *testing.T) {
	out := make(chan unit.Cmd, 4)
	b := &白饭{}
	c := unit.Context{ID: 9, Kind: KindRice, Out: out, Stream: 11}
	sense := unit.Sense{Self: unit.Snapshot{ID: 9, Slot: 0}}
	b.Handle(c, sense)
	first := named(drain(out), "rice")
	b.Handle(c, sense)
	second := named(drain(out), "rice")
	if first == nil || second == nil {
		t.Fatal("白饭该报贴图")
	}
	if first.Amount != second.Amount || first.Amount < 0 || first.Amount > riceKinds-1 {
		t.Fatalf("贴图=%v / %v", first.Amount, second.Amount)
	}
}

func TestNoCallBeforeSevenSeconds(t *testing.T) {
	a := &大肥鱼{nextRice: 100}
	field := unit.HexField()
	bowl := unit.Snapshot{ID: 9, Kind: KindRice, OwnerID: 1, X: 200, Y: 0, Radius: bowlRadius, HP: 1}
	out := make(chan unit.Cmd, 32)
	a.Handle(ctx(out), unit.Sense{Time: 0, Self: selfAt(0, 0), Field: field})
	drain(out)
	a.Handle(ctx(out), unit.Sense{
		Time: 6.9, Self: selfAt(0, 0), Nearby: []unit.Snapshot{bowl}, Field: field,
	})
	cmds := drain(out)
	if named(cmds, "call") != nil {
		t.Fatal("未满 7 秒不该喊饭")
	}
	if teleportOf(cmds, 9) != nil {
		t.Fatal("未满 7 秒碗不该动")
	}
}

func TestCallPinsAndPullsNearestBowl(t *testing.T) {
	a := &大肥鱼{nextRice: 100}
	field := unit.HexField()
	near := unit.Snapshot{ID: 9, Kind: KindRice, OwnerID: 1, X: 200, Y: 0, Radius: bowlRadius, HP: 1}
	far := unit.Snapshot{ID: 10, Kind: KindRice, OwnerID: 1, X: 400, Y: 0, Radius: bowlRadius, HP: 1}
	other := unit.Snapshot{ID: 11, Kind: KindRice, OwnerID: 2, X: 40, Y: 0, Radius: bowlRadius, HP: 1}
	bowls := []unit.Snapshot{near, far, other}
	out := make(chan unit.Cmd, 32)
	a.Handle(ctx(out), unit.Sense{Time: 0, Self: selfAt(0, 0), Field: field})
	drain(out)

	a.Handle(ctx(out), unit.Sense{Time: 7, Self: selfAt(0, 0), Nearby: bowls, Field: field})
	cmds := drain(out)
	if named(cmds, "call") == nil {
		t.Fatal("满 7 秒该喊饭")
	}
	if !standHeld(cmds, true) {
		t.Fatal("喊饭该站住")
	}
	if teleportOf(cmds, 9) != nil {
		t.Fatal("开口这一拍碗还在原点")
	}

	a.Handle(ctx(out), unit.Sense{Time: 7.1, Self: selfAt(15, 0), Nearby: bowls, Field: field})
	cmds = drain(out)
	self := teleportOf(cmds, 1)
	if self == nil || math.Abs(self.X) > 1e-6 || math.Abs(self.Y) > 1e-6 {
		t.Fatalf("该拉回开口那一点，got %v", self)
	}
	if !velZero(cmds, 1) {
		t.Fatal("喊饭速度该是 0")
	}

	a.Handle(ctx(out), unit.Sense{Time: 7.2, Self: selfAt(0, 0), Nearby: bowls, Field: field})
	cmds = drain(out)
	got := teleportOf(cmds, 9)
	if got == nil || math.Abs(got.X-170) > 1e-6 || math.Abs(got.Y) > 1e-6 {
		t.Fatalf("0.2 秒该走出 30，got %v", got)
	}
	if teleportOf(cmds, 10) != nil || teleportOf(cmds, 11) != nil {
		t.Fatal("更远的碗和别人的碗不该动")
	}

	a.Handle(ctx(out), unit.Sense{Time: 7.4, Self: selfAt(0, 0), Nearby: bowls, Field: field})
	cmds = drain(out)
	got = teleportOf(cmds, 9)
	if got == nil || math.Abs(got.X-140) > 1e-6 || math.Abs(got.Y) > 1e-6 {
		t.Fatalf("0.4 秒该走出 60，got %v", got)
	}
}

func TestCallRushStartsMealWithoutHeal(t *testing.T) {
	a := &大肥鱼{nextRice: 100}
	field := unit.HexField()
	bowl := unit.Snapshot{ID: 9, Kind: KindRice, OwnerID: 1, X: 50, Y: 0, Radius: bowlRadius, HP: 1}
	out := make(chan unit.Cmd, 32)
	a.Handle(ctx(out), unit.Sense{Time: 0, Self: selfAt(0, 0), Field: field})
	drain(out)
	a.Handle(ctx(out), unit.Sense{
		Time: 7, Self: selfAt(0, 0), Nearby: []unit.Snapshot{bowl}, Field: field,
	})
	drain(out)
	a.Handle(ctx(out), unit.Sense{
		Time: 7.2, Self: selfAt(0, 0), Nearby: []unit.Snapshot{bowl}, Field: field,
	})
	cmds := drain(out)
	if !a.eating {
		t.Fatal("冲到重叠该开始吃白饭")
	}
	if _, ok := healOf(cmds); ok {
		t.Fatal("刚吃上不该回血")
	}
	landed := teleportOf(cmds, 9)
	if landed == nil || math.Abs(landed.X-(maidRadius+bowlRadius)) > 1e-6 {
		t.Fatalf("该停在接触点，got %v", landed)
	}

	bowl.X = landed.X
	a.Handle(ctx(out), unit.Sense{
		Time: 7.2 + eatSec, Self: selfAt(0, 0), Nearby: []unit.Snapshot{bowl}, Field: field,
	})
	cmds = drain(out)
	h, ok := healOf(cmds)
	if !ok || h.Amount != mealHeal || h.UnitID != 1 {
		t.Fatalf("吃完该回 %v，got %v %v", mealHeal, ok, h)
	}
}

func TestWhaleDoesNotCallUntilSevenAfterLeave(t *testing.T) {
	field := unit.HexField()
	bowl := unit.Snapshot{ID: 9, Kind: KindRice, OwnerID: 1, X: 80, Y: 0, Radius: bowlRadius, HP: 1}
	a := &大肥鱼{form: true, until: 100, phase: phaseSurface, phaseUntil: 100, nextRice: 100}
	out := make(chan unit.Cmd, 32)
	a.Handle(ctx(out), unit.Sense{
		Time: 30, Self: selfAt(0, 0), Nearby: []unit.Snapshot{bowl}, Field: field,
	})
	cmds := drain(out)
	if named(cmds, "call") != nil || teleportOf(cmds, 9) != nil {
		t.Fatal("虎鲸不该喊饭")
	}

	a = &大肥鱼{form: true, until: 6, phase: phaseDive, nextRice: 100}
	a.Handle(ctx(out), unit.Sense{Time: 6, Self: selfAt(0, 0), Field: field})
	drain(out)
	a.Handle(ctx(out), unit.Sense{Time: 12.9, Self: selfAt(0, 0), Field: field})
	if named(drain(out), "call") != nil {
		t.Fatal("回到原身后未满 7 秒不该喊")
	}
	a.Handle(ctx(out), unit.Sense{Time: 13, Self: selfAt(0, 0), Field: field})
	if named(drain(out), "call") == nil {
		t.Fatal("回到原身再满 7 秒该喊")
	}
}

func TestCallWithoutBowlStillSpeaks(t *testing.T) {
	a := &大肥鱼{nextRice: 100}
	out := make(chan unit.Cmd, 32)
	a.Handle(ctx(out), unit.Sense{Time: 0, Self: selfAt(0, 0), Field: unit.HexField()})
	drain(out)
	a.Handle(ctx(out), unit.Sense{Time: 7, Self: selfAt(0, 0), Field: unit.HexField()})
	cmds := drain(out)
	if named(cmds, "call") == nil {
		t.Fatal("没有碗也该喊饭")
	}
	for _, c := range cmds {
		tp, ok := c.(unit.Teleport)
		if ok && tp.UnitID != 1 {
			t.Fatalf("没有碗不该传送别人 %v", tp)
		}
	}
}

func ctx(out chan unit.Cmd) unit.Context {
	return unit.Context{ID: 1, Kind: KindOrca, Out: out, Stream: 1}.WithBirths()
}

func selfAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{ID: 1, Slot: 0, X: x, Y: y, Radius: maidRadius, HP: maidHP, MaxHP: maidHP}
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

func hasRadius(cmds []unit.Cmd, r float64) bool {
	for _, c := range cmds {
		s, ok := c.(unit.SetRadius)
		if ok && s.Radius == r {
			return true
		}
	}
	return false
}

func hasVision(cmds []unit.Cmd, v float64) bool {
	for _, c := range cmds {
		s, ok := c.(unit.SetVision)
		if ok && s.Vision == v {
			return true
		}
	}
	return false
}

func despawned(cmds []unit.Cmd, id uint64) bool {
	for _, c := range cmds {
		d, ok := c.(unit.Despawn)
		if ok && d.UnitID == id {
			return true
		}
	}
	return false
}

func spawned(cmds []unit.Cmd, kind string) bool {
	for _, c := range cmds {
		s, ok := c.(unit.Spawn)
		if ok && s.Kind == kind {
			return true
		}
	}
	return false
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

func teleportOf(cmds []unit.Cmd, id uint64) *unit.Teleport {
	for i := range cmds {
		tp, ok := cmds[i].(unit.Teleport)
		if ok && tp.UnitID == id {
			return &tp
		}
	}
	return nil
}

func healOf(cmds []unit.Cmd) (unit.Heal, bool) {
	for _, c := range cmds {
		h, ok := c.(unit.Heal)
		if ok {
			return h, true
		}
	}
	return unit.Heal{}, false
}

func standHeld(cmds []unit.Cmd, hold bool) bool {
	for _, c := range cmds {
		st, ok := c.(unit.Stand)
		if ok && st.UnitID == 1 && st.Hold == hold {
			return true
		}
	}
	return false
}

func velZero(cmds []unit.Cmd, id uint64) bool {
	for _, c := range cmds {
		v, ok := c.(unit.SetVelocity)
		if ok && v.UnitID == id && v.VX == 0 && v.VY == 0 {
			return true
		}
	}
	return false
}

func passHeld(cmds []unit.Cmd, hold bool) bool {
	for _, c := range cmds {
		p, ok := c.(unit.Pass)
		if ok && p.Hold == hold {
			return true
		}
	}
	return false
}

func named(cmds []unit.Cmd, name string) *unit.FX {
	for i := range cmds {
		fx, ok := cmds[i].(unit.FX)
		if ok && fx.Name == name {
			return &fx
		}
	}
	return nil
}
