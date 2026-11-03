package sim

import (
	"math"
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func dollOf(m *Match) *unit {
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.kind == character.KindFujihimeDoll {
			return u
		}
	}
	return nil
}

// 藤色 是一条额外血条：挨打整笔被池子吃掉、本体不掉血，打穿那一下多出来的直接无视；
// 池子每少一层就掉一层。这里把整条链跑一遍：小打 → 打穿 → 等幻役接手 → 掉层气绝后整包还回来。
// 结付间隔和每层点数都跟着角色包的参数走，测试只假设「一层 = 一层的池子」。
func TestFujihimeShieldEatsDamageThenDollReturnsIt(t *testing.T) {
	m := NewMatchSeeded(11)
	m.SetSlot(0, character.KindFujihime)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()
	time.Sleep(40 * time.Millisecond)

	screams := 0
	step := func() {
		m.Tick()
		time.Sleep(150 * time.Microsecond)
		m.mu.Lock()
		for _, f := range m.fx {
			if f.Name == "fujihime-scream" {
				screams++
			}
		}
		m.mu.Unlock()
	}
	hit := func(from, to uint64, amount float64) {
		m.mu.Lock()
		m.offerDamageLocked(unitpkg.Damage{From: from, To: to, Amount: amount})
		m.mu.Unlock()
		for i := 0; i < 8; i++ {
			step()
		}
	}
	snapshot := func() (me, foe *unit) {
		m.mu.Lock()
		defer m.mu.Unlock()
		return fighterByKind(m, character.KindFujihime), fighterByKind(m, character.KindDummy)
	}

	// 1) 等开局第一层落地，用它的池子当「一层」的标尺。
	per := 0.0
	for i := 0; i < 20*60 && per <= 0; i++ {
		step()
		if me, _ := snapshot(); me != nil {
			m.mu.Lock()
			per = me.absorb
			m.mu.Unlock()
		}
	}
	if per <= 0 {
		t.Fatal("开局该结付一层")
	}
	me, foe := snapshot()
	if me == nil || foe == nil {
		t.Fatal("missing fighters")
	}
	meID, foeID := me.id, foe.id
	m.mu.Lock()
	hp0 := me.hp
	if markStacksOf(me, "藤色") != 1 {
		m.mu.Unlock()
		t.Fatal("1 层标记没挂上")
	}
	m.mu.Unlock()

	// 2) 小打一下：池子吃掉，本体不掉血，层数不掉。
	small := per / 4
	hit(foeID, meID, small)
	m.mu.Lock()
	me = m.units[meID]
	if me.hp != hp0 {
		m.mu.Unlock()
		t.Fatalf("护盾还在，本体不该掉血：%v → %v", hp0, me.hp)
	}
	if want := per - small; me.absorb != want {
		m.mu.Unlock()
		t.Fatalf("该被吃掉 %v 点，剩 %v（该剩 %v）", small, me.absorb, want)
	}
	if markStacksOf(me, "藤色") != 1 {
		m.mu.Unlock()
		t.Fatal("没掉满一层不该掉层")
	}
	m.mu.Unlock()

	// 3) 打穿：多余的伤害直接无视，本体还是不掉血，层数清零。
	hit(foeID, meID, 999)
	m.mu.Lock()
	me = m.units[meID]
	if me.hp != hp0 {
		m.mu.Unlock()
		t.Fatalf("打穿那一下也不该穿透本体：%v", me.hp)
	}
	if me.absorb != 0 || markStacksOf(me, "藤色") != 0 {
		m.mu.Unlock()
		t.Fatalf("打穿后池子和层数都该是 0：%v / %d", me.absorb, markStacksOf(me, "藤色"))
	}
	m.mu.Unlock()

	// 4) 等到累计结付满次数，幻役接手整包池子。
	var dollID uint64
	var dollLayers int
	var dollPool float64
	for i := 0; i < 90*60 && dollID == 0; i++ {
		step()
		m.mu.Lock()
		if d := dollOf(m); d != nil && d.absorb > 0 {
			dollID, dollLayers, dollPool = d.id, markStacksOf(d, "藤色"), d.absorb
		}
		m.mu.Unlock()
	}
	if dollID == 0 {
		t.Fatal("累计结付够了也没等到幻役")
	}
	if want := int(math.Ceil(dollPool/per - 1e-9)); dollLayers != want {
		t.Fatalf("幻役层数该是 %d（池子 %v / 一层 %v），实际 %d", want, dollPool, per, dollLayers)
	}
	m.mu.Lock()
	if got := m.units[meID].absorb; got != 0 {
		m.mu.Unlock()
		t.Fatalf("池子该整包搬到幻役身上，藤姬剩 %v", got)
	}
	m.mu.Unlock()

	// 5) 打掉一整层：幻役气绝，整包按掉之前的层数还给藤姬。
	hit(foeID, dollID, per)
	m.mu.Lock()
	defer m.mu.Unlock()
	if u := m.units[dollID]; u != nil && !u.stopped {
		t.Fatal("幻役掉一层就该气绝")
	}
	me = m.units[meID]
	if want := float64(dollLayers) * per; me.absorb != want {
		t.Fatalf("幻役的 %d 层该整包还给藤姬（%v），实际 %v", dollLayers, want, me.absorb)
	}
	if markStacksOf(me, "藤色") != dollLayers {
		t.Fatalf("藤姬该拿到 %d 层，实际 %d", dollLayers, markStacksOf(me, "藤色"))
	}
	if screams == 0 {
		t.Fatal("气绝演出没发出来")
	}
}

