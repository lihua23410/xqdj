package 时计_铭刻

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

func hasCmd[T any](cmds []unit.Cmd, want func(T) bool) bool {
	for _, c := range cmds {
		if v, ok := c.(T); ok && want(v) {
			return true
		}
	}
	return false
}

type rig struct {
	out  chan unit.Cmd
	ctx  unit.Context
	self unit.Snapshot
	a    *铭刻
}

// newRig 起一只已经 boot 过的铭刻。出牌计划被清空：这些单测直接调方法，
// 不让随机的出牌时机污染断言（排程本身另有专门用例）。
func newRig(t *testing.T) *rig {
	t.Helper()
	out := make(chan unit.Cmd, 512)
	r := &rig{
		out: out,
		ctx: unit.Context{State: &unit.MatchState{}, ID: 1, Kind: KindInscribed, Out: out, Stream: 7},
		self: unit.Snapshot{
			ID: 1, Kind: KindInscribed, Role: unit.RoleFighter, Slot: 0,
			X: 100, Y: 100, Radius: inscribedRadius,
			HP: inscribedHP, MaxHP: inscribedHP, FaceX: 1,
		},
		a: &铭刻{},
	}
	r.a.Handle(r.ctx, unit.Sense{Time: 0, Self: r.self})
	r.a.plan = nil
	drain(out)
	return r
}

func (r *rig) sense(t float64, nearby ...unit.Snapshot) []unit.Cmd {
	r.a.Handle(r.ctx, unit.Sense{Time: t, Self: r.self, Nearby: nearby})
	return drain(r.out)
}

func (r *rig) playAt(t float64, idx int, nearby ...unit.Snapshot) []unit.Cmd {
	r.a.playCard(r.ctx, unit.Sense{Time: t, Self: r.self, Nearby: nearby}, idx)
	return drain(r.out)
}

func TestBootDealsMarkAndOneCard(t *testing.T) {
	r := newRig(t)
	if len(r.a.hand) != 2 {
		t.Fatalf("开局手牌该是 2 张，实得 %d: %v", len(r.a.hand), r.a.hand)
	}
	if r.a.hand[0] != cardMark {
		t.Fatalf("开局第一张必须是铭刻，实得 %v", r.a.hand)
	}
	if len(r.a.deck) != 7 {
		t.Fatalf("开局牌库该剩 7 张，实得 %d", len(r.a.deck))
	}
	if r.a.progress != 0 {
		t.Fatalf("开局抽卡进度该是 0，实得 %v", r.a.progress)
	}
	if r.a.have {
		t.Fatal("开局不该有记录")
	}
}

func TestOnePointPerSecondAndDrawEveryFive(t *testing.T) {
	r := newRig(t)
	for i := 1; i <= 4; i++ {
		r.a.charge(r.ctx, unit.Sense{Time: float64(i), Self: r.self})
	}
	if len(r.a.hand) != 2 {
		t.Fatalf("第 4 秒还不该抽牌，手牌 %d", len(r.a.hand))
	}
	if math.Abs(r.a.progress-4) > 1e-9 {
		t.Fatalf("第 4 秒该攒到 4 点，实得 %v", r.a.progress)
	}
	r.a.charge(r.ctx, unit.Sense{Time: 5, Self: r.self})
	if len(r.a.hand) != 3 {
		t.Fatalf("第 5 秒该抽到第 3 张，手牌 %d", len(r.a.hand))
	}
	if len(r.a.deck) != 6 {
		t.Fatalf("抽完该剩 6 张，实得 %d", len(r.a.deck))
	}
	if math.Abs(r.a.progress) > 1e-9 {
		t.Fatalf("抽完进度该清零，实得 %v", r.a.progress)
	}
	r.a.charge(r.ctx, unit.Sense{Time: 10, Self: r.self})
	if len(r.a.hand) != 4 {
		t.Fatalf("第 10 秒该抽到第 4 张（5 秒一张），手牌 %d", len(r.a.hand))
	}
}

