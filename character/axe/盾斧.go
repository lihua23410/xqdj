package 盾斧

import (
	"embed"
	"math"
	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const KindAxe = "盾斧"

const (
	axeRadius = 18.0
	axeSpeed  = 165.0
	axeSlow   = 90.0
	axeHP     = 100.0
	axeVision = 9999.0
	axeColor  = "#9a9a9a"

	seekR      = 74.0 * 18.0 / 54.0
	seekSpan   = 120.0
	axeSeekR   = 104.0
	chouR      = 104.0
	chouSpan   = 60.0
	shieldSpan = 300.0

	slashBeat  = 0.25
	slashGap   = 0.25
	slashLife  = 0.75
	thrustSp   = 350.0
	thrustLife = 1.5
	fillStand  = 1.0
	bottleWait = 0.5
	axeWind    = 0.2
	axeStand   = 1.0
	spinT      = 0.35
	tsuiGap    = 0.4
	flipDown   = 0.8
	flipUp     = 0.25
	waveWait   = 0.4
	waveGap    = 0.3

	slashDmg    = 5.0
	thrustDmg   = 3.0
	redBonus    = 2.0
	daiDmg      = 18.0
	tsuiDmg     = 15.0
	chouSpinDmg = 15.0
	chouFanDmg  = 30.0
	waveDmg     = 15.0

	phialEmpty = 0
	phialWhite = 1
	phialGold  = 2
)

const (
	formShield = iota
	formSword
	formAxe
)

const (
	stepIdle = iota
	stepSlash1
	stepThrust
	stepSlash2
	stepFill
	stepBottle
	stepAxeIn
	stepAxeHold
	stepAxeIdle
	stepDai
	stepTsui1
	stepTsuiWait
	stepTsui2
	stepChouSpin
	stepChouFlip
	stepChouFan
	stepChouWait
	stepWave1
	stepWave2
	stepWave3
)

func init() {
	p := unit.NewPack(KindAxe, assets)
	p.Register(unit.Spec{
		Kind:    KindAxe,
		Role:    unit.RoleFighter,
		Radius:  axeRadius,
		MaxHP:   axeHP,
		Speed:   axeSpeed,
		Vision:  axeVision,
		Fighter: true,
		Look:    unit.Look{Color: axeColor, Ghost: 220, FX: []string{"axe"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &盾斧{form: formShield, hx: 0, hy: 1}
	})
}

type 盾斧 struct {
	form      int
	redShield bool
	redSword  bool
	phial     int
	energy    int
	step      int
	until     float64
	slashN    int
	lockID    uint64
	thrustDir [2]float64
	hx, hy    float64
	x, y      float64
	seen      map[uint64][2]float64
	slot      int
	booted    bool
}

func (a *盾斧) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		a.confirm(ctx, e)
	case unit.Collision:
		a.onHit(ctx, e)
	case unit.Sense:
		a.tick(ctx, e)
	}
}

func (a *盾斧) confirm(ctx unit.Context, d unit.IncomingDamage) {
	amt := d.Amount
	if a.form == formShield {
		if src, ok := a.seen[d.From]; ok && a.front(src[0], src[1]) {
			if a.redShield {
				amt *= 0.25
			} else {
				amt *= 0.5
			}
		}
	}
	ctx.Out <- unit.ConfirmDamage{Token: d.Token, UnitID: ctx.ID, Amount: amt}
}

func (a *盾斧) front(ox, oy float64) bool {
	dx, dy := ox-a.x, oy-a.y
	if math.Hypot(dx, dy) < 1e-6 {
		return true
	}
	fn := math.Hypot(a.hx, a.hy)
	if fn < 1e-6 {
		return true
	}
	dot := (dx*a.hx + dy*a.hy) / (math.Hypot(dx, dy) * fn)
	if dot > 1 {
		dot = 1
	} else if dot < -1 {
		dot = -1
	}
	return math.Acos(dot) <= unit.Deg(shieldSpan)/2
}

func (a *盾斧) onHit(ctx unit.Context, e unit.Collision) {
	if a.step != stepThrust || !unit.EnemyTarget(e, a.slot) {
		return
	}
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: false}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: a.strikeDmg(thrustDmg)}
	a.beginSlash(ctx, e.Time, true)
}

