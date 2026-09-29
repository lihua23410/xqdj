package 炭翁

import (
	"math"
	"testing"

	"xqdj/internal/unit"
)

func TestOpeningDoesNotTransferOrMint(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	ctx, out := pipe(1)
	a.Handle(ctx, sense(0, selfAt(0, 0), foeAt(10, 0)))
	if len(a.brain.coals) != 0 {
		t.Fatalf("coals=%d", len(a.brain.coals))
	}
	if len(damages(drain(out))) != 0 {
		t.Fatal("opening should not burn")
	}
}

func TestSelfCoalAtTwoAndAHalf(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	ctx, _ := pipe(1)
	a.Handle(ctx, sense(selfEvery, selfAt(0, 0), foeAt(200, 0)))
	if len(a.brain.coals) != 1 || a.brain.coals[0].value != selfValue || a.brain.coals[0].holder != 1 {
		t.Fatalf("coals=%+v", a.brain.coals)
	}
	if math.Abs(a.brain.coals[0].due-(selfEvery+coalFuse)) > 1e-9 {
		t.Fatalf("due=%v", a.brain.coals[0].due)
	}
}

func TestSelfCoalRidesTheTouch(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	ctx, out := pipe(1)
	a.Handle(ctx, sense(0, selfAt(0, 0), foeAt(200, 0)))
	drain(out)
	a.Handle(ctx, sense(selfEvery, selfAt(0, 0), foeAt(20, 0)))
	if len(a.brain.coals) != 1 || a.brain.coals[0].holder != 2 {
		t.Fatalf("coals=%+v", a.brain.coals)
	}
}

func TestBounceHandsOffAfterSeparation(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	ctx, out := pipe(1)
	a.Handle(ctx, sense(0, selfAt(0, 0), foeAt(200, 0)))
	a.Handle(ctx, sense(selfEvery, selfAt(0, 0), foeAt(200, 0)))
	drain(out)
	foe := foeAt(40, 0)
	a.Handle(ctx, unit.Collision{Time: selfEvery + 0.01, Other: foe})
	a.Handle(ctx, sense(selfEvery+0.02, selfAt(0, 0), foeAt(80, 0)))
	if len(a.brain.coals) != 1 || a.brain.coals[0].holder != 2 {
		t.Fatalf("coals=%+v", a.brain.coals)
	}
}

func TestRepeatedBounceWaitsOutTheCooldown(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	ctx, out := pipe(1)
	a.Handle(ctx, sense(0, selfAt(0, 0), foeAt(200, 0)))
	a.Handle(ctx, sense(selfEvery, selfAt(0, 0), foeAt(200, 0)))
	drain(out)
	foe := foeAt(40, 0)
	a.Handle(ctx, unit.Collision{Time: selfEvery + 0.01, Other: foe})
	a.Handle(ctx, sense(selfEvery+0.02, selfAt(0, 0), foeAt(80, 0)))
	a.brain.coals = append(a.brain.coals, placed{holder: 1, value: 4, due: 9})
	a.Handle(ctx, unit.Collision{Time: selfEvery + 0.03, Other: foe})
	a.Handle(ctx, sense(selfEvery+0.04, selfAt(0, 0), foeAt(80, 0)))
	for _, c := range a.brain.coals {
		if c.value == 4 && c.holder != 1 {
			t.Fatal("cooldown should keep the next coal")
		}
	}
	drain(out)
	a.Handle(ctx, unit.Collision{Time: selfEvery + 0.02 + handCD, Other: foe})
	a.Handle(ctx, sense(selfEvery+0.02+handCD, selfAt(0, 0), foeAt(80, 0)))
	for _, c := range a.brain.coals {
		if c.value == 4 && c.holder != 2 {
			t.Fatalf("holder=%d", c.holder)
		}
	}
}

