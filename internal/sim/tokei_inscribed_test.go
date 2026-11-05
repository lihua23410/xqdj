package sim

import (
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

// 时计·铭刻 的整条链：开局 2 张手牌 → 每 10s 抽一张 → 出牌触发冲刺索敌攻击。
// 这里只断言「抽卡 HUD 在动」「冲刺真的打出去了」两件事，具体数值由包内单测覆盖。
func TestInscribedDrawsCardsAndDashes(t *testing.T) {
	m := NewMatchSeeded(17)
	m.SetSlot(0, character.KindInscribed)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()
	time.Sleep(40 * time.Millisecond)

	hp0 := 0.0
	m.mu.Lock()
	if u := fighterByKind(m, character.KindDummy); u != nil {
		hp0 = u.hp
	}
	m.mu.Unlock()
	if hp0 <= 0 {
		t.Fatal("missing dummy")
	}

	cards, dashes, maxCount := 0, 0, 0
	for i := 0; i < 60*60; i++ { // 60 秒模拟时间，够抽 6 张
		m.Tick()
		time.Sleep(120 * time.Microsecond)
		m.mu.Lock()
		for _, f := range m.fx {
			switch f.Name {
			case "tokei-inscribed-cards":
				cards++
				if n := int(f.VY); n > maxCount {
					maxCount = n
				}
			case "tokei-inscribed-dash":
				dashes++
			}
		}
		m.mu.Unlock()
	}

	if cards == 0 {
		t.Fatal("该有手牌 HUD 的演出")
	}
	if maxCount < 3 {
		t.Fatalf("第 10 秒该抽到第 3 张，HUD 最多只见到 %d 张", maxCount)
	}
	if dashes == 0 {
		t.Fatal("出牌该触发冲刺索敌攻击")
	}

	m.mu.Lock()
	foe := fighterByKind(m, character.KindDummy)
	hp1 := 0.0
	if foe != nil {
		hp1 = foe.hp
	}
	m.mu.Unlock()
	if hp1 >= hp0 {
		t.Fatalf("冲刺该打到靶子：%v -> %v", hp0, hp1)
	}
}

// 没有记录时的 曜时之旅/溯时 是死牌：任何回溯演出都必须晚于第一次「铭刻」。
func TestInscribedRewindNeverPrecedesAMark(t *testing.T) {
	m := NewMatchSeeded(19)
	m.SetSlot(0, character.KindInscribed)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()
	time.Sleep(40 * time.Millisecond)

	firstMark, firstRewind := -1, -1
	for i := 0; i < 60*60 && (firstMark < 0 || firstRewind < 0); i++ {
		m.Tick()
		time.Sleep(120 * time.Microsecond)
		m.mu.Lock()
		for _, f := range m.fx {
			switch f.Name {
			case "tokei-inscribed-mark":
				if firstMark < 0 {
					firstMark = i
				}
			case "tokei-inscribed-rewind":
				if firstRewind < 0 {
					firstRewind = i
				}
			}
		}
		m.mu.Unlock()
	}

	if firstRewind >= 0 && firstMark < 0 {
		t.Fatal("没有铭刻就回溯了，死牌不该被打出")
	}
	if firstRewind >= 0 && firstRewind < firstMark {
		t.Fatalf("回溯(%d)早于铭刻(%d)", firstRewind, firstMark)
	}
}

// 铭刻的资源包要能真发出去：皮肤脚本、样式、两张 buff 图标都在包里。
func TestInscribedPackServesFx(t *testing.T) {
	pack, ok := unitpkg.Packs()[character.KindInscribed]
	if !ok || pack.Base != "/ball/时计·铭刻" {
		t.Fatalf("pack=%+v", pack)
	}
	have := map[string]bool{}
	for _, f := range pack.Files {
		have[f] = true
	}
	for _, f := range []string{"fx/tokei_inscribed.css", "fx/tokei_inscribed.js", "fx/trace.svg", "fx/guard.svg"} {
		if !have[f] {
			t.Fatalf("包缺 %s：%v", f, pack.Files)
		}
	}
	look := unitpkg.Looks()[character.KindInscribed]
	if look.Color == "" || len(look.FX) == 0 || look.FX[0] != "tokei-inscribed" {
		t.Fatalf("look=%+v", look)
	}
	spec, ok := unitpkg.Lookup(character.KindInscribed)
	if !ok || spec.MaxHP != 60 || spec.Speed != 165 || spec.Vision != 400 {
		t.Fatalf("spec=%+v", spec)
	}
}
