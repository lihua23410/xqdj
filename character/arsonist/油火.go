package 纵火者

import (
	"encoding/json"
	"math"
	"sort"
	"xqdj/internal/unit"
)

const (
	phaseIdle = iota
	phasePaint
	phaseCool
)

type walker struct {
	inside       bool
	phase        int
	left, until  float64
	x, y         float64
	seen         uint64
	trails, seed uint64
}
type scorch struct {
	on          bool
	next, until float64
	seen        uint64
}

// bottleTrail 记下这只油瓶上一拍的位置，用来把飞行路径连成一道细油。
type bottleTrail struct {
	x, y, distance float64
	seen, seed     uint64
}
type 油火 struct {
	owner   uint64
	slot    int
	last    float64
	seen    bool
	frame   uint64
	nextTok uint64
	surface oilSurface
	walk    map[uint64]*walker
	bottles map[uint64]*bottleTrail
	burn    map[uint64]*scorch
	slow    map[uint64]uint64
	noBurns json.RawMessage
}

func (f *油火) Handle(ctx unit.Context, ev unit.Event) {
	if s, ok := ev.(unit.Sense); ok {
		f.onSense(ctx, s)
	}
}
func (f *油火) ownerAlive(s unit.Sense) bool {
	for _, o := range s.Nearby {
		if o.ID == f.owner && o.Role == unit.RoleFighter && o.HP > 0 {
			return true
		}
	}
	return false
}
func (f *油火) onSense(ctx unit.Context, s unit.Sense) {
	if !f.ownerAlive(s) {
		f.clearSlow(ctx)
		f.surface = oilSurface{}
		f.walk = nil
		f.burn = nil
		f.emit(ctx)
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	dt := 0.0
	if f.seen {
		dt = math.Max(0, s.Time-f.last)
	}
	f.last, f.seen = s.Time, true
	f.frame++
	f.surface.advance(s.Time)
	f.takeMarks(ctx, s)
	f.bottleTrails(ctx, s)
	f.walkers(ctx, s, dt)
	f.surface.advance(s.Time)
	f.scorch(ctx, s)
	if s.ForecastThreats {
		f.forecastIgnition(ctx, s)
	}
	f.syncSlow(ctx, s)
	f.emit(ctx)
}

// Stepping into flames can be avoided; ticks of an existing burn follow the
// victim after leaving the oil and do not get a spatial forecast.
func (f *油火) forecastIgnition(ctx unit.Context, s unit.Sense) {
	var areas []unit.Threat
	for _, o := range s.Nearby {
		if !unit.Hittable(o, f.slot) || o.HP <= 0 {
			continue
		}
		spec, ok := unit.Lookup(o.Kind)
		if !ok || !spec.ObserveThreats || f.burn[o.ID] != nil {
			continue
		}
		const horizon = .15
		reach := o.Radius + math.Max(400, math.Hypot(o.VX, o.VY))*horizon
		left, right := gridCoord(o.X-reach), gridCoord(o.X+reach)
		for y := gridCoord(o.Y - reach); y <= gridCoord(o.Y+reach); y++ {
			for x := left; x <= right; {
				c := f.surface.cell(point{x, y})
				if c == nil || c.state != flame || c.due <= s.Time {
					x++
					continue
				}
				start, until := x, math.Min(s.Time+horizon, c.due)
				for x++; x <= right; x++ {
					c = f.surface.cell(point{x, y})
					if c == nil || c.state != flame || c.due <= s.Time {
						break
					}
					until = math.Min(until, c.due)
				}
				areas = append(areas, unit.Threat{Shape: unit.ThreatRect, From: f.owner, Slot: f.slot, Target: o.ID,
					X: float64(start) * surfaceStep, Y: float64(y) * surfaceStep,
					X2: float64(x) * surfaceStep, Y2: float64(y+1) * surfaceStep,
					ActiveAt: s.Time, Until: until,
				})
			}
		}
	}
	ctx.Out <- unit.SetThreats{UnitID: ctx.ID, Areas: areas}
}
func (f *油火) takeMarks(ctx unit.Context, s unit.Sense) {
	for _, o := range s.Nearby {
		if o.OwnerID != f.owner {
			continue
		}
		switch o.Kind {
		case KindArsonOilMark:
			f.surface.addOilSplash(o.X, o.Y, oilRadius, unit.DeriveStream(ctx.Stream, o.ID, unit.StreamChild), s.Time)
			ctx.Out <- unit.Despawn{UnitID: o.ID}
		case KindArsonFireMark:
			f.surface.addFire(o.X, o.Y, fireRadius, s.Time)
			ctx.Out <- unit.Despawn{UnitID: o.ID}
		}
	}
}

// bottleTrails 把油瓶上一拍到这一拍的位置连成一道细油。瓶子被挡住碎裂时
// 不会留下空档：碎落点本身就被随即铺下的泼溅油面盖住。
func (f *油火) bottleTrails(ctx unit.Context, s unit.Sense) {
	for _, o := range s.Nearby {
		if o.OwnerID != f.owner || o.Kind != KindArsonOil {
			continue
		}
		if f.bottles == nil {
			f.bottles = map[uint64]*bottleTrail{}
		}
		t := f.bottles[o.ID]
		if t == nil {
			f.bottles[o.ID] = &bottleTrail{x: o.X, y: o.Y, seen: f.frame, seed: unit.DeriveStream(ctx.Stream, o.ID, unit.StreamChild)}
			continue
		}
		t.seen = f.frame
		if math.Hypot(o.X-t.x, o.Y-t.y) > 1e-6 {
			f.surface.addOilRibbon(t.x, t.y, o.X, o.Y, bottleTrailWidth, bottleTrailWidth, t.distance, t.seed, s.Time)
			t.distance += math.Hypot(o.X-t.x, o.Y-t.y)
		}
		t.x, t.y = o.X, o.Y
	}
	for id, t := range f.bottles {
		if t.seen != f.frame {
			delete(f.bottles, id)
		}
	}
}
func bodyUnit(o unit.Snapshot) bool { return o.Role == unit.RoleFighter || o.Mortal }
func isGap(moved, speed, dt float64) bool {
	if moved <= 48 {
		return false
	}
	return moved > math.Max(640, speed)*math.Max(0, dt)*1.5+12
}

// trailRadius 让拖尾贴着油滩的那一头最粗，越拖越收回到身体半径。等粗的
// 胶囊和大圆盘几乎外切，接缝处会留下细腰和两个凹角。
func trailRadius(base, drawn float64) float64 {
	if drawn >= trailLen {
		return base
	}
	return base * (1 + (trailHead-1)*(1-drawn/trailLen))
}
func (f *油火) walkers(ctx unit.Context, s unit.Sense, dt float64) {
	if f.walk == nil {
		f.walk = map[uint64]*walker{}
	}
	for _, o := range s.Nearby {
		if !bodyUnit(o) || o.HP <= 0 {
			continue
		}
		inside := f.surface.touches(o, oil)
		w := f.walk[o.ID]
		if w == nil {
			f.walk[o.ID] = &walker{inside: inside, x: o.X, y: o.Y, seen: f.frame}
			continue
		}
		w.seen = f.frame
		moved := math.Hypot(o.X-w.x, o.Y-w.y)
		gap := isGap(moved, math.Hypot(o.VX, o.VY), dt)
		if w.phase == phaseCool && s.Time+1e-9 >= w.until {
			w.phase = phaseIdle
		}
		if w.phase == phaseIdle && w.inside && !inside && moved > 1e-6 {
			// Burning oil away under a stationary body is not a body leaving oil.
			prior := o
			prior.X, prior.Y = w.x, w.y
			if f.surface.touches(prior, oil) {
				w.phase = phasePaint
				w.left = trailLen
				w.trails++
				w.seed = unit.DeriveStream(ctx.Stream, o.ID^w.trails*0x9e3779b97f4a7c15, unit.StreamChild)
			}
		}
		if w.phase == phasePaint {
			if gap {
				f.surface.addOil(o.X, o.Y, o.X, o.Y, o.Radius, s.Time)
			} else if moved > 1e-6 {
				use := math.Min(moved, w.left)
				t := use / moved
				drawn := trailLen - w.left
				f.surface.addOilRibbon(w.x, w.y, w.x+(o.X-w.x)*t, w.y+(o.Y-w.y)*t,
					trailRadius(o.Radius, drawn), trailRadius(o.Radius, drawn+use), drawn, w.seed, s.Time)
				w.left -= use
				if w.left <= 1e-6 {
					w.left = 0
					w.phase = phaseCool
					w.until = s.Time + trailCool
				}
			}
		}
		w.inside = inside
		w.x, w.y = o.X, o.Y
	}
	for id, w := range f.walk {
		if w.seen != f.frame {
			delete(f.walk, id)
		}
	}
}
func (f *油火) scorch(ctx unit.Context, s unit.Sense) {
	for _, o := range s.Nearby {
		if !unit.Hittable(o, f.slot) || o.HP <= 0 {
			continue
		}
		touching := f.surface.touches(o, flame)
		b := f.burn[o.ID]
		if b == nil {
			if !touching {
				continue
			}
			if f.burn == nil {
				f.burn = map[uint64]*scorch{}
			}
			b = &scorch{}
			f.burn[o.ID] = b
		}
		b.seen = f.frame
		if !touching && b.on && s.Time > b.until+1e-9 {
			b.on = false
		}
		if touching {
			if !b.on {
				b.on = true
				b.next = s.Time + burnTick
			}
			b.until = s.Time + burnLife
		}
		if b.on && f.surface.touches(o, oil) {
			f.surface.igniteBody(o, s.Time)
			if f.surface.touches(o, flame) {
				b.until = s.Time + burnLife
			}
		}
		for b.on && s.Time+1e-9 >= b.next && b.next <= b.until+1e-9 {
			ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: burnTickDmg, NoFreeze: true}
			b.next += burnTick
		}
		if s.Time > b.until+1e-9 {
			b.on = false
		}
	}
	for id, b := range f.burn {
		if b.seen != f.frame || !b.on {
			delete(f.burn, id)
		}
	}
}

