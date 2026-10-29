package 非因果

import (
	"testing"
	"xqdj/internal/unit"
)

func TestActorOwnsIndependentProbeCopy(t *testing.T) {
	tape := &Tape{
		DeathTime: 1,
		Drops:     []float64{1}, Bins: []float64{2}, SelfHP: []float64{3},
		Self:     []Sample{{X: 4}},
		Fighters: []FighterTrack{{Samples: []Sample{{X: 5}}}},
		Minions:  []MinionTrack{{Samples: []Sample{{X: 6}}}},
	}
	a := unit.NewActor(KindAcausal, unit.SpawnInfo{Probe: tape}).(*非因果)
	b := unit.NewActor(KindAcausal, unit.SpawnInfo{Probe: tape}).(*非因果)
	tape.DeathTime = 99
	tape.Drops[0], tape.Bins[0], tape.SelfHP[0] = 99, 99, 99
	tape.Self[0].X, tape.Fighters[0].Samples[0].X, tape.Minions[0].Samples[0].X = 99, 99, 99
	a.tape.Fighters[0].Samples[0].X = 88
	if b.tape.DeathTime != 1 || b.tape.Drops[0] != 1 || b.tape.Bins[0] != 2 || b.tape.SelfHP[0] != 3 || b.tape.Self[0].X != 4 || b.tape.Fighters[0].Samples[0].X != 5 || b.tape.Minions[0].Samples[0].X != 6 {
		t.Fatal("actor probe shares mutable storage with another actor or source")
	}
	if got := unit.NewActor(KindAcausal, unit.SpawnInfo{}).(*非因果); got.tape != nil {
		t.Fatal("actor without a probe inherited another actor's tape")
	}
}
