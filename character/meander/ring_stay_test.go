package 迷走

import (
	"math"
	"math/rand/v2"
	"testing"

	"xqdj/internal/unit"
)

func ringStayActor(seed uint64) *迷走 {
	rings := []ring{{0, 0, 100}, {120, 0, 100}}
	return &迷走{
		ring:        0,
		spin:        1,
		ringReadyAt: 12, // 在第 10 秒进入当前环。
		rng:         rand.New(rand.NewPCG(seed+1, seed+9)),
		maze:        maze{rings: rings, junc: intersections(rings)},
	}
}

func TestBootRequiresTwoSecondsOnFirstRing(t *testing.T) {
	a := &迷走{}
	out := make(chan unit.Cmd, 32)
	ctx := unit.Context{ID: 1, Out: out, Stream: 1}.WithBirths()
	a.Handle(ctx, unit.Sense{Time: 10, Field: unit.HexField()})
	if !a.booted {
		t.Fatal("maze did not boot")
	}
	var j junction
	found := false
	for _, candidate := range a.maze.junc {
		if candidate.a == a.ring || candidate.b == a.ring {
			j, found = candidate, true
			break
		}
	}
	if !found {
		t.Fatal("first ring has no junction")
	}
	for _, now := range []float64{10, 11, 11.999999} {
		if _, _, ok := a.tryJunction(unit.Sense{Time: now}, j.x, j.y); ok {
			t.Fatalf("first ring allowed a junction before two seconds at %v", now)
		}
	}
	if _, _, ok := a.tryJunction(unit.Sense{Time: 12}, j.x, j.y); !ok {
		t.Fatal("first ring should allow a junction at exactly two seconds")
	}
}

func TestJunctionRestartsStayOnlyWhenRingChanges(t *testing.T) {
	stayed, switched := 0, 0
	for seed := uint64(0); seed < 32; seed++ {
		a := ringStayActor(seed)
		if _, _, ok := a.tryJunction(unit.Sense{Time: 12}, 60, 80); !ok {
			t.Fatalf("seed %d: junction should unlock at two seconds", seed)
		}
		changed := a.ring != 0
		if changed {
			switched++
		} else {
			stayed++
		}
		// 另一交点距离足够远，且已过交点冷却；只有换环才应继续锁住。
		for _, now := range []float64{12.5, 13.999999} {
			_, _, ok := a.tryJunction(unit.Sense{Time: now}, 60, -80)
			if changed && ok {
				t.Fatalf("seed %d: switched again before two seconds at %v", seed, now)
			}
			if !changed {
				if !ok {
					t.Fatalf("seed %d: staying on the same ring restarted the timer", seed)
				}
				break
			}
		}
		if changed {
			if _, _, ok := a.tryJunction(unit.Sense{Time: 14}, 60, -80); !ok {
				t.Fatalf("seed %d: new ring should unlock at exactly two seconds", seed)
			}
		}
	}
	if stayed == 0 || switched == 0 {
		t.Fatalf("need both exit outcomes, stayed=%d switched=%d", stayed, switched)
	}
}

func TestRailCorrectionCannotBypassRingStay(t *testing.T) {
	for _, pos := range []struct {
		name string
		x, y float64
	}{
		{name: "near junction", x: 70, y: math.Sqrt(7500)},
		{name: "on other ring", x: 120, y: 100},
		{name: "at current center", x: 0, y: 0},
	} {
		t.Run(pos.name, func(t *testing.T) {
			a := ringStayActor(1)
			out := make(chan unit.Cmd, 4)
			a.rail(unit.Context{ID: 1, Out: out}, unit.Sense{
				Time: 11.999999,
				Self: unit.Snapshot{X: pos.x, Y: pos.y},
			}, pt{})
			if a.ring != 0 {
				t.Fatal("rail correction switched rings before two seconds")
			}
			cmd := <-out
			teleport, ok := cmd.(unit.Teleport)
			if !ok || math.Abs(math.Hypot(teleport.X, teleport.Y)-100) > 1e-6 {
				t.Fatalf("should project back onto the current ring, got %+v", cmd)
			}
		})
	}
}

func TestRailCorrectionStartsNewRingStay(t *testing.T) {
	a := ringStayActor(1)
	out := make(chan unit.Cmd, 4)
	a.rail(unit.Context{ID: 1, Out: out}, unit.Sense{
		Time: 12,
		Self: unit.Snapshot{X: 120, Y: 100},
	}, pt{})
	if a.ring != 1 {
		t.Fatal("rail correction should allow the nearest ring after two seconds")
	}
	if _, _, ok := a.tryJunction(unit.Sense{Time: 13.999999}, 60, 80); ok {
		t.Fatal("rail correction did not restart the two second stay")
	}
	if _, _, ok := a.tryJunction(unit.Sense{Time: 14}, 60, 80); !ok {
		t.Fatal("corrected ring should unlock at exactly two seconds")
	}
}

func TestReverseDoesNotRestartRingStay(t *testing.T) {
	a := ringStayActor(1)
	a.flipSpin(11.5)
	if _, _, ok := a.tryJunction(unit.Sense{Time: 11.999999}, 60, 80); ok {
		t.Fatal("reversing should not shorten the two second stay")
	}
	if _, _, ok := a.tryJunction(unit.Sense{Time: 12}, 60, 80); !ok {
		t.Fatal("reversing on the same ring should not restart the timer")
	}
}
