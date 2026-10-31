package sim

import unitpkg "xqdj/internal/unit"

// 约束跟随直属单位的实际生命周期，而非主人能否收到 Sense。
// 所有速度入口统一经 setVel，因此不拦墙挤出、碰撞分离或瞬移的位置修正。
func (m *Match) refreshOwnedVelocityLockLocked(owner uint64, kind string) {
	u := m.units[owner]
	if u == nil || u.stopped || u.zeroOwnedKind == "" || u.zeroOwnedKind != kind {
		return
	}
	present := false
	for _, id := range m.order {
		child := m.units[id]
		if child != nil && !child.stopped && child.owner == owner && child.kind == kind {
			present = true
			break
		}
	}
	if present == u.velocityLocked {
		return
	}
	u.velocityLocked = present
	if present {
		for _, fs := range u.fsList {
			m.notePushLocked(u, "瞬时", fs.baseSpeed, true)
		}
		u.fsList = nil
		u.holdVel = vec{}
		u.decelT = 0
		u.setVel(vec{})
	}
	m.noteHoldLocked(u, "零速", present)
	m.send(u, unitpkg.OwnedPresenceChanged{
		Time: m.time, Kind: kind, Present: present, SelfStunned: u.stun,
	})
}
