package 优昙华院

import (
	"math"

	"xqdj/internal/unit"
)

const (
	SkillSteer = iota // 测例里表示强制不放
	SkillMind
	SkillBreak
	SkillLaser
	SkillAspect
	SkillGas
	SkillCrown
	SkillDose
	SkillCount
)

const (
	angleBins = 16
	dashDirs  = 8
	dashRungs = 3
)

var gasDist = [...]float64{0, 54, 108}

type Action struct {
	Skill uint8
	ABin  uint8
	DBin  uint8
}

func angleOf(self, enemy unit.Snapshot, bin uint8) float64 {
	dx := enemy.X - self.X
	dy := enemy.Y - self.Y
	base := math.Atan2(dy, dx)
	k := int(bin) % angleBins
	return base + float64(k)*2*math.Pi/float64(angleBins)
}

func gasPos(self, enemy unit.Snapshot, bin uint8) (float64, float64) {
	b := int(bin) % (dashDirs * dashRungs)
	dir := b % dashDirs
	rung := b / dashDirs
	ang := angleOf(self, enemy, uint8(dir*(angleBins/dashDirs)))
	dist := gasDist[rung]
	x := enemy.X + math.Cos(ang)*dist
	y := enemy.Y + math.Sin(ang)*dist
	return clampHex(x, y, 8)
}
