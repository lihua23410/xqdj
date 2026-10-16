package 分身者_百战

import (
	"math"
	"sync"

	"xqdj/internal/unit"
)

type dashJob struct {
	mu     sync.Mutex
	speed  float64
	left   float64
	ux, uy float64
	dmg    float64
	block  bool
	pass   bool
	slot   int
	cruise float64
	hit    map[uint64]bool
	lastT  float64
	done   bool
	booted bool
}

var dashJobs sync.Map // uint64 -> *dashJob

// stackGuard：招3重叠钉住时挡伤。
var stackGuard sync.Map // uint64 -> struct{}

func setStackGuard(id uint64, on bool) {
	if on {
		stackGuard.Store(id, struct{}{})
		return
	}
	stackGuard.Delete(id)
}

func isStackGuarded(id uint64) bool {
	_, ok := stackGuard.Load(id)
	return ok
}

func clearAllStackGuards() {
	stackGuard.Range(func(k, _ any) bool {
		stackGuard.Delete(k)
		return true
	})
}

func startDash(id uint64, ux, uy, speed, dist, dmg float64, block, pass bool, slot int) {
	n := math.Hypot(ux, uy)
	if n < 1e-6 {
		ux, uy = 1, 0
		n = 1
	}
	ux, uy = ux/n, uy/n
	dashJobs.Store(id, &dashJob{
		speed:  speed,
		left:   dist,
		ux:     ux,
		uy:     uy,
		dmg:    dmg,
		block:  block,
		pass:   pass,
		slot:   slot,
		cruise: shadowCruise,
		hit:    map[uint64]bool{},
	})
}

func clearDash(id uint64) {
	dashJobs.Delete(id)
}

func getDash(id uint64) *dashJob {
	v, ok := dashJobs.Load(id)
	if !ok {
		return nil
	}
	return v.(*dashJob)
}

func dashesDone(ids []uint64) bool {
	if len(ids) == 0 {
		return true
	}
	for _, id := range ids {
		j := getDash(id)
		if j == nil {
			continue
		}
		j.mu.Lock()
		done := j.done
		j.mu.Unlock()
		if !done {
			return false
		}
	}
	return true
}

func (j *dashJob) blocking() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return !j.done && j.block
}

func (j *dashJob) isDone() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.done
}

func (j *dashJob) finish(ctx unit.Context, id uint64, now float64, wall bool, nx, ny float64) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.finishLocked(ctx, id, now, wall, nx, ny)
}

// finishLocked：持锁收招。须在持锁期间写出巡航速度，避免与 tick 的冲刺速乱序。
func (j *dashJob) finishLocked(ctx unit.Context, id uint64, now float64, wall bool, nx, ny float64) {
	if j.done {
		return
	}
	j.done = true
	ux, uy := j.ux, j.uy
	pass := j.pass
	cruise := j.cruise
	if wall {
		ux, uy = reflectDir(ux, uy, nx, ny)
	}
	if pass {
		ctx.Out <- unit.Pass{UnitID: id, Hold: false}
	}
	ctx.Out <- unit.SetVelocity{UnitID: id, VX: ux * cruise, VY: uy * cruise}
	_ = now
}

func (j *dashJob) tick(ctx unit.Context, id uint64, s unit.Sense) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.done {
		return
	}
	if !j.booted {
		j.booted = true
		j.lastT = s.Time
		if j.pass {
			ctx.Out <- unit.Pass{UnitID: id, Hold: true}
		}
	}
	dt := s.Time - j.lastT
	if dt < 0 {
		dt = 0
	}
	j.lastT = s.Time
	j.left -= j.speed * dt
	if j.left <= 0 {
		j.finishLocked(ctx, id, s.Time, false, 0, 0)
		return
	}
	// 持锁写冲刺速：WallHit/finish 必须等这帧写完再 snap，巡航才能盖在最后
	ctx.Out <- unit.SetVelocity{UnitID: id, VX: j.ux * j.speed, VY: j.uy * j.speed}
}

func (j *dashJob) onCollision(ctx unit.Context, id uint64, e unit.Collision) {
	j.mu.Lock()
	if j.done || !unit.EnemyTarget(e, j.slot) {
		j.mu.Unlock()
		return
	}
	if j.hit[e.Other.ID] {
		j.mu.Unlock()
		return
	}
	j.hit[e.Other.ID] = true
	dmg := j.dmg
	j.mu.Unlock()
	ctx.Out <- unit.Damage{From: id, To: e.Other.ID, Amount: dmg}
}

func reflectDir(ux, uy, nx, ny float64) (float64, float64) {
	dot := ux*nx + uy*ny
	rx, ry := ux-2*dot*nx, uy-2*dot*ny
	n := math.Hypot(rx, ry)
	if n < 1e-6 {
		return ux, uy
	}
	return rx / n, ry / n
}
