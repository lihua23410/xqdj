package sim

import (
	"math"
	"sort"
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

// startTokei 开一局 时计 vs 靶子。靶子 99999 血且不还手，正好当不动的沙包。
func startTokei(t *testing.T, seed uint64) (*Match, *unit, *unit) {
	t.Helper()
	m := NewMatchSeeded(seed)
	m.SetSlot(0, character.KindTokei)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	t.Cleanup(m.End)
	time.Sleep(20 * time.Millisecond)
	var me, foe *unit
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.role != unitpkg.RoleFighter {
			continue
		}
		if u.slot == 0 {
			me = u
		} else {
			foe = u
		}
	}
	if me == nil || foe == nil {
		t.Fatal("missing fighters")
	}
	return m, me, foe
}

// runTokei 跑 secs 秒模拟时间，返回期间最后一次 clock 读数（小时、停针/满仓标记）和 FX 计数。
// prev/prevFull 是上一段的读数：这段里没有新的 clock FX（比如停针期间什么都不变）就沿用它们。
func runTokei(m *Match, secs, prev float64, prevFull bool) (float64, bool, map[string]int) {
	hours, full := prev, prevFull
	fx := map[string]int{}
	drain := func() {
		m.mu.Lock()
		for _, f := range m.fx {
			fx[f.Name]++
			if f.Name == "clock" {
				hours, full = f.Amount, f.VX > 0
			}
		}
		m.mu.Unlock()
	}
	m.mu.Lock()
	start := m.time
	m.mu.Unlock()
	for i := 0; i < 40000; i++ {
		m.Tick()
		time.Sleep(300 * time.Microsecond)
		drain()
		m.mu.Lock()
		now := m.time
		done := m.phase != PhaseRunning // 分出胜负后 Tick 早退，模拟时间不再走，别在这儿空转
		m.mu.Unlock()
		if done || now-start >= secs {
			// 多跑几拍，让角色协程刚发的指令落进引擎
			for j := 0; j < 4; j++ {
				m.Tick()
				time.Sleep(2 * time.Millisecond)
				drain()
			}
			break
		}
	}
	return hours, full, fx
}

// momentsXY 场上属于它的时刻实体（id → 位置）。数量就是档数。
func momentsXY(m *Match, me *unit) map[uint64][2]float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[uint64][2]float64{}
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && !u.stopped && u.kind == character.KindTokeiMoment && u.owner == me.id {
			out[id] = [2]float64{u.p.X, u.p.Y}
		}
	}
	return out
}

func posOf(m *Match, u *unit) (float64, float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return u.p.X, u.p.Y
}

func velOf(m *Match, u *unit) (float64, float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return u.v.X, u.v.Y
}

func hpOf(m *Match, u *unit) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return u.hp
}

// marksOf 读单位身上的标记层数（引擎自己的账）。
func marksOf(m *Match, u *unit) map[string]int {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]int{}
	for k, v := range u.marks {
		if v != nil && v.stacks > 0 {
			out[k] = v.stacks
		}
	}
	return out
}

// hit 从靶子给时计一记伤害，跑 secs 秒让两段式结算走完。
func hit(m *Match, me, foe *unit, amount, secs, hours float64, full bool) (float64, bool, map[string]int) {
	m.mu.Lock()
	m.offerDamageLocked(unitpkg.Damage{From: foe.id, To: me.id, Amount: amount})
	m.mu.Unlock()
	return runTokei(m, secs, hours, full)
}

// freeze 把某只钉住（放弃巡航、速度清零）。测回拨/陷阱的几何时要先把两边都按住，
// 否则 170 的巡航几秒就跑出几十像素，位置断言全成噪声。
func freeze(m *Match, u *unit) {
	m.mu.Lock()
	u.stand = true
	u.setVel(vec{})
	m.applyCmdLocked(unitpkg.SetCruise{UnitID: u.id, Speed: 0})
	m.mu.Unlock()
}

