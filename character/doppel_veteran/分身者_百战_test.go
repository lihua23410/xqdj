package 分身者_百战

import (
	"math"
	"math/rand/v2"
	"testing"

	"xqdj/internal/unit"

	_ "xqdj/map/hex"
)

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

func TestBootHidesAndSpawnsShadows(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &百战{}
	a.Handle(unit.Context{ID: 1, Kind: KindDoppelVeteran, Out: out}, unit.Sense{
		Time: 0,
		Self: unit.Snapshot{ID: 1, Slot: 0, X: 10, Y: -5, HP: bodyHP, MaxHP: bodyHP},
	})
	cmds := drain(out)
	if !hasNoHealthNumbers(cmds, 1) {
		t.Fatalf("missing NoHealthNumbers: %v", cmds)
	}
	if !hasAim(cmds, 1, 0) {
		t.Fatalf("missing SetAim 0: %v", cmds)
	}
	n := 0
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok && s.Kind == KindShadow {
			n++
		}
	}
	if n != shadowCount {
		t.Fatalf("want %d shadows, got %d: %v", shadowCount, n, cmds)
	}
}

func TestShadowDeathLeviesBody(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &百战{
		knownShadow: map[uint64]struct{}{10: {}, 11: {}},
	}
	alive := []unit.Snapshot{
		{ID: 11, Kind: KindShadow, OwnerID: 1},
	}
	a.syncShadowDeaths(unit.Context{ID: 1, Kind: KindDoppelVeteran, Out: out}, alive)
	cmds := drain(out)
	if !hasDamageNoFreeze(cmds, 1, 1, levyAmount) {
		t.Fatalf("want body levy on shadow death: %v", cmds)
	}
	if !hasHeal(cmds, 11, deathHeal) {
		t.Fatalf("want survivor heal: %v", cmds)
	}
}

func TestReformSnapsOrbitSpeedToCruise(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &百战{}
	// 环转切向约 ωr ≈ 251，重整不得原速带进自由飞
	omega := 2 * math.Pi / zhouPeriod
	orbitSp := omega * zhouRadius
	shadows := []unit.Snapshot{
		{ID: 10, Kind: KindShadow, OwnerID: 1, VX: orbitSp, VY: 0},
		{ID: 11, Kind: KindShadow, OwnerID: 1, VX: 0, VY: 0},
	}
	a.beginReform(unit.Context{ID: 1, Kind: KindDoppelVeteran, Out: out}, 3, shadows)
	cmds := drain(out)
	if !hasCruiseVel(cmds, 10, shadowCruise) {
		t.Fatalf("orbit leftover must snap to cruise: %v", cmds)
	}
	if !hasCruiseVel(cmds, 11, shadowCruise) {
		t.Fatalf("stopped shadow must kick cruise: %v", cmds)
	}
}

func hasCruiseVel(cmds []unit.Cmd, id uint64, cruise float64) bool {
	for _, c := range cmds {
		v, ok := c.(unit.SetVelocity)
		if !ok || v.UnitID != id {
			continue
		}
		sp := math.Hypot(v.VX, v.VY)
		if math.Abs(sp-cruise) < 1e-6 {
			return true
		}
	}
	return false
}

func TestShadowBlocksDuringWeiDash(t *testing.T) {
	startDash(42, 1, 0, weiDashSp, 100, weiDamage, true, true, 0)
	defer clearDash(42)
	out := make(chan unit.Cmd, 8)
	sh := &战影{owner: 1, slot: 0}
	sh.Handle(unit.Context{ID: 42, Kind: KindShadow, Out: out}, unit.IncomingDamage{
		Token: 9, From: 99, Amount: 20, Time: 1,
	})
	cmds := drain(out)
	if !hasBlock(cmds, 42, 9) {
		t.Fatalf("want BlockDamage: %v", cmds)
	}
}

