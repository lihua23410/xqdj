package 迷走

import (
	"io/fs"
	"testing"

	"xqdj/internal/unit"
)

func drainAudioCommands(out <-chan unit.Cmd) (damage, attacks, switches int) {
	for len(out) > 0 {
		switch c := (<-out).(type) {
		case unit.Damage:
			damage++
		case unit.FX:
			if c.Name == "attack" {
				attacks++
			}
			if c.Name == "meander-switch" {
				switches++
			}
		}
	}
	return
}

func TestLitAttackSoundFollowsDamage(t *testing.T) {
	a := &点亮{}
	out := make(chan unit.Cmd, 16)
	ctx := unit.Context{ID: 1, Out: out}
	s := unit.Sense{
		Time: 10, Field: unit.HexField(),
		Self: unit.Snapshot{Radius: 100},
		Nearby: []unit.Snapshot{
			{ID: 2, Slot: 1, Role: unit.RoleFighter, X: 20, Radius: 18},
			{ID: 3, Slot: 1, Mortal: true, Y: 20, Radius: 14},
		},
	}
	a.Handle(ctx, s)
	if damage, attacks, _ := drainAudioCommands(out); damage != 2 || attacks != 1 {
		t.Fatalf("one sound per damage batch: damage=%d attacks=%d", damage, attacks)
	}
	s.Time = 10.1
	a.Handle(ctx, s)
	if damage, attacks, _ := drainAudioCommands(out); damage != 0 || attacks != 0 {
		t.Fatalf("cooldown should be silent: damage=%d attacks=%d", damage, attacks)
	}
	s.Time = 10.4
	a.Handle(ctx, s)
	if damage, attacks, _ := drainAudioCommands(out); damage != 2 || attacks != 1 {
		t.Fatalf("sound should retrigger after cooldown: damage=%d attacks=%d", damage, attacks)
	}
	for _, tc := range []struct {
		name string
		edit func(*unit.Sense)
	}{
		{name: "friendly", edit: func(s *unit.Sense) { s.Nearby[0].Slot = 0 }},
		{name: "outside", edit: func(s *unit.Sense) { s.Nearby[0].X = 200 }},
		{name: "wall", edit: func(s *unit.Sense) {
			s.Walls = []unit.WallView{{X1: 20, Y1: -30, X2: 20, Y2: 30, Radius: 6}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s.Time += 1
			s.Nearby = []unit.Snapshot{{ID: 4, Slot: 1, Role: unit.RoleFighter, X: 20, Radius: 18}}
			tc.edit(&s)
			a.Handle(ctx, s)
			if damage, attacks, _ := drainAudioCommands(out); damage != 0 || attacks != 0 {
				t.Fatalf("no attack means no sound: damage=%d attacks=%d", damage, attacks)
			}
		})
	}
}

func TestRingSwitchSoundOnlyOnDifferentRing(t *testing.T) {
	stayed, changed := 0, 0
	for seed := uint64(0); seed < 32; seed++ {
		a := ringStayActor(seed)
		out := make(chan unit.Cmd, 8)
		ctx := unit.Context{ID: 1, Out: out}
		s := unit.Sense{Time: 11, Self: unit.Snapshot{X: 60, Y: 80}}
		a.rail(ctx, s, pt{})
		if _, _, switches := drainAudioCommands(out); switches != 0 {
			t.Fatal("locked ring should be silent")
		}
		s.Time = 12
		a.rail(ctx, s, pt{})
		want := 0
		if a.ring != 0 {
			want, changed = 1, changed+1
		} else {
			stayed++
		}
		if _, _, switches := drainAudioCommands(out); switches != want {
			t.Fatalf("seed %d: want %d switch sounds, got %d", seed, want, switches)
		}
		a.flipSpin(12.1)
		s.Time = 12.2
		a.rail(ctx, s, pt{})
		if _, _, switches := drainAudioCommands(out); switches != 0 {
			t.Fatal("staying on or reversing along the same ring should be silent")
		}
	}
	if stayed == 0 || changed == 0 {
		t.Fatalf("missing exit outcomes: stayed=%d changed=%d", stayed, changed)
	}
	a := ringStayActor(1)
	out := make(chan unit.Cmd, 8)
	a.rail(unit.Context{ID: 1, Out: out}, unit.Sense{
		Time: 12, Self: unit.Snapshot{X: 120, Y: 100},
	}, pt{})
	if _, _, switches := drainAudioCommands(out); switches != 1 {
		t.Fatal("correcting onto a different ring should sound once")
	}
}

func TestMeanderAudioAssetsEmbedded(t *testing.T) {
	for _, file := range []string{"audio.js", "attack.wav", "switch.wav", "audio-sources.json", "LICENSE-CC0-1.0.txt"} {
		data, err := fs.ReadFile(unit.BallFS(), KindMeander+"/fx/"+file)
		if err != nil || len(data) == 0 {
			t.Fatalf("audio resource %s is unavailable: %v", file, err)
		}
	}
}
