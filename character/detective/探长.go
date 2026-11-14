// 探长开局铺案发现场和指控。踩到的敌方记下疑点；同边撞上敌方记下调查。
// 二十秒后一审。有罪就进入审判开枪，否则再掷一次，进我执或伪证。
package 探长

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

const KindDetective = "探长"

const (
	KindScene   = "探长案发现场"
	KindCharge  = "探长指控"
	KindForgery = "探长伪证"
	KindBullet  = "探长弹"
)

const (
	bodyRadius = 18.0
	bodyHP     = 100.0
	cruise     = 160.0
	vision     = 9999.0
	bodyColor  = "#c4893c"

	sceneRadius   = 42.0
	chargeRadius  = 13.0
	forgeryRadius = 13.0
	bulletRadius  = 4.0

	bulletSpeed     = 1024.0
	judgmentDamage  = 11.0 // 正义 XI
	obsessionDamage = 20.0 // 审判 XX
	magazine        = 7
	shotGap         = 0.4
	reloadGap       = 5.0

	trialAt     = 20.0
	trialLen    = 3.0
	forgeryWait = 20.0

	investigateNeed = 4
	touchSlack      = 1.0

	placeGap = 36.0

	markSelf       = "探长"
	markAssistant  = "助手"
	markSuspicion  = "疑点"
	markInquiry    = "调查"
	markSuspect    = "嫌疑人"
	markAccomplice = "共犯"
	markGuilty     = "有罪"
	markAdmonition = "申诫"

	iconSuspicion  = "/ball/探长/status/suspicion.svg"
	iconInquiry    = "/ball/探长/status/inquiry.svg"
	iconSuspect    = "/ball/探长/status/suspect.svg"
	iconAccomplice = "/ball/探长/status/accomplice.svg"
	iconGuilty     = "/ball/探长/status/guilty.svg"
	iconAdmonition = "/ball/探长/status/admonition.svg"

	immunePool = 1e9
)

//go:embed fx status
var assets embed.FS

