package sim

import (
	"encoding/json"
	"testing"
	"xqdj/character"
	unitpkg "xqdj/internal/unit"
	圆 "xqdj/map/circle"
	火山 "xqdj/map/volcano"
)

// 上一场收尾时角色协程还没写完的指令，不能落进下一场。
// 症状：火山的减速（M 区 0.5）会"继承"到下一场，因为两场的 ID 都从 1 开始。
func TestEndDiscardsPreviousMatchCommands(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetField(火山.Name)
	m.SetSlot(0, character.KindMelee)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	// 直接塞进队列，模拟"上一场没走完的那一拍"：1 号是小球，减速没带寿命，谁也摘不掉。
	m.cmds <- unitpkg.AddFSComponent{UnitID: 1, Zone: unitpkg.FSZoneM, Token: 999, Value: 0.5}
	m.cmds <- unitpkg.Spawn{Kind: 火山.KindLava, X: 3, Y: 3}
	m.End()

	m.SetField(圆.Name)
	m.Start()
	defer m.End()
	for i := 0; i < 4; i++ {
		m.Tick()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range m.order {
		u := m.units[id]
		if u == nil {
			continue
		}
		if u.kind == 火山.KindLava {
			t.Fatalf("上一场的熔岩出现在下一场：unit %d", id)
		}
		if u.role == unitpkg.RoleFighter {
			if mul := u.cruiseMul(); mul != 1 {
				t.Fatalf("上一场的减速带进下一场：unit %d M=%v", id, mul)
			}
		}
	}
}

// End 之后必须真的收工：队列读空、角色协程全退出（含中途离场的子弹/熔岩那类）。
func TestEndLeavesNoPendingCommands(t *testing.T) {
	m := NewMatchSeeded(2)
	m.SetField(火山.Name)
	m.SetSlot(0, character.KindMelee)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	for i := 0; i < TickHz/2; i++ {
		m.Tick()
	}
	m.mu.Lock()
	helper := m.addUnitLocked(火山.KindLava, vec{10, 10}, vec{}, 0, 0)
	if helper == nil {
		m.mu.Unlock()
		t.Fatal("拿不到一个可离场的单位")
	}
	m.removeLocked(helper) // 已离场：它已经不在 m.units 里了
	us := make([]*unit, 0, len(m.actors))
	us = append(us, m.actors...)
	m.mu.Unlock()
	m.End()
	if n := len(m.cmds); n != 0 {
		t.Fatalf("End 之后队列还剩 %d 条指令", n)
	}
	for _, u := range us {
		select {
		case <-u.exited:
		default:
			t.Fatalf("End 之后 unit %d 的角色协程还没退", u.id)
		}
	}
	select {
	case <-helper.exited:
	default:
		t.Fatal("End 之后已离场单位的协程还没退")
	}
}

// End 之后快照里不能还挂着上一场的特效和顿帧：客户端每帧照单全收重放，
// 留着就会在选人界面、以及下一场开头继续演上一场的东西。
func TestEndClearsLeftoverFXAndHitStop(t *testing.T) {
	m := NewMatchSeeded(4)
	m.SetField(火山.Name)
	m.SetSlot(0, character.KindMelee)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	m.mu.Lock()
	m.cmds <- unitpkg.FX{Name: "erupt", Kind: 火山.KindVolcano, UnitID: 1}
	m.mu.Unlock()
	m.Tick()
	m.mu.Lock()
	if len(m.fx) == 0 {
		m.mu.Unlock()
		t.Fatal("特效没进快照，这个用例就白测了")
	}
	m.hitStop = 2
	m.mu.Unlock()

	m.End()
	var msg struct {
		Effects []unitpkg.FX `json:"effects"`
		HitStop int          `json:"hitStop"`
	}
	if err := json.Unmarshal(m.SnapshotJSON(), &msg); err != nil {
		t.Fatal(err)
	}
	if len(msg.Effects) != 0 {
		t.Fatalf("End 之后快照还挂着 %d 个特效：%+v", len(msg.Effects), msg.Effects)
	}
	if msg.HitStop != 0 {
		t.Fatalf("End 之后 hitStop=%d", msg.HitStop)
	}
}

// 真实场景：第一场在火山里把小球泡在岩浆里减速，第二场换圆，减速不能跟过来。
func TestVolcanoSlowDoesNotFollowIntoNextMatch(t *testing.T) {
	m := NewMatchSeeded(3)
	m.SetField(火山.Name)
	m.SetSlot(0, character.KindMelee)
	m.SetSlot(1, character.KindMelee)
	m.Start()
	m.mu.Lock()
	parked := false
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.role != unitpkg.RoleFighter || u.slot != 0 {
			continue
		}
		// 钉在火山口里：cruise 0、速度 0，只受熔岩的减速
		u.p = vec{0, 0}
		u.setVel(vec{})
		u.cruise = 0
		if u.cruiseFS != nil {
			u.cruiseFS.baseSpeed = 0
		}
		parked = true
	}
	m.mu.Unlock()
	if !parked {
		t.Fatal("没有 0 号小球")
	}
	for i := 0; i < TickHz/2; i++ {
		m.Tick()
	}
	m.mu.Lock()
	slowed := false
	for _, id := range m.order {
		if u := m.units[id]; u != nil && u.role == unitpkg.RoleFighter && u.cruiseMul() != 1 {
			slowed = true
		}
	}
	m.mu.Unlock()
	if !slowed {
		t.Fatal("第一场没吃到火山减速，这个用例就白测了")
	}

	m.End()
	m.SetField(圆.Name)
	m.Start()
	defer m.End()
	for i := 0; i < 4; i++ {
		m.Tick()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.role != unitpkg.RoleFighter {
			continue
		}
		if mul := u.cruiseMul(); mul != 1 {
			t.Fatalf("火山减速跟进了第二场：unit %d M=%v", id, mul)
		}
	}
}
