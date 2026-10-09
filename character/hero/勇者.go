// 勇者：开局场上散落四件遗物（村好剑、死战士、死法师、死圣女），**靠运气撞到**才收编。
// 剑给他近战（手里握着一件武器图，身前扇区摸到人就砍，照盾斧/钉与锤那套）；
// 三位同伴跟在他身后，分别提供免伤 / 弹幕 / 治疗。
// 史莱姆是活随从，只敌视勇者、不敌视敌人——靠把它的 Slot 设成敌人的槽来实现：
// Hittable 只认「不同槽」，所以它咬得到勇者、咬不到敌人，敌人也打不到它。
// 击杀活随从涨经验，每 2^当前等级 升一级：回当时最大生命值的 20%、上限 +10、攻击 +1，可无限叠。
package 勇者

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

const KindHero = "勇者"

const (
	KindSwordPick   = "村好剑"
	KindWarriorPick = "死战士"
	KindMagePick    = "死法师"
	KindSaintPick   = "死圣女"
	KindWarrior     = "战士"
	KindMage        = "法师"
	KindSaint       = "圣女"
	KindSlime       = "史莱姆"
	KindMageShot    = "法师弹"
)

const (
	heroRadius = 18.0
	heroHP     = 100.0
	heroSpeed  = 170.0
	heroVision = 9999.0
	heroColor  = "#e0bc6a" // 金

	relicRadius  = 12.0
	friendRadius = 14.0

	warriorColor = "#d1483c" // 红
	mageColor    = "#4a72d8" // 蓝
	saintColor   = "#f2f4ef" // 白
	swordColor   = "#c2ccd6" // 钢
	slimeColor   = "#45c4f5" // 蓝
	deadTint     = "#6b6f78" // 遗物：死灰

	// 村好剑：手里握着一件武器图，近战是勇者自己按「身前扇区」算的，不是引擎的弧单位。
	swordDamage = 5.0
	swordCD     = 0.35
	swordRange  = 30.0 // 从球心算的够到距离
	swordArcDeg = 90.0 // 身前张角（度）

	// 战士
	shieldCD = 6.0

	// 法师
	mageShotCD     = 2.5
	mageBurst      = 3
	mageBurstGap   = 0.12
	mageShotDamage = 3.0
	mageShotSpeed  = 420.0
	mageShotRadius = 6.0

	// 圣女
	saintHealCD = 4.0
	saintHeal   = 4.0

	// 史莱姆
	slimeRadius = 14.0
	slimeHP     = 12.0
	// 不追人：照靶子那种移动，速度也照它的量级。
	slimeSpeed   = 165.0
	slimeDamage  = 4.0
	slimeHitCD   = 0.5
	slimeSpawnCD = 4.0
	slimeMax     = 4

	// 跟随队形
	followStep     = 4.0
	followGap      = 34.0
	followTrailMax = 48

	// 升级
	levelHPGain   = 10.0
	levelAtkGain  = 1.0
	levelHealRatio = 0.2 // 升级恢复当时最大生命值的比例
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindHero, assets)
	p.Register(unit.Spec{
		Kind:    KindHero,
		Role:    unit.RoleFighter,
		Radius:  heroRadius,
		MaxHP:   heroHP,
		Speed:   heroSpeed,
		Vision:  heroVision,
		Fighter: true,
		Look:    unit.Look{Color: heroColor, FX: []string{"hero"}},
	}, func(unit.SpawnInfo) unit.Actor { return &勇者{level: 1, maxHP: heroHP} })

	// 四件遗物：躺在地上等人捡，什么都不做，只被勇者碰。
	relic := func(kind string, color string) {
		p.Register(unit.Spec{
			Kind:   kind,
			Role:   unit.RoleHelper,
			Radius: relicRadius,
			MaxHP:  1,
			Speed:  0,
			Vision: 0,
			Look:   unit.Look{Color: color, FX: []string{"hero-relic"}},
		}, func(unit.SpawnInfo) unit.Actor { return 遗物{} })
	}
	relic(KindSwordPick, swordColor)
	relic(KindWarriorPick, deadTint)
	relic(KindMagePick, deadTint)
	relic(KindSaintPick, deadTint)

	// 三位同伴：勇者自己 Teleport 它们，所以不用速度也不用视野。
	friend := func(kind string, color string) {
		p.Register(unit.Spec{
			Kind:   kind,
			Role:   unit.RoleHelper,
			Radius: friendRadius,
			MaxHP:  1,
			Speed:  0,
			Vision: 0,
			Look:   unit.Look{Color: color, FX: []string{"hero-friend"}},
		}, func(unit.SpawnInfo) unit.Actor { return 同伴{} })
	}
	friend(KindWarrior, warriorColor)
	friend(KindMage, mageColor)
	friend(KindSaint, saintColor)

	// 史莱姆：活随从，吃伤害、不计胜负。Slot 由生成时定成敌人的槽。
	p.Register(unit.Spec{
		Kind:   KindSlime,
		Role:   unit.RoleMinion,
		Radius: slimeRadius,
		MaxHP:  slimeHP,
		Speed:  slimeSpeed,
		Vision: heroVision,
		Mortal: true,
		Cruise: true,
		Look:   unit.Look{Color: slimeColor, FX: []string{"hero-slime"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &史莱姆{owner: info.OwnerID, slot: info.Slot}
	})

	// 法师弹
	p.Register(unit.Spec{
		Kind:    KindMageShot,
		Role:    unit.RoleProjectile,
		Radius:  mageShotRadius,
		MaxHP:   1,
		Speed:   mageShotSpeed,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: mageColor, Trail: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &法师弹{owner: info.OwnerID, slot: info.Slot}
	})
}

type point struct{ x, y float64 }

type 勇者 struct {
	x, y   float64
	r      float64
	slot   int
	faceX  float64
	faceY  float64
	booted bool
	rng    *rand.Rand

	// 收编状态（全靠撞运气，不主动去找）
	hasSword   bool
	hasWarrior bool
	hasMage    bool
	hasSaint   bool

	// 剑的冷却
	swordReadyAt float64

	// 战士的免伤盾
	shieldReady   bool
	shieldReadyAt float64

	// 法师的连发队列
	mageReadyAt float64
	mageAmmo    int
	mageFireAt  float64

	// 圣女
	saintReadyAt float64

	// 史莱姆
	slimeReadyAt float64

	// 等级
	level int
	exp   int
	maxHP float64
	atk   float64

	// 跟随：勇者自己的面包屑
	trail []point

	nearby []unit.Snapshot
	field  unit.Field
}

func (h *勇者) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		// 战士的免伤：这一次整包取消（照抄地慧星闪避的两段式口径）。
		if h.shieldReady {
			h.shieldReady = false
			unit.BlockHit(ctx, e)
			ctx.Out <- unit.FX{
				Name: "shield", Kind: ctx.Kind, UnitID: ctx.ID, Slot: h.slot,
				X: h.x, Y: h.y,
			}
			return
		}
		unit.ConfirmHit(ctx, e)
	case unit.Kill:
		if e.Mortal {
			h.gainExp(ctx, 1)
		}
	case unit.Sense:
		h.onSense(ctx, e)
	}
}