func TestStayOverlappedHandsOnePerCooldown(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	ctx, out := pipe(1)
	a.Handle(ctx, sense(0, selfAt(0, 0), foeAt(20, 0)))
	drain(out)
	a.Handle(ctx, sense(selfEvery, selfAt(0, 0), foeAt(20, 0)))
	if len(a.brain.coals) != 1 || a.brain.coals[0].holder != 2 {
		t.Fatalf("coals=%+v", a.brain.coals)
	}
	drain(out)
	a.brain.coals = append(a.brain.coals, placed{holder: 1, value: 4, due: 9}, placed{holder: 1, value: 8, due: 12})
	a.Handle(ctx, sense(selfEvery+0.1, selfAt(0, 0), foeAt(20, 0)))
	for _, c := range a.brain.coals {
		if c.holder == 1 && c.value != 4 && c.value != 8 {
			t.Fatalf("coals=%+v", a.brain.coals)
		}
		if c.value == 4 && c.holder != 1 {
			t.Fatal("cooldown should keep the next coal")
		}
	}
	a.Handle(ctx, sense(selfEvery+handCD, selfAt(0, 0), foeAt(20, 0)))
	for _, c := range a.brain.coals {
		if c.value == 4 && c.holder != 2 {
			t.Fatalf("soonest coal holder=%d", c.holder)
		}
		if c.value == 8 && c.holder != 1 {
			t.Fatalf("later coal holder=%d", c.holder)
		}
	}
}

func TestEighthCoalDoesNotDetonate(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	a.brain.owner = 1
	a.brain.nextSelf = 100
	for i := 0; i < coalCap-1; i++ {
		a.brain.coals = append(a.brain.coals, placed{holder: 1, value: 1, due: float64(20 + i)})
	}
	a.brain.pending = []booked{{at: 1, value: 3}}
	ctx, out := pipe(1)
	a.Handle(ctx, sense(1, selfAt(0, 0), foeAt(200, 0)))
	if len(damages(drain(out))) != 0 {
		t.Fatal("the eighth coal should not detonate anyone")
	}
	if a.brain.held(1) != coalCap {
		t.Fatalf("held=%d", a.brain.held(1))
	}
}

func TestNinthCoalDetonatesTheOldest(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	a.brain.owner = 1
	a.brain.nextSelf = 100
	for i := 0; i < coalCap; i++ {
		a.brain.coals = append(a.brain.coals, placed{holder: 1, value: float64(i + 1), due: float64(20 + i)})
	}
	a.brain.pending = []booked{{at: 1, value: 20}}
	ctx, out := pipe(1)
	a.Handle(ctx, sense(1, selfAt(0, 0), foeAt(200, 0)))
	ds := damages(drain(out))
	if len(ds) != 1 || ds[0].Amount != 1 || ds[0].To != 1 || !ds[0].NoFreeze {
		t.Fatalf("damages=%+v", ds)
	}
	if a.brain.held(1) != coalCap || len(a.brain.pending) != 0 {
		t.Fatalf("held=%d pending=%v", a.brain.held(1), a.brain.pending)
	}
	var kept bool
	for _, c := range a.brain.coals {
		if c.value == 1 {
			t.Fatal("oldest coal should be gone")
		}
		if c.value == 20 && c.holder == 1 {
			kept = true
		}
	}
	if !kept {
		t.Fatalf("new coal missing: %+v", a.brain.coals)
	}
}

func TestFullEnemyDetonatesOldestOnArrival(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	a.brain.owner = 1
	a.brain.nextSelf = 100
	a.brain.coals = []placed{{holder: 1, value: 6, due: 3}}
	for i := 0; i < coalCap; i++ {
		a.brain.coals = append(a.brain.coals, placed{holder: 2, value: float64(i + 1), due: float64(4 + i)})
	}
	ctx, out := pipe(1)
	a.Handle(ctx, sense(1, selfAt(0, 0), foeAt(20, 0)))
	ds := damages(drain(out))
	if len(ds) != 1 || ds[0].Amount != 1 || ds[0].To != 2 || ds[0].NoFreeze {
		t.Fatalf("damages=%+v", ds)
	}
	if a.brain.held(2) != coalCap || a.brain.held(1) != 0 {
		t.Fatalf("self=%d foe=%d coals=%+v", a.brain.held(1), a.brain.held(2), a.brain.coals)
	}
	var kept bool
	for _, c := range a.brain.coals {
		if c.value == 1 {
			t.Fatal("oldest on the enemy should be gone")
		}
		if c.value == 6 && c.holder == 2 {
			kept = true
		}
	}
	if !kept {
		t.Fatalf("arrived coal missing: %+v", a.brain.coals)
	}
}

