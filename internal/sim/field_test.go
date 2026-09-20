package sim

import (
	"encoding/json"
	"testing"
	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestSelectSnapshotListsFields(t *testing.T) {
	m := NewMatchSeeded(1)
	var msg struct {
		Fields []string `json:"fields"`
		Field  string   `json:"field"`
		Shape  string   `json:"fieldShape"`
		Walls  []struct {
			Hard   bool `json:"hard"`
			Square bool `json:"square"`
			Field  bool `json:"field"`
		} `json:"walls"`
	}
	if err := json.Unmarshal(m.SnapshotJSON(), &msg); err != nil {
		t.Fatal(err)
	}
	if len(msg.Fields) < 2 || msg.Fields[0] != unitpkg.NameHex || msg.Fields[1] != unitpkg.NameCircle {
		t.Fatalf("fields=%v", msg.Fields)
	}
	if msg.Field != unitpkg.NameHex || msg.Shape != unitpkg.ShapeHex {
		t.Fatalf("default field=%s shape=%s", msg.Field, msg.Shape)
	}
	m.SetField(unitpkg.NameCircle)
	if err := json.Unmarshal(m.SnapshotJSON(), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Field != unitpkg.NameCircle || msg.Shape != unitpkg.ShapeCircle {
		t.Fatalf("circle field=%s shape=%s", msg.Field, msg.Shape)
	}
	if len(msg.Walls) != 1 || !msg.Walls[0].Hard || !msg.Walls[0].Square || !msg.Walls[0].Field {
		t.Fatalf("select walls=%+v", msg.Walls)
	}
}

func TestSetFieldIgnoredAfterStart(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindMelee)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	defer m.End()
	m.SetField(unitpkg.NameCircle)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.spec.name != unitpkg.NameHex {
		t.Fatalf("field changed after start: %s", m.spec.name)
	}
}

func TestCircleFieldInstallsHardWallAndWalkableSpawn(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetField(unitpkg.NameCircle)
	m.SetSlot(0, character.KindMelee)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.spec.shape != unitpkg.ShapeCircle {
		t.Fatalf("shape=%s", m.spec.shape)
	}
	if len(m.walls) != 1 || !m.walls[0].hard || !m.walls[0].square || !m.walls[0].field {
		t.Fatalf("walls=%+v", m.walls)
	}
	f := m.liveFieldLocked()
	n := 0
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.role != unitpkg.RoleFighter {
			continue
		}
		n++
		if !f.Walkable(u.p.X, u.p.Y, u.radius) {
			t.Fatalf("spawn (%v,%v) not walkable", u.p.X, u.p.Y)
		}
	}
	if n != 2 {
		t.Fatalf("fighters=%d", n)
	}
}

func TestBreakWallsPassesHardWall(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetField(unitpkg.NameCircle)
	m.SetSlot(0, character.KindGodfather)
	m.SetSlot(1, character.KindMelee)
	m.Start()
	defer m.End()
	m.mu.Lock()
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.role == unitpkg.RoleFighter {
			u.p = vec{180, 180}
			u.setVel(vec{0, 0})
		}
	}
	shot := m.addUnitLocked(character.KindGodfatherShot, vec{0, 80}, vec{0, -600}, 1, 0)
	if shot == nil {
		m.mu.Unlock()
		t.Fatal("missing shot")
	}
	sid := shot.id
	if len(m.walls) != 1 {
		m.mu.Unlock()
		t.Fatalf("walls=%d", len(m.walls))
	}
	wid := m.walls[0].id
	m.mu.Unlock()
	for i := 0; i < 24; i++ {
		m.Tick()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.walls) != 1 || m.walls[0].id != wid {
		t.Fatalf("hard wall gone: %+v", m.walls)
	}
	shot = m.units[sid]
	if shot == nil || shot.stopped {
		t.Fatal("shot should pass 硬墙")
	}
	if shot.p.Y > -10 {
		t.Fatalf("shot y=%v want past the bar", shot.p.Y)
	}
}
