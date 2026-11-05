package 纵火者

import (
	"fmt"
	"testing"
	"xqdj/internal/unit"
)

// Same footprint as the old disk benchmark; overlap becomes shared coverage.
func denseOil(n int) oilSurface {
	g := oilSurface{}
	for i := 0; i < n; i++ {
		x, y := float64(i%50)*8-196, float64(i/50)*8-156
		g.addOil(x, y, x, y, 18, 0)
	}
	return g
}
func BenchmarkOilFieldIdle(b *testing.B) {
	for _, n := range []int{100, 500, 2000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			f := &油火{owner: 1, surface: denseOil(n)}
			out := make(chan unit.Cmd, 16)
			ctx := fieldCtx(out)
			s := sense(0, unit.Snapshot{ID: 7}, []unit.Snapshot{ownerAt()})
			f.onSense(ctx, s)
			drain(out)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				s.Time += 1.0 / 60
				f.onSense(ctx, s)
				for len(out) > 0 {
					<-out
				}
			}
		})
	}
}
func BenchmarkOilSurfaceBurning(b *testing.B) {
	for _, n := range []int{100, 500, 2000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				g := denseOil(n)
				g.addFire(-196, -156, fireRadius, 0)
				b.StartTimer()
				for frame := 1; frame <= 120; frame++ {
					g.advance(float64(frame) / 60)
					g.payload()
				}
			}
		})
	}
}
