package 吹笛人

import (
	"math"
	"testing"

	"xqdj/internal/unit"
)

func piperAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindPiper, Role: unit.RoleFighter, Slot: 0,
		X: x, Y: y, Radius: bodyRadius, Vision: vision,
	}
}

func enemyAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{ID: 2, Kind: "靶子", Role: unit.RoleFighter, Slot: 1, X: x, Y: y, Radius: 18}
}

func ratAt(id uint64, x, y float64, mark string) unit.Snapshot {
	s := unit.Snapshot{
		ID: id, Kind: KindRat, Role: unit.RoleMinion, Slot: 0,
		X: x, Y: y, Radius: ratRadius, HP: ratHP, MaxHP: ratHP, Mortal: true, OwnerID: 1,
	}
	if mark != "" {
		s.Marks = []unit.Mark{{Kind: mark, Stacks: 1}}
	}
	return s
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

func hasDespawn(cmds []unit.Cmd, id uint64) bool {
	for _, c := range cmds {
		if d, ok := c.(unit.Despawn); ok && d.UnitID == id {
			return true
		}
	}
	return false
}

func lastSpawn(cmds []unit.Cmd, kind string) *unit.Spawn {
	var got *unit.Spawn
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == kind {
			v := s
			got = &v
		}
	}
	return got
}

func lastVel(cmds []unit.Cmd) *unit.SetVelocity {
	var got *unit.SetVelocity
	for _, c := range cmds {
		if v, ok := c.(unit.SetVelocity); ok {
			x := v
			got = &x
		}
	}
	return got
}

func lastMark(cmds []unit.Cmd) *unit.StackMark {
	var got *unit.StackMark
	for _, c := range cmds {
		if m, ok := c.(unit.StackMark); ok {
			x := m
			got = &x
		}
	}
	return got
}

func damages(cmds []unit.Cmd) []unit.Damage {
	var out []unit.Damage
	for _, c := range cmds {
		if d, ok := c.(unit.Damage); ok {
			out = append(out, d)
		}
	}
	return out
}

func lastStack(cmds []unit.Cmd, id uint64) *unit.StackMark {
	var got *unit.StackMark
	for _, c := range cmds {
		if m, ok := c.(unit.StackMark); ok && m.UnitID == id {
			x := m
			got = &x
		}
	}
	return got
}

func hasFX(cmds []unit.Cmd, name string) bool {
	for _, c := range cmds {
		if f, ok := c.(unit.FX); ok && f.Name == name {
			return true
		}
	}
	return false
}

func hasStand(cmds []unit.Cmd, hold bool) bool {
	for _, c := range cmds {
		if s, ok := c.(unit.Stand); ok && s.Hold == hold {
			return true
		}
	}
	return false
}

func lastPass(cmds []unit.Cmd) *unit.Pass {
	var got *unit.Pass
	for _, c := range cmds {
		if p, ok := c.(unit.Pass); ok {
			x := p
			got = &x
		}
	}
	return got
}

// 老鼠和钉与锤的人偶同一套：看得见、打得死，但常驻 Pass，谁都能穿过它，不跟人挤。
func TestRatHoldsPass(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	r := newRat(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	ctx := unit.Context{ID: 20, Kind: KindRat, Out: out}
	self := unit.Snapshot{ID: 20, Kind: KindRat, Role: unit.RoleMinion, Slot: 0, Radius: ratRadius, Vision: vision}
	near := []unit.Snapshot{piperAt(0, -120)}
	r.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: near})
	p := lastPass(drain(out))
	if p == nil || !p.Hold {
		t.Fatalf("老鼠该常驻 Pass: %+v", p)
	}
	// 只发一次，不来回切。
	r.Handle(ctx, unit.Sense{Time: 0.2, Self: self, Nearby: near})
	if p := lastPass(drain(out)); p != nil {
		t.Fatalf("Pass 不该反复发: %+v", p)
	}
}

