package sim

import (
	"fmt"
	"testing"
	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestFlickerVsIntegratedDoesNotRetainSidestepSpeed(t *testing.T) {
	for seed := uint64(1); seed <= 10; seed++ {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			m := NewHeadlessMatchSeeded(seed)
			m.SetSlot(0, character.KindFlicker)
			m.SetSlot(1, character.KindIntegrated)
			m.Start()
			t.Cleanup(m.End)
			m.mu.Lock()
			f := fighterSlot(m, 0)
			m.mu.Unlock()
			fastFor, lastTime := 0.0, 0.0
			sawSide := false
			for i := 0; i < 60*45; i++ {
				m.Tick()
				m.mu.Lock()
				now, speed, passing := m.time, f.v.len(), f.pass
				knocked, ended := len(f.fsList) > 0, m.phase != PhaseRunning
				m.mu.Unlock()
				if passing {
					sawSide = true
				}
				// A slide can start and hit a wall within one Tick, so allow
				// the next Sense to restore cruise before counting a leak.
				if !passing && !knocked && speed > 300 {
					fastFor += now - lastTime
				} else {
					fastFor = 0
				}
				lastTime = now
				if fastFor > .25 {
					t.Fatalf("t=%.3f: speed %.1f persisted for %.3fs after sidestep; want cruise 170", now, speed, fastFor)
				}
				if ended {
					break
				}
			}
			if !sawSide {
				t.Fatal("match never exercised a sidestep")
			}
		})
	}
}

func TestFlickerDamageSourceSurvivesProjectileRemoval(t *testing.T) {
	m := NewHeadlessMatchSeeded(2077)
	m.SetSlot(0, character.KindFlicker)
	m.SetSlot(1, character.KindRanged)
	m.Start()
	defer m.End()
	m.mu.Lock()
	defer m.mu.Unlock()
	f := fighterSlot(m, 0)
	enemy := fighterSlot(m, 1)
	got := tapEvents(f)
	bullet := m.addUnitLocked("子弹", vec{}, vec{}, enemy.id, 1)
	if bullet == nil {
		t.Fatal("missing projectile")
	}
	m.offerDamageLocked(unitpkg.Damage{From: bullet.id, To: f.id, Amount: 10})
	m.removeLocked(bullet)
	hits := eventsOf[unitpkg.IncomingDamage](got)
	if len(hits) != 1 || hits[0].SourceRole != unitpkg.RoleProjectile || hits[0].SourceKind != "子弹" || hits[0].SourceOwnerID != enemy.id {
		t.Fatalf("source metadata=%+v", hits)
	}
}

func TestFlickerDodgesLavaBeforeTakingDamage(t *testing.T) {
	m := NewHeadlessMatchSeeded(2077)
	m.SetField("火山")
	m.SetSlot(0, character.KindFlicker)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()
	m.mu.Lock()
	f := fighterSlot(m, 0)
	f.p = vec{-82, 0}
	f.setVel(vec{170, 0})
	startHP := f.hp
	m.mu.Unlock()

	dodged := false
	for i := 0; i < 24; i++ {
		m.Tick()
		m.mu.Lock()
		for _, fx := range m.fx {
			if fx.UnitID == f.id && fx.Name == "reflex" {
				dodged = true
			}
		}
		hp := f.hp
		m.mu.Unlock()
		if hp < startHP {
			t.Fatalf("lava hit before the predicted dodge: HP %g", hp)
		}
	}
	if !dodged {
		t.Fatal("expected a sidestep before entering the crater")
	}
}

func TestFlickerCanDamageWithoutEnemyProjectiles(t *testing.T) {
	m := NewHeadlessMatchSeeded(2077)
	m.SetSlot(0, character.KindFlicker)
	m.SetSlot(1, character.KindDummy)
	m.Start()
	defer m.End()
	for i := 0; i < 60*12; i++ {
		m.Tick()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	target := fighterSlot(m, 1)
	if target != nil && target.hp >= target.maxHP {
		t.Fatal("flicker never attacks a non-shooting enemy")
	}
}
