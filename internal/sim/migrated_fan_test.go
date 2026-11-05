package sim

import (
	"math"
	"testing"
	"xqdj/internal/audit"
	unitpkg "xqdj/internal/unit"
)

func TestMigratedFanGeometryHasRearGap(t *testing.T) {
	for kind, degrees := range map[string]float64{
		"pink贴身扇环": 180, "pink电锯扇环": 144, "盾斧突刺扇环": 120, "回声突进扇环": 90, "钓鱼佬高速扇环": 180, "血契饿扑扇环": 110, "教父青年撞击扇环": 90, "我与周旋久本体扇环": 180, "周旋久扇环": 180, "人偶使冲刺扇环": 120,
	} {
		t.Run(kind, func(t *testing.T) {
			spec, ok := unitpkg.Lookup(kind)
			if !ok || !spec.Attach || math.Abs(spec.ArcSpan-unitpkg.Deg(degrees)) > 1e-9 {
				t.Fatalf("bad spec %+v", spec)
			}
			arc := ccdShape{p: vec{}, v: vec{}, face: vec{1, 0}, r: spec.Radius, arcInner: spec.ArcInner, arcSpan: spec.ArcSpan}
			front := ccdShape{p: vec{spec.Radius + 28, 0}, v: vec{-20, 0}, r: 18}
			back := ccdShape{p: vec{-spec.Radius - 28, 0}, v: vec{20, 0}, r: 18}
			if _, _, hit := sweptPairShapes(arc, front, 1); !hit {
				t.Fatal("missed front")
			}
			if _, _, hit := sweptPairShapes(arc, back, 1); hit {
				t.Fatal("hit behind")
			}
		})
	}
}

func TestPartnerFansCanContactTheirOwnerWithoutChangingOrdinaryFans(t *testing.T) {
	owner := &unit{id: 1, role: unitpkg.RoleFighter, slot: 0}
	ally := &unit{id: 2, owner: 1, role: unitpkg.RoleMinion, mortal: true, slot: 0}
	fan := &unit{id: 3, owner: 2, role: unitpkg.RoleProjectile, slot: 0, attach: true, attachFriends: true}
	if !attachHits(fan, owner) || !attachHits(fan, ally) {
		t.Fatal("partner fan lost friendly contact")
	}
	fan.attachFriends = false
	if attachHits(fan, owner) || attachHits(fan, ally) {
		t.Fatal("ordinary fan acquired friendly contact")
	}
}

func TestEnginePanicsOnActualFighterCollisionSender(t *testing.T) {
	for name, event := range map[string]unitpkg.Event{"body": unitpkg.Collision{}, "fan notification": unitpkg.FanHit{}} {
		t.Run(name, func(t *testing.T) {
			m := newDeliveryMatch(t)
			m.mu.Lock()
			defer m.mu.Unlock()
			actor := m.units[m.order[0]]
			actor.actor = deliveryActorFunc(func(ctx unitpkg.Context, ev unitpkg.Event) {
				switch ev.(type) {
				case unitpkg.Collision, unitpkg.FanHit:
					ctx.Out <- unitpkg.Damage{From: 987654, To: actor.id, Amount: 5}
				}
			})
			m.send(actor, event)
			defer func() {
				if recover() == nil {
					t.Fatal("forged From bypassed engine contact guard")
				}
			}()
			m.settleHitsLocked()
		})
	}
}

func TestRuntimeGuardKeepsOtherFighterEventsLegal(t *testing.T) {
	m := NewMatchSeeded(1)
	m.applyCmdLocked(actorCommand{origin: audit.Origin{ID: 1, Role: unitpkg.RoleFighter}, command: unitpkg.Damage{From: 1, To: 2, Amount: 1}})
}

func TestFanTurnsWithOwnerBeforeReturningSweep(t *testing.T) {
	m := NewMatchSeeded(1)
	owner := &unit{id: 1, p: vec{}, v: vec{-640, 0}, face: vec{1, 0}}
	fan := &unit{id: 2, owner: 1, attach: true, p: vec{}, v: vec{640, 0}, face: vec{1, 0}, arcSpan: math.Pi / 2, arcInner: 18, radius: 20}
	m.units = map[uint64]*unit{1: owner, 2: fan}
	m.order = []uint64{1, 2}
	m.stickFollowersLocked()
	if fan.face.X > -.99 || fan.v.X != -640 {
		t.Fatalf("stale returning fan: %+v", fan)
	}
	// The old target is now behind this return's direction.
	if _, _, hit := sweptPairShapes(fan.shape(), ccdShape{p: vec{40, 0}, r: 18}, .02); hit {
		t.Fatal("return damaged the old rear target")
	}
}

func TestContactFanMatchesRunWithAuditedSenders(t *testing.T) {
	for _, kind := range []string{"粉红色迷人的pink", "盾斧", "回声", "钓鱼佬", "血契", "教父(青年)", "我与周旋久", "人偶使"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			m := NewHeadlessMatchSeeded(23)
			m.SetSlot(0, kind)
			m.SetSlot(1, "靶子")
			m.Start()
			defer m.End()
			for i := 0; i < 240; i++ {
				m.Tick()
			}
		})
	}
}
