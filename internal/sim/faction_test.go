package sim

import (
	"math"
	"testing"
	"time"
	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func fighterByKind(m *Match, kind string) *unit {
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.kind == kind {
			return u
		}
	}
	return nil
}

func TestFactionAmpOutAndAmpIn(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindMenreiki)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	defer m.End()
	m.mu.Lock()
	src := fighterByKind(m, character.KindMenreiki)
	dst := fighterByKind(m, character.KindRanged)
	if src == nil || dst == nil {
		m.mu.Unlock()
		t.Fatal("missing fighters")
	}
	m.markFactionLocked(unitpkg.MarkFaction{
		UnitID: src.id, Faction: unitpkg.FactionRed,
		AmpOut: 1.25, AmpIn: 0.75, Collect: true,
	})
	m.markFactionLocked(unitpkg.MarkFaction{
		UnitID: dst.id, Faction: unitpkg.FactionCyan, Cycle: true,
	})
	before := dst.hp
	m.harmLocked(src.id, dst.id, 10)
	if math.Abs(dst.hp-(before-12.5)) > 1e-6 {
		t.Fatalf("diff faction dmg hp=%v want %v", dst.hp, before-12.5)
	}
	m.markFactionLocked(unitpkg.MarkFaction{
		UnitID: dst.id, Faction: unitpkg.FactionRed, Cycle: true,
	})
	before = src.hp
	m.harmLocked(dst.id, src.id, 10)
	if math.Abs(src.hp-(before-7.5)) > 1e-6 {
		t.Fatalf("same faction incoming hp=%v want %v", src.hp, before-7.5)
	}
	m.mu.Unlock()
}

func TestFactionCyclesOnWallNotOnPair(t *testing.T) {
	m := NewMatchSeeded(2)
	m.SetSlot(0, character.KindMenreiki)
	m.SetSlot(1, character.KindMelee)
	m.Start()
	defer m.End()
	m.mu.Lock()
	u := fighterByKind(m, character.KindMenreiki)
	o := fighterByKind(m, character.KindMelee)
	if u == nil || o == nil {
		m.mu.Unlock()
		t.Fatal("missing fighters")
	}
	m.markFactionLocked(unitpkg.MarkFaction{
		UnitID: u.id, Faction: unitpkg.FactionRed,
		Cycle: true, Collect: true, AmpOut: 1.25, AmpIn: 0.75,
	})
	start := u.faction
	m.cycleFactionLocked(u)
	if u.faction == "" || u.faction == start {
		m.mu.Unlock()
		t.Fatalf("wall cycle stayed %q", u.faction)
	}
	afterWall := u.faction
	n := vec{1, 0}
	m.send(u, unitpkg.Collision{Time: m.time, Other: o.snap(), NX: n.X, NY: n.Y})
	m.send(o, unitpkg.Collision{Time: m.time, Other: u.snap(), NX: -n.X, NY: -n.Y})
	if u.faction != afterWall {
		m.mu.Unlock()
		t.Fatalf("pair collision changed faction %q -> %q", afterWall, u.faction)
	}
	m.mu.Unlock()
}

func TestHealClampsToMaxHP(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindMenreiki)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	u := fighterByKind(m, character.KindMenreiki)
	if u == nil {
		t.Fatal("no menreiki")
	}
	u.hp = u.maxHP - 10
	m.applyCmdLocked(unitpkg.Heal{UnitID: u.id, Amount: 25})
	if math.Abs(u.hp-u.maxHP) > 1e-6 {
		t.Fatalf("hp=%v want %v", u.hp, u.maxHP)
	}
}

func TestMenreikiMarksBothFighters(t *testing.T) {
	m := NewMatchSeeded(4)
	m.SetSlot(0, character.KindMenreiki)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	defer m.End()
	for i := 0; i < 8; i++ {
		m.Tick()
		time.Sleep(2 * time.Millisecond)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	src := fighterByKind(m, character.KindMenreiki)
	dst := fighterByKind(m, character.KindRanged)
	if src == nil || dst == nil {
		t.Fatal("missing fighters")
	}
	if !unitpkg.ValidFaction(src.faction) || !src.factionCollect || src.factionAmpOut != 1.15 {
		t.Fatalf("self mark %+v collect=%v amp=%v", src.faction, src.factionCollect, src.factionAmpOut)
	}
	if !unitpkg.ValidFaction(dst.faction) || !dst.factionCycle || dst.factionAmpOut != 0 {
		t.Fatalf("enemy mark %+v cycle=%v ampOut=%v", dst.faction, dst.factionCycle, dst.factionAmpOut)
	}
	if len(src.factionBarrage) != 4 {
		t.Fatalf("barrage kinds=%v", src.factionBarrage)
	}
}

func TestFactionBarrageWhenFourSeen(t *testing.T) {
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindMenreiki)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	defer m.End()
	m.mu.Lock()
	src := fighterByKind(m, character.KindMenreiki)
	dst := fighterByKind(m, character.KindRanged)
	if src == nil || dst == nil {
		m.mu.Unlock()
		t.Fatal("missing fighters")
	}
	dst.p = src.p.add(vec{40, 0})
	before := dst.hp
	m.markFactionLocked(unitpkg.MarkFaction{
		UnitID: src.id, Faction: unitpkg.FactionCyan,
		Cycle: true, Collect: true, AmpOut: 1.25,
		Barrage: []string{"面具青", "面具红", "面具紫", "面具苍"},
	})
	for _, f := range unitpkg.AllFactions() {
		src.noteFaction(f)
	}
	m.maybeFactionCollectLocked(src)
	got := map[string]int{}
	for _, id := range m.order {
		o := m.units[id]
		if o == nil || o.stopped || o.owner != src.id {
			continue
		}
		got[o.kind]++
	}
	for _, kind := range []string{"面具青", "面具红", "面具紫", "面具苍"} {
		if got[kind] != 1 {
			m.mu.Unlock()
			t.Fatalf("shot %s count=%d all=%v", kind, got[kind], got)
		}
	}
	if len(src.seenList()) != 1 {
		t.Fatalf("seen after barrage=%v", src.seenList())
	}
	for _, d := range m.pending {
		m.applyCmdLocked(d)
	}
	m.pending = nil
	if dst.hp != before {
		t.Fatalf("aoe hp %v -> %v", before, dst.hp)
	}
	m.mu.Unlock()
}
