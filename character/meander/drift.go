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
)

type ringDrift struct {
	ax, ay   float64
	vx, vy   float64
	nextTurn float64

	track  bool
	guideR float64
	ang    float64
	omega  float64
}

func initDrifts(m maze, rng *rand.Rand, now float64) []ringDrift {
	out := make([]ringDrift, len(m.rings))
	for i, r := range m.rings {
		dir := rng.Float64() * 2 * math.Pi
		d := ringDrift{
			ax:       r.x,
			ay:       r.y,
			vx:       driftSpeed * math.Cos(dir),
			vy:       driftSpeed * math.Sin(dir),
			nextTurn: now + rng.Float64()*driftTurnMax,
		}
		if i > 0 && m.guideR > 0 {
			d.track = true
			d.guideR = m.guideR
			d.ang = math.Atan2(r.y, r.x)
			span := omegaMax - omegaMin
			mag := omegaMin + rng.Float64()*span
			if rng.Float64() < 0.5 {
				mag = -mag
			}
			d.omega = mag
		}
		out[i] = d
	}
	return out
}

// driftCenters：轨环锚点沿引导圆滑；六条圆心相对当下锚点栓绳抖。
func driftCenters(rings []ring, drifts []ringDrift, f unit.Field, rng *rand.Rand, now, dt float64) {
	if dt <= 0 || len(rings) == 0 || len(drifts) != len(rings) || rng == nil {
		return
	}
	for i := range rings {
		d := &drifts[i]
		if d.track {
			oldAx, oldAy := d.ax, d.ay
			d.ang += d.omega * dt
			d.ax = d.guideR * math.Cos(d.ang)
			d.ay = d.guideR * math.Sin(d.ang)
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
