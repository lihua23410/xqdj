// 老鼠：吹笛人的概念神。默认去贴敌人；有可偷的东西就去叼。攻击环不偷。
// 叼到的东西用「以被偷 Kind 为名」的标记记着。被打死由主人叠瘟疫；活着贴着也能叠。
package 吹笛人

import (
	"math"

	"xqdj/internal/unit"
)

const (
	ratRadius   = 9.0
	ratHP       = 8.0
	ratChase    = 260.0 // 出猎的速度
	ratCarry    = 180.0 // 叼着东西往回跑
	ratIdle     = 140.0 // 没敌人可贴时飘回主人脚边
	ratRetarget = 0.5   // 隔这么久重挑一次目标
	ratReach    = 20.0  // 离目标球面这么近就算叼住，不用贴上去
	ratRingGap  = 18.0  // 待命时停在主人球面外这么远
	ratStandoff = 16.0  // 贴敌时停在球面外这么远，不咬
	ratColor    = "#8d8d8d"
)

type 老鼠 struct {
	owner      uint64
	slot       int
	booted     bool
	carrying   bool
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
	if r.carrying {
		r.runTo(ctx, s, r.owner, ratCarry, ratRingGap)
		return
	}
	t := r.find(s)
	if t == nil || s.Time+1e-9 >= r.nextPick {
		r.pick(s)
		t = r.find(s)
	}
	if t == nil {
		if e := r.nearestFoe(s); e != nil {
			r.hover(ctx, s, e, ratChase, ratStandoff)
			return
		}
		r.runTo(ctx, s, r.owner, ratIdle, ratRingGap)
		return
	}
	r.stalk(ctx, s, t)
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

// pick 挑视野里最近的一件「东西」。
func (r *老鼠) pick(s unit.Sense) {
	r.nextPick = s.Time + ratRetarget
	r.hasTarget = false
	best := math.MaxFloat64
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !stealable(*o, r.slot, s.Self.ID) {
			continue
		}
		dx, dy := o.X-s.Self.X, o.Y-s.Self.Y
		d := dx*dx + dy*dy
		if d >= best {
			continue
		}
		best = d
		r.targetID, r.targetKind, r.hasTarget = o.ID, o.Kind, true
	}
}

func stealable(o unit.Snapshot, slot int, selfID uint64) bool {
	if o.Slot == slot || o.ID == selfID {
		return false
	}
	if o.Role == unit.RoleFighter || o.Mortal {
		return false
	}
	if o.ArcSpan != 0 {
		return false
	}
	return true
}

// nearestFoe 最近的可命中敌人：敌方战斗机或活随从。没东西可偷时去贴它，把瘟疫圈送上去。
func (r *老鼠) nearestFoe(s unit.Sense) *unit.Snapshot {
	best := math.MaxFloat64
	var got *unit.Snapshot
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, r.slot) {
			continue
		}
		dx, dy := o.X-s.Self.X, o.Y-s.Self.Y
		if d := dx*dx + dy*dy; d < best {
			best = d
			got = o
		}
	}
	return got
}

// stalk 朝目标中心冲，进判定圈就下嘴。不把瞄点放在圈沿上：steer 靠近会提前刹停，
// 瞄在沿上时有的老鼠会停在刚够不着的地方。
func (r *老鼠) stalk(ctx unit.Context, s unit.Sense, t *unit.Snapshot) {
	if math.Hypot(t.X-s.Self.X, t.Y-s.Self.Y) <= t.Radius+s.Self.Radius+ratReach {
		r.snatch(ctx, t)
		return
	}
	r.steer(ctx, s, t.X, t.Y, ratChase)
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

// hover 停在目标自己这一侧的待命点上，不穿过去绕到对面。
func (r *老鼠) hover(ctx unit.Context, s unit.Sense, o *unit.Snapshot, speed, gap float64) {
	dx, dy := o.X-s.Self.X, o.Y-s.Self.Y
	d := math.Hypot(dx, dy)
	if d < 1e-6 {
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID}
		return
	}
	want := o.Radius + s.Self.Radius + gap
	r.steer(ctx, s, o.X-dx/d*want, o.Y-dy/d*want, speed)
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