func TestLeaveAndTouchAgain(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	ctx, out := pipe(1)
	a.Handle(ctx, sense(0, selfAt(0, 0), foeAt(200, 0)))
	a.Handle(ctx, sense(selfEvery, selfAt(0, 0), foeAt(200, 0)))
	drain(out)
	a.Handle(ctx, sense(selfEvery+0.1, selfAt(0, 0), foeAt(20, 0)))
	if a.brain.coals[0].holder != 2 {
		t.Fatalf("holder=%d", a.brain.coals[0].holder)
	}
}

func TestBeyondVisionDoesNotHandOff(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	ctx, _ := pipe(1)
	a.Handle(ctx, sense(selfEvery, selfAt(0, 0), foeAt(bodyVision+1, 0)))
	if len(a.brain.coals) != 1 || a.brain.coals[0].holder != 1 {
		t.Fatalf("coals=%+v", a.brain.coals)
	}
}

func TestVisionEdgeHandsOff(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	ctx, _ := pipe(1)
	a.Handle(ctx, sense(selfEvery, selfAt(0, 0), foeAt(bodyVision, 0)))
	if len(a.brain.coals) != 1 || a.brain.coals[0].holder != 2 {
		t.Fatalf("coals=%+v", a.brain.coals)
	}
}

func TestOpeningFrameHandsOff(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	ctx, _ := pipe(1)
	a.Handle(ctx, unit.IncomingDamage{Token: 1, From: 9, Amount: 4, Time: 0})
	a.Handle(ctx, sense(0, selfAt(0, 0), foeAt(100, 0)))
	if len(a.brain.coals) != 1 || a.brain.coals[0].holder != 2 || a.brain.coals[0].value != 4 {
		t.Fatalf("coals=%+v", a.brain.coals)
	}
}

func TestVisionRingOnBodyOnly(t *testing.T) {
	body, ok := unit.Lookup(KindCharcoal)
	if !ok || !body.Look.VisionRing {
		t.Fatal("charcoal should draw a vision ring")
	}
	ember, ok := unit.Lookup(KindCharcoalEmber)
	if !ok || ember.Look.VisionRing {
		t.Fatal("ember should not draw a vision ring")
	}
}

func TestHandFXOnTransfer(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	ctx, out := pipe(1)
	a.Handle(ctx, sense(selfEvery, selfAt(0, 0), foeAt(40, 0)))
	var hand *unit.FX
	for _, c := range drain(out) {
		fx, ok := c.(unit.FX)
		if ok && fx.Name == "hand" {
			cp := fx
			hand = &cp
		}
	}
	if hand == nil || hand.Slot != 2 || hand.X != 0 || hand.Y != 0 || hand.VX != 40 || hand.VY != 0 {
		t.Fatalf("hand=%+v", hand)
	}
}

func TestCoalOutsideVisionStillEmitsFX(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	a.brain.owner = 1
	a.brain.nextSelf = 100
	a.brain.coals = []placed{{holder: 2, value: 6, due: 20}}
	ctx, out := pipe(1)
	a.Handle(ctx, unit.Sense{Time: 3, Self: selfAt(0, 0)})
	var got bool
	for _, c := range drain(out) {
		fx, ok := c.(unit.FX)
		if ok && fx.Name == "coal" && fx.Slot == 2 {
			got = true
		}
	}
	if !got {
		t.Fatal("a coal on someone outside vision should still be drawn")
	}
}

func TestIncomingBecomesCoalNotHP(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	ctx, out := pipe(1)
	a.Handle(ctx, unit.IncomingDamage{Token: 3, From: 9, Amount: 10, Time: 0.5})
	cmds := drain(out)
	if _, ok := cmds[0].(unit.BlockDamage); !ok {
		t.Fatalf("cmds=%v", cmds)
	}
	a.Handle(ctx, sense(0.5, selfAt(0, 0), foeAt(200, 0)))
	if len(a.brain.coals) != 1 || a.brain.coals[0].value != 10 || a.brain.coals[0].holder != 1 {
		t.Fatalf("coals=%+v", a.brain.coals)
	}
}

func TestExpireBeforeBook(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	a.brain.owner = 1
	a.brain.coals = []placed{{holder: 1, value: 8, due: 2}}
	a.brain.pending = []booked{{at: 2, value: 5}}
	ctx, out := pipe(1)
	a.Handle(ctx, sense(2, selfAt(0, 0), foeAt(20, 0)))
	ds := damages(drain(out))
	if len(ds) != 1 || ds[0].Amount != 8 || !ds[0].NoFreeze || ds[0].To != 1 {
		t.Fatalf("damages=%+v", ds)
	}
	if len(a.brain.coals) != 1 || a.brain.coals[0].value != 5 || a.brain.coals[0].holder != 2 {
		t.Fatalf("booked coal should transfer, coals=%+v", a.brain.coals)
	}
	if math.Abs(a.brain.coals[0].due-(2+coalFuse)) > 1e-9 {
		t.Fatalf("due=%v", a.brain.coals[0].due)
	}
}

