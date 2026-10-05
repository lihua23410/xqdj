package sim

import (
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

// 击杀记功：勇者要能知道「我杀了一个活随从」，引擎原来不发这个事实。
func TestKillCreditsKillerOrItsOwner(t *testing.T) {
	m := NewMatchSeeded(31)
	m.SetSlot(0, character.KindMelee)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()
	time.Sleep(20 * time.Millisecond)

	m.mu.Lock()
	defer m.mu.Unlock()
	// 自己找 slot 0 的战斗机：不依赖别的测试文件里的辅助函数（那些文件可能不在）。
	var hero *unit
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.role == unitpkg.RoleFighter && u.slot == 0 {
			hero = u
			break
		}
	}
	if hero == nil {
		t.Fatal("missing fighter")
	}

	// 1) 亲自动手杀的：记到勇者头上，并且认得出是活随从。
	victim := m.addUnitLocked(character.KindRat, vec{0, 0}, vec{}, hero.id, 0)
	if victim == nil {
		t.Fatal("add minion failed")
	}
	got := tapEvents(hero)
	m.applyHurtLocked(hero, victim, 999, false, false)
	kills := eventsOf[unitpkg.Kill](got)
	if len(kills) != 1 {
		t.Fatalf("亲手击杀该记 1 笔，got %v", kills)
	}
	if kills[0].VictimID != victim.id || kills[0].VictimKind != character.KindRat || !kills[0].Mortal {
		t.Fatalf("击杀内容不对: %+v", kills[0])
	}

	// 2) 自己的弹杀的：记到主人（勇者）头上，不是记给那颗弹。
	victim2 := m.addUnitLocked(character.KindRat, vec{20, 0}, vec{}, hero.id, 0)
	shot := m.addUnitLocked(character.KindMiuShot, vec{10, 0}, vec{}, hero.id, 0)
	if shot == nil {
		t.Fatal("add shot failed")
	}
	got2 := tapEvents(hero)
	shotGot := tapEvents(shot)
	m.applyHurtLocked(shot, victim2, 999, false, false)
	if kills := eventsOf[unitpkg.Kill](got2); len(kills) != 1 || kills[0].VictimID != victim2.id {
		t.Fatalf("弹杀该记给主人: %v", kills)
	}
	if kills := eventsOf[unitpkg.Kill](shotGot); len(kills) != 0 {
		t.Fatalf("不该记给那颗弹: %v", kills)
	}

	// 3) 没打死不发。
	victim3 := m.addUnitLocked(character.KindRat, vec{-20, 0}, vec{}, hero.id, 0)
	got3 := tapEvents(hero)
	if victim3.hp <= 1 {
		t.Fatalf("这个活随从血太少，测不出「没打死」: hp=%v", victim3.hp)
	}
	m.applyHurtLocked(hero, victim3, 1, false, false)
	if kills := eventsOf[unitpkg.Kill](got3); len(kills) != 0 {
		t.Fatalf("没打死不该发击杀: %v", kills)
	}

	// 4) 环境致死（没有来源）不发。
	got4 := tapEvents(hero)
	m.applyHurtLocked(nil, victim3, 999, false, false)
	if kills := eventsOf[unitpkg.Kill](got4); len(kills) != 0 {
		t.Fatalf("环境致死不该发击杀: %v", kills)
	}
}

// 发起者在确认之前就自毁了（剑弧、子弹都是这个形态），击杀仍要记到主人头上。
// 这是实战里踩到的坑：等确认时才去找发起者，m.units[from] 已经没了，击杀就丢。
func TestKillCreditSurvivesDespawnedKiller(t *testing.T) {
	m := NewMatchSeeded(32)
	m.SetSlot(0, character.KindMelee)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()
	time.Sleep(20 * time.Millisecond)

	m.mu.Lock()
	defer m.mu.Unlock()
	var hero *unit
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.role == unitpkg.RoleFighter && u.slot == 0 {
			hero = u
			break
		}
	}
	if hero == nil {
		t.Fatal("missing fighter")
	}

	victim := m.addUnitLocked(character.KindRat, vec{0, 0}, vec{}, 0, 0)
	shot := m.addUnitLocked(character.KindMiuShot, vec{10, 0}, vec{}, hero.id, 0)
	if victim == nil || shot == nil {
		t.Fatal("setup failed")
	}
	got := tapEvents(hero)

	m.offerDamageLocked(unitpkg.Damage{From: shot.id, To: victim.id, Amount: 999})
	// 发起者发完伤害就当拍自毁。
	m.removeLocked(shot)
	// 目标这一拍才确认。
	var token uint64
	for tok, off := range m.pendingDmg {
		if off.to == victim.id {
			token = tok
		}
	}
	if token == 0 {
		t.Fatal("没找到待确认的伤害")
	}
	m.confirmDamageLocked(unitpkg.ConfirmDamage{Token: token, UnitID: victim.id, Amount: 999})

	kills := eventsOf[unitpkg.Kill](got)
	if len(kills) != 1 || kills[0].VictimID != victim.id || !kills[0].Mortal {
		t.Fatalf("发起者自毁后，击杀仍该记给主人: %v", kills)
	}
}