func TestDieVolleyLaunchesDash(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	shadows := []unit.Snapshot{
		{ID: 10, Kind: KindShadow, OwnerID: 1, Slot: 0, X: 0, Y: 0, Radius: shadowRadius, HP: shadowHP, Mortal: true},
		{ID: 11, Kind: KindShadow, OwnerID: 1, Slot: 0, X: 0, Y: 0, Radius: shadowRadius, HP: shadowHP, Mortal: true},
	}
	a := &百战{
		booted:     true,
		phase:      phaseDieWait,
		phaseUntil: 0,
		stackID:    10,
		order:      []uint64{11, 10},
		lastT:      0,
		hasAim:     true,
		aimX:       200,
		aimY:       0,
		liveX:      200,
		liveY:      0,
		passing:    map[uint64]bool{},
		standing:   map[uint64]bool{},
	}
	ctx := unit.Context{ID: 1, Kind: KindDoppelVeteran, Out: out}
	a.Handle(ctx, unit.Sense{
		Time:   dieWait,
		Self:   unit.Snapshot{ID: 1, Slot: 0, HP: bodyHP, MaxHP: bodyHP, Vision: bodyVision},
		Nearby: append(shadows, unit.Snapshot{ID: 99, Kind: "敌", Role: unit.RoleFighter, Slot: 1, X: 200, Y: 0, Radius: 18, AimPriority: 15}),
	})
	cmds := drain(out)
	if a.phase != phaseDieVolley {
		t.Fatalf("want die volley, got phase %d", a.phase)
	}
	if getDash(11) == nil {
		t.Fatalf("first shadow should start dash: cmds=%v", cmds)
	}
	if !hasSetVel(cmds, 11) {
		t.Fatalf("body should drive dash velocity for id 11: %v", cmds)
	}
	clearDash(11)
	clearDash(10)
}

func hasSetVel(cmds []unit.Cmd, id uint64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.SetVelocity); ok && v.UnitID == id && (v.VX*v.VX+v.VY*v.VY) > 1 {
			return true
		}
	}
	return false
}

func TestDrawMoveBagCoversAllThree(t *testing.T) {
	a := &百战{rng: rand.New(rand.NewPCG(7, 11))}
	seen := map[int]bool{}
	for i := 0; i < 3; i++ {
		seen[a.drawMove()] = true
	}
	if len(seen) != 3 || !seen[0] || !seen[1] || !seen[2] {
		t.Fatalf("first bag must contain all three moves, got %v", seen)
	}
	// 第二袋再洗，仍应能抽到围刺
	seen2 := map[int]bool{}
	for i := 0; i < 3; i++ {
		seen2[a.drawMove()] = true
	}
	if !seen2[0] {
		t.Fatalf("second bag missing wei: %v", seen2)
	}
}

func TestZhouGatherTimesOutNearWall(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	a := &百战{
		booted:     true,
		phase:      phaseZhouGather,
		axisID:     10,
		orbitAng:   map[uint64]float64{11: 0},
		gatherFrom: 0,
		lastT:      0,
		passing:    map[uint64]bool{},
		standing:   map[uint64]bool{},
		hasAim:     true,
		aimX:       100, aimY: 0, liveX: 100, liveY: 0,
	}
	// 目标环位点在场外，Clamp 后仍到不了 —— 超时应进准备
	shadows := []unit.Snapshot{
		{ID: 10, Kind: KindShadow, OwnerID: 1, Slot: 0, X: 250, Y: 0, Radius: shadowRadius, HP: shadowHP, Mortal: true},
		{ID: 11, Kind: KindShadow, OwnerID: 1, Slot: 0, X: 0, Y: 0, Radius: shadowRadius, HP: shadowHP, Mortal: true},
	}
	ctx := unit.Context{ID: 1, Kind: KindDoppelVeteran, Out: out}
	a.Handle(ctx, unit.Sense{
		Time:   zhouGatherMax,
		Self:   unit.Snapshot{ID: 1, Slot: 0, HP: bodyHP, MaxHP: bodyHP, Vision: bodyVision},
		Nearby: shadows,
		Field:  unit.HexField(),
	})
	_ = drain(out)
	if a.phase != phaseZhouAim {
		t.Fatalf("want force aim after gather timeout, got phase %d", a.phase)
	}
}

