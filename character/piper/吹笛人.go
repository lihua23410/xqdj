// 吹笛人自己不出伤。战斗状态围绕老鼠组织：不停地吹笛召鼠，老鼠死亡时给附近敌人叠瘟疫。
package 吹笛人

import (
	"embed"
	"math"

	"xqdj/internal/unit"
)

const KindPiper = "吹笛人"
const KindRat = "吹笛人老鼠"

const (
	bodyRadius = 18.0
	bodyHP     = 100.0
	cruise     = 165.0
	vision     = 9999.0
	bodyColor  = "#7b5cd6"

	pipeFirst   = 0.5  // 开局第一声笛
	pipeGap     = 2.5  // 之后每隔这么久吹一次
	pipeWind    = 0.4  // 吹笛站定（持 Stand）
	ratCap      = 6    // 场上最多几只老鼠
	ratSpawnGap = 48.0 // 老鼠从球边多远冒出来（落在待命环外，不让它一出生就挤本体）

	deliverR = 40.0 // 叼着赃物的老鼠进这个圈算入洞
	lootHeal = 3.0  // 每入洞一件，本体回这么多血

	plagueKind  = "瘟疫"
	plagueR     = 50.0  // 老鼠倒下时这一圈内叠瘟疫
	plagueStick = 100.0 // 活老鼠贴着：球面外再这么远仍算跟住。老鼠停在 16 外，敌人还会跑
	plagueHold  = 5.0   // 活老鼠要连续贴这么久才叠一层
	plagueTick  = 3.0   // 有瘟疫的单位每隔这么久跳一次
	plagueMul   = 1     // 每跳扣 层数 × 这个
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindPiper, assets)
	p.Register(unit.Spec{
		Kind: KindPiper, Role: unit.RoleFighter, Radius: bodyRadius, MaxHP: bodyHP,
		Speed: cruise, Vision: vision, Fighter: true,
		Look: unit.Look{Color: bodyColor, Glow: true, FX: []string{"piper"}},
	}, func(unit.SpawnInfo) unit.Actor { return &吹笛人{} })
	// 老鼠是活随从：有血、能被打死（对手打死它就能把东西抢回来），但不计入胜负。
	p.Register(unit.Spec{
		Kind: KindRat, Role: unit.RoleMinion, Radius: ratRadius, MaxHP: ratHP,
		Speed: ratChase, Vision: vision, Mortal: true,
		Look: unit.Look{Color: ratColor, Ghost: 200, FX: []string{"rat"}},
	}, func(info unit.SpawnInfo) unit.Actor { return newRat(info) })
}

// ratView 是上一拍一只老鼠的样子。mark 非空表示它叼着赃物，mark 的值就是被偷物的 Kind。
type ratView struct {
	mark string
	x, y float64
}

type 吹笛人 struct {
	booted    bool
	slot      int
	nextPipe  float64
	standTil  float64
	enemyID   uint64
	enemySlot int
	hasEnemy  bool
	prev      map[uint64]ratView
	plagueAt  map[uint64]float64
	nearAt    map[uint64]float64
}

func (a *吹笛人) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	a.onSense(ctx, s)
}

func (a *吹笛人) onSense(ctx unit.Context, s unit.Sense) {
	a.slot = s.Self.Slot
	if !a.booted {
		a.booted = true
		a.nextPipe = s.Time + pipeFirst
		a.prev = map[uint64]ratView{}
		a.plagueAt = map[uint64]float64{}
		a.nearAt = map[uint64]float64{}
	}
	if a.standTil > 0 && s.Time+1e-9 >= a.standTil {
		a.standTil = 0
		ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}
	}
	a.noteEnemy(s)
	a.pulseNear(ctx, s)
	a.tickPlague(ctx, s)
	rats := a.collect(ctx, s)
	if s.Time+1e-9 >= a.nextPipe && rats < ratCap {
		a.nextPipe = s.Time + pipeGap
		a.pipe(ctx, s, rats)
	}
}

