// 闪烁预判飞弹，以侧闪接三连智能弹；平时低频点射。
package 闪烁

import (
	"embed"
	"encoding/json"
	"math"
	"math/rand/v2"
	"xqdj/internal/unit"
)

const KindFlicker = "闪烁"
const (
	bodyRadius       = 18.0
	bodyHP           = 100.0
	cruise           = 170.0
	sight            = 400.0
	anticipation     = 0.15
	sideSpeed        = 720.0
	sideDist         = 80.0
	reflexCooldown   = 1.4
	focusDuration    = 0.35
	burstGap         = 0.1
	burstCount       = 3
	beamLen          = 280.0
	beamWidth        = 8.0
	beamDamage       = 3.0
	perfectDamage    = 5.0
	shotDamage       = 2.0
	shotGap          = 0.9
	projectileKeep   = 0.35
	nonMissileReject = 0.25
	trailLife        = 0.25
	trailSample      = 0.012
	bodyColor        = "#b7fff1"
	bloodBullet      = "外科手术血液子弹"
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindFlicker, assets)
	registerBullets(p)
	p.Register(unit.Spec{Kind: KindFlicker, Role: unit.RoleFighter, Radius: bodyRadius,
		MaxHP: bodyHP, Speed: cruise, Vision: sight, Fighter: true,
		Look: unit.Look{Color: bodyColor, FX: []string{"flicker"}},
	}, func(unit.SpawnInfo) unit.Actor { return &闪烁{} })
}

type phase uint8

const (
	phIdle phase = iota
	phSide
	phFocus
)

type trailPoint struct{ X, Y, T float64 }

// 完整且有界的观众态，寿命使用对局时间，暂停和导出共用。
type visualState struct {
	Phase                                  phase
	ReadyAt, SideAt, SideUntil, FocusUntil float64
	AimX, AimY                             float64
	Aim                                    bool
	ThreatX, ThreatY                       float64
	Trail                                  []trailPoint
}
type 闪烁 struct {
	phase                                                   phase
	dirX, dirY, exitX, exitY                                float64
	sideAt, sideUntil, focusUntil, readyAt, shotAt, burstAt float64
	burstIndex                                              int
	sourceID                                                uint64
	perfect, wantCruise, passing, rearmSound, pending       bool
	pendingOwner, pendingFrom                               uint64
	lastX, lastY, lastT, lastHP                             float64
	haveLast                                                bool
	rng                                                     *rand.Rand
	visual                                                  visualState
}

func (a *闪烁) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		if a.phase == phSide && e.Time < a.sideUntil && missileDamage(e) {
			ctx.Out <- unit.ConfirmDamage{Token: e.Token, UnitID: ctx.ID, Amount: e.Amount * projectileKeep}
			return
		}
		if !missileDamage(e) && a.reflexReady(e.Time) && a.roll(ctx) < nonMissileReject {
			a.readyAt = e.Time + reflexCooldown
			a.pending, a.rearmSound = true, true
			a.pendingOwner, a.pendingFrom = e.SourceOwnerID, e.From
			unit.BlockHit(ctx, e)
			return
		}
		ctx.Out <- unit.ConfirmDamage{Token: e.Token, UnitID: ctx.ID, Amount: e.Amount}
	case unit.Stunned:
		a.pending = false
		a.pendingOwner, a.pendingFrom = 0, 0
		if a.phase == phSide {
			a.finishSide(ctx, e.Time)
		}
		// 眩晕取消余下连射，醒来不补积压的枪；冷却继续走。
		if a.phase == phFocus {
			a.phase = phIdle
		}
		a.visual.Phase = a.phase
		a.visual.FocusUntil, a.visual.SideUntil = e.Time, e.Time
		a.emitVisual(ctx)
	case unit.WallHit:
		if a.phase == phSide || a.wantCruise {
			a.exitX, a.exitY = reflect(a.dirX, a.dirY, e.NX, e.NY)
			a.finishSide(ctx, e.Time)
			a.wantCruise = true
		}
	case unit.Sense:
		a.onSense(ctx, e)
	}
}