func TestPlanIsTwoOrThreeAndInsideTheWindow(t *testing.T) {
	seen := [4]int{}
	for i := 0; i < 300; i++ {
		r := newRig(t)
		for w := 0; w < 60; w++ {
			win := float64(w * 10)
			plan := r.a.rollPlan(win)
			if len(plan) < 2 || len(plan) > 3 {
				t.Fatalf("一个窗口该出 2~3 张，实得 %d", len(plan))
			}
			for j, at := range plan {
				if at < win || at >= win+windowGap {
					t.Fatalf("出牌时刻 %v 跑出窗口 [%v,%v)", at, win, win+windowGap)
				}
				if j > 0 && at < plan[j-1]+playGap-1e-9 {
					t.Fatalf("两张牌间隔不足 %v: %v", playGap, plan)
				}
			}
			seen[len(plan)]++
		}
	}
	// 2 张和 3 张都得出现过，否则「均匀掷 2~3」就是假的。
	for i := 2; i < len(seen); i++ {
		if seen[i] == 0 {
			t.Fatalf("出 %d 张的窗口一次都没有，分布不对: %v", i, seen)
		}
	}
	if seen[0] != 0 || seen[1] != 0 {
		t.Fatalf("不该有 0/1 张的窗口，实得 %v", seen)
	}
}

func TestMarkRecordsThenSinksBacktrack(t *testing.T) {
	r := newRig(t)
	r.a.hand = []uint8{cardMark, cardTrace}
	r.a.progress = 4
	r.a.traceAt = []float64{2, 3}
	r.a.have = false
	cmds := r.playAt(1, 0)

	if !r.a.have {
		t.Fatal("打出铭刻后该有记录")
	}
	if n := len(r.a.deck); n == 0 || r.a.deck[n-1] != cardBacktrack {
		t.Fatalf("牌库底该是溯时: %v", r.a.deck)
	}
	if len(r.a.rec.hand) != 1 || r.a.rec.hand[0] != cardTrace {
		t.Fatalf("记录的手牌不该含铭刻本身: %v", r.a.rec.hand)
	}
	if r.a.rec.trace != 2 || r.a.rec.progress != 4 {
		t.Fatalf("记录该带上时痕层数与抽卡进度: %+v", r.a.rec)
	}
	if !hasCmd(cmds, func(c unit.FX) bool { return c.Name == fxMark }) {
		t.Fatalf("该发铭刻演出: %v", cmds)
	}
}

func TestRewindCardsHaveDifferentJobs(t *testing.T) {
	r := newRig(t)
	r.a.hand = []uint8{cardTrace, cardGuard, cardEmber, cardJourney}
	r.a.have = false

	// 没有记录：两张回溯牌都打不出来。
	if r.a.playable(cardJourney) || r.a.playable(cardBacktrack) {
		t.Fatal("没有记录时回溯牌不该被打出")
	}

	// 曜时之旅只刷新 buff：buff 掉下去了才打。
	r.a.have = true
	r.a.maxHP = 60
	r.a.rec = record{hp: 60, hand: []uint8{cardTrace}, trace: 2, guardPool: 30}
	r.a.traceAt = []float64{99, 99}
	r.a.guardPool = 30
	if r.a.playable(cardJourney) {
		t.Fatal("buff 都是满的时候不该用曜时之旅")
	}
	r.a.traceAt = r.a.traceAt[:1]
	if !r.a.playable(cardJourney) {
		t.Fatal("时痕掉层之后该用曜时之旅补回来")
	}
	// 溯时不管血，只管牌：记录里的牌更多才打。
	if r.a.playable(cardBacktrack) {
		t.Fatal("记录里的牌不比手里多时不该用溯时")
	}
	r.a.rec.hand = []uint8{cardTrace, cardGuard, cardEmber}
	r.a.hand = []uint8{cardTrace}
	if !r.a.playable(cardBacktrack) {
		t.Fatal("记录里的牌更多时该用溯时")
	}
}

