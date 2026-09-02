package unit

type Snapshot struct {
	ID      uint64  `json:"id"`
	Kind    string  `json:"kind"`
	Role    string  `json:"role"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	VX      float64 `json:"vx"`
	VY      float64 `json:"vy"`
	Radius  float64 `json:"radius"`
	HP      float64 `json:"hp"`
	MaxHP   float64 `json:"maxHp"`
	Vision  float64 `json:"vision"`
	OwnerID uint64  `json:"ownerId"`
	Slot    int     `json:"slot"`
}

type Sense struct {
	Time   float64
	Self   Snapshot
	Nearby []Snapshot
}

type Collision struct {
	Time  float64
	Other Snapshot
	NX    float64
	NY    float64
}

type WallHit struct {
	Time float64
	NX   float64
	NY   float64
}

type Cmd any

type SetVelocity struct {
	UnitID uint64
	VX     float64
	VY     float64
}

type Damage struct {
	From   uint64
	To     uint64
	Amount float64
}

type Spawn struct {
	Kind    string
	X       float64
	Y       float64
	VX      float64
	VY      float64
	OwnerID uint64
	Slot    int
}

type Despawn struct {
	UnitID uint64
}

type FX struct {
	Name   string  `json:"name"`
	UnitID uint64  `json:"unitId"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	VX     float64 `json:"vx"`
	VY     float64 `json:"vy"`
	Slot   int     `json:"slot"`
	Amount float64 `json:"amount"`
}

type Context struct {
	ID   uint64
	Kind string
	Out  chan<- Cmd
}

type Event any

type Actor interface {
	Handle(ctx Context, ev Event)
}

type SpawnInfo struct {
	OwnerID uint64
	Slot    int
}
