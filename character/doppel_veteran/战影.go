package 分身者_百战

import "xqdj/internal/unit"

type 战影 struct {
	owner uint64
	slot  int
}

func (sh *战影) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		sh.onDamage(ctx, e)
	case unit.WallHit:
		if j := getDash(ctx.ID); j != nil {
			j.finish(ctx, ctx.ID, e.Time, true, e.NX, e.NY)
		}
	case unit.Collision:
		if j := getDash(ctx.ID); j != nil {
			j.onCollision(ctx, ctx.ID, e)
		}
	}
}

func (sh *战影) onDamage(ctx unit.Context, d unit.IncomingDamage) {
	if isStackGuarded(ctx.ID) {
		unit.BlockHit(ctx, d)
		return
	}
	if j := getDash(ctx.ID); j != nil && j.blocking() {
		unit.BlockHit(ctx, d)
		return
	}
	// 折损改由本体 syncShadowDeaths 在消失时结算，避免 Sense/Confirm 血量不同步漏折
	unit.ConfirmHit(ctx, d)
	clearDash(ctx.ID)
}
