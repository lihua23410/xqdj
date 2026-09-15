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
	eng, ok := looks[character.KindEngine]
	if !ok || eng.Color == "" || eng.VisionRing || len(eng.FX) == 0 || eng.FX[0] != "engine" {
		t.Fatalf("内燃机 look=%+v", eng)
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
	ep, ok := unitpkg.Packs()[character.KindEngine]
	if !ok {
		t.Fatal("missing engine pack")
	}
	have = map[string]bool{}
	for _, f := range ep.Files {
		have[f] = true
	}
	for _, f := range []string{"fx/engine.css", "fx/engine.js", "fx/shot.js"} {
		if !have[f] {
			t.Fatalf("内燃机 pack missing %s in %v", f, ep.Files)
		}
	}
	rd, ok := looks[character.KindRadar]
	if !ok || rd.Color == "" || len(rd.FX) == 0 || rd.FX[0] != "radar" {
		t.Fatalf("雷达 look=%+v", rd)
	}
	rp, ok := unitpkg.Packs()[character.KindRadar]
	if !ok {
		t.Fatal("missing radar pack")
	}
	have = map[string]bool{}
	for _, f := range rp.Files {
		have[f] = true
	}
	for _, f := range []string{"fx/radar.css", "fx/radar.js", "fx/shot.js"} {
		if !have[f] {
			t.Fatalf("雷达 pack missing %s in %v", f, rp.Files)
		}
	}
	pr, ok := looks[character.KindPrisoner]
	if !ok || pr.Color == "" || len(pr.FX) == 0 || pr.FX[0] != "prisoner" {
		t.Fatalf("囚徒 look=%+v", pr)
	}
	pp, ok := unitpkg.Packs()[character.KindPrisoner]
	if !ok {
		t.Fatal("missing prisoner pack")
	}
	have = map[string]bool{}
	for _, f := range pp.Files {
		have[f] = true
	}
	for _, f := range []string{"fx/prisoner.css", "fx/prisoner.js", "fx/shot.js", "fx/gallows.png", "fx/chair.svg"} {
		if !have[f] {
			t.Fatalf("囚徒 pack missing %s in %v", f, pp.Files)
		}
	}
	cg, ok := looks[character.KindCage]
	if !ok || !cg.Overlay || len(cg.FX) == 0 || cg.FX[0] != "cage" {
		t.Fatalf("囚笼 look=%+v", cg)
	}
	gl, ok := looks[character.KindGallows]
	if !ok || !gl.Overlay || len(gl.FX) == 0 || gl.FX[0] != "gallows" {
		t.Fatalf("绞刑架 look=%+v", gl)
	}
	ch, ok := looks[character.KindChair]
	if !ok || !ch.Overlay || len(ch.FX) == 0 || ch.FX[0] != "chair" {
		t.Fatalf("电椅 look=%+v", ch)
	}
	rp2, ok := looks[character.KindReaper]
	if !ok || rp2.Color == "" || len(rp2.FX) == 0 || rp2.FX[0] != "reaper" {
		t.Fatalf("收割者 look=%+v", rp2)
	}
	sk, ok := looks[character.KindSickle]
	if !ok || !sk.Overlay || len(sk.FX) == 0 || sk.FX[0] != "sickle" {
		t.Fatalf("收割者镰刀 look=%+v", sk)
	}
	rpk, ok := unitpkg.Packs()[character.KindReaper]
	if !ok {
		t.Fatal("missing reaper pack")
	}
	have = map[string]bool{}
	for _, f := range rpk.Files {
		have[f] = true
	}
	for _, f := range []string{"fx/reaper.css", "fx/sickle.css", "fx/sickle.js", "fx/shot.js"} {
		if !have[f] {
			t.Fatalf("收割者 pack missing %s in %v", f, rpk.Files)
		}
	}
}
