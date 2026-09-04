package 面灵气

import (
	"embed"
	"math"
	"math/rand/v2"
	"xqdj/internal/unit"
)

//go:embed fx faction
var assets embed.FS

const KindMenreiki = "面灵气"
const KindMenreikiArc = "面灵气弧"

const (
	menreikiRadius    = 18.0
	menreikiSpeed     = 150.0
	menreikiHP        = 75.0
	menreikiVision    = 9999.0
	menreikiRedDamage = 8.0
	menreikiHitCD     = 0.1
	menreikiRegen     = 1.0
	menreikiRegenGap  = 1.0
	menreikiCyanCD    = 1.4
	menreikiPaleSpeed = 250.0
	menreikiAmpOut    = 1.15
	menreikiAmpIn     = 0.85
	menreikiArcInner  = menreikiRadius
	menreikiArcOuter  = menreikiRadius + 2
	menreikiArcSpan   = 150.0
	menreikiArcColor  = "#ff3b3b"
	cyanKind          = "青弹"
	cyanRadius        = 6.0
	cyanSpeed         = 170.0
	cyanDamage        = 5.0
	cyanBounces       = 2
	maskRadius        = 7.0
	maskSpeed         = 300.0
	maskDamage        = 7.0
	maskBounces       = 4
	maskQing          = "面具青"
	maskHong          = "面具红"
	maskZi            = "面具紫"
	maskCang          = "面具苍"
)

var menreikiBarrage = []string{maskQing, maskHong, maskZi, maskCang}

var maskLooks = []struct {
	kind  string
	color string
}{
	{maskQing, "#3ec8e0"},
	{maskHong, "#ff3b3b"},
	{maskZi, "#b44cff"},
	{maskCang, "#8dffb0"},
}

