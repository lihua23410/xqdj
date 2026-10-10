package unit

import (
	"math"
	"math/rand/v2"
	"sync"
)

const (
	MinExtent = 240.0
	MaxExtent = 320.0
)

// Shape 名只是登记键；具体判定由 map 包 RegisterOutline。
const (
	ShapeHex    = "hex"
	ShapeCircle = "circle"
	ShapeSquare = "square"
)

type WallKind int

const (
	WallEdge WallKind = iota // 场边。零值兼容旧测试。
	WallHard
	WallCapsule
)

func (k WallKind) Edge() bool    { return k == WallEdge }
func (k WallKind) Hard() bool    { return k == WallHard }
func (k WallKind) Capsule() bool { return k == WallCapsule }

type FieldWall struct {
	Kind   WallKind
	X1, Y1 float64
	X2, Y2 float64
	Radius float64
	// Period 是绕中点顺时针转一圈的秒数。0 表示不动。
	Period float64
}

// Field 是一份场地的几何与可选开场演员。具体场地目录项由 map/* 构造并 RegisterField。
type Field struct {
	Name   string
	Shape  string
	Extent float64
	Walls  []FieldWall
	// BootKind 非空时，开打在场心冒出该 Kind（场地控制器等）。空则不冒。
	BootKind string
}

// HexField 无名六边形几何，给测试和默认 liveField 用；不是场地目录项。
func HexField() Field {
	return Field{Shape: ShapeHex, Extent: HexRadius}
}

var (
	fieldsByName = map[string]Field{}
	fieldOrder   []string
)

func RegisterField(f Field) {
	if f.Name == "" {
		panic("unit: empty field name")
	}
	if _, ok := fieldsByName[f.Name]; ok {
		panic("unit: duplicate field " + f.Name)
	}
	fieldsByName[f.Name] = cloneField(f)
	fieldOrder = append(fieldOrder, f.Name)
}

func FieldNames() []string {
	out := make([]string, len(fieldOrder))
	copy(out, fieldOrder)
	return out
}

func LookupField(name string) (Field, bool) {
	f, ok := fieldsByName[name]
	if !ok {
		return Field{}, false
	}
	return cloneField(f), true
}

func cloneField(f Field) Field {
	out := f
	if len(f.Walls) > 0 {
		out.Walls = append([]FieldWall(nil), f.Walls...)
	}
	return out
}

var (
	liveMu    sync.RWMutex
	liveField = HexField()
)

func SetLiveField(f Field) {
	liveMu.Lock()
	liveField = f
	liveMu.Unlock()
}

func LiveField() Field {
	liveMu.RLock()
	defer liveMu.RUnlock()
	return cloneField(liveField)
}

func (f Field) size() float64 {
	if f.Extent >= MinExtent {
		return f.Extent
	}
	return HexRadius
}

func (f Field) OutlineContains(x, y, radius float64) bool {
	return f.outline().Contains(x, y, radius)
}

func HexContains(x, y, radius float64) bool {
	return LiveField().Walkable(x, y, radius)
}

func (f Field) Walkable(x, y, radius float64) bool {
	if !f.OutlineContains(x, y, radius) {
		return false
	}
	return !f.overlapsWall(x, y, radius)
}

func (f Field) overlapsWall(x, y, radius float64) bool {
	for _, w := range f.Walls {
		switch {
		case w.Kind.Hard():
			if distToOBB(x, y, w) < radius-1e-9 {
				return true
			}
		case w.Kind.Capsule():
			if distToSeg(x, y, w.X1, w.Y1, w.X2, w.Y2) < radius+w.Radius-1e-9 {
				return true
			}
		}
	}
	return false
}

