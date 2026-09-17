package 人偶使

import (
	"math"
	"xqdj/internal/unit"
)

func (a *人偶使) invoke(ctx unit.Context, s unit.Sense, enemy unit.Snapshot, sk uint8) bool {
	switch sk {
	case SkillAtkFan:
		return a.castFan(ctx, s, enemy, unit.Deg(atkFanDeg), atkFanHits, atkFanDmg, atkFanWind, atkFanLife, 1, poseAd)
	case SkillFan:
		return a.castFan(ctx, s, enemy, unit.Deg(skFanDeg), skFanHits, skFanDmg, atkFanWind, atkFanLife, 1, poseAc)
	case SkillAtkRect:
		return a.castRect(ctx, s, enemy)
	case SkillPlace:
		return a.castPlace(ctx, s)
	case SkillN22:
		if a.playingCard || a.levelOf(SkillN22) >= 1 {
			return a.castChain(ctx, s, enemy)
		}
		return a.castChase(ctx, s, enemy)
	case SkillN26:
		if a.playingCard || a.levelOf(SkillN26) >= 1 {
			return a.castSpell26(ctx, s)
		}
		return a.castSpecial26(ctx, s)
	case SkillN24:
		return a.castSpell24(ctx, s, enemy)
	case SkillN62:
		return a.castSpell62(ctx, s, enemy)
	case CardDemon:
		return a.castDemon(ctx, s, enemy)
	case CardHourai:
		return a.castHourai(ctx, s, enemy)
	case CardBattle:
		return a.castBattle(ctx, s)
	case CardSpirit:
		return a.castSpirit(ctx, s, enemy)
	}
	return false
}

func (a *人偶使) castFan(ctx unit.Context, s unit.Sense, enemy unit.Snapshot, span float64, hits int, dmg, wind, life float64, nDolls int, pose uint8) bool {
	if enemy.ID == 0 {
		return false
	}
	ux, uy := toward(s.Self, enemy)
	ox, oy := offset(s.Self.X, s.Self.Y, ux, uy, dollReach)
	ox, oy = clampHex(ox, oy, dollRadius)
	for i := 0; i < nDolls; i++ {
		px, py := ox, oy
		if nDolls > 1 {
			perpX, perpY := -uy, ux
			off := (float64(i) - float64(nDolls-1)/2) * 16
			px, py = clampHex(ox+perpX*off, oy+perpY*off, dollRadius)
		}
		spawnDollAt(ctx, s, px, py, dollSpec{mode: dollStrike, pose: pose, recallAt: s.Time + wind + life})
	}
	a.setJob(ctx, s, job{
		kind: SkillAtkFan, until: s.Time + wind + life, next: s.Time + wind,
		left: hits, ox: ox, oy: oy, ux: ux, uy: uy, dmg: dmg, span: span, reach: fanR,
		wind: s.Time + wind, gap: life / float64(hits),
	})
	return true
}

func (a *人偶使) castRect(ctx unit.Context, s unit.Sense, enemy unit.Snapshot) bool {
	if enemy.ID == 0 {
		return false
	}
	ux, uy := toward(s.Self, enemy)
	ox, oy := offset(s.Self.X, s.Self.Y, ux, uy, dollReach)
	ox, oy = clampHex(ox, oy, dollRadius)
	for i := 0; i < 4; i++ {
		perpX, perpY := -uy, ux
		col, row := float64(i%2)*18-9, float64(i/2)*18-9
		px, py := clampHex(ox+ux*row+perpX*col, oy+uy*row+perpY*col, dollRadius)
		spawnDollAt(ctx, s, px, py, dollSpec{mode: dollStrike, pose: poseAh, recallAt: s.Time + atkRectWind + atkRectLife})
	}
	a.setJob(ctx, s, job{
		kind: SkillAtkRect, until: s.Time + atkRectWind + atkRectLife, next: s.Time + atkRectWind,
		left: 1, ox: ox, oy: oy, ux: ux, uy: uy, dmg: atkRectDmg, knock: true,
		wind: s.Time + atkRectWind, gap: atkRectLife,
	})
	return true
}

func (a *人偶使) castPlace(ctx unit.Context, s unit.Sense) bool {
	ux, uy := randDir(a)
	x, y := offset(s.Self.X, s.Self.Y, ux, uy, dollFar)
	x, y = clampHex(x, y, dollRadius)
	spawnDollAt(ctx, s, x, y, dollSpec{mode: dollShoot, shootAt: s.Time + placeWind, shots: placeShots})
	return true
}

