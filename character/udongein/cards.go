package 优昙华院

import "xqdj/internal/unit"

func (a *优昙华院) initDeckLocked() {
	a.deck = make([]uint8, 0, deckSize)
	for i := 0; i < 3; i++ {
		a.deck = append(a.deck, SkillVolley, SkillMind, SkillAspect, SkillLaser)
	}
	for i := 0; i < 4; i++ {
		a.deck = append(a.deck, SkillDose, SkillCrown)
	}
	a.rng.Shuffle(len(a.deck), func(i, j int) { a.deck[i], a.deck[j] = a.deck[j], a.deck[i] })
	a.hand = a.hand[:0]
	a.progress = 0
}

func (a *优昙华院) charge() {
	a.cardMu.Lock()
	defer a.cardMu.Unlock()
	a.progress += cardFill
	a.drawReadyLocked()
}

func (a *优昙华院) drawReadyLocked() {
	if a.rng == nil {
		return
	}
	for len(a.hand) < cardSlots && a.progress+1e-9 >= 1 && len(a.deck) > 0 {
		a.progress -= 1
		i := a.rng.IntN(len(a.deck))
		a.hand = append(a.hand, a.deck[i])
		a.deck = append(a.deck[:i], a.deck[i+1:]...)
	}
	if len(a.hand) >= cardSlots && a.progress > 1 {
		a.progress = 1
	}
}

func (a *优昙华院) hudCards() (packed, prog float64, n int) {
	a.cardMu.Lock()
	defer a.cardMu.Unlock()
	code := 0
	for i, sk := range a.hand {
		if i >= cardSlots {
			break
		}
		code |= int(sk) << (i * 4)
	}
	return float64(code), a.progress, len(a.hand)
}

func (a *优昙华院) tryCard(ctx unit.Context, s unit.Sense, enemy *unit.Snapshot) bool {
	if a.locked {
		return false
	}
	sk, ok := a.takeCard(enemy != nil)
	if !ok {
		return false
	}
	a.publish(ctx.ID)
	act := Action{Skill: sk}
	var target unit.Snapshot
	if enemy != nil {
		target = *enemy
	}
	if !a.invoke(ctx, s, target, act, sk) {
		return false
	}
	a.lockAnim(s.Time, sk)
	a.charge()
	a.publish(ctx.ID)
	return true
}

func (a *优昙华院) takeCard(hasEnemy bool) (uint8, bool) {
	a.cardMu.Lock()
	defer a.cardMu.Unlock()
	for i, sk := range a.hand {
		cost := cardCostOf(sk)
		if i+cost > len(a.hand) {
			continue
		}
		if sk != SkillDose && !hasEnemy {
			continue
		}
		if isAttackCard(sk) {
			a.upgrades[sk]++
		}
		a.hand = append(a.hand[:i], a.hand[i+cost:]...)
		a.drawReadyLocked()
		return sk, true
	}
	return 0, false
}