func TestMarkNeedsHealthyHand(t *testing.T) {
	r := newRig(t)
	r.a.have = false
	r.a.maxHP = 60
	r.a.hp = 20
	if !r.a.playable(cardMark) {
		t.Fatal("没有记录时无条件做第一次记录")
	}
	r.a.have = true
	r.a.hp = 20
	r.a.hand = []uint8{cardTrace, cardGuard, cardEmber}
	if r.a.playable(cardMark) {
		t.Fatal("残血时不该记录")
	}
	r.a.hp = 60
	r.a.hand = []uint8{cardTrace, cardGuard}
	if r.a.playable(cardMark) {
		t.Fatal("牌不够时不该记录")
	}
	r.a.hand = append(r.a.hand, cardEmber)
	if !r.a.playable(cardMark) {
		t.Fatal("健康且攒够牌时该记录")
	}
}

func TestJourneyRefreshesBuffsButNotHealth(t *testing.T) {
	r := newRig(t)
	r.a.have = true
	r.a.rec = record{
		hp: 30, maxHP: 45, x: 40, y: 60, trace: 2, guard: 1, guardPool: 5,
		hand:     []uint8{cardGuard},
		progress: 3,
	}
	r.a.hand = []uint8{cardJourney, cardTrace}
	r.a.progress = 9
	r.a.hp = 12
	r.a.traceAt = []float64{99}
	cmds := r.playAt(20, 0)

	// 牌不动。
	if len(r.a.hand) != 1 || r.a.hand[0] != cardTrace {
		t.Fatalf("曜时之旅不该动牌，实得 %v", r.a.hand)
	}
	// 一滴血都不回。
	if math.Abs(r.a.hp-12) > 1e-9 {
		t.Fatalf("回溯不该回血，实得 hp=%v", r.a.hp)
	}
	if hasCmd(cmds, func(c unit.SetHP) bool { return true }) {
		t.Fatalf("回溯不该再发 SetHP: %v", cmds)
	}
	// buff 刷新成满寿命、额外血条补回记录值。
	if len(r.a.traceAt) != 2 {
		t.Fatalf("时痕该刷回 2 层，实得 %v", r.a.traceAt)
	}
	for _, at := range r.a.traceAt {
		if math.Abs(at-(20+buffLife)) > 1e-9 {
			t.Fatalf("刷新后的时痕该是满寿命，实得 %v", r.a.traceAt)
		}
	}
	if math.Abs(r.a.guardPool-5) > 1e-9 || len(r.a.guardAt) != 1 {
		t.Fatalf("长守该补回记录时的池子，实得 pool=%v layers=%d", r.a.guardPool, len(r.a.guardAt))
	}
	if !hasCmd(cmds, func(c unit.SetAbsorb) bool { return math.Abs(c.Amount-5) < 1e-9 }) {
		t.Fatalf("该把额外血条写回去: %v", cmds)
	}
	if !hasCmd(cmds, func(c unit.Teleport) bool { return c.X == 40 && c.Y == 60 }) {
		t.Fatalf("该瞬移回记录点: %v", cmds)
	}
}

func TestBacktrackRestoresHandAndProgress(t *testing.T) {
	r := newRig(t)
	r.a.have = true
	r.a.rec = record{
		hp: 30, maxHP: 45,
		trace: 2, guard: 1,
		hand:     []uint8{cardGuard, cardEmber},
		progress: 2,
		cdLeft:   11,
	}
	r.a.hand = []uint8{cardBacktrack, cardTrace}
	r.a.progress = 3
	r.a.traceAt = []float64{99, 99, 99}
	r.a.hp = 12
	cmds := r.playAt(20, 0)

	if len(r.a.hand) != 3 || r.a.hand[0] != cardGuard || r.a.hand[1] != cardEmber || r.a.hand[2] != cardMark {
		t.Fatalf("溯时该把手牌读回记录值再加一张铭刻，实得 %v", r.a.hand)
	}
	if math.Abs(r.a.progress-2) > 1e-9 {
		t.Fatalf("溯时该把抽卡进度也读回来，实得 %v", r.a.progress)
	}
	if math.Abs(r.a.cdReadyAt-31) > 1e-9 {
		t.Fatalf("冷却剩余 11s 该在 t=20 还原成 31，实得 %v", r.a.cdReadyAt)
	}
	// 状态不动：血、buff 层数、位置都不该被碰。
	if math.Abs(r.a.hp-12) > 1e-9 || len(r.a.traceAt) != 3 {
		t.Fatalf("回溯不该再回血/回 buff：hp=%v trace=%d", r.a.hp, len(r.a.traceAt))
	}
	if hasCmd(cmds, func(c unit.SetHP) bool { return true }) ||
		hasCmd(cmds, func(c unit.SetDamageBonus) bool { return true }) ||
		hasCmd(cmds, func(c unit.Teleport) bool { return true }) {
		t.Fatalf("回溯不该再发这些状态指令: %v", cmds)
	}
}

