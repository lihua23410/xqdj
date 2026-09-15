package 优昙华院

import (
	"math"
	"xqdj/internal/unit"
)

func (a *优昙华院) castVolley(ctx unit.Context, s unit.Sense, enemy unit.Snapshot, act Action) bool {
	a.volleyAng = angleOf(s.Self, enemy, act.ABin)
	a.volleyLeft = volleyCount
	a.volleyNext = s.Time
	a.volleyHold = s.Time + float64(volleyCount)*volleyGap
	a.tickVolley(ctx, s)
	return true
}

func (a *优昙华院) tickVolley(ctx unit.Context, s unit.Sense) {
	if a.volleyLeft <= 0 {
		if s.Time+1e-9 >= a.volleyHold {
			a.volleyHold = 0
		}
		return
	}
	if s.Time+1e-9 < a.volleyNext {
		return
	}
	i := volleyCount - a.volleyLeft
	off := float64(i-(volleyCount-1)/2) * volleySpread
	ang := a.volleyAng + off
	ux, uy := math.Cos(ang), math.Sin(ang)
	gap := s.Self.Radius + volleyRadius + 1.5
	spawnShot(ctx, KindSeekShot, s.Self.X, s.Self.Y, ux, uy, volleySpeed, gap, s.Self.Slot)
	shotFX(ctx, s, ux, uy)
	a.volleyLeft--
	a.volleyNext = s.Time + volleyGap
	if a.volleyLeft <= 0 {
		a.volleyHold = s.Time + volleyGap
	}
}

func (a *优昙华院) castMind(ctx unit.Context, s unit.Sense, enemy unit.Snapshot, act Action) bool {
	ang := angleOf(s.Self, enemy, act.ABin)
	ux, uy := math.Cos(ang), math.Sin(ang)
	gap := s.Self.Radius + mindRadius + 1.5
	spawnShot(ctx, KindMindShot, s.Self.X, s.Self.Y, ux, uy, mindSpeed(0), gap, s.Self.Slot)
	shotFX(ctx, s, ux, uy)
	return true
}

func (a *优昙华院) castBreak(ctx unit.Context, s unit.Sense, enemy unit.Snapshot) bool {
	hit := false
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role != unit.RoleFighter || o.Slot == s.Self.Slot {
			continue
		}
		if math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) > breakRange+o.Radius {
			continue
		}
		hit = true
		deal(ctx, ctx.ID, o.ID, SkillBreak, breakDamage)
		dx, dy := o.X-s.Self.X, o.Y-s.Self.Y
		n := math.Hypot(dx, dy)
		if n < 1e-6 {
			dx, dy = enemy.X-s.Self.X, enemy.Y-s.Self.Y
			n = math.Hypot(dx, dy)
		}
		if n < 1e-6 {
			dx, dy, n = 1, 0, 1
		}
		ctx.Out <- unit.SetVelocity{UnitID: o.ID, VX: dx / n * breakKnock, VY: dy / n * breakKnock}
	}
	if !hit {
		return false
	}
	ctx.Out <- unit.FX{
		Name: "break", Kind: ctx.Kind, UnitID: ctx.ID,
		X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot, Amount: breakRange,
	}
	return true
}

func (a *优昙华院) castLaser(ctx unit.Context, s unit.Sense, enemy unit.Snapshot, act Action) bool {
	ang := angleOf(s.Self, enemy, act.ABin)
	ux, uy := math.Cos(ang), math.Sin(ang)
	a.lockX, a.lockY = s.Self.X, s.Self.Y
	a.holdVX, a.holdVY = s.Self.VX, s.Self.VY
	a.laserUX, a.laserUY = ux, uy
	a.laserFrom = s.Time + laserWind
	a.laserUntil = a.laserFrom + laserLife
	a.laserHitAt = a.laserFrom
	a.laserOn = false
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: a.lockX, Y: a.lockY}
	ctx.Out <- unit.FX{
		Name: "laser-warn", Kind: ctx.Kind, UnitID: ctx.ID,
		X: a.lockX, Y: a.lockY, VX: ux, VY: uy, Slot: s.Self.Slot, Amount: laserWind,
	}
	return true
}

