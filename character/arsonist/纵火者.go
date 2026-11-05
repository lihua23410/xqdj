// 纵火者自己不出伤。战斗状态围绕油瓶和火瓶：三瓶油、一瓶火，碎了铺油或起火。
package 纵火者

import (
	"embed"
	"math"

	"xqdj/internal/unit"
)

const (
	KindArsonist      = "纵火者"
	KindArsonOil      = "纵火者油瓶"
	KindArsonFire     = "纵火者火瓶"
	KindArsonOilMark  = "纵火者油落点"
	KindArsonFireMark = "纵火者火落点"
	KindArsonField    = "纵火者油火"
)

const (
	bodyRadius = 18.0
	bodyHP     = 100.0
	cruise     = 165.0
	vision     = 9999.0
	bodyColor  = "#1a120e"

	bottleRadius = 8.0
	bottleSpeed  = 300.0
	// 油瓶飞过的路径漏下一道细油，比单位拖尾窄得多，落进油滩就连成一摊。
	bottleTrailWidth = 8.0
	firstThrow       = 0.6
	throwGap         = 2.5
	trailCool        = 0.5
	oilRadius        = 54.0
	fireRadius       = 36.0
	trailLen         = 90.0
	// 拖尾起点比身体半径粗，沿长度收回到身体半径，免得从油滩里长出一根细管。
	trailHead = 1.9

	bottleFireLife = 1.0
	oilFireLife    = 2.0
	burnLife       = 1.0
	burnTick       = 0.2
	burnTickDmg    = 1.0
	spreadSpeed    = 80.0
	oilSlow        = 0.6 // 油滩和火滩里的敌人移动速度都减少 40%，不叠乘
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindArsonist, assets)
	p.Register(unit.Spec{
		Kind: KindArsonist, Role: unit.RoleFighter, Radius: bodyRadius, MaxHP: bodyHP,
		Speed: cruise, Vision: vision, Fighter: true,
		Look: unit.Look{Color: bodyColor, Glow: true, FX: []string{"arson"}},
	}, func(unit.SpawnInfo) unit.Actor { return &纵火者{} })
	p.Register(unit.Spec{
		Kind: KindArsonOil, Role: unit.RoleProjectile, Radius: bottleRadius, MaxHP: 1,
		Speed: bottleSpeed, Vision: 0, Fighter: false, Pass: true,
		Look: unit.Look{Color: "#3a2a18", Overlay: true, FX: []string{"arson-oil"}},
	}, newBottle(false))
	p.Register(unit.Spec{
		Kind: KindArsonFire, Role: unit.RoleProjectile, Radius: bottleRadius, MaxHP: 1,
		Speed: bottleSpeed, Vision: 0, Fighter: false, Pass: true,
		Look: unit.Look{Color: "#6a3018", Overlay: true, FX: []string{"arson-fire"}},
	}, newBottle(true))
	p.Register(unit.Spec{
		Kind: KindArsonOilMark, Role: unit.RoleHelper, Radius: 0, MaxHP: 1,
		Speed: 0, Vision: 0, Nonsolid: true,
	}, func(unit.SpawnInfo) unit.Actor { return marker{} })
	p.Register(unit.Spec{
		Kind: KindArsonFireMark, Role: unit.RoleHelper, Radius: 0, MaxHP: 1,
		Speed: 0, Vision: 0, Nonsolid: true,
	}, func(unit.SpawnInfo) unit.Actor { return marker{} })
	p.Register(unit.Spec{
		Kind: KindArsonField, Role: unit.RoleHelper, Radius: 0, MaxHP: 1,
		Speed: 0, Vision: vision, Nonsolid: true,
		Look: unit.Look{FX: []string{"arson-field"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &油火{owner: info.OwnerID, slot: info.Slot}
	})
}

type marker struct{}

func (marker) Handle(unit.Context, unit.Event) {}

type 纵火者 struct {
	booted bool
	step   int // 0、1、2 油瓶，3 火瓶
	hold   bool
	next   float64
}

func (a *纵火者) Handle(ctx unit.Context, ev unit.Event) {
	if d, ok := ev.(unit.IncomingDamage); ok {
		unit.ConfirmHit(ctx, d)
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	a.onSense(ctx, s)
}

func (a *纵火者) onSense(ctx unit.Context, s unit.Sense) {
	if !a.booted {
		a.booted = true
		a.step = 0
		a.next = s.Time + firstThrow
		ctx.Out <- ctx.Born(unit.Spawn{
			Kind: KindArsonField, OwnerID: ctx.ID, Slot: s.Self.Slot,
			X: s.Self.X, Y: s.Self.Y,
		})
	}
	if s.Time+1e-9 < a.next && !a.hold {
		return
	}
	t := unit.Seek(s)
	if t == nil {
		a.hold = true
		return
	}
	a.throw(ctx, s, t.X, t.Y)
	a.step++
	if a.step > 3 {
		a.step = 0
	}
	a.hold = false
	a.next = s.Time + throwGap
}

func (a *纵火者) throw(ctx unit.Context, s unit.Sense, tx, ty float64) {
	dx, dy := tx-s.Self.X, ty-s.Self.Y
	d := math.Hypot(dx, dy)
	if d < 1e-6 {
		dx, dy, d = 1, 0, 1
	}
	dx /= d
	dy /= d
	spawn := bodyRadius + bottleRadius
	if spawn > d-0.5 {
		spawn = math.Max(0, d-0.5)
	}
	kind := KindArsonFire
	if a.step < 3 {
		kind = KindArsonOil
	}
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: kind,
		X:    s.Self.X + dx*spawn, Y: s.Self.Y + dy*spawn,
		VX: dx * bottleSpeed, VY: dy * bottleSpeed,
		OwnerID: ctx.ID, Slot: s.Self.Slot,
		Data: bottleAim{X: tx, Y: ty, SX: s.Self.X + dx*spawn, SY: s.Self.Y + dy*spawn},
	})
}