// tripWatch 跑到踩影触发为止，返回那次 trip 的方向（= 朝触发者）与触发那一刻时计的位置。
// 方向要取 FX 里的，不能读冲撞之后的速度：撞上靶子会被弹回来，速度方向是反的。
func tripWatch(m *Match, me *unit, secs float64) (dirX, dirY, x, y float64, fx map[string]int, fired bool) {
	fx = map[string]int{}
	m.mu.Lock()
	start := m.time
	m.mu.Unlock()
	for i := 0; i < 4000; i++ {
		m.Tick()
		time.Sleep(300 * time.Microsecond)
		m.mu.Lock()
		for _, f := range m.fx {
			fx[f.Name]++
			if f.Name == "trip" && !fired {
				dirX, dirY, fired = f.VX, f.VY, true
				x, y = me.p.X, me.p.Y
			}
		}
		now := m.time
		m.mu.Unlock()
		if fired || now-start >= secs {
			break
		}
	}
	for j := 0; j < 4; j++ {
		m.Tick()
		time.Sleep(2 * time.Millisecond)
	}
	return dirX, dirY, x, y, fx, fired
}

func alive(m *Match, u *unit) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	got := m.units[u.id]
	return got != nil && !got.stopped
}

// 8s 节拍：开场先等满一次，到点刻一档（时针 +2h），并 Spawn 一个钉住不动的透明时刻实体。
func TestTokeiSavesOneMomentPerBeat(t *testing.T) {
	m, me, _ := startTokei(t, 11)
	hours, full, _ := runTokei(m, 5.0, 0, false)
	if n := len(momentsXY(m, me)); n != 0 {
		t.Fatalf("5s 就刻档了：%d 个", n)
	}
	if hours != 0 {
		t.Fatalf("5s 表针 %v，应该还在 0:00", hours)
	}

	// 先跑到 7.9s 记住自己的位置，再跨过 8s 节拍：档该钉在刻档那一瞬的位置上
	hours, full, _ = runTokei(m, 2.9, hours, full)
	bx, by := posOf(m, me)
	hours, full, _ = runTokei(m, 0.4, hours, full)
	pins := momentsXY(m, me)
	if len(pins) != 1 {
		t.Fatalf("8.3s 档数=%d，想要 1", len(pins))
	}
	if hours != 2 {
		t.Fatalf("8.3s 表针 %v，想要 2:00（先推时间再留时刻）", hours)
	}
	if full {
		t.Fatal("一档就报满仓")
	}
	var pinID uint64
	var pinX, pinY float64
	for id, p := range pins {
		pinID, pinX, pinY = id, p[0], p[1]
	}
	if d := math.Hypot(pinX-bx, pinY-by); d > 25 {
		t.Fatalf("时刻没钉在刻档位置：偏移 %v", d)
	}
	// 钉住 = 之后再也不动（它不实心、不撞人、不可瞄准）
	m.mu.Lock()
	nonsolid := !m.units[pinID].solid
	m.mu.Unlock()
	if !nonsolid {
		t.Fatal("时刻实体必须不实心（Nonsolid）：可穿过、不撞人、不进 Hittable")
	}
	spec, _ := unitpkg.Lookup(character.KindTokeiMoment)
	if spec.Role != unitpkg.RoleHelper || spec.AimPriority != 0 || spec.VisionBlock {
		t.Fatalf("时刻实体规格不对：role=%s aim=%d visionBlock=%v", spec.Role, spec.AimPriority, spec.VisionBlock)
	}
	if spec.Radius != 22 {
		t.Fatalf("判定圈=%v，想要 22", spec.Radius)
	}
	hours, full, _ = runTokei(m, 2.0, hours, full)
	now := momentsXY(m, me)
	if p, ok := now[pinID]; !ok || math.Hypot(p[0]-pinX, p[1]-pinY) > 0.01 {
		t.Fatalf("时刻实体跟着跑了：(%v,%v) → %v", pinX, pinY, now[pinID])
	}

	// 第二档：5.0 + 2.9 + 0.4 + 2.0 = 10.3s，再 6s 就到 16.3s，落在第二档后
	hours, _, _ = runTokei(m, 6.0, hours, full)
	if n := len(momentsXY(m, me)); n != 2 {
		t.Fatalf("16.3s 档数=%d，想要 2", n)
	}
	if hours != 4 {
		t.Fatalf("第二档表针 %v，想要 4:00", hours)
	}
}

