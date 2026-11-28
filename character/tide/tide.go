package tide

import (
	"embed"
	"encoding/json"
	"math"

	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const KindTide = "潮"

const (
	bodyRadius   = 18.0
	acceleration = 80.0
	widthScale   = 0.25

	liveHitInterval = 0.38
	liveWakeMinGap  = 0.20
	slowFactor      = 0.76
	slowDuration    = 0.65
	pushOutForce    = 35 * 60
	pushFwdForce    = 25 * 60

	wakeMinSpeed    = 80.0
	wakeStep        = 22.0
	wakeTeleportGap = 90.0
	wakeLife        = 1.70
	maxWakeSegs     = 30
	wakeDamage      = 1.3
	wakeHitInterval = 0.48
	wakeSlowDur     = 0.55
	wakeUndertow    = 50.0

	surfMinAge      = 0.55
	surfSegCD       = 1.25
	surfBoostDur    = 1.00
	surfBoostAccel  = 50.0
	surfBoostMaxSpd = 350.0
	surfKickMaxSpd  = 310.0
	surfKickImpulse = 21.0
	surfKickCD      = 0.50
	surfSplashDmg   = 2.0
	surfSplashCD    = 0.45

	ebbMinDrop    = 60.0
	ebbMinPrevSpd = 120.0
	ebbDuration   = 0.50
	ebbCooldown   = 1.15
	ebbMaxLaunch  = 250.0
	ebbMinLaunch  = 134.0

	shieldMinSpeed    = 130.0
	shieldSpeedSpan   = 260.0
	shieldMaxReduce   = 0.26
	shieldMaxShed     = 125.0
	shieldSplashCD    = 0.70
	shieldSplashRad   = 80.0
	shieldSplashForce = 58 * 60

	stateKeyframeGap = 0.25
	slowDomain       = 0x746964655f736c6f
)

func init() {
	p := unit.NewPack(KindTide, assets)
	p.Register(unit.Spec{
		Kind: KindTide, Role: unit.RoleFighter, Fighter: true,
		Radius: bodyRadius, MaxHP: 100, Vision: 9999,
		Acceleration: acceleration, TurnSpeedLoss: 1,
		Look: unit.Look{Color: "#4fbbd3", FX: []string{"tide"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &fighter{
			slowToken: unit.DeriveStream(info.Stream, 0, slowDomain),
			nextHit:   make(map[uint64]float64),
			wakeHitAt: make(map[uint64]float64),
			surfHitAt: make(map[uint64]float64),
			lastEmit:  -1,
		}
	})
}

type fighter struct {
	slowToken uint64
	nextHit   map[uint64]float64
	wakeHitAt map[uint64]float64
	surfHitAt map[uint64]float64

	hasPrev      bool
	lastSpeed    float64
	expectedDrop float64

	pendingShed     float64
	pendingAbsorbed float64
	shedSplashCD    float64

	ebbActive bool
	ebbStart  float64
	ebbUntil  float64
	ebbCD     float64
	ebbDrop   float64
	ebbRadius float64
	ebbPower  float64

	hasLastWake bool
	lastWakeX   float64
	lastWakeY   float64
	nextWakeID  uint64
	wake        []wakeSegment

	surfUntil  float64
	surfKickCD float64

	stateDirty bool
	stateRev   uint64
	lastEmit   float64
	stateJSON  json.RawMessage
	viewBuf    []wakeView
}

type ebbView struct {
	Active bool    `json:"active"`
	Start  float64 `json:"start,omitempty"`
	Until  float64 `json:"until,omitempty"`
	R      float64 `json:"r,omitempty"`
	Power  float64 `json:"power,omitempty"`
}

type wakeView struct {
	ID    uint64  `json:"id"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	FX    float64 `json:"fx"`
	FY    float64 `json:"fy"`
	HL    float64 `json:"hl"`
	NW    float64 `json:"nw"`
	PW    float64 `json:"pw"`
	Born  float64 `json:"born"`
	Life  float64 `json:"life"`
	Flare float64 `json:"flare,omitempty"`
}

type statePayload struct {
	Rev       uint64     `json:"rev"`
	SurfUntil float64    `json:"surfUntil,omitempty"`
	Ebb       ebbView    `json:"ebb"`
	Segs      []wakeView `json:"segs,omitempty"`
}

func (f *fighter) Handle(ctx unit.Context, event unit.Event) {
	switch ev := event.(type) {
	case unit.IncomingDamage:
		f.handleIncoming(ctx, ev)
		return
	case unit.Stunned:
		if f.ebbActive {
			f.ebbActive = false
			f.stateDirty = true
		}
		return
	}
	s, ok := event.(unit.Sense)
	if !ok {
		return
	}

	pruneCooldowns(f.nextHit, s.Time)
	pruneCooldowns(f.wakeHitAt, s.Time)
	pruneCooldowns(f.surfHitAt, s.Time)

	self := s.Self
	speed := math.Hypot(self.VX, self.VY)
	dirX, dirY := 0.0, 0.0
	if speed > 1e-6 {
		dirX, dirY = self.VX/speed, self.VY/speed
	}

	effectiveDrop := 0.0
	if f.hasPrev {
		effectiveDrop = (f.lastSpeed - f.expectedDrop) - speed
	}
	f.expectedDrop = 0

	f.applyMomentumShed(ctx, s, speed, dirX, dirY)
	f.updateEbb(ctx, s, speed, dirX, dirY, effectiveDrop)
	f.expireWake(s.Time)

	var negW, posW float64
	if speed > 1e-6 {
		frame := waterFrame{origin: point{self.X, self.Y}, forward: point{dirX, dirY}}
		w := newWater(self.Radius, speed*widthScale)
		for _, wall := range s.Walls {
			w.clip(frame, wall)
		}
		negW, posW = w.sideWidths()
		f.hitLiveWater(ctx, s, speed, frame, &w)
		f.recordWake(s, speed, negW, posW)
	}

	f.handleSurfAndWake(ctx, s, speed, dirX, dirY)

	f.lastSpeed = speed
	f.hasPrev = true
	f.emitState(ctx, s.Time)
}

func (f *fighter) handleIncoming(ctx unit.Context, d unit.IncomingDamage) {
	if d.Speed <= shieldMinSpeed || d.Amount <= 0 {
		unit.ConfirmHit(ctx, d)
		return
	}
	ratio := clamp((d.Speed-shieldMinSpeed)/shieldSpeedSpan, 0, 1) * shieldMaxReduce
	absorbed := d.Amount * ratio
	taken := d.Amount - absorbed
	if taken <= 0 {
		unit.BlockHit(ctx, d)
	} else {
		ctx.Out <- unit.ConfirmDamage{Token: d.Token, UnitID: ctx.ID, Amount: taken}
	}
	f.pendingShed = math.Min(shieldMaxShed, f.pendingShed+math.Min(d.Speed*0.30, absorbed*16))
	f.pendingAbsorbed += absorbed
}

func (f *fighter) applyMomentumShed(ctx unit.Context, s unit.Sense, speed, dirX, dirY float64) {
	if f.pendingShed <= 0 && f.pendingAbsorbed <= 0 {
		return
	}
	self := s.Self
	if speed > 1e-6 && !s.Knock && !s.MovementLocked && f.pendingShed > 0 {
		shed := math.Min(f.pendingShed, speed*0.40)
		ctx.Out <- unit.Force{UnitID: ctx.ID, AX: -dirX * shed * 60, AY: -dirY * shed * 60}
		f.expectedDrop += shed
	}
	if f.pendingAbsorbed >= 1.0 && s.Time+1e-9 >= f.shedSplashCD {
		f.shedSplashCD = s.Time + shieldSplashCD
		splashDmg := clamp(1.6+f.pendingAbsorbed*0.42, 1.6, 3.3)
		for _, target := range s.Nearby {
			if !unit.Hittable(target, self.Slot) {
				continue
			}
			dx, dy := target.X-self.X, target.Y-self.Y
			dist := math.Hypot(dx, dy)
			if dist > shieldSplashRad+target.Radius {
				continue
			}
			if blockedByWall(self.X, self.Y, target.X, target.Y, s.Walls) {
				continue
			}
			ctx.Out <- unit.Damage{From: ctx.ID, To: target.ID, Amount: splashDmg, NoFreeze: true}
			px, py := 1.0, 0.0
			if dist > 1e-6 {
				px, py = dx/dist, dy/dist
			}
			ctx.Out <- unit.Force{UnitID: target.ID, AX: px * shieldSplashForce, AY: py * shieldSplashForce}
		}
		ctx.Out <- unit.FX{
			Name: "tide-shed", UnitID: ctx.ID, Kind: KindTide,
			X: self.X, Y: self.Y, VX: shieldSplashRad, Amount: f.pendingAbsorbed,
		}
	}
	f.pendingShed = 0
	f.pendingAbsorbed = 0
}

func (f *fighter) updateEbb(ctx unit.Context, s unit.Sense, speed, dirX, dirY, effectiveDrop float64) {
	self := s.Self
	if !s.Knock && !s.MovementLocked && f.hasPrev && effectiveDrop >= ebbMinDrop && f.lastSpeed >= ebbMinPrevSpd && s.Time+1e-9 >= f.ebbCD {
		f.ebbActive = true
		f.ebbStart = s.Time
		f.ebbUntil = s.Time + ebbDuration
		f.ebbCD = s.Time + ebbCooldown
		f.ebbDrop = effectiveDrop
		f.ebbRadius = clamp(bodyRadius+f.lastSpeed*0.36, 78, 156)
		f.ebbPower = clamp(effectiveDrop/220.0, 0.35, 1.0)
		ebbDmg := clamp(1.8+effectiveDrop/95.0, 2.2, 4.4)
		for _, target := range s.Nearby {
			if !unit.Hittable(target, self.Slot) {
				continue
			}
			if math.Hypot(target.X-self.X, target.Y-self.Y) > f.ebbRadius+target.Radius {
				continue
			}
			if blockedByWall(self.X, self.Y, target.X, target.Y, s.Walls) {
				continue
			}
			ctx.Out <- unit.Damage{From: ctx.ID, To: target.ID, Amount: ebbDmg, NoFreeze: true}
			ctx.Out <- unit.AddFSComponent{
				UnitID: target.ID, Zone: unit.FSZoneM, Token: f.slowToken, Value: slowFactor, ExpiresAt: s.Time + slowDuration,
			}
		}
		ctx.Out <- unit.FX{
			Name: "tide-ebb", UnitID: ctx.ID, Kind: KindTide,
			X: self.X, Y: self.Y, VX: f.ebbRadius, VY: f.ebbPower, Amount: effectiveDrop,
		}
		f.stateDirty = true
	}

	if !f.ebbActive {
		return
	}
	if s.Time+1e-9 >= f.ebbUntil {
		f.ebbActive = false
		f.stateDirty = true
		if !s.Knock && !s.MovementLocked && speed > 1e-6 {
			targetSpeed := clamp(124+f.ebbDrop*0.48, ebbMinLaunch, ebbMaxLaunch)
			if speed < targetSpeed {
				dv := targetSpeed - speed
				ctx.Out <- unit.Force{UnitID: ctx.ID, AX: dirX * dv * 60, AY: dirY * dv * 60}
				ctx.Out <- unit.FX{
					Name: "tide-launch", UnitID: ctx.ID, Kind: KindTide,
					X: self.X, Y: self.Y, VX: dirX, VY: dirY, Amount: targetSpeed,
				}
			}
		}
		return
	}

	pullAccel := 240 + 220*f.ebbPower
	for _, target := range s.Nearby {
		if !unit.Hittable(target, self.Slot) {
			continue
		}
		dx, dy := self.X-target.X, self.Y-target.Y
		dist := math.Hypot(dx, dy)
		if dist > f.ebbRadius+target.Radius || dist <= bodyRadius+target.Radius+4 {
			continue
		}
		if blockedByWall(self.X, self.Y, target.X, target.Y, s.Walls) {
			continue
		}
		ctx.Out <- unit.Force{
			UnitID: target.ID,
			AX:     (dx / dist) * pullAccel,
			AY:     (dy / dist) * pullAccel,
		}
	}
}

func (f *fighter) expireWake(now float64) {
	if len(f.wake) == 0 {
		return
	}
	n := 0
	for _, seg := range f.wake {
		if now+1e-9 < seg.bornAt+seg.life {
			f.wake[n] = seg
			n++
		}
	}
	if n != len(f.wake) {
		f.wake = f.wake[:n]
		f.stateDirty = true
	}
}

func (f *fighter) hitLiveWater(ctx unit.Context, s unit.Sense, speed float64, frame waterFrame, w *water) {
	self := s.Self
	dmg := clamp(1.7+speed/175.0, 1.7, 3.6)
	for _, target := range s.Nearby {
		if !unit.Hittable(target, self.Slot) || f.nextHit[target.ID] > s.Time+1e-9 {
			continue
		}
		center := frame.local(target.X, target.Y)
		side := w.hitSide(center, target.Radius)
		if side == 0 {
			continue
		}
		f.nextHit[target.ID] = s.Time + liveHitInterval
		ctx.Out <- unit.Damage{From: ctx.ID, To: target.ID, Amount: dmg, NoFreeze: true}
		ctx.Out <- unit.AddFSComponent{
			UnitID: target.ID, Zone: unit.FSZoneM, Token: f.slowToken, Value: slowFactor, ExpiresAt: s.Time + slowDuration,
		}
		ctx.Out <- unit.Force{
			UnitID: target.ID,
			AX:     -frame.forward.y*side*pushOutForce + frame.forward.x*pushFwdForce,
			AY:     frame.forward.x*side*pushOutForce + frame.forward.y*pushFwdForce,
		}
	}
}

func (f *fighter) recordWake(s unit.Sense, speed, negW, posW float64) {
	self := s.Self
	if !f.hasLastWake {
		f.lastWakeX, f.lastWakeY = self.X, self.Y
		f.hasLastWake = true
		return
	}
	dx, dy := self.X-f.lastWakeX, self.Y-f.lastWakeY
	dist := math.Hypot(dx, dy)
	if dist > wakeTeleportGap {
		f.lastWakeX, f.lastWakeY = self.X, self.Y
		return
	}
	if speed < wakeMinSpeed || dist < wakeStep {
		return
	}
	f.nextWakeID++
	seg := wakeSegment{
		id:       f.nextWakeID,
		x:        0.5 * (self.X + f.lastWakeX),
		y:        0.5 * (self.Y + f.lastWakeY),
		fx:       dx / dist,
		fy:       dy / dist,
		halfLen:  dist * 0.5,
		negW:     negW,
		posW:     posW,
		bornAt:   s.Time,
		life:     wakeLife,
		flaredAt: -1,
		surfedAt: -1,
	}
	f.wake = append(f.wake, seg)
	if len(f.wake) > maxWakeSegs {
		copy(f.wake, f.wake[len(f.wake)-maxWakeSegs:])
		f.wake = f.wake[:maxWakeSegs]
	}
	f.lastWakeX, f.lastWakeY = self.X, self.Y
	f.stateDirty = true
}

func (f *fighter) handleSurfAndWake(ctx unit.Context, s unit.Sense, speed, dirX, dirY float64) {
	if len(f.wake) == 0 {
		return
	}
	self := s.Self
	if speed >= 30 && !s.Knock && !s.MovementLocked {
		for i := range f.wake {
			seg := &f.wake[i]
			if s.Time-seg.bornAt+1e-9 < surfMinAge {
				continue
			}
			if seg.surfedAt >= 0 && s.Time+1e-9 < seg.surfedAt+surfSegCD {
				continue
			}
			hit, _, _ := seg.testCircle(self.X, self.Y, self.Radius, s.Time)
			if !hit {
				continue
			}
			seg.surfedAt = s.Time
			seg.flaredAt = s.Time
			seg.bornAt = math.Min(s.Time, seg.bornAt+0.9)
			f.surfUntil = s.Time + surfBoostDur
			f.stateDirty = true
			if speed < surfKickMaxSpd && s.Time+1e-9 >= f.surfKickCD {
				f.surfKickCD = s.Time + surfKickCD
				ctx.Out <- unit.Force{UnitID: ctx.ID, AX: dirX * surfKickImpulse * 60, AY: dirY * surfKickImpulse * 60}
			}
			ctx.Out <- unit.FX{
				Name: "tide-surf", UnitID: ctx.ID, Kind: KindTide,
				X: seg.x, Y: seg.y, VX: seg.fx, VY: seg.fy, Amount: math.Max(seg.negW, seg.posW),
			}
			for _, target := range s.Nearby {
				if !unit.Hittable(target, self.Slot) || f.surfHitAt[target.ID] > s.Time+1e-9 {
					continue
				}
				if thit, _, _ := seg.testCircle(target.X, target.Y, target.Radius, s.Time); !thit {
					continue
				}
				if blockedByWall(seg.x, seg.y, target.X, target.Y, s.Walls) {
					continue
				}
				f.surfHitAt[target.ID] = s.Time + surfSplashCD
				ctx.Out <- unit.Damage{From: ctx.ID, To: target.ID, Amount: surfSplashDmg, NoFreeze: true}
			}
		}
	}

	if s.Time < f.surfUntil+1e-9 && speed > 1e-6 && speed < surfBoostMaxSpd && !s.Knock && !s.MovementLocked {
		ctx.Out <- unit.Force{UnitID: ctx.ID, AX: dirX * surfBoostAccel, AY: dirY * surfBoostAccel}
	}

	for _, target := range s.Nearby {
		if !unit.Hittable(target, self.Slot) {
			continue
		}
		for i := len(f.wake) - 1; i >= 0; i-- {
			seg := f.wake[i]
			hit, inX, inY := seg.testCircle(target.X, target.Y, target.Radius, s.Time)
			if !hit {
				continue
			}
			if blockedByWall(seg.x, seg.y, target.X, target.Y, s.Walls) {
				continue
			}
			if inX != 0 || inY != 0 {
				ctx.Out <- unit.Force{UnitID: target.ID, AX: inX * wakeUndertow, AY: inY * wakeUndertow}
			}
			liveReady := f.nextHit[target.ID]
			liveNotSameWindow := liveReady <= 0 || (liveReady-liveHitInterval)+liveWakeMinGap <= s.Time+1e-9
			if f.wakeHitAt[target.ID] <= s.Time+1e-9 && liveNotSameWindow {
				f.wakeHitAt[target.ID] = s.Time + wakeHitInterval
				ctx.Out <- unit.Damage{From: ctx.ID, To: target.ID, Amount: wakeDamage, NoFreeze: true}
				ctx.Out <- unit.AddFSComponent{
					UnitID: target.ID, Zone: unit.FSZoneM, Token: f.slowToken, Value: slowFactor, ExpiresAt: s.Time + wakeSlowDur,
				}
			}
			break
		}
	}
}

func (f *fighter) emitState(ctx unit.Context, now float64) {
	needKeyframe := f.lastEmit < 0 || now+1e-9 >= f.lastEmit+stateKeyframeGap
	if !f.stateDirty && !needKeyframe {
		return
	}
	if f.stateDirty || f.stateJSON == nil {
		f.stateRev++
		f.viewBuf = f.viewBuf[:0]
		for _, seg := range f.wake {
			flare := 0.0
			if seg.flaredAt >= 0 {
				flare = seg.flaredAt
			}
			f.viewBuf = append(f.viewBuf, wakeView{
				ID: seg.id, X: round2(seg.x), Y: round2(seg.y),
				FX: round4(seg.fx), FY: round4(seg.fy),
				HL: round2(seg.halfLen), NW: round2(seg.negW), PW: round2(seg.posW),
				Born: round3(seg.bornAt), Life: round2(seg.life), Flare: round3(flare),
			})
		}
		payload := statePayload{
			Rev:       f.stateRev,
			SurfUntil: round3(f.surfUntil),
			Ebb: ebbView{
				Active: f.ebbActive,
				Start:  round3(f.ebbStart),
				Until:  round3(f.ebbUntil),
				R:      round2(f.ebbRadius),
				Power:  round3(f.ebbPower),
			},
			Segs: f.viewBuf,
		}
		if raw, err := json.Marshal(payload); err == nil {
			f.stateJSON = raw
		}
		f.stateDirty = false
	}
	f.lastEmit = now
	ctx.Out <- unit.FX{Name: "tide-state", UnitID: ctx.ID, Kind: KindTide, Data: f.stateJSON}
}

func pruneCooldowns(m map[uint64]float64, now float64) {
	for id, ready := range m {
		if now+1e-9 >= ready {
			delete(m, id)
		}
	}
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
func round3(v float64) float64 { return math.Round(v*1000) / 1000 }
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }
