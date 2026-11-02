package audit

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"xqdj/internal/unit"
)

func TestRepositoryCollisionBoundary(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	if err := CheckSources(filepath.Join(filepath.Dir(file), "../..")); err != nil {
		t.Fatal(err)
	}
}

func TestSourceAuditFollowsCollisionHelpers(t *testing.T) {
	for _, tc := range []struct {
		name, handler, helpers string
		bad                    bool
	}{
		{"direct", "case u.Collision: ctx.Out <- u.Damage{}", "", true},
		{"receiver helper", "case u.Collision: a.hit(ctx)", "func(a *F) hit(ctx u.Context){ deal(ctx) }; func deal(ctx u.Context){ctx.Out <- u.Damage{}}", true},
		{"type alias", "case u.Collision: ctx.Out <- Hurt{}", "type Hurt = u.Damage", true},
		{"owner fan callback cannot duplicate damage", "case u.FanHit: ctx.Out <- u.Damage{}", "", true},
		{"conversion", "case u.Collision: var d u.Damage; ctx.Out <- u.Damage(d)", "", true},
		{"sense damage allowed", "case u.Sense: ctx.Out <- u.Damage{}; case u.Collision: a.touch()", "func(a *F) touch(){}", false},
		{"incoming confirmation allowed", "case u.Collision: a.touch(); case u.IncomingDamage: u.ConfirmHit(ctx,e)", "func(a *F) touch(){}", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "character", "fixture")
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			source := `package fixture; import u "xqdj/internal/unit"; type F struct{}; func init(){u.Register(u.Spec{Fighter:true},func(u.SpawnInfo)u.Actor{return &F{}})};func(a *F)Handle(ctx u.Context,ev u.Event){switch e:=ev.(type){` + tc.handler + `}};` + tc.helpers
			if err := os.WriteFile(filepath.Join(dir, "fighter.go"), []byte(source), 0644); err != nil {
				t.Fatal(err)
			}
			err := CheckSources(root)
			if (err != nil) != tc.bad {
				t.Fatalf("bad=%v error=%v", tc.bad, err)
			}
			if tc.bad && (!strings.Contains(err.Error(), "fighter.go:") || !strings.Contains(err.Error(), "Handle[")) {
				t.Fatal(err)
			}
		})
	}
}

func TestRuntimeRejectsBodyDamageEvenWithForgedSource(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("body collision escaped")
		}
	}()
	CheckCommand(Origin{ID: 1, Kind: "illegal", Role: unit.RoleFighter, Collision: true}, unit.Damage{From: 99, To: 2, Amount: 6})
}

func TestRuntimeAllowsFanAndNonContactDamage(t *testing.T) {
	CheckCommand(Origin{ID: 99, Role: unit.RoleProjectile, Fan: true, Collision: true}, unit.Damage{From: 99, To: 2, Amount: 6})
	CheckCommand(Origin{ID: 1, Role: unit.RoleFighter}, unit.Damage{From: 1, To: 2, Amount: 6})
	CheckCommand(Origin{ID: 1, Role: unit.RoleFighter, Collision: true}, unit.SetHP{UnitID: 1, HP: 70, MaxHP: 70})
}

func TestRuntimeRejectsFanQuotingAStranger(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("fan quoted a stranger")
		}
	}()
	CheckCommand(Origin{ID: 24, Role: unit.RoleProjectile, Fan: true}, unit.Damage{From: 2, To: 3, Amount: 6})
}

func TestRuntimeRejectsForgedFanNotification(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("fighter forged a fan notification")
		}
	}()
	CheckCommand(Origin{ID: 1, Role: unit.RoleFighter, Collision: true}, unit.FanContact{From: 99})
}
