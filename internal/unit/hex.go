package unit

import "math"

// HexRadius 是平顶六边形外接圆半径。角色瞬移夹紧用这个，不要再抄 280。
const HexRadius = 280.0

func HexContains(x, y, radius float64) bool {
	ap := HexRadius * math.Sqrt(3) / 2
	limit := ap - radius
	for i := 0; i < 6; i++ {
		a := (float64(i) + 0.5) * math.Pi / 3
		if math.Cos(a)*x+math.Sin(a)*y > limit+1e-6 {
			return false
		}
	}
	return true
}
