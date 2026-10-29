package 传票员

import (
	"math"
	"testing"

	"xqdj/internal/unit"

	// 场地外形判定由 map 包在 init 里登记；这里要用 Field.Walkable / Clamp，所以得拉进来。
	_ "xqdj/map/hex"
)

func drain(out chan unit.Cmd) []unit.Cmd {
	var got []unit.Cmd
	for {
		select {
		case c := <-out:
			got = append(got, c)
		default:
			return got
		}
	}
}

func pipe(id uint64) (unit.Context, chan unit.Cmd) {
	out := make(chan unit.Cmd, 64)
	return unit.Context{ID: id, Kind: KindSubpoena, Out: out, Stream: 7}.WithBirths(), out
}

func selfAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindSubpoena, Role: unit.RoleFighter, Slot: 0,
		X: x, Y: y, Radius: bodyRadius, HP: bodyHP, MaxHP: bodyHP,
		Vision: bodyVision, AimPriority: 15,
	}
}

func foeAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 2, Kind: "靶子", Role: unit.RoleFighter, Slot: 1,
		X: x, Y: y, Radius: 18, HP: 100, MaxHP: 100, AimPriority: 15,
	}
}

func senseAt(t float64) unit.Sense {
	return unit.Sense{Time: t, Self: selfAt(0, 0), Nearby: []unit.Snapshot{foeAt(200, 0)}}
}

func hitAt(t float64, from uint64, amount float64) unit.IncomingDamage {
	return unit.IncomingDamage{Token: uint64(t*1000) + 1, From: from, Amount: amount, Time: t}
}

func ticketsIn(cmds []unit.Cmd) []unit.Spawn {
	var out []unit.Spawn
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == KindTicket {
			out = append(out, s)
		}
	}
	return out
}

func standIn(cmds []unit.Cmd) (unit.Stand, bool) {
	for _, c := range cmds {
		if s, ok := c.(unit.Stand); ok {
			return s, true
		}
	}
	return unit.Stand{}, false
}

func fxIn(cmds []unit.Cmd, name string) (unit.FX, bool) {
	for _, c := range cmds {
		if f, ok := c.(unit.FX); ok && f.Name == name {
			return f, true
		}
	}
	return unit.FX{}, false
}

func confirmedIn(cmds []unit.Cmd) (unit.ConfirmDamage, bool) {
	for _, c := range cmds {
		if d, ok := c.(unit.ConfirmDamage); ok {
			return d, true
		}
	}
	return unit.ConfirmDamage{}, false
}

func rolled(r float64) *传票员 {
	a := newSubpoena(7)
	a.roll = func() float64 { return r }
	return a
}

// 传票员永远确认这一笔：票面要记的是真掉的血。
func TestHitConfirmsAndStandsWithoutFiringYet(t *testing.T) {
	a := rolled(0.1) // 正常那张，才有「站住写票」
	ctx, out := pipe(1)
	a.Handle(ctx, hitAt(0, 2, 12))
	cmds := drain(out)

	d, ok := confirmedIn(cmds)
	if !ok || math.Abs(d.Amount-12) > 1e-9 || d.UnitID != 1 {
		t.Fatalf("confirm=%+v ok=%v", d, ok)
	}
	st, ok := standIn(cmds)
	if !ok || !st.Hold {
		t.Fatalf("stand=%+v ok=%v", st, ok)
	}
	if len(ticketsIn(cmds)) != 0 {
		t.Fatal("票要等站完 0.8 秒才出")
	}
}

func TestFiresOneTicketAtSourceAfterStand(t *testing.T) {
	a := newSubpoena(7)
	a.roll = func() float64 { return 0.1 } // 正常
	ctx, out := pipe(1)
	a.Handle(ctx, hitAt(0, 2, 12))
	drain(out)

	a.Handle(ctx, senseAt(0.7))
	if n := len(ticketsIn(drain(out))); n != 0 {
		t.Fatalf("0.7 秒还没站完，出了 %d 张", n)
	}

	a.Handle(ctx, senseAt(0.8))
	cmds := drain(out)
	sp := ticketsIn(cmds)
	if len(sp) != 1 {
		t.Fatalf("tickets=%d", len(sp))
	}
	if math.Abs(sp[0].VX-ticketSpeed) > 1e-6 || math.Abs(sp[0].VY) > 1e-6 {
		t.Fatalf("该直着朝右飞：v=(%v,%v)", sp[0].VX, sp[0].VY)
	}
	if math.Abs(sp[0].X-(bodyRadius+ticketRadius+ticketGap)) > 1e-6 {
		t.Fatalf("出票点 x=%v", sp[0].X)
	}
	if sp[0].OwnerID != 1 || sp[0].Slot != 0 {
		t.Fatalf("spawn owner/slot=%d/%d", sp[0].OwnerID, sp[0].Slot)
	}
	if st, ok := standIn(cmds); !ok || st.Hold {
		t.Fatalf("站完要放下 Stand：%+v ok=%v", st, ok)
	}
}

