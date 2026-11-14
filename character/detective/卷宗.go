package 探长

import (
	"sync"

	"xqdj/internal/unit"
)

// 卷宗是这一只探长自己的账。同场另一只探长另有一本。
type 卷宗 struct {
	mu            sync.Mutex
	suspicion     map[uint64]int
	investigation map[uint64]int
	guilty        map[uint64]bool
	suspect       uint64
	admonition    int
	pending       int // 我执写入，探长本人来掷申诫
}

func newBook() *卷宗 {
	return &卷宗{
		suspicion:     map[uint64]int{},
		investigation: map[uint64]int{},
		guilty:        map[uint64]bool{},
	}
}

// 案卷按探长分开。标记和探长会同时写，卷宗自己加锁。
type 案卷 struct {
	mu    sync.Mutex
	books map[uint64]*卷宗
}

func bookOf(scope *unit.MatchState, id uint64) *卷宗 {
	shelf := unit.StateFor[案卷](scope)
	shelf.mu.Lock()
	defer shelf.mu.Unlock()
	if shelf.books == nil {
		shelf.books = map[uint64]*卷宗{}
	}
	b := shelf.books[id]
	if b == nil {
		b = newBook()
		shelf.books[id] = b
	}
	return b
}

func (b *卷宗) addSuspicion(id uint64) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.suspicion[id]++
	b.recompute()
	return b.suspicion[id]
}

func (b *卷宗) addInvestigation(id uint64) (n int, newly bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.investigation[id]++
	n = b.investigation[id]
	if n >= investigateNeed && !b.guilty[id] {
		b.guilty[id] = true
		newly = true
	}
	return n, newly
}

func (b *卷宗) markGuilty(id uint64) bool {
	if id == 0 {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.guilty[id] {
		return false
	}
	b.guilty[id] = true
	return true
}

// implicate 在嫌疑人被这次检定定成有罪时，把其余有疑点的人一并定成有罪。
func (b *卷宗) implicate() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for id, n := range b.suspicion {
		if n > 0 {
			b.guilty[id] = true
		}
	}
}

func (b *卷宗) noteAdmonition() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pending++
	b.admonition++
	return b.admonition
}

func (b *卷宗) takePending() (n, stacks int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n = b.pending
	b.pending = 0
	return n, b.admonition
}

type 账 struct {
	suspicion     map[uint64]int
	investigation map[uint64]int
	guilty        map[uint64]bool
	suspect       uint64
	admonition    int
}

func (b *卷宗) snapshot() 账 {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := 账{
		suspicion:     make(map[uint64]int, len(b.suspicion)),
		investigation: make(map[uint64]int, len(b.investigation)),
		guilty:        make(map[uint64]bool, len(b.guilty)),
		suspect:       b.suspect,
		admonition:    b.admonition,
	}
	for id, n := range b.suspicion {
		out.suspicion[id] = n
	}
	for id, n := range b.investigation {
		out.investigation[id] = n
	}
	for id, on := range b.guilty {
		out.guilty[id] = on
	}
	return out
}

func (b *卷宗) suspectOf(id uint64) (suspicion, investigation int, guilty bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.suspicion[id], b.investigation[id], b.guilty[id]
}

func (b *卷宗) anyGuilty() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, on := range b.guilty {
		if on {
			return true
		}
	}
	return false
}

// recompute 在已持锁时调用。疑点并列时，原来的嫌疑人还在并列里就留着，否则取编号最小的。
func (b *卷宗) recompute() {
	bestN := 0
	var ids []uint64
	for id, n := range b.suspicion {
		if n > bestN {
			bestN = n
			ids = []uint64{id}
			continue
		}
		if n == bestN && n > 0 {
			ids = append(ids, id)
		}
	}
	if bestN == 0 {
		b.suspect = 0
		return
	}
	for _, id := range ids {
		if id == b.suspect {
			return
		}
	}
	best := ids[0]
	for _, id := range ids[1:] {
		if id < best {
			best = id
		}
	}
	b.suspect = best
}
