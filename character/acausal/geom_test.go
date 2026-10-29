package 非因果

import (
	"testing"

	"xqdj/internal/unit"
)

func TestBodyShieldsGhostAtSameContactTime(t *testing.T) {
	for _, ghostKind := range []int{hitFighterGhost, hitMinionGhost} {
		for _, tc := range []struct {
			name          string
			bodyX, ghostX float64
			bodyR, ghostR float64
			wantBody      bool
		}{
			{name: "coincident", bodyX: 40, ghostX: 40, bodyR: 18, ghostR: 18, wantBody: true},
			{name: "body ahead", bodyX: 30, ghostX: 40, bodyR: 18, ghostR: 18, wantBody: true},
			{name: "ghost ahead", bodyX: 40, ghostX: 30, bodyR: 18, ghostR: 18},
			{name: "larger body", bodyX: 40, ghostX: 40, bodyR: 20, ghostR: 18, wantBody: true},
			{name: "larger ghost", bodyX: 40, ghostX: 40, bodyR: 18, ghostR: 20},
			{name: "start inside both", bodyR: 18, ghostR: 18, wantBody: true},
			{name: "tie within tolerance", bodyX: 40 + 2e-8, ghostX: 40, bodyR: 18, ghostR: 18, wantBody: true},
			{name: "ghost earlier outside tolerance", bodyX: 40 + 2e-6, ghostX: 40, bodyR: 18, ghostR: 18},
		} {
			t.Run(tc.name+map[int]string{hitFighterGhost: "/fighter", hitMinionGhost: "/minion"}[ghostKind], func(t *testing.T) {
				body := circle{id: 3, x: tc.bodyX, r: tc.bodyR, kind: hitBody}
				ghost := circle{id: 9, x: tc.ghostX, r: tc.ghostR, kind: ghostKind}
				wantKind, wantID := ghostKind, ghost.id
				if tc.wantBody {
					wantKind, wantID = hitBody, body.id
				}
				for _, circles := range [][]circle{{ghost, body}, {body, ghost}} {
					kind, id := stepShot(0, 0, 80, 0, 5, circles, nil, nil)
					if kind != wantKind || id != wantID {
						t.Errorf("first candidate=%d: got kind=%d id=%d, want kind=%d id=%d", circles[0].kind, kind, id, wantKind, wantID)
					}
				}
			})
		}
	}
}

func TestWallAndBoundaryShieldCoincidentGhost(t *testing.T) {
	ghosts := []circle{{id: 9, r: 18, kind: hitFighterGhost}, {id: 10, r: 14, kind: hitMinionGhost}}
	kind, _ := stepShot(0, 0, 40, 0, 5, ghosts,
		[]wallSeg{{y1: -30, y2: 30, r: 2}}, nil)
	if kind != hitWall {
		t.Fatalf("wall tie hit kind=%d", kind)
	}
	kind, _ = stepShot(0, 0, 40, 0, 5, ghosts, nil, func(float64, float64) (float64, bool) { return 0, true })
	if kind != hitOut {
		t.Fatalf("boundary tie hit kind=%d", kind)
	}
}

func TestBodyConsumesBulletWithoutDamagingCoincidentShadow(t *testing.T) {
	a := &非因果{
		tape: &Tape{Fighters: []FighterTrack{{
			Slot: 1, MaxHP: 100, Radius: 18, Samples: []Sample{{X: 40}},
		}}},
		fighters: []ghostRef{{id: 9, ready: true}}, shadow: []float64{100},
		shotID: 10, shotLive: true, shotR: 5, shotDmg: 10, shotSpeed: 2400,
	}
	out := make(chan unit.Cmd, 16)
	a.stepBullet(unit.Context{ID: 7, Kind: KindAcausal, Out: out}, unit.Sense{
		Time:   1,
		Nearby: []unit.Snapshot{{ID: 3, Role: unit.RoleFighter, Slot: 1, HP: 100, Radius: 18, X: 40}},
	})
	if a.shadow[0] != 100 {
		t.Fatalf("body failed to shield shadow: HP=%v", a.shadow[0])
	}
	if a.shotLive || a.shotID != 0 || !a.shotWait || a.nextShot != 1+shotWait {
		t.Fatal("blocked bullet did not disappear and start its normal cooldown")
	}
	despawned := false
	for len(out) > 0 {
		switch cmd := (<-out).(type) {
		case unit.Despawn:
			despawned = cmd.UnitID == 10
		case unit.Damage:
			t.Fatalf("blocked bullet dealt damage: %+v", cmd)
		case unit.SetHP:
			if cmd.UnitID == 9 {
				t.Fatal("blocked bullet updated shadow HP")
			}
		}
	}
	if !despawned {
		t.Fatal("blocked bullet entity was not removed")
	}
}
