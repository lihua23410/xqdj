package sim

import (
	"math"

	unitpkg "xqdj/internal/unit"
)

type impulseFS struct {
	token     uint64
	dir       vec
	baseSpeed float64
	onWall    bool
	expiresAt float64
}

type fsComp struct {
	zone      unitpkg.FSZone
	value     float64
	expiresAt float64
}

type cruiseFS struct {
	dir        vec
	baseSpeed  float64
	components map[uint64]fsComp
}

func newCruiseFS(dir vec, base float64) *cruiseFS {
	if dir.len2() < 1e-12 {
		dir = vec{1, 0}
	} else {
		dir = dir.norm()
	}
	if base < 0 {
		base = 0
	}
	return &cruiseFS{
		dir:        dir,
		baseSpeed:  base,
		components: make(map[uint64]fsComp),
	}
}

func (c *cruiseFS) output() (speed float64, suppressed bool) {
	if c == nil {
		return 0, false
	}
	var dp, dt, fp float64
	ft := 1.0
	m := 1.0
	for _, comp := range c.components {
		switch comp.zone {
		case unitpkg.FSZoneDp:
			dp += comp.value
		case unitpkg.FSZoneDt:
			dt += comp.value
		case unitpkg.FSZoneFp:
			fp += comp.value
		case unitpkg.FSZoneFt:
			ft *= comp.value
		case unitpkg.FSZoneM:
			m *= comp.value
		}
	}
	speed = ft * ((c.baseSpeed+dp)*(1+dt) + fp) * m
	if speed < 0 {
		speed = 0
	}
	return speed, math.Abs(m) < 1e-12
}

func (u *unit) syncCruise() {
	if u == nil || u.cruiseFS == nil {
		return
	}
	u.cruise, _ = u.cruiseFS.output()
}

func (u *unit) skipDecel() bool {
	if u == nil {
		return true
	}
	if u.held {
		return true
	}
	if len(u.fsList) > 0 {
		return true
	}
	if u.cruiseFS == nil {
		return false
	}
	_, suppressed := u.cruiseFS.output()
	return suppressed
}

func (u *unit) expireFS(now float64) {
	if u == nil {
		return
	}
	if u.cruiseFS != nil {
		for tok, c := range u.cruiseFS.components {
			if c.expiresAt > 0 && now+1e-9 >= c.expiresAt {
				delete(u.cruiseFS.components, tok)
			}
		}
	}
	n := 0
	for _, fs := range u.fsList {
		if fs.expiresAt > 0 && now+1e-9 >= fs.expiresAt {
			continue
		}
		u.fsList[n] = fs
		n++
	}
	u.fsList = u.fsList[:n]
}

func (u *unit) expireOnWallFS() {
	if u == nil {
		return
	}
	n := 0
	for _, fs := range u.fsList {
		if fs.onWall {
			continue
		}
		u.fsList[n] = fs
		n++
	}
	u.fsList = u.fsList[:n]
}

func (u *unit) upsertImpulse(token uint64, dir vec, speed float64, onWall bool, expiresAt float64) {
	if dir.len2() < 1e-12 {
		dir = vec{1, 0}
	} else {
		dir = dir.norm()
	}
	if speed < 0 {
		speed = 0
	}
	next := impulseFS{
		token:     token,
		dir:       dir,
		baseSpeed: speed,
		onWall:    onWall,
		expiresAt: expiresAt,
	}
	for i := range u.fsList {
		if u.fsList[i].token == token {
			u.fsList[i] = next
			return
		}
	}
	u.fsList = append(u.fsList, next)
}

func (u *unit) removeImpulse(token uint64) {
	if u == nil {
		return
	}
	n := 0
	for _, fs := range u.fsList {
		if fs.token == token {
			continue
		}
		u.fsList[n] = fs
		n++
	}
	u.fsList = u.fsList[:n]
}