func (h *勇者) onSense(ctx unit.Context, s unit.Sense) {
	if h.rng == nil {
		h.rng = rand.New(rand.NewPCG(ctx.ID*2654435761, 0x9e3779b97f4a7c15))
	}
	h.x, h.y = s.Self.X, s.Self.Y
	h.r = s.Self.Radius
	h.slot = s.Self.Slot
	h.nearby = s.Nearby
	h.field = s.Field
	if n := math.Hypot(s.Self.VX, s.Self.VY); n > 1e-6 {
		h.faceX, h.faceY = s.Self.VX/n, s.Self.VY/n
	} else if h.faceX == 0 && h.faceY == 0 {
		h.faceX, h.faceY = 1, 0
	}
	// 等级从 1 起、上限按本体血量起。构造时就设好，免得第一拍感知还没到就吃到了击杀。
	if h.level < 1 {
		h.level = 1
	}
	if h.maxHP <= 0 {
		h.maxHP = s.Self.MaxHP
	}

	if !h.booted {
		h.booted = true
		h.scatterRelics(ctx, s)
	}

	h.pickup(ctx, s)
	h.syncAtkMarks(ctx, s)
	h.follow(ctx, s)
	h.tickSword(ctx, s)
	h.tickWarrior(ctx, s)
	h.tickMage(ctx, s)
	h.tickSaint(ctx, s)
	h.tickSlimes(ctx, s)
	h.reportHUD(ctx)
}

