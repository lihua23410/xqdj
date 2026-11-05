// 藤姬：紫色主题的成长型近战。开局那圈藤弧很轻（1 伤），全部强度都压在「藤色」上。
//
// 藤色 是一条额外血条（护盾池），同一刻只在一个实体身上：开局和之后每 10 秒给己方
// 随机一个角色结付一层，每层 10 点池子；换人接手时整包搬过去，不是复制。
// 池子 >0 时挨打整笔被吃掉，本体不掉血，打穿那一下多出来的伤害直接无视；
// 池子每少 10 点掉一层，层数就是 ceil(池子/10)。每层再给 +1 平加伤害（SetDamageBonus）。
// 累计结付每满 6 层召唤一个幻役（0.1 血），把整包池子搬过去再给它 3 层；幻役穿人，
// 高速直线飞、撞墙反弹，贴到人打一下（同目标有 CD），那一下的点数就是它身上藤色的 +X。
// 幻役一失去藤色（池子掉了一层）就气绝，把整包（按掉之前那几层）还给藤姬。
//
// 实现口径：池子只由引擎保存（SetAbsorb），藤姬每拍从快照的 Absorb 读回来推层数，
// 所以掉血、打穿、转移都不会算两遍，也不需要碰生命上限。
package 藤姬

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const KindFujihime = "藤姬"
const KindFujihimeArc = "藤弧"
const KindFujihimeDoll = "幻役"

const (
	fujiRadius = 18.0
	fujiSpeed  = 172.0
	fujiHP     = 15.0
	fujiVision = 9999.0
	fujiColor  = "#a877e8"
	dollColor  = "#c9a2f2"

	// 藤弧：整圈包住身体（和分身弧同一口径），只打 1 点。
	arcInner  = fujiRadius
	arcOuter  = fujiRadius + 2
	arcDamage = 1.5
	arcRearm  = 0.25

	dollRadius = 15.0
	dollHP     = 0.1 // 0 血在引擎里太别扭：给个象征性的 0.1，靠丢藤色气绝

	// 幻役：穿人（Pass），高速直线飞、撞墙直接反弹，贴到人就打一下。
	// 它自己没有任何基础伤害：一下就是身上藤色的 +X。
	dollSpeed  = 560.0 // 飞行速度
	dollHitCD  = 0.6   // 同一个目标多久能再挨它一下
	dollVision = 260.0 // 只用来在感知里认出贴上来的人

	// 幻役在场时把藤姬的嘲讽度压到最低：敌人先去打幻役。
	fujiQuietAim   = 255 // 还锁得住，但排最后
	fujiDefaultAim = 15  // 战斗机默认嘲讽度

	toseKind           = "藤色"
	toseIcon           = "/ball/藤姬/fx/tose.svg"
	toseGrantEvery     = 5.0  // 每这么多秒结付一层
	toseSummonEvery    = 6    // 累计结付这么多层就召唤一个幻役
	toseDollLayers     = 3    // 幻役出生时额外得到的层数
	tosePoolPerLayer   = 15.0 // 每层的池子
	toseDamagePerLayer = 1.5  // 每层的平加伤害（我方角色每有 X 层就 +X）

	// 吃小怪：CD 好了、身边 240 内有血量低于 20 的敌方活随从，就当场吞掉它，
	// 换一只幻役出来并喂它一层藤色。
	devourCD     = 20.0
	devourRange  = 240.0
	devourMaxHP  = 20.0
	devourLayers = 1
)

// 数值合约：层数是池子算出来的（ceil(池子/15)），看守和对局测试都按这几个数走。
const (
	ToseKind           = toseKind
	TosePoolPerLayer   = tosePoolPerLayer
	ToseDamagePerLayer = toseDamagePerLayer
)