func TestDueCoalDoesNotSlipOut(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	a.brain.owner = 1
	a.brain.coals = []placed{{holder: 1, value: 8, due: 2}}
	ctx, _ := pipe(1)
	a.Handle(ctx, sense(2, selfAt(0, 0), foeAt(20, 0)))
	if len(a.brain.coals) != 0 {
		t.Fatalf("due coal left on someone: %+v", a.brain.coals)
	}
}

func TestSameInstantCoalsSum(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	a.brain.owner = 1
	a.brain.nextSelf = 100
	a.brain.coals = []placed{
		{holder: 2, value: 4, due: 3},
		{holder: 2, value: 6, due: 3},
	}
	ctx, out := pipe(1)
	a.Handle(ctx, sense(3, selfAt(0, 0), foeAt(200, 0)))
	ds := damages(drain(out))
	if len(ds) != 1 || ds[0].Amount != 10 || ds[0].To != 2 || ds[0].NoFreeze || ds[0].From != 1 {
		t.Fatalf("damages=%+v", ds)
	}
}

func TestDeathSkipsBookAndTransfer(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	a.brain.owner = 1
	a.brain.nextSelf = 2
	a.brain.coals = []placed{
		{holder: 1, value: 6, due: 2},
		{holder: 1, value: 4, due: 9},
		{holder: 2, value: 3, due: 9},
	}
	ctx, out := pipe(1)
	me := selfAt(0, 0)
	me.HP = 6
	a.Handle(ctx, senseSnap(2, me, foeAt(20, 0)))
	ds := damages(drain(out))
	if len(ds) != 1 || ds[0].Amount != 6 || !ds[0].NoFreeze {
		t.Fatalf("damages=%+v", ds)
	}
	if len(a.brain.coals) != 1 || a.brain.coals[0].holder != 2 || a.brain.coals[0].value != 3 {
		t.Fatalf("coals=%+v", a.brain.coals)
	}
	t.Cleanup(func() { orphans.Delete(uint64(1)) })
}

func TestDeathSpawnsEmberForForeignCoal(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	a.brain.owner = 7
	a.brain.nextSelf = 100
	a.brain.coals = []placed{
		{holder: 7, value: 6, due: 2},
		{holder: 2, value: 3, due: 9},
	}
	ctx, out := pipe(7)
	me := selfAt(0, 0)
	me.ID = 7
	me.HP = 6
	a.Handle(ctx, senseSnap(2, me, foeAt(20, 0)))
	cmds := drain(out)
	var spawned *unit.Spawn
	for i := range cmds {
		if s, ok := cmds[i].(unit.Spawn); ok && s.Kind == KindCharcoalEmber {
			cp := s
			spawned = &cp
		}
	}
	if spawned == nil || spawned.Slot != 7 {
		t.Fatalf("spawn=%v cmds=%v", spawned, cmds)
	}
	if _, ok := orphans.Load(uint64(7)); !ok {
		t.Fatal("missing orphan ledger")
	}
	t.Cleanup(func() { orphans.Delete(uint64(7)) })

	ember := &烬{token: 7}
	ectx, eout := pipe(99)
	ectx.Kind = KindCharcoalEmber
	ember.Handle(ectx, sense(9, unit.Snapshot{ID: 99, Role: unit.RoleHelper}, foeAt(0, 0)))
	ds := damages(drain(eout))
	if len(ds) != 1 || ds[0].Amount != 3 || ds[0].To != 2 || ds[0].From != 7 || ds[0].NoFreeze {
		t.Fatalf("ember damages=%+v", ds)
	}
}

func TestExpiryFromCharcoalConfirms(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	a.brain.kinds[4] = KindCharcoal
	ctx, out := pipe(1)
	a.Handle(ctx, unit.IncomingDamage{Token: 8, From: 4, Amount: 12, Time: 1})
	if _, ok := drain(out)[0].(unit.ConfirmDamage); !ok {
		t.Fatal("expiry should confirm")
	}
	if len(a.brain.pending) != 0 {
		t.Fatal("expiry must not become a new coal")
	}
}

