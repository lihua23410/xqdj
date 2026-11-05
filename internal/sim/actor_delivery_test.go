package sim

import (
	"sync"
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestSendPreservesDamageBehindSenseBurst(t *testing.T) {
	m := NewHeadlessMatchSeeded(1)
	u := &unit{inbox: newActorInbox(m.actorWork), stop: make(chan struct{})}
	for i := 0; i < 64; i++ {
		m.send(u, unitpkg.Sense{Time: float64(i) * DT})
	}
	m.send(u, unitpkg.IncomingDamage{Token: 1, Amount: 7})
	for i := 0; i < 65; i++ {
		ev, ok := u.inbox.next(u.stop)
		if !ok {
			t.Fatal("inbox closed")
		}
		u.inbox.finish()
		if _, ok := ev.(unitpkg.IncomingDamage); ok {
			return
		}
	}
	t.Fatal("damage disappeared behind queued Sense events")
}

func TestActorInboxKeepsEverySenseInOrder(t *testing.T) {
	q := newActorInbox(nil)
	stop := make(chan struct{})
	for i := 0; i < 1024; i++ {
		if !q.push(unitpkg.Sense{Time: float64(i)}) {
			t.Fatal("open inbox rejected event")
		}
	}
	for i := 0; i < 1024; i++ {
		ev, ok := q.next(stop)
		if !ok || ev.(unitpkg.Sense).Time != float64(i) {
			t.Fatalf("Sense %d lost or reordered", i)
		}
		q.finish()
	}
	if n := q.work.pending.Load(); n != 0 {
		t.Fatalf("completed queue has %d pending events", n)
	}
}

func TestActorInboxCloseCancelsQueuedButNotActiveEvent(t *testing.T) {
	q := newActorInbox(nil)
	stop := make(chan struct{})
	for i := 0; i < 100; i++ {
		q.push(unitpkg.Sense{})
	}
	q.next(stop)
	q.close()
	q.close()
	if n := q.work.pending.Load(); n != 1 {
		t.Fatalf("active handler lost its work count: %d", n)
	}
	if q.push(unitpkg.IncomingDamage{}) {
		t.Fatal("closed inbox accepted event")
	}
	if _, ok := q.next(stop); ok {
		t.Fatal("closed inbox retained queued events")
	}
	q.finish()
	if n := q.work.pending.Load(); n != 0 {
		t.Fatalf("work leaked after handler completed: %d", n)
	}
}

type deliveryActorFunc func(unitpkg.Context, unitpkg.Event)

func (f deliveryActorFunc) Handle(ctx unitpkg.Context, ev unitpkg.Event) { f(ctx, ev) }

func newDeliveryMatch(t *testing.T) *Match {
	t.Helper()
	m := NewHeadlessMatchSeeded(5)
	m.SetSlot(0, character.KindDummy)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	t.Cleanup(m.End)
	return m
}

func TestTickWaitsForSlowActorAndDrainsFullCommandChannel(t *testing.T) {
	m := newDeliveryMatch(t)
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	emitted := make(chan struct{})
	var first sync.Once
	m.mu.Lock()
	u := fighterSlot(m, 0)
	u.actor = deliveryActorFunc(func(ctx unitpkg.Context, ev unitpkg.Event) {
		if _, ok := ev.(unitpkg.Sense); !ok {
			return
		}
		first.Do(func() {
			// More than the command channel can hold. The engine must
			// receive while waiting for this Handle call to finish.
			for i := 0; i < 1024; i++ {
				ctx.Out <- unitpkg.FX{}
			}
			close(emitted)
			<-release
		})
	})
	m.mu.Unlock()
	m.Tick() // Delivers the first Sense; handling remains asynchronous.
	done := make(chan struct{})
	go func() { m.Tick(); close(done) }()
	select {
	case <-emitted:
	case <-time.After(3 * time.Second):
		t.Fatal("engine stopped receiving while actor filled command channel")
	}
	select {
	case <-done:
		t.Fatal("simulation advanced while the previous Sense was still executing")
	case <-time.After(10 * time.Millisecond):
	}
	unblock()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("simulation did not resume after actor completed")
	}
}

func TestCommandBatchDefersNewReactions(t *testing.T) {
	m := newDeliveryMatch(t)
	m.mu.Lock()
	defer m.mu.Unlock()
	left, right := fighterSlot(m, 0), fighterSlot(m, 1)
	before := right.hp
	m.cmds <- unitpkg.Damage{From: left.id, To: right.id, Amount: 7}
	m.drainCmdsLocked()
	if len(m.pendingDmg) != 1 || right.hp != before {
		t.Fatal("a response escaped its command batch boundary")
	}
	m.drainCmdsLocked()
	if len(m.pendingDmg) != 0 || right.hp != before-7 {
		t.Fatal("next batch failed to confirm the pending damage")
	}
}

func TestActorBatchStillRunsActorsConcurrently(t *testing.T) {
	m := newDeliveryMatch(t)
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	m.mu.Lock()
	for slot := 0; slot < 2; slot++ {
		u := fighterSlot(m, slot)
		u.actor = deliveryActorFunc(func(_ unitpkg.Context, _ unitpkg.Event) {
			entered <- struct{}{}
			<-release
		})
		m.send(u, unitpkg.Sense{})
	}
	m.mu.Unlock()
	done := make(chan struct{})
	go func() { m.mu.Lock(); m.drainCmdsLocked(); m.mu.Unlock(); close(done) }()
	for i := 0; i < 2; i++ {
		select {
		case <-entered:
		case <-time.After(3 * time.Second):
			t.Fatal("actors were executed sequentially")
		}
	}
	unblock()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("actor batch failed to finish")
	}
}
