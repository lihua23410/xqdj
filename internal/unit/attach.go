package unit

import "math"

// Deg 把角度转成 ArcSpan 用的弧度。写在角色包里，例如 ArcSpan: unit.Deg(120)。
func Deg(d float64) float64 {
	return d * math.Pi / 180
}

type AttachState struct {
	Armed   bool
	ReadyAt float64
}

func HasOwned(s Sense, owner uint64, kind string) bool {
	for i := range s.Nearby {
		if s.Nearby[i].OwnerID == owner && s.Nearby[i].Kind == kind {
			return true
		}
	}
	return false
}

func EnemyFighter(e Collision, slot int) bool {
	return e.Other.Role == RoleFighter && e.Other.Slot != slot
}

// Hittable：敌方战斗机或敌方活随从。索敌用 Aimable / Seek；出伤用这个。不实心的不算。
func Hittable(o Snapshot, slot int) bool {
	if o.Slot == slot || o.Nonsolid {
		return false
	}
	return o.Role == RoleFighter || o.Mortal
}

func EnemyTarget(e Collision, slot int) bool {
	return Hittable(e.Other, slot)
}

const (
	DefaultFighterAim uint8 = 15
	DefaultMortalAim  uint8 = 60
)

// DefaultAimPriority：Spec.AimPriority 未写（0）时战斗机 15、活随从 60、其余 0。
func DefaultAimPriority(s Spec) uint8 {
	if s.AimPriority != 0 {
		return s.AimPriority
	}
	if s.Role == RoleFighter {
		return DefaultFighterAim
	}
	if s.Mortal {
		return DefaultMortalAim
	}
	return 0
}

// Aimable：敌方可瞄准（瞄准优先度 1–255）。
func Aimable(o Snapshot, slot int) bool {
	return o.Slot != slot && o.AimPriority > 0
}

// Seek 视野内敌方可瞄准：数字越小越先，相同则当下最近。
func Seek(s Sense) *Snapshot {
	return SeekIf(s, nil)
}

// SeekIf 在 Seek 之上再过滤。extra 为 nil 则不过滤。
func SeekIf(s Sense, extra func(Snapshot) bool) *Snapshot {
	var best *Snapshot
	bestPri := uint8(255)
	bestD2 := math.MaxFloat64
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !Aimable(*o, s.Self.Slot) {
			continue
		}
		if extra != nil && !extra(*o) {
			continue
		}
		dx, dy := o.X-s.Self.X, o.Y-s.Self.Y
		d2 := dx*dx + dy*dy
		if best != nil && (o.AimPriority > bestPri || (o.AimPriority == bestPri && d2 >= bestD2)) {
			continue
		}
		best = o
		bestPri = o.AimPriority
		bestD2 = d2
	}
	return best
}

func SetAim(ctx Context, id uint64, v uint8) {
	ctx.Out <- SetAimPriority{From: ctx.ID, UnitID: id, Value: v}
}

// RearmAttach 主人每帧 Sense 调用。true 时 Spawn 一发 kind。
// 打中后弹会 Despawn；这里等 cd 再挂。
func RearmAttach(s Sense, owner uint64, kind string, cd float64, st *AttachState) bool {
	if HasOwned(s, owner, kind) {
		st.Armed = true
		return false
	}
	if st.Armed {
		st.Armed = false
		st.ReadyAt = s.Time + cd
	}
	if s.Time+1e-9 < st.ReadyAt {
		return false
	}
	return true
}

func SpawnAttach(ctx Context, s Sense, kind string) {
	ctx.Out <- Spawn{
		Kind:    kind,
		X:       s.Self.X,
		Y:       s.Self.Y,
		VX:      s.Self.VX,
		VY:      s.Self.VY,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	}
}
