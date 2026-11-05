package ospf

import (
	"math"
	"reflect"
	"testing"

	"xqdj/internal/unit"
	奇美拉 "xqdj/map/chimera"
)

func TestAreaCutBoundariesAndMatchSharing(t *testing.T) {
	info := unit.SpawnInfo{State: &unit.MatchState{}, MatchSeed: 17, Field: unit.HexField()}
	cut := areasFor(info)
	info.Stream = 99
	if !reflect.DeepEqual(cut, areasFor(info)) || !reflect.DeepEqual(cut, cutAreas(info.Field, 17)) {
		t.Fatal("area cut changed with fighter stream")
	}
	radius := unit.HexRadius * math.Sqrt(3) / 4
	if got := cut.at(radius, 0); got != 0 {
		t.Fatalf("circle boundary is area%d", got)
	}
	for i, area := range cut.areas {
		ray := cut.angle + float64(i)*2*math.Pi/3
		for _, angle := range []float64{ray, ray + 0.01} {
			if got := cut.at(2*radius*math.Cos(angle), 2*radius*math.Sin(angle)); got != area+1 {
				t.Fatalf("ray %d, angle %g: area%d, want area%d", i, angle, got, area+1)
			}
		}
		before := ray - 0.01
		if got := cut.at(2*radius*math.Cos(before), 2*radius*math.Sin(before)); got != cut.areas[(i+2)%3]+1 {
			t.Fatalf("before ray %d: area%d", i, got)
		}
	}
}

func TestChimeraRoomsAndCorridors(t *testing.T) {
	cut := cutAreas(奇美拉.Field(), 12)
	centers := []func() (float64, float64){奇美拉.North, 奇美拉.East, 奇美拉.South, 奇美拉.West}
	for i, center := range centers {
		x, y := center()
		if got := cut.at(x, y); got != cut.areas[i] {
			t.Fatalf("room %d: area%d, want area%d", i, got, cut.areas[i])
		}
	}
	northDoor := unit.HexRadius*math.Sqrt(3)/2 + 160
	for _, c := range []struct {
		p    point
		area int
	}{
		{point{}, 0}, {point{0, northDoor - 1}, 0}, {point{0, northDoor}, cut.areas[0]},
	} {
		if got := cut.at(c.p.x, c.p.y); got != c.area {
			t.Errorf("%+v: area%d, want area%d", c.p, got, c.area)
		}
	}
}

func TestABRElectionAndTie(t *testing.T) {
	n := &network{}
	first := addRouter(n, 1, point{10, 0})
	addRouter(n, 1, point{-10, 0})
	n.elect()
	if n.bridges[1] != (bridge{}) {
		t.Fatal("elected without a BR")
	}
	br := addRouter(n, 0, point{})
	n.elect()
	if got := n.bridges[1]; got != (bridge{first, br}) {
		t.Fatal("distance tie should keep earlier installation")
	}
	closer := addRouter(n, 1, point{5, 0})
	n.elect()
	if got := n.bridges[1]; got != (bridge{closer, br}) || n.bridges[0] != (bridge{}) {
		t.Fatal("new nearest node should become ABR without electing one in the backbone")
	}
}

func addRouter(n *network, area int, p point) *router {
	r := &router{point: p, area: area}
	n.areas[area] = append(n.areas[area], r)
	return r
}

// noFx 去掉纯表现命令。这些 FX 只给皮肤画航线，不影响网络行为断言。
func noFx(cs commands) commands {
	out := make(commands, 0, len(cs))
	for _, c := range cs {
		if _, ok := c.(unit.FX); ok {
			continue
		}
		out = append(out, c)
	}
	return out
}

func flights(cs commands) []flight {
	var out []flight
	for _, c := range cs {
		if s, ok := c.(unit.Spawn); ok {
			if hop, ok := s.Data.(flight); ok {
				out = append(out, hop)
			}
		}
	}
	return out
}