func TestPipeSummonsRat(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &吹笛人{}
	ctx := unit.Context{ID: 1, Kind: KindPiper, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0, Self: piperAt(0, 0)})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{Time: pipeFirst, Self: piperAt(0, 0)})
	cmds := drain(out)
	sp := lastSpawn(cmds, KindRat)
	if sp == nil {
		t.Fatalf("吹笛应当召鼠: %v", cmds)
	}
	if sp.OwnerID != 1 || sp.Slot != 0 {
		t.Fatalf("老鼠的归属错了: %+v", sp)
	}
	if !hasStand(cmds, true) {
		t.Fatalf("吹笛应当站定: %v", cmds)
	}
	if math.Hypot(sp.VX, sp.VY) <= 0 {
		t.Fatalf("老鼠应当有初速: %+v", sp)
	}
}

func TestRatCapStopsSummoning(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &吹笛人{}
	ctx := unit.Context{ID: 1, Kind: KindPiper, Out: out}
	near := []unit.Snapshot{piperAt(0, 0)}
	for i := 0; i < ratCap; i++ {
		near = append(near, ratAt(uint64(11+i), float64(i*10), 20, ""))
	}
	a.Handle(ctx, unit.Sense{Time: 0, Self: piperAt(0, 0), Nearby: near})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{Time: pipeFirst, Self: piperAt(0, 0), Nearby: near})
	if sp := lastSpawn(drain(out), KindRat); sp != nil {
		t.Fatalf("满编不该再召: %+v", sp)
	}
}

func TestRatStealsNearestThing(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	r := newRat(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	ctx := unit.Context{ID: 20, Kind: KindRat, Out: out}
	self := unit.Snapshot{ID: 20, Kind: KindRat, Role: unit.RoleMinion, Slot: 0, Radius: ratRadius, Vision: vision}
	near := []unit.Snapshot{
		piperAt(0, -200),
		enemyAt(300, 0),
		{ID: 7, Kind: "原型机_远程子弹", Role: unit.RoleProjectile, Slot: 1, X: 40, Y: 0, Radius: 6},
		{ID: 8, Kind: "狼人月亮", Role: unit.RoleHelper, Slot: 1, X: 200, Y: 0, Radius: 34},
		ratAt(9, 10, 0, ""),
	}
	self.X, self.Y = 0, 0
	r.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: near})
	v := lastVel(drain(out))
	if v == nil || v.VX <= 0 {
		t.Fatalf("老鼠该朝最近的敌方弹跑: %+v", v)
	}
	// 够到了：那件东西当场消失，老鼠带上「嘴里是什么」的标记。
	self.X, self.Y = 28, 0
	r.Handle(ctx, unit.Sense{Time: 0.2, Self: self, Nearby: near})
	cmds := drain(out)
	if !hasDespawn(cmds, 7) {
		t.Fatalf("该叼走那发弹: %v", cmds)
	}
	if m := lastMark(cmds); m == nil || m.Kind != "原型机_远程子弹" {
		t.Fatalf("该记下赃物是什么: %+v", m)
	}
	// 叼着东西就往主人那边跑，速度是 ratCarry。
	self.X, self.Y = 28, 0
	r.Handle(ctx, unit.Sense{Time: 0.3, Self: self, Nearby: near})
	v = lastVel(drain(out))
	if v == nil || v.VY >= 0 {
		t.Fatalf("该往主人（上方）跑: %+v", v)
	}
	if sp := math.Hypot(v.VX, v.VY); sp > ratCarry+1e-6 || sp < ratCarry*0.6 {
		t.Fatalf("回程速率该接近 ratCarry（会让开同类，略慢可以）: %+v", v)
	}
}

func TestRatKeepsClosingWhenShortOfSteal(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	r := newRat(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	ctx := unit.Context{ID: 20, Kind: KindRat, Out: out}
	self := unit.Snapshot{ID: 20, Kind: KindRat, Role: unit.RoleMinion, Slot: 0, Radius: ratRadius, Vision: vision}
	// 子弹 r=6；老鼠 r=9；判定 20 → 叼住中心距 35。停在 38：旧逻辑瞄在圈沿，steer 刹停。
	bullet := unit.Snapshot{ID: 7, Kind: "收割者镰刀", Role: unit.RoleProjectile, Slot: 1, X: 40, Y: 0, Radius: 6}
	near := []unit.Snapshot{piperAt(0, -200), bullet}
	self.X, self.Y = 2, 0
	r.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: near})
	cmds := drain(out)
	if hasDespawn(cmds, 7) {
		t.Fatalf("38 还不够到，不该已经叼走: %v", cmds)
	}
	v := lastVel(cmds)
	if v == nil || v.VX <= 0 {
		t.Fatalf("刚够不着也该继续朝镰刀走，不能停: %+v", v)
	}
}

