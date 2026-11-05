package unit

import (
	"math"
	"sync"
)

// FanHit is delivered by the engine after an attack fan has quoted its damage.
// The owner handles movement/combo effects; damage belongs to From, the fan.
type FanHit struct {
	From  uint64
	Kind  string
	Time  float64
	Other Snapshot
}

type FanContact struct {
	From      uint64
	Collision Collision
}

// FanAttack is an ability's current attack window. Epoch changes for each dash.
// Once allows one fan per window; PerTarget keeps a piercing fan until the window
// ends, with one hit per target. Sweep quotes the first contact's frontal group.
type FanAttack struct {
	Active                           bool
	Epoch, Until, ReadyAt            float64
	Damage, Cooldown                 float64
	SweepRadius                      float64
	Once, PerTarget, Sweep, NoFreeze bool
	TargetID                         uint64
	FriendKind                       string
	Hits                             int
	FX, EffectKind                   string
	self                             Snapshot
	nearby                           []Snapshot
}

type fanKey struct {
	owner uint64
	kind  string
}
type attackFans struct {
	sync.Mutex
	entries map[fanKey]*attackFanState
}
type attackFanState struct {
	sync.Mutex
	config                 FanAttack
	present, spawned, used bool
	next, born             float64
	hit                    map[uint64]bool
}

func attackState(scope *MatchState, owner uint64, kind string) *attackFanState {
	all := StateFor[attackFans](scope)
	all.Lock()
	defer all.Unlock()
	if all.entries == nil {
		all.entries = map[fanKey]*attackFanState{}
	}
	k := fanKey{owner, kind}
	if all.entries[k] == nil {
		all.entries[k] = &attackFanState{}
	}
	return all.entries[k]
}

// SyncAttackFan runs after the owner's state update. Shared mutable state stays
// in MatchState and is synchronized; no mutable actor pointer goes through Data.
func SyncAttackFan(ctx Context, s Sense, kind string, cfg FanAttack) {
	if ctx.State == nil {
		return
	} // isolated actor tests may omit a match
	st := attackState(ctx.State, ctx.ID, kind)
	st.Lock()
	defer st.Unlock()
	old := st.config
	if !cfg.Active {
		if old.Active || st.present {
			ctx.Out <- DespawnOwned{OwnerID: ctx.ID, Kind: kind}
		}
		st.config = cfg
		st.present, st.spawned, st.used = false, false, false
		return
	}
	if !old.Active || cfg.Epoch != old.Epoch {
		if st.present {
			ctx.Out <- DespawnOwned{OwnerID: ctx.ID, Kind: kind}
		}
		st.present, st.spawned, st.used = false, false, false
		st.next = cfg.ReadyAt
		st.hit = map[uint64]bool{}
	}
	cfg.self = s.Self
	if cfg.Sweep {
		cfg.nearby = append([]Snapshot(nil), s.Nearby...)
	}
	st.config = cfg
	if st.present && s.Time > st.born+1e-9 && !HasOwned(s, ctx.ID, kind) {
		st.present = false
	}
	if st.present || (cfg.Once && st.spawned) || s.Time+1e-9 < math.Max(st.next, cfg.ReadyAt) || cfg.Damage <= 0 || (cfg.Until > 0 && s.Time >= cfg.Until) {
		return
	}
	st.present, st.spawned, st.used = true, true, false
	st.born = s.Time
	SpawnAttach(ctx, s, kind)
}

// RegisterAttackFan uses the existing Attach geometry and contact rules.
// Persistent windows use Pass, which is also used by existing piercing attacks.
func (p *Pack) RegisterAttackFan(spec Spec) {
	if !spec.Attach || spec.ArcSpan <= 0 || spec.Fighter || spec.Role != RoleProjectile {
		panic("unit: attack fan requires Attach and ArcSpan")
	}
	p.Register(spec, func(info SpawnInfo) Actor {
		state := attackState(info.State, info.OwnerID, spec.Kind)
		state.Lock()
		epoch := state.config.Epoch
		state.Unlock()
		return &attackFan{state: state, owner: info.OwnerID, slot: info.Slot, span: spec.ArcSpan, radius: spec.Radius, epoch: epoch}
	})
}

type attackFan struct {
	state        *attackFanState
	owner        uint64
	slot         int
	span, radius float64
	epoch        float64
}

func (a *attackFan) Handle(ctx Context, ev Event) {
	e, ok := ev.(Collision)
	if !ok {
		return
	}
	st := a.state
	st.Lock()
	defer st.Unlock()
	c := st.config
	if !c.Active || st.used || a.epoch != c.Epoch || (c.Until > 0 && e.Time >= c.Until) || e.Time+1e-9 < math.Max(st.next, c.ReadyAt) {
		return
	}
	valid := func(o Snapshot) bool {
		return (c.TargetID == 0 || o.ID == c.TargetID) && (Hittable(o, a.slot) || (c.FriendKind != "" && o.Kind == c.FriendKind))
	}
	if !valid(e.Other) || st.hit[e.Other.ID] {
		return
	}
	quote := func(o Snapshot) {
		if st.hit[o.ID] {
			return
		}
		if c.PerTarget {
			st.hit[o.ID] = true
		}
		n := c.Hits
		if n < 1 {
			n = 1
		}
		for i := 0; i < n; i++ {
			ctx.Out <- Damage{From: ctx.ID, To: o.ID, Amount: c.Damage, NoFreeze: c.NoFreeze}
		}
		if c.FX != "" {
			ctx.Out <- FX{Name: c.FX, Kind: c.EffectKind, UnitID: a.owner, Slot: c.self.Slot, X: o.X, Y: o.Y, VX: c.self.VX, VY: c.self.VY}
		}
	}
	quote(e.Other)
	if c.Sweep {
		radius := c.SweepRadius
		if radius <= 0 {
			radius = a.radius
		}
		for _, o := range c.nearby {
			if o.ID != e.Other.ID && valid(o) && fanContains(c.self, o, radius, a.span) {
				quote(o)
			}
		}
	}
	ctx.Out <- FanContact{From: ctx.ID, Collision: e}
	if !c.PerTarget {
		st.present = false
		st.used = c.Once
		st.next = e.Time + c.Cooldown
		ctx.Out <- Despawn{UnitID: ctx.ID}
	}
}

func fanContains(self, other Snapshot, radius, span float64) bool {
	dx, dy := other.X-self.X, other.Y-self.Y
	d := math.Hypot(dx, dy)
	if d > radius+other.Radius {
		return false
	}
	fx, fy := self.VX, self.VY
	if math.Hypot(fx, fy) < 1e-9 {
		fx, fy = self.FaceX, self.FaceY
	}
	if math.Hypot(fx, fy) < 1e-9 {
		fx = 1
	}
	return d < 1e-9 || (dx*fx+dy*fy)/(d*math.Hypot(fx, fy)) >= math.Cos(span/2)-1e-9
}
