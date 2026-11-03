package sim

import (
	"strings"
	"testing"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
	圆 "xqdj/map/circle"
)

func TestOSPFOwnedTrafficAndDeathCleanup(t *testing.T) {
	m := NewHeadlessMatchSeeded(3)
	m.SetField(圆.Name)
	m.SetSlot(0, character.KindOSPF)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()

	body := unitsOf(m, character.KindOSPF)[0]
	birth := body.p
	m.Tick()
	m.Tick()
	nodes := unitsOf(m, character.KindOSPFNode)
	if len(nodes) != 1 || nodes[0].p != birth || nodes[0].owner != body.id {
		t.Fatalf("first node should stay at the owner's birth position %v", birth)
	}

	m.mu.Lock()
	body.p = vec{}
	m.time = 2
	m.emitLocked()
	m.mu.Unlock()
	m.Tick()

	m.mu.Lock()
	defer m.mu.Unlock()
	var shot *unit
	for _, u := range m.units {
		if strings.HasPrefix(u.kind, "OSPF报文") {
			shot = u
			break
		}
	}
	if shot == nil || shot.owner != body.id || !shot.passWalls || !shot.pass {
		t.Fatal("installation should emit a piercing packet owned by the fighter")
	}
	kind := shot.kind
	m.removeLocked(body)
	// Commands queued at the death boundary cannot recreate owned traffic.
	for _, child := range []string{character.KindOSPFNode, kind} {
		m.applyCmdLocked(unitpkg.Spawn{Kind: child, OwnerID: body.id})
	}
	for _, u := range m.units {
		if u.owner == body.id {
			t.Fatalf("owned unit %d (%s) survived its fighter", u.id, u.kind)
		}
	}
}
