package 迷走

import (
	"math"
	"testing"

	"xqdj/internal/unit"
)

func TestMeanderKeepsAdvancingAroundDriftingRing(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		midVX, midVY, driftVX float64
	}{
		{name: "stationary ring"},
		{name: "ring moves with tangent", midVX: 120, midVY: 90, driftVX: 50},
		{name: "ring moves against tangent", midVX: -120, midVY: 90, driftVX: -50},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &迷走{
				booted: true,
				maze:   maze{rings: []ring{{0, 0, 100}}},
				ring:   0, spin: -1, ringReadyAt: 10,
				drifts: []ringDrift{{vx: tc.driftVX, nextTurn: 10}},
				mid:    midPoint{vx: tc.midVX, vy: tc.midVY},
				lastT:  1,
			}
			out := make(chan unit.Cmd, 16)
			ctx := unit.Context{ID: 1, Out: out, Stream: 1}
			s := unit.Sense{
				Time: 1, Field: unit.HexField(),
				Self: unit.Snapshot{X: 0, Y: 100, VX: 170},
			}
			const dt = 1.0 / 60
			start := math.Pi / 2
			for frame := 0; frame < 20; frame++ {
				s.Time += dt
				s.Self.X += s.Self.VX * dt
				s.Self.Y += s.Self.VY * dt
				a.Handle(ctx, s)
				for len(out) > 0 {
					switch c := (<-out).(type) {
					case unit.Teleport:
						s.Self.X, s.Self.Y = c.X, c.Y
					case unit.SetVelocity:
						s.Self.VX, s.Self.VY = c.VX, c.VY
					}
				}
				if math.Abs(math.Hypot(s.Self.VX, s.Self.VY)-cruise) > 1e-6 {
					t.Fatalf("tangent velocity changed at frame %d: %+v", frame, s.Self)
				}
			}
			r := a.maze.rings[a.ring]
			angle := math.Atan2(s.Self.Y-r.y, s.Self.X-r.x)
			travel := math.Abs(angle-start) * r.r
			want := cruise * 20 * dt
			if math.Abs(travel-want) > want*0.02 {
				t.Fatalf("only advanced %.3f along moving ring, want %.3f; velocity magnitude=%.1f", travel, want, math.Hypot(s.Self.VX, s.Self.VY))
			}
		})
	}
}
