package sim

import (
	"fmt"
	"strconv"
	"time"
	"xqdj/internal/replay"
	unitpkg "xqdj/internal/unit"
)

// QueueOpening 提交下一场开场演出，按提交顺序播放。仅在选人/结束阶段接受，
// 不支持局中插播。调用者负责准备数据；引擎只校验、归档和等待结束回执。
func (m *Match) QueueOpening(p replay.Presentation) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.phase != PhaseSelect && m.phase != PhaseEnded {
		return "", fmt.Errorf("opening can only be queued before a match")
	}
	return m.queueOpeningLocked(p)
}

func (m *Match) queueOpeningLocked(p replay.Presentation) (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	if !unitpkg.HasPack(p.Pack) {
		return "", fmt.Errorf("unknown presentation pack %q", p.Pack)
	}
	m.presentationN++
	p = p.Clone()
	p.ID = strconv.FormatUint(m.presentationN, 10)
	m.pendingOpening = append(m.pendingOpening, p)
	return p.ID, nil
}

func (m *Match) takeOpeningLocked() []replay.Presentation {
	if m.openingUntil.IsZero() {
		return nil
	}
	return m.activeOpening
}

func (m *Match) holdingOpening() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.openingUntil.IsZero() {
		return false
	}
	if time.Now().Before(m.openingUntil) {
		return true
	}
	m.openingUntil = time.Time{}
	return false
}

// FinishOpening 只接受当前队列最后一段的回执，防止前一场的迟到回执跳过演出。
func (m *Match) FinishOpening(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := len(m.activeOpening)
	if n > 0 && m.activeOpening[n-1].ID == id {
		m.openingUntil = time.Time{}
	}
}
