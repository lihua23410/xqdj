package 太刀

import (
	"embed"
	"math"

	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const KindTachi = "太刀"

const (
	tachiRadius = 18.0
	tachiSpeed  = 140.0
	tachiHP     = 100.0
	tachiVision = 9999.0
	tachiColor  = "#dfe8ff"

	// 刃色（从低到高）：无 -> 白 -> 黄 -> 红。
	bladeNone   = 0
	bladeWhite  = 1
	bladeYellow = 2
	bladeRed    = 3

	// 纳刀节奏：每 5 秒进入一次纳刀，窗口 0.8 秒。
	sheathePeriod = 5.0
	sheatheWindow = 0.8

	// 居合：原地瞬间拔刀斩，不突进。
	iaidoReach = 125.0
	iaidoHalf  = 42.0 // 前方扇区半张角（度），整片约 84 度
	iaidoInvuln = 0.20

	// 常态基础挥砍：武器挥舞扫到敌人出伤，不升级刃色。
	normalSwingPeriod = 0.55
	swingReach        = 92.0
	swingHalf         = 58.0
	swingDmg          = 2.0

	// 登龙 + 解放斩（红刃居合命中派生）。
	helmSpeed  = 620.0
	helmTime   = 0.55
	helmMain   = 16.0
	burstHits  = 6
	burstDmg   = 5.0
	burstGap   = 0.14
	burstTail  = 0.10
	spinSpeed  = 480.0
	spinTime   = 0.22
	spinReach  = 125.0
	spinHalf   = 90.0 // 大回旋 180 度：半张角 90 度

	// 见切：间隔小于 1 秒的两次及以上攻击触发。
	foresightGap    = 0.5
	foresightInvuln = 1.0
	foresightSpinAt = 0.25
	retreatDist     = 90.0
	retreatPad      = tachiRadius + 4
)

const (
	phaseNormal = iota
	phaseSheathe
	phaseHelm
	phaseForesight
)

func init() {
	p := unit.NewPack(KindTachi, assets)
	p.Register(unit.Spec{
		Kind:    KindTachi,
		Role:    unit.RoleFighter,
		Radius:  tachiRadius,
		MaxHP:   tachiHP,
		Speed:   tachiSpeed,
		Vision:  tachiVision,
		Fighter: true,
		Look:    unit.Look{Color: tachiColor, Ghost: 260, FX: []string{"tachi"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &太刀{nextSheatheAt: sheathePeriod, lastHitAt: -foresightGap, hx: 1}
	})
}

type 太刀 struct {
	slot int

	level int
	phase int
	until float64

	nextSheatheAt float64
	invulnUntil   float64
	lastHitAt     float64

	x, y   float64
	hx, hy float64
	seen   map[uint64]unit.Snapshot

	lockID uint64
	lockX  float64
	lockY  float64

	burstIdx    int
	burstNextAt float64
	spinDone    bool
	spinAt      float64
	booted      bool
	swingAt     float64
}

func (t *太刀) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		t.onIncoming(ctx, e)
	case unit.Sense:
		t.tick(ctx, e)
	}
}

func (t *太刀) onIncoming(ctx unit.Context, d unit.IncomingDamage) {
	if d.Time < t.invulnUntil {
		unit.BlockHit(ctx, d)
		return
	}
	// 间隔小于 2 秒的第二次及以上攻击：见切（无敌 + 后撤 + 大回旋）。
	if d.Time-t.lastHitAt >= 0 && d.Time-t.lastHitAt < foresightGap {
		unit.BlockHit(ctx, d)
		t.startForesight(ctx, d)
		return
	}
	t.lastHitAt = d.Time
	if t.phase == phaseSheathe {
		unit.BlockHit(ctx, d)
		t.startIaido(ctx, d)
		return
	}
	unit.ConfirmHit(ctx, d)
}

func (t *太刀) tick(ctx unit.Context, s unit.Sense) {
	t.remember(s)
	if !t.booted {
		t.booted = true
		t.emitBlade(ctx)
		t.emitSheathe(ctx)
	}
	switch t.phase {
	case phaseNormal:
		if s.Time+1e-9 >= t.nextSheatheAt {
			t.phase = phaseSheathe
			t.until = s.Time + sheatheWindow
			t.nextSheatheAt = s.Time + sheathePeriod
			ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
			ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
			t.emitSheathe(ctx)
		} else {
			t.trySwing(ctx, s)
		}
	case phaseSheathe:
		if s.Time+1e-9 >= t.until {
			t.phase = phaseNormal
			ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}
			t.emitSheathe(ctx)
		}
	case phaseHelm:
		t.tickHelm(ctx, s)
	case phaseForesight:
		t.tickForesight(ctx, s)
	}
}

func (t *太刀) remember(s unit.Sense) {
	t.x, t.y = s.Self.X, s.Self.Y
	t.slot = s.Self.Slot
	if sp := math.Hypot(s.Self.VX, s.Self.VY); sp > 1e-6 {
		t.hx, t.hy = s.Self.VX/sp, s.Self.VY/sp
	}
	t.seen = make(map[uint64]unit.Snapshot, len(s.Nearby))
	for _, o := range s.Nearby {
		t.seen[o.ID] = o
	}
}

