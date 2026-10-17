// 粉红色迷人的pink：三个形态（平淡/大笑/狂笑）靠击杀场上不断刷新的人群升级。
// 每次击杀有 25% 概率掉落装备，装备会覆盖当前形态的默认攻击方式。
package pink

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

//go:embed *.png fx
var assets embed.FS

const KindPink = "粉红色迷人的pink"

const (
	KindCrowd = "人群"

	KindBottle  = "瓶"
	KindPistol  = "手枪"
	KindShotgun = "霰弹枪"
	KindRifle   = "步枪"
	KindFuel    = "油箱"
	KindGatling = "加特林"
	KindRocket  = "火箭筒"

	KindBottleShot  = "Pink瓶"
	KindPistolShot  = "Pink手枪弹"
	KindShotgunShot = "Pink霰弹"
	KindRifleShot   = "Pink步枪弹"
	KindGatlingShot = "Pink加特林弹"
	KindRocketShot  = "Pink火箭弹"
	KindFire        = "粉红火焰"
)

const (
	formPlain = iota
	formLaugh
	formManic
)

const (
	weaponDefault = iota
	weaponBottle
	weaponPistol
	weaponShotgun
	weaponRifle
	weaponFuel
	weaponGatling
	weaponRocket
)

const (
	pinkRadius = 18.0
	pinkHP     = 100.0
	pinkSpeed  = 155.0
	pinkVision = 9999.0
	pinkColor  = "#ff6fae"

	crowdRadius = 14.0
	crowdHP     = 6.0
	crowdSpeed  = 155.0
	crowdSpawn  = 3.5
	crowdMax    = 6

	itemRadius = 12.0

	killPerForm     = 2
	crowdKillHeal   = 3.0
	crowdKillInvuln = 0.5
	dropChance      = 0.50

	meleeCD    = 0.7
	meleeRange = 78.0
	meleeHalf  = 72.0
	axeDamage  = 6.0
	sawHit     = 3.0
	sawHits    = 3

	bottleCD           = 4.0
	bottleShots        = 3
	bottleSpeed        = 440.0
	bottleImpactDamage = 5.0

	pistolCD    = 0.5
	pistolShots = 20
	pistolSpeed = 560.0
	pistolDmg   = 5.0

	shotgunCD      = 2.0
	shotgunPellets = 6
	shotgunUses    = 7
	shotgunSpeed   = 620.0
	shotgunDmg     = 5.0
	shotgunLife    = 0.34

	rifleCD    = 2.0
	rifleShots = 7
	rifleSpeed = 920.0
	rifleDmg   = 10.0

	fuelDashes = 3
	fuelSpeed  = 690.0
	fuelTime   = 0.32
	fuelCD     = 1.15
	fuelRange  = 260.0
	fuelHitR   = 48.0

	gatlingCD    = 0.2
	gatlingSpeed = 390.0
	gatlingDmg   = 4.0
	gatlingMag   = 60
	reloadTime   = 6.0

	rocketCD    = 6.0
	rocketShots = 5
	rocketSpeed = 430.0
	rocketDmg   = 20.0
	rocketR     = 120.0

	fireRadius = 108.0
	fireLife   = 5.0
	fireGap    = 0.5
	fireDamage = 4.0
	fireSlow   = 0.60
	bottleStun = 1.2

	// 两件需要自己清理的速度附件各用独立 token。
	meleeFSToken = uint64(0x50494e4b01)
	fuelFSToken  = uint64(0x50494e4b02)
)