func (a *人偶使) castChase(ctx unit.Context, s unit.Sense, enemy unit.Snapshot) bool {
	if enemy.ID == 0 {
		return false
	}
	n := 0
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind == KindNingyushiDoll && o.OwnerID == ctx.ID && stateOf(o.ID) == dollIdle {
			orderChase(o.ID, s.Time+placeWind, s.Time+placeWind+chaseLife, enemy.X, enemy.Y)
			n++
		}
	}
	return n > 0
}

func (a *人偶使) castSpecial26(ctx unit.Context, s unit.Sense) bool {
	ux, uy := randDir(a)
	x, y := offset(s.Self.X, s.Self.Y, ux, uy, dollReach)
	x, y = clampHex(x, y, dollRadius)
	spawnDollAt(ctx, s, x, y, dollSpec{mode: dollEllipse, ux: ux, uy: uy, armAt: s.Time + placeWind, until: s.Time + placeWind + chaseLife, rangeOn: true})
	return true
}

func (a *人偶使) castChain(ctx unit.Context, s unit.Sense, enemy unit.Snapshot) bool {
	pts := [][2]float64{{s.Self.X, s.Self.Y}}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind == KindNingyushiDoll && o.OwnerID == ctx.ID && stateOf(o.ID) == dollIdle {
			pts = append(pts, [2]float64{o.X, o.Y})
		}
	}
	if enemy.ID != 0 {
		pts = append(pts, [2]float64{enemy.X, enemy.Y})
	} else {
		hx, hy := a.hx, a.hy
		if math.Hypot(hx, hy) < 1e-6 {
			hx, hy = 1, 0
		}
		pts = append(pts, [2]float64{s.Self.X + hx*80, s.Self.Y + hy*80})
	}
	hops := len(pts) - 1
	until := s.Time + placeWind + laserHold
	a.setJob(ctx, s, job{
		kind: SkillN22, until: until, next: s.Time + placeWind,
		left: hops, paths: pts, idx: 1,
		dmg: float64(a.spellLevel(SkillN22)+5) * 1.4, wind: s.Time + placeWind,
	})
	if until > a.lockUntil {
		a.lockUntil = until
	}
	return true
}

func (a *人偶使) castSpell26(ctx unit.Context, s unit.Sense) bool {
	ux, uy := randDir(a)
	x, y := offset(s.Self.X, s.Self.Y, ux, uy, dollReach)
	x, y = clampHex(x, y, dollRadius)
	spawnDollAt(ctx, s, x, y, dollSpec{
		mode: dollEllipseRecall, ux: ux, uy: uy,
		armAt: s.Time + placeWind,
		dmg:   float64(a.spellLevel(SkillN26)+5) * 1.4, once: true, rangeOn: true, slow: true,
	})
	return true
}

func (a *人偶使) castSpell24(ctx unit.Context, s unit.Sense, enemy unit.Snapshot) bool {
	ux, uy := faceOf(a, s, enemy)
	for i := 0; i < spell24N; i++ {
		ang := float64(i) * 2 * math.Pi / float64(spell24N)
		px := s.Self.X + math.Cos(ang)*spell24Ring
		py := s.Self.Y + math.Sin(ang)*spell24Ring
		spawnDollAt(ctx, s, px, py, dollSpec{
			mode: dollStrike, pose: poseAh, ang: ang, follow: true,
			until: s.Time + spell24Wind + spell24Life, bornGen: a.abortGen,
		})
	}
	a.rush(ctx, ux, uy, spell24Rush)
	a.setJob(ctx, s, job{
		kind: SkillN24, until: s.Time + spell24Wind + spell24Life,
		next: s.Time + spell24Wind, left: spell24Hits,
		ox: s.Self.X, oy: s.Self.Y, ux: ux, uy: uy,
		dmg:  float64(a.spellLevel(SkillN24)+2) * 1.4,
		wind: s.Time + spell24Wind, selfUX: ux, selfUY: uy,
		gap: spell24Life / float64(spell24Hits),
	})
	return true
}

func (a *人偶使) castSpell62(ctx unit.Context, s unit.Sense, enemy unit.Snapshot) bool {
	ux, uy := faceOf(a, s, enemy)
	pushBomb(bombSpec{kind: bombBounce, dmg: float64(a.spellLevel(SkillN62)+5) * 1.4, tx: enemy.X, ty: enemy.Y})
	gap := s.Self.Radius + bombR + 2
	ctx.Out <- unit.Spawn{
		Kind: KindNingyushiBomb, X: s.Self.X + ux*gap, Y: s.Self.Y + uy*gap,
		VX: ux * bounceSp, VY: uy * bounceSp, OwnerID: ctx.ID, Slot: s.Self.Slot,
	}
	return true
}

