package autochess

import (
	"testing"

	_ "xqdj/character"
)

func TestBondsAreSmallAndCoverTheShop(t *testing.T) {
	costOf, _ := buildCatalog()
	seen := map[string]string{}
	for _, def := range bondBook {
		if len(def.Members) < 2 || len(def.Members) > 4 {
			t.Fatalf("%s has %d members", def.Name, len(def.Members))
		}
		if len(def.Tiers) == 0 || def.Tiers[0].Need < 2 {
			t.Fatalf("%s tiers start too low", def.Name)
		}
		for _, kind := range def.Members {
			if costOf[kind] == 0 {
				t.Fatalf("%s member %s is not in the shop", def.Name, kind)
			}
			if prev, ok := seen[kind]; ok {
				t.Fatalf("%s is in both %s and %s", kind, prev, def.Name)
			}
			seen[kind] = def.Name
		}
	}
	for kind := range costOf {
		if seen[kind] == "" {
			t.Fatalf("%s has no bond", kind)
		}
	}
	if len(seen) != len(costOf) {
		t.Fatalf("bonds %d, shop %d", len(seen), len(costOf))
	}
}

func TestBondCountsUniqueFieldedKinds(t *testing.T) {
	board := []piece{{kind: "弓箭"}, {kind: "弓箭"}, {kind: "太刀"}}
	if n := uniqueBondCount(board, "怪猎组"); n != 2 {
		t.Fatalf("count = %d", n)
	}
	mod := modsFor(board, "弓箭")
	if mod.Dmg != 1.18 || mod.HP != 0 {
		t.Fatalf("pair mods = %+v", mod)
	}
	benchOnly := []piece{{kind: "酒翁"}}
	if modsFor(append(benchOnly, piece{kind: "炭翁"}), "酒翁").HP == 0 {
		t.Fatal("two elders should reach the first tier")
	}
	full := []piece{{kind: "弓箭"}, {kind: "太刀"}, {kind: "盾斧"}}
	if got := modsFor(full, "盾斧").Dmg; got != 1.32 {
		t.Fatalf("trio damage = %v", got)
	}
	if modsFor([]piece{{kind: "老虎机"}}, "老虎机").Dmg != 0 {
		t.Fatal("one of a pair should not activate")
	}
	if modsFor([]piece{{kind: "老虎机"}, {kind: "粉红色迷人的pink"}}, "老虎机").Dmg != 1.2 {
		t.Fatal("pair bond did not activate")
	}
}

func TestBotFieldsBondPairAheadOfALonePiece(t *testing.T) {
	s := New(1)
	s.bot.level = 2
	s.bot.bench = []piece{
		{id: 1, kind: "弓箭", star: 1},
		{id: 2, kind: "太刀", star: 1},
		{id: 3, kind: "酒翁", star: 1},
	}
	s.autofieldLocked(&s.bot)
	if len(s.bot.board) != 2 {
		t.Fatalf("board = %+v", s.bot.board)
	}
	got := map[string]bool{}
	for _, p := range s.bot.board {
		got[p.kind] = true
	}
	if !got["弓箭"] || !got["太刀"] || got["酒翁"] {
		t.Fatalf("board = %+v bench = %+v", s.bot.board, s.bot.bench)
	}
}
