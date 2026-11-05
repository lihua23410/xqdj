package 中转站

import (
	"math"
	"testing"

	"xqdj/internal/unit"
	_ "xqdj/map/hex"
)

func TestFirstLeaveHasNoShot(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 8)
	a := &中转站{}
	a.Handle(ctx(matchState, out), senseAt(0, 0, 1.0/60))
	cmds := drain(out)
	sp := onlySpawn(t, cmds, KindRelayNode)
	if sp.VX != 0 || sp.VY != 0 {
		t.Fatalf("first node spawn=%+v", sp)
	}
	if hasKind(cmds, KindRelayShot) {
		t.Fatal("stacked on the fighter should not shoot yet")
	}
}

func TestSecondLeaveDoesNotEncodeHop(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 8)
	a := &中转站{}
	c := ctx(matchState, out)
	a.Handle(c, senseAt(0, 0, 0))
	drain(out)
	a.Handle(c, senseAt(80, 0, dropGap))
	sp := onlySpawn(t, drain(out), KindRelayNode)
	if sp.VX != 0 || sp.VY != 0 || sp.X != 80 {
		t.Fatalf("spawn=%+v", sp)
	}
}

func TestDropWaitsTwoSeconds(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 8)
	a := &中转站{}
	c := ctx(matchState, out)
	a.Handle(c, senseAt(0, 0, 0))
	drain(out)
	a.Handle(c, senseAt(10, 0, dropGap-0.01))
	if len(drain(out)) != 0 {
		t.Fatal("early drop")
	}
	a.Handle(c, senseAt(10, 0, dropGap))
	onlySpawn(t, drain(out), KindRelayNode)
}

func TestOpeningShotPicksNearest(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 8)
	n := &随从{owner: 1, slot: 0}
	self := nodeSnap(4, 0, 0)
	fighter := fighterSnap(1, 40, 0)
	far := nodeSnap(9, 200, 0)
	n.Handle(ctx(matchState, out), unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{fighter, far}})
	sp := onlySpawn(t, drain(out), KindRelayShot)
	if sp.OwnerID != 1 || sp.VX <= 0 {
		t.Fatalf("spawn=%+v", sp)
	}
}

func TestSkipsSamePointAndSender(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 8)
	n := bootedNode(matchState, out)
	self := nodeSnap(4, 0, 0)
	sender := nodeSnap(9, 20, 0)
	next := nodeSnap(8, 90, 0)
	close := nodeSnap(7, 0.4, 0)
	shot := aimed(matchState, 20, 0, 0, 9, self.ID, nil)
	n.Handle(ctx(matchState, out), unit.Sense{
		Time: 2, Self: self, Nearby: []unit.Snapshot{sender, next, close, shot},
	})
	sp := onlySpawn(t, drain(out), KindRelayShot)
	if sp.OwnerID != 8 || sp.VX <= 0 {
		t.Fatalf("spawn=%+v want node 8", sp)
	}
}

func TestOnlySenderAbsorbs(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 8)
	n := bootedNode(matchState, out)
	self := nodeSnap(4, 0, 0)
	sender := nodeSnap(9, 50, 0)
	shot := aimed(matchState, 20, 0, 0, 9, self.ID, nil)
	n.Handle(ctx(matchState, out), unit.Sense{
		Time: 2, Self: self, Nearby: []unit.Snapshot{sender, shot},
	})
	cmds := drain(out)
	if _, ok := cmds[0].(unit.Despawn); !ok || hasKind(cmds, KindRelayShot) {
		t.Fatalf("cmds=%v", cmds)
	}
}

func TestIgnoresShotAimedElsewhere(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 8)
	n := bootedNode(matchState, out)
	shot := shotSnap(20, 0, 0)
	shot.OwnerID = 99
	n.Handle(ctx(matchState, out), unit.Sense{
		Time: 2, Self: nodeSnap(4, 0, 0), Nearby: []unit.Snapshot{shot},
	})
	if len(drain(out)) != 0 {
		t.Fatal("other target's shot should pass")
	}
}

func TestTiePrefersFighterThenEarlier(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 8)
	n := &随从{owner: 1, slot: 0}
	self := nodeSnap(4, 0, 0)
	fighter := fighterSnap(1, 100, 0)
	node := nodeSnap(3, -100, 0)
	n.Handle(ctx(matchState, out), unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{fighter, node}})
	sp := onlySpawn(t, drain(out), KindRelayShot)
	if sp.OwnerID != 1 {
		t.Fatalf("tie should prefer fighter, got %d", sp.OwnerID)
	}

	out = make(chan unit.Cmd, 8)
	n = &随从{owner: 1, slot: 0}
	older := nodeSnap(3, 80, 0)
	newer := nodeSnap(9, -80, 0)
	far := fighterSnap(1, 0, 500)
	n.Handle(ctx(matchState, out), unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{older, newer, far}})
	sp = onlySpawn(t, drain(out), KindRelayShot)
	if sp.OwnerID != 3 {
		t.Fatalf("tie should prefer earlier node, got %d", sp.OwnerID)
	}
}

