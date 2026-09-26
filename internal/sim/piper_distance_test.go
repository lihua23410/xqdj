package sim

import (
	"encoding/json"
	"math"
	"testing"

	"xqdj/character"
)

// 吹笛人的老鼠不该跟同类、跟本体挤成一团，也不该贴着敌人蹭：跑一场真对局数一数。
// 观感问题的量化口径：把「老鼠圆心落在本体圆里」「两只老鼠互相重叠」「贴到敌人球面 1 以内」
// 这三种帧数出来，按老鼠·帧占比卡上限。
func TestPiperRatsKeepDistance(t *testing.T) {
	m := NewMatchSeeded(7)
	m.SetSlot(0, character.KindPiper)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()

	var msg struct {
		Units []struct {
			ID     uint64  `json:"id"`
			Kind   string  `json:"kind"`
			X      float64 `json:"x"`
			Y      float64 `json:"y"`
			Radius float64 `json:"radius"`
			HP     float64 `json:"hp"`
		} `json:"units"`
	}
	type pos struct {
		x, y, r float64
	}
	var (
		ratFrames, touchFrames, ratOverlap, piperOverlap int
		lastHP                                           float64
		first                                            = true
		bites                                            int
	)
	for i := 0; i < 60*40; i++ {
		m.Tick()
		if err := json.Unmarshal(m.SnapshotJSON(), &msg); err != nil {
			t.Fatal(err)
		}
		var ex, ey, er, px, py, pr float64
		var rats []pos
		for _, u := range msg.Units {
			switch u.Kind {
			case character.KindDummy:
				ex, ey, er = u.X, u.Y, u.Radius
				if first {
					lastHP, first = u.HP, false
				} else if u.HP < lastHP-1e-9 {
					bites++
					lastHP = u.HP
				}
			case character.KindPiper:
				px, py, pr = u.X, u.Y, u.Radius
			case character.KindRat:
				rats = append(rats, pos{u.X, u.Y, u.Radius})
			}
		}
		for _, r := range rats {
			ratFrames++
			if math.Hypot(r.x-ex, r.y-ey) <= er+r.r+1 {
				touchFrames++
			}
			if math.Hypot(r.x-px, r.y-py) < pr+r.r-1 {
				piperOverlap++
			}
		}
		for a := 0; a < len(rats); a++ {
			for b := a + 1; b < len(rats); b++ {
				if math.Hypot(rats[a].x-rats[b].x, rats[a].y-rats[b].y) < rats[a].r+rats[b].r-1 {
					ratOverlap++
				}
			}
		}
	}
	if ratFrames == 0 {
		t.Fatal("整场没有老鼠出场")
	}
	if bites == 0 {
		t.Fatal("一次都没咬到：指挥没接上")
	}
	t.Logf("老鼠·帧=%d 咬中=%d 贴敌人=%d 压本体=%d 鼠互相重叠=%d", ratFrames, bites, touchFrames, piperOverlap, ratOverlap)
	// 贴着敌人应当是「扑上去咬一口」的瞬间，不是常态。
	if touchFrames*100 > ratFrames*5 {
		t.Fatalf("老鼠贴敌人太久：%d/%d 老鼠·帧", touchFrames, ratFrames)
	}
	// 老鼠之间靠各有各的角度和互相排斥分开，不该堆成一坨。
	// （本体压过老鼠是允许的：老鼠常驻 Pass，谁都能穿过它。）
	if ratOverlap*100 > ratFrames*5 {
		t.Fatalf("老鼠互相重叠太多：%d/%d 老鼠·帧", ratOverlap, ratFrames)
	}
}
