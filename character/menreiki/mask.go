package 面灵气

import (
	"math"

	"xqdj/internal/unit"
)

func setLive(scope *unit.MatchState, id uint64, faction string) {
	shared := stateFor(scope)
	shared.liveMu.Lock()
	shared.liveFaction[id] = faction
	shared.liveMu.Unlock()
}

func ownerFaction(scope *unit.MatchState, id uint64) string {
	shared := stateFor(scope)
	shared.liveMu.Lock()
	defer shared.liveMu.Unlock()
	return shared.liveFaction[id]
}

func armMask(scope *unit.MatchState, owner uint64, i int) {
	if i < 0 || i > 2 {
		return
	}
	shared := stateFor(scope)
	shared.liveMu.Lock()
	v := shared.maskReady[owner]
	v[i] = true
	shared.maskReady[owner] = v
	shared.liveMu.Unlock()
}

func armAllMasks(scope *unit.MatchState, owner uint64) {
	shared := stateFor(scope)
	shared.liveMu.Lock()
	shared.maskReady[owner] = [3]bool{true, true, true}
	shared.liveMu.Unlock()
}

func spendMask(scope *unit.MatchState, owner uint64, i int) bool {
	if i < 0 || i > 2 {
		return false
	}
	shared := stateFor(scope)
	shared.liveMu.Lock()
	defer shared.liveMu.Unlock()
	v := shared.maskReady[owner]
	if !v[i] {
		return false
	}
	v[i] = false
	shared.maskReady[owner] = v
	return true
}

// maskHitReady 面具对同一目标的出手 CD。三只面具可能同一帧圈住同一个人，
// 不加限制的话一次贴脸就是三发，多单位/大体型目标还会反复触发。
func maskHitReady(scope *unit.MatchState, owner, target uint64, now float64) bool {
	if owner == 0 || target == 0 {
		return false
	}
	shared := stateFor(scope)
	shared.liveMu.Lock()
	defer shared.liveMu.Unlock()
	m := shared.maskHitAt[owner]
	if m == nil {
		m = map[uint64]float64{}
		shared.maskHitAt[owner] = m
	}
	if now+1e-9 < m[target] {
		return false
	}
	m[target] = now + maskHitCD
	return true
}

type 面具 struct {
	owner  uint64
	slot   int
	index  int
	passed bool
}

func (a *面具) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		if !a.passed {
			a.passed = true
			ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
		}
	case unit.Collision:
		a.hit(ctx, e)
	}
}

func (a *面具) hit(ctx unit.Context, e unit.Collision) {
	if e.Other.ID == a.owner {
		return
	}
	if hitTarget(e.Other, a.slot) {
		if !spendMask(ctx.State, a.owner, a.index) {
			return
		}
		if !maskHitReady(ctx.State, a.owner, e.Other.ID, e.Time) {
			return
		}
		strike(ctx, e.Other, maskDmg(ownerFaction(ctx.State, a.owner)))
	}
}

func shotClearable(o unit.Snapshot, owner uint64) bool {
	if o.Role != unit.RoleProjectile {
		return false
	}
	if o.PassWalls || o.ArcSpan > 1e-6 {
		return false
	}
	if o.OwnerID == owner {
		return false
	}
	switch o.Kind {
	case KindMenreikiMask1, KindMenreikiMask2, KindMenreikiMask3, KindMenreikiShot:
		return false
	}
	return true
}

type 面灵气弹 struct {
	owner  uint64
	slot   int
	booted bool
	ux, uy float64
	next   float64
	tries  int
	dead   bool
}

func (b *面灵气弹) Handle(ctx unit.Context, ev unit.Event) {
	if b.dead {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		b.seek(ctx, e)
	case unit.Collision:
		if e.Other.ID == b.owner {
			return
		}
		if hitTarget(e.Other, b.slot) {
			strike(ctx, e.Other, maskDmg(ownerFaction(ctx.State, b.owner)))
			b.die(ctx)
			return
		}
		if e.Other.Role != unit.RoleFighter && e.Other.Role != unit.RoleProjectile {
			b.die(ctx)
		}
	case unit.WallHit:
		b.die(ctx)
	}
}

func (b *面灵气弹) seek(ctx unit.Context, s unit.Sense) {
	if !b.booted {
		n := math.Hypot(s.Self.VX, s.Self.VY)
		if n > 1e-6 {
			b.ux, b.uy = s.Self.VX/n, s.Self.VY/n
		} else {
			b.ux, b.uy = 1, 0
		}
		b.booted = true
		b.lock(s)
		b.next = s.Time + shotRetarget
	}
	if b.tries < shotRetargetN && s.Time+1e-9 >= b.next {
		b.lock(s)
		b.tries++
		b.next = s.Time + shotRetarget
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: b.ux * shotSpeed, VY: b.uy * shotSpeed}
}

func (b *面灵气弹) lock(s unit.Sense) {
	if e := enemyOf(s); e != nil {
		dx, dy := e.X-s.Self.X, e.Y-s.Self.Y
		if n := math.Hypot(dx, dy); n > 1e-6 {
			b.ux, b.uy = dx/n, dy/n
		}
	}
}

func (b *面灵气弹) die(ctx unit.Context) {
	if b.dead {
		return
	}
	b.dead = true
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}
