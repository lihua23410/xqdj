package 藤姬

import (
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

// applyAbsorb 在单测里扮演引擎：把下发出去的 SetAbsorb 写回快照，
// 不然下一拍读到的还是旧池子。
func applyAbsorb(cmds []unit.Cmd, snaps ...*unit.Snapshot) {
	for _, c := range cmds {
		v, ok := c.(unit.SetAbsorb)
		if !ok {
			continue
		}
		for _, s := range snaps {
			if s.ID == v.UnitID {
				s.Absorb = v.Amount
			}
		}
	}
}

func TestDollOnFieldMakesFujihimeQuiet(t *testing.T) {
	out := make(chan unit.Cmd, 256)
	f := &藤姬{}
	ctx := unit.Context{State: &unit.MatchState{}, ID: 1, Kind: KindFujihime, Out: out, Stream: 7}
	self := unit.Snapshot{
		ID: 1, Kind: KindFujihime, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: fujiRadius, HP: fujiHP, MaxHP: fujiHP,
	}
	doll := unit.Snapshot{
		ID: 9, Kind: KindFujihimeDoll, Role: unit.RoleMinion, Slot: 0, OwnerID: 1,
		X: 60, Y: 0, Radius: dollRadius, HP: dollHP, MaxHP: dollHP,
	}
	f.nextGrant = 1e9

	// 场上没有幻役时不动嘲讽度。
	f.Handle(ctx, unit.Sense{Time: 0, Self: self})
	if hasCmd(drain(out), func(c unit.SetAimPriority) bool { return true }) {
		t.Fatal("没有幻役不该改嘲讽度")
	}

	// 幻役在场：压到最低。
	f.Handle(ctx, unit.Sense{Time: 0.1, Self: self, Nearby: []unit.Snapshot{doll}})
	cmds := drain(out)
	if !hasCmd(cmds, func(c unit.SetAimPriority) bool {
		return c.UnitID == 1 && c.Value == fujiQuietAim
	}) {
		t.Fatalf("幻役在场该把嘲讽度压到 %d: %v", fujiQuietAim, cmds)
	}

	// 幻役没了：恢复。
	f.Handle(ctx, unit.Sense{Time: 0.2, Self: self})
	cmds = drain(out)
	if !hasCmd(cmds, func(c unit.SetAimPriority) bool {
		return c.UnitID == 1 && c.Value == fujiDefaultAim
	}) {
		t.Fatalf("幻役走了该恢复到 %d: %v", fujiDefaultAim, cmds)
	}
}

func TestNextGrantAtAlignsToTheInterval(t *testing.T) {
	step := toseGrantEvery
	for _, tc := range []struct{ now, want float64 }{
		{0, step}, {0.02, step}, {step * 0.99, step}, {step, 2 * step}, {step * 2.35, 3 * step},
	} {
		if got := nextGrantAt(tc.now); got != tc.want {
			t.Fatalf("nextGrantAt(%v)=%v want %v", tc.now, got, tc.want)
		}
	}
}

func TestDollHitsOnContactWithPerTargetCooldown(t *testing.T) {
	out := make(chan unit.Cmd, 256)
	d := &幻役{}
	ctx := unit.Context{State: &unit.MatchState{}, ID: 9, Kind: KindFujihimeDoll, Out: out}
	self := unit.Snapshot{
		ID: 9, Kind: KindFujihimeDoll, Role: unit.RoleMinion, Slot: 0, OwnerID: 1,
		X: 0, Y: 0, Radius: dollRadius, HP: dollHP, MaxHP: dollHP,
		Marks: []unit.Mark{{Kind: toseKind, Stacks: 3}},
	}
	touching := unit.Snapshot{
		ID: 2, Kind: "靶子", Role: unit.RoleFighter, Slot: 1,
		X: dollRadius + fujiRadius - 2, Y: 0, Radius: fujiRadius, AimPriority: unit.DefaultFighterAim,
	}
	far := touching
	far.X = 400

	// 贴上来就打一下，基础伤 0，真正的点数由引擎补藤色的 +X。
	d.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{touching}})
	cmds := drain(out)
	if !hasCmd(cmds, func(c unit.Damage) bool { return c.To == 2 && c.Amount == dollDamage }) {
		t.Fatalf("贴上来的敌人该挨一下（基础 %v）: %v", dollDamage, cmds)
	}
	if !hasCmd(cmds, func(c unit.FX) bool { return c.Name == "fujihime-doll-hit" }) {
		t.Fatalf("没有撞中演出: %v", cmds)
	}

	// 0.1 秒后还贴着：同一个目标 CD 里不重复打。
	d.Handle(ctx, unit.Sense{Time: 0.1, Self: self, Nearby: []unit.Snapshot{touching}})
	if hasCmd(drain(out), func(c unit.Damage) bool { return c.To == 2 }) {
		t.Fatal("同一个目标在 CD 里不该重复挨打")
	}

	// 离得远：CD 到了也打不着。
	d.Handle(ctx, unit.Sense{Time: dollHitCD + 0.1, Self: self, Nearby: []unit.Snapshot{far}})
	if hasCmd(drain(out), func(c unit.Damage) bool { return c.To == 2 }) {
		t.Fatal("没贴上不该出伤")
	}

	// CD 到了又贴着：再打一下。
	d.Handle(ctx, unit.Sense{Time: dollHitCD + 0.2, Self: self, Nearby: []unit.Snapshot{touching}})
	if !hasCmd(drain(out), func(c unit.Damage) bool { return c.To == 2 }) {
		t.Fatal("过了 CD 该能再打一下")
	}

	// 身上没有藤色：没有基础伤害可打。
	bare := self
	bare.Marks = nil
	d2 := &幻役{}
	d2.Handle(ctx, unit.Sense{Time: 0, Self: bare, Nearby: []unit.Snapshot{touching}})
	if hasCmd(drain(out), func(c unit.Damage) bool { return c.To == 2 }) {
		t.Fatal("没有藤色就不该有伤害")
	}
}

