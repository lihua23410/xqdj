package unit

type Snapshot struct {
	ID          uint64   `json:"id"`
	Kind        string   `json:"kind"`
	Role        string   `json:"role"`
	X           float64  `json:"x"`
	Y           float64  `json:"y"`
	VX          float64  `json:"vx"`
	VY          float64  `json:"vy"`
	Radius      float64  `json:"radius"`
	HP          float64  `json:"hp"`
	MaxHP       float64  `json:"maxHp"`
	Vision      float64  `json:"vision"`
	OwnerID     uint64   `json:"ownerId"`
	Slot        int      `json:"slot"`
	Semi        bool     `json:"semi"`
	FaceX       float64  `json:"faceX"`
	FaceY       float64  `json:"faceY"`
	PassWalls   bool     `json:"passWalls"`
	Mortal      bool     `json:"mortal,omitempty"`
	BreakWalls  bool     `json:"breakWalls,omitempty"`
	ArcSpan     float64  `json:"arcSpan,omitempty"`
	ArcInner    float64  `json:"arcInner,omitempty"`
	Faction     string   `json:"faction,omitempty"`
	Seen        []string `json:"seen,omitempty"`
	Marks       []Mark   `json:"marks,omitempty"`
	AimPriority uint8    `json:"aimPriority,omitempty"`
}

type Sense struct {
	Time   float64
	Self   Snapshot
	Nearby []Snapshot
	Field  Field
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
	Kind WallKind
}

// FactionChanged 阵营刚变时发给该战斗机。撞墙轮换和 MarkFaction 改派系都会发；Stun 拦不住。
type FactionChanged struct {
	Faction string
}

type Cmd any

type SetVelocity struct {
	UnitID uint64
	VX     float64
	VY     float64
}

// SetCruise 改自己巡航 FS 的 BaseSpeed。不改当前速度。快于巡航会被拉回；慢于巡航但仍在动会沿当前方向被推上去。
type SetCruise struct {
	UnitID uint64
	Speed  float64
}

// FSZone 是巡航 FS 分量袋子的区。空袋 Σ=0、Π=1。
type FSZone int

const (
	FSZoneDp FSZone = iota + 1
	FSZoneDt
	FSZoneFp
	FSZoneFt
	FSZoneM
)

// AddFS 给目标加一条瞬时 FS。Token 由塞入者自 mint；同 token 覆盖。ExpiresAt=0 表示直到 Remove 或单位销毁。
type AddFS struct {
	UnitID    uint64
	DX, DY    float64
	BaseSpeed float64
	OnWall    bool
	ExpiresAt float64
	Token     uint64
}

// RemoveFS 撤一条瞬时 FS。token 不存在则 no-op。
type RemoveFS struct {
	UnitID uint64
	Token  uint64
}

// AddFSComponent 往目标巡航 FS 的袋子里塞一分量。同 token 覆盖。ExpiresAt=0 表示直到 Remove。
type AddFSComponent struct {
	UnitID    uint64
	Zone      FSZone
	Token     uint64
	Value     float64
	ExpiresAt float64
}

// RemoveFSComponent 撤巡航 FS 上的一分量。token 不存在则 no-op。
type RemoveFSComponent struct {
	UnitID uint64
	Token  uint64
}

// SetFSDirection 改巡航 FS 的朝向，不改当前速度。
type SetFSDirection struct {
	UnitID uint64
	VX     float64
	VY     float64
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

// SetHP 直接设置单位的当前血量和最大血量，不触发伤害特效。用于活随从动态初始化血量。
type SetHP struct {
	UnitID uint64
	HP     float64
	MaxHP  float64
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
	Hard    bool // 硬墙：不拆、拆墙弹穿过
	Square  bool // 方端判定
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

// SetAimPriority 改瞄准优先度。From 是改的人，UnitID 是被改的。
// 只能改战斗机或活随从。当前为 0 时只有自己（From == UnitID）能改。
type SetAimPriority struct {
	From   uint64
	UnitID uint64
	Value  uint8
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

// Stand 站定令牌。Hold 时速率为 0 也不往巡航推。虚弱、居合、锁敌用这个；没带令牌则从静止恢复巡航。
type Stand struct {
	UnitID uint64
	Hold   bool
}

// Stun 令牌。Hold 时引擎不发 Sense（自身攻击停在冷却），IncomingDamage / 撞墙仍到。
// Until>0 时到点自动放下；Until=0 维持旧语义，需 Hold=false 才解除。
type Stun struct {
	UnitID uint64
	Hold   bool
	Until  float64
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