func init() {
	p := unit.NewPack(KindFujihime, assets)
	p.Register(unit.Spec{
		Kind:    KindFujihime,
		Role:    unit.RoleFighter,
		Radius:  fujiRadius,
		MaxHP:   fujiHP,
		Speed:   fujiSpeed,
		Vision:  fujiVision,
		Fighter: true,
		Look:    unit.Look{Color: fujiColor, Glow: true, FX: []string{"fujihime"}},
	}, func(unit.SpawnInfo) unit.Actor { return &藤姬{} })

	// 藤弧：整圈贴身的扇环，打中后缩回去 rearm。
	p.Register(unit.Spec{
		Kind:     KindFujihimeArc,
		Role:     unit.RoleProjectile,
		Radius:   arcOuter,
		MaxHP:    1,
		Speed:    fujiSpeed,
		Vision:   0,
		Attach:   true,
		ArcSpan:  2 * math.Pi,
		ArcInner: arcInner,
		Look:     unit.Look{Color: fujiColor, Overlay: true, Glow: true, FX: []string{"fujihime-arc"}},
	}, func(info unit.SpawnInfo) unit.Actor { return &藤弧{slot: info.Slot} })

	// 幻役：藤色的容器，0.1 血。穿人（Pass）不挡路，高速追人，
	// CD 到了就冲一下，冲刺贴到人才出伤。
	p.Register(unit.Spec{
		Kind:   KindFujihimeDoll,
		Role:   unit.RoleMinion,
		Radius: dollRadius,
		MaxHP:  dollHP,
		Speed:  dollSpeed,
		Vision: dollVision,
		Mortal: true,
		Cruise: true,
		Pass:   true,
		Look:   unit.Look{Color: dollColor, Ghost: 120, FX: []string{"fujihime-doll"}},
	}, func(unit.SpawnInfo) unit.Actor { return &幻役{} })

	// 藤色看守：只记账的实体，不进任何人的感知、也不上页面（Hidden）。
	// 它不带主人，所以藤姬倒下之后它还在，继续按池子掉层。
	p.Register(unit.Spec{
		Kind:     KindToseKeeper,
		Role:     unit.RoleHelper,
		Radius:   1,
		MaxHP:    1,
		Speed:    0,
		Vision:   fujiVision,
		Nonsolid: true,
		Hidden:   true,
	}, func(info unit.SpawnInfo) unit.Actor { return newKeeper(info.Data) })
}

type 藤姬 struct {
	arc    unit.AttachState
	rng    *rand.Rand
	booted bool

	nextGrant float64 // 下一次结付时刻
	grants    int     // 累计结付次数（只增不减）

	holder uint64  // 藤色现在在谁身上（0 = 没有）
	pool   float64 // 上一拍看到的池子
	layers int     // 池子对应的层数（也是已下发的标记/加伤）
	doll   bool    // 当前持有人是不是幻役

	// 幻役刚出生时还没进快照，池子要等它出现那一拍再交。
	waitDoll float64

	// 幻役在场时降低自己的嘲讽度，走了再恢复。
	quiet bool

	// 幻役最后一拍的位置：它被别处打死时，气绝要留在原地。
	dollX, dollY float64

	// 吃小怪的 CD：上来先压满一拍，开局不白吃。
	devourAt float64
}

func (f *藤姬) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		unit.ConfirmHit(ctx, e)
	case unit.Sense:
		f.tick(ctx, e)
	}
}

func (f *藤姬) tick(ctx unit.Context, s unit.Sense) {
	if !f.booted {
		f.booted = true
		f.rng = unit.NewRNG(ctx.Stream)
		f.devourAt = s.Time + devourCD
	}
	if unit.RearmAttach(s, ctx.ID, KindFujihimeArc, arcRearm, &f.arc) {
		unit.SpawnAttach(ctx, s, KindFujihimeArc)
	}
	// 先看当前持有人，再把新幻役接进账本：换人那一拍不能拿旧快照当成掉层。
	f.syncHolder(ctx, s)
	f.adoptDoll(ctx, s)
	f.quietWhileDoll(ctx, s)
	if s.Time+1e-9 >= f.nextGrant {
		f.nextGrant = nextGrantAt(s.Time)
		f.grant(ctx, s)
	}
	f.tryDevour(ctx, s)
}

// nextGrantAt 把结付对齐到整十秒；开局那一拍也算一次。
func nextGrantAt(t float64) float64 {
	return (math.Floor(t/toseGrantEvery+1e-9) + 1) * toseGrantEvery
}