func TestCooldownSwallowsHitsWithinOneSecond(t *testing.T) {
	a := newSubpoena(7)
	a.roll = func() float64 { return 0.1 }
	ctx, out := pipe(1)
	a.Handle(ctx, hitAt(0, 2, 5))
	drain(out)

	a.Handle(ctx, hitAt(0.4, 2, 5))
	cmds := drain(out)
	if len(a.shots) != 1 {
		t.Fatalf("CD 内不该再开票：shots=%d", len(a.shots))
	}
	if _, ok := standIn(cmds); ok {
		t.Fatal("CD 内不该再站")
	}

	a.Handle(ctx, senseAt(0.8))
	if n := len(ticketsIn(drain(out))); n != 1 {
		t.Fatalf("只该出第一笔的一张：%d", n)
	}
}

func TestSecondTriggerAfterCooldown(t *testing.T) {
	a := newSubpoena(7)
	a.roll = func() float64 { return 0.1 }
	ctx, out := pipe(1)
	a.Handle(ctx, hitAt(0, 2, 5))
	a.Handle(ctx, senseAt(0.8))
	drain(out)

	a.Handle(ctx, hitAt(0.85, 2, 7))
	cmds := drain(out)
	if st, ok := standIn(cmds); !ok || !st.Hold {
		t.Fatalf("CD 过了要再站：%+v ok=%v", st, ok)
	}
	if a.shots[len(a.shots)-1].dmg != 7 {
		t.Fatalf("第二张票面=%v", a.shots[len(a.shots)-1].dmg)
	}

	a.Handle(ctx, senseAt(1.65))
	if n := len(ticketsIn(drain(out))); n != 1 {
		t.Fatalf("第二笔该出一张：%d", n)
	}
}

func TestMisfireNoneScreamsWithoutStandingAndHalvesCooldown(t *testing.T) {
	a := rolled(0.75) // n=1：0.70–0.80 是空窗
	ctx, out := pipe(1)
	a.Handle(ctx, hitAt(0, 2, 9))
	cmds := drain(out)
	if st, ok := standIn(cmds); ok && st.Hold {
		t.Fatal("空窗没有票要写，不该站")
	}

	a.Handle(ctx, senseAt(0)) // 下一帧就把故障喊出来
	cmds = drain(out)
	if n := len(ticketsIn(cmds)); n != 0 {
		t.Fatalf("故障不该出票：%d", n)
	}
	if _, ok := fxIn(cmds, "fault"); !ok {
		t.Fatal("故障该喊一声")
	}
	// CD 减半：0.4 秒就能再触发（正常是 0.8）。
	if math.Abs(a.ready-emptyCD) > 1e-9 {
		t.Fatalf("ready=%v", a.ready)
	}
	a.Handle(ctx, hitAt(0.45, 2, 9))
	if len(a.shots) != 1 {
		t.Fatalf("0.45 秒该能再触发：shots=%d", len(a.shots))
	}
}

func TestMisfireTwoSendsPairAroundAim(t *testing.T) {
	a := rolled(0.85) // n=1：0.80–0.90 是两张
	ctx, out := pipe(1)
	a.Handle(ctx, hitAt(0, 2, 9))
	drain(out)
	a.Handle(ctx, senseAt(0.8))
	cmds := drain(out)

	sp := ticketsIn(cmds)
	if len(sp) != 2 {
		t.Fatalf("tickets=%d", len(sp))
	}
	for _, s := range sp {
		ang := math.Atan2(s.VY, s.VX)
		if math.Abs(math.Abs(ang)-twoSpread) > 1e-6 {
			t.Fatalf("两张该在瞄准线左右各偏 %.3f：ang=%v", twoSpread, ang)
		}
		if s.OwnerID != 1 || s.Slot != 0 {
			t.Fatalf("spawn owner/slot=%d/%d", s.OwnerID, s.Slot)
		}
	}
	if _, ok := fxIn(cmds, "fault"); !ok {
		t.Fatal("故障该喊一声")
	}
}

