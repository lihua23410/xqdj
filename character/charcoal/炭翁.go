// 炭翁把打在自己身上的伤害记成炭。视野内索到敌方就交一颗，到期才扣持有者的生命。
package 炭翁

import (
	"embed"
	"math"
	"sync"

	"xqdj/internal/unit"
)

const (
	KindCharcoal      = "炭翁"
	KindCharcoalEmber = "炭烬"
)

const (
	bodyRadius = 18.0
	bodyHP     = 100.0
	bodyCruise = 240.0
	bodyVision = 150.0
	bodyColor  = "#8d939b"

	selfEvery = 2.5
	selfValue = 6.0
	coalFuse  = 12.0
	coalWhite = 0.4
	handCD    = 1.2
	coalCap   = 6
	timeEps   = 1e-9
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindCharcoal, assets)
	p.Register(unit.Spec{
		Kind:    KindCharcoal,
		Role:    unit.RoleFighter,
		Radius:  bodyRadius,
		MaxHP:   bodyHP,
		Speed:   bodyCruise,
		Vision:  bodyVision,
		Fighter: true,
		Look: unit.Look{
			Color:      bodyColor,
			VisionRing: true,
			FX:         []string{"charcoal"},
		},
	}, func(info unit.SpawnInfo) unit.Actor {
		b := newBrain()
		b.slot = info.Slot
		return &炭翁{brain: b}
	})
	p.Register(unit.Spec{
		Kind:      KindCharcoalEmber,
		Role:      unit.RoleHelper,
		Radius:    0,
		MaxHP:     1,
		Vision:    9999, // 主人若其实还活着，烬要看得见他才停手。
		PassWalls: true,
		ArcSpan:   1, // 老鼠不偷攻击环。烬只负责主人倒下后把交出去的炭烧完。
		Look: unit.Look{
			FX: []string{"charcoal"},
		},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &烬{token: uint64(info.Slot)}
	})
}

type placed struct {
	holder uint64
	value  float64
	due    float64
}

type booked struct {
	at    float64
	value float64
}

type brain struct {
	owner    uint64
	slot     int
	nextSelf float64
	nextHand float64
	coals    []placed
	pending  []booked
	kinds    map[uint64]string
}

func newBrain() *brain {
	return &brain{
		nextSelf: selfEvery,
		kinds:    map[uint64]string{},
	}
}

var orphans sync.Map // owner id -> *brain

type 炭翁 struct {
	brain *brain
}

func (a *炭翁) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		a.onHit(ctx, e)
	case unit.Sense:
		a.onSense(ctx, e)
	}
}

func (a *炭翁) onHit(ctx unit.Context, d unit.IncomingDamage) {
	if a.brain.owner == 0 {
		a.brain.owner = ctx.ID
	}
	if d.From == ctx.ID || a.brain.kinds[d.From] == KindCharcoal {
		unit.ConfirmHit(ctx, d)
		return
	}
	unit.BlockHit(ctx, d)
	if d.Amount <= 0 {
		return
	}
	a.brain.pending = append(a.brain.pending, booked{at: d.Time, value: d.Amount})
}

func (a *炭翁) onSense(ctx unit.Context, s unit.Sense) {
	if a.brain.owner == 0 {
		a.brain.owner = ctx.ID
	}
	a.brain.slot = s.Self.Slot
	for i := range s.Nearby {
		if s.Nearby[i].Kind != "" {
			a.brain.kinds[s.Nearby[i].ID] = s.Nearby[i].Kind
		}
	}
	a.steer(ctx, s)
	died := a.brain.catchUp(ctx, s)
	if died {
		if a.brain.hasForeign() {
			if _, loaded := orphans.LoadOrStore(ctx.ID, a.brain); !loaded {
				ctx.Out <- unit.Spawn{
					Kind: KindCharcoalEmber,
					X:    s.Self.X, Y: s.Self.Y,
					Slot: int(ctx.ID),
				}
			}
		}
		a.brain.emitFX(ctx, s)
		return
	}
	a.brain.transfer(ctx, s)
	a.brain.emitFX(ctx, s)
}

