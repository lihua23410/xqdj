package 人偶使

import (
	"math"

	"xqdj/internal/unit"
)

const (
	SkillNone = iota
	SkillAtkFan
	SkillAtkRect
	SkillFan
	SkillPlace
	SkillN22
	SkillN26
	SkillN24
	SkillN62
	CardDemon
	CardHourai
	CardBattle
	CardSpirit
	SkillCount
)

func energyCostOf(sk uint8) float64 {
	switch sk {
	case SkillNone, SkillAtkFan, SkillAtkRect:
		return 0
	case CardHourai:
		return 4
	case CardBattle:
		return 2
	case SkillFan, SkillPlace, SkillN22, SkillN26, SkillN24, SkillN62, CardDemon, CardSpirit:
		return 1
	default:
		return 1
	}
}

func isNumberCard(sk uint8) bool {
	switch sk {
	case SkillN22, SkillN26, SkillN24, SkillN62:
		return true
	default:
		return false
	}
}

func shouldCall(sk uint8) bool {
	switch sk {
	case SkillN22, SkillN26, SkillN24, SkillN62, CardDemon, CardHourai, CardBattle:
		return true
	default:
		return false
	}
}

func lockSpan(sk uint8) float64 {
	switch sk {
	case SkillAtkFan, SkillFan:
		return atkFanWind + atkFanLife
	case SkillAtkRect:
		return atkRectWind + atkRectLife
	case SkillPlace, SkillN22, SkillN26, CardDemon:
		return 0.8
	case SkillN24:
		return spell24Wind + spell24Life
	case SkillN62:
		return 0.4
	case CardHourai:
		return houraiLife
	case CardBattle:
		return 0.4
	case CardSpirit:
		return 0.15
	default:
		return 0.4
	}
}

func (a *人偶使) ready(s unit.Sense, sk uint8) bool {
	if sk == SkillNone || int(sk) >= SkillCount {
		return false
	}
	if a.lockedOut(s.Time) {
		return false
	}
	if energyCostOf(sk) > a.energy+1e-9 {
		return false
	}
	if (sk == SkillN24 || sk == SkillN62) && a.levelOf(sk) < 1 && !a.playingCard {
		return false
	}
	return true
}

func (a *人偶使) levelOf(sk uint8) int {
	if int(sk) >= SkillCount {
		return 0
	}
	return a.level[sk]
}

func (a *人偶使) pickCast(s unit.Sense, enemy *unit.Snapshot) uint8 {
	if a.locked {
		if a.force == SkillNone || a.force >= SkillCount {
			return SkillNone
		}
		if !a.ready(s, a.force) || !a.shouldCast(s, enemy, a.force, false) {
			return SkillNone
		}
		return a.force
	}
	cands := make([]uint8, 0, 8)
	for _, sk := range []uint8{SkillN22, SkillN26, SkillN24, SkillN62, SkillPlace, SkillFan, SkillAtkRect, SkillAtkFan} {
		if !a.ready(s, sk) {
			continue
		}
		if a.shouldCast(s, enemy, sk, false) {
			cands = append(cands, sk)
		}
	}
	if a.energy+1e-9 < 2 {
		tight := cands[:0]
		for _, sk := range cands {
			if energyCostOf(sk) <= 0 {
				tight = append(tight, sk)
				continue
			}
			if sk == SkillPlace && a.liveDolls == 0 {
				tight = append(tight, sk)
			}
			if sk == SkillN22 && a.idleDolls >= 2 && a.levelOf(SkillN22) < 1 {
				tight = append(tight, sk)
			}
		}
		cands = tight
	}
	if len(cands) == 0 {
		return SkillNone
	}
	if a.rng == nil {
		return cands[0]
	}
	return cands[a.rng.IntN(len(cands))]
}

func (a *人偶使) shouldCast(s unit.Sense, enemy *unit.Snapshot, sk uint8, fromCard bool) bool {
	if enemy == nil && sk != SkillPlace && sk != SkillN26 && !fromCard {
		if sk != SkillN24 {
			return false
		}
	}
	idle := a.idleDolls
	dist := 0.0
	if enemy != nil {
		dist = math.Hypot(enemy.X-s.Self.X, enemy.Y-s.Self.Y)
	}
	switch sk {
	case SkillAtkFan:
		return enemy != nil && dist <= dollReach+fanR+enemy.Radius+8
	case SkillAtkRect:
		return enemy != nil && dist <= dollReach+rectW*0.5+enemy.Radius
	case SkillFan:
		return enemy != nil && dist <= dollReach+fanR+enemy.Radius+8 && a.energy+1e-9 >= 1
	case SkillPlace:
		return idle+a.liveDolls < 4
	case SkillN22:
		if a.levelOf(SkillN22) >= 1 || fromCard {
			return enemy != nil
		}
		return idle >= 1 && enemy != nil
	case SkillN26:
		return true
	case SkillN24:
		return enemy != nil
	case SkillN62:
		return enemy != nil && a.hasFace()
	case CardDemon, CardHourai, CardBattle:
		return enemy != nil
	case CardSpirit:
		return enemy != nil && dist < 160
	default:
		return false
	}
}

func (a *人偶使) hasFace() bool {
	return math.Hypot(a.hx, a.hy) > 1e-6
}

func toward(self, enemy unit.Snapshot) (float64, float64) {
	dx, dy := enemy.X-self.X, enemy.Y-self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return 1, 0
	}
	return dx / n, dy / n
}

func offset(x, y, ux, uy, dist float64) (float64, float64) {
	return x + ux*dist, y + uy*dist
}
