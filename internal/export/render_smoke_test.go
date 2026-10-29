package export

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	非因果 "xqdj/character/acausal"
	_ "xqdj/character/knight"
	"xqdj/internal/replay"
	"xqdj/internal/unit"
	"xqdj/internal/web"
)

// 显式开启：XQDJ_EXPORT_SMOKE=1 go test ./internal/export -run TestRenderSmoke -v
// 使用真实页面、Chromium、截图和 ffmpeg，缩短的仅是片头和定格段。
func TestRenderSmoke(t *testing.T) {
	if os.Getenv("XQDJ_EXPORT_SMOKE") != "1" {
		t.Skip("set XQDJ_EXPORT_SMOKE=1 for browser/video verification")
	}
	chrome, ffmpeg := findChrome(), findFFmpeg()
	if chrome == "" || ffmpeg == "" {
		t.Fatal("smoke test needs Chromium and ffmpeg")
	}
	for _, tc := range []struct {
		name        string
		probe, team bool
		banner      bool
	}{{"ordinary", false, false, false}, {"probe", true, false, false}, {"team-probe", true, true, false}, {"banner", false, false, true}, {"sequence", true, false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			r := smokeRecording(tc.probe, tc.team)
			if tc.banner {
				p := replay.Presentation{ID: "banner-1", Pack: "演出测试", Type: "banner", Duration: 0.12, Audio: !tc.probe, Data: json.RawMessage(`{"message":"准备开场"}`)}
				r.Presentations = append([]replay.Presentation{p}, r.Presentations...)
			}
			tl := newTimeline(r)
			tl.TitleFrames, tl.HoldStartFrames, tl.EndFrames = 3, 2, 3
			tl.TotalFrames = tl.TitleFrames + tl.OpeningFrames + tl.HoldStartFrames + tl.BattleFrames + tl.EndFrames
			dir := t.TempDir()
			if root := os.Getenv("XQDJ_EXPORT_ARTIFACTS"); root != "" {
				dir = filepath.Join(root, tc.name)
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			j := &Job{ID: tc.name, Frames: r.Frames, Initial: r.Initial, Presentations: r.Presentations, Timeline: tl, total: tl.TotalFrames, dir: dir, ctx: ctx, cancel: cancel}
			m := &Manager{chrome: chrome, ffmpeg: ffmpeg, jobs: map[string]*Job{j.ID: j}}
			mux := http.NewServeMux()
			m.Handle(mux)
			mux.Handle("/ball/", http.StripPrefix("/ball/", http.FileServer(http.FS(unit.BallFS()))))
			mux.Handle("/", http.FileServer(http.FS(web.FS)))
			srv := httptest.NewServer(mux)
			defer srv.Close()
			pageURL := srv.URL + "/?export=" + j.ID
			verifySmokePage(t, m, j, pageURL, tc.probe, tc.team)
			if err := m.render(j, pageURL); err != nil {
				t.Fatal(err)
			}
			probeBin := filepath.Join(filepath.Dir(ffmpeg), "ffprobe.exe")
			if p, err := exec.LookPath("ffprobe"); err == nil {
				probeBin = p
			}
			out, err := exec.CommandContext(ctx, probeBin, "-v", "error", "-select_streams", "v:0", "-count_frames", "-show_entries", "stream=nb_read_frames", "-of", "default=noprint_wrappers=1:nokey=1", j.file).CombinedOutput()
			if err != nil {
				t.Fatalf("ffprobe: %v %s", err, out)
			}
			if strings.TrimSpace(string(out)) != strconv.Itoa(j.total) {
				t.Fatalf("video frames=%s want %d", out, j.total)
			}
			first := j.Timeline.TitleFrames + j.Timeline.OpeningFrames + j.Timeline.HoldStartFrames
			still := filepath.Join(j.dir, "encoded-battle-first.png")
			if out, err := exec.CommandContext(ctx, ffmpeg, "-y", "-v", "error", "-i", j.file, "-vf", fmt.Sprintf("select=eq(n\\,%d)", first), "-frames:v", "1", still).CombinedOutput(); err != nil {
				t.Fatalf("extract battle frame: %v %s", err, out)
			}
			data, err := os.ReadFile(still)
			if err != nil {
				t.Fatal(err)
			}
			img, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			rr, gg, bb, _ := img.At(60, 500).RGBA()
			if rr+gg+bb == 0 {
				t.Fatal("encoded battle starts with a black stage")
			}
			t.Logf("validated %d video frames: %s", j.total, j.file)
		})
	}
}

