package 人偶使

import (
	"math"
	"testing"
	"xqdj/internal/unit"
)

func TestSeekBombDirectHitDetonatesOnSense(t *testing.T) {
	spec := bombSpec{kind: bombSeek, dmg: demonDmg, tx: 400, ty: 0, explode: demonR}
	b := newBomb(unit.SpawnInfo{OwnerID: 1, Slot: 0, Data: spec})
	out := make(chan unit.Cmd, 16)
	ctx := unit.Context{State: &unit.MatchState{}, ID: 7, Kind: KindNingyushiBomb, Out: out}
	struck := foe(210, 0)
	bystander := foe(240, 0)
	bystander.ID = 3
	far := foe(400, 0)
	far.ID = 4

	b.Handle(ctx, unit.Collision{Time: 0.1, Other: struck})
	cmds := drain(out)
	if hasDamageTo(cmds, struck.ID) || hasDamageTo(cmds, bystander.ID) || hasDespawn(cmds, 7) {
		t.Fatalf("direct hit before the first sense must not blast from (0,0): %v", cmds)
	}

	b.Handle(ctx, unit.Sense{
		Time:   0.1,
		Self:   unit.Snapshot{ID: 7, X: 200, Y: 0, Radius: bombR},
		Nearby: []unit.Snapshot{bystander, far},
	})
	cmds = drain(out)
	if !hasDamage(cmds, struck.ID, demonDmg) || !hasDamage(cmds, bystander.ID, demonDmg) {
		t.Fatalf("blast should hit the body and the neighbor, cmds=%v", cmds)
	}
	if hasDamageTo(cmds, far.ID) {
		t.Fatal("unit outside the explode radius should be missed")
	}
	fx := lastFX(cmds, "blast")
	if fx == nil || math.Abs(fx.X-200) > 1e-6 || math.Abs(fx.Y) > 1e-6 {
		t.Fatalf("blast should sit on the bomb, not the origin: %+v", fx)
	}
}