// clock FX 里带的窗口起止：前端靠它在窗口内把两根针连续推到下一拍（先推时间再留时刻）。
func TestTokeiClockFxCarriesBeatWindow(t *testing.T) {
	m, me, _ := startTokei(t, 25)
	freeze(m, me)
	type rec struct{ amt, x, y float64 }
	var recs []rec
	m.mu.Lock()
	start := m.time
	m.mu.Unlock()
	for i := 0; i < 4000; i++ {
		m.Tick()
		time.Sleep(300 * time.Microsecond)
		m.mu.Lock()
		for _, f := range m.fx {
			if f.Name != "clock" {
				continue
			}
			// 认领残影时会补发一次同样的读数，去重
			if n := len(recs); n > 0 && recs[n-1].amt == f.Amount && recs[n-1].x == f.X && recs[n-1].y == f.Y {
				continue
			}
			recs = append(recs, rec{f.Amount, f.X, f.Y})
		}
		now := m.time
		m.mu.Unlock()
		if now-start >= 26.0 {
			break
		}
	}
	if len(recs) < 4 {
		t.Fatalf("clock FX 太少：%d，想要 ≥4（开场 + 三拍）", len(recs))
	}
	for i, r := range recs {
		if d := r.y - r.x; math.Abs(d-8) > 1e-9 {
			t.Fatalf("第 %d 个 clock 的窗口宽 %v，想要 8", i, d)
		}
		if want := float64(i * 2); math.Abs(r.amt-want) > 1e-9 {
			t.Fatalf("第 %d 个 clock 的读数 %v，想要 %v", i, r.amt, want)
		}
		if i > 0 {
			// 下一拍的窗口从前一拍实际到点那一刻起算，允许一个帧粒度的抖动
			if gap := r.x - recs[i-1].y; gap < -1e-9 || gap > 0.05 {
				t.Fatalf("窗口没接上：%v → %v（差 %v）", recs[i-1].y, r.x, gap)
			}
		}
	}
}

// 真实情况的那一版：敌人踩到的是一个**离时计很远的旧时刻**，而且它正在往远处跑。
// 时计被拉过去之后必须追得上、把弧送到脸上（这是踩影唯一的意义）。
func TestTokeiTrapCatchesFleeingBait(t *testing.T) {
	m, me, foe := startTokei(t, 31)
	runTokei(m, 8.4, 0, false) // 刻下第一档
	pins := momentsXY(m, me)
	if len(pins) != 1 {
		t.Fatalf("档数=%d，想要 1", len(pins))
	}
	var px, py float64
	for _, p := range pins {
		px, py = p[0], p[1]
	}
	// 时计在档位另一侧 300 处；敌人放在圈外，速度朝远侧（逃离档位）
	m.mu.Lock()
	me.p = vec{px + 300, py}
	me.setVel(vec{})
	foe.p = vec{px + 70, py}
	foe.setVel(vec{165, 0})
	m.applyCmdLocked(unitpkg.SetCruise{UnitID: foe.id, Speed: 165})
	m.applyCmdLocked(unitpkg.SetFSDirection{UnitID: foe.id, VX: 1, VY: 0})
	m.mu.Unlock()
	runTokei(m, 0.3, 0, false)
	// 踩进圈（圈内 40），然后继续朝 +x 跑
	m.mu.Lock()
	foe.p = vec{px + 35, py}
	foe.setVel(vec{165, 0})
	m.mu.Unlock()
	before := hpOf(m, foe)
	_, _, fx := runTokei(m, 0.8, 0, false)
	if fx["trip"] == 0 {
		t.Fatal("踩影没触发")
	}
	if after := hpOf(m, foe); after >= before {
		t.Fatalf("敌人正在逃跑时踩影打不到：hp %v → %v（触发时距离 35，敌人 165 逃）", before, after)
	}
}

// 踩影的目的就是「把弧送到踩圈的人脸上」：触发之后必须真的打中。
// 两种摆放距离对照：40 以内都算进圈，但太近会跟人重叠（半径 18+18=36）。
func TestTokeiTrapRamActuallyHits(t *testing.T) {
	for _, dist := range []float64{30, 39} {
		m, me, foe := startTokei(t, 26+uint64(dist))
		freeze(m, foe)
		hours, full, _ := runTokei(m, 8.4, 0, false)
		pins := momentsXY(m, me)
		if len(pins) != 1 {
			t.Fatalf("档数=%d，想要 1", len(pins))
		}
		var px, py float64
		for _, p := range pins {
			px, py = p[0], p[1]
		}
		m.mu.Lock()
		me.p = vec{px + 300, py}
		foe.p = vec{px + 70, py} // 先在圈外立好，才有「进入」这个边沿
		me.setVel(vec{})
		m.mu.Unlock()
		runTokei(m, 0.3, hours, full)
		m.mu.Lock()
		foe.p = vec{px + dist, py}
		m.mu.Unlock()
		before := hpOf(m, foe)
		runTokei(m, 0.6, hours, full)
		after := hpOf(m, foe)
		if after >= before {
			t.Errorf("踩影后没打到人（摆放距离 %v，重叠阈值 36）：hp %v → %v", dist, before, after)
		}
	}
}