func TestGuardGivesExtraHealthPool(t *testing.T) {
	r := newRig(t)
	r.a.hand = []uint8{cardGuard}
	cmds := r.playAt(1, 0)
	if !hasCmd(cmds, func(c unit.SetAbsorb) bool { return math.Abs(c.Amount-guardPool) < 1e-9 }) {
		t.Fatalf("长守该给一条 %v 点的额外血条: %v", guardPool, cmds)
	}
	if len(r.a.guardAt) != 1 {
		t.Fatalf("长守该记一层到期时刻，实得 %d", len(r.a.guardAt))
	}
	// 到点之后整层掉光，池子清零。
	r.sense(1 + buffLife + 0.1)
	if r.a.guardPool > 1e-9 || len(r.a.guardAt) != 0 {
		t.Fatalf("长守过期后池子该清零，实得 pool=%v layers=%d", r.a.guardPool, len(r.a.guardAt))
	}
}

func TestTraceExpiresAndRecalculates(t *testing.T) {
	r := newRig(t)
	r.a.hand = []uint8{cardTrace}
	cmds := r.playAt(1, 0)
	if !hasCmd(cmds, func(c unit.SetDamageBonus) bool { return math.Abs(c.Amount-traceStep) < 1e-9 }) {
		t.Fatalf("一层时痕该给 %v 加伤: %v", traceStep, cmds)
	}
	// 再补一层，到期时刻比第一层晚 5 秒。
	r.a.traceAt = append(r.a.traceAt, 1+buffLife+5)
	cmds = r.sense(1 + buffLife + 0.05)
	if !hasCmd(cmds, func(c unit.SetDamageBonus) bool { return math.Abs(c.Amount-traceStep) < 1e-9 }) {
		t.Fatalf("第一层过期后该只剩一层 %v：%v", traceStep, cmds)
	}
	if len(r.a.traceAt) != 1 {
		t.Fatalf("该剩一层，实得 %v", r.a.traceAt)
	}
	cmds = r.sense(1 + buffLife + 5.1)
	if !hasCmd(cmds, func(c unit.SetDamageBonus) bool { return math.Abs(c.Amount) < 1e-9 }) {
		t.Fatalf("全过期后加伤该归零：%v", cmds)
	}
}

func TestTouchDrawRunsOnTwentySecondCooldown(t *testing.T) {
	r := newRig(t)
	self := r.self
	touching := unit.Snapshot{ID: 2, Kind: "x", Role: unit.RoleFighter, Slot: 1, X: self.X + 20, Y: self.Y, Radius: 18}
	far := unit.Snapshot{ID: 3, Kind: "x", Role: unit.RoleFighter, Slot: 1, X: self.X + 200, Y: self.Y, Radius: 18, AimPriority: 15}

	before := len(r.a.hand)
	r.a.touchDraw(r.ctx, unit.Sense{Time: 5, Self: self, Nearby: []unit.Snapshot{far}})
	if len(r.a.hand) != before {
		t.Fatal("没贴到人不该抽")
	}
	r.a.touchDraw(r.ctx, unit.Sense{Time: 5, Self: self, Nearby: []unit.Snapshot{touching}})
	if len(r.a.hand) != before+1 {
		t.Fatalf("弧贴到人该抽一张，手牌 %d -> %d", before, len(r.a.hand))
	}
	if math.Abs(r.a.cdReadyAt-25) > 1e-9 {
		t.Fatalf("抽完该进 20s 冷却（t=5 → 25），实得 %v", r.a.cdReadyAt)
	}
	mid := len(r.a.hand)
	r.a.touchDraw(r.ctx, unit.Sense{Time: 20, Self: self, Nearby: []unit.Snapshot{touching}})
	if len(r.a.hand) != mid {
		t.Fatal("冷却没到不该再抽")
	}
	r.a.touchDraw(r.ctx, unit.Sense{Time: 25, Self: self, Nearby: []unit.Snapshot{touching}})
	if len(r.a.hand) != mid+1 {
		t.Fatal("冷却到了该再抽一张")
	}
}

