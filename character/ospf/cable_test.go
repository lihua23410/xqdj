package ospf

import (
	"testing"

	"xqdj/internal/unit"
)

// 报文一帧的位移可能直接冲过折点。折点判定把冲过去的报文拉回落点再转向。
func TestPacketOvershootsBendWithoutTrippingCableGate(t *testing.T) {
	n := &network{owner: 1}
	to := addRouter(n, 0, point{200, 0})
	to.id = 2
	p := &packet{flight: flight{
		network: n,
		target:  to,
		path:    []point{{0, 100}},
	}}
	out := make(chan unit.Cmd, 8)
	ctx := unit.Context{ID: 7, Out: out}

	p.Handle(ctx, unit.Sense{Self: unit.Snapshot{X: 0, Y: 110}, Time: 1})

	var teleported bool
	for len(out) > 0 {
		if _, ok := (<-out).(unit.Teleport); ok {
			teleported = true
		}
	}
	if !teleported {
		t.Fatal("报文没有落到折点上")
	}
	if len(p.path) != 0 {
		t.Fatalf("折点没被吃掉: %v", p.path)
	}
}
