package sim

import (
	"encoding/json"
	"math"
	"testing"

	"xqdj/character"
	"xqdj/character/hff"
	unitpkg "xqdj/internal/unit"
)

func TestSettleQuotesMergeSelf(t *testing.T) {
	host := uint64(1)
	rows := []paperRow{
		{id: 1, slot: 0, role: unitpkg.RoleFighter},
		{id: 2, slot: 1, role: unitpkg.RoleFighter},
		{id: 3, slot: 1, role: unitpkg.RoleFighter},
		{id: 4, slot: 0, mortal: true},
		{id: 5, slot: 1},
	}
	loss := map[uint64]float64{1: 30, 2: 10, 3: 20, 4: 100, 5: 1000}
	long := settleQuotes(host, rows, loss, true)
	if len(long) != 1 || long[0].To != 1 || math.Abs(long[0].Amount-30) > 1e-9 {
		t.Fatalf("long %+v", long)
	}
	short := settleQuotes(host, rows, loss, false)
	if len(short) != 2 {
		t.Fatalf("short %+v", short)
	}
	if short[0].To != 2 || math.Abs(short[0].Amount-40) > 1e-9 || short[1].To != 3 || math.Abs(short[1].Amount-20) > 1e-9 {
		t.Fatalf("short %+v", short)
	}
}

