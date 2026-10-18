package 非因果

import (
	"math"

	"xqdj/internal/unit"
)

// 锁影弹慢速转向残影上的采样点。碰到任何实心单位或墙就碎、断锁，
// 下一发等频谱射速，不穿透。残影是助手，引擎不发碰撞，命中靠距离。
type 锁影弹 struct {
	owner  uint64
	bin    int
	amount float64
	armed  bool
	got    bool
	dead   bool
}

func (m *锁影弹) Handle(ctx unit.Context, ev unit.Event) {
	if m.dead {
		return
	}
	m.arm(ctx, ev)
	switch e := ev.(type) {
	case unit.Sense:
		m.onSense(ctx, e)
	case unit.Collision:
		m.onBody(ctx, e)
	case unit.WallHit:
		m.onWall(ctx, e)
	}
}

func (m *锁影弹) arm(ctx unit.Context, ev unit.Event) {
	if m.armed {
		return
	}
	m.armed = true
	b := loadBook(m.owner)
	if b == nil {
		return
	}
	sh, ok := b.takeShot()
	if !ok {
		return
	}
	m.bin = sh.bin
	m.amount = sh.amount
	m.got = true
	if s, ok := ev.(unit.Sense); ok {
		ctx.Out <- unit.FX{
			Name: fxLockArm, UnitID: ctx.ID, Kind: KindLock,
			X: s.Self.X, Y: s.Self.Y, Amount: m.amount, Slot: m.bin,
		}
	}
}

func (m *锁影弹) onSense(ctx unit.Context, s unit.Sense) {
	res := findResidual(s, m.owner)
	if res == nil {
		return
	}
	vx, vy := steer(s.Self.VX, s.Self.VY, res.X-s.Self.X, res.Y-s.Self.Y, seekTurn, missileSpeed, tickDT)
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: vx, VY: vy}
	reach := s.Self.Radius + res.Radius
	if reach <= 0 {
		reach = missileRadius + residualRadius
	}
	if math.Hypot(res.X-s.Self.X, res.Y-s.Self.Y) > reach {
		return
	}
	m.settle(ctx, s.Time, s.Self.X, s.Self.Y, true, res.ID, 0, false)
}

func (m *锁影弹) onBody(ctx unit.Context, e unit.Collision) {
	if e.Other.ID == m.owner || e.Other.Kind == KindResidual || e.Other.Nonsolid {
		return
	}
	m.settle(ctx, e.Time, e.Other.X, e.Other.Y, false, 0, e.Other.ID, true)
}

func (m *锁影弹) onWall(ctx unit.Context, e unit.WallHit) {
	m.settle(ctx, e.Time, e.X, e.Y, false, 0, 0, false)
}

func (m *锁影弹) settle(ctx unit.Context, now, x, y float64, ghost bool, residualID, blocker uint64, unitBlock bool) {
	if m.dead {
		return
	}
	m.dead = true
	b := loadBook(m.owner)
	var g ghostEnd
	if b != nil {
		amt := 0.0
		if ghost && m.got {
			amt = m.amount
		}
		g = b.endMissile(now, ghost && m.got, amt)
	}
	if ghost && g.hit {
		ctx.Out <- unit.FX{
			Name: fxGhostHit, UnitID: residualID, Kind: KindResidual,
			X: x, Y: y, Amount: m.amount, VX: g.remain, VY: float64(ctx.ID), Slot: m.bin,
		}
		if g.down && g.enemy != 0 {
			ctx.Out <- unit.FX{
				Name: fxGhostDown, UnitID: m.owner, Kind: KindNoncausal,
				X: x, Y: y, Amount: m.amount, VX: float64(g.enemy), VY: g.max, Slot: m.bin,
			}
			ctx.Out <- unit.Despawn{UnitID: g.enemy}
		}
	} else {
		kind := 0.0
		if unitBlock {
			kind = 1
		}
		ctx.Out <- unit.FX{
			Name: fxBlock, UnitID: ctx.ID, Kind: KindLock,
			X: x, Y: y, Amount: 0, VX: float64(blocker), VY: kind, Slot: m.bin,
		}
	}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

func findResidual(s unit.Sense, owner uint64) *unit.Snapshot {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind == KindResidual && o.OwnerID == owner {
			return o
		}
	}
	return nil
}

// steer 把当前速度朝目标拧，每拍不超过 turn*dt，速率保持 speed。
func steer(vx, vy, dx, dy, turn, speed, dt float64) (float64, float64) {
	if speed < 0 {
		speed = 0
	}
	hx, hy := vx, vy
	if math.Hypot(hx, hy) < 1e-9 {
		hx, hy = dx, dy
	}
	if math.Hypot(hx, hy) < 1e-9 {
		return speed, 0
	}
	hAng := math.Atan2(hy, hx)
	if math.Hypot(dx, dy) < 1e-9 {
		return math.Cos(hAng) * speed, math.Sin(hAng) * speed
	}
	delta := wrapPi(math.Atan2(dy, dx) - hAng)
	max := turn * dt
	if max < 0 {
		max = 0
	}
	if delta > max {
		delta = max
	} else if delta < -max {
		delta = -max
	}
	ang := hAng + delta
	return math.Cos(ang) * speed, math.Sin(ang) * speed
}

func wrapPi(a float64) float64 {
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	for a < -math.Pi {
		a += 2 * math.Pi
	}
	return a
}
