package 泵之心

import (
	"embed"
	"math"
	modules "xqdj/character/engine_heart/parts"
	"xqdj/character/engine_heart/parts/api"
	"xqdj/internal/unit"
)

const KindEngineHeart = "内燃机(泵之心)"
const KindHeartField = "泵之心零件场"
const KindHeartSlag = "泵之心熔渣弹"

//go:embed fx parts/*/fx.*
var assets embed.FS

func init() {
	p := unit.NewPack(KindEngineHeart, assets)
	for _, d := range modules.Catalog() {
		if d.Register != nil {
			d.Register(p)
		}
	}
	p.Register(unit.Spec{Kind: KindEngineHeart, Role: unit.RoleFighter, Fighter: true, Radius: 18, MaxHP: 100, Speed: 150, Vision: 9999,
		Look: unit.Look{Color: "#c88d59", Ghost: 310, FX: []string{"pump-heart"}}}, func(info unit.SpawnInfo) unit.Actor {
		return &heart{stream: info.Stream}
	})
	p.Register(unit.Spec{Kind: KindHeartField, Role: unit.RoleHelper, RequiresOwner: true, Nonsolid: true, Vision: 9999,
		Look: unit.Look{FX: []string{"pump-floor"}}}, func(info unit.SpawnInfo) unit.Actor {
		return &floor{owner: info.OwnerID}
	})
	p.Register(unit.Spec{Kind: KindHeartSlag, Role: unit.RoleProjectile, Radius: 5, Speed: 400, RequiresOwner: true,
		Look: unit.Look{Color: "#ffc260", Trail: true, Glow: true}}, func(info unit.SpawnInfo) unit.Actor {
		return &slag{slot: info.Slot}
	})
}

type heart struct {
	stream        uint64
	boot          bool
	start, period float64
	fired, echoed bool
	beats         int
	weapons       map[int]unit.Actor
}

func (h *heart) syncWeapons(ctx unit.Context, w *world) {
	if h.weapons == nil {
		h.weapons = map[int]unit.Actor{}
	}
	for _, d := range modules.Catalog() {
		if d.Weapon == nil {
			continue
		}
		if w.has(d.ID) {
			if h.weapons[d.ID] == nil {
				h.weapons[d.ID] = d.Weapon()
			}
		} else if actor := h.weapons[d.ID]; actor != nil {
			actor.Handle(ctx, api.Detached{})
			delete(h.weapons, d.ID)
		}
	}
}
func (h *heart) weaponsHandle(ctx unit.Context, ev unit.Event) {
	for _, d := range modules.Catalog() {
		if actor := h.weapons[d.ID]; actor != nil {
			actor.Handle(ctx, ev)
		}
	}
}

func (h *heart) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		w := worldFor(ctx.State, ctx.ID)
		w.Lock()
		blocked := w.blockOffer(ctx, e)
		w.Unlock()
		if blocked {
			unit.BlockHit(ctx, e)
		} else {
			unit.ConfirmHit(ctx, e)
		}
	case unit.Stunned:
		w := worldFor(ctx.State, ctx.ID)
		w.Lock()
		w.suspended = true
		h.weaponsHandle(ctx, e)
		w.Unlock()
	case unit.Collision, unit.WallHit, unit.FanHit:
		w := worldFor(ctx.State, ctx.ID)
		w.Lock()
		h.syncWeapons(ctx, w)
		h.weaponsHandle(ctx, ev)
		w.Unlock()
	case unit.Sense:
		h.sense(ctx, e)
	}
}

func (h *heart) sense(ctx unit.Context, s unit.Sense) {
	w := worldFor(ctx.State, ctx.ID)
	w.Lock()
	defer w.Unlock()
	w.self = s.Self
	if !h.boot {
		h.boot = true
		w.initialize(h.stream, s)
		h.period, h.start = 1, s.Time
		h.begin(ctx, w, s)
		ctx.Out <- ctx.Born(unit.Spawn{Kind: KindHeartField, OwnerID: ctx.ID, Slot: s.Self.Slot})
		ctx.Out <- unit.NoFrameFreeze{UnitID: ctx.ID, Hold: true}
	}
	h.syncWeapons(ctx, w)
	h.weaponsHandle(ctx, s)
	if w.suspended {
		w.suspended = false
		h.start, h.fired, h.echoed = s.Time, false, false
		h.period = w.period()
		h.begin(ctx, w, s)
	} else if s.Time >= h.start+h.period-1e-9 {
		h.start, h.fired, h.echoed = s.Time, false, false
		h.period = w.period()
		h.begin(ctx, w, s)
	}
	phase := (s.Time - h.start) / h.period
	w.phase, w.periodValue, w.beats = phase, h.period, h.beats
	w.radius = w.reach()
	if phase < .4 {
		h.pull(ctx, w, s)
	} else if !h.fired {
		h.fired = true
		h.contract(ctx, w, s, false)
	}
	if phase >= .6 && !h.echoed && w.build().Echo {
		h.echoed = true
		h.contract(ctx, w, s, true)
	}
	if !w.build().Capture {
		w.ammo = false
	}
}

