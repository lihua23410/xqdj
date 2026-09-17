package 人偶使

import "xqdj/internal/unit"

func (a *人偶使) initDeckLocked() {
	a.deck = make([]uint8, 0, deckSize)
	for i := 0; i < 4; i++ {
		a.deck = append(a.deck, SkillN26, SkillN24, CardSpirit)
	}
	a.deck = append(a.deck, SkillN62, SkillN62, SkillN22, SkillN22, CardDemon, CardBattle, CardBattle, CardHourai)
	a.rng.Shuffle(len(a.deck), func(i, j int) { a.deck[i], a.deck[j] = a.deck[j], a.deck[i] })
	a.hand = a.hand[:0]
	a.progress = 0
}

func (a *人偶使) charge() {
	a.cardMu.Lock()
	defer a.cardMu.Unlock()
	a.progress += cardFill
	a.drawReadyLocked()
}

func (a *人偶使) drawReadyLocked() {
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

func (a *人偶使) hudCards() (packed, prog float64, n int) {
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

func (a *人偶使) tryCard(ctx unit.Context, s unit.Sense, enemy *unit.Snapshot) bool {
	if a.locked {
		return false
	}
	if a.energy+1e-9 < 1 {
		return false
	}
	sk, ok := a.takeCard(s, enemy)
	if !ok {
		return false
	}
	if !a.invokePaid(ctx, s, enemy, sk, true) {
		return false
	}
	if isNumberCard(sk) {
		if a.level[sk] < 1 {
			a.level[sk] = 1
		} else {
			a.level[sk]++
		}
	}
	return true
}

func (a *人偶使) takeCard(s unit.Sense, enemy *unit.Snapshot) (uint8, bool) {
	a.cardMu.Lock()
	defer a.cardMu.Unlock()
	idx := make([]int, 0, len(a.hand))
	for i, sk := range a.hand {
		if energyCostOf(sk) > a.energy+1e-9 {
			continue
		}
		if !a.shouldCast(s, enemy, sk, true) {
			continue
		}
		if a.energy+1e-9 < energyCostOf(sk)+1 && a.energy+1e-9 < a.energyCap-1e-6 && sk != CardSpirit {
			continue
		}
		idx = append(idx, i)
	}
	if len(idx) == 0 {
		return 0, false
	}
	pick := idx[a.rng.IntN(len(idx))]
	sk := a.hand[pick]
	a.hand = append(a.hand[:pick], a.hand[pick+1:]...)
	a.drawReadyLocked()
	return sk, true
}
