package 圆

import (
	"math"
	"math/rand/v2"
	"testing"

	"xqdj/internal/unit"
)

func TestCircleHardWallBlocksCenter(t *testing.T) {
	f := Field()
	if f.Walkable(0, 0, 18) {
		t.Fatal("circle 场心硬墙 should block")
	}
	if !f.Walkable(0, 80, 18) {
		t.Fatal("above the bar should be walkable")
	}
}

func TestCircleRandomWalkableAvoidsWall(t *testing.T) {
	f := Field()
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

func TestCircleSweepCatchesInwardBiasedGraze(t *testing.T) {
	o := newCircleOutline(unit.HexRadius)
	const radius = 18.0
	limit := unit.HexRadius - radius
	px, py := limit-0.2, 0.0
	vx, vy := -40.0, 3000.0
	dt := 1.0 / 60
	if px*vx+py*vy > 1e-9 {
		t.Fatal("fixture should be biased inward")
	}
	ex, ey := px+vx*dt, py+vy*dt
	if math.Hypot(ex, ey) <= limit {
		t.Fatalf("fixture should leave the circle, end radius %v", math.Hypot(ex, ey))
	}
	tt, nx, ny, hit := o.Sweep(px, py, vx, vy, 1, 0, radius, dt, false)
	if !hit {
		t.Fatal("fast graze should hit the outline")
	}
	if tt < 0 || tt > dt {
		t.Fatalf("hit time %v outside [0,%v]", tt, dt)
	}
	ax, ay := px+vx*tt, py+vy*tt
	if math.Abs(math.Hypot(ax, ay)-limit) > 1e-4 {
		t.Fatalf("contact (%v,%v) not on the rim %v", ax, ay, limit)
	}
	if math.Abs(nx-ax/limit) > 1e-6 || math.Abs(ny-ay/limit) > 1e-6 {
		t.Fatalf("normal (%v,%v) does not match contact", nx, ny)
	}
}

func TestCircleSweepIgnoresSlowInwardMove(t *testing.T) {
	o := newCircleOutline(unit.HexRadius)
	if _, _, _, hit := o.Sweep(100, 0, -20, 0, 1, 0, 18, 1.0/60, false); hit {
		t.Fatal("slow move toward the center stays inside this tick")
	}
}

func TestRegistered(t *testing.T) {
	f, ok := unit.LookupField(Name)
	if !ok || f.Shape != unit.ShapeCircle || len(f.Walls) != 1 {
		t.Fatalf("field=%+v ok=%v", f, ok)
	}
}