func TestExternalFloodAndNSSATranslation(t *testing.T) {
	n := &network{}
	br := addRouter(n, 0, point{})
	normal := addRouter(n, 1, point{100, 0})
	addRouter(n, 2, point{0, 100})
	nssa := addRouter(n, 3, point{-100, 0})
	n.elect()

	for _, tc := range []struct {
		kind lsa
		from *router
		want []*router
	}{
		{externalLSA, br, []*router{normal}},
		{externalLSA, normal, []*router{br}},
		{nssaLSA, nssa, []*router{br, normal}},
		{routerLSA, normal, nil},
		{networkLSA, br, nil},
	} {
		var cs commands
		a := n.originate(tc.kind, tc.from, destination{})
		a.propagate, a.forwarding = tc.kind == nssaLSA, tc.from
		n.deliver(tc.from, round(a, tc.from.area), &cs)
		for _, target := range tc.want {
			hops := flights(cs)
			if len(hops) != 1 || hops[0].target != target || hops[0].flood.kind != externalLSA {
				t.Fatalf("type %d: expected one type-5 hop into area%d, got %+v", tc.kind, target.area, hops)
			}
			cs = nil
			n.deliver(target, hops[0].flood, &cs)
		}
		if len(cs) != 0 {
			t.Fatalf("type %d: unexpected extra hop %+v", tc.kind, flights(cs))
		}
	}
}

func TestOverlappingRoutersAndConcurrentFlights(t *testing.T) {
	n := &network{}
	dr := addRouter(n, 0, point{})
	from := addRouter(n, 0, point{.5, 0})
	overlap := addRouter(n, 0, point{.6, 0})
	far := addRouter(n, 0, point{100, 0})
	last := addRouter(n, 0, point{200, 0})
	var cs commands
	for i := 0; i < 2; i++ {
		n.deliver(from, round(n.originate(routerLSA, from, destination{node: from}), 0), &cs)
	}
	cs = noFx(cs)
	hops := flights(cs)
	if len(cs) != 6 {
		t.Fatalf("wanted a blast for each of the four instant receipts and two physical hops, got %d commands", len(cs))
	}
	if len(hops) != 2 {
		t.Fatalf("wanted two physical hops, got %d", len(hops))
	}
	cs = nil
	for _, hop := range hops {
		if hop.target != far || dr.db[hop.flood.lsaKey].sequence < hop.flood.sequence || overlap.db[hop.flood.lsaKey].sequence < hop.flood.sequence {
			t.Fatal("expected overlapped nodes to be visited before launch")
		}
		n.deliver(far, hop.flood, &cs)
	}
	if next := flights(cs); len(next) != 2 || next[0].target != last || next[1].target != last {
		t.Fatal("both arrivals should forward independently")
	}
}

func TestSendClockHoldsAndCatchUp(t *testing.T) {
	a := &fighter{
		layout: areaMap{radius: 10, areas: []int{1, 0, 2}},
		birth:  point{100, 0}, net: &network{}, nextSend: sendInterval,
	}
	a.advance(unit.Sense{Time: 0})
	a.advance(unit.Sense{Time: 5, Self: unit.Snapshot{X: 100}})
	if a.nextSend != 1 || a.nextDrop != 6 || len(a.net.areas[2]) != 3 {
		t.Fatalf("while full: next send=%g, next drop=%g, stub nodes=%d", a.nextSend, a.nextDrop, len(a.net.areas[2]))
	}
	a.advance(unit.Sense{Time: 10})
	if a.nextSend != 11 || a.nextDrop != 12 || len(a.net.areas[0]) != 3 {
		t.Fatalf("after catch-up: next send=%g, next drop=%g, backbone nodes=%d", a.nextSend, a.nextDrop, len(a.net.areas[0]))
	}
}

