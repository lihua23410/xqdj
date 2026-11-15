package ospf

import (
	"reflect"
	"sync"
	"testing"

	"xqdj/internal/unit"
)

// 报文的下一跳和学习在两只协程里同时碰同一张表。以前这会把进程打崩。
func TestLinkStateReadAndWriteCanOverlap(t *testing.T) {
	n := &network{}
	r := addRouter(n, 0, point{})
	key := lsaKey{kind: routerLSA, advertiser: r}
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for seq := uint64(1); seq <= 300; seq++ {
				r.learn(announcement{lsaKey: key, sequence: seq})
			}
		}()
		go func() {
			defer wg.Done()
			f := flood{announcement: announcement{lsaKey: key, sequence: 150}}
			for i := 0; i < 300; i++ {
				n.nextLocal(r, f)
			}
		}()
	}
	wg.Wait()
}

func TestForkedNetworkDoesNotShareTables(t *testing.T) {
	ctx, stop := drainContext(t)
	defer stop()
	a := &fighter{
		layout:   areaMap{radius: 80, areas: []int{1, 0, 2}},
		birth:    point{30, 0},
		nextSend: sendInterval,
	}
	for i := 0; i < 40; i++ {
		a.Handle(ctx, unit.Sense{Time: float64(i), Self: unit.Snapshot{X: 30, Slot: 0}})
	}
	if a.net == nil || a.net.routers == 0 {
		t.Fatal("network did not install")
	}
	b := unit.ForkActor(a, unit.NewFork()).(*fighter)
	if b.net == a.net {
		t.Fatal("fork kept the parent network")
	}
	for area := range a.net.areas {
		if len(a.net.areas[area]) != len(b.net.areas[area]) {
			t.Fatalf("area %d routers %d vs %d", area, len(a.net.areas[area]), len(b.net.areas[area]))
		}
		for i, src := range a.net.areas[area] {
			dst := b.net.areas[area][i]
			if dst == src {
				t.Fatalf("router %d in area %d is shared", src.id, area)
			}
			if src.db != nil && dst.db != nil && mapPtr(src.db.rec) == mapPtr(dst.db.rec) && len(src.db.rec) > 0 {
				t.Fatalf("router %d shares its link-state table", src.id)
			}
			if mapPtr(src.imported) != 0 && mapPtr(src.imported) == mapPtr(dst.imported) {
				t.Fatalf("router %d shares imported summaries", src.id)
			}
		}
	}
	if mapPtr(a.net.translated) != 0 && mapPtr(a.net.translated) == mapPtr(b.net.translated) {
		t.Fatal("shared translation table")
	}
	if mapPtr(a.net.sent) != 0 && mapPtr(a.net.sent) == mapPtr(b.net.sent) {
		t.Fatal("shared cable table")
	}
}

func mapPtr(m any) uintptr { return reflect.ValueOf(m).Pointer() }

func drainContext(t *testing.T) (unit.Context, func()) {
	t.Helper()
	out := make(chan unit.Cmd, 64)
	done := make(chan struct{})
	go func() {
		for range out {
		}
		close(done)
	}()
	return unit.Context{ID: 1, Out: out, State: &unit.MatchState{}, Field: unit.HexField()}, func() {
		close(out)
		<-done
	}
}
