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