func (a *闪烁) onSense(ctx unit.Context, s unit.Sense) {
	if s.Self.HP <= 0 {
		a.pending = false
		a.pendingOwner, a.pendingFrom = 0, 0
		return
	}
	if a.haveLast && s.Self.HP < a.lastHP-1e-9 {
		a.perfect = false
	}
	jumped := a.haveLast && math.Hypot(s.Self.X-a.lastX, s.Self.Y-a.lastY) > math.Hypot(s.Self.VX, s.Self.VY)*math.Max(1.0/60, s.Time-a.lastT)+48
	yield := s.Knock || (!jumped && a.external(s))
	if a.pending {
		a.launchPending(ctx, s, yield)
	}
	if a.phase == phSide {
		a.sampleTrail(s)
		if s.Time+1e-9 >= a.sideUntil {
			a.finishSide(ctx, a.sideUntil)
		} else if !yield {
			a.push(ctx, a.dirX, a.dirY, sideSpeed)
		}
	}
	if a.wantCruise && a.phase != phSide && !yield {
		a.push(ctx, a.exitX, a.exitY, cruise)
		ctx.Out <- unit.SetFSDirection{UnitID: ctx.ID, VX: a.exitX, VY: a.exitY}
		a.wantCruise = false
	}
	if a.phase == phFocus && s.Time > a.focusUntil+1e-9 {
		a.phase = phIdle
	}
	if a.phase == phIdle && s.Time+1e-9 >= a.readyAt && !yield {
		if a.rearmSound {
			a.sound(ctx, s, "armed")
			a.rearmSound = false
		}
		if bullet := imminent(s); bullet != nil {
			a.begin(ctx, s, *bullet)
		}
	}
	target := a.target(s, a.phase != phIdle)
	a.visual.Aim = target != nil
	if target != nil {
		a.visual.AimX, a.visual.AimY = target.X, target.Y
	}
	if a.phase == phFocus && a.burstIndex < burstCount && s.Time+1e-9 >= a.burstAt {
		// 每拍最多一枪，错过窗口不集中补发；没有目标也消耗本次机会。
		boost := a.perfect && a.burstIndex == burstCount-1
		if target != nil {
			damage := beamDamage
			if boost {
				damage = perfectDamage
			}
			a.fire(ctx, s, *target, damage, boost)
		}
		a.burstIndex++
		a.burstAt = s.Time + burstGap
		a.shotAt = s.Time + shotGap
	} else if a.phase == phIdle && target != nil && s.Time+1e-9 >= a.shotAt {
		a.fire(ctx, s, *target, shotDamage, false)
		a.shotAt = s.Time + shotGap
	}
	a.paint(ctx, s)
	a.lastX, a.lastY, a.lastT, a.lastHP = s.Self.X, s.Self.Y, s.Time, s.Self.HP
	a.haveLast = true
}

func (a *闪烁) target(s unit.Sense, counter bool) *unit.Snapshot {
	inRange := func(o unit.Snapshot) bool { return math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) <= beamLen }
	if counter && a.sourceID != 0 {
		if o := unit.SeekIf(s, func(o unit.Snapshot) bool { return o.ID == a.sourceID && inRange(o) }); o != nil {
			return o
		}
	}
	return unit.SeekIf(s, inRange)
}

func missileDamage(e unit.IncomingDamage) bool {
	return e.SourceRole == unit.RoleProjectile && e.ArcSpan == 0 && e.SourceKind != bloodBullet
}

func (a *闪烁) reflexReady(t float64) bool {
	return a.phase == phIdle && !a.pending && t+1e-9 >= a.readyAt
}

func (a *闪烁) roll(ctx unit.Context) float64 {
	if a.rng == nil {
		a.rng = unit.NewRNG(ctx.Stream)
	}
	return a.rng.Float64()
}

func seekID(s unit.Sense, id uint64) *unit.Snapshot {
	if id == 0 {
		return nil
	}
	return unit.SeekIf(s, func(o unit.Snapshot) bool { return o.ID == id })
}

func enemyBullet(o, self unit.Snapshot) bool {
	return o.Role == unit.RoleProjectile && o.ArcSpan == 0 && o.Slot != self.Slot && o.Kind != bloodBullet && math.Hypot(o.VX, o.VY) >= 1
}

// 最早接触本体的时间；擦身而过或远离的弹不触发。
func contactTime(self, bullet unit.Snapshot, vx, vy, horizon float64) (float64, bool) {
	x, y := bullet.X-self.X, bullet.Y-self.Y
	dx, dy := bullet.VX-vx, bullet.VY-vy
	r := self.Radius + bullet.Radius + 2
	dot := x*dx + y*dy
	if dot >= 0 {
		return 0, false
	}
	c := x*x + y*y - r*r
	if c <= 0 {
		return 0, true
	}
	v2 := dx*dx + dy*dy
	disc := dot*dot - v2*c
	if v2 < 1e-9 || disc < 0 {
		return 0, false
	}
	t := (-dot - math.Sqrt(disc)) / v2
	return t, t >= 0 && t <= horizon
}