func init() {
	p := unit.NewPack(KindDetective, assets)
	p.Register(unit.Spec{
		Kind:    KindDetective,
		Role:    unit.RoleFighter,
		Radius:  bodyRadius,
		MaxHP:   bodyHP,
		Speed:   cruise,
		Vision:  vision,
		Fighter: true,
		Look:    unit.Look{Color: bodyColor, FX: []string{"detective"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &探长{}
	})
	registerMarker(p, KindScene, sceneRadius, "#efe6d4", "detective-scene", modeScene)
	registerMarker(p, KindCharge, chargeRadius, "#f4f0e4", "detective-charge", modeCharge)
	registerMarker(p, KindForgery, forgeryRadius, "#9a9286", "detective-forgery", modeForgery)
	p.Register(unit.Spec{
		Kind:    KindBullet,
		Role:    unit.RoleProjectile,
		Radius:  bulletRadius,
		MaxHP:   1,
		Speed:   bulletSpeed,
		Fighter: false,
		Look:    unit.Look{Color: "#d5dde2", FX: []string{"detective-slug"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		dmg := judgmentDamage
		if n, ok := info.Data.(float64); ok && n > 0 {
			dmg = n
		}
		return &子弹{owner: info.OwnerID, slot: info.Slot, damage: dmg}
	})
}

// odds 以 7 层 50% 为基准：10% × 5^(x/7)。0 层是 10%，7 层正好 50%，超过 1 就按必定。
// 有罪、分支和申诫共用这一条。
func odds(x float64) float64 {
	return clampOdds(0.10 * math.Pow(5, x/7))
}

func clampOdds(p float64) float64 {
	if p > 1 {
		return 1
	}
	if p < 0 {
		return 0
	}
	return p
}

// 场心那一行审判结果。只有有罪和无罪。
const (
	verdictClear float64 = iota
	verdictGuilty
)

type phase int

const (
	phaseInvestigate phase = iota
	phaseTrial
	phaseJudgment
	phaseObsession
	phaseForgery
)

type 探长 struct {
	ready   bool
	born    unit.Context
	rng     *rand.Rand
	roll    func() float64
	slot    int
	boot    float64
	enemies int

	phase         phase
	firstTrial    bool
	trialUntil    float64
	trialSuspect  uint64
	trialFirst    bool
	immune        bool
	condemn       bool
	secondTrialAt float64
	bareRetrial   bool // 无罪之后的二审，不铺伪证；再无罪则自尽

	shotsLeft int
	nextShot  float64

	touching    map[contact]struct{}
	lastSuspect uint64
	absorbSaved map[uint64]float64
	standing    bool
	suspectHeld uint64
}

type contact struct{ a, b uint64 }

func (a *探长) Handle(ctx unit.Context, ev unit.Event) {
	if d, ok := ev.(unit.IncomingDamage); ok {
		if a.immune && !(a.condemn && d.From == ctx.ID) {
			unit.BlockHit(ctx, d)
			return
		}
		a.condemn = false
		unit.ConfirmHit(ctx, d)
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		if c, hit := ev.(unit.Collision); hit {
			a.onBump(c)
		}
		return
	}
	a.onSense(ctx, s)
}

func (a *探长) onSense(ctx unit.Context, s unit.Sense) {
	if !a.ready {
		a.ready = true
		a.born = ctx.WithBirths()
		a.rng = unit.NewRNG(ctx.Stream)
		a.slot = s.Self.Slot
		a.boot = s.Time
		a.enemies = enemySeats(s, a.slot)
		a.placeOpening(s)
		a.syncRole(ctx, s)
	}
	a.born.Out = ctx.Out
	a.collectContacts(s)
	a.rollAdmonitions(ctx)
	a.syncMarks(ctx, s)

	if a.phase == phaseTrial {
		a.holdTrial(ctx, s)
		if s.Time+1e-9 < a.trialUntil {
			return
		}
		a.finishTrial(ctx, s)
		return
	}
	if a.secondTrialAt > 0 && s.Time+1e-9 >= a.secondTrialAt && a.phase != phaseTrial {
		a.secondTrialAt = 0
		a.beginTrial(ctx, s, false)
		return
	}
	if !a.firstTrial && s.Time+1e-9 >= a.boot+trialAt && a.phase != phaseTrial {
		a.firstTrial = true
		a.beginTrial(ctx, s, true)
		return
	}
	if a.phase != phaseJudgment && bookOf(ctx.State, ctx.ID).anyGuilty() {
		a.enterJudgment(s.Time)
	}
	if a.phase == phaseJudgment || a.phase == phaseObsession {
		a.shoot(ctx, s)
	}
}

func (a *探长) placeOpening(s unit.Sense) {
	n := a.enemies*4 + 1
	var taken [][2]float64
	a.drop(s, KindScene, sceneRadius, 1, &taken)
	a.drop(s, KindCharge, chargeRadius, n, &taken)
}

func (a *探长) drop(s unit.Sense, kind string, radius float64, n int, taken *[][2]float64) {
	for i := 0; i < n; i++ {
		x, y, ok := a.spot(s.Field, radius, *taken)
		if !ok {
			continue
		}
		*taken = append(*taken, [2]float64{x, y})
		a.born.Out <- a.born.Born(unit.Spawn{
			Kind:    kind,
			X:       x,
			Y:       y,
			OwnerID: a.born.ID,
			Slot:    a.slot,
		})
	}
}

func (a *探长) spot(field unit.Field, radius float64, taken [][2]float64) (float64, float64, bool) {
	for try := 0; try < 80; try++ {
		x, y, ok := field.RandomWalkable(a.rng, radius)
		if !ok {
			continue
		}
		if crowded(x, y, taken, placeGap) {
			continue
		}
		return x, y, true
	}
	return 0, 0, false
}

func crowded(x, y float64, taken [][2]float64, gap float64) bool {
	for _, p := range taken {
		dx, dy := x-p[0], y-p[1]
		if dx*dx+dy*dy < gap*gap {
			return true
		}
	}
	return false
}

func enemySeats(s unit.Sense, slot int) int {
	seen := map[int]struct{}{}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Slot == slot || o.Role != unit.RoleFighter {
			continue
		}
		seen[o.Seat] = struct{}{}
	}
	return len(seen)
}

func (a *探长) onBump(c unit.Collision) {
	if !a.ready {
		return
	}
	foe := c.Other
	if foe.Slot == a.slot || !body(&foe) {
		return
	}
	a.noteTouch(a.born.ID, foe.ID, c.Time)
}

func (a *探长) collectContacts(s unit.Sense) {
	allies, foes := sides(s, a.born.ID, a.slot)
	if a.touching == nil {
		a.touching = map[contact]struct{}{}
	}
	now := map[contact]struct{}{}
	for _, ally := range allies {
		for i := range foes {
			foe := &foes[i]
			// 助手的碰撞事件到不了探长。弹开的这一拍用双方速率把分开的距离补回来。
			limit := ally.Radius + foe.Radius + touchSlack
			if ally.ID != a.born.ID {
				limit += (math.Hypot(ally.VX, ally.VY) + math.Hypot(foe.VX, foe.VY)) / 60
			}
			if math.Hypot(ally.X-foe.X, ally.Y-foe.Y) > limit {
				continue
			}
			key := contact{ally.ID, foe.ID}
			now[key] = struct{}{}
			if _, was := a.touching[key]; was {
				continue
			}
			a.noteTouch(ally.ID, foe.ID, s.Time)
		}
	}
	a.touching = now
}

func (a *探长) noteTouch(ally, foe uint64, now float64) {
	if a.touching == nil {
		a.touching = map[contact]struct{}{}
	}
	key := contact{ally, foe}
	if _, was := a.touching[key]; was {
		return
	}
	a.touching[key] = struct{}{}
	_, just := bookOf(a.born.State, a.born.ID).addInvestigation(foe)
	if just && a.phase != phaseTrial && a.phase != phaseJudgment {
		a.enterJudgment(now)
	}
}

func sides(s unit.Sense, self uint64, slot int) (allies []unit.Snapshot, foes []unit.Snapshot) {
	allies = append(allies, s.Self)
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if !body(o) {
			continue
		}
		if o.Slot == slot {
			if o.ID != self {
				allies = append(allies, *o)
			}
			continue
		}
		foes = append(foes, *o)
	}
	return allies, foes
}

func body(o *unit.Snapshot) bool {
	if o.Nonsolid {
		return false
	}
	return o.Role == unit.RoleFighter || o.Mortal
}

func (a *探长) beginTrial(ctx unit.Context, s unit.Sense, first bool) {
	a.phase = phaseTrial
	a.trialFirst = first
	a.trialUntil = s.Time + trialLen
	a.immune = true
	a.trialSuspect = bookOf(ctx.State, ctx.ID).snapshot().suspect
	a.holdTrial(ctx, s)
	if a.trialSuspect != 0 {
		a.holdUnit(ctx, a.trialSuspect, true)
		ctx.Out <- unit.Stun{UnitID: a.trialSuspect, Hold: true, Until: a.trialUntil}
		a.suspectHeld = a.trialSuspect
	}
	ctx.Out <- unit.FX{
		Name: "judge", Kind: ctx.Kind,
		X: 0, Y: 0, Slot: s.Self.Slot, Amount: trialLen,
	}
}

func (a *探长) holdTrial(ctx unit.Context, s unit.Sense) {
	a.holdSelf(ctx, true)
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID}
	if a.suspectHeld != 0 {
		ctx.Out <- unit.SetVelocity{UnitID: a.suspectHeld}
	}
	if a.absorbSaved == nil {
		a.absorbSaved = map[uint64]float64{}
	}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Slot == a.slot || !body(o) {
			continue
		}
		if _, ok := a.absorbSaved[o.ID]; ok {
			continue
		}
		a.absorbSaved[o.ID] = o.Absorb
		ctx.Out <- unit.SetAbsorb{UnitID: o.ID, Amount: immunePool}
	}
}

