package unit

import "testing"

func TestStreamReplays(t *testing.T) {
	a := NewRNG(7)
	b := NewRNG(7)
	if a.Uint64() != b.Uint64() {
		t.Fatal("same stream should replay")
	}
	c := NewRNG(8)
	d := NewRNG(9)
	if c.Uint64() == d.Uint64() && c.Uint64() == d.Uint64() {
		t.Fatal("different streams stayed in lockstep")
	}
}

func TestBornReplaysWithoutArrivalOrder(t *testing.T) {
	parent := DeriveStream(42, 1, StreamRoot)
	first := Context{Stream: parent}.WithBirths()
	a := first.Born(Spawn{Kind: "a"}).Stream
	b := first.Born(Spawn{Kind: "b"}).Stream
	second := Context{Stream: parent}.WithBirths()
	if second.Born(Spawn{}).Stream != a || second.Born(Spawn{}).Stream != b {
		t.Fatal("same parent and birth index must replay the stream")
	}
	other := Context{Stream: DeriveStream(42, 2, StreamRoot)}.WithBirths()
	if other.Born(Spawn{}).Stream == a {
		t.Fatal("different parent reused a child stream")
	}
}
