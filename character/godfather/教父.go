// 教父不转向。球自己不出伤。战斗状态围绕活随从组织。
package 教父

import (
	"embed"
	"math"
	"math/rand/v2"
	"xqdj/internal/unit"
)

const KindGodfather = "教父"
const KindAssassin = "教父暗杀者"
const KindSniper = "教父狙击者"
const KindDealer = "教父贩卖者"
const KindGodfatherShot = "教父狙击弹"
const KindDrug = "教父药物"

const (
	godfatherRadius = 18.0
	godfatherHP     = 100.0
	godfatherCruise = 170.0
	godfatherVision = 0.0
	godfatherColor  = "#1a1410"

	minionRadius = 14.0
	minionCruise = 150.0

	assassinHP     = 15.0
	assassinVision = 140.0
	assassinDash   = 280.0
	assassinSpeed  = 640.0
	assassinDmg    = 7.0
	assassinHits   = 2
	assassinCD     = 1.0
	assassinColor  = "#8b1e18"

	sniperHP     = 14.0
	sniperVision = 9999.0
	sniperAim    = 7.0
	sniperShots  = 1
	sniperColor  = "#8a93a0"

	shotDmg    = 20.0
	shotSpeed  = 900.0
	shotRadius = 5.0
	shotColor  = "#ff6a5a"

	drugHealBoss = 7.0
	drugHealMin  = 10.0
	drugRadius   = 10.0
	drugColor    = "#f4f4f4"

	dealerHP     = 10.0
	dealerDrops  = 2
	dealerGap    = 5.0
	dealerVision = minionRadius + drugRadius
	dealerColor  = "#4a6b3a"

	leavePause = 0.15
	leaveSpeed = 720.0
	leaveFade  = 0.2

	minionCap  = 3
	openFirst  = 1.0
	openGap    = 1.0
	refillWait = 2.5
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindGodfather, assets)
	p.Register(unit.Spec{
		Kind:    KindGodfather,
		Role:    unit.RoleFighter,
		Radius:  godfatherRadius,
		MaxHP:   godfatherHP,
		Speed:   godfatherCruise,
		Vision:  godfatherVision,
		Fighter: true,
		Look:    unit.Look{Color: godfatherColor, FX: []string{"godfather"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return &教父{}
	})
	p.Register(unit.Spec{
		Kind:   KindAssassin,
		Role:   unit.RoleMinion,
		Radius: minionRadius,
		MaxHP:  assassinHP,
		Speed:  minionCruise,
		Vision: assassinVision,
		Mortal: true,
		Look:   unit.Look{Color: assassinColor, Ghost: 280, FX: []string{"minion", "assassin"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &暗杀者{owner: info.OwnerID, slot: info.Slot, attacks: assassinHits}
	})
	p.Register(unit.Spec{
		Kind:   KindSniper,
		Role:   unit.RoleMinion,
		Radius: minionRadius,
		MaxHP:  sniperHP,
		Speed:  minionCruise,
		Vision: sniperVision,
		Mortal: true,
		Look:   unit.Look{Color: sniperColor, FX: []string{"minion", "sniper"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &狙击者{owner: info.OwnerID, slot: info.Slot, shots: sniperShots}
	})
	p.Register(unit.Spec{
		Kind:   KindDealer,
		Role:   unit.RoleMinion,
		Radius: minionRadius,
		MaxHP:  dealerHP,
		Speed:  minionCruise,
		Vision: dealerVision,
		Mortal: true,
		Look:   unit.Look{Color: dealerColor, FX: []string{"minion", "dealer"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &贩卖者{owner: info.OwnerID, slot: info.Slot, left: dealerDrops}
	})
	p.Register(unit.Spec{
		Kind:       KindGodfatherShot,
		Role:       unit.RoleProjectile,
		Radius:     shotRadius,
		MaxHP:      1,
		Speed:      shotSpeed,
		Vision:     0,
		BreakWalls: true,
		Look:       unit.Look{Color: shotColor, Glow: true, Overlay: true, FX: []string{"slug"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &狙击弹{slot: info.Slot, hit: map[uint64]bool{}}
	})
	p.Register(unit.Spec{
		Kind:   KindDrug,
		Role:   unit.RoleHelper,
		Radius: drugRadius,
		MaxHP:  1,
		Speed:  0,
		Vision: 0,
		Look:   unit.Look{Color: drugColor, FX: []string{"drug"}},
	}, func(unit.SpawnInfo) unit.Actor {
		return 药物{}
	})
}

type 教父 struct {
	booted     bool
	opened     int
	lastN      int
	clockUntil float64
	pending    []string
	rng        *rand.Rand
	slot       int
}

func (a *教父) Handle(ctx unit.Context, ev unit.Event) {
	if unit.AcceptHit(ctx, ev) {
		return
	}
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	a.slot = s.Self.Slot
	if !a.booted {
		a.booted = true
		a.rng = rand.New(rand.NewPCG(s.Self.ID, uint64(s.Time*1e6)+1))
	}
	a.tickSpawn(ctx, s)
	a.tryPickup(ctx, s)
}

func (a *教父) tickSpawn(ctx unit.Context, s unit.Sense) {
	n := countMinions(s, ctx.ID)
	live := liveKinds(s, ctx.ID)
	var rest []string
	for _, k := range a.pending {
		if !live[k] {
			rest = append(rest, k)
		}
	}
	a.pending = rest
	for a.opened < minionCap {
		next := openFirst + float64(a.opened)*openGap
		if s.Time+1e-9 < next {
			break
		}
		a.spawnOne(ctx, s)
		a.opened++
	}
	if a.opened >= minionCap {
		if n >= minionCap {
			a.clockUntil = 0
		} else {
			if n < a.lastN && a.clockUntil == 0 {
				a.clockUntil = s.Time + refillWait
			}
			if a.clockUntil > 0 && s.Time+1e-9 >= a.clockUntil {
				a.spawnOne(ctx, s)
				a.clockUntil = 0
				if n+1 < minionCap {
					a.clockUntil = s.Time + refillWait
				}
			}
		}
	}
	a.lastN = n
}

func (a *教父) spawnOne(ctx unit.Context, s unit.Sense) {
	have := liveKinds(s, ctx.ID)
	for _, k := range a.pending {
		have[k] = true
	}
	k := pickKind(have, a.rng)
	if k == "" {
		return
	}
	a.pending = append(a.pending, k)
	x, y, vx, vy := edgeKick(a.rng, minionRadius)
	ctx.Out <- unit.Spawn{
		Kind: k, X: x, Y: y, VX: vx, VY: vy,
		OwnerID: ctx.ID, Slot: s.Self.Slot,
	}
}

func (a *教父) tryPickup(ctx unit.Context, s unit.Sense) {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind != KindDrug || o.Slot != s.Self.Slot {
			continue
		}
		if !overlap(s.Self, *o) {
			continue
		}
		ctx.Out <- unit.Heal{UnitID: ctx.ID, Amount: drugHealBoss}
		ctx.Out <- unit.Despawn{UnitID: o.ID}
	}
}

var minionKinds = []string{KindAssassin, KindSniper, KindDealer}

func countMinions(s unit.Sense, owner uint64) int {
	n := 0
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID == owner && isMinionKind(o.Kind) {
			n++
		}
	}
	return n
}

func liveKinds(s unit.Sense, owner uint64) map[string]bool {
	have := map[string]bool{}
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.OwnerID == owner && isMinionKind(o.Kind) {
			have[o.Kind] = true
		}
	}
	return have
}

func isMinionKind(k string) bool {
	return k == KindAssassin || k == KindSniper || k == KindDealer
}

func pickKind(have map[string]bool, rng *rand.Rand) string {
	var pool []string
	for _, k := range minionKinds {
		if !have[k] {
			pool = append(pool, k)
		}
	}
	if len(pool) == 0 {
		return ""
	}
	if rng == nil {
		return pool[0]
	}
	return pool[rng.IntN(len(pool))]
}

func overlap(a, b unit.Snapshot) bool {
	return math.Hypot(a.X-b.X, a.Y-b.Y) <= a.Radius+b.Radius
}

func enemyFighter(s unit.Sense) *unit.Snapshot {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Role == unit.RoleFighter && o.Slot != s.Self.Slot {
			return o
		}
	}
	return nil
}

func pickupDrug(ctx unit.Context, s unit.Sense, extra func()) bool {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind != KindDrug || o.Slot != s.Self.Slot {
			continue
		}
		if !overlap(s.Self, *o) {
			continue
		}
		ctx.Out <- unit.Heal{UnitID: ctx.ID, Amount: drugHealMin}
		ctx.Out <- unit.Despawn{UnitID: o.ID}
		if extra != nil {
			extra()
		}
		return true
	}
	return false
}