func (a *探长) finishTrial(ctx unit.Context, s unit.Sense) {
	a.immune = false
	a.holdSelf(ctx, false)
	if a.suspectHeld != 0 {
		a.holdUnit(ctx, a.suspectHeld, false)
		a.suspectHeld = 0
	}
	for id, prev := range a.absorbSaved {
		ctx.Out <- unit.SetAbsorb{UnitID: id, Amount: prev}
	}
	a.absorbSaved = nil

	first := a.trialFirst
	suspect := a.trialSuspect
	sus, inquiries, already := 0, 0, false
	if suspect != 0 {
		sus, inquiries, already = bookOf(ctx.State, ctx.ID).suspectOf(suspect)
		if !already && a.chance(odds(float64(sus))) {
			bookOf(ctx.State, ctx.ID).markGuilty(suspect)
			bookOf(ctx.State, ctx.ID).implicate()
		}
	}
	a.syncMarks(ctx, s)
	bare := !first && a.bareRetrial
	if !first {
		a.bareRetrial = false
	}
	if bookOf(ctx.State, ctx.ID).anyGuilty() {
		a.shoutVerdict(ctx, verdictGuilty)
		a.enterJudgment(s.Time)
		a.shoot(ctx, s)
		return
	}
	a.phase = phaseInvestigate
	if !first || suspect == 0 || !a.chance(odds(float64(sus+inquiries))) {
		a.shoutVerdict(ctx, verdictClear)
		if bare {
			a.condemn = true
			ctx.Out <- unit.Damage{From: ctx.ID, To: ctx.ID, Amount: bodyHP * 10, NoFreeze: true}
			return
		}
		if first {
			a.secondTrialAt = s.Time + forgeryWait
			a.bareRetrial = true
		}
		return
	}
	if a.chance(0.5) {
		a.shoutVerdict(ctx, verdictClear)
		a.enterObsession(s)
		a.shoot(ctx, s)
		return
	}
	a.shoutVerdict(ctx, verdictClear)
	a.phase = phaseForgery
	var taken [][2]float64
	a.drop(s, KindForgery, forgeryRadius, a.enemies*4+1, &taken)
	a.secondTrialAt = s.Time + forgeryWait
}

