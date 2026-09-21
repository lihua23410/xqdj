package 内燃机

import (
	"embed"
	"math"
	"xqdj/internal/unit"
)

const KindEngine = "内燃机"

const (
	engineRadius = 18.0
	engineHP     = 100.0
	engineAtkCD  = 0.5
	engineFrail  = 2.0
	engineBlastT = 1.0
	engineBlast  = 30.0
	engineColor  = "#ff5a1f"
	engineGhost  = 280.0
)

var (
	engineVision = [...]float64{0, 96, 120, 144, 168, 192, 216}
	engineCruise = [...]float64{0, 120, 156, 192, 228, 264, 300}
	engineAtk    = [...]float64{0, 0, 0, 1, 1, 2, 3}
	engineFill   = [...]float64{0, 2, 2, 2, 2, 4, 6}
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindEngine, assets)
	p.Register(unit.Spec{
		Kind:    KindEngine,
		Role:    unit.RoleFighter,
		Radius:  engineRadius,
		MaxHP:   engineHP,
		Speed:   engineCruise[1],
		Vision:  engineVision[1],
		Fighter: true,
		Look: unit.Look{
			Color: engineColor,
			Ghost: engineGhost,
			FX:    []string{"engine"},
		},
	}, func(unit.SpawnInfo) unit.Actor {
		return &内燃机{gear: 1}
	})
}

type 内燃机 struct {
	gear       int
	heat       float64
	lastT      float64
	atkReadyAt float64
	frail      bool
	blasted    bool
	blastAt    float64
	frailUntil float64
	hx, hy     float64
	x, y       float64
	vx, vy     float64
	ex, ey     float64
	hasEnemy   bool
	slot       int
	booted     bool
}

func (e *内燃机) Handle(ctx unit.Context, ev unit.Event) {
	switch ev := ev.(type) {
	case unit.IncomingDamage:
		unit.ConfirmHit(ctx, ev)
	case unit.Sense:
		e.onSense(ctx, ev)
	}
}

func (e *内燃机) onSense(ctx unit.Context, s unit.Sense) {
	e.slot = s.Self.Slot
	e.x, e.y = s.Self.X, s.Self.Y
	e.vx, e.vy = s.Self.VX, s.Self.VY
	e.rememberDir(s.Self.VX, s.Self.VY)
	if en := enemyOf(s); en != nil {
		e.ex, e.ey = en.X, en.Y
		e.hasEnemy = true
	}
	dt := 0.0
	if e.booted {
		dt = s.Time - e.lastT
	}
	e.lastT = s.Time
	if !e.booted {
		e.booted = true
		e.applyStats(ctx)
		ctx.Out <- unit.NoFrameFreeze{UnitID: ctx.ID, Hold: true}
	}
	wasFrail := e.frail
	if e.frail {
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
		if !e.blasted && s.Time+1e-9 >= e.blastAt {
			e.blasted = true
			e.blast(ctx, s)
		}
		if s.Time+1e-9 >= e.frailUntil {
			e.endFrail(ctx)
		}
	}
	if !wasFrail {
		e.addHeat(ctx, dt, s.Time)
	}
	e.tickAtk(ctx, s)
	e.emitHeat(ctx)
}

func (e *内燃机) addHeat(ctx unit.Context, dt, now float64) {
	if dt <= 0 {
		return
	}
	fill := engineFill[e.gear]
	if fill <= 0 {
		return
	}
	e.heat += dt / fill
	if e.heat < 1 {
		return
	}
	e.heat = 0
	if e.gear < 6 {
		e.gear++
		e.applyStats(ctx)
		return
	}
	e.breakNow(ctx, now)
}

func (e *内燃机) tickAtk(ctx unit.Context, s unit.Sense) {
	if s.Time+1e-9 < e.atkReadyAt {
		return
	}
	e.atkReadyAt = s.Time + engineAtkCD
	if e.frail {
		return
	}
	amt := engineAtk[e.gear]
	if amt <= 0 {
		return
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: amt}
	}
}

func (e *内燃机) blast(ctx unit.Context, s unit.Sense) {
	if engineBlast > 0 {
		for i := range s.Nearby {
			o := &s.Nearby[i]
			if !unit.Hittable(*o, s.Self.Slot) {
				continue
			}
			ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: engineBlast}
		}
	}
	ctx.Out <- unit.FX{
		Name:   "blast",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		X:      s.Self.X,
		Y:      s.Self.Y,
		Slot:   s.Self.Slot,
		Amount: engineVision[6],
	}
}

func (e *内燃机) breakNow(ctx unit.Context, t float64) {
	e.heat = 0
	e.frail = true
	e.blasted = false
	e.blastAt = t + engineBlastT
	e.frailUntil = t + engineFrail
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
}

func (e *内燃机) endFrail(ctx unit.Context) {
	e.frail = false
	e.blasted = false
	e.gear = 1
	e.heat = 0
	e.applyStats(ctx)
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}
	ux, uy := e.heading()
	sp := engineCruise[1]
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: ux * sp, VY: uy * sp}
}

func (e *内燃机) applyStats(ctx unit.Context) {
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: engineCruise[e.gear]}
	ctx.Out <- unit.SetVision{UnitID: ctx.ID, Vision: engineVision[e.gear]}
}

func (e *内燃机) emitHeat(ctx unit.Context) {
	frac := e.heat
	if frac > 1 {
		frac = 1
	}
	if frac < 0 {
		frac = 0
	}
	frail := 0.0
	if e.frail {
		frail = 1
		frac = 0
	}
	ctx.Out <- unit.FX{
		Name:   "heat",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		Slot:   e.slot,
		Amount: float64(e.gear),
		VX:     frac,
		VY:     frail,
	}
}

func (e *内燃机) rememberDir(vx, vy float64) {
	if math.Hypot(vx, vy) < 1e-6 {
		return
	}
	e.hx, e.hy = vx, vy
}

func (e *内燃机) heading() (float64, float64) {
	if n := math.Hypot(e.hx, e.hy); n > 1e-6 {
		return e.hx / n, e.hy / n
	}
	if e.hasEnemy {
		dx, dy := e.ex-e.x, e.ey-e.y
		if n := math.Hypot(dx, dy); n > 1e-6 {
			return dx / n, dy / n
		}
	}
	return 1, 0
}

func enemyOf(s unit.Sense) *unit.Snapshot {
	return unit.Seek(s)
}