func TestLayersComeFromPool(t *testing.T) {
	per := tosePoolPerLayer
	for _, tc := range []struct {
		pool float64
		want int
	}{
		{0, 0}, {per * 0.5, 1}, {per, 1}, {per + 1, 2}, {per * 2.5, 3}, {per * 3, 3},
	} {
		if got := layersOf(tc.pool); got != tc.want {
			t.Fatalf("layersOf(%v)=%d want %d", tc.pool, got, tc.want)
		}
	}
}

func TestFirstSenseGrantsOneShieldLayer(t *testing.T) {
	out := make(chan unit.Cmd, 256)
	f := &藤姬{}
	ctx := unit.Context{State: &unit.MatchState{}, ID: 1, Kind: KindFujihime, Out: out, Stream: 7}
	self := unit.Snapshot{
		ID: 1, Kind: KindFujihime, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: fujiRadius, HP: fujiHP, MaxHP: fujiHP,
	}
	f.Handle(ctx, unit.Sense{Time: 0, Self: self})
	cmds := drain(out)

	if !hasCmd(cmds, func(c unit.SetAbsorb) bool { return c.UnitID == 1 && c.Amount == tosePoolPerLayer }) {
		t.Fatalf("第一层该给 10 点额外血条: %v", cmds)
	}
	if !hasCmd(cmds, func(c unit.StackMark) bool {
		return c.UnitID == 1 && c.Kind == toseKind && c.Delta == 1 && c.Icon == toseIcon
	}) {
		t.Fatalf("没有结付第一层: %v", cmds)
	}
	if !hasCmd(cmds, func(c unit.SetDamageBonus) bool {
		return c.UnitID == 1 && c.Amount == toseDamagePerLayer
	}) {
		t.Fatalf("第一层该给 +%v 平a: %v", toseDamagePerLayer, cmds)
	}
	if hasCmd(cmds, func(c unit.SetHP) bool { return true }) {
		t.Fatalf("额外血条不该再动生命上限: %v", cmds)
	}
	if !hasCmd(cmds, func(c unit.Spawn) bool { return c.Kind == KindFujihimeArc && c.OwnerID == 1 }) {
		t.Fatalf("开局该挂上藤弧: %v", cmds)
	}
	if !hasCmd(cmds, func(c unit.FX) bool { return c.Name == "fujihime-tose" }) {
		t.Fatalf("结付演出缺失: %v", cmds)
	}
}