func imminent(s unit.Sense) *unit.Snapshot {
	var best *unit.Snapshot
	bestT := math.Inf(1)
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !enemyBullet(*o, s.Self) {
			continue
		}
		t, hit := contactTime(s.Self, *o, s.Self.VX, s.Self.VY, anticipation)
		if !hit {
			continue
		}
		sp := math.Hypot(o.VX, o.VY)
		if !o.PassWalls && clipReach(o.X, o.Y, o.VX/sp, o.VY/sp, sp*t, o.Radius, s.Field, s.Walls)+1e-6 < sp*t {
			continue
		}
		if best == nil || t < bestT-1e-9 || (math.Abs(t-bestT) < 1e-9 && o.ID < best.ID) {
			best, bestT = o, t
		}
	}
	return best
}

func sideScore(s unit.Sense, dx, dy float64) (float64, float64) {
	clear := clipReach(s.Self.X, s.Self.Y, dx, dy, sideDist, s.Self.Radius, s.Field, s.Walls)
	score := clear * 2
	duration := clear / sideSpeed
	end := s.Self
	end.X += dx * clear
	end.Y += dy * clear
	for _, o := range s.Nearby {
		if !enemyBullet(o, s.Self) {
			continue
		}
		if _, hit := contactTime(s.Self, o, dx*sideSpeed, dy*sideSpeed, duration); hit {
			score -= 1000
		}
		o.X += o.VX * duration
		o.Y += o.VY * duration
		gap := math.Hypot(o.X-end.X, o.Y-end.Y) - o.Radius - end.Radius
		score += math.Min(0, gap-24) * 3
	}
	return score, clear
}

func pickSide(s unit.Sense, ax, ay float64) (dx, dy, clear float64) {
	sp := math.Hypot(ax, ay)
	if sp < 1e-9 {
		ax, ay, sp = 1, 0, 1
	}
	px, py := -ay/sp, ax/sp
	left, lc := sideScore(s, px, py)
	right, rc := sideScore(s, -px, -py)
	if right > left {
		return -px, -py, rc
	}
	return px, py, lc
}

func (a *闪烁) begin(ctx unit.Context, s unit.Sense, bullet unit.Snapshot) {
	dx, dy, clear := pickSide(s, bullet.VX, bullet.VY)
	if clear < 10 {
		return
	}
	a.readyAt = s.Time + reflexCooldown
	a.startSlide(ctx, s, dx, dy, bullet.OwnerID, bullet.X, bullet.Y, true)
}

func (a *闪烁) launchPending(ctx unit.Context, s unit.Sense, yield bool) {
	owner, from := a.pendingOwner, a.pendingFrom
	a.pending = false
	a.pendingOwner, a.pendingFrom = 0, 0
	ax, ay, threatX, threatY := pendingAxis(s, owner, from)
	dx, dy, clear := pickSide(s, ax, ay)
	source := owner
	if source == 0 {
		source = from
	}
	if clear < 10 {
		a.openFocus(ctx, s, source)
		return
	}
	a.startSlide(ctx, s, dx, dy, source, threatX, threatY, !yield)
}

func pendingAxis(s unit.Sense, owner, from uint64) (ax, ay, threatX, threatY float64) {
	if owner != 0 {
		if o := seekID(s, owner); o != nil {
			return o.X - s.Self.X, o.Y - s.Self.Y, o.X, o.Y
		}
	} else if o := seekID(s, from); o != nil {
		return o.X - s.Self.X, o.Y - s.Self.Y, o.X, o.Y
	}
	vx, vy := s.Self.VX, s.Self.VY
	if math.Hypot(vx, vy) < 1e-6 {
		vx, vy = 1, 0
	}
	return vx, vy, s.Self.X + vx, s.Self.Y + vy
}

func (a *闪烁) startSlide(ctx unit.Context, s unit.Sense, dx, dy float64, source uint64, threatX, threatY float64, push bool) {
	a.phase, a.perfect = phSide, true
	a.dirX, a.dirY, a.exitX, a.exitY = dx, dy, dx, dy
	a.sideAt, a.sideUntil = s.Time, s.Time+sideDist/sideSpeed
	a.sourceID, a.wantCruise, a.rearmSound = source, false, true
	a.visual.ThreatX, a.visual.ThreatY = threatX, threatY
	a.visual.Trail = a.visual.Trail[:0]
	a.sampleTrail(s)
	a.setPass(ctx, true)
	if push {
		a.push(ctx, dx, dy, sideSpeed)
	}
	a.sound(ctx, s, "reflex")
}