func (a *优昙华院) armLaser(ctx unit.Context, s unit.Sense) {
	if a.laserOn || s.Time+1e-9 < a.laserFrom {
		return
	}
	a.laserOn = true
	ctx.Out <- unit.Spawn{
		Kind: KindLaser, X: a.lockX, Y: a.lockY,
		VX: a.laserUX, VY: a.laserUY, OwnerID: ctx.ID, Slot: s.Self.Slot,
	}
	ctx.Out <- unit.FX{
		Name: "laser", Kind: ctx.Kind, UnitID: ctx.ID,
		X: a.lockX, Y: a.lockY, VX: a.laserUX, VY: a.laserUY, Slot: s.Self.Slot,
	}
}

func (a *优昙华院) lockPose(ctx unit.Context, s unit.Sense) {
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: a.lockX, Y: a.lockY}
}

func (a *优昙华院) tickLaser(ctx unit.Context, s unit.Sense) {
	if s.Time+1e-9 < a.laserHitAt {
		return
	}
	a.laserHitAt = s.Time + laserTick
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role != unit.RoleFighter || o.Slot == s.Self.Slot {
			continue
		}
		if !laserHits(a.lockX, a.lockY, a.laserUX, a.laserUY, *o) {
			continue
		}
		deal(ctx, ctx.ID, o.ID, SkillLaser, laserDamage)
	}
}

func (a *优昙华院) stopLaser(ctx unit.Context) {
	a.laserUntil = 0
	a.laserFrom = 0
	a.laserOn = false
	ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: KindLaser}
	vx, vy := a.holdVX, a.holdVY
	if math.Hypot(vx, vy) < 1e-6 {
		vx, vy = a.laserUX*udongeinSpeed, a.laserUY*udongeinSpeed
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: vx, VY: vy}
}

func laserHits(x, y, ux, uy float64, e unit.Snapshot) bool {
	vx, vy := e.X-x, e.Y-y
	along := vx*ux + vy*uy
	if along < -e.Radius || along > laserLen {
		return false
	}
	side := math.Abs(vx*uy - vy*ux)
	return side <= laserHalf+e.Radius
}

func (a *优昙华院) castAspect(ctx unit.Context, s unit.Sense, enemy unit.Snapshot, act Action) bool {
	ang := angleOf(s.Self, enemy, act.ABin)
	cx, cy := aspectPos(s.Self, enemy, ang)
	a.aspectUX, a.aspectUY = math.Cos(ang), math.Sin(ang)
	a.cloneX, a.cloneY = cx, cy
	a.aspectLeft = aspectShots
	a.aspectNext = s.Time
	a.aspectHold = s.Time + float64(aspectShots)*aspectGap
	ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: KindAspect}
	ctx.Out <- unit.Spawn{
		Kind: KindAspect, X: cx, Y: cy,
		OwnerID: ctx.ID, Slot: s.Self.Slot,
	}
	ctx.Out <- unit.FX{
		Name: "clone", Kind: ctx.Kind, UnitID: ctx.ID,
		X: cx, Y: cy, Slot: s.Self.Slot,
	}
	a.tickAspect(ctx, s)
	return true
}

func (a *优昙华院) tickAspect(ctx unit.Context, s unit.Sense) {
	if a.aspectLeft <= 0 {
		if s.Time+1e-9 >= a.aspectHold {
			ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: KindAspect}
			a.aspectHold = 0
		}
		return
	}
	if s.Time+1e-9 < a.aspectNext {
		return
	}
	a.fireAt(ctx, s, a.cloneX, a.cloneY, a.aspectUX, a.aspectUY)
	a.aspectLeft--
	a.aspectNext = s.Time + aspectGap
	if a.aspectLeft <= 0 {
		a.aspectHold = s.Time + aspectGap
	}
}

func (a *优昙华院) fireAt(ctx unit.Context, s unit.Sense, x, y, ux, uy float64) {
	n := math.Hypot(ux, uy)
	if n < 1e-6 {
		ux, uy, n = 1, 0, 1
	} else {
		ux, uy = ux/n, uy/n
	}
	gap := shotRadius + 2
	ctx.Out <- unit.Spawn{
		Kind:    KindAspectShot,
		X:       x + ux*gap,
		Y:       y + uy*gap,
		VX:      ux * aspectSpeed,
		VY:      uy * aspectSpeed,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	}
	ctx.Out <- unit.FX{
		Name: "shot", Kind: ctx.Kind, UnitID: ctx.ID,
		X: x, Y: y, VX: ux, VY: uy, Slot: s.Self.Slot,
	}
}