func TestSixthGrantSummonsDollAndMovesPool(t *testing.T) {
	out := make(chan unit.Cmd, 512)
	f := &藤姬{}
	ctx := unit.Context{State: &unit.MatchState{}, ID: 1, Kind: KindFujihime, Out: out, Stream: 7}
	self := unit.Snapshot{
		ID: 1, Kind: KindFujihime, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: fujiRadius, HP: fujiHP, MaxHP: fujiHP,
	}
	for i := 0; i <= 5; i++ {
		f.Handle(ctx, unit.Sense{Time: float64(i) * toseGrantEvery, Self: self})
		cmds := drain(out)
		applyAbsorb(cmds, &self)
	}
	six := 6 * tosePoolPerLayer
	all := float64(6+toseDollLayers) * tosePoolPerLayer
	if f.grants != 6 || f.layers != 6 || f.pool != six {
		t.Fatalf("六次结付后 grants=%d layers=%d pool=%v", f.grants, f.layers, f.pool)
	}
	if f.waitDoll != all {
		t.Fatalf("幻役该接手 %v，waitDoll=%v", all, f.waitDoll)
	}

	// 幻役出生后的下一拍：整包搬过去，藤姬自己退空。
	doll := unit.Snapshot{
		ID: 9, Kind: KindFujihimeDoll, Role: unit.RoleMinion, Slot: 0, OwnerID: 1,
		X: 40, Y: 0, Radius: dollRadius, HP: dollHP, MaxHP: dollHP,
	}
	f.Handle(ctx, unit.Sense{Time: 5*toseGrantEvery + 0.02, Self: self, Nearby: []unit.Snapshot{doll}})
	cmds := drain(out)
	if f.holder != 9 || f.layers != 6+toseDollLayers || f.pool != all || !f.doll {
		t.Fatalf("整包该搬到幻役：holder=%d layers=%d pool=%v doll=%v", f.holder, f.layers, f.pool, f.doll)
	}
	if !hasCmd(cmds, func(c unit.SetAbsorb) bool { return c.UnitID == 9 && c.Amount == all }) {
		t.Fatalf("幻役该拿到 90 点池子（6+3 层）: %v", cmds)
	}
	if !hasCmd(cmds, func(c unit.StackMark) bool { return c.UnitID == 9 && c.Delta == 6+toseDollLayers }) {
		t.Fatalf("幻役该一次拿到 9 层: %v", cmds)
	}
	if !hasCmd(cmds, func(c unit.SetAbsorb) bool { return c.UnitID == 1 && c.Amount == 0 }) {
		t.Fatalf("藤姬身上的池子该撤掉: %v", cmds)
	}
	if !hasCmd(cmds, func(c unit.ClearMarks) bool { return c.UnitID == 1 && c.Kind == toseKind }) {
		t.Fatalf("藤姬身上的标记该清掉: %v", cmds)
	}
}