func TestFighterRelaysPastSender(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 8)
	a := &中转站{booted: true, nextDrop: 100, slot: 0}
	self := fighterSnap(1, 0, 0)
	sender := nodeSnap(5, -40, 0)
	other := nodeSnap(6, 70, 0)
	shot := aimed(matchState, 20, 0, 0, 5, self.ID, nil)
	a.Handle(ctx(matchState, out), unit.Sense{
		Time: 1, Self: self, Nearby: []unit.Snapshot{sender, other, shot},
	})
	sp := onlySpawn(t, drain(out), KindRelayShot)
	if sp.OwnerID != 6 || sp.VX <= 0 {
		t.Fatalf("spawn=%+v", sp)
	}
}

func TestTriangleCanPassBackToEarlierHand(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 8)
	n := bootedNode(matchState, out)
	self := nodeSnap(4, 0, 0)       // C
	sender := nodeSnap(8, 30, 0)    // B, just passed
	earlier := nodeSnap(3, -10, 40) // A
	shot := aimed(matchState, 20, 0, 0, 8, self.ID, nil)
	n.Handle(ctx(matchState, out), unit.Sense{
		Time: 2, Self: self, Nearby: []unit.Snapshot{sender, earlier, shot},
	})
	sp := onlySpawn(t, drain(out), KindRelayShot)
	if sp.OwnerID != 3 {
		t.Fatalf("should pass to A, got %d", sp.OwnerID)
	}
}

func TestPreferNeverReceivedOverNear(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 8)
	n := bootedNode(matchState, out)
	self := nodeSnap(4, 0, 0)
	near := nodeSnap(8, 25, 0)
	far := nodeSnap(9, 180, 0)
	sender := nodeSnap(5, -40, 0)
	tr := newTrack()
	tr.mark(5)
	tr.mark(8)
	tr.mark(4)
	shot := aimed(matchState, 20, 0, 0, 5, self.ID, tr)
	n.Handle(ctx(matchState, out), unit.Sense{
		Time: 2, Self: self, Nearby: []unit.Snapshot{near, far, sender, shot},
	})
	sp := onlySpawn(t, drain(out), KindRelayShot)
	if sp.OwnerID != 9 {
		t.Fatalf("never-received far should win, got %d", sp.OwnerID)
	}
}

func TestPreferOldestReceived(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 8)
	n := bootedNode(matchState, out)
	self := nodeSnap(4, 0, 0)
	near := nodeSnap(8, 25, 0)
	far := nodeSnap(9, 180, 0)
	sender := nodeSnap(5, -40, 0)
	tr := newTrack()
	tr.mark(9) // oldest
	tr.mark(8)
	tr.mark(5)
	tr.mark(4)
	shot := aimed(matchState, 20, 0, 0, 5, self.ID, tr)
	n.Handle(ctx(matchState, out), unit.Sense{
		Time: 2, Self: self, Nearby: []unit.Snapshot{near, far, sender, shot},
	})
	sp := onlySpawn(t, drain(out), KindRelayShot)
	if sp.OwnerID != 9 {
		t.Fatalf("oldest received should win, got %d", sp.OwnerID)
	}
}

func TestShotHomesOnFighter(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 4)
	b := &中转弹{slot: 0, target: 1}
	body := fighterSnap(1, 40, 30)
	b.Handle(ctx(matchState, out), unit.Sense{
		Self:   shotSnap(7, 0, 0),
		Nearby: []unit.Snapshot{body},
	})
	v := lastVel(drain(out), 0)
	if v == nil {
		t.Fatal("missing steer")
	}
	want := math.Atan2(30, 40)
	got := math.Atan2(v.VY, v.VX)
	if math.Abs(want-got) > 1e-6 {
		t.Fatalf("ang=%v want %v", got, want)
	}
	if math.Abs(math.Hypot(v.VX, v.VY)-shotSpeed) > 1e-6 {
		t.Fatalf("speed=%v", math.Hypot(v.VX, v.VY))
	}
}

func TestShotDoesNotHomeOnNode(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 4)
	b := &中转弹{slot: 0, target: 9}
	b.Handle(ctx(matchState, out), unit.Sense{
		Self:   shotSnap(7, 0, 0),
		Nearby: []unit.Snapshot{nodeSnap(9, 40, 0)},
	})
	if lastVel(drain(out), 0) != nil {
		t.Fatal("node-bound shot should not steer")
	}
}

