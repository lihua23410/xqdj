package 外科手术

import (
	"sync"
	"xqdj/internal/unit"
)

type clinic struct {
	sync.Mutex
	operations map[uint64]*operation
	blood      map[uint64]float64
}

func (c *clinic) operation(id uint64) *operation {
	c.Lock()
	defer c.Unlock()
	if c.operations == nil {
		c.operations = map[uint64]*operation{}
	}
	if c.operations[id] == nil {
		c.operations[id] = &operation{}
	}
	return c.operations[id]
}

func (c *clinic) markBlood(id uint64, now float64) {
	c.Lock()
	defer c.Unlock()
	if c.blood == nil {
		c.blood = map[uint64]float64{}
	}
	c.blood[id] = now + 3
}

func (c *clinic) isBlood(id uint64, now float64) bool {
	c.Lock()
	defer c.Unlock()
	return c.blood[id] >= now && c.blood[id] != 0
}

func (c *clinic) prune(now float64) {
	c.Lock()
	defer c.Unlock()
	for id, until := range c.blood {
		if until < now {
			delete(c.blood, id)
		}
	}
}

type operation struct {
	sync.Mutex
	self                          unit.Snapshot
	stage                         int
	readyAt, windUntil, stunUntil float64
	knocked, knifeOn, needleOn    bool
	crossPending                  bool
	stunned                       bool
	heading, lastBody             point
	route                         []point
	routeGoal                     point
	plan                          knifePlan
	serial, current               uint64
	wounds                        []*wound
	next, direction               int
	charge, lastTime              float64
	clockStarted                  bool
}

type punctureHit struct {
	At    float64
	Count int
}
type wound struct {
	ID     uint64  `json:"id"`
	A      point   `json:"a"`
	B      point   `json:"b"`
	Points []point `json:"points"`
	Closed []bool  `json:"closed"`
	Drawn  float64 `json:"drawn"`
	// Derived each sense for the view: true while any part of the wound is unsewn.
	Bleeding  bool `json:"bleeding"`
	nextSpray [6]float64
	hits      map[uint64]punctureHit
	sprayed   bool
}

func (w *wound) complete() bool {
	if w.Drawn < 1 || len(w.Closed) == 0 {
		return false
	}
	for _, closed := range w.Closed {
		if !closed {
			return false
		}
	}
	return true
}

func (w *wound) firstUnsewn(direction int) int {
	if direction > 0 {
		for i, closed := range w.Closed {
			if !closed {
				return i + 1
			}
		}
	} else {
		for i := len(w.Closed) - 1; i >= 0; i-- {
			if !w.Closed[i] {
				return i
			}
		}
	}
	return 0
}

func (o *operation) wound(id uint64) *wound {
	for _, w := range o.wounds {
		if w.ID == id {
			return w
		}
	}
	return nil
}

func (o *operation) hasOpen() bool {
	for _, w := range o.wounds {
		if w.Drawn >= 1 && !w.complete() {
			return true
		}
	}
	return false
}

func (o *operation) nearest(p point) (best *wound, direction int) {
	distance := 1e20
	for _, w := range o.wounds {
		if w.Drawn < 1 || w.complete() {
			continue
		}
		for _, d := range []int{1, -1} {
			n := w.firstUnsewn(d) - d
			candidate := w.Points[n].sub(p).length()
			if candidate < distance-1e-8 {
				best, direction, distance = w, d, candidate
			}
		}
	}
	return
}

type commands []unit.Cmd

func (c *commands) add(cmds ...unit.Cmd) { *c = append(*c, cmds...) }
func (c commands) send(ctx unit.Context) {
	for _, cmd := range c {
		if spawn, ok := cmd.(unit.Spawn); ok {
			cmd = ctx.Born(spawn)
		}
		ctx.Out <- cmd
	}
}
