package 靶子

import (
	"embed"
	"xqdj/internal/unit"
)

const KindDummy = "靶子"

//go:embed fx
var assets embed.FS

const (
	// dpsWindow 是「近 5 秒 DPS」看的那段窗口。
	dpsWindow = 5.0
	// dpsSayEvery 是播报节流：气泡每 0.2 秒最多换一次字。
	dpsSayEvery = 0.2
)

func init() {
	p := unit.NewPack(KindDummy, assets)
	p.Register(unit.Spec{
		Kind:    KindDummy,
		Role:    unit.RoleFighter,
		Radius:  18,
		MaxHP:   99999,
		Speed:   165,
		Vision:  9999,
		Fighter: true,
		Look:    unit.Look{Color: "#4aa3ff", FX: []string{"dummy-dps"}},
	}, func(info unit.SpawnInfo) unit.Actor {
		return &靶子{slot: info.Slot}
	})
}

// 靶子 挨打照单全收，并把自己吃到的伤害换算成 DPS，一直挂在头顶的气泡里。
// 气泡里的大字是全程 DPS（累计伤害 ÷ 开局至今），小字是近 5 秒 DPS 和累计伤害。
// 气泡由前端 dummy-dps 皮肤画；这里只负责每 0.2 秒把数字用 FX 送过去。
type 靶子 struct {
	slot int

	started bool
	start   float64 // 开局时间：第一次收到感知（或伤害）的时刻
	total   float64
	hits    []dpsHit

	said    bool
	lastSay float64
}

type dpsHit struct {
	at     float64
	amount float64
}

func (a *靶子) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		unit.ConfirmHit(ctx, e)
		a.record(e.Time, e.Amount)
		a.say(ctx, e.Time)
	case unit.Sense:
		a.observe(e.Time)
		a.say(ctx, e.Time)
	}
}

// observe 用第一帧感知定下开局时间。这样一枪没打时也会一直报 0.0，
// 而且第一枪落下时的 DPS 不会拿一帧去当分母。
func (a *靶子) observe(t float64) {
	if !a.started {
		a.started = true
		a.start = t
	}
}

func (a *靶子) record(t, amount float64) {
	if amount <= 0 {
		return
	}
	if !a.started {
		a.started = true
		a.start = t
	}
	a.total += amount
	a.hits = append(a.hits, dpsHit{at: t, amount: amount})
}

func (a *靶子) say(ctx unit.Context, t float64) {
	if !a.started {
		return
	}
	if a.said && t-a.lastSay < dpsSayEvery-1e-9 {
		return
	}
	a.said = true
	a.lastSay = t
	a.prune(t)
	dps, recent, total := a.read(t)
	ctx.Out <- unit.FX{
		Name:   "dps",
		Kind:   ctx.Kind,
		UnitID: ctx.ID,
		Slot:   a.slot,
		Amount: dps,
		VX:     recent,
		VY:     total,
	}
}

func (a *靶子) prune(t float64) {
	keep := a.hits[:0]
	for _, h := range a.hits {
		if t-h.at <= dpsWindow+1e-9 {
			keep = append(keep, h)
		}
	}
	a.hits = keep
}

func (a *靶子) read(t float64) (dps, recent, total float64) {
	span := t - a.start
	if span < 1e-6 {
		span = 1e-6
	}
	dps = a.total / span

	window := dpsWindow
	if span < window {
		window = span
	}
	var inWindow float64
	for _, h := range a.hits {
		inWindow += h.amount
	}
	if window > 1e-6 {
		recent = inWindow / window
	}
	return dps, recent, a.total
}