func TestShotDamagesWithoutDespawn(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 4)
	b := &中转弹{slot: 0}
	c := ctx(matchState, out)
	foe := unit.Snapshot{ID: 3, Role: unit.RoleFighter, Slot: 1, X: 10, Y: 0, Radius: 18}
	b.Handle(c, unit.Collision{Time: 1, Other: foe})
	cmds := drain(out)
	if len(cmds) != 2 {
		t.Fatalf("cmds=%v", cmds)
	}
	d, ok := cmds[0].(unit.Damage)
	if !ok || d.Amount != shotDamage || d.To != 3 {
		t.Fatalf("damage=%v", cmds[0])
	}
	fx, ok := cmds[1].(unit.FX)
	if !ok || fx.Name != "relay-hit" || fx.X != foe.X || fx.Y != foe.Y || fx.Kind != KindRelay {
		t.Fatalf("hit FX=%v", cmds[1])
	}
	b.Handle(c, unit.Collision{Time: 1.05, Other: foe})
	if len(drain(out)) != 0 {
		t.Fatal("inside 0.1s")
	}
	b.Handle(c, unit.Collision{Time: 1.1, Other: foe})
	if len(drain(out)) != 2 {
		t.Fatal("after 0.1s")
	}
}

func TestRoutingPresentationTracksRealNextHop(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 16)
	n := &随从{owner: 1, slot: 0}
	self := nodeSnap(4, 0, 0)
	near := fighterSnap(1, 40, 0)
	c := ctx(matchState, out)
	c.ID = self.ID
	n.Handle(c, unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{near, nodeSnap(9, 200, 0)}})
	cmds := drain(out)
	sp := onlySpawn(t, cmds, KindRelayShot)
	join, forward := 0, 0
	for _, cmd := range cmds {
		if fx, ok := cmd.(unit.FX); ok {
			switch fx.Name {
			case "relay-join":
				join++
				if fx.UnitID != self.ID || fx.Kind != KindRelay {
					t.Fatalf("neighbor FX=%+v", fx)
				}
			case "relay-forward":
				forward++
				if fx.UnitID != self.ID || fx.Amount != float64(sp.OwnerID) || fx.VX != near.X || fx.VY != near.Y {
					t.Fatalf("route FX must follow chosen spawn: %+v vs %+v", fx, sp)
				}
			}
		}
	}
	if join != 1 || forward != 1 {
		t.Fatalf("join=%d forward=%d", join, forward)
	}
	n.Handle(c, unit.Sense{Time: 1.1, Self: self, Nearby: []unit.Snapshot{near}})
	if len(drain(out)) != 0 {
		t.Fatal("idle neighbor must not repeat cues")
	}
	shot := aimed(matchState, 20, self.X, self.Y, near.ID, self.ID, nil)
	n.Handle(c, unit.Sense{Time: 2, Self: self, Nearby: []unit.Snapshot{near, shot}})
	cmds = drain(out)
	if len(cmds) != 2 {
		t.Fatalf("absorb with no next hop: %v", cmds)
	}
	if fx, ok := cmds[1].(unit.FX); !ok || fx.Name != "relay-arrive" || fx.UnitID != self.ID {
		t.Fatalf("arrival FX=%v", cmds[1])
	}
}

func TestShotIgnoresFriendAndWall(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 4)
	b := &中转弹{slot: 0}
	c := ctx(matchState, out)
	friend := unit.Snapshot{ID: 8, Role: unit.RoleFighter, Slot: 0, Radius: 18}
	b.Handle(c, unit.Collision{Time: 1, Other: friend})
	b.Handle(c, unit.WallHit{Time: 1})
	cmds := drain(out)
	if !lastDespawn(cmds) {
		t.Fatal("同边战斗机该打掉这发弹")
	}
	for _, cmd := range cmds {
		if _, ok := cmd.(unit.Damage); ok {
			t.Fatal("同边不该出伤")
		}
	}
}

func TestCapBlocksOpening(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	setLive(matchState, 0, shotCap)
	out := make(chan unit.Cmd, 8)
	n := &随从{owner: 1, slot: 0}
	self := nodeSnap(4, 0, 0)
	fighter := fighterSnap(1, 40, 0)
	n.Handle(ctx(matchState, out), unit.Sense{Time: 1, Self: self, Nearby: []unit.Snapshot{fighter}})
	if hasKind(drain(out), KindRelayShot) {
		t.Fatal("full cap should not open")
	}
}

func TestExistingMinionDoesNotOpenAgain(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 8)
	n := bootedNode(matchState, out)
	drain(out)
	self := nodeSnap(4, 0, 0)
	fighter := fighterSnap(1, 80, 0)
	n.Handle(ctx(matchState, out), unit.Sense{Time: 2, Self: self, Nearby: []unit.Snapshot{fighter}})
	if hasKind(drain(out), KindRelayShot) {
		t.Fatal("already-opened minion should not fire extra shots")
	}
}

