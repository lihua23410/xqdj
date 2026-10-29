package unit

// Sample 是轨迹上的一个点。
type Sample struct {
	X, Y float64
}

// FighterTrack 是一只敌方战斗机在探针里的轨。
type FighterTrack struct {
	Slot, Seat int
	Kind       string
	MaxHP      float64
	Radius     float64
	Samples    []Sample
}

// MinionTrack 是一只敌方活随从在探针里的轨。Index 是同一主人、同一种类里的出生序号。
type MinionTrack struct {
	OwnerSlot, OwnerSeat int
	Kind                 string
	Index                int
	MaxHP                float64
	Radius               float64
	Samples              []Sample
}

// ProbeTape 是探针交给真局的那一份记录。
type ProbeTape struct {
	DeathTime float64 // < 0 表示探针里没倒下
	SpawnVX   float64
	SpawnVY   float64
	Drops     []float64
	Bins      []float64
	Self      []Sample
	SelfHP    []float64
	Fighters  []FighterTrack
	Minions   []MinionTrack
}

// Clone 返回独立的探针记录；各单位不会共享可写切片。
func (t *ProbeTape) Clone() *ProbeTape {
	if t == nil {
		return nil
	}
	cp := *t
	cp.Drops = append([]float64(nil), t.Drops...)
	cp.Bins = append([]float64(nil), t.Bins...)
	cp.Self = append([]Sample(nil), t.Self...)
	cp.SelfHP = append([]float64(nil), t.SelfHP...)
	cp.Fighters = append([]FighterTrack(nil), t.Fighters...)
	for i := range cp.Fighters {
		cp.Fighters[i].Samples = append([]Sample(nil), t.Fighters[i].Samples...)
	}
	cp.Minions = append([]MinionTrack(nil), t.Minions...)
	for i := range cp.Minions {
		cp.Minions[i].Samples = append([]Sample(nil), t.Minions[i].Samples...)
	}
	return &cp
}
