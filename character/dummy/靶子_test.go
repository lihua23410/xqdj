package 靶子

import (
	"math"
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

func hasConfirm(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if _, ok := c.(unit.ConfirmDamage); ok {
			return true
		}
	}
	return false
}

func findDPS(cmds []unit.Cmd) (unit.FX, bool) {
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == "dps" {
			return fx, true
		}
	}
	return unit.FX{}, false
}

func near(got, want float64) bool {
	return math.Abs(got-want) < 1e-9
}

func newDummy(t *testing.T, out chan unit.Cmd) (unit.Actor, unit.Context) {
	t.Helper()
	actor := unit.NewActor(KindDummy, unit.SpawnInfo{Slot: 1})
	if actor == nil {
		t.Fatal("靶子没有登记")
	}
	return actor, unit.Context{ID: 9, Kind: KindDummy, Out: out}
}

// 第一枪之前也一直报 0；第一枪之后的全程 DPS 从开局算，不拿一帧当分母。
func TestDummyReportsDPSFromSpawn(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	actor, ctx := newDummy(t, out)

	actor.Handle(ctx, unit.Sense{Time: 0, Self: unit.Snapshot{ID: 9, Slot: 1}})
	fx, ok := findDPS(drain(out))
	if !ok || !near(fx.Amount, 0) || !near(fx.VX, 0) || !near(fx.VY, 0) {
		t.Fatalf("开局该报 0：%+v ok=%v", fx, ok)
	}
	if fx.UnitID != 9 || fx.Kind != KindDummy || fx.Slot != 1 {
		t.Fatalf("播报该带自己的身份：%+v", fx)
	}

	actor.Handle(ctx, unit.IncomingDamage{Token: 1, Amount: 10, Time: 1})
	cmds := drain(out)
	if !hasConfirm(cmds) {
		t.Fatalf("靶子该确认伤害：%v", cmds)
	}
	fx, ok = findDPS(cmds)
	if !ok || !near(fx.Amount, 10) || !near(fx.VX, 10) || !near(fx.VY, 10) {
		t.Fatalf("1 秒吃 10 伤，全程和近 5 秒都该是 10：%+v ok=%v", fx, ok)
	}

	actor.Handle(ctx, unit.Sense{Time: 2, Self: unit.Snapshot{ID: 9, Slot: 1}})
	fx, ok = findDPS(drain(out))
	if !ok || !near(fx.Amount, 5) || !near(fx.VX, 5) || !near(fx.VY, 10) {
		t.Fatalf("停手后全程 DPS 该往下走：%+v ok=%v", fx, ok)
	}

	actor.Handle(ctx, unit.Sense{Time: 7, Self: unit.Snapshot{ID: 9, Slot: 1}})
	fx, ok = findDPS(drain(out))
	if !ok || !near(fx.Amount, 10.0/7) || !near(fx.VX, 0) || !near(fx.VY, 10) {
		t.Fatalf("超过 5 秒没挨打，近 5 秒该归零：%+v ok=%v", fx, ok)
	}
}

// 近 5 秒 DPS 只看窗口里的伤害；窗口外的旧伤仍算进累计。
func TestDummyFiveSecondWindow(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	actor, ctx := newDummy(t, out)

	actor.Handle(ctx, unit.Sense{Time: 0, Self: unit.Snapshot{ID: 9, Slot: 1}})
	drain(out)
	for _, t0 := range []float64{1, 2, 3} {
		actor.Handle(ctx, unit.IncomingDamage{Token: uint64(t0), Amount: 10, Time: t0})
		drain(out)
	}

	actor.Handle(ctx, unit.Sense{Time: 4, Self: unit.Snapshot{ID: 9, Slot: 1}})
	fx, ok := findDPS(drain(out))
	if !ok || !near(fx.Amount, 7.5) || !near(fx.VX, 7.5) || !near(fx.VY, 30) {
		t.Fatalf("4 秒吃 30 伤：%+v ok=%v", fx, ok)
	}

	actor.Handle(ctx, unit.Sense{Time: 7.5, Self: unit.Snapshot{ID: 9, Slot: 1}})
	fx, ok = findDPS(drain(out))
	if !ok || !near(fx.Amount, 4) || !near(fx.VX, 2) || !near(fx.VY, 30) {
		t.Fatalf("7.5 秒时只剩 3 秒那一枪在窗口里：%+v ok=%v", fx, ok)
	}

	actor.Handle(ctx, unit.Sense{Time: 9, Self: unit.Snapshot{ID: 9, Slot: 1}})
	fx, ok = findDPS(drain(out))
	if !ok || !near(fx.Amount, 30.0/9) || !near(fx.VX, 0) || !near(fx.VY, 30) {
		t.Fatalf("9 秒时窗口全空：%+v ok=%v", fx, ok)
	}
}

// 开局很久之后才挨第一枪：分母仍是开局至今，而不是从第一枪算起。
func TestDummyUsesSpawnAsStart(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	actor, ctx := newDummy(t, out)

	actor.Handle(ctx, unit.Sense{Time: 10, Self: unit.Snapshot{ID: 9, Slot: 1}})
	drain(out)
	actor.Handle(ctx, unit.IncomingDamage{Token: 1, Amount: 10, Time: 11})
	fx, ok := findDPS(drain(out))
	if !ok || !near(fx.Amount, 10) || !near(fx.VX, 10) {
		t.Fatalf("第 11 秒挨 10 伤，DPS 该按 1 秒算：%+v ok=%v", fx, ok)
	}
}
