package 钓鱼佬

import (
	"testing"

	"xqdj/internal/unit"
)

func TestFishBirthValuesFollowSpawnWhenFactoriesInterleave(t *testing.T) {
	outA, outB := make(chan unit.Cmd, 2), make(chan unit.Cmd, 2)
	a, b := &钓鱼佬{}, &钓鱼佬{}
	// Two matches can use the same owner ID and enqueue their births before
	// either engine consumes the command. Consume B first, as scheduling may do.
	a.launch(unit.Context{ID: 1, Out: outA}, 0, 0, 1, 0, 20, 0, 1, false)
	b.launch(unit.Context{ID: 1, Out: outB}, 0, 0, 1, 0, 80, 0, 1, true)
	spawnA, spawnB := (<-outA).(unit.Spawn), (<-outB).(unit.Spawn)
	makeFish := func(spawn unit.Spawn) *鱼 {
		return unit.NewActor(spawn.Kind, unit.SpawnInfo{
			OwnerID: spawn.OwnerID, Slot: spawn.Slot, Stream: spawn.Stream, Data: spawn.Data,
		}).(*鱼)
	}
	fishB, fishA := makeFish(spawnB), makeFish(spawnA)
	if fishB.job.y != 80 || !fishB.job.spent || fishA.job.y != 20 || fishA.job.spent {
		t.Fatalf("birth values crossed matches: A=%+v B=%+v", fishA.job, fishB.job)
	}
}
