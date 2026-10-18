package sim

import (
	"math"
	"sort"
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

// startArcher 开一局 弓箭 vs 靶子：靶子 99999 血、不还手，正好当量伤害的沙包。
func startArcher(t *testing.T, seed uint64) (*Match, *unit, *unit) {
	t.Helper()
	m := NewMatchSeeded(seed)
	m.SetSlot(0, character.KindArcher)
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

// standApart 把两只钉住（放弃巡航）、沿 +y 摆到间距 dist：弓箭在下、靶子在上，瞄准方向就是 +y。
// 走 y 轴是为了让"横移"（垂直于弹道的 x 方向）两端都还在场内——沿 x 摆到 260 再横移 40 就出场，
// 靶子会被引擎夹回来，测到的几何就不是我们要的那个了。
func standApart(m *Match, me, foe *unit, dist float64) {
	m.mu.Lock()
	for _, u := range []*unit{me, foe} {
		u.stand = true
		u.setVel(vec{})
		m.applyCmdLocked(unitpkg.SetCruise{UnitID: u.id, Speed: 0})
	}
	me.p = vec{0, -dist / 2}
	foe.p = vec{0, dist / 2}
	m.mu.Unlock()
}

// runArcher 跑 secs 秒模拟时间，返回这段里的 FX 计数（复用 tokei_test 里的推进器）。
func runArcher(m *Match, secs float64) map[string]int {
	_, _, fx := runTokei(m, secs, 0, false)
	delete(fx, "clock")
	return fx
}

// liveArrows 场上属于该槽的箭（两种箭都算）。
func liveArrows(m *Match, me *unit) []*unit {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*unit
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.stopped || u.owner != me.id {
			continue
		}
		if u.kind == character.KindArrow || u.kind == character.KindVolleyArrow {
			out = append(out, u)
		}
	}
	return out
}

func dist2(m *Match, a, b *unit) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return math.Hypot(a.p.X-b.p.X, a.p.Y-b.p.Y)
}

// 蓄力射击：站定 0.5s → 一箭 7 伤 → 后摇 0.5s，前后摇就是它的节拍（1.0s/箭，最短的一档）。
func TestArcherChargeShootRecover(t *testing.T) {
	m, me, foe := startArcher(t, 41)
	standApart(m, me, foe, 120) // 中距 → 蓄力射击
	before := hpOf(m, foe)
	fx := runArcher(m, 1.2)
	if fx["draw"] == 0 {
		t.Fatal("没蓄力")
	}
	if fx["shoot"] != 1 {
		t.Fatalf("1.2s 内出了 %d 箭，想要 1（0.5 蓄力 + 0.5 后摇，第二箭在 1.5s）", fx["shoot"])
	}
	if got := before - hpOf(m, foe); math.Abs(got-5) > 0.01 {
		t.Fatalf("一次蓄射伤害 %v，想要 5", got)
	}
	// 再跑 1.0s（合计 ~2.2s）：节拍 1.0s → 该出第二箭了（1.25s 的旧节拍这时只有一箭）
	fx = runArcher(m, 1.0)
	if fx["shoot"] != 1 {
		t.Fatalf("第二段又出了 %d 箭，想要 1（合计两箭）", fx["shoot"])
	}
	if got := before - hpOf(m, foe); math.Abs(got-10) > 0.01 {
		t.Fatalf("2.2s 后总伤害 %v，想要 10（两箭）", got)
	}
}

// 箭撞场边即消，不弹射（弹射的箭会一直在场上飞）。
func TestArcherArrowDiesOnWall(t *testing.T) {
	m, me, _ := startArcher(t, 42)
	m.mu.Lock()
	me.p = vec{200, 0}
	me.setVel(vec{})
	arrow := m.addUnitLocked(character.KindArrow, vec{200, 0}, vec{400, 0}, me.id, me.slot)
	id := arrow.id
	m.mu.Unlock()
	runArcher(m, 1.0)
	m.mu.Lock()
	u := m.units[id]
	live := u != nil && !u.stopped
	m.mu.Unlock()
	if live {
		t.Fatal("箭撞场边没消失（还在飞/在弹）")
	}
}