func (a *探长) shoutVerdict(ctx unit.Context, result float64) {
	ctx.Out <- unit.FX{
		Name: "verdict", Kind: ctx.Kind,
		X: 0, Y: 0, Slot: a.slot, Amount: result,
	}
}

func (a *探长) enterJudgment(now float64) {
	if a.phase == phaseJudgment {
		return
	}
	a.phase = phaseJudgment
	a.shotsLeft = magazine
	a.nextShot = now
}

func (a *探长) enterObsession(s unit.Sense) {
	a.phase = phaseObsession
	a.shotsLeft = magazine
	a.nextShot = s.Time
}

func (a *探长) shoot(ctx unit.Context, s unit.Sense) {
	if a.shotsLeft <= 0 || s.Time+1e-9 < a.nextShot {
		return
	}
	target := a.aim(s)
	if target == nil {
		return
	}
	dx, dy := target.X-s.Self.X, target.Y-s.Self.Y
	n := math.Hypot(dx, dy)
	if n < 1e-6 {
		return
	}
	ux, uy := dx/n, dy/n
	gap := s.Self.Radius + bulletRadius + 1.5
	dmg := judgmentDamage
	if a.phase == phaseObsession {
		dmg = obsessionDamage
	}
	ctx.Out <- a.born.Born(unit.Spawn{
		Kind:    KindBullet,
		X:       s.Self.X + ux*gap,
		Y:       s.Self.Y + uy*gap,
		VX:      ux * bulletSpeed,
		VY:      uy * bulletSpeed,
		OwnerID: ctx.ID,
		Slot:    s.Self.Slot,
		Data:    dmg,
	})
	ctx.Out <- unit.FX{
		Name: "detective-shot", Kind: ctx.Kind, UnitID: ctx.ID,
		X: s.Self.X, Y: s.Self.Y, VX: ux, VY: uy, Slot: s.Self.Slot,
	}
	if a.phase == phaseObsession {
		bookOf(ctx.State, ctx.ID).noteAdmonition()
		a.rollAdmonitions(ctx)
	}
	a.shotsLeft--
	if a.shotsLeft > 0 {
		a.nextShot = s.Time + shotGap
		return
	}
	if a.phase == phaseJudgment {
		a.shotsLeft = magazine
		a.nextShot = s.Time + reloadGap
		return
	}
	a.nextShot = math.Inf(1)
}

func (a *探长) aim(s unit.Sense) *unit.Snapshot {
	b := bookOf(a.born.State, a.born.ID).snapshot()
	var best *unit.Snapshot
	bestD := math.MaxFloat64
	consider := func(o *unit.Snapshot) {
		if o == nil || o.ID == s.Self.ID || !body(o) || o.Slot == a.slot {
			return
		}
		d := math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y)
		if d < bestD {
			best = o
			bestD = d
		}
	}
	if a.phase == phaseJudgment {
		for i := range s.Nearby {
			o := &s.Nearby[i]
			if !b.guilty[o.ID] {
				continue
			}
			if o.ID == b.suspect {
				return o
			}
			consider(o)
		}
		if b.guilty[a.trialSuspect] {
			for i := range s.Nearby {
				if s.Nearby[i].ID == a.trialSuspect {
					return &s.Nearby[i]
				}
			}
		}
		return best
	}
	for i := range s.Nearby {
		if s.Nearby[i].ID == b.suspect {
			o := &s.Nearby[i]
			if body(o) && o.Slot != a.slot {
				return o
			}
		}
	}
	for i := range s.Nearby {
		consider(&s.Nearby[i])
	}
	return best
}