func verifySmokePage(t *testing.T, m *Manager, j *Job, pageURL string, probe, team bool) {
	t.Helper()
	ch, err := launchChrome(j.ctx, m.chrome, j.dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ch.close()
	if err = ch.page.setViewport(j.ctx, 1080, 1920); err != nil {
		t.Fatal(err)
	}
	if err = ch.page.navigate(j.ctx, pageURL); err != nil {
		t.Fatal(err)
	}
	if err = ch.page.waitFor(j.ctx, "window.__exportReady === true", 45*time.Second); err != nil {
		msg, _ := ch.page.evalString(j.ctx, "String(window.__exportError || '')")
		t.Fatalf("ready: %v %s", err, msg)
	}
	first := j.Timeline.TitleFrames + j.Timeline.OpeningFrames + j.Timeline.HoldStartFrames
	step := func(i int, label string) map[string]any {
		t.Helper()
		value, err := ch.page.evalString(j.ctx, fmt.Sprintf(`window.__exportStep(%d).then(() => JSON.stringify({intro: document.querySelectorAll('.presentation-layer .intro-ball').length,x:document.querySelector('.presentation-layer .intro-ball')?.style.left,units:document.querySelectorAll('#units > .ball').length,time:performance.now(),voices:soundVoices.size,hud:document.getElementById('export-root').style.getPropertyValue('--intro-hud'),nameOpacity:getComputedStyle(document.getElementById('export-name-0')).opacity,rootDisplay:getComputedStyle(document.getElementById('export-root')).display,rootOpacity:getComputedStyle(document.getElementById('export-root')).opacity,arenaOpacity:getComputedStyle(document.getElementById('export-arena')).opacity,presentation:document.querySelector('.presentation-layer')?.dataset.presentation,banner:document.querySelector('.presentation-layer .test-banner')?.textContent,audio:JSON.parse(window.__exportAudio())}))`, i))
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err = json.Unmarshal([]byte(value), &got); err != nil {
			t.Fatal(err)
		}
		for _, raw := range got["audio"].([]any) {
			ev := raw.(map[string]any)
			minimum := first
			for _, p := range j.Presentations {
				if p.Audio {
					minimum = j.Timeline.TitleFrames
				}
			}
			if ev["frame"].(float64) < float64(minimum) {
				t.Fatalf("early audio: %s", value)
			}
		}
		if label != "" {
			if got["hud"] != "1" || got["nameOpacity"] != "1" || got["rootDisplay"] != "block" {
				t.Fatalf("hidden export layout: %s", value)
			}
			pngBytes, err := ch.page.screenshot(j.ctx)
			if err != nil {
				t.Fatal(err)
			}
			if label == "battle-first" {
				img, err := png.Decode(bytes.NewReader(pngBytes))
				if err != nil {
					t.Fatal(err)
				}
				rr, gg, bb, _ := img.At(60, 500).RGBA()
				if rr+gg+bb == 0 {
					t.Fatal("battle transition captured an unpainted black stage")
				}
			}
			if err = os.WriteFile(filepath.Join(j.dir, label+".png"), pngBytes, 0644); err != nil {
				t.Fatal(err)
			}
		}
		return got
	}
	before := step(0, "")
	rewindStart := j.Timeline.TitleFrames
	if len(j.Presentations) > 0 && j.Presentations[0].Type == "banner" {
		banner := step(j.Timeline.TitleFrames, "banner")
		if banner["banner"] != "准备开场" || banner["presentation"] != "banner" {
			t.Fatalf("generic banner did not render: %+v", banner)
		}
		rewindStart += j.Timeline.PresentationFrames[0]
	}
	if probe {
		forward := step(rewindStart+30, "probe-death")
		rewound := step(j.Timeline.TitleFrames+j.Timeline.OpeningFrames-1, "probe-rewound")
		want := float64(2)
		if team {
			want = 4
		}
		if forward["intro"] != want || rewound["intro"] != want || forward["x"] == rewound["x"] {
			t.Fatalf("probe did not rewind: %v -> %v", forward, rewound)
		}
		if forward["time"] != float64(0) || rewound["time"] != float64(0) {
			t.Fatal("probe advanced the battle clock")
		}
	}
	hold := step(j.Timeline.TitleFrames+j.Timeline.OpeningFrames, "first-hold")
	if hold["intro"] != float64(0) {
		t.Fatal("probe layer survived transition")
	}
	battle := step(first, "battle-first")
	if battle["intro"] != float64(0) || len(battle["audio"].([]any)) == 0 {
		t.Fatalf("missing battle or first audio: %+v", battle)
	}
	if battle["voices"] != before["voices"] {
		t.Fatal("presentation audio voice leaked after cleanup")
	}
	step(first+1, "")
	step(j.total-1, "winner")
	audio, err := ch.page.evalString(j.ctx, "window.__exportAudio()")
	if err != nil {
		t.Fatal(err)
	}
	var events []audioEvent
	if err = json.Unmarshal([]byte(audio), &events); err != nil {
		t.Fatal(err)
	}
	if len(j.Presentations) > 0 && j.Presentations[0].Audio {
		if len(events) != 3 || events[0].Frame != j.Timeline.TitleFrames || events[1].Stop == nil || events[1].Frame != j.Timeline.TitleFrames+j.Timeline.OpeningFrames || events[2].Frame != first {
			t.Fatalf("presentation audio timeline: %s", audio)
		}
	} else if len(events) != 1 || events[0].Frame != first {
		t.Fatalf("first effect duplicated or shifted: %s", audio)
	}
}

func smokeRecording(probe, team bool) replay.Recording {
	slots := [2][]string{{"非因果"}, {"小骑士"}}
	units := []unit.Snapshot{{ID: 1, Kind: "非因果", Role: unit.RoleFighter, Slot: 0, X: -80, Radius: 18, HP: 100, MaxHP: 100}, {ID: 2, Kind: "小骑士", Role: unit.RoleFighter, Slot: 1, X: 80, Radius: 18, HP: 100, MaxHP: 100}}
	if team {
		slots = [2][]string{{"非因果", "小骑士"}, {"小骑士", "小骑士"}}
		units = append(units, unit.Snapshot{ID: 3, Kind: "小骑士", Role: unit.RoleFighter, Slot: 0, Seat: 1, X: -40, Y: 50, Radius: 18, HP: 100, MaxHP: 100}, unit.Snapshot{ID: 4, Kind: "小骑士", Role: unit.RoleFighter, Slot: 1, Seat: 1, X: 40, Y: 50, Radius: 18, HP: 100, MaxHP: 100})
	}
	packs := unit.Packs()
	looks := unit.Looks()
	snap := map[string]any{"phase": "running", "slots": slots, "roster": slots, "team": team, "units": units, "looks": looks, "packs": map[string]unit.PackInfo{"非因果": packs["非因果"], "小骑士": packs["小骑士"], "演出测试": packs["演出测试"]}, "field": "六边形", "fieldShape": "hex", "hexRadius": 280, "time": 0, "seq": 0}
	initial, _ := json.Marshal(snap)
	snap["effects"] = []unit.FX{{Name: "impact", Kind: "小骑士", X: 80}}
	snap["time"], snap["seq"] = 1.0/60, 1
	f0, _ := json.Marshal(snap)
	snap["effects"] = []unit.FX{}
	snap["phase"], snap["winner"], snap["seq"] = "ended", "小骑士", 2
	f1, _ := json.Marshal(snap)
	r := replay.Recording{Initial: initial, Frames: []json.RawMessage{f0, f1}}
	if probe {
		film := &非因果.IntroFilm{Death: 1}
		for _, tm := range []float64{0, 0.5, 1} {
			var bodies []非因果.IntroBody
			for i, u := range units {
				bodies = append(bodies, 非因果.IntroBody{K: u.Kind, S: u.Slot, X: u.X + tm*float64(i+1)*60, Y: u.Y, R: u.Radius})
			}
			film.Frames = append(film.Frames, 非因果.IntroFrame{T: tm, B: bodies})
		}
		p, err := 非因果.RewindPresentation(film)
		if err != nil {
			panic(err)
		}
		p.ID = "rewind-1"
		r.Presentations = []replay.Presentation{p}
	}
	return r
}

// 第二个测试播放器使用文本数据，没有死亡时间、轨迹或倒带规则。
func init() {
	unit.NewPack("演出测试", fstest.MapFS{"fx/banner.js": {Data: []byte(`arena.registerPresentation("banner", {
 mount(ctx,data) { const el=document.createElement("div");el.className="test-banner";el.textContent=data.message;el.style.cssText="position:absolute;top:40%;left:20%;font-size:30px;color:white";ctx.root.appendChild(el);ctx.playAudio("/sfx/default-hit.wav",{});return el; },
 render(el,elapsed) { el.dataset.elapsed=String(elapsed); },
 destroy(el) { el.remove(); }
});`)}})
}