func init() {
	p := unit.NewPack(KindPink, assets)
	p.Register(unit.Spec{
		Kind: KindPink, Role: unit.RoleFighter, Radius: pinkRadius, MaxHP: pinkHP,
		Speed: pinkSpeed, Vision: pinkVision, Fighter: true,
		Look: unit.Look{Color: pinkColor, FX: []string{"pink"}},
	}, func(unit.SpawnInfo) unit.Actor { return &pink{} })

	p.Register(unit.Spec{
		Kind: KindCrowd, Role: unit.RoleMinion, Radius: crowdRadius, MaxHP: crowdHP,
		Speed: crowdSpeed, Vision: pinkVision, Mortal: true, Cruise: true,
		Look: unit.Look{Color: "#f8f8f8", FX: []string{"pink-crowd"}},
	}, func(info unit.SpawnInfo) unit.Actor { return &crowd{owner: info.OwnerID} })

	for _, kind := range []string{
		KindBottle, KindPistol, KindShotgun, KindRifle, KindFuel, KindGatling, KindRocket,
	} {
		kind := kind
		p.Register(unit.Spec{
			Kind: kind, Role: unit.RoleHelper, Radius: itemRadius, MaxHP: 1,
			Speed: 0, Vision: 0, Nonsolid: true,
			Look: unit.Look{Color: "#ffd4e7", FX: []string{"pink-item"}},
		}, func(info unit.SpawnInfo) unit.Actor {
			return &item{slot: info.Slot, kind: kind}
		})
	}

	p.Register(unit.Spec{
		Kind: KindBottleShot, Role: unit.RoleProjectile, Radius: 7, MaxHP: 1,
		Speed: bottleSpeed, Look: unit.Look{Color: "#ffb36b", Trail: true, FX: []string{"pink-bottle-shot"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &bottleShot{owner: info.OwnerID, slot: info.Slot}
	})

	p.Register(unit.Spec{
		Kind: KindFire, Role: unit.RoleHelper, Radius: fireRadius, MaxHP: 1,
		Speed: 0, Vision: pinkVision, Nonsolid: true,
		Look: unit.Look{Color: "#ff713d", FX: []string{"pink-fire"}},
	}, func(unit.SpawnInfo) unit.Actor { return &fireArea{} })

	registerBullet := func(kind string, radius, speed, damage, life float64, fx string) {
		p.Register(unit.Spec{
			Kind: kind, Role: unit.RoleProjectile, Radius: radius, MaxHP: 1, Speed: speed,
			Look: unit.Look{Color: "#ff8fbe", Trail: true, FX: []string{fx}},
		}, func(info unit.SpawnInfo) unit.Actor {
			return &bullet{
				owner: info.OwnerID, slot: info.Slot, kind: kind,
				damage: damage, life: life,
			}
		})
	}
	registerBullet(KindPistolShot, 4, pistolSpeed, pistolDmg, 0, "pink-shot")
	registerBullet(KindShotgunShot, 3, shotgunSpeed, shotgunDmg, shotgunLife, "pink-shot")
	registerBullet(KindRifleShot, 4, rifleSpeed, rifleDmg, 0, "pink-shot")
	registerBullet(KindGatlingShot, 4, gatlingSpeed, gatlingDmg, 0, "pink-shot")
	registerBullet(KindRocketShot, 10, rocketSpeed, rocketDmg, 0, "pink-rocket")
}

type pink struct {
	slot   int
	booted bool
	rng    *rand.Rand

	form   int
	weapon int
	ammo   int
	kills  int

	nextAttackAt float64
	nextCrowdAt  float64
	nextStateAt  float64
	lastBumpAt   float64

	reloading  bool
	reloadAt   float64
	reloads    int
	reloadTick float64

	fuelUntil   float64
	fuelDrop    bool
	fuelHits    map[uint64]bool
	invulnUntil float64

	x, y   float64
	now    float64
	hx, hy float64
	seen   map[uint64]unit.Snapshot
}

func (p *pink) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		if e.Time < p.invulnUntil {
			unit.BlockHit(ctx, e)
			return
		}
		unit.ConfirmHit(ctx, e)
	case unit.Kill:
		if e.VictimKind == KindCrowd {
			p.onCrowdKill(ctx, e)
		}
	case unit.Collision:
		p.onCollision(ctx, e)
	case unit.Sense:
		p.tick(ctx, e)
	}
}

func (p *pink) onCollision(ctx unit.Context, e unit.Collision) {
	// 平淡形态的默认攻击就是碰撞；装备武器后改用装备攻击。
	if p.form != formPlain || p.weapon != weaponDefault {
		return
	}
	if e.Time-p.lastBumpAt < 0.18 || !unit.EnemyTarget(e, p.slot) {
		return
	}
	p.lastBumpAt = e.Time
	ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: 6}
	ctx.Out <- unit.FX{
		Name: "bump", Kind: KindPink, UnitID: ctx.ID, Slot: p.slot,
		X: e.Other.X, Y: e.Other.Y, VX: p.hx, VY: p.hy,
	}
}

