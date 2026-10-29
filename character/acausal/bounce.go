package 非因果

import (
	"math"

	"xqdj/internal/unit"
)

type ghostPath struct {
	id                 uint64
	x, y, nextX, nextY float64
	r                  float64
}

// 影继续走探针轨迹；只有非因果本体响应接触，不改变影的实体或运动规则。
func (a *非因果) bounceGhosts(ctx unit.Context, s unit.Sense) {
	vx, vy := s.Self.VX, s.Self.VY
	speed := math.Hypot(vx, vy)
	if speed < 1e-9 {
		a.contacts = nil
		return
	}
	radius := s.Self.Radius
	if radius <= 0 {
		radius = bodyRadius
	}
	contacts := map[uint64]bool{}
	bestTime := 2.0
	var bestID uint64
	var normalX, normalY float64
	for _, g := range a.ghostPaths(ctx.ID, s) {
		x, y := s.Self.X-g.x, s.Self.Y-g.y
		reach := radius + g.r
		overlap := x*x+y*y <= reach*reach+1e-9
		if overlap {
			contacts[g.id] = true
			if a.contacts[g.id] {
				continue
			}
		}
		// 用相对运动扫下一拍，避免高速接近或影掠过时漏掉接触。
		nextX := x + vx*tickDT - (g.nextX - g.x)
		nextY := y + vy*tickDT - (g.nextY - g.y)
		t, hit := shotHitsCircle(x, y, nextX, nextY, 0, 0, reach)
		if !hit {
			continue
		}
		nx, ny := x+(nextX-x)*t, y+(nextY-y)*t
		if n := math.Hypot(nx, ny); n > 1e-9 {
			nx, ny = nx/n, ny/n
		} else {
			// 圆心完全重合时没有几何法线，沿来向退回；接触锁防止连拍翻转。
			nx, ny = -vx/speed, -vy/speed
		}
		if vx*nx+vy*ny >= -1e-9 {
			continue // 已在离开影，不能再把本体反弹回去。
		}
		if t < bestTime-1e-9 || (math.Abs(t-bestTime) <= 1e-9 && g.id < bestID) {
			bestTime, bestID = t, g.id
			normalX, normalY = nx, ny
		}
	}
	a.contacts = contacts
	if bestID == 0 {
		return
	}
	a.contacts[bestID] = true
	dot := vx*normalX + vy*normalY
	// 直接施加反射所需的速度增量，避免 SetVelocity 再乘一次已有的巡航倍率。
	ctx.Out <- unit.Force{UnitID: ctx.ID, AX: -2 * dot * normalX / tickDT, AY: -2 * dot * normalY / tickDT}
}

func (a *非因果) ghostPaths(owner uint64, s unit.Sense) []ghostPath {
	live := map[uint64]bool{}
	for _, o := range s.Nearby {
		if o.OwnerID == owner {
			live[o.ID] = true
		}
	}
	var paths []ghostPath
	add := func(g ghostRef, samples []Sample, radius float64) {
		if !g.ready || g.dead || !live[g.id] || len(samples) == 0 {
			return
		}
		x, y := sampleAt(samples, s.Time)
		nx, ny := sampleAt(samples, s.Time+tickDT)
		paths = append(paths, ghostPath{id: g.id, x: x, y: y, nextX: nx, nextY: ny, r: radius})
	}
	add(a.self, a.tape.Self, bodyRadius)
	for i, g := range a.fighters {
		radius := a.tape.Fighters[i].Radius
		if radius <= 0 {
			radius = bodyRadius
		}
		add(g, a.tape.Fighters[i].Samples, radius)
	}
	for i, g := range a.minions {
		radius := a.tape.Minions[i].Radius
		if radius <= 0 {
			radius = 14
		}
		add(g, a.tape.Minions[i].Samples, radius)
	}
	return paths
}
