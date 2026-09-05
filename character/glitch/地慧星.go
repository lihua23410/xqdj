package 地慧星

import (
	"embed"
	"math"
	"math/rand/v2"
	"xqdj/internal/unit"
)

const KindGlitch = "地慧星"
const KindGlitchGhost = "地慧星残影"
const KindGlitchSlash = "地慧星斩击"
const KindGlitchArc = "地慧星弧"
const KindGlitchShot = "地慧星弹"

const (
	glitchRadius = 18.0
	glitchSpeed  = 165.0
	glitchHP     = 100.0
	glitchVision = 9999.0
	glitchDamage = 6.0
	glitchHitCD  = 0.1

	glitchMarkKind    = "剑痕"
	glitchMarkIcon    = "/ball/地慧星/status/jianhen.png"
	glitchGhostOdds   = 0.20
	glitchDodgeCD     = 12.0
	glitchSlashCD     = 20.0                 // 居合冷却：打中
	glitchSlashMissCD = 10.0                 // 居合冷却：没打中
	glitchSlashWindup = 2.0                  // 蓄力时长；到点出刀并结算伤害
	glitchSlashFade   = 1.0                  // 出刀后刀光再收宽的时间（只影响特效）
	glitchSlashLife   = 3.0                  // 刀光单位存活；应 = 蓄力+收宽
	glitchSlashBox    = 5 * 2 * glitchRadius // 判定盒宽度 = 5 倍球直径；长度无限
	glitchWallBoost   = 50.0
	glitchArcInner    = glitchRadius
	glitchArcOuter    = glitchRadius + 2
	glitchArcSpan     = 60.0
	glitchColor       = "#4ec4ff"

	glitchShotRadius  = 12.0
	glitchShotSpeed   = 600.0
	glitchShotDamage  = 9.0
	glitchCagePad     = glitchRadius + 4
	glitchCageCorners = 8
)

//go:embed fx status
var assets embed.FS

