package 小骑士

import (
	"testing"
	"xqdj/internal/unit"
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
	}
	if !marked {
		t.Fatalf("missing 锁血 mark: %v", cmds)
	}
	k.Handle(ctx, unit.IncomingDamage{Token: 2, Amount: 50, Time: 3})
	if !hasBlock(drain(out)) {
		t.Fatal("lock should block follow-up")
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
	if k.locking || !cleared {
		t.Fatalf("lock should expire locking=%v cleared=%v cmds=%v", k.locking, cleared, cmds)
	}
	k.Handle(ctx, unit.IncomingDamage{Token: 3, Amount: 9, Time: 12.1})
	end := drain(out)
	if !hasConfirm(end) || hasBlock(end) {
		t.Fatalf("second lethal after lock should confirm: %v", end)
	}
}