func TestPacketCrossesOutsideChimeraAndArrives(t *testing.T) {
	n := &network{owner: 9}
	x, y := 奇美拉.North()
	start := point{x, y}
	x, y = 奇美拉.East()
	dr := addRouter(n, 0, point{x, y})
	receiver := addRouter(n, 0, point{x + 50, y})
	var cs commands
	n.transmit(start, receiver, round(n.originate(routerLSA, dr, destination{node: dr}), 0), &cs)
	sp := noFx(cs)[0].(unit.Spawn)
	p := &packet{flight: sp.Data.(flight)}
	out := make(chan unit.Cmd, 10)
	ctx := unit.Context{ID: 10, Out: out}
	field := 奇美拉.Field()
	outside := unit.Snapshot{X: (start.x + receiver.x) / 2, Y: (start.y + receiver.y) / 2, VX: sp.VX, VY: sp.VY}
	if field.OutlineContains(outside.X, outside.Y, 0) {
		t.Fatal("fixture does not cross outside Chimera")
	}
	p.Handle(ctx, unit.Sense{Self: outside, Field: field})
	if len(out) != 0 {
		t.Fatal("packet stopped outside the field")
	}
	p.Handle(ctx, unit.Sense{Self: unit.Snapshot{X: receiver.x, Y: receiver.y}, Field: field})
	if c, ok := (<-out).(unit.Despawn); !ok || c.UnitID != ctx.ID {
		t.Fatal("arrival should consume the old hop")
	}
	if c, ok := (<-out).(unit.Spawn); !ok || c.Kind != arrivalBlastKind || c.X != receiver.x || c.Y != receiver.y || c.OwnerID != n.owner {
		t.Fatal("arrival should blast at the designated receiver")
	}
	if c, ok := (<-out).(unit.Spawn); !ok || c.OwnerID != n.owner || c.Data.(flight).target != dr {
		t.Fatal("arrival should relay only from the designated receiver to the DR")
	}
}

func TestInstallationAdvertisesOnlyEligibleAreas(t *testing.T) {
	n := &network{}
	normal := addRouter(n, 1, point{90, 0})
	addRouter(n, 1, point{110, 0})
	addRouter(n, 2, point{0, 100})
	stub := addRouter(n, 2, point{0, 110})
	addRouter(n, 3, point{-100, 0})
	nssa := addRouter(n, 3, point{-110, 0})

	var cs commands
	n.install(point{}, 0, &cs)
	got := map[lsa][4]int{}
	for _, hop := range flights(cs) {
		counts := got[hop.flood.kind]
		counts[hop.target.area]++
		got[hop.flood.kind] = counts
	}
	if !reflect.DeepEqual(got, map[lsa][4]int{
		summaryLSA: {3, 1, 1, 1},
		asbrLSA:    {2, 1, 0, 0},
	}) {
		t.Fatalf("initial election advertisements: %v", got)
	}
	for _, hop := range flights(cs) {
		if hop.target.area >= 2 && hop.target != stub && hop.target != nssa {
			t.Fatalf("default summary targeted an unexpected node: %+v", hop.target)
		}
	}

	cs = nil
	n.install(point{-1, 0}, 0, &cs)
	got = map[lsa][4]int{}
	for _, hop := range flights(cs) {
		counts := got[hop.flood.kind]
		counts[hop.target.area]++
		got[hop.flood.kind] = counts
	}
	if n.bridges[1].abr != normal || !reflect.DeepEqual(got, map[lsa][4]int{
		routerLSA:  {1, 0, 0, 0},
		networkLSA: {1, 0, 0, 0},
		summaryLSA: {0, 1, 0, 0},
		asbrLSA:    {0, 1, 0, 0},
	}) {
		t.Fatalf("new backbone node advertisements: %v", got)
	}
}

func TestPacketHitCooldownIsPerTarget(t *testing.T) {
	p := &packet{}
	out := make(chan unit.Cmd, 10)
	ctx := unit.Context{ID: 1, Out: out}
	hit := func(id uint64, at float64) {
		p.Handle(ctx, unit.Collision{Time: at, Other: unit.Snapshot{ID: id, Slot: 1, Role: unit.RoleFighter}})
	}
	hit(2, 0)
	hit(2, .01)
	hit(3, .01)
	hit(2, .1)
	if len(out) != 3 {
		t.Fatalf("got %d damage commands, want 3", len(out))
	}
	for _, target := range []uint64{2, 3, 2} {
		if d, ok := (<-out).(unit.Damage); !ok || d.To != target || d.Amount != 2 || d.NoFreeze {
			t.Fatalf("unexpected hit: %+v", d)
		}
	}
}

