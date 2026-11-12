// 闪烁预判会命中的攻击并短闪，借侧闪残像夹射，收招时偶尔还击。
package 闪烁

import (
	"embed"
	"encoding/json"
	"math"
	"xqdj/internal/unit"
)

const KindFlicker = "闪烁"
const (
	bodyRadius     = 18.0
	bodyHP         = 100.0
	cruise         = 170.0
	sight          = 9999.0
	anticipation   = 0.15
	sideSpeed      = 720.0
	sideDist       = 24.0
	sideDuration   = sideDist / sideSpeed
	reflexCooldown = 0.25
	shotRange      = 280.0
	shotGap        = 0.9
	counterDamage  = 1.0
	trailLife      = 0.25
	trailSample    = 0.012
	bodyColor      = "#b7fff1"
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindFlicker, assets)
	registerBullets(p)
	p.Register(unit.Spec{Kind: KindFlicker, Role: unit.RoleFighter, Radius: bodyRadius,
		MaxHP: bodyHP, Speed: cruise, Vision: sight, Fighter: true, ObserveThreats: true,
		Look: unit.Look{Color: bodyColor, FX: []string{"flicker"}},
	}, func(unit.SpawnInfo) unit.Actor { return &闪烁{} })
}

type phase uint8

const (
	phIdle phase = iota
	phSide
)

type trailPoint struct{ X, Y, T float64 }
type shotEcho struct{ X, Y, Until float64 }

// 完整且有界的观众态，寿命使用对局时间，暂停和导出共用。
type visualState struct {
	Phase                        phase
	ReadyAt, Cooldown, SideUntil float64
	ThreatX, ThreatY             float64
	Trail                        []trailPoint
	Echo                         *shotEcho `json:",omitempty"`
}
type 闪烁 struct {
	phase                           phase
	dirX, dirY, exitX, exitY        float64
	sideUntil, readyAt, shotAt      float64
	counterSource                   uint64
	counterPending                  bool
	shotLeft                        bool
	sideStartX, sideStartY          float64
	echo                            shotEcho
	wantCruise, passing, rearmSound bool
	lastX, lastY, lastT             float64
	haveLast                        bool
	visual                          visualState
}

func (a *闪烁) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		unit.ConfirmHit(ctx, e)
	case unit.Stunned:
		if a.phase == phSide {
			a.finishSide(ctx, e.Time)
		}
		a.counterPending = false
		a.emitVisual(ctx)
	case unit.WallHit:
		if a.phase == phSide || a.wantCruise {
			a.exitX, a.exitY = reflect(a.exitX, a.exitY, e.NX, e.NY)
			if a.phase == phSide {
				a.leaveEcho(e.Time, e.X, e.Y)
			}
			a.finishSide(ctx, e.Time)
		}
	case unit.Sense:
		a.onSense(ctx, e)
	}
}

func (a *闪烁) onSense(ctx unit.Context, s unit.Sense) {
	if s.Self.HP <= 0 {
		a.counterPending = false
		return
	}
	jumped := a.haveLast && math.Hypot(s.Self.X-a.lastX, s.Self.Y-a.lastY) > math.Hypot(s.Self.VX, s.Self.VY)*math.Max(1.0/60, s.Time-a.lastT)+48
	yield := s.MovementLocked || s.Knock || (!jumped && a.external(s))
	if a.phase == phSide {
		a.sampleTrail(s)
		if s.Time+1e-9 >= a.sideUntil {
			a.leaveEcho(s.Time, s.Self.X, s.Self.Y)
			a.finishSide(ctx, a.sideUntil)
		} else if !yield && !a.tryDodge(ctx, s) {
			a.push(ctx, a.dirX, a.dirY, sideSpeed)
		}
	}
	if a.wantCruise && a.phase != phSide && !yield {
		a.push(ctx, a.exitX, a.exitY, cruise)
		ctx.Out <- unit.SetFSDirection{UnitID: ctx.ID, VX: a.exitX, VY: a.exitY}
		a.wantCruise = false
	}
	if a.counterPending {
		a.fireCounter(ctx, s)
	}
	a.counterPending = false
	if a.phase == phIdle && s.Time+1e-9 >= a.readyAt && !yield {
		if !a.tryDodge(ctx, s) && a.rearmSound {
			a.sound(ctx, s, "armed")
			a.rearmSound = false
		}
	}
	if s.Time+1e-9 >= a.shotAt {
		if target := shotTarget(s, 0); target != nil {
			a.fire(ctx, s, *target, false)
			a.shotAt = s.Time + shotGap
		}
	}
	a.paint(ctx, s)
	a.lastX, a.lastY, a.lastT = s.Self.X, s.Self.Y, s.Time
	a.haveLast = true
}

func (a *闪烁) tryDodge(ctx unit.Context, s unit.Sense) bool {
	var remaining float64
	if a.phase == phSide {
		remaining = a.sideUntil - s.Time
		speed := sideSpeed * dodgeScale(s)
		s.Self.VX, s.Self.VY = a.dirX*speed, a.dirY*speed
	}
	dx, dy, threat, ok := planDodge(ctx.State, s, remaining)
	if !ok {
		return false
	}
	if a.phase == phSide {
		a.fireCounter(ctx, s)
	}
	a.readyAt = s.Time + reflexCooldown
	a.counterSource = threat.From
	a.startSlide(ctx, s, dx, dy, threat.X, threat.Y)
	return true
}