func aspectPos(self, enemy unit.Snapshot, ang float64) (float64, float64) {
	ux, uy := math.Cos(ang), math.Sin(ang)
	dist := aspectReach
	ex, ey := enemy.X-self.X, enemy.Y-self.Y
	along := ex*ux + ey*uy
	clear := enemy.Radius + udongeinRadius + 10
	if along > 0 && along < dist+clear {
		cand := along - clear
		if cand < aspectMin {
			dist = aspectMin
		} else {
			dist = cand
		}
	}
	if dist > aspectMax {
		dist = aspectMax
	}
	x, y := self.X+ux*dist, self.Y+uy*dist
	x, y = clampHex(x, y, udongeinRadius)
	x, y = leashTo(self.X, self.Y, x, y, aspectMax)
	if math.Hypot(x-enemy.X, y-enemy.Y) < clear {
		x, y = self.X-uy*aspectMin, self.Y+ux*aspectMin
		x, y = clampHex(x, y, udongeinRadius)
		x, y = leashTo(self.X, self.Y, x, y, aspectMax)
	}
	return x, y
}

func leashTo(ox, oy, x, y, maxDist float64) (float64, float64) {
	dx, dy := x-ox, y-oy
	n := math.Hypot(dx, dy)
	if n <= maxDist || n < 1e-6 {
		return x, y
	}
	s := maxDist / n
	return ox + dx*s, oy + dy*s
}

func clampHex(x, y, radius float64) (float64, float64) {
	if unit.HexContains(x, y, radius) {
		return x, y
	}
	n := math.Hypot(x, y)
	if n < 1e-6 {
		return 0, 0
	}
	limit := unit.HexRadius - radius - 4
	if limit < 8 {
		limit = 8
	}
	s := limit / n
	return x * s, y * s
}

func (a *优昙华院) castGas(ctx unit.Context, s unit.Sense, enemy unit.Snapshot, act Action) bool {
	x, y := gasPos(s.Self, enemy, act.DBin)
	ctx.Out <- unit.Spawn{
		Kind: KindGas, X: x, Y: y,
		OwnerID: ctx.ID, Slot: s.Self.Slot,
	}
	ctx.Out <- unit.FX{
		Name: "gas", Kind: ctx.Kind, UnitID: ctx.ID,
		X: x, Y: y, Slot: s.Self.Slot, Amount: gasRadius,
	}
	return true
}

func (a *优昙华院) castCrown(ctx unit.Context, s unit.Sense, enemy unit.Snapshot, act Action) bool {
	ang := angleOf(s.Self, enemy, act.ABin)
	ux, uy := math.Cos(ang), math.Sin(ang)
	gap := s.Self.Radius + crownBaseR + 1.5
	spawnShot(ctx, KindCrown, s.Self.X, s.Self.Y, ux, uy, crownSpeed, gap, s.Self.Slot)
	ctx.Out <- unit.FX{
		Name: "crown", Kind: ctx.Kind, UnitID: ctx.ID,
		X: s.Self.X, Y: s.Self.Y, VX: ux, VY: uy, Slot: s.Self.Slot,
	}
	return true
}

func (a *优昙华院) castDose(ctx unit.Context, s unit.Sense) bool {
	a.doses++
	if a.doses >= doseMax {
		a.doses = 0
		a.blast(ctx, s)
	} else {
		ctx.Out <- unit.FX{
			Name: "dose", Kind: ctx.Kind, UnitID: ctx.ID,
			X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot, Amount: float64(a.doses),
		}
	}
	a.publish(ctx.ID)
	return true
}

func (a *优昙华院) blast(ctx unit.Context, s unit.Sense) {
	ctx.Out <- unit.Spawn{
		Kind: KindBlast, X: s.Self.X, Y: s.Self.Y,
		OwnerID: ctx.ID, Slot: s.Self.Slot,
	}
	ctx.Out <- unit.FX{
		Name: "blast", Kind: ctx.Kind, UnitID: ctx.ID,
		X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot, Amount: blastRadius,
	}
	ctx.Out <- unit.FX{
		Name: "dose", Kind: ctx.Kind, UnitID: ctx.ID,
		X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot, Amount: 0,
	}
}