// 非致命伤不回拨：累计再多也不弹档（回拨扳机只剩致命伤）。
func TestTokeiNonLethalNeverRewinds(t *testing.T) {
	m, me, foe := startTokei(t, 23)
	freeze(m, me)
	freeze(m, foe)
	hours, full, _ := runTokei(m, 16.4, 0, false) // 刻满两档
	if n := len(momentsXY(m, me)); n != 2 {
		t.Fatalf("档数=%d，想要 2", n)
	}
	hours, full, _ = hit(m, me, foe, 40, 0.3, hours, full)
	if n := len(momentsXY(m, me)); n != 2 {
		t.Fatalf("非致命伤回拨了：%d 档", n)
	}
	if hp := hpOf(m, me); hp != 60 {
		t.Fatalf("血=%v，想要 60（40 点照常吃）", hp)
	}
	_ = hours
	_ = full
}

// 先推时间再留下时刻：第一档记的是 2:00（不是 0:00），第二档 4:00——残影针位就是它刻下时的时间。
func TestTokeiMomentStampsAdvanceFirst(t *testing.T) {
	m, me, _ := startTokei(t, 22)
	freeze(m, me)
	seen := map[uint64]float64{}
	m.mu.Lock()
	start := m.time
	m.mu.Unlock()
	for i := 0; i < 4000; i++ {
		m.Tick()
		time.Sleep(300 * time.Microsecond)
		m.mu.Lock()
		for _, f := range m.fx {
			if f.Name == "moment" {
				seen[f.UnitID] = f.Amount
			}
		}
		now := m.time
		m.mu.Unlock()
		if now-start >= 17.0 { // 跑到第二档之后
			break
		}
	}
	pins := momentsXY(m, me)
	if len(pins) != 2 {
		t.Fatalf("档数=%d，想要 2", len(pins))
	}
	stamps := make([]float64, 0, 2)
	for id := range pins {
		stamps = append(stamps, seen[id])
	}
	sort.Float64s(stamps)
	if len(stamps) != 2 || stamps[0] != 2 || stamps[1] != 4 {
		t.Fatalf("残影针位=%v，想要 [2 4]（先推时间再留时刻）", stamps)
	}
}

// 走到 12:00 停针；六个刻度都落下后不再刻。
func TestTokeiCapsAtSixMoments(t *testing.T) {
	m, me, _ := startTokei(t, 12)
	hours, full, _ := runTokei(m, 50.0, 0, false)
	if n := len(momentsXY(m, me)); n != 6 {
		t.Fatalf("50s 档数=%d，想要 6（封顶）", n)
	}
	if hours != 12 {
		t.Fatalf("满仓表针 %v，想要 12:00", hours)
	}
	if !full {
		t.Fatal("满仓没报标记：前端分不出 12:00 和 0:00")
	}
	hours, full, _ = runTokei(m, 12.0, hours, full) // 还在满仓，继续等也不该多刻
	if n := len(momentsXY(m, me)); n != 6 {
		t.Fatalf("满仓后又刻了：%d 档", n)
	}
	if hours != 12 || !full {
		t.Fatalf("满仓状态被改：%v / %v", hours, full)
	}
}

