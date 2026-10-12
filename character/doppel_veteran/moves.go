package 分身者_百战

import (
	"math"
	"sort"

	"xqdj/internal/unit"
)

func (a *百战) startWei(ctx unit.Context, s unit.Sense, shadows []unit.Snapshot) {
	a.clearPasses(ctx, shadows)
	a.clearStands(ctx, shadows)
	a.centerFixed = false
	// 开招即锁当前索敌目标，整段围刺不换锁
	if en := unit.Seek(s); en != nil {
		a.lockID = en.ID
		a.aimX, a.aimY = en.X, en.Y
		a.liveX, a.liveY = en.X, en.Y
		a.hasAim = true
	}
	a.assignOrbitAngles(shadows)
	a.gatherFrom = s.Time
	a.phase = phaseWeiGather
	a.tickWeiGather(ctx, s, shadows, 0)
}

// tickWeiGather：跑向环位；已到的先跟着转，等齐（或超时）再进完整绕转。
func (a *百战) tickWeiGather(ctx unit.Context, s unit.Sense, shadows []unit.Snapshot, dt float64) {
	cx, cy := a.center() // 聚拢阶段仍跟敌
	omega := 2 * math.Pi / weiPeriod
	for id := range a.orbitAng {
		a.orbitAng[id] += omega * dt
	}
	ready := true
	spots := map[uint64]ringSpot{}
	step := dieGatherSp * dt
	if step < 1 {
		step = dieGatherSp / 60
	}
	for _, sh := range shadows {
		ang := a.orbitAng[sh.ID]
		tx := cx + math.Cos(ang)*weiRadius
		ty := cy + math.Sin(ang)*weiRadius
		tx, ty = s.Field.Clamp(tx, ty, shadowRadius)
		dx, dy := tx-sh.X, ty-sh.Y
		dist := math.Hypot(dx, dy)
		if dist <= orbitArriveR {
			// 已到环跟着转：实心，不带 Pass
			a.setPass(ctx, sh.ID, false)
			x, y := placeOnOrbit(ctx, sh.ID, cx, cy, ang, weiRadius, omega)
			spots[sh.ID] = ringSpot{x: x, y: y, r: shadowRadius}
			continue
		}
		ready = false
		if step >= dist {
			a.setPass(ctx, sh.ID, false)
			x, y := placeOnOrbit(ctx, sh.ID, cx, cy, ang, weiRadius, omega)
			spots[sh.ID] = ringSpot{x: x, y: y, r: shadowRadius}
			continue
		}
		// 仅走位未到环者持 Pass
		a.setPass(ctx, sh.ID, true)
		ux, uy := dx/dist, dy/dist
		nx, ny := s.Field.Clamp(sh.X+ux*step, sh.Y+uy*step, shadowRadius)
		ctx.Out <- unit.Teleport{UnitID: sh.ID, X: nx, Y: ny}
		ctx.Out <- unit.SetVelocity{UnitID: sh.ID, VX: ux * dieGatherSp, VY: uy * dieGatherSp}
	}
	a.scrapeOrbitAt(ctx, s, spots, weiOrbitDmg)
	if !ready && s.Time+timeEps >= a.gatherFrom+weiGatherMax {
		ready = true
	}
	if !ready {
		return
	}
	for _, sh := range shadows {
		a.setPass(ctx, sh.ID, false)
	}
	a.phase = phaseWeiOrbit
	a.phaseUntil = s.Time + weiOrbitDur
	a.tickWeiOrbit(ctx, s, shadows, 0)
}

func (a *百战) tickWeiOrbit(ctx unit.Context, s unit.Sense, shadows []unit.Snapshot, dt float64) {
	cx, cy := a.center()
	omega := 2 * math.Pi / weiPeriod
	spots := map[uint64]ringSpot{}
	for _, sh := range shadows {
		ang := a.orbitAng[sh.ID]
		ang += omega * dt
		a.orbitAng[sh.ID] = ang
		x, y := placeOnOrbit(ctx, sh.ID, cx, cy, ang, weiRadius, omega)
		spots[sh.ID] = ringSpot{x: x, y: y, r: shadowRadius}
	}
	a.scrapeOrbitAt(ctx, s, spots, weiOrbitDmg)
	if s.Time+timeEps >= a.phaseUntil {
		a.freezeCenter()
		a.phase = phaseWeiPause
		a.phaseUntil = s.Time + weiPauseDur
	}
}