func TestEmberHitsOnlyInsideRadius(t *testing.T) {
	r := newRig(t)
	r.a.hand = []uint8{cardEmber}
	self := r.self
	near := unit.Snapshot{ID: 2, Kind: "x", Role: unit.RoleFighter, Slot: 1, X: self.X + 100, Y: self.Y, Radius: 18}
	far := unit.Snapshot{ID: 3, Kind: "x", Role: unit.RoleFighter, Slot: 1, X: self.X + 200, Y: self.Y, Radius: 18}
	cmds := r.playAt(1, 0, near, far)

	hitNear, hitFar := false, false
	for _, c := range cmds {
		if d, ok := c.(unit.Damage); ok {
			if d.To == 2 && math.Abs(d.Amount-emberDamage) < 1e-9 {
				hitNear = true
			}
			if d.To == 3 {
				hitFar = true
			}
		}
	}
	if !hitNear || hitFar {
		t.Fatalf("烬时之斩只该打半径 %v 内的敌人: near=%v far=%v", emberRadius, hitNear, hitFar)
	}
	// 手牌少一张烬时之斩，又多一张牌库底的牌。
	if len(r.a.hand) != 1 {
		t.Fatalf("烬时之斩该抽回牌库底一张，手牌实得 %v", r.a.hand)
	}
}

func TestEmptyDeckRefillsWithEight(t *testing.T) {
	r := newRig(t)
	r.a.deck = nil
	r.a.drawTop()
	if len(r.a.deck) != 7 {
		t.Fatalf("空牌库该先重铸 8 张再抽，剩 %d", len(r.a.deck))
	}
}

func TestDashIsStraightAndDealsNoDamage(t *testing.T) {
	r := newRig(t)
	self := r.self
	enemy := unit.Snapshot{ID: 2, Kind: "x", Role: unit.RoleFighter, Slot: 1, X: self.X + 200, Y: self.Y, Radius: 18, AimPriority: 15}
	r.a.attack(r.ctx, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{enemy}})
	cmds := drain(r.out)
	if !hasCmd(cmds, func(c unit.SetVelocity) bool {
		return math.Abs(math.Hypot(c.VX, c.VY)-dashSpeed) < 1e-6 && c.VX > 0
	}) {
		t.Fatalf("该朝敌人冲到 %v 速: %v", dashSpeed, cmds)
	}
	if !r.a.dashing {
		t.Fatal("出牌后该处于冲刺状态")
	}

	// 冲进人堆里也不出伤——伤害全归常驻弧。
	self.X = enemy.X - 20
	r.self = self
	cmds = r.sense(1.05, enemy)
	if hasCmd(cmds, func(c unit.Damage) bool { return true }) {
		t.Fatalf("冲刺不该自己结算伤害: %v", cmds)
	}
}

