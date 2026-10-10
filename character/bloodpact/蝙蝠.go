package 血契

import (
	"math"

	"xqdj/internal/unit"
)

// 血契蝠 烧 6 血召出的活随从（一次一只、场上至多 3 只）。追着最近的敌人咬：
// 贴身 3 伤、0.5 秒一口；12 秒寿终。它咬的伤只要打在血线连着的目标身上
//（战斗机或活随从）就按那条线的深度回流——来源不挑，所以这是在外勤把血抽回总部。
type 血契蝠 struct {
	owner uint64
	slot  int

	bornAt   float64
	nextBite float64
	nextSeek float64
	targetID uint64
	booted   bool
}

func (bat *血契蝠) Handle(ctx unit.Context, ev unit.Event) {
	// 活随从照常吃伤害（AcceptHit 处理报价）；弹体不抢它的事件。
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	// Pass：不与单位碰撞（蛆宝宝互吸同款）——巡航会把追击中的蝙蝠往目标
	// 身上顶，实心身体和引擎推挤顶牛会卡着重叠；穿过去由下嘴判定圈管出伤。
	// 出生那一拍就发令拿上。
	if !bat.booted {
		bat.booted = true
		bat.bornAt = s.Time
		bat.nextBite = s.Time
		bat.nextSeek = s.Time
		ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
		return
	}
	if s.Time-bat.bornAt >= batLife {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	bat.hunt(ctx, s)
}

// hunt 追最近的可打目标：贴身咬一口（同目标 0.5s CD），咬完朝下一个最近去。
func (bat *血契蝠) hunt(ctx unit.Context, s unit.Sense) {
	if s.Time+1e-9 >= bat.nextSeek {
		bat.nextSeek = s.Time + batSeekEvery
		bat.targetID = 0
	}

	var tgt *unit.Snapshot
	if bat.targetID != 0 {
		tgt = findNearby(s.Nearby, bat.targetID)
		if tgt != nil && !unit.Hittable(*tgt, bat.slot) {
			tgt = nil
		}
	}
	if tgt == nil {
		bestD := math.MaxFloat64
		for i := range s.Nearby {
			o := &s.Nearby[i]
			if !unit.Hittable(*o, bat.slot) {
				continue
			}
			d := math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y)
			if d < bestD {
				bestD, tgt = d, o
			}
		}
		if tgt != nil {
			bat.targetID = tgt.ID
		}
	}
	if tgt == nil {
		return
	}

	d := math.Hypot(tgt.X-s.Self.X, tgt.Y-s.Self.Y)
	ux, uy := unitDir(tgt.X-s.Self.X, tgt.Y-s.Self.Y)
	// 贴身判定按物理接触圈（自己 + 对方），别多给一圈：多一圈会把蝙蝠
	// 顶进对方身体里，触发引擎的重叠断言。
	if d <= s.Self.Radius+tgt.Radius {
		// 贴身：下嘴。
		if s.Time+1e-9 >= bat.nextBite {
			bat.nextBite = s.Time + batBiteCD
			ctx.Out <- unit.Damage{From: ctx.ID, To: tgt.ID, Amount: batDamage}
			ctx.Out <- unit.FX{
				Name: "blood-bat-bite", Kind: KindBloodBat, UnitID: ctx.ID, Slot: bat.slot,
				X: tgt.X, Y: tgt.Y,
			}
		}
		return
	}
	// 追：朝目标方向匀速改道（engine 的巡航带子会拉速率，方向由 SetVelocity 给）。
	ctx.Out <- unit.SetVelocity{
		UnitID: ctx.ID, VX: ux * batSpeed, VY: uy * batSpeed,
	}
}
