package 静电魔球

import (
	"encoding/json"
	"math"

	"xqdj/internal/unit"
)

const (
	jumpRange   = 250.0
	arcHalf     = 6.0
	arcWindup   = 0.34
	arcLead     = 0.29
	arcOverrun  = 22.0
	hopRecovery = 0.14
)

var chainDamage = [...]float64{13, 10, 7}

type discharge struct {
	from    unit.Snapshot
	hits    int
	readyAt float64
	warning *warnedArc
}

type warnedArc struct {
	line   arcLine
	effect unit.FX
}

type arcLine struct {
	x, y   float64
	ux, uy float64
	length float64
}

type warningVisual struct {
	Started float64 `json:"started"`
	At      float64 `json:"at"`
	Width   float64 `json:"width"`
}

type impactVisual struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func (a *静电魔球) startChain(ctx unit.Context, s unit.Sense) bool {
	c := &discharge{from: s.Self}
	if !c.plan(ctx, s) {
		return false
	}
	a.charge = 0
	a.syncMark(ctx)
	a.setBoost(ctx, false)
	a.chain = c
	return true
}

func (a *静电魔球) stopChain(ctx unit.Context, cancelled bool) {
	if a.chain == nil {
		return
	}
	if a.chain.warning != nil {
		ctx.Out <- unit.SetThreats{UnitID: ctx.ID}
		if cancelled {
			ctx.Out <- unit.FX{Name: "static-cancel", Kind: ctx.Kind, UnitID: ctx.ID}
		}
	}
	a.chain = nil
}

func (a *静电魔球) stepChain(ctx unit.Context, s unit.Sense) {
	c := a.chain
	if s.Time+timeEps < c.readyAt {
		if c.warning != nil {
			ctx.Out <- c.warning.effect
		}
		return
	}
	if c.warning == nil {
		if !c.plan(ctx, s) {
			a.stopChain(ctx, false)
		}
		return
	}
	// 失去感知太久就熄灭，恢复行动时不会突然补放已经过期的预警。
	if s.Time > c.readyAt+0.05 {
		a.stopChain(ctx, true)
		return
	}
	hit := c.strike(ctx, s)
	if hit == nil || c.hits+1 == len(chainDamage) {
		a.stopChain(ctx, false)
		return
	}
	c.hits++
	c.from = *hit
	c.readyAt = s.Time + hopRecovery
	ctx.Out <- unit.SetThreats{UnitID: ctx.ID}
	c.warning = nil
}

func (c *discharge) plan(ctx unit.Context, s unit.Sense) bool {
	from := c.from
	if from.ID == s.Self.ID {
		from = s.Self
	}
	target := pickHop(from, s)
	if target == nil {
		if from.ID == s.Self.ID || dist(from, s.Self) > jumpRange {
			return false
		}
		// 单挑时可经自己中继。回流不出伤，下一次攻击仍需重新预警。
		line := makeArc(from.X, from.Y, s.Self.X, s.Self.Y, 0)
		line = line.clipped(s)
		relay := line.effect(ctx, s, "static-relay")
		ctx.Out <- relay
		if line.length+timeEps < dist(from, s.Self) {
			return false
		}
		c.from = s.Self
		c.readyAt = s.Time + hopRecovery
		return true
	}

	aimX := target.X + target.VX*arcLead
	aimY := target.Y + target.VY*arcLead
	line := makeArc(from.X, from.Y, aimX, aimY, arcOverrun).clipped(s)
	if line.length <= timeEps {
		return false
	}
	c.readyAt = s.Time + arcWindup
	warn := line.effect(ctx, s, "static-warn")
	warn.Data, _ = json.Marshal(warningVisual{Started: s.Time, At: c.readyAt, Width: arcHalf * 2})
	c.warning = &warnedArc{line: line, effect: warn}
	ctx.Out <- warn
	if s.ForecastThreats {
		ctx.Out <- unit.SetThreats{UnitID: ctx.ID, Areas: []unit.Threat{{
			Shape: unit.ThreatRay, From: ctx.ID, Slot: s.Self.Slot,
			X: warn.X, Y: warn.Y, X2: warn.VX, Y2: warn.VY, Radius: arcHalf,
			ObservedAt: s.Time, ActiveAt: c.readyAt, Until: c.readyAt,
		}}}
	}
	return true
}