func (a *盾斧) tick(ctx unit.Context, s unit.Sense) {
	a.x, a.y = s.Self.X, s.Self.Y
	a.slot = s.Self.Slot
	a.note(s)
	a.face(s.Self.VX, s.Self.VY)
	if !a.booted {
		a.booted = true
		a.emitLook(ctx)
		a.emitPose(ctx, 90)
		a.emitPhial(ctx)
	}
	switch a.step {
	case stepIdle:
		if hit := a.firstInFan(s, seekR, seekSpan); hit != nil {
			a.energy = 0
			a.lockID = hit.ID
			a.toSword(ctx)
			a.beginSlash(ctx, s.Time, false)
		}
	case stepSlash1, stepSlash2:
		a.tickSlash(ctx, s)
	case stepThrust:
		a.tickThrust(ctx, s)
	case stepFill:
		if s.Time+1e-9 >= a.until {
			a.fillBottles(ctx)
			a.until = s.Time + bottleWait
			a.step = stepBottle
		}
	case stepBottle:
		if s.Time+1e-9 >= a.until {
			a.convertRed(ctx, s.Time)
		}
	case stepAxeIn:
		if s.Time+1e-9 >= a.until {
			a.hold(ctx, true)
			a.until = s.Time + axeStand
			a.step = stepAxeHold
		}
	case stepAxeHold:
		if s.Time+1e-9 >= a.until {
			a.form = formAxe
			a.step = stepAxeIdle
			a.hold(ctx, false)
			ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: axeSlow}
			a.emitLook(ctx)
			a.emitPose(ctx, 120)
		}
	case stepAxeIdle:
		if hit := a.firstInFan(s, axeSeekR, seekSpan); hit != nil {
			a.lockID = hit.ID
			a.turnTo(ctx, s.Self.X, s.Self.Y, hit.X, hit.Y)
			a.hold(ctx, true)
			a.step = stepDai
			a.until = s.Time + spinT
			a.emitPose(ctx, 300)
			a.emitAnim(ctx, s, "dai", 300, 120, spinT)
		}
	case stepDai:
		if s.Time+1e-9 >= a.until {
			a.turnToEnemy(ctx, s)
			a.hitFan(ctx, s, axeSeekR, seekSpan, daiDmg, false)
			a.step = stepTsui1
			a.until = s.Time + spinT
			a.emitAnim(ctx, s, "tsui", 120, 120, spinT)
		}
	case stepTsui1:
		if s.Time+1e-9 >= a.until {
			a.turnToEnemy(ctx, s)
			a.hitFan(ctx, s, axeSeekR, seekSpan, tsuiDmg, false)
			a.step = stepTsuiWait
			a.until = s.Time + tsuiGap
		}
	case stepTsuiWait:
		if s.Time+1e-9 >= a.until {
			a.turnToEnemy(ctx, s)
			a.step = stepTsui2
			a.until = s.Time + spinT
			a.emitAnim(ctx, s, "tsui-back", 120, 120, spinT)
		}
	case stepTsui2:
		if s.Time+1e-9 >= a.until {
			a.turnToEnemy(ctx, s)
			a.hitFan(ctx, s, axeSeekR, seekSpan, tsuiDmg, false)
			a.step = stepChouSpin
			a.until = s.Time + spinT
			a.emitAnim(ctx, s, "chou", 120, 120, spinT)
		}
	case stepChouSpin:
		if s.Time+1e-9 >= a.until {
			a.turnToEnemy(ctx, s)
			a.hitFan(ctx, s, axeSeekR, seekSpan, chouSpinDmg, false)
			a.step = stepChouFlip
			a.until = s.Time + flipDown + flipUp
			a.emitAnim(ctx, s, "flip", flipDown, flipUp, 0)
		}
	case stepChouFlip:
		if s.Time+1e-9 >= a.until {
			a.turnToEnemy(ctx, s)
			a.hitFan(ctx, s, chouR, chouSpan, chouFanDmg, false)
			a.step = stepChouWait
			a.until = s.Time + waveWait
		}
	case stepChouWait:
		if s.Time+1e-9 >= a.until {
			a.turnToEnemy(ctx, s)
			a.wave(ctx, s, 0)
			a.step = stepWave1
			a.until = s.Time + waveGap
		}
	case stepWave1:
		if s.Time+1e-9 >= a.until {
			a.turnToEnemy(ctx, s)
			a.wave(ctx, s, 1)
			a.step = stepWave2
			a.until = s.Time + waveGap
		}
	case stepWave2:
		if s.Time+1e-9 >= a.until {
			a.turnToEnemy(ctx, s)
			a.wave(ctx, s, 2)
			a.dropRed(ctx)
			a.step = stepIdle
			a.hold(ctx, false)
		}
	}
}