func TestCatchRelaysAtCap(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 8)
	n := bootedNode(matchState, out)
	drain(out)
	setLive(matchState, 0, shotCap)
	self := nodeSnap(4, 0, 0)
	next := nodeSnap(8, 90, 0)
	sender := nodeSnap(9, -40, 0)
	shot := aimed(matchState, 20, 0, 0, 9, self.ID, nil)
	n.Handle(ctx(matchState, out), unit.Sense{
		Time: 2, Self: self, Nearby: []unit.Snapshot{next, sender, shot},
	})
	if !hasKind(drain(out), KindRelayShot) {
		t.Fatal("relay replaces a live shot")
	}
}

func TestShotLeavesOutline(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 2)
	b := &中转弹{slot: 0}
	c := ctx(matchState, out)
	b.Handle(c, unit.Sense{Self: unit.Snapshot{X: 100, Y: 0}})
	if len(drain(out)) != 0 {
		t.Fatal("inside stays")
	}
	b.Handle(c, unit.Sense{Self: unit.Snapshot{X: 0, Y: -640}})
	if _, ok := drain(out)[0].(unit.Despawn); !ok {
		t.Fatal("outside despawns")
	}
}

func TestAcceptsHit(t *testing.T) {
	matchState := &unit.MatchState{}

	resetPass(matchState)
	out := make(chan unit.Cmd, 2)
	(&中转站{}).Handle(ctx(matchState, out), unit.IncomingDamage{Token: 3, Amount: 5})
	if _, ok := drain(out)[0].(unit.ConfirmDamage); !ok {
		t.Fatal("should confirm")
	}
}

func ctx(matchState *unit.MatchState, out chan unit.Cmd) unit.Context {
	return unit.Context{State: matchState, ID: 1, Kind: KindRelay, Out: out}
}

func senseAt(x, y, time float64) unit.Sense {
	return unit.Sense{Time: time, Self: fighterSnap(1, x, y)}
}

func fighterSnap(id uint64, x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: id, Kind: KindRelay, Role: unit.RoleFighter, Slot: 0,
		X: x, Y: y, Radius: bodyRadius,
	}
}

func nodeSnap(id uint64, x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: id, Kind: KindRelayNode, Role: unit.RoleHelper, Slot: 0,
		X: x, Y: y, Radius: nodeRadius, OwnerID: 1,
	}
}

func shotSnap(id uint64, x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: id, Kind: KindRelayShot, Role: unit.RoleProjectile, Slot: 0,
		X: x, Y: y, Radius: shotRadius,
	}
}

func aimed(matchState *unit.MatchState, id uint64, x, y float64, sender, target uint64, tr *hopTrack) unit.Snapshot {
	s := shotSnap(id, x, y)
	s.OwnerID = target
	if tr == nil {
		tr = newTrack()
	}
	queuePass(matchState, passInfo{sender: sender, target: target, x: x, y: y, track: tr})
	return s
}

func bootedNode(matchState *unit.MatchState, out chan unit.Cmd) *随从 {
	n := &随从{owner: 1, slot: 0}
	n.Handle(ctx(matchState, out), unit.Sense{Time: 0, Self: nodeSnap(4, 0, 0)})
	drain(out)
	return n
}

func drain(out <-chan unit.Cmd) []unit.Cmd {
	var cmds []unit.Cmd
	for {
		select {
		case c := <-out:
			cmds = append(cmds, c)
		default:
			return cmds
		}
	}
}

func onlySpawn(t *testing.T, cmds []unit.Cmd, kind string) unit.Spawn {
	t.Helper()
	var sp *unit.Spawn
	for _, c := range cmds {
		v, ok := c.(unit.Spawn)
		if !ok || v.Kind != kind {
			continue
		}
		if sp != nil {
			t.Fatalf("two %s", kind)
		}
		cp := v
		sp = &cp
	}
	if sp == nil {
		t.Fatalf("missing %s in %v", kind, cmds)
	}
	return *sp
}

func lastDespawn(cmds []unit.Cmd) bool {
	for _, c := range cmds {
		if _, ok := c.(unit.Despawn); ok {
			return true
		}
	}
	return false
}

func lastVel(cmds []unit.Cmd, id uint64) *unit.SetVelocity {
	var v *unit.SetVelocity
	for _, c := range cmds {
		sv, ok := c.(unit.SetVelocity)
		if !ok {
			continue
		}
		if id != 0 && sv.UnitID != id {
			continue
		}
		cp := sv
		v = &cp
	}
	return v
}

func hasKind(cmds []unit.Cmd, kind string) bool {
	for _, c := range cmds {
		if v, ok := c.(unit.Spawn); ok && v.Kind == kind {
			return true
		}
	}
	return false
}