func (h *heart) begin(ctx unit.Context, w *world, s unit.Sense) {
	h.beats++
	w.nudge(s.Field, s.Self.X, s.Self.Y)
	if !s.Knock {
		x, y := heading(s.Self)
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: x * 60, VY: y * 60}
		ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: 150}
	}
}

func (h *heart) pull(ctx unit.Context, w *world, s unit.Sense) {
	b := w.build()
	force := b.Pull
	for _, o := range s.Nearby {
		if !unit.Hittable(o, s.Self.Slot) || math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) > w.reach()+o.Radius {
			continue
		}
		x, y := normalized(s.Self.X-o.X, s.Self.Y-o.Y)
		ax, ay := x*force, y*force
		if b.Flow != nil {
			ax, ay = b.Flow(x, y, force)
		}
		ctx.Out <- unit.Force{UnitID: o.ID, AX: ax, AY: ay}
	}
}

// Use a beat-local stream so extra Sense frames and part draws cannot reroll it.
func (h *heart) partDash(w *world, self unit.Snapshot) (float64, float64, bool) {
	nearest := -1
	best := math.MaxFloat64
	for i, p := range w.parts {
		if p.Equipped || p.Flying {
			continue
		}
		dx, dy := p.X-self.X, p.Y-self.Y
		if d := dx*dx + dy*dy; d < best {
			nearest, best = i, d
		}
	}
	if nearest < 0 || best < 1e-12 {
		return 0, 0, false
	}
	rng := unit.NewRNG(unit.DeriveStream(h.stream, uint64(h.beats), 0x50415254))
	if rng.IntN(100) >= 33 {
		return 0, 0, false
	}
	p := w.parts[nearest]
	x, y := normalized(p.X-self.X, p.Y-self.Y)
	return x, y, true
}

func (h *heart) contract(ctx unit.Context, w *world, s unit.Sense, echo bool) {
	b := w.build()
	dmg, push := b.Damage, b.Push
	if echo {
		dmg *= .4
		push *= .4
	}
	for _, o := range s.Nearby {
		if !unit.Hittable(o, s.Self.Slot) || math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) > w.reach()+o.Radius {
			continue
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: o.ID, Amount: dmg, NoFreeze: true}
		x, y := normalized(o.X-s.Self.X, o.Y-s.Self.Y)
		ctx.Out <- unit.AddFS{UnitID: o.ID, DX: x, DY: y, BaseSpeed: push, OnWall: true, ExpiresAt: s.Time + .12, Token: unit.DeriveStream(ctx.ID, o.ID, 101)}
	}
	seekingPart := false
	if !echo && !s.Knock {
		x, y := heading(s.Self)
		if px, py, ok := h.partDash(w, s.Self); ok {
			x, y, seekingPart = px, py, true
			ctx.Out <- unit.SetFSDirection{UnitID: ctx.ID, VX: x, VY: y}
			ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: x * 150, VY: y * 150}
		} else if b.Steer != nil {
			x, y = b.Steer(x, y, h.beats)
		}
		speed := b.Dash
		ctx.Out <- unit.AddFS{UnitID: ctx.ID, DX: x, DY: y, BaseSpeed: speed, OnWall: true, ExpiresAt: s.Time + .13, Token: api.CoreDashToken(ctx.ID)}
	}
	if !echo && w.ammo && b.Capture {
		x, y := heading(s.Self)
		ctx.Out <- ctx.Born(unit.Spawn{Kind: KindHeartSlag, OwnerID: ctx.ID, Slot: s.Self.Slot, X: s.Self.X + x*25, Y: s.Self.Y + y*25, VX: x * 400, VY: y * 400})
		w.ammo = false
	}
	amount := 1.0
	if echo {
		amount = .4
	}
	ctx.Out <- unit.FX{Name: "heart-beat", Kind: KindEngineHeart, UnitID: ctx.ID, X: s.Self.X, Y: s.Self.Y, VX: b.Range, VY: dmg, Slot: s.Self.Slot, Amount: amount}
	h.weaponsHandle(ctx, api.Pulse{Sense: s, Beat: h.beats, Echo: echo, SeekingPart: seekingPart})
}

func heading(s unit.Snapshot) (float64, float64) {
	if math.Hypot(s.VX, s.VY) > 1e-6 {
		return normalized(s.VX, s.VY)
	}
	return normalized(s.FaceX, s.FaceY)
}
func normalized(x, y float64) (float64, float64) {
	n := math.Hypot(x, y)
	if n < 1e-9 {
		return 1, 0
	}
	return x / n, y / n
}

type slag struct {
	slot int
	born float64
	boot bool
}

func (a *slag) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		if !a.boot {
			a.boot = true
			a.born = e.Time
		}
		if e.Time-a.born > 2 {
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		}
	case unit.WallHit:
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	case unit.Collision:
		if unit.Hittable(e.Other, a.slot) {
			ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: 5, NoFreeze: true}
		}
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}