// scatterRelics 开局把四件遗物撒在随机可行走点上。
func (h *勇者) scatterRelics(ctx unit.Context, s unit.Sense) {
	for _, kind := range []string{KindSwordPick, KindWarriorPick, KindMagePick, KindSaintPick} {
		x, y, ok := s.Field.RandomWalkable(h.rng, relicRadius+6)
		if !ok {
			x, y = s.Field.Clamp(h.x*-0.5, h.y*0.5, relicRadius+6)
		}
		ctx.Out <- unit.Spawn{Kind: kind, X: x, Y: y, OwnerID: ctx.ID, Slot: h.slot}
	}
}

// pickup 撞到遗物就收编：遗物消失，剑变持有，同伴从遗物位置冒出来跟着走。
// 靠的是随机碰撞，没有主动寻路。
func (h *勇者) pickup(ctx unit.Context, s unit.Sense) {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		var flag *bool
		var follower string
		var code float64 // 给页面认是哪一件：1 剑 / 2 战士 / 3 法师 / 4 圣女
		switch o.Kind {
		case KindSwordPick:
			flag, code = &h.hasSword, 1
		case KindWarriorPick:
			flag, follower, code = &h.hasWarrior, KindWarrior, 2
		case KindMagePick:
			flag, follower, code = &h.hasMage, KindMage, 3
		case KindSaintPick:
			flag, follower, code = &h.hasSaint, KindSaint, 4
		default:
			continue
		}
		if *flag {
			continue
		}
		// 先判接触再改状态：只是看得见不算捡到。
		if math.Hypot(o.X-h.x, o.Y-h.y) > o.Radius+h.r+4 {
			continue
		}
		*flag = true
		ctx.Out <- unit.Despawn{UnitID: o.ID}
		if follower != "" {
			ctx.Out <- unit.Spawn{Kind: follower, X: o.X, Y: o.Y, OwnerID: ctx.ID, Slot: h.slot}
		}
		ctx.Out <- unit.FX{
			Name: "relic", Kind: ctx.Kind, UnitID: ctx.ID, Slot: h.slot,
			X: o.X, Y: o.Y, Amount: code,
		}
	}
}

// follow 让三位同伴沿勇者走过的面包屑排队跟在后面（经典 RPG 跟随）。
func (h *勇者) follow(ctx unit.Context, s unit.Sense) {
	if !h.hasWarrior && !h.hasMage && !h.hasSaint {
		return
	}
	if n := len(h.trail); n == 0 {
		h.trail = append(h.trail, point{h.x, h.y})
	} else {
		last := h.trail[n-1]
		if math.Hypot(h.x-last.x, h.y-last.y) >= followStep {
			h.trail = append(h.trail, point{h.x, h.y})
			if len(h.trail) > followTrailMax {
				h.trail = append([]point(nil), h.trail[len(h.trail)-followTrailMax:]...)
			}
		}
	}
	order := []string{KindWarrior, KindMage, KindSaint}
	which := []bool{h.hasWarrior, h.hasMage, h.hasSaint}
	idx := 0
	for i, kind := range order {
		if !which[i] {
			continue
		}
		idx++
		if tx, ty, ok := h.behindAlong(float64(idx) * followGap); ok {
			for j := range s.Nearby {
				if s.Nearby[j].Kind == kind && s.Nearby[j].OwnerID == ctx.ID {
					ctx.Out <- unit.Teleport{UnitID: s.Nearby[j].ID, X: tx, Y: ty}
					break
				}
			}
		}
	}
}

// behindAlong 沿面包屑往回退 dist，给跟随者算落点。
func (h *勇者) behindAlong(dist float64) (float64, float64, bool) {
	n := len(h.trail)
	if n == 0 {
		return 0, 0, false
	}
	acc := 0.0
	for i := n - 1; i > 0; i-- {
		a, b := h.trail[i], h.trail[i-1]
		seg := math.Hypot(b.x-a.x, b.y-a.y)
		if seg < 1e-9 {
			continue
		}
		if acc+seg >= dist {
			t := (dist - acc) / seg
			return a.x + (b.x-a.x)*t, a.y + (b.y-a.y)*t, true
		}
		acc += seg
	}
	tail := h.trail[0]
	return tail.x, tail.y, true
}

