package sim

import (
	"errors"
	"math/rand/v2"

	unitpkg "xqdj/internal/unit"
	六边形 "xqdj/map/hex"
	大六边形 "xqdj/map/widehex"
)

var (
	ErrBattleInProgress = errors.New("已有对局进行中，请先结束当前对局")
	ErrScheduledRoster  = errors.New("赛程对战需要两组各两只不同的战斗机。新加入的战斗机需要重启服务后才能参战")
)

type ScheduledBattle struct {
	Field  string `json:"field"`
	Status string `json:"status"`
	Result string `json:"result,omitempty"`
}

// StartScheduled installs both sides and the random field as one operation.
// Retrying with the same id returns the original attempt.
func (m *Match) StartScheduled(id string, sides [][]string) (ScheduledBattle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if battle, ok := m.scheduled[id]; ok {
		return battle, nil
	}
	if m.phase != PhaseSelect && m.phase != PhaseEnded {
		return ScheduledBattle{}, ErrBattleInProgress
	}
	if len(sides) != 2 || len(sides[0]) != 2 || len(sides[1]) != 2 {
		return ScheduledBattle{}, ErrScheduledRoster
	}
	seen := map[string]bool{"靶子": true}
	for _, side := range sides {
		for _, fighter := range side {
			if seen[fighter] || !unitpkg.IsFighterKind(fighter) {
				return ScheduledBattle{}, ErrScheduledRoster
			}
			seen[fighter] = true
		}
	}

	fields := [...]string{六边形.Name, 大六边形.Name}
	spec, ok := lookupField(fields[rand.IntN(len(fields))])
	if !ok {
		return ScheduledBattle{}, errors.New("赛程场地不可用")
	}
	m.team = true
	m.slots = [2][]string{sides[0], sides[1]}
	m.slots = sidesOrEmpty(m.slots)
	m.rememberTeamLocked()
	m.spec = spec
	battle := ScheduledBattle{Field: spec.name, Status: "running"}
	if m.scheduled == nil {
		m.scheduled = make(map[string]ScheduledBattle)
	}
	m.scheduled[id] = battle
	m.startLocked(id)
	return battle, nil
}

func (m *Match) ScheduledStatus(id string) (ScheduledBattle, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	battle, ok := m.scheduled[id]
	return battle, ok
}

func (m *Match) finishScheduledLocked(status, result string) {
	if m.scheduledID == "" {
		return
	}
	battle := m.scheduled[m.scheduledID]
	if battle.Status != "running" {
		return
	}
	battle.Status = status
	battle.Result = result
	m.scheduled[m.scheduledID] = battle
}