func TestRatIgnoresFighterAndMortal(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	r := newRat(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	ctx := unit.Context{ID: 20, Kind: KindRat, Out: out}
	self := unit.Snapshot{ID: 20, Kind: KindRat, Role: unit.RoleMinion, Slot: 0, Radius: ratRadius, Vision: vision}
	self.X, self.Y = 0, 0
	near := []unit.Snapshot{
		piperAt(0, -120),
		enemyAt(30, 0),
		{ID: 5, Kind: "教父暗杀者", Role: unit.RoleMinion, Slot: 1, X: 40, Y: 0, Radius: 14, Mortal: true},
	}
	r.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: near})
	cmds := drain(out)
	if hasDespawn(cmds, 5) || hasDespawn(cmds, 2) {
		t.Fatalf("本体和活随从偷不走: %v", cmds)
	}
	if ds := damages(cmds); len(ds) != 0 {
		t.Fatalf("贴敌不该咬人: %+v", ds)
	}
}

func TestRatOrbitsEnemyWhenIdle(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	r := newRat(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	ctx := unit.Context{ID: 20, Kind: KindRat, Out: out}
	self := unit.Snapshot{ID: 20, Kind: KindRat, Role: unit.RoleMinion, Slot: 0, Radius: ratRadius, Vision: vision}
	self.X, self.Y = 0, 0
	near := []unit.Snapshot{piperAt(0, -200), enemyAt(120, 0)}
	r.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: near})
	cmds := drain(out)
	if hasDespawn(cmds, 2) {
		t.Fatalf("贴敌不该把人叼走: %v", cmds)
	}
	if ds := damages(cmds); len(ds) != 0 {
		t.Fatalf("贴敌不该咬人: %+v", ds)
	}
	v := lastVel(cmds)
	if v == nil || v.VX <= 0 {
		t.Fatalf("没东西可偷该去贴敌人: %+v", v)
	}
}

func TestRatStealsClone(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	r := newRat(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	ctx := unit.Context{ID: 20, Kind: KindRat, Out: out}
	self := unit.Snapshot{ID: 20, Kind: KindRat, Role: unit.RoleMinion, Slot: 0, Radius: ratRadius, Vision: vision}
	self.X, self.Y = 0, 0
	clone := unit.Snapshot{ID: 8, Kind: "分身", Role: unit.RoleClone, Slot: 1, X: 40, Y: 0, Radius: 18}
	near := []unit.Snapshot{piperAt(0, -200), enemyAt(120, 0), clone}
	r.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: near})
	cmds := drain(out)
	if !hasDespawn(cmds, 8) {
		t.Fatalf("分身该叼走: %v", cmds)
	}
	if m := lastMark(cmds); m == nil || m.Kind != "分身" {
		t.Fatalf("该记下嘴里是分身: %+v", m)
	}
}

func TestRatIgnoresAttackArc(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	r := newRat(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	ctx := unit.Context{ID: 20, Kind: KindRat, Out: out}
	self := unit.Snapshot{ID: 20, Kind: KindRat, Role: unit.RoleMinion, Slot: 0, Radius: ratRadius, Vision: vision}
	self.X, self.Y = 0, 0
	arc := unit.Snapshot{
		ID: 6, Kind: "小骑士弧", Role: unit.RoleProjectile, Slot: 1,
		X: 20, Y: 0, Radius: 20, ArcSpan: unit.Deg(120), ArcInner: 18,
	}
	bullet := unit.Snapshot{ID: 7, Kind: "原型机_远程子弹", Role: unit.RoleProjectile, Slot: 1, X: 80, Y: 0, Radius: 6}
	near := []unit.Snapshot{piperAt(0, -200), arc, bullet}
	r.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: near})
	cmds := drain(out)
	if hasDespawn(cmds, 6) {
		t.Fatalf("攻击环偷不走: %v", cmds)
	}
	v := lastVel(cmds)
	if v == nil || v.VX <= 0 {
		t.Fatalf("该去偷更远的弹，不该围着攻击环转: %+v", v)
	}
	// 只剩攻击环：当没东西可偷，飘回主人。
	r.Handle(ctx, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{piperAt(0, -200), arc}})
	cmds = drain(out)
	if hasDespawn(cmds, 6) {
		t.Fatalf("只有攻击环时也不该偷: %v", cmds)
	}
	v = lastVel(cmds)
	if v == nil || v.VY >= 0 {
		t.Fatalf("只有攻击环该飘回主人: %+v", v)
	}
}

