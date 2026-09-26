// 老鼠：吹笛人的概念神。平时自己挑对手场上的东西叼走，被指挥时掉头扑上去咬。
// 叼到的东西用「以被偷 Kind 为名」的标记记着，主人靠它知道老鼠嘴里是什么，也靠它把半路
// 被打死的东西还给原主人。
//
// 老鼠和老鼠、老鼠和本体挤在一起不好看，所以：一是每只老鼠按自己的 id 分到一个接近角，
// 回巢、咬人、下手都走各自的角度，不往同一个点上堆；二是咬人隔着身位咬（ratBite），
// 咬完沿径向退到 ratStandoff 再站定，不贴着敌人蹭。全程不持 Pass——持了虽然不互推，
// 但会互相穿过去，看起来像老鼠钻进本体里，而且对手的弹也会从它身上穿过去。
package 吹笛人

import (
	"math"

	"xqdj/internal/unit"
)

const (
	ratRadius   = 9.0
	ratHP       = 8.0
	ratChase    = 260.0 // 出猎 / 扑敌的速度
	ratCarry    = 180.0 // 叼着东西往回跑
	ratIdle     = 140.0 // 没东西可偷时飘回主人脚边
	ratRetarget = 0.5   // 隔这么久重挑一次目标
	ratReach    = 4.0   // 够到目标多近算叼住
	ratRingGap  = 18.0  // 待命时停在主人球面外这么远
	ratBite     = 15.0  // 咬的判定：离敌方球面这么近就咬下去，不用真的贴上
	ratBiteIn   = 10.0  // 扑上去瞄的站位（比判定近一点，保证会咬）
	ratBack     = 0.9   // 咬完退开这么久
	ratStandoff = 34.0  // 退到敌方球面外这么远
	biteBase    = 3.0   // 每口咬伤，再加指挥带下来的赃物加成
	ratColor    = "#8d8d8d"
)

type 老鼠 struct {
	owner      uint64
	slot       int
	booted     bool
	carrying   bool
	bitten     bool
	backUntil  float64
	targetID   uint64
	targetKind string
	hasTarget  bool
	nextPick   float64
}

func newRat(info unit.SpawnInfo) *老鼠 {
	return &老鼠{owner: info.OwnerID, slot: info.Slot}
}

func (r *老鼠) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	if s, ok := ev.(unit.Sense); ok {
		r.onSense(ctx, s)
	}
}

func (r *老鼠) onSense(ctx unit.Context, s unit.Sense) {
	if !r.booted {
		r.booted = true
		// 常驻 Pass，和钉与锤的人偶同一套：看得见、打得死，但谁都能穿过它，
		// 所以老鼠不挤老鼠、不挤本体，也不跟敌人顶牛。
		ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
	}
	commanded := s.Self.AimPriority >= cmdAim
	if !commanded {
		r.bitten = false
	}
	switch {
	case commanded && !r.bitten:
		r.attack(ctx, s, int(s.Self.AimPriority)-cmdAim)
	case r.bitten && s.Time+1e-9 < r.backUntil:
		r.retreat(ctx, s)
	case r.carrying:
		r.runTo(ctx, s, r.owner, ratCarry, ratRingGap)
	default:
		t := r.find(s)
		if t == nil || s.Time+1e-9 >= r.nextPick {
			r.pick(s)
			t = r.find(s)
		}
		if t == nil {
			r.runTo(ctx, s, r.owner, ratIdle, ratRingGap)
		} else {
			r.stalk(ctx, s, t)
		}
	}
}

// find 取当前锁定目标的最新快照；它已经不在场上了就返回 nil。
func (r *老鼠) find(s unit.Sense) *unit.Snapshot {
	if !r.hasTarget {
		return nil
	}
	for i := range s.Nearby {
		if s.Nearby[i].ID == r.targetID {
			return &s.Nearby[i]
		}
	}
	return nil
}

// pick 挑最近的一件「东西」：敌方 slot，且不是战斗机、不是活随从。墙不在感知里，偷不到。
func (r *老鼠) pick(s unit.Sense) {
	r.nextPick = s.Time + ratRetarget
	r.hasTarget = false
	best := math.MaxFloat64
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Slot == r.slot || o.ID == s.Self.ID {
			continue
		}
		if o.Role == unit.RoleFighter || o.Mortal {
			continue
		}
		dx, dy := o.X-s.Self.X, o.Y-s.Self.Y
		if d := dx*dx + dy*dy; d < best {
			best = d
			r.targetID, r.targetKind, r.hasTarget = o.ID, o.Kind, true
		}
	}
}

// stalk 绕到目标自己的角度上再下嘴，免得六只老鼠全挤在同一个点。
func (r *老鼠) stalk(ctx unit.Context, s unit.Sense, t *unit.Snapshot) {
	if math.Hypot(t.X-s.Self.X, t.Y-s.Self.Y) <= t.Radius+s.Self.Radius+ratReach {
		r.snatch(ctx, t)
		return
	}
	px, py := ringPoint(t.X, t.Y, t.Radius+s.Self.Radius+ratReach, ratAngle(s))
	r.steer(ctx, s, px, py, ratChase)
}