// 藤色的加伤要作用在藤姬自己的藤弧上：结付一层后，弧打靶子的每一下是 1+层。
func TestFujihimeArcCarriesToseBonus(t *testing.T) {
	m := NewMatchSeeded(5)
	m.SetSlot(0, character.KindFujihime)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()
	time.Sleep(40 * time.Millisecond)

	for i := 0; i < 240; i++ {
		m.Tick()
		time.Sleep(150 * time.Microsecond)
	}
	m.mu.Lock()
	me := fighterByKind(m, character.KindFujihime)
	foe := fighterByKind(m, character.KindDummy)
	stacks := markStacksOf(me, "藤色")
	if stacks != 1 {
		m.mu.Unlock()
		t.Fatalf("跑 4 秒该只有开局那 1 层，实际 %d", stacks)
	}
	var arc *unit
	for _, id := range m.order {
		if u := m.units[id]; u != nil && u.kind == character.KindFujihimeArc {
			arc = u
		}
	}
	if arc == nil {
		m.mu.Unlock()
		t.Fatal("没有藤弧")
	}
	before := foe.hp
	bonus := me.damageBonus
	if bonus <= 0 {
		m.mu.Unlock()
		t.Fatalf("1 层藤色该给持有者加伤，实际 %v", bonus)
	}
	m.offerDamageLocked(unitpkg.Damage{From: arc.id, To: foe.id, Amount: 1})
	var got float64
	for _, off := range m.pendingDmg {
		if off.to == foe.id {
			got = off.amount
		}
	}
	m.mu.Unlock()
	if want := 1 + bonus; got != want {
		t.Fatalf("藤弧的伤害该带上藤色的 +X：1+%v=%v，提议是 %v（靶子 %v）", bonus, want, got, before)
	}
}

// 藤姬的资源包要能真发出去：皮肤、气绝脚本、藤色图标都在包里。
func TestFujihimePackServesFx(t *testing.T) {
	pack, ok := unitpkg.Packs()[character.KindFujihime]
	if !ok || pack.Base != "/ball/藤姬" {
		t.Fatalf("pack=%+v", pack)
	}
	have := map[string]bool{}
	for _, f := range pack.Files {
		have[f] = true
	}
	for _, f := range []string{"fx/fujihime.css", "fx/fujihime.js", "fx/tose.svg"} {
		if !have[f] {
			t.Fatalf("包缺 %s：%v", f, pack.Files)
		}
	}
	fh, err := unitpkg.BallFS().Open("藤姬/fx/tose.svg")
	if err != nil {
		t.Fatalf("藤色图标打不开：%v", err)
	}
	_ = fh.Close()
	look := unitpkg.Looks()[character.KindFujihime]
	if look.Color == "" || len(look.FX) == 0 || look.FX[0] != "fujihime" {
		t.Fatalf("look=%+v", look)
	}
	if arc, ok := unitpkg.Lookup(character.KindFujihimeArc); !ok || arc.ArcSpan < 6.28 {
		t.Fatalf("藤弧该是整圈：%+v", arc)
	}
}

// 幻役的基础伤害是 0，全靠藤色的 +X：固定加伤必须在「非正就丢弃」之前算上。
func TestDamageBonusTurnsZeroBaseIntoPureX(t *testing.T) {
	m := NewMatchSeeded(9)
	m.SetSlot(0, character.KindFujihime)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()
	time.Sleep(40 * time.Millisecond)

	m.mu.Lock()
	me := fighterByKind(m, character.KindFujihime)
	foe := fighterByKind(m, character.KindDummy)
	m.applyCmdLocked(unitpkg.SetDamageBonus{UnitID: me.id, Amount: 3})
	m.offerDamageLocked(unitpkg.Damage{From: me.id, To: foe.id, Amount: 0})
	var got float64
	for _, off := range m.pendingDmg {
		if off.to == foe.id {
			got = off.amount
		}
	}
	m.mu.Unlock()
	if got != 3 {
		t.Fatalf("0 基础伤 + 3 点加伤该变成 3，实际 %v", got)
	}
}
