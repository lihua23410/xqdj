package sim

import (
	"sync"
	"testing"
	"time"

	"xqdj/character"
	非因果 "xqdj/character/acausal"
	unitpkg "xqdj/internal/unit"
	奇美拉 "xqdj/map/chimera"
	圆 "xqdj/map/circle"
)

func TestConcurrentMatchProbeIsolation(t *testing.T) {
	a, b := NewMatchSeeded(19), NewMatchSeeded(27)
	foes := []string{character.KindKnight, character.KindMelee}
	matches := []*Match{a, b}
	var wg sync.WaitGroup
	for i, m := range matches {
		m.SetSlot(0, 非因果.KindAcausal)
		m.SetSlot(1, foes[i])
		t.Cleanup(m.End)
		wg.Add(1)
		go func() { defer wg.Done(); m.Start() }()
	}
	wg.Wait()

	// 普通局的创建、开局与重置，都不能清掉其他场的探针。
	other := NewMatchSeeded(30)
	other.SetSlot(0, character.KindDummy)
	other.SetSlot(1, character.KindDummy)
	other.Start()
	other.End()
	other.Start()
	other.End()
	for i, m := range matches {
		tape := m.probeTape
		if tape == nil || len(tape.Fighters) != 1 || tape.Fighters[0].Kind != foes[i] {
			t.Fatalf("match %d has another match's probe: %+v", i, tape)
		}
		// 尚未 Tick，演员协程没有收到事件。同步启动该演员，验证出生参数
		// 确实绑定了本场记录，而不是在首次 Sense 时查询全局记录。
		var actor *unit
		for _, u := range m.units {
			if u.kind == 非因果.KindAcausal {
				actor = u
				break
			}
		}
		if actor == nil {
			t.Fatal("missing acausal actor")
		}
		out := make(chan unitpkg.Cmd, 256)
		actor.actor.Handle(unitpkg.Context{ID: actor.id, Out: out, Stream: actor.stream}, unitpkg.Sense{Self: actor.snap(), Field: m.liveFieldLocked()})
		found := false
		for len(out) > 0 {
			if spawn, ok := (<-out).(unitpkg.Spawn); ok && spawn.Kind == 非因果.KindSelfGhost {
				found = true
				if spawn.X != tape.Self[0].X || spawn.Y != tape.Self[0].Y {
					t.Fatalf("match %d spawned the wrong ghost", i)
				}
			}
		}
		if !found {
			t.Fatalf("match %d lost its probe before actor boot", i)
		}
	}
	before := b.probeTape
	a.End()
	if b.probeTape != before {
		t.Fatal("ending one match changed the other probe")
	}
}

func TestConcurrentMatchBirthFieldIsolation(t *testing.T) {
	for _, name := range []string{圆.Name, 奇美拉.Name} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m := NewMatchSeeded(1)
			m.UseTeam()
			m.SetField(name)
			m.SetSlot(0, character.KindDummy)
			m.SetSlot(1, character.KindDummy)
			m.Start()
			defer m.End()
			// 星弹在第一次 Sense 前就碰墙，碎片仍应按出生场地约束。
			m.mu.Lock()
			u := m.addUnitLocked("回声星弹", vec{}, vec{}, 0, 0)
			field := m.liveFieldLocked()
			m.mu.Unlock()
			u.inbox <- unitpkg.WallHit{X: 500, Y: 500}
			deadline := time.After(3 * time.Second)
			shards := 0
			for shards < 16 {
				select {
				case cmd := <-m.cmds:
					if spawn, ok := cmd.(unitpkg.Spawn); ok && spawn.Kind == "回声碎片" {
						shards++
						if !field.Walkable(spawn.X, spawn.Y, 4) {
							t.Fatalf("fragment (%v,%v) outside %s", spawn.X, spawn.Y, name)
						}
					}
				case <-deadline:
					t.Fatalf("only got %d fragments", shards)
				}
			}
		})
	}
}

type fieldObserver struct{ fields chan unitpkg.Field }

func (a fieldObserver) Handle(ctx unitpkg.Context, _ unitpkg.Event) { a.fields <- ctx.Field }

func TestActorRetainsLatestOwnField(t *testing.T) {
	for _, f := range []unitpkg.Field{圆.Field(), 奇美拉.Field()} {
		t.Run(f.Name, func(t *testing.T) {
			t.Parallel()
			observed := make(chan unitpkg.Field, 3)
			u := &unit{actor: fieldObserver{observed}, inbox: make(chan unitpkg.Event, 3), stop: make(chan struct{}), exited: make(chan struct{})}
			go runActor(u, unitpkg.Context{Field: f})
			defer func() { close(u.stop); <-u.exited }()
			u.inbox <- unitpkg.WallHit{}
			updated := f
			updated.Walls = []unitpkg.FieldWall{{X1: 123, X2: 456}}
			u.inbox <- unitpkg.Sense{Field: updated}
			u.inbox <- unitpkg.IncomingDamage{}
			for i := 0; i < 3; i++ {
				select {
				case got := <-observed:
					if got.Name != f.Name {
						t.Fatalf("got %s, want %s", got.Name, f.Name)
					}
					if i > 0 && (len(got.Walls) != 1 || got.Walls[0].X1 != 123) {
						t.Fatal("lost updated walls after Sense")
					}
				case <-time.After(3 * time.Second):
					t.Fatal("actor did not receive event")
				}
			}
		})
	}
}
