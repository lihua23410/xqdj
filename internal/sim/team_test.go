package sim

import (
	"math"
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestSideWipeEndsWithAlliesAlive(t *testing.T) {
	m := NewMatchSeeded(1)
	m.UseTeam()
	m.mu.Lock()
	m.slots[0] = []string{character.KindDummy}
	m.slots[1] = []string{character.KindDummy, character.KindMelee, character.KindRanged}
	m.mu.Unlock()
	m.Start()
	t.Cleanup(m.End)

	m.mu.Lock()
	want := m.lineup[1][0] + "、" + m.lineup[1][1] + "、" + m.lineup[1][2]
	var gone []*unit
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.role == unitpkg.RoleFighter && u.slot == 0 {
			gone = append(gone, u)
		}
	}
	for _, u := range gone {
		m.removeLocked(u)
	}
	m.checkWinLocked()
	phase, winner := m.phase, m.winner
	alive := 0
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && !u.stopped && u.role == unitpkg.RoleFighter && u.slot == 1 {
			alive++
		}
	}
	m.mu.Unlock()
	if phase != PhaseEnded {
		t.Fatal("1v3 should end when the single side is gone")
	}
	if winner != want {
		t.Fatalf("winner %q want %q", winner, want)
	}
	if alive != 3 {
		t.Fatalf("allies alive %d", alive)
	}
}

func TestBothSidesGoneIsDraw(t *testing.T) {
	m := NewMatchSeeded(2)
	m.SetSlot(0, character.KindDummy)
	m.SetSlot(1, character.KindMelee)
	m.Start()
	t.Cleanup(m.End)
	m.mu.Lock()
	var all []*unit
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.role == unitpkg.RoleFighter {
			all = append(all, u)
		}
	}
	for _, u := range all {
		m.removeLocked(u)
	}
	m.checkWinLocked()
	phase, winner := m.phase, m.winner
	m.mu.Unlock()
	if phase != PhaseEnded || winner != "平局" {
		t.Fatalf("phase %s winner %s", phase, winner)
	}
}

func TestDuelWinnerIsTheSideKind(t *testing.T) {
	m := NewMatchSeeded(3)
	m.SetSlot(0, character.KindDummy)
	m.SetSlot(1, character.KindMelee)
	m.Start()
	t.Cleanup(m.End)
	m.mu.Lock()
	var gone []*unit
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.role == unitpkg.RoleFighter && u.slot == 0 {
			gone = append(gone, u)
		}
	}
	for _, u := range gone {
		m.removeLocked(u)
	}
	m.checkWinLocked()
	winner := m.winner
	m.mu.Unlock()
	if winner != character.KindMelee {
		t.Fatalf("winner %s", winner)
	}
}

func TestTeamSpawnKeepsSidesApart(t *testing.T) {
	m := NewMatchSeeded(4)
	m.UseTeam()
	m.mu.Lock()
	m.slots[0] = []string{character.KindDummy, character.KindDummy}
	m.slots[1] = []string{character.KindDummy, character.KindDummy}
	m.mu.Unlock()
	m.Start()
	t.Cleanup(m.End)
	m.mu.Lock()
	defer m.mu.Unlock()
	var left, right []vec
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.role != unitpkg.RoleFighter {
			continue
		}
		if u.slot == 0 {
			left = append(left, u.p)
		} else {
			right = append(right, u.p)
		}
	}
	if len(left) != 2 || len(right) != 2 {
		t.Fatalf("spawned %d %d", len(left), len(right))
	}
	min := math.MaxFloat64
	for _, a := range left {
		for _, b := range right {
			if d := a.sub(b).len(); d < min {
				min = d
			}
		}
	}
	if min < teamSep-1 {
		t.Fatalf("cross-side distance %v", min)
	}
}

