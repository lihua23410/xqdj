package unit

type Spec struct {
	Kind      string
	Role      string
	Radius    float64
	MaxHP     float64
	Speed     float64
	Vision    float64
	Fighter   bool
	Semi      bool
	PassWalls bool
	StartHP   float64 // 0 表示开局满血（MaxHP）
	Shell     bool    // 贴在主人身上的环，撞到东西会碎并通知主人
	Attach    bool    // 每帧贴主人；不挡伤、不碎。不撞墙。只和敌方战斗机做 CCD
	ArcSpan   float64 // 扇环张角（弧度）。0 = 不是扇环。2π = 整圈细环
	ArcInner  float64 // 扇环内径。外径用 Radius。角色自己填
	Look      Look
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
