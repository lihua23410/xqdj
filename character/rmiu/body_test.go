package r缪

import (
	"testing"
	"xqdj/internal/unit"
)

func cloneAt(id uint64, x, y, hp float64) unit.Snapshot {
	return unit.Snapshot{
		ID: id, Kind: KindMiu, Role: unit.RoleMinion, Slot: 0, OwnerID: 1, Mortal: true,
		X: x, Y: y, HP: hp, MaxHP: miuHP, Radius: miuRadius,
	}
}

func bodySelf(x, y, hp, vx, vy float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindRMiu, Role: unit.RoleFighter, Slot: 0,
		X: x, Y: y, VX: vx, VY: vy, HP: hp, MaxHP: miuHP, Radius: miuRadius,
	}
}

func TestBodyLethalSwapsThenHitsClone(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	r := &R缪{
		booted: true, hp: 8, x: 0, y: 0, vx: 10, vy: 0,
		hasBest: true, bestID: 2, bestHP: 90, bestX: 40, bestY: 0, bestVX: 0, bestVY: 20,
		meleeReadyAt: 999, fireReadyAt: 999,
	}
	ctx := unit.Context{ID: 1, Kind: KindRMiu, Out: out}
	r.Handle(ctx, unit.IncomingDamage{Token: 7, From: 99, Amount: 20})
	cmds := drain(out)

	var block bool
	var dmg *unit.Damage
	var bodyHP, cloneHP *unit.SetHP
	var teleports int
	for _, c := range cmds {
		switch v := c.(type) {
		case unit.BlockDamage:
			block = v.Token == 7
		case unit.ConfirmDamage:
			t.Fatal("lethal must not confirm on body")
		case unit.Damage:
			cp := v
			dmg = &cp
		case unit.SetHP:
			if v.UnitID == 1 {
				cp := v
				bodyHP = &cp
			}
			if v.UnitID == 2 {
				cp := v
				cloneHP = &cp
			}
		case unit.Teleport:
			teleports++
		}
	}
	if !block {
		t.Fatal("missing BlockHit")
	}
	if bodyHP == nil || bodyHP.HP != 90 {
		t.Fatalf("body HP=%v want 90", bodyHP)
	}
	if cloneHP == nil || cloneHP.HP != 8 {
		t.Fatalf("clone should take body HP first, got %v", cloneHP)
	}
	if dmg == nil || dmg.To != 2 || dmg.Amount != 20 {
		t.Fatalf("hit should land on clone after swap: %v", dmg)
	}
	if teleports < 2 {
		t.Fatalf("want two teleports, got %d", teleports)
	}

	// 同帧 Sense 不得再换一次
	r.Handle(ctx, unit.Sense{
		Time:   1,
		Self:   bodySelf(40, 0, 90, 0, 20),
		Nearby: []unit.Snapshot{cloneAt(2, 0, 0, 8)},
	})
	extra := 0
	for _, c := range drain(out) {
		if _, ok := c.(unit.Teleport); ok {
			extra++
		}
	}
	if extra != 0 {
		t.Fatalf("lethal frame must not swap again, extra teleports=%d", extra)
	}
}

func TestBodyChipConfirms(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	r := &R缪{hp: 80, hasBest: true, bestID: 2, bestHP: 90}
	ctx := unit.Context{ID: 1, Kind: KindRMiu, Out: out}
	r.Handle(ctx, unit.IncomingDamage{Token: 3, From: 99, Amount: 10})
	cmds := drain(out)
	ok := false
	for _, c := range cmds {
		if v, okc := c.(unit.ConfirmDamage); okc && v.Token == 3 {
			ok = true
		}
		if _, isDmg := c.(unit.Damage); isDmg {
			t.Fatalf("chip should not redirect: %v", cmds)
		}
	}
	if !ok {
		t.Fatal("chip should ConfirmHit")
	}
}

func TestBodySwapsWhenCloneHasMoreHP(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	r := &R缪{booted: true, speed: miuBaseSpeed, meleeReadyAt: 999, fireReadyAt: 999}
	ctx := unit.Context{ID: 1, Kind: KindRMiu, Out: out}
	r.Handle(ctx, unit.Sense{
		Time:   1,
		Self:   bodySelf(0, 0, 40, 170, 0),
		Nearby: []unit.Snapshot{cloneAt(2, 50, 0, 90)},
	})
	var bodyHP *unit.SetHP
	tele := 0
	for _, c := range drain(out) {
		switch v := c.(type) {
		case unit.SetHP:
			if v.UnitID == 1 {
				cp := v
				bodyHP = &cp
			}
		case unit.Teleport:
			tele++
		}
	}
	if bodyHP == nil || bodyHP.HP != 90 {
		t.Fatalf("body should inherit clone HP, got %v", bodyHP)
	}
	if tele < 2 {
		t.Fatalf("want swap teleports, got %d", tele)
	}
	if r.lockedTargetID != 0 {
		t.Fatal("swap should drop move lock")
	}
}

func TestBodySwapLeavesMarksOnBodies(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	r := &R缪{
		booted: true, speed: miuBaseSpeed, meleeReadyAt: 999, fireReadyAt: 999,
		marks: []unit.Mark{{Kind: "剑痕", Stacks: 2, Icon: "jianhen.png"}},
	}
	ctx := unit.Context{ID: 1, Kind: KindRMiu, Out: out}
	clone := cloneAt(2, 50, 0, 90)
	clone.Marks = []unit.Mark{{Kind: "诅咒", Stacks: 1, Icon: "curse.png"}}
	self := bodySelf(0, 0, 40, 170, 0)
	self.Marks = []unit.Mark{{Kind: "剑痕", Stacks: 2, Icon: "jianhen.png"}}
	r.Handle(ctx, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{clone}})
	cmds := drain(out)

	cleared := map[uint64]bool{}
	got := map[uint64][]unit.StackMark{}
	for _, c := range cmds {
		switch v := c.(type) {
		case unit.ClearMarks:
			cleared[v.UnitID] = true
		case unit.StackMark:
			got[v.UnitID] = append(got[v.UnitID], v)
		}
	}
	if !cleared[1] || !cleared[2] {
		t.Fatalf("both bodies should clear marks: %v", cleared)
	}
	if len(got[1]) != 1 || got[1][0].Kind != "诅咒" || got[1][0].Delta != 1 {
		t.Fatalf("body should keep clone's curse: %v", got[1])
	}
	if len(got[2]) != 1 || got[2][0].Kind != "剑痕" || got[2][0].Delta != 2 {
		t.Fatalf("clone should keep body's 剑痕: %v", got[2])
	}
}
