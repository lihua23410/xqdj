// 大六边形：平顶六边形外形，场心到场边原三倍默认连缩两次四分之一。
// 开局两只中心对称的旋涡绕场心逆时针转；战斗机、活随从、子弹碰上就送到对面往场心偏一截，方向倒转。Helper 不送。
// 场心一条挡视线硬墙，几何和转速同圆那份，由场地演员砌，不是预放墙。
package 大六边形

import (
	"embed"
	"math"

	"xqdj/internal/unit"
	_ "xqdj/map/hex" // ShapeHex 判定在六边形包
)

const Name = "大六边形"

const (
	KindWidehex = "大六边形"
	KindVortex  = "大六边形旋涡"
)

// Extent 场心到场边最远点。要改大小只动这一处，再把 fx/widehex.css 的宽高按 2R 和 √3·R 对齐。
const Extent = 27 * unit.HexRadius / 16

const (
	orbitR      = unit.HexRadius // 旋涡钉在默认六边形那么远的圆上
	vortexR     = 28.0
	spinPeriod  = 12.0 // 绕场心一圈的秒数；spin 弧度每秒逆时针为正
	spinOmega   = 2 * math.Pi / spinPeriod
	warpPad     = 8.0
	warpCD      = 0.4
	armTime     = 0.25
	vortexColor = "#4ec8e0"

	barHalf   = 110.0 // 同圆：开局水平，两端留绕行空隙
	barR      = 6.0
	barPeriod = 8.0
)

//go:embed fx
var assets embed.FS

func Field() unit.Field {
	return unit.Field{
		Name: Name, Shape: unit.ShapeHex, Extent: Extent,
		BootKind: KindWidehex,
	}
}

func init() {
	unit.RegisterField(Field())

	p := unit.NewPack(Name, assets)
	p.Register(unit.Spec{
		Kind: KindWidehex, Role: unit.RoleHelper, Radius: 0, MaxHP: 1,
		Speed: 0, Vision: 9999, Fighter: false, Nonsolid: true,
	}, func(unit.SpawnInfo) unit.Actor {
		return &大六边形{}
	})
	p.Register(unit.Spec{
		Kind: KindVortex, Role: unit.RoleHelper, Radius: vortexR, MaxHP: 1,
		Speed: 0, Vision: 0, Fighter: false,
		Look: unit.Look{Color: vortexColor, Glow: true, FX: []string{"vortex"}},
	}, func(unit.SpawnInfo) unit.Actor { return vortexMark{} })
}

type vortexMark struct{}

func (vortexMark) Handle(unit.Context, unit.Event) {}

func poseAt(t float64) (x1, y1, x2, y2 float64) {
	th := spinOmega * t
	c, s := math.Cos(th), math.Sin(th)
	x1, y1 = orbitR*c, orbitR*s
	return x1, y1, -x1, -y1
}

type 大六边形 struct {
	booted bool
	spun   bool
	wall   uint64
	a, b   uint64
	inside map[uint64]uint64
	cool   map[uint64]float64
}

func (a *大六边形) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !a.booted {
		a.boot(ctx, s)
		return
	}
	a.bind(s)
	a.bindWall(s)
	a.startBar(ctx)
	if a.a == 0 || a.b == 0 {
		return
	}
	a.orbit(ctx, s)
	if s.Time+1e-9 >= armTime {
		a.warp(ctx, s)
	}
}

func (a *大六边形) boot(ctx unit.Context, s unit.Sense) {
	a.booted = true
	a.inside = map[uint64]uint64{}
	a.cool = map[uint64]float64{}
	x1, y1, x2, y2 := poseAt(s.Time)
	ctx.Out <- ctx.Born(unit.Spawn{Kind: KindVortex, X: x1, Y: y1, OwnerID: ctx.ID, Slot: s.Self.Slot})
	ctx.Out <- ctx.Born(unit.Spawn{Kind: KindVortex, X: x2, Y: y2, OwnerID: ctx.ID, Slot: s.Self.Slot})
	ctx.Out <- unit.PlaceWall{
		OwnerID: ctx.ID, Slot: s.Self.Slot, Kind: KindWidehex,
		X1: -barHalf, Y1: 0, X2: barHalf, Y2: 0,
		Radius: barR, Life: 0, Hard: true,
		VisionBlock: true,
	}
}