// snatch 叼住：目标当场消失，自己带上「嘴里是什么」的标记，转头往主人那边跑。
func (r *老鼠) snatch(ctx unit.Context, t *unit.Snapshot) {
	ctx.Out <- unit.Despawn{UnitID: t.ID}
	ctx.Out <- unit.StackMark{UnitID: ctx.ID, Kind: t.Kind, Delta: 1}
	ctx.Out <- unit.FX{Name: "snatch", Kind: ctx.Kind, UnitID: ctx.ID, X: t.X, Y: t.Y, Slot: r.slot}
	r.targetKind = t.Kind
	r.carrying = true
	r.hasTarget = false
}

// attack 被指挥了：扑向敌方战斗机，隔着身位就咬，不往身上撞。一次指挥里每只老鼠只咬一口。
func (r *老鼠) attack(ctx unit.Context, s unit.Sense, bonus int) {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role != unit.RoleFighter || o.Slot == r.slot {
			continue
		}
		gap := math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) - o.Radius - s.Self.Radius
		if gap <= ratBite {
			if !r.bitten {
				r.bitten = true
				r.backUntil = s.Time + ratBack
				ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: biteBase + float64(bonus)}
				ctx.Out <- unit.FX{Name: "bite", Kind: ctx.Kind, UnitID: ctx.ID, X: o.X, Y: o.Y, Slot: r.slot}
			}
			r.backOff(ctx, s, o)
			return
		}
		px, py := ringPoint(o.X, o.Y, o.Radius+s.Self.Radius+ratBiteIn, ratAngle(s))
		r.steer(ctx, s, px, py, ratChase)
		return
	}
}

// backOff 咬完沿「敌人 → 自己」这条线径直退开，别贴着敌人一直蹭。
// 教父的暗杀者也是这个路子：碰到就断触、进冷却。
func (r *老鼠) backOff(ctx unit.Context, s unit.Sense, o *unit.Snapshot) {
	dx, dy := s.Self.X-o.X, s.Self.Y-o.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		dx, dy, n = 1, 0, 1
	}
	want := o.Radius + s.Self.Radius + ratStandoff
	r.steer(ctx, s, o.X+dx/n*want, o.Y+dy/n*want, ratCarry)
}

// retreat 咬完那一下的退开：退到自己那个角度的站位上；敌人没了就停下。
func (r *老鼠) retreat(ctx unit.Context, s unit.Sense) {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role != unit.RoleFighter || o.Slot == r.slot {
			continue
		}
		r.backOff(ctx, s, o)
		return
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID}
}

// runTo 跑到某个单位身边的一个环位点上：`gap` 是停在它球面外多远。
func (r *老鼠) runTo(ctx unit.Context, s unit.Sense, id uint64, speed, gap float64) {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.ID != id {
			continue
		}
		px, py := ringPoint(o.X, o.Y, o.Radius+s.Self.Radius+gap, ratAngle(s))
		r.steer(ctx, s, px, py, speed)
		return
	}
}

// steer 朝一个点走，快到了就减速，免得在目标点上来回蹭；再叠一层对自己人的排斥，
// 这样即使没有碰撞体积，也不会互相穿过去或者被本体压在身下。
func (r *老鼠) steer(ctx unit.Context, s unit.Sense, tx, ty, speed float64) {
	vx, vy := 0.0, 0.0
	dx, dy := tx-s.Self.X, ty-s.Self.Y
	if d := math.Hypot(dx, dy); d > 1e-6 {
		sp := math.Min(speed, d*6)
		if sp < 20 {
			sp = 0
		}
		vx, vy = dx/d*sp, dy/d*sp
	}
	ax, ay := r.avoid(s)
	vx, vy = vx+ax, vy+ay
	if n := math.Hypot(vx, vy); n > speed {
		vx, vy = vx/n*speed, vy/n*speed
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: vx, VY: vy}
}

// avoid 对自己人（本体与同类）的排斥：贴太近就往外让，让开的力度随距离衰减。
func (r *老鼠) avoid(s unit.Sense) (float64, float64) {
	const keepGap = 10.0
	var ax, ay float64
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.ID != r.owner && o.OwnerID != r.owner {
			continue
		}
		dx, dy := s.Self.X-o.X, s.Self.Y-o.Y
		d := math.Hypot(dx, dy)
		if d < 1e-6 {
			dx, dy, d = 1, 0, 1
		}
		keep := o.Radius + s.Self.Radius + keepGap
		if d >= keep {
			continue
		}
		w := (keep - d) / keep
		ax += dx / d * w * ratChase
		ay += dy / d * w * ratChase
	}
	return ax, ay
}

// ratAngle 每只老鼠一个固定接近角（按 id 分），最多六只互不重叠。
func ratAngle(s unit.Sense) float64 {
	return float64(s.Self.ID%6) * math.Pi / 3
}

func ringPoint(x, y, r, ang float64) (float64, float64) {
	return x + math.Cos(ang)*r, y + math.Sin(ang)*r
}
