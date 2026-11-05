// 老虎机：每次实际受到伤害后进入 5 秒无敌抽奖，三只转轮决定后续攻击或奖励。
package 老虎机

import (
	cryptorand "crypto/rand"
	"embed"
	"math"
	"math/big"
	"math/rand/v2"

	"xqdj/internal/unit"
)

//go:embed fx
var assets embed.FS

const KindSlot = "老虎机"

const (
	KindSlotCherryShot     = "老虎机樱桃弹"
	KindSlotWatermelonShot = "老虎机西瓜弹"
	KindSlotCrownShot      = "老虎机皇冠弹"
	KindSlotLuckyShot      = "老虎机幸运7弹"
	KindSlotBombShot       = "老虎机炸弹弹"
	KindSlotCoinShot       = "老虎机金币弹"
	KindSlotJackpotShot    = "老虎机超大幸运7"
	KindSlotCherryBall     = "老虎机樱桃球"
)

const (
	symbolCrown = iota
	symbolLucky7
	symbolWatermelon
	symbolCoin
	symbolBomb
	symbolCherry
	symbolCount
)

const (
	slotRadius = 18.0
	slotHP     = 100.0
	slotSpeed  = 150.0
	slotVision = 9999.0
	slotColor  = "#cf2e4b"

	drawDuration = 5.0
	drawStateGap = 0.5

	cherryDamage     = 5.0
	watermelonDamage = 6.0
	crownDamage      = 16.0
	bombDamage       = 7.0
	coinDamage       = 10.0
	jackpotDamage    = 67.0
	luckyMinDamage   = 1.0
	luckyMaxDamage   = 20.0

	luckyStun      = 2.0
	luckySlow      = 0.93
	luckySlowToken = uint64(0x534c4f5400000000)

	bombTimer       = 15.0
	bombBurstDamage = 50.0
	bombMark        = "老虎机炸弹"
	bombMarkIcon    = "/ball/老虎机/fx/炸弹.png"

	bombRadius = 82.0
	shotRadius = 7.0

	cherryBallRadius = 14.0
	cherryBallHP     = 30.0
	cherryBallSpeed  = 105.0
	cherryBallCD     = 0.8

	jackpotCount = 14
)

func init() {
	p := unit.NewPack(KindSlot, assets)
	p.Register(unit.Spec{
		Kind: KindSlot, Role: unit.RoleFighter, Radius: slotRadius, MaxHP: slotHP,
		Speed: slotSpeed, Vision: slotVision, Fighter: true,
		Look: unit.Look{Color: slotColor, FX: []string{"slot"}},
	}, func(unit.SpawnInfo) unit.Actor { return &slotMachine{} })

	p.Register(unit.Spec{
		Kind: KindSlotCherryBall, Role: unit.RoleMinion, Radius: cherryBallRadius,
		MaxHP: cherryBallHP, Speed: cherryBallSpeed, Vision: slotVision,
		Mortal: true, Cruise: true, AimPriority: 42,
		Look: unit.Look{Color: "#e62255", FX: []string{"slot-cherry-ball"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &cherryBall{owner: info.OwnerID, slot: info.Slot, at: -1}
	})

	registerShot := func(kind string, radius float64, color string) {
		p.Register(unit.Spec{
			Kind: kind, Role: unit.RoleProjectile, Radius: radius, MaxHP: 1,
			Speed: 1, Look: unit.Look{Color: color, Trail: true, FX: []string{"slot-bullet"}},
		}, func(info unit.SpawnInfo) unit.Actor {
			return &slotBullet{owner: info.OwnerID, slot: info.Slot, kind: kind}
		})
	}
	registerShot(KindSlotCherryShot, 6, "#e62255")
	registerShot(KindSlotWatermelonShot, 7, "#3fca68")
	registerShot(KindSlotCrownShot, 9, "#ffd33d")
	registerShot(KindSlotLuckyShot, 8, "#ff3d4d")
	registerShot(KindSlotBombShot, 10, "#55505f")
	registerShot(KindSlotCoinShot, 7, "#ffd84a")
	registerShot(KindSlotJackpotShot, 18, "#ff304e")
}

type slotMachine struct {
	slot   int
	booted bool
	rng    *rand.Rand

	drawing     bool
	drawUntil   float64
	nextStateAt float64
	reels       [3]int
	pending     int

	bombHeld     bool
	bombAt       float64
	bombTarget   uint64
	bombTargetAt float64
	nextBombFx   float64

	now    float64
	x, y   float64
	hx, hy float64
	seen   map[uint64]unit.Snapshot
}

func (m *slotMachine) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		if m.drawing {
			unit.BlockHit(ctx, e)
			return
		}
		// 触发抽奖的这一击仍然生效；抽奖无敌从下一次伤害开始。
		unit.ConfirmHit(ctx, e)
		m.beginDraw(ctx, e.Time)
	case unit.Collision:
		m.onCollision(ctx, e)
	case unit.Sense:
		m.onSense(ctx, e)
	}
}

