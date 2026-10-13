package 奇美拉

import (
	"math"

	"xqdj/internal/unit"
)

const (
	doorHalf = 72.0
	neck     = 160.0
	arcSteps = 48
)

func hexApo() float64 { return unit.HexRadius * math.Sqrt(3) / 2 }

func xCut() float64 { return unit.HexRadius - doorHalf/math.Sqrt(3) }

func squareHalf() float64 {
	return unit.HexRadius * (3 - math.Sqrt(3)) / 2
}

func octApo() float64 { return unit.HexRadius * math.Cos(math.Pi/8) }

func yFarN() float64 { return hexApo() + neck }

func North() (x, y float64) { return 0, yFarN() + hexApo() }

func yFarS() float64 { return -(hexApo() + neck) }

func South() (x, y float64) { return 0, yFarS() - squareHalf() }

func xFarE() float64 { return xCut() + neck }

func chordDist() float64 {
	return math.Sqrt(unit.HexRadius*unit.HexRadius - doorHalf*doorHalf)
}

func East() (x, y float64) { return xFarE() + chordDist(), 0 }

func xFarW() float64 { return -(xCut() + neck) }

func West() (x, y float64) { return xFarW() - octApo(), 0 }

type pt struct{ x, y float64 }

func hexVert(i int, cx, cy float64) pt {
	a := float64(i) * math.Pi / 3
	return pt{cx + unit.HexRadius*math.Cos(a), cy + unit.HexRadius*math.Sin(a)}
}

func octVert(i int, cx, cy float64) pt {
	a := -math.Pi/8 + float64(i)*math.Pi/4
	return pt{cx + unit.HexRadius*math.Cos(a), cy + unit.HexRadius*math.Sin(a)}
}

// contour 逆时针外轮廓，不含重复的闭合点。门洞不在轮廓上。
func contour() []pt {
	r := unit.HexRadius
	apo := hexApo()
	xfE := xFarE()
	xfW := xFarW()
	yfN := yFarN()
	yfS := yFarS()
	cxE, _ := East()
	cxW, _ := West()
	_, cyN := North()
	_, cyS := South()
	h := squareHalf()
	d := chordDist()

	var pts []pt
	add := func(p pt) {
		if len(pts) > 0 {
			q := pts[len(pts)-1]
			if math.Hypot(p.x-q.x, p.y-q.y) < 1e-6 {
				return
			}
		}
		pts = append(pts, p)
	}

	// 东通道下壁，再沿圆的外弧回到上壁。
	add(pt{xCut(), -doorHalf})
	add(pt{xfE, -doorHalf})
	a0 := math.Atan2(-doorHalf, -d)
	a1 := math.Atan2(doorHalf, -d)
	if a1 < a0 {
		a1 += 2 * math.Pi
	}
	for i := 1; i <= arcSteps; i++ {
		a := a0 + (a1-a0)*float64(i)/float64(arcSteps)
		add(pt{cxE + r*math.Cos(a), r * math.Sin(a)})
	}
	add(pt{xCut(), doorHalf})
	add(hexVert(1, 0, 0))
	add(pt{doorHalf, apo})

	// 北通道东壁，绕火山回到西壁。
	add(pt{doorHalf, yfN})
	add(hexVert(5, 0, cyN))
	for _, i := range []int{0, 1, 2, 3, 4} {
		add(hexVert(i, 0, cyN))
	}
	add(pt{-doorHalf, yfN})
	add(pt{-doorHalf, apo})
	add(hexVert(2, 0, 0))
	add(pt{-xCut(), doorHalf})

	// 西通道北壁，绕八边形回到南壁。
	add(pt{xfW, doorHalf})
	for _, i := range []int{1, 2, 3, 4, 5, 6, 7, 0} {
		add(octVert(i, cxW, 0))
	}
	add(pt{xfW, -doorHalf})
	add(pt{-xCut(), -doorHalf})
	add(hexVert(4, 0, 0))
	add(pt{-doorHalf, -apo})

	// 南通道西壁，绕正方形回到东壁，再闭合到东通道下壁。
	add(pt{-doorHalf, yfS})
	add(pt{-h, yfS})
	add(pt{-h, cyS - h})
	add(pt{h, cyS - h})
	add(pt{h, yfS})
	add(pt{doorHalf, yfS})
	add(pt{doorHalf, -apo})
	add(hexVert(5, 0, 0))
	add(pt{xCut(), -doorHalf})
	if len(pts) > 1 {
		pts = pts[:len(pts)-1]
	}
	return pts
}

func farthest() float64 {
	m := 0.0
	for _, p := range contour() {
		if d := math.Hypot(p.x, p.y); d > m {
			m = d
		}
	}
	cxE, _ := East()
	if d := cxE + unit.HexRadius; d > m {
		m = d
	}
	cxW, _ := West()
	if d := -(cxW - octApo()); d > m {
		m = d
	}
	return m
}

// RegionContains 左页（0）在西边八边形内，右页（1）在东边圆内。
func RegionContains(side int, x, y, radius float64) bool {
	switch side {
	case 0:
		cx, cy := West()
		return octContains(x-cx, y-cy, radius)
	case 1:
		cx, cy := East()
		return math.Hypot(x-cx, y-cy) <= unit.HexRadius-radius+1e-6
	default:
		return false
	}
}

func octContains(x, y, radius float64) bool {
	limit := octApo() - radius
	for i := 0; i < 8; i++ {
		a := float64(i) * math.Pi / 4
		if math.Cos(a)*x+math.Sin(a)*y > limit+1e-6 {
			return false
		}
	}
	return true
}
