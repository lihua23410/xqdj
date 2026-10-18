// 反重力猫咪：场心是质点，猫和子弹吃同一股引力。
// 切向速度够快，猫就像空间站，边掉边飞，停在轨道上。
// 子弹的切向速度不够，于是往场心掉，越掉越快。伤害按当时的速率算。
// 子弹碰到场心就消失。猫被减速以后，轨道也会往下掉。
package 反重力猫咪

import (
	"embed"
	"math"

	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const (
	KindHajimi       = "反重力猫咪"
	KindHajimiBullet = "反重力弹"
)

const (
	hajimiRadius = 14.0
	hajimiSpeed  = 178.0 // 半径 140 上的圆轨道速率
	hajimiHP     = 90.0
	hajimiVision = 9999.0
	hajimiColor  = "#8b7be8"

	orbitR      = 140.0
	gravSoft    = 48.0 // 场心附近把力收住，避免速率没有上限
	swallow     = 16.0 // 子弹离场心比这近，算撞上，消失
	fireGap     = 0.9
	bulletR     = 5.0
	bulletTang  = 0.28 // 相对当地圆轨道速率。故意不够，所以会掉下去
	bulletIn    = 0.50
	bulletColor = "#d9d4ff"
	dmgPerSpeed = 0.05
	dmgMin      = 1.0
	dmgMax      = 16.0
)

// gravG 让半径 orbitR 上的圆轨道速率等于 hajimiSpeed。
var gravG float64

func init() {
	a := hajimiSpeed * hajimiSpeed / orbitR
	gravG = a * math.Pow(orbitR*orbitR+gravSoft*gravSoft, 1.5) / orbitR

	p := unit.NewPack(KindHajimi, assets)
	p.Register(unit.Spec{
		Kind:    KindHajimi,
		Role:    unit.RoleFighter,
		Radius:  hajimiRadius,
		MaxHP:   hajimiHP,
		Speed:   hajimiSpeed,
		Vision:  hajimiVision,
		Fighter: true,
		Look:    unit.Look{Color: hajimiColor, Glow: true, FX: []string{"hajimi"}},
	}, func(unit.SpawnInfo) unit.Actor { return &哈基米{} })
	p.Register(unit.Spec{
		Kind: KindHajimiBullet, Role: unit.RoleProjectile,
		Radius: bulletR, MaxHP: 1, Speed: 0, Vision: 0,
		Look: unit.Look{Color: bulletColor, Glow: true, Trail: true, FX: []string{"hajimi-bullet"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &落体弹{owner: info.OwnerID, slot: info.Slot}
	})
}

type 哈基米 struct {
	booted bool
	next   float64
}

func (a *哈基米) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		unit.ConfirmHit(ctx, e)
		return
	case unit.Sense:
		a.onSense(ctx, e)
	}
}

func (a *哈基米) onSense(ctx unit.Context, s unit.Sense) {
	x, y := s.Self.X, s.Self.Y
	if !a.booted {
		a.booted = true
		a.next = s.Time + 0.35
		vx, vy := orbitalVelocity(x, y, 1)
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: vx, VY: vy}
		ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: math.Hypot(vx, vy)}
		ctx.Out <- unit.NoFrameFreeze{UnitID: ctx.ID, Hold: true}
	} else {
		// 速率被谁改掉就留在那个值上，巡航不把它拉回空间站速度，轨道才会往下掉。
		ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: math.Hypot(s.Self.VX, s.Self.VY)}
	}
	ax, ay := gravAccel(x, y)
	ctx.Out <- unit.Force{UnitID: ctx.ID, AX: ax, AY: ay}
	if s.Time >= a.next {
		a.next = s.Time + fireGap
		a.fire(ctx, s)
	}
}

func (a *哈基米) fire(ctx unit.Context, s unit.Sense) {
	x, y := s.Self.X, s.Self.Y
	r := math.Hypot(x, y)
	if r < 8 {
		return
	}
	tx, ty := -y/r, x/r
	if x*s.Self.VY-y*s.Self.VX < 0 {
		tx, ty = -tx, -ty
	}
	sp := circSpeed(r)
	vx := (tx*bulletTang - x/r*bulletIn) * sp
	vy := (ty*bulletTang - y/r*bulletIn) * sp
	n := math.Hypot(vx, vy)
	if n < 1 {
		return
	}
	ux, uy := vx/n, vy/n
	d := hajimiRadius + bulletR + 6
	ctx.Out <- unit.Spawn{
		Kind: KindHajimiBullet,
		X:    x + ux*d, Y: y + uy*d,
		VX: vx, VY: vy,
		OwnerID: ctx.ID, Slot: s.Self.Slot,
	}
	ctx.Out <- unit.FX{
		Name: "hajimi-fire", Kind: ctx.Kind, UnitID: ctx.ID, Slot: s.Self.Slot,
		X: s.Self.X, Y: s.Self.Y, VX: vx, VY: vy,
	}
}

type 落体弹 struct {
	owner uint64
	slot  int
	speed float64
	dead  bool
}

func (b *落体弹) Handle(ctx unit.Context, ev unit.Event) {
	if b.dead {
		return
	}
	switch e := ev.(type) {
	case unit.Sense:
		b.speed = math.Hypot(e.Self.VX, e.Self.VY)
		if math.Hypot(e.Self.X, e.Self.Y) < swallow {
			b.vanish(ctx)
			return
		}
		ax, ay := gravAccel(e.Self.X, e.Self.Y)
		ctx.Out <- unit.Force{UnitID: ctx.ID, AX: ax, AY: ay}
	case unit.Collision:
		if e.Other.Slot == b.slot {
			return
		}
		if e.Other.Role == unit.RoleFighter || e.Other.Mortal {
			ctx.Out <- unit.Damage{From: b.owner, To: e.Other.ID, Amount: bulletDamage(b.speed)}
		}
		b.vanish(ctx)
	}
}

func (b *落体弹) vanish(ctx unit.Context) {
	b.dead = true
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

func bulletDamage(speed float64) float64 {
	d := speed * dmgPerSpeed
	if d < dmgMin {
		return dmgMin
	}
	if d > dmgMax {
		return dmgMax
	}
	return d
}

func orbitalVelocity(x, y, dir float64) (float64, float64) {
	r := math.Hypot(x, y)
	if r < 8 {
		return 0, circSpeed(8)
	}
	tx, ty := -y/r*dir, x/r*dir
	sp := circSpeed(r)
	return tx * sp, ty * sp
}

func circSpeed(r float64) float64 {
	if r < 1 {
		r = 1
	}
	return math.Sqrt(gravMag(r) * r)
}

func gravMag(r float64) float64 {
	return gravG * r / math.Pow(r*r+gravSoft*gravSoft, 1.5)
}

func gravAccel(x, y float64) (float64, float64) {
	r := math.Hypot(x, y)
	if r < 1e-4 {
		return 0, 0
	}
	a := gravMag(r)
	return -x / r * a, -y / r * a
}
