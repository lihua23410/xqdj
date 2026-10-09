package 血契

import (
	"math"

	"xqdj/internal/unit"
)

// 血契弧 主人（RearmAttach）挂在身前的薄扇环。碰到敌方战斗机或敌方活随从
// 结 7、在被咬的那位身上闪一口，就地散架——建血线的判定在主人那一侧
// （弧散那一拍眼前有战斗机就建线，锤击同款口径）。
type 血契弧 struct {
	slot int
}

func (a *血契弧) Handle(ctx unit.Context, ev unit.Event) {
	e, ok := ev.(unit.Collision)
	if !ok || !unit.EnemyTarget(e, a.slot) {
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: biteDamage}
	ctx.Out <- unit.FX{
		Name: "blood-bite", Kind: ctx.Kind, UnitID: ctx.ID, Slot: a.slot,
		X: e.Other.X, Y: e.Other.Y,
	}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

// findNearby 在感知里按 id 找人（血线目标、索敌目标都要用）。
func findNearby(units []unit.Snapshot, id uint64) *unit.Snapshot {
	for i := range units {
		if units[i].ID == id {
			return &units[i]
		}
	}
	return nil
}

func unitDir(dx, dy float64) (float64, float64) {
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return 1, 0
	}
	return dx / n, dy / n
}