func init() {
	p := unit.NewPack(KindGlitch, assets)
	p.Register(unit.Spec{
		Kind:    KindGlitch,
		Role:    unit.RoleFighter,
		Radius:  glitchRadius,
		MaxHP:   glitchHP,
		Speed:   glitchSpeed,
		Vision:  glitchVision,
		Fighter: true,
		Look:    unit.Look{Color: glitchColor, FX: []string{"glitch"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &地慧星{slashReadyAt: glitchSlashCD}
	})
	p.Register(unit.Spec{
		Kind:     KindGlitchArc,
		Role:     unit.RoleProjectile,
		Radius:   glitchArcOuter,
		MaxHP:    1,
		Speed:    glitchSpeed,
		Vision:   0,
		Fighter:  false,
		Attach:   true,
		ArcSpan:  unit.Deg(glitchArcSpan),
		ArcInner: glitchArcInner,
		Look:     unit.Look{Color: glitchColor, Overlay: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &地慧星弧{owner: info.OwnerID, slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:    KindGlitchGhost,
		Role:    unit.RoleHelper,
		Radius:  glitchRadius,
		MaxHP:   1,
		Speed:   0,
		Vision:  glitchVision,
		Fighter: false,
		Look:    unit.Look{Color: glitchColor, FX: []string{"glitch-still"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &地慧星残影{slot: info.Slot}
	})
	p.Register(unit.Spec{
		Kind:    KindGlitchSlash,
		Role:    unit.RoleHelper,
		Radius:  1,
		MaxHP:   1,
		Speed:   0,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: glitchColor, Overlay: true, FX: []string{"slash"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &地慧星斩击{}
	})
	p.Register(unit.Spec{
		Kind:    KindGlitchShot,
		Role:    unit.RoleProjectile,
		Radius:  glitchShotRadius,
		MaxHP:   1,
		Speed:   glitchShotSpeed,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: glitchColor, Trail: true, FX: []string{"crescent"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &地慧星弹{owner: info.OwnerID}
	})
}

type 地慧星 struct {
	arc            unit.AttachState
	dodgeReadyAt   float64
	slashReadyAt   float64
	slashHolding   bool
	slashHoldUntil float64
	slashLockX     float64
	slashLockY     float64
	slashHoldVX    float64
	slashHoldVY    float64
	slashUX        float64
	slashUY        float64
	boostPending   int
	x, y           float64
	vx, vy         float64
	enemyX, enemyY float64
	slot           int
	booted         bool
}

type 地慧星残影 struct {
	slot   int
	inside bool
}

func (g *地慧星残影) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	enemy := ghostOverlap(s, g.slot)
	if enemy != nil && !g.inside {
		ctx.Out <- unit.StackMark{
			UnitID: enemy.ID,
			Kind:   glitchMarkKind,
			Delta:  1,
			Icon:   glitchMarkIcon,
		}
	}
	g.inside = enemy != nil
}

func ghostOverlap(s unit.Sense, slot int) *unit.Snapshot {
	sr := s.Self.Radius
	if sr < 1e-6 {
		sr = glitchRadius
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role != unit.RoleFighter || o.Slot == slot {
			continue
		}
		r := o.Radius
		if r < 1e-6 {
			r = glitchRadius
		}
		if math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) <= sr+r+1e-9 {
			return o
		}
	}
	return nil
}

type 地慧星斩击 struct {
	dieAt  float64
	booted bool
}

func (s *地慧星斩击) Handle(ctx unit.Context, ev unit.Event) {
	sense, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	if !s.booted {
		s.booted = true
		s.dieAt = sense.Time + glitchSlashLife
	}
	if sense.Time+1e-9 >= s.dieAt {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

type 地慧星弹 struct {
	owner uint64
}

func (b *地慧星弹) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Collision:
		if e.Other.ID == b.owner {
			return
		}
		if e.Other.Role != unit.RoleFighter {
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
			return
		}
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: glitchShotDamage}
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	case unit.WallHit:
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

func (g *地慧星) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		g.onIncoming(ctx, e)
	case unit.Sense:
		g.remember(e)
		if unit.RearmAttach(e, ctx.ID, KindGlitchArc, glitchHitCD, &g.arc) {
			unit.SpawnAttach(ctx, e, KindGlitchArc)
		}
		g.applyBoost(ctx, e)
		g.maybeSlash(ctx, e)
	case unit.WallHit:
		g.boostPending++
	}
}

func (g *地慧星) remember(s unit.Sense) {
	g.x, g.y = s.Self.X, s.Self.Y
	g.vx, g.vy = s.Self.VX, s.Self.VY
	g.slot = s.Self.Slot
	g.booted = true
	if e := fighterOf(s); e != nil {
		g.enemyX, g.enemyY = e.X, e.Y
	}
}

func (g *地慧星) onIncoming(ctx unit.Context, d unit.IncomingDamage) {
	if d.Time+1e-9 >= g.dodgeReadyAt {
		unit.BlockHit(ctx, d)
		g.dropGhost(ctx, g.x, g.y)
		g.dodgeReadyAt = d.Time + glitchDodgeCD
		if g.slashHolding {
			return
		}
		nx, ny := farthestCageSpot(g.enemyX, g.enemyY)
		ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: nx, Y: ny}
		g.x, g.y = nx, ny
		g.fireDodgeShot(ctx, nx, ny)
		return
	}
	unit.ConfirmHit(ctx, d)
	g.boostPending = 0
	g.clampCruise(ctx)
}

type 地慧星弧 struct {
	owner  uint64
	slot   int
	x, y   float64
	booted bool
}

func (a *地慧星弧) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Sense:
		a.x, a.y = e.Self.X, e.Self.Y
		a.booted = true
	case unit.Collision:
		if !unit.EnemyFighter(e, a.slot) {
			return
		}
		ctx.Out <- unit.Damage{
			From:      ctx.ID,
			To:        e.Other.ID,
			Amount:    glitchDamage,
			MarkKind:  glitchMarkKind,
			MarkDelta: 1,
			MarkIcon:  glitchMarkIcon,
		}
		if a.booted && rand.Float64() < glitchGhostOdds {
			ctx.Out <- unit.Spawn{
				Kind:    KindGlitchGhost,
				X:       a.x,
				Y:       a.y,
				OwnerID: a.owner,
				Slot:    a.slot,
			}
		}
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
	}
}

func (g *地慧星) maybeSlash(ctx unit.Context, s unit.Sense) {
	if g.slashHolding {
		g.lockSlashPose(ctx)
		if s.Time+1e-9 < g.slashHoldUntil {
			return
		}
		g.slashHolding = false
		g.x, g.y = g.slashLockX, g.slashLockY
		g.vx, g.vy = g.slashHoldVX, g.slashHoldVY
		ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: g.slashLockX, Y: g.slashLockY}
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: g.vx, VY: g.vy}
		g.releaseSlash(ctx, s)
		return
	}
	if s.Time+1e-9 < g.slashReadyAt {
		return
	}
	enemy := fighterOf(s)
	if enemy == nil {
		return
	}
	g.slashLockX, g.slashLockY = s.Self.X, s.Self.Y
	g.slashHoldVX, g.slashHoldVY = s.Self.VX, s.Self.VY
	g.aimSlash(*enemy)
	g.slashHolding = true
	g.slashHoldUntil = s.Time + glitchSlashWindup
	g.lockSlashPose(ctx)
	ctx.Out <- unit.Spawn{
		Kind:    KindGlitchSlash,
		X:       g.slashLockX,
		Y:       g.slashLockY,
		VX:      g.slashUX,
		VY:      g.slashUY,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	}
	ctx.Out <- unit.FX{
		Name:   "iai",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		X:      s.Self.X,
		Y:      s.Self.Y,
		Slot:   s.Self.Slot,
	}
}

func (g *地慧星) aimSlash(enemy unit.Snapshot) {
	dx := enemy.X - g.slashLockX
	dy := enemy.Y - g.slashLockY
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		dx, dy, n = 1, 0, 1
	}
	g.slashUX, g.slashUY = dx/n, dy/n
}

