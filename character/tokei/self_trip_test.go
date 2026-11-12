package 时计

import (
	"math"
	"testing"

	"xqdj/internal/unit"
)

func selfClock(moments ...moment) *时计 {
	return &时计{
		booted: true, hp: 100, nextAt: 100, hand: 4, slot: 0,
		carved:  map[int]bool{2: true, 4: true},
		moments: moments,
	}
}

func foeAt(x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: 9, Slot: 1, Role: unit.RoleFighter, Radius: 18,
		X: x, Y: y, AimPriority: unit.DefaultFighterAim,
	}
}

func selfSense(now, x, y float64, nearby ...unit.Snapshot) unit.Sense {
	return unit.Sense{
		Time:  now,
		Field: unit.HexField(),
		Self: unit.Snapshot{
			ID: 1, Slot: 0, Role: unit.RoleFighter, Radius: tokeiRadius,
			X: x, Y: y, HP: 100, MaxHP: 100,
		},
		Nearby: nearby,
	}
}

func hasTeleport(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if _, ok := c.(unit.Teleport); ok {
			return true
		}
	}
	return false
}

func tripAt(cmds []unit.Cmd) (unit.FX, bool) {
	for _, c := range cmds {
		fx, ok := c.(unit.FX)
		if ok && fx.Name == "trip" {
			return fx, true
		}
	}
	return unit.FX{}, false
}

func TestSelfStepSkipsOwnMomentAndRamsNearest(t *testing.T) {
	// 自己踩在离敌人最近的 A 上，出发档必须是更远的 B。
	a := selfClock(
		moment{hour: 2, x: 0, y: 0},
		moment{hour: 4, x: 160, y: 0},
	)
	clockCommands(a, selfSense(1, 80, 0, foeAt(0, 55)))
	cmds := clockCommands(a, selfSense(1.1, 20, 0, foeAt(0, 55)))
	fx, ok := tripAt(cmds)
	if !ok || fx.X != 160 || fx.Y != 0 {
		t.Fatalf("should ram from the other moment, fx=%+v ok=%v", fx, ok)
	}
	var tp unit.Teleport
	var vel unit.SetVelocity
	for _, c := range cmds {
		if v, ok := c.(unit.Teleport); ok {
			tp = v
		}
		if v, ok := c.(unit.SetVelocity); ok {
			vel = v
		}
	}
	if tp.X != 160 || tp.Y != 0 {
		t.Fatalf("blinked away from the moment: %+v", tp)
	}
	if vel.VX >= 0 {
		t.Fatalf("charge should leave the moment toward the enemy, vel=%+v", vel)
	}
	if a.moments[0].silent != 1.1+tripSilence || a.moments[1].silent != 1.1+tripSilence {
		t.Fatalf("both moments should cool: %+v", a.moments)
	}
	if a.arcUntil != 1.1+arcWindow {
		t.Fatalf("arc window = %v", a.arcUntil)
	}
}

func TestSelfStepSkipsCoolingMoment(t *testing.T) {
	a := selfClock(
		moment{hour: 2, x: 0, y: 0},
		moment{hour: 4, x: 0, y: 50, silent: 10},
		moment{hour: 6, x: 180, y: 0},
	)
	clockCommands(a, selfSense(1, 80, 0, foeAt(0, 120)))
	cmds := clockCommands(a, selfSense(1.1, 20, 0, foeAt(0, 120)))
	fx, ok := tripAt(cmds)
	if !ok || math.Hypot(fx.X-180, fx.Y) > 1e-6 {
		t.Fatalf("cooling moment joined the election: %+v", fx)
	}
	if a.moments[1].silent != 10 || a.moments[0].silent != 1.1+tripSilence || a.moments[2].silent != 1.1+tripSilence {
		t.Fatalf("silence changed wrong moments: %+v", a.moments)
	}
}

func TestSelfStepDoesNothingWithoutAnotherMoment(t *testing.T) {
	a := selfClock(moment{hour: 2, x: 0, y: 0})
	clockCommands(a, selfSense(1, 80, 0, foeAt(0, 55)))
	cmds := clockCommands(a, selfSense(1.1, 20, 0, foeAt(0, 55)))
	if hasTeleport(cmds) || a.moments[0].silent != 0 || a.arcUntil != 0 {
		t.Fatalf("lone moment entered the self ram: silent=%v arc=%v cmds=%v", a.moments[0].silent, a.arcUntil, cmds)
	}
}

func TestSelfStepAllOthersCoolingSkipsTheFlow(t *testing.T) {
	a := selfClock(
		moment{hour: 2, x: 0, y: 0},
		moment{hour: 4, x: 0, y: 50, silent: 10},
	)
	clockCommands(a, selfSense(1, 80, 0, foeAt(0, 120)))
	cmds := clockCommands(a, selfSense(1.1, 20, 0, foeAt(0, 120)))
	if _, ok := tripAt(cmds); ok || hasTeleport(cmds) || a.arcUntil != 0 || a.moments[0].silent != 0 || a.moments[1].silent != 10 {
		t.Fatalf("no electable moment still entered the flow: %+v cmds=%v", a.moments, cmds)
	}
}

