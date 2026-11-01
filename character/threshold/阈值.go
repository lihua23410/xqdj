// 阈值：开局 3 层。报价高于层数时，这一记只按层数结算，层数再加上多出来的一半。
// 有目标才在场地里随机落一笔，不跟目标走。边数是层数的整数部分，至少 3。
// 外接圆半径按层数放大：3 层是 48，层数翻倍半径翻倍。落笔时转一个随机角度，之后不再转。
// 图形先亮 1 秒，这一秒不结算。之后再留一阵，层数翻倍就多留一倍；这段时间里碰到的敌人各打一次，伤害等于出手时的层数。
// 层数越低画得越密。开局间隔 0.5 秒，往上按平方根拉长。
// 出手后层数减去（当前层数 − 开局层数）/ 2。
package 阈值

import (
	"embed"
	"math"
	"math/rand/v2"
	"sync"

	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const KindThreshold = "阈值"

const (
	bodyRadius = 18.0   // 碰撞半径
	bodySpeed  = 148.0  // 巡航。画图形不改自身速度
	bodyVision = 9999.0 // 全场索敌
	bodyHP     = 100.0  // 最大生命，开局满血
	fireAtMin  = 0.5    // 开局层数时两次落笔之间的秒数；更高层数按平方根拉长

	kindSeal        = "阈值印"
	sealRadiusAtMin = 48.0 // 开局层数的外接圆半径。层数按比例放大
	sealWindup      = 1.0  // 先画这么久，这段不结算
	sealLifeAtMin   = 0.5  // 开局层数，亮完之后再留这么久。层数按比例拉长
	sealVision      = 9999.0

	layerMin = 3.0 // 开局层数。少于 3 边画不成多边形
)

func init() {
	p := unit.NewPack(KindThreshold, assets)
	p.Register(unit.Spec{
		Kind:    KindThreshold,
		Role:    unit.RoleFighter,
		Radius:  bodyRadius,
		MaxHP:   bodyHP,
		Speed:   bodySpeed,
		Vision:  bodyVision,
		Fighter: true,
		Look:    unit.Look{Color: "#d6ef6a", FX: []string{"threshold"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &阈值{layers: layerMin, slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:     kindSeal,
		Role:     unit.RoleHelper,
		Radius:   sealRadiusAtMin,
		MaxHP:    1,
		Speed:    0,
		Vision:   sealVision,
		Nonsolid: true,
		Fighter:  false,
		Look:     unit.Look{Color: "#d6ef6a", FX: []string{"seal"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		s, ok := takeSeal(info.Stream)
		if !ok {
			s = seal{sides: int(layerMin), damage: layerMin, reach: sealReach(layerMin), life: sealLife(layerMin)}
		}
		return &阈值印{
			owner: info.OwnerID, slot: info.Slot,
			sides: s.sides, damage: s.damage, reach: s.reach, life: s.life, spin: s.spin,
		}
	})
}

// shotDamage 等于出手时的层数。
func shotDamage(x float64) float64 {
	return x
}

// fireInterval 开局层数是 0.5 秒。层数上去按平方根拉长。
func fireInterval(layers float64) float64 {
	if layers < layerMin {
		layers = layerMin
	}
	return fireAtMin * math.Sqrt(layers/layerMin)
}

// sealReach 外接圆半径。3 层是 48，层数翻倍半径翻倍。
func sealReach(layers float64) float64 {
	if layers < layerMin {
		layers = layerMin
	}
	return sealRadiusAtMin * layers / layerMin
}

// sealLife 亮完之后再留多久。3 层是 0.5 秒，层数翻倍时间翻倍。
func sealLife(layers float64) float64 {
	if layers < layerMin {
		layers = layerMin
	}
	return sealLifeAtMin * layers / layerMin
}

// sidesOf 边数取层数的整数部分。低于 3 仍画三角形。
func sidesOf(layers float64) int {
	n := int(math.Floor(layers))
	if n < 3 {
		return 3
	}
	return n
}

// seal 是落笔那一瞬交给图形的边数和伤害。工厂在另一边取走。
type seal struct {
	sides  int
	damage float64
	reach  float64
	life   float64
	spin   float64
}

var (
	sealMu sync.Mutex
	seals  = map[uint64]seal{}
)

func putSeal(stream uint64, s seal) {
	sealMu.Lock()
	seals[stream] = s
	sealMu.Unlock()
}

func takeSeal(stream uint64) (seal, bool) {
	sealMu.Lock()
	s, ok := seals[stream]
	if ok {
		delete(seals, stream)
	}
	sealMu.Unlock()
	return s, ok
}

type 阈值 struct {
	layers    float64
	slot      int
	fireReady float64
	shown     float64
	didShow   bool
	rng       *rand.Rand
}

func (a *阈值) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		a.onHit(ctx, e)
	case unit.Sense:
		a.shoot(ctx, e)
	}
}

func (a *阈值) onHit(ctx unit.Context, d unit.IncomingDamage) {
	amt := d.Amount
	if amt > a.layers {
		amt = a.layers
		// 多出来的一半加回层数。用的是改写前的报价。
		a.layers += (d.Amount - a.layers) / 2
		a.paint(ctx)
	}
	ctx.Out <- unit.ConfirmDamage{Token: d.Token, UnitID: ctx.ID, Amount: amt}
}

func (a *阈值) shoot(ctx unit.Context, s unit.Sense) {
	a.paint(ctx)
	if s.Time < a.fireReady {
		return
	}
	target := unit.Seek(s)
	if target == nil {
		return
	}
	sides := sidesOf(a.layers)
	damage := shotDamage(a.layers)
	reach := sealReach(a.layers)
	life := sealLife(a.layers)
	if a.rng == nil {
		a.rng = unit.NewRNG(ctx.Stream)
	}
	x, y := placeSeal(a.rng, s.Field, reach)
	spin := a.rng.Float64() * 2 * math.Pi
	// 这一笔的间隔按落笔前的层数算：层数越高，下一笔等得越久。
	a.fireReady = s.Time + fireInterval(a.layers)
	// 减少量 =（当前层数 − 开局层数）/ 2。正好开局层数时减 0。
	a.layers -= (a.layers - layerMin) / 2
	sp := ctx.Born(unit.Spawn{
		Kind:    kindSeal,
		X:       x,
		Y:       y,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	})
	putSeal(sp.Stream, seal{sides: sides, damage: damage, reach: reach, life: life, spin: spin})
	ctx.Out <- sp
	a.paint(ctx)
}

func (a *阈值) paint(ctx unit.Context) {
	if a.didShow && a.shown == a.layers {
		return
	}
	a.didShow = true
	a.shown = a.layers
	ctx.Out <- unit.FX{
		Name:   "threshold-read",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		Slot:   a.slot,
		Amount: a.layers,
	}
}

// placeSeal 在场地里随机挑一个放得下这只外接圆的点。放不下就落在场心。
func placeSeal(rng *rand.Rand, field unit.Field, reach float64) (float64, float64) {
	x, y, ok := field.RandomWalkable(rng, reach)
	if ok {
		return x, y
	}
	return field.Clamp(0, 0, 0)
}

// 阈值印 停在落笔时随机挑的位置。先亮 sealWindup 秒，再留下 life 秒。
// 留下的这段里，每个碰到的敌人只结算一次。
type 阈值印 struct {
	owner  uint64
	slot   int
	sides  int
	damage float64
	reach  float64
	life   float64
	spin   float64
	bornAt float64
	boomAt float64
	endAt  float64
	armed  bool
	hit    map[uint64]bool
	dead   bool
}

func (b *阈值印) Handle(ctx unit.Context, ev unit.Event) {
	if b.dead {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !b.armed {
		b.armed = true
		b.bornAt = s.Time
		b.boomAt = s.Time + sealWindup
		b.endAt = b.boomAt + b.life
		if b.reach > 0 {
			ctx.Out <- unit.SetRadius{UnitID: ctx.ID, Radius: b.reach}
		}
	}
	b.paint(ctx, s)
	if s.Time+1e-9 < b.boomAt {
		return
	}
	b.strike(ctx, s)
	if s.Time+1e-9 < b.endAt {
		return
	}
	b.dead = true
	ctx.Out <- unit.FX{
		Name:   "seal-pop",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		Slot:   b.slot,
		X:      s.Self.X,
		Y:      s.Self.Y,
		VX:     b.reach,
		VY:     b.spin,
		Amount: float64(b.sides),
	}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

func (b *阈值印) paint(ctx unit.Context, s unit.Sense) {
	ctx.Out <- unit.FX{
		Name:   "seal",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		Slot:   b.slot,
		X:      s.Self.X,
		Y:      s.Self.Y,
		VX:     b.bornAt,
		VY:     b.boomAt,
		Amount: float64(b.sides),
	}
	ctx.Out <- unit.FX{
		Name:   "seal-turn",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		Slot:   b.slot,
		Amount: b.spin,
	}
}

func (b *阈值印) strike(ctx unit.Context, s unit.Sense) {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.ID == b.owner || !unit.Hittable(*o, b.slot) {
			continue
		}
		if b.hit[o.ID] || !sealTouches(s.Self.X, s.Self.Y, b.sides, b.reach, b.spin, o.X, o.Y, o.Radius) {
			continue
		}
		if b.hit == nil {
			b.hit = map[uint64]bool{}
		}
		b.hit[o.ID] = true
		ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: b.damage}
	}
}

// sealTouches 身体圆和正多边形相交就算碰到。顶点落在外接圆上。
// spin 是落笔时定下的转角，0 时三角形尖朝上。
func sealTouches(cx, cy float64, sides int, reach, spin, x, y, radius float64) bool {
	if sides < 3 {
		sides = 3
	}
	if reach < 0 {
		reach = 0
	}
	verts := make([][2]float64, sides)
	for i := 0; i < sides; i++ {
		ang := spin - math.Pi/2 + float64(i)*2*math.Pi/float64(sides)
		verts[i] = [2]float64{cx + reach*math.Cos(ang), cy + reach*math.Sin(ang)}
	}
	if insideConvex(x, y, verts) {
		return true
	}
	for i := 0; i < sides; i++ {
		j := (i + 1) % sides
		if distToSeg(x, y, verts[i][0], verts[i][1], verts[j][0], verts[j][1]) <= radius {
			return true
		}
	}
	return false
}

func insideConvex(x, y float64, verts [][2]float64) bool {
	sign := 0
	n := len(verts)
	for i := 0; i < n; i++ {
		x1, y1 := verts[i][0], verts[i][1]
		x2, y2 := verts[(i+1)%n][0], verts[(i+1)%n][1]
		cross := (x2-x1)*(y-y1) - (y2-y1)*(x-x1)
		switch {
		case cross > 1e-6:
			if sign < 0 {
				return false
			}
			sign = 1
		case cross < -1e-6:
			if sign > 0 {
				return false
			}
			sign = -1
		}
	}
	return true
}

func distToSeg(px, py, x1, y1, x2, y2 float64) float64 {
	dx, dy := x2-x1, y2-y1
	l2 := dx*dx + dy*dy
	if l2 < 1e-12 {
		return math.Hypot(px-x1, py-y1)
	}
	t := ((px-x1)*dx + (py-y1)*dy) / l2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return math.Hypot(px-(x1+t*dx), py-(y1+t*dy))
}
