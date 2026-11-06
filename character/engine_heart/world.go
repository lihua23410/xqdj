package 泵之心

import (
	"encoding/json"
	"math"
	"sort"
	"sync"
	modules "xqdj/character/engine_heart/parts"
	"xqdj/character/engine_heart/parts/api"
	"xqdj/internal/unit"
)

type part struct {
	Type         int     `json:"type"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	VX           float64 `json:"vx"`
	VY           float64 `json:"vy"`
	Equipped     bool    `json:"equipped"`
	Flying       bool    `json:"flying"`
	Dropped      uint64  `json:"-"`
	RestX, RestY float64 `json:"-"`
	RestSet      bool    `json:"-"`
}

const partClearance = 26.0
const partPickupRadius = 18.0

type world struct {
	sync.Mutex
	parts                            []part
	stack                            []int
	self                             unit.Snapshot
	phase, periodValue, radius, wear float64
	beats                            int
	dropSeq                          uint64
	suspended, ammo                  bool
	winch                            int
	initialized                      bool
	seen                             map[uint64]seenAttack
	blocked                          map[uint64]float64
	lastHP                           float64
	observedHP                       bool
	field                            unit.Field
	outline                          []scenePoint
}
type scenePoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
type worlds struct {
	sync.Mutex
	owners map[uint64]*world
}

func worldFor(state *unit.MatchState, id uint64) *world {
	all := unit.StateFor[worlds](state)
	all.Lock()
	defer all.Unlock()
	if all.owners == nil {
		all.owners = make(map[uint64]*world)
	}
	if all.owners[id] == nil {
		all.owners[id] = &world{winch: -1}
	}
	return all.owners[id]
}
func (w *world) has(t int) bool {
	for _, p := range w.parts {
		if p.Type == t && p.Equipped {
			return true
		}
	}
	return false
}

func (w *world) installPart(i int) bool {
	p := &w.parts[i]
	if p.Equipped || p.Flying || w.has(p.Type) {
		return false
	}
	p.Equipped = true
	if modules.IsWeapon(p.Type) {
		w.stack = append([]int{i}, w.stack...)
	} else {
		w.stack = append(w.stack, i)
	}
	return true
}
func (w *world) build() api.Build {
	ids := make([]int, 0, len(w.stack))
	for _, p := range w.parts {
		if p.Equipped {
			ids = append(ids, p.Type)
		}
	}
	return modules.Build(ids)
}
func (w *world) period() float64 { return w.build().Period }
func (w *world) reach() float64  { return w.build().Range }
func (w *world) initialize(stream uint64, s unit.Sense) {
	if w.initialized {
		return
	}
	w.initialized = true
	w.field = s.Field
	shape := s.Field.Shape
	if shape == "" {
		shape = unit.ShapeHex
	}
	edge := unit.MakeOutline(shape, s.Field.Extent)
	for i := 0; i < 96; i++ {
		x, y, _, _ := edge.SampleEdge(float64(i)/96, 0)
		w.outline = append(w.outline, scenePoint{math.Round(x*10) / 10, math.Round(y*10) / 10})
	}
	rng := unit.NewRNG(stream)
	for _, definition := range modules.Select(rng, 5) {
		var p part
		p.Type = definition.ID
		found := false
		for attempt := 0; attempt < 500; attempt++ {
			x, y, ok := s.Field.RandomWalkable(rng, partClearance)
			if !ok {
				continue
			}
			if math.Hypot(x-s.Self.X, y-s.Self.Y) < 56 {
				continue
			}
			clear := true
			for _, q := range w.parts {
				if math.Hypot(x-q.X, y-q.Y) < 48 {
					clear = false
					break
				}
			}
			if clear {
				p.X, p.Y = x, y
				found = true
				break
			}
		}
		if !found { // Dense fields still use a legal sample, never a raw origin.
			for attempt := 0; attempt < 100; attempt++ {
				x, y, ok := s.Field.RandomWalkable(rng, partClearance)
				if ok && s.Field.Walkable(x, y, partClearance) {
					p.X, p.Y = x, y
					found = true
					break
				}
			}
		}
		if !found {
			p.X, p.Y = s.Field.Clamp(s.Self.X+40, s.Self.Y, partClearance)
		}
		p.RestX, p.RestY, p.RestSet = p.X, p.Y, true
		w.parts = append(w.parts, p)
	}
	w.self = s.Self
	w.lastHP, w.observedHP = s.Self.HP, true
}

func (w *world) nudge(field unit.Field, _, _ float64) {
	for i := range w.parts {
		p := &w.parts[i]
		if p.Equipped || p.Flying {
			continue
		}
		if !p.RestSet {
			p.RestX, p.RestY, p.RestSet = p.X, p.Y, true
		}
		a := float64(w.beats)*2.399963 + float64(p.Type)*1.618
		x, y := p.RestX+math.Cos(a)*.35, p.RestY+math.Sin(a)*.35
		if field.Walkable(x, y, partClearance) {
			p.X, p.Y = x, y
		}
	}
}

func (w *world) hurt(amount, now float64) {
	if len(w.stack) == 0 {
		w.wear = 0
		return
	}
	w.wear += amount
	for w.wear >= 12-1e-9 && len(w.stack) > 0 {
		w.wear -= 12
		i := w.stack[len(w.stack)-1]
		w.stack = w.stack[:len(w.stack)-1]
		p := &w.parts[i]
		p.Equipped = false
		p.Flying = true
		w.dropSeq++
		p.Dropped = w.dropSeq
		a := float64(p.Type)*2.399963 + now*.7
		sx, sy := w.field.Clamp(w.self.X, w.self.Y, partClearance)
		p.X, p.Y = slide(w.field, sx, sy, math.Cos(a)*32, math.Sin(a)*32)
		p.VX, p.VY = math.Cos(a)*520, math.Sin(a)*520
	}
	if len(w.stack) == 0 {
		w.wear = 0
	}
	if !w.build().Capture {
		w.ammo = false
	}
}

type floor struct {
	owner    uint64
	last     float64
	boot     bool
	previous unit.Snapshot
}

func (f *floor) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	w := worldFor(ctx.State, f.owner)
	w.Lock()
	defer w.Unlock()
	var owner *unit.Snapshot
	for i := range s.Nearby {
		if s.Nearby[i].ID == f.owner {
			owner = &s.Nearby[i]
			break
		}
	}
	if owner == nil {
		return
	}
	dt := 0.0
	if f.boot {
		dt = math.Max(0, s.Time-f.last)
	}
	f.last = s.Time
	if !f.boot {
		f.previous = *owner
		f.boot = true
	}
	w.self = *owner
	w.field = s.Field
	if w.observedHP && owner.HP > 0 && owner.HP < w.lastHP {
		w.hurt(w.lastHP-owner.HP, s.Time)
	}
	w.lastHP, w.observedHP = owner.HP, true
	w.intercept(ctx, s, *owner, dt)
	for i := range w.parts {
		p := &w.parts[i]
		if p.Flying {
			advance(p, s.Field, dt)
		}
	}
	w.winch = -1
	if speed := w.build().WinchSpeed; speed > 0 {
		latest := -1
		for i, p := range w.parts {
			if !p.Equipped && p.Dropped > 0 && (latest < 0 || p.Dropped > w.parts[latest].Dropped) {
				latest = i
			}
		}
		if latest >= 0 && !w.parts[latest].Flying {
			p := &w.parts[latest]
			dx, dy := normalized(owner.X-p.X, owner.Y-p.Y)
			d := math.Min(speed*dt, math.Hypot(owner.X-p.X, owner.Y-p.Y))
			p.X, p.Y = slide(s.Field, p.X, p.Y, dx*d, dy*d)
			p.RestX, p.RestY, p.RestSet = p.X, p.Y, true
			w.winch = latest
		}
	}
	type pickup struct {
		i int
		t float64
	}
	var picks []pickup
	for i, p := range w.parts {
		if p.Equipped || p.Flying {
			continue
		}
		t, d := segmentDistance(f.previous.X, f.previous.Y, owner.X, owner.Y, p.X, p.Y)
		// Do not collect along teleport discontinuities.
		if math.Hypot(owner.X-f.previous.X, owner.Y-f.previous.Y) > 40 {
			t = 1
			d = math.Hypot(owner.X-p.X, owner.Y-p.Y)
		}
		if d <= owner.Radius+partPickupRadius {
			picks = append(picks, pickup{i, t})
		}
	}
	sort.SliceStable(picks, func(i, j int) bool { return picks[i].t < picks[j].t })
	for _, p := range picks {
		if !w.installPart(p.i) {
			continue
		}
		ctx.Out <- unit.FX{Name: "heart-equip", Kind: KindEngineHeart, UnitID: f.owner, X: owner.X, Y: owner.Y, Slot: owner.Slot, Amount: float64(w.parts[p.i].Type)}
	}
	f.previous = *owner
	types := make([]int, 0, len(w.stack))
	for _, i := range w.stack {
		types = append(types, w.parts[i].Type)
	}
	guards := w.guards(s.Time)
	data, _ := json.Marshal(struct {
		Outline   []scenePoint `json:"outline"`
		Guards    []api.Guard  `json:"guards"`
		Echo      bool         `json:"echo"`
		Parts     []part       `json:"parts"`
		Stack     []int        `json:"stack"`
		Phase     float64      `json:"phase"`
		Period    float64      `json:"period"`
		Radius    float64      `json:"radius"`
		Wear      float64      `json:"wear"`
		Beats     int          `json:"beats"`
		Ammo      bool         `json:"ammo"`
		Suspended bool         `json:"suspended"`
		Winch     int          `json:"winch"`
	}{w.outline, guards, w.build().Echo, w.parts, types, w.phase, w.periodValue, w.reach(), w.wear, w.beats, w.ammo, w.suspended, w.winch})
	ctx.Out <- unit.FX{Name: "heart-state", Kind: KindEngineHeart, UnitID: f.owner, Slot: owner.Slot, X: owner.X, Y: owner.Y, Data: data}
}

func segmentDistance(ax, ay, bx, by, x, y float64) (float64, float64) {
	dx, dy := bx-ax, by-ay
	n := dx*dx + dy*dy
	t := 0.0
	if n > 1e-12 {
		t = math.Max(0, math.Min(1, ((x-ax)*dx+(y-ay)*dy)/n))
	}
	return t, math.Hypot(x-ax-t*dx, y-ay-t*dy)
}

func slide(field unit.Field, x, y, dx, dy float64) (float64, float64) {
	steps := max(1, int(math.Ceil(math.Hypot(dx, dy)/2)))
	for i := 0; i < steps; i++ {
		sx, sy := dx/float64(steps), dy/float64(steps)
		if field.Walkable(x+sx, y+sy, partClearance) {
			x += sx
			y += sy
			continue
		}
		if field.Walkable(x+sx, y, partClearance) {
			x += sx
		} else if field.Walkable(x, y+sy, partClearance) {
			y += sy
		} else {
			break
		}
	}
	return x, y
}

func advance(p *part, field unit.Field, dt float64) {
	sp := math.Hypot(p.VX, p.VY)
	if sp < 1e-9 {
		p.VX, p.VY = 0, 0
		p.Flying = false
		return
	}
	t := math.Min(dt, sp/650)
	next := math.Max(0, sp-650*dt)
	distance := (sp + math.Max(0, sp-650*t)) * .5 * t
	n := max(1, int(math.Ceil(distance/2)))
	for i := 0; i < n; i++ {
		dx, dy := normalized(p.VX, p.VY)
		d := distance / float64(n)
		x, y := p.X+dx*d, p.Y+dy*d
		if field.Walkable(x, y, partClearance) {
			p.X, p.Y = x, y
			continue
		}
		cx, cy := field.Clamp(x, y, partClearance)
		nx, ny := cx-x, cy-y
		if math.Hypot(nx, ny) > 1e-8 {
			nx, ny = normalized(nx, ny)
			dot := p.VX*nx + p.VY*ny
			if dot < 0 {
				p.VX -= 2 * dot * nx
				p.VY -= 2 * dot * ny
			}
		} else {
			p.VX, p.VY = -p.VX, -p.VY
		}
		if field.Walkable(cx, cy, partClearance) {
			p.X, p.Y = cx, cy
		}
	}
	dx, dy := normalized(p.VX, p.VY)
	p.VX, p.VY = dx*next, dy*next
	if next <= 1e-8 {
		p.VX, p.VY = 0, 0
		p.Flying = false
		p.RestX, p.RestY, p.RestSet = p.X, p.Y, true
	}
}
