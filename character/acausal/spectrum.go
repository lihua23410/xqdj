package 非因果

import "math"

const (
	spectrumCoeff = 3.0
	speedMin      = 200.0 / 3 * 1.5 * 1.3
	speedMax      = 800.0 / 3 * 1.5 * 1.3
	displayBins   = 48
	tickDT        = 1.0 / 60.0
)

// Magnitudes 把每个瞬间掉的血变成频率图。
// 柱子高度是每秒掉血。打出去的伤害再乘 spectrumCoeff。
// 两秒内一口掉光 100，高度是 50，伤害是 150；四十秒匀速掉光 100，高度是 2.5，伤害是 7.5。
func Magnitudes(drops []float64) []float64 {
	n := len(drops)
	if n == 0 {
		return nil
	}
	bins := n/2 + 1
	out := make([]float64, bins)
	nf := float64(n)
	for k := 0; k < bins; k++ {
		var re, im float64
		for i, x := range drops {
			if x == 0 {
				continue
			}
			ang := -2 * math.Pi * float64(k*i) / nf
			re += x * math.Cos(ang)
			im += x * math.Sin(ang)
		}
		h := math.Hypot(re, im) / nf * 60
		if h < 1e-6 {
			h = 0
		}
		out[k] = h
	}
	return out
}

// Draw 在场上那一排柱子里抽。权重是高度的平方：矮柱子数量多，按高度线性抽会几乎全部落在小伤害上。
// 伤害就是这根柱子画出来的高度（已经乘过系数）。越靠右的柱子越急，弹越快。
func Draw(bins []float64, u float64) (bin int, dmg, speed float64) {
	cols := Bucket(bins, displayBins)
	if len(cols) == 0 {
		return 0, 0, speedMin
	}
	for i := range cols {
		cols[i] *= spectrumCoeff
	}
	var sum float64
	for _, h := range cols {
		sum += h * h
	}
	if sum < 1e-12 || u < 0 {
		return 0, 0, speedMin
	}
	if u > 1 {
		u = 1
	}
	mark := u * sum
	acc := 0.0
	bin = len(cols) - 1
	for i, h := range cols {
		acc += h * h
		if acc >= mark-1e-12 {
			bin = i
			break
		}
	}
	dmg = cols[bin]
	speed = speedMin
	if len(cols) > 1 {
		t := float64(bin) / float64(len(cols)-1)
		speed = speedMin + (speedMax-speedMin)*t
	}
	return bin, dmg, speed
}

// Bucket 把完整频率图收成固定根数，给场上的图用。
func Bucket(bins []float64, n int) []float64 {
	if n <= 0 {
		return nil
	}
	out := make([]float64, n)
	if len(bins) == 0 {
		return out
	}
	for i, h := range bins {
		k := i * n / len(bins)
		if k >= n {
			k = n - 1
		}
		if h > out[k] {
			out[k] = h
		}
	}
	return out
}
