package sim

import (
	"math"
	"testing"

	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestFisherPoliceArrest(t *testing.T) {
	for _, name := range []string{"单挑", "组队"} {
		t.Run(name, func(t *testing.T) {
			m := NewMatchSeeded(9)
			defer m.End()
			if name == "组队" {
				m.UseTeam()
				m.SetSideLen(0, 2)
				m.SetSideLen(1, 1)
				m.SetSeat(0, 0, character.KindFisher)
				m.SetSeat(0, 1, character.KindDummy)
				m.SetSeat(1, 0, character.KindDummy)
				m.SetField("六边形")
			} else {
				m.SetSlot(0, character.KindFisher)
				m.SetSlot(1, character.KindDummy)
			}
			m.Start()

			m.mu.Lock()
			fisher := m.livingFighterLocked(0)
			if fisher == nil || fisher.kind != character.KindFisher {
				m.mu.Unlock()
				t.Fatal("钓鱼佬没有上场")
			}
			fisherID := fisher.id
			fisher.p = vec{}
			m.holdStillLocked(fisherID, 20)
			m.applyCmdLocked(unitpkg.Spawn{Kind: "钓鱼佬出警", OwnerID: fisherID, Data: 3.0})
			m.mu.Unlock()

			var siren, car, officer, arrest bool
			var lastOfficerPos vec
			for i := 0; i < 12*TickHz && !arrest; i++ {
				m.Tick()
				for _, fx := range m.fx {
					switch fx.Name {
					case "police-siren":
						siren = true
						if math.Abs(m.time-3) > 2*DT {
							t.Fatalf("警笛时间 %.3f，应为第 3 秒", m.time)
						}
					case "arrest":
						arrest = fx.UnitID == fisherID
					}
				}
				for _, id := range m.order {
					u := m.units[id]
					if u.kind == "钓鱼佬警车" && !car {
						car = true
						if !siren || m.liveFieldLocked().OutlineContains(u.p.X, u.p.Y, 0) {
							t.Fatal("警车应随警笛从场外出现")
						}
					}
					if u.kind == "钓鱼佬警察" {
						if u.radius != fisher.radius || u.v.len() > 50+1e-6 {
							t.Fatalf("警察应与战斗机同大、速度不超过 50: radius=%v speed=%v", u.radius, u.v.len())
						}
						if officer && u.p.sub(lastOfficerPos).len() > 50*DT+1e-6 {
							t.Fatal("警察实际位移超过每秒 50 的上限")
						}
						lastOfficerPos = u.p
						officer = true
						if !car || m.time < 4.25 {
							t.Fatal("警察应在警车驶入停车后出现")
						}
					}
				}
			}
			if !siren || !car || !officer || !arrest || m.units[fisherID] != nil {
				t.Fatalf("出警流程缺失: 警笛=%v 警车=%v 警察=%v 逮捕=%v", siren, car, officer, arrest)
			}
			if name == "单挑" {
				if m.phase != PhaseEnded || m.winner != character.KindDummy {
					t.Fatalf("单挑未判对手获胜: phase=%v winner=%q", m.phase, m.winner)
				}
			} else if m.phase != PhaseRunning || m.livingFighterLocked(0) == nil {
				t.Fatal("组队逮捕应只让钓鱼佬出局")
			}
		})
	}
}