func TestMisfireFiveSpreadsEvenly(t *testing.T) {
	a := rolled(0.95) // n=1：0.90 以上是五张
	ctx, out := pipe(1)
	a.Handle(ctx, hitAt(0, 2, 9))
	drain(out)
	a.Handle(ctx, senseAt(0.8))
	cmds := drain(out)

	sp := ticketsIn(cmds)
	if len(sp) != fiveN {
		t.Fatalf("tickets=%d", len(sp))
	}
	angs := make([]float64, len(sp))
	for i, s := range sp {
		angs[i] = math.Atan2(s.VY, s.VX)
	}
	for i := 1; i < len(angs); i++ {
		d := angs[i] - angs[i-1]
		if d < 0 {
			d += 2 * math.Pi
		}
		if math.Abs(d-2*math.Pi/fiveN) > 1e-6 {
			t.Fatalf("五张该均分：相邻差 %v", d)
		}
	}
	if _, ok := fxIn(cmds, "fault"); !ok {
		t.Fatal("故障该喊一声")
	}
}

func TestSourceGoneFallsBackToRandomDirection(t *testing.T) {
	a := newSubpoena(7)
	a.roll = func() float64 { return 0.1 }
	ctx, out := pipe(1)
	a.Handle(ctx, hitAt(0, 99, 6)) // 99 不在 Nearby 里
	drain(out)
	a.Handle(ctx, senseAt(0.8))
	sp := ticketsIn(drain(out))
	if len(sp) != 1 {
		t.Fatalf("tickets=%d", len(sp))
	}
	if got := math.Hypot(sp[0].VX, sp[0].VY); math.Abs(got-ticketSpeed) > 1e-6 {
		t.Fatalf("速率=%v", got)
	}
}

// 开票站着的 0.8 秒里挨的打也累加到票面，出票即重置。
func TestWindupDamageStacksOnTicket(t *testing.T) {
	a := rolled(0.1) // 正常一张，才有开票窗口
	ctx, out := pipe(1)
	a.Handle(ctx, hitAt(0, 2, 10))
	a.Handle(ctx, hitAt(0.3, 2, 6))
	a.Handle(ctx, hitAt(0.6, 2, 4))
	drain(out)

	if len(a.shots) != 1 || math.Abs(a.shots[0].dmg-20) > 1e-9 {
		t.Fatalf("窗口内没累加：shots=%+v", a.shots)
	}
	a.Handle(ctx, senseAt(0.8))
	sp := ticketsIn(drain(out))
	if len(sp) != 1 {
		t.Fatalf("tickets=%d", len(sp))
	}
	got, _ := tickets.Load(sp[0].Stream)
	tickets.Delete(sp[0].Stream)
	if v, _ := got.(float64); math.Abs(v-20) > 1e-9 {
		t.Fatalf("票面=%v want 20", got)
	}
}

// 两张故障也是每张都带累计总额。
func TestWindupStackAppliesToEveryTicket(t *testing.T) {
	a := rolled(0.85) // 两张
	ctx, out := pipe(1)
	a.Handle(ctx, hitAt(0, 2, 10))
	a.Handle(ctx, hitAt(0.4, 2, 10))
	drain(out)
	a.Handle(ctx, senseAt(0.8))
	sp := ticketsIn(drain(out))
	if len(sp) != 2 {
		t.Fatalf("tickets=%d", len(sp))
	}
	for _, s := range sp {
		got, _ := tickets.Load(s.Stream)
		tickets.Delete(s.Stream)
		if v, _ := got.(float64); math.Abs(v-20) > 1e-9 {
			t.Fatalf("每张都该带 20，实际 %v", got)
		}
	}
}