func (a *盾斧) beginSlash(ctx unit.Context, now float64, second bool) {
	if second {
		a.step = stepSlash2
	} else {
		a.step = stepSlash1
	}
	a.slashN = 0
	a.until = now + slashLife
	a.hold(ctx, true)
	a.emitAnim(ctx, unit.Sense{Time: now, Self: unit.Snapshot{X: a.x, Y: a.y, VX: a.hx, VY: a.hy}}, "slash", 150, 30, slashBeat)
}

func (a *盾斧) tickSlash(ctx unit.Context, s unit.Sense) {
	elapsed := s.Time - (a.until - slashLife)
	if a.slashN == 0 {
		a.hitFan(ctx, s, seekR, seekSpan, a.strikeDmg(slashDmg), true)
		a.slashN = 1
	} else if a.slashN == 1 && elapsed+1e-9 >= slashGap {
		a.hitFan(ctx, s, seekR, seekSpan, a.strikeDmg(slashDmg), false)
		a.slashN = 2
	}
	if s.Time+1e-9 < a.until {
		return
	}
	if a.step == stepSlash1 {
		a.aimThrust(s)
		a.step = stepThrust
		a.until = s.Time + thrustLife
		a.hold(ctx, false)
		ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: a.thrustDir[0] * thrustSp, VY: a.thrustDir[1] * thrustSp}
		return
	}
	a.endCombo(ctx, s.Time)
}

func (a *盾斧) tickThrust(ctx unit.Context, s unit.Sense) {
	a.aimThrust(s)
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: a.thrustDir[0] * thrustSp, VY: a.thrustDir[1] * thrustSp}
	if s.Time+1e-9 >= a.until {
		a.endCombo(ctx, s.Time)
	}
}

func (a *盾斧) aimThrust(s unit.Sense) {
	tx, ty, ok := a.lockPos(s)
	if !ok {
		if hit := a.firstInFan(s, seekR*2, 360); hit != nil {
			a.lockID = hit.ID
			tx, ty, ok = hit.X, hit.Y, true
		}
	}
	if !ok {
		if math.Hypot(a.thrustDir[0], a.thrustDir[1]) < 1e-6 {
			a.thrustDir = [2]float64{a.hx, a.hy}
		}
		return
	}
	dx, dy := tx-s.Self.X, ty-s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	a.thrustDir = [2]float64{dx / n, dy / n}
	a.hx, a.hy = a.thrustDir[0], a.thrustDir[1]
}

func (a *盾斧) lockPos(s unit.Sense) (x, y float64, ok bool) {
	if a.lockID == 0 {
		return 0, 0, false
	}
	for i := range s.Nearby {
		if s.Nearby[i].ID == a.lockID {
			return s.Nearby[i].X, s.Nearby[i].Y, true
		}
	}
	p, ok := a.seen[a.lockID]
	return p[0], p[1], ok
}

