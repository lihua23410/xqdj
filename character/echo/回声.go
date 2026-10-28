// Package 回声：看不见、只靠声呐找人的远程。
//
// 它平时什么都看不见（Vision 0），只朝自己移动的方向放星弹；星弹炸成 16 枚声呐碎片，
// 碎片碰到敌人只做一件事：在它身上补一格扫描。每个敌人各有一份 10 秒的扫描窗口，
// 满 10 格锁定，碎片才开始掉血；窗口到点由引擎清掉。
package 回声

import (
	"embed"
	"math"
	"strconv"

	"xqdj/internal/unit"
)

const KindEcho = "回声"

// 星弹与碎片。两个都是全局唯一的 Kind。
const (
	KindStar  = "回声星弹"
	KindShard = "回声碎片"
)

const (
	echoRadius = 18.0
	echoHP     = 90.0
	echoSpeed  = 165.0
	echoVision = 0.0 // 盲：感知里只有自己的随从
	echoColor  = "#3b4775"

	// 只有真的在走才攒发射条：站住、被锁在原地都不放。
	moveEps   = 40.0
	fireEvery = 1.8

	starSpeed  = 340.0
	starRadius = 9.0
	starLife   = 2.2
	starGap    = 1.0
	shardN     = 16

	shardSpeed   = 250.0
	shardRadius  = 4.0
	shardLife    = 4.0
	shardBounces = 8
	shardDmg     = 3.0

	countMax  = 10
	countLife = 5.0
)

