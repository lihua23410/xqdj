package sim

import (
	"math"
	"testing"
	"time"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func parkStill(u *unit, x, y float64) {
	if u == nil {
		return
	}
	u.p = vec{x, y}
	u.stand = true
	u.setVel(vec{})
}

func pastorGlyphs(m *Match, owner uint64) []*unit {
	var out []*unit
	for _, id := range m.order {
		u := m.units[id]
		if u != nil && !u.stopped && u.kind == character.KindGlyph && u.owner == owner {
			out = append(out, u)
		}
	}
	return out
}

func startPastor(t *testing.T, foe string) *Match {
	t.Helper()
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindPastor)
	m.SetSlot(1, foe)
	m.Start()
	return m
}

func TestPastorInnerRing(t *testing.T) {
	m := startPastor(t, character.KindDummy)
	defer m.End()
	waitTicks(m, 6)
	m.mu.Lock()
	p := fighterByKind(m, character.KindPastor)
	o := fighterByKind(m, character.KindDummy)
	parkStill(p, 0, 0)
	parkStill(o, 220, 0)
	m.mu.Unlock()
	waitTicks(m, 8)

	m.mu.Lock()
	defer m.mu.Unlock()
	p = fighterByKind(m, character.KindPastor)
	gs := pastorGlyphs(m, p.id)
	if len(gs) != 4 {
		t.Fatalf("glyphs=%d want 4", len(gs))
	}
	for _, g := range gs {
		d := math.Hypot(g.p.X-p.p.X, g.p.Y-p.p.Y)
		if math.Abs(d-46) > 3 {
			t.Fatalf("glyph radius %v", d)
		}
	}
}

func TestPastorGlyphDamagesThenVanishes(t *testing.T) {
	m := startPastor(t, character.KindDummy)
	defer m.End()
	waitTicks(m, 6)
	m.mu.Lock()
	p := fighterByKind(m, character.KindPastor)
	o := fighterByKind(m, character.KindDummy)
	parkStill(p, 0, 0)
	parkStill(o, 220, 0)
	m.mu.Unlock()
	waitTicks(m, 6)

	m.mu.Lock()
	p = fighterByKind(m, character.KindPastor)
	o = fighterByKind(m, character.KindDummy)
	gs := pastorGlyphs(m, p.id)
	if len(gs) == 0 || o == nil {
		m.mu.Unlock()
		t.Fatal("missing")
	}
	parkStill(o, gs[0].p.X, gs[0].p.Y)
	hp := o.hp
	m.mu.Unlock()
	waitTicks(m, 12)

	m.mu.Lock()
	defer m.mu.Unlock()
	p = fighterByKind(m, character.KindPastor)
	o = fighterByKind(m, character.KindDummy)
	if o.hp != hp-2 {
		t.Fatalf("hp=%v want %v", o.hp, hp-2)
	}
	if len(pastorGlyphs(m, p.id)) != 3 {
		t.Fatalf("glyphs=%d want 3", len(pastorGlyphs(m, p.id)))
	}
}

func TestPastorGlyphDoesNotHitStop(t *testing.T) {
	m := startPastor(t, character.KindDummy)
	defer m.End()
	waitTicks(m, 6)
	m.mu.Lock()
	p := fighterByKind(m, character.KindPastor)
	o := fighterByKind(m, character.KindDummy)
	parkStill(p, 0, 0)
	parkStill(o, 220, 0)
	m.mu.Unlock()
	waitTicks(m, 6)

	m.mu.Lock()
	p = fighterByKind(m, character.KindPastor)
	o = fighterByKind(m, character.KindDummy)
	gs := pastorGlyphs(m, p.id)
	if len(gs) == 0 || o == nil {
		m.mu.Unlock()
		t.Fatal("missing")
	}
	if !p.noFrameFreeze {
		m.mu.Unlock()
		t.Fatal("pastor missing no-frame-freeze")
	}
	parkStill(o, gs[0].p.X, gs[0].p.Y)
	hp := o.hp
	m.mu.Unlock()

	saw := false
	for i := 0; i < 20; i++ {
		m.Tick()
		time.Sleep(2 * time.Millisecond)
		m.mu.Lock()
		o = fighterByKind(m, character.KindDummy)
		stopped := m.hitStop != 0
		lost := o != nil && o.hp < hp
		m.mu.Unlock()
		if stopped {
			t.Fatal("glyph hit stopped the frame")
		}
		if lost {
			saw = true
			break
		}
	}
	if !saw {
		t.Fatal("glyph never hit")
	}
}

