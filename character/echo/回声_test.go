package 回声

import (
	"testing"

	"xqdj/internal/unit"

	// 场地外形判定由 map 包在 init 里登记；这里要用 Field.Clamp，所以得拉进来。
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

func countSpawns(cmds []unit.Cmd, kind string) int {
	n := 0
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == kind {
			n++
		}
	}
	return n
}

func firstSpawn(cmds []unit.Cmd, kind string) (unit.Spawn, bool) {
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == kind {
			return s, true
		}
	}
	return unit.Spawn{}, false
}

func hasDespawn(cmds []unit.Cmd, id uint64) bool {
	for _, c := range cmds {
		if d, ok := c.(unit.Despawn); ok && d.UnitID == id {
			return true
		}
	}
	return false
}

func stackOf(cmds []unit.Cmd, id uint64, kind string) (unit.StackMark, bool) {
	for _, c := range cmds {
		if s, ok := c.(unit.StackMark); ok && s.UnitID == id && s.Kind == kind && s.Delta > 0 {
			return s, true
		}
	}
	return unit.StackMark{}, false
}

func damageOf(cmds []unit.Cmd) (unit.Damage, bool) {
	for _, c := range cmds {
		if d, ok := c.(unit.Damage); ok {
			return d, true
		}
	}
	return unit.Damage{}, false
}

func fxOf(cmds []unit.Cmd, name string) (unit.FX, bool) {
	for _, c := range cmds {
		if f, ok := c.(unit.FX); ok && f.Name == name {
			return f, true
		}
	}
	return unit.FX{}, false
}

func echoSelf(vx, x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindEcho, Role: unit.RoleFighter, Slot: 0,
		X: x, Y: y, VX: vx, Radius: echoRadius,
		HP: echoHP, MaxHP: echoHP,
	}
}

func echoFoe(id uint64, stacks int, kind string) unit.Snapshot {
	o := unit.Snapshot{
		ID: id, Kind: "靶子", Role: unit.RoleFighter, Slot: 1,
		X: 50, Y: 0, Radius: 18,
	}
	if stacks > 0 {
		o.Marks = []unit.Mark{{Kind: kind, Stacks: stacks}}
	}
	return o
}

func TestFiresOnlyWhileMoving(t *testing.T) {
	out := make(chan unit.Cmd, 128)
	ctx := unit.Context{ID: 1, Kind: KindEcho, Out: out, Stream: 7}.WithBirths()
	e := &回声{}
	send := func(tm, vx float64) {
		e.Handle(ctx, unit.Sense{
			Time: tm, Self: echoSelf(vx, 0, 0), Field: unit.HexField(),
		})
	}

	for i := 0; i <= 30; i++ {
		send(float64(i)*0.1, 0)
	}
	if n := countSpawns(drain(out), KindStar); n != 0 {
		t.Fatalf("站着放了 %d 枚星弹，应为 0", n)
	}

	for i := 31; i <= 51; i++ {
		send(float64(i)*0.1, 120)
	}
	cmds := drain(out)
	if n := countSpawns(cmds, KindStar); n != 1 {
		t.Fatalf("走了 2 秒放了 %d 枚星弹，应为 1", n)
	}
	star, _ := firstSpawn(cmds, KindStar)
	if star.VX <= 0 || star.VY != 0 {
		t.Fatalf("星弹速度 = (%v,%v)，应朝 +X", star.VX, star.VY)
	}
	if star.OwnerID != 1 || star.Slot != 0 {
		t.Fatalf("星弹 owner/slot = %d/%d，应为 1/0", star.OwnerID, star.Slot)
	}

	for i := 52; i <= 75; i++ {
		send(float64(i)*0.1, 0)
	}
	if n := countSpawns(drain(out), KindStar); n != 0 {
		t.Fatalf("停下后又放了 %d 枚星弹，应为 0", n)
	}
}

func TestStarBurstsIntoSixteenOnWall(t *testing.T) {
	out := make(chan unit.Cmd, 128)
	ctx := unit.Context{ID: 2, Kind: KindStar, Out: out, Stream: 1}.WithBirths()
	a := &星弹{owner: 7, slot: 0}

	a.Handle(ctx, unit.WallHit{Time: 0.1, X: 20, Y: 0, NX: -1, NY: 0, Kind: unit.WallEdge})
	cmds := drain(out)
	if n := countSpawns(cmds, KindShard); n != shardN {
		t.Fatalf("炸出 %d 枚碎片，应为 %d", n, shardN)
	}
	for _, c := range cmds {
		s, ok := c.(unit.Spawn)
		if !ok || s.Kind != KindShard {
			continue
		}
		if s.OwnerID != 7 || s.Slot != 0 {
			t.Fatalf("碎片 owner/slot = %d/%d，应为 7/0", s.OwnerID, s.Slot)
		}
	}
	if _, ok := fxOf(cmds, "echo-burst"); !ok {
		t.Fatal("炸开时应有声呐爆散")
	}
	if !hasDespawn(cmds, 2) {
		t.Fatal("星弹炸完应消失")
	}

	// 一枚星弹只炸一次。
	a.Handle(ctx, unit.WallHit{Time: 0.2, X: 20, Y: 0, NX: -1, NY: 0, Kind: unit.WallEdge})
	if n := countSpawns(drain(out), KindShard); n != 0 {
		t.Fatalf("第二次撞击又炸出 %d 枚碎片", n)
	}
}

