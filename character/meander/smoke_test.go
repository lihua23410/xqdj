package 迷走_test

import (
	"testing"

	"xqdj/internal/sim"
	"xqdj/internal/unit"

	_ "xqdj/character"
	_ "xqdj/map"
)

func TestMeanderBootsWithoutPanic(t *testing.T) {
	if !unit.IsFighterKind("迷走") {
		t.Fatal("迷走 not registered")
	}
	m := sim.NewMatch()
	m.SetSlot(0, "迷走")
	m.SetSlot(1, "靶子")
	m.Start()
	defer m.End()
	for i := 0; i < 180; i++ {
		m.Tick()
	}
}
