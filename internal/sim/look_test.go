package sim

import (
	"testing"
	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestLooksComeFromCharacterSpecs(t *testing.T) {
	looks := unitpkg.Looks()
	m, ok := looks[character.KindMenreiki]
	if !ok || m.Color == "" || len(m.FX) == 0 || m.FX[0] != "chroma" {
		t.Fatalf("面灵气 look=%+v", m)
	}
	w, ok := looks[character.KindWaller]
	if !ok || w.WallGuide != 155 {
		t.Fatalf("筑墙者 look=%+v", w)
	}
	if looks["紫弹"].Glow == false || looks["紫弹"].Trail == false {
		t.Fatalf("紫弹 look=%+v", looks["紫弹"])
	}
	if looks[character.KindMelee].VisionRing == false {
		t.Fatalf("近战 look=%+v", looks[character.KindMelee])
	}
	if looks["面具青"].Color != "#3ec8e0" || looks["面具红"].Color != "#ff3b3b" {
		t.Fatalf("mask looks 青=%+v 红=%+v", looks["面具青"], looks["面具红"])
	}
	if looks["面具紫"].Color != "#b44cff" || looks["面具苍"].Color != "#8dffb0" {
		t.Fatalf("mask looks 紫=%+v 苍=%+v", looks["面具紫"], looks["面具苍"])
	}
	pack, ok := unitpkg.Packs()[character.KindMenreiki]
	if !ok || pack.Base != "/ball/面灵气" {
		t.Fatalf("面灵气 pack=%+v", pack)
	}
	wantFiles := []string{"faction/qing.png", "faction/hong.png", "faction/zi.png", "faction/cang.png", "fx/faction.js"}
	have := map[string]bool{}
	for _, f := range pack.Files {
		have[f] = true
	}
	for _, f := range wantFiles {
		if !have[f] {
			t.Fatalf("面灵气 pack missing %s in %v", f, pack.Files)
		}
	}
	if len(pack.Factions) != 4 {
		t.Fatalf("面灵气 factions=%+v", pack.Factions)
	}
	wantIcon := map[string]string{
		"青": "/ball/面灵气/faction/qing.png",
		"红": "/ball/面灵气/faction/hong.png",
		"紫": "/ball/面灵气/faction/zi.png",
		"苍": "/ball/面灵气/faction/cang.png",
	}
	for _, f := range pack.Factions {
		if wantIcon[f.ID] != f.Icon || f.Color == "" {
			t.Fatalf("面灵气 faction %+v", f)
		}
	}
	f, err := unitpkg.BallFS().Open("面灵气/faction/qing.png")
	if err != nil {
		t.Fatalf("open faction icon: %v", err)
	}
	_ = f.Close()
	g, ok := looks[character.KindGlitch]
	if !ok || g.Color == "" || g.Base == "" || len(g.FX) == 0 || g.FX[0] != "glitch" {
		t.Fatalf("地慧星 look=%+v", g)
	}
	ghost, ok := looks[character.KindGlitchGhost]
	if !ok || ghost.Color == "" || ghost.Base != g.Base || len(ghost.FX) == 0 || ghost.FX[0] != "glitch-still" {
		t.Fatalf("地慧星残影 look=%+v", ghost)
	}
	sl, ok := looks[character.KindGlitchSlash]
	if !ok || !sl.Overlay || len(sl.FX) == 0 || sl.FX[0] != "slash" {
		t.Fatalf("地慧星斩击 look=%+v", sl)
	}
	shot, ok := looks[character.KindGlitchShot]
	if !ok || shot.Color == "" || !shot.Trail || len(shot.FX) == 0 || shot.FX[0] != "crescent" {
		t.Fatalf("地慧星弹 look=%+v", shot)
	}
	if looks["紫弹"].Overlay == false {
		t.Fatalf("紫弹 should overlay=%+v", looks["紫弹"])
	}
	gp, ok := unitpkg.Packs()[character.KindGlitch]
	if !ok {
		t.Fatal("missing glitch pack")
	}
	have = map[string]bool{}
	for _, f := range gp.Files {
		have[f] = true
	}
	for _, f := range []string{"fx/shot.js", "fx/iai.mp3"} {
		if !have[f] {
			t.Fatalf("地慧星 pack missing %s in %v", f, gp.Files)
		}
	}
}