func distToSeg(px, py, x1, y1, x2, y2 float64) float64 {
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

func distToOBB(px, py float64, w FieldWall) float64 {
	cx, cy := closestOnOBB(px, py, w)
	return math.Hypot(px-cx, py-cy)
}

func closestOnOBB(px, py float64, w FieldWall) (float64, float64) {
	dx, dy := w.X2-w.X1, w.Y2-w.Y1
	l := math.Hypot(dx, dy)
	if l < 1e-12 {
		return w.X1, w.Y1
	}
	ux, uy := dx/l, dy/l
	nx, ny := -uy, ux
	lx := (px-w.X1)*ux + (py-w.Y1)*uy
	ly := (px-w.X1)*nx + (py-w.Y1)*ny
	if lx < 0 {
		lx = 0
	} else if lx > l {
		lx = l
	}
	hw := w.Radius
	if ly < -hw {
		ly = -hw
	} else if ly > hw {
		ly = hw
	}
	return w.X1 + ux*lx + nx*ly, w.Y1 + uy*lx + ny*ly
}

func (f Field) SampleEdge(t, inset float64) (x, y, nx, ny float64) {
	if t < 0 {
		t = 0
	}
	t = t - math.Floor(t)
	return f.outline().SampleEdge(t, inset)
}

func (f Field) NearestEdgeDir(x, y float64) (dx, dy float64) {
	return f.outline().NearestEdgeDir(x, y)
}

func (f Field) RandomWalkable(rng *rand.Rand, radius float64) (x, y float64, ok bool) {
	if rng == nil {
		return 0, 0, false
	}
	R := f.size()
	for i := 0; i < 160; i++ {
		x = (rng.Float64()*2 - 1) * R
		y = (rng.Float64()*2 - 1) * R
		if f.Walkable(x, y, radius) {
			return x, y, true
		}
	}
	return 0, 0, false
}

func (f Field) Clamp(x, y, radius float64) (float64, float64) {
	if f.Walkable(x, y, radius) {
		return x, y
	}
	if !f.OutlineContains(x, y, radius) {
		lo, hi := 0.0, 1.0
		ox, oy := x, y
		for i := 0; i < 24; i++ {
			mid := 0.5 * (lo + hi)
			if f.OutlineContains(ox*mid, oy*mid, radius) {
				lo = mid
			} else {
				hi = mid
			}
		}
		x, y = ox*lo, oy*lo
	}
	for pass := 0; pass < 8; pass++ {
		moved := false
		for _, w := range f.Walls {
			var d, nx, ny, need float64
			switch {
			case w.Kind.Hard():
				d = distToOBB(x, y, w)
				need = radius + 0.5
				cx, cy := closestOnOBB(x, y, w)
				n := math.Hypot(x-cx, y-cy)
				if n < 1e-9 {
					nx, ny = f.NearestEdgeDir(x, y)
				} else {
					nx, ny = (x-cx)/n, (y-cy)/n
				}
			case w.Kind.Capsule():
				d = distToSeg(x, y, w.X1, w.Y1, w.X2, w.Y2)
				need = radius + w.Radius + 0.5
				dx, dy := w.X2-w.X1, w.Y2-w.Y1
				l2 := dx*dx + dy*dy
				t := 0.0
				if l2 > 1e-12 {
					t = ((x-w.X1)*dx + (y-w.Y1)*dy) / l2
					if t < 0 {
						t = 0
					} else if t > 1 {
						t = 1
					}
				}
				cx, cy := w.X1+dx*t, w.Y1+dy*t
				n := math.Hypot(x-cx, y-cy)
				if n < 1e-9 {
					nx, ny = f.NearestEdgeDir(x, y)
				} else {
					nx, ny = (x-cx)/n, (y-cy)/n
				}
			default:
				continue
			}
			if d < need {
				x += nx * (need - d)
				y += ny * (need - d)
				moved = true
			}
		}
		if !moved {
			break
		}
	}
	if !f.OutlineContains(x, y, radius) {
		lo, hi := 0.0, 1.0
		ox, oy := x, y
		for i := 0; i < 16; i++ {
			mid := 0.5 * (lo + hi)
			if f.OutlineContains(ox*mid, oy*mid, radius) {
				lo = mid
			} else {
				hi = mid
			}
		}
		x, y = ox*lo, oy*lo
	}
	return x, y
}