func (m *slotMachine) onCollision(ctx unit.Context, e unit.Collision) {
	if !m.bombHeld || !unit.EnemyTarget(e, m.slot) {
		return
	}
	m.bombHeld = false
	m.bombTarget = e.Other.ID
	m.bombTargetAt = m.now + bombTimer
	ctx.Out <- unit.StackMark{UnitID: e.Other.ID, Kind: bombMark, Delta: 1, Icon: bombMarkIcon}
	ctx.Out <- unit.FX{
		Name: "slot-bomb-transfer", Kind: KindSlot, UnitID: ctx.ID, Slot: m.slot,
		X: e.Other.X, Y: e.Other.Y, Amount: bombTimer,
	}
}

func (m *slotMachine) onSense(ctx unit.Context, s unit.Sense) {
	m.now, m.slot = s.Time, s.Self.Slot
	m.x, m.y = s.Self.X, s.Self.Y
	if speed := math.Hypot(s.Self.VX, s.Self.VY); speed > 1e-6 {
		m.hx, m.hy = s.Self.VX/speed, s.Self.VY/speed
	}
	m.seen = make(map[uint64]unit.Snapshot, len(s.Nearby))
	for i := range s.Nearby {
		m.seen[s.Nearby[i].ID] = s.Nearby[i]
	}
	if !m.booted {
		m.booted = true
		if m.rng == nil {
			m.rng = unit.NewRNG(ctx.ID)
		}
		ctx.Out <- unit.FX{Name: "slot-idle", Kind: KindSlot, UnitID: ctx.ID, Slot: m.slot, X: m.x, Y: m.y}
	}

	if m.drawing {
		if s.Time+1e-9 < m.drawUntil {
			if s.Time+1e-9 >= m.nextStateAt {
				m.nextStateAt = s.Time + drawStateGap
				m.emitDrawState(ctx, false)
			}
		} else {
			m.finishDraw(ctx, s)
		}
	}
	m.tickBomb(ctx, s)
}

func (m *slotMachine) beginDraw(ctx unit.Context, now float64) {
	if m.rng == nil {
		m.rng = unit.NewRNG(ctx.ID)
	}
	m.now = now
	m.drawing = true
	m.drawUntil = now + drawDuration
	m.nextStateAt = now + drawStateGap
	m.roll()
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	m.emitDrawState(ctx, true)
}

func (m *slotMachine) roll() {
	for i := range m.reels {
		m.reels[i] = secureIntN(symbolCount, m.rng)
	}
}

// secureIntN 使用操作系统熵源产生无偏整数；仅在熵源异常时退回 PCG。
func secureIntN(n int, fallback *rand.Rand) int {
	if n <= 1 {
		return 0
	}
	v, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(n)))
	if err == nil {
		return int(v.Int64())
	}
	if fallback != nil {
		return fallback.IntN(n)
	}
	return 0
}

