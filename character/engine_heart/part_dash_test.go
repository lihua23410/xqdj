package 泵之心

import (
	"math"
	"testing"
	"xqdj/character/engine_heart/parts/api"
	"xqdj/internal/unit"
)

func heartWithPartRoll(t *testing.T, w *world, self unit.Snapshot, trigger bool) *heart {
	t.Helper()
	for beat := 1; beat < 1000; beat++ {
		h := &heart{stream: 91, beats: beat}
		if _, _, ok := h.partDash(w, self); ok == trigger {
			return h
		}
	}
	t.Fatal("could not exercise the requested heartbeat branch")
	return nil
}

func TestPartDashChoosesNearestCollectiblePart(t *testing.T) {
	w := &world{parts: []part{
		{Type: 9, Equipped: true, Y: 1},
		{Type: 0, X: 80},
		{Type: 1, Y: 40},
		{Type: 11, Y: 10, Flying: true},
	}}
	self := unit.Snapshot{ID: 1, VX: 150, Radius: 18, HP: 100}
	h := heartWithPartRoll(t, w, self, true)
	x, y, ok := h.partDash(w, self)
	if !ok || math.Abs(x) > 1e-9 || math.Abs(y-1) > 1e-9 {
		t.Fatalf("did not target the closest loose, stopped part: (%f,%f,%v)", x, y, ok)
	}
	out := make(chan unit.Cmd, 128)
	h.contract(unit.Context{ID: 1, Kind: KindEngineHeart, Out: out}, w, unit.Sense{Self: self, Field: unit.HexField(), Time: .4}, false)
	foundDash, foundCruise := false, false
	for len(out) > 0 {
		switch c := (<-out).(type) {
		case unit.AddFS:
			if c.UnitID == 1 && c.Token == api.CoreDashToken(1) {
				foundDash = true
				if math.Abs(c.DX) > 1e-9 || math.Abs(c.DY-1) > 1e-9 {
					t.Fatal("steering equipment diverted the part-seeking dash")
				}
			}
		case unit.SetFSDirection:
			foundCruise = true
			if math.Abs(c.VX) > 1e-9 || math.Abs(c.VY-1) > 1e-9 {
				t.Fatal("cruise direction did not follow the part-seeking turn")
			}
		}
	}
	if !foundDash || !foundCruise {
		t.Fatal("part-seeking heartbeat did not steer the actual movement commands")
	}
}

func TestPartDashRollIsBeatLocalAndApproximatelyThirtyThreePercent(t *testing.T) {
	w := &world{parts: []part{{X: 80}}}
	self := unit.Snapshot{}
	h := &heart{stream: 123}
	count := 0
	for beat := 1; beat <= 10000; beat++ {
		h.beats = beat
		x, y, ok := h.partDash(w, self)
		x2, y2, again := h.partDash(w, self)
		if ok != again || x != x2 || y != y2 {
			t.Fatal("extra perception calls rerolled a heartbeat")
		}
		if ok {
			count++
		}
	}
	if count < 3150 || count > 3450 {
		t.Fatalf("expected a 33%% branch over independent beats, got %d/10000", count)
	}
}

func TestNoCollectiblePartAndEchoLeaveNormalMovementIntact(t *testing.T) {
	self := unit.Snapshot{ID: 1, VX: 150, HP: 100}
	for _, parts := range [][]part{nil, {{Type: 0, Equipped: true, Y: 40}}, {{Type: 0, Flying: true, Y: 40}}} {
		w := &world{parts: parts}
		h := &heart{stream: 91, beats: 1}
		if _, _, ok := h.partDash(w, self); ok {
			t.Fatal("redirected without a collectible part")
		}
		out := make(chan unit.Cmd, 128)
		h.contract(unit.Context{ID: 1, Out: out}, w, unit.Sense{Self: self}, false)
		for len(out) > 0 {
			switch c := (<-out).(type) {
			case unit.SetFSDirection, unit.SetVelocity:
				t.Fatal("no-part branch changed the existing movement")
			case unit.AddFS:
				if c.UnitID == 1 && (c.DX != 1 || c.DY != 0) {
					t.Fatal("no-part branch changed normal dash heading")
				}
			}
		}
	}
	w := &world{parts: []part{{Y: 40}}}
	h := heartWithPartRoll(t, w, self, true)
	out := make(chan unit.Cmd, 128)
	h.contract(unit.Context{ID: 1, Out: out}, w, unit.Sense{Self: self}, true)
	for len(out) > 0 {
		switch (<-out).(type) {
		case unit.AddFS, unit.SetFSDirection, unit.SetVelocity:
			t.Fatal("weak echo introduced another dash or probability roll")
		}
	}
}

func TestRamPreservesChargeInsteadOfOverridingPartSeeking(t *testing.T) {
	a, ctx, out, s := testWeapon(t, 12)
	a.Handle(ctx, api.Pulse{Sense: s})
	a.Handle(ctx, api.Pulse{Sense: s})
	a.Handle(ctx, api.Pulse{Sense: s, SeekingPart: true})
	if len(out) != 0 {
		t.Fatal("ram overrode the reserved part-seeking movement")
	}
	a.Handle(ctx, api.Pulse{Sense: s})
	fired := false
	for len(out) > 0 {
		if fs, ok := (<-out).(unit.AddFS); ok && fs.UnitID == ctx.ID {
			fired = true
		}
	}
	if !fired {
		t.Fatal("yielding to a part consumed the stored ram charge")
	}
}