// 致命回拨：血/位置回档、档用掉、表针按已提交读数 −2h。
func TestTokeiRewindsOnLethalTurnsHandBack(t *testing.T) {
	m, me, foe := startTokei(t, 13)
	freeze(m, me)
	freeze(m, foe)
	// 先挨 10 点，再等 8s 刻档：档里记的是 90 血、钟点 2:00
	hours, full, _ := hit(m, me, foe, 10, 0.3, 0, false)
	if hp := hpOf(m, me); hp != 90 {
		t.Fatalf("先挨 10 点后血=%v，想要 90", hp)
	}
	hours, full, _ = runTokei(m, 7.6, hours, full)
	hours, full, _ = runTokei(m, 0.5, hours, full)
	pins := momentsXY(m, me)
	if len(pins) != 1 || hours != 2 {
		t.Fatalf("档没刻上：档数=%d 表针=%v", len(pins), hours)
	}
	saveX, saveY := posOf(m, me)

	m.mu.Lock()
	me.p = vec{saveX + 120, saveY}
	m.mu.Unlock()
	hours, full, fx := hit(m, me, foe, 150, 0.06, hours, full)
	if n := len(momentsXY(m, me)); n != 0 {
		t.Fatalf("回拨后档数=%d，想要 0", n)
	}
	if hours != 0 {
		t.Fatalf("回拨后表针 %v，想要 −2h 到 0:00", hours)
	}
	if fx["rewind"] == 0 {
		t.Fatal("回拨没发演出 FX")
	}
	if hp := hpOf(m, me); hp != 90 {
		t.Fatalf("回拨后血=%v，想要档里的 90", hp)
	}
	x, y := posOf(m, me)
	if d := math.Hypot(x-saveX, y-saveY); d > 20 {
		t.Fatalf("回拨没回到档位：偏移 %v", d)
	}
	_ = full
}

// 致命伤：有档就回拨 + BlockDamage 整包拦下，人活着。
func TestTokeiBlocksLethalDamageWithMoment(t *testing.T) {
	m, me, foe := startTokei(t, 14)
	freeze(m, me)
	freeze(m, foe)
	hours, full, _ := runTokei(m, 8.4, 0, false)
	if n := len(momentsXY(m, me)); n != 1 {
		t.Fatalf("档数=%d，想要 1", n)
	}
	hours, full, fx := hit(m, me, foe, 150, 0.3, hours, full)
	if !alive(m, me) {
		t.Fatal("有档还被致命伤打死了")
	}
	if hp := hpOf(m, me); hp != 100 {
		t.Fatalf("回拨后血=%v，想要档里的 100", hp)
	}
	if n := len(momentsXY(m, me)); n != 0 {
		t.Fatalf("致命回拨后档数=%d，想要 0（用掉一档）", n)
	}
	if fx["rewind"] == 0 {
		t.Fatal("回拨没发演出 FX")
	}
}

// 档空了再吃致命伤就照常死：回拨保不了没档的人。
func TestTokeiDiesFromLethalWithoutMoment(t *testing.T) {
	m, me, foe := startTokei(t, 15)
	freeze(m, me)
	freeze(m, foe)
	// 还没到第一个 8s 节拍，手上没档
	if n := len(momentsXY(m, me)); n != 0 {
		t.Fatalf("开局就有档：%d", n)
	}
	runTokei(m, 1.5, 0, false)
	hit(m, me, foe, 150, 0.5, 0, false)
	if alive(m, me) {
		t.Fatal("没档还挡住了致命伤")
	}
}

// 回拨的 Mark 口径：当下的先清、档里的装回；顺带解控（档是清醒时拍的，快照里没有的不该有）。
func TestTokeiRollbackRestoresMarksAndClearsStun(t *testing.T) {
	m, me, foe := startTokei(t, 16)
	freeze(m, me)
	freeze(m, foe)
	// 刻档前挂上「甲×2」，档里就该记着它
	m.mu.Lock()
	m.applyCmdLocked(unitpkg.StackMark{UnitID: me.id, Kind: "甲", Delta: 2, Icon: "a"})
	m.mu.Unlock()
	hours, full, _ := runTokei(m, 8.4, 0, false)
	if n := len(momentsXY(m, me)); n != 1 {
		t.Fatalf("档数=%d，想要 1", n)
	}
	if got := marksOf(m, me)["甲"]; got != 2 {
		t.Fatalf("档没记住标记：%v", marksOf(m, me))
	}
	// 刻档之后换一身标记（乙）再上眩晕
	m.mu.Lock()
	m.applyCmdLocked(unitpkg.ClearMarks{UnitID: me.id})
	m.applyCmdLocked(unitpkg.StackMark{UnitID: me.id, Kind: "乙", Delta: 1, Icon: "b"})
	m.applyCmdLocked(unitpkg.Stun{UnitID: me.id, Hold: true})
	m.mu.Unlock()
	hours, full, _ = runTokei(m, 0.2, hours, full)
	if got := marksOf(m, me); got["乙"] != 1 || got["甲"] != 0 {
		t.Fatalf("前置没生效：%v", got)
	}
	// 眩晕里照样吃 IncomingDamage：致命伤触发回拨 → 标记装回档里的、眩晕解除
	hours, full, _ = hit(m, me, foe, 150, 0.3, hours, full)
	got := marksOf(m, me)
	if got["甲"] != 2 {
		t.Fatalf("回拨没装回档里的标记：%v", got)
	}
	if got["乙"] != 0 {
		t.Fatalf("当下的标记没清掉：%v", got)
	}
	m.mu.Lock()
	stunned := me.stun
	m.mu.Unlock()
	if stunned {
		t.Fatal("回拨没解控")
	}
}

