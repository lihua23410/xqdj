package 静电魔球

import (
	"math"
	"testing"

	"xqdj/internal/unit"
)

func mk(id uint64, slot int, role string, x, y float64) unit.Snapshot {
	return unit.Snapshot{ID: id, Slot: slot, Role: role, X: x, Y: y}
}

// 走一遍完整链条，返回（各节点 id 序列、命中敌人的伤害序列）。
func runChain(self unit.Snapshot, cands []unit.Snapshot) ([]uint64, []float64) {
	var path []uint64
	var dmg []float64
	from := self
	for hits, hops := 0, 0; hits < chainHits && hops < hopCap; hops++ {
		node, ok := nextHop(from, self, cands, jumpRange)
		if !ok {
			break
		}
		path = append(path, node.ID)
		if node.ID != self.ID {
			dmg = append(dmg, chainDamage[hits])
			hits++
		}
		from = node
	}
	return path, dmg
}

// 单人局：唯一的敌人在射程内，链条把自己当中继点，四跳全部落在对手身上。
func TestChainSoloRelay(t *testing.T) {
	self := mk(1, 0, unit.RoleFighter, 0, 0)
	foe := mk(2, 1, unit.RoleFighter, 150, 0)
	path, dmg := runChain(self, []unit.Snapshot{foe})
	wantPath := []uint64{2, 1, 2, 1, 2, 1, 2}
	wantDmg := chainDamage[:]
	if len(path) != len(wantPath) {
		t.Fatalf("路径 %v，想要 %v", path, wantPath)
	}
	for i := range wantPath {
		if path[i] != wantPath[i] {
			t.Fatalf("路径 %v，想要 %v", path, wantPath)
		}
	}
	for i := range wantDmg {
		if dmg[i] != wantDmg[i] {
			t.Fatalf("伤害 %v，想要 %v", dmg, wantDmg)
		}
	}
}

// 双敌人：链条在两人之间分散，不经过自己。
func TestChainSpreadsOverTwoFoes(t *testing.T) {
	self := mk(1, 0, unit.RoleFighter, 0, 0)
	e1 := mk(2, 1, unit.RoleFighter, 100, 0)
	e2 := mk(3, 1, unit.RoleFighter, 140, 30)
	path, dmg := runChain(self, []unit.Snapshot{e1, e2})
	if len(dmg) != chainHits {
		t.Fatalf("命中 %d 跳，想要 %d", len(dmg), chainHits)
	}
	seen := map[uint64]bool{}
	for _, id := range path {
		seen[id] = true
	}
	if !seen[2] || !seen[3] {
		t.Fatalf("两只敌人都该挨电，路径 %v", path)
	}
	for _, id := range path {
		if id == 1 {
			t.Fatalf("有第二个目标时不该中继自己，路径 %v", path)
		}
	}
	// 衰减次序仍然是从大到小
	for i := 1; i < len(dmg); i++ {
		if dmg[i] >= dmg[i-1] {
			t.Fatalf("伤害没衰减：%v", dmg)
		}
	}
}

// 敌人在射程外：链条压根不该开始。
func TestChainOutOfRange(t *testing.T) {
	self := mk(1, 0, unit.RoleFighter, 0, 0)
	foe := mk(2, 1, unit.RoleFighter, jumpRange+10, 0)
	if _, ok := nextHop(self, self, []unit.Snapshot{foe}, jumpRange); ok {
		t.Fatal("射程外不该有下一跳")
	}
}

// 自己在当前节点时不回退自己；当前节点不被当成候选。
func TestChainNoSelfLoop(t *testing.T) {
	self := mk(1, 0, unit.RoleFighter, 0, 0)
	if _, ok := nextHop(self, self, nil, jumpRange); ok {
		t.Fatal("没有候选时从自己出发不该有下一跳")
	}
	foe := mk(2, 1, unit.RoleFighter, 50, 0)
	if node, ok := nextHop(foe, self, []unit.Snapshot{foe}, jumpRange); !ok || node.ID != self.ID {
		t.Fatalf("孤敌应该回退自己中继，拿到 %v ok=%v", node, ok)
	}
}

// 中继跳的距离也要在射程内才回退。
func TestRelayRespectsRange(t *testing.T) {
	self := mk(1, 0, unit.RoleFighter, 0, 0)
	far := mk(2, 1, unit.RoleFighter, jumpRange+5, 0)
	if _, ok := nextHop(far, self, []unit.Snapshot{far}, jumpRange); ok {
		t.Fatal("中继点出了射程就不该回退")
	}
}

// 链条优先打战斗机：身边凑着一堆活随从时也不上杂鱼的当。
func TestChainPrefersFighterOverCloserMinion(t *testing.T) {
	self := mk(1, 0, unit.RoleFighter, 0, 0)
	minion := unit.Snapshot{ID: 2, Slot: 1, Role: unit.RoleMinion, Mortal: true, X: 60, Y: 0}
	foe := mk(3, 1, unit.RoleFighter, 200, 0)
	path, _ := runChain(self, []unit.Snapshot{minion, foe})
	if len(path) == 0 || path[0] != 3 {
		t.Fatalf("第一跳该越过贴脸随从去打战斗机，路径 %v", path)
	}
}

// hittables 的过滤：同槽友军和拟弹不进候选，敌方活随进候选。
func TestHittablesFilter(t *testing.T) {
	self := mk(1, 0, unit.RoleFighter, 0, 0)
	s := unit.Sense{
		Self: self,
		Nearby: []unit.Snapshot{
			mk(2, 0, unit.RoleFighter, 10, 0),                                  // 同槽友军
			mk(3, 1, unit.RoleFighter, 20, 0),                                  // 敌方战斗机
			mk(4, 1, unit.RoleMinion, 30, 0),                                   // 敌方随从，不 Mortal
			{ID: 5, Slot: 1, Role: unit.RoleMinion, Mortal: true, X: 40, Y: 0}, // 敌方活随从
		},
	}
	got := hittables(s)
	if len(got) != 2 || got[0].ID != 3 || got[1].ID != 5 {
		t.Fatalf("候选应是 [3 5]，拿到 %v", got)
	}
}

// 放电触发距离走的是最近候选，不是索敌目标。
func TestNearestHittable(t *testing.T) {
	self := mk(1, 0, unit.RoleFighter, 0, 0)
	s := unit.Sense{
		Self: self,
		Nearby: []unit.Snapshot{
			mk(2, 1, unit.RoleFighter, 200, 0),
			{ID: 3, Slot: 1, Role: unit.RoleMinion, Mortal: true, X: 80, Y: 0},
		},
	}
	best := nearestHittable(s)
	if best == nil || best.ID != 3 {
		t.Fatalf("最近候选应是 3，拿到 %+v", best)
	}
	if d := dist(self, *best); math.Abs(d-80) > 1e-9 {
		t.Fatalf("距离应是 80，拿到 %v", d)
	}
}