func TestDollLosingLayerScreamsAndReturnsWholePacket(t *testing.T) {
	out := make(chan unit.Cmd, 256)
	f := &藤姬{}
	ctx := unit.Context{State: &unit.MatchState{}, ID: 1, Kind: KindFujihime, Out: out, Stream: 7}
	self := unit.Snapshot{
		ID: 1, Kind: KindFujihime, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: fujiRadius, HP: fujiHP, MaxHP: fujiHP,
	}
	eight := 8 * tosePoolPerLayer
	f.holder, f.doll, f.pool, f.layers = 9, true, 9*tosePoolPerLayer, 9
	f.nextGrant = 1e9
	doll := unit.Snapshot{
		ID: 9, Kind: KindFujihimeDoll, Role: unit.RoleMinion, Slot: 0, OwnerID: 1,
		X: 40, Y: 0, Radius: dollRadius, HP: dollHP, MaxHP: dollHP, Absorb: eight,
	}
	f.Handle(ctx, unit.Sense{Time: 61, Self: self, Nearby: []unit.Snapshot{doll}})
	cmds := drain(out)

	if f.holder != 1 || f.layers != 9 || f.pool != 9*tosePoolPerLayer {
		t.Fatalf("该整包收回藤姬：holder=%d layers=%d pool=%v", f.holder, f.layers, f.pool)
	}
	if !hasCmd(cmds, func(c unit.Despawn) bool { return c.UnitID == 9 }) {
		t.Fatalf("幻役该被撤掉: %v", cmds)
	}
	if !hasCmd(cmds, func(c unit.FX) bool { return c.Name == "fujihime-scream" && c.UnitID == 9 }) {
		t.Fatalf("该在原地气绝: %v", cmds)
	}
	if !hasCmd(cmds, func(c unit.SetAbsorb) bool { return c.UnitID == 1 && c.Amount == 9*tosePoolPerLayer }) {
		t.Fatalf("藤姬该拿到 9 层的池子: %v", cmds)
	}
	if !hasCmd(cmds, func(c unit.StackMark) bool { return c.UnitID == 1 && c.Delta == 9 }) {
		t.Fatalf("藤姬该拿到 9 层标记: %v", cmds)
	}
}

func TestPoolDrainedToZeroClearsLayers(t *testing.T) {
	out := make(chan unit.Cmd, 256)
	f := &藤姬{}
	ctx := unit.Context{State: &unit.MatchState{}, ID: 1, Kind: KindFujihime, Out: out, Stream: 7}
	self := unit.Snapshot{
		ID: 1, Kind: KindFujihime, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: fujiRadius, HP: fujiHP, MaxHP: fujiHP, Absorb: 0,
	}
	f.holder, f.pool, f.layers = 1, 20, 2
	f.nextGrant = 1e9

	f.Handle(ctx, unit.Sense{Time: 12, Self: self})
	cmds := drain(out)
	if f.layers != 0 {
		t.Fatalf("池子打空该剩 0 层，实际 %d", f.layers)
	}
	if !hasCmd(cmds, func(c unit.StackMark) bool { return c.UnitID == 1 && c.Delta == -2 }) {
		t.Fatalf("没有掉层表现: %v", cmds)
	}
	if !hasCmd(cmds, func(c unit.SetDamageBonus) bool { return c.UnitID == 1 && c.Amount == 0 }) {
		t.Fatalf("平a 加伤该跟着清零: %v", cmds)
	}
}

func TestGrantsNeverLoseLayersWhenHolderChanges(t *testing.T) {
	out := make(chan unit.Cmd, 2048)
	f := &藤姬{}
	ctx := unit.Context{State: &unit.MatchState{}, ID: 1, Kind: KindFujihime, Out: out, Stream: 3}
	self := unit.Snapshot{
		ID: 1, Kind: KindFujihime, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: fujiRadius, HP: fujiHP, MaxHP: fujiHP,
	}
	ally := unit.Snapshot{
		ID: 2, Kind: "队友", Role: unit.RoleFighter, Slot: 0,
		X: 90, Y: 0, Radius: fujiRadius, HP: 100, MaxHP: 100,
	}
	for i := 0; i < 7; i++ {
		f.Handle(ctx, unit.Sense{Time: float64(i) * toseGrantEvery, Self: self, Nearby: []unit.Snapshot{ally}})
		cmds := drain(out)
		applyAbsorb(cmds, &self, &ally)
		// 场上没有幻役，池子只涨不减：层数就该等于结付次数。
		if f.layers != f.grants || f.pool != float64(f.grants)*tosePoolPerLayer {
			t.Fatalf("第 %d 次结付后该有 %d 层，实际 %d 层 %v 池子（holder=%d）", f.grants, f.grants, f.layers, f.pool, f.holder)
		}
	}
}