func (t *太刀) startIaido(ctx unit.Context, d unit.IncomingDamage) {
	// 拔刀：原地瞬间斩击，不突进。纳刀结束，恢复移动。
	t.phase = phaseNormal
	t.invulnUntil = d.Time + iaidoInvuln
	t.emitSheathe(ctx)
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}

	o, ok := t.attacker(d.From)
	tx, ty := t.x+t.hx*iaidoReach, t.y+t.hy*iaidoReach
	if ok {
		tx, ty = o.X, o.Y
		t.lockID = o.ID
		t.lockX, t.lockY = o.X, o.Y
	} else {
		t.lockID = 0
	}
	t.aim(tx, ty)

	hit := t.slashCone(ctx, t.x, t.y, t.hx, t.hy, iaidoReach, iaidoHalf, t.iaidoDamage())
	t.fx(ctx, "iaido", t.x, t.y, float64(t.level))

	if hit {
		if t.level == bladeRed {
			t.startHelm(ctx, d.Time, tx, ty)
			return
		}
		t.levelUp(ctx)
	}
}

// trySwing 是常态基础挥砍：武器扫到前方敌人就出伤，不升级刃色。
func (t *太刀) trySwing(ctx unit.Context, s unit.Sense) {
	if s.Time+1e-9 < t.swingAt {
		return
	}
	// 随机巡航，敌人落入前方扇形才挥砍，不主动追敌。
	if !t.slashCone(ctx, t.x, t.y, t.hx, t.hy, swingReach, swingHalf, swingDmg) {
		return
	}
	t.fx(ctx, "swing", t.x, t.y, float64(t.level))
	t.swingAt = s.Time + normalSwingPeriod
}

func (t *太刀) startHelm(ctx unit.Context, now, tx, ty float64) {
	t.phase = phaseHelm
	t.until = now + helmTime
	t.invulnUntil = now + helmTime + float64(burstHits)*burstGap + burstTail
	t.lockX, t.lockY = tx, ty
	t.burstIdx = -1
	t.aim(tx, ty)
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
	t.fx(ctx, "thrust", t.x, t.y, float64(t.level))
}

func (t *太刀) tickHelm(ctx unit.Context, s unit.Sense) {
	if t.burstIdx < 0 {
		if o, ok := t.seen[t.lockID]; ok {
			t.lockX, t.lockY = o.X, o.Y
		}
		t.aim(t.lockX, t.lockY)
		t.dash(ctx, s.Time, t.hx, t.hy, helmSpeed, t.until-s.Time)
		if s.Time+1e-9 >= t.until {
			t.burstIdx = 0
			t.burstNextAt = s.Time
			ctx.Out <- unit.RemoveFS{UnitID: ctx.ID, Token: 1}
			ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: false}
			ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
			ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
			t.dealBurst(ctx, helmMain)
			t.fx(ctx, "helm", t.x, t.y, float64(t.level))
			t.until = s.Time + float64(burstHits)*burstGap + burstTail
		}
		return
	}
	for s.Time+1e-9 >= t.burstNextAt && t.burstIdx < burstHits {
		t.dealBurst(ctx, burstDmg)
		t.fx(ctx, "burst", t.lockX, t.lockY, float64(t.burstIdx))
		t.burstIdx++
		t.burstNextAt = s.Time + burstGap
	}
	if t.burstIdx >= burstHits && s.Time+1e-9 >= t.until {
		t.level = bladeYellow
		t.emitBlade(ctx)
		ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}
		t.phase = phaseNormal
	}
}

func (t *太刀) startForesight(ctx unit.Context, d unit.IncomingDamage) {
	t.phase = phaseForesight
	t.until = d.Time + foresightInvuln
	t.invulnUntil = d.Time + foresightInvuln
	t.spinDone = false
	t.spinAt = d.Time + foresightSpinAt
	t.emitSheathe(ctx)

	o, ok := t.attacker(d.From)
	tx, ty := t.x+t.hx, t.y+t.hy
	if ok {
		t.lockID = o.ID
		t.lockX, t.lockY = o.X, o.Y
		tx, ty = o.X, o.Y
	}
	// 后撤：背对目标方向。
	dx, dy := t.x-tx, t.y-ty
	if math.Hypot(dx, dy) < 1e-6 {
		dx, dy = -t.hx, -t.hy
	}
	n := math.Hypot(dx, dy)
	if n > 1e-6 {
		dx, dy = dx/n, dy/n
	}
	rx, ry := t.x+dx*retreatDist, t.y+dy*retreatDist
	rx, ry = unit.LiveField().Clamp(rx, ry, retreatPad)
	ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: rx, Y: ry}
	t.x, t.y = rx, ry
	t.fx(ctx, "foresight", rx, ry, 0)
}

