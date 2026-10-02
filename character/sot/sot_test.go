package 酒翁

import (
	"math"
	"math/rand/v2"
	"testing"

	"xqdj/internal/unit"
)

func sotAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindSot, Role: unit.RoleFighter, Slot: 0,
		X: x, Y: y, Radius: bodyRadius, Vision: vision, AimPriority: 15,
	}
}

func enemyAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 2, Kind: "靶子", Role: unit.RoleFighter, Slot: 1,
		X: x, Y: y, Radius: 18, AimPriority: 15,
	}
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

func spawns(cmds []unit.Cmd, kind string) []unit.Spawn {
	var out []unit.Spawn
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == kind {
			out = append(out, s)
		}
	}
	return out
}

func lastMark(cmds []unit.Cmd, id uint64) *unit.StackMark {
	var got *unit.StackMark
	for _, c := range cmds {
		if m, ok := c.(unit.StackMark); ok && m.UnitID == id {
			v := m
			got = &v
		}
	}
	return got
}

func marksOn(cmds []unit.Cmd, id uint64) []unit.StackMark {
	var out []unit.StackMark
	for _, c := range cmds {
		if m, ok := c.(unit.StackMark); ok && m.UnitID == id {
			out = append(out, m)
		}
	}
	return out
}

func lastFS(cmds []unit.Cmd, id uint64) *unit.AddFSComponent {
	var got *unit.AddFSComponent
	for _, c := range cmds {
		if v, ok := c.(unit.AddFSComponent); ok && v.UnitID == id {
			x := v
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

func boot(a *酒翁, out chan unit.Cmd, near []unit.Snapshot) {
	ctx := unit.Context{ID: 1, Kind: KindSot, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0, Self: sotAt(0, 0), Nearby: near, Field: unit.HexField()})
	_ = drain(out)
}

func TestFirstSmashDropsOnEnemyHead(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &酒翁{}
	near := []unit.Snapshot{sotAt(0, 0), enemyAt(80, -40)}
	boot(a, out, near)
	ctx := unit.Context{ID: 1, Kind: KindSot, Out: out}
	a.Handle(ctx, unit.Sense{Time: smashFirst - 0.1, Self: sotAt(0, 0), Nearby: near, Field: unit.HexField()})
	if n := len(spawns(drain(out), KindSotWine)); n != 0 {
		t.Fatalf("到点前不该砸: %d", n)
	}
	a.Handle(ctx, unit.Sense{Time: smashFirst, Self: sotAt(0, 0), Nearby: near, Field: unit.HexField()})
	cmds := drain(out)
	wines := spawns(cmds, KindSotWine)
	want := rand.New(rand.NewPCG(1, 1)).IntN(3) + 1
	if len(wines) != 1 {
		t.Fatalf("到点先砸第 1 坛, got %d want 1 (本轮共 %d)", len(wines), want)
	}
	topX, topY := dropPoint(unit.HexField(), 80, wineRadius)
	wantX, wantY := insetFromOutline(unit.HexField(), topX, topY, wineRadius, wineSpawnPad)
	if math.Abs(wines[0].X-wantX) > 1 || math.Abs(wines[0].Y-wantY) > 1 {
		t.Fatalf("该砸在敌人头顶场顶内侧: got (%v,%v) want ~(%v,%v)", wines[0].X, wines[0].Y, wantX, wantY)
	}
	if wines[0].VY >= 0 {
		t.Fatalf("该往下掉: %+v", wines[0])
	}
}

func TestDropPointKeepsXOnCircle(t *testing.T) {
	f := unit.CircleField()
	x, y := dropPoint(f, 80, wineRadius)
	if math.Abs(x-80) > 1e-6 {
		t.Fatalf("圆场不该被中心墙横向推开: got x=%v", x)
	}
	wantY := math.Sqrt((f.Extent-wineRadius)*(f.Extent-wineRadius) - 80*80)
	if math.Abs(y-wantY) > 0.5 {
		t.Fatalf("该在该 x 的圆顶: y=%v want %v", y, wantY)
	}
	if !f.OutlineContains(x, y, wineRadius) {
		t.Fatalf("场顶该在轮廓里: (%v,%v)", x, y)
	}
	cx, cy := dropPoint(f, 0, wineRadius)
	if math.Abs(cx) > 1e-6 || math.Abs(cy-(f.Extent-wineRadius)) > 0.5 {
		t.Fatalf("x=0 该砸在正顶: (%v,%v)", cx, cy)
	}
}

func TestDropPointHexTopAtX(t *testing.T) {
	f := unit.HexField()
	x, y := dropPoint(f, 80, wineRadius)
	if math.Abs(x-80) > 1e-6 {
		t.Fatalf("六边形该保持 x: got %v", x)
	}
	if !f.OutlineContains(x, y, wineRadius) || f.OutlineContains(x, y+1, wineRadius) {
		t.Fatalf("该贴着该 x 的场顶: (%v,%v)", x, y)
	}
}

func TestWineSpawnsInsetOnCircle(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &酒翁{}
	near := []unit.Snapshot{sotAt(0, 0), enemyAt(200, 40)}
	boot(a, out, near)
	ctx := unit.Context{ID: 1, Kind: KindSot, Out: out}
	a.Handle(ctx, unit.Sense{Time: smashFirst, Self: sotAt(0, 0), Nearby: near, Field: unit.CircleField()})
	wines := spawns(drain(out), KindSotWine)
	if len(wines) != 1 {
		t.Fatalf("该出第 1 坛: %d", len(wines))
	}
	f := unit.CircleField()
	if !f.OutlineContains(wines[0].X, wines[0].Y, wineRadius+wineSpawnPad) {
		t.Fatalf("圆场不该贴边生成: (%v,%v) r=%v", wines[0].X, wines[0].Y, math.Hypot(wines[0].X, wines[0].Y))
	}
	if wines[0].X*wines[0].VX+wines[0].Y*wines[0].VY > 1e-6 {
		t.Fatalf("不该带着朝场外的径向速度: %+v", wines[0])
	}
}

func TestClipOutwardKillsRadial(t *testing.T) {
	vx, vy := clipOutward(200, 180, 70, -50)
	if 200*vx+180*vy > 1e-6 {
		t.Fatalf("该去掉朝外分量: %v %v", vx, vy)
	}
}

func TestSmashIntervalsAndNoReseek(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &酒翁{}
	near := []unit.Snapshot{sotAt(0, 0), enemyAt(80, 0)}
	boot(a, out, near)
	ctx := unit.Context{ID: 1, Kind: KindSot, Out: out}
	a.Handle(ctx, unit.Sense{Time: smashFirst, Self: sotAt(0, 0), Nearby: near, Field: unit.HexField()})
	_ = drain(out)
	if len(a.drops) == 0 {
		t.Skip("这次随机只砸 1 坛，隔档测不到")
	}
	moved := []unit.Snapshot{sotAt(0, 0), enemyAt(-90, 0)}
	a.Handle(ctx, unit.Sense{Time: smashFirst + gap12 - 0.05, Self: sotAt(0, 0), Nearby: moved, Field: unit.HexField()})
	if n := len(spawns(drain(out), KindSotWine)); n != 0 {
		t.Fatalf("0.8s 前不该出第 2 坛: %d", n)
	}
	a.Handle(ctx, unit.Sense{Time: smashFirst + gap12, Self: sotAt(0, 0), Nearby: moved, Field: unit.HexField()})
	cmds := drain(out)
	wines := spawns(cmds, KindSotWine)
	if len(wines) != 1 {
		t.Fatalf("0.8s 该出第 2 坛: %d", len(wines))
	}
	if wines[0].X < 0 {
		t.Fatalf("不二次索敌，第 2 坛还该在开砸时的 x: %+v", wines[0])
	}
}

func TestFillThenSpendAt10s(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &酒翁{}
	near := []unit.Snapshot{sotAt(0, 0), enemyAt(40, 0)}
	boot(a, out, near)
	ctx := unit.Context{ID: 1, Kind: KindSot, Out: out}
	a.Handle(ctx, unit.Sense{Time: smashFirst, Self: sotAt(0, 0), Nearby: near, Field: unit.HexField()})
	_ = drain(out)
	a.Handle(ctx, unit.Sense{Time: smashFirst + 2, Self: sotAt(0, 0), Nearby: near, Field: unit.HexField()})
	_ = drain(out)
	self := sotAt(0, 0)
	a.Handle(ctx, unit.Sense{Time: 10, Self: self, Nearby: near, Field: unit.HexField()})
	cmds := drain(out)
	ms := marksOn(cmds, 1)
	if len(ms) < 1 || ms[0].Delta != fillStacks {
		t.Fatalf("0 层该先补 3: %+v", ms)
	}
	spent := 0
	for _, m := range ms {
		if m.Delta < 0 {
			spent = -m.Delta
		}
	}
	if spent < 1 || spent > 3 {
		t.Fatalf("补完 3 层该耗 1~3: %+v", ms)
	}
	if n := len(spawns(cmds, KindSotWine)); n != 1 {
		t.Fatalf("当帧先出第 1 坛: %d", n)
	}
}

func TestZeroStacksAfterFirstSmashDropsOne(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &酒翁{}
	near := []unit.Snapshot{sotAt(0, 0), enemyAt(40, 0)}
	boot(a, out, near)
	a.first = false
	a.nextSmash = 15
	a.nextFill = 999
	ctx := unit.Context{ID: 1, Kind: KindSot, Out: out}
	a.Handle(ctx, unit.Sense{Time: 15, Self: sotAt(0, 0), Nearby: near, Field: unit.HexField()})
	cmds := drain(out)
	if m := lastMark(cmds, 1); m != nil {
		t.Fatalf("0 层不该耗醉意: %+v", m)
	}
	if n := len(spawns(cmds, KindSotWine)); n != 1 {
		t.Fatalf("0 层该砸 1 瓶: %d", n)
	}
}

func TestLaterSmashScalesWithDrunk(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &酒翁{}
	near := []unit.Snapshot{sotAt(0, 0), enemyAt(40, 0)}
	boot(a, out, near)
	a.first = false
	a.nextSmash = 15
	a.nextFill = 999
	me := sotAt(0, 0)
	me.Marks = []unit.Mark{{Kind: drunkKind, Stacks: 5}}
	ctx := unit.Context{ID: 1, Kind: KindSot, Out: out}
	a.Handle(ctx, unit.Sense{Time: 15, Self: me, Nearby: near, Field: unit.HexField()})
	cmds := drain(out)
	m := lastMark(cmds, 1)
	if m == nil || m.Delta >= 0 || -m.Delta > 5 || -m.Delta < 1 {
		t.Fatalf("该耗 1~5 层: %+v", m)
	}
	got := len(spawns(cmds, KindSotWine)) + len(a.drops)
	if got != -m.Delta {
		t.Fatalf("砸的坛数该等于消耗: smash=%d spend=%d", got, -m.Delta)
	}
}

func TestWineHitsEnemyAndLeavesStain(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	w := &酒{owner: 1, slot: 0, booted: true, x: 10, y: 20}
	ctx := unit.Context{ID: 9, Kind: KindSotWine, Out: out}
	w.Handle(ctx, unit.Collision{Other: enemyAt(10, 20)})
	cmds := drain(out)
	ds := damages(cmds)
	if len(ds) != 1 || ds[0].To != 2 || ds[0].Amount != wineDmg {
		t.Fatalf("碰敌该出伤: %+v", ds)
	}
	st := lastSpawn(cmds, KindSotStain)
	if st == nil || math.Abs(st.X-10) > 1e-9 || math.Abs(st.Y-20) > 1e-9 {
		t.Fatalf("该在原地留场: %+v", st)
	}
}

func TestWineHitsBulletLeavesStain(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	w := &酒{owner: 1, slot: 0, booted: true, x: 8, y: -4}
	ctx := unit.Context{ID: 9, Kind: KindSotWine, Out: out}
	bullet := unit.Snapshot{ID: 7, Kind: "原型机_远程子弹", Role: unit.RoleProjectile, Slot: 1, X: 8, Y: -4, Radius: 6}
	w.Handle(ctx, unit.Collision{Other: bullet})
	cmds := drain(out)
	if len(damages(cmds)) != 0 {
		t.Fatalf("抵消子弹不该出伤: %+v", damages(cmds))
	}
	st := lastSpawn(cmds, KindSotStain)
	if st == nil || math.Abs(st.X-8) > 1e-9 || math.Abs(st.Y+4) > 1e-9 {
		t.Fatalf("撞子弹该留场: %+v", st)
	}
}

func TestWineHitsMinion(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	w := &酒{owner: 1, slot: 0, booted: true, x: 0, y: 0}
	ctx := unit.Context{ID: 9, Kind: KindSotWine, Out: out}
	minion := unit.Snapshot{ID: 5, Kind: "教父暗杀者", Role: unit.RoleMinion, Slot: 1, Mortal: true, X: 0, Y: 0, Radius: 14}
	w.Handle(ctx, unit.Collision{Other: minion})
	if ds := damages(drain(out)); len(ds) != 1 || ds[0].To != 5 {
		t.Fatalf("活随从也该挨打: %+v", ds)
	}
}

func TestWinePassesAlly(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	w := &酒{owner: 1, slot: 0, booted: true, x: 0, y: 0}
	ctx := unit.Context{ID: 9, Kind: KindSotWine, Out: out}
	w.Handle(ctx, unit.Collision{Other: sotAt(0, 0)})
	cmds := drain(out)
	if lastSpawn(cmds, KindSotStain) != nil || len(damages(cmds)) != 0 {
		t.Fatalf("穿过自己人不该碎: %v", cmds)
	}
}

func TestWineShattersOnWall(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	w := &酒{owner: 1, slot: 0}
	ctx := unit.Context{ID: 9, Kind: KindSotWine, Out: out}
	w.Handle(ctx, unit.WallHit{Kind: unit.WallHard, X: 12, Y: -8})
	st := lastSpawn(drain(out), KindSotStain)
	if st == nil || math.Abs(st.X-12) > 1e-9 || math.Abs(st.Y+8) > 1e-9 {
		t.Fatalf("撞硬墙该留场: %+v", st)
	}
}

func TestStainStacksOnEnterOnly(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	f := &酒场{owner: 1, slot: 0}
	ctx := unit.Context{ID: 8, Kind: KindSotStain, Out: out}
	self := unit.Snapshot{ID: 8, Kind: KindSotStain, Role: unit.RoleHelper, Slot: 0, X: 0, Y: 0, Radius: stainRadius}
	enemy := enemyAt(10, 0)
	near := []unit.Snapshot{sotAt(-200, 0), enemy}
	f.Handle(ctx, unit.Sense{Time: 1, Self: self, Nearby: near})
	if m := lastMark(drain(out), 2); m == nil || m.Delta != 1 || m.Kind != drunkKind {
		t.Fatalf("踏进该叠 1 层: %+v", m)
	}
	f.Handle(ctx, unit.Sense{Time: 1.1, Self: self, Nearby: near})
	if m := lastMark(drain(out), 2); m != nil {
		t.Fatalf("站着不该再叠: %+v", m)
	}
	far := enemyAt(200, 0)
	f.Handle(ctx, unit.Sense{Time: 1.2, Self: self, Nearby: []unit.Snapshot{sotAt(-200, 0), far}})
	_ = drain(out)
	f.Handle(ctx, unit.Sense{Time: 1.3, Self: self, Nearby: near})
	if m := lastMark(drain(out), 2); m == nil || m.Delta != 1 {
		t.Fatalf("离开再进该再叠: %+v", m)
	}
}

func TestOwnerAlsoGetsDrunk(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	f := &酒场{owner: 1, slot: 0}
	ctx := unit.Context{ID: 8, Kind: KindSotStain, Out: out}
	self := unit.Snapshot{ID: 8, Kind: KindSotStain, Role: unit.RoleHelper, Slot: 0, X: 0, Y: 0, Radius: stainRadius}
	me := sotAt(5, 0)
	f.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{me}})
	if m := lastMark(drain(out), 1); m == nil || m.Delta != 1 {
		t.Fatalf("自己踩场也该叠: %+v", m)
	}
}