func (a *盾斧) endCombo(ctx unit.Context, now float64) {
	ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: false}
	a.toShield(ctx)
	a.hold(ctx, true)
	a.step = stepFill
	a.until = now + fillStand
	a.emitPhial(ctx)
}

func (a *盾斧) fillBottles(ctx unit.Context) {
	switch a.energy {
	case 1:
		if a.phial == phialWhite {
			a.phial = phialGold
		} else if a.phial == phialEmpty {
			a.phial = phialWhite
		}
	case 2:
		a.phial = phialGold
	}
	a.emitPhial(ctx)
}

func (a *盾斧) convertRed(ctx unit.Context, now float64) {
	if a.phial != phialGold {
		a.step = stepIdle
		a.hold(ctx, false)
		return
	}
	a.phial = phialEmpty
	a.emitPhial(ctx)
	if !a.redShield {
		a.redShield = true
		a.step = stepIdle
		a.hold(ctx, false)
		a.emitLook(ctx)
		return
	}
	if !a.redSword {
		a.redSword = true
		a.step = stepIdle
		a.hold(ctx, false)
		a.emitLook(ctx)
		return
	}
	a.step = stepAxeIn
	a.until = now + axeWind
	a.hold(ctx, false)
	a.emitLook(ctx)
}

func (a *盾斧) dropRed(ctx unit.Context) {
	a.redShield = false
	a.redSword = false
	a.phial = phialEmpty
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: axeSpeed}
	a.toShield(ctx)
	a.emitPhial(ctx)
}

func (a *盾斧) toSword(ctx unit.Context) {
	a.form = formSword
	a.emitLook(ctx)
	pose := 120.0
	a.emitPose(ctx, pose)
}

func (a *盾斧) toShield(ctx unit.Context) {
	a.form = formShield
	a.emitLook(ctx)
	a.emitPose(ctx, 90)
}

func (a *盾斧) turnToEnemy(ctx unit.Context, s unit.Sense) {
	var hit *unit.Snapshot
	if a.lockID != 0 {
		for i := range s.Nearby {
			o := &s.Nearby[i]
			if o.ID == a.lockID && unit.Hittable(*o, s.Self.Slot) {
				hit = o
				break
			}
		}
	}
	if hit == nil {
		hit = unit.Seek(s)
	}
	if hit == nil {
		return
	}
	a.lockID = hit.ID
	a.turnTo(ctx, s.Self.X, s.Self.Y, hit.X, hit.Y)
}

func (a *盾斧) turnTo(ctx unit.Context, x, y, tx, ty float64) {
	dx, dy := tx-x, ty-y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	a.hx, a.hy = dx/n, dy/n
	ctx.Out <- unit.FX{Name: "face", Kind: ctx.Kind, UnitID: ctx.ID, VX: a.hx, VY: a.hy}
}

func (a *盾斧) hold(ctx unit.Context, on bool) {
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: on}
	if on {
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	}
}

func (a *盾斧) strikeDmg(base float64) float64 {
	if a.redSword && a.form != formAxe {
		return base + redBonus
	}
	return base
}

func (a *盾斧) hitFan(ctx unit.Context, s unit.Sense, r, span, dmg float64, energy bool) {
	fighter := false
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) || !inFan(s.Self, a.hx, a.hy, *o, r, span) {
			continue
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: dmg}
		if o.Role == unit.RoleFighter {
			fighter = true
		}
	}
	if energy && fighter && a.energy < 2 {
		a.energy++
		a.emitPhial(ctx)
	}
}