func TestSelfStepWithoutEnemySkipsTheFlow(t *testing.T) {
	a := selfClock(
		moment{hour: 2, x: 0, y: 0},
		moment{hour: 4, x: 160, y: 0},
	)
	clockCommands(a, selfSense(1, 80, 0))
	cmds := clockCommands(a, selfSense(1.1, 20, 0))
	if _, ok := tripAt(cmds); ok || hasTeleport(cmds) || a.arcUntil != 0 || a.moments[0].silent != 0 || a.moments[1].silent != 0 {
		t.Fatalf("no target still entered the flow: %+v cmds=%v", a.moments, cmds)
	}
}

func TestSelfStepStandingDoesNotRepeat(t *testing.T) {
	a := selfClock(
		moment{hour: 2, x: 0, y: 0},
		moment{hour: 4, x: 160, y: 0},
	)
	clockCommands(a, selfSense(1, 80, 0, foeAt(0, 55)))
	clockCommands(a, selfSense(1.1, 20, 0, foeAt(0, 55)))
	cmds := clockCommands(a, selfSense(1.2, 10, 0, foeAt(0, 55)))
	if hasTeleport(cmds) {
		t.Fatal("standing inside retriggered")
	}
}

func TestSelfStepOnCoolingMomentDoesNotFire(t *testing.T) {
	a := selfClock(
		moment{hour: 2, x: 0, y: 0, silent: 10},
		moment{hour: 4, x: 160, y: 0},
	)
	clockCommands(a, selfSense(1, 80, 0, foeAt(220, 0)))
	cmds := clockCommands(a, selfSense(1.1, 20, 0, foeAt(220, 0)))
	if hasTeleport(cmds) || a.moments[1].silent != 0 {
		t.Fatal("stepping on a cooling moment still launched")
	}
}

func TestCarveUnderfootIsNotASelfStep(t *testing.T) {
	a := &时计{
		booted: true, hp: 100, nextAt: 0, hand: 0,
		carved: map[int]bool{},
		moments: []moment{
			{hour: 2, x: 150, y: 0, pin: 7},
		},
	}
	cmds := clockCommands(a, selfSense(1, 0, 0, foeAt(0, 80), unit.Snapshot{
		ID: 7, Kind: KindTokeiMoment, OwnerID: 1, X: 150, Y: 0,
	}))
	if hasTeleport(cmds) || a.moments[0].silent != 0 {
		t.Fatalf("carving underfoot launched: silent=%v cmds=%v", a.moments[0].silent, cmds)
	}
	if len(a.moments) != 2 || !a.moments[1].selfIn {
		t.Fatalf("new moment = %+v", a.moments)
	}
}

func TestOpeningMomentIsBottomOfTheStack(t *testing.T) {
	a := &时计{hp: 100}
	s := unit.Sense{
		Time: 0, Field: unit.HexField(),
		Self: unit.Snapshot{
			ID: 1, Slot: 0, Role: unit.RoleFighter, Radius: tokeiRadius,
			X: 12, Y: -4, HP: 100, MaxHP: 100,
		},
	}
	cmds := clockCommands(a, s)
	if !a.booted || a.hand != 0 || a.nextAt != saveGap || len(a.moments) != 1 {
		t.Fatalf("boot hand=%d next=%v moments=%d", a.hand, a.nextAt, len(a.moments))
	}
	open := a.moments[0]
	if open.hour != 0 || !open.selfIn || open.x != 12 || open.y != -4 || a.carved[2] {
		t.Fatalf("opening = %+v carved=%v", open, a.carved)
	}
	born := 0
	for _, c := range cmds {
		sp, ok := c.(unit.Spawn)
		if ok && sp.Kind == KindTokeiMoment && sp.X == 12 && sp.Y == -4 {
			born++
		}
	}
	if born != 1 {
		t.Fatalf("born %d opening moments", born)
	}
	clockCommands(a, s)
	if len(a.moments) != 1 {
		t.Fatalf("second sense added a moment: %d", len(a.moments))
	}
	a.nextAt = 8
	s.Time = 8
	s.Self.X, s.Self.Y, s.Self.HP = 40, 10, 80
	clockCommands(a, s)
	if len(a.moments) != 2 || a.moments[0].hour != 0 || a.moments[1].hour != 2 || a.hand != 2 {
		t.Fatalf("stack hours=%d,%d hand=%d", a.moments[0].hour, a.moments[1].hour, a.hand)
	}
	a.hp = 80
	clockCommands(a, unit.IncomingDamage{Time: 8, Amount: 80})
	if len(a.moments) != 1 || a.moments[0].hour != 0 || a.hand != 0 {
		t.Fatalf("rewind ate the opening moment: %+v hand=%d", a.moments, a.hand)
	}
}
