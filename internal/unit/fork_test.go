package unit

import (
	"math/rand/v2"
	"sync"
	"testing"
)

type secret struct {
	n    int
	s    []int
	m    map[string]int
	ch   chan int
	mu   sync.Mutex
	rng  *rand.Rand
	next *secret
	fn   func() int
}

func (s *secret) Handle(Context, Event) {}

type valueActor struct{ n int }

func (valueActor) Handle(Context, Event) {}

type emptyActor struct{}

func (emptyActor) Handle(Context, Event) {}

type pointerValueActor struct{ value *int }

func (pointerValueActor) Handle(Context, Event) {}

func TestForkActorValue(t *testing.T) {
	cloned, ok := ForkActor(valueActor{n: 4}, nil).(valueActor)
	if !ok || cloned.n != 4 {
		t.Fatalf("%T %+v", cloned, cloned)
	}
	if _, ok := ForkActor(emptyActor{}, nil).(emptyActor); !ok {
		t.Fatal("empty actor")
	}
	number := 0
	original := pointerValueActor{value: &number}
	clonedPointer, ok := ForkActor(original, nil).(pointerValueActor)
	if !ok || clonedPointer.value == nil || clonedPointer.value == original.value || *clonedPointer.value != number {
		t.Fatalf("single-pointer value actor lost its private copy: %+v", clonedPointer)
	}
	var typedNil *secret
	if ForkActor(Actor(typedNil), nil) != nil {
		t.Fatal("typed nil")
	}
}

func TestForkKeepsPrivateStateApart(t *testing.T) {
	s := &secret{
		n: 3, s: []int{1, 2}, m: map[string]int{"a": 1},
		ch: make(chan int), rng: rand.New(rand.NewPCG(1, 2)),
		fn: func() int { return 7 },
	}
	s.next = s
	s.mu.Lock()
	cloned, ok := ForkActor(s, nil).(*secret)
	s.mu.Unlock()
	if !ok || cloned == nil || cloned == s {
		t.Fatal("fork")
	}
	s.n = 9
	s.s[0] = 8
	s.m["a"] = 4
	if cloned.n != 3 || cloned.s[0] != 1 || cloned.m["a"] != 1 {
		t.Fatalf("shared state %+v", cloned)
	}
	if cloned.next != cloned {
		t.Fatal("cycle")
	}
	if cloned.ch != nil {
		t.Fatal("channel copied")
	}
	if cloned.fn() != 7 {
		t.Fatal("func")
	}
	if s.rng.Uint64() != cloned.rng.Uint64() {
		t.Fatal("rng diverged from the same copied source")
	}
}