func (a *大六边形) bind(s unit.Sense) {
	if a.a != 0 && a.b != 0 {
		return
	}
	var ids []uint64
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind == KindVortex && o.OwnerID == s.Self.ID {
			ids = append(ids, o.ID)
		}
	}
	if len(ids) < 2 {
		return
	}
	a.a, a.b = ids[0], ids[1]
}

func (a *大六边形) bindWall(s unit.Sense) {
	if a.wall != 0 {
		return
	}
	for i := range s.Walls {
		w := &s.Walls[i]
		if w.OwnerID == s.Self.ID && w.VisionBlock {
			a.wall = w.ID
			return
		}
	}
}

func (a *大六边形) startBar(ctx unit.Context) {
	if a.spun || a.wall == 0 {
		return
	}
	a.spun = true
	ctx.Out <- unit.SetWallMotion{WallID: a.wall, Spin: -2 * math.Pi / barPeriod}
}

func (a *大六边形) orbit(ctx unit.Context, s unit.Sense) {
	x1, y1, x2, y2 := poseAt(s.Time)
	ctx.Out <- unit.Teleport{UnitID: a.a, X: x1, Y: y1}
	ctx.Out <- unit.Teleport{UnitID: a.b, X: x2, Y: y2}
}

func (a *大六边形) warp(ctx unit.Context, s unit.Sense) {
	va, vb := a.vortex(s, a.a), a.vortex(s, a.b)
	if va == nil || vb == nil {
		return
	}
	cur := map[uint64]uint64{}
	a.touch(s.Self, *va, *vb, cur)
	for i := range s.Nearby {
		a.touch(s.Nearby[i], *va, *vb, cur)
	}
	for id, src := range cur {
		if a.inside[id] == src {
			continue
		}
		if s.Time < a.cool[id] {
			continue
		}
		dst := *vb
		if src == a.b {
			dst = *va
		}
		o := a.unit(s, id)
		if o == nil {
			continue
		}
		a.send(ctx, s, *o, dst)
	}
	a.inside = cur
}

func (a *大六边形) touch(o unit.Snapshot, va, vb unit.Snapshot, cur map[uint64]uint64) {
	if !warpable(o) {
		return
	}
	switch {
	case overlapping(o, va):
		cur[o.ID] = va.ID
	case overlapping(o, vb):
		cur[o.ID] = vb.ID
	}
}

func warpable(o unit.Snapshot) bool {
	if o.Radius <= 0 {
		return false
	}
	return o.Role == unit.RoleFighter || o.Role == unit.RoleProjectile || o.Mortal
}

func overlapping(a, b unit.Snapshot) bool {
	return math.Hypot(a.X-b.X, a.Y-b.Y) <= a.Radius+b.Radius+1e-6
}

func (a *大六边形) vortex(s unit.Sense, id uint64) *unit.Snapshot {
	if s.Self.ID == id {
		return &s.Self
	}
	for i := range s.Nearby {
		if s.Nearby[i].ID == id {
			return &s.Nearby[i]
		}
	}
	return nil
}

func (a *大六边形) unit(s unit.Sense, id uint64) *unit.Snapshot {
	if s.Self.ID == id {
		return &s.Self
	}
	for i := range s.Nearby {
		if s.Nearby[i].ID == id {
			return &s.Nearby[i]
		}
	}
	return nil
}

func (a *大六边形) send(ctx unit.Context, s unit.Sense, o, dst unit.Snapshot) {
	n := math.Hypot(dst.X, dst.Y)
	if n < 1e-6 {
		return
	}
	off := dst.Radius + o.Radius + warpPad
	x := dst.X - dst.X/n*off
	y := dst.Y - dst.Y/n*off
	ctx.Out <- unit.Teleport{UnitID: o.ID, X: x, Y: y}
	ctx.Out <- unit.FX{Name: "swap", Kind: KindVortex, UnitID: o.ID, X: o.X, Y: o.Y}
	ctx.Out <- unit.FX{Name: "swap", Kind: KindVortex, UnitID: o.ID, X: x, Y: y}
	dx, dy := o.VX, o.VY
	sp := math.Hypot(dx, dy)
	if sp > 1e-6 {
		ctx.Out <- unit.SetVelocity{UnitID: o.ID, VX: -dx, VY: -dy}
	} else {
		dx, dy = o.FaceX, o.FaceY
	}
	if math.Hypot(dx, dy) > 1e-6 {
		ctx.Out <- unit.SetFSDirection{UnitID: o.ID, VX: -dx, VY: -dy}
	}
	a.cool[o.ID] = s.Time + warpCD
}