func (p *pink) onCrowdKill(ctx unit.Context, e unit.Kill) {
	p.kills++
	ctx.Out <- unit.Heal{UnitID: ctx.ID, Amount: crowdKillHeal}
	if until := p.now + crowdKillInvuln; until > p.invulnUntil {
		p.invulnUntil = until
	}
	if p.kills%killPerForm == 0 && p.form < formManic {
		p.form++
		p.clearWeapon()
		p.emitState(ctx)
		ctx.Out <- unit.FX{
			Name: "form-up", Kind: KindPink, UnitID: ctx.ID, Slot: p.slot,
			X: p.x, Y: p.y, Amount: float64(p.form),
		}
	}
	if p.rng == nil || p.rng.Float64() >= dropChance {
		return
	}
	x, y := p.x, p.y
	if pos, ok := p.seen[e.VictimID]; ok {
		x, y = pos.X, pos.Y
	}
	kind := p.randomDrop()
	ctx.Out <- unit.Spawn{Kind: kind, X: x, Y: y, OwnerID: ctx.ID, Slot: p.slot}
	ctx.Out <- unit.FX{
		Name: "drop", Kind: KindPink, UnitID: ctx.ID, Slot: p.slot,
		X: x, Y: y, Amount: float64(weaponOf(kind)),
	}
}

func (p *pink) randomDrop() string {
	pool := []string{KindBottle, KindPistol}
	if p.form >= formLaugh {
		pool = append(pool, KindShotgun, KindRifle)
	}
	if p.form >= formManic {
		pool = append(pool, KindFuel, KindGatling, KindRocket)
	}
	return pool[p.rng.IntN(len(pool))]
}

func (p *pink) tick(ctx unit.Context, s unit.Sense) {
	p.now = s.Time
	p.x, p.y = s.Self.X, s.Self.Y
	p.slot = s.Self.Slot

	seen := make(map[uint64]unit.Snapshot, len(s.Nearby))
	for i := range s.Nearby {
		seen[s.Nearby[i].ID] = s.Nearby[i]
	}
	p.seen = seen
	if speed := math.Hypot(s.Self.VX, s.Self.VY); speed > 1e-6 {
		p.hx, p.hy = s.Self.VX/speed, s.Self.VY/speed
	}

	if !p.booted {
		p.booted = true
		if p.rng == nil {
			p.rng = unit.NewRNG(ctx.ID)
		}
		p.nextCrowdAt = s.Time
		p.nextAttackAt = s.Time
		p.nextStateAt = s.Time + 0.5
		p.emitState(ctx)
	}

	p.tickCrowd(ctx, s)
	p.tryPickup(ctx, s)
	p.tickReload(ctx, s)
	if p.fuelUntil > 0 {
		p.tickFuel(ctx, s)
	}
	p.tickWeapon(ctx, s)
	if s.Time+1e-9 >= p.nextStateAt {
		p.nextStateAt = s.Time + 0.5
		p.emitState(ctx)
	}
}

func (p *pink) tickCrowd(ctx unit.Context, s unit.Sense) {
	alive := 0
	for i := range s.Nearby {
		if s.Nearby[i].Kind == KindCrowd {
			alive++
		}
	}
	if alive >= crowdMax || s.Time+1e-9 < p.nextCrowdAt {
		return
	}
	x, y, ok := s.Field.RandomWalkable(p.rng, crowdRadius+6)
	if !ok {
		return
	}
	if math.Hypot(x-p.x, y-p.y) < pinkRadius+crowdRadius+18 {
		return
	}
	p.nextCrowdAt = s.Time + crowdSpawn
	ang := p.rng.Float64() * 2 * math.Pi
	// 人群放到敌方槽：pink 能打，对手不能替 pink 打。
	ctx.Out <- unit.Spawn{
		Kind: KindCrowd, OwnerID: ctx.ID, Slot: 1 - p.slot, X: x, Y: y,
		VX: math.Cos(ang) * crowdSpeed, VY: math.Sin(ang) * crowdSpeed,
	}
	ctx.Out <- unit.FX{
		Name: "crowd-spawn", Kind: KindPink, UnitID: ctx.ID, Slot: p.slot,
		X: x, Y: y,
	}
}

