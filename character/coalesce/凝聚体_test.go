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
