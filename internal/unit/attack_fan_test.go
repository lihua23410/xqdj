package unit_test

import (
	"testing"
	_ "xqdj/character"
	"xqdj/internal/unit"
)

func fanFixture(t *testing.T, kind string, cfg unit.FanAttack) (unit.Context, unit.Actor, chan unit.Cmd, unit.Sense) {
	t.Helper()
	out := make(chan unit.Cmd, 128)
	ctx := unit.Context{State: &unit.MatchState{}, ID: 1, Kind: "owner", Out: out}.WithBirths()
	s := unit.Sense{Time: 1, Self: unit.Snapshot{ID: 1, Role: unit.RoleFighter, Radius: 18, VX: 100, Slot: 0}}
	unit.SyncAttackFan(ctx, s, kind, cfg)
	for len(out) > 0 {
		<-out
	}
	actor := unit.NewActor(kind, unit.SpawnInfo{State: ctx.State, OwnerID: 1, Slot: 0})
	if actor == nil {
		t.Fatal(kind)
	}
	return ctx, actor, out, s
}
func damages(out chan unit.Cmd) []unit.Damage {
	var result []unit.Damage
	for len(out) > 0 {
		if d, ok := (<-out).(unit.Damage); ok {
			result = append(result, d)
		}
	}
	return result
}

func TestMigratedFansQuoteFromTheFanOnBirthFrame(t *testing.T) {
	for _, kind := range []string{"pink贴身扇环", "盾斧突刺扇环", "回声突进扇环", "钓鱼佬高速扇环", "血契饿扑扇环", "教父青年撞击扇环", "我与周旋久本体扇环", "周旋久扇环", "人偶使冲刺扇环"} {
		t.Run(kind, func(t *testing.T) {
			ctx, actor, out, _ := fanFixture(t, kind, unit.FanAttack{Active: true, Once: true, Damage: 7})
			ctx.ID = 99
			hit := unit.Collision{Time: 1.01, Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1}}
			actor.Handle(ctx, hit)
			actor.Handle(ctx, hit)
			d := damages(out)
			if len(d) != 1 || d[0].From != 99 || d[0].Amount != 7 {
				t.Fatalf("damage=%+v", d)
			}
		})
	}
}

func TestFuelFanHitsEachTargetThreeTimesWithoutRepeating(t *testing.T) {
	ctx, actor, out, _ := fanFixture(t, "pink电锯扇环", unit.FanAttack{Active: true, Once: true, PerTarget: true, Damage: 3, Hits: 3, NoFreeze: true})
	ctx.ID = 99
	for _, id := range []uint64{2, 2, 3} {
		actor.Handle(ctx, unit.Collision{Time: 1.01, Other: unit.Snapshot{ID: id, Role: unit.RoleFighter, Slot: 1}})
	}
	d := damages(out)
	if len(d) != 6 {
		t.Fatalf("hits=%d", len(d))
	}
	for _, hit := range d {
		if hit.Amount != 3 || !hit.NoFreeze || hit.From != 99 {
			t.Fatal(hit)
		}
	}
}

func TestFanExpiryAndLockedTarget(t *testing.T) {
	ctx, actor, out, _ := fanFixture(t, "教父青年撞击扇环", unit.FanAttack{Active: true, Once: true, Damage: 2.5, TargetID: 2, Until: 2})
	ctx.ID = 99
	actor.Handle(ctx, unit.Collision{Time: 1.01, Other: unit.Snapshot{ID: 3, Role: unit.RoleFighter, Slot: 1}})
	actor.Handle(ctx, unit.Collision{Time: 2, Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1}})
	if d := damages(out); len(d) != 0 {
		t.Fatal(d)
	}
	actor.Handle(ctx, unit.Collision{Time: 1.1, Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1}})
	if d := damages(out); len(d) != 1 || d[0].To != 2 {
		t.Fatal(d)
	}
}

func TestN24UsesRemainingDamageAtContact(t *testing.T) {
	cfg := unit.FanAttack{Active: true, Once: true, Damage: 4 * 4.2}
	ctx, actor, out, s := fanFixture(t, "人偶使冲刺扇环", cfg)
	cfg.Damage = 2 * 4.2
	unit.SyncAttackFan(ctx, s, "人偶使冲刺扇环", cfg)
	ctx.ID = 99
	actor.Handle(ctx, unit.Collision{Time: 1.01, Other: unit.Snapshot{ID: 2, Role: unit.RoleFighter, Slot: 1}})
	if d := damages(out); len(d) != 1 || d[0].Amount != 8.4 {
		t.Fatal(d)
	}
}

func TestAxeGroupSweepDoesNotHurtBehind(t *testing.T) {
	cfg := unit.FanAttack{Active: true, Once: true, Damage: 3, Sweep: true}
	ctx, actor, out, s := fanFixture(t, "盾斧突刺扇环", cfg)
	s.Nearby = []unit.Snapshot{{ID: 3, X: 30, Radius: 18, Role: unit.RoleFighter, Slot: 1}, {ID: 4, X: -30, Radius: 18, Role: unit.RoleFighter, Slot: 1}}
	unit.SyncAttackFan(ctx, s, "盾斧突刺扇环", cfg)
	ctx.ID = 99
	actor.Handle(ctx, unit.Collision{Time: 1.01, Other: unit.Snapshot{ID: 2, X: 36, Role: unit.RoleFighter, Slot: 1}})
	d := damages(out)
	if len(d) != 2 {
		t.Fatal(d)
	}
	for _, hit := range d {
		if hit.To == 4 {
			t.Fatal("rear target hit")
		}
	}
}

func TestPartnerFanKeepsFriendlyDamage(t *testing.T) {
	ctx, actor, out, _ := fanFixture(t, "周旋久扇环", unit.FanAttack{Active: true, Damage: 3, Cooldown: .25, FriendKind: "我与周旋久"})
	ctx.ID = 99
	actor.Handle(ctx, unit.Collision{Time: 1.01, Other: unit.Snapshot{ID: 1, Kind: "我与周旋久", Role: unit.RoleFighter, Slot: 0}})
	if d := damages(out); len(d) != 1 || d[0].To != 1 {
		t.Fatal(d)
	}
}