func lastHeal(cmds []unit.Cmd) *unit.Heal {
	var got *unit.Heal
	for _, c := range cmds {
		if h, ok := c.(unit.Heal); ok {
			x := h
			got = &x
		}
	}
	return got
}

func TestPiperDeliversLoot(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &吹笛人{}
	ctx := unit.Context{ID: 1, Kind: KindPiper, Out: out}
	rat := ratAt(11, 30, 0, "狼人月亮")
	a.Handle(ctx, unit.Sense{Time: 0, Self: piperAt(0, 0), Nearby: []unit.Snapshot{piperAt(0, 0), rat}})
	cmds := drain(out)
	if !hasDespawn(cmds, 11) {
		t.Fatalf("跑回身边的赃物该入洞: %v", cmds)
	}
	if lastStack(cmds, 2) != nil {
		t.Fatalf("入洞不该叠瘟疫: %v", cmds)
	}
	h := lastHeal(cmds)
	if h == nil || h.UnitID != 1 || math.Abs(h.Amount-lootHeal) > 1e-9 {
		t.Fatalf("入洞该给本体回 %v 血: %+v", lootHeal, h)
	}
}

func TestSpillReturnsLootToOwner(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &吹笛人{}
	ctx := unit.Context{ID: 1, Kind: KindPiper, Out: out}
	enemy := enemyAt(200, 0)
	rat := ratAt(11, 260, 60, "狼人月亮")
	a.Handle(ctx, unit.Sense{Time: 0, Self: piperAt(0, 0), Nearby: []unit.Snapshot{piperAt(0, 0), enemy, rat}})
	if sp := lastSpawn(drain(out), "狼人月亮"); sp != nil {
		t.Fatalf("还在场的赃物不该还: %+v", sp)
	}
	// 下一拍老鼠被打死了：东西在它倒下的地方还给原主人。
	a.Handle(ctx, unit.Sense{Time: 0.1, Self: piperAt(0, 0), Nearby: []unit.Snapshot{piperAt(0, 0), enemy}})
	sp := lastSpawn(drain(out), "狼人月亮")
	if sp == nil {
		t.Fatal("老鼠被打死该把赃物还回原处")
	}
	if sp.OwnerID != 2 || sp.Slot != 1 {
		t.Fatalf("该还给原主人: %+v", sp)
	}
	if math.Abs(sp.X-260) > 1e-6 || math.Abs(sp.Y-60) > 1e-6 {
		t.Fatalf("该掉在它倒下的位置: %+v", sp)
	}
}