// layersOf 池子 → 层数。掉满 10 点才少一层，25 点还是 3 层。
func layersOf(pool float64) int {
	if pool <= 0 {
		return 0
	}
	return int(math.Ceil(pool/tosePoolPerLayer - 1e-9))
}

func poolOf(layers int) float64 { return float64(layers) * tosePoolPerLayer }

// bonus 是「每有 X 层就额外 +X」的那份平加伤害。
func (f *藤姬) bonus() float64 { return float64(f.layers) * toseDamagePerLayer }

// grant 结付一层：随机挑一个己方角色，接手整包再加一层。
func (f *藤姬) grant(ctx unit.Context, s unit.Sense) {
	f.grants++
	target := f.pickHolder(ctx, s)
	if target == f.holder {
		f.addLayers(ctx, s, 1)
	} else {
		f.moveTo(ctx, s, target, f.pool+tosePoolPerLayer)
	}
	if f.grants%toseSummonEvery == 0 {
		f.summon(ctx, s, toseDollLayers)
	}
}

// tryDevour 吃掉身边一个没多少血的敌方活随从，换一只幻役出来，并喂它一层藤色。
// 只在场上没有自己的幻役时动手：幻役出生要靠「接手整包」认领这一包池子，
// 场上已经有旧的，新出来的那只就接不到，等于白给。
func (f *藤姬) tryDevour(ctx unit.Context, s unit.Sense) {
	if s.Time+1e-9 < f.devourAt || f.findDoll(ctx.ID, s) != 0 {
		return
	}
	victim := f.devourTarget(s)
	if victim == nil {
		return
	}
	f.devourAt = s.Time + devourCD
	ctx.Out <- unit.Despawn{UnitID: victim.ID}
	ctx.Out <- unit.FX{
		Name: "fujihime-doll-hit", Kind: KindFujihime, UnitID: victim.ID,
		Slot: s.Self.Slot, X: victim.X, Y: victim.Y,
	}
	f.summon(ctx, s, devourLayers)
}

// devourTarget 240 以内、血量低于 devourMaxHP 的敌方活随从里最近的那个。
func (f *藤姬) devourTarget(s unit.Sense) *unit.Snapshot {
	best := devourRange * devourRange
	var got *unit.Snapshot
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !o.Mortal || o.Slot == s.Self.Slot || o.HP >= devourMaxHP {
			continue
		}
		dx, dy := o.X-s.Self.X, o.Y-s.Self.Y
		if d := dx*dx + dy*dy; d <= best {
			best, got = d, o
		}
	}
	return got
}

// addLayers 给当前持有人加 n 层的池子，标记和加伤跟着走。
// 池子在队友身上时不在这里改标记：那是看守的活，两边都写会叠双。
func (f *藤姬) addLayers(ctx unit.Context, s unit.Sense, n int) {
	if f.holder == 0 {
		return
	}
	f.pool += float64(n) * tosePoolPerLayer
	ctx.Out <- unit.SetAbsorb{UnitID: f.holder, Amount: f.pool}
	if f.holder == ctx.ID {
		f.retune(ctx, f.holder)
	}
	f.flash(ctx, s, f.holder)
}

// moveTo 把整包池子搬到 target（规则 5：藤色同时只在一个实体上）。
// 旧持有人是幻役的话，它这一下就等于失去藤色——气绝。
func (f *藤姬) moveTo(ctx unit.Context, s unit.Sense, target uint64, pool float64) {
	if old := f.holder; old != 0 && old != target {
		if f.doll {
			f.scream(ctx, s, old)
			ctx.Out <- unit.Despawn{UnitID: old}
		}
		ctx.Out <- unit.ClearMarks{UnitID: old, Kind: toseKind}
		ctx.Out <- unit.SetAbsorb{UnitID: old}
		ctx.Out <- unit.SetDamageBonus{UnitID: old}
	}
	f.holder, f.doll = target, target != 0 && f.isDoll(s, target)
	f.pool = pool
	f.layers = layersOf(pool)
	ctx.Out <- unit.SetAbsorb{UnitID: target, Amount: pool}
	ctx.Out <- unit.SetDamageBonus{UnitID: target, Amount: f.bonus()}
	ctx.Out <- unit.StackMark{UnitID: target, Kind: toseKind, Delta: f.layers, Icon: toseIcon}
	if target != ctx.ID && !f.doll {
		f.sendKeeper(ctx, s, target)
	}
	f.flash(ctx, s, target)
}