// 出了票就重置：CD 里（没窗的时候）再挨的打不能记到下一张上。
func TestNoStackOutsideWindup(t *testing.T) {
	a := rolled(0.75) // 空窗：不出票、不站
	ctx, out := pipe(1)
	a.Handle(ctx, hitAt(0, 2, 9))
	a.Handle(ctx, senseAt(0))
	drain(out)

	a.Handle(ctx, hitAt(0.2, 2, 6)) // 空窗 CD（0.4）里，没有窗，没地方记
	if len(a.shots) != 0 {
		t.Fatalf("没有窗口却攒了票：%+v", a.shots)
	}

	a.roll = func() float64 { return 0.1 }
	a.Handle(ctx, hitAt(0.45, 2, 7)) // 下一笔从头记
	if len(a.shots) != 1 || math.Abs(a.shots[0].dmg-7) > 1e-9 {
		t.Fatalf("shots=%+v", a.shots)
	}
}

func TestNormalChanceFallsWithRecentHits(t *testing.T) {
	cases := []struct {
		n    int
		want float64
	}{
		{1, 0.70},
		{2, 0.65},
		{7, 0.40},
		{12, 0.15},
		{13, 0.10},
		{40, 0.10},
		{0, 0.70},
	}
	for _, c := range cases {
		if got := normalChance(c.n); math.Abs(got-c.want) > 1e-9 {
			t.Fatalf("normalChance(%d)=%v want %v", c.n, got, c.want)
		}
	}
}

func TestOutcomeBucketsAtOneHit(t *testing.T) {
	cases := []struct {
		r    float64
		want outcome
	}{
		{0.10, outcomeOne},
		{0.69, outcomeOne},
		{0.75, outcomeNone},
		{0.85, outcomeTwo},
		{0.95, outcomeFive},
	}
	for _, c := range cases {
		a := rolled(c.r)
		a.hits = []float64{0}
		if got := a.rollOutcome(); got != c.want {
			t.Fatalf("r=%v → %v want %v", c.r, got, c.want)
		}
	}
}

func TestThreeSecondWindowForgetsOldHits(t *testing.T) {
	a := newSubpoena(7)
	for _, t0 := range []float64{0, 1, 2, 3.5, 3.6} {
		a.noteHit(t0)
	}
	if len(a.hits) != 4 {
		t.Fatalf("3 秒窗里该有 4 下：%v", a.hits)
	}
	if got := normalChance(len(a.hits)); math.Abs(got-0.55) > 1e-9 {
		t.Fatalf("4 下时正常几率=%v", got)
	}
}

// 票面从发射者交给票：Spawn 没有载荷字段，靠 Stream 对号。
func TestTicketInheritsRecordedDamage(t *testing.T) {
	a := newSubpoena(7)
	a.roll = func() float64 { return 0.1 }
	ctx, out := pipe(1)
	a.Handle(ctx, hitAt(0, 2, 12.5))
	drain(out)
	a.Handle(ctx, senseAt(0.8))
	sp := ticketsIn(drain(out))
	if len(sp) != 1 {
		t.Fatalf("tickets=%d", len(sp))
	}
	actor := unit.NewActor(KindTicket, unit.SpawnInfo{Stream: sp[0].Stream, Slot: sp[0].Slot})
	tk, ok := actor.(*传票)
	if !ok {
		t.Fatalf("actor=%T", actor)
	}
	if math.Abs(tk.dmg-12.5) > 1e-9 {
		t.Fatalf("票面=%v", tk.dmg)
	}
	if _, left := tickets.Load(sp[0].Stream); left {
		t.Fatal("载荷该在工厂取走后就删掉")
	}
}

func TestTicketServesRecordedDamageOnContact(t *testing.T) {
	tk := &传票{dmg: 12.5, slot: 0}
	ctx, out := pipe(7)
	tk.Handle(ctx, unit.Sense{
		Time:   1,
		Self:   unit.Snapshot{ID: 7, Slot: 0, Radius: ticketRadius, VX: ticketSpeed},
		Nearby: []unit.Snapshot{foeAt(0, 0)},
	})
	cmds := drain(out)

	var dmg unit.Damage
	found := false
	for _, c := range cmds {
		if d, ok := c.(unit.Damage); ok {
			dmg, found = d, true
		}
	}
	if !found || math.Abs(dmg.Amount-12.5) > 1e-9 || dmg.From != 7 || dmg.To != 2 {
		t.Fatalf("damage=%+v found=%v", dmg, found)
	}
	gone := false
	for _, c := range cmds {
		if d, ok := c.(unit.Despawn); ok && d.UnitID == 7 {
			gone = true
		}
	}
	if !gone {
		t.Fatal("结算完该消失")
	}
}