func (a *百战) tickWeiPause(ctx unit.Context, s unit.Sense, shadows []unit.Snapshot) {
	cx, cy := a.center() // 已 freeze，不再跟敌
	spots := map[uint64]ringSpot{}
	for _, sh := range shadows {
		ang := a.orbitAng[sh.ID]
		x := cx + math.Cos(ang)*weiRadius
		y := cy + math.Sin(ang)*weiRadius
		ctx.Out <- unit.Teleport{UnitID: sh.ID, X: x, Y: y}
		ctx.Out <- unit.SetVelocity{UnitID: sh.ID, VX: 0, VY: 0}
		spots[sh.ID] = ringSpot{x: x, y: y, r: shadowRadius}
	}
	a.scrapeOrbitAt(ctx, s, spots, weiOrbitDmg)
	if s.Time+timeEps < a.phaseUntil {
		return
	}
	a.clearDashWait()
	a.dashIDs = make([]uint64, 0, len(shadows))
	for _, sh := range shadows {
		ang := a.orbitAng[sh.ID]
		x := cx + math.Cos(ang)*weiRadius
		y := cy + math.Sin(ang)*weiRadius
		dx, dy := cx-x, cy-y
		dist := math.Hypot(dx, dy) + weiOvershoot
		if dist < weiOvershoot {
			dist = weiOvershoot
		}
		startDash(sh.ID, dx, dy, weiDashSp, dist, weiDamage, true, true, a.slot)
		a.dashIDs = append(a.dashIDs, sh.ID)
	}
	a.phase = phaseWeiDash
	a.driveDashes(ctx, s)
}

func (a *百战) startZhou(ctx unit.Context, s unit.Sense, shadows []unit.Snapshot) {
	a.clearPasses(ctx, shadows)
	a.clearStands(ctx, shadows)
	a.axisID = shadows[a.rng.IntN(len(shadows))].ID
	orbiters := make([]unit.Snapshot, 0, len(shadows)-1)
	for _, sh := range shadows {
		if sh.ID != a.axisID {
			orbiters = append(orbiters, sh)
		}
	}
	a.orbitAng = map[uint64]float64{}
	n := len(orbiters)
	for i, sh := range orbiters {
		if n == 0 {
			break
		}
		a.orbitAng[sh.ID] = float64(i) * 2 * math.Pi / float64(n)
	}
	a.gatherFrom = s.Time
	a.phase = phaseZhouGather
	a.tickZhouGather(ctx, s, shadows, 0)
}

func (a *百战) axisPos(shadows []unit.Snapshot) (unit.Snapshot, bool) {
	for _, sh := range shadows {
		if sh.ID == a.axisID {
			return sh, true
		}
	}
	return unit.Snapshot{}, false
}

