// 雷达不转向。激光是从球心伸出的射线，开局朝正北，顺时针转，被墙裁断，只打敌方战斗机。
package 雷达

import (
	"embed"
	"math"
	"xqdj/internal/unit"
)

const KindRadar = "雷达"

const (
	radarRadius = 18.0
	radarHP     = 100.0
	radarCruise = 160.0
	radarVision = 9999.0
	radarDamage = 7.0
	radarHitCD  = 0.3 // 同一目标两次激光之间的间隔
	radarWidth  = 4.0 // 判定总宽；看起来是细线
	radarSpin   = 5.5 // 顺时针转一圈的秒数
	radarColor  = "#3eb87a"
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindRadar, assets)
	p.Register(unit.Spec{
		Kind:    KindRadar,
		Role:    unit.RoleFighter,
		Radius:  radarRadius,
		MaxHP:   radarHP,
		Speed:   radarCruise,
		Vision:  radarVision,
		Fighter: true,
		Look: unit.Look{
			Color: radarColor,
			FX:    []string{"radar"},
		},
	}, func(unit.SpawnInfo) unit.Actor {
		return &雷达{hitAt: map[uint64]float64{}}
	})
}

type 雷达 struct {
	hitAt map[uint64]float64 // 每个目标上次被激光打中的时间
}

func (r *雷达) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	dx, dy := laserDir(s.Time)
	reach := clipReach(s.Self.X, s.Self.Y, dx, dy, visionOf(s), radarWidth/2, s.Field, s.Walls)
	r.sweep(ctx, s, dx, dy, reach)
	r.emitBeam(ctx, s, dx, dy, reach)
}

func (r *雷达) sweep(ctx unit.Context, s unit.Sense, dx, dy, reach float64) {
	if r.hitAt == nil {
		r.hitAt = map[uint64]float64{}
	}
	half := radarWidth / 2
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		if !rayHits(s.Self.X, s.Self.Y, dx, dy, reach, o.X, o.Y, o.Radius, half) {
			continue
		}
		if at, ok := r.hitAt[o.ID]; ok && s.Time+1e-9 < at+radarHitCD {
			continue
		}
		r.hitAt[o.ID] = s.Time
		ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: radarDamage}
	}
}

func (r *雷达) emitBeam(ctx unit.Context, s unit.Sense, dx, dy, reach float64) {
	ctx.Out <- unit.FX{
		Name:   "beam",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		Slot:   s.Self.Slot,
		X:      s.Self.X,
		Y:      s.Self.Y,
		VX:     dx, // 世界坐标方向；前端 Y 轴向上
		VY:     dy,
		Amount: reach,
	}
}

func visionOf(s unit.Sense) float64 {
	if s.Self.Vision > 0 {
		return s.Self.Vision
	}
	return radarVision
}

// clipReach：沿射线找第一处被场边 / 硬墙 / 胶囊墙挡住的距离。
// Field.Walkable 含场边与场地预放墙；演员砌的硬墙只在 Sense.Walls 里，须另查。
func clipReach(px, py, ux, uy, maxLen, halfW float64, field unit.Field, walls []unit.WallView) float64 {
	if maxLen <= 0 {
		return 0
	}
	step := 4.0
	if halfW > step {
		step = halfW
	}
	prev := 0.0
	for t := step; t <= maxLen+1e-9; t += step {
		if t > maxLen {
			t = maxLen
		}
		if !blockedAt(px+ux*t, py+uy*t, halfW, field, walls) {
			prev = t
			if t >= maxLen {
				return maxLen
			}
			continue
		}
		lo, hi := prev, t
		for j := 0; j < 18; j++ {
			mid := 0.5 * (lo + hi)
			if !blockedAt(px+ux*mid, py+uy*mid, halfW, field, walls) {
				lo = mid
			} else {
				hi = mid
			}
		}
		return hi
	}
	return maxLen
}

func blockedAt(x, y, halfW float64, field unit.Field, walls []unit.WallView) bool {
	if !field.Walkable(x, y, halfW) {
		return true
	}
	for i := range walls {
		w := &walls[i]
		if segDist(x, y, w.X1, w.Y1, w.X2, w.Y2) < w.Radius+halfW {
			return true
		}
	}
	return false
}

func segDist(px, py, x1, y1, x2, y2 float64) float64 {
	dx, dy := x2-x1, y2-y1
	l2 := dx*dx + dy*dy
	if l2 < 1e-12 {
		return math.Hypot(px-x1, py-y1)
	}
	t := ((px-x1)*dx + (py-y1)*dy) / l2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return math.Hypot(px-(x1+dx*t), py-(y1+dy*t))
}

// laserDir 开局正北 (0,1)，之后顺时针。世界 +Y 是观众看见的上方。
func laserDir(t float64) (float64, float64) {
	ang := math.Pi/2 - (2*math.Pi/radarSpin)*t
	return math.Cos(ang), math.Sin(ang)
}

// rayHits：从 (px,py) 沿单位向量 (ux,uy) 伸出 length 的射线。
// along < 0 是球的另一侧，不算。
func rayHits(px, py, ux, uy, length, hx, hy, hr, halfW float64) bool {
	dx, dy := hx-px, hy-py
	along := dx*ux + dy*uy
	if along < 0 || along > length {
		return false
	}
	perp := math.Hypot(dx-ux*along, dy-uy*along)
	return perp <= hr+halfW
}
