package 纵火者

import (
	"math"
	"testing"

	"xqdj/internal/unit"
)

func ownerAt() unit.Snapshot {
	return unit.Snapshot{
		ID: 1, Kind: KindArsonist, Role: unit.RoleFighter, Slot: 0,
		HP: bodyHP, MaxHP: bodyHP, Radius: bodyRadius, Vision: vision, AimPriority: 15,
	}
}

func foeAt(id uint64, x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: id, Kind: "靶子", Role: unit.RoleFighter, Slot: 1,
		HP: 100, MaxHP: 100, Radius: bodyRadius, X: x, Y: y, AimPriority: 15,
	}
}

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

func spawns(cmds []unit.Cmd, kind string) []unit.Spawn {
	var got []unit.Spawn
	for _, c := range cmds {
		s, ok := c.(unit.Spawn)
		if ok && s.Kind == kind {
			got = append(got, s)
		}
	}
	return got
}

func damages(cmds []unit.Cmd) []unit.Damage {
	var got []unit.Damage
	for _, c := range cmds {
		if d, ok := c.(unit.Damage); ok {
			got = append(got, d)
		}
	}
	return got
}

func sense(t float64, self unit.Snapshot, near []unit.Snapshot) unit.Sense {
	return unit.Sense{Time: t, Self: self, Nearby: near, Field: unit.HexField()}
}

func TestThrowAlternatesAndDoesNotSkip(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	a := &纵火者{}
	ctx := unit.Context{ID: 1, Kind: KindArsonist, Out: out}
	self := ownerAt()
	near := []unit.Snapshot{foeAt(2, 120, 0)}

	a.Handle(ctx, sense(0, self, near))
	if n := len(spawns(drain(out), KindArsonField)); n != 1 {
		t.Fatalf("开局该放出油火，得到 %d", n)
	}

	a.Handle(ctx, sense(firstThrow-0.1, self, near))
	early := drain(out)
	if len(spawns(early, KindArsonOil))+len(spawns(early, KindArsonFire)) != 0 {
		t.Fatalf("到点前不该扔")
	}

	a.Handle(ctx, sense(firstThrow, self, near))
	got := spawns(drain(out), KindArsonOil)
	if len(got) != 1 {
		t.Fatalf("第一瓶该是油瓶，得到 %d", len(got))
	}
	if math.Hypot(got[0].VX, got[0].VY) < bottleSpeed-1e-6 {
		t.Fatalf("油瓶速率 = %v", math.Hypot(got[0].VX, got[0].VY))
	}

	a.Handle(ctx, sense(firstThrow+throwGap, self, near))
	if n := len(spawns(drain(out), KindArsonOil)); n != 1 {
		t.Fatalf("第二瓶该是油瓶，得到 %d", n)
	}

	a.Handle(ctx, sense(firstThrow+2*throwGap, self, nil))
	cmds := drain(out)
	if len(spawns(cmds, KindArsonOil))+len(spawns(cmds, KindArsonFire)) != 0 {
		t.Fatalf("没有目标不该扔，也不该跳种")
	}

	a.Handle(ctx, sense(firstThrow+2*throwGap+0.1, self, near))
	if n := len(spawns(drain(out), KindArsonOil)); n != 1 {
		t.Fatalf("人出现后该补扔留着的第三瓶油，得到 %d", n)
	}

	a.Handle(ctx, sense(firstThrow+2*throwGap+10, self, near))
	cmds = drain(out)
	if len(spawns(cmds, KindArsonFire)) != 1 || len(spawns(cmds, KindArsonOil)) != 0 {
		t.Fatalf("醒过来只补一瓶火，cmds=%v", cmds)
	}
}

func TestBottleBlock(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	b := newBottle(false)(unit.SpawnInfo{
		OwnerID: 1, Slot: 0,
		Data: bottleAim{X: 200, Y: 0, SX: 40, SY: 0},
	}).(*bottle)
	ctx := unit.Context{ID: 8, Kind: KindArsonOil, Out: out}

	b.Handle(ctx, unit.Collision{Other: unit.Snapshot{
		ID: 3, Role: unit.RoleFighter, Slot: 0, Radius: bodyRadius, X: 40, Y: 0,
	}})
	if len(spawns(drain(out), KindArsonOilMark)) != 0 {
		t.Fatalf("同边不该把瓶子挡住")
	}

	b.Handle(ctx, unit.Collision{Other: unit.Snapshot{
		ID: 4, Role: unit.RoleProjectile, Slot: 1, Radius: 6, X: 40, Y: 0,
	}})
	if len(spawns(drain(out), KindArsonOilMark)) != 0 || len(damages(drain(out))) != 0 {
		t.Fatalf("弹不该挡住瓶子，也不该出冲击伤害")
	}

	b.Handle(ctx, unit.Collision{Other: foeAt(2, 50, 0)})
	cmds := drain(out)
	if len(damages(cmds)) != 0 {
		t.Fatalf("碎裂不该出冲击伤害")
	}
	marks := spawns(cmds, KindArsonOilMark)
	if len(marks) != 1 || marks[0].X != 40 || marks[0].Y != 0 {
		t.Fatalf("该在瓶子的位置留油落点: %+v", marks)
	}

	wall := newBottle(true)(unit.SpawnInfo{
		OwnerID: 1, Slot: 0,
		Data: bottleAim{X: 200, Y: 0, SX: 10, SY: 4},
	}).(*bottle)
	wall.Handle(ctx, unit.WallHit{X: 10, Y: 4})
	marks = spawns(drain(out), KindArsonFireMark)
	if len(marks) != 1 || marks[0].X != 10 || marks[0].Y != 4 {
		t.Fatalf("撞墙该在阻拦处留火落点: %+v", marks)
	}

	fly := newBottle(false)(unit.SpawnInfo{
		OwnerID: 1, Slot: 0,
		Data: bottleAim{X: 50, Y: 0, SX: 0, SY: 0},
	}).(*bottle)
	fly.x, fly.y = 0, 0
	fly.have = true
	fly.Handle(ctx, unit.Sense{Self: unit.Snapshot{X: 80, Y: 0}})
	marks = spawns(drain(out), KindArsonOilMark)
	if len(marks) != 1 || marks[0].X != 50 || marks[0].Y != 0 {
		t.Fatalf("飞过记下的点该在那一点碎: %+v", marks)
	}
}

func TestFighterBodyDoesNotDamage(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	a := &纵火者{}
	ctx := unit.Context{ID: 1, Kind: KindArsonist, Out: out}
	a.Handle(ctx, unit.Collision{Other: foeAt(2, 20, 0)})
	if len(damages(drain(out))) != 0 {
		t.Fatalf("本体不该出伤")
	}
}

func fieldCtx(out chan unit.Cmd) unit.Context {
	return unit.Context{ID: 7, Kind: KindArsonField, Out: out}
}

func (f *油火) sense(out chan unit.Cmd, t float64, near []unit.Snapshot) []unit.Cmd {
	f.Handle(fieldCtx(out), sense(t, unit.Snapshot{ID: 7, Kind: KindArsonField, Role: unit.RoleHelper}, near))
	return drain(out)
}
