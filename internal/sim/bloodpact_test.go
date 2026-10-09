package sim

import (
	"math"
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

// startPact 开一局 血契 vs foe，返回对局、血契、对面战斗机。
func startPact(t *testing.T, seed uint64, foe string) (*Match, *unit, *unit) {
	t.Helper()
	m := NewMatchSeeded(seed)
	m.SetSlot(0, character.KindBloodPact)
	m.SetSlot(1, foe)
	m.Start()
	t.Cleanup(m.End)
	time.Sleep(20 * time.Millisecond)
	var pact, victim *unit
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.role != unitpkg.RoleFighter {
			continue
		}
		if u.slot == 0 {
			pact = u
		} else {
			victim = u
		}
	}
	if pact == nil || victim == nil {
		t.Fatal("missing fighters")
	}
	return m, pact, victim
}

// lockBoth 把两只都钉住，并把对面摆到血契正前方 dist 处（面朝默认 +x）。
// 巡航必须走 SetCruise 改巡航 FS 本体：直接写 u.cruise 会被引擎每拍同步回去，
// 只要被推一下，对面就按自己的巡航飞走，血线当场断。
func lockBoth(m *Match, pact, victim *unit, dist float64) {
	m.mu.Lock()
	pact.stand = true
	pact.setVel(vec{})
	m.applyCmdLocked(unitpkg.SetCruise{UnitID: pact.id, Speed: 0})
	victim.p = vec{pact.p.X + dist, pact.p.Y}
	victim.stand = true
	victim.setVel(vec{})
	m.applyCmdLocked(unitpkg.SetCruise{UnitID: victim.id, Speed: 0})
	m.mu.Unlock()
}

// tickN 跑 n 拍，把这几拍里出现的 FX 名字收下来。
func tickN(m *Match, n int) map[string]int {
	seen := map[string]int{}
	for i := 0; i < n; i++ {
		m.Tick()
		time.Sleep(time.Millisecond)
		m.mu.Lock()
		for _, f := range m.fx {
			seen[f.Name]++
		}
		m.mu.Unlock()
	}
	return seen
}

// tickUntil 一直跑到**模拟时间**推进 secs 秒（撞上停帧时真实拍数是它的好几倍），
// 返回这段里出现的 FX 名字。测 CD / 蔫置计时这类按秒算的东西必须用这个。
func tickUntil(m *Match, secs float64) map[string]int {
	seen := map[string]int{}
	m.mu.Lock()
	start := m.time
	m.mu.Unlock()
	for i := 0; i < 6000; i++ {
		m.Tick()
		time.Sleep(time.Millisecond)
		m.mu.Lock()
		for _, f := range m.fx {
			seen[f.Name]++
		}
		now := m.time
		m.mu.Unlock()
		if now-start >= secs {
			break
		}
	}
	return seen
}

// putAway 把对面摆到血契朝场心方向 dist 处。
// 血契出生点就在场边，往 +x 推 90 直接就出场子了，会被引擎夹回来跑到别处去，
// 于是"墙在中间"这种几何就假了。
func putAway(m *Match, pact, victim *unit, dist float64) {
	m.mu.Lock()
	ux, uy := inwardUnit(pact.p.X, pact.p.Y)
	victim.p = vec{pact.p.X + ux*dist, pact.p.Y + uy*dist}
	victim.setVel(vec{})
	m.mu.Unlock()
}

func inwardUnit(x, y float64) (float64, float64) {
	n := math.Hypot(x, y)
	if n < 1e-6 {
		return 1, 0
	}
	return -x / n, -y / n
}

// noSeek 把对面板子上的可瞄准摘掉：unit.Seek 就索不到人，血契不会饿扑，
// 于是只留下要测的那套（咬合/血线/回流）。Hittable 不受影响，照样咬得动。
func noSeek(m *Match, victim *unit) {
	m.mu.Lock()
	m.applyCmdLocked(unitpkg.SetAimPriority{From: victim.id, UnitID: victim.id, Value: 0})
	m.mu.Unlock()
}

func pactHP(m *Match, u *unit) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return u.hp
}

func setPactHP(m *Match, u *unit, hp float64) {
	m.mu.Lock()
	u.hp = hp
	m.mu.Unlock()
}