// tickSword 持剑时的近战：勇者自己算「身前扇区里够得着的人」，命中进 CD。
// 不用引擎的弧单位——剑是手里握着的武器图，命中判定在角色这边（盾斧/钉与锤都是这么做的）。
func (h *勇者) tickSword(ctx unit.Context, s unit.Sense) {
	if !h.hasSword || s.Time+1e-9 < h.swordReadyAt {
		return
	}
	cosHalf := math.Cos(unit.Deg(swordArcDeg) / 2)
	amt := swordDamage + h.atk
	hit := false
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, h.slot) {
			continue
		}
		dx, dy := o.X-h.x, o.Y-h.y
		d := math.Hypot(dx, dy)
		if d > swordRange+o.Radius {
			continue
		}
		if d > 1e-6 && (dx*h.faceX+dy*h.faceY)/d < cosHalf {
			continue
		}
		hit = true
		ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: amt}
		ctx.Out <- unit.FX{
			Name: "sword-hit", Kind: ctx.Kind, UnitID: ctx.ID, Slot: h.slot,
			X: o.X, Y: o.Y, Amount: amt,
		}
	}
	if hit {
		h.swordReadyAt = s.Time + swordCD
	}
}

// syncAtkMarks 把当前攻击力写到法师弹身上——法师弹是独立单位，拿不到勇者的字段，
// 所以用标记递过去。没写上就退化成基础伤害，不会打空。
func (h *勇者) syncAtkMarks(ctx unit.Context, s unit.Sense) {
	want := int(h.atk)
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID != ctx.ID || o.Kind != KindMageShot {
			continue
		}
		cur := 0
		for _, m := range o.Marks {
			if m.Kind == markAtk {
				cur = m.Stacks
			}
		}
		if want != cur {
			ctx.Out <- unit.StackMark{UnitID: o.ID, Kind: markAtk, Delta: want - cur}
		}
	}
}

func (h *勇者) tickWarrior(ctx unit.Context, s unit.Sense) {
	if !h.hasWarrior || h.shieldReady {
		return
	}
	if s.Time+1e-9 < h.shieldReadyAt {
		return
	}
	h.shieldReady = true
	h.shieldReadyAt = s.Time + shieldCD
	ctx.Out <- unit.FX{
		Name: "shield-up", Kind: ctx.Kind, UnitID: ctx.ID, Slot: h.slot,
		X: h.x, Y: h.y,
	}
}

// tickMage 每 mageShotCD 起一轮三连发，发完再等下一轮。
func (h *勇者) tickMage(ctx unit.Context, s unit.Sense) {
	if !h.hasMage {
		return
	}
	if h.mageAmmo == 0 && s.Time+1e-9 >= h.mageReadyAt {
		h.mageAmmo = mageBurst
		h.mageFireAt = s.Time
		h.mageReadyAt = s.Time + mageShotCD + mageBurstGap*float64(mageBurst)
	}
	if h.mageAmmo <= 0 || s.Time+1e-9 < h.mageFireAt {
		return
	}
	h.mageAmmo--
	h.mageFireAt = s.Time + mageBurstGap

	// 从法师身上打出去。
	fx, fy := h.x, h.y
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind == KindMage && o.OwnerID == ctx.ID {
			fx, fy = o.X, o.Y
			break
		}
	}
	enemy := h.enemyFighter(s)
	if enemy == nil {
		return
	}
	dx, dy := enemy.X-fx, enemy.Y-fy
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		dx, dy, n = 1, 0, 1
	}
	ux, uy := dx/n, dy/n
	ctx.Out <- unit.Spawn{
		Kind: KindMageShot, OwnerID: ctx.ID, Slot: h.slot,
		X: fx + ux*(friendRadius+mageShotRadius+2), Y: fy + uy*(friendRadius+mageShotRadius+2),
		VX: ux * mageShotSpeed, VY: uy * mageShotSpeed,
	}
}