func (a *盾斧) wave(ctx unit.Context, s unit.Sense, i int) {
	seg := chouR / 3
	near, far := float64(i)*seg, float64(i+1)*seg
	mid := (near + far) / 2
	width := 2 * mid * math.Tan(unit.Deg(chouSpan/2))
	a.hitRect(ctx, s, near, far, width, waveDmg)
	slam := math.Max(width, far-near) * 0.55
	if slam < 22 {
		slam = 22
	}
	ctx.Out <- unit.FX{
		Name: "wave", Kind: ctx.Kind, UnitID: ctx.ID,
		X: s.Self.X + a.hx*mid, Y: s.Self.Y + a.hy*mid,
		VX: a.hx * slam, VY: a.hy * slam,
		Amount: float64(i + 1), Slot: s.Self.Slot,
	}
}

func (a *盾斧) hitRect(ctx unit.Context, s unit.Sense, near, far, width, dmg float64) {
	px, py := -a.hy, a.hx
	half := width / 2
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		dx, dy := o.X-s.Self.X, o.Y-s.Self.Y
		along := dx*a.hx + dy*a.hy
		side := dx*px + dy*py
		rr := o.Radius
		if along+rr < near || along-rr > far || math.Abs(side) > half+rr {
			continue
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: dmg}
	}
}

func (a *盾斧) firstInFan(s unit.Sense, r, span float64) *unit.Snapshot {
	return unit.SeekIf(s, func(o unit.Snapshot) bool {
		return inFan(s.Self, a.hx, a.hy, o, r, span)
	})
}

func inFan(self unit.Snapshot, hx, hy float64, o unit.Snapshot, r, spanDeg float64) bool {
	dx, dy := o.X-self.X, o.Y-self.Y
	dist := math.Hypot(dx, dy)
	if dist > r+o.Radius {
		return false
	}
	if dist < 1e-6 {
		return true
	}
	fn := math.Hypot(hx, hy)
	if fn < 1e-6 {
		return true
	}
	dot := (dx*hx + dy*hy) / (dist * fn)
	if dot > 1 {
		dot = 1
	} else if dot < -1 {
		dot = -1
	}
	return math.Acos(dot) <= unit.Deg(spanDeg)/2
}

func (a *盾斧) face(vx, vy float64) {
	if math.Hypot(vx, vy) < 1e-6 {
		return
	}
	n := math.Hypot(vx, vy)
	a.hx, a.hy = vx/n, vy/n
}

func (a *盾斧) note(s unit.Sense) {
	if a.seen == nil {
		a.seen = map[uint64][2]float64{}
	}
	a.seen[s.Self.ID] = [2]float64{s.Self.X, s.Self.Y}
	for i := range s.Nearby {
		o := s.Nearby[i]
		a.seen[o.ID] = [2]float64{o.X, o.Y}
	}
}

func (a *盾斧) emitLook(ctx unit.Context) {
	form := float64(a.form)
	rs, rw := 0.0, 0.0
	if a.redShield {
		rs = 1
	}
	if a.redSword {
		rw = 1
	}
	ctx.Out <- unit.FX{Name: "look", Kind: ctx.Kind, UnitID: ctx.ID, Amount: form, VX: rs, VY: rw}
}

func (a *盾斧) emitPose(ctx unit.Context, deg float64) {
	ctx.Out <- unit.FX{Name: "pose", Kind: ctx.Kind, UnitID: ctx.ID, Amount: deg}
}

func (a *盾斧) emitAnim(ctx unit.Context, s unit.Sense, name string, from, to, dur float64) {
	ctx.Out <- unit.FX{
		Name: name, Kind: ctx.Kind, UnitID: ctx.ID,
		X: s.Self.X, Y: s.Self.Y, VX: from, VY: to, Amount: dur, Slot: s.Self.Slot,
	}
}

func (a *盾斧) emitPhial(ctx unit.Context) {
	amt := 0.0
	if a.step == stepSlash1 || a.step == stepThrust || a.step == stepSlash2 {
		amt = float64(a.energy)
	} else if a.phial == phialWhite {
		amt = 10
	} else if a.phial == phialGold {
		amt = 20
	}
	ctx.Out <- unit.FX{Name: "phial", Kind: ctx.Kind, UnitID: ctx.ID, Amount: amt}
}
