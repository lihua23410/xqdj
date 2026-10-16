package 小骑士

import (
	"testing"
	"xqdj/internal/unit"
	_ "xqdj/map/hex"
)

func drain(out <-chan unit.Cmd) []unit.Cmd {
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

func hasBlock(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if _, ok := c.(unit.BlockDamage); ok {
			return true
		}
	}
	return false
}

func hasFX(cmds []unit.Cmd, name string) bool {
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == name {
			return true
		}
	}
	return false
}

func hasConfirm(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if _, ok := c.(unit.ConfirmDamage); ok {
			return true
		}
	}
	return false
}

func TestNonLethalConfirms(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	k := &小骑士{hp: 85}
	ctx := unit.Context{ID: 1, Kind: KindKnight, Out: out}
	k.Handle(ctx, unit.IncomingDamage{Token: 1, Amount: 10, Time: 1})
	cmds := drain(out)
	if !hasConfirm(cmds) || hasBlock(cmds) {
		t.Fatalf("non-lethal should confirm: %v", cmds)
	}
}

func TestFirstLethalLocksHP(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	k := &小骑士{hp: 8}
	ctx := unit.Context{ID: 1, Kind: KindKnight, Out: out}
	k.Handle(ctx, unit.IncomingDamage{Token: 1, Amount: 20, Time: 2})
	cmds := drain(out)
	if !hasBlock(cmds) || hasConfirm(cmds) {
		t.Fatalf("first lethal should block: %v", cmds)
	}
	if !k.usedLock || !k.locking || k.lockUntil != 2+lockHP {
		t.Fatalf("lock state used=%v locking=%v until=%v", k.usedLock, k.locking, k.lockUntil)
	}
	marked := false
	for _, c := range cmds {
		if v, ok := c.(unit.StackMark); ok && v.Kind == markLock && v.Delta == 1 {
			marked = true
		}
		if fx, ok := c.(unit.FX); ok && fx.Name == "lock-hit" {
			t.Fatalf("opening lock should not splash: %v", cmds)
		}
	}
	if !marked || !hasFX(cmds, "lock") {
		t.Fatalf("missing 锁血 mark or lock fx: %v", cmds)
	}
	k.Handle(ctx, unit.IncomingDamage{Token: 2, Amount: 50, Time: 3})
	follow := drain(out)
	if !hasBlock(follow) || !hasFX(follow, "lock-hit") {
		t.Fatalf("lock should block follow-up and splash: %v", follow)
	}
}

func TestLockExpiresAndSecondLethalDies(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	k := &小骑士{hp: 5, usedLock: true, locking: true, lockUntil: 12}
	ctx := unit.Context{ID: 1, Kind: KindKnight, Out: out}
	k.Handle(ctx, unit.Sense{Time: 12, Self: unit.Snapshot{ID: 1, HP: 5, X: 0, Y: 0}})
	cmds := drain(out)
	cleared := false
	for _, c := range cmds {
		if v, ok := c.(unit.ClearMarks); ok && v.Kind == markLock {
			cleared = true
		}
	}
	if k.locking || !cleared || !hasFX(cmds, "lock-end") {
		t.Fatalf("lock should expire locking=%v cleared=%v cmds=%v", k.locking, cleared, cmds)
	}
	k.Handle(ctx, unit.IncomingDamage{Token: 3, Amount: 9, Time: 12.1})
	end := drain(out)
	if !hasConfirm(end) || hasBlock(end) {
		t.Fatalf("second lethal after lock should confirm: %v", end)
	}
}

func TestBlinkFXCarriesLanding(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	k := &小骑士{hp: 85}
	ctx := unit.Context{ID: 1, Kind: KindKnight, Out: out}
	self := unit.Snapshot{ID: 1, Role: unit.RoleFighter, Slot: 0, X: 0, Y: 0, Radius: knightRadius, HP: 85}
	enemy := unit.Snapshot{
		ID: 2, Role: unit.RoleFighter, Slot: 1,
		X: 80, Y: 0, Radius: 18, AimPriority: unit.DefaultFighterAim,
	}
	k.Handle(ctx, unit.Sense{Time: 6, Self: self, Nearby: []unit.Snapshot{enemy}})
	cmds := drain(out)
	var blink unit.FX
	var tp unit.Teleport
	gotBlink, gotTP := false, false
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == "blink" {
			blink, gotBlink = fx, true
		}
		if v, ok := c.(unit.Teleport); ok {
			tp, gotTP = v, true
		}
	}
	if !gotBlink || !gotTP {
		t.Fatalf("blink/teleport missing: %v", cmds)
	}
	if blink.X != self.X || blink.Y != self.Y {
		t.Fatalf("blink origin (%v,%v) want (%v,%v)", blink.X, blink.Y, self.X, self.Y)
	}
	if blink.VX != tp.X || blink.VY != tp.Y {
		t.Fatalf("blink landing (%v,%v) teleport (%v,%v)", blink.VX, blink.VY, tp.X, tp.Y)
	}
	if !hasFX(cmds, "dash") {
		t.Fatalf("landing dash missing: %v", cmds)
	}
}
