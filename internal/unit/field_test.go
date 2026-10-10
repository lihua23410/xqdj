package unit

import "testing"

func TestHexFieldShape(t *testing.T) {
	f := HexField()
	if f.Shape != ShapeHex || f.Extent != HexRadius {
		t.Fatalf("%+v", f)
	}
}