// sendKeeper 池子落在队友身上时，派一个看守跟着这本账：
// 层数和每层加伤从此由它维护，藤姬倒下之后也不会停。
// 看守不带主人，所以不会跟着她一起被撤。
func (f *藤姬) sendKeeper(ctx unit.Context, s unit.Sense, holder uint64) {
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: KindToseKeeper,
		X:    s.Self.X, Y: s.Self.Y,
		Slot: s.Self.Slot,
		Data: keeperData{owner: ctx.ID, holder: holder},
	})
}

// retune 让标记和加伤跟上当前池子的层数（池子掉层时用）。
func (f *藤姬) retune(ctx unit.Context, id uint64) {
	next := layersOf(f.pool)
	if next == f.layers {
		return
	}
	ctx.Out <- unit.StackMark{UnitID: id, Kind: toseKind, Delta: next - f.layers, Icon: toseIcon}
	ctx.Out <- unit.SetDamageBonus{UnitID: id, Amount: float64(next) * toseDamagePerLayer}
	f.layers = next
}

// syncHolder 每拍从快照读回持有人的池子：掉层就更新标记；幻役掉层即气绝。
func (f *藤姬) syncHolder(ctx unit.Context, s unit.Sense) {
	if f.holder == 0 {
		return
	}
	snap := f.snapOf(s, f.holder)
	if snap == nil {
		// 持有人没了。幻役是死在别处的（被一下打空之类）：原地气绝，整包还给藤姬。
		n, wasDoll := f.layers, f.doll
		id, x, y := f.holder, f.dollX, f.dollY
		f.holder, f.doll, f.pool, f.layers = 0, false, 0, 0
		if wasDoll {
			f.screamAt(ctx, s, id, x, y)
			if n > 0 {
				f.takePool(ctx, s, poolOf(n))
			}
		}
		return
	}
	f.pool = snap.Absorb
	if f.doll {
		f.dollX, f.dollY = snap.X, snap.Y
	}
	if snap.Absorb <= 0 && f.layers == 0 {
		return
	}
	next := layersOf(f.pool)
	if next >= f.layers {
		return
	}
	if f.doll {
		// 幻役失去藤色：整包按掉之前那几层还给藤姬，然后气绝。
		n := f.layers
		f.scream(ctx, s, f.holder)
		ctx.Out <- unit.ClearMarks{UnitID: f.holder, Kind: toseKind}
		ctx.Out <- unit.Despawn{UnitID: f.holder}
		f.holder, f.doll, f.pool, f.layers = 0, false, 0, 0
		f.takePool(ctx, s, poolOf(n))
		return
	}
	if f.holder != ctx.ID {
		// 池子在队友身上：掉层归看守管，这里只跟自己的账本。
		f.layers = layersOf(f.pool)
		return
	}
	f.retune(ctx, f.holder)
}

// takePool 把一整包池子接到藤姬自己身上。
func (f *藤姬) takePool(ctx unit.Context, s unit.Sense, pool float64) {
	if pool <= 0 {
		return
	}
	f.holder, f.doll = ctx.ID, false
	f.pool = pool
	f.layers = layersOf(pool)
	ctx.Out <- unit.SetAbsorb{UnitID: ctx.ID, Amount: pool}
	ctx.Out <- unit.SetDamageBonus{UnitID: ctx.ID, Amount: f.bonus()}
	ctx.Out <- unit.StackMark{UnitID: ctx.ID, Kind: toseKind, Delta: f.layers, Icon: toseIcon}
}

// quietWhileDoll 幻役在场时把自己降到最低嘲讽度（敌人先去打幻役），幻役没了就恢复。
func (f *藤姬) quietWhileDoll(ctx unit.Context, s unit.Sense) {
	want := f.findDoll(ctx.ID, s) != 0
	if want == f.quiet {
		return
	}
	f.quiet = want
	aim := uint8(fujiDefaultAim)
	if want {
		aim = fujiQuietAim
	}
	unit.SetAim(ctx, ctx.ID, aim)
}

