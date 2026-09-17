package 面灵气

import (
	"math"
	"sync"
	"xqdj/internal/unit"
)

var (
	liveMu      sync.Mutex
	liveFaction = map[uint64]string{}
	maskReady   = map[uint64][3]bool{}
)

func setLive(id uint64, faction string) {
	liveMu.Lock()
	liveFaction[id] = faction
	liveMu.Unlock()
}

func ownerFaction(id uint64) string {
	liveMu.Lock()
	defer liveMu.Unlock()
	return liveFaction[id]
}

func armMask(owner uint64, i int) {
	if i < 0 || i > 2 {
		return
	}
	liveMu.Lock()
	v := maskReady[owner]
	v[i] = true
	maskReady[owner] = v
	liveMu.Unlock()
}

func armAllMasks(owner uint64) {
	liveMu.Lock()
	maskReady[owner] = [3]bool{true, true, true}
	liveMu.Unlock()
}

func spendMask(owner uint64, i int) bool {
	if i < 0 || i > 2 {
		return false
	}
	liveMu.Lock()
	defer liveMu.Unlock()
	v := maskReady[owner]
	if !v[i] {
		return false
	}
	v[i] = false
	maskReady[owner] = v
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
		if !spendMask(a.owner, a.index) {
			return
		}
		strike(ctx, e.Other, maskDmg(ownerFaction(a.owner)))
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
			strike(ctx, e.Other, maskDmg(ownerFaction(b.owner)))
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
