// Package 传票员：挨打就开票。
//
// 受伤时先确认这一笔（票面要记的是真掉的血），把最近 3 秒挨的次数记下来；
// 内置 CD 到了就 Stand 0.8 秒，把这一笔金额写在票上，然后把票朝伤害来源甩出去。
// 站着的这 0.8 秒里再挨的打也累加在同一张票上（出票即重置）。
// 来源已经不在场上（弹、随从打完就没了）就改甩随机方向。
//
// 每次开票掷一次，按 70/10/10/10 分四种：
//   - 正常：一张票；
//   - 故障（空窗）：一张都不开，也不站（没票可写），CD 减半到 0.4 秒，喊「发生了一些故障」；
//   - 故障（两张）：两张同额，都朝来源，左右各偏 8°；来源没了就一起随机；
//   - 故障（五张）：五张同额，绕一圈均分。
//
// 最近 3 秒挨得越密，正常那一档的几率越低，掉下来的概率平摊给三种故障。
//
// 传票是匀减速的：初速 420，每秒减 300，1.4 秒后停在 294 px 内。
// 票不是子弹：RoleHelper 不参与 CCD，子弹、碰撞都影响不到它；但墙和场边仍会挡住它——
// 位置自己积分后 Teleport，撞场边 / 预放墙用 Field.Clamp、撞现造的胶囊墙用 Sense.Walls，
// 都是推到刚好不压住的位置再反射速度，不会穿出去。
// 敌方战斗机或活随从近了（圆心距 ≤ 双方半径和），按票面金额结算，票随即消失。
package 传票员

import (
	"embed"
	"math"
	"math/rand/v2"
	"sync"

	"xqdj/internal/unit"
)

const (
	KindSubpoena = "传票员"
	KindTicket   = "传票员传票"
)

const (
	bodyRadius = 18.0
	bodyHP     = 100.0
	bodySpeed  = 170.0
	bodyVision = 9999.0 // 要看见伤害来源才瞄得准；看不见就随机方向
	bodyColor  = "#e6d7ac"

	triggerCD = 0.8 // 内置 CD：触发后 0.8 秒内不再开票
	standDur  = 0.8 // 挨打站住写票的时间
	emptyCD   = 0.4 // 空窗故障：不出票，也不站，CD 减半
	hitWindow = 3.0 // 统计最近多少秒挨了几下

	baseNormal  = 0.70 // 1 下时的正常几率
	normalStep  = 0.05 // 每多挨一下降多少
	normalFloor = 0.10 // 降到底

	ticketSpeed  = 420.0
	ticketDecel  = 300.0 // 匀减速，直到 0
	ticketRadius = 10.0
	ticketGap    = 8.0
	ticketColor  = "#f6eed6"

	twoSpread = 8 * math.Pi / 180 // 两张故障票从瞄准线左右偏这么多
	fiveN     = 5

	dt = 1.0 / 60.0
)

// tickets 把「这张票该记多少伤」从发射者交给新单位。Spawn 没有载荷字段，
// 父子之间只靠 ctx.Born 定下的 Stream 对号；工厂取一次就删，不留引用。
var tickets sync.Map // uint64 -> float64

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindSubpoena, assets)
	p.Register(unit.Spec{
		Kind:    KindSubpoena,
		Role:    unit.RoleFighter,
		Radius:  bodyRadius,
		MaxHP:   bodyHP,
		Speed:   bodySpeed,
		Vision:  bodyVision,
		Fighter: true,
		Look: unit.Look{
			Color: bodyColor,
			FX:    []string{"subpoena"},
		},
	}, func(info unit.SpawnInfo) unit.Actor {
		return newSubpoena(info.Stream)
	})
	p.Register(unit.Spec{
		Kind:   KindTicket,
		Role:   unit.RoleHelper, // 不是子弹：不吃 CCD，子弹打不掉，也不碰墙
		Radius: ticketRadius,
		MaxHP:  1,
		Speed:  ticketSpeed,
		Vision: 9999, // 自己按 Nearby 判碰，得看得见全场
		Look:   unit.Look{Color: ticketColor, FX: []string{"ticket"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		dmg, _ := tickets.LoadAndDelete(info.Stream)
		value, _ := dmg.(float64)
		return &传票{dmg: value, slot: info.Slot}
	})
}

type outcome int

const (
	outcomeOne  outcome = iota // 70%：正常，一张
	outcomeNone                // 故障：一张都不开
	outcomeTwo                 // 故障：两张
	outcomeFive                // 故障：五张
)

type shot struct {
	at      float64 // 这一拍开票
	dmg     float64 // 票面
	from    uint64  // 伤害来源（开票时也许已经没了）
	outcome outcome // 触发那一拍就掷好，免得站着的 1 秒里计数又变
}

type 传票员 struct {
	rng   *rand.Rand
	roll  func() float64 // 掷点；测试里可以换掉
	ready float64        // CD 到期时刻
	shots []shot
	hits  []float64 // 最近 3 秒挨打时刻
}

func newSubpoena(stream uint64) *传票员 {
	a := &传票员{rng: unit.NewRNG(stream)}
	a.roll = a.rng.Float64
	return a
}

func (a *传票员) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		a.onHit(ctx, e)
	case unit.Sense:
		a.onSense(ctx, e)
	}
}

