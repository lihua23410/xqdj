package 回声

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"xqdj/internal/sim"
	"xqdj/internal/unit"
)

func TestFrozenScanDoesNotRefreshAndResetsBeforeNextCycle(t *testing.T) {
	out := make(chan unit.Cmd, 128)
	ctx := unit.Context{ID: 1, Kind: KindEcho, Out: out}
	e := &回声{booted: true}
	scanKind, freezeKind := countKind(1), lockedKind(countKind(1))
	send := func(now float64, foe unit.Snapshot) []unit.Cmd {
		e.Handle(ctx, unit.Sense{Time: now, Self: echoSelf(120, 0, 0), Nearby: []unit.Snapshot{foe}})
		return drain(out)
	}
	cmds := send(1, echoFoe(9, 14, scanKind)) // 同拍多枚碎片也只开启一个十格窗口。
	mark, ok := stackOf(cmds, 9, freezeKind)
	if !ok || mark.Delta != 10 || mark.Lifetime != 6 || !mark.Hidden {
		t.Fatalf("invalid frozen window: %+v", mark)
	}
	foe := echoFoe(9, 1, scanKind) // 模拟已在指令队列里的迟到扫描层。
	foe.Marks = append(foe.Marks, unit.Mark{Kind: freezeKind, Stacks: 10, Lifetime: 6, NextDecayAt: 7.02, Hidden: true})
	for _, now := range []float64{1.2, 2, 6.99} {
		for _, cmd := range send(now, foe) {
			switch cmd.(type) {
			case unit.StackMark, unit.ClearMarks:
				t.Fatalf("frozen phase changed marks at %v: %+v", now, cmd)
			}
		}
		if e.locks[9].until != 7.02 {
			t.Fatal("later hits refreshed the fixed window")
		}
	}
	cmds = send(7.02, foe)
	cleared := map[string]bool{}
	for _, cmd := range cmds {
		if c, ok := cmd.(unit.ClearMarks); ok && c.UnitID == 9 {
			cleared[c.Kind] = true
		}
	}
	if !cleared[scanKind] || !cleared[freezeKind] || e.mode != modeScan || e.charge != 0 {
		t.Fatal("window did not clear both its frozen state and stale scans")
	}
	if _, ok := stackOf(send(7.03, foe), 9, freezeKind); ok {
		t.Fatal("stale snapshot reopened the previous cycle")
	}
	fresh := echoFoe(9, 1, scanKind)
	fresh.Marks[0].Lifetime, fresh.Marks[0].NextDecayAt = 10, 17.1
	send(7.1, fresh) // 清零与下一次命中之间未必能观察到空快照。
	if e.locks[9] != nil {
		t.Fatal("fresh hit stayed trapped in the previous reset state")
	}
	if _, ok := stackOf(send(7.2, echoFoe(9, 10, scanKind)), 9, freezeKind); !ok {
		t.Fatal("fresh scans could not begin a new cycle")
	}
}

func TestFrozenShardNeverAddsOrRefreshesMarks(t *testing.T) {
	for _, otherOwner := range []uint64{1, 2} {
		out := make(chan unit.Cmd, 16)
		kind := countKind(1)
		foe := echoFoe(9, 0, kind)
		foe.Marks = []unit.Mark{{Kind: lockedKind(countKind(otherOwner)), Stacks: 10, Lifetime: 6, NextDecayAt: 7}}
		shard := &碎片{slot: 0, markKind: kind}
		shard.Handle(unit.Context{ID: 3, Out: out}, unit.Collision{Time: 2, Other: foe})
		cmds := drain(out)
		_, added := stackOf(cmds, 9, kind)
		if added != (otherOwner != 1) || !hasDespawn(cmds, 3) {
			t.Fatalf("wrong owner isolation or locked hit behavior: owner=%d commands=%+v", otherOwner, cmds)
		}
	}
}

func TestExpiredLockCannotDealDamageWithoutAnotherSense(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	e := &回声{mode: modeDash, aimX: 1, locks: map[uint64]*scanLock{9: {until: 7}}}
	e.Handle(unit.Context{ID: 1, Out: out}, unit.Collision{Time: 7, Other: echoFoe(9, 10, countKind(1))})
	cmds := drain(out)
	if _, ok := damageOf(cmds); ok || e.mode != modeScan || !hasSpeed(cmds, echoSpeed) {
		t.Fatal("expired dash dealt damage or kept dashing while Sense was suppressed")
	}
}

func TestTargetsKeepIndependentFixedLockWindows(t *testing.T) {
	out := make(chan unit.Cmd, 128)
	ctx := unit.Context{ID: 1, Out: out}
	e := &回声{booted: true, lastTime: 0.9}
	kind := countKind(1)
	freeze := func(id uint64, until float64) unit.Snapshot {
		o := echoFoe(id, 0, kind)
		o.Marks = []unit.Mark{{Kind: lockedKind(kind), Stacks: 10, Lifetime: 6, NextDecayAt: until}}
		return o
	}
	send := func(now float64, foes ...unit.Snapshot) []unit.Cmd {
		e.Handle(ctx, unit.Sense{Time: now, Self: echoSelf(120, 0, 0), Nearby: foes})
		return drain(out)
	}
	send(1, echoFoe(9, 10, kind), echoFoe(10, 9, kind))
	send(2, freeze(9, 7), echoFoe(10, 10, kind))
	if e.locks[9].until != 7 || e.locks[10].until != 8 {
		t.Fatal("targets shared a lock timer")
	}
	for _, cmd := range send(7, echoFoe(9, 0, kind), freeze(10, 8)) {
		if c, ok := cmd.(unit.ClearMarks); ok && (c.UnitID != 9 || (c.Kind != kind && c.Kind != lockedKind(kind))) {
			t.Fatalf("expiry cleared another target or another owner's marks: %+v", c)
		}
	}
	if !e.lockedAt(7) || e.locks[10].reset || e.mode == modeScan {
		t.Fatal("one expired target ended another target's window")
	}
	send(8, echoFoe(9, 0, kind), freeze(10, 8))
	if e.lockedAt(8) || e.mode != modeScan {
		t.Fatal("last target expiry did not return to scanning")
	}
}

