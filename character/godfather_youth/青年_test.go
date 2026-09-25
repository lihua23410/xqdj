package 教父青年

import (
	"math"
	"testing"

	"xqdj/internal/unit"
)

func ctxOf() (unit.Context, chan unit.Cmd) {
	out := make(chan unit.Cmd, 32)
	return unit.Context{ID: 1, Kind: KindYouth, Out: out}, out
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

func self(hp float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindYouth, Role: unit.RoleFighter, Slot: 0,
		Radius: bodyRadius, HP: hp, MaxHP: bodyHP, AimPriority: 15,
	}
}

func foe(id uint64, x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: id, Kind: "靶子", Role: unit.RoleFighter, Slot: 1,
		X: x, Y: y, Radius: 18, HP: 100, MaxHP: 100, AimPriority: 15,
	}
}

func senseAt(t, hp float64, nearby []unit.Snapshot) unit.Sense {
	return unit.Sense{Time: t, Self: self(hp), Nearby: nearby}
}

func heals(cmds []unit.Cmd) float64 {
	var n float64
	for _, c := range cmds {
		if h, ok := c.(unit.Heal); ok {
			n += h.Amount
		}
	}
	return n
}

func shots(cmds []unit.Cmd) []unit.Spawn {
	var out []unit.Spawn
	for _, c := range cmds {
		if s, ok := c.(unit.Spawn); ok {
			out = append(out, s)
		}
	}
	return out
}

func TestOpeningKeepsThreeDoses(t *testing.T) {
	a := &教父青年{roll: func() float64 { return 0.5 }}
	ctx, out := ctxOf()
	a.Handle(ctx, senseAt(0, 100, []unit.Snapshot{foe(2, 400, 0)}))
	cmds := drain(out)
	if a.doses != 3 || heals(cmds) != 0 {
		t.Fatalf("doses=%d heal=%v", a.doses, heals(cmds))
	}
	if len(shots(cmds)) != 1 || shots(cmds)[0].Kind != KindYouthShot {
		t.Fatalf("shot=%+v", shots(cmds))
	}
}

func TestClockSpendsOneAndBuffsCruise(t *testing.T) {
	a := &教父青年{roll: func() float64 { return 0.5 }}
	ctx, out := ctxOf()
	a.Handle(ctx, senseAt(0, 100, nil))
	_ = drain(out)
	a.Handle(ctx, senseAt(30, 100, nil))
	cmds := drain(out)
	if a.doses != 2 || heals(cmds) != 0 {
		t.Fatalf("doses=%d heal=%v", a.doses, heals(cmds))
	}
	var cruiseCmd *unit.SetCruise
	for _, c := range cmds {
		if v, ok := c.(unit.SetCruise); ok && v.Speed == buffCruise {
			cruiseCmd = &v
		}
	}
	if cruiseCmd == nil || a.buffUntil != 38 {
		t.Fatalf("buff cruise=%v until=%v", cruiseCmd, a.buffUntil)
	}
}

func TestLowHPForcesFifty(t *testing.T) {
	a := &教父青年{roll: func() float64 { return 0.5 }}
	ctx, out := ctxOf()
	a.Handle(ctx, senseAt(0, 40, nil))
	_ = drain(out)
	a.Handle(ctx, senseAt(1, 15, nil))
	if got := heals(drain(out)); got != 35 {
		t.Fatalf("heal=%v want 35", got)
	}
	if a.doses != 2 {
		t.Fatalf("doses=%d", a.doses)
	}
}

func TestSameFrameSpendsOnce(t *testing.T) {
	a := &教父青年{roll: func() float64 { return 0.5 }}
	ctx, out := ctxOf()
	a.Handle(ctx, senseAt(0, 40, nil))
	_ = drain(out)
	a.Handle(ctx, senseAt(30, 10, nil))
	if a.doses != 2 {
		t.Fatalf("doses=%d", a.doses)
	}
	a.prevHP = 50
	a.Handle(ctx, senseAt(30.02, 50, nil))
	if a.doses != 2 {
		t.Fatalf("spent again doses=%d", a.doses)
	}
}

func TestVolleyLocksAtThree(t *testing.T) {
	a := &教父青年{roll: func() float64 { return 0.5 }}
	ctx, out := ctxOf()
	far := []unit.Snapshot{foe(2, 400, 0)}
	a.Handle(ctx, senseAt(0, 100, far))
	_ = drain(out)
	a.buffUntil = 8
	a.Handle(ctx, senseAt(0.12, 100, far))
	a.Handle(ctx, senseAt(0.24, 100, far))
	got := shots(drain(out))
	if len(got) != 2 {
		t.Fatalf("follow-up shots=%d", len(got))
	}
	for _, s := range got {
		if s.Kind != KindYouthShot {
			t.Fatalf("kind=%s", s.Kind)
		}
	}
	if a.phase != phIdle {
		t.Fatalf("phase=%d", a.phase)
	}
}

