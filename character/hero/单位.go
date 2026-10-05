package 勇者

import (
	"math"

	"xqdj/internal/unit"
)

// markAtk 是勇者写在法师弹身上的「当前攻击力」。法师弹是独立单位、拿不到勇者的字段，
// 所以用标记把攻击力递过去；没来得及写就是 0，退化成基础伤害，不会打空。
// （村好剑的近战不走这里：那是勇者自己在 Sense 里算的，直接用 h.atk。）
const markAtk = "勇者攻击力"

// 遗物 躺在地上什么都不做，只等勇者用身体撞到它。
type 遗物 struct{}

func (遗物) Handle(unit.Context, unit.Event) {}

// 同伴 的位置由勇者每帧 Teleport 决定，自己不需要任何逻辑。
// 它是 RoleHelper：引擎对 Helper 恒判非实心，所以天然没有碰撞体积。
type 同伴 struct{}

func (同伴) Handle(unit.Context, unit.Event) {}

// 史莱姆 是活随从，但**不参与碰撞**：它常驻 Pass，和别的单位之间不推不弹，
// 墙还是照弹——所以它就当一段会飘的动画。它也不追人（照靶子那样沿初速直线飞、
// 撞墙弹开）。它的 Slot 是敌人的槽，于是：
//   - unit.Seek 只看得到勇者（不同槽且可瞄准），看不见敌人；
//   - 敌人也把它当同槽，打不到它、它也咬不到敌人。
//
// 只有勇者自己撞上来才挨咬。
type 史莱姆 struct {
	owner      uint64
	slot       int
	x, y       float64
	hitReadyAt float64
	booted     bool
}

func (s *史莱姆) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	sense, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !s.booted {
		s.booted = true
		// 常驻 Pass：没有碰撞体积，但墙照样挡它、它也不会飞出场地。
		ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
	}
	s.x, s.y = sense.Self.X, sense.Self.Y
	prey := unit.Seek(sense)
	if prey == nil {
		return
	}
	n := math.Hypot(prey.X-s.x, prey.Y-s.y)
	if n <= slimeRadius+prey.Radius+2 && sense.Time+1e-9 >= s.hitReadyAt {
		s.hitReadyAt = sense.Time + slimeHitCD
		ctx.Out <- unit.Damage{From: ctx.ID, To: prey.ID, Amount: slimeDamage}
	}
}

// 法师弹 是法师打出的三连发之一：撞到可打目标结算一次即消失，撞墙也消失。
type 法师弹 struct {
	owner uint64
	slot  int
	atk   float64
}

func (b *法师弹) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		b.slot = e.Self.Slot
		b.atk = markAtkOf(e.Self)
	case unit.Collision:
		if e.Other.ID == b.owner {
			return
		}
		if !unit.Hittable(e.Other, b.slot) {
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			return
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: mageShotDamage + b.atk}
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	case unit.WallHit:
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

// markAtkOf 从自己身上的标记读勇者当前攻击力。
func markAtkOf(self unit.Snapshot) float64 {
	for _, m := range self.Marks {
		if m.Kind == markAtk {
			return float64(m.Stacks)
		}
	}
	return 0
}