// summon 召唤一个幻役，登记要交给它的整包（现有 + extra 层）。
func (f *藤姬) summon(ctx unit.Context, s unit.Sense, extra int) {
	ang := f.rng.Float64() * 2 * math.Pi
	x := s.Self.X + math.Cos(ang)*(fujiRadius+dollRadius+4)
	y := s.Self.Y + math.Sin(ang)*(fujiRadius+dollRadius+4)
	// 初速朝最近的敌人丢出去；没有敌人就随机方向。之后不追人，撞墙反弹。
	ux, uy := math.Cos(ang), math.Sin(ang)
	if prey := unit.Seek(s); prey != nil {
		dx, dy := prey.X-s.Self.X, prey.Y-s.Self.Y
		if n := math.Hypot(dx, dy); n > 1e-6 {
			ux, uy = dx/n, dy/n
		}
	}
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind:    KindFujihimeDoll,
		X:       x,
		Y:       y,
		VX:      ux * dollSpeed,
		VY:      uy * dollSpeed,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
		Stream:  unit.DeriveStream(ctx.Stream, uint64(f.grants), 0x7e1),
	})
	f.waitDoll = f.pool + float64(extra)*tosePoolPerLayer
}

// adoptDoll 等新幻役出现在感知里，再把整包池子交给它。
func (f *藤姬) adoptDoll(ctx unit.Context, s unit.Sense) {
	if f.waitDoll <= 0 {
		return
	}
	id := f.findDoll(ctx.ID, s)
	if id == 0 {
		return
	}
	want := f.waitDoll
	f.waitDoll = 0
	f.moveTo(ctx, s, id, want)
}

func (f *藤姬) findDoll(owner uint64, s unit.Sense) uint64 {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind == KindFujihimeDoll && o.OwnerID == owner {
			return o.ID
		}
	}
	return 0
}

func (f *藤姬) snapOf(s unit.Sense, id uint64) *unit.Snapshot {
	if id == 0 {
		return nil
	}
	if s.Self.ID == id {
		return &s.Self
	}
	for i := range s.Nearby {
		if s.Nearby[i].ID == id {
			return &s.Nearby[i]
		}
	}
	return nil
}

func (f *藤姬) isDoll(s unit.Sense, id uint64) bool {
	snap := f.snapOf(s, id)
	return snap != nil && snap.Kind == KindFujihimeDoll
}

// pickHolder 从己方角色里随机挑一个：自己、同槽的战斗机、自己名下的幻役。
func (f *藤姬) pickHolder(ctx unit.Context, s unit.Sense) uint64 {
	pool := make([]uint64, 0, 4)
	pool = append(pool, ctx.ID)
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Slot != s.Self.Slot || o.ID == ctx.ID {
			continue
		}
		if o.Kind == KindFujihimeDoll && o.OwnerID == ctx.ID {
			pool = append(pool, o.ID)
			continue
		}
		if o.Role == unit.RoleFighter {
			pool = append(pool, o.ID)
		}
	}
	return pool[f.rng.IntN(len(pool))]
}

func (f *藤姬) flash(ctx unit.Context, s unit.Sense, id uint64) {
	x, y := s.Self.X, s.Self.Y
	if snap := f.snapOf(s, id); snap != nil {
		x, y = snap.X, snap.Y
	}
	ctx.Out <- unit.FX{
		Name: "fujihime-tose", Kind: KindFujihime, UnitID: id, Slot: s.Self.Slot,
		X: x, Y: y, Amount: float64(f.layers),
	}
}

func (f *藤姬) scream(ctx unit.Context, s unit.Sense, id uint64) {
	x, y := s.Self.X, s.Self.Y
	if snap := f.snapOf(s, id); snap != nil {
		x, y = snap.X, snap.Y
	}
	f.screamAt(ctx, s, id, x, y)
}

// screamAt 在指定位置竖起「气绝」：幻役不管怎么死的，都得在原地喊一声。
func (f *藤姬) screamAt(ctx unit.Context, s unit.Sense, id uint64, x, y float64) {
	ctx.Out <- unit.FX{
		Name: "fujihime-scream", Kind: KindFujihime, UnitID: id, Slot: s.Self.Slot,
		X: x, Y: y,
	}
}
