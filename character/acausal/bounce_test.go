package 非因果

import (
	"math"
	"reflect"
	"testing"

	"xqdj/internal/unit"
)

func ghostBounceFixture(kind string, samples []Sample) (*非因果, unit.Sense) {
	a := &非因果{
		booted: true, tape: &Tape{DeathTime: -1}, shotLive: true,
		seenMinion: map[uint64]taggedMinion{}, nextIndex: map[minionKey]int{},
	}
	g := ghostRef{id: 9, ready: true, hp: 100}
	role := unit.RoleHelper
	switch kind {
	case KindSelfGhost:
		a.self = g
		a.tape.Self, a.tape.SelfHP = samples, []float64{100}
	case KindFighterGhost:
		a.fighters, a.shadow = []ghostRef{g}, []float64{100}
		a.tape.Fighters = []FighterTrack{{Slot: 1, MaxHP: 100, Radius: 18, Samples: samples}}
	case KindMinionGhost:
		role = unit.RoleMinion
		a.minions = []ghostRef{g}
		a.tape.Minions = []MinionTrack{{MaxHP: 100, Radius: 18, Samples: samples}}
	}
	s := unit.Sense{
		Self:   unit.Snapshot{ID: 7, Kind: KindAcausal, Role: unit.RoleFighter, HP: 100, Radius: 18, VX: 160},
		Nearby: []unit.Snapshot{{ID: 9, OwnerID: 7, Kind: kind, Role: role, HP: 100, Radius: 18}},
	}
	return a, s
}

func ghostBounceCommands(a *非因果, s unit.Sense) []unit.Cmd {
	out := make(chan unit.Cmd, 128)
	a.Handle(unit.Context{ID: 7, Kind: KindAcausal, Out: out}, s)
	var commands []unit.Cmd
	for len(out) > 0 {
		commands = append(commands, <-out)
	}
	return commands
}

func bounceVelocity(t *testing.T, self unit.Snapshot, commands []unit.Cmd) (unit.SetVelocity, bool) {
	t.Helper()
	var velocity unit.SetVelocity
	found := false
	for _, cmd := range commands {
		if f, ok := cmd.(unit.Force); ok {
			if f.UnitID != 7 || found {
				t.Fatalf("bounce must only change the acausal body once: %+v", f)
			}
			velocity = unit.SetVelocity{UnitID: f.UnitID, VX: self.VX + f.AX*tickDT, VY: self.VY + f.AY*tickDT}
			found = true
		}
		if v, ok := cmd.(unit.SetVelocity); ok {
			t.Fatalf("bounce must not rescale velocity through SetVelocity: %+v", v)
		}
	}
	return velocity, found
}

func TestBodyBouncesOffEveryGhostKindWithoutChangingItsTrack(t *testing.T) {
	for _, kind := range []string{KindSelfGhost, KindFighterGhost, KindMinionGhost} {
		t.Run(kind, func(t *testing.T) {
			a, s := ghostBounceFixture(kind, []Sample{{X: 35}})
			original := a.tape.Clone()
			v, ok := bounceVelocity(t, s.Self, ghostBounceCommands(a, s))
			if !ok || math.Abs(v.VX+160) > 1e-9 || math.Abs(v.VY) > 1e-9 {
				t.Fatalf("missing body reflection: %+v, found=%v", v, ok)
			}
			if !reflect.DeepEqual(original, a.tape) {
				t.Fatal("body collision changed the recorded ghost track")
			}
			// 下一拍指令还没落地时，即使感知仍带旧方向，持续重叠也不能连拍翻向。
			if _, ok := bounceVelocity(t, s.Self, ghostBounceCommands(a, s)); ok {
				t.Fatal("continuous contact reflected repeatedly")
			}
			s.Self.X = -100
			if _, ok := bounceVelocity(t, s.Self, ghostBounceCommands(a, s)); ok {
				t.Fatal("separated body reflected")
			}
			s.Self.X = 0
			if _, ok := bounceVelocity(t, s.Self, ghostBounceCommands(a, s)); !ok {
				t.Fatal("new contact after separation did not reflect")
			}
		})
	}
}

func TestGhostBounceApproachGeometry(t *testing.T) {
	for _, tc := range []struct {
		name       string
		samples    []Sample
		vx, vy     float64
		wantX      float64
		wantBounce bool
	}{
		{name: "oblique reflection", samples: []Sample{{X: 35}}, vx: 160, vy: 120, wantX: -160, wantBounce: true},
		{name: "already modified speed", samples: []Sample{{X: 35}}, vx: 320, wantX: -320, wantBounce: true},
		{name: "fast approach", samples: []Sample{{X: 100}}, vx: 12000, wantX: -12000, wantBounce: true},
		{name: "moving ghost crosses next step", samples: []Sample{{X: 60}, {X: 0}}, vx: 160, wantX: -160, wantBounce: true},
		{name: "coincident centers", samples: []Sample{{}}, vx: 160, wantX: -160, wantBounce: true},
		{name: "already separating", samples: []Sample{{X: -35}}, vx: 160},
		{name: "tangent", samples: []Sample{{X: 36}}, vy: 160},
		{name: "stationary body", samples: []Sample{{X: 35}}},
		{name: "out of reach", samples: []Sample{{X: 100}}, vx: 160},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, s := ghostBounceFixture(KindFighterGhost, tc.samples)
			s.Self.VX, s.Self.VY = tc.vx, tc.vy
			v, ok := bounceVelocity(t, s.Self, ghostBounceCommands(a, s))
			if ok != tc.wantBounce {
				t.Fatalf("bounce=%v, want %v", ok, tc.wantBounce)
			}
			if ok && (math.Abs(v.VX-tc.wantX) > 1e-8 || math.Abs(v.VY-tc.vy) > 1e-8 || math.Abs(math.Hypot(v.VX, v.VY)-math.Hypot(tc.vx, tc.vy)) > 1e-8) {
				t.Fatalf("reflection changed the wrong component or speed: %+v", v)
			}
		})
	}
}

func TestGhostBounceIgnoresMissingGhostsProbeAndDeath(t *testing.T) {
	for _, kind := range []string{KindSelfGhost, KindFighterGhost, KindMinionGhost} {
		a, s := ghostBounceFixture(kind, []Sample{{X: 35}})
		s.Nearby = nil
		if _, ok := bounceVelocity(t, s.Self, ghostBounceCommands(a, s)); ok {
			t.Fatalf("missing %s still reflected the body", kind)
		}
	}
	a, s := ghostBounceFixture(KindFighterGhost, []Sample{{X: 35}})
	a.silent = true
	if commands := ghostBounceCommands(a, s); len(commands) != 0 {
		t.Fatalf("silent probe acquired gameplay behavior: %+v", commands)
	}
	a.silent = false
	a.tape.DeathTime = 0
	if _, ok := bounceVelocity(t, s.Self, ghostBounceCommands(a, s)); ok {
		t.Fatal("body reflected after its recorded death time")
	}
}