func TestBackboneSummaryIsReoriginatedOnArrival(t *testing.T) {
	n := &network{}
	br := addRouter(n, 0, point{})
	abr := addRouter(n, 1, point{100, 0})
	neighbor := addRouter(n, 1, point{125, 0})
	stub := addRouter(n, 2, point{0, 100})
	n.elect()

	a := n.originate(summaryLSA, stub, destination{area: 2})
	var cs commands
	n.deliver(br, round(a, 0), &cs)
	crossing := flights(cs)
	if len(crossing) != 1 || crossing[0].target != abr || !crossing[0].flood.importSummary || len(abr.db) != 0 {
		t.Fatal("area1 should learn the backbone summary on arrival")
	}
	cs = nil
	n.deliver(abr, crossing[0].flood, &cs)
	n.deliver(abr, crossing[0].flood, &cs)
	local := flights(cs)
	if len(local) != 1 || local[0].target != neighbor {
		t.Fatal("area1 ABR should publish one local copy")
	}
	b := local[0].flood.announcement
	if b.advertiser != abr || b.kind != a.kind || b.destination != a.destination || b.sequence <= a.sequence {
		t.Fatalf("expected a new summary authored by the area1 ABR, got %+v", b)
	}
}

func TestNewNeighborSyncsDRDatabaseWithoutReplayingOldPackets(t *testing.T) {
	n := &network{}
	dr := addRouter(n, 1, point{100, 0})
	old := n.originate(externalLSA, dr, destination{})
	latest := n.originate(externalLSA, dr, destination{})
	var cs commands
	n.deliver(dr, round(latest, 1), &cs)
	n.deliver(dr, round(old, 1), &cs)
	n.install(point{150, 0}, 1, &cs)
	neighbor := n.areas[1][1]
	if got := neighbor.db[old.lsaKey]; got != latest {
		t.Fatalf("neighbor synchronized %+v, want %+v", got, latest)
	}
	n.deliver(neighbor, round(old, 1), &cs)
	n.deliver(neighbor, round(latest, 1), &cs)
	for _, hop := range flights(cs) {
		if hop.flood.kind == externalLSA {
			t.Fatal("historical, older or duplicate announcements should not generate attacks")
		}
	}
}

func TestNSSAExportsPendingRouteWhenBackboneAppears(t *testing.T) {
	n := &network{}
	asbr := addRouter(n, 3, point{100, 0})
	var cs commands
	n.inject(asbr.point, asbr, &cs)
	localOnly := n.originate(nssaLSA, asbr, destination{area: 3})
	localOnly.forwarding = asbr
	n.deliver(asbr, round(localOnly, 3), &cs)
	if len(flights(cs)) != 0 || len(asbr.db) != 2 {
		t.Fatal("isolated NSSA should keep both routes in its database")
	}
	n.install(point{}, 0, &cs)
	var count int
	for _, hop := range flights(cs) {
		switch hop.flood.kind {
		case asbrLSA:
			t.Fatal("the NSSA's internal ASBR must not produce type 4")
		case externalLSA:
			count++
			a := hop.flood.announcement
			if a.advertiser != asbr || a.forwarding != asbr || a.destination != (destination{}) || hop.target.area != 0 {
				t.Fatalf("unexpected type-5 translation: %+v", a)
			}
		}
	}
	if count != 1 {
		t.Fatalf("translated %d routes, want only the route with P-bit set", count)
	}
}