// 时刻会被吹笛人的老鼠叼走（stealable 口径：Helper、非战斗机、无弧 → 天然在可偷清单里）。
// 被偷就是档池少一、回拨锚点没了；该钟点仍算已刻，永不再落 2:00，下一拍走到 4:00 才落新档。
func TestTokeiDropsSlotWhenMomentStolen(t *testing.T) {
	m, me, foe := startTokei(t, 20)
	freeze(m, me)
	freeze(m, foe)
	hours, full, _ := runTokei(m, 8.4, 0, false)
	pins := momentsXY(m, me)
	if len(pins) != 1 || hours != 2 {
		t.Fatalf("档数=%d 表针=%v，想要 1 / 2:00", len(pins), hours)
	}
	var pinID uint64
	for id := range pins {
		pinID = id
	}
	m.mu.Lock()
	if u := m.units[pinID]; u != nil {
		m.removeLocked(u)
	}
	m.mu.Unlock()
	hours, full, _ = runTokei(m, 0.3, hours, full)
	if n := len(momentsXY(m, me)); n != 0 {
		t.Fatalf("被偷了还留着 %d 档", n)
	}
	if hours != 2 {
		t.Fatalf("被偷不该拨表针：%v", hours)
	}
	// 走到下一拍：4:00 落下（不是补回 2:00）
	hours, _, _ = runTokei(m, 8.0, hours, full)
	pins = momentsXY(m, me)
	if len(pins) != 1 {
		t.Fatalf("过了节拍档数=%d，想要 1（4:00）", len(pins))
	}
	if hours != 4 {
		t.Fatalf("表针 %v，想要 4:00（2:00 已用永不回补）", hours)
	}
}

// 时计死了，时刻随 owner 链一起消失（引擎现成口径，这里把它钉住）。
func TestTokeiMomentsDieWithOwner(t *testing.T) {
	m, me, foe := startTokei(t, 21)
	freeze(m, me)
	freeze(m, foe)
	runTokei(m, 8.4, 0, false)
	if n := len(momentsXY(m, me)); n != 1 {
		t.Fatalf("档数=%d，想要 1", n)
	}
	m.mu.Lock()
	m.removeLocked(me) // 引擎的死亡路径：连带 owner 链上的东西一起走
	m.mu.Unlock()
	runTokei(m, 0.2, 0, false)
	if n := len(momentsXY(m, me)); n != 0 {
		t.Fatalf("主人没了还剩 %d 个时刻", n)
	}
}

// 踩影：敌方进圈就把时计拉过去撞（350 朝触发者），圈外不进不触发。
func TestTokeiTrapTeleportsAndRams(t *testing.T) {
	m, me, foe := startTokei(t, 17)
	freeze(m, foe)
	hours, full, _ := runTokei(m, 8.4, 0, false)
	pins := momentsXY(m, me)
	if len(pins) != 1 {
		t.Fatalf("档数=%d，想要 1", len(pins))
	}
	var px, py float64
	for _, p := range pins {
		px, py = p[0], p[1]
	}
	// 时计先走开 260，靶子摆在圈外（判定圈 22 + 靶子半径 18 = 40，摆 70）
	m.mu.Lock()
	me.p = vec{px + 260, py}
	foe.p = vec{px + 70, py}
	m.mu.Unlock()
	hours, full, fx := runTokei(m, 0.2, hours, full)
	if fx["trip"] != 0 {
		t.Fatal("圈外就触发了")
	}
	// 推进圈里：应该瞬到时刻位置并朝靶子冲（方向看 FX，撞上被弹回来也不影响判据）
	m.mu.Lock()
	foe.p = vec{px + 30, py}
	m.mu.Unlock()
	dirX, dirY, _, _, fx, fired := tripWatch(m, me, 0.4)
	if !fired || fx["trip"] == 0 {
		t.Fatal("进圈没触发")
	}
	if dirX < 0.9 {
		t.Fatalf("冲的方向不对：(%v,%v)，触发者在 +x", dirX, dirY)
	}
	// 落脚点：在「档位→触发者」这条线上、离触发者 双方半径和+跑道（18+18+60=96）处，
	// 也就是档位那一侧退出去一截当跑道；不能原地不动，也别飞过头跑到触发者背后
	lx, ly := posOf(m, me)
	m.mu.Lock()
	bx, by := foe.p.X, foe.p.Y
	m.mu.Unlock()
	// 落脚点必须在「档位→触发者」这条连线上（跑道的方向是对的），而且不能跑到触发者背后。
	// 不比对具体距离：这一拍它可能已经在冲了，量到的距离随冲刺进度变。
	ax, ay := bx-px, by-py
	an := math.Hypot(ax, ay)
	perp := math.Abs((lx-px)*ay-(ly-py)*ax) / an
	if perp > 20 {
		t.Fatalf("落脚点不在档位→触发者连线上：横向偏差 %v", perp)
	}
	if proj := ((lx-px)*ax + (ly-py)*ay) / an; proj > an+20 {
		t.Fatalf("跑到触发者背后去了：沿轴投影 %v > 触发者距离 %v", proj, an)
	}
	if d := math.Hypot(lx-px, ly-py); d > 140 {
		t.Fatalf("没被拉到档位那一带：离档位 %v", d)
	}
}

