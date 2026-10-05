package sim

import (
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func startHeroMatch(t *testing.T, seed uint64, foe string) (*Match, *unit, *unit) {
	t.Helper()
	m := NewMatchSeeded(seed)
	m.SetSlot(0, character.KindHero)
	m.SetSlot(1, foe)
	m.Start()
	t.Cleanup(m.End)
	for i := 0; i < 30 && len(unitsOf(m, character.KindHero)) == 0; i++ {
		m.Tick()
		time.Sleep(time.Millisecond)
	}
	heroes := unitsOf(m, character.KindHero)
	if len(heroes) == 0 {
		t.Fatal("missing hero")
	}
	var foeU *unit
	for _, u := range unitsOf(m, foe) {
		foeU = u
	}
	return m, heroes[0], foeU
}

func heroHUD(t *testing.T, m *Match, hero *unit) (maxHP, hp float64) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	return hero.maxHP, hero.hp
}

// 开局撒四件遗物；撞上一件就收编：剑变持有，三位同伴下场跟着。
// （剑的实战效果由 TestHeroSwordHitsInFront 验；这里只验遗物被消耗。）
func TestHeroCollectsRelics(t *testing.T) {
	m, hero, _ := startHeroMatch(t, 11, character.KindDummy)

	// 等开局把遗物撒齐。
	for i := 0; i < 40; i++ {
		if len(unitsOf(m, character.KindSwordPick))+
			len(unitsOf(m, character.KindWarriorPick))+
			len(unitsOf(m, character.KindMagePick))+
			len(unitsOf(m, character.KindSaintPick)) == 4 {
			break
		}
		m.Tick()
		time.Sleep(time.Millisecond)
	}

	type pickup struct {
		pick     string
		follower string
	}
	cases := []pickup{
		{character.KindSwordPick, ""},
		{character.KindWarriorPick, character.KindWarrior},
		{character.KindMagePick, character.KindMage},
		{character.KindSaintPick, character.KindSaint},
	}
	for _, c := range cases {
		found := unitsOf(m, c.pick)
		if len(found) == 0 {
			t.Fatalf("遗物 %s 没撒出来", c.pick)
		}
		// 把勇者直接挪到遗物上，省得等它自己撞。
		m.mu.Lock()
		hero.p = found[0].p
		m.mu.Unlock()
		for i := 0; i < 20 && len(unitsOf(m, c.pick)) > 0; i++ {
			m.Tick()
			time.Sleep(2 * time.Millisecond)
		}
		if n := len(unitsOf(m, c.pick)); n != 0 {
			t.Fatalf("%s 该被捡走，还剩 %d", c.pick, n)
		}
		if c.follower == "" {
			continue
		}
		for i := 0; i < 20 && len(unitsOf(m, c.follower)) == 0; i++ {
			m.Tick()
			time.Sleep(2 * time.Millisecond)
		}
		if n := len(unitsOf(m, c.follower)); n == 0 {
			t.Fatalf("捡了 %s 之后该出现 %s", c.pick, c.follower)
		}
	}
}

// 等级：lv1 要 2^1 = 2 点经验；升上去回满到旧上限、上限 +10。
func TestHeroLevelsUpOnLiveMinionKill(t *testing.T) {
	m, hero, _ := startHeroMatch(t, 12, character.KindDummy)
	beforeMax, _ := heroHUD(t, m, hero)
	if beforeMax != 100 {
		t.Fatalf("开局上限该是 100，got %v", beforeMax)
	}

	kill := func() {
		m.mu.Lock()
		m.send(hero, unitpkg.Kill{
			VictimID: 999, VictimKind: character.KindSlime,
			VictimRole: unitpkg.RoleMinion, Mortal: true,
		})
		m.mu.Unlock()
		m.Tick()
		time.Sleep(2 * time.Millisecond)
	}

	// 第 1 点经验：不该升。
	kill()
	if got, _ := heroHUD(t, m, hero); got != 100 {
		t.Fatalf("1 点经验不该升级，上限 %v", got)
	}
	// 第 2 点经验：升到 lv2，上限 110、血回到旧的满值 100。
	kill()
	gotMax, gotHP := heroHUD(t, m, hero)
	if gotMax != 110 {
		t.Fatalf("lv2 上限该是 110，got %v", gotMax)
	}
	if gotHP != 100 {
		t.Fatalf("升级按字面该回满到旧上限 100（例子里是 100/110），got %v", gotHP)
	}

	// 被非活随从击杀不算经验。
	m.mu.Lock()
	m.send(hero, unitpkg.Kill{VictimID: 998, VictimKind: "墙", Mortal: false})
	m.mu.Unlock()
	m.Tick()
	if got, _ := heroHUD(t, m, hero); got != 110 {
		t.Fatalf("非活随从不该给经验，上限 %v", got)
	}
}