func TestPaperDiscardsSandbox(t *testing.T) {
	hff.SetRatesForTest(map[string]float64{character.KindPastor: 0.5})
	defer hff.SetRatesForTest(nil)
	m := NewMatchSeeded(3)
	m.skipProbe = true
	m.SetSlot(0, hff.KindHFF)
	m.SetSlot(1, character.KindOrca)
	m.Start()
	defer m.End()
	host := findKind(m, hff.KindHFF)
	if host == nil {
		t.Fatal("missing fighter")
	}
	hostID := host.id
	sawOpening, sawPortrait, sawPaper := false, false, false
	var frozenTime, lastSceneTime float64
	for i := 0; i < 1000; i++ {
		m.Tick()
		var msg struct {
			Units []struct {
				ID   uint64 `json:"id"`
				Kind string `json:"kind"`
			} `json:"units"`
			Time      float64    `json:"time"`
			SceneTime *float64   `json:"sceneTime"`
			Paper     bool       `json:"paper"`
			PaperView *paperView `json:"paperView"`
		}
		if err := json.Unmarshal(m.SnapshotJSON(), &msg); err != nil {
			t.Fatal(err)
		}
		if msg.Paper {
			if msg.SceneTime == nil {
				t.Fatal("sandbox frame does not carry the clock that timestamps its effects")
			}
			if !sawPaper {
				frozenTime = msg.Time
			}
			if msg.Time != frozenTime || *msg.SceneTime < lastSceneTime {
				t.Fatalf("sandbox clock moved backwards or advanced the real match: match=%v scene=%v", msg.Time, *msg.SceneTime)
			}
			lastSceneTime = *msg.SceneTime
		} else if msg.SceneTime != nil {
			t.Fatal("real arena frame is using the sandbox clock")
		}
		for _, u := range msg.Units {
			if !msg.Paper && u.ID == hostID && u.Kind == hff.KindHFF && msg.PaperView != nil && msg.PaperView.OpenProgress > 0 {
				view := msg.PaperView
				if view.HostID != hostID || view.Preview == nil || view.Preview.Kind != character.KindPastor {
					t.Fatalf("opening frame has no stock preview for host %d: %+v", hostID, view)
				}
				sawOpening = true
				if view.CutIn > 0 {
					sawPortrait = true
					if view.OpenProgress != float64(paperPortraitAt)/paperOpenTicks || view.Elapsed != 0 {
						t.Fatalf("portrait did not hold the opening frame: %+v", view)
					}
				}
			}
			if msg.Paper && u.ID == hostID && u.Kind == character.KindPastor &&
				msg.PaperView != nil && msg.PaperView.Stock == character.KindPastor && !msg.PaperView.Settled {
				if msg.PaperView.HostID != hostID || msg.PaperView.Preview != nil {
					t.Fatalf("sandbox frame did not switch from preview to transformed host: %+v", msg.PaperView)
				}
				sawPaper = true
			}
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !sawOpening || !sawPaper {
		t.Fatal("opening or sandbox never shown")
	}
	if lastSceneTime <= frozenTime+1 {
		t.Fatalf("sandbox clock did not advance while the real match was frozen: match=%v scene=%v", frozenTime, lastSceneTime)
	}
	if !sawPortrait {
		t.Fatal("portrait never shown during the opening hold")
	}
	if m.time < 3 || m.time > 8 {
		t.Fatalf("match time %v", m.time)
	}
	if u := m.units[hostID]; u != nil && u.kind != hff.KindHFF {
		t.Fatalf("host left as %s", u.kind)
	}
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && u.kind == character.KindGlyph {
			t.Fatal("sandbox glyph leaked onto the real field")
		}
	}
}

func TestRiceDoesNotBecomeWhale(t *testing.T) {
	hff.SetRatesForTest(map[string]float64{character.KindOrca: 1})
	defer hff.SetRatesForTest(nil)
	m := NewMatchSeeded(5)
	m.skipProbe = true
	m.SetSlot(0, hff.KindHFF)
	m.SetSlot(1, character.KindPastor)
	m.Start()
	defer m.End()
	m.mu.Lock()
	fish := findKindLocked(m, hff.KindHFF)
	fish.hp = 50
	m.addUnitLocked(character.KindRice, fish.p, vec{}, fish.id, fish.slot)
	m.mu.Unlock()
	for i := 0; i < 120; i++ {
		m.Tick()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	fish = findKindLocked(m, hff.KindHFF)
	if fish == nil {
		t.Fatal("down")
	}
	if fish.hp <= 50 || fish.hp > 58 || math.Abs(fish.radius-18) > 1e-6 || fish.kind != hff.KindHFF {
		t.Fatalf("hp %v radius %v kind %s", fish.hp, fish.radius, fish.kind)
	}
	if findKindLocked(m, character.KindRice) != nil {
		t.Fatal("bowl stayed")
	}
}

func TestRiceYieldsToOrca(t *testing.T) {
	hff.SetRatesForTest(map[string]float64{character.KindOrca: 1})
	defer hff.SetRatesForTest(nil)
	m := NewMatchSeeded(6)
	m.skipProbe = true
	m.SetSlot(0, hff.KindHFF)
	m.SetSlot(1, character.KindPastor)
	m.Start()
	defer m.End()
	m.mu.Lock()
	fish := findKindLocked(m, hff.KindHFF)
	fish.hp = 50
	m.addUnitLocked(character.KindOrca, fish.p, vec{}, 0, 0)
	orca := findKindLocked(m, character.KindOrca)
	m.addUnitLocked(character.KindRice, fish.p, vec{}, orca.id, orca.slot)
	m.mu.Unlock()
	for i := 0; i < 100; i++ {
		m.Tick()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	fish = findKindLocked(m, hff.KindHFF)
	if fish == nil {
		t.Fatal("down")
	}
	if math.Abs(fish.hp-50) > 1e-6 {
		t.Fatalf("she ate anyway, hp %v", fish.hp)
	}
}

func TestWipeDumpsOnlyWhenSomeoneElseClears(t *testing.T) {
	m := &Match{ledger: &paperLedger{born: map[uint64]bool{1: true}, loss: map[uint64]float64{}}}
	u := &unit{id: 1, hp: 40, role: unitpkg.RoleFighter}
	m.notePaperWipeLocked(u, 2)
	if math.Abs(m.ledger.loss[1]-40) > 1e-9 {
		t.Fatalf("arrest %v", m.ledger.loss[1])
	}
	m.notePaperWipeLocked(u, 1)
	m.notePaperWipeLocked(u, 0)
	if math.Abs(m.ledger.loss[1]-40) > 1e-9 {
		t.Fatalf("self or unknown added %v", m.ledger.loss[1])
	}
}

func TestKnownFakeLossSettlesOnce(t *testing.T) {
	hff.SetRatesForTest(map[string]float64{character.KindOrca: 0.5})
	defer hff.SetRatesForTest(nil)
	m := NewMatchSeeded(9)
	m.skipProbe = true
	m.SetSlot(0, hff.KindHFF)
	m.SetSlot(1, character.KindOrca)
	m.Start()
	defer m.End()
	host := findKind(m, hff.KindHFF)
	opp := findKind(m, character.KindOrca)
	if host == nil || opp == nil {
		t.Fatal("missing fighter")
	}
	hostID, oppID := host.id, opp.id
	injected := false
	var long bool
	var baseHost, baseOpp float64
	var led *paperLedger
	parentMoved := false
	shared := false
	sawOpeningHold, sawClosingHold := false, false
	for i := 0; i < 2000; i++ {
		m.Tick()
		m.mu.Lock()
		if m.paper != nil && m.paper.view.OpenProgress > 0 {
			sawOpeningHold = true
			if m.paper.child.time != m.time {
				t.Fatal("sandbox advanced during the opening hold")
			}
		}
		if !injected && m.paper != nil && m.paper.child != nil {
			ch := m.paper.child
			ch.mu.Lock()
			hu, ou := ch.units[hostID], ch.units[oppID]
			if hu != nil && ou != nil && m.units[hostID] != nil && m.units[oppID] != nil {
				baseHost = m.units[hostID].hp
				baseOpp = m.units[oppID].hp
				ch.applyHurtLocked(ou, hu, 30, false, false)
				ch.applyHurtLocked(hu, ou, 10, false, false)
				long = m.paper.long
				led = ch.ledger
				agentLog("A", "paper_test.go:inject", "fake damage", map[string]any{
					"long": long, "sameHost": hu == m.units[hostID], "sameOpp": ou == m.units[oppID],
					"parentHost": m.units[hostID].hp, "parentOpp": m.units[oppID].hp,
					"childHost": hu.hp, "childOpp": ou.hp, "baseHost": baseHost, "baseOpp": baseOpp,
				})
				injected = true
				shared = hu == m.units[hostID] || ou == m.units[oppID]
				parentMoved = m.units[hostID].hp != baseHost || m.units[oppID].hp != baseOpp
			}
			ch.mu.Unlock()
		}
		if m.paperClose != nil && m.seq-m.paperCloseSeq <= paperFreezeTicks+paperFlightTicks {
			sawClosingHold = true
			if m.units[hostID].hp != baseHost || m.units[oppID].hp != baseOpp {
				t.Fatal("real HP changed before the settlement shot arrived")
			}
		}
		idle := injected && m.paper == nil && !m.settling && len(m.settleQ) == 0 && m.hitStop == 0
		var hostHP, oppHP float64
		if u := m.units[hostID]; u != nil {
			hostHP = u.hp
		}
		if u := m.units[oppID]; u != nil {
			oppHP = u.hp
		}
		m.mu.Unlock()
		if idle {
			if !sawOpeningHold || !sawClosingHold {
				t.Fatal("missing opening or closing hold")
			}
			agentLog("B", "paper_test.go:done", "after settlement", map[string]any{
				"long": long, "host": hostHP, "opp": oppHP, "baseHost": baseHost, "baseOpp": baseOpp,
				"hostDrop": baseHost - hostHP, "oppDrop": baseOpp - oppHP,
			})
			if shared {
				t.Fatal("sandbox unit aliases the real unit")
			}
			if parentMoved {
				t.Fatal("sandbox damage changed real HP")
			}
			if led == nil {
				t.Fatal("missing ledger")
			}
			delta := led.loss[hostID] - led.loss[oppID]
			wantHost, wantOpp := 0.0, 0.0
			if long == (delta > 0) {
				wantHost = math.Abs(delta)
			} else {
				wantOpp = math.Abs(delta) * paperProfitScale
			}
			if math.Abs((baseHost-hostHP)-wantHost) > 1e-6 || math.Abs((baseOpp-oppHP)-wantOpp) > 1e-6 {
				t.Fatalf("long %v loss %v host drop %v want %v opp drop %v want %v", long, led.loss, baseHost-hostHP, wantHost, baseOpp-oppHP, wantOpp)
			}
			var frame struct {
				View *paperView `json:"paperView"`
			}
			if err := json.Unmarshal(m.SnapshotJSON(), &frame); err != nil {
				t.Fatal(err)
			}
			view := frame.View
			if view == nil || !view.Settled || view.Long != long || math.Abs(view.Self-wantHost) > 1e-6 || math.Abs(view.Enemy-wantOpp) > 1e-6 {
				t.Fatalf("settlement slip %+v, want self %v and enemy %v", view, wantHost, wantOpp)
			}
			if len(view.Candles) != int(paperSec) || math.Abs(view.Candles[len(view.Candles)-1].Close-(wantOpp-wantHost)) > 1e-6 {
				t.Fatalf("last candle must close at the settlement difference: %+v", view.Candles)
			}
			for _, impact := range view.Targets {
				want := wantOpp
				if impact.ID == hostID {
					want = wantHost
				}
				if impact.ImpactSeq == 0 || impact.Loss == nil || math.Abs(*impact.Loss-want) > 1e-6 {
					t.Fatalf("impact %+v, want loss %v", impact, want)
				}
			}
			return
		}
	}
	t.Fatal("settlement did not finish")
}

func TestLethalPaperImpactKeepsReadoutBeforeWinner(t *testing.T) {
	m := NewMatchSeeded(9)
	m.skipProbe = true
	m.SetSlot(0, hff.KindHFF)
	m.SetSlot(1, character.KindOrca)
	m.Start()
	defer m.End()

	m.mu.Lock()
	m.removeLocked(findKindLocked(m, character.KindOrca))
	m.paperClose = &paperView{Settled: true, ImpactTicks: paperImpactTicks}
	m.paperCloseSeq = m.seq
	m.seq += paperFreezeTicks + paperFlightTicks
	m.paperCloseUntil = m.seq + paperImpactTicks
	matchTime := m.time
	m.checkWinLocked()
	m.mu.Unlock()

	for range paperImpactTicks - 1 {
		m.Tick()
	}
	if m.phase != PhaseRunning || m.time != matchTime || m.paperViewLocked() == nil {
		t.Fatalf("readout was interrupted: phase %v, battle time %v", m.phase, m.time)
	}
	m.Tick()
	if m.phase != PhaseEnded || m.winner != hff.KindHFF || m.time != matchTime {
		t.Fatalf("expected victory after readout: phase %v, winner %q, battle time %v", m.phase, m.winner, m.time)
	}
}

func TestSettleQuotesFoldMinionLoss(t *testing.T) {
	host := uint64(1)
	rows := []paperRow{
		{id: 1, slot: 0, role: unitpkg.RoleFighter},
		{id: 2, slot: 1, role: unitpkg.RoleFighter},
		{id: 3, slot: 1, role: unitpkg.RoleMinion, mortal: true, owner: 2},
		{id: 4, slot: 1, role: unitpkg.RoleMinion, mortal: true, owner: 2},
		{id: 5, slot: 1, role: unitpkg.RoleMinion, mortal: true, owner: 2, passHP: true},
		{id: 6, slot: 1, role: unitpkg.RoleMinion, mortal: true},
	}
	loss := map[uint64]float64{1: 34, 2: 6, 3: 10, 4: 8, 5: 50, 6: 4}
	long := settleQuotes(host, rows, loss, true)
	// 34 − (6+10+8) = 10 她付；转伤的 50 已在主人账上不再加。没主人的 4 单独比，34−4=30 也是她付。
	if len(long) != 1 || long[0].To != 1 || math.Abs(long[0].Amount-40) > 1e-9 {
		t.Fatalf("long %+v", long)
	}
	short := settleQuotes(host, rows, loss, false)
	if len(short) != 2 || short[0].To != 2 || math.Abs(short[0].Amount-20) > 1e-9 || short[1].To != 6 || math.Abs(short[1].Amount-60) > 1e-9 {
		t.Fatalf("short %+v", short)
	}
}

func findKind(m *Match, kind string) *unit {
	m.mu.Lock()
	defer m.mu.Unlock()
	return findKindLocked(m, kind)
}

func findKindLocked(m *Match, kind string) *unit {
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && !u.stopped && u.kind == kind {
			return u
		}
	}
	return nil
}