func TestTicketStopsAfterUniformDeceleration(t *testing.T) {
	tk := &传票{dmg: 4, slot: 0}
	ctx, out := pipe(7)
	dist, x := 0.0, 0.0
	for i := 0; i < 300; i++ {
		tk.Handle(ctx, unit.Sense{
			Time: float64(i) * dt,
			Self: unit.Snapshot{ID: 7, Slot: 0, Radius: ticketRadius, VX: ticketSpeed},
		})
		moved := false
		for _, c := range drain(out) {
			if tp, ok := c.(unit.Teleport); ok {
				dist += math.Abs(tp.X - x)
				x = tp.X
				moved = true
			}
		}
		if !moved {
			break
		}
	}
	if math.Abs(dist-294) > 1e-6 {
		t.Fatalf("停下来的距离=%v，该是 v²/2a = 294", dist)
	}
}

func TestTicketIgnoresFriendlyContact(t *testing.T) {
	tk := &传票{dmg: 9, slot: 0}
	ctx, out := pipe(7)
	ally := unit.Snapshot{ID: 3, Kind: "靶子", Role: unit.RoleFighter, Slot: 0, Radius: 18, HP: 100, MaxHP: 100}
	tk.Handle(ctx, unit.Sense{
		Self:   unit.Snapshot{ID: 7, Slot: 0, Radius: ticketRadius},
		Nearby: []unit.Snapshot{ally},
	})
	for _, c := range drain(out) {
		if d, ok := c.(unit.Damage); ok {
			t.Fatalf("同槽不该吃票：%+v", d)
		}
		if _, ok := c.(unit.Despawn); ok {
			t.Fatal("同槽碰到不该把票弄没")
		}
	}
}

// 赛中现造的胶囊墙（筑墙者那类）也要挡纸：走 Sense.Walls，自己推出去再反射。
func TestTicketBouncesOffCapsuleWall(t *testing.T) {
	f := unit.HexField()
	wall := unit.WallView{ID: 1, X1: 100, Y1: -200, X2: 100, Y2: 200, Radius: 6}
	tk := &传票{dmg: 1, slot: 0}
	ctx, out := pipe(7)
	crossed, bounced := false, false
	for i := 0; i < 200; i++ {
		tk.Handle(ctx, unit.Sense{
			Time:  float64(i) * dt,
			Self:  unit.Snapshot{ID: 7, Slot: 0, Radius: ticketRadius, VX: ticketSpeed},
			Field: f,
			Walls: []unit.WallView{wall},
		})
		for _, c := range drain(out) {
			if tp, ok := c.(unit.Teleport); ok && tp.X > 100 {
				crossed = true
			}
		}
		if tk.vx < -1 {
			bounced = true
		}
	}
	if crossed {
		t.Fatal("票穿过胶囊墙了")
	}
	if !bounced {
		t.Fatal("撞胶囊墙后该往回走")
	}
}

// 墙照样挡纸：撞上就贴墙弹回来，不会穿出去。
func TestTicketBouncesOffWall(t *testing.T) {
	f := unit.HexField()
	f.Walls = []unit.FieldWall{{Kind: unit.WallCapsule, X1: 100, Y1: -200, X2: 100, Y2: 200, Radius: 6}}
	tk := &传票{dmg: 1, slot: 0}
	ctx, out := pipe(7)
	crossed, bounced := false, false
	for i := 0; i < 200; i++ {
		tk.Handle(ctx, unit.Sense{
			Time:  float64(i) * dt,
			Self:  unit.Snapshot{ID: 7, Slot: 0, Radius: ticketRadius, VX: ticketSpeed},
			Field: f,
		})
		for _, c := range drain(out) {
			if tp, ok := c.(unit.Teleport); ok && tp.X > 100 {
				crossed = true
			}
		}
		if tk.vx < -1 {
			bounced = true
		}
	}
	if crossed {
		t.Fatal("票穿墙了")
	}
	if !bounced {
		t.Fatal("撞墙后该往回走")
	}
}
