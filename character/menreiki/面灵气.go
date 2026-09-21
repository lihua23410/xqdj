package 面灵气

import (
	"embed"
	"math"
	"math/rand/v2"
	"xqdj/internal/unit"
)

//go:embed fx faction status
var assets embed.FS

const KindMenreiki = "面灵气"
const KindMenreikiMask1 = "面灵气面具1"
const KindMenreikiMask2 = "面灵气面具2"
const KindMenreikiMask3 = "面灵气面具3"
const KindMenreikiShot = "面灵气弹"

const (
	menreikiRadius   = 18.0
	menreikiSpeed    = 150.0
	menreikiHP       = 100.0
	menreikiVision   = 9999.0
	menreikiRegen    = 1.0
	menreikiRegenGap = 1.0

	maskDamage    = 3.5
	maskRadius    = 10.0
	maskRadiusRed = 15.0
	maskOrbitPad  = 6.0
	maskSpinT     = 2.0
	maskSpinPaleT = 1.2
	maskHitArc    = 2 * math.Pi / 3
	paleCruise    = 250.0
	paleTurn      = 90 * math.Pi / 180
	paleSteerGap  = 0.15
	redDamageMul  = 1.35

	shotRadius    = 6.0
	shotSpeed     = 300.0
	shotRetarget  = 1.0
	shotRetargetN = 1
	shotArc       = 2 * math.Pi
	shotFirst     = math.Pi / 3

	hook2At = 10.0
	hook3At = 30.0

	ampOut12 = 1.10
	ampIn12  = 0.90
	ampOut3  = 1.5
	ampIn3   = 0.75

	breakStacks = 4
	stunSecs    = 2.0
	shareSecs   = 5.0
	shareMul    = 0.5
	breakKind   = "破甲"
	breakIcon   = "/ball/面灵气/status/break.png"
)

var maskKinds = []string{KindMenreikiMask1, KindMenreikiMask2, KindMenreikiMask3}

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
		return &面灵气{hook: 1}
	})
	for i, kind := range maskKinds {
		i, kind := i, kind
		p.Register(unit.Spec{
			Kind:      kind,
			Role:      unit.RoleHelper,
			Radius:    maskRadius,
			MaxHP:     1,
			Speed:     menreikiSpeed,
			Vision:    0,
			Fighter:   false,
			PassWalls: true,
			Look:      unit.Look{Color: "#f4f0e8", Overlay: true, FX: []string{maskFX(i)}},
		}, func(info unit.SpawnInfo) unit.Actor {
			return &面具{owner: info.OwnerID, slot: info.Slot, index: i}
		})
	}
	p.Register(unit.Spec{
		Kind:    KindMenreikiShot,
		Role:    unit.RoleProjectile,
		Radius:  shotRadius,
		MaxHP:   1,
		Speed:   shotSpeed,
		Vision:  9999,
		Fighter: false,
		Look:    unit.Look{Color: "#3ec8e0", Trail: true},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &面灵气弹{owner: info.OwnerID, slot: info.Slot}
	})
}

func maskFX(i int) string {
	switch i {
	case 1:
		return "mask2"
	case 2:
		return "mask3"
	default:
		return "mask1"
	}
}

type 面灵气 struct {
	hook       int
	marked     bool
	enemyID    uint64
	angle      float64
	lastT      float64
	spun       [3]float64
	shotOnce   [3]bool
	hitSpun    [3]float64
	paid       bool
	regenReady float64
	paleOn     bool
	stunID     uint64
	stunUntil  float64
	pinTok     uint64
	fsSeq      uint32
	frozen     bool
	shareUntil float64
	breakN     int
	prevEHP    float64
	x, y       float64
	vx, vy     float64
	slot       int
	ex, ey     float64
	hasEnemy   bool
	lastFac    string
	steerReady float64
	steerWait  bool
}

func (m *面灵气) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		unit.ConfirmHit(ctx, e)
		m.shareHit(ctx, e)
	case unit.WallHit:
		if m.lastFac == unit.FactionPale {
			m.steerWait = true
		}
	case unit.Collision:
		if m.lastFac == unit.FactionPale && hitTarget(e.Other, m.slot) {
			m.steerWait = true
		}
	case unit.Sense:
		m.tickSense(ctx, e)
	}
}

