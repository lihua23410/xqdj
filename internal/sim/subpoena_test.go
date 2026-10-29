package sim

import (
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

// 传票员挨打 → 站 1 秒 → 朝来源开票 → 票减速飞过去 → 撞上按票面结算。
// 靶子不还手、不挪窝，正好当那张票的收件人；这一局只走这一条链路。
func TestSubpoenaTicketSettlesOnSource(t *testing.T) {
	m := NewMatchSeeded(20261029)
	m.SetSlot(0, character.KindSubpoena)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	t.Cleanup(m.End)

	m.mu.Lock()
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
		m.mu.Unlock()
		t.Fatal("missing fighters")
	}
	// 钉住两只：票要停在 294 px 外，距离得在射程里，而且靶子不能自己走开。
	me.p = vec{0, -120}
	foe.p = vec{0, 120}
	for _, u := range []*unit{me, foe} {
		u.stand = true
		u.setVel(vec{})
		m.applyCmdLocked(unitpkg.SetCruise{UnitID: u.id, Speed: 0})
	}
	foeHP := foe.hp
	m.offerDamageLocked(unitpkg.Damage{From: foe.id, To: me.id, Amount: 10})
	m.mu.Unlock()

	var ticket *unit
	for i := 0; i < 3000 && ticket == nil; i++ {
		m.Tick()
		time.Sleep(200 * time.Microsecond)
		m.mu.Lock()
		for _, id := range m.order {
			u := m.units[id]
			if u != nil && !u.stopped && u.kind == character.KindTicket {
				ticket = u
			}
		}
		m.mu.Unlock()
	}
	if ticket == nil {
		t.Fatal("挨打站完 1 秒该开出一张传票")
	}

	// 票撞上之后还要等靶子那一拍确认伤害，所以不能看到票没了就收工。
	hit, hpNow := false, foeHP
	for i := 0; i < 1200 && !hit; i++ {
		m.Tick()
		time.Sleep(200 * time.Microsecond)
		m.mu.Lock()
		hpNow = foe.hp
		hit = hpNow <= foeHP-10+1e-9
		m.mu.Unlock()
	}
	if !hit {
		t.Fatalf("票该在靶子身上结算 10 点，靶子 hp=%v（开局 %v）", hpNow, foeHP)
	}
}