func TestRatDeathSpreadsPlague(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &吹笛人{}
	ctx := unit.Context{ID: 1, Kind: KindPiper, Out: out}
	enemy := enemyAt(260, 60)
	minion := unit.Snapshot{
		ID: 5, Kind: "教父暗杀者", Role: unit.RoleMinion, Slot: 1,
		X: 270, Y: 50, Radius: 14, Mortal: true,
	}
	far := enemyAt(0, 200)
	far.ID = 3
	rat := ratAt(11, 260, 60, "")
	near := []unit.Snapshot{piperAt(0, 0), enemy, minion, far, rat}
	a.Handle(ctx, unit.Sense{Time: 0, Self: piperAt(0, 0), Nearby: near})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{Time: 0.1, Self: piperAt(0, 0), Nearby: []unit.Snapshot{piperAt(0, 0), enemy, minion, far}})
	cmds := drain(out)
	if !hasFX(cmds, "plague") {
		t.Fatalf("老鼠死亡该发瘟疫圈: %v", cmds)
	}
	if m := lastStack(cmds, 2); m == nil || m.Kind != plagueKind || m.Delta != 1 {
		t.Fatalf("近处敌方战斗机该叠瘟疫: %+v", lastStack(cmds, 2))
	}
	if m := lastStack(cmds, 5); m == nil || m.Kind != plagueKind || m.Delta != 1 {
		t.Fatalf("近处敌方活随从该叠瘟疫: %+v", lastStack(cmds, 5))
	}
	if m := lastStack(cmds, 3); m != nil {
		t.Fatalf("圈外不该叠瘟疫: %+v", m)
	}
	ds := damages(cmds)
	got := map[uint64]float64{}
	for _, d := range ds {
		got[d.To] = d.Amount
	}
	if got[2] != plagueAmount(1) || got[5] != plagueAmount(1) {
		t.Fatalf("倒下当帧该按叠完后的层数跳血: %+v", ds)
	}
	if _, ok := got[3]; ok {
		t.Fatalf("圈外不该跳血: %+v", ds)
	}
}

func TestDeliverDoesNotSpreadPlague(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &吹笛人{}
	ctx := unit.Context{ID: 1, Kind: KindPiper, Out: out}
	enemy := enemyAt(20, 0)
	rat := ratAt(11, 30, 0, "狼人月亮")
	a.Handle(ctx, unit.Sense{Time: 0, Self: piperAt(0, 0), Nearby: []unit.Snapshot{piperAt(0, 0), enemy, rat}})
	cmds := drain(out)
	if lastStack(cmds, 2) != nil || hasFX(cmds, "plague") {
		t.Fatalf("入洞不该当战死叠瘟疫: %v", cmds)
	}
	a.Handle(ctx, unit.Sense{Time: 0.1, Self: piperAt(0, 0), Nearby: []unit.Snapshot{piperAt(0, 0), enemy}})
	if cmds := drain(out); lastStack(cmds, 2) != nil || hasFX(cmds, "plague") {
		t.Fatalf("入洞后也不该补一层瘟疫: %v", cmds)
	}
}

func TestLivingRatsPulsePlague(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &吹笛人{}
	ctx := unit.Context{ID: 1, Kind: KindPiper, Out: out}
	// 老鼠停在球面外 16：中心距 18+9+16=43，旧的中心距 40 根本够不着。
	enemy := enemyAt(80, 0)
	rat := ratAt(11, 80-(18+ratRadius+ratStandoff), 0, "")
	near := []unit.Snapshot{piperAt(0, 0), enemy, rat}
	a.Handle(ctx, unit.Sense{Time: 0, Self: piperAt(0, 0), Nearby: near})
	if cmds := drain(out); lastStack(cmds, 2) != nil || len(damages(cmds)) != 0 {
		t.Fatalf("刚贴上不该立刻叠: %v", cmds)
	}
	a.Handle(ctx, unit.Sense{Time: plagueHold - 0.1, Self: piperAt(0, 0), Nearby: near})
	if cmds := drain(out); lastStack(cmds, 2) != nil || len(damages(cmds)) != 0 {
		t.Fatalf("贴满 5 秒前不该叠: %v", cmds)
	}
	a.Handle(ctx, unit.Sense{Time: plagueHold, Self: piperAt(0, 0), Nearby: near})
	cmds := drain(out)
	if m := lastStack(cmds, 2); m == nil || m.Kind != plagueKind || m.Delta != 1 {
		t.Fatalf("连续贴满 5 秒该叠瘟疫: %+v", lastStack(cmds, 2))
	}
	ds := damages(cmds)
	if len(ds) != 1 || ds[0].To != 2 || ds[0].Amount != plagueAmount(1) {
		t.Fatalf("叠上当帧该跳一口: %+v", ds)
	}
	far := []unit.Snapshot{piperAt(0, 0), enemyAt(250, 0), rat}
	a.Handle(ctx, unit.Sense{Time: plagueHold + 0.1, Self: piperAt(0, 0), Nearby: far})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{Time: plagueHold + 0.2, Self: piperAt(0, 0), Nearby: near})
	a.Handle(ctx, unit.Sense{Time: plagueHold*2, Self: piperAt(0, 0), Nearby: near})
	if cmds := drain(out); lastStack(cmds, 2) != nil {
		t.Fatalf("中途离开后计时该清零，满 5 秒前不该再叠: %v", cmds)
	}
	enemy.Marks = []unit.Mark{{Kind: plagueKind, Stacks: 1}}
	near[1] = enemy
	a.Handle(ctx, unit.Sense{Time: plagueHold+0.2+plagueHold, Self: piperAt(0, 0), Nearby: near})
	ds = damages(drain(out))
	if len(ds) != 1 || ds[0].Amount != plagueAmount(2) {
		t.Fatalf("重新贴满 5 秒该再叠并跳: %+v", ds)
	}
}