// 静默：同一时刻被踩后 3s 内不再触发（站圈里不算、出圈再进也不算），过了静默再来才算。
func TestTokeiTrapSilenceAndEdgeTrigger(t *testing.T) {
	m, me, foe := startTokei(t, 18)
	freeze(m, foe)
	hours, full, _ := runTokei(m, 8.4, 0, false)
	pins := momentsXY(m, me)
	if len(pins) != 1 {
		t.Fatalf("档数=%d，想要 1", len(pins))
	}
	var px, py float64
	for _, p := range pins {
		px, py = p[0], p[1]
	}
	m.mu.Lock()
	me.p = vec{px + 300, py + 300}
	foe.p = vec{px + 30, py}
	m.mu.Unlock()
	_, _, fx := runTokei(m, 0.2, hours, full)
	if fx["trip"] == 0 {
		t.Fatal("进圈第一下没触发")
	}
	// 站圈里不动：不该重复触发
	_, _, fx = runTokei(m, 0.6, hours, full)
	if fx["trip"] != 0 {
		t.Fatalf("站圈里又触发了 %d 次", fx["trip"])
	}
	// 出圈再进，但还在 3s 静默内：也不触发
	m.mu.Lock()
	foe.p = vec{px + 70, py}
	m.mu.Unlock()
	runTokei(m, 0.3, hours, full)
	m.mu.Lock()
	foe.p = vec{px + 30, py}
	m.mu.Unlock()
	_, _, fx = runTokei(m, 0.3, hours, full)
	if fx["trip"] != 0 {
		t.Fatalf("静默期内出圈再进又触发了 %d 次", fx["trip"])
	}
	// 熬过静默（第一次触发到现在 > 3s），出圈再进：这回该算
	m.mu.Lock()
	foe.p = vec{px + 70, py}
	m.mu.Unlock()
	runTokei(m, 2.4, hours, full)
	m.mu.Lock()
	foe.p = vec{px + 30, py}
	m.mu.Unlock()
	_, _, fx = runTokei(m, 0.3, hours, full)
	if fx["trip"] == 0 {
		t.Fatal("过了静默再进圈还是没触发")
	}
}

