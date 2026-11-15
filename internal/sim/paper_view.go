package sim

import (
	"math"

	unitpkg "xqdj/internal/unit"
)

const (
	paperOpenTicks     = 84
	paperPortraitAt    = 30
	paperPortraitTicks = 90
	paperFreezeTicks   = 66
	paperFlightTicks   = 54
	paperImpactTicks   = 96
	paperCloseTicks    = paperFreezeTicks + paperFlightTicks + paperImpactTicks
)

// paperView accompanies the spectator frame. Both the chart and the closing
// slip use damage quotes, before shields and other real-world mitigation.
type paperView struct {
	Stock         string            `json:"stock"`
	Long          bool              `json:"long"`
	Line          float64           `json:"line"`
	HostID        uint64            `json:"hostId"`
	HostX         float64           `json:"hostX"`
	HostY         float64           `json:"hostY"`
	Preview       *unitpkg.Snapshot `json:"preview,omitempty"`
	Elapsed       float64           `json:"elapsed"`
	Duration      float64           `json:"duration"`
	Self          float64           `json:"self"`
	Enemy         float64           `json:"enemy"`
	Candles       []paperCandle     `json:"candles"`
	OpenProgress  float64           `json:"openProgress,omitempty"`
	CutIn         float64           `json:"cutIn,omitempty"`
	Settled       bool              `json:"settled,omitempty"`
	CloseProgress float64           `json:"closeProgress,omitempty"`
	Flight        float64           `json:"flight,omitempty"`
	ImpactTicks   int               `json:"impactTicks,omitempty"`
	Targets       []*paperTarget    `json:"targets,omitempty"`
}

type paperTarget struct {
	ID        uint64   `json:"id"`
	X         float64  `json:"x"`
	Y         float64  `json:"y"`
	Self      bool     `json:"self,omitempty"`
	ImpactSeq uint64   `json:"impactSeq,omitempty"`
	Loss      *float64 `json:"loss,omitempty"`
}

type paperCandle struct {
	Open  float64 `json:"open"`
	High  float64 `json:"high"`
	Low   float64 `json:"low"`
	Close float64 `json:"close"`
}

func (r *paperRun) advance() {
	if r.view != nil && r.openFrames < paperOpenTicks+paperPortraitTicks {
		r.openFrames++
		portraitFrames := min(paperPortraitTicks, max(0, r.openFrames-paperPortraitAt))
		openFrame := r.openFrames - portraitFrames
		r.view.OpenProgress = float64(openFrame) / paperOpenTicks
		r.view.CutIn = 0
		if portraitFrames > 0 && openFrame == paperPortraitAt {
			r.view.CutIn = float64(portraitFrames) / paperPortraitTicks
		}
		return
	}
	r.frames++
	r.child.Tick()
	if r.view != nil {
		r.view.OpenProgress = 0
		r.view.Preview = nil
		r.view.Elapsed = min(paperSec, max(0, r.child.time-(r.until-paperSec)))
		r.view.quote(r.host, settleQuotes(r.host, r.rows, r.child.ledger.loss, r.long))
	}
	r.closing = r.child.time+1e-9 >= r.until || r.frames >= paperCap
}

func (m *Match) paperViewLocked() *paperView {
	if m.paper != nil {
		return m.paper.view
	}
	if m.paperClose == nil || (m.seq >= m.paperCloseUntil && !m.settling) {
		return nil
	}
	view := *m.paperClose
	age := float64(m.seq - m.paperCloseSeq)
	view.CloseProgress = age / max(1, float64(m.paperCloseUntil-m.paperCloseSeq))
	view.Flight = min(1, max(0, (age-paperFreezeTicks)/paperFlightTicks))
	return &view
}

func (m *Match) preparePaperCloseLocked(run *paperRun) {
	view := run.view
	view.quote(run.host, m.settleQ)
	view.Settled = true
	view.ImpactTicks = paperImpactTicks
	view.Targets = make([]*paperTarget, len(m.settleQ))
	for i, q := range m.settleQ {
		if u := m.units[q.To]; u.takesHit() {
			view.Targets[i] = &paperTarget{ID: u.id, X: u.p.X, Y: u.p.Y, Self: q.To == run.host}
		}
	}
	m.paperClose = view
	m.paperCloseSeq = m.seq
	m.paperCloseUntil = m.seq + paperCloseTicks
	m.fx = m.fx[:0]
}

func (m *Match) linkPaperImpactLocked(q unitpkg.Damage, token uint64) {
	if m.paperClose == nil {
		return
	}
	targets := m.paperClose.Targets
	target := targets[len(targets)-len(m.settleQ)]
	if target == nil {
		return
	}
	target.ImpactSeq = m.seq + 1
	m.paperCloseUntil = max(m.paperCloseUntil, target.ImpactSeq+paperImpactTicks)
	if u := m.units[q.To]; u != nil {
		target.X, target.Y = u.p.X, u.p.Y
	}
	if token == 0 {
		target.finish(0)
		return
	}
	offer := m.pendingDmg[token]
	offer.paperTarget = target
	m.pendingDmg[token] = offer
}

func (t *paperTarget) finish(loss float64) {
	if t != nil {
		t.Loss = &loss
	}
}

func (v *paperView) quote(host uint64, quotes []unitpkg.Damage) {
	v.Self, v.Enemy = 0, 0
	for _, q := range quotes {
		if q.To == host {
			v.Self += q.Amount
		} else {
			v.Enemy += q.Amount
		}
	}
	index := max(0, int(math.Ceil(v.Elapsed-1e-9))-1)
	for len(v.Candles) <= index {
		var open float64
		if n := len(v.Candles); n > 0 {
			open = v.Candles[n-1].Close
		}
		v.Candles = append(v.Candles, paperCandle{open, open, open, open})
	}
	c := &v.Candles[index]
	c.Close = v.Enemy - v.Self
	c.High = max(c.High, c.Close)
	c.Low = min(c.Low, c.Close)
}
