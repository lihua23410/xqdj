package unit

import "testing"

// 布局：self 在原点；A 高优先度在 (100,0)；B 低优先度在 (60,80)。
// 挡视线墙竖在 x=50，只压 self→A 的线段，不压 self→B。
// Seek 默认查视线：带挡视线标签的实体压住连线段的目标跳过（los.go）。

func losSense(walls []WallView, extra ...Snapshot) Sense {
	nearby := []Snapshot{
		{ID: 2, Slot: 1, X: 100, Y: 0, AimPriority: 5},
		{ID: 3, Slot: 1, X: 60, Y: 80, AimPriority: 60},
	}
	nearby = append(nearby, extra...)
	return Sense{
		Self:   Snapshot{ID: 1, Slot: 0, X: 0, Y: 0},
		Nearby: nearby,
		Walls:  walls,
	}
}

func losBlockingWall() WallView {
	return WallView{ID: 9, X1: 50, Y1: -50, X2: 50, Y2: 50, Radius: 6, VisionBlock: true}
}

func TestSeekSkipsBehindVisionBlockWall(t *testing.T) {
	s := losSense([]WallView{losBlockingWall()})
	got := Seek(s)
	if got == nil || got.ID != 3 {
		t.Fatalf("got %#v, want visible low-priority ID 3", got)
	}
}

func TestSeekReturnsNilWhenAllBlocked(t *testing.T) {
	s := Sense{
		Self:   Snapshot{ID: 1, Slot: 0, X: 0, Y: 0},
		Nearby: []Snapshot{{ID: 2, Slot: 1, X: 100, Y: 0, AimPriority: 5}},
		Walls:  []WallView{losBlockingWall()},
	}
	if got := Seek(s); got != nil {
		t.Fatalf("got %#v, want nil when the only target is blocked", got)
	}
}

func TestSeekIgnoresWallWithoutTag(t *testing.T) {
	w := losBlockingWall()
	w.VisionBlock = false
	got := Seek(losSense([]WallView{w}))
	if got == nil || got.ID != 2 {
		t.Fatalf("got %#v, want untagged wall transparent", got)
	}
}

func TestSeekTangentBlocks(t *testing.T) {
	// 线段 (0,0)→(100,0) 与轴 (50,3)→(60,3)、半径 3 恰好相切：距离 == 3。
	w := WallView{ID: 9, X1: 50, Y1: 3, X2: 60, Y2: 3, Radius: 3, VisionBlock: true}
	s := losSense([]WallView{w})
	if !sightBlocked(s, s.Nearby[0]) {
		t.Fatal("tangent wall should block")
	}
	if got := Seek(s); got == nil || got.ID != 3 {
		t.Fatalf("got %#v, want tangent to count as blocked", got)
	}
}

func TestSeekUnitBlocksSight(t *testing.T) {
	// 敌方胖子单位站在线段上：不可瞄准（优先度 0）但体积挡视线。
	fat := Snapshot{ID: 4, Slot: 1, X: 50, Y: 0, Radius: 30, VisionBlock: true}
	got := Seek(losSense(nil, fat))
	if got == nil || got.ID != 3 {
		t.Fatalf("got %#v, want fat unit to block A", got)
	}
}

func TestSeekOwnUnitAlsoBlocks(t *testing.T) {
	// 自家单位带标签：敌我不分，同样挡自家索敌。
	own := Snapshot{ID: 5, Slot: 0, OwnerID: 1, X: 50, Y: 0, Radius: 20, VisionBlock: true}
	got := Seek(losSense(nil, own))
	if got == nil || got.ID != 3 {
		t.Fatalf("got %#v, want own tagged unit to block A", got)
	}
}

func TestSeekWallAsideDoesNotBlock(t *testing.T) {
	w := WallView{ID: 9, X1: 50, Y1: 100, X2: 60, Y2: 100, Radius: 3, VisionBlock: true}
	got := Seek(losSense([]WallView{w}))
	if got == nil || got.ID != 2 {
		t.Fatalf("got %#v, want off-path wall to stay transparent", got)
	}
}

func TestSeekTargetTagDoesNotBlock(t *testing.T) {
	// 目标自己带标签：不影响对其本身的索敌。
	tagged := Snapshot{ID: 2, Slot: 1, X: 100, Y: 0, AimPriority: 5, VisionBlock: true}
	s := Sense{
		Self:   Snapshot{ID: 1, Slot: 0, X: 0, Y: 0},
		Nearby: []Snapshot{tagged},
	}
	got := Seek(s)
	if got == nil || got.ID != 2 {
		t.Fatalf("got %#v, want target's own tag ignored", got)
	}
}
