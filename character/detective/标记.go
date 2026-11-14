package 探长

import "xqdj/internal/unit"

const (
	modeScene = iota
	modeCharge
	modeForgery
)

func registerMarker(p *unit.Pack, kind string, radius float64, color, fx string, mode int) {
	p.Register(unit.Spec{
		Kind:    kind,
		Role:    unit.RoleHelper,
		Radius:  radius,
		MaxHP:   1,
		Speed:   0,
		Vision:  radius + 48,
		Fighter: false,
		Look:    unit.Look{Color: color, FX: []string{fx}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &标记{owner: info.OwnerID, slot: info.Slot, mode: mode}
	})
}

// 标记不挡路。敌方战斗机或活随从的圆心踏进半径，算碰到一次；离开再进来再记。
type 标记 struct {
	owner  uint64
	slot   int
	mode   int
	inside map[uint64]struct{}
	dead   bool
}

func (m *标记) Handle(ctx unit.Context, ev unit.Event) {
	if m.dead {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if m.inside == nil {
		m.inside = map[uint64]struct{}{}
	}
	now := map[uint64]struct{}{}
	reach := s.Self.Radius
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !body(o) || o.Slot == m.slot {
			continue
		}
		dx, dy := o.X-s.Self.X, o.Y-s.Self.Y
		if dx*dx+dy*dy > (reach+o.Radius)*(reach+o.Radius) {
			continue
		}
		now[o.ID] = struct{}{}
		if _, was := m.inside[o.ID]; was {
			continue
		}
		b := bookOf(ctx.State, m.owner)
		b.addSuspicion(o.ID)
		if m.mode == modeScene {
			continue
		}
		m.dead = true
		m.inside = now
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	m.inside = now
}
