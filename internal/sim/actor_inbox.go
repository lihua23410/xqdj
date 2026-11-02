package sim

import (
	"sync"
	"sync/atomic"

	unitpkg "xqdj/internal/unit"
)

// actorWork counts queued events and handlers still executing. A handler's
// commands have reached cmds before it marks its event complete.
type actorWork struct {
	pending atomic.Int64
	wake    chan struct{}
}

func newActorWork() *actorWork { return &actorWork{wake: make(chan struct{}, 1)} }

func (w *actorWork) changed(n int64) {
	w.pending.Add(n)
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

// actorInbox preserves every event, including every Sense, in FIFO order.
// Enqueue never waits for Handle: a handler can itself be waiting to send a
// command to the engine. The engine provides backpressure at batch boundaries.
type actorInbox struct {
	mu     sync.Mutex
	queue  []unitpkg.Event
	head   int
	closed bool
	ready  chan struct{}
	work   *actorWork
}

func newActorInbox(work *actorWork) *actorInbox {
	if work == nil {
		work = newActorWork()
	}
	return &actorInbox{ready: make(chan struct{}, 1), work: work}
}

func (q *actorInbox) push(ev unitpkg.Event) bool {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return false
	}
	q.queue = append(q.queue, ev)
	q.work.changed(1)
	q.mu.Unlock()
	q.signal()
	return true
}

func (q *actorInbox) signal() {
	select {
	case q.ready <- struct{}{}:
	default:
	}
}

func (q *actorInbox) next(stop <-chan struct{}) (unitpkg.Event, bool) {
	for {
		select {
		case <-stop:
			return nil, false
		default:
		}
		q.mu.Lock()
		if q.head < len(q.queue) {
			ev := q.queue[q.head]
			q.queue[q.head] = nil
			q.head++
			if q.head == len(q.queue) {
				q.queue = q.queue[:0]
				q.head = 0
			}
			q.mu.Unlock()
			return ev, true
		}
		closed := q.closed
		q.mu.Unlock()
		if closed {
			return nil, false
		}
		select {
		case <-stop:
			return nil, false
		case <-q.ready:
		}
	}
}

func (q *actorInbox) finish() { q.work.changed(-1) }

func (q *actorInbox) close() {
	q.mu.Lock()
	if !q.closed {
		q.closed = true
		n := len(q.queue) - q.head
		q.queue = nil
		q.head = 0
		if n > 0 {
			q.work.changed(-int64(n))
		}
	}
	q.mu.Unlock()
	q.signal()
}