// tickZhouGather：轴 a 站住；其余按招3同款步进走向自己的环位点，到齐再进准备。
func (a *百战) tickZhouGather(ctx unit.Context, s unit.Sense, shadows []unit.Snapshot, dt float64) {
	ax, ok := a.axisPos(shadows)
	if !ok {
		if len(shadows) == 0 {
			a.beginReform(ctx, s.Time, shadows)
			return
		}
		a.axisID = shadows[0].ID
		ax = shadows[0]
		orbiters := make([]unit.Snapshot, 0, len(shadows)-1)
		for _, sh := range shadows {
			if sh.ID != a.axisID {
				orbiters = append(orbiters, sh)
			}
		}
		a.orbitAng = map[uint64]float64{}
		n := len(orbiters)
		for i, sh := range orbiters {
			if n == 0 {
				break
			}
			a.orbitAng[sh.ID] = float64(i) * 2 * math.Pi / float64(n)
		}
	}
	a.setStand(ctx, ax.ID, true)
	ctx.Out <- unit.SetVelocity{UnitID: ax.ID, VX: 0, VY: 0}

	omega := 2 * math.Pi / zhouPeriod
	for id := range a.orbitAng {
		a.orbitAng[id] += omega * dt
	}

	ready := true
	spots := map[uint64]ringSpot{}
	step := dieGatherSp * dt
	if step < 1 {
		step = dieGatherSp / 60
	}
	for _, sh := range shadows {
		if sh.ID == a.axisID {
			continue
		}
		ang := a.orbitAng[sh.ID]
		tx := ax.X + math.Cos(ang)*zhouRadius
		ty := ax.Y + math.Sin(ang)*zhouRadius
		tx, ty = s.Field.Clamp(tx, ty, shadowRadius)
		dx, dy := tx-sh.X, ty-sh.Y
		dist := math.Hypot(dx, dy)
		a.setPass(ctx, sh.ID, true)
		if dist <= orbitArriveR {
			x, y := placeOnOrbit(ctx, sh.ID, ax.X, ax.Y, ang, zhouRadius, omega)
			spots[sh.ID] = ringSpot{x: x, y: y, r: shadowRadius}
			continue
		}
		ready = false
		if step >= dist {
			x, y := placeOnOrbit(ctx, sh.ID, ax.X, ax.Y, ang, zhouRadius, omega)
			spots[sh.ID] = ringSpot{x: x, y: y, r: shadowRadius}
			continue
		}
		ux, uy := dx/dist, dy/dist
		nx, ny := s.Field.Clamp(sh.X+ux*step, sh.Y+uy*step, shadowRadius)
		ctx.Out <- unit.Teleport{UnitID: sh.ID, X: nx, Y: ny}
		ctx.Out <- unit.SetVelocity{UnitID: sh.ID, VX: ux * dieGatherSp, VY: uy * dieGatherSp}
	}
	a.scrapeOrbitAt(ctx, s, spots, zhouOrbitDmg)
	if !ready && s.Time+timeEps >= a.gatherFrom+zhouGatherMax {
		ready = true // 墙边夹死：超时强行开转
	}
	if !ready {
		return
	}
	for _, sh := range shadows {
		if sh.ID == a.axisID {
			continue
		}
		a.setPass(ctx, sh.ID, false)
	}
	a.phase = phaseZhouAim
	a.phaseUntil = s.Time + zhouStand
	a.tickZhouAim(ctx, s, shadows, 0)
}

func (a *百战) tickZhouAim(ctx unit.Context, s unit.Sense, shadows []unit.Snapshot, dt float64) {
	ax, ok := a.axisPos(shadows)
	if !ok {
		if len(shadows) == 0 {
			a.beginReform(ctx, s.Time, shadows)
			return
		}
		a.axisID = shadows[0].ID
		ax = shadows[0]
	}
	a.setStand(ctx, ax.ID, true)
	ctx.Out <- unit.SetVelocity{UnitID: ax.ID, VX: 0, VY: 0}
	omega := 2 * math.Pi / zhouPeriod
	spots := map[uint64]ringSpot{}
	for _, sh := range shadows {
		if sh.ID == a.axisID {
			continue
		}
		ang := a.orbitAng[sh.ID]
		ang += omega * dt
		a.orbitAng[sh.ID] = ang
		x, y := placeOnOrbit(ctx, sh.ID, ax.X, ax.Y, ang, zhouRadius, omega)
		spots[sh.ID] = ringSpot{x: x, y: y, r: shadowRadius}
	}
	a.scrapeOrbitAt(ctx, s, spots, zhouOrbitDmg)
	if s.Time+timeEps < a.phaseUntil {
		return
	}
	a.setStand(ctx, ax.ID, false)
	cx, cy := a.center()
	dx, dy := cx-ax.X, cy-ax.Y
	a.clearDashWait()
	startDash(ax.ID, dx, dy, zhouDashSp, zhouDashLen, zhouDamage, false, false, a.slot)
	a.dashIDs = []uint64{ax.ID}
	a.phase = phaseZhouDash
	a.driveDashes(ctx, s)
}

