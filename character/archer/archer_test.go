package 弓箭

import (
	"math"
	"testing"

	"xqdj/internal/unit"
	_ "xqdj/map/hex" // 注册 hex 轮廓：安全向算法要 Clamp/Walkable，没注册会 panic
)

// feed 喂一次事件，收下角色发出的指令。
func feed(a unit.Actor, id uint64, ev unit.Event) []unit.Cmd {
	out := make(chan unit.Cmd, 64)
	a.Handle(unit.Context{ID: id, Kind: KindArcher, Out: out}, ev)
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

// addFSDir 从指令里取那条滑步 FS 的方向。
func addFSDir(cmds []unit.Cmd) (float64, float64, bool) {
	for _, c := range cmds {
		if fs, ok := c.(unit.AddFS); ok {
			return fs.DX, fs.DY, true
		}
	}
	return 0, 0, false
}

func archerSelf() unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindArcher, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: archerRadius, AimPriority: unit.DefaultFighterAim,
	}
}

func foeAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 2, Kind: "靶子", Role: unit.RoleFighter, Slot: 1,
		X: x, Y: y, Radius: 18, AimPriority: unit.DefaultFighterAim,
	}
}

// 弹：形态就是敌方弹（RoleProjectile、没有弧）。
func bulletAt(x, y, vx, vy float64) unit.Snapshot {
	return unit.Snapshot{ID: 3, Kind: "弹", Role: unit.RoleProjectile, Slot: 1, X: x, Y: y, VX: vx, VY: vy, Radius: 5}
}

// 蓄力中贴脸 → 收弓：cancel FX 必须带自己的坐标。
// 不带 X/Y 时前端拿 ctx.x/ctx.y 画特效会按 (0,0) 画，于是场心堆一圈圈环（线上踩过这个坑）。
func TestCancelFxCarriesPosition(t *testing.T) {
	f := &弓箭{step: stepCharge, until: 9.0, booted: true, slot: 0}
	self := archerSelf()
	self.X, self.Y = 33, -44
	foe := foeAt(33, 6) // 距离 50 < 90：贴脸，触发取消
	cmds := feed(f, 1, unit.Sense{Time: 1.0, Self: self, Nearby: []unit.Snapshot{foe}})
	found := false
	for _, c := range cmds {
		fx, ok := c.(unit.FX)
		if !ok || fx.Name != "cancel" {
			continue
		}
		found = true
		if fx.X != 33 || fx.Y != -44 {
			t.Fatalf("cancel FX 坐标不对：(%v,%v)，想要 (33,-44)（不带就会堆在场心）", fx.X, fx.Y)
		}
	}
	if !found {
		t.Fatalf("贴脸蓄力没收弓：%v", cmds)
	}
}

// 贴脸 → 后撤步，方向必须是"安全向"：先看敌距分（往远处撤）。
func TestSafeDirRunsAwayFromEnemy(t *testing.T) {
	f := &弓箭{}
	self := archerSelf()
	foe := foeAt(0, 70) // 距离 70 < 90：贴脸
	cmds := feed(f, 1, unit.Sense{Time: 1.0, Self: self, Nearby: []unit.Snapshot{foe}})
	ux, uy, ok := addFSDir(cmds)
	if !ok {
		t.Fatalf("贴脸没后撤：%v", cmds)
	}
	if uy > -0.7 {
		t.Fatalf("没往远离敌人的方向撤：(%v,%v)，敌人在 +y", ux, uy)
	}
}

