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

func lastLoot(cmds []unit.Cmd) (float64, bool) {
	got, ok := 0.0, false
	for _, c := range cmds {
		if f, is := c.(unit.FX); is && f.Name == "loot" {
			got, ok = f.Amount, true
		}
	}
	return got, ok
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

func setAims(cmds []unit.Cmd) []unit.SetAimPriority {
	var out []unit.SetAimPriority
	for _, c := range cmds {
		if s, ok := c.(unit.SetAimPriority); ok {
			out = append(out, s)
		}
	}
	return out
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
	v := lastVel(cmds)
	if v == nil || v.VY >= 0 {
		t.Fatalf("没东西可偷就该飘回主人（上方）: %+v", v)
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
	if n, ok := lastLoot(cmds); !ok || n != 1 {
		t.Fatalf("赃物应当 +1: %v", n)
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

// 指挥：吃掉赃物，把「咬伤加成」写进每只老鼠的瞄准优先度；窗口一过放它们回去偷东西。
func TestCommandSendsRatsToAttack(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	a := &吹笛人{}
	ctx := unit.Context{ID: 1, Kind: KindPiper, Out: out}
	enemy := enemyAt(120, 0)
	loot := []unit.Snapshot{
		piperAt(0, 0), enemy,
		ratAt(11, 30, 0, "狼人月亮"), ratAt(12, -30, 0, "钓鱼佬鱼塘"), ratAt(13, 0, 30, "面灵气面具1"),
	}
	a.Handle(ctx, unit.Sense{Time: 0, Self: piperAt(0, 0), Nearby: loot})
	if n, _ := lastLoot(drain(out)); n != 3 {
		t.Fatalf("三只老鼠都该入洞: %v", n)
	}
	near := []unit.Snapshot{piperAt(0, 0), enemy, ratAt(14, 40, 0, ""), ratAt(15, -40, 0, "")}
	a.Handle(ctx, unit.Sense{Time: cmdFirst, Self: piperAt(0, 0), Nearby: near})
	cmds := drain(out)
	if !hasFX(cmds, "command") {
		t.Fatalf("到点该发指挥: %v", cmds)
	}
	if n, _ := lastLoot(cmds); n != 0 {
		t.Fatalf("指挥该吃掉赃物: %v", n)
	}
	aims := setAims(cmds)
	if len(aims) != 2 {
		t.Fatalf("该给两只老鼠下命令: %+v", aims)
	}
	for _, s := range aims {
		if s.From != 1 || s.Value != uint8(cmdAim+lootCap) {
			t.Fatalf("指挥标记该带上赃物加成: %+v", s)
		}
	}
	// 窗口结束：带着指挥标记的老鼠被放回去偷东西。
	aimed := ratAt(14, 40, 0, "")
	aimed.AimPriority = uint8(cmdAim + lootCap)
	other := ratAt(15, -40, 0, "")
	other.AimPriority = uint8(cmdAim + lootCap)
	after := []unit.Snapshot{piperAt(0, 0), enemy, aimed, other}
	a.Handle(ctx, unit.Sense{Time: cmdFirst + cmdWindow, Self: piperAt(0, 0), Nearby: after})
	for _, s := range setAims(drain(out)) {
		if s.Value != unit.DefaultMortalAim {
			t.Fatalf("窗口结束该恢复默认瞄准: %+v", s)
		}
	}
}

// 咬人是隔着身位咬的：不碰到敌人也能咬到，而且咬完沿径向退开。
func TestRatBitesFromRangeThenBacksOff(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	r := newRat(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	ctx := unit.Context{ID: 20, Kind: KindRat, Out: out}
	self := unit.Snapshot{
		ID: 20, Kind: KindRat, Role: unit.RoleMinion, Slot: 0,
		Radius: ratRadius, Vision: vision, AimPriority: uint8(cmdAim),
	}
	self.X, self.Y = 0, 0
	// 中心距 39 = 18 + 9 + 12：球面间隙 12，够得着但没贴上。
	near := []unit.Snapshot{{ID: 2, Kind: "靶子", Role: unit.RoleFighter, Slot: 1, X: 39, Y: 0, Radius: 18}}
	r.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: near})
	cmds := drain(out)
	ds := damages(cmds)
	if len(ds) != 1 || ds[0].Amount != biteBase {
		t.Fatalf("隔着身位也该咬到: %+v", ds)
	}
	if !hasFX(cmds, "bite") {
		t.Fatalf("该发咬的特效: %v", cmds)
	}
	v := lastVel(cmds)
	if v == nil || v.VX >= 0 {
		t.Fatalf("咬完该沿径向退开（敌人 +x 方向，速度该朝 -x）: %+v", v)
	}
}

func TestRatBitesWhenCommanded(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	r := newRat(unit.SpawnInfo{OwnerID: 1, Slot: 0})
	ctx := unit.Context{ID: 20, Kind: KindRat, Out: out}
	self := unit.Snapshot{
		ID: 20, Kind: KindRat, Role: unit.RoleMinion, Slot: 0,
		Radius: ratRadius, Vision: vision, AimPriority: uint8(cmdAim + 2),
	}
	self.X, self.Y = 0, 0
	near := []unit.Snapshot{{ID: 2, Kind: "靶子", Role: unit.RoleFighter, Slot: 1, X: 20, Y: 0, Radius: 18}}
	r.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: near})
	ds := damages(drain(out))
	if len(ds) != 1 || ds[0].To != 2 || ds[0].Amount != biteBase+2 {
		t.Fatalf("被指挥贴上去该咬一口（含赃物加成）: %+v", ds)
	}
	// 同一次指挥里不再咬第二口，而且咬完立刻退开，不再贴着敌人。
	r.Handle(ctx, unit.Sense{Time: 0.1, Self: self, Nearby: near})
	cmds := drain(out)
	if ds := damages(cmds); len(ds) != 0 {
		t.Fatalf("一次指挥只咬一口: %+v", ds)
	}
	if v := lastVel(cmds); v == nil || v.VX >= 0 {
		t.Fatalf("咬完该从敌人身上退开: %+v", v)
	}
	// 没被指挥就不咬。
	self.AimPriority = unit.DefaultMortalAim
	r.Handle(ctx, unit.Sense{Time: 0.2, Self: self, Nearby: near})
	if ds := damages(drain(out)); len(ds) != 0 {
		t.Fatalf("没指挥不该咬人: %+v", ds)
	}
}

func TestCommandNeedsRatsAndEnemy(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &吹笛人{}
	ctx := unit.Context{ID: 1, Kind: KindPiper, Out: out}
	// 场上没有老鼠：不指挥。
	a.Handle(ctx, unit.Sense{Time: cmdFirst, Self: piperAt(0, 0), Nearby: []unit.Snapshot{piperAt(0, 0), enemyAt(100, 0)}})
	if hasFX(drain(out), "command") {
		t.Fatal("没有老鼠不该指挥")
	}
}