func TestStarBurstsOnUnit(t *testing.T) {
	out := make(chan unit.Cmd, 128)
	ctx := unit.Context{ID: 2, Kind: KindStar, Out: out, Stream: 1}.WithBirths()
	a := &星弹{owner: 7, slot: 0}

	a.Handle(ctx, unit.Collision{Time: 0.1, Other: echoFoe(9, 0, ""), NX: 1, NY: 0})
	cmds := drain(out)
	if n := countSpawns(cmds, KindShard); n != shardN {
		t.Fatalf("撞人炸出 %d 枚碎片，应为 %d", n, shardN)
	}
	if !hasDespawn(cmds, 2) {
		t.Fatal("星弹撞人后应消失")
	}
}

func TestShardBouncesEightThenDies(t *testing.T) {
	out := make(chan unit.Cmd, 128)
	ctx := unit.Context{ID: 3, Kind: KindShard, Out: out}
	s := &碎片{slot: 0, markKind: countKind(1)}

	hit := func(tm float64) {
		s.Handle(ctx, unit.WallHit{Time: tm, X: 0, Y: 0, NX: -1, NY: 0, Kind: unit.WallEdge})
	}
	for i := 0; i < shardBounces; i++ {
		hit(float64(i) * 0.1)
	}
	if hasDespawn(drain(out), 3) {
		t.Fatalf("弹了 %d 次就消失了", shardBounces)
	}
	hit(0.9)
	if !hasDespawn(drain(out), 3) {
		t.Fatal("第 9 次碰墙还没消失")
	}
}

func TestShardCountsButDoesNotDamageBeforeTen(t *testing.T) {
	out := make(chan unit.Cmd, 128)
	ctx := unit.Context{ID: 3, Kind: KindShard, Out: out}
	kind := countKind(1)
	s := &碎片{slot: 0, markKind: kind}

	s.Handle(ctx, unit.Collision{Time: 1, Other: echoFoe(9, 0, kind)})
	cmds := drain(out)
	mark, ok := stackOf(cmds, 9, kind)
	if !ok {
		t.Fatal("第一次命中应记一层扫描")
	}
	if mark.Lifetime != countLife {
		t.Fatalf("扫描寿命 = %v，应为每层 %v 秒", mark.Lifetime, countLife)
	}
	if mark.Icon != countIcon {
		t.Fatalf("扫描图标 = %q", mark.Icon)
	}
	if _, ok := damageOf(cmds); ok {
		t.Fatal("不到 10 层不该掉血")
	}
	if ping, ok := fxOf(cmds, "echo-ping"); !ok || ping.Amount != 1 {
		t.Fatalf("命中应有声呐回落：%+v %v", ping, ok)
	}
	if !hasDespawn(cmds, 3) {
		t.Fatal("碎片命中后应消失")
	}
}

func TestShardDamagesAtTen(t *testing.T) {
	out := make(chan unit.Cmd, 128)
	ctx := unit.Context{ID: 3, Kind: KindShard, Out: out}
	kind := countKind(1)
	s := &碎片{slot: 0, markKind: kind}

	s.Handle(ctx, unit.Collision{Time: 1, Other: echoFoe(9, 9, kind)})
	cmds := drain(out)
	if _, ok := stackOf(cmds, 9, kind); !ok {
		t.Fatal("第 10 枚应把层数补到 10")
	}
	d, ok := damageOf(cmds)
	if !ok {
		t.Fatal("第 10 枚命中应造成伤害")
	}
	if d.Amount != shardDmg || d.To != 9 || !d.NoFreeze {
		t.Fatalf("伤害 = %+v，应为 %v / 9 / 不停帧", d, shardDmg)
	}
	if ping, ok := fxOf(cmds, "echo-ping"); !ok || ping.Amount != 10 {
		t.Fatalf("锁定那一下的声呐应带 10：%+v %v", ping, ok)
	}
}

func TestShardAtTenAddsFreshStackAndDamages(t *testing.T) {
	out := make(chan unit.Cmd, 128)
	ctx := unit.Context{ID: 3, Kind: KindShard, Out: out}
	kind := countKind(1)
	s := &碎片{slot: 0, markKind: kind}

	s.Handle(ctx, unit.Collision{Time: 1, Other: echoFoe(9, 10, kind)})
	cmds := drain(out)
	if _, ok := stackOf(cmds, 9, kind); !ok {
		t.Fatal("已经锁定时也要补新层，旧层靠寿命自己掉")
	}
	if _, ok := damageOf(cmds); !ok {
		t.Fatal("已经锁定时应继续造成伤害")
	}
}