// 压测：2、4 已刻（针在 4）→ 致命用掉 4（针 −2h → 2）→ 再致命用掉 2（针 → 0）→
// 之后空过已用的 2、4，在 6 落下。也钉死「弹掉 4 时针必须回头，不能停在 4」。
func TestTokeiSkipsSpentHourAfterRewind(t *testing.T) {
	m, me, foe := startTokei(t, 27)
	freeze(m, me)
	freeze(m, foe)
	hours, full, _ := runTokei(m, 16.4, 0, false)
	if n := len(momentsXY(m, me)); n != 2 || hours != 4 {
		t.Fatalf("前置：档数=%d 表针=%v，想要 2 / 4:00", n, hours)
	}
	// 第一次致命：弹掉 4:00，针 −2h → 2（若误对齐档点会停在 4，回拨看起来没动）
	hours, full, _ = hit(m, me, foe, 150, 0.3, hours, full)
	if n := len(momentsXY(m, me)); n != 1 || hours != 2 {
		t.Fatalf("弹 4 后：档数=%d 表针=%v，想要 1 / 2:00", n, hours)
	}
	// 第二次致命：弹掉 2:00，针再 −2h → 0
	hours, full, _ = hit(m, me, foe, 150, 0.3, hours, full)
	if n := len(momentsXY(m, me)); n != 0 || hours != 0 {
		t.Fatalf("弹 2 后：档数=%d 表针=%v，想要 0 / 0:00", n, hours)
	}
	// 空过 2
	hours, full, _ = runTokei(m, 8.2, hours, full)
	if n := len(momentsXY(m, me)); n != 0 || hours != 2 {
		t.Fatalf("空过 2：档数=%d 表针=%v，想要 0 / 2:00", n, hours)
	}
	// 空过 4
	hours, full, _ = runTokei(m, 8.2, hours, full)
	if n := len(momentsXY(m, me)); n != 0 || hours != 4 {
		t.Fatalf("空过 4：档数=%d 表针=%v，想要 0 / 4:00", n, hours)
	}
	// 落 6
	hours, _, _ = runTokei(m, 8.2, hours, full)
	if n := len(momentsXY(m, me)); n != 1 || hours != 6 {
		t.Fatalf("落 6：档数=%d 表针=%v，想要 1 / 6:00", n, hours)
	}
}

// 无常态弧：贴着敌人站也不出伤；只有踩影后的临时弧才能打到人。
func TestTokeiNoArcWithoutTrip(t *testing.T) {
	m, me, foe := startTokei(t, 28)
	freeze(m, me)
	freeze(m, foe)
	hours, full, _ := runTokei(m, 8.4, 0, false)
	pins := momentsXY(m, me)
	if len(pins) != 1 {
		t.Fatalf("档数=%d，想要 1", len(pins))
	}
	var px, py float64
	for _, p := range pins {
		px, py = p[0], p[1]
	}
	// 两人贴身站在远离时刻的地方：旧设计会刮伤，新设计不该有弧
	m.mu.Lock()
	me.p = vec{px + 200, py}
	foe.p = vec{px + 220, py}
	m.mu.Unlock()
	before := hpOf(m, foe)
	runTokei(m, 1.0, hours, full)
	if after := hpOf(m, foe); after < before {
		t.Fatalf("没踩影就出伤了：hp %v → %v", before, after)
	}
}

// 弹体进圈不算（朝弹撞毫无意义）；时计眩晕时收不到 Sense，踩影也不触发。
func TestTokeiTrapIgnoresProjectilesAndStun(t *testing.T) {
	m, me, foe := startTokei(t, 19)
	freeze(m, foe)
	hours, full, _ := runTokei(m, 8.4, 0, false)
	pins := momentsXY(m, me)
	if len(pins) != 1 {
		t.Fatalf("档数=%d，想要 1", len(pins))
	}
	var px, py float64
	for _, p := range pins {
		px, py = p[0], p[1]
	}
	m.mu.Lock()
	me.p = vec{px + 300, py + 300}
	foe.p = vec{px + 70, py} // 靶子在圈外
	m.mu.Unlock()
	// 敌方弹体压在圈心：不该触发
	m.mu.Lock()
	shot := m.addUnitLocked(character.KindGodfatherShot, vec{px, py}, vec{}, foe.id, 1)
	m.mu.Unlock()
	if shot == nil {
		t.Fatal("拿不到一发弹体做对照")
	}
	_, _, fx := runTokei(m, 0.3, hours, full)
	if fx["trip"] != 0 {
		t.Fatalf("弹体触发踩影 %d 次", fx["trip"])
	}
	// 把靶子也推进圈里，但时计处于眩晕：感知停发，踩影不触发
	m.mu.Lock()
	foe.p = vec{px + 30, py}
	m.applyCmdLocked(unitpkg.Stun{UnitID: me.id, Hold: true})
	m.mu.Unlock()
	_, _, fx = runTokei(m, 0.4, hours, full)
	if fx["trip"] != 0 {
		t.Fatalf("眩晕中还触发了 %d 次", fx["trip"])
	}
	m.mu.Lock()
	m.applyCmdLocked(unitpkg.Stun{UnitID: me.id, Hold: false})
	m.mu.Unlock()
}