const freezeTestOwner = "回声冻结对局测试"
const freezeTestFoe = "回声冻结靶子测试"

func init() {
	unit.Register(unit.Spec{Kind: freezeTestOwner, Role: unit.RoleFighter, Fighter: true, Radius: 18, MaxHP: 90, Vision: 9999},
		func(info unit.SpawnInfo) unit.Actor { return &回声{slot: info.Slot} })
	unit.Register(unit.Spec{Kind: freezeTestFoe, Role: unit.RoleFighter, Fighter: true, Radius: 18, MaxHP: 10000, Vision: 9999},
		func(unit.SpawnInfo) unit.Actor { return &freezeFixture{} })
}

type freezeFixture struct {
	seeded, completed, stale, restarted bool
	seedAt, until, nextShard            float64
}

func (f *freezeFixture) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	var owner unit.Snapshot
	for _, o := range s.Nearby {
		if o.Kind == freezeTestOwner {
			owner = o
		}
	}
	if owner.ID == 0 {
		return
	}
	kind := countKind(owner.ID)
	add := func(n int) {
		ctx.Out <- unit.StackMark{UnitID: ctx.ID, Kind: kind, Delta: n, Lifetime: 10, Hidden: true}
	}
	if !f.seeded {
		f.seeded, f.seedAt = true, s.Time
		add(9)
		ctx.Out <- unit.StackMark{UnitID: ctx.ID, Kind: "其他角色标记", Delta: 4, Hidden: true}
	}
	if !f.completed && s.Time >= f.seedAt+8 {
		f.completed = true
		add(1)
	}
	frozen := false
	for _, mark := range s.Self.Marks {
		if mark.Kind == lockedKind(kind) {
			frozen, f.until = true, mark.NextDecayAt
		}
	}
	if frozen {
		if !f.stale {
			f.stale = true
			add(1) // 已排队的旧命中不得带进下一轮，也不得延长锁定。
		}
		if s.Time >= f.nextShard && s.Time < f.until-0.1 {
			f.nextShard = s.Time + 0.25
			ctx.Out <- ctx.Born(unit.Spawn{Kind: KindShard, OwnerID: owner.ID, Slot: owner.Slot, X: s.Self.X, Y: s.Self.Y})
		}
	} else if f.until > 0 && s.Time >= f.until+0.2 && !f.restarted {
		f.restarted = true
		add(1)
	}
}

// 通过公开对局接口验证真实逐层过期、碎片并发命中和固定窗口，不修改引擎。
func TestFrozenScanInMatchSurvivesOldDecayAndStartsFreshCycle(t *testing.T) {
	m := sim.NewMatchSeeded(42)
	m.SetSlot(0, freezeTestOwner)
	m.SetSlot(1, freezeTestFoe)
	m.Start()
	defer m.End()
	deadline, firstSeen := 0.0, 0.0
	sawOldExpiry, sawCleared := false, false
	for i := 0; i < 1100; i++ {
		m.Tick()
		time.Sleep(time.Millisecond)
		var snapshot struct {
			Time  float64
			Units []unit.Snapshot
		}
		if err := json.Unmarshal(m.SnapshotJSON(), &snapshot); err != nil {
			t.Fatal(err)
		}
		var owner, foe unit.Snapshot
		for _, o := range snapshot.Units {
			if o.Kind == freezeTestOwner {
				owner = o
			} else if o.Kind == freezeTestFoe {
				foe = o
			}
		}
		kind := countKind(owner.ID)
		for _, mark := range foe.Marks {
			if mark.Kind != lockedKind(kind) {
				continue
			}
			if deadline == 0 {
				deadline, firstSeen = mark.NextDecayAt, snapshot.Time
			}
			if mark.Stacks != 10 || mark.Lifetime != 6 || mark.NextDecayAt != deadline {
				t.Fatalf("frozen mark accumulated, decayed or refreshed: %+v", mark)
			}
			if snapshot.Time > 11 {
				sawOldExpiry = true
			}
		}
		if deadline == 0 {
			continue
		}
		if snapshot.Time < deadline && markStacks(foe, kind) > 1 {
			t.Fatal("hits accumulated scan marks during the frozen window")
		}
		if snapshot.Time > deadline+0.08 && snapshot.Time < deadline+0.16 {
			if markStacks(foe, kind) != 0 || markStacks(foe, lockedKind(kind)) != 0 {
				t.Fatal("expired cycle left marks behind")
			}
			sawCleared = true
		}
		if snapshot.Time > deadline+0.4 {
			if !sawOldExpiry || !sawCleared || math.Abs(deadline-firstSeen-6) > 2*sim.DT || markStacks(foe, kind) != 1 || markStacks(foe, "其他角色标记") != 4 {
				t.Fatal("window duration, clean restart, or mark ownership is wrong")
			}
			damage := foe.MaxHP - foe.HP
			if damage <= 0 || math.Mod(damage, 6) != 0 {
				t.Fatalf("dash damage must be six per hit, got total %v", damage)
			}
			return
		}
	}
	t.Fatal("scan cycle did not finish")
}
