// 筑墙者(百战)不转向。自己的胶囊墙不满两堵就砌；满了先转到法线重合，再沿这条法线对冲。
package 筑墙者_百战

import (
	"embed"
	"math"
	"math/rand/v2"
	"sort"

	"xqdj/internal/unit"
)

const KindVeteran = "筑墙者(百战)"

const (
	vetRadius = 18.0
	vetHP     = 100.0
	vetCruise = 155.0
	vetVision = 9999.0
	vetColor  = "#b85c38"

	wallEvery  = 1.27
	wallLen    = 155.0
	wallRadius = 6.0
	wallScrape = 3.0
	wallGap    = 0.3

	turnRate   = math.Pi / 2 // 每秒 90°
	tick       = 1.0 / 60.0
	chargeV0   = 300.0
	chargeAcc  = 500.0
	ramDamage  = 8.0
	stunRadius = wallLen / 2
	stunDur    = 2.8

	parallelEps = 0.5 * math.Pi / 180
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindVeteran, assets)
	p.Register(unit.Spec{
		Kind:    KindVeteran,
		Role:    unit.RoleFighter,
		Radius:  vetRadius,
		MaxHP:   vetHP,
		Speed:   vetCruise,
		Vision:  vetVision,
		Fighter: true,
		Look:    unit.Look{Color: vetColor, FX: []string{"veteran"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &百战{
			nextBuild: wallEvery,
			savedLeft: wallEvery,
			rng:       rand.New(rand.NewPCG(uint64(info.Slot+1)*0x9E3779B97F4A7C15, 0xB7A11)),
		}
	})
}

const (
	phaseIdle = iota
	phasePair
)

type 百战 struct {
	nextBuild  float64
	savedLeft  float64
	phase      int
	locked     bool
	chargeFrom float64
	axisX      float64
	axisY      float64
	rng        *rand.Rand
}

func (a *百战) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	switch e := ev.(type) {
	case unit.WallSlam:
		a.onSlam(e)
	case unit.Sense:
		a.onSense(ctx, e)
	}
}

func (a *百战) onSlam(e unit.WallSlam) {
	a.phase = phaseIdle
	a.locked = false
	a.nextBuild = e.Time + wallEvery
}

func (a *百战) onSense(ctx unit.Context, s unit.Sense) {
	walls := ownWalls(s.Walls, ctx.ID)
	if len(walls) >= 2 {
		if a.phase == phaseIdle {
			a.savedLeft = wallEvery
			a.phase = phasePair
		}
		a.drivePair(ctx, s, walls[0], walls[1])
		return
	}
	if a.phase != phaseIdle {
		if len(walls) == 1 {
			stopWall(ctx, walls[0].ID)
		}
		a.nextBuild = s.Time + a.savedLeft
		a.phase = phaseIdle
		a.locked = false
	}
	if s.Time+1e-9 < a.nextBuild {
		return
	}
	if !a.place(ctx, s, walls) {
		return
	}
	a.nextBuild = s.Time + wallEvery
}

func (a *百战) place(ctx unit.Context, s unit.Sense, walls []unit.WallView) bool {
	if a.rng == nil {
		a.rng = rand.New(rand.NewPCG(ctx.ID|1, 0xB7A11))
	}
	var x, y, ang float64
	var ok bool
	if len(walls) == 0 {
		x, y, ang, ok = sampleWall(s.Field, a.rng)
	} else {
		x, y, ang, ok = a.behind(s, walls[0])
	}
	if !ok {
		return false
	}
	dx, dy := math.Cos(ang)*wallLen/2, math.Sin(ang)*wallLen/2
	ctx.Out <- unit.PlaceWall{
		OwnerID:   ctx.ID,
		Slot:      s.Self.Slot,
		Kind:      ctx.Kind,
		X1:        x + dx,
		Y1:        y + dy,
		X2:        x - dx,
		Y2:        y - dy,
		Radius:    wallRadius,
		Life:      -1,
		Amount:    wallScrape,
		HitGap:    wallGap,
		WithOwner: true,
	}
	return true
}

