package ospf

import (
	"testing"

	"xqdj/internal/unit"
)

// 报文一帧的位移可能直接冲过折点，落点本来就在折线外侧。折点判定排在
// 偏离检查之前，这样冲过折点的报文会被拉回落点再转向，而不是被误判成
// 偏离网线（旧顺序在这里会 panic）。
func TestPacketOvershootsBendWithoutTrippingCableGate(t *testing.T) {
	n := &network{owner: 1}
	to := addRouter(n, 0, point{200, 0})
	to.id = 2
	p := &packet{flight: flight{
		network: n,
		target:  to,
		path:    []point{{0, 100}},
		cable:   []point{{0, 0}, {0, 100}, {200, 0}},
	}}
	out := make(chan unit.Cmd, 8)
	ctx := unit.Context{ID: 7, Out: out}

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("冲过折点被误判成偏离网线: %v", r)
			}
		}()
		p.Handle(ctx, unit.Sense{Self: unit.Snapshot{X: 0, Y: 110}, Time: 1})
	}()

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

// Sense 不一定每 tick 都来，一帧可能走好几个 tick 的距离。容差要按实际
// 步长算，否则直线飞行也会被误判。
func TestPacketLargeStepOnStraightCablePassesGate(t *testing.T) {
	n := &network{owner: 1}
	to := addRouter(n, 0, point{0, 200})
	to.id = 2
	p := &packet{flight: flight{
		network: n,
		target:  to,
		cable:   []point{{0, 0}, {0, 200}},
	}}
	p.lastTime = 0.5
	out := make(chan unit.Cmd, 8)
	ctx := unit.Context{ID: 7, Out: out}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("大步长直线飞行被误判成偏离网线: %v", r)
		}
	}()
	// dt = 0.1 秒，这一帧走 26 单位，仍然贴着网线。
	p.Handle(ctx, unit.Sense{Self: unit.Snapshot{X: 0, Y: 26}, Time: 0.6})
}

// 真正偏离的情况必须照旧 panic：守门不能因为上面两个放行而失效。
func TestPacketOffCableStillPanics(t *testing.T) {
	n := &network{owner: 1}
	to := addRouter(n, 0, point{0, 200})
	to.id = 2
	p := &packet{flight: flight{
		network: n,
		target:  to,
		cable:   []point{{0, 0}, {0, 200}},
	}}
	p.lastTime = 0.5
	out := make(chan unit.Cmd, 8)
	ctx := unit.Context{ID: 7, Out: out}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("偏离网线却没有 panic")
		}
	}()
	p.Handle(ctx, unit.Sense{Self: unit.Snapshot{X: 60, Y: 26}, Time: 0.6})
}