func TestLivingRatsStickWhileEnemyWouldHaveLeftOldRadius(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &吹笛人{}
	ctx := unit.Context{ID: 1, Kind: KindPiper, Out: out}
	// 敌人跑开后老鼠还在追：中心距 90，旧 40 会断，球面外 100 还算贴着。
	enemy := enemyAt(90, 0)
	rat := ratAt(11, 0, 0, "")
	near := []unit.Snapshot{piperAt(0, 0), enemy, rat}
	a.Handle(ctx, unit.Sense{Time: 0, Self: piperAt(0, 0), Nearby: near})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{Time: plagueHold, Self: piperAt(0, 0), Nearby: near})
	if m := lastStack(drain(out), 2); m == nil || m.Kind != plagueKind {
		t.Fatalf("追着跑的老鼠仍该叠瘟疫: %+v", m)
	}
}

func TestPlagueTicksStacksEvery3s(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &吹笛人{}
	ctx := unit.Context{ID: 1, Kind: KindPiper, Out: out}
	enemy := enemyAt(40, 0)
	enemy.Marks = []unit.Mark{{Kind: plagueKind, Stacks: 2}}
	near := []unit.Snapshot{piperAt(0, 0), enemy}
	a.Handle(ctx, unit.Sense{Time: 0, Self: piperAt(0, 0), Nearby: near})
	if ds := damages(drain(out)); len(ds) != 0 {
		t.Fatalf("刚叠上不该立刻跳血: %+v", ds)
	}
	a.Handle(ctx, unit.Sense{Time: plagueTick - 0.1, Self: piperAt(0, 0), Nearby: near})
	if ds := damages(drain(out)); len(ds) != 0 {
		t.Fatalf("3 秒前不该跳血: %+v", ds)
	}
	a.Handle(ctx, unit.Sense{Time: plagueTick, Self: piperAt(0, 0), Nearby: near})
	ds := damages(drain(out))
	if len(ds) != 1 || ds[0].From != 1 || ds[0].To != 2 || ds[0].Amount != plagueAmount(2) {
		t.Fatalf("每 3 秒该扣层数×%d: %+v", plagueMul, ds)
	}
	enemy.Marks = []unit.Mark{{Kind: plagueKind, Stacks: 3}}
	near[1] = enemy
	a.Handle(ctx, unit.Sense{Time: plagueTick * 2, Self: piperAt(0, 0), Nearby: near})
	ds = damages(drain(out))
	if len(ds) != 1 || ds[0].Amount != plagueAmount(3) {
		t.Fatalf("跳血该跟当前层数×%d 走: %+v", plagueMul, ds)
	}
}

func TestPiperNeverCommands(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &吹笛人{}
	ctx := unit.Context{ID: 1, Kind: KindPiper, Out: out}
	near := []unit.Snapshot{piperAt(0, 0), enemyAt(100, 0), ratAt(14, 40, 0, "")}
	a.Handle(ctx, unit.Sense{Time: 0, Self: piperAt(0, 0), Nearby: near})
	a.Handle(ctx, unit.Sense{Time: 3, Self: piperAt(0, 0), Nearby: near})
	a.Handle(ctx, unit.Sense{Time: 8, Self: piperAt(0, 0), Nearby: near})
	cmds := drain(out)
	if hasFX(cmds, "command") || hasFX(cmds, "loot") || hasFX(cmds, "bite") {
		t.Fatalf("不该再指挥、报赃物数或咬人: %v", cmds)
	}
}