func (a *探长) rollAdmonitions(ctx unit.Context) {
	n, stacks := bookOf(ctx.State, ctx.ID).takePending()
	if n <= 0 {
		return
	}
	// 申诫一层层加上去。X 用加上这一层之后的层数。
	start := stacks - n
	for i := 1; i <= n; i++ {
		if a.chance(odds(float64(start + i))) {
			a.condemn = true
			ctx.Out <- unit.Damage{From: ctx.ID, To: ctx.ID, Amount: bodyHP * 10, NoFreeze: true}
			return
		}
	}
}

func (a *探长) chance(p float64) bool {
	if p <= 0 {
		return false
	}
	if p >= 1 {
		return true
	}
	if a.roll != nil {
		return a.roll() < p
	}
	return a.rng.Float64() < p
}

func (a *探长) holdSelf(ctx unit.Context, hold bool) {
	if a.standing == hold {
		if hold {
			ctx.Out <- unit.SetVelocity{UnitID: ctx.ID}
		}
		return
	}
	a.standing = hold
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: hold}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID}
}

func (a *探长) holdUnit(ctx unit.Context, id uint64, hold bool) {
	ctx.Out <- unit.Stand{UnitID: id, Hold: hold}
	ctx.Out <- unit.SetVelocity{UnitID: id}
}

func (a *探长) syncRole(ctx unit.Context, s unit.Sense) {
	ctx.Out <- unit.StackMark{UnitID: ctx.ID, Kind: markSelf, Delta: 1, Hidden: true}
}

func (a *探长) syncMarks(ctx unit.Context, s unit.Sense) {
	view := bookOf(ctx.State, ctx.ID).snapshot()
	a.paint(ctx, s.Self, view)
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Slot == a.slot && o.ID != ctx.ID && body(o) {
			if stacks(o, markAssistant) == 0 {
				ctx.Out <- unit.StackMark{UnitID: o.ID, Kind: markAssistant, Delta: 1, Hidden: true}
			}
		}
		a.paint(ctx, *o, view)
	}
	if view.suspect == a.lastSuspect {
		return
	}
	if a.lastSuspect != 0 {
		ctx.Out <- unit.ClearMarks{UnitID: a.lastSuspect, Kind: markSuspect}
	}
	a.lastSuspect = view.suspect
	if view.suspect != 0 {
		ctx.Out <- unit.StackMark{UnitID: view.suspect, Kind: markSuspect, Delta: 1, Icon: iconSuspect}
	}
}

func (a *探长) paint(ctx unit.Context, o unit.Snapshot, view 账) {
	setStacks(ctx, o, markSuspicion, view.suspicion[o.ID], iconSuspicion)
	setStacks(ctx, o, markInquiry, view.investigation[o.ID], iconInquiry)
	wantAccomplice := 0
	if o.Slot != a.slot && o.ID != view.suspect && view.suspicion[o.ID] > 0 {
		wantAccomplice = 1
	}
	setStacks(ctx, o, markAccomplice, wantAccomplice, iconAccomplice)
	wantGuilty := 0
	if view.guilty[o.ID] {
		wantGuilty = 1
	}
	setStacks(ctx, o, markGuilty, wantGuilty, iconGuilty)
	if o.ID == ctx.ID {
		setStacks(ctx, o, markAdmonition, view.admonition, iconAdmonition)
	}
}

func setStacks(ctx unit.Context, o unit.Snapshot, kind string, want int, icon string) {
	got := stacks(&o, kind)
	if want == got {
		return
	}
	ctx.Out <- unit.StackMark{UnitID: o.ID, Kind: kind, Delta: want - got, Icon: icon}
}

func stacks(o *unit.Snapshot, kind string) int {
	for _, m := range o.Marks {
		if m.Kind == kind {
			return m.Stacks
		}
	}
	return 0
}
