package 凝聚体

import (
	"testing"

	"xqdj/internal/unit"
)

func TestNoOwnChunkLeavesAim(t *testing.T) {
	a, out := newCoalesce()
	sense(a, out, 1, nil)
	if aims := aimCmds(drain(out)); len(aims) != 0 {
		t.Fatalf("no chunk should not rewrite aim: %v", aims)
	}
}

func TestOwnChunkCoversBodyOnce(t *testing.T) {
	a, out := newCoalesce()
	chunk := ownChunk(1, 2, 200, 0)
	sense(a, out, 1, []unit.Snapshot{chunk})
	aims := aimCmds(drain(out))
	if len(aims) != 1 || aims[0].Value != bodyAimCovered || aims[0].UnitID != 1 {
		t.Fatalf("first own chunk should set body to %d: %v", bodyAimCovered, aims)
	}

	sense(a, out, 1, []unit.Snapshot{chunk})
	if aims = aimCmds(drain(out)); len(aims) != 0 {
		t.Fatalf("chunk still present should not rewrite: %v", aims)
	}
}

func TestLastChunkRestoresBodyAim(t *testing.T) {
	a, out := newCoalesce()
	sense(a, out, 1, []unit.Snapshot{ownChunk(1, 2, 200, 0)})
	drain(out)

	sense(a, out, 1, nil)
	aims := aimCmds(drain(out))
	if len(aims) != 1 || aims[0].Value != unit.DefaultFighterAim || aims[0].UnitID != 1 {
		t.Fatalf("last chunk gone should restore %d: %v", unit.DefaultFighterAim, aims)
	}
}

func TestOtherChunkDoesNotCover(t *testing.T) {
	a, out := newCoalesce()
	other := ownChunk(99, 2, 200, 0)
	sense(a, out, 1, []unit.Snapshot{other})
	if aims := aimCmds(drain(out)); len(aims) != 0 {
		t.Fatalf("another coalesce's chunk should not cover: %v", aims)
	}
}

func TestAbsorbedChunkDoesNotCover(t *testing.T) {
	a, out := newCoalesce()
	sense(a, out, 1, []unit.Snapshot{ownChunk(1, 2, 10, 0)})
	if aims := aimCmds(drain(out)); len(aims) != 0 {
		t.Fatalf("chunk absorbed this sense should not cover: %v", aims)
	}
}

func TestChunkHalvesDamage(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	c := &massChunk{owner: 1, slot: 0, x: 3, y: 4, hp: 1}
	ctx := unit.Context{ID: 9, Kind: massKind(3), Out: out}
	c.Handle(ctx, unit.IncomingDamage{Token: 7, Amount: 4})
	cmds := drain(out)
	got, ok := confirmOf(cmds)
	if !ok || got.Amount != 2 || got.Token != 7 || got.UnitID != 9 {
		t.Fatalf("4 点报价该减成 2: %+v %v", got, ok)
	}
	fx := breakFX(cmds)
	if fx == nil || fx.Amount != 2 {
		t.Fatalf("减完仍够打碎，该播破裂: %+v", fx)
	}
}

func TestChunkSurvivesBelowOne(t *testing.T) {
	out := make(chan unit.Cmd, 8)
	c := &massChunk{owner: 1, slot: 0, hp: 1}
	ctx := unit.Context{ID: 9, Kind: massKind(1), Out: out}
	c.Handle(ctx, unit.IncomingDamage{Token: 8, Amount: 1})
	cmds := drain(out)
	got, ok := confirmOf(cmds)
	if !ok || got.Amount != 0.5 {
		t.Fatalf("1 点报价该减成 0.5: %+v %v", got, ok)
	}
	if fx := breakFX(cmds); fx != nil {
		t.Fatalf("减完还剩半血，不该碎: %+v", fx)
	}
	if c.hp != 0.5 {
		t.Fatalf("hp=%v", c.hp)
	}
}

func confirmOf(cmds []unit.Cmd) (unit.ConfirmDamage, bool) {
	for _, c := range cmds {
		if d, ok := c.(unit.ConfirmDamage); ok {
			return d, true
		}
	}
	return unit.ConfirmDamage{}, false
}

func breakFX(cmds []unit.Cmd) *unit.FX {
	for _, c := range cmds {
		if fx, ok := c.(unit.FX); ok && fx.Name == "coalesce-chunk-break" {
			return &fx
		}
	}
	return nil
}

func TestChunkSpecAim(t *testing.T) {
	for mass := minChunkMass; mass <= maxChunkMass; mass++ {
		spec, ok := unit.Lookup(massKind(mass))
		if !ok {
			t.Fatalf("missing %s", massKind(mass))
		}
		if spec.AimPriority != chunkAim || !spec.Mortal {
			t.Fatalf("%s aim=%d mortal=%v", spec.Kind, spec.AimPriority, spec.Mortal)
		}
		shot, ok := unit.Lookup(shotKind(mass))
		if !ok {
			t.Fatalf("missing %s", shotKind(mass))
		}
		if shot.AimPriority != 0 || shot.Role != unit.RoleProjectile {
			t.Fatalf("%s aim=%d role=%s", shot.Kind, shot.AimPriority, shot.Role)
		}
	}
}

func newCoalesce() (*凝聚体, chan unit.Cmd) {
	return &凝聚体{}, make(chan unit.Cmd, 32)
}

func sense(a *凝聚体, out chan unit.Cmd, id uint64, nearby []unit.Snapshot) {
	a.Handle(unit.Context{ID: id, Kind: KindCoalesce, Out: out}, unit.Sense{
		Self: unit.Snapshot{
			ID: id, Kind: KindCoalesce, Role: unit.RoleFighter,
			Radius: bodyRadius, HP: bodyHP, MaxHP: bodyHP,
			AimPriority: unit.DefaultFighterAim,
		},
		Nearby: nearby,
	})
}

func ownChunk(owner, id uint64, x, y float64) unit.Snapshot {
	return unit.Snapshot{
		ID: id, Kind: massKind(1), OwnerID: owner,
		X: x, Y: y, Radius: 6, Mortal: true, AimPriority: chunkAim,
	}
}

func drain(out chan unit.Cmd) []unit.Cmd {
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

func aimCmds(cmds []unit.Cmd) []unit.SetAimPriority {
	var aims []unit.SetAimPriority
	for _, c := range cmds {
		if v, ok := c.(unit.SetAimPriority); ok {
			aims = append(aims, v)
		}
	}
	return aims
}