func (a *传票员) onHit(ctx unit.Context, d unit.IncomingDamage) {
	unit.ConfirmHit(ctx, d) // 照单全收：先真的掉血，票面才记得准
	if d.Amount <= 0 {
		return
	}
	a.noteHit(d.Time)
	if d.Time < a.ready-1e-9 {
		// CD 中：正处在开票窗口里的话，这一笔也记在那张票上。出票就重置。
		if n := len(a.shots); n > 0 && d.Time < a.shots[n-1].at-1e-9 {
			a.shots[n-1].dmg += d.Amount
		}
		return
	}
	a.ready = d.Time + triggerCD
	sh := shot{
		at:      d.Time + standDur,
		dmg:     d.Amount,
		from:    d.From,
		outcome: a.rollOutcome(),
	}
	if sh.outcome == outcomeNone {
		// 空窗：没有票要写，不用站；CD 减半，机器抖一下接着来。
		sh.at = d.Time
		a.ready = d.Time + emptyCD
	} else {
		ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	}
	a.shots = append(a.shots, sh)
}

// noteHit 把这一拍记进 3 秒窗。窗口只影响下一掷的几率，不影响 CD。
func (a *传票员) noteHit(t float64) {
	cut := t - hitWindow
	keep := a.hits[:0]
	for _, v := range a.hits {
		if v >= cut {
			keep = append(keep, v)
		}
	}
	a.hits = append(keep, t)
}

func (a *传票员) onSense(ctx unit.Context, s unit.Sense) {
	if len(a.shots) == 0 {
		return
	}
	for len(a.shots) > 0 && s.Time >= a.shots[0].at-1e-9 {
		sh := a.shots[0]
		a.shots = a.shots[1:]
		a.fire(ctx, s, sh)
	}
	if len(a.shots) == 0 {
		ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}
	}
}

func (a *传票员) fire(ctx unit.Context, s unit.Sense, sh shot) {
	base := a.aimAt(s, sh.from)
	switch sh.outcome {
	case outcomeNone:
		a.scream(ctx, s)
	case outcomeTwo:
		a.scream(ctx, s)
		a.launch(ctx, s, base-twoSpread, sh.dmg)
		a.launch(ctx, s, base+twoSpread, sh.dmg)
	case outcomeFive:
		a.scream(ctx, s)
		start := a.rng.Float64() * 2 * math.Pi
		for i := 0; i < fiveN; i++ {
			a.launch(ctx, s, start+float64(i)*2*math.Pi/fiveN, sh.dmg)
		}
	default:
		a.launch(ctx, s, base, sh.dmg)
	}
}

// aimAt 朝伤害来源现在的位置。来源已经不在场上（打完就消失的弹、随从）就随机一个方向。
func (a *传票员) aimAt(s unit.Sense, from uint64) float64 {
	for i := range s.Nearby {
		o := s.Nearby[i]
		if o.ID != from {
			continue
		}
		dx, dy := o.X-s.Self.X, o.Y-s.Self.Y
		if math.Hypot(dx, dy) < 1e-9 {
			break
		}
		return math.Atan2(dy, dx)
	}
	return a.rng.Float64() * 2 * math.Pi
}

func (a *传票员) launch(ctx unit.Context, s unit.Sense, ang, dmg float64) {
	off := bodyRadius + ticketRadius + ticketGap
	sp := ctx.Born(unit.Spawn{
		Kind:    KindTicket,
		X:       s.Self.X + math.Cos(ang)*off,
		Y:       s.Self.Y + math.Sin(ang)*off,
		VX:      math.Cos(ang) * ticketSpeed,
		VY:      math.Sin(ang) * ticketSpeed,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	})
	tickets.Store(sp.Stream, dmg)
	ctx.Out <- sp
}

// scream 发故障气泡。FX 名不能叫 "scream"：前端把它当成声音即时通道
// （app.js 的 IMMEDIATE_AUDIO），只会拿到一个没有坐标的 audioOnly 上下文，画面那一遍直接跳过。
func (a *传票员) scream(ctx unit.Context, s unit.Sense) {
	ctx.Out <- unit.FX{
		Name: "fault", Kind: KindSubpoena, UnitID: ctx.ID, Slot: s.Self.Slot,
		X: s.Self.X, Y: s.Self.Y,
	}
}

// rollOutcome 按窗口里挨了几下掷一次。n=1 时正好 70/10/10/10；
// 之后正常每多挨一下降 5 个点，降到 10% 为止，剩下的平摊给三种故障。
func (a *传票员) rollOutcome() outcome {
	p := normalChance(len(a.hits))
	each := (1 - p) / 3
	r := a.roll()
	switch {
	case r < p:
		return outcomeOne
	case r < p+each:
		return outcomeNone
	case r < p+2*each:
		return outcomeTwo
	default:
		return outcomeFive
	}
}

