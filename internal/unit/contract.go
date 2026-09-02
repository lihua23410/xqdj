package unit

type Snapshot struct {
	ID        uint64  `json:"id"`
	Kind      string  `json:"kind"`
	Role      string  `json:"role"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	VX        float64 `json:"vx"`
	VY        float64 `json:"vy"`
	Radius    float64 `json:"radius"`
	HP        float64 `json:"hp"`
	MaxHP     float64 `json:"maxHp"`
	Vision    float64 `json:"vision"`
	OwnerID   uint64  `json:"ownerId"`
	Slot      int     `json:"slot"`
	Semi      bool    `json:"semi"`
	FaceX     float64 `json:"faceX"`
	FaceY     float64 `json:"faceY"`
	PassWalls bool    `json:"passWalls"`
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

type SwapOwned struct {
	UnitID uint64
}

type PlaceWall struct {
	OwnerID uint64
	Slot    int
	Kind    string
	X1, Y1  float64
	X2, Y2  float64
	Radius  float64
	Life    float64
	Amount  float64
}

type FX struct {
	Name   string  `json:"name"`
	UnitID uint64  `json:"unitId"`
	Kind   string  `json:"kind"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	VX     float64 `json:"vx"`
	VY     float64 `json:"vy"`
	Slot   int     `json:"slot"`
	Amount float64 `json:"amount"`
}

// Force 给目标加加速度，引擎做 v += (AX,AY)*dt。不是改写速度，快的球仍能撞上。
type Force struct {
	UnitID uint64
	AX     float64
	AY     float64
}

type Teleport struct {
	UnitID uint64
	X      float64
	Y      float64
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