func TestDieWaitPinsFrozenStack(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &百战{
		booted:     true,
		phase:      phaseDieWait,
		phaseUntil: 99,
		stackID:    10,
		stackX:     30,
		stackY:     40,
		stackFixed: true,
		lastT:      0,
		passing:    map[uint64]bool{},
		standing:   map[uint64]bool{},
	}
	// b 已漂到别处，停住阶段仍应钉在冻结点
	shadows := []unit.Snapshot{
		{ID: 10, Kind: KindShadow, OwnerID: 1, Slot: 0, X: 90, Y: 90, Radius: shadowRadius, HP: shadowHP, Mortal: true},
		{ID: 11, Kind: KindShadow, OwnerID: 1, Slot: 0, X: 91, Y: 91, Radius: shadowRadius, HP: shadowHP, Mortal: true},
	}
	ctx := unit.Context{ID: 1, Kind: KindDoppelVeteran, Out: out}
	a.Handle(ctx, unit.Sense{
		Time:   1,
		Self:   unit.Snapshot{ID: 1, Slot: 0, HP: bodyHP, MaxHP: bodyHP},
		Nearby: shadows,
	})
	cmds := drain(out)
	if !hasTeleportTo(cmds, 11, 30, 40) {
		t.Fatalf("wait must pin to frozen stack, not live b: %v", cmds)
	}
}

func hasTeleportTo(cmds []unit.Cmd, id uint64, x, y float64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.Teleport); ok && v.UnitID == id && v.X == x && v.Y == y {
			return true
		}
	}
	return false
}

func TestZhouGatherWalksNotFlashOrbit(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	a := &百战{
		booted:   true,
		phase:    phaseZhouGather,
		axisID:   10,
		orbitAng: map[uint64]float64{11: 0, 12: math.Pi},
		lastT:    0,
		passing:  map[uint64]bool{},
		standing: map[uint64]bool{},
		hasAim:   true,
		aimX:     100, aimY: 0, liveX: 100, liveY: 0,
	}
	shadows := []unit.Snapshot{
		{ID: 10, Kind: KindShadow, OwnerID: 1, Slot: 0, X: 0, Y: 0, Radius: shadowRadius, HP: shadowHP, Mortal: true},
		{ID: 11, Kind: KindShadow, OwnerID: 1, Slot: 0, X: 120, Y: 0, Radius: shadowRadius, HP: shadowHP, Mortal: true},
		{ID: 12, Kind: KindShadow, OwnerID: 1, Slot: 0, X: -120, Y: 0, Radius: shadowRadius, HP: shadowHP, Mortal: true},
	}
	ctx := unit.Context{ID: 1, Kind: KindDoppelVeteran, Out: out}
	a.Handle(ctx, unit.Sense{
		Time:   0.1,
		Self:   unit.Snapshot{ID: 1, Slot: 0, HP: bodyHP, MaxHP: bodyHP, Vision: bodyVision},
		Nearby: shadows,
	})
	if a.phase != phaseZhouGather {
		t.Fatalf("still gathering, got phase %d", a.phase)
	}
	cmds := drain(out)
	// 未到齐时不应已经进入环上闪现式 orbit（准备段）
	if a.phase == phaseZhouAim {
		t.Fatalf("must not enter aim before walk-in finishes: %v", cmds)
	}
}

func TestReformPicksAMove(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	a := &百战{
		booted:     true,
		phase:      phaseReform,
		phaseUntil: 0,
		lastT:      0,
		rng:        rand.New(rand.NewPCG(1, 2)),
		moveBag:    []int{0}, // 强制下一抽为围刺
	}
	ctx := unit.Context{ID: 1, Kind: KindDoppelVeteran, Out: out}
	shadows := []unit.Snapshot{
		{ID: 10, Kind: KindShadow, OwnerID: 1, Slot: 0, X: 48, Y: 0, Radius: shadowRadius, HP: shadowHP, Mortal: true},
		{ID: 11, Kind: KindShadow, OwnerID: 1, Slot: 0, X: 0, Y: 48, Radius: shadowRadius, HP: shadowHP, Mortal: true},
		{ID: 12, Kind: KindShadow, OwnerID: 1, Slot: 0, X: -48, Y: 0, Radius: shadowRadius, HP: shadowHP, Mortal: true},
		{ID: 13, Kind: KindShadow, OwnerID: 1, Slot: 0, X: 0, Y: -48, Radius: shadowRadius, HP: shadowHP, Mortal: true},
	}
	enemy := unit.Snapshot{ID: 99, Kind: "敌", Role: unit.RoleFighter, Slot: 1, X: 100, Y: 0, Radius: 18, AimPriority: 15}
	nearby := append(shadows, enemy)
	a.Handle(ctx, unit.Sense{
		Time:   reformDur,
		Self:   unit.Snapshot{ID: 1, Slot: 0, HP: bodyHP, MaxHP: bodyHP, Vision: bodyVision},
		Nearby: nearby,
	})
	_ = drain(out)
	if a.phase != phaseWeiGather {
		t.Fatalf("want wei gather, got phase %d", a.phase)
	}
}