func (f *油火) mintSlow(id uint64) uint64 {
	if f.slow == nil {
		f.slow = map[uint64]uint64{}
	}
	if tok, ok := f.slow[id]; ok {
		return tok
	}
	f.nextTok++
	f.slow[id] = f.nextTok
	return f.nextTok
}

func (f *油火) syncSlow(ctx unit.Context, s unit.Sense) {
	in := map[uint64]struct{}{}
	for _, o := range s.Nearby {
		if !unit.Hittable(o, f.slot) || o.HP <= 0 {
			continue
		}
		if !f.surface.touches(o, oil) && !f.surface.touches(o, flame) {
			continue
		}
		in[o.ID] = struct{}{}
		ctx.Out <- unit.AddFSComponent{UnitID: o.ID, Zone: unit.FSZoneM, Token: f.mintSlow(o.ID), Value: oilSlow}
	}
	for id, tok := range f.slow {
		if _, ok := in[id]; !ok {
			ctx.Out <- unit.RemoveFSComponent{UnitID: id, Token: tok}
			delete(f.slow, id)
		}
	}
}

func (f *油火) clearSlow(ctx unit.Context) {
	for id, tok := range f.slow {
		ctx.Out <- unit.RemoveFSComponent{UnitID: id, Token: tok}
	}
	f.slow = nil
}

type burningView struct {
	ID    uint64  `json:"id"`
	Until float64 `json:"until"`
}

func (f *油火) emit(ctx unit.Context) {
	ctx.Out <- unit.FX{Name: "arson-surface", UnitID: ctx.ID, Kind: KindArsonField, Slot: f.slot, Amount: float64(f.surface.revision), Data: f.surface.payload()}
	if f.noBurns == nil {
		f.noBurns = json.RawMessage("[]")
	}
	data := f.noBurns
	if len(f.burn) > 0 {
		burns := make([]burningView, 0, len(f.burn))
		for id, b := range f.burn {
			if b.on {
				burns = append(burns, burningView{id, b.until})
			}
		}
		sort.Slice(burns, func(i, j int) bool { return burns[i].ID < burns[j].ID })
		var err error
		data, err = json.Marshal(burns)
		if err != nil {
			panic(err)
		}
	}
	ctx.Out <- unit.FX{Name: "arson-burning", UnitID: ctx.ID, Kind: KindArsonField, Slot: f.slot, Data: data}
}
