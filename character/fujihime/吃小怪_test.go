package 藤姬

import (
	"testing"

	"xqdj/internal/unit"
)

// 吃小怪：CD 好了、身边 240 内有血量低于 20 的敌方活随从，就吞掉它换一只幻役，并喂一层藤色。
// 一条龙验完：目标被撤、幻役出生、下一只幻役接手整包 + 1 层、CD 压满。

func devourRig() (*藤姬, unit.Context, chan unit.Cmd, unit.Snapshot) {
	out := make(chan unit.Cmd, 256)
	f := &藤姬{}
	ctx := unit.Context{State: &unit.MatchState{}, ID: 1, Kind: KindFujihime, Out: out, Stream: 7}
	self := unit.Snapshot{
		ID: 1, Kind: KindFujihime, Role: unit.RoleFighter, Slot: 0,
		X: 0, Y: 0, Radius: fujiRadius, HP: fujiHP, MaxHP: fujiHP,
	}
	f.booted = true
	f.rng = unit.NewRNG(7)
	f.nextGrant = 1e9
	f.devourAt = 0
	return f, ctx, out, self
}

func minionAt(id uint64, x, y, hp float64) unit.Snapshot {
	return unit.Snapshot{
		ID: id, Kind: "小随从", Role: unit.RoleMinion, Mortal: true, Slot: 1,
		X: x, Y: y, Radius: 10, HP: hp, MaxHP: 50,
	}
}

func TestDevourEatsSmallMinionIntoDoll(t *testing.T) {
	f, ctx, out, self := devourRig()
	bug := minionAt(5, 100, 0, 15)

	f.Handle(ctx, unit.Sense{Time: 3, Self: self, Nearby: []unit.Snapshot{bug}})
	cmds := drain(out)

	if !hasCmd(cmds, func(c unit.Despawn) bool { return c.UnitID == 5 }) {
		t.Fatalf("该把小随从吞掉: %v", cmds)
	}
	if !hasCmd(cmds, func(c unit.Spawn) bool { return c.Kind == KindFujihimeDoll }) {
		t.Fatalf("吞掉之后该召一只幻役: %v", cmds)
	}
	// 幻役接手整包 + 1 层。
	if want := f.pool + float64(devourLayers)*tosePoolPerLayer; f.waitDoll != want {
		t.Fatalf("幻役该接手 %v，waitDoll=%v", want, f.waitDoll)
	}
	if f.devourAt != 3+devourCD {
		t.Fatalf("CD 该压到 %v，实际 %v", 3+devourCD, f.devourAt)
	}
}

func TestDevourSkipsFatOrFarMinion(t *testing.T) {
	for _, tc := range []struct {
		name string
		bug  unit.Snapshot
	}{
		{"血太厚", minionAt(5, 100, 0, 25)},
		{"太远", minionAt(5, devourRange+40, 0, 5)},
	} {
		f, ctx, out, self := devourRig()
		f.Handle(ctx, unit.Sense{Time: 3, Self: self, Nearby: []unit.Snapshot{tc.bug}})
		cmds := drain(out)
		if hasCmd(cmds, func(c unit.Despawn) bool { return c.UnitID == 5 }) {
			t.Fatalf("%s 时不该吞: %v", tc.name, cmds)
		}
		if hasCmd(cmds, func(c unit.Spawn) bool { return c.Kind == KindFujihimeDoll }) {
			t.Fatalf("%s 时不该召幻役: %v", tc.name, cmds)
		}
	}
}

// 场上有自己的幻役时不吃：新幻役接不到这一包池子，等于白给。
func TestDevourWaitsForNoDoll(t *testing.T) {
	f, ctx, out, self := devourRig()
	bug := minionAt(5, 100, 0, 15)
	doll := unit.Snapshot{
		ID: 9, Kind: KindFujihimeDoll, Role: unit.RoleMinion, Slot: 0, OwnerID: 1,
		X: 60, Y: 0, Radius: dollRadius, HP: dollHP, MaxHP: dollHP,
	}
	f.Handle(ctx, unit.Sense{Time: 3, Self: self, Nearby: []unit.Snapshot{bug, doll}})
	cmds := drain(out)
	if hasCmd(cmds, func(c unit.Despawn) bool { return c.UnitID == 5 }) {
		t.Fatalf("幻役在场时不该吞: %v", cmds)
	}
}

// 队友自己人、或者不是活随从的东西，不是食物。
func TestDevourOnlyEatsEnemyMinions(t *testing.T) {
	f, ctx, out, self := devourRig()
	ally := minionAt(5, 100, 0, 5)
	ally.Slot = 0
	notMortal := minionAt(6, 90, 0, 5)
	notMortal.Mortal = false
	f.Handle(ctx, unit.Sense{Time: 3, Self: self, Nearby: []unit.Snapshot{ally, notMortal}})
	cmds := drain(out)
	if hasCmd(cmds, func(c unit.Despawn) bool { return true }) {
		t.Fatalf("同边/非活随从都不该吞: %v", cmds)
	}
}
