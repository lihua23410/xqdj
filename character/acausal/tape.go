package 非因果

import "sync"

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

// Tape 是探针交给真局的那一份记录。
type Tape struct {
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

var (
	tapeMu sync.Mutex
	live   *Tape
)

// Install 换上这一场的探针记录。nil 表示这场没有非因果。
func Install(t *Tape) {
	if t != nil && t.Bins == nil {
		t.Bins = Magnitudes(t.Drops)
	}
	tapeMu.Lock()
	live = t
	tapeMu.Unlock()
}

// Current 复制当前记录。没有则是 nil。
func Current() *Tape {
	tapeMu.Lock()
	defer tapeMu.Unlock()
	if live == nil {
		return nil
	}
	cp := *live
	return &cp
}