func (m *slotMachine) finishDraw(ctx unit.Context, s unit.Sense) {
	m.resolve(ctx, s)
	ctx.Out <- unit.FX{
		Name: "slot-result", Kind: KindSlot, UnitID: ctx.ID, Slot: m.slot,
		X: m.x, Y: m.y, Amount: encodeReels(m.reels), VX: boolValue(m.triple()),
		VY: float64(m.reels[0]),
	}
	if m.pending > 0 {
		m.pending--
		m.roll()
		m.drawUntil = s.Time + drawDuration
		m.nextStateAt = s.Time + drawStateGap
		m.emitDrawState(ctx, true)
		return
	}
	m.drawing = false
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}
}

func (m *slotMachine) resolve(ctx unit.Context, s unit.Sense) {
	if m.triple() {
		switch m.reels[0] {
		case symbolCherry:
			angle := m.rng.Float64() * 2 * math.Pi
			ctx.Out <- unit.Spawn{
				Kind:    KindSlotCherryBall,
				X:       m.x + math.Cos(angle)*(slotRadius+cherryBallRadius+14),
				Y:       m.y + math.Sin(angle)*(slotRadius+cherryBallRadius+14),
				OwnerID: ctx.ID, Slot: m.slot,
				VX: math.Cos(angle) * cherryBallSpeed,
				VY: math.Sin(angle) * cherryBallSpeed,
			}
		case symbolBomb:
			m.bombHeld = true
			m.bombTarget = 0
			m.bombAt = s.Time + bombTimer
			ctx.Out <- unit.FX{Name: "slot-bomb-hold", Kind: KindSlot, UnitID: ctx.ID, Slot: m.slot, X: m.x, Y: m.y, Amount: bombTimer}
		case symbolWatermelon:
			ctx.Out <- unit.Heal{UnitID: ctx.ID, Amount: 9999}
		case symbolCoin:
			m.pending += 5
			ctx.Out <- unit.FX{Name: "slot-coin-queue", Kind: KindSlot, UnitID: ctx.ID, Slot: m.slot, X: m.x, Y: m.y, Amount: 5}
		case symbolLucky7:
			m.fireJackpot(ctx, s)
		case symbolCrown:
			m.fireCrownJackpot(ctx, s)
		}
		return
	}
	m.fireResultBullets(ctx, s)
}

func (m *slotMachine) fireResultBullets(ctx unit.Context, s unit.Sense) {
	base := m.attackAngle(s)
	for i, symbol := range m.reels {
		angle := base + float64(i-1)*unit.Deg(11)
		m.spawnSymbol(ctx, symbol, angle, 1)
	}
}

func (m *slotMachine) fireCrownJackpot(ctx unit.Context, s unit.Sense) {
	base := m.attackAngle(s)
	for i := 0; i < 5; i++ {
		angle := base + float64(i-2)*unit.Deg(8)
		m.spawnSymbol(ctx, symbolCrown, angle, 1.7)
	}
}

func (m *slotMachine) fireJackpot(ctx unit.Context, s unit.Sense) {
	base := m.rng.Float64() * 2 * math.Pi
	for i := 0; i < jackpotCount; i++ {
		angle := base + 2*math.Pi*float64(i)/jackpotCount + (m.rng.Float64()-0.5)*0.08
		speed := 390 + m.rng.Float64()*140
		m.spawnShot(ctx, KindSlotJackpotShot, angle, speed, 18)
	}
	ctx.Out <- unit.FX{Name: "slot-jackpot", Kind: KindSlot, UnitID: ctx.ID, Slot: m.slot, X: m.x, Y: m.y, Amount: jackpotCount}
}

func (m *slotMachine) spawnSymbol(ctx unit.Context, symbol int, angle, scale float64) {
	kind, speed, radius := symbolShot(symbol)
	if kind == "" {
		return
	}
	m.spawnShot(ctx, kind, angle, speed, radius*scale)
}

func (m *slotMachine) spawnShot(ctx unit.Context, kind string, angle, speed, radius float64) {
	d := slotRadius + radius + 4
	ux, uy := math.Cos(angle), math.Sin(angle)
	ctx.Out <- unit.Spawn{
		Kind: kind, X: m.x + ux*d, Y: m.y + uy*d,
		VX: ux * speed, VY: uy * speed, OwnerID: ctx.ID, Slot: m.slot,
	}
}