func (a *炭翁) steer(ctx unit.Context, s unit.Sense) {
	t := unit.Seek(s)
	if t == nil {
		return
	}
	dx, dy := t.X-s.Self.X, t.Y-s.Self.Y
	if math.Hypot(dx, dy) < 1e-6 {
		return
	}
	ctx.Out <- unit.SetFSDirection{UnitID: ctx.ID, VX: dx, VY: dy}
}

func (b *brain) catchUp(ctx unit.Context, s unit.Sense) bool {
	now := s.Time
	hp := s.Self.HP
	for {
		expT, expOK := b.nextDue(now)
		bookT, bookOK := b.nextBook(now)
		if !expOK && !bookOK {
			return false
		}
		if expOK && (!bookOK || expT <= bookT+timeEps) {
			loss := b.expireAt(ctx, expT)
			hp -= loss
			if hp <= timeEps {
				b.pending = nil
				b.dropHolder(b.owner)
				return true
			}
			continue
		}
		idx := b.bookOne(now)
		if idx < 0 {
			continue
		}
		loss := b.evictOldest(ctx, b.coals[idx].holder, idx)
		hp -= loss
		if hp <= timeEps {
			b.pending = nil
			b.dropHolder(b.owner)
			return true
		}
	}
}

func (b *brain) nextDue(now float64) (float64, bool) {
	t := math.Inf(1)
	ok := false
	for _, c := range b.coals {
		if c.due <= now+timeEps && c.due < t {
			t = c.due
			ok = true
		}
	}
	return t, ok
}

func (b *brain) nextBook(now float64) (float64, bool) {
	t := math.Inf(1)
	ok := false
	if b.nextSelf <= now+timeEps {
		t = b.nextSelf
		ok = true
	}
	for _, h := range b.pending {
		if h.at <= now+timeEps && h.at < t {
			t = h.at
			ok = true
		}
	}
	return t, ok
}

func (b *brain) bookOne(now float64) int {
	hitI := -1
	hitAt := math.Inf(1)
	for i, h := range b.pending {
		if h.at <= now+timeEps && h.at < hitAt {
			hitAt = h.at
			hitI = i
		}
	}
	if b.nextSelf <= now+timeEps && (hitI < 0 || b.nextSelf <= hitAt+timeEps) {
		b.coals = append(b.coals, placed{holder: b.owner, value: selfValue, due: b.nextSelf + coalFuse})
		b.nextSelf += selfEvery
		return len(b.coals) - 1
	}
	if hitI < 0 {
		return -1
	}
	h := b.pending[hitI]
	b.pending = append(b.pending[:hitI], b.pending[hitI+1:]...)
	b.coals = append(b.coals, placed{holder: b.owner, value: h.value, due: h.at + coalFuse})
	return len(b.coals) - 1
}

func (b *brain) expireAt(ctx unit.Context, t float64) float64 {
	sum := map[uint64]float64{}
	keep := make([]placed, 0, len(b.coals))
	for _, c := range b.coals {
		if c.due <= t+timeEps {
			sum[c.holder] += c.value
			continue
		}
		keep = append(keep, c)
	}
	b.coals = keep
	var self float64
	for id, amt := range sum {
		if amt <= 0 || id == 0 {
			continue
		}
		if id == b.owner {
			self += amt
			ctx.Out <- unit.Damage{From: b.owner, To: id, Amount: amt, NoFreeze: true}
			continue
		}
		ctx.Out <- unit.Damage{From: b.owner, To: id, Amount: amt}
	}
	return self
}

func (b *brain) dropHolder(id uint64) {
	keep := b.coals[:0]
	for _, c := range b.coals {
		if c.holder != id {
			keep = append(keep, c)
		}
	}
	b.coals = keep
}

func (b *brain) hasForeign() bool {
	for _, c := range b.coals {
		if c.holder != 0 && c.holder != b.owner {
			return true
		}
	}
	return false
}

func (b *brain) transfer(ctx unit.Context, s unit.Sense) {
	if s.Time+timeEps < b.nextHand {
		return
	}
	t := unit.Seek(s)
	if t == nil || dist2(s.Self, *t) > bodyVision*bodyVision+timeEps {
		return
	}
	idx, ok := b.giveOne(t.ID)
	if !ok {
		return
	}
	ctx.Out <- unit.FX{
		Name: "hand", Kind: KindCharcoal, UnitID: ctx.ID,
		X: s.Self.X, Y: s.Self.Y,
		VX: t.X, VY: t.Y,
		Slot: int(t.ID),
	}
	b.evictOldest(ctx, t.ID, idx)
	b.nextHand = s.Time + handCD
}

