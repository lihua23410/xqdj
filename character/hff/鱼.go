// 大肥鱼(幻方量化) 只在真实场地上巡航。到点盯盘，抽一只操盘股票，
// 把整场抄进沙盘；交割时按看涨或看空写回差额，对手付的两倍，她自己付的一倍。
package hff

import (
	"embed"
	"math"
	"math/rand/v2"

	"xqdj/internal/unit"
)

const KindHFF = "大肥鱼(幻方量化)"

const (
	bodyRadius = 18.0
	bodyHP     = 100.0
	bodyCruise = 130.0
	bodyVision = 0.0
	bodyColor  = "#243456"
	cruiseSec  = 3.0
	eatSec     = 1.5
	mealHeal   = 8.0 // 与大肥鱼吃白饭当笔结算相同
	kindRice   = "大肥鱼白饭"
	kindOrca   = "大肥鱼"
)

//go:embed fx
var assets embed.FS

func init() {
	p := unit.NewPack(KindHFF, assets)
	p.Register(unit.Spec{
		Kind:    KindHFF,
		Role:    unit.RoleFighter,
		Radius:  bodyRadius,
		MaxHP:   bodyHP,
		Speed:   bodyCruise,
		Vision:  bodyVision,
		Fighter: true,
		Look:    unit.Look{Color: bodyColor, FX: []string{"hff"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &鱼{rng: unit.NewRNG(info.Stream)}
	})
}

type 鱼 struct {
	rng         *rand.Rand
	started     bool
	cruiseUntil float64
	armed       bool
	chewing     bool
	mealID      uint64
	mealUntil   float64
}

func (a *鱼) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		unit.ConfirmHit(ctx, e)
	case unit.PaperDone:
		a.cruiseUntil = e.Time + cruiseSec
		a.armed = false
		a.chewing = false
	case unit.Stare:
		a.stare(ctx, e)
	case unit.Sense:
		a.sense(ctx, e)
	}
}

func (a *鱼) sense(ctx unit.Context, s unit.Sense) {
	if a.rng == nil {
		a.rng = unit.NewRNG(ctx.Stream)
	}
	if !a.started {
		a.started = true
		a.cruiseUntil = s.Time + cruiseSec
	}
	if a.chewing {
		a.chew(ctx, s)
		return
	}
	if a.beginMeal(ctx, s) {
		return
	}
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: bodyCruise}
	if a.armed || s.Time+1e-9 < a.cruiseUntil {
		return
	}
	a.armed = true
	ctx.Out <- unit.Ready{UnitID: ctx.ID}
}

func (a *鱼) stare(ctx unit.Context, s unit.Stare) {
	pool := Pool()
	if len(pool) == 0 || a.rng == nil {
		ctx.Out <- unit.Paper{UnitID: ctx.ID}
		return
	}
	stock := pool[a.rng.IntN(len(pool))]
	lo, hi := SpanX(s.Field)
	line, long := Side(s.Self.X, lo, hi, stock.Rate)
	ctx.Out <- unit.Paper{UnitID: ctx.ID, Stock: stock.Name, Long: long, Line: line}
}

func (a *鱼) beginMeal(ctx unit.Context, s unit.Sense) bool {
	bowl, ok := overlappingRice(s)
	if !ok || orcaOnBowl(s, bowl) {
		return false
	}
	a.chewing = true
	a.mealID = bowl.ID
	a.mealUntil = s.Time + eatSec
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
	ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
	return true
}

func (a *鱼) chew(ctx unit.Context, s unit.Sense) {
	bowl, ok := bowlByID(s, a.mealID)
	if !ok {
		a.chewing = false
		ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}
		ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: bodyCruise}
		return
	}
	if s.Time+1e-9 < a.mealUntil {
		ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: true}
		ctx.Out <- unit.SetVelocity{UnitID: ctx.ID, VX: 0, VY: 0}
		return
	}
	a.chewing = false
	ctx.Out <- unit.Stand{UnitID: ctx.ID, Hold: false}
	ctx.Out <- unit.Despawn{UnitID: bowl.ID}
	ctx.Out <- unit.Heal{UnitID: ctx.ID, Amount: mealHeal}
	ctx.Out <- unit.SetCruise{UnitID: ctx.ID, Speed: bodyCruise}
}

func overlappingRice(s unit.Sense) (unit.Snapshot, bool) {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind != kindRice {
			continue
		}
		if math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) <= s.Self.Radius+o.Radius {
			return *o, true
		}
	}
	return unit.Snapshot{}, false
}

func bowlByID(s unit.Sense, id uint64) (unit.Snapshot, bool) {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.ID == id && o.Kind == kindRice {
			return *o, true
		}
	}
	return unit.Snapshot{}, false
}

// orcaOnBowl 同一拍原身也压在这只碗上时，原身先吃。
func orcaOnBowl(s unit.Sense, bowl unit.Snapshot) bool {
	for i := range s.Nearby {
		o := &s.Nearby[i]
		if o.Kind != kindOrca || o.HP <= 0 || o.ID != bowl.OwnerID {
			continue
		}
		if math.Hypot(o.X-bowl.X, o.Y-bowl.Y) <= o.Radius+bowl.Radius {
			return true
		}
	}
	return false
}

// SpanX 沿场地外形采样，取最左和最右的横坐标。
func SpanX(f unit.Field) (lo, hi float64) {
	lo, hi = math.Inf(1), math.Inf(-1)
	const n = 720
	for i := 0; i < n; i++ {
		x, _, _, _ := f.SampleEdge(float64(i)/n, 0)
		if x < lo {
			lo = x
		}
		if x > hi {
			hi = x
		}
	}
	return lo, hi
}

// Side 圆心横坐标小于等于决定线为看涨。
func Side(x, lo, hi, rate float64) (line float64, long bool) {
	if !(hi > lo) {
		return x, true
	}
	line = lo + rate*(hi-lo)
	return line, x <= line
}
