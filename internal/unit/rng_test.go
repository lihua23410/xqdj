package unit

import "testing"

func TestNewRNGDoesNotPinOnID(t *testing.T) {
	a := NewRNG(1)
	b := NewRNG(1)
	if a.Uint64() == b.Uint64() && a.Uint64() == b.Uint64() {
		t.Fatal("same id should not replay the stream")
	}
}