func (p *pink) tryPickup(ctx unit.Context, s unit.Sense) {
	weapons := map[string]int{
		KindBottle: weaponBottle, KindPistol: weaponPistol, KindShotgun: weaponShotgun,
		KindRifle: weaponRifle, KindFuel: weaponFuel, KindGatling: weaponGatling,
		KindRocket: weaponRocket,
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		w, ok := weapons[o.Kind]
		if !ok || o.OwnerID != s.Self.ID {
			continue
		}
		if math.Hypot(o.X-p.x, o.Y-p.y) > pinkRadius+o.Radius+10 {
			continue
		}
		p.equip(ctx, w, s.Time)
		ctx.Out <- unit.Despawn{UnitID: o.ID}
		return
	}
}

func (p *pink) equip(ctx unit.Context, weapon int, now float64) {
	p.weapon = weapon
	p.ammo = weaponAmmo(weapon)
	p.nextAttackAt = now
	p.reloading = false
	p.reloads = 0
	p.reloadAt = 0
	p.fuelUntil = 0
	p.fuelDrop = false
	p.fuelHits = nil
	p.emitState(ctx)
}

func weaponOf(kind string) int {
	switch kind {
	case KindBottle:
		return weaponBottle
	case KindPistol:
		return weaponPistol
	case KindShotgun:
		return weaponShotgun
	case KindRifle:
		return weaponRifle
	case KindFuel:
		return weaponFuel
	case KindGatling:
		return weaponGatling
	case KindRocket:
		return weaponRocket
	}
	return weaponDefault
}

func weaponAmmo(weapon int) int {
	switch weapon {
	case weaponBottle:
		return bottleShots
	case weaponPistol:
		return pistolShots
	case weaponShotgun:
		return shotgunUses
	case weaponRifle:
		return rifleShots
	case weaponFuel:
		return fuelDashes
	case weaponGatling:
		return gatlingMag
	case weaponRocket:
		return rocketShots
	}
	return 0
}

func (p *pink) tickReload(ctx unit.Context, s unit.Sense) {
	if !p.reloading {
		return
	}
	if s.Time+1e-9 >= p.reloadAt {
		p.reloading = false
		p.ammo = gatlingMag
		p.nextAttackAt = s.Time
		p.emitState(ctx)
		ctx.Out <- unit.FX{
			Name: "reload-done", Kind: KindPink, UnitID: ctx.ID, Slot: p.slot,
			X: p.x, Y: p.y,
		}
		return
	}
	if s.Time+1e-9 >= p.reloadTick {
		p.reloadTick = s.Time + 0.1
		ctx.Out <- unit.FX{
			Name: "reload", Kind: KindPink, UnitID: ctx.ID, Slot: p.slot,
			X: p.x, Y: p.y, Amount: p.reloadAt - s.Time,
		}
	}
}

func (p *pink) tickWeapon(ctx unit.Context, s unit.Sense) {
	if p.reloading || (p.fuelUntil > 0 && s.Time < p.fuelUntil) || s.Time+1e-9 < p.nextAttackAt {
		return
	}
	target, ok := p.nearestEnemy(s)
	if !ok {
		return
	}
	p.aim(target.X, target.Y)

	switch p.weapon {
	case weaponBottle:
		p.fireProjectile(ctx, KindBottleShot, target, bottleSpeed, 11)
		p.spend(ctx, bottleCD, s.Time)
	case weaponPistol:
		p.fireProjectile(ctx, KindPistolShot, target, pistolSpeed, 8)
		p.spend(ctx, pistolCD, s.Time)
	case weaponShotgun:
		base := math.Atan2(target.Y-p.y, target.X-p.x)
		for i := 0; i < shotgunPellets; i++ {
			offset := (float64(i) - float64(shotgunPellets-1)/2) * unit.Deg(6)
			p.spawnBullet(ctx, KindShotgunShot, base+offset, shotgunSpeed)
		}
		p.spend(ctx, shotgunCD, s.Time)
		ctx.Out <- unit.FX{
			Name: "shotgun", Kind: KindPink, UnitID: ctx.ID, Slot: p.slot,
			X: p.x, Y: p.y, VX: p.hx, VY: p.hy,
		}
	case weaponRifle:
		p.fireProjectile(ctx, KindRifleShot, target, rifleSpeed, 8)
		p.spend(ctx, rifleCD, s.Time)
	case weaponGatling:
		p.fireProjectile(ctx, KindGatlingShot, target, gatlingSpeed, 8)
		p.ammo--
		if p.ammo <= 0 {
			if p.reloads == 0 {
				p.beginReload(ctx, s.Time)
			} else {
				p.dropWeapon(ctx)
			}
		} else {
			p.nextAttackAt = s.Time + gatlingCD
			p.emitAmmo(ctx)
		}
	case weaponRocket:
		p.fireProjectile(ctx, KindRocketShot, target, rocketSpeed, 18)
		p.spend(ctx, rocketCD, s.Time)
	case weaponFuel:
		if math.Hypot(target.X-p.x, target.Y-p.y) <= fuelRange {
			p.startFuelDash(ctx, s.Time)
		}
	default:
		if p.form >= formLaugh {
			p.melee(ctx, s, s.Time)
		}
	}
}

