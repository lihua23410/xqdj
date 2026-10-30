// 反重力猫咪：场心是质点，猫和子弹吃同一股引力。
// 子弹切向速度不够，往场心掉，越掉越快，碰到场心消失。伤害按速率算。
// 猫每次受到这股引力，都顺飞行方向再补一点点能量，轨道不设上限。比当地逃逸还快就不再补。
package 反重力猫咪

import (
	"embed"
	"math"
	"math/rand/v2"

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
	fireGap     = 1.1
	bulletR     = 5.0
	bulletTang  = 0.28 // 相对当地圆轨道速率。故意不够，所以会掉下去
	bulletIn    = 0.50
	bulletColor = "#d9d4ff"
	dmgPerSpeed = 0.025
	dmgMin      = 1.0
	dmgMax      = 8.0
	antiGain    = 0.04 // 受到引力时，顺速度再补的比例。只补一点点势能

	gravityFieldPeriod   = 20.0
	gravityFieldDuration = 3.0
	gravityFieldRadius   = 420.0
	gravityFieldMinSpeed = 220.0
	orbitBulletMinR      = 72.0
	orbitBulletMaxR      = 160.0
	orbitBulletSpin      = 1.8
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
	booted     bool
	next       float64
	rng        *rand.Rand
	nextField  float64
	fieldUntil float64
	fieldX     float64
	fieldY     float64
	fieldSeen  map[uint64]bool
	fieldLast  float64
	orbiters   map[uint64]*orbitBullet
}

type orbitBullet struct {
	angle   float64
	radius  float64
	angular float64
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
		a.nextField = s.Time + gravityFieldPeriod
		vx, vy := orbitalVelocity(x, y, 1)
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: vx, VY: vy}
		ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: math.Hypot(vx, vy)}
		ctx.Out <- unit.NoFrameFreeze{UnitID: ctx.ID, Hold: true}
		ctx.Out <- unit.FX{
			Name: "hajimi-meow", Kind: KindHajimi, UnitID: ctx.ID, Slot: s.Self.Slot,
			X: x, Y: y,
		}
	} else {
		// 速率被谁改掉就留在那个值上，巡航不把它拉回空间站速度，轨道才会往下掉。
		ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: math.Hypot(s.Self.VX, s.Self.VY)}
	}
	ax, ay := catPull(x, y, s.Self.VX, s.Self.VY)
	ctx.Out <- unit.Force{UnitID: ctx.ID, AX: ax, AY: ay}
	if s.Time >= a.next {
		a.next = s.Time + fireGap
		a.fire(ctx, s)
	}
	if a.nextField <= 0 {
		a.nextField = s.Time + gravityFieldPeriod
	}
	if s.Time+1e-9 >= a.nextField {
		a.nextField = s.Time + gravityFieldPeriod
		a.startGravityField(ctx, s)
	}
	if a.fieldUntil > 0 {
		if s.Time+1e-9 < a.fieldUntil {
			a.applyGravityField(ctx, s)
		} else {
			a.releaseOrbiters(ctx, s)
			a.fieldUntil = 0
			a.fieldSeen = nil
			a.orbiters = nil
		}
	}
}

func (a *哈基米) startGravityField(ctx unit.Context, s unit.Sense) {
	if a.rng == nil {
		a.rng = unit.NewRNG(ctx.Stream)
	}
	a.fieldUntil = s.Time + gravityFieldDuration
	a.fieldX, a.fieldY = s.Self.X, s.Self.Y
	a.fieldSeen = make(map[uint64]bool)
	a.fieldLast = s.Time
	a.orbiters = make(map[uint64]*orbitBullet)
	ctx.Out <- unit.FX{
		Name: "gravity-field", Kind: ctx.Kind, UnitID: ctx.ID, Slot: s.Self.Slot,
		X: a.fieldX, Y: a.fieldY, VX: gravityFieldDuration, Amount: gravityFieldRadius,
	}
}

func (a *哈基米) applyGravityField(ctx unit.Context, s unit.Sense) {
	if a.rng == nil {
		a.rng = unit.NewRNG(ctx.Stream)
	}
	if a.orbiters == nil {
		a.orbiters = make(map[uint64]*orbitBullet)
	}
	dt := s.Time - a.fieldLast
	if dt <= 0 || dt > 0.2 {
		dt = 1.0 / 60.0
	}
	a.fieldLast = s.Time
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role != unit.RoleProjectile && o.Role != unit.RoleClone {
			continue
		}
		// 近战弧跟着人走，不是飞过来的子弹。吹笛人的老鼠偷东西也是这么排除的。
		if o.ArcSpan != 0 {
			continue
		}
		if math.Hypot(o.X-a.fieldX, o.Y-a.fieldY) > gravityFieldRadius+o.Radius {
			continue
		}
		if o.Role == unit.RoleProjectile && o.Kind == KindHajimiBullet && o.OwnerID == ctx.ID {
			a.orbitOwnBullet(ctx, s, o, dt)
			continue
		}
		if a.fieldSeen[o.ID] {
			continue
		}
		a.fieldSeen[o.ID] = true
		if o.Role == unit.RoleProjectile && o.Slot != s.Self.Slot {
			ctx.Out <- ctx.Born(unit.Spawn{
				Kind: KindHajimiBullet, X: o.X, Y: o.Y, VX: o.VX, VY: o.VY,
				OwnerID: ctx.ID, Slot: s.Self.Slot,
			})
			ctx.Out <- unit.Despawn{UnitID: o.ID}
			ctx.Out <- unit.FX{
				Name: "gravity-convert", Kind: ctx.Kind, UnitID: ctx.ID, Slot: s.Self.Slot,
				X: o.X, Y: o.Y, VX: o.VX, VY: o.VY,
			}
			continue
		}
		speed := math.Hypot(o.VX, o.VY)
		if speed < 1e-6 {
			speed = gravityFieldMinSpeed
		}
		angle := a.rng.Float64() * 2 * math.Pi
		ctx.Out <- unit.SetVelocity{
			UnitID: o.ID, VX: math.Cos(angle) * speed, VY: math.Sin(angle) * speed,
		}
		ctx.Out <- unit.FX{
			Name: "gravity-twist", Kind: ctx.Kind, UnitID: ctx.ID, Slot: s.Self.Slot,
			X: o.X, Y: o.Y, VX: math.Cos(angle), VY: math.Sin(angle), Amount: speed,
		}
	}
}