func (h *勇者) tickSaint(ctx unit.Context, s unit.Sense) {
	if !h.hasSaint || s.Time+1e-9 < h.saintReadyAt {
		return
	}
	h.saintReadyAt = s.Time + saintHealCD
	amt := saintHeal + h.atk
	ctx.Out <- unit.Heal{UnitID: ctx.ID, Amount: amt}
	ctx.Out <- unit.FX{
		Name: "heal-saint", Kind: ctx.Kind, UnitID: ctx.ID, Slot: h.slot,
		X: h.x, Y: h.y, Amount: amt,
	}
}

// tickSlimes 定期在随机可行走点冒一只史莱姆，场上最多 slimeMax 只。
func (h *勇者) tickSlimes(ctx unit.Context, s unit.Sense) {
	alive := 0
	for i := range s.Nearby {
		if s.Nearby[i].Kind == KindSlime {
			alive++
		}
	}
	if alive >= slimeMax || s.Time+1e-9 < h.slimeReadyAt {
		return
	}
	enemy := h.enemyFighter(s)
	if enemy == nil {
		return
	}
	// 别生成在勇者身上：史莱姆的 Pass 要到它第一拍感知才挂上，那一拍之前它还是实心的，
	// 叠着生成会把勇者顶一下。
	var x, y float64
	ok := false
	for try := 0; try < 24; try++ {
		x, y, ok = s.Field.RandomWalkable(h.rng, slimeRadius+6)
		if !ok {
			break
		}
		if math.Hypot(x-h.x, y-h.y) >= heroRadius+slimeRadius+8 {
			break
		}
		ok = false
	}
	if !ok {
		return
	}
	h.slimeReadyAt = s.Time + slimeSpawnCD
	// Slot 设成敌人的槽：这样它只敌视勇者，敌人也碰不到它。
	// 不追人：给一个随机初速，之后就像靶子那样直线飞、撞墙弹开。
	ang := h.rng.Float64() * 2 * math.Pi
	ctx.Out <- unit.Spawn{
		Kind: KindSlime, OwnerID: ctx.ID, Slot: enemy.Slot,
		X: x, Y: y,
		VX: math.Cos(ang) * slimeSpeed, VY: math.Sin(ang) * slimeSpeed,
	}
}

// gainExp 经验够就升级：按当时的最大生命值回 20% → 上限 +10 → 攻击 +1，可无限叠。
func (h *勇者) gainExp(ctx unit.Context, n int) {
	h.exp += n
	for h.exp >= h.expNeed() {
		h.exp -= h.expNeed()
		h.level++
		h.atk += levelAtkGain
		oldMax := h.maxHP
		if oldMax <= 0 {
			oldMax = heroHP
		}
		h.maxHP = oldMax + levelHPGain
		// 先在旧上限下回 20%（满了就浪费，恢复不越过当时的上限），再把上限抬上去。
		// 回血走 Heal：引擎按真实血量结算、自动夹上限并弹 +n 绿字。
		ctx.Out <- unit.Heal{UnitID: ctx.ID, Amount: oldMax * levelHealRatio}
		ctx.Out <- unit.SetHP{UnitID: ctx.ID, HP: -1, MaxHP: h.maxHP}
		ctx.Out <- unit.FX{
			Name: "levelup", Kind: ctx.Kind, UnitID: ctx.ID, Slot: h.slot,
			X: h.x, Y: h.y, Amount: float64(h.level),
		}
	}
}

// expNeed 当前等级升下一级要多少经验：2^当前等级。
func (h *勇者) expNeed() int {
	return 1 << uint(h.level)
}

// reportHUD 把等级/经验发给页面画在球上（FX 不进任何 Actor 的感知，最私密）。
func (h *勇者) reportHUD(ctx unit.Context) {
	ctx.Out <- unit.FX{
		Name: "hero-lv", Kind: ctx.Kind, UnitID: ctx.ID, Slot: h.slot,
		X: h.x, Y: h.y,
		Amount: float64(h.level), VX: float64(h.exp), VY: float64(h.expNeed()),
	}
}

func (h *勇者) enemyFighter(s unit.Sense) *unit.Snapshot {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role == unit.RoleFighter && o.Slot != h.slot {
			return o
		}
	}
	return nil
}
