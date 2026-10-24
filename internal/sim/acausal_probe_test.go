package sim

import (
	"math"
	"testing"

	"xqdj/character"
	"xqdj/character/acausal"
)

func TestUnfinishedProbeDoesNotZeroDamage(t *testing.T) {
	st := &probeState{hp: 100, have: true, death: -1, drops: []float64{0, 0, 0}}
	st.closeUnfinished()
	bins := 非因果.Magnitudes(st.drops)
	_, dmg, _ := 非因果.Draw(bins, 0.5)
	if dmg < 1 {
		t.Fatalf("unfinished probe damage %v", dmg)
	}
}

func TestTwoAcausalPanic(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("两只非因果应该 panic")
		}
	}()
	m := NewMatchSeeded(3)
	m.mu.Lock()
	m.slots[0] = []string{非因果.KindAcausal}
	m.slots[1] = []string{非因果.KindAcausal}
	m.mu.Unlock()
	m.Start()
}

func TestAcausalProbeFlipsAndDiesOnClock(t *testing.T) {
	m := NewMatchSeeded(9)
	m.mu.Lock()
	m.slots[0] = []string{非因果.KindAcausal}
	m.slots[1] = []string{character.KindKnight}
	m.mu.Unlock()
	m.Start()
	t.Cleanup(m.End)

	tape := 非因果.Current()
	if tape == nil || tape.DeathTime < 0 {
		t.Fatalf("death %v", tape)
	}
	if len(tape.Fighters) != 1 || tape.Fighters[0].Kind != character.KindKnight {
		t.Fatalf("fighters %+v", tape.Fighters)
	}
	if len(tape.Fighters[0].Samples) < 2 {
		t.Fatalf("samples %d", len(tape.Fighters[0].Samples))
	}
	if len(tape.Self) < 2 {
		t.Fatalf("self samples %d", len(tape.Self))
	}
	if len(tape.SelfHP) != len(tape.Self) {
		t.Fatalf("self hp %d samples %d", len(tape.SelfHP), len(tape.Self))
	}
	if tape.Fighters[0].MaxHP <= 0 {
		t.Fatal("missing shadow cap")
	}
	m.mu.Lock()
	var me *unit
	for _, u := range m.units {
		if u != nil && u.kind == 非因果.KindAcausal {
			me = u
		}
	}
	m.mu.Unlock()
	if me == nil {
		t.Fatal("missing fighter")
	}
	if math.Abs(me.v.X+tape.SpawnVX) > 1e-3 || math.Abs(me.v.Y+tape.SpawnVY) > 1e-3 {
		t.Fatalf("v=(%v,%v) spawn=(%v,%v)", me.v.X, me.v.Y, tape.SpawnVX, tape.SpawnVY)
	}
	if m.intro == nil || m.intro.Death < 0 || len(m.intro.Frames) == 0 {
		t.Fatal("missing intro")
	}

	for i := 0; i < 8; i++ {
		m.Tick()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var shot *unit
	for _, u := range m.units {
		if u != nil && !u.stopped && u.kind == 非因果.KindLockShot {
			shot = u
		}
	}
	if shot == nil {
		t.Fatal("lock shot missing")
	}
}
