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
}

func (m *Match) clearMarksLocked(c unitpkg.ClearMarks) {
	u := m.units[c.UnitID]
	if u == nil {
		return
	}
	if c.Kind == "" {
		u.marks = nil
		return
	}
	delete(u.marks, c.Kind)
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
