package 老虎机

import (
	"math"
	"testing"

	"xqdj/internal/unit"
)

func TestDamageStartsFiveSecondInvulnerableDraw(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	m := &slotMachine{rng: unit.NewRNG(1)}
	ctx := unit.Context{ID: 1, Kind: KindSlot, Out: out}
	m.Handle(ctx, unit.IncomingDamage{Token: 7, From: 2, Amount: 10, Time: 1})
	cmds := drain(out)
	if !hasConfirm(cmds, 7) {
		t.Fatal("triggering damage should still be confirmed")
	}
	if !m.drawing || m.drawUntil != 6 {
		t.Fatalf("drawing=%v until=%v, want five seconds", m.drawing, m.drawUntil)
	}
	if !hasFX(cmds, "slot-draw") {
		t.Fatal("slot-draw FX missing")
	}
	m.Handle(ctx, unit.IncomingDamage{Token: 8, From: 2, Amount: 10, Time: 2})
	if !hasBlock(drain(out), 8) {
		t.Fatal("damage during draw should be blocked")
	}
}

func TestNonTripleFiresThreeMatchingSymbolBullets(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	m := &slotMachine{
		slot: 0, rng: unit.NewRNG(1), reels: [3]int{symbolCrown, symbolCoin, symbolCherry},
		x: 0, y: 0, hx: 1,
	}
	target := unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 100, Radius: 18, AimPriority: 15}
	m.resolve(unit.Context{ID: 1, Kind: KindSlot, Out: out}, unit.Sense{Self: unit.Snapshot{ID: 1, Slot: 0}, Nearby: []unit.Snapshot{target}})
	cmds := drain(out)
	for _, kind := range []string{KindSlotCrownShot, KindSlotCoinShot, KindSlotCherryShot} {
		if !hasSpawn(cmds, kind) {
			t.Fatalf("missing symbol bullet %s", kind)
		}
	}
}

func TestTripleOutcomes(t *testing.T) {
	t.Run("cherry spawns cherry ball", func(t *testing.T) {
		out := make(chan unit.Cmd, 8)
		m := &slotMachine{slot: 0, rng: unit.NewRNG(1), reels: [3]int{symbolCherry, symbolCherry, symbolCherry}, x: 0, y: 0}
		m.resolve(unit.Context{ID: 1, Kind: KindSlot, Out: out}, unit.Sense{Self: unit.Snapshot{ID: 1, Slot: 0}})
		if !hasSpawn(drain(out), KindSlotCherryBall) {
			t.Fatal("triple cherry should spawn a cherry ball")
		}
	})
	t.Run("coin queues five draws", func(t *testing.T) {
		out := make(chan unit.Cmd, 8)
		m := &slotMachine{slot: 0, rng: unit.NewRNG(1), reels: [3]int{symbolCoin, symbolCoin, symbolCoin}, x: 0, y: 0}
		m.resolve(unit.Context{ID: 1, Kind: KindSlot, Out: out}, unit.Sense{Self: unit.Snapshot{ID: 1, Slot: 0}})
		if m.pending != 5 {
			t.Fatalf("pending draws=%d, want 5", m.pending)
		}
	})
	t.Run("watermelon heals", func(t *testing.T) {
		out := make(chan unit.Cmd, 8)
		m := &slotMachine{slot: 0, rng: unit.NewRNG(1), reels: [3]int{symbolWatermelon, symbolWatermelon, symbolWatermelon}, x: 0, y: 0}
		m.resolve(unit.Context{ID: 1, Kind: KindSlot, Out: out}, unit.Sense{Self: unit.Snapshot{ID: 1, Slot: 0}})
		healed := false
		for _, cmd := range drain(out) {
			if h, ok := cmd.(unit.Heal); ok && h.UnitID == 1 && h.Amount > slotHP {
				healed = true
			}
		}
		if !healed {
			t.Fatal("triple watermelon should fully heal")
		}
	})
	t.Run("bomb is armed", func(t *testing.T) {
		out := make(chan unit.Cmd, 8)
		m := &slotMachine{slot: 0, rng: unit.NewRNG(1), reels: [3]int{symbolBomb, symbolBomb, symbolBomb}, x: 0, y: 0}
		m.resolve(unit.Context{ID: 1, Kind: KindSlot, Out: out}, unit.Sense{Time: 10, Self: unit.Snapshot{ID: 1, Slot: 0}})
		if !m.bombHeld || m.bombAt != 25 {
			t.Fatalf("bomb held=%v at=%v, want transferable 15s bomb", m.bombHeld, m.bombAt)
		}
	})
}

func TestLuckyBulletRollsDamageAndAppliesPermanentSlow(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	b := &slotBullet{owner: 1, slot: 0, kind: KindSlotLuckyShot, rng: unit.NewRNG(2)}
	b.Handle(unit.Context{ID: 9, Kind: KindSlotLuckyShot, Out: out}, unit.Collision{
		Time: 1, Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1},
	})
	cmds := drain(out)
	damage := 0.0
	slowed, stunned := false, false
	for _, cmd := range cmds {
		switch e := cmd.(type) {
		case unit.Damage:
			damage = e.Amount
		case unit.AddFSComponent:
			if e.Zone == unit.FSZoneM && math.Abs(e.Value-luckySlow) < 1e-9 && e.ExpiresAt == 0 {
				slowed = true
			}
		case unit.Stun:
			if e.UnitID == 2 && e.Until == 3 {
				stunned = true
			}
		}
	}
	if damage < luckyMinDamage || damage > luckyMaxDamage {
		t.Fatalf("lucky damage=%v, want 1..20", damage)
	}
	if !slowed || !stunned {
		t.Fatalf("slow=%v stun=%v, want both", slowed, stunned)
	}
}

func TestBombTransfersToEnemyOnCollision(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	m := &slotMachine{slot: 0, now: 5, bombHeld: true, bombAt: 20}
	m.onCollision(unit.Context{ID: 1, Kind: KindSlot, Out: out}, unit.Collision{
		Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1, X: 20, Y: 0},
	})
	cmds := drain(out)
	marked := false
	for _, cmd := range cmds {
		if mark, ok := cmd.(unit.StackMark); ok && mark.UnitID == 2 && mark.Kind == bombMark {
			marked = true
		}
	}
	if !marked || m.bombHeld || m.bombTarget != 2 || m.bombTargetAt != 20 {
		t.Fatalf("bomb transfer failed: marked=%v state=%+v", marked, m)
	}
}

func drain(out chan unit.Cmd) []unit.Cmd {
	var cmds []unit.Cmd
	for {
		select {
		case cmd := <-out:
			cmds = append(cmds, cmd)
		default:
			return cmds
		}
	}
}

func hasConfirm(cmds []unit.Cmd, token uint64) bool {
	for _, cmd := range cmds {
		if c, ok := cmd.(unit.ConfirmDamage); ok && c.Token == token {
			return true
		}
	}
	return false
}

func hasBlock(cmds []unit.Cmd, token uint64) bool {
	for _, cmd := range cmds {
		if c, ok := cmd.(unit.BlockDamage); ok && c.Token == token {
			return true
		}
	}
	return false
}

func hasFX(cmds []unit.Cmd, name string) bool {
	for _, cmd := range cmds {
		if fx, ok := cmd.(unit.FX); ok && fx.Name == name {
			return true
		}
	}
	return false
}

func hasSpawn(cmds []unit.Cmd, kind string) bool {
	for _, cmd := range cmds {
		if s, ok := cmd.(unit.Spawn); ok && s.Kind == kind {
			return true
		}
	}
	return false
}