func TestTransferKeepsRemainingPool(t *testing.T) {
	out := make(chan unit.Cmd, 256)
	f := &藤姬{}
	ctx := unit.Context{State: &unit.MatchState{}, ID: 1, Kind: KindFujihime, Out: out, Stream: 3}
	self := unit.Snapshot{
		ID: 1, Kind: KindFujihime, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: fujiRadius, HP: fujiHP, MaxHP: fujiHP, Absorb: 25,
	}
	ally := unit.Snapshot{
		ID: 2, Kind: "队友", Role: unit.RoleFighter, Slot: 0,
		X: 90, Y: 0, Radius: fujiRadius, HP: 100, MaxHP: 100,
	}
	s := unit.Sense{Time: 10, Self: self, Nearby: []unit.Snapshot{ally}}
	// 残池 25 点 = 3 层；换人接手时搬的是剩下的池子，不是层数×10。
	f.holder, f.pool, f.layers = 1, 1.5*tosePoolPerLayer, 2
	f.moveTo(ctx, s, 2, 2.5*tosePoolPerLayer)
	cmds := drain(out)

	if f.holder != 2 || f.pool != 2.5*tosePoolPerLayer || f.layers != 3 {
		t.Fatalf("换人该搬残池 1.5+1 层 = 2.5 层、3 层，实际 %v 点 %d 层", f.pool, f.layers)
	}
	if !hasCmd(cmds, func(c unit.SetAbsorb) bool { return c.UnitID == 2 && c.Amount == 2.5*tosePoolPerLayer }) {
		t.Fatalf("接手方该拿到剩的池子: %v", cmds)
	}
	if !hasCmd(cmds, func(c unit.SetAbsorb) bool { return c.UnitID == 1 && c.Amount == 0 }) {
		t.Fatalf("原持有人该清空池子: %v", cmds)
	}
	if !hasCmd(cmds, func(c unit.SetDamageBonus) bool { return c.UnitID == 2 && c.Amount == 3*toseDamagePerLayer }) {
		t.Fatalf("接手方该拿到 +4 平a: %v", cmds)
	}
}

func TestDollKilledElsewhereStillScreams(t *testing.T) {
	out := make(chan unit.Cmd, 256)
	f := &藤姬{}
	ctx := unit.Context{State: &unit.MatchState{}, ID: 1, Kind: KindFujihime, Out: out, Stream: 7}
	self := unit.Snapshot{
		ID: 1, Kind: KindFujihime, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: fujiRadius, HP: fujiHP, MaxHP: fujiHP,
	}
	pool := 3 * tosePoolPerLayer
	doll := unit.Snapshot{
		ID: 9, Kind: KindFujihimeDoll, Role: unit.RoleMinion, Slot: 0, OwnerID: 1,
		X: 70, Y: -20, Radius: dollRadius, HP: dollHP, MaxHP: dollHP, Absorb: pool,
	}
	f.nextGrant = 1e9
	f.holder, f.doll, f.pool, f.layers = 9, true, pool, layersOf(pool)

	// 先看见它在哪。
	f.Handle(ctx, unit.Sense{Time: 30, Self: self, Nearby: []unit.Snapshot{doll}})
	drain(out)

	// 下一拍它已经不在了（被一下打空、被撤掉都算死亡）：原地气绝 + 整包还回来。
	f.Handle(ctx, unit.Sense{Time: 30.1, Self: self})
	cmds := drain(out)
	if !hasCmd(cmds, func(c unit.FX) bool {
		return c.Name == "fujihime-scream" && c.UnitID == 9 && c.X == 70 && c.Y == -20
	}) {
		t.Fatalf("幻役在别处死了也该在原地气绝: %v", cmds)
	}
	if f.holder != 1 || f.layers != layersOf(pool) || f.pool != pool {
		t.Fatalf("整包该还给藤姬：holder=%d layers=%d pool=%v", f.holder, f.layers, f.pool)
	}
}