func (c *discharge) strike(ctx unit.Context, s unit.Sense) *unit.Snapshot {
	// 保留原来预警的两端。新出现的墙只能缩短电弧。
	line := c.warning.line.clipped(s)
	hit := line.firstContact(s, c.from.ID)
	fx := line.effect(ctx, s, "static-bolt")
	if hit != nil {
		fx.Amount = chainDamage[c.hits]
		fx.Data, _ = json.Marshal(impactVisual{X: hit.X, Y: hit.Y})
		ctx.Out <- unit.Damage{From: ctx.ID, To: hit.ID, Amount: fx.Amount, NoFreeze: c.hits > 0}
	}
	ctx.Out <- fx
	return hit
}

// 优先选射程内的战斗机，再选活随从。这一步只定预警线的方向。
func pickHop(from unit.Snapshot, s unit.Sense) *unit.Snapshot {
	var best *unit.Snapshot
	bestDistance := math.Inf(1)
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.ID == from.ID || !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		d := dist(from, *o)
		if d > jumpRange {
			continue
		}
		if best != nil && (o.Role == unit.RoleFighter) != (best.Role == unit.RoleFighter) {
			if o.Role != unit.RoleFighter {
				continue
			}
		} else if d >= bestDistance {
			continue
		}
		best, bestDistance = o, d
	}
	return best
}

func makeArc(x, y, tx, ty, overrun float64) arcLine {
	dx, dy := tx-x, ty-y
	d := math.Hypot(dx, dy)
	if d <= timeEps {
		return arcLine{x: x, y: y, ux: 1, length: overrun}
	}
	return arcLine{x: x, y: y, ux: dx / d, uy: dy / d, length: math.Min(jumpRange, d+overrun)}
}

func (l arcLine) effect(ctx unit.Context, s unit.Sense, name string) unit.FX {
	return unit.FX{Name: name, Kind: ctx.Kind, UnitID: ctx.ID, Slot: s.Self.Slot,
		X: l.x, Y: l.y, VX: l.x + l.ux*l.length, VY: l.y + l.uy*l.length}
}

func (l arcLine) firstContact(s unit.Sense, fromID uint64) *unit.Snapshot {
	if l.length <= timeEps {
		return nil
	}
	var first *unit.Snapshot
	nearest := math.Inf(1)
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.ID == fromID || !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		dx, dy := o.X-l.x, o.Y-l.y
		along := dx*l.ux + dy*l.uy
		across := math.Abs(dx*l.uy - dy*l.ux)
		if along < 0 || along > l.length || across > o.Radius+arcHalf || along >= nearest {
			continue
		}
		if len(s.Walls) > 0 && across > timeEps {
			// 球体虽擦到走廊边缘，也不能从墙的另一侧接触电弧。
			_, _, hold := unit.Detour(l.x+l.ux*along, l.y+l.uy*along, 0, o.X, o.Y, s.Walls, unit.DetourHold{}, s.Time)
			if hold.End != 0 {
				continue
			}
		}
		first, nearest = o, along
	}
	return first
}

func (l arcLine) clipped(s unit.Sense) arcLine {
	if l.length <= timeEps {
		return l
	}
	// 每 4 单位取样一次找场边，再细分最后一小段。
	prev := 0.0
	for d := math.Min(4, l.length); ; d = math.Min(d+4, l.length) {
		if !s.Field.Walkable(l.x+l.ux*d, l.y+l.uy*d, arcHalf) {
			lo, hi := prev, d
			for i := 0; i < 8; i++ {
				mid := (lo + hi) / 2
				if s.Field.Walkable(l.x+l.ux*mid, l.y+l.uy*mid, arcHalf) {
					lo = mid
				} else {
					hi = mid
				}
			}
			l.length = lo
			break
		}
		if d >= l.length {
			break
		}
		prev = d
	}
	// 动态硬墙只在 Walls 里；复用绕墙判定对整段做胶囊检查。
	blocked := func(distance float64) bool {
		_, _, hold := unit.Detour(l.x, l.y, arcHalf, l.x+l.ux*distance, l.y+l.uy*distance, s.Walls, unit.DetourHold{}, s.Time)
		return hold.End != 0
	}
	if len(s.Walls) > 0 && blocked(l.length) {
		lo, hi := 0.0, l.length
		for i := 0; i < 12; i++ {
			mid := (lo + hi) / 2
			if blocked(mid) {
				hi = mid
			} else {
				lo = mid
			}
		}
		l.length = lo
	}
	return l
}
