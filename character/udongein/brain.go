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
	SkillVolley
	SkillCount
)

const (
	angleBins = 16
	dashDirs  = 8
	dashRungs = 3
)

var gasDist = [...]float64{0, 54, 108}

var attackSkills = []uint8{SkillVolley, SkillMind, SkillAspect, SkillLaser}

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

func isAttackCard(sk uint8) bool {
	switch sk {
	case SkillVolley, SkillMind, SkillAspect, SkillLaser:
		return true
	default:
		return false
	}
}

func cardCostOf(sk uint8) int {
	if sk == SkillCrown || sk == SkillDose {
		return 3
	}
	return 1
}

func skillFire(sk uint8) float64 {
	switch sk {
	case SkillVolley:
		return float64(volleyCount) * volleyGap
	case SkillMind:
		return mindFire
	case SkillLaser:
		return laserWind + laserLife
	case SkillAspect:
		return float64(aspectShots) * aspectGap
	case SkillCrown:
		return animCrown
	case SkillDose:
		return animDose
	case SkillBreak, SkillGas:
		return animRing
	default:
		return mindFire
	}
}

func skillRecover(sk uint8) float64 {
	switch sk {
	case SkillCrown, SkillDose:
		return recoverCard
	default:
		return recoverAtk
	}
}

func skillAnim(sk uint8) float64 {
	return skillFire(sk) + skillRecover(sk)
}