func TestPastorBodyDoesNotDamage(t *testing.T) {
	m := startPastor(t, character.KindDummy)
	defer m.End()
	waitTicks(m, 6)
	m.mu.Lock()
	p := fighterByKind(m, character.KindPastor)
	o := fighterByKind(m, character.KindDummy)
	parkStill(p, 0, 0)
	parkStill(o, 220, 0)
	m.mu.Unlock()
	waitTicks(m, 6)

	m.mu.Lock()
	p = fighterByKind(m, character.KindPastor)
	o = fighterByKind(m, character.KindDummy)
	parkStill(p, 0, 0)
	parkStill(o, 0, 0)
	hp := o.hp
	m.mu.Unlock()
	for i := 0; i < 10; i++ {
		m.mu.Lock()
		p = fighterByKind(m, character.KindPastor)
		o = fighterByKind(m, character.KindDummy)
		parkStill(p, 0, 0)
		parkStill(o, 0, 0)
		m.mu.Unlock()
		m.Tick()
		time.Sleep(2 * time.Millisecond)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	o = fighterByKind(m, character.KindDummy)
	if o.hp != hp {
		t.Fatalf("body hit hp=%v want %v", o.hp, hp)
	}
}

func TestPastorPushesToThreeRingsThenDropsOldest(t *testing.T) {
	m := startPastor(t, character.KindDummy)
	defer m.End()
	waitTicks(m, 4)
	m.mu.Lock()
	p := fighterByKind(m, character.KindPastor)
	o := fighterByKind(m, character.KindDummy)
	parkStill(p, 0, 0)
	parkStill(o, 240, 40)
	m.mu.Unlock()

	var gotSecond, gotThird bool
	for i := 0; i < 400; i++ {
		m.mu.Lock()
		p = fighterByKind(m, character.KindPastor)
		o = fighterByKind(m, character.KindDummy)
		parkStill(p, 0, 0)
		parkStill(o, 240, 40)
		now := m.time
		n46, n78, n110 := 0, 0, 0
		for _, g := range pastorGlyphs(m, p.id) {
			d := math.Hypot(g.p.X, g.p.Y)
			switch {
			case math.Abs(d-46) < 4:
				n46++
			case math.Abs(d-78) < 4:
				n78++
			case math.Abs(d-110) < 4:
				n110++
			}
		}
		m.mu.Unlock()
		if now > 1.2 && n46 == 5 && n78 == 4 && n110 == 0 {
			gotSecond = true
		}
		if now > 4.2 && n46 == 5 && n78 == 5 && n110 == 4 {
			gotThird = true
			break
		}
		m.Tick()
		time.Sleep(time.Millisecond)
	}
	if !gotSecond || !gotThird {
		t.Fatalf("rings second=%v third=%v", gotSecond, gotThird)
	}
}

func TestPastorRingSpins(t *testing.T) {
	m := startPastor(t, character.KindDummy)
	defer m.End()
	waitTicks(m, 6)
	m.mu.Lock()
	p := fighterByKind(m, character.KindPastor)
	o := fighterByKind(m, character.KindDummy)
	parkStill(p, 0, 0)
	parkStill(o, 240, 0)
	m.mu.Unlock()
	waitTicks(m, 6)

	m.mu.Lock()
	p = fighterByKind(m, character.KindPastor)
	gs := pastorGlyphs(m, p.id)
	if len(gs) == 0 {
		m.mu.Unlock()
		t.Fatal("missing glyphs")
	}
	g0 := gs[0]
	id := g0.id
	a0 := math.Atan2(g0.p.Y-p.p.Y, g0.p.X-p.p.X)
	t0 := m.time
	m.mu.Unlock()
	waitTicks(m, 30)

	m.mu.Lock()
	defer m.mu.Unlock()
	p = fighterByKind(m, character.KindPastor)
	var g *unit
	for _, u := range pastorGlyphs(m, p.id) {
		if u.id == id {
			g = u
			break
		}
	}
	if g == nil {
		t.Fatal("glyph left the ring")
	}
	d := math.Hypot(g.p.X-p.p.X, g.p.Y-p.p.Y)
	if math.Abs(d-46) > 3 {
		t.Fatalf("glyph radius %v", d)
	}
	a1 := math.Atan2(g.p.Y-p.p.Y, g.p.X-p.p.X)
	delta := math.Atan2(math.Sin(a1-a0), math.Cos(a1-a0))
	dt := m.time - t0
	if dt < 0.2 {
		t.Fatalf("dt=%v", dt)
	}
	speed := -delta / dt
	slow := 2 * math.Pi / 2.5
	fast := 2 * math.Pi / 1.5
	if speed < slow-0.25 || speed > fast+0.25 {
		t.Fatalf("spin speed=%v want %v..%v", speed, slow, fast)
	}
}

func TestPastorDeathRemovesGlyphs(t *testing.T) {
	m := startPastor(t, character.KindDummy)
	defer m.End()
	waitTicks(m, 8)
	m.mu.Lock()
	p := fighterByKind(m, character.KindPastor)
	if p == nil {
		m.mu.Unlock()
		t.Fatal("missing pastor")
	}
	id := p.id
	from := fighterByKind(m, character.KindDummy)
	p.hp = 1
	m.applyCmdLocked(unitpkg.Damage{From: from.id, To: id, Amount: 50})
	m.mu.Unlock()
	time.Sleep(30 * time.Millisecond)
	m.mu.Lock()
	m.settleHitsLocked()
	if fighterByKind(m, character.KindPastor) != nil {
		m.mu.Unlock()
		t.Fatal("pastor still up")
	}
	if n := len(pastorGlyphs(m, id)); n != 0 {
		m.mu.Unlock()
		t.Fatalf("glyphs left %d", n)
	}
	m.mu.Unlock()
}