func (m *面灵气) tickSense(ctx unit.Context, s unit.Sense) {
	m.x, m.y = s.Self.X, s.Self.Y
	m.vx, m.vy = s.Self.VX, s.Self.VY
	m.slot = s.Self.Slot
	if e := enemyOf(s); e != nil {
		m.ex, m.ey = e.X, e.Y
		m.hasEnemy = true
		m.enemyID = e.ID
	}
	if !m.marked {
		ctx.Out <- unit.NoFrameFreeze{UnitID: ctx.ID, Hold: true}
	}
	setLive(ctx.ID, s.Self.Faction)
	m.tickHook(ctx, s)
	m.grant(ctx, s)
	m.watchBreak(ctx, s)
	m.orbit(ctx, s)
	m.factionTick(ctx, s)
	m.collect(ctx, s)
	m.holdStun(ctx, s)
	switched := s.Self.Faction == unit.FactionPale && m.lastFac != unit.FactionPale
	pending := m.steerWait && (m.lastFac == unit.FactionPale || s.Self.Faction == unit.FactionPale)
	if switched || pending {
		m.steerPale(ctx, s.Time, switched)
	}
	m.steerWait = false
	m.lastFac = s.Self.Faction
	if s.Time+1e-9 < m.shareUntil {
		ctx.Out <- unit.FX{
			Name: "share", Kind: ctx.Kind, UnitID: ctx.ID,
			X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot,
		}
	}
}

func (m *面灵气) tickHook(ctx unit.Context, s unit.Sense) {
	want := 1
	if s.Time+1e-9 >= hook3At {
		want = 3
	} else if s.Time+1e-9 >= hook2At {
		want = 2
	}
	if want != m.hook {
		if want == 3 {
			m.paid = false
			if e := enemyOf(s); e != nil {
				ctx.Out <- unit.ClearFactionSeen{UnitID: e.ID}
			}
		}
		m.hook = want
		m.restamp(ctx, s, true)
	}
	ctx.Out <- unit.FX{
		Name: "hook", Kind: ctx.Kind, UnitID: ctx.ID,
		X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot,
		Amount: float64(m.hook),
	}
}

func (m *面灵气) grant(ctx unit.Context, s unit.Sense) {
	if m.marked {
		if e := enemyOf(s); e != nil {
			m.enemyID = e.ID
		}
		return
	}
	selfF := unit.PickFaction(rand.IntN(4))
	ctx.Out <- unit.MarkFaction{
		UnitID:  ctx.ID,
		Faction: selfF,
		AmpOut:  ampOut12,
		AmpIn:   ampIn12,
	}
	m.marked = true
	e := enemyOf(s)
	if e == nil {
		return
	}
	m.enemyID = e.ID
	ctx.Out <- unit.MarkFaction{
		UnitID:  e.ID,
		Faction: unit.PickFaction(rand.IntN(4)),
	}
}

func (m *面灵气) restamp(ctx unit.Context, s unit.Sense, cycle bool) {
	out, in := amps(m.hook)
	selfF := s.Self.Faction
	if selfF == "" {
		selfF = unit.PickFaction(rand.IntN(4))
	}
	ctx.Out <- unit.MarkFaction{
		UnitID:  ctx.ID,
		Faction: selfF,
		Cycle:   cycle,
		AmpOut:  out,
		AmpIn:   in,
	}
	e := enemyOf(s)
	if e == nil {
		return
	}
	ef := e.Faction
	if ef == "" {
		ef = unit.PickFaction(rand.IntN(4))
	}
	ctx.Out <- unit.MarkFaction{
		UnitID:  e.ID,
		Faction: ef,
		Cycle:   cycle,
		Collect: cycle,
	}
}

func amps(hook int) (out, in float64) {
	if hook >= 3 {
		return ampOut3, ampIn3
	}
	return ampOut12, ampIn12
}