// 弹道分：敌人那边当然不能去，另一侧压着一条弹道走廊也不能落进去——落点必须在走廊外。
// （等权打分下它可能选"侧后方"而不是正后方：只要同时躲开两条威胁就符合设计稿。）
func TestSafeDirAvoidsBallisticCorridor(t *testing.T) {
	f := &弓箭{}
	self := archerSelf()
	foe := foeAt(0, 70)
	shot := bulletAt(-200, -70, 400, 0) // 弹从 -x 侧朝 +x 平飞，弹道压在 -y 方向
	cmds := feed(f, 1, unit.Sense{Time: 1.0, Self: self, Nearby: []unit.Snapshot{foe, shot}})
	ux, uy, ok := addFSDir(cmds)
	if !ok {
		t.Fatalf("没后撤：%v", cmds)
	}
	lx, ly := self.X+ux*retreatDist, self.Y+uy*retreatDist
	corridor := shot.Radius + archerRadius + threatBuffer
	if d := rayDist(lx, ly, shot.X, shot.Y, shot.VX, shot.VY); d < corridor {
		t.Fatalf("撤进了弹道走廊：落点离弹线 %v < %v（方向 %v,%v）", d, corridor, ux, uy)
	}
	if d := math.Hypot(lx-foe.X, ly-foe.Y); d < 70 {
		t.Fatalf("撤得更贴近敌人了：%v < 70（方向 %v,%v）", d, ux, uy)
	}
}

// 场地分：撤出去会出场、被夹回来的方向要吃亏——贴边时不该再往边角撤。
func TestSafeDirKeepsInsideField(t *testing.T) {
	f := &弓箭{}
	self := archerSelf()
	self.Y = 220 // 离上边界只差一点（六边形内切半径 242.5）
	foe := foeAt(0, 170)
	cmds := feed(f, 1, unit.Sense{Time: 1.0, Self: self, Nearby: []unit.Snapshot{foe}})
	ux, uy, ok := addFSDir(cmds)
	if !ok {
		t.Fatalf("贴脸没后撤：%v", cmds)
	}
	cx, cy := unit.HexField().Clamp(self.X+ux*retreatDist, self.Y+uy*retreatDist, archerRadius)
	if loss := math.Hypot(cx-(self.X+ux*retreatDist), cy-(self.Y+uy*retreatDist)); loss > 20 {
		t.Fatalf("撤进了死角（被夹回 %v）：(%v,%v)", loss, ux, uy)
	}
}

func hasSpawn(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if _, ok := c.(unit.Spawn); ok {
			return true
		}
	}
	return false
}

func hasFX(cmds []unit.Cmd, name string) bool {
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == name {
			return true
		}
	}
	return false
}

// 没锁到人：不起手、不出箭。
func TestIdleWithoutLockDoesNotShoot(t *testing.T) {
	f := &弓箭{}
	cmds := feed(f, 1, unit.Sense{Time: 1.0, Self: archerSelf()})
	if hasSpawn(cmds) || hasFX(cmds, "shoot") || hasFX(cmds, "volley") || hasFX(cmds, "draw") {
		t.Fatalf("没锁还放了：%v", cmds)
	}
}

// 蓄力到点时锁丢了：收弓，这一箭哑火。
func TestChargeLostLockDoesNotShoot(t *testing.T) {
	f := &弓箭{step: stepCharge, until: 1.0, booted: true, slot: 0}
	cmds := feed(f, 1, unit.Sense{Time: 1.0, Self: archerSelf()})
	if hasSpawn(cmds) || hasFX(cmds, "shoot") || hasFX(cmds, "volley") {
		t.Fatalf("丢锁还出箭：%v", cmds)
	}
	if !hasFX(cmds, "cancel") {
		t.Fatalf("丢锁没收弓：%v", cmds)
	}
	if f.step != stepIdle {
		t.Fatalf("step=%d want idle", f.step)
	}
}

// 齐射站定到点时锁丢了：同样不出箭。
func TestVolleyLostLockDoesNotShoot(t *testing.T) {
	f := &弓箭{step: stepVolley, until: 1.0, booted: true, slot: 0}
	cmds := feed(f, 1, unit.Sense{Time: 1.0, Self: archerSelf()})
	if hasSpawn(cmds) || hasFX(cmds, "shoot") || hasFX(cmds, "volley") {
		t.Fatalf("丢锁还齐射：%v", cmds)
	}
	if !hasFX(cmds, "cancel") {
		t.Fatalf("丢锁没收弓：%v", cmds)
	}
}
