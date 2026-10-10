package unit

import "fmt"

// Outline 是一份场地外形的判定（包含、采边、扫掠、约束）。
// 具体算法由 map 包 RegisterOutline 挂上；引擎不内置 hex/circle/square 分支。
type Outline interface {
	Contains(x, y, radius float64) bool
	SampleEdge(t, inset float64) (x, y, nx, ny float64)
	NearestEdgeDir(x, y float64) (dx, dy float64)
	// Sweep 在 dt 内相对场边的最早撞击。semi 时按胶囊朝向算外延。
	Sweep(px, py, vx, vy, faceX, faceY, radius, dt float64, semi bool) (t, nx, ny float64, hit bool)
	// Constrain 把圆心推回场内；hint 是可选的撞击法线；skin 为接触皮厚。
	Constrain(x, y, radius, hintNx, hintNy, faceX, faceY, skin float64, semi bool) (ox, oy float64)
}

type outlineFactory func(extent float64) Outline

var outlines = map[string]outlineFactory{}

// RegisterOutline 登记一种 Shape 名对应的外形工厂。由 map/* 在 init 里调用。
func RegisterOutline(shape string, make func(extent float64) Outline) {
	if shape == "" {
		panic("unit: empty outline shape")
	}
	if make == nil {
		panic("unit: nil outline factory")
	}
	if _, ok := outlines[shape]; ok {
		panic("unit: duplicate outline " + shape)
	}
	outlines[shape] = make
}

// MakeOutline 按 Shape + Extent 构造判定器。未登记则 panic。
func MakeOutline(shape string, extent float64) Outline {
	fn, ok := outlines[shape]
	if !ok {
		panic(fmt.Sprintf("unit: outline %q not registered (import the map package)", shape))
	}
	if extent < MinExtent {
		extent = HexRadius
	}
	return fn(extent)
}

func (f Field) outline() Outline {
	shape := f.Shape
	if shape == "" {
		// 旧测试/零值 Sense.Field：按六边形键查找（仍须 map/hex 登记实现）。
		shape = ShapeHex
	}
	return MakeOutline(shape, f.size())
}
