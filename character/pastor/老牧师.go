// 老牧师不索敌、不转向。出伤是身外三圈经文里的字：碰到敌方才扣血，确认扣了才消失。
package 老牧师

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

const (
	KindPastor = "老牧师"
	KindGlyph  = "老牧师字"
)

const (
	bodyRadius = 18.0
	bodyHP     = 100.0
	bodyCruise = 160.0
	bodyColor  = "#f4f4f4"

	glyphDamage = 2.0
	glyphRadius = 8.0
	glyphVision = 9999.0

	screamDur = 0.696599
	timeEps   = 1e-9

	// 每一圈顺时针转一圈的对局时间，出现时在这区间里抽一次。停帧不走。
	ringTurnMin = 1.5
	ringTurnMax = 2.5
)

var ringRadius = [3]float64{46, 78, 110}

var phrases = []string{"哎呦我去", "巴哈哈哈哈", "我丢雷老某", "呦吼吼吼吼"}

var phraseDur = []float64{0.998458, 3.065034, 1.857596, 1.718277}

// runeAlphabet 的下标 +1 写进字的血量，页面用这个取汉字。
const runeAlphabet = "哎呦我去巴哈丢雷老某吼"

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindPastor, assets)
	p.Register(unit.Spec{
		Kind:    KindPastor,
		Role:    unit.RoleFighter,
		Radius:  bodyRadius,
		MaxHP:   bodyHP,
		Speed:   bodyCruise,
		Vision:  0,
		Fighter: true,
		Look: unit.Look{
			Color: bodyColor,
			FX:    []string{"pastor"},
		},
	}, func(unit.SpawnInfo) unit.Actor {
		return &老牧师{}
	})
	p.Register(unit.Spec{
		Kind:      KindGlyph,
		Role:      unit.RoleHelper,
		Radius:    glyphRadius,
		MaxHP:     16,
		Vision:    glyphVision,
		PassWalls: true,
		Nonsolid:  true,
		Look: unit.Look{
			Color: "#161616",
			FX:    []string{"pastor-glyph"},
		},
	}, func(unit.SpawnInfo) unit.Actor {
		return &字{}
	})
}

type glyphSlot struct {
	id   uint64
	live bool
	code int
}

type ring struct {
	active bool
	slots  []glyphSlot
	spin   float64
}

type 老牧师 struct {
	rings       [3]ring
	omegas      [3]float64
	phrase      int
	verseUntil  float64
	screamUntil float64
	lastT       float64
	haveT       bool
	started     bool
	sx, sy      float64
	slot        int
	known       map[uint64]struct{}
	rng         *rand.Rand
}

func (a *老牧师) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		a.onHit(ctx, e)
	case unit.Sense:
		a.onSense(ctx, e)
	}
}

func (a *老牧师) onHit(ctx unit.Context, d unit.IncomingDamage) {
	unit.ConfirmHit(ctx, d)
	if d.Amount <= 0 {
		return
	}
	if a.screamUntil > 0 && d.Time+timeEps < a.screamUntil {
		return
	}
	a.screamUntil = d.Time + screamDur
	ctx.Out <- unit.FX{
		Name:   "scream",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		X:      a.sx,
		Y:      a.sy,
		Slot:   a.slot,
		Amount: screamDur,
	}
}

func (a *老牧师) onSense(ctx unit.Context, s unit.Sense) {
	if a.known == nil {
		a.known = map[uint64]struct{}{}
	}
	a.sx, a.sy, a.slot = s.Self.X, s.Self.Y, s.Self.Slot
	if !a.started {
		a.started = true
		a.rng = rand.New(rand.NewPCG(s.Self.ID|1, 0x6F617374))
		ctx.Out <- unit.NoFrameFreeze{UnitID: ctx.ID, Hold: true}
		a.phrase = 0
		a.rings[0] = newRing(0)
		a.ensureOmega(0)
		a.haveT = true
		a.lastT = s.Time
		a.verseUntil = s.Time + phraseDur[0]
		a.spawnRing(ctx, s, 0)
		a.fxVerse(ctx, s, 0)
		return
	}
	a.sync(s)
	if a.verseUntil > 0 && s.Time+timeEps >= a.verseUntil {
		a.push(ctx, s)
	}
	a.advance(s.Time)
	a.place(ctx, s)
}

func (a *老牧师) push(ctx unit.Context, s unit.Sense) {
	for i := range a.rings[2].slots {
		id := a.rings[2].slots[i].id
		if id == 0 {
			continue
		}
		ctx.Out <- unit.Despawn{UnitID: id}
		delete(a.known, id)
	}
	a.phrase = (a.phrase + 1) % len(phrases)
	carried := a.rings[0].spin
	a.rings[2] = cloneRing(a.rings[1])
	a.rings[1] = cloneRing(a.rings[0])
	a.rings[0] = newRing(a.phrase)
	a.rings[0].spin = carried
	a.verseUntil += phraseDur[a.phrase]
	a.spawnRing(ctx, s, 0)
	a.fxVerse(ctx, s, a.phrase)
}