// 咬合：扇区里咬到敌方战斗机结 7 伤，而且受 0.3 秒 CD 管——不是每拍都能咬。
// 量 blood-bite FX 的次数而不是 HP 差；开局先把血契压进熔断（HP 9），
// 蝙蝠召不出来——它们是实体，会把对面从咬距上挤开，主人的咬就乱了。
// 熔断只熄召蝙蝠/饿扑/收线，不熄免费咬合，正好单独量它。
func TestBloodBiteRespectsCooldown(t *testing.T) {
	m, pact, victim := startPact(t, 41, character.KindDummy)
	setPactHP(m, pact, 9) // 熔断区里，蝙蝠和扑都熄，只剩咬
	lockBoth(m, pact, victim, 34) // 圆心距 34 ≤ 外径 26 + 对方半径 18
	noSeek(m, victim)
	seen := tickUntil(m, 1.0)
	bites := seen["blood-bite"]
	// 1 秒最多咬 4 次（0.3 秒 CD）；每拍都能咬的坏法会到几十次。
	if bites < 2 {
		t.Fatalf("咬合该咬到好几次，实际咬了 %d 次：%v", bites, seen)
	}
	if bites > 4 {
		t.Fatalf("咬合该受 0.3 秒 CD 管，实际咬了 %d 次（像是每拍都在咬）", bites)
	}
}

// 咬合命中敌方战斗机才建血线；建上之后对面掉血按浅寄生 25% 回流。
func TestBloodLineFormsAndFeedsShallow(t *testing.T) {
	m, pact, victim := startPact(t, 42, character.KindDummy)
	lockBoth(m, pact, victim, 34)
	noSeek(m, victim)

	seen := tickN(m, 6)
	if seen["blood-bond"] == 0 {
		t.Fatalf("咬到敌方战斗机该建血线，没见过 blood-bond：%v", seen)
	}
	// 血线连着期间，对面掉多少血就按 25% 回流（浅寄生）。
	setPactHP(m, pact, 50)
	m.mu.Lock()
	before := pact.hp
	m.offerDamageLocked(unitpkg.Damage{From: victim.id, To: victim.id, Amount: 20})
	m.mu.Unlock()

	tickN(m, 6)
	after := pactHP(m, pact)
	if after-before < 4.5 {
		t.Fatalf("20 伤该回流 5（25%%），血契 %v -> %v", before, after)
	}
}

// 血线 >90 绷断：溅血、巡航 −30 蔫置三秒、期间不能重连、到点还回巡航。
func TestBloodLineSnapsAndWilts(t *testing.T) {
	m, pact, victim := startPact(t, 43, character.KindDummy)
	lockBoth(m, pact, victim, 34)
	noSeek(m, victim)

	if seen := tickUntil(m, 0.2); seen["blood-bond"] == 0 {
		t.Fatal("该先建上血线")
	}

	// 血线长 336：把两边各摆到场地两半（场心对称 ±200，距 400 跨过断线档）。
	// 出生点其实挨着场心，从原地朝任何方向都放不出 >336 的距离
	// （六边形内最远也就 264 左右），只能两边一起挪。
	m.mu.Lock()
	pact.p = vec{0, -200}
	pact.setVel(vec{})
	victim.p = vec{0, 200}
	victim.setVel(vec{})
	m.mu.Unlock()

	seen := tickUntil(m, 0.4)
	if seen["blood-snap"] == 0 {
		t.Fatalf("超过 90 该绷断并溅血，没见过 blood-snap：%v", seen)
	}
	m.mu.Lock()
	cruise := pact.cruise
	m.mu.Unlock()
	if math.Abs(cruise-(170-30)) > 1e-6 {
		t.Fatalf("蔫置该把巡航压到 140，实际 %.1f", cruise)
	}

	// 蔫置期间不能重连：把对面拉回嘴边也不该建线。
	m.mu.Lock()
	victim.p = vec{pact.p.X + 34, pact.p.Y}
	victim.setVel(vec{})
	m.mu.Unlock()
	if seen := tickUntil(m, 1.0); seen["blood-bond"] != 0 {
		t.Fatalf("蔫置三秒内不该重连，却建线了：%v", seen)
	}

	// 三秒过去：巡航还回来，可以重连（重连可能就发生在下面这段里）。
	tickWilt := tickUntil(m, 2.2)
	m.mu.Lock()
	cruise = pact.cruise
	m.mu.Unlock()
	if math.Abs(cruise-170) > 1e-6 {
		t.Fatalf("蔫置结束该还回 170，实际 %.1f", cruise)
	}
	if tickWilt["blood-bond"] == 0 {
		if seen := tickUntil(m, 0.5); seen["blood-bond"] == 0 {
			t.Fatalf("蔫置结束该能重连：%v / %v", tickWilt, seen)
		}
	}
}