func (t *太刀) tickForesight(ctx unit.Context, s unit.Sense) {
	if !t.spinDone && s.Time+1e-9 >= t.spinAt {
		t.spinDone = true
		t.doSightSpin(ctx, s)
	}
	if s.Time+1e-9 >= t.until {
		t.phase = phaseNormal
	}
}

func (t *太刀) doSightSpin(ctx unit.Context, s unit.Sense) {
	if o, ok := t.seen[t.lockID]; ok {
		t.lockX, t.lockY = o.X, o.Y
	}
	t.aim(t.lockX, t.lockY)
	_ = t.slashCone(ctx, t.x, t.y, t.hx, t.hy, spinReach, spinHalf, t.spinDamage())
	t.levelUp(ctx)
	t.dash(ctx, s.Time, t.hx, t.hy, spinSpeed, spinTime)
	t.fx(ctx, "spin", t.x, t.y, float64(t.level))
}

func (t *太刀) dash(ctx unit.Context, now, dx, dy, speed, dur float64) {
	if dur < 1e-6 {
		dur = 0.05
	}
	ctx.Out <- unit.AddFS{
		UnitID:    ctx.ID,
		Token:     1,
		DX:        dx,
		DY:        dy,
		BaseSpeed: speed,
		OnWall:    true,
		ExpiresAt: now + dur,
	}
}

func (t *太刀) slashCone(ctx unit.Context, x, y, ux, uy, reach, half, dmg float64) bool {
	hit := false
	for _, o := range t.seen {
		if !unit.Hittable(o, t.slot) || !inCone(x, y, ux, uy, o, reach, half) {
			continue
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: dmg}
		hit = true
	}
	return hit
}

func (t *太刀) dealBurst(ctx unit.Context, amount float64) {
	if o, ok := t.seen[t.lockID]; ok && unit.Hittable(o, t.slot) {
		ctx.Out <- unit.Damage{From: ctx.ID, To: t.lockID, Amount: amount}
		return
	}
	if o, ok := t.nearestEnemy(); ok {
		ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: amount}
	}
}

func (t *太刀) attacker(from uint64) (unit.Snapshot, bool) {
	if o, ok := t.seen[from]; ok && o.OwnerID != 0 {
		if p, ok2 := t.seen[o.OwnerID]; ok2 {
			return p, true
		}
	}
	if o, ok := t.seen[from]; ok && unit.Hittable(o, t.slot) {
		return o, true
	}
	return t.nearestEnemy()
}

func (t *太刀) nearestEnemy() (unit.Snapshot, bool) {
	best := unit.Snapshot{}
	bestD := math.MaxFloat64
	found := false
	for _, o := range t.seen {
		if !unit.Hittable(o, t.slot) {
			continue
		}
		d := math.Hypot(o.X-t.x, o.Y-t.y)
		if d < bestD {
			bestD, best, found = d, o, true
		}
	}
	return best, found
}

func (t *太刀) aim(tx, ty float64) {
	dx, dy := tx-t.x, ty-t.y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	t.hx, t.hy = dx/n, dy/n
}

func (t *太刀) levelUp(ctx unit.Context) {
	if t.level >= bladeRed {
		return
	}
	t.level++
	t.emitBlade(ctx)
	t.fx(ctx, "levelup", t.x, t.y, float64(t.level))
}

func (t *太刀) iaidoDamage() float64 {
	return []float64{6, 9, 12, 16}[t.level]
}

func (t *太刀) spinDamage() float64 {
	return []float64{8, 11, 14, 16}[t.level]
}

func (t *太刀) emitBlade(ctx unit.Context) {
	ctx.Out <- unit.FX{Name: "blade", UnitID: ctx.ID, Kind: ctx.Kind, X: t.x, Y: t.y, Slot: t.slot, Amount: float64(t.level)}
}

func (t *太刀) emitSheathe(ctx unit.Context) {
	v := 0.0
	if t.phase == phaseSheathe {
		v = 1
	}
	ctx.Out <- unit.FX{Name: "sheathe", UnitID: ctx.ID, Kind: ctx.Kind, X: t.x, Y: t.y, Slot: t.slot, Amount: v}
}

func (t *太刀) fx(ctx unit.Context, name string, x, y, amount float64) {
	ctx.Out <- unit.FX{Name: name, UnitID: ctx.ID, Kind: ctx.Kind, X: x, Y: y, VX: t.hx, VY: t.hy, Slot: t.slot, Amount: amount}
}

func inCone(x, y, ux, uy float64, o unit.Snapshot, reach, halfDeg float64) bool {
	dx, dy := o.X-x, o.Y-y
	d := math.Hypot(dx, dy)
	if d > reach+o.Radius {
		return false
	}
	if d < 1e-6 {
		return true
	}
	fn := math.Hypot(ux, uy)
	if fn < 1e-6 {
		return true
	}
	dot := (dx*ux + dy*uy) / (d * fn)
	if dot > 1 {
		dot = 1
	} else if dot < -1 {
		dot = -1
	}
	return math.Acos(dot) <= unit.Deg(halfDeg)
}