func (a *老牧师) sync(s unit.Sense) {
	seen := map[uint64]struct{}{}
	var fresh []uint64
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind != KindGlyph || o.OwnerID != s.Self.ID {
			continue
		}
		seen[o.ID] = struct{}{}
		if _, ok := a.known[o.ID]; !ok {
			fresh = append(fresh, o.ID)
		}
	}
	for i := 1; i < len(fresh); i++ {
		j := i
		for j > 0 && fresh[j] < fresh[j-1] {
			fresh[j], fresh[j-1] = fresh[j-1], fresh[j]
			j--
		}
	}
	for ri := range a.rings {
		if !a.rings[ri].active {
			continue
		}
		for si := range a.rings[ri].slots {
			sl := &a.rings[ri].slots[si]
			if sl.id == 0 {
				continue
			}
			if _, ok := seen[sl.id]; ok {
				continue
			}
			delete(a.known, sl.id)
			sl.id = 0
			sl.live = false
		}
	}
	fi := 0
	for ri := 0; ri < len(a.rings) && fi < len(fresh); ri++ {
		if !a.rings[ri].active {
			continue
		}
		for si := range a.rings[ri].slots {
			if fi >= len(fresh) {
				return
			}
			sl := &a.rings[ri].slots[si]
			if !sl.live || sl.id != 0 {
				continue
			}
			sl.id = fresh[fi]
			a.known[sl.id] = struct{}{}
			fi++
		}
	}
}

func (a *老牧师) place(ctx unit.Context, s unit.Sense) {
	for ri := range a.rings {
		if !a.rings[ri].active {
			continue
		}
		n := len(a.rings[ri].slots)
		for si := range a.rings[ri].slots {
			sl := &a.rings[ri].slots[si]
			if !sl.live || sl.id == 0 {
				continue
			}
			x, y := glyphXY(s.Self.X, s.Self.Y, ri, si, n, a.rings[ri].spin)
			ctx.Out <- unit.Teleport{UnitID: sl.id, X: x, Y: y}
		}
	}
}

func (a *老牧师) spawnRing(ctx unit.Context, s unit.Sense, ri int) {
	n := len(a.rings[ri].slots)
	for si := range a.rings[ri].slots {
		sl := &a.rings[ri].slots[si]
		if !sl.live || sl.id != 0 {
			continue
		}
		x, y := glyphXY(s.Self.X, s.Self.Y, ri, si, n, a.rings[ri].spin)
		ctx.Out <- unit.Spawn{
			Kind:    KindGlyph,
			X:       x,
			Y:       y,
			VX:      float64(sl.code),
			OwnerID: ctx.ID,
			Slot:    s.Self.Slot,
		}
	}
}

func (a *老牧师) fxVerse(ctx unit.Context, s unit.Sense, phrase int) {
	ctx.Out <- unit.FX{
		Name:   "verse",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		X:      s.Self.X,
		Y:      s.Self.Y,
		Slot:   s.Self.Slot,
		Amount: float64(phrase),
	}
}

func newRing(phrase int) ring {
	rs := []rune(phrases[phrase])
	slots := make([]glyphSlot, len(rs))
	for i, r := range rs {
		slots[i] = glyphSlot{live: true, code: runeCode(r)}
	}
	return ring{active: true, slots: slots}
}

func runeCode(r rune) int {
	for i, c := range []rune(runeAlphabet) {
		if c == r {
			return i + 1
		}
	}
	return 1
}

func (a *老牧师) advance(t float64) {
	if !a.haveT {
		a.lastT = t
		a.haveT = true
		return
	}
	dt := t - a.lastT
	if dt < 0 {
		dt = 0
	}
	a.lastT = t
	for i := range a.rings {
		if !a.rings[i].active {
			continue
		}
		a.ensureOmega(i)
		a.rings[i].spin += a.omegas[i] * dt
	}
}

func (a *老牧师) ensureOmega(i int) {
	if a.omegas[i] != 0 {
		return
	}
	if a.rng == nil {
		a.rng = rand.New(rand.NewPCG(1, 1))
	}
	span := ringTurnMax - ringTurnMin
	turn := ringTurnMin + a.rng.Float64()*span
	a.omegas[i] = 2 * math.Pi / turn
}

func cloneRing(r ring) ring {
	if r.slots != nil {
		slots := make([]glyphSlot, len(r.slots))
		copy(slots, r.slots)
		r.slots = slots
	}
	return r
}

func glyphXY(x, y float64, ring, i, n int, spin float64) (float64, float64) {
	if n < 1 {
		n = 1
	}
	ang := -math.Pi/2 + float64(i)*2*math.Pi/float64(n) - spin
	r := ringRadius[ring]
	return x + r*math.Cos(ang), y + r*math.Sin(ang)
}

type pendingHit struct {
	id uint64
	hp float64
}

type 字 struct {
	code    int
	latched bool
	pending []pendingHit
}

func (g *字) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !g.latched {
		g.code = int(math.Round(s.Self.VX))
		if g.code < 1 {
			g.code = 1
		}
		g.latched = true
		ctx.Out <- unit.SetHP{UnitID: ctx.ID, HP: float64(g.code), MaxHP: 16}
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	}
	if len(g.pending) > 0 {
		if glyphConfirmed(s, g.pending) {
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			return
		}
		g.pending = nil
	}
	g.strike(ctx, s)
}

func (g *字) strike(ctx unit.Context, s unit.Sense) {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !unit.Hittable(*o, s.Self.Slot) {
			continue
		}
		dx, dy := o.X-s.Self.X, o.Y-s.Self.Y
		reach := s.Self.Radius + o.Radius
		if dx*dx+dy*dy > reach*reach {
			continue
		}
		g.pending = append(g.pending, pendingHit{id: o.ID, hp: o.HP})
		ctx.Out <- unit.Damage{
			From:   ctx.ID,
			To:     o.ID,
			Amount: glyphDamage,
		}
	}
}

func glyphConfirmed(s unit.Sense, pending []pendingHit) bool {
	loss := false
	for _, p := range pending {
		var found *unit.Snapshot
		for i := range s.Nearby {
			if s.Nearby[i].ID == p.id {
				found = &s.Nearby[i]
				break
			}
		}
		if found == nil || found.HP < p.hp-1e-6 {
			loss = true
		}
	}
	return loss
}
