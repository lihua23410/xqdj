package 钓鱼佬

import (
	"math"
	"math/rand/v2"
	"sync"
	"xqdj/internal/unit"
)

type 鱼塘 struct{}

func (鱼塘) Handle(unit.Context, unit.Event) {}

type fishJob struct {
	y     float64
	dmg   float64
	speed float64
	spent bool
}

var fishQ struct {
	sync.Mutex
	jobs []fishJob
}

func pushFish(j fishJob) {
	fishQ.Lock()
	fishQ.jobs = append(fishQ.jobs, j)
	fishQ.Unlock()
}

func popFish() fishJob {
	fishQ.Lock()
	defer fishQ.Unlock()
	if len(fishQ.jobs) == 0 {
		return fishJob{y: 10, dmg: fishDamage(10), speed: fishSpeed(10)}
	}
	j := fishQ.jobs[0]
	fishQ.jobs = fishQ.jobs[1:]
	return j
}

func resetFishQ() {
	fishQ.Lock()
	fishQ.jobs = nil
	fishQ.Unlock()
}

type 鱼 struct {
	job     fishJob
	owner   uint64
	slot    int
	bounced int
	booted  bool
	passed  bool
	spent   bool
	steady  bool
	speed   float64
	drag    float64
	lastT   float64
	x, y    float64
	ux, uy  float64
	ex, ey  float64
	hasE    bool
}

func newFish(info unit.SpawnInfo) *鱼 {
	j := popFish()
	return &鱼{job: j, owner: info.OwnerID, slot: info.Slot, spent: j.spent}
}

func (f *鱼) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		f.onSense(ctx, e)
	case unit.Collision:
		f.onHit(ctx, e)
	case unit.WallHit:
		f.onWall(ctx, e)
	}
}

func (f *鱼) onSense(ctx unit.Context, s unit.Sense) {
	dt := s.Time - f.lastT
	if dt < 0 {
		dt = 0
	}
	if dt > 0.2 {
		dt = 0.2
	}
	f.lastT = s.Time
	f.x, f.y = s.Self.X, s.Self.Y
	if e := enemyOf(s); e != nil {
		f.ex, f.ey = e.X, e.Y
		f.hasE = true
	}
	if !f.booted {
		f.booted = true
		n := math.Hypot(s.Self.VX, s.Self.VY)
		if n > 1e-6 {
			f.speed = n
			f.ux, f.uy = s.Self.VX/n, s.Self.VY/n
		} else {
			f.speed = f.job.speed
			f.ux, f.uy = 1, 0
		}
		if f.hasE && !f.spent {
			f.chase(ctx)
		}
	}
	if !f.passed {
		f.passed = true
		ctx.Out <- unit.Pass{UnitID: ctx.ID, Hold: true}
	}
	if f.spent {
		f.coast(ctx, s, dt)
	}
	ctx.Out <- unit.FX{
		Name: "weight", Kind: ctx.Kind, UnitID: ctx.ID,
		Amount: f.job.y, Slot: s.Self.Slot,
	}
}

func (f *鱼) onHit(ctx unit.Context, c unit.Collision) {
	if f.spent {
		return
	}
	o := c.Other
	if o.ID == f.owner {
		return
	}
	if !unit.Hittable(o, f.slot) {
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: f.job.dmg}
	f.goInert()
}

func (f *鱼) onWall(ctx unit.Context, _ unit.WallHit) {
	if f.spent {
		return
	}
	f.bounced++
	if f.hasE {
		f.chase(ctx)
	}
	if f.bounced >= maxBounce {
		f.goInert()
	}
}

func (f *鱼) goInert() {
	if f.spent {
		return
	}
	f.spent = true
	if f.drag < 1 {
		f.drag = 50 + rand.Float64()*260
	}
}

func (f *鱼) chase(ctx unit.Context) {
	dx, dy := f.ex-f.x, f.ey-f.y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	f.ux, f.uy = dx/n, dy/n
	sp := f.speed
	if sp < 1 {
		sp = f.job.speed
		f.speed = sp
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: f.ux * sp, VY: f.uy * sp}
}

func (f *鱼) coast(ctx unit.Context, s unit.Sense, dt float64) {
	if f.speed <= 8 {
		f.speed = 0
		ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: 0}
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
		return
	}
	n := math.Hypot(s.Self.VX, s.Self.VY)
	if n > 1e-6 {
		f.ux, f.uy = s.Self.VX/n, s.Self.VY/n
	}
	if f.drag < 1 {
		f.drag = 50 + rand.Float64()*260
	}
	j := 1.0
	if !f.steady {
		j = 0.55 + rand.Float64()*0.9
	}
	f.speed -= f.drag * j * dt
	if f.speed <= 8 {
		f.speed = 0
		ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: 0}
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
		return
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: f.ux * f.speed, VY: f.uy * f.speed}
}