func (a *哈基米) orbitOwnBullet(ctx unit.Context, s unit.Sense, o *unit.Snapshot, dt float64) {
	orb := a.orbiters[o.ID]
	if orb == nil {
		radius := math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y)
		if radius < orbitBulletMinR {
			radius = orbitBulletMinR
		} else if radius > orbitBulletMaxR {
			radius = orbitBulletMaxR
		}
		angle := math.Atan2(o.Y-s.Self.Y, o.X-s.Self.X)
		if math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) < 1e-6 {
			angle = a.rng.Float64() * 2 * math.Pi
		}
		spin := orbitBulletSpin
		if a.rng.IntN(2) == 0 {
			spin = -spin
		}
		orb = &orbitBullet{angle: angle, radius: radius, angular: spin}
		a.orbiters[o.ID] = orb
		ctx.Out <- unit.FX{
			Name: "gravity-capture", Kind: ctx.Kind, UnitID: ctx.ID, Slot: s.Self.Slot,
			X: o.X, Y: o.Y, VX: o.VX, VY: o.VY,
		}
	}
	orb.angle += orb.angular * dt
	tx := s.Self.X + math.Cos(orb.angle)*orb.radius
	ty := s.Self.Y + math.Sin(orb.angle)*orb.radius
	if s.Field.Shape != "" {
		tx, ty = s.Field.Clamp(tx, ty, o.Radius)
	}
	ctx.Out <- unit.Teleport{UnitID: o.ID, X: tx, Y: ty}
	tangent := orb.angle + math.Copysign(math.Pi/2, orb.angular)
	speed := circSpeed(orb.radius)
	ctx.Out <- unit.SetVelocity{
		UnitID: o.ID, VX: math.Cos(tangent) * speed, VY: math.Sin(tangent) * speed,
	}
}

func (a *哈基米) releaseOrbiters(ctx unit.Context, s unit.Sense) {
	for id, orb := range a.orbiters {
		speed := gravityReleaseSpeed(orb.radius)
		ctx.Out <- unit.SetVelocity{
			UnitID: id, VX: math.Cos(orb.angle) * speed, VY: math.Sin(orb.angle) * speed,
		}
		x := s.Self.X + math.Cos(orb.angle)*orb.radius
		y := s.Self.Y + math.Sin(orb.angle)*orb.radius
		ctx.Out <- unit.FX{
			Name: "gravity-release", Kind: ctx.Kind, UnitID: ctx.ID, Slot: s.Self.Slot,
			X: x, Y: y, VX: math.Cos(orb.angle), VY: math.Sin(orb.angle), Amount: speed,
		}
	}
}

// gravityReleaseSpeed 把轨道半径上的重力势能转成释放速度：v = sqrt(2*g(r)*r)。
func gravityReleaseSpeed(r float64) float64 {
	if r < 1 {
		r = 1
	}
	v := math.Sqrt(2 * gravMag(r) * r)
	if v < gravityFieldMinSpeed {
		return gravityFieldMinSpeed
	}
	if v > 420 {
		return 420
	}
	return v
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
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: KindHajimiBullet,
		X:    x + ux*d, Y: y + uy*d,
		VX: vx, VY: vy,
		OwnerID: ctx.ID, Slot: s.Self.Slot,
	})
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

// catPull 是猫受到的引力，再顺着当前速度补一点点。
// 不看半径停手。比当地逃逸还快就只留引力，避免速率没有上限。
func catPull(x, y, vx, vy float64) (float64, float64) {
	ax, ay := gravAccel(x, y)
	if escaped(x, y, vx, vy) {
		return ax, ay
	}
	sp := math.Hypot(vx, vy)
	mag := gravMag(math.Hypot(x, y)) * antiGain
	if sp > 20 {
		return ax + vx/sp*mag, ay + vy/sp*mag
	}
	r := math.Hypot(x, y)
	if r < 1 {
		return ax, ay
	}
	return ax + x/r*mag, ay + y/r*mag
}

func escaped(x, y, vx, vy float64) bool {
	r2 := x*x + y*y
	need := 2 * gravG / math.Sqrt(r2+gravSoft*gravSoft)
	return vx*vx+vy*vy >= need
}
