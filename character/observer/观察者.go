// 观察者：没有自己的普通攻击，持续观察一个敌人并记录经过身边的敌方弹体。
// 观测值上限 75，看着目标每秒涨 10，每记下发再加 5。满了之后先按记下的顺序复现，再打特殊胶片里的回响。
// 胶片中的视线全部保留，其余弹体保留前 51%，向上取整；回响仍按原来的打出顺序发射。
// 新记下的伤害为原来的 85%。回响在打出那一发之上再乘一次 85%。
package 观察者

import (
	"embed"
	"hash/fnv"
	"math"
	"math/rand/v2"

	tanyongchen "xqdj/character/tanyongchen"
	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const KindObserver = "观察者"
const KindObserverBeam = "观察者视线"
const KindObserverKunaiCopy = "观察者苦无复现"

const (
	observerRadius = 18.0
	observerHP     = 100.0
	observerSpeed  = 140.0
	observerVision = 9999.0
	observerColor  = "#b9d7df"

	observeGain     = 10.0
	observeDrain    = 15.0
	projectileGain  = 5.0
	gaugeMax        = 75.0
	recordRadius    = 240.0
	kindLimit       = 3
	replayScale     = 0.85
	replayGap       = 0.15
	stateGap        = 0.25
	beamDamage      = 16.0
	kunaiCopyDamage = 3.0
	beamSpeed       = 680.0
	beamRadius      = 5.0
)

func init() {
	p := unit.NewPack(KindObserver, assets)
	p.Register(unit.Spec{
		Kind: KindObserver, Role: unit.RoleFighter, Radius: observerRadius,
		MaxHP: observerHP, Speed: observerSpeed, Vision: observerVision, Fighter: true,
		Look: unit.Look{Color: observerColor, FX: []string{"observer"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &observer{rng: unit.NewRNG(info.Stream)}
	})
	p.Register(unit.Spec{
		Kind: KindObserverBeam, Role: unit.RoleProjectile, Radius: beamRadius,
		MaxHP: 1, Speed: beamSpeed,
		Look: unit.Look{Color: "#dffbff", Trail: true, FX: []string{"observer-beam"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &beam{owner: info.OwnerID, slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind: KindObserverKunaiCopy, Role: unit.RoleProjectile, Radius: 7,
		MaxHP: 1, Speed: 480,
		Look: unit.Look{Color: "#dffbff", Trail: true, FX: []string{"observer-kunai-copy"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &kunaiCopy{owner: info.OwnerID, slot: info.Slot}
	})
}

type observer struct {
	slot   int
	booted bool
	rng    *rand.Rand

	gauge       float64
	targetID    uint64
	targetGroup uint64
	lastTime    float64
	lastStateAt float64

	projectileSeen map[uint64]bool
	kinds          []string   // 先遇上的种类，最多三种，打出去之前不换
	records        []string   // 按看到的顺序，一发一条
	film           []shotNote // 上一轮真正飞出去的，含视线
	fired          []shotNote // 这一轮已经飞出去的，打完再写回 film

	replaying bool
	replay    []shotNote
	replayAt  float64

	x, y   float64
	hx, hy float64
}

func (o *observer) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		ctx.Out <- unit.ConfirmDamage{Token: e.Token, UnitID: ctx.ID, Amount: e.Amount}
	case unit.Sense:
		o.onSense(ctx, e)
	}
}

func (o *observer) onSense(ctx unit.Context, s unit.Sense) {
	o.x, o.y, o.slot = s.Self.X, s.Self.Y, s.Self.Slot
	if speed := math.Hypot(s.Self.VX, s.Self.VY); speed > 1e-6 {
		o.hx, o.hy = s.Self.VX/speed, s.Self.VY/speed
	}
	if o.projectileSeen == nil {
		o.projectileSeen = make(map[uint64]bool)
	}
	if !o.booted {
		o.booted = true
		o.lastTime = s.Time
		o.lastStateAt = s.Time
		ctx.Out <- unit.FX{Name: "observer-idle", Kind: KindObserver, UnitID: ctx.ID, Slot: o.slot, X: o.x, Y: o.y}
	}
	dt := s.Time - o.lastTime
	if dt < 0 || dt > 0.2 {
		dt = 1.0 / 60.0
	}
	o.lastTime = s.Time

	target := o.chooseTarget(s)
	if target == nil {
		o.targetID = 0
		o.gauge = math.Max(0, o.gauge-observeDrain*dt)
		o.dropReplay()
		o.emitState(ctx, s.Time)
		return
	}
	if target.ID != o.targetID {
		group := observationGroup(*target)
		if o.targetID != 0 && group != o.targetGroup {
			o.gauge = 0
		}
		o.targetGroup = group
		o.targetID = target.ID
		ctx.Out <- unit.FX{
			Name: "observer-lock", Kind: KindObserver, UnitID: ctx.ID, Slot: o.slot,
			X: target.X, Y: target.Y, VY: float64(target.ID),
		}
	}
	o.gauge = math.Min(gaugeMax, o.gauge+observeGain*dt)
	o.recordProjectiles(ctx, s)
	if o.gauge >= gaugeMax && !o.replaying {
		o.gauge = 0
		o.beginVolley(s.Time)
		ctx.Out <- unit.FX{
			Name: "observer-replay", Kind: KindObserver, UnitID: ctx.ID, Slot: o.slot,
			X: o.x, Y: o.y, Amount: float64(len(o.replay)),
		}
	}
	if o.replaying {
		o.tickReplay(ctx, s, target)
	}
	o.emitState(ctx, s.Time)
}

func (o *observer) chooseTarget(s unit.Sense) *unit.Snapshot {
	for i := range s.Nearby {
		n := &s.Nearby[i]
		if n.ID != o.targetID {
			continue
		}
		if n.Slot != s.Self.Slot && observableTarget(*n) && unit.Aimable(*n, s.Self.Slot) && unit.CanSee(s, *n) {
			return n
		}
		break
	}
	return unit.SeekIf(s, observableTarget)
}

func observableTarget(o unit.Snapshot) bool {
	return o.Role == unit.RoleFighter || (o.Mortal && o.AimPriority <= unit.DefaultFighterAim)
}

func observationGroup(o unit.Snapshot) uint64 {
	if o.Role != unit.RoleFighter && o.OwnerID != 0 {
		return o.OwnerID
	}
	return o.ID
}

func (o *observer) recordProjectiles(ctx unit.Context, s unit.Sense) {
	for i := range s.Nearby {
		p := &s.Nearby[i]
		if p.Role != unit.RoleProjectile || p.Slot == o.slot {
			continue
		}
		if p.ArcSpan > 1e-6 {
			continue // 近战扇弧不是可复制弹体
		}
		if math.Hypot(p.X-o.x, p.Y-o.y) > recordRadius+p.Radius {
			continue
		}
		if o.projectileSeen[p.ID] {
			continue
		}
		o.projectileSeen[p.ID] = true
		fresh := !o.hasKind(p.Kind)
		if fresh && len(o.kinds) >= kindLimit {
			continue // 第四种不记。这一发已经见过，种类空出来也不补。
		}
		if fresh {
			o.kinds = append(o.kinds, p.Kind)
		}
		o.records = append(o.records, p.Kind)
		o.gauge = math.Min(gaugeMax, o.gauge+projectileGain)
		if !fresh {
			continue
		}
		ctx.Out <- unit.FX{
			Name: "observer-record", Kind: KindObserver, UnitID: ctx.ID, Slot: o.slot,
			X: p.X, Y: p.Y, Amount: kindTone(p.Kind), VX: float64(len(o.kinds)),
		}
	}
}

func (o *observer) hasKind(kind string) bool {
	for _, k := range o.kinds {
		if k == kind {
			return true
		}
	}
	return false
}

func (o *observer) dropReplay() {
	if !o.replaying && len(o.replay) == 0 {
		return
	}
	o.replay = nil
	o.replaying = false
	o.commitFilm()
}

func (o *observer) beginVolley(now float64) {
	o.fired = nil
	o.replay = nil
	if len(o.records) == 0 {
		o.replay = append(o.replay, shotNote{kind: KindObserverBeam, scale: 1})
	} else {
		for _, kind := range o.records {
			o.replay = append(o.replay, shotNote{kind: kind, scale: replayScale})
		}
		o.records = nil
		o.kinds = nil
	}
	otherCount := 0
	for _, prev := range o.film {
		if prev.kind != KindObserverBeam {
			otherCount++
		}
	}
	otherLeft := echoCount(otherCount)
	for _, prev := range o.film {
		if prev.kind != KindObserverBeam {
			if otherLeft == 0 {
				continue
			}
			otherLeft--
		}
		o.replay = append(o.replay, shotNote{kind: prev.kind, scale: prev.scale * replayScale})
	}
	o.film = nil
	o.replaying = true
	o.replayAt = now
}

func echoCount(n int) int {
	if n <= 0 {
		return 0
	}
	return (n*51 + 99) / 100
}

func (o *observer) commitFilm() {
	o.film = append([]shotNote(nil), o.fired...)
	o.fired = nil
}

type shotNote struct {
	kind  string
	scale float64
}

func (o *observer) tickReplay(ctx unit.Context, s unit.Sense, target *unit.Snapshot) {
	if target == nil {
		o.dropReplay()
		return
	}
	if len(o.replay) == 0 {
		o.replaying = false
		return
	}
	if s.Time+1e-9 < o.replayAt {
		return
	}
	shot := o.replay[0]
	o.replay = o.replay[1:]
	if o.spawnReplay(ctx, shot, target) {
		o.fired = append(o.fired, shot)
	}
	o.replayAt = s.Time + replayGap
	if len(o.replay) == 0 {
		o.replaying = false
		o.commitFilm()
	}
}

func (o *observer) spawnReplay(ctx unit.Context, shot shotNote, target *unit.Snapshot) bool {
	if target == nil {
		return false
	}
	kind := shot.kind
	if kind == tanyongchen.KindTanKunaiShot {
		kind = KindObserverKunaiCopy
	}
	spec, ok := unit.Lookup(kind)
	if !ok || spec.Role != unit.RoleProjectile {
		return false
	}
	dx, dy := target.X-o.x, target.Y-o.y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return false
	}
	ux, uy := dx/n, dy/n
	speed := spec.Speed
	if speed < 180 {
		speed = 420
	}
	d := observerRadius + math.Max(6, spec.Radius) + 4
	ctx.Out <- ctx.Born(unit.Spawn{
		Kind: kind, X: o.x + ux*d, Y: o.y + uy*d,
		VX: ux * speed, VY: uy * speed,
		OwnerID: ctx.ID, Slot: o.slot,
		DamageScale: shot.scale,
	})
	fxName := "observer-copy-shot"
	if kind == KindObserverBeam && shot.scale >= 1 {
		fxName = "observer-beam-fire"
	}
	ctx.Out <- unit.FX{
		Name: fxName, Kind: KindObserver, UnitID: ctx.ID, Slot: o.slot,
		X: o.x, Y: o.y, VX: ux, VY: uy,
	}
	return true
}

func (o *observer) emitState(ctx unit.Context, now float64) {
	if now+1e-9 < o.lastStateAt+stateGap {
		return
	}
	o.lastStateAt = now
	ctx.Out <- unit.FX{
		Name: "observer-state", Kind: KindObserver, UnitID: ctx.ID, Slot: o.slot,
		X: o.x, Y: o.y, Amount: o.gauge / gaugeMax * 100, // 观测条百分比
		VX: o.stateMark(), VY: float64(o.targetID),
	}
}

func (o *observer) stateMark() float64 {
	n := len(o.kinds)
	if len(o.film)+len(o.fired) > 0 {
		n += 4
	}
	return float64(n)
}

func kindTone(kind string) float64 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(kind))
	return float64(h.Sum32() % 360)
}

type beam struct {
	owner uint64
	slot  int
	dead  bool
}

type kunaiCopy struct {
	owner uint64
	slot  int
	dead  bool
}

func (k *kunaiCopy) Handle(ctx unit.Context, ev unit.Event) {
	if k.dead {
		return
	}
	switch e := ev.(type) {
	case unit.Collision:
		if e.Other.ID == k.owner {
			return
		}
		if !unit.Hittable(e.Other, k.slot) {
			if e.Other.Role == unit.RoleProjectile {
				k.dead = true
				ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			}
			return
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: kunaiCopyDamage}
		ctx.Out <- unit.FX{Name: "observer-kunai-hit", Kind: KindObserver, UnitID: k.owner, Slot: k.slot, X: e.Other.X, Y: e.Other.Y}
		k.dead = true
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	case unit.WallHit:
		k.dead = true
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

func (b *beam) Handle(ctx unit.Context, ev unit.Event) {
	if b.dead {
		return
	}
	switch e := ev.(type) {
	case unit.Collision:
		if e.Other.ID == b.owner {
			return
		}
		if !unit.Hittable(e.Other, b.slot) {
			if e.Other.Role == unit.RoleProjectile {
				b.vanish(ctx)
			}
			return
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: beamDamage}
		ctx.Out <- unit.FX{Name: "observer-beam-hit", Kind: KindObserver, UnitID: b.owner, Slot: b.slot, X: e.Other.X, Y: e.Other.Y}
		b.vanish(ctx)
	case unit.WallHit:
		b.vanish(ctx)
	}
}

func (b *beam) vanish(ctx unit.Context) {
	b.dead = true
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}
