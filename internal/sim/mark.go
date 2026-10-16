package sim

import (
	"sort"

	unitpkg "xqdj/internal/unit"
)

type stackMark struct {
	kind   string
	stacks int
	icon   string
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
		cur = &stackMark{kind: c.Kind, icon: c.Icon}
		u.marks[c.Kind] = cur
	}
	cur.stacks += c.Delta
	if c.Icon != "" {
		cur.icon = c.Icon
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
		out = append(out, unitpkg.Mark{Kind: m.kind, Stacks: m.stacks, Icon: m.icon})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
