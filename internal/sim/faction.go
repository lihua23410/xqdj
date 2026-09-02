package sim

import (
	"math"

	unitpkg "xqdj/internal/unit"
)

func (m *Match) markFactionLocked(c unitpkg.MarkFaction) {
	u := m.units[c.UnitID]
	if u == nil || u.stopped || u.role != unitpkg.RoleFighter {
		return
	}
	f := c.Faction
	if !unitpkg.ValidFaction(f) {
		f = unitpkg.PickFaction(m.rng.IntN(4))
	}
	prev := u.faction
	u.faction = f
	u.factionCycle = c.Cycle
	u.factionAmpOut = c.AmpOut
	u.factionAmpIn = c.AmpIn
	u.factionCollect = c.Collect
	if len(c.Barrage) > 0 {
		u.factionBarrage = append([]string(nil), c.Barrage...)
	}
	u.noteFaction(f)
	m.maybeFactionCollectLocked(u)
	if prev != f {
		m.fx = append(m.fx, unitpkg.FX{
			Name: "faction", UnitID: u.id, Kind: f,
			X: u.p.X, Y: u.p.Y, Slot: u.slot,
		})
	}
}

func (m *Match) cycleFactionLocked(u *unit) {
	if u == nil || !u.factionCycle || u.faction == "" {
		return
	}
	if m.time+1e-9 < u.factionNext {
		return
	}
	next := unitpkg.PickOtherFaction(u.faction, m.rng.IntN(3))
	u.faction = next
	u.factionNext = m.time + 0.2
	u.noteFaction(next)
	m.maybeFactionCollectLocked(u)
	m.fx = append(m.fx, unitpkg.FX{
		Name: "faction", UnitID: u.id, Kind: next,
		X: u.p.X, Y: u.p.Y, Slot: u.slot,
	})
}

func (m *Match) maybeFactionCollectLocked(u *unit) {
	if u == nil || !u.factionCollect {
		return
	}
	if len(u.factionSeen) < len(unitpkg.AllFactions()) {
		return
	}
	n := len(u.factionBarrage)
	if n == 0 {
		return
	}
	base := math.Atan2(u.v.Y, u.v.X)
	if u.v.len2() < 1e-12 {
		base = 0
	}
	step := 2 * math.Pi / float64(n)
	for i, kind := range u.factionBarrage {
		spec, ok := unitpkg.Lookup(kind)
		if !ok || spec.Fighter {
			continue
		}
		ang := base + step*float64(i)
		ux, uy := math.Cos(ang), math.Sin(ang)
		gap := u.radius + spec.Radius + 1.5
		m.addUnitLocked(kind, vec{u.p.X + ux*gap, u.p.Y + uy*gap}, vec{ux * spec.Speed, uy * spec.Speed}, u.id, u.slot)
	}
	u.factionSeen = map[string]bool{}
	u.noteFaction(u.faction)
}

func (m *Match) factionBearer(u *unit) *unit {
	if u == nil {
		return nil
	}
	if u.faction != "" {
		return u
	}
	if u.owner != 0 {
		if o := m.units[u.owner]; o != nil && o.faction != "" {
			return o
		}
	}
	return u
}

func (m *Match) scaleDamage(atk, def *unit, amount float64) float64 {
	var af, df string
	if atk != nil {
		af = atk.faction
	}
	if def != nil {
		df = def.faction
	}
	if atk != nil && atk.factionAmpOut != 0 && af != "" && df != "" && af != df {
		amount *= atk.factionAmpOut
	}
	if def != nil && def.factionAmpIn != 0 && af != "" && df != "" && af == df {
		amount *= def.factionAmpIn
	}
	return amount
}

func (m *Match) harmLocked(fromID, toID uint64, amount float64) {
	to := m.units[toID]
	if to == nil || to.stopped || to.role != unitpkg.RoleFighter || amount <= 0 {
		return
	}
	from := m.units[fromID]
	amount = m.scaleDamage(m.factionBearer(from), to, amount)
	m.fx = append(m.fx, unitpkg.FX{
		Name: "hurt", UnitID: to.id, Kind: to.kind,
		X: to.p.X, Y: to.p.Y, Slot: to.slot, Amount: amount,
	})
	to.hp -= amount
	m.hitStop = HitStopFrames
	if to.hp <= 0 {
		to.hp = 0
		m.removeLocked(to)
	} else {
		m.swapOwnedLocked(to.id)
	}
}