// 弧只砍身前那 90° 一扇：背后的、正侧面的都不算。
func TestArcIsDirectional(t *testing.T) {
	sp, ok := unit.Lookup(KindInscribedArc)
	if !ok || math.Abs(sp.ArcSpan-unit.Deg(arcSpanDeg)) > 1e-9 {
		t.Fatalf("弧该是 %v° 的一扇，实得 %+v", arcSpanDeg, sp)
	}

	r := newRig(t)
	self := r.self // 静止、面朝 +x
	before := len(r.a.hand)
	for _, tc := range []struct {
		name   string
		dx, dy float64
	}{
		{"身后", -20, 0},
		{"正侧面", 0, 20},
	} {
		o := unit.Snapshot{ID: 2, Kind: "x", Role: unit.RoleFighter, Slot: 1, X: self.X + tc.dx, Y: self.Y + tc.dy, Radius: 18}
		r.a.touchDraw(r.ctx, unit.Sense{Time: 5, Self: self, Nearby: []unit.Snapshot{o}})
		if len(r.a.hand) != before {
			t.Fatalf("%s的敌人不该触发命中抽底", tc.name)
		}
	}

	// 弧跟着速度转：往后跑，原来正前方那个人就跑到刀面背后了。
	moving := self
	moving.VX = -165
	target := unit.Snapshot{ID: 3, Kind: "x", Role: unit.RoleFighter, Slot: 1, X: self.X + 20, Y: self.Y, Radius: 18}
	r.a.touchDraw(r.ctx, unit.Sense{Time: 5, Self: moving, Nearby: []unit.Snapshot{target}})
	if len(r.a.hand) != before {
		t.Fatal("往后跑时身后的敌人不该算命中")
	}

	// 同样一个目标，速度朝 +x 时就在刀面里了。
	ahead := self
	ahead.VX = 165
	r.a.touchDraw(r.ctx, unit.Sense{Time: 5, Self: ahead, Nearby: []unit.Snapshot{target}})
	if len(r.a.hand) != before+1 {
		t.Fatal("正对着的敌人该触发命中抽底")
	}
}

func TestArcHitsAndRespawns(t *testing.T) {
	r := newRig(t)
	self := r.self

	// 1) 弧还没长出来：第一条 Sense 就把它挂上。
	cmds := r.sense(1)
	if !hasCmd(cmds, func(c unit.Spawn) bool { return c.Kind == KindInscribedArc }) {
		t.Fatalf("该长出常驻弧: %v", cmds)
	}

	// 弧自己的身体打中人：出伤 + 自己散掉。
	out := make(chan unit.Cmd, 32)
	arcCtx := unit.Context{State: &unit.MatchState{}, ID: 9, Kind: KindInscribedArc, Out: out, Stream: 3}
	(&铭刻弧{slot: 0}).Handle(arcCtx, unit.Collision{Time: 1, Other: unit.Snapshot{
		ID: 2, Slot: 1, Role: unit.RoleFighter, X: self.X + 18, Y: self.Y, Radius: 18,
	}})
	hit := drain(out)
	if !hasCmd(hit, func(c unit.Damage) bool { return c.To == 2 && math.Abs(c.Amount-arcDamage) < 1e-9 }) {
		t.Fatalf("弧该出 %v 伤害: %v", arcDamage, hit)
	}
	if !hasCmd(hit, func(c unit.Despawn) bool { return c.UnitID == 9 }) {
		t.Fatalf("打中之后弧该散掉: %v", hit)
	}

	// 2) 弧还活着（在 Nearby 里）时不会重复挂。
	live := unit.Snapshot{ID: 9, Kind: KindInscribedArc, Role: unit.RoleProjectile, OwnerID: 1, Slot: 0, Radius: arcOuter}
	if hasCmd(r.sense(1.05, live), func(c unit.Spawn) bool { return c.Kind == KindInscribedArc }) {
		t.Fatal("弧还在身上就不该再挂一条")
	}

	// 3) 弧碎了（从 Nearby 里消失）：等 arcCD 才重新长出来。
	for t2 := 1.10; t2 < 1.05+arcCD; t2 += 0.05 {
		if hasCmd(r.sense(t2), func(c unit.Spawn) bool { return c.Kind == KindInscribedArc }) {
			t.Fatalf("t=%.2f 不该已经重挂", t2)
		}
	}
	if !hasCmd(r.sense(1.15+arcCD), func(c unit.Spawn) bool { return c.Kind == KindInscribedArc }) {
		t.Fatal("冷却到了该重挂弧")
	}
}
