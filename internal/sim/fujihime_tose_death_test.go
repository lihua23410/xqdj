package sim

import (
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

// 藤色 挂在队友身上时，藤姬死了它也该继续生效：护盾池、每层 +X、标记都不该跟着主人一起消失。
// （挂在幻役身上是另一回事：幻役是藤姬名下的单位，主人没了它本来就一起没。）
func TestToseKeepsWorkingAfterFujihimeDies(t *testing.T) {
	m := NewMatchSeeded(23)
	m.UseTeam()
	m.mu.Lock()
	m.slots[0] = []string{character.KindFujihime, character.KindTokei}
	m.slots[1] = []string{character.KindDummy}
	m.mu.Unlock()
	m.Start()
	defer m.End()
	time.Sleep(40 * time.Millisecond)

	step := func() {
		m.Tick()
		time.Sleep(150 * time.Microsecond)
	}

	var (
		absorb   float64
		bonus    float64
		layers   int
		buddyID  uint64
		fujiID   uint64
		inheried bool
	)
	for i := 0; i < 60*120 && !inheried; i++ { // 最多 120s 模拟时间
		step()
		m.mu.Lock()
		fuji := fighterByKind(m, character.KindFujihime)
		buddy := fighterByKind(m, character.KindTokei)
		if fuji != nil && buddy != nil && buddy.absorb > 0 {
			buddyID, fujiID = buddy.id, fuji.id
			absorb, bonus = buddy.absorb, buddy.damageBonus
			for _, mk := range buddy.marks {
				if mk.kind == "藤色" {
					layers = mk.stacks
				}
			}
			inheried = true
		}
		m.mu.Unlock()
	}
	if !inheried || layers <= 0 {
		t.Fatalf("队友一直没拿到藤色（absorb=%v layers=%d），这一例没测到东西", absorb, layers)
	}

	// 藤姬倒下。
	m.mu.Lock()
	if u := m.units[fujiID]; u != nil {
		m.removeLocked(u)
	}
	m.mu.Unlock()
	for i := 0; i < 10; i++ {
		step()
	}

	m.mu.Lock()
	buddy := m.units[buddyID]
	if buddy == nil || buddy.stopped {
		m.mu.Unlock()
		t.Fatal("队友不该跟着藤姬一起没")
	}
	gotAbsorb, gotBonus := buddy.absorb, buddy.damageBonus
	gotLayers := 0
	for _, mk := range buddy.marks {
		if mk.kind == "藤色" {
			gotLayers = mk.stacks
		}
	}
	m.mu.Unlock()

	if gotAbsorb != absorb {
		t.Fatalf("藤姬死后池子变了：%v（死前 %v）", gotAbsorb, absorb)
	}
	if gotBonus != bonus {
		t.Fatalf("藤姬死后加伤变了：%v（死前 %v）", gotBonus, bonus)
	}
	if gotLayers != layers {
		t.Fatalf("藤姬死后藤色层数变了：%d（死前 %d）", gotLayers, layers)
	}

	// 池子还在，就还得真的吃伤：打一笔小于池子的，本体不掉血。
	m.mu.Lock()
	hpBefore := buddy.hp
	dummy := fighterByKind(m, character.KindDummy)
	dummyID := uint64(0)
	if dummy != nil {
		dummyID = dummy.id
	}
	m.mu.Unlock()
	if dummyID == 0 {
		t.Fatal("靶子没了")
	}
	m.mu.Lock()
	m.offerDamageLocked(unitpkg.Damage{From: dummyID, To: buddyID, Amount: absorb / 2})
	m.mu.Unlock()
	for i := 0; i < 8; i++ {
		step()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if buddy.hp != hpBefore {
		t.Fatalf("藤姬死后护盾不吃伤了：本体 %v -> %v", hpBefore, buddy.hp)
	}
	if buddy.absorb >= absorb {
		t.Fatalf("藤姬死后护盾没被扣：%v（原 %v）", buddy.absorb, absorb)
	}
}