// 站定蓄力时弓要跟着目标转：不然"弓指的方向"和"出膛方向"（开火那一拍重新瞄）会不一致。
func TestArcherTracksTargetWhileCharging(t *testing.T) {
	m, me, foe := startArcher(t, 50)
	standApart(m, me, foe, 120) // 目标先在 +y
	var fx, fy float64
	got := false
	for i := 0; i < 60; i++ {
		if i == 10 { // 蓄力中途把目标挪到 +x 侧，朝向得跟着变
			m.mu.Lock()
			foe.p = vec{120, 0}
			m.mu.Unlock()
		}
		m.Tick()
		time.Sleep(300 * time.Microsecond)
		m.mu.Lock()
		for _, f := range m.fx {
			if f.Name == "face" {
				fx, fy, got = f.VX, f.VY, true
			}
		}
		m.mu.Unlock()
	}
	if !got {
		t.Fatal("蓄力期间没发 face")
	}
	m.mu.Lock()
	dx, dy := foe.p.X-me.p.X, foe.p.Y-me.p.Y
	m.mu.Unlock()
	n := math.Hypot(dx, dy)
	wantX, wantY := dx/n, dy/n
	if dot := fx*wantX + fy*wantY; dot < 0.99 {
		t.Fatalf("弓没跟着目标转：最后朝向 (%v,%v)，目标方向 (%v,%v)", fx, fy, wantX, wantY)
	}
}

// 箭的眩晕必须"看得出来"：目标是真的被停住 0.2s、到点速度还回来。
// 只发 Stun 是看不见的（它只停感知，球照旧按速度飞）——这个用例就是钉这一点。
func TestArcherArrowFreezesTargetMomentum(t *testing.T) {
	m, me, foe := startArcher(t, 49)
	standApart(m, me, foe, 120)
	// 给靶子一条速度（否则它本来就是静止的，测不出"被停住"）
	m.mu.Lock()
	foe.stand = false
	m.applyCmdLocked(unitpkg.SetCruise{UnitID: foe.id, Speed: 165})
	foe.setVel(vec{165, 0})
	// 箭直接生在它面前一点点，保证第一拍就中（它同时在往 +x 飘）
	arrow := m.addUnitLocked(character.KindVolleyArrow, vec{0, 38}, vec{0, 320}, me.id, me.slot)
	m.mu.Unlock()
	if arrow == nil {
		t.Fatal("箭没生成")
	}
	// 命中发生在第一拍（t≈0），眩晕窗口是 [0, 0.2]；runArcher 会多跑 4 拍收尾，
	// 所以这里要挑窗口内的时点量（0.02+收尾≈0.1），别贴着 0.2 的边界量。
	runArcher(m, 0.02)
	vx, vy := velOf(m, foe)
	if sp := math.Hypot(vx, vy); sp > 20 {
		t.Fatalf("中箭没被停住：速度 %v（想要 0）", sp)
	}
	runArcher(m, 0.3) // 眩晕到期（>0.2s）
	vx, vy = velOf(m, foe)
	if sp := math.Hypot(vx, vy); sp < 100 {
		t.Fatalf("眩晕到期没把速度还回来：%v（想要 ~165）", sp)
	}
}

// 箭的 0.2s 眩晕：同波不叠加——四支隔几拍打上，最后一支 +0.25s 后必须已经醒了。
func TestArcherArrowStunDoesNotStack(t *testing.T) {
	m, me, foe := startArcher(t, 43)
	standApart(m, me, foe, 120)
	// 每拍补一支箭，让它隔几拍命中（同一帧同值覆盖本来就不会叠，这里测的是"覆盖"不是"累加"）
	for i := 0; i < 4; i++ {
		m.mu.Lock()
		foe.p = vec{60, 0}
		m.addUnitLocked(character.KindVolleyArrow, vec{20, 0}, vec{320, 0}, me.id, me.slot)
		m.mu.Unlock()
		runArcher(m, 0.03)
	}
	m.mu.Lock()
	mid := foe.stun
	m.mu.Unlock()
	if !mid {
		t.Fatal("中箭没眩晕")
	}
	runArcher(m, 0.3) // 最后一支 +0.25s 之后
	m.mu.Lock()
	still := foe.stun
	until := foe.stunUntil
	m.mu.Unlock()
	if still {
		t.Fatalf("眩晕叠加了：直到 %v 还晕着（覆盖口径下这时该醒）", until)
	}
}

// 齐射：四支并排（心隔 8）、同时出膛；对心命中四支全中 20 伤。
func TestArcherVolleyFourParallelArrows(t *testing.T) {
	m, me, foe := startArcher(t, 44)
	standApart(m, me, foe, 260) // > 180 且无弹道威胁 → 齐射
	var offs []float64
	for i := 0; i < 900; i++ {
		m.Tick()
		time.Sleep(300 * time.Microsecond)
		got := liveArrows(m, me)
		if len(got) == 4 {
			m.mu.Lock()
			for _, u := range got {
				offs = append(offs, u.p.X) // 瞄准方向是 +y，垂直方向（并排方向）就是 x
			}
			m.mu.Unlock()
			break
		}
	}
	if len(offs) != 4 {
		t.Fatalf("没抓到四支并排的箭：%v", offs)
	}
	sort.Float64s(offs)
	want := []float64{-12, -4, 4, 12}
	for i := range want {
		// 量的是出膛后一拍的 y：允许一点点瞄准偏差（箭是平行的，不散）
		if math.Abs(offs[i]-want[i]) > 0.1 {
			t.Fatalf("并排间隔不对：%v，想要 %v（心隔 8）", offs, want)
		}
	}
	before := hpOf(m, foe)
	runArcher(m, 1.2) // 箭飞 260px
	if got := before - hpOf(m, foe); math.Abs(got-20) > 0.01 {
		t.Fatalf("对心齐射伤害 %v，想要 20（四支全中）", got)
	}
}

