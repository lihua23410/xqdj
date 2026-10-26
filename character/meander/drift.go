package 迷走

import (
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

const (
	driftSpeed   = 50.0
	driftLeash   = 50.0
	driftTurnMin = 1.5
	driftTurnMax = 2.5

	omegaMin = 0.25
	omegaMax = 0.45

	// 中点走全图的巡航速度：与一般战斗机（130～195）同量级，先取 150，
	// 与迷走 170 的切向速度错开一点，免得环心平移和绕环锁相。
	midSpeed = 150.0
	// 撞墙改向的冷却。迷走每拍骑轨会反复进出墙的接触区，不挡一下会按拍翻转。
	midTurnCD = 0.5
)

// midPoint 是整张迷径的平移中心：心环锚点钉在它身上，五条轨环再绕心环圆心转。
// 它没有碰撞体，不参与场边扫掠；可行域是场地轮廓内缩（引导圆半径+栓绳），越界自己反射。
type midPoint struct {
	x, y     float64
	vx, vy   float64
	nextTurn float64
}

type ringDrift struct {
	ax, ay   float64
	vx, vy   float64
	nextTurn float64

	track  bool
	guideR float64
	ang    float64
	omega  float64
}

// initDrifts：中点从场心附近的铺图点起步、随机朝向巡航；六条圆心各自的栓绳抖从开局错开。
func initDrifts(m maze, rng *rand.Rand, now float64) ([]ringDrift, midPoint) {
	out := make([]ringDrift, len(m.rings))
	var mid midPoint
	if rng == nil || len(m.rings) == 0 {
		return out, mid
	}
	mid = midPoint{x: m.rings[0].x, y: m.rings[0].y, nextTurn: now + midTurnCD}
	dir := rng.Float64() * 2 * math.Pi
	mid.vx, mid.vy = midSpeed*math.Cos(dir), midSpeed*math.Sin(dir)

	for i, r := range m.rings {
		vdir := rng.Float64() * 2 * math.Pi
		d := ringDrift{
			ax:       r.x,
			ay:       r.y,
			vx:       driftSpeed * math.Cos(vdir),
			vy:       driftSpeed * math.Sin(vdir),
			nextTurn: now + rng.Float64()*driftTurnMax,
			guideR:   m.guideR,
		}
		if i > 0 && m.guideR > 0 {
			d.track = true
			d.ang = math.Atan2(r.y-mid.y, r.x-mid.x)
			span := omegaMax - omegaMin
			mag := omegaMin + rng.Float64()*span
			if rng.Float64() < 0.5 {
				mag = -mag
			}
			d.omega = mag
		}
		out[i] = d
	}
	return out, mid
}

// stepMid：中点按自己的速度走；只碰场边轮廓，不碰内墙。
// 越出内缩域就沿最近边推回，并对该边反射速度。
func stepMid(mid *midPoint, inset float64, f unit.Field, dt float64) {
	if mid == nil || dt <= 0 {
		return
	}
	mid.x += mid.vx * dt
	mid.y += mid.vy * dt
	if f.OutlineContains(mid.x, mid.y, inset) {
		return
	}
	nx, ny := f.NearestEdgeDir(mid.x, mid.y)
	mid.x, mid.y = pushInside(f, mid.x, mid.y, inset)
	reflectMid(mid, nx, ny)
}

// pushInside：沿最近边内法线一步步退，直到点周围 inset 圆落回轮廓内。
func pushInside(f unit.Field, x, y, inset float64) (float64, float64) {
	for i := 0; i < 64; i++ {
		if f.OutlineContains(x, y, inset) {
			return x, y
		}
		nx, ny := f.NearestEdgeDir(x, y)
		x -= nx * 2
		y -= ny * 2
	}
	return 0, 0
}

// reflectMid：速度对着法线做镜面反射；方向正负不影响结果（与引擎 reflectVelocity 同式）。
func reflectMid(p *midPoint, nx, ny float64) {
	l := math.Hypot(nx, ny)
	if l < 1e-9 {
		return
	}
	nx, ny = nx/l, ny/l
	if vr := p.vx*nx + p.vy*ny; vr != 0 {
		p.vx -= 2 * vr * nx
		p.vy -= 2 * vr * ny
	}
}

// driftCenters：中点在走；心环锚点钉在中点上，轨环锚点绕心环圆心转；
// 六条圆心相对各自当下锚点再栓绳抖。
func driftCenters(rings []ring, drifts []ringDrift, mid *midPoint, f unit.Field, rng *rand.Rand, now, dt float64) {
	if dt <= 0 || len(rings) == 0 || len(drifts) != len(rings) || rng == nil || mid == nil {
		return
	}
	stepMid(mid, drifts[0].guideR+driftLeash, f, dt)

	step := func(i int, gx, gy float64) {
		d := &drifts[i]
		if d.track {
			oldAx, oldAy := d.ax, d.ay
			d.ang += d.omega * dt
			d.ax = gx + d.guideR*math.Cos(d.ang)
			d.ay = gy + d.guideR*math.Sin(d.ang)
			// 锚点带走相对位移，再叠栓绳抖
			rings[i].x += d.ax - oldAx
			rings[i].y += d.ay - oldAy
		}
		if now+1e-9 >= d.nextTurn {
			ang := rng.Float64() * 2 * math.Pi
			d.vx = driftSpeed * math.Cos(ang)
			d.vy = driftSpeed * math.Sin(ang)
			span := driftTurnMax - driftTurnMin
			d.nextTurn = now + driftTurnMin + rng.Float64()*span
		}
		nx := rings[i].x + d.vx*dt
		ny := rings[i].y + d.vy*dt
		nx, ny = constrainCenter(nx, ny, d, f)
		rings[i].x, rings[i].y = nx, ny
	}

	// 心环：锚点先跟着中点整段平移，再抖自己的 50。
	d0 := &drifts[0]
	oldAx, oldAy := d0.ax, d0.ay
	d0.ax, d0.ay = mid.x, mid.y
	rings[0].x += d0.ax - oldAx
	rings[0].y += d0.ay - oldAy
	step(0, 0, 0)

	gx, gy := rings[0].x, rings[0].y
	for i := 1; i < len(rings); i++ {
		step(i, gx, gy)
	}
}

func constrainCenter(x, y float64, d *ringDrift, f unit.Field) (float64, float64) {
	x, y = leashPull(x, y, d)
	if f.Walkable(x, y, centerPad) {
		return x, y
	}
	d.vx, d.vy = -d.vx, -d.vy
	cx, cy := f.Clamp(x, y, centerPad)
	if f.Walkable(cx, cy, centerPad) {
		return leashPull(cx, cy, d)
	}
	bx := d.ax + (x-d.ax)*0.5
	by := d.ay + (y-d.ay)*0.5
	bx, by = f.Clamp(bx, by, centerPad)
	if f.Walkable(bx, by, centerPad) {
		return leashPull(bx, by, d)
	}
	return d.ax, d.ay
}

func leashPull(x, y float64, d *ringDrift) (float64, float64) {
	dx, dy := x-d.ax, y-d.ay
	dist := math.Hypot(dx, dy)
	if dist <= driftLeash+1e-9 || dist < 1e-9 {
		return x, y
	}
	s := driftLeash / dist
	x, y = d.ax+dx*s, d.ay+dy*s
	bounceAway(d, dx, dy)
	return x, y
}

// bounceAway：速度在朝外径向分量上反弹（圆心边界反弹，不使人反绕）。
func bounceAway(d *ringDrift, dx, dy float64) {
	dist := math.Hypot(dx, dy)
	if dist < 1e-9 {
		d.vx, d.vy = -d.vx, -d.vy
		return
	}
	ux, uy := dx/dist, dy/dist
	vr := d.vx*ux + d.vy*uy
	if vr > 0 {
		d.vx -= 2 * vr * ux
		d.vy -= 2 * vr * uy
	}
}
