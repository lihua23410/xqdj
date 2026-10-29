package 回声

import (
	"sort"

	"xqdj/internal/unit"
)

type scanLock struct {
	until   float64
	resetAt float64
	reset   bool // 等清零快照落地，防止旧感知重复开启上一轮锁定
}

func lockedKind(scanKind string) string { return scanKind + "锁定" }

func (e *回声) lockedAt(now float64) bool {
	for _, lock := range e.locks {
		if !lock.reset && now+1e-9 < lock.until {
			return true
		}
	}
	return false
}

func (e *回声) expireLocks(ctx unit.Context, now float64) {
	var expired []uint64
	for id, lock := range e.locks {
		if !lock.reset && now+1e-9 >= lock.until {
			expired = append(expired, id)
		}
	}
	sort.Slice(expired, func(i, j int) bool { return expired[i] < expired[j] })
	for _, id := range expired {
		e.locks[id].reset = true
		e.locks[id].resetAt = now
		e.charge = 0
		ctx.Out <- unit.ClearMarks{UnitID: id, Kind: countKind(ctx.ID)}
		ctx.Out <- unit.ClearMarks{UnitID: id, Kind: lockedKind(countKind(ctx.ID))}
	}
}

// 扫描层保留逐层十秒寿命。满环后换成同时创建的锁定层：六秒内不变，
// 到点一起清除；即使本体眩晕或倒下，也不会留下永久标记。
func (e *回声) syncLocks(ctx unit.Context, s unit.Sense) {
	if e.locks == nil {
		e.locks = map[uint64]*scanLock{}
	}
	scanKind := countKind(ctx.ID)
	freezeKind := lockedKind(scanKind)
	live := map[uint64]bool{}
	// 用引擎实际落地的过期时间校准窗口，不让后续命中刷新它。
	for _, o := range s.Nearby {
		live[o.ID] = true
		for _, mark := range o.Marks {
			if mark.Kind != freezeKind || mark.Stacks <= 0 {
				continue
			}
			lock := e.locks[o.ID]
			if lock == nil {
				lock = &scanLock{until: s.Time + lockLife}
				e.locks[o.ID] = lock
			}
			if !lock.reset && mark.NextDecayAt > 0 {
				lock.until = mark.NextDecayAt
			}
		}
	}
	e.expireLocks(ctx, s.Time)
	for _, o := range s.Nearby {
		scans := markStacks(o, scanKind)
		if lock := e.locks[o.ID]; lock != nil {
			if !lock.reset || markStacks(o, freezeKind) > 0 {
				continue
			}
			fresh := false
			for _, mark := range o.Marks {
				if mark.Kind == scanKind && mark.Lifetime > 0 && mark.NextDecayAt-mark.Lifetime+1e-9 >= lock.resetAt {
					fresh = true
				}
			}
			if scans > 0 && !fresh {
				continue
			}
			// 清零后立即到来的新命中也能开启下一轮，不必先观察到一帧零层。
			delete(e.locks, o.ID)
		}
		if o.Slot == s.Self.Slot || !unit.Hittable(o, s.Self.Slot) || o.Nonsolid || scans < countMax {
			continue
		}
		e.locks[o.ID] = &scanLock{until: s.Time + lockLife}
		ctx.Out <- unit.ClearMarks{UnitID: o.ID, Kind: scanKind}
		ctx.Out <- unit.StackMark{UnitID: o.ID, Kind: freezeKind, Delta: countMax, Icon: countIcon, Lifetime: lockLife, Hidden: true}
	}
	for id := range e.locks {
		if !live[id] {
			delete(e.locks, id)
		}
	}
}