// 史莱姆只咬勇者、不咬敌人；这是靠把它的 Slot 设成敌人的槽实现的。
func TestSlimeOnlyBitesTheHero(t *testing.T) {
	m, hero, foe := startHeroMatch(t, 13, character.KindDummy)
	if foe == nil {
		t.Fatal("missing foe")
	}

	m.mu.Lock()
	// 把勇者钉住：史莱姆不追人，只有贴着才咬得到。
	hero.cruise = 0
	hero.stand = true
	hero.setVel(vec{})
	// 放在 50 外、朝勇者漂过来——它不追人，靠漂。别叠着放：
	// Pass 要到它第一拍感知才挂上，叠着生成会被引擎当实心顶开。
	slime := m.addUnitLocked(character.KindSlime, vec{hero.p.X + 50, hero.p.Y}, vec{-165, 0}, hero.id, foe.slot)
	heroHP0, foeHP0 := hero.hp, foe.hp
	m.mu.Unlock()
	if slime == nil {
		t.Fatal("add slime failed")
	}

	for i := 0; i < 90; i++ {
		m.Tick()
		time.Sleep(1500 * time.Microsecond)
	}

	m.mu.Lock()
	heroHP, foeHP := hero.hp, foe.hp
	m.mu.Unlock()
	if heroHP >= heroHP0 {
		t.Fatalf("史莱姆该咬到勇者: %v -> %v", heroHP0, heroHP)
	}
	if foeHP != foeHP0 {
		t.Fatalf("史莱姆不该咬到敌人: %v -> %v", foeHP0, foeHP)
	}
}

// 史莱姆不追人、也没有碰撞体积：常驻 Pass（单位之间不推不弹，墙照样挡），
// 速度只由初速和撞墙决定，不会朝勇者拐弯。
func TestSlimeDriftsAndHoldsPass(t *testing.T) {
	m, hero, foe := startHeroMatch(t, 14, character.KindDummy)

	m.mu.Lock()
	// 放场地中心附近、给一个朝 +x 的初速，短时间里撞不到墙。
	slime := m.addUnitLocked(character.KindSlime, vec{-40, 0}, vec{165, 0}, hero.id, foe.slot)
	slime.p = vec{-40, 0}
	slime.setVel(vec{165, 0})
	m.mu.Unlock()
	if slime == nil {
		t.Fatal("add slime failed")
	}

	m.Tick()
	time.Sleep(2 * time.Millisecond)
	// Pass 是角色在收到第一拍感知时才发的，指令要等下一拍才落到场上。
	gotPass := false
	for i := 0; i < 5 && !gotPass; i++ {
		m.Tick()
		time.Sleep(2 * time.Millisecond)
		m.mu.Lock()
		gotPass = slime.pass
		m.mu.Unlock()
	}
	if !gotPass {
		t.Fatal("史莱姆该常驻 Pass：它不能有碰撞体积")
	}
	m.mu.Lock()
	before := slime.v
	m.mu.Unlock()

	// 20 拍（约 0.33 秒、走 ~55）还撞不到墙，速度方向不该变。
	for i := 0; i < 20; i++ {
		m.Tick()
		time.Sleep(1500 * time.Microsecond)
	}
	m.mu.Lock()
	after := slime.v
	m.mu.Unlock()
	if after != before {
		t.Fatalf("史莱姆该直线漂、不追人: v %v -> %v", before, after)
	}
}

// 村好剑是「手里握着的一件武器」，不是引擎的弧单位：勇者自己算身前扇区，砍到就掉血。
// 顺手验它是**有方向**的——背后的人不该挨。
func TestHeroSwordHitsInFront(t *testing.T) {
	m, hero, foe := startHeroMatch(t, 15, character.KindDummy)

	// startHeroMatch 只等勇者本体出现；遗物是它第一拍感知才撒的，先等它们出来。
	for i := 0; i < 40 && len(unitsOf(m, character.KindSwordPick)) == 0; i++ {
		m.Tick()
		time.Sleep(time.Millisecond)
	}
	// 剑只可能因为被捡走而消失；这时候还没消失就先把它捡了，保证手上一定有剑。
	if sword := unitsOf(m, character.KindSwordPick); len(sword) > 0 {
		m.mu.Lock()
		hero.p = sword[0].p
		m.mu.Unlock()
		for i := 0; i < 20 && len(unitsOf(m, character.KindSwordPick)) > 0; i++ {
			m.Tick()
			time.Sleep(2 * time.Millisecond)
		}
		if len(unitsOf(m, character.KindSwordPick)) != 0 {
			t.Fatal("剑该被捡走")
		}
	}

	// 面朝 +x：正前方 40 一只、背后 40 一只（都在够到的 44 内，只有前方在扇区内）。
	// 放远一点避免叠着生成——实心那一拍会被引擎顶开。
	m.mu.Lock()
	hero.setVel(vec{170, 0})
	front := m.addUnitLocked(character.KindSlime, vec{hero.p.X + 40, hero.p.Y}, vec{}, hero.id, foe.slot)
	back := m.addUnitLocked(character.KindSlime, vec{hero.p.X - 40, hero.p.Y}, vec{}, hero.id, foe.slot)
	if front == nil || back == nil {
		m.mu.Unlock()
		t.Fatal("setup failed")
	}
	frontHP, backHP := front.hp, back.hp
	m.mu.Unlock()

	for i := 0; i < 8; i++ {
		m.Tick()
		time.Sleep(2 * time.Millisecond)
	}

	m.mu.Lock()
	f, b := front.hp, back.hp
	m.mu.Unlock()
	if f >= frontHP {
		t.Fatalf("身前的人该被剑砍到: %v -> %v", frontHP, f)
	}
	if b != backHP {
		t.Fatalf("背后的人不该挨: %v -> %v", backHP, b)
	}
}
