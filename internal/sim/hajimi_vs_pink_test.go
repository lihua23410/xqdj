package sim

import (
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

// 反重力猫咪 vs 粉红色迷人的pink 诊断：
//  1. 猫必须**吃伤害**（漏了 ConfirmHit 就会完全免疫——线上踩过一次）；
//  2. 摸清"场结束后就不动了"是谁不动：场（battle.go 的 fire）给出去的减速是
//     AddFSComponent{FSZoneM}，自带 ExpiresAt 会自己过期，所以真卡住只能是别的原因
//     （速度被乘成 0 + 巡航没拉回来，或者 pink 自己的状态机锁住）。这里同时记双方的静止时长。
func TestHajimiVsPinkTakesDamage(t *testing.T) {
	var (
		damagedAny bool
		maxCat     float64
	)
	for _, seed := range []uint64{80, 81, 82} {
		hp0, hp1, catStall, pinkStall, frames := pinkRound(t, seed, 20)
		if hp1 < hp0 {
			damagedAny = true
		}
		if catStall > maxCat {
			maxCat = catStall
		}
		t.Logf("seed %d 20s：猫 hp %.0f→%.0f 最长静止 %.2fs；pink 最长静止 %.2fs（%d 帧）",
			seed, hp0, hp1, catStall, pinkStall, frames)
		if catStall > 3 {
			t.Fatalf("seed %d：猫静止了 %.2fs —— 稳态下不该出现", seed, catStall)
		}
	}
	if !damagedAny {
		t.Fatal("三轮下来猫一直没吃到 pink 的伤害（漏了 ConfirmHit？）")
	}
	t.Logf("猫最长静止 %.2fs", maxCat)
}

// pinkRound 打一局，返回双方起始/结束 HP 与"速率<5 的最长连续时长"。
func pinkRound(t *testing.T, seed uint64, seconds float64) (hp0, hp1, catStall, pinkStall float64, frames int) {
	t.Helper()
	m := NewMatchSeeded(seed)
	m.SetSlot(0, character.KindHajimi)
	m.SetSlot(1, character.KindPink)
	m.Start()
	defer m.End()
	time.Sleep(30 * time.Millisecond)

	var cat, pink *unit
	pinkHP0 := 0.0
	m.mu.Lock()
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.role != unitpkg.RoleFighter {
			continue
		}
		if u.slot == 0 {
			cat = u
		} else {
			pink = u
		}
	}
	hp0 = cat.hp
	pinkHP0 = pink.hp
	start := m.time
	m.mu.Unlock()

	stall, last := 0.0, start
	slowFrames := 0
	var catStallAcc, pinkStallAcc float64
	for i := 0; i < 40000; i++ {
		m.Tick()
		time.Sleep(200 * time.Microsecond)
		m.mu.Lock()
		now, phase := m.time, m.phase
		dt := now - last
		last = now
		cu, pu := m.units[cat.id], m.units[pink.id]
		m.mu.Unlock()
		if cu == nil || pu == nil || cu.stopped || pu.stopped {
			break
		}
		frames++
		// 巡航 178 被压到 40~150 就说明 pink 的减速确实吃上了（fireSlow=0.6 → 期望 106.8）
		if sp := cu.v.len(); sp > 40 && sp < 150 {
			slowFrames++
		}
		if cu.v.len() < 5 {
			stall += dt
			if stall > catStallAcc {
				catStallAcc = stall
			}
		} else {
			stall = 0
		}
		if pu.v.len() < 5 {
			pinkStall += dt
			if pinkStall > pinkStallAcc {
				pinkStallAcc = pinkStall
			}
		} else {
			pinkStall = 0
		}
		if phase != PhaseRunning || now-start > seconds {
			break
		}
	}
	hp1 = hpOf(m, cat)
	elapsed := float64(frames) * 0.016
	if frames > 0 {
		m.mu.Lock()
		now := m.time
		m.mu.Unlock()
		elapsed = now - start
	}
	dealt := pinkHP0 - hpOf(m, pink)
	dead := ""
	if p := m.units[pink.id]; p != nil && p.stopped {
		dead = "（pink 已阵亡）"
	}
	t.Logf("  seed %d：速率被压到 40~150 的帧 %d / %d；猫对 pink 造成 %.0f 伤 / %.1fs = %.1f dps% s",
		seed, slowFrames, frames, dealt, elapsed, dealt/elapsed, dead)
	return hp0, hp1, catStallAcc, pinkStallAcc, frames
}