func (a *百战) behind(s unit.Sense, wall unit.WallView) (x, y, ang float64, ok bool) {
	enemy := unit.Seek(s)
	if enemy == nil {
		return 0, 0, 0, false
	}
	cx, cy := center(wall)
	dx, dy := enemy.X-cx, enemy.Y-cy
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return 0, 0, 0, false
	}
	x = enemy.X + dx/n*(wallLen/2)
	y = enemy.Y + dy/n*(wallLen/2)
	if !s.Field.OutlineContains(x, y, 0) {
		return 0, 0, 0, false
	}
	return x, y, a.rng.Float64() * math.Pi, true
}

func (a *百战) drivePair(ctx unit.Context, s unit.Sense, w1, w2 unit.WallView) {
	ax, ay := approach(w1, w2)
	// 墙身垂直于墙心连线时，两条法线都落在这条连线上。
	target := wrapPi(math.Atan2(ay, ax) + math.Pi/2)
	d1 := shortest(wallAng(w1), target)
	d2 := shortest(wallAng(w2), target)
	if !a.locked {
		if math.Abs(d1) < parallelEps && math.Abs(d2) < parallelEps {
			a.locked = true
			a.chargeFrom = s.Time
			a.axisX, a.axisY = ax, ay
		} else {
			ctx.Out <- motion(w1.ID, spinToward(d1), 0, 0, 0)
			ctx.Out <- motion(w2.ID, spinToward(d2), 0, 0, 0)
			return
		}
	}
	speed := chargeV0 + chargeAcc*(s.Time-a.chargeFrom)
	ctx.Out <- motion(w1.ID, 0, a.axisX*speed, a.axisY*speed, ramDamage)
	ctx.Out <- motion(w2.ID, 0, -a.axisX*speed, -a.axisY*speed, ramDamage)
}

func spinToward(d float64) float64 {
	if math.Abs(math.Abs(d)-math.Pi/2) <= 1e-8 {
		d = -math.Pi / 2
	}
	if math.Abs(d) < parallelEps {
		return 0
	}
	rate := turnRate
	if math.Abs(d) < turnRate*tick {
		rate = math.Abs(d) / tick
	}
	return math.Copysign(rate, d)
}

func motion(id uint64, spin, vx, vy, ram float64) unit.SetWallMotion {
	m := unit.SetWallMotion{WallID: id, Spin: spin, VX: vx, VY: vy, Ram: ram}
	if ram > 0 {
		m.StunRadius = stunRadius
		m.StunDur = stunDur
	}
	return m
}

func stopWall(ctx unit.Context, id uint64) {
	ctx.Out <- unit.SetWallMotion{WallID: id}
}

func ownWalls(all []unit.WallView, owner uint64) []unit.WallView {
	var out []unit.WallView
	for _, w := range all {
		if w.OwnerID == owner {
			out = append(out, w)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func wallAng(w unit.WallView) float64 {
	return math.Atan2(w.Y2-w.Y1, w.X2-w.X1)
}

func center(w unit.WallView) (x, y float64) {
	return (w.X1 + w.X2) / 2, (w.Y1 + w.Y2) / 2
}

func approach(w1, w2 unit.WallView) (float64, float64) {
	x1, y1 := center(w1)
	x2, y2 := center(w2)
	dx, dy := x2-x1, y2-y1
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return 1, 0
	}
	return dx / n, dy / n
}

// shortest 是从 a 转到 b 的有符号短弧，范围 (-π/2, π/2]。正值表示逆时针。
func shortest(a, b float64) float64 {
	d := wrapPi(b) - wrapPi(a)
	if d > math.Pi/2 {
		d -= math.Pi
	}
	if d <= -math.Pi/2 {
		d += math.Pi
	}
	return d
}

func wrapPi(x float64) float64 {
	x = math.Mod(x, math.Pi)
	if x < 0 {
		x += math.Pi
	}
	return x
}

func sampleWall(f unit.Field, rng *rand.Rand) (x, y, ang float64, ok bool) {
	ext := f.Extent
	if ext <= 0 {
		ext = unit.HexRadius
	}
	for i := 0; i < 64; i++ {
		x = (rng.Float64()*2 - 1) * ext
		y = (rng.Float64()*2 - 1) * ext
		if f.OutlineContains(x, y, 0) {
			return x, y, rng.Float64() * math.Pi, true
		}
	}
	return 0, 0, 0, false
}