func (p *pink) melee(ctx unit.Context, s unit.Sense, now float64) {
	damage := axeDamage
	hits := 1
	if p.form == formManic {
		damage = sawHit
		hits = sawHits
	}
	hit := false
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, p.slot) || !inFan(p.x, p.y, p.hx, p.hy, *o, meleeRange, meleeHalf) {
			continue
		}
		for n := 0; n < hits; n++ {
			ctx.Out <- unit.Damage{
				From: ctx.ID, To: o.ID, Amount: damage,
				NoFreeze: hits > 1,
			}
		}
		hit = true
	}
	if p.form == formLaugh {
		ctx.Out <- unit.AddFS{
			UnitID: ctx.ID, Token: meleeFSToken, DX: p.hx, DY: p.hy,
			BaseSpeed: 255, OnWall: true, ExpiresAt: now + 0.15,
		}
	}
	p.nextAttackAt = now + meleeCD
	ctx.Out <- unit.FX{
		Name: "melee", Kind: KindPink, UnitID: ctx.ID, Slot: p.slot,
		X: p.x, Y: p.y, VX: p.hx, VY: p.hy,
		Amount: float64(p.form*10 + boolInt(hit)),
	}
}

func (p *pink) startFuelDash(ctx unit.Context, now float64) {
	p.fuelHits = make(map[uint64]bool)
	p.fuelUntil = now + fuelTime
	p.invulnUntil = now + fuelTime + 0.05
	p.ammo--
	p.fuelDrop = p.ammo <= 0
	p.nextAttackAt = now + fuelCD
	ctx.Out <- unit.AddFS{
		UnitID: ctx.ID, Token: fuelFSToken, DX: p.hx, DY: p.hy,
		BaseSpeed: fuelSpeed, OnWall: true, ExpiresAt: p.fuelUntil,
	}
	ctx.Out <- unit.FX{
		Name: "fuel", Kind: KindPink, UnitID: ctx.ID, Slot: p.slot,
		X: p.x, Y: p.y, VX: p.hx, VY: p.hy, Amount: float64(p.ammo),
	}
	p.emitAmmo(ctx)
	p.fuelStrike(ctx)
}

func (p *pink) tickFuel(ctx unit.Context, s unit.Sense) {
	if s.Time+1e-9 >= p.fuelUntil {
		p.fuelUntil = 0
		ctx.Out <- unit.RemoveFS{UnitID: ctx.ID, Token: fuelFSToken}
		if p.fuelDrop {
			p.dropWeapon(ctx)
		}
		return
	}
	p.fuelStrike(ctx)
}

func (p *pink) fuelStrike(ctx unit.Context) {
	if p.fuelHits == nil {
		p.fuelHits = make(map[uint64]bool)
	}
	for id, o := range p.seen {
		if p.fuelHits[id] || !unit.Hittable(o, p.slot) {
			continue
		}
		if math.Hypot(o.X-p.x, o.Y-p.y) > fuelHitR+o.Radius {
			continue
		}
		p.fuelHits[id] = true
		for n := 0; n < sawHits; n++ {
			ctx.Out <- unit.Damage{
				From: ctx.ID, To: id, Amount: sawHit,
				NoFreeze: true,
			}
		}
		ctx.Out <- unit.FX{
			Name: "fuel-hit", Kind: KindPink, UnitID: ctx.ID, Slot: p.slot,
			X: o.X, Y: o.Y, VX: p.hx, VY: p.hy,
		}
	}
}

