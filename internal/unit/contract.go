package unit

type Snapshot struct {
	ID         uint64   `json:"id"`
	Kind       string   `json:"kind"`
	Role       string   `json:"role"`
	X          float64  `json:"x"`
	Y          float64  `json:"y"`
	VX         float64  `json:"vx"`
	VY         float64  `json:"vy"`
	Radius     float64  `json:"radius"`
	HP         float64  `json:"hp"`
	MaxHP      float64  `json:"maxHp"`
	Vision     float64  `json:"vision"`
	OwnerID    uint64   `json:"ownerId"`
	Slot       int      `json:"slot"`
	Semi       bool     `json:"semi"`
	FaceX      float64  `json:"faceX"`
	FaceY      float64  `json:"faceY"`
	PassWalls  bool     `json:"passWalls"`
	Mortal     bool     `json:"mortal,omitempty"`
	BreakWalls bool     `json:"breakWalls,omitempty"`
	ArcSpan    float64  `json:"arcSpan,omitempty"`
	ArcInner   float64  `json:"arcInner,omitempty"`
	Faction    string   `json:"faction,omitempty"`
	Seen       []string `json:"seen,omitempty"`
	Marks      []Mark   `json:"marks,omitempty"`
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

// SetCruise 改自己的巡航。不改当前速度。快于巡航会被拉回；慢于巡航但仍在动会沿当前方向被推上去。
type SetCruise struct {
	UnitID uint64
	Speed  float64
}

// SetVision 改自己的感知半径。视野为 0 则看不见场上其它单位（自己的随从仍能进感知）。
type SetVision struct {
	UnitID uint64
	Vision float64
}

// SetArcSpan 改自己这发扇环弹的张角（弧度）。只有 Spec.Attach 的单位吃。
type SetArcSpan struct {
	UnitID uint64
	Span   float64
}

// SetRadius 改自己的碰撞半径。花冠长大用这个。
type SetRadius struct {
	UnitID uint64
	Radius float64
}

type Damage struct {
	From      uint64
	To        uint64
	Amount    float64
	MarkKind  string
	MarkDelta int
	MarkIcon  string
}

// IncomingDamage 引擎准备扣血。战斗机必须回 ConfirmDamage 才会真正掉 HP；回 BlockDamage 则整包取消。
type IncomingDamage struct {
	Token  uint64
	From   uint64
	Amount float64
	Time   float64
	Speed  float64
}

type ConfirmDamage struct {
	Token  uint64
	UnitID uint64
	Amount float64
}

type BlockDamage struct {
	Token  uint64
	UnitID uint64
}

// GuardBreak 壳（Spec.Shell）被物理撞碎时通知主人。DespawnOwned 摘壳不会发这个。
type GuardBreak struct {
	Time float64
	From uint64
}

func ConfirmHit(ctx Context, d IncomingDamage) {
	ctx.Out <- ConfirmDamage{Token: d.Token, UnitID: ctx.ID, Amount: d.Amount}
}

func BlockHit(ctx Context, d IncomingDamage) {
	ctx.Out <- BlockDamage{Token: d.Token, UnitID: ctx.ID}
}

func AcceptHit(ctx Context, ev Event) bool {
	d, ok := ev.(IncomingDamage)
	if !ok {
		return false
	}
	ConfirmHit(ctx, d)
	return true
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

type DespawnOwned struct {
	OwnerID uint64
	Kind    string
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

// Pass 令牌。Hold 时与其他单位相撞不改双方速度；墙和胶囊墙仍弹。Hold=false 放下。
type Pass struct {
	UnitID uint64
	Hold   bool
}

// NoFrameFreeze 令牌。Hold 时该单位（及其随从）造成的伤害不停帧。Hold=false 放下。
type NoFrameFreeze struct {
	UnitID uint64
	Hold   bool
}

// Stun 令牌。Hold 时引擎不发 Sense（自身攻击停在冷却），IncomingDamage / 撞墙仍到。
type Stun struct {
	UnitID uint64
	Hold   bool
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
