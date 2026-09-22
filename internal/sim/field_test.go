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
	if len(msg.Fields) < 2 || msg.Fields[0] != unitpkg.NameCircle || msg.Fields[1] != unitpkg.NameHex {
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

func TestSetFieldUnknownIgnored(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetField("没有这份")
	m.mu.Lock()
	if m.spec.name != unitpkg.NameHex {
		m.mu.Unlock()
		t.Fatalf("default changed: %s", m.spec.name)
	}
	m.mu.Unlock()
	m.SetField(unitpkg.NameCircle)
	m.SetField("没有这份")
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.spec.name != unitpkg.NameCircle {
		t.Fatalf("circle replaced: %s", m.spec.name)
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

func TestCircleWallTurnsClockwise(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetField(unitpkg.NameCircle)
	m.SetSlot(0, character.KindMelee)
	m.SetSlot(1, character.KindMelee)
	m.Start()
	defer m.End()
	for i := 0; i < 2*TickHz; i++ {
		m.Tick()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	w := m.walls[0]
	mid := w.a.add(w.b).mul(0.5)
	if mid.len() > 0.2 {
		t.Fatalf("pivot %+v", mid)
	}
	if !wallEndsNear(w, vec{0, 110}, vec{0, -110}) {
		t.Fatalf("wall a=%+v b=%+v", w.a, w.b)
	}
}

func TestHardNailFollowsSpin(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetField(unitpkg.NameCircle)
	m.SetSlot(0, character.KindMelee)
	m.SetSlot(1, character.KindMelee)
	m.Start()
	defer m.End()
	m.mu.Lock()
	u := m.addUnitLocked(character.KindSickle, vec{0, 6}, vec{}, 1, 0)
	m.nailToWallLocked(u, unitpkg.Spawn{HardNail: true})
	id := u.id
	m.mu.Unlock()
	for i := 0; i < 2*TickHz; i++ {
		m.Tick()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	got := m.units[id]
	if got == nil || got.stopped {
		t.Fatal("sickle gone")
	}
	want := vec{6, 0}
	if got.p.sub(want).len() > 0.5 {
		t.Fatalf("sickle %+v want %+v", got.p, want)
	}
}

func TestSpinShovesOnce(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetField(unitpkg.NameCircle)
	m.SetSlot(0, character.KindReaper)
	m.SetSlot(1, character.KindReaper)
	m.Start()
	defer m.End()
	m.mu.Lock()
	var parked *unit
	hits := 0
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.role != unitpkg.RoleFighter {
			continue
		}
		u.setVel(vec{})
		u.cruise = 0
		if u.cruiseFS != nil {
			u.cruiseFS.baseSpeed = 0
		}
		if u.slot == 0 {
			u.p = vec{0, 30}
			parked = u
			u.tap = func(ev unitpkg.Event) {
				if e, ok := ev.(unitpkg.WallHit); ok && e.Kind.Hard() {
					hits++
				}
			}
			continue
		}
		u.p = vec{220, 0}
	}
	if parked == nil {
		m.mu.Unlock()
		t.Fatal("missing fighter")
	}
	m.mu.Unlock()
	for i := 0; i < 2*TickHz; i++ {
		m.Tick()
	}
	if hits != 1 {
		t.Fatalf("hard wall hits=%d", hits)
	}
}

func wallEndsNear(w *barrier, a, b vec) bool {
	const tol = 1.0
	return (w.a.sub(a).len() < tol && w.b.sub(b).len() < tol) ||
		(w.a.sub(b).len() < tol && w.b.sub(a).len() < tol)
}