func TestNewArea1ABRRepublishesReachableSummaries(t *testing.T) {
	n := &network{}
	br := addRouter(n, 0, point{})
	oldABR := addRouter(n, 1, point{100, 0})
	newABR := addRouter(n, 1, point{300, 0})
	stub := addRouter(n, 2, point{0, 100})
	n.elect()

	a := n.originate(summaryLSA, stub, destination{area: 2})
	var cs commands
	n.deliver(br, round(a, 0), &cs)
	oldHop := flights(cs)[0]
	n.deliver(oldABR, oldHop.flood, &cs)
	previous := lsaKey{kind: summaryLSA, destination: a.destination, advertiser: oldABR}
	if oldABR.db[previous].sequence == 0 {
		t.Fatal("old ABR never advertised the learned summary")
	}
	cs = nil
	n.install(point{290, 0}, 0, &cs)
	if n.bridges[1].abr != newABR || oldABR.db[previous].sequence != 0 {
		t.Fatal("new BR should change the ABR and withdraw the old summaries")
	}
	var imported, backbone, asbr bool
	for _, hop := range flights(cs) {
		f := hop.flood
		imported = imported || (hop.target.area == 1 && f.kind == summaryLSA && f.advertiser == newABR && f.destination == a.destination && !f.importSummary)
		backbone = backbone || (hop.target.area == 1 && f.kind == summaryLSA && f.advertiser == newABR && f.destination.area == 0)
		asbr = asbr || (hop.target.area == 1 && f.kind == asbrLSA && f.advertiser == newABR && f.destination.node == n.areas[0][1])
	}
	if !imported || !backbone || !asbr {
		t.Fatalf("new ABR announcements: imported=%v backbone=%v new ASBR=%v", imported, backbone, asbr)
	}
}

func TestNewABRSyncsPreviouslyIsolatedExternalRouteSilently(t *testing.T) {
	n := &network{}
	asbr := addRouter(n, 1, point{100, 0})
	a := n.originate(externalLSA, asbr, destination{})
	var cs commands
	n.deliver(asbr, round(a, 1), &cs)
	n.install(point{}, 0, &cs)
	if got := n.areas[0][0].db[a.lsaKey]; got != a {
		t.Fatalf("backbone learned %+v, want %+v", got, a)
	}
	for _, hop := range flights(cs) {
		if hop.flood.kind == externalLSA {
			t.Fatal("database exchange should not replay the historical type-5 shot")
		}
	}
}

func TestArrivalBlastDamagesEachEnemyInRangeOnce(t *testing.T) {
	blast := &arrivalBlast{owner: 8}
	out := make(chan unit.Cmd, 12)
	ctx := unit.Context{ID: 9, Out: out}
	s := unit.Sense{
		Self: unit.Snapshot{X: 100, Y: 40},
		Nearby: []unit.Snapshot{
			{ID: 1, Slot: 1, X: 129, Y: 40, Radius: 5, Role: unit.RoleFighter},
			{ID: 2, Slot: 1, X: 100, Y: 40, Mortal: true},
			{ID: 3, Slot: 1, X: 130, Y: 40, Radius: 5, Role: unit.RoleFighter},
			{ID: 4, Slot: 0, X: 100, Y: 40, Role: unit.RoleFighter},
			{ID: 5, Slot: 1, X: 100, Y: 40, Role: unit.RoleHelper},
		},
	}
	blast.Handle(ctx, s)
	blast.Handle(ctx, s)
	if len(out) != 4 {
		t.Fatalf("expected two hits, one FX, and one despawn, got %d commands", len(out))
	}
	for _, target := range []uint64{1, 2} {
		if d, ok := (<-out).(unit.Damage); !ok || d.From != blast.owner || d.To != target || d.Amount != 2 {
			t.Fatalf("unexpected blast hit: %+v", d)
		}
	}
	if fx, ok := (<-out).(unit.FX); !ok || fx.Amount != blastRadius || fx.X != s.Self.X || fx.Y != s.Self.Y {
		t.Fatalf("unexpected blast effect: %+v", fx)
	}
	if d, ok := (<-out).(unit.Despawn); !ok || d.UnitID != ctx.ID {
		t.Fatal("blast should despawn after one pulse")
	}
}
