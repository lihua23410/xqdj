package 回声

import (
	"testing"

	"xqdj/internal/unit"
)

func audioFXCount(cmds []unit.Cmd, name string) int {
	n := 0
	for _, cmd := range cmds {
		if fx, ok := cmd.(unit.FX); ok && fx.Name == name {
			n++
		}
	}
	return n
}

func TestSoundCuesFollowTheActualScanAndDashCycle(t *testing.T) {
	out := make(chan unit.Cmd, 256)
	ctx := unit.Context{ID: 1, Kind: KindEcho, Out: out}.WithBirths()
	e := &回声{}
	sense := func(now float64, foes ...unit.Snapshot) []unit.Cmd {
		e.Handle(ctx, unit.Sense{Time: now, Self: echoSelf(120, 0, 0), Nearby: foes, Field: unit.HexField()})
		return drain(out)
	}
	if audioFXCount(sense(0), "echo-scan") != 1 || audioFXCount(sense(.1), "echo-scan") != 0 {
		t.Fatal("scan entry should sound once")
	}
	var fired []unit.Cmd
	for i := 2; i <= 20; i++ {
		fired = append(fired, sense(float64(i)/10)...)
	}
	if audioFXCount(fired, "echo-fire") != countSpawns(fired, KindStar) || audioFXCount(fired, "echo-fire") != 1 {
		t.Fatal("every actual star launch needs exactly one cue")
	}
	foe := echoFoe(9, 10, countKind(1))
	locked := sense(2.1, foe)
	if audioFXCount(locked, "echo-found") != 1 {
		t.Fatal("new six-second window needs a lock cue")
	}
	if audioFXCount(sense(2.2, foe), "echo-found") != 0 {
		t.Fatal("existing window replayed its lock cue")
	}
	for i := 0; i < 2; i++ {
		e.Handle(ctx, unit.Collision{Time: 2.3, Other: foe})
	}
	if audioFXCount(drain(out), "echo-ram") != 1 {
		t.Fatal("one dash must only sound its one damaging ram")
	}
	e.Handle(ctx, unit.WallHit{Time: 2.4, X: 100, Y: 12})
	wall := drain(out)
	if audioFXCount(wall, "echo-wall") != 1 || e.mode != modeLock {
		t.Fatal("wall impact did not signal the re-aim gap")
	}
	if audioFXCount(sense(2.55, foe), "echo-dash") != 1 {
		t.Fatal("next wall-to-wall dash is missing its visual effect")
	}
	if audioFXCount(sense(8.1, foe), "echo-release") != 0 || e.mode != modeScan {
		t.Fatal("last window expiry must return to scanning silently")
	}
	if audioFXCount(sense(8.2), "echo-release") != 0 {
		t.Fatal("scan state replayed lock release")
	}
}

func TestProjectileSoundCuesDoNotRepeatAfterDespawn(t *testing.T) {
	out := make(chan unit.Cmd, 64)
	ctx := unit.Context{ID: 3, Kind: KindShard, Out: out}
	shard := &碎片{slot: 0}
	for i := 0; i < 9; i++ {
		shard.Handle(ctx, unit.WallHit{Time: float64(i), X: 20, Y: 30})
	}
	cmds := drain(out)
	if audioFXCount(cmds, "echo-bounce") != 8 || audioFXCount(cmds, "echo-fade") != 1 || !hasDespawn(cmds, 3) {
		t.Fatal("eight bounces should end in exactly one fade cue")
	}
	shard.Handle(ctx, unit.WallHit{Time: 10})
	if len(drain(out)) != 0 {
		t.Fatal("despawned shard emitted another event")
	}
	star := &星弹{}
	star.Handle(ctx, unit.Sense{Time: 0})
	star.Handle(ctx, unit.Sense{Time: 3})
	star.Handle(ctx, unit.Sense{Time: 4})
	if audioFXCount(drain(out), "echo-fade") != 1 {
		t.Fatal("expired star needs exactly one fade cue")
	}
}
