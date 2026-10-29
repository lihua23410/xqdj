package sim

import (
	"sort"

	unitpkg "xqdj/internal/unit"
)

type stackMark struct {
	kind     string
	stacks   int
	icon     string
	lifetime float64
	hidden   bool
	hits     []float64 // 每层叠上去的绝对时间；只对 Lifetime > 0 的标记有值
}

func (m *Match) stackMarkLocked(c unitpkg.StackMark) {
	u := m.units[c.UnitID]
	if u == nil || u.stopped || c.Kind == "" || c.Delta == 0 {
		return
	}
	if u.marks == nil {
		u.marks = map[string]*stackMark{}
	}
	prev := 0
	if cur := u.marks[c.Kind]; cur != nil {
		prev = cur.stacks
	}
	cur := u.marks[c.Kind]
	if cur == nil {
		if c.Delta < 0 {
			return
		}
		cur = &stackMark{kind: c.Kind, icon: c.Icon, lifetime: c.Lifetime, hidden: c.Hidden}
		u.marks[c.Kind] = cur
	}
	if cur.lifetime > 0 {
		switch {
		case c.Delta > 0:
			for i := 0; i < c.Delta; i++ {
				cur.hits = append(cur.hits, m.time)
			}
		case c.Delta < 0:
			n := -c.Delta
			if n > len(cur.hits) {
				n = len(cur.hits)
			}
			cur.hits = cur.hits[n:]
		}
		cur.stacks = len(cur.hits)
	} else if c.Lifetime > 0 {
		// 原本是永久标记，第一次带寿命的层把已有层一起盖上寿命。
		cur.lifetime = c.Lifetime
		for i := 0; i < cur.stacks+c.Delta; i++ {
			cur.hits = append(cur.hits, m.time)
		}
		cur.stacks = len(cur.hits)
	} else {
		cur.stacks += c.Delta
	}
	if c.Icon != "" {
		cur.icon = c.Icon
	}
	if c.Hidden {
		cur.hidden = true
	}
	if cur.stacks <= 0 {
		delete(u.marks, c.Kind)
	}
	next := 0
	if cur := u.marks[c.Kind]; cur != nil {
		next = cur.stacks
	}
	m.noteMarkLocked(u, c.Kind, prev, next)
}

func (m *Match) clearMarksLocked(c unitpkg.ClearMarks) {
	u := m.units[c.UnitID]
	if u == nil {
		return
	}
	if c.Kind == "" {
		if len(u.marks) == 0 {
			return
		}
		keys := make([]string, 0, len(u.marks))
		for k, mk := range u.marks {
			if mk != nil && mk.stacks > 0 {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		u.marks = nil
		for _, k := range keys {
			m.noteMarkLocked(u, k, 1, 0)
		}
		return
	}
	prev := 0
	if cur := u.marks[c.Kind]; cur != nil {
		prev = cur.stacks
	}
	delete(u.marks, c.Kind)
	m.noteMarkLocked(u, c.Kind, prev, 0)
}

// decayMarksLocked 按每层自己的寿命掉层：每一层从叠上去那一拍起活 Lifetime 秒。
// 叠新层不延长旧层；想维持锁定，就得在旧层过期前继续打中。
func (m *Match) decayMarksLocked() {
	for _, u := range m.units {
		if u == nil || len(u.marks) == 0 {
			continue
		}
		for kind, mk := range u.marks {
			if mk == nil || mk.lifetime <= 0 || len(mk.hits) == 0 {
				continue
			}
			prev := mk.stacks
			cut := 0
			for cut < len(mk.hits) && m.time-mk.hits[cut] >= mk.lifetime {
				cut++
			}
			if cut == 0 {
				continue
			}
			mk.hits = mk.hits[cut:]
			mk.stacks = len(mk.hits)
			if mk.stacks <= 0 {
				delete(u.marks, kind)
				m.noteMarkLocked(u, kind, prev, 0)
			} else {
				m.noteMarkLocked(u, kind, prev, mk.stacks)
			}
		}
	}
}

func (u *unit) markList() []unitpkg.Mark {
	if u == nil || len(u.marks) == 0 {
		return nil
	}
	keys := make([]string, 0, len(u.marks))
	for k := range u.marks {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]unitpkg.Mark, 0, len(keys))
	for _, k := range keys {
		m := u.marks[k]
		if m == nil || m.stacks <= 0 {
			continue
		}
		next := 0.0
		if m.lifetime > 0 && len(m.hits) > 0 {
			next = m.hits[0] + m.lifetime
		}
		out = append(out, unitpkg.Mark{
			Kind: m.kind, Stacks: m.stacks, Icon: m.icon,
			Lifetime: m.lifetime, NextDecayAt: next, Hidden: m.hidden,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
