package 泵之心_test

import (
	"encoding/json"
	"testing"
	heart "xqdj/character/engine_heart"
	"xqdj/internal/sim"
	"xqdj/internal/unit"
)

func TestTwoHeartsKeepSeparateFivePartSelectionsInRealMatch(t *testing.T) {
	m := sim.NewMatchSeeded(89)
	m.SetSlot(0, heart.KindEngineHeart)
	m.SetSlot(1, heart.KindEngineHeart)
	m.Start()
	defer m.End()
	sets := map[uint64]map[int]bool{}
	for tick := 0; tick < 900; tick++ {
		m.Tick()
		var frame struct {
			Effects []unit.FX `json:"effects"`
		}
		if err := json.Unmarshal(m.SnapshotJSON(), &frame); err != nil {
			t.Fatal(err)
		}
		for _, fx := range frame.Effects {
			if fx.Name != "heart-state" {
				continue
			}
			var data struct {
				Parts []struct {
					Type int `json:"type"`
				} `json:"parts"`
				Stack []int `json:"stack"`
			}
			if err := json.Unmarshal(fx.Data, &data); err != nil {
				t.Fatal(err)
			}
			if len(data.Parts) != 5 {
				t.Fatalf("owner %d has %d parts", fx.UnitID, len(data.Parts))
			}
			set := map[int]bool{}
			for _, p := range data.Parts {
				if set[p.Type] {
					t.Fatal("duplicate selected component")
				}
				set[p.Type] = true
			}
			if old := sets[fx.UnitID]; old != nil {
				for id := range set {
					if !old[id] {
						t.Fatal("match rerolled an absent component")
					}
				}
			} else {
				sets[fx.UnitID] = set
			}
			installed := map[int]bool{}
			for _, id := range data.Stack {
				if !set[id] || installed[id] {
					t.Fatal("invalid assembly")
				}
				installed[id] = true
			}
		}
	}
	if len(sets) != 2 {
		t.Fatalf("expected independent scenes for both fighters, got %d", len(sets))
	}
}