func (u *unit) applyImpulseOrClamp() {
	if u == nil {
		return
	}
	if len(u.fsList) > 0 {
		sum := vec{}
		for _, fs := range u.fsList {
			sum = sum.add(fs.dir.mul(fs.baseSpeed))
		}
		u.setVel(sum)
		return
	}
	if u.cruiseFS == nil {
		return
	}
	if _, suppressed := u.cruiseFS.output(); suppressed {
		if u.v.len2() > 1e-12 {
			u.cruiseFS.dir = u.v.norm()
		}
		u.setVel(vec{})
	}
}

func (u *unit) cruiseDir() vec {
	if u != nil && u.cruiseFS != nil && u.cruiseFS.dir.len2() > 1e-12 {
		return u.cruiseFS.dir.norm()
	}
	if u != nil && u.v.len2() > 1e-12 {
		return u.v.norm()
	}
	return vec{1, 0}
}

func (m *Match) expireFSLocked() {
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.stopped {
			continue
		}
		u.expireFS(m.time)
	}
}

func (m *Match) syncCruiseLocked() {
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.stopped {
			continue
		}
		u.syncCruise()
	}
}

func (m *Match) expireStunLocked() {
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.stopped || !u.stun || u.stunUntil <= 0 {
			continue
		}
		if m.time+1e-9 >= u.stunUntil {
			u.stun = false
			u.stunUntil = 0
		}
	}
}

func (m *Match) applyImpulseLocked() {
	for _, id := range m.order {
		u := m.units[id]
		if u == nil || u.stopped {
			continue
		}
		u.applyImpulseOrClamp()
	}
}

func (m *Match) applyAddFSLocked(c unitpkg.AddFS) {
	u := m.units[c.UnitID]
	if u == nil || u.stopped {
		return
	}
	u.upsertImpulse(c.Token, vec{c.DX, c.DY}, c.BaseSpeed, c.OnWall, c.ExpiresAt)
	u.applyImpulseOrClamp()
}

func (m *Match) applyRemoveFSLocked(c unitpkg.RemoveFS) {
	u := m.units[c.UnitID]
	if u == nil || u.stopped {
		return
	}
	u.removeImpulse(c.Token)
	u.applyImpulseOrClamp()
}

func (m *Match) applyAddFSComponentLocked(c unitpkg.AddFSComponent) {
	u := m.units[c.UnitID]
	if u == nil || u.stopped {
		return
	}
	if u.cruiseFS == nil {
		u.cruiseFS = newCruiseFS(u.v, u.cruise)
	}
	if u.cruiseFS.components == nil {
		u.cruiseFS.components = make(map[uint64]fsComp)
	}
	u.cruiseFS.components[c.Token] = fsComp{
		zone:      c.Zone,
		value:     c.Value,
		expiresAt: c.ExpiresAt,
	}
	u.syncCruise()
	u.applyImpulseOrClamp()
}

func (m *Match) applyRemoveFSComponentLocked(c unitpkg.RemoveFSComponent) {
	u := m.units[c.UnitID]
	if u == nil || u.stopped || u.cruiseFS == nil {
		return
	}
	delete(u.cruiseFS.components, c.Token)
	u.syncCruise()
	u.applyImpulseOrClamp()
}

func (m *Match) applySetFSDirectionLocked(c unitpkg.SetFSDirection) {
	u := m.units[c.UnitID]
	if u == nil || u.stopped || u.cruiseFS == nil {
		return
	}
	d := vec{c.VX, c.VY}
	if d.len2() < 1e-12 {
		return
	}
	u.cruiseFS.dir = d.norm()
}

func (m *Match) applySetCruiseLocked(c unitpkg.SetCruise) {
	u := m.units[c.UnitID]
	if u == nil || u.stopped {
		return
	}
	speed := c.Speed
	if speed < 0 {
		speed = 0
	}
	if u.cruiseFS != nil {
		u.cruiseFS.baseSpeed = speed
		u.syncCruise()
		return
	}
	u.cruise = speed
}
