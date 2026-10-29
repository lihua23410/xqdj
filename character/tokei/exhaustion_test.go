package 时计

import (
	"encoding/json"
	"math"
	"testing"
	"time"
	"xqdj/internal/sim"
	"xqdj/internal/unit"
)

func exhaustedClock() *时计 {
	return &时计{booted: true, hp: 100, nextAt: 100, carved: map[int]bool{2: true, 4: true, 6: true, 8: true, 10: true, 12: true}, arcUntil: 100}
}

func clockCommands(a *时计, ev unit.Event) []unit.Cmd {
	out := make(chan unit.Cmd, 128)
	a.Handle(unit.Context{ID: 1, Kind: KindTokei, Out: out}, ev)
	var cmds []unit.Cmd
	for len(out) > 0 {
		cmds = append(cmds, <-out)
	}
	return cmds
}

func despawnCount(cmds []unit.Cmd) int {
	n := 0
	for _, cmd := range cmds {
		if c, ok := cmd.(unit.Despawn); ok && c.UnitID == 1 {
			n++
		}
	}
	return n
}

func TestPermanentExhaustionStartsThreeSecondDeathOnce(t *testing.T) {
	a := exhaustedClock()
	s := unit.Sense{Time: 50, Self: unit.Snapshot{ID: 1, HP: 100}}
	cmds := clockCommands(a, s)
	if despawnCount(cmds) != 0 || !a.retiring || a.arcUntil != 0 || a.retireAt != 53 {
		t.Fatal("exhausted clock did not retire and cancel attack")
	}
	if despawnCount(clockCommands(a, unit.Sense{Time: 52.99})) != 0 || a.retireAt != 53 {
		t.Fatal("repeated Sense restarted the death timer")
	}
	cmds = clockCommands(a, unit.IncomingDamage{Token: 9, Amount: 999})
	if len(cmds) != 1 {
		t.Fatalf("dying clock emitted gameplay commands: %+v", cmds)
	}
	if _, ok := cmds[0].(unit.BlockDamage); !ok {
		t.Fatal("damage interrupted the death presentation")
	}
	cmds = clockCommands(a, unit.Stunned{Time: 51, Until: 100})
	cleared := false
	for _, cmd := range cmds {
		if stun, ok := cmd.(unit.Stun); ok && !stun.Hold && stun.UnitID == 1 {
			cleared = true
		}
		if fx, ok := cmd.(unit.FX); ok && (fx.Name != "overclock" || fx.X != 50 || fx.Y != 53) {
			t.Fatalf("death window changed: %+v", fx)
		}
	}
	if !cleared || a.retireAt != 53 || despawnCount(cmds) != 0 {
		t.Fatal("stun interrupted or restarted death")
	}
	if despawnCount(clockCommands(a, unit.Sense{Time: 53})) != 1 {
		t.Fatal("clock did not despawn at the three-second deadline")
	}
}

func TestIntermediateEmptyClockAndRemainingMomentDoNotRetire(t *testing.T) {
	a := exhaustedClock()
	delete(a.carved, 6)
	clockCommands(a, unit.Sense{Time: 50, Self: unit.Snapshot{ID: 1, HP: 100}})
	if a.retiring {
		t.Fatal("empty intermediate clock retired")
	}
	a = exhaustedClock()
	a.moments = []moment{{pin: 9}}
	clockCommands(a, unit.Sense{Time: 50, Self: unit.Snapshot{ID: 1, HP: 100}, Nearby: []unit.Snapshot{{ID: 9, Kind: KindTokeiMoment, OwnerID: 1}}})
	if a.retiring {
		t.Fatal("remaining moment was ignored")
	}
}

func TestFinalRewindAndStolenMomentBothRetire(t *testing.T) {
	a := exhaustedClock()
	a.moments = []moment{{pin: 9, hour: 12, hp: 100, maxHP: 100}}
	clockCommands(a, unit.IncomingDamage{Time: 50, Amount: 999})
	if !a.retiring || a.retireAt != 53 || len(a.moments) != 0 {
		t.Fatal("final rewind left an idle living clock")
	}
	a = exhaustedClock()
	a.moments = []moment{{pin: 9, hour: 12}}
	clockCommands(a, unit.Sense{Time: 50, Self: unit.Snapshot{ID: 1, HP: 100}})
	if !a.retiring || a.retireAt != 53 || len(a.moments) != 0 {
		t.Fatal("last stolen moment did not trigger death on the next Sense")
	}
}

const deathTestClock = "时计私产退场测试"
const deathTestFoe = "时计私产干扰测试"

func init() {
	unit.Register(unit.Spec{Kind: deathTestClock, Role: unit.RoleFighter, Fighter: true, Radius: 18, MaxHP: 100, Vision: 9999},
		func(unit.SpawnInfo) unit.Actor { return exhaustedClock() })
	unit.Register(unit.Spec{Kind: deathTestFoe, Role: unit.RoleFighter, Fighter: true, Radius: 18, MaxHP: 100, Vision: 9999},
		func(unit.SpawnInfo) unit.Actor { return &deathInterference{} })
}

type deathInterference struct{ sent bool }

func (a *deathInterference) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok || a.sent {
		return
	}
	for _, o := range s.Nearby {
		if o.Kind == deathTestClock && o.HP == 0 {
			a.sent = true
			ctx.Out <- unit.Stun{UnitID: o.ID, Hold: true, Until: s.Time + 30}
			ctx.Out <- unit.Heal{UnitID: o.ID, Amount: 100}
			ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: 999}
		}
	}
}

// 只使用引擎公开入口验证私产逻辑，不为测试增加引擎指令或测试钩子。
func TestPrivateDeathRunsInMatchThroughStunAndPause(t *testing.T) {
	m := sim.NewMatchSeeded(53)
	m.SetSlot(0, deathTestClock)
	m.SetSlot(1, deathTestFoe)
	m.Start()
	defer m.End()
	type snapshot struct {
		Time    float64
		Phase   string
		Winner  string
		Units   []unit.Snapshot
		Effects []unit.FX
	}
	read := func() snapshot {
		var s snapshot
		if err := json.Unmarshal(m.SnapshotJSON(), &s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	deadline := 0.0
	paused := false
	for i := 0; i < 400; i++ {
		m.Tick()
		time.Sleep(time.Millisecond)
		s := read()
		for _, fx := range s.Effects {
			if fx.Name != "overclock" {
				continue
			}
			if math.Abs(fx.Y-fx.X-3) > 1e-9 || (deadline != 0 && fx.Y != deadline) {
				t.Fatalf("death deadline changed: %+v", fx)
			}
			deadline = fx.Y
		}
		if deadline == 0 {
			continue
		}
		if !paused && s.Time > deadline-2 {
			m.Pause()
			for j := 0; j < 5; j++ {
				m.Tick()
			}
			if read().Time != s.Time {
				t.Fatal("paused match advanced the death animation")
			}
			m.Start()
			paused = true
		}
		present := false
		for _, u := range s.Units {
			if u.Kind == deathTestClock {
				present = true
			}
		}
		if !present {
			if !paused || s.Time < deadline-1e-9 || s.Time > deadline+4*sim.DT || s.Phase != "ended" || s.Winner != deathTestFoe {
				t.Fatalf("unexpected retirement: time=%v deadline=%v phase=%s winner=%s", s.Time, deadline, s.Phase, s.Winner)
			}
			return
		}
	}
	t.Fatal("clock failed to retire after three seconds")
}
