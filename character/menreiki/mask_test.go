package 面灵气

import "testing"

func TestMaskHitGateIndependent(t *testing.T) {
	const owner uint64 = 7
	if spendMask(owner, 0) {
		t.Fatal("unarmed should not hit")
	}
	armMask(owner, 0)
	if !spendMask(owner, 0) {
		t.Fatal("armed mask 0 should hit")
	}
	if spendMask(owner, 0) {
		t.Fatal("spent mask 0 should not hit again")
	}
	if spendMask(owner, 1) {
		t.Fatal("mask 1 should stay independent")
	}
	armAllMasks(owner)
	for i := 0; i < 3; i++ {
		if !spendMask(owner, i) {
			t.Fatalf("arm all: mask %d", i)
		}
	}
}