func (b *brain) giveOne(id uint64) (int, bool) {
	if id == 0 || id == b.owner {
		return -1, false
	}
	best := -1
	bestDue := math.Inf(1)
	for i, c := range b.coals {
		if c.holder != b.owner {
			continue
		}
		if c.due < bestDue {
			bestDue = c.due
			best = i
		}
	}
	if best < 0 {
		return -1, false
	}
	b.coals[best].holder = id
	return best, true
}

func (b *brain) evictOldest(ctx unit.Context, holder uint64, keep int) float64 {
	if holder == 0 || b.held(holder) <= coalCap {
		return 0
	}
	old := b.oldestExcept(holder, keep)
	if old < 0 {
		return 0
	}
	c := b.coals[old]
	b.coals = append(b.coals[:old], b.coals[old+1:]...)
	if c.value <= 0 {
		return 0
	}
	if holder == b.owner {
		ctx.Out <- unit.Damage{From: b.owner, To: holder, Amount: c.value, NoFreeze: true}
		return c.value
	}
	ctx.Out <- unit.Damage{From: b.owner, To: holder, Amount: c.value}
	return 0
}

func (b *brain) held(id uint64) int {
	n := 0
	for _, c := range b.coals {
		if c.holder == id {
			n++
		}
	}
	return n
}

func (b *brain) oldestExcept(holder uint64, keep int) int {
	best := -1
	bestDue := math.Inf(1)
	for i, c := range b.coals {
		if i == keep || c.holder != holder {
			continue
		}
		if best < 0 || c.due < bestDue-timeEps || (c.due <= bestDue+timeEps && i < best) {
			bestDue = c.due
			best = i
		}
	}
	return best
}

func (b *brain) emitFX(ctx unit.Context, s unit.Sense) {
	ctx.Out <- unit.FX{Name: "coals", Kind: KindCharcoal, UnitID: ctx.ID, Amount: -1}
	count := map[uint64]int{}
	for _, c := range b.coals {
		if c.holder == 0 {
			continue
		}
		count[c.holder]++
	}
	idx := map[uint64]int{}
	for _, c := range b.coals {
		if c.holder == 0 {
			continue
		}
		i := idx[c.holder]
		idx[c.holder] = i + 1
		n := count[c.holder]
		white := 0.0
		if s.Time+timeEps >= c.due-coalWhite {
			white = 1
		}
		ctx.Out <- unit.FX{
			Name: "coal", Kind: KindCharcoal, UnitID: ctx.ID,
			VX: float64(i), VY: float64(n),
			Amount: white,
			Slot:   int(c.holder),
		}
	}
}

func (b *brain) expireForeign(ctx unit.Context, now float64) {
	b.dropHolder(b.owner)
	for {
		expT, ok := b.nextDue(now)
		if !ok {
			return
		}
		b.expireAt(ctx, expT)
	}
}

func dist2(a, b unit.Snapshot) float64 {
	dx, dy := a.X-b.X, a.Y-b.Y
	return dx*dx + dy*dy
}

type 烬 struct {
	token uint64
}

func (h *烬) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	v, loaded := orphans.Load(h.token)
	if !loaded {
		ctx.Out <- unit.FX{Name: "coals", Kind: KindCharcoal, UnitID: ctx.ID, Amount: -1}
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.ID == h.token && o.HP > timeEps {
			orphans.Delete(h.token)
			ctx.Out <- unit.FX{Name: "coals", Kind: KindCharcoal, UnitID: ctx.ID, Amount: -1}
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			return
		}
	}
	b := v.(*brain)
	b.expireForeign(ctx, s.Time)
	if !b.hasForeign() {
		orphans.Delete(h.token)
		ctx.Out <- unit.FX{Name: "coals", Kind: KindCharcoal, UnitID: ctx.ID, Amount: -1}
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	b.emitFX(ctx, s)
}
