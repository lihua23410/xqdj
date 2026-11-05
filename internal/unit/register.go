package unit

type Spec struct {
	Kind           string
	Role           string
	Radius         float64
	MaxHP          float64
	Speed          float64
	Vision         float64
	Fighter        bool
	Semi           bool
	RequiresOwner  bool    // 处理出生指令时，主人必须仍在场
	PassWalls      bool    // 不撞墙。出生即持有；也可用 PassWalls 令牌挂摘
	PassInnerWalls bool    // 只穿场内墙，场边仍挡住
	BodiesOnly     bool    // 只与战斗机和活随从碰撞，不接触其他攻击单位
	Mortal         bool    // 活随从：吃 Damage / Heal，不计入胜负
	BreakWalls     bool    // 击中胶囊墙则该截提前消失并穿过；硬墙穿过不拆；场边仍撞
	StartHP        float64 // 0 表示开局满血（MaxHP）
	AimPriority    uint8   // 0 = 未写：战斗机 15、活随从 60、其余 0
	Nonsolid       bool    // 不实心：不碰、不进 Hittable。战斗机仍进快照
	VisionBlock    bool    // 挡视线：带标签的实体挡索敌视线（Seek 跳过被压住连线段的目标），不进感知过滤、不挡物理
	Cruise         bool    // 吃巡航带子。战斗机默认有；活随从要写
	Shell          bool    // 贴在主人身上的环，撞到东西会碎并通知主人
	Attach         bool    // 每帧贴主人；不挡伤、不碎。不撞墙。只和敌方战斗机、活随从做 CCD
	AttachFriends  bool    // 扇环也接触同边活体和主人；实际可伤的搭档由攻击窗口限定
	Pass           bool    // 出生即持 Pass；用于持续穿人的攻击判定
	// Hidden 的实体不进别人的感知，也不进页面：它只是场上的一本账。
	// 老鼠叼不走、瘟疫找不到、AOE 也够不着——因为它们根本看不见它。
	Hidden   bool
	ArcSpan  float64 // 扇环张角（弧度）。0 = 不是扇环。2π = 整圈细环
	ArcInner float64 // 扇环内径。外径用 Radius。角色自己填
	Look     Look

	// 指定 Kind 的直属单位在场时，拒绝一切速度与瞬时推动；位置修正和瞬移照常。
	ZeroVelocityWhileOwned string
}

type factory func(SpawnInfo) Actor

var (
	specs     = map[string]Spec{}
	factories = map[string]factory{}
	fighters  []string
)

func Register(s Spec, fn func(SpawnInfo) Actor) {
	if s.Kind == "" {
		panic("unit: empty kind")
	}
	if _, ok := specs[s.Kind]; ok {
		panic("unit: duplicate kind " + s.Kind)
	}
	specs[s.Kind] = s
	factories[s.Kind] = fn
	if s.Fighter {
		fighters = append(fighters, s.Kind)
	}
}

func NewActor(kind string, info SpawnInfo) Actor {
	fn, ok := factories[kind]
	if !ok {
		return nil
	}
	return fn(info)
}

func Lookup(kind string) (Spec, bool) {
	s, ok := specs[kind]
	return s, ok
}

func FighterKinds() []string {
	out := make([]string, len(fighters))
	copy(out, fighters)
	return out
}

func IsFighterKind(kind string) bool {
	s, ok := specs[kind]
	return ok && s.Fighter
}