func (m *面灵气) orbit(ctx unit.Context, s unit.Sense) {
	dt := 0.0
	if m.lastT > 0 {
		dt = s.Time - m.lastT
	} else {
		armAllMasks(ctx.ID)
	}
	m.lastT = s.Time
	omega := 2 * math.Pi / maskSpinT
	if s.Self.Faction == unit.FactionPale {
		omega = 2 * math.Pi / maskSpinPaleT
	}
	m.angle -= omega * dt
	r := maskRadius
	if s.Self.Faction == unit.FactionRed {
		r = maskRadiusRed
	}
	orbit := s.Self.Radius + r + maskOrbitPad
	have := map[string]*unit.Snapshot{}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID != ctx.ID {
			continue
		}
		have[o.Kind] = o
	}
	var mx, my [3]float64
	for i, kind := range maskKinds {
		ang := m.angle - float64(i)*2*math.Pi/3
		cs, sn := math.Cos(ang), math.Sin(ang)
		x := s.Self.X + cs*orbit
		y := s.Self.Y + sn*orbit
		mx[i], my[i] = x, y
		vx := s.Self.VX + omega*sn*orbit
		vy := s.Self.VY - omega*cs*orbit
		cur := have[kind]
		if dt > 0 {
			m.hitSpun[i] += omega * dt
			for m.hitSpun[i] >= maskHitArc {
				m.hitSpun[i] -= maskHitArc
				armMask(ctx.ID, i)
			}
		}
		var hit []unit.Snapshot
		for j := range s.Nearby {
			o := &s.Nearby[j]
			if !hitTarget(*o, s.Self.Slot) {
				continue
			}
			if math.Hypot(x-o.X, y-o.Y) <= r+o.Radius {
				hit = append(hit, *o)
			}
		}
		if len(hit) > 0 && spendMask(ctx.ID, i) {
			dmg := maskDmg(s.Self.Faction)
			for k := range hit {
				strike(ctx, hit[k], dmg)
			}
		}
		if cur == nil {
			ctx.Out <- unit.Spawn{
				Kind:    kind,
				X:       x,
				Y:       y,
				VX:      vx,
				VY:      vy,
				OwnerID: ctx.ID,
				Slot:    s.Self.Slot,
			}
			continue
		}
		ctx.Out <- unit.Teleport{UnitID: cur.ID, X: x, Y: y}
		ctx.Out <- unit.SetVelocity{UnitID: cur.ID, VX: vx, VY: vy}
		if math.Abs(cur.Radius-r) > 1e-6 {
			ctx.Out <- unit.SetRadius{UnitID: cur.ID, Radius: r}
		}
		if s.Self.Faction == unit.FactionCyan && dt > 0 {
			m.spun[i] += omega * dt
			if !m.shotOnce[i] {
				if m.spun[i] >= shotFirst {
					m.spun[i] = 0
					m.shotOnce[i] = true
					m.fireShot(ctx, s, x, y, cs, sn)
				}
			} else {
				for m.spun[i] >= shotArc {
					m.spun[i] -= shotArc
					m.fireShot(ctx, s, x, y, cs, sn)
				}
			}
		} else if s.Self.Faction != unit.FactionCyan {
			m.spun[i] = 0
			m.shotOnce[i] = false
		}
	}
	if s.Self.Faction == unit.FactionPurple {
		m.eatShots(ctx, s, mx, my, r)
	}
}

func (m *面灵气) eatShots(ctx unit.Context, s unit.Sense, mx, my [3]float64, r float64) {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !shotClearable(*o, ctx.ID) {
			continue
		}
		for k := 0; k < 3; k++ {
			if math.Hypot(o.X-mx[k], o.Y-my[k]) <= o.Radius+r {
				ctx.Out <- unit.Despawn{UnitID: o.ID}
				break
			}
		}
	}
}

func (m *面灵气) fireShot(ctx unit.Context, s unit.Sense, x, y, nx, ny float64) {
	ux, uy := -ny, nx
	if e := enemyOf(s); e != nil {
		dx, dy := e.X-x, e.Y-y
		if n := math.Hypot(dx, dy); n > 1e-6 {
			ux, uy = dx/n, dy/n
		}
	}
	gap := maskRadius + shotRadius + 1.5
	if s.Self.Faction == unit.FactionRed {
		gap = maskRadiusRed + shotRadius + 1.5
	}
	ctx.Out <- unit.Spawn{
		Kind:    KindMenreikiShot,
		X:       x + ux*gap,
		Y:       y + uy*gap,
		VX:      ux * shotSpeed,
		VY:      uy * shotSpeed,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
	}
	ctx.Out <- unit.FX{
		Name: "shot", Kind: ctx.Kind,
		X: x, Y: y, VX: ux * shotSpeed, VY: uy * shotSpeed,
		Slot: s.Self.Slot,
	}
}

