package 外科手术

import (
	"encoding/json"
	"math"
	"math/rand/v2"
	"xqdj/internal/unit"
)

type woundField struct {
	owner uint64
	rng   *rand.Rand
}

func chargeRate(count int) float64 {
	if count == 0 {
		return 0
	}
	return 10 + 2*float64(count-1)
}

func (o *operation) advance(now float64) bool {
	if o.clockStarted {
		open := 0
		for _, w := range o.wounds {
			if !w.complete() {
				open++
			}
		}
		o.charge = math.Min(burstCharge, o.charge+math.Max(0, now-o.lastTime)*chargeRate(open))
	}
	o.lastTime, o.clockStarted = now, true
	if o.charge < burstCharge-1e-8 {
		return false
	}
	burst := false
	for _, w := range o.wounds {
		if !w.complete() {
			continue
		}
		burst = true
		for i := range w.Closed {
			w.Closed[i] = false
		}
		w.hits = map[uint64]punctureHit{}
	}
	if burst {
		o.charge = 0
	}
	return burst
}

type surgeryView struct {
	Stage     int      `json:"stage"`
	Charge    float64  `json:"charge"`
	Capacity  float64  `json:"capacity"`
	Immune    bool     `json:"immune"`
	Wounds    []*wound `json:"wounds"`
	Warning   []point  `json:"warning,omitempty"`
	WindUntil float64  `json:"windUntil,omitempty"`
	Heading   point    `json:"heading"`
}

func (f *woundField) Handle(ctx unit.Context, ev unit.Event) {
	s, ok := ev.(unit.Sense)
	if !ok {
		return
	}
	world := unit.StateFor[clinic](ctx.State)
	world.prune(s.Time)
	o := world.operation(f.owner)
	if f.rng == nil {
		f.rng = unit.NewRNG(ctx.Stream)
	}
	var cmds commands
	o.Lock()
	if o.advance(s.Time) {
		cmds.add(unit.FX{Name: "surgery-burst", Kind: KindSurgery, UnitID: f.owner, X: o.self.X, Y: o.self.Y})
	}
	for _, w := range o.wounds {
		// A wound bleeds for as long as any part of it is unsewn. 大出血 only
		// reopens closed wounds; it never switches spraying on or off.
		w.Bleeding = w.Drawn >= 1 && !w.complete()
		if !w.Bleeding {
			continue
		}
		if !w.sprayed {
			w.sprayed = true
			for i := range w.nextSpray {
				w.nextSpray[i] = s.Time + float64(i)/6
			}
		}
		d := norm(w.B.sub(w.A))
		normal := point{-d.Y, d.X}
		for i := range w.nextSpray {
			if s.Time+1e-9 < w.nextSpray[i] {
				continue
			}
			w.nextSpray[i] += 1
			if w.nextSpray[i] <= s.Time {
				w.nextSpray[i] = s.Time + 1
			}
			t := (float64(i) + .5) / 6
			segment := min(len(w.Closed)-1, int(t*float64(len(w.Closed))))
			if w.Closed[segment] {
				continue
			}
			p := lerp(w.A, w.B, t)
			for _, side := range []float64{-1, 1} {
				a := math.Atan2(normal.Y*side, normal.X*side) + (f.rng.Float64()*2-1)*unit.Deg(20)
				v := point{math.Cos(a), math.Sin(a)}
				start := p.add(v.mul(8))
				cmds.add(unit.Spawn{Kind: KindBloodBullet, OwnerID: f.owner, Slot: o.self.Slot,
					X: start.X, Y: start.Y, VX: v.X * 240, VY: v.Y * 240})
			}
		}
	}
	v := surgeryView{Stage: o.stage, Charge: o.charge, Capacity: burstCharge, Immune: o.immune(s.Time), Wounds: o.wounds, Heading: o.heading}
	if o.stage == winding {
		v.Warning = []point{o.plan.Start, o.plan.End}
		v.WindUntil = o.windUntil
	}
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	cmds.add(unit.FX{Name: "surgery-state", UnitID: f.owner, Kind: KindSurgery, Data: data})
	o.Unlock()
	cmds.send(ctx)
}