func TestPlaceRelaxesWhenGapDoesNotFit(t *testing.T) {
	m := NewMatchSeeded(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	others := []spawnSpot{{p: vec{0, 0}, r: 18, side: 0}}
	p := m.placeLocked(18, 1, others, 600)
	d := p.sub(others[0].p).len()
	if d >= 600 {
		t.Fatalf("gap should relax, dist %v", d)
	}
	if d < 18+18+spawnPad-1 {
		t.Fatalf("still need non-overlap, dist %v", d)
	}
}

func TestDrugVanishesWithItsGodfather(t *testing.T) {
	m := NewMatchSeeded(5)
	m.SetSlot(0, character.KindGodfather)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	t.Cleanup(m.End)
	m.mu.Lock()
	var boss *unit
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.kind == character.KindGodfather && u.slot == 0 {
			boss = u
			break
		}
	}
	if boss == nil {
		m.mu.Unlock()
		t.Fatal("no boss")
	}
	other := m.addUnitLocked(character.KindGodfather, vec{50, 0}, vec{}, 0, 0)
	drug := m.addUnitLocked(character.KindDrug, vec{20, 0}, vec{}, boss.id, 0)
	if other == nil || drug == nil {
		m.mu.Unlock()
		t.Fatal("add")
	}
	drugID, otherID := drug.id, other.id
	m.removeLocked(boss)
	drugGone := m.units[drugID] == nil
	mate := m.units[otherID]
	m.mu.Unlock()
	if !drugGone {
		t.Fatal("drug stayed")
	}
	if mate == nil || mate.stopped {
		t.Fatal("teammate gone")
	}
}

func TestSlimeKillCreditsSummoner(t *testing.T) {
	m := NewMatchSeeded(9)
	m.SetSlot(0, character.KindHero)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	t.Cleanup(m.End)
	for i := 0; i < 30 && len(unitsOf(m, character.KindHero)) == 0; i++ {
		m.Tick()
		time.Sleep(time.Millisecond)
	}
	heroes := unitsOf(m, character.KindHero)
	if len(heroes) == 0 {
		t.Fatal("no hero")
	}
	hero := heroes[0]
	m.mu.Lock()
	mate := m.addUnitLocked(character.KindMelee, vec{80, 0}, vec{}, 0, 0)
	slime := m.addUnitLocked(character.KindSlime, vec{120, 0}, vec{}, hero.id, 1)
	if mate == nil || slime == nil {
		m.mu.Unlock()
		t.Fatal("spawn")
	}
	slimeID, mateID := slime.id, mate.id
	m.mu.Unlock()
	m.cmds <- unitpkg.Damage{From: mateID, To: slimeID, Amount: 999}
	var exp float64
	for i := 0; i < 40; i++ {
		m.Tick()
		time.Sleep(time.Millisecond)
		m.mu.Lock()
		for _, fx := range m.fx {
			if fx.Name == "hero-lv" && fx.UnitID == hero.id && fx.VX > exp {
				exp = fx.VX
			}
		}
		dead := m.units[slimeID] == nil
		m.mu.Unlock()
		if dead && exp >= 1 {
			return
		}
	}
	t.Fatalf("slime kill exp %v", exp)
}

func TestTeamPickAppendsAndDrops(t *testing.T) {
	m := NewMatchSeeded(1)
	m.UseTeam()
	m.SetSlot(0, character.KindDummy)
	m.SetSlot(0, character.KindMelee)
	m.SetSlot(0, character.KindRanged)
	m.SetSlot(0, character.KindWaller)
	m.RemoveSeat(0, 1)
	m.mu.Lock()
	got := append([]string{}, m.slots[0]...)
	m.mu.Unlock()
	want := []string{character.KindDummy, character.KindRanged}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("slots %v want %v", got, want)
	}
	m.Start()
	m.mu.Lock()
	phase := m.phase
	m.mu.Unlock()
	if phase != PhaseSelect {
		t.Fatalf("empty right side must not start, phase %s", phase)
	}
}