func TestDrunkSlowsEnemyNotSelf(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &酒翁{}
	enemy := enemyAt(40, 0)
	enemy.Marks = []unit.Mark{{Kind: drunkKind, Stacks: 2}}
	me := sotAt(0, 0)
	me.Marks = []unit.Mark{{Kind: drunkKind, Stacks: 4}}
	near := []unit.Snapshot{me, enemy}
	boot(a, out, near)
	ctx := unit.Context{ID: 1, Kind: KindSot, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0.2, Self: me, Nearby: near, Field: unit.HexField()})
	cmds := drain(out)
	fs := lastFS(cmds, 2)
	if fs == nil || fs.Zone != unit.FSZoneM || math.Abs(fs.Value-0.64) > 1e-9 {
		t.Fatalf("2 层该 ×0.64: %+v", fs)
	}
	if lastFS(cmds, 1) != nil {
		t.Fatalf("自己不该被减速")
	}
}

func TestEnemyDamageKeepsDrunk(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &酒翁{}
	enemy := enemyAt(40, 0)
	enemy.HP, enemy.MaxHP = 100, 100
	enemy.Marks = []unit.Mark{{Kind: drunkKind, Stacks: 7}}
	near := []unit.Snapshot{sotAt(0, 0), enemy}
	boot(a, out, near)
	enemy.HP = 90
	ctx := unit.Context{ID: 1, Kind: KindSot, Out: out}
	a.Handle(ctx, unit.Sense{Time: 0.2, Self: sotAt(0, 0), Nearby: []unit.Snapshot{sotAt(0, 0), enemy}, Field: unit.HexField()})
	cmds := drain(out)
	if m := lastMark(cmds, 2); m != nil {
		t.Fatalf("挨打不该减层: %+v", m)
	}
	fs := lastFS(cmds, 2)
	want := math.Pow(drunkMul, 7)
	if fs == nil || math.Abs(fs.Value-want) > 1e-9 {
		t.Fatalf("7 层该仍按 ×0.8^7: %+v want %v", fs, want)
	}
}

func TestWineFallsWithGravity(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	w := &酒{owner: 1, slot: 0}
	ctx := unit.Context{ID: 9, Kind: KindSotWine, Out: out}
	self := unit.Snapshot{ID: 9, Kind: KindSotWine, Role: unit.RoleProjectile, Slot: 0, X: 0, Y: 100, Radius: wineRadius, VY: -50}
	w.Handle(ctx, unit.Sense{Time: 1, Self: self})
	cmds := drain(out)
	var g *unit.Force
	for _, c := range cmds {
		if v, ok := c.(unit.Force); ok {
			x := v
			g = &x
		}
	}
	if g == nil || g.AY >= 0 {
		t.Fatalf("该往下加速: %+v", g)
	}
}
