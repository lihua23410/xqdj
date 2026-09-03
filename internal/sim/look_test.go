package sim

import (
	"testing"
	"xqdj/character"
	unitpkg "xqdj/internal/unit"
)

func TestLooksComeFromCharacterSpecs(t *testing.T) {
	looks := unitpkg.Looks()
	m, ok := looks[character.KindMenreiki]
	if !ok || m.Color == "" || !m.Chroma {
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
	g, ok := looks[character.KindGlitch]
	if !ok || g.Color == "" || len(g.FX) == 0 || g.FX[0] != "glitch" {
		t.Fatalf("地慧星 look=%+v", g)
	}
	ghost, ok := looks[character.KindGlitchGhost]
	if !ok || ghost.Color == "" || len(ghost.FX) == 0 || ghost.FX[0] != "glitch-still" {
		t.Fatalf("地慧星残影 look=%+v", ghost)
	}
	sl, ok := looks[character.KindGlitchSlash]
	if !ok || len(sl.FX) == 0 || sl.FX[0] != "slash" {
		t.Fatalf("地慧星斩击 look=%+v", sl)
	}
}