func (a *人偶使) castDemon(ctx unit.Context, s unit.Sense, enemy unit.Snapshot) bool {
	if enemy.ID == 0 {
		return false
	}
	dx, dy := enemy.X-s.Self.X, enemy.Y-s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		dx, dy, n = 1, 0, 1
	}
	ux, uy := dx/n, dy/n
	pushBomb(bombSpec{kind: bombSeek, dmg: demonDmg, tx: enemy.X, ty: enemy.Y, explode: demonR})
	gap := s.Self.Radius + bombR + 2
	ctx.Out <- unit.Spawn{
		Kind: KindNingyushiBomb, X: s.Self.X + ux*gap, Y: s.Self.Y + uy*gap,
		VX: ux * demonSp, VY: uy * demonSp, OwnerID: ctx.ID, Slot: s.Self.Slot,
	}
	return true
}

func (a *人偶使) castHourai(ctx unit.Context, s unit.Sense, enemy unit.Snapshot) bool {
	if enemy.ID == 0 {
		return false
	}
	ux, uy := toward(s.Self, enemy)
	a.setJob(ctx, s, job{
		kind: CardHourai, until: s.Time + houraiLife, next: s.Time,
		left: houraiHits, ux: ux, uy: uy, dmg: houraiDmg, ox: s.Self.X, oy: s.Self.Y,
		gap: houraiLife / float64(houraiHits),
	})
	perpX, perpY := -uy, ux
	for i := 0; i < houraiBeams; i++ {
		off := (float64(i) - 1.5) * 14
		ctx.Out <- unit.Spawn{
			Kind: KindNingyushiBeam, X: s.Self.X + perpX*off, Y: s.Self.Y + perpY*off,
			VX: ux, VY: uy, OwnerID: ctx.ID, Slot: s.Self.Slot,
		}
	}
	return true
}

func (a *人偶使) castBattle(ctx unit.Context, s unit.Sense) bool {
	for i := 0; i < orbitN; i++ {
		ang := float64(i) * 2 * math.Pi / orbitN
		x, y := math.Cos(ang)*orbitR, math.Sin(ang)*orbitR
		spawnDollAt(ctx, s, x, y, dollSpec{
			mode: dollOrbit, ang: ang, rangeOn: true,
			arriveIn: orbitExpand,
			until:    s.Time + orbitExpand + orbitLife,
		})
	}
	return true
}

func (a *人偶使) castSpirit(ctx unit.Context, s unit.Sense, enemy unit.Snapshot) bool {
	if enemy.ID == 0 {
		return false
	}
	pushEnemy(ctx, s.Self.X, s.Self.Y, enemy, knockDist)
	ctx.Out <- unit.FX{Name: "break", Kind: ctx.Kind, UnitID: ctx.ID, X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot, Amount: knockDist}
	return true
}

func (a *人偶使) spellLevel(sk uint8) int {
	lv := a.levelOf(sk)
	if lv < 1 {
		return 1
	}
	return lv
}

func (a *人偶使) setJob(ctx unit.Context, s unit.Sense, j job) {
	a.abortJobFx(ctx, s)
	a.job = j
	if a.job.kind == SkillAtkFan {
		a.emitFan(ctx, s, true)
	}
}

func (a *人偶使) emitFan(ctx unit.Context, s unit.Sense, on bool) {
	amt := 0.0
	if on {
		amt = a.job.span
	}
	ctx.Out <- unit.FX{
		Name: "fan", Kind: ctx.Kind, UnitID: ctx.ID,
		X: a.job.ox, Y: a.job.oy, VX: a.job.ux, VY: a.job.uy,
		Amount: amt, Slot: s.Self.Slot,
	}
}

func (a *人偶使) abortJobFx(ctx unit.Context, s unit.Sense) {
	if a.job.kind == SkillAtkFan {
		a.emitFan(ctx, s, false)
	}
	if a.job.kind == SkillN24 {
		a.restoreCruise(ctx)
	}
	if a.job.kind == SkillN22 || a.job.kind == CardHourai {
		ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: KindNingyushiBeam}
	}
}

func (a *人偶使) tickJob(ctx unit.Context, s unit.Sense) {
	if a.job.kind == 0 {
		return
	}
	if s.Time+1e-9 >= a.job.until && a.job.left <= 0 {
		if a.job.kind == SkillAtkFan {
			a.emitFan(ctx, s, false)
		}
		if a.job.kind == SkillN22 {
			recallAll(ctx, s)
		}
		if a.job.kind == SkillN24 {
			a.restoreCruise(ctx)
		}
		if a.job.kind == SkillN22 || a.job.kind == CardHourai {
			ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: KindNingyushiBeam}
		}
		a.job = job{}
		return
	}
	switch a.job.kind {
	case SkillAtkFan:
		a.emitFan(ctx, s, true)
		a.tickFan(ctx, s)
	case SkillAtkRect:
		a.tickRect(ctx, s)
	case SkillN24:
		a.tickRush(ctx, s)
		a.tickRect(ctx, s)
	case SkillN22:
		a.tickChain(ctx, s)
	case CardHourai:
		a.tickHourai(ctx, s)
	}
}