func (p *pink) beginReload(ctx unit.Context, now float64) {
	p.reloading = true
	p.reloadAt = now + reloadTime
	p.reloadTick = now
	p.reloads++
	p.ammo = 0
	p.nextAttackAt = p.reloadAt
	ctx.Out <- unit.FX{
		Name: "reload-start", Kind: KindPink, UnitID: ctx.ID, Slot: p.slot,
		X: p.x, Y: p.y, Amount: reloadTime,
	}
	p.emitAmmo(ctx)
}

func (p *pink) fireProjectile(ctx unit.Context, kind string, target unit.Snapshot, speed, radius float64) {
	dx, dy := target.X-p.x, target.Y-p.y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	ux, uy := dx/n, dy/n
	gap := pinkRadius + radius + 2
	ctx.Out <- unit.Spawn{
		Kind: kind, X: p.x + ux*gap, Y: p.y + uy*gap,
		VX: ux * speed, VY: uy * speed, OwnerID: ctx.ID, Slot: p.slot,
	}
	ctx.Out <- unit.FX{
		Name: "shoot", Kind: KindPink, UnitID: ctx.ID, Slot: p.slot,
		X: p.x, Y: p.y, VX: ux, VY: uy,
	}
}

func (p *pink) spawnBullet(ctx unit.Context, kind string, angle, speed float64) {
	ux, uy := math.Cos(angle), math.Sin(angle)
	ctx.Out <- unit.Spawn{
		Kind: kind, X: p.x + ux*(pinkRadius+8), Y: p.y + uy*(pinkRadius+8),
		VX: ux * speed, VY: uy * speed, OwnerID: ctx.ID, Slot: p.slot,
	}
}

func (p *pink) spend(ctx unit.Context, cooldown, now float64) {
	p.nextAttackAt = now + cooldown
	if p.ammo > 0 {
		p.ammo--
	}
	if p.ammo <= 0 {
		p.dropWeapon(ctx)
		return
	}
	p.emitAmmo(ctx)
}

func (p *pink) dropWeapon(ctx unit.Context) {
	p.clearWeapon()
	p.emitState(ctx)
}

func (p *pink) clearWeapon() {
	p.weapon = weaponDefault
	p.ammo = 0
	p.reloading = false
	p.reloadAt = 0
	p.reloads = 0
	p.fuelUntil = 0
	p.fuelDrop = false
	p.fuelHits = nil
}

func (p *pink) nearestEnemy(s unit.Sense) (unit.Snapshot, bool) {
	var best unit.Snapshot
	bestDistance := math.MaxFloat64
	found := false
	for i := range s.Nearby {
		o := s.Nearby[i]
		if !unit.Aimable(o, p.slot) {
			continue
		}
		distance := math.Hypot(o.X-p.x, o.Y-p.y)
		if distance < bestDistance {
			bestDistance, best, found = distance, o, true
		}
	}
	return best, found
}

func (p *pink) aim(x, y float64) {
	dx, dy := x-p.x, y-p.y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	p.hx, p.hy = dx/n, dy/n
}

func (p *pink) emitState(ctx unit.Context) {
	ctx.Out <- unit.FX{Name: "form", Kind: KindPink, UnitID: ctx.ID, Slot: p.slot, X: p.x, Y: p.y, Amount: float64(p.form)}
	ctx.Out <- unit.FX{Name: "weapon", Kind: KindPink, UnitID: ctx.ID, Slot: p.slot, X: p.x, Y: p.y, Amount: float64(p.weapon)}
	p.emitAmmo(ctx)
}

func (p *pink) emitAmmo(ctx unit.Context) {
	ctx.Out <- unit.FX{Name: "ammo", Kind: KindPink, UnitID: ctx.ID, Slot: p.slot, X: p.x, Y: p.y, Amount: float64(p.ammo)}
}

func inFan(x, y, ux, uy float64, o unit.Snapshot, reach, halfDeg float64) bool {
	dx, dy := o.X-x, o.Y-y
	distance := math.Hypot(dx, dy)
	if distance > reach+o.Radius {
		return false
	}
	if distance < 1e-6 {
		return true
	}
	facing := math.Hypot(ux, uy)
	if facing < 1e-6 {
		return true
	}
	dot := (dx*ux + dy*uy) / (distance * facing)
	if dot > 1 {
		dot = 1
	} else if dot < -1 {
		dot = -1
	}
	return math.Acos(dot) <= unit.Deg(halfDeg)
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