// 回流分档：0–5 秒 25%、5–10 秒 50%、10 秒后 75%。
// 每档都往对面灌一大笔伤害（100），这样血契自己那几口咬合的回流只是零头，
// 量出来就是这一档的回流比例。
func TestBloodFeedDeepensOverTime(t *testing.T) {
	m, pact, victim := startPact(t, 44, character.KindDummy)
	lockBoth(m, pact, victim, 34)
	noSeek(m, victim)
	if seen := tickUntil(m, 0.2); seen["blood-bond"] == 0 {
		t.Fatal("该先建上血线")
	}

	feed := func() float64 {
		setPactHP(m, pact, 20)
		m.mu.Lock()
		before := pact.hp
		m.offerDamageLocked(unitpkg.Damage{From: victim.id, To: victim.id, Amount: 100})
		m.mu.Unlock()
		tickUntil(m, 0.2)
		return pactHP(m, pact) - before
	}

	// 还没到 5 秒：浅寄生 25%。
	if got := feed(); got < 24 || got > 30 {
		t.Fatalf("0–5 秒档该回流 25（25%%），实际 %v", got)
	}
	// 拖到 6 秒档：50%。
	tickUntil(m, 6.0)
	if got := feed(); got < 49 || got > 56 {
		t.Fatalf("5–10 秒档该回流 50（50%%），实际 %v", got)
	}
	// 再拖到 10 秒档：75%。
	tickUntil(m, 4.5)
	if got := feed(); got < 74 || got > 82 {
		t.Fatalf("10 秒后该回流 75（75%%），实际 %v", got)
	}
}

// 饿扑：开局无血线时烧 5 血、朝索敌目标以 400 速扑 0.5 秒。
func TestBloodPounceBurnsFiveAndDashes(t *testing.T) {
	m, pact, victim := startPact(t, 45, character.KindDummy)
	m.mu.Lock()
	pact.stand = true
	pact.setVel(vec{})
	m.applyCmdLocked(unitpkg.SetCruise{UnitID: pact.id, Speed: 0})
	victim.stand = true
	victim.setVel(vec{})
	m.applyCmdLocked(unitpkg.SetCruise{UnitID: victim.id, Speed: 0})
	hp0 := pact.hp
	m.mu.Unlock()
	// 目标摆到场心方向 120 处（不 noSeek，得有索敌对象），量扑出去那一下的速度。
	putAway(m, pact, victim, 120)

	// 只看头 8 拍：这时还没扑到人身上，回流掺不进来。
	peak := 0.0
	for i := 0; i < 8; i++ {
		m.Tick()
		time.Sleep(time.Millisecond)
		m.mu.Lock()
		if sp := math.Hypot(pact.v.X, pact.v.Y); sp > peak {
			peak = sp
		}
		m.mu.Unlock()
	}
	if got := hp0 - pactHP(m, pact); got < 4.5 {
		t.Fatalf("饿扑该烧 5 血，实际 %v", got)
	}
	if peak < 350 {
		t.Fatalf("饿扑该以 400 速扑出去，峰值速度只有 %.0f", peak)
	}
}

// 收线：烧 8 血把敌人朝自己拉回来，贴到 38 提前结束。
func TestBloodReelDragsThemBack(t *testing.T) {
	m, pact, victim := startPact(t, 46, character.KindDummy)
	lockBoth(m, pact, victim, 34)
	noSeek(m, victim)
	if seen := tickUntil(m, 0.2); seen["blood-bond"] == 0 {
		t.Fatal("该先建上血线")
	}

	// 对面跑到 90 开外：收线该把它拽到身前。
	putAway(m, pact, victim, 90)

	seen := map[string]int{}
	cost := 0.0
	minD := 1e9
	m.mu.Lock()
	start := m.time
	m.mu.Unlock()
	for i := 0; i < 4000; i++ {
		m.Tick()
		time.Sleep(time.Millisecond)
		m.mu.Lock()
		if d := math.Hypot(victim.p.X-pact.p.X, victim.p.Y-pact.p.Y); d < minD {
			minD = d
		}
		for _, f := range m.fx {
			seen[f.Name]++
			if f.Name == "blood-reel" {
				cost = f.Amount
			}
		}
		now := m.time
		m.mu.Unlock()
		if now-start >= 2.0 {
			break
		}
	}
	if seen["blood-reel"] == 0 {
		t.Fatal("有血线时该起手收线")
	}
	if cost != 8 {
		t.Fatalf("收线该烧 8 血，特效报的是 %v", cost)
	}
	if minD > 41 {
		t.Fatalf("收线该把对面拽到身前 38 以内，最近只到 %.1f", minD)
	}
}