func TestBuffVolleyIsFiveWithSpread(t *testing.T) {
	a := &教父青年{roll: func() float64 { return 1 }}
	ctx, out := ctxOf()
	a.buffUntil = 8
	far := []unit.Snapshot{foe(2, 400, 0)}
	a.Handle(ctx, senseAt(0, 100, far))
	spawn := shots(drain(out))
	if len(spawn) != 1 || spawn[0].Kind != KindYouthBuffShot {
		t.Fatalf("first=%+v", spawn)
	}
	ang := math.Atan2(spawn[0].VY, spawn[0].VX)
	if math.Abs(ang-spread) > 1e-6 {
		t.Fatalf("ang=%v", ang)
	}
	for _, tm := range []float64{0.12, 0.24, 0.36, 0.48} {
		a.Handle(ctx, senseAt(tm, 100, far))
	}
	if n := len(shots(drain(out))); n != 4 {
		t.Fatalf("rest=%d", n)
	}
}

func TestCloseRamsForTen(t *testing.T) {
	a := &教父青年{roll: func() float64 { return 0.5 }}
	ctx, out := ctxOf()
	a.Handle(ctx, senseAt(0, 100, []unit.Snapshot{foe(2, 80, 0)}))
	cmds := drain(out)
	if len(shots(cmds)) != 0 {
		t.Fatal("shot while close")
	}
	var dmg float64
	for _, c := range cmds {
		if d, ok := c.(unit.Damage); ok {
			dmg += d.Amount
		}
	}
	if dmg != 0 {
		t.Fatalf("opened inside should wait for overlap, dmg=%v", dmg)
	}
	a.Handle(ctx, unit.Collision{Time: 0.05, Other: foe(2, 30, 0)})
	cmds = drain(out)
	var got float64
	var stunned bool
	for _, c := range cmds {
		if d, ok := c.(unit.Damage); ok {
			got += d.Amount
		}
		if _, ok := c.(unit.Stun); ok {
			stunned = true
		}
	}
	var dropped bool
	for _, c := range cmds {
		if p, ok := c.(unit.Pass); ok && !p.Hold {
			dropped = true
		}
	}
	if got != ramDmg || stunned || a.phase != phRam || dropped {
		t.Fatalf("dmg=%v stun=%v phase=%d dropped=%v", got, stunned, a.phase, dropped)
	}
}

func TestBuffRamStunsAndReturns(t *testing.T) {
	a := &教父青年{roll: func() float64 { return 0.5 }}
	ctx, out := ctxOf()
	a.buffUntil = 8
	a.Handle(ctx, senseAt(0, 100, []unit.Snapshot{foe(2, 80, 0)}))
	_ = drain(out)
	target := foe(2, 30, 0)
	a.Handle(ctx, unit.Collision{Time: 0.05, Other: target})
	cmds := drain(out)
	var stun *unit.Stun
	for _, c := range cmds {
		if s, ok := c.(unit.Stun); ok {
			stun = &s
		}
	}
	if stun == nil || math.Abs(stun.Until-(0.05+stunDur)) > 1e-9 || a.phase != phReturn || a.ramX >= 0 {
		t.Fatalf("stun=%v phase=%d ramX=%v", stun, a.phase, a.ramX)
	}
	a.buffUntil = 0.06
	a.Handle(ctx, unit.Collision{Time: 0.1, Other: target})
	var early float64
	for _, c := range drain(out) {
		if d, ok := c.(unit.Damage); ok {
			early += d.Amount
		}
	}
	if early != 0 {
		t.Fatalf("inside buff ram cd dmg=%v", early)
	}
	a.Handle(ctx, unit.Collision{Time: 0.25, Other: target})
	var n float64
	for _, c := range drain(out) {
		if d, ok := c.(unit.Damage); ok {
			n += d.Amount
		}
	}
	if n != ramDmg || a.phase != phReturn {
		t.Fatalf("return dmg=%v phase=%d", n, a.phase)
	}
}