func (a *百战) tickZhouDash(ctx unit.Context, s unit.Sense, shadows []unit.Snapshot, dt float64) {
	// 轴已走满：先收招，同帧不再 placeOnOrbit，避免切向速盖掉 kickCruise
	if dashesDone(a.dashIDs) {
		a.beginReform(ctx, s.Time, shadows)
		return
	}
	ax, ok := a.axisPos(shadows)
	if !ok {
		return
	}
	omega := 2 * math.Pi / zhouPeriod
	spots := map[uint64]ringSpot{}
	for _, sh := range shadows {
		if sh.ID == a.axisID {
			continue
		}
		ang := a.orbitAng[sh.ID]
		ang += omega * dt
		a.orbitAng[sh.ID] = ang
		x, y := placeOnOrbit(ctx, sh.ID, ax.X, ax.Y, ang, zhouRadius, omega)
		spots[sh.ID] = ringSpot{x: x, y: y, r: shadowRadius}
	}
	a.scrapeOrbitAt(ctx, s, spots, zhouOrbitDmg)
}

func (a *百战) startDie(ctx unit.Context, s unit.Sense, shadows []unit.Snapshot) {
	a.stackID = shadows[a.rng.IntN(len(shadows))].ID
	a.rebuildDieOrder(shadows)
	a.shotIdx = 0
	a.stackFixed = false
	a.gatherFrom = s.Time
	a.clearDashWait()
	a.dashIDs = nil
	a.phase = phaseDieGather
	a.tickDieGather(ctx, s, shadows, 0)
}

func (a *百战) stackPos(shadows []unit.Snapshot) (unit.Snapshot, bool) {
	for _, sh := range shadows {
		if sh.ID == a.stackID {
			return sh, true
		}
	}
	return unit.Snapshot{}, false
}

func (a *百战) rebuildDieOrder(shadows []unit.Snapshot) {
	rest := make([]uint64, 0, len(shadows))
	for _, sh := range shadows {
		if sh.ID != a.stackID {
			rest = append(rest, sh.ID)
		}
	}
	sort.Slice(rest, func(i, j int) bool { return rest[i] < rest[j] })
	a.order = append(rest, a.stackID)
}

func (a *百战) tickDieGather(ctx unit.Context, s unit.Sense, shadows []unit.Snapshot, dt float64) {
	bx, ok := a.stackPos(shadows)
	if !ok {
		if len(shadows) == 0 {
			a.beginReform(ctx, s.Time, shadows)
			return
		}
		a.stackID = shadows[0].ID
		bx = shadows[0]
		a.rebuildDieOrder(shadows)
	}
	a.setPass(ctx, bx.ID, true)
	a.setStand(ctx, bx.ID, true)
	setStackGuard(bx.ID, true) // 叠点本体重叠期挡伤
	ctx.Out <- unit.SetVelocity{UnitID: bx.ID, VX: 0, VY: 0}

	ready := true
	step := dieGatherSp * dt
	if step < 1 {
		step = dieGatherSp / 60
	}
	for _, sh := range shadows {
		if sh.ID == a.stackID {
			continue
		}
		a.setPass(ctx, sh.ID, true)
		dx, dy := bx.X-sh.X, bx.Y-sh.Y
		dist := math.Hypot(dx, dy)
		if dist <= dieStackR {
			ctx.Out <- unit.Teleport{UnitID: sh.ID, X: bx.X, Y: bx.Y}
			ctx.Out <- unit.SetVelocity{UnitID: sh.ID, VX: 0, VY: 0}
			a.setStand(ctx, sh.ID, true)
			setStackGuard(sh.ID, true)
			continue
		}
		ready = false
		setStackGuard(sh.ID, false) // 还在路上，可受伤
		if step >= dist {
			ctx.Out <- unit.Teleport{UnitID: sh.ID, X: bx.X, Y: bx.Y}
			ctx.Out <- unit.SetVelocity{UnitID: sh.ID, VX: 0, VY: 0}
			setStackGuard(sh.ID, true)
			continue
		}
		ux, uy := dx/dist, dy/dist
		ctx.Out <- unit.Teleport{UnitID: sh.ID, X: sh.X + ux*step, Y: sh.Y + uy*step}
		ctx.Out <- unit.SetVelocity{UnitID: sh.ID, VX: ux * dieGatherSp, VY: uy * dieGatherSp}
	}
	if !ready && s.Time+timeEps >= a.gatherFrom+dieGatherMax {
		ready = true
	}
	if !ready {
		return
	}
	// 停住：钉死叠点，之后不再跟 b 活坐标追
	a.stackX, a.stackY = bx.X, bx.Y
	a.stackFixed = true
	for _, sh := range shadows {
		a.setPass(ctx, sh.ID, true)
		a.setStand(ctx, sh.ID, true)
		setStackGuard(sh.ID, true)
		ctx.Out <- unit.Teleport{UnitID: sh.ID, X: a.stackX, Y: a.stackY}
		ctx.Out <- unit.SetVelocity{UnitID: sh.ID, VX: 0, VY: 0}
	}
	a.phase = phaseDieWait
	a.phaseUntil = s.Time + dieWait
}