// collect 结算自己的老鼠：跑回身边又叼着赃物的算入洞，上一拍还在、这一拍没了的算被打死。
// 返回这一拍场上的老鼠数。
func (a *吹笛人) collect(ctx unit.Context, s unit.Sense) int {
	now := make(map[uint64]ratView, ratCap)
	delivered := map[uint64]struct{}{}
	n := 0
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind != KindRat || o.OwnerID != ctx.ID {
			continue
		}
		n++
		v := ratView{x: o.X, y: o.Y}
		if len(o.Marks) > 0 {
			v.mark = o.Marks[0].Kind
		}
		if v.mark != "" && math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) <= deliverR {
			ctx.Out <- unit.Despawn{UnitID: o.ID}
			// 赃物进屋，本体回一口血（引擎自己封顶，也不会触发 hit-stop）。
			ctx.Out <- unit.Heal{UnitID: ctx.ID, Amount: lootHeal}
			ctx.Out <- unit.FX{Name: "stash", Kind: ctx.Kind, X: o.X, Y: o.Y, Slot: a.slot}
			delivered[o.ID] = struct{}{}
			continue
		}
		now[o.ID] = v
	}
	for id, v := range a.prev {
		if _, alive := now[id]; alive {
			continue
		}
		if _, ok := delivered[id]; ok {
			continue
		}
		if v.mark != "" {
			a.spill(ctx, v)
		}
		a.spreadPlague(ctx, s, v.x, v.y)
	}
	a.prev = now
	return n
}

// pulseNear 活老鼠贴着敌人并连续待满 plagueHold 才叠一层。中途离开计时清零。
// 「贴着」按球面外 plagueStick 算：老鼠本来就停在球面外 16，敌人一跑还会再拉开一截。
func (a *吹笛人) pulseNear(ctx unit.Context, s unit.Sense) {
	if a.nearAt == nil {
		a.nearAt = map[uint64]float64{}
	}
	var rats []unit.Snapshot
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind == KindRat && o.OwnerID == ctx.ID && len(o.Marks) == 0 {
			rats = append(rats, *o)
		}
	}
	seen := map[uint64]struct{}{}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		close := false
		for _, rat := range rats {
			if stuckTo(rat, *o) {
				close = true
				break
			}
		}
		if !close {
			delete(a.nearAt, o.ID)
			continue
		}
		seen[o.ID] = struct{}{}
		since, ok := a.nearAt[o.ID]
		if !ok {
			a.nearAt[o.ID] = s.Time
			continue
		}
		if s.Time+1e-9 < since+plagueHold {
			continue
		}
		ctx.Out <- unit.StackMark{UnitID: o.ID, Kind: plagueKind, Delta: 1}
		a.hitPlague(ctx, o, markStacks(*o, plagueKind)+1)
		a.nearAt[o.ID] = s.Time
		if a.plagueAt == nil {
			a.plagueAt = map[uint64]float64{}
		}
		a.plagueAt[o.ID] = s.Time + plagueTick
	}
	for id := range a.nearAt {
		if _, ok := seen[id]; !ok {
			delete(a.nearAt, id)
		}
	}
}

// spill 把被打死的老鼠嘴里那件东西在它倒下的地方还给原主人。对手已经不在了就不还。
func (a *吹笛人) spill(ctx unit.Context, v ratView) {
	if !a.hasEnemy {
		return
	}
	ctx.Out <- unit.Spawn{
		Kind: v.mark, X: v.x, Y: v.y,
		OwnerID: a.enemyID, Slot: a.enemySlot,
	}
	ctx.Out <- unit.FX{Name: "spill", Kind: ctx.Kind, X: v.x, Y: v.y, Slot: a.slot}
}

func (a *吹笛人) noteEnemy(s unit.Sense) {
	a.hasEnemy = false
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role == unit.RoleFighter && o.Slot != s.Self.Slot {
			a.enemyID, a.enemySlot, a.hasEnemy = o.ID, o.Slot, true
			return
		}
	}
}