func (a *闪烁) fireCounter(ctx unit.Context, s unit.Sense) {
	if target := shotTarget(s, a.counterSource); target != nil {
		a.fire(ctx, s, *target, true)
	}
}

func shotTarget(s unit.Sense, source uint64) *unit.Snapshot {
	inRange := func(o unit.Snapshot) bool {
		return math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) <= shotRange
	}
	if source != 0 {
		if target := unit.SeekIf(s, func(o unit.Snapshot) bool { return o.ID == source && inRange(o) }); target != nil {
			return target
		}
	}
	return unit.SeekIf(s, inRange)
}

func seekID(s unit.Sense, id uint64) *unit.Snapshot {
	if id == 0 {
		return nil
	}
	return unit.SeekIf(s, func(o unit.Snapshot) bool { return o.ID == id })
}

func (a *闪烁) startSlide(ctx unit.Context, s unit.Sense, dx, dy, threatX, threatY float64) {
	if a.phase != phSide {
		a.sideStartX, a.sideStartY = s.Self.X, s.Self.Y
		a.visual.Trail = a.visual.Trail[:0]
	}
	a.phase = phSide
	a.dirX, a.dirY, a.exitX, a.exitY = dx, dy, dx, dy
	a.sideUntil = s.Time + sideDuration
	a.wantCruise, a.rearmSound = false, true
	a.visual.ThreatX, a.visual.ThreatY = threatX, threatY
	a.sampleTrail(s)
	a.setPass(ctx, true)
	a.push(ctx, dx, dy, sideSpeed)
	a.sound(ctx, s, "reflex")
}

func (a *闪烁) finishSide(ctx unit.Context, t float64) {
	if a.phase != phSide {
		return
	}
	a.phase = phIdle
	a.sideUntil = t
	a.wantCruise = true
	a.counterPending = true
	a.setPass(ctx, false)
}

func (a *闪烁) leaveEcho(t, x, y float64) {
	if math.Hypot(x-a.sideStartX, y-a.sideStartY) >= 8 {
		a.echo = shotEcho{X: a.sideStartX, Y: a.sideStartY, Until: t + shotGap}
	}
}

func (a *闪烁) fire(ctx unit.Context, s unit.Sense, target unit.Snapshot, counter bool) {
	dx, dy := target.X-s.Self.X, target.Y-s.Self.Y
	angle := math.Atan2(dy, dx)
	data := bulletData{Target: target.ID, Until: s.Time + 1}
	if counter {
		data.Smart, data.Damage = true, counterDamage
		spawnRound(ctx, s.Self, kindSmart, angle, data)
	} else {
		data.Damage, data.BendAt = splitDamage, s.Time+splitDelay
		spread := splitAngle
		if a.shotLeft {
			spread = -spread
		}
		a.shotLeft = !a.shotLeft
		echo := a.echo
		a.echo = shotEcho{}
		if s.Time < echo.Until && !blockedAt(echo.X, echo.Y, roundRadius, s.Field, s.Walls) {
			// 两发朝各自外侧展开，从不同的位置折向同一个目标。
			if dx*(echo.Y-s.Self.Y)-dy*(echo.X-s.Self.X) < 0 {
				spread = -splitAngle
			} else {
				spread = splitAngle
			}
			origin := s.Self
			origin.X, origin.Y = echo.X, echo.Y
			echoAngle := math.Atan2(target.Y-echo.Y, target.X-echo.X)
			spawnRound(ctx, origin, kindSplit, echoAngle+spread, data)
		}
		spawnRound(ctx, s.Self, kindSplit, angle-spread, data)
	}
	a.sound(ctx, s, "shot")
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
	a.visual.Echo = nil
	if s.Time < a.echo.Until {
		a.visual.Echo = &a.echo
	}
	a.emitVisual(ctx)
}
func (a *闪烁) emitVisual(ctx unit.Context) {
	a.visual.Phase, a.visual.SideUntil = a.phase, a.sideUntil
	a.visual.ReadyAt, a.visual.Cooldown = a.readyAt, reflexCooldown
	data, err := json.Marshal(a.visual)
	if err != nil {
		panic(err)
	}
	ctx.Out <- unit.FX{Name: "flicker-state", Kind: ctx.Kind, UnitID: ctx.ID, Data: data}
}
func (a *闪烁) external(s unit.Sense) bool {
	sp := math.Hypot(s.Self.VX, s.Self.VY)
	dx, dy := a.dirX, a.dirY
	if a.wantCruise {
		// 撞墙后的侧闪余速沿收招方向，不能误认成外部击退而一直跳过恢复。
		dx, dy = a.exitX, a.exitY
	}
	return sp > 300 && (dx == 0 && dy == 0 || (s.Self.VX*dx+s.Self.VY*dy)/sp < 0.5)
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