func (g *地慧星) lockSlashPose(ctx unit.Context) {
	g.x, g.y = g.slashLockX, g.slashLockY
	g.vx, g.vy = 0, 0
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: g.slashLockX, Y: g.slashLockY}
}

func (g *地慧星) releaseSlash(ctx unit.Context, s unit.Sense) {
	enemy := fighterOf(s)
	ghosts := 0
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind == KindGlitchGhost && o.OwnerID == ctx.ID {
			ghosts++
		}
	}
	hit := enemy != nil && slashHits(g.slashLockX, g.slashLockY, g.slashUX, g.slashUY, *enemy)
	if hit {
		amt := slashAmount(int(glitchDamage), markStacks(*enemy, glitchMarkKind), ghosts)
		if amt > 0 {
			ctx.Out <- unit.Damage{From: ctx.ID, To: enemy.ID, Amount: amt}
		}
		ctx.Out <- unit.ClearMarks{UnitID: enemy.ID, Kind: glitchMarkKind}
		ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: KindGlitchGhost}
		g.slashReadyAt = s.Time + glitchSlashCD
		return
	}
	g.slashReadyAt = s.Time + glitchSlashMissCD
}

func fighterOf(s unit.Sense) *unit.Snapshot {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role == unit.RoleFighter {
			return o
		}
	}
	return nil
}

func slashHits(x, y, ux, uy float64, enemy unit.Snapshot) bool {
	r := enemy.Radius
	if r < 1e-6 {
		r = glitchRadius
	}
	dist := math.Abs((enemy.X-x)*uy - (enemy.Y-y)*ux)
	return dist <= glitchSlashBox/2+r+1e-9
}

func (g *地慧星) applyBoost(ctx unit.Context, s unit.Sense) {
	if g.slashHolding || g.boostPending <= 0 {
		return
	}
	sp := math.Hypot(s.Self.VX, s.Self.VY)
	if sp < 1e-6 {
		g.boostPending = 0
		return
	}
	ns := sp + float64(g.boostPending)*glitchWallBoost
	g.boostPending = 0
	g.vx, g.vy = s.Self.VX/sp*ns, s.Self.VY/sp*ns
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: g.vx, VY: g.vy}
}

func (g *地慧星) clampCruise(ctx unit.Context) {
	sp := math.Hypot(g.vx, g.vy)
	if sp < 1e-6 {
		return
	}
	g.vx, g.vy = g.vx/sp*glitchSpeed, g.vy/sp*glitchSpeed
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: g.vx, VY: g.vy}
}

func (g *地慧星) dropGhost(ctx unit.Context, x, y float64) {
	ctx.Out <- unit.Spawn{
		Kind:    KindGlitchGhost,
		X:       x,
		Y:       y,
		OwnerID: ctx.ID,
		Slot:    g.slot,
	}
}

func (g *地慧星) fireDodgeShot(ctx unit.Context, x, y float64) {
	dx := g.enemyX - x
	dy := g.enemyY - y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		dx, dy, n = 1, 0, 1
	}
	ux, uy := dx/n, dy/n
	gap := glitchRadius + glitchShotRadius + 1.5
	ctx.Out <- unit.Spawn{
		Kind:    KindGlitchShot,
		X:       x + ux*gap,
		Y:       y + uy*gap,
		VX:      ux * glitchShotSpeed,
		VY:      uy * glitchShotSpeed,
		OwnerID: ctx.ID,
		Slot:    g.slot,
	}
}

func farthestCageSpot(ex, ey float64) (float64, float64) {
	ap := unit.HexRadius * math.Sqrt(3) / 2
	limit := ap - glitchCagePad
	bestX, bestY, bestD := 0.0, 0.0, -1.0
	for i := 0; i < glitchCageCorners; i++ {
		ang := float64(i) * 2 * math.Pi / glitchCageCorners
		ux, uy := math.Cos(ang), math.Sin(ang)
		r := math.Inf(1)
		for j := 0; j < 6; j++ {
			a := (float64(j) + 0.5) * math.Pi / 3
			den := ux*math.Cos(a) + uy*math.Sin(a)
			if den <= 1e-9 {
				continue
			}
			if cand := limit / den; cand < r {
				r = cand
			}
		}
		if math.IsInf(r, 1) {
			continue
		}
		x, y := ux*r, uy*r
		d := (x-ex)*(x-ex) + (y-ey)*(y-ey)
		if d > bestD {
			bestD = d
			bestX, bestY = x, y
		}
	}
	return bestX, bestY
}

func markStacks(u unit.Snapshot, kind string) int {
	for _, m := range u.Marks {
		if m.Kind == kind {
			return m.Stacks
		}
	}
	return 0
}

func slashAmount(base, marks, ghosts int) float64 {
	if marks < 0 {
		marks = 0
	}
	if ghosts < 0 {
		ghosts = 0
	}
	return float64(base+marks*2) * float64(1+ghosts)
}