func (a *闪烁) openFocus(ctx unit.Context, s unit.Sense, source uint64) {
	a.phase, a.perfect = phFocus, true
	a.sourceID, a.wantCruise, a.rearmSound = source, false, true
	a.sideAt, a.sideUntil = s.Time, s.Time
	a.focusUntil, a.burstAt, a.burstIndex = s.Time+focusDuration, s.Time+0.04, 0
	a.sound(ctx, s, "reflex")
}

func (a *闪烁) finishSide(ctx unit.Context, t float64) {
	if a.phase != phSide {
		return
	}
	a.phase = phFocus
	a.sideUntil = t
	a.focusUntil, a.burstAt, a.burstIndex = t+focusDuration, t+0.04, 0
	a.wantCruise = true
	a.setPass(ctx, false)
}

func (a *闪烁) fire(ctx unit.Context, s unit.Sense, target unit.Snapshot, damage float64, boost bool) {
	dx, dy := target.X-s.Self.X, target.Y-s.Self.Y
	d := math.Hypot(dx, dy)
	if d < 1e-9 {
		dx, dy, d = 1, 0, 1
	}
	dx, dy = dx/d, dy/d
	kind := kindBullet
	smart := a.phase == phFocus
	if smart {
		kind = kindSmart
	}
	if boost {
		kind = kindBoost
	}
	ctx.Out <- ctx.Born(unit.Spawn{Kind: kind, OwnerID: ctx.ID, Slot: s.Self.Slot,
		X: s.Self.X, Y: s.Self.Y, VX: dx * bulletSpeed, VY: dy * bulletSpeed,
		Data: bulletData{Target: target.ID, Smart: smart, Damage: damage, Boost: boost, Until: s.Time + 1.0}})
	name := "shot"
	if boost {
		name = "overdrive"
	}
	a.sound(ctx, s, name)
}

func (a *闪烁) sound(ctx unit.Context, s unit.Sense, name string) {
	ctx.Out <- unit.FX{Name: name, Kind: ctx.Kind, UnitID: ctx.ID, Slot: s.Self.Slot, X: s.Self.X, Y: s.Self.Y}
}
func (a *闪烁) sampleTrail(s unit.Sense) {
	n := len(a.visual.Trail)
	if n == 0 || s.Time-a.visual.Trail[n-1].T >= trailSample {
		a.visual.Trail = append(a.visual.Trail, trailPoint{X: s.Self.X, Y: s.Self.Y, T: s.Time})
	}
}
func (a *闪烁) paint(ctx unit.Context, s unit.Sense) {
	trail := a.visual.Trail[:0]
	for _, p := range a.visual.Trail {
		if s.Time-p.T <= trailLife {
			trail = append(trail, p)
		}
	}
	a.visual.Trail = trail
	a.visual.Phase, a.visual.ReadyAt = a.phase, a.readyAt
	a.visual.SideAt, a.visual.SideUntil, a.visual.FocusUntil = a.sideAt, a.sideUntil, a.focusUntil
	a.emitVisual(ctx)
}
func (a *闪烁) emitVisual(ctx unit.Context) {
	data, err := json.Marshal(a.visual)
	if err != nil {
		panic(err)
	}
	ctx.Out <- unit.FX{Name: "flicker-state", Kind: ctx.Kind, UnitID: ctx.ID, Data: data}
}
func (a *闪烁) external(s unit.Sense) bool {
	sp := math.Hypot(s.Self.VX, s.Self.VY)
	return sp > 300 && (a.dirX == 0 && a.dirY == 0 || (s.Self.VX*a.dirX+s.Self.VY*a.dirY)/sp < 0.5)
}
func (a *闪烁) setPass(ctx unit.Context, hold bool) {
	if a.passing != hold {
		a.passing = hold
		ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: hold}
	}
}
func (a *闪烁) push(ctx unit.Context, x, y, speed float64) {
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: x * speed, VY: y * speed}
}
func reflect(vx, vy, nx, ny float64) (float64, float64) {
	n := math.Hypot(nx, ny)
	if n < 1e-9 {
		return vx, vy
	}
	nx, ny = nx/n, ny/n
	dot := vx*nx + vy*ny
	return vx - 2*dot*nx, vy - 2*dot*ny
}
