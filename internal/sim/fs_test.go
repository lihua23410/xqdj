package sim

import (
	"math"
	"testing"
	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func startFSMatch(t *testing.T) *Match {
	t.Helper()
	m := NewMatchSeeded(1)
	m.SetSlot(0, character.KindWaller)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	t.Cleanup(m.End)
	return m
}

func mustWaller(t *testing.T, m *Match) *unit {
	t.Helper()
	u := fighterByKind(m, character.KindWaller)
	if u == nil {
		t.Fatal("missing 筑墙者")
	}
	return u
}

func tapEvents(u *unit) *[]unitpkg.Event {
	got := &[]unitpkg.Event{}
	u.tap = func(ev unitpkg.Event) {
		*got = append(*got, ev)
	}
	return got
}

func eventsOf[T any](got *[]unitpkg.Event) []T {
	var out []T
	if got == nil {
		return out
	}
	for _, ev := range *got {
		if v, ok := ev.(T); ok {
			out = append(out, v)
		}
	}
	return out
}

func TestFSSetCruiseWritesBaseSpeedNotVelocity(t *testing.T) {
	m := startFSMatch(t)
	m.mu.Lock()
	defer m.mu.Unlock()
	u := mustWaller(t, m)
	u.setVel(vec{80, 0})
	base := u.cruise
	m.applyCmdLocked(unitpkg.SetCruise{UnitID: u.id, Speed: 300})
	if u.cruise != 300 {
		t.Fatalf("cruise=%v", u.cruise)
	}
	if u.cruiseFS == nil || u.cruiseFS.baseSpeed != 300 {
		t.Fatalf("baseSpeed=%v", u.cruiseFS)
	}
	if math.Abs(u.v.X-80) > 1e-9 || u.v.Y != 0 {
		t.Fatalf("vel=%+v", u.v)
	}
	if base == 300 {
		t.Fatal("spec cruise already 300")
	}
}

func TestFSFpComponentAddsToCruise(t *testing.T) {
	m := startFSMatch(t)
	m.mu.Lock()
	defer m.mu.Unlock()
	u := mustWaller(t, m)
	u.cruiseFS.baseSpeed = 150
	u.syncCruise()
	m.applyCmdLocked(unitpkg.AddFSComponent{
		UnitID: u.id, Zone: unitpkg.FSZoneFp, Token: 1, Value: 100,
	})
	if math.Abs(u.cruise-250) > 1e-9 {
		t.Fatalf("cruise=%v want 250", u.cruise)
	}
	m.applyCmdLocked(unitpkg.RemoveFSComponent{UnitID: u.id, Token: 1})
	if math.Abs(u.cruise-150) > 1e-9 {
		t.Fatalf("cruise=%v want 150 after remove", u.cruise)
	}
}

func TestFSComponentSameTokenOverwrites(t *testing.T) {
	m := startFSMatch(t)
	m.mu.Lock()
	defer m.mu.Unlock()
	u := mustWaller(t, m)
	u.cruiseFS.baseSpeed = 150
	u.syncCruise()
	m.applyCmdLocked(unitpkg.AddFSComponent{
		UnitID: u.id, Zone: unitpkg.FSZoneFp, Token: 7, Value: 100,
	})
	m.applyCmdLocked(unitpkg.AddFSComponent{
		UnitID: u.id, Zone: unitpkg.FSZoneFp, Token: 7, Value: 50, ExpiresAt: 9,
	})
	if math.Abs(u.cruise-200) > 1e-9 {
		t.Fatalf("cruise=%v want 200", u.cruise)
	}
	c, ok := u.cruiseFS.components[7]
	if !ok || math.Abs(c.expiresAt-9) > 1e-9 {
		t.Fatalf("comp=%+v ok=%v", c, ok)
	}
}

func TestFSRemoveMissingTokenIsNoop(t *testing.T) {
	m := startFSMatch(t)
	m.mu.Lock()
	defer m.mu.Unlock()
	u := mustWaller(t, m)
	want := u.cruise
	m.applyCmdLocked(unitpkg.RemoveFSComponent{UnitID: u.id, Token: 99})
	m.applyCmdLocked(unitpkg.RemoveFS{UnitID: u.id, Token: 99})
	if u.cruise != want {
		t.Fatalf("cruise=%v want %v", u.cruise, want)
	}
}

func TestFSMZeroClampsCruiseAndVelocity(t *testing.T) {
	m := startFSMatch(t)
	m.mu.Lock()
	u := mustWaller(t, m)
	u.setVel(vec{200, 0})
	u.cruiseFS.baseSpeed = 150
	u.syncCruise()
	m.applyCmdLocked(unitpkg.AddFSComponent{
		UnitID: u.id, Zone: unitpkg.FSZoneM, Token: 2, Value: 0,
		ExpiresAt: 0.05,
	})
	m.applyCmdLocked(unitpkg.Stun{UnitID: u.id, Hold: true})
	if u.cruise != 0 {
		m.mu.Unlock()
		t.Fatalf("cruise=%v", u.cruise)
	}
	if u.v.len() > 1e-9 {
		m.mu.Unlock()
		t.Fatalf("v=%+v", u.v)
	}
	m.mu.Unlock()
	for i := 0; i < 10; i++ {
		m.Tick()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if math.Abs(u.cruise-150) > 1e-9 {
		t.Fatalf("cruise after expire=%v", u.cruise)
	}
	if math.Abs(u.v.X-150) > 1e-6 || math.Abs(u.v.Y) > 1e-6 {
		t.Fatalf("want resume along saved heading, vel=%+v", u.v)
	}
}

func TestFSImpulseHoldsConstantSpeed(t *testing.T) {
	m := startFSMatch(t)
	m.mu.Lock()
	u := mustWaller(t, m)
	u.p = vec{}
	if o := fighterByKind(m, character.KindRanged); o != nil {
		o.p = vec{-120, 80}
		o.setVel(vec{})
	}
	m.applyCmdLocked(unitpkg.AddFS{
		UnitID: u.id, DX: 1, DY: 0, BaseSpeed: 400, Token: 3,
	})
	m.applyCmdLocked(unitpkg.Stun{UnitID: u.id, Hold: true})
	if math.Abs(u.v.X-400) > 1e-9 || u.v.Y != 0 {
		m.mu.Unlock()
		t.Fatalf("v=%+v", u.v)
	}
	m.mu.Unlock()
	for i := 0; i < 12; i++ {
		m.Tick()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(u.fsList) != 1 {
		t.Fatalf("fsList=%d", len(u.fsList))
	}
	if math.Abs(u.v.X-400) > 1e-6 || math.Abs(u.v.Y) > 1e-6 {
		t.Fatalf("after ticks v=%+v", u.v)
	}
}

func TestFSImpulseExpires(t *testing.T) {
	m := startFSMatch(t)
	m.mu.Lock()
	u := mustWaller(t, m)
	u.p = vec{}
	m.applyCmdLocked(unitpkg.AddFS{
		UnitID: u.id, DX: 1, DY: 0, BaseSpeed: 400, Token: 4,
		ExpiresAt: 0.05,
	})
	m.mu.Unlock()
	for i := 0; i < 10; i++ {
		m.Tick()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(u.fsList) != 0 {
		t.Fatalf("fsList=%d after expire", len(u.fsList))
	}
}

func TestFSImpulseOnWallExpiresSameFrame(t *testing.T) {
	m := startFSMatch(t)
	m.mu.Lock()
	u := mustWaller(t, m)
	u.p = vec{HexRadius - u.radius - 12, 0}
	if o := fighterByKind(m, character.KindRanged); o != nil {
		o.p = vec{-120, 80}
		o.setVel(vec{})
	}
	m.applyCmdLocked(unitpkg.AddFS{
		UnitID: u.id, DX: 1, DY: 0, BaseSpeed: 600, OnWall: true, Token: 5,
	})
	m.applyCmdLocked(unitpkg.Stun{UnitID: u.id, Hold: true})
	m.mu.Unlock()
	hit := false
	for i := 0; i < 20; i++ {
		m.Tick()
		m.mu.Lock()
		if len(u.fsList) == 0 {
			hit = true
			if u.v.X >= 0 {
				vx := u.v
				m.mu.Unlock()
				t.Fatalf("expected reflected v.X<0 got %+v", vx)
			}
			m.mu.Unlock()
			break
		}
		m.mu.Unlock()
	}
	if !hit {
		t.Fatal("OnWall FS never expired")
	}
	m.Tick()
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(u.fsList) != 0 {
		t.Fatal("impulse returned after wall")
	}
	if u.v.X >= 0 {
		t.Fatalf("next frame rewrote old direction v=%+v", u.v)
	}
}

func TestFSSetDirectionDoesNotChangeVelocity(t *testing.T) {
	m := startFSMatch(t)
	m.mu.Lock()
	defer m.mu.Unlock()
	u := mustWaller(t, m)
	u.setVel(vec{40, 0})
	m.applyCmdLocked(unitpkg.SetFSDirection{UnitID: u.id, VX: 0, VY: 1})
	if math.Abs(u.cruiseFS.dir.X) > 1e-9 || math.Abs(u.cruiseFS.dir.Y-1) > 1e-9 {
		t.Fatalf("dir=%+v", u.cruiseFS.dir)
	}
	if math.Abs(u.v.X-40) > 1e-9 || u.v.Y != 0 {
		t.Fatalf("vel=%+v", u.v)
	}
}

func TestStunUntilRestoresSense(t *testing.T) {
	m := startFSMatch(t)
	m.mu.Lock()
	u := mustWaller(t, m)
	got := tapEvents(u)
	m.applyCmdLocked(unitpkg.Stun{UnitID: u.id, Hold: true, Until: m.time + 0.05})
	if !u.stun {
		m.mu.Unlock()
		t.Fatal("stun not held")
	}
	m.mu.Unlock()
	for i := 0; i < 10; i++ {
		m.Tick()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if u.stun {
		t.Fatal("stun still held after Until")
	}
	senses := eventsOf[unitpkg.Sense](got)
	if len(senses) == 0 {
		t.Fatalf("missing Sense after Until time=%v stunUntil=%v n=%d", m.time, u.stunUntil, len(*got))
	}
}

func TestStunUntilZeroStaysHeld(t *testing.T) {
	m := startFSMatch(t)
	m.mu.Lock()
	u := mustWaller(t, m)
	m.applyCmdLocked(unitpkg.Stun{UnitID: u.id, Hold: true})
	m.mu.Unlock()
	for i := 0; i < 8; i++ {
		m.Tick()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !u.stun {
		t.Fatal("Hold without Until should stay")
	}
	m.applyCmdLocked(unitpkg.Stun{UnitID: u.id, Hold: false})
	if u.stun || u.stunUntil != 0 {
		t.Fatalf("stun=%v until=%v", u.stun, u.stunUntil)
	}
}

func TestFactionChangedOnMarkAndCycle(t *testing.T) {
	m := startFSMatch(t)
	m.mu.Lock()
	defer m.mu.Unlock()
	u := mustWaller(t, m)
	got := tapEvents(u)
	m.applyCmdLocked(unitpkg.MarkFaction{
		UnitID: u.id, Faction: unitpkg.FactionRed, Cycle: true,
	})
	marked := eventsOf[unitpkg.FactionChanged](got)
	if len(marked) != 1 || marked[0].Faction != unitpkg.FactionRed {
		t.Fatalf("mark got %#v", marked)
	}
	*got = nil
	m.applyCmdLocked(unitpkg.MarkFaction{
		UnitID: u.id, Faction: unitpkg.FactionRed, Cycle: true,
	})
	if extra := eventsOf[unitpkg.FactionChanged](got); len(extra) != 0 {
		t.Fatalf("same faction should not resend: %#v", extra)
	}
	m.cycleFactionLocked(u)
	cycled := eventsOf[unitpkg.FactionChanged](got)
	if len(cycled) != 1 || cycled[0].Faction == "" || cycled[0].Faction == unitpkg.FactionRed {
		t.Fatalf("cycle got %#v", cycled)
	}
}
