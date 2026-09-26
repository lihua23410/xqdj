// 吹笛人自己不出伤。战斗状态围绕老鼠组织：不停地吹笛召鼠，攒下的赃物拿去指挥鼠群扑敌。
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
	lootCap  = 3    // 赃物最多给指挥加这么多伤害，指挥时被吃掉

	cmdFirst  = 3.0 // 第一声指挥
	cmdCD     = 8.0 // 指挥冷却
	cmdWindow = 2.5 // 指挥持续多久
	cmdWind   = 0.4 // 挥棒站定
	cmdAim    = 61  // 指挥标记：老鼠的瞄准优先度 = cmdAim + 赃物加成
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
	nextCmd   float64
	cmdUntil  float64
	cmdBonus  int
	standTil  float64
	loot      int
	enemyID   uint64
	enemySlot int
	hasEnemy  bool
	prev      map[uint64]ratView
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
		a.nextCmd = s.Time + cmdFirst
		a.prev = map[uint64]ratView{}
	}
	if a.standTil > 0 && s.Time+1e-9 >= a.standTil {
		a.standTil = 0
		ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}
	}
	a.noteEnemy(s)
	rats := a.collect(ctx, s)
	if s.Time+1e-9 >= a.nextPipe && rats < ratCap {
		a.nextPipe = s.Time + pipeGap
		a.pipe(ctx, s, rats)
	}
	if a.cmdUntil > 0 && s.Time+1e-9 >= a.cmdUntil {
		a.cmdUntil = 0
	}
	if a.cmdUntil == 0 && s.Time+1e-9 >= a.nextCmd && a.hasEnemy && rats > 0 {
		a.command(ctx, s)
	}
	a.order(ctx, s)
	a.report(ctx, s)
}

// collect 结算自己的老鼠：跑回身边又叼着赃物的算入洞，上一拍还在、这一拍没了的算被打死。
// 返回这一拍场上的老鼠数。
func (a *吹笛人) collect(ctx unit.Context, s unit.Sense) int {
	now := make(map[uint64]ratView, ratCap)
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
			a.loot++
			// 赃物进屋，本体回一口血（引擎自己封顶，也不会触发 hit-stop）。
			ctx.Out <- unit.Heal{UnitID: ctx.ID, Amount: lootHeal}
			ctx.Out <- unit.FX{Name: "stash", Kind: ctx.Kind, X: o.X, Y: o.Y, Slot: a.slot}
			now[o.ID] = ratView{} // 已入洞，别在下面当成战死
			continue
		}
		now[o.ID] = v
	}
	for id, v := range a.prev {
		if _, alive := now[id]; alive || v.mark == "" {
			continue
		}
		a.spill(ctx, v)
	}
	a.prev = now
	return n
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

// command 指挥：把手上的赃物吃掉，让全场老鼠扑上去咬。加成写在瞄准优先度里带给老鼠。
func (a *吹笛人) command(ctx unit.Context, s unit.Sense) {
	a.nextCmd = s.Time + cmdCD
	a.cmdUntil = s.Time + cmdWindow
	a.cmdBonus = a.loot
	if a.cmdBonus > lootCap {
		a.cmdBonus = lootCap
	}
	a.loot = 0
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID}
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
	a.standTil = s.Time + cmdWind
	ctx.Out <- unit.FX{
		Name: "command", Kind: ctx.Kind, UnitID: ctx.ID,
		X: s.Self.X, Y: s.Self.Y, Amount: float64(a.cmdBonus), Slot: a.slot,
	}
}

// order 每帧对齐一次指挥标记：窗口内让所有老鼠待命攻击，窗口一过放它们回去偷东西。
// 指挥窗口里新生的老鼠下一拍补上标记。
func (a *吹笛人) order(ctx unit.Context, s unit.Sense) {
	want := uint8(0)
	if a.cmdUntil > 0 {
		want = uint8(cmdAim + a.cmdBonus)
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind != KindRat || o.OwnerID != ctx.ID {
			continue
		}
		switch {
		case want > 0 && o.AimPriority != want:
			unit.SetAim(ctx, o.ID, want)
		case want == 0 && o.AimPriority >= cmdAim:
			unit.SetAim(ctx, o.ID, unit.DefaultMortalAim)
		}
	}
}

// report 每帧报一次赃物数，前端靠它画鼠洞进度。
func (a *吹笛人) report(ctx unit.Context, s unit.Sense) {
	ctx.Out <- unit.FX{
		Name: "loot", Kind: ctx.Kind, UnitID: ctx.ID,
		X: s.Self.X, Y: s.Self.Y, Amount: float64(a.loot), Slot: a.slot,
	}
}
