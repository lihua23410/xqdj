package 静电魔球

import (
	"math"
	"testing"

	"xqdj/internal/unit"
	_ "xqdj/map/hex"
)

func fighter(id uint64, slot int, x, y float64) unit.Snapshot {
	return unit.Snapshot{ID: id, Slot: slot, Role: unit.RoleFighter, X: x, Y: y, Radius: 18}
}

func beginDischarge(t *testing.T, nearby ...unit.Snapshot) (*静电魔球, unit.Context, unit.Sense, chan unit.Cmd, unit.FX) {
	t.Helper()
	out := make(chan unit.Cmd, 64)
	ctx := unit.Context{ID: 1, Kind: KindStaticBall, Out: out}
	s := unit.Sense{Time: 1, Self: fighter(1, 0, 0, 0), Nearby: nearby, Field: unit.HexField(), ForecastThreats: true}
	a := &静电魔球{charge: chargeMax}
	a.Handle(ctx, s)
	cmds := drain(out)
	warn, ok := effect(cmds, "static-warn")
	if !ok || len(damages(cmds)) != 0 || a.charge != 0 {
		t.Fatalf("满电应先清空电荷并预警: %+v", cmds)
	}
	return a, ctx, s, out, warn
}

func TestDischargeMissesWhenTargetLeavesWarning(t *testing.T) {
	a, ctx, s, out, warn := beginDischarge(t, fighter(2, 1, 140, 0))
	s.Time += arcWindup / 2
	s.Nearby[0].Y = 50
	s.Self.Y = -32
	a.Handle(ctx, s)
	cmds := drain(out)
	next, ok := effect(cmds, "static-warn")
	if !ok || next.X != warn.X || next.Y != warn.Y || next.VX != warn.VX || next.VY != warn.VY || len(damages(cmds)) != 0 {
		t.Fatalf("预警应固定且未出伤: %+v", cmds)
	}
	s.Time += arcWindup / 2
	a.Handle(ctx, s)
	cmds = drain(out)
	bolt, ok := effect(cmds, "static-bolt")
	if !ok || bolt.VY != warn.VY || bolt.Amount != 0 || len(damages(cmds)) != 0 || a.chain != nil {
		t.Fatalf("目标离线应打空并断链: %+v", cmds)
	}
}

func TestDischargeCanHitEnemyThatEntersWarning(t *testing.T) {
	decoy := unit.Snapshot{ID: 4, Slot: 1, Role: unit.RoleMinion, Mortal: true, X: 70, Y: 50, Radius: 12}
	a, ctx, s, out, _ := beginDischarge(t, fighter(2, 1, 150, 0), fighter(3, 0, 40, 0), decoy)
	s.Time += arcWindup
	s.Nearby[0].Y = 45
	s.Nearby[2].Y = 0
	a.Handle(ctx, s)
	got := damages(drain(out))
	if len(got) != 1 || got[0].To != 4 || got[0].Amount != chainDamage[0] {
		t.Fatalf("电弧应打到进线的活随从，友军不拦截: %+v", got)
	}
}

func TestDischargeWallCanInterruptWarnedArc(t *testing.T) {
	a, ctx, s, out, warn := beginDischarge(t, fighter(2, 1, 140, 0))
	s.Time += arcWindup
	s.Walls = []unit.WallView{{ID: 9, X1: 70, Y1: -40, X2: 70, Y2: 40, Radius: 6}}
	a.Handle(ctx, s)
	cmds := drain(out)
	bolt, ok := effect(cmds, "static-bolt")
	if !ok || bolt.VX >= warn.VX || math.Abs(bolt.VX-58) > 0.1 || len(damages(cmds)) != 0 {
		t.Fatalf("砌到线路上的墙应截断电弧: %+v", cmds)
	}
}

func TestDischargeSoloRelayStillRequiresAnotherWarning(t *testing.T) {
	a, ctx, s, out, _ := beginDischarge(t, fighter(2, 1, 140, 0))
	s.Time += arcWindup
	a.Handle(ctx, s)
	if got := damages(drain(out)); len(got) != 1 || got[0].Amount != chainDamage[0] {
		t.Fatalf("首跳未命中: %+v", got)
	}
	s.Time += hopRecovery
	a.Handle(ctx, s)
	cmds := drain(out)
	relay, ok := effect(cmds, "static-relay")
	if !ok || relay.X != 140 || relay.VX != 0 || len(damages(cmds)) != 0 {
		t.Fatalf("回流应到自己且不出伤: %+v", cmds)
	}
	s.Time += hopRecovery
	a.Handle(ctx, s)
	cmds = drain(out)
	if _, ok := effect(cmds, "static-warn"); !ok || len(damages(cmds)) != 0 {
		t.Fatalf("第二击应重新预警: %+v", cmds)
	}
	s.Time += arcWindup
	a.Handle(ctx, s)
	if got := damages(drain(out)); len(got) != 1 || got[0].Amount != chainDamage[1] || !got[0].NoFreeze {
		t.Fatalf("第二击应衰减并且不停帧: %+v", got)
	}
}

func TestDischargeChainsFromLastContactToAnotherEnemy(t *testing.T) {
	a, ctx, s, out, _ := beginDischarge(t, fighter(2, 1, 130, 0), fighter(3, 1, 155, 65))
	s.Time += arcWindup
	a.Handle(ctx, s)
	if got := damages(drain(out)); len(got) != 1 || got[0].To != 2 {
		t.Fatalf("首跳应命中最近战斗机: %+v", got)
	}
	s.Nearby[0].X = 90
	s.Nearby[0].Y = -40
	s.Time += hopRecovery
	a.Handle(ctx, s)
	cmds := drain(out)
	warn, ok := effect(cmds, "static-warn")
	if !ok || warn.X != 130 || warn.Y != 0 || len(damages(cmds)) != 0 {
		t.Fatalf("下一跳应从上一跳的接触点预警: %+v", cmds)
	}
	s.Time += arcWindup
	a.Handle(ctx, s)
	if got := damages(drain(out)); len(got) != 1 || got[0].To != 3 || got[0].Amount != chainDamage[1] {
		t.Fatalf("下一跳应命中另一名敌人并衰减: %+v", got)
	}
}

func TestDischargeStunCancelsWarning(t *testing.T) {
	a, ctx, s, out, _ := beginDischarge(t, fighter(2, 1, 140, 0))
	a.Handle(ctx, unit.Stunned{Time: s.Time + 0.1, Until: s.Time + 0.5})
	if _, ok := effect(drain(out), "static-cancel"); !ok {
		t.Fatal("被眩晕应取消预警")
	}
	s.Time += 0.5
	a.Handle(ctx, s)
	if got := damages(drain(out)); len(got) != 0 {
		t.Fatalf("恢复后不应补发旧电弧: %+v", got)
	}
}

func drain(out chan unit.Cmd) []unit.Cmd {
	var cmds []unit.Cmd
	for len(out) > 0 {
		cmds = append(cmds, <-out)
	}
	return cmds
}

func effect(cmds []unit.Cmd, name string) (unit.FX, bool) {
	for _, cmd := range cmds {
		if fx, ok := cmd.(unit.FX); ok && fx.Name == name {
			return fx, true
		}
	}
	return unit.FX{}, false
}

func damages(cmds []unit.Cmd) []unit.Damage {
	var out []unit.Damage
	for _, cmd := range cmds {
		if damage, ok := cmd.(unit.Damage); ok {
			out = append(out, damage)
		}
	}
	return out
}
