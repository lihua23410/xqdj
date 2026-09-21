package 面灵气

import (
	"math"
	"testing"
	"xqdj/internal/unit"
)

func TestMaskOrbitHitsMortalMinion(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	m := &面灵气{hook: 1}
	ctx := unit.Context{ID: 1, Kind: KindMenreiki, Out: out}
	orbit := menreikiRadius + maskRadius + maskOrbitPad
	self := unit.Snapshot{
		ID: 1, Kind: KindMenreiki, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: menreikiRadius, Faction: unit.FactionCyan,
	}
	minion := unit.Snapshot{
		ID: 8, Kind: "教父暗杀者", Role: unit.RoleMinion, Mortal: true, Slot: 1,
		X: orbit, Y: 0, Radius: 14, HP: 30, MaxHP: 30,
	}
	m.Handle(ctx, unit.Sense{Time: 0, Self: self, Nearby: []unit.Snapshot{minion}})
	if !hasDamageTo(drain(out), 8) {
		t.Fatal("mask overlap should damage mortal minion")
	}
}

func drain(out chan unit.Cmd) []unit.Cmd {
	var cmds []unit.Cmd
	for {
		select {
		case c := <-out:
			cmds = append(cmds, c)
		default:
			return cmds
		}
	}
}

func hasDamageTo(cmds []unit.Cmd, to uint64) bool {
	for _, c := range cmds {
		if d, ok := c.(unit.Damage); ok && d.To == to {
			return true
		}
	}
	return false
}

func TestMaskHitGateIndependent(t *testing.T) {
	const owner uint64 = 7
	if spendMask(owner, 0) {
		t.Fatal("unarmed should not hit")
	}
	armMask(owner, 0)
	if !spendMask(owner, 0) {
		t.Fatal("armed mask 0 should hit")
	}
	if spendMask(owner, 0) {
		t.Fatal("spent mask 0 should not hit again")
	}
	if spendMask(owner, 1) {
		t.Fatal("mask 1 should stay independent")
	}
	armAllMasks(owner)
	for i := 0; i < 3; i++ {
		if !spendMask(owner, i) {
			t.Fatalf("arm all: mask %d", i)
		}
	}
}

func TestCyanCollectPinsWithoutWritingCruise(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	m := &面灵气{hook: 2}
	ctx := unit.Context{ID: 1, Kind: KindMenreiki, Out: out}
	self := unit.Snapshot{
		ID: 1, Kind: KindMenreiki, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: menreikiRadius,
	}
	enemy := unit.Snapshot{
		ID: 2, Kind: "筑墙者", Role: unit.RoleFighter, Slot: 1,
		X: 80, Y: 0, Radius: 18, Faction: unit.FactionCyan,
		Seen: unit.AllFactions(), AimPriority: unit.DefaultFighterAim,
	}
	m.Handle(ctx, unit.Sense{Time: 10, Self: self, Nearby: []unit.Snapshot{enemy}})
	hit := drain(out)
	until := 10 + stunSecs
	st := lastStun(hit)
	if st == nil || st.UnitID != 2 || !st.Hold || math.Abs(st.Until-until) > 1e-9 {
		t.Fatalf("stun=%v cmds=%v", st, hit)
	}
	pin := lastAddFSComponent(hit, 2)
	if pin == nil || pin.Zone != unit.FSZoneM || pin.Value != 0 || math.Abs(pin.ExpiresAt-until) > 1e-9 {
		t.Fatalf("pin=%v cmds=%v", pin, hit)
	}
	if lastCruise(hit, 2) != nil || lastVel(hit, 2) != nil {
		t.Fatalf("must not write target cruise/vel: %v", hit)
	}
	if lastNamed(hit, "stun") == nil {
		t.Fatalf("missing stun fx: %v", hit)
	}

	m.Handle(ctx, unit.Sense{Time: 10.2, Self: self, Nearby: []unit.Snapshot{enemy}})
	mid := drain(out)
	if lastStun(mid) != nil || lastCruise(mid, 2) != nil || lastVel(mid, 2) != nil {
		t.Fatalf("must not babysit pin: %v", mid)
	}
	if lastNamed(mid, "stun") == nil {
		t.Fatalf("stun overlay should keep ticking: %v", mid)
	}

	m.Handle(ctx, unit.Sense{Time: until, Self: self, Nearby: []unit.Snapshot{enemy}})
	end := drain(out)
	if lastStun(end) != nil || lastCruise(end, 2) != nil || lastVel(end, 2) != nil {
		t.Fatalf("engine owns release, cmds=%v", end)
	}
}

func TestCyanLostTargetClearsPin(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	m := &面灵气{hook: 2}
	ctx := unit.Context{ID: 1, Kind: KindMenreiki, Out: out}
	self := unit.Snapshot{
		ID: 1, Kind: KindMenreiki, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: menreikiRadius,
	}
	enemy := unit.Snapshot{
		ID: 2, Kind: "筑墙者", Role: unit.RoleFighter, Slot: 1,
		X: 80, Y: 0, Radius: 18, Faction: unit.FactionCyan,
		Seen: unit.AllFactions(), AimPriority: unit.DefaultFighterAim,
	}
	m.Handle(ctx, unit.Sense{Time: 10, Self: self, Nearby: []unit.Snapshot{enemy}})
	_ = drain(out)
	m.Handle(ctx, unit.Sense{Time: 10.2, Self: self})
	end := drain(out)
	if lastRemoveFSComponent(end, 2) == nil {
		t.Fatalf("lost target should lift pin: %v", end)
	}
	st := lastStun(end)
	if st == nil || st.Hold {
		t.Fatalf("lost target should drop stun: %v", end)
	}
}

func lastStun(cmds []unit.Cmd) *unit.Stun {
	var st *unit.Stun
	for _, c := range cmds {
		if v, ok := c.(unit.Stun); ok {
			cp := v
			st = &cp
		}
	}
	return st
}

func lastVel(cmds []unit.Cmd, id uint64) *unit.SetVelocity {
	var v *unit.SetVelocity
	for _, c := range cmds {
		if x, ok := c.(unit.SetVelocity); ok && x.UnitID == id {
			cp := x
			v = &cp
		}
	}
	return v
}

func lastCruise(cmds []unit.Cmd, id uint64) *unit.SetCruise {
	var v *unit.SetCruise
	for _, c := range cmds {
		if x, ok := c.(unit.SetCruise); ok && x.UnitID == id {
			cp := x
			v = &cp
		}
	}
	return v
}

func lastAddFSComponent(cmds []unit.Cmd, id uint64) *unit.AddFSComponent {
	var v *unit.AddFSComponent
	for _, c := range cmds {
		if x, ok := c.(unit.AddFSComponent); ok && x.UnitID == id {
			cp := x
			v = &cp
		}
	}
	return v
}

func lastRemoveFSComponent(cmds []unit.Cmd, id uint64) *unit.RemoveFSComponent {
	var v *unit.RemoveFSComponent
	for _, c := range cmds {
		if x, ok := c.(unit.RemoveFSComponent); ok && x.UnitID == id {
			cp := x
			v = &cp
		}
	}
	return v
}

func lastNamed(cmds []unit.Cmd, name string) *unit.FX {
	var fx *unit.FX
	for _, c := range cmds {
		if v, ok := c.(unit.FX); ok && v.Name == name {
			cp := v
			fx = &cp
		}
	}
	return fx
}