func TestEnterCloseSetsAimForTwoSeconds(t *testing.T) {
	a := &教父青年{roll: func() float64 { return 0.5 }}
	ctx, out := ctxOf()
	far := foe(2, 400, 0)
	a.Handle(ctx, senseAt(0, 100, []unit.Snapshot{far}))
	if aimOf(drain(out), 2) != nil {
		t.Fatal("opening far should not mark")
	}
	near := foe(2, 80, 0)
	a.Handle(ctx, senseAt(0.1, 100, []unit.Snapshot{near}))
	got := aimOf(drain(out), 2)
	if got == nil || got.Value != markAim || got.From != 1 {
		t.Fatalf("enter aim=%+v", got)
	}
	a.Handle(ctx, senseAt(1, 100, []unit.Snapshot{near}))
	if aimOf(drain(out), 2) != nil {
		t.Fatal("still inside should not mark again")
	}
	a.Handle(ctx, senseAt(2.1, 100, []unit.Snapshot{near}))
	got = aimOf(drain(out), 2)
	if got == nil || got.Value != 15 {
		t.Fatalf("restore=%+v", got)
	}
}

func TestReenterKeepsOriginalAim(t *testing.T) {
	a := &教父青年{roll: func() float64 { return 0.5 }}
	ctx, out := ctxOf()
	a.Handle(ctx, senseAt(0, 100, []unit.Snapshot{foe(2, 400, 0)}))
	_ = drain(out)
	a.Handle(ctx, senseAt(0.1, 100, []unit.Snapshot{foe(2, 80, 0)}))
	_ = drain(out)
	a.Handle(ctx, senseAt(0.2, 100, []unit.Snapshot{foe(2, 400, 0)}))
	_ = drain(out)
	marked := foe(2, 80, 0)
	marked.AimPriority = 2
	a.Handle(ctx, senseAt(0.3, 100, []unit.Snapshot{marked}))
	_ = drain(out)
	if a.marks[2].prev != 15 || math.Abs(a.marks[2].until-2.3) > 1e-9 {
		t.Fatalf("mark=%+v", a.marks[2])
	}
}

func TestMinionEnterSetsAim(t *testing.T) {
	a := &教父青年{roll: func() float64 { return 0.5 }}
	ctx, out := ctxOf()
	m := foe(4, 400, 0)
	m.Role = unit.RoleMinion
	m.Mortal = true
	m.AimPriority = 60
	a.Handle(ctx, senseAt(0, 100, []unit.Snapshot{m}))
	_ = drain(out)
	m.X = 50
	a.Handle(ctx, senseAt(0.1, 100, []unit.Snapshot{m}))
	got := aimOf(drain(out), 4)
	if got == nil || got.Value != markAim {
		t.Fatalf("minion aim=%+v", got)
	}
}

func aimOf(cmds []unit.Cmd, id uint64) *unit.SetAimPriority {
	var got *unit.SetAimPriority
	for _, c := range cmds {
		if v, ok := c.(unit.SetAimPriority); ok && v.UnitID == id {
			got = &v
		}
	}
	return got
}

func TestCloseDuringCooldownStillRams(t *testing.T) {
	a := &教父青年{roll: func() float64 { return 0.5 }}
	ctx, out := ctxOf()
	a.readyAt = 10
	a.buffUntil = 8
	a.Handle(ctx, senseAt(1, 100, []unit.Snapshot{foe(2, 80, 0)}))
	cmds := drain(out)
	var pass, stun bool
	for _, c := range cmds {
		if p, ok := c.(unit.Pass); ok && p.Hold {
			pass = true
		}
		if _, ok := c.(unit.Stun); ok {
			stun = true
		}
	}
	if !pass || a.phase != phRam {
		t.Fatalf("pass=%v phase=%d", pass, a.phase)
	}
	if stun {
		t.Fatal("stun before overlap")
	}
	a.Handle(ctx, unit.Collision{Time: 1.05, Other: foe(2, 20, 0)})
	stun = false
	for _, c := range drain(out) {
		if _, ok := c.(unit.Stun); ok {
			stun = true
		}
	}
	if !stun || a.phase != phReturn {
		t.Fatalf("stun=%v phase=%d", stun, a.phase)
	}
}

func TestBuffShotKnocks(t *testing.T) {
	b := &手枪弹{owner: 1, slot: 0, dmg: buffShotDmg, knock: true, ux: 1, uy: 0, aimed: true}
	out := make(chan unit.Cmd, 8)
	ctx := unit.Context{ID: 9, Kind: KindYouthBuffShot, Out: out}
	b.Handle(ctx, unit.Collision{Time: 1, Other: foe(2, 10, 0), NX: -1})
	cmds := drain(out)
	var fs *unit.AddFS
	var dmg float64
	for _, c := range cmds {
		if v, ok := c.(unit.AddFS); ok {
			fs = &v
		}
		if d, ok := c.(unit.Damage); ok {
			dmg = d.Amount
		}
	}
	if dmg != buffShotDmg || fs == nil || fs.BaseSpeed != knockSpeed || !fs.OnWall || fs.DX != 1 {
		t.Fatalf("dmg=%v fs=%+v", dmg, fs)
	}
}
