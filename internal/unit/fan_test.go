package unit

import "testing"

func TestKeepFanOnlyScalesFanHits(t *testing.T) {
	fan := IncomingDamage{Amount: 10, ArcSpan: 0.4}
	if !fan.FromFan() {
		t.Fatal("span should count as fan")
	}
	if got := fan.KeepFan(0.4); got != 4 {
		t.Fatalf("keep=%v", got)
	}
	if got := fan.KeepFan(2); got != 10 {
		t.Fatalf("keep above 1 = %v", got)
	}
	if got := fan.KeepFan(-1); got != 0 {
		t.Fatalf("keep below 0 = %v", got)
	}
	shot := IncomingDamage{Amount: 10}
	if shot.FromFan() {
		t.Fatal("bullet is not a fan")
	}
	if got := shot.KeepFan(0.4); got != 10 {
		t.Fatalf("bullet keep=%v", got)
	}
}