func normalChance(n int) float64 {
	if n < 1 {
		n = 1
	}
	p := baseNormal - normalStep*float64(n-1)
	if p < normalFloor {
		p = normalFloor
	}
	return p
}

// 传票：匀减速到停，停在场上当一张会咬人的纸。
//
// 它不是子弹（RoleHelper）：引擎不给它做 CCD，子弹、碰撞都影响不到它，也就打不掉它。
// 代价是它也不会被物理推动，位置得自己在 Handle 里积分，每帧 Teleport 一次。
type 传票 struct {
	dmg  float64
	slot int

	x, y   float64 // 自己维护的位置
	vx, vy float64 // 自己维护的速度（逐帧匀减速）
	ready  bool    // 第一帧感知里把出生速度和位置接过来
}

func (t *传票) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	// 票面数字每帧都发一次：前端脚本是异步加载的，只发一次可能被漏掉。
	ctx.Out <- unit.FX{
		Name: "ticket", Kind: KindTicket, UnitID: ctx.ID, Slot: t.slot,
		X: s.Self.X, Y: s.Self.Y, Amount: t.dmg,
	}
	if !t.ready {
		t.ready = true
		t.x, t.y = s.Self.X, s.Self.Y
		t.vx, t.vy = s.Self.VX, s.Self.VY
	}
	if t.serve(ctx, s) {
		return
	}
	t.step(ctx, s)
}

// serve 自己判碰：敌方战斗机 / 活随从的圆心距进了双方半径和，就按票面结算，票随即消失。
func (t *传票) serve(ctx unit.Context, s unit.Sense) bool {
	for i := range s.Nearby {
		o := s.Nearby[i]
		if !unit.Hittable(o, t.slot) {
			continue
		}
		dx, dy := o.X-t.x, o.Y-t.y
		reach := o.Radius + ticketRadius
		if dx*dx+dy*dy > reach*reach {
			continue
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: t.dmg}
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return true
	}
	return false
}

// step 走一帧匀减速：用首末速度的平均值推进，位移就是 v²/2a。
// 墙和场边照样挡纸：撞到不可进入的地方就用 Field.Clamp 推回来，再按推出的法线反射速度。
func (t *传票) step(ctx unit.Context, s unit.Sense) {
	sp := math.Hypot(t.vx, t.vy)
	if sp <= 1e-9 {
		return
	}
	next := sp - ticketDecel*dt
	if next < 0 {
		next = 0
	}
	k := next / sp
	nvx, nvy := t.vx*k, t.vy*k

	x := t.x + (t.vx+nvx)*0.5*dt
	y := t.y + (t.vy+nvy)*0.5*dt
	t.vx, t.vy = nvx, nvy

	if s.Field.Extent > 0 && !s.Field.Walkable(x, y, ticketRadius) {
		cx, cy := s.Field.Clamp(x, y, ticketRadius)
		dx, dy := cx-x, cy-y
		if l := math.Hypot(dx, dy); l > 1e-9 {
			dx, dy = dx/l, dy/l
			if d := t.vx*dx + t.vy*dy; d < 0 {
				t.vx -= 2 * d * dx
				t.vy -= 2 * d * dy
			}
		}
		x, y = cx, cy
	}
	x, y = t.bounce(s.Walls, x, y)
	t.x, t.y = x, y
	ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: x, Y: y}
}

// bounce 处理场上现造的胶囊墙（场地预放的墙走 Field.Clamp）。
// 圆心被推进墙里时推到刚好不压住的位置，再沿法线反射速度。
func (t *传票) bounce(walls []unit.WallView, x, y float64) (float64, float64) {
	for _, w := range walls {
		dx, dy := w.X2-w.X1, w.Y2-w.Y1
		l2 := dx*dx + dy*dy
		seg := 0.0
		if l2 > 1e-12 {
			seg = ((x-w.X1)*dx + (y-w.Y1)*dy) / l2
			if seg < 0 {
				seg = 0
			} else if seg > 1 {
				seg = 1
			}
		}
		cx, cy := w.X1+dx*seg, w.Y1+dy*seg
		px, py := x-cx, y-cy
		d := math.Hypot(px, py)
		need := w.Radius + ticketRadius
		if d >= need {
			continue
		}
		if d < 1e-9 {
			px, py = -dy, dx
			if d = math.Hypot(px, py); d < 1e-9 {
				continue
			}
		}
		nx, ny := px/d, py/d
		x, y = cx+nx*need, cy+ny*need
		if vn := t.vx*nx + t.vy*ny; vn < 0 {
			t.vx -= 2 * vn * nx
			t.vy -= 2 * vn * ny
		}
	}
	return x, y
}