// 齐射的横移口径：出膛后目标横移 20 → 只中 2 支（10 伤）；横移 40 → 一支不中。
// 齐射的横移口径（设计稿：全中走廊 ±12、掉支带 12~36、超过 36 才整个躲开）：
//
//	横移 0  → 四支全中 20 伤
//	横移 20 → 掉到 2~3 支（10~15 伤）
//	横移 40 → 一支不中
//
// 注意命中会把靶子推开一点（箭撞单位有位置分离），所以掉支带里的支数会在 2~3 之间浮动。
func TestArcherVolleySpreadAgainstLateralMove(t *testing.T) {
	for _, c := range []struct {
		lat      float64
		min, max float64
	}{
		{0, 20, 20},
		{20, 10, 15},
		{40, 0, 0},
	} {
		m, me, foe := startArcher(t, 45+uint64(c.lat))
		standApart(m, me, foe, 260)
		fired := false
		for i := 0; i < 900; i++ {
			m.Tick()
			time.Sleep(300 * time.Microsecond)
			if len(liveArrows(m, me)) == 4 {
				fired = true
				break
			}
		}
		if !fired {
			t.Fatalf("横移 %v 这一例齐射没出膛", c.lat)
		}
		// 出膛之后才把靶子横移（飞行 0.8s，来得及）——测的是"飞在路上的目标横移"
		m.mu.Lock()
		foe.p = vec{c.lat, 130}
		m.mu.Unlock()
		before := hpOf(m, foe)
		runArcher(m, 1.2)
		got := before - hpOf(m, foe)
		if got < c.min-0.01 || got > c.max+0.01 {
			t.Fatalf("横移 %v 时齐射伤害 %v，想要 %v~%v（支数 %v~%v）", c.lat, got, c.min, c.max, c.min/5, c.max/5)
		}
	}
}

// 翻滚：挨报价那一拍 Block + 朝安全向滚开，并打断自己的蓄力（这箭哑火）；CD 内的攻击照常吃。
func TestArcherRollBlocksInterruptsAndCoolsDown(t *testing.T) {
	m, me, foe := startArcher(t, 47)
	standApart(m, me, foe, 120)
	runArcher(m, 0.25) // 蓄力进行到一半
	hp0 := hpOf(m, me)
	x0, y0 := posOf(m, me)
	m.mu.Lock()
	m.offerDamageLocked(unitpkg.Damage{From: foe.id, To: me.id, Amount: 30})
	m.mu.Unlock()
	fx := runArcher(m, 0.35)
	if hp := hpOf(m, me); hp != hp0 {
		t.Fatalf("翻滚没挡下这一击：%v → %v", hp0, hp)
	}
	x, y := posOf(m, me)
	if moved := math.Hypot(x-x0, y-y0); moved < 30 {
		t.Fatalf("没滚开：位移 %v", moved)
	}
	if fx["roll"] == 0 {
		t.Fatal("没发翻滚 FX")
	}
	if fx["shoot"] != 0 {
		t.Fatalf("翻滚没打断蓄力：被打断那一拍还出了 %d 箭", fx["shoot"])
	}
	// CD 内再来一记：照常吃
	m.mu.Lock()
	m.offerDamageLocked(unitpkg.Damage{From: foe.id, To: me.id, Amount: 30})
	m.mu.Unlock()
	runArcher(m, 0.3)
	if hp := hpOf(m, me); math.Abs(hp-(hp0-30)) > 0.01 {
		t.Fatalf("翻滚 CD 内没吃到伤害：hp %v，想要 %v", hp, hp0-30)
	}
}

// 后撤步射击：贴脸时立刻朝安全向滑步约 90，撤的过程中射一支箭。
func TestArcherRetreatsWhenClose(t *testing.T) {
	m, me, foe := startArcher(t, 48)
	standApart(m, me, foe, 70) // < 90 贴脸
	fx := runArcher(m, 0.5)
	if fx["dash"] == 0 {
		t.Fatal("贴脸没后撤")
	}
	if fx["shoot"] == 0 {
		t.Fatal("撤的过程中没射箭")
	}
	if d := dist2(m, me, foe); d < 70+40 {
		t.Fatalf("后撤距离不够：70 → %v，想要 ~160（+90）", d)
	}
}