func TestMortalMinionTakesCoal(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	ctx, out := pipe(1)
	a.Handle(ctx, sense(0, selfAt(0, 0), foeAt(200, 0)))
	a.Handle(ctx, sense(selfEvery, selfAt(0, 0), foeAt(200, 0)))
	drain(out)
	minion := foeAt(40, 0)
	minion.Role = unit.RoleMinion
	minion.Mortal = true
	minion.ID = 3
	a.Handle(ctx, unit.Collision{Time: selfEvery + 0.01, Other: minion})
	far := minion
	far.X = 80
	hollow := foeAt(12, 0)
	hollow.Nonsolid = true
	hollow.Radius = 0
	hollow.AimPriority = 0
	hollow.ID = 4
	friendly := foeAt(8, 0)
	friendly.Role = unit.RoleMinion
	friendly.Mortal = true
	friendly.Slot = 0
	friendly.ID = 5
	a.Handle(ctx, unit.Sense{
		Time: selfEvery + 0.02, Self: selfAt(0, 0),
		Nearby: []unit.Snapshot{far, hollow, friendly},
	})
	if len(a.brain.coals) != 1 || a.brain.coals[0].holder != 3 {
		t.Fatalf("coals=%+v", a.brain.coals)
	}
}

func TestSteersTowardSeek(t *testing.T) {
	a := &炭翁{brain: newBrain()}
	ctx, out := pipe(1)
	foe := foeAt(30, 40)
	foe.AimPriority = 15
	a.Handle(ctx, unit.Sense{Time: 0, Self: selfAt(0, 0), Nearby: []unit.Snapshot{foe}})
	var dir *unit.SetFSDirection
	for _, c := range drain(out) {
		if d, ok := c.(unit.SetFSDirection); ok {
			cp := d
			dir = &cp
		}
	}
	if dir == nil || dir.VX != 30 || dir.VY != 40 {
		t.Fatalf("dir=%v", dir)
	}
}

func TestEmberDespawnsWhenOwnerLives(t *testing.T) {
	b := newBrain()
	b.owner = 7
	b.coals = []placed{{holder: 2, value: 3, due: 9}}
	orphans.Store(uint64(7), b)
	t.Cleanup(func() { orphans.Delete(uint64(7)) })
	ember := &烬{token: 7}
	ctx, out := pipe(99)
	owner := selfAt(0, 0)
	owner.ID = 7
	owner.HP = 40
	ember.Handle(ctx, unit.Sense{
		Time: 9, Nearby: []unit.Snapshot{owner},
	})
	cmds := drain(out)
	var gone bool
	for _, c := range cmds {
		if d, ok := c.(unit.Despawn); ok && d.UnitID == 99 {
			gone = true
		}
	}
	if !gone {
		t.Fatal("ember should leave while the owner still stands")
	}
	if _, ok := orphans.Load(uint64(7)); ok {
		t.Fatal("ledger should return to the owner")
	}
	if len(damages(cmds)) != 0 {
		t.Fatal("living owner keeps the clock")
	}
}

func selfAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindCharcoal, Role: unit.RoleFighter, Slot: 0,
		X: x, Y: y, Radius: bodyRadius, HP: bodyHP, MaxHP: bodyHP,
		Vision: bodyVision, AimPriority: 15,
	}
}

func foeAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 2, Kind: "雷达", Role: unit.RoleFighter, Slot: 1,
		X: x, Y: y, Radius: 18, HP: 100, MaxHP: 100, AimPriority: 15,
	}
}

func sense(t float64, self unit.Snapshot, foe unit.Snapshot) unit.Sense {
	return senseSnap(t, self, foe)
}

func senseSnap(t float64, self unit.Snapshot, foe unit.Snapshot) unit.Sense {
	return unit.Sense{Time: t, Self: self, Nearby: []unit.Snapshot{foe}}
}

func pipe(id uint64) (unit.Context, chan unit.Cmd) {
	out := make(chan unit.Cmd, 16)
	return unit.Context{ID: id, Kind: KindCharcoal, Out: out}, out
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

func damages(cmds []unit.Cmd) []unit.Damage {
	var ds []unit.Damage
	for _, c := range cmds {
		if d, ok := c.(unit.Damage); ok {
			ds = append(ds, d)
		}
	}
	return ds
}