// 中间垫一堵普通墙：收线拉不动，对面顶在墙上——墙后是对手的安全区。
func TestBloodReelBlockedByWall(t *testing.T) {
	m, pact, victim := startPact(t, 47, character.KindDummy)
	lockBoth(m, pact, victim, 34)
	noSeek(m, victim)
	if seen := tickUntil(m, 0.2); seen["blood-bond"] == 0 {
		t.Fatal("该先建上血线")
	}

	// 建完线再立墙、再把对面挪到墙后：这样弧够不到它，只有收线会碰墙。
	m.mu.Lock()
	p0 := pact.p
	ux, uy := inwardUnit(p0.X, p0.Y)
	px, py := -uy, ux // 沿这条线摆横墙
	m.placeWallLocked(unitpkg.PlaceWall{
		OwnerID: pact.id, Slot: 0, Kind: character.KindWaller,
		X1: p0.X + ux*50 - px*40, Y1: p0.Y + uy*50 - py*40,
		X2: p0.X + ux*50 + px*40, Y2: p0.Y + uy*50 + py*40,
		Radius: 6, Life: 30, Amount: 0,
	})
	m.mu.Unlock()
	putAway(m, pact, victim, 90)

	seen := tickUntil(m, 2.0)
	if seen["blood-reel"] == 0 {
		t.Fatal("有血线时该起手收线")
	}
	m.mu.Lock()
	d := math.Hypot(victim.p.X-pact.p.X, victim.p.Y-pact.p.Y)
	m.mu.Unlock()
	if d < 60 {
		t.Fatalf("墙该把收线挡住（对面顶在墙外 50+6+18=74），实际距离 %.1f", d)
	}
}

// 熔断：HP ≤ 10 时耗血技能全熄，只剩免费的基础弧。
// 说明：开局 20ms 里 HP 还是满的，第一批蝙蝠在压血前就已召出去（合法——
// 那一刻还没进熔断区），它们这 1 秒里还会咬两口。所以断言看的是
// 「窗口内没有新的烧血动作」，不是血量差：蝙蝠召发 FX（blood-bats）、
// 饿扑（blood-pounce）、收线（blood-reel）都不该再出现。
func TestBloodBlackoutStopsBurningSkills(t *testing.T) {
	m, pact, victim := startPact(t, 48, character.KindDummy)
	// victim 一开始就摆在 260 远（> 断线 224）：血线建不上、收线无目标，
	// 蝙蝠够不着也就没有回流泵血——熔断线附近没有互相拉锯的干扰。
	// 开局那批烧血（饿扑 5 + 召蝠 6）发生在满血时，合法，不计数。
	putAway(m, pact, victim, 260)
	lockBoth(m, pact, victim, 260)
	tickUntil(m, 0.2) // 让开局那批命令 drain 完、饿扑冲完

	setPactHP(m, pact, 1) // 熔断线 10：回流零头（<10）泵不回熔断线之上
	hp0 := pactHP(m, pact)
	seen := tickUntil(m, 1.0)
	if seen["blood-pounce"] != 0 {
		t.Fatal("熔断时不该饿扑")
	}
	if seen["blood-reel"] != 0 {
		t.Fatal("熔断时不该收线")
	}
	if seen["blood-bats"] != 0 {
		t.Fatal("熔断时不该再召蝙蝠")
	}
	if got := pactHP(m, pact); got < hp0 {
		t.Fatalf("熔断时不该再烧自己的血，%v -> %v", hp0, got)
	}
}

// 熔断只熄烧血技能：免费的咬合照常咬（这条单独测，远离熔断测试的干扰链）。
func TestBloodBiteStillWorksInBlackout(t *testing.T) {
	m, pact, victim := startPact(t, 49, character.KindDummy)
	setPactHP(m, pact, 9) // 熔断区里，蝙蝠/饿扑/收线全熄
	lockBoth(m, pact, victim, 34)
	noSeek(m, victim)
	if seen := tickUntil(m, 0.6); seen["blood-bite"] == 0 {
		t.Fatalf("熔断只熄耗血技能，基础弧该照常咬：%v", seen)
	}
}
