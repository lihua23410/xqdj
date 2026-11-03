package 藤姬

import (
	"math"

	"xqdj/internal/unit"
)

// 藤弧：整圈贴在身上的扇环，打中一发就缩回去，隔 arcRearm 再挂。
// 伤害只有 arcDamage，剩下的全靠藤姬写在这只身上的固定加伤（SetDamageBonus）。
type 藤弧 struct {
	slot int
}

func (a *藤弧) Handle(ctx unit.Context, ev unit.Event) {
	e, ok := ev.(unit.Collision)
	if !ok || !unit.EnemyTarget(e, a.slot) {
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: arcDamage}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

// 幻役：藤色的容器，0.1 血，穿人（Pass）。
// 它像颗台球：出生时朝最近的敌人甩出去，之后直线飞、撞墙直接反弹，不追人。
// 贴到人就打一下，同一个目标隔 dollHitCD 才能再挨；它自己没有任何基础伤害，
// 那一下的点数就是身上藤色的 +X（由出伤入口的 SetDamageBonus 补上），没藤色就不出伤。
type 幻役 struct {
	x, y float64
	slot int
	hit  map[uint64]float64 // 每个目标的下一次可命中时刻
}

func (d *幻役) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		unit.ConfirmHit(ctx, e)
	case unit.Sense:
		d.tick(ctx, e)
	}
}

func (d *幻役) tick(ctx unit.Context, s unit.Sense) {
	d.x, d.y, d.slot = s.Self.X, s.Self.Y, s.Self.Slot
	if d.hit == nil {
		d.hit = map[uint64]float64{}
	}
	if markStacks(s.Self, toseKind) <= 0 {
		return // 没有藤色就没有伤害
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, d.slot) {
			continue
		}
		if math.Hypot(o.X-d.x, o.Y-d.y) > s.Self.Radius+o.Radius {
			continue
		}
		if s.Time+1e-9 < d.hit[o.ID] {
			continue
		}
		d.hit[o.ID] = s.Time + dollHitCD
		ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: dollDamage}
		ctx.Out <- unit.FX{
			Name: "fujihime-doll-hit", Kind: KindFujihime, UnitID: ctx.ID, Slot: d.slot,
			X: o.X, Y: o.Y,
		}
	}
}

// dollDamage 是幻役的基础伤害：0，全部靠藤色的 +X。
const dollDamage = 0.0

// markStacks 读快照上某种标记的层数。
func markStacks(snap unit.Snapshot, kind string) int {
	for _, m := range snap.Marks {
		if m.Kind == kind {
			return m.Stacks
		}
	}
	return 0
}