func (m *面灵气) factionTick(ctx unit.Context, s unit.Sense) {
	if s.Self.Faction == unit.FactionPurple && s.Time >= m.regenReady {
		ctx.Out <- unit.Heal{UnitID: ctx.ID, Amount: menreikiRegen}
		m.regenReady = s.Time + menreikiRegenGap
	}
	if s.Self.Faction == unit.FactionPale {
		if !m.paleOn {
			ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: paleCruise}
			m.paleOn = true
			m.writeSpeed(ctx, paleCruise)
		}
		return
	}
	if m.paleOn {
		ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: menreikiSpeed}
		m.paleOn = false
		m.writeSpeed(ctx, menreikiSpeed)
	}
}

func (m *面灵气) writeSpeed(ctx unit.Context, speed float64) {
	n := math.Hypot(m.vx, m.vy)
	ux, uy := 1.0, 0.0
	if n > 1e-6 {
		ux, uy = m.vx/n, m.vy/n
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: ux * speed, VY: uy * speed}
	m.vx, m.vy = ux*speed, uy*speed
}

func (m *面灵气) steerPale(ctx unit.Context, t float64, force bool) {
	if !m.hasEnemy {
		return
	}
	if !force && t+1e-9 < m.steerReady {
		return
	}
	sp := math.Hypot(m.vx, m.vy)
	if m.paleOn {
		sp = paleCruise
	}
	if sp < 1e-6 {
		return
	}
	dx, dy := m.ex-m.x, m.ey-m.y
	tn := math.Hypot(dx, dy)
	ux, uy := m.vx/sp, m.vy/sp
	if tn > 1e-6 {
		ux, uy = turnToward(ux, uy, dx/tn, dy/tn, paleTurn)
	}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: ux * sp, VY: uy * sp}
	m.vx, m.vy = ux*sp, uy*sp
	m.steerReady = t + paleSteerGap
}

func turnToward(ux, uy, tx, ty, maxRad float64) (float64, float64) {
	ang := math.Atan2(ux*ty-uy*tx, ux*tx+uy*ty)
	if ang > maxRad {
		ang = maxRad
	} else if ang < -maxRad {
		ang = -maxRad
	}
	c, s := math.Cos(ang), math.Sin(ang)
	return ux*c - uy*s, ux*s + uy*c
}

func (m *面灵气) collect(ctx unit.Context, s unit.Sense) {
	if m.hook < 2 {
		return
	}
	e := enemyOf(s)
	if e == nil {
		return
	}
	if len(e.Seen) < len(unit.AllFactions()) {
		return
	}
	if m.hook < 3 && m.paid {
		return
	}
	m.paid = m.hook < 3
	m.pay(ctx, s, e)
	if m.hook >= 3 {
		ctx.Out <- unit.ClearFactionSeen{UnitID: e.ID}
	}
}

func (m *面灵气) pay(ctx unit.Context, s unit.Sense, e *unit.Snapshot) {
	switch e.Faction {
	case unit.FactionRed:
		strike(ctx, *e, maskDmg(s.Self.Faction))
	case unit.FactionCyan:
		m.stun(ctx, s, e)
	case unit.FactionPale:
		ctx.Out <- unit.StackMark{
			UnitID: e.ID,
			Kind:   breakKind,
			Delta:  breakStacks,
			Icon:   breakIcon,
		}
	case unit.FactionPurple:
		m.shareUntil = s.Time + shareSecs
		m.enemyID = e.ID
		ctx.Out <- unit.FX{
			Name: "share", Kind: ctx.Kind, UnitID: ctx.ID,
			X: s.Self.X, Y: s.Self.Y, Slot: s.Self.Slot,
		}
	}
}