func TestDieStackBlocksDamage(t *testing.T) {
	setStackGuard(42, true)
	defer setStackGuard(42, false)
	out := make(chan unit.Cmd, 8)
	sh := &战影{owner: 1, slot: 0}
	sh.Handle(unit.Context{ID: 42, Kind: KindShadow, Out: out}, unit.IncomingDamage{
		Token: 3, From: 99, Amount: 20, Time: 1,
	})
	cmds := drain(out)
	if !hasBlock(cmds, 42, 3) {
		t.Fatalf("stacked shadow should block: %v", cmds)
	}
}

func TestZhouOrbitDealsSeven(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &百战{slot: 0, orbitHitAt: map[uint64]map[uint64]float64{}}
	enemy := unit.Snapshot{ID: 99, Kind: "敌", Role: unit.RoleFighter, Slot: 1, X: 10, Y: 0, Radius: 18}
	a.scrapeOrbitAt(unit.Context{ID: 1, Kind: KindDoppelVeteran, Out: out}, unit.Sense{
		Time:   1,
		Self:   unit.Snapshot{ID: 1, Slot: 0},
		Nearby: []unit.Snapshot{enemy},
	}, map[uint64]ringSpot{10: {x: 0, y: 0, r: shadowRadius}}, zhouOrbitDmg)
	cmds := drain(out)
	if !hasDamage(cmds, 10, 99, zhouOrbitDmg) {
		t.Fatalf("want zhou orbit 7: %v", cmds)
	}
}

func TestWeiPauseFreezesCenter(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	a := &百战{
		booted:      true,
		phase:       phaseWeiPause,
		phaseUntil:  99,
		centerFixed: true,
		centerX:     50,
		centerY:     0,
		orbitAng:    map[uint64]float64{10: 0},
		lastT:       0,
		hasAim:      true,
		liveX:       200, // 敌人已跑远，停转不得跟
		liveY:       0,
		passing:     map[uint64]bool{},
		standing:    map[uint64]bool{},
	}
	shadows := []unit.Snapshot{
		{ID: 10, Kind: KindShadow, OwnerID: 1, Slot: 0, X: 120, Y: 0, Radius: shadowRadius, HP: shadowHP, Mortal: true},
	}
	ctx := unit.Context{ID: 1, Kind: KindDoppelVeteran, Out: out}
	a.Handle(ctx, unit.Sense{
		Time: 1,
		Self: unit.Snapshot{ID: 1, Slot: 0, HP: bodyHP, MaxHP: bodyHP, Vision: bodyVision},
		Nearby: append(shadows, unit.Snapshot{
			ID: 99, Kind: "敌", Role: unit.RoleFighter, Slot: 1, X: 200, Y: 0, Radius: 18, AimPriority: 15,
		}),
	})
	cmds := drain(out)
	wantX := 50 + weiRadius // ang=0
	if !hasTeleportTo(cmds, 10, wantX, 0) {
		t.Fatalf("pause must pin to frozen center x=%v: %v", wantX, cmds)
	}
}

func hasNoHealthNumbers(cmds []unit.Cmd, id uint64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.NoHealthNumbers); ok && v.UnitID == id && v.Hold {
			return true
		}
	}
	return false
}

func hasAim(cmds []unit.Cmd, id uint64, pri uint8) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.SetAimPriority); ok && v.UnitID == id && v.Value == pri {
			return true
		}
	}
	return false
}

func hasHeal(cmds []unit.Cmd, id uint64, amount float64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.Heal); ok && v.UnitID == id && v.Amount == amount {
			return true
		}
	}
	return false
}

func hasBlock(cmds []unit.Cmd, id uint64, token uint64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.BlockDamage); ok && v.UnitID == id && v.Token == token {
			return true
		}
	}
	return false
}

func hasDamage(cmds []unit.Cmd, from, to uint64, amount float64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.Damage); ok && v.From == from && v.To == to && v.Amount == amount {
			return true
		}
	}
	return false
}

func hasDamageNoFreeze(cmds []unit.Cmd, from, to uint64, amount float64) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.Damage); ok && v.From == from && v.To == to && v.Amount == amount && v.NoFreeze {
			return true
		}
	}
	return false
}
