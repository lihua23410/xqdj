package sim

import (
	"encoding/json"
	"testing"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
	奇美拉 "xqdj/map/chimera"
)

func TestDuelHidesChimera(t *testing.T) {
	m := NewMatchSeeded(1)
	var msg struct {
		Fields []string `json:"fields"`
		Field  string   `json:"field"`
	}
	if err := json.Unmarshal(m.SnapshotJSON(), &msg); err != nil {
		t.Fatal(err)
	}
	for _, name := range msg.Fields {
		if name == 奇美拉.Name {
			t.Fatal("duel lists chimera")
		}
	}
	if msg.Field != "六边形" {
		t.Fatalf("default %s", msg.Field)
	}
	m.SetField(奇美拉.Name)
	if err := json.Unmarshal(m.SnapshotJSON(), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Field != "六边形" {
		t.Fatalf("duel accepted chimera: %s", msg.Field)
	}
}

func TestTeamDefaultsToChimeraAndSpawnsInRegions(t *testing.T) {
	m := NewMatchSeeded(3)
	m.UseTeam()
	m.mu.Lock()
	m.slots[0] = []string{character.KindDummy, character.KindDummy}
	m.slots[1] = []string{character.KindDummy}
	name := m.spec.name
	m.mu.Unlock()
	if name != 奇美拉.Name {
		t.Fatalf("team field %s", name)
	}
	var msg struct {
		Fields []string `json:"fields"`
	}
	if err := json.Unmarshal(m.SnapshotJSON(), &msg); err != nil {
		t.Fatal(err)
	}
	seen := false
	for _, n := range msg.Fields {
		if n == 奇美拉.Name {
			seen = true
		}
	}
	if !seen {
		t.Fatal("team list missing chimera")
	}
	m.Start()
	t.Cleanup(m.End)
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.role != unitpkg.RoleFighter {
			continue
		}
		if !奇美拉.RegionContains(u.slot, u.p.X, u.p.Y, u.radius) {
			t.Fatalf("slot %d spawned at %v,%v outside its region", u.slot, u.p.X, u.p.Y)
		}
	}
}