func (a *人偶使) tickFan(ctx unit.Context, s unit.Sense) {
	if a.job.left <= 0 || s.Time+1e-9 < a.job.next {
		return
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role != unit.RoleFighter || o.Slot == s.Self.Slot {
			continue
		}
		if fanHit(a.job.ox, a.job.oy, a.job.ux, a.job.uy, a.job.span, a.job.reach, *o) {
			deal(ctx, ctx.ID, o.ID, a.job.dmg)
		}
	}
	a.job.left--
	a.job.next = s.Time + a.job.gap
}

func (a *人偶使) tickRect(ctx unit.Context, s unit.Sense) {
	if a.job.left <= 0 || s.Time+1e-9 < a.job.next {
		return
	}
	ox, oy := a.job.ox, a.job.oy
	w, h := rectW, rectH
	if a.job.kind == SkillN24 {
		w = spell24Len
		ox = s.Self.X + a.job.ux*(w*0.5)
		oy = s.Self.Y + a.job.uy*(w*0.5)
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role != unit.RoleFighter || o.Slot == s.Self.Slot {
			continue
		}
		if !rectHit(ox, oy, a.job.ux, a.job.uy, w, h, *o) {
			continue
		}
		deal(ctx, ctx.ID, o.ID, a.job.dmg)
		if a.job.knock {
			pushEnemy(ctx, s.Self.X, s.Self.Y, *o, knockDist)
			a.stun(ctx, s, o)
		}
	}
	a.job.left--
	a.job.next = s.Time + a.job.gap
}

func (a *人偶使) rush(ctx unit.Context, ux, uy, sp float64) {
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: sp}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: ux * sp, VY: uy * sp}
}

func (a *人偶使) tickRush(ctx unit.Context, s unit.Sense) {
	a.rush(ctx, a.job.ux, a.job.uy, spell24Rush)
}

func (a *人偶使) restoreCruise(ctx unit.Context) {
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: ningyushiSpeed}
}

func (a *人偶使) tickChain(ctx unit.Context, s unit.Sense) {
	if s.Time+1e-9 < a.job.next || a.job.left <= 0 {
		return
	}
	hit := false
	for i := 1; i < len(a.job.paths); i++ {
		x1, y1 := a.job.paths[i-1][0], a.job.paths[i-1][1]
		x2, y2 := a.job.paths[i][0], a.job.paths[i][1]
		ctx.Out <- unit.Spawn{
			Kind: KindNingyushiBeam, X: x1, Y: y1, VX: x2 - x1, VY: y2 - y1,
			OwnerID: ctx.ID, Slot: s.Self.Slot,
		}
		if hit {
			continue
		}
		for j := range s.Nearby {
			o := &s.Nearby[j]
			if o.Role != unit.RoleFighter || o.Slot == s.Self.Slot {
				continue
			}
			if segHits(x1, y1, x2, y2, laserHalf, *o) {
				deal(ctx, ctx.ID, o.ID, a.job.dmg)
				hit = true
				break
			}
		}
	}
	a.job.left = 0
	a.job.idx = len(a.job.paths)
	a.job.until = s.Time + laserHold
}

func (a *人偶使) tickHourai(ctx unit.Context, s unit.Sense) {
	if a.job.left <= 0 || s.Time+1e-9 < a.job.next {
		return
	}
	perpX, perpY := -a.job.uy, a.job.ux
	for b := 0; b < houraiBeams; b++ {
		off := (float64(b) - 1.5) * 14
		ox, oy := s.Self.X+perpX*off, s.Self.Y+perpY*off
		for i := range s.Nearby {
			o := &s.Nearby[i]
			if o.Role != unit.RoleFighter || o.Slot == s.Self.Slot {
				continue
			}
			if laserHits(ox, oy, a.job.ux, a.job.uy, laserHalf, laserLen, *o) {
				deal(ctx, ctx.ID, o.ID, a.job.dmg)
			}
		}
	}
	a.job.left--
	a.job.next = s.Time + a.job.gap
}

func recallAll(ctx unit.Context, s unit.Sense) {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind == KindNingyushiDoll && o.OwnerID == ctx.ID && stateOf(o.ID) == dollIdle {
			orderRecall(o.ID)
		}
	}
}