const countIcon = "/ball/" + KindEcho + "/fx/count.svg"

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindEcho, assets)
	p.Register(unit.Spec{
		Kind:    KindEcho,
		Role:    unit.RoleFighter,
		Radius:  echoRadius,
		MaxHP:   echoHP,
		Speed:   echoSpeed,
		Vision:  echoVision,
		Fighter: true,
		Look:    unit.Look{Color: echoColor, FX: []string{"echo"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &回声{}
	})

	// 星弹：只负责炸，不出伤。
	p.Register(unit.Spec{
		Kind:   KindStar,
		Role:   unit.RoleProjectile,
		Radius: starRadius,
		MaxHP:  1,
		Speed:  starSpeed,
		Vision: 0,
		Look:   unit.Look{Color: "#ffd45c", Glow: true, FX: []string{"echo-star"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &星弹{owner: info.OwnerID, slot: info.Slot}
	})

	// 碎片：声呐。在墙上弹，碰到敌人补一格扫描；满 10 格才掉血。
	p.Register(unit.Spec{
		Kind:   KindShard,
		Role:   unit.RoleProjectile,
		Radius: shardRadius,
		MaxHP:  1,
		Speed:  shardSpeed,
		Vision: 0,
		Look:   unit.Look{Color: "#8fecff", FX: []string{"echo-shard"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &碎片{slot: info.Slot, markKind: countKind(info.OwnerID)}
	})
}

// countKind 把扫描挂在放声呐的这只回声名下。同队两只回声各扫各的，互不串味。
func countKind(owner uint64) string {
	return "回声计数" + strconv.FormatUint(owner, 10)
}

func markStacks(o unit.Snapshot, kind string) int {
	for i := range o.Marks {
		if o.Marks[i].Kind == kind {
			return o.Marks[i].Stacks
		}
	}
	return 0
}

// 回声本体：瞎的。唯一做的事是按移动方向放星弹，剩下的全交给碎片和引擎的扫描窗口。
type 回声 struct {
	lastTime float64
	charge   float64
	booted   bool
}

func (e *回声) Handle(ctx unit.Context, ev unit.Event) {
	switch v := ev.(type) {
	case unit.IncomingDamage:
		unit.ConfirmHit(ctx, v)
	case unit.Sense:
		e.onSense(ctx, v)
	}
}

func (e *回声) onSense(ctx unit.Context, s unit.Sense) {
	dt := s.Time - e.lastTime
	e.lastTime = s.Time
	if !e.booted {
		e.booted = true
		return
	}
	if dt <= 0 || dt > 0.5 {
		return
	}
	if math.Hypot(s.Self.VX, s.Self.VY) <= moveEps {
		return
	}
	e.charge += dt
	if e.charge < fireEvery {
		return
	}
	e.charge = 0
	e.fire(ctx, s)
}

func (e *回声) fire(ctx unit.Context, s unit.Sense) {
	vx, vy := s.Self.VX, s.Self.VY
	l := math.Hypot(vx, vy)
	if l < 1e-9 {
		return
	}
	ux, uy := vx/l, vy/l
	gap := s.Self.Radius + starRadius + starGap
	x, y := s.Field.Clamp(s.Self.X+ux*gap, s.Self.Y+uy*gap, starRadius)
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: KindStar, X: x, Y: y,
		VX: ux * starSpeed, VY: uy * starSpeed,
		OwnerID: ctx.ID, Slot: s.Self.Slot,
	})
}

// 星弹：一个会飞的声呐浮标。撞墙、撞人、撞弹都算引爆，炸完自己消失。
type 星弹 struct {
	owner  uint64
	slot   int
	booted bool
	bornAt float64
	done   bool
}

func (a *星弹) Handle(ctx unit.Context, ev unit.Event) {
	if a.done {
		return
	}
	switch v := ev.(type) {
	case unit.Sense:
		if !a.booted {
			a.booted = true
			a.bornAt = v.Time
			return
		}
		if v.Time-a.bornAt >= starLife {
			a.die(ctx)
		}
	case unit.Collision:
		// 法线从对方指向自己；取星弹圆心，别让碎片全生在被撞单位身体里。
		x := v.Other.X + v.NX*(v.Other.Radius+starRadius)
		y := v.Other.Y + v.NY*(v.Other.Radius+starRadius)
		a.burst(ctx, x, y)
	case unit.WallHit:
		a.burst(ctx, v.X, v.Y)
	}
}

func (a *星弹) burst(ctx unit.Context, x, y float64) {
	if a.done {
		return
	}
	a.done = true
	ctx.Out <- unit.FX{Name: "echo-burst", Kind: KindStar, X: x, Y: y}
	field := unit.LiveField()
	r := starRadius + shardRadius + 1
	for i := 0; i < shardN; i++ {
		ang := 2 * math.Pi * float64(i) / shardN
		ux, uy := math.Cos(ang), math.Sin(ang)
		sx, sy := field.Clamp(x+ux*r, y+uy*r, shardRadius)
		ctx.Out <- ctx.Born(unit.Spawn{
			Kind: KindShard, X: sx, Y: sy,
			VX: ux * shardSpeed, VY: uy * shardSpeed,
			OwnerID: a.owner, Slot: a.slot,
		})
	}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

func (a *星弹) die(ctx unit.Context) {
	if a.done {
		return
	}
	a.done = true
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

// 碎片：声呐碎片。弹 8 次，第 9 次碰墙消失；碰到敌人补一格扫描就走。
type 碎片 struct {
	slot     int
	markKind string
	bounces  int
	booted   bool
	bornAt   float64
	done     bool
}

func (s *碎片) Handle(ctx unit.Context, ev unit.Event) {
	if s.done {
		return
	}
	switch v := ev.(type) {
	case unit.Sense:
		if !s.booted {
			s.booted = true
			s.bornAt = v.Time
			return
		}
		if v.Time-s.bornAt >= shardLife {
			s.die(ctx)
		}
	case unit.WallHit:
		s.bounces++
		if s.bounces > shardBounces {
			s.die(ctx)
		}
	case unit.Collision:
		if unit.Hittable(v.Other, s.slot) {
			stacks := markStacks(v.Other, s.markKind)
			// 每次命中都补新的一层：旧层自己过期，新层从这一拍重新活 countLife 秒。
			// 层数可以超过 10；10 只是锁定门槛，不是上限。
			ctx.Out <- unit.StackMark{
				UnitID: v.Other.ID, Kind: s.markKind, Delta: 1, Icon: countIcon,
				Lifetime: countLife,
			}
			stacks++
			if stacks >= countMax {
				ctx.Out <- unit.Damage{From: ctx.ID, To: v.Other.ID, Amount: shardDmg, NoFreeze: true}
			}
			ctx.Out <- unit.FX{
				Name: "echo-ping", Kind: KindShard, UnitID: v.Other.ID,
				X: v.Other.X, Y: v.Other.Y, Amount: float64(stacks),
			}
		}
		s.die(ctx)
	}
}

func (s *碎片) die(ctx unit.Context) {
	if s.done {
		return
	}
	s.done = true
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}
