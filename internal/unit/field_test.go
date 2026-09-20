package unit

import (
	"math/rand/v2"
	"testing"
)

func TestHexWalkableCenter(t *testing.T) {
	SetLiveField(HexField())
	if !HexContains(0, 0, 18) {
		t.Fatal("hex center should be walkable")
	}
}

func TestCircleHardWallBlocksCenter(t *testing.T) {
	f := CircleField()
	if f.Walkable(0, 0, 18) {
		t.Fatal("circle 场心硬墙 should block")
	}
	if !f.Walkable(0, 80, 18) {
		t.Fatal("above the bar should be walkable")
	}
}

func TestCircleRandomWalkableAvoidsWall(t *testing.T) {
	f := CircleField()
	rng := rand.New(rand.NewPCG(1, 2))
	for i := 0; i < 20; i++ {
		x, y, ok := f.RandomWalkable(rng, 18)
		if !ok {
			t.Fatal("expected a walkable point")
		}
		if !f.Walkable(x, y, 18) {
			t.Fatalf("(%v,%v) not walkable", x, y)
		}
	}
}
