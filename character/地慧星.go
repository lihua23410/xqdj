package character

import (
	"math"
	"math/rand/v2"
	"xqdj/internal/unit"
)

const KindGlitch = "地慧星"
const KindGlitchGhost = "地慧星残影"
const KindGlitchSlash = "地慧星斩击"

const (
	glitchRadius = 18.0
	glitchSpeed  = 165.0
	glitchHP     = 100.0
	glitchVision = 9999.0
	glitchDamage = 6.0
	glitchHitCD  = 0.1

	glitchMarkKind  = "剑痕"
	glitchMarkIcon  = "/status/jianhen.png"
	glitchGhostOdds = 0.10
	glitchDodgeCD   = 12.0
	glitchSlashCD   = 20.0
	glitchSlashLife = 1.25
	glitchWallBoost = 50.0
	glitchHexCircum = 280.0
)

func init() {
	unit.Register(unit.Spec{
		Kind:    KindGlitch,
		Role:    unit.RoleFighter,
		Radius:  glitchRadius,
		MaxHP:   glitchHP,
		Speed:   glitchSpeed,
		Vision:  glitchVision,
		Fighter: true,
		Look:    unit.Look{Color: "#4ec4ff", FX: []string{"glitch"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &地慧星{slashReadyAt: glitchSlashCD}
	})
	unit.Register(unit.Spec{
		Kind:    KindGlitchGhost,
		Role:    unit.RoleHelper,
		Radius:  glitchRadius,
		MaxHP:   1,
		Speed:   0,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: "#4ec4ff", FX: []string{"glitch-still"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return 地慧星残影{}
	})
	unit.Register(unit.Spec{
		Kind:    KindGlitchSlash,
		Role:    unit.RoleHelper,
		Radius:  1,
		MaxHP:   1,
		Speed:   0,
		Vision:  0,
		Fighter: false,
		Look:    unit.Look{Color: "#4ec4ff", FX: []string{"slash"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &地慧星斩击{}
	})
}

type 地慧星 struct {
	hitReadyAt   float64
	dodgeReadyAt float64
	slashReadyAt float64
	boostPending int
	x, y         float64
	vx, vy       float64
	slot         int
	booted       bool
}

type 地慧星残影 struct{}

func (地慧星残影) Handle(unit.Context, unit.Event) {}

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

func (g *地慧星) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		g.onIncoming(ctx, e)
	case unit.Collision:
		g.hit(ctx, e)
	case unit.WallHit:
		g.boostPending++
	case unit.Sense:
		g.remember(e)
		g.applyBoost(ctx, e)
		g.maybeSlash(ctx, e)
	}
}

func (g *地慧星) remember(s unit.Sense) {
	g.x, g.y = s.Self.X, s.Self.Y
	g.vx, g.vy = s.Self.VX, s.Self.VY
	g.slot = s.Self.Slot
	g.booted = true
}

func (g *地慧星) onIncoming(ctx unit.Context, d unit.IncomingDamage) {
	if d.Time+1e-9 >= g.dodgeReadyAt {
		unit.BlockHit(ctx, d)
		g.dropGhost(ctx, g.x, g.y)
		nx, ny := g.randomSpot()
		ctx.Out <- unit.Teleport{UnitID: ctx.ID, X: nx, Y: ny}
		g.x, g.y = nx, ny
		g.dodgeReadyAt = d.Time + glitchDodgeCD
		return
	}
	unit.ConfirmHit(ctx, d)
	g.boostPending = 0
	g.clampCruise(ctx)
}

func (g *地慧星) hit(ctx unit.Context, e unit.Collision) {
	if e.Other.Role != unit.RoleFighter {
		return
	}
	if e.Time < g.hitReadyAt {
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: e.Other.ID, Amount: glitchDamage}
	ctx.Out <- unit.StackMark{
		UnitID: e.Other.ID,
		Kind:   glitchMarkKind,
		Delta:  1,
		Icon:   glitchMarkIcon,
	}
	if rand.Float64() < glitchGhostOdds {
		g.dropGhost(ctx, g.x, g.y)
	}
	g.hitReadyAt = e.Time + glitchHitCD
}

func (g *地慧星) maybeSlash(ctx unit.Context, s unit.Sense) {
	if s.Time+1e-9 < g.slashReadyAt {
		return
	}
	var enemy *unit.Snapshot
	ghosts := 0
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role == unit.RoleFighter {
			enemy = o
		}
		if o.Kind == KindGlitchGhost && o.OwnerID == ctx.ID {
			ghosts++
		}
	}
	if enemy == nil {
		return
	}
	g.slashReadyAt = s.Time + glitchSlashCD
	marks := markStacks(*enemy, glitchMarkKind)
	amt := slashAmount(int(glitchDamage), marks, ghosts)
	if amt > 0 {
		ctx.Out <- unit.Damage{From: ctx.ID, To: enemy.ID, Amount: amt}
	}
	ctx.Out <- unit.ClearMarks{UnitID: enemy.ID, Kind: glitchMarkKind}
	ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: KindGlitchGhost}
	dx := enemy.X - s.Self.X
	dy := enemy.Y - s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		dx, dy, n = 1, 0, 1
	}
	ux, uy := dx/n, dy/n
	ctx.Out <- unit.Spawn{
		Kind:    KindGlitchSlash,
		X:       enemy.X,
		Y:       enemy.Y,
		VX:      ux,
		VY:      uy,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	}
}

func (g *地慧星) applyBoost(ctx unit.Context, s unit.Sense) {
	if g.boostPending <= 0 {
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

func (g *地慧星) randomSpot() (float64, float64) {
	for i := 0; i < 24; i++ {
		ang := rand.Float64() * 2 * math.Pi
		r := rand.Float64() * (glitchHexCircum - glitchRadius - 16)
		x := math.Cos(ang) * r
		y := math.Sin(ang) * r
		if !insideGlitchHex(x, y, glitchRadius+4) {
			continue
		}
		if g.booted && math.Hypot(x-g.x, y-g.y) < 48 {
			continue
		}
		return x, y
	}
	return 0, 0
}

func insideGlitchHex(x, y, radius float64) bool {
	ap := glitchHexCircum * math.Sqrt(3) / 2
	limit := ap - radius
	for i := 0; i < 6; i++ {
		a := (float64(i) + 0.5) * math.Pi / 3
		if math.Cos(a)*x+math.Sin(a)*y > limit+1e-6 {
			return false
		}
	}
	return true
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