// pipe 吹笛：站定一小会儿，从球边的环位上放一只老鼠出去。落点按当前鼠数取，不带随机，
// 免得新老鼠一出来就挤在本体身上。
func (a *吹笛人) pipe(ctx unit.Context, s unit.Sense, rats int) {
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID}
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
	a.standTil = s.Time + pipeWind
	ctx.Out <- unit.FX{Name: "pipe", Kind: ctx.Kind, UnitID: ctx.ID, X: s.Self.X, Y: s.Self.Y, Slot: a.slot}
	ang := float64(rats%6) * math.Pi / 3
	ux, uy := math.Cos(ang), math.Sin(ang)
	ctx.Out <- unit.Spawn{
		Kind: KindRat, OwnerID: ctx.ID, Slot: a.slot,
		X: s.Self.X + ux*ratSpawnGap, Y: s.Self.Y + uy*ratSpawnGap,
		VX: ux * ratChase, VY: uy * ratChase,
	}
}

// spreadPlague 老鼠倒下：给倒下处一小圈内的敌方单位叠一层瘟疫，并立刻按叠完后的层数跳一口。
func (a *吹笛人) spreadPlague(ctx unit.Context, s unit.Sense, x, y float64) {
	if a.plagueAt == nil {
		a.plagueAt = map[uint64]float64{}
	}
	ctx.Out <- unit.FX{Name: "plague", Kind: ctx.Kind, X: x, Y: y, Slot: a.slot}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		if math.Hypot(o.X-x, o.Y-y) > plagueR {
			continue
		}
		ctx.Out <- unit.StackMark{UnitID: o.ID, Kind: plagueKind, Delta: 1}
		n := markStacks(*o, plagueKind) + 1
		a.hitPlague(ctx, o, n)
		a.plagueAt[o.ID] = s.Time + plagueTick
	}
}

func plagueAmount(stacks int) float64 {
	if stacks <= 0 {
		return 0
	}
	return float64(stacks * plagueMul)
}

func (a *吹笛人) hitPlague(ctx unit.Context, o *unit.Snapshot, stacks int) {
	amt := plagueAmount(stacks)
	if amt <= 0 {
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: amt}
	ctx.Out <- unit.FX{Name: "plague-tick", Kind: ctx.Kind, UnitID: o.ID, X: o.X, Y: o.Y, Amount: amt, Slot: a.slot}
}

// tickPlague 有瘟疫的敌方单位每 3 秒扣 层数 的血。刚叠上那一拍已经跳过一口，从那一拍再计时。
func (a *吹笛人) tickPlague(ctx unit.Context, s unit.Sense) {
	if a.plagueAt == nil {
		a.plagueAt = map[uint64]float64{}
	}
	seen := map[uint64]struct{}{}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		n := markStacks(*o, plagueKind)
		if n <= 0 {
			delete(a.plagueAt, o.ID)
			continue
		}
		seen[o.ID] = struct{}{}
		next, ok := a.plagueAt[o.ID]
		if !ok {
			a.plagueAt[o.ID] = s.Time + plagueTick
			continue
		}
		if s.Time+1e-9 < next {
			continue
		}
		a.hitPlague(ctx, o, n)
		a.plagueAt[o.ID] = s.Time + plagueTick
	}
	for id := range a.plagueAt {
		if _, ok := seen[id]; !ok {
			delete(a.plagueAt, id)
		}
	}
}

// stuckTo 老鼠是否还贴着这个敌人。中心距 ≤ 双方半径 + 追赶余量，不拿死 40 去卡移动中的球。
func stuckTo(rat, o unit.Snapshot) bool {
	return math.Hypot(rat.X-o.X, rat.Y-o.Y) <= o.Radius+rat.Radius+plagueStick
}

func markStacks(u unit.Snapshot, kind string) int {
	for _, m := range u.Marks {
		if m.Kind == kind {
			return m.Stacks
		}
	}
	return 0
}