func init() {
	p := unit.NewPack(KindMenreiki, assets)
	p.RegisterFactions([]unit.FactionLook{
		{ID: unit.FactionCyan, File: "faction/qing.png", Color: "#3ec8e0"},
		{ID: unit.FactionRed, File: "faction/hong.png", Color: "#ff3b3b"},
		{ID: unit.FactionPurple, File: "faction/zi.png", Color: "#b44cff"},
		{ID: unit.FactionPale, File: "faction/cang.png", Color: "#8dffb0"},
	})
	p.Register(unit.Spec{
		Kind:    KindMenreiki,
		Role:    unit.RoleFighter,
		Radius:  menreikiRadius,
		MaxHP:   menreikiHP,
		Speed:   menreikiSpeed,
		Vision:  menreikiVision,
		Fighter: true,
		Look:    unit.Look{Color: "hsl(200 92% 60%)", Ghost: 220, FX: []string{"chroma"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &面灵气{}
	})
	p.Register(unit.Spec{
		Kind:     KindMenreikiArc,
		Role:     unit.RoleProjectile,
		Radius:   menreikiArcOuter,
		MaxHP:    1,
		Speed:    menreikiSpeed,
		Vision:   0,
		Fighter:  false,
		Attach:   true,
		ArcSpan:  unit.Deg(menreikiArcSpan),
		ArcInner: menreikiArcInner,
		Look:     unit.Look{Color: menreikiArcColor, Overlay: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &面灵气弧{slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:    cyanKind,
		Role:    unit.RoleProjectile,
		Radius:  cyanRadius,
		MaxHP:   1,
		Speed:   cyanSpeed,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: "#3ec8e0"},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &青弹{owner: info.OwnerID}
	})
	for _, shot := range maskLooks {
		shot := shot
		p.Register(unit.Spec{
			Kind:    shot.kind,
			Role:    unit.RoleProjectile,
			Radius:  maskRadius,
			MaxHP:   1,
			Speed:   maskSpeed,
			Vision:  0,
			Fighter: false,
			Look:    unit.Look{Color: shot.color, Trail: true},
		}, func(info unit.SpawnInfo) unit.Actor {
			return &面具弹{owner: info.OwnerID}
		})
	}
}

type 面灵气 struct {
	selfMarked  bool
	enemyMarked bool
	faction     string
	arc         unit.AttachState
	fireReady   float64
	regenReady  float64
}

func (m *面灵气) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	m.sense(ctx, s)
	if m.faction == unit.FactionRed {
		if unit.RearmAttach(s, ctx.ID, KindMenreikiArc, menreikiHitCD, &m.arc) {
			unit.SpawnAttach(ctx, s, KindMenreikiArc)
		}
	} else if m.arc.Armed || unit.HasOwned(s, ctx.ID, KindMenreikiArc) {
		ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: KindMenreikiArc}
		m.arc.Armed = false
		m.arc.ReadyAt = 0
	}
}

func (m *面灵气) sense(ctx unit.Context, s unit.Sense) {
	m.grant(ctx, s)
	m.faction = s.Self.Faction
	switch m.faction {
	case unit.FactionPurple:
		if s.Time >= m.regenReady {
			ctx.Out <- unit.Heal{UnitID: ctx.ID, Amount: menreikiRegen}
			m.regenReady = s.Time + menreikiRegenGap
		}
	case unit.FactionCyan:
		m.shoot(ctx, s)
	case unit.FactionPale:
		m.speedUp(ctx, s)
	}
}

func (m *面灵气) grant(ctx unit.Context, s unit.Sense) {
	if !m.selfMarked {
		ctx.Out <- unit.MarkFaction{
			UnitID:  ctx.ID,
			Faction: unit.PickFaction(rand.IntN(4)),
			Cycle:   true,
			AmpOut:  menreikiAmpOut,
			AmpIn:   menreikiAmpIn,
			Collect: true,
			Barrage: menreikiBarrage,
		}
		m.selfMarked = true
	}
	if m.enemyMarked {
		return
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role != unit.RoleFighter || o.Slot == s.Self.Slot {
			continue
		}
		ctx.Out <- unit.MarkFaction{
			UnitID:  o.ID,
			Faction: unit.PickFaction(rand.IntN(4)),
			Cycle:   true,
		}
		m.enemyMarked = true
		return
	}
}

type 面灵气弧 struct {
	slot int
}

func (a *面灵气弧) Handle(ctx unit.Context, ev unit.Event) {
	e, ok := ev.(unit.Collision)
	if !ok || !unit.EnemyFighter(e, a.slot) {
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: menreikiRedDamage}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}

func (m *面灵气) shoot(ctx unit.Context, s unit.Sense) {
	if s.Time < m.fireReady {
		return
	}
	var target *unit.Snapshot
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role != unit.RoleFighter || o.Slot == s.Self.Slot {
			continue
		}
		target = o
		break
	}
	if target == nil {
		return
	}
	dx := target.X - s.Self.X
	dy := target.Y - s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	ux, uy := dx/n, dy/n
	gap := s.Self.Radius + cyanRadius + 1.5
	ctx.Out <- unit.Spawn{
		Kind:    cyanKind,
		X:       s.Self.X + ux*gap,
		Y:       s.Self.Y + uy*gap,
		VX:      ux * cyanSpeed,
		VY:      uy * cyanSpeed,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	}
	m.fireReady = s.Time + menreikiCyanCD
	ctx.Out <- unit.FX{
		Name: "shot", Kind: ctx.Kind,
		X: s.Self.X, Y: s.Self.Y,
		VX: ux * cyanSpeed, VY: uy * cyanSpeed,
		Slot: s.Self.Slot,
	}
}

func (m *面灵气) speedUp(ctx unit.Context, s unit.Sense) {
	dx, dy := s.Self.VX, s.Self.VY
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		dx, dy, n = 1, 0, 1
	}
	ctx.Out <- unit.SetVelocity{
		UnitID: ctx.ID,
		VX:     dx / n * menreikiPaleSpeed,
		VY:     dy / n * menreikiPaleSpeed,
	}
}

type 青弹 struct {
	owner   uint64
	bounces int
}

func (b *青弹) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Collision:
		if e.Other.ID == b.owner {
			return
		}
		if e.Other.Role != unit.RoleFighter {
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			return
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: cyanDamage}
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	case unit.WallHit:
		b.bounces++
		if b.bounces >= cyanBounces {
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		}
	}
}

type 面具弹 struct {
	owner   uint64
	bounces int
}

func (b *面具弹) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Collision:
		if e.Other.ID == b.owner {
			return
		}
		if e.Other.Role != unit.RoleFighter {
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			return
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: maskDamage}
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	case unit.WallHit:
		b.bounces++
		if b.bounces >= maskBounces {
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		}
	}
}