func symbolShot(symbol int) (string, float64, float64) {
	switch symbol {
	case symbolCherry:
		return KindSlotCherryShot, 520, 6
	case symbolWatermelon:
		return KindSlotWatermelonShot, 450, 7
	case symbolCrown:
		return KindSlotCrownShot, 410, 9
	case symbolLucky7:
		return KindSlotLuckyShot, 560, 8
	case symbolBomb:
		return KindSlotBombShot, 380, 10
	case symbolCoin:
		return KindSlotCoinShot, 500, 7
	}
	return "", 0, 0
}

func (m *slotMachine) attackAngle(s unit.Sense) float64 {
	if target := unit.Seek(s); target != nil {
		return math.Atan2(target.Y-m.y, target.X-m.x)
	}
	if math.Hypot(m.hx, m.hy) > 1e-6 {
		return math.Atan2(m.hy, m.hx)
	}
	return m.rng.Float64() * 2 * math.Pi
}

func (m *slotMachine) triple() bool {
	return m.reels[0] == m.reels[1] && m.reels[1] == m.reels[2]
}

func (m *slotMachine) emitDrawState(ctx unit.Context, initial bool) {
	name := "slot-state"
	if initial {
		name = "slot-draw"
	}
	remaining := math.Max(0, m.drawUntil-m.now)
	ctx.Out <- unit.FX{
		Name: name, Kind: KindSlot, UnitID: ctx.ID, Slot: m.slot,
		X: m.x, Y: m.y, Amount: encodeReels(m.reels), VX: drawDuration, VY: remaining,
	}
}

func (m *slotMachine) tickBomb(ctx unit.Context, s unit.Sense) {
	if m.bombTarget != 0 {
		target, ok := m.seen[m.bombTarget]
		if !ok {
			ctx.Out <- unit.ClearMarks{UnitID: m.bombTarget, Kind: bombMark}
			m.bombTarget = 0
			return
		}
		if s.Time+1e-9 >= m.bombTargetAt {
			ctx.Out <- unit.Damage{From: ctx.ID, To: target.ID, Amount: bombBurstDamage}
			ctx.Out <- unit.ClearMarks{UnitID: target.ID, Kind: bombMark}
			ctx.Out <- unit.FX{Name: "slot-bomb-boom", Kind: KindSlot, UnitID: ctx.ID, Slot: m.slot, X: target.X, Y: target.Y, Amount: bombRadius}
			m.bombTarget = 0
			return
		}
		if s.Time+1e-9 >= m.nextBombFx {
			m.nextBombFx = s.Time + 0.5
			ctx.Out <- unit.FX{Name: "slot-bomb-hold", Kind: KindSlot, UnitID: ctx.ID, Slot: m.slot, X: target.X, Y: target.Y, Amount: m.bombTargetAt - s.Time}
		}
		return
	}
	if !m.bombHeld {
		return
	}
	if s.Time+1e-9 >= m.bombAt {
		ctx.Out <- unit.Damage{From: ctx.ID, To: ctx.ID, Amount: bombBurstDamage}
		ctx.Out <- unit.FX{Name: "slot-bomb-boom", Kind: KindSlot, UnitID: ctx.ID, Slot: m.slot, X: m.x, Y: m.y, Amount: bombRadius}
		m.bombHeld = false
		return
	}
	if s.Time+1e-9 >= m.nextBombFx {
		m.nextBombFx = s.Time + 0.5
		ctx.Out <- unit.FX{Name: "slot-bomb-hold", Kind: KindSlot, UnitID: ctx.ID, Slot: m.slot, X: m.x, Y: m.y, Amount: m.bombAt - s.Time}
	}
}

func encodeReels(reels [3]int) float64 {
	return float64(reels[0]*100 + reels[1]*10 + reels[2])
}

func boolValue(v bool) float64 {
	if v {
		return 1
	}
	return 0
}