func (a *百战) dieStackPoint(shadows []unit.Snapshot) (x, y float64, ok bool) {
	if a.stackFixed {
		return a.stackX, a.stackY, true
	}
	bx, ok := a.stackPos(shadows)
	if !ok {
		return 0, 0, false
	}
	return bx.X, bx.Y, true
}

func (a *百战) tickDieWait(ctx unit.Context, s unit.Sense, shadows []unit.Snapshot) {
	sx, sy, ok := a.dieStackPoint(shadows)
	if !ok {
		a.beginReform(ctx, s.Time, shadows)
		return
	}
	for _, sh := range shadows {
		a.setPass(ctx, sh.ID, true)
		a.setStand(ctx, sh.ID, true)
		setStackGuard(sh.ID, true)
		ctx.Out <- unit.Teleport{UnitID: sh.ID, X: sx, Y: sy}
		ctx.Out <- unit.SetVelocity{UnitID: sh.ID, VX: 0, VY: 0}
	}
	if s.Time+timeEps < a.phaseUntil {
		return
	}
	a.shotIdx = 0
	a.nextShot = s.Time
	a.dashIDs = nil
	a.phase = phaseDieVolley
	a.tickDieVolley(ctx, s, shadows)
}

func (a *百战) tickDieVolley(ctx unit.Context, s unit.Sense, shadows []unit.Snapshot) {
	live := map[uint64]unit.Snapshot{}
	for _, sh := range shadows {
		live[sh.ID] = sh
	}

	// 先发射，再钉住未发射的——避免同帧 Vel0 盖掉冲刺速度
	for a.shotIdx < len(a.order) && s.Time+timeEps >= a.nextShot {
		id := a.order[a.shotIdx]
		a.shotIdx++
		a.nextShot = s.Time + dieGap
		sh, ok := live[id]
		if !ok {
			continue
		}
		a.setPass(ctx, id, false)
		a.setStand(ctx, id, false)
		setStackGuard(id, false) // 撞出去后恢复可受伤
		cx, cy := a.center()
		dx, dy := cx-sh.X, cy-sh.Y
		startDash(id, dx, dy, dieDashSp, dieDashLen, dieDamage, false, false, a.slot)
		a.dashIDs = append(a.dashIDs, id)
	}
	a.driveDashes(ctx, s)

	fired := map[uint64]struct{}{}
	for i := 0; i < a.shotIdx && i < len(a.order); i++ {
		fired[a.order[i]] = struct{}{}
	}
	sx, sy, pinOK := a.dieStackPoint(shadows)
	for _, sh := range shadows {
		if _, ok := fired[sh.ID]; ok {
			continue
		}
		if j := getDash(sh.ID); j != nil && !j.isDone() {
			continue
		}
		a.setPass(ctx, sh.ID, true)
		a.setStand(ctx, sh.ID, true)
		setStackGuard(sh.ID, true)
		if pinOK {
			ctx.Out <- unit.Teleport{UnitID: sh.ID, X: sx, Y: sy}
		}
		ctx.Out <- unit.SetVelocity{UnitID: sh.ID, VX: 0, VY: 0}
	}

	if a.shotIdx < len(a.order) {
		return
	}
	if dashesDone(a.dashIDs) {
		a.beginReform(ctx, s.Time, shadows)
	}
}