func (m *面灵气) stun(ctx unit.Context, s unit.Sense, e *unit.Snapshot) {
	if m.frozen {
		m.clearPin(ctx)
	}
	until := s.Time + stunSecs
	m.stunID = e.ID
	m.stunUntil = until
	m.pinTok = m.nextFSToken(ctx.ID)
	m.frozen = true
	ctx.Out <- unit.AddFSComponent{
		UnitID: e.ID, Zone: unit.FSZoneM, Token: m.pinTok,
		Value: 0, ExpiresAt: until,
	}
	ctx.Out <- unit.Stun{UnitID: e.ID, Hold: true, Until: until}
	ctx.Out <- unit.FX{
		Name: "stun", Kind: ctx.Kind, UnitID: e.ID,
		X: e.X, Y: e.Y, Slot: s.Self.Slot,
	}
}

func (m *面灵气) holdStun(ctx unit.Context, s unit.Sense) {
	if !m.frozen || m.stunID == 0 {
		return
	}
	alive := false
	var tx, ty float64
	for i := range s.Nearby {
		if s.Nearby[i].ID == m.stunID {
			alive = true
			tx, ty = s.Nearby[i].X, s.Nearby[i].Y
			break
		}
	}
	if !alive {
		m.clearPin(ctx)
		return
	}
	if s.Time+1e-9 >= m.stunUntil {
		m.frozen = false
		m.stunID = 0
		m.pinTok = 0
		return
	}
	ctx.Out <- unit.FX{
		Name: "stun", Kind: ctx.Kind, UnitID: m.stunID,
		X: tx, Y: ty, Slot: s.Self.Slot,
	}
}

func (m *面灵气) clearPin(ctx unit.Context) {
	if m.stunID == 0 {
		m.frozen = false
		return
	}
	if m.pinTok != 0 {
		ctx.Out <- unit.RemoveFSComponent{UnitID: m.stunID, Token: m.pinTok}
		m.pinTok = 0
	}
	ctx.Out <- unit.Stun{UnitID: m.stunID, Hold: false}
	m.frozen = false
	m.stunID = 0
}

func (m *面灵气) nextFSToken(owner uint64) uint64 {
	m.fsSeq++
	return owner<<32 | uint64(m.fsSeq)
}

func maskDmg(faction string) float64 {
	if faction == unit.FactionRed {
		return maskDamage * redDamageMul
	}
	return maskDamage
}

func enemyOf(s unit.Sense) *unit.Snapshot {
	return unit.Seek(s)
}

func hitTarget(other unit.Snapshot, slot int) bool {
	return unit.Hittable(other, slot)
}

func strike(ctx unit.Context, to unit.Snapshot, amount float64) {
	extra := markStacks(to, breakKind)
	ctx.Out <- unit.Damage{From: ctx.ID, To: to.ID, Amount: amount + float64(extra)}
	if extra > 0 {
		ctx.Out <- unit.ClearMarks{UnitID: to.ID, Kind: breakKind}
	}
}

func markStacks(u unit.Snapshot, kind string) int {
	for _, mk := range u.Marks {
		if mk.Kind == kind {
			return mk.Stacks
		}
	}
	return 0
}

func (m *面灵气) shareHit(ctx unit.Context, d unit.IncomingDamage) {
	if m.shareUntil <= 0 || d.Time+1e-9 >= m.shareUntil {
		return
	}
	if m.enemyID == 0 || d.Amount <= 0 || d.From == ctx.ID {
		return
	}
	extra := m.breakN
	ctx.Out <- unit.Damage{From: ctx.ID, To: m.enemyID, Amount: d.Amount*shareMul + float64(extra)}
	if extra > 0 {
		ctx.Out <- unit.ClearMarks{UnitID: m.enemyID, Kind: breakKind}
		m.breakN = 0
	}
}

func (m *面灵气) watchBreak(ctx unit.Context, s unit.Sense) {
	e := enemyOf(s)
	if e == nil {
		return
	}
	n := markStacks(*e, breakKind)
	if n > 0 && m.prevEHP > 0 && e.HP < m.prevEHP-0.5 {
		ctx.Out <- unit.Damage{From: ctx.ID, To: e.ID, Amount: float64(n)}
		ctx.Out <- unit.ClearMarks{UnitID: e.ID, Kind: breakKind}
		n = 0
	}
	m.prevEHP = e.HP
	m.breakN = n
}
