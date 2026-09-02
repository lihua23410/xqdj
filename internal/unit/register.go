package unit

type Spec struct {
	Kind    string
	Role    string
	Radius  float64
	MaxHP   float64
	Speed   float64
	Vision  float64
	Fighter bool
}

type factory func(SpawnInfo) Actor

var (
	specs     = map[string]Spec{}
	factories = map[string]factory{}
	fighters  []string
)

func Register(s Spec, fn func(SpawnInfo) Actor) {
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
