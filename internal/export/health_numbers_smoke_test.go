package export

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "xqdj/character/doppel_veteran"
	_ "xqdj/character/godfather"
	_ "xqdj/character/hammer"
	_ "xqdj/character/maggot"
	_ "xqdj/character/mezhou"
	_ "xqdj/character/piper"
	_ "xqdj/character/ranged"
	_ "xqdj/character/rmiu"
	_ "xqdj/character/twin"
	"xqdj/internal/replay"
	"xqdj/internal/unit"
	"xqdj/internal/web"
)

// 与视频烟测共用开关；检查真实角色资源和 DOM，避免只验证数字格式。
func TestHealthNumbersSmoke(t *testing.T) {
	if os.Getenv("XQDJ_EXPORT_SMOKE") != "1" {
		t.Skip("set XQDJ_EXPORT_SMOKE=1 for browser verification")
	}
	units := []unit.Snapshot{
		{ID: 1, Kind: "小骑士", Role: unit.RoleFighter, Radius: 18, HP: 73.4, MaxHP: 100, X: -160, Y: 150},
		{ID: 2, Kind: "无下限术士", Role: unit.RoleFighter, HP: 41.4, MaxHP: 150, Seat: 1, NoHealthNumbers: true},
		{ID: 3, Kind: "无下限术士", Role: unit.RoleFighter, HP: 88.6, MaxHP: 150, Seat: 2, NoHealthNumbers: true},
		{ID: 4, Kind: "非因果", Role: unit.RoleFighter, Radius: 18, HP: 75, MaxHP: 100, Slot: 1, X: 160, Y: 150},
		{ID: 11, Kind: "红半", Role: unit.RoleMinion, Mortal: true, Radius: 18, HP: 9999, MaxHP: 9999, OwnerID: 2, X: -60, Y: 150, Semi: true, FaceX: 1},
		{ID: 12, Kind: "蓝半", Role: unit.RoleMinion, Mortal: true, Radius: 18, HP: 9999, MaxHP: 9999, OwnerID: 3, X: 60, Y: 150, Semi: true, FaceX: 1},
		{ID: 13, Kind: "红半", Role: unit.RoleMinion, Mortal: true, Radius: 18, HP: 9999, MaxHP: 9999, OwnerID: 999, X: 200, Y: 50, Semi: true, FaceX: 1},
		{ID: 50, Kind: "小骑士", Role: unit.RoleFighter, Radius: 18, HP: 100, MaxHP: 100, Slot: 1, Seat: 1, X: -200, Y: 50, NoHealthNumbers: true},
		{ID: 51, Kind: "测试分身", Role: unit.RoleClone, Radius: 12, HP: 1, X: -200, Y: -150},
		{ID: 52, Kind: "子弹", Role: unit.RoleProjectile, Radius: 6, HP: 1, X: 0, Y: -150},
		{ID: 53, Kind: "测试辅助物", Role: unit.RoleHelper, Radius: 8, HP: 1, X: 200, Y: -150},
	}
	expected := map[string]string{"hp-1": "73", "hp-4": "75", "hp-11": "41", "hp-12": "89"}
	for i, entry := range []struct {
		kind string
		hp   float64
		want string
	}{
		{"教父暗杀者", 14.4, "14"}, {"教父狙击者", 8, "8"}, {"教父贩卖者", 3, "3"},
		{"战影", 23, "23"}, {"周旋久", 31, "31"}, {"钉与锤人偶", 7.5, "8"},
		{"缪", 17, "17"}, {"蛆宝宝", 0.4, "1"}, {"苍蝇", 5, "5"}, {"苍蝇卵", 1, "1"}, {"吹笛人老鼠", 3.2, "3"},
	} {
		id := uint64(21 + i)
		units = append(units, unit.Snapshot{ID: id, Kind: entry.kind, Role: unit.RoleMinion, Mortal: true, Radius: 14, HP: entry.hp, MaxHP: 50, OwnerID: 1, X: float64(i%6)*70 - 175, Y: 55 - float64(i/6)*85, FaceX: 1})
		expected[fmt.Sprintf("hp-%d", id)] = entry.want
	}
	for i, entry := range []struct {
		kind, skin string
		hp         float64
		want       string
	}{
		{"非因果战斗机影", "战影", 6.4, "6"},
		{"非因果活随从影", "红半", 17.8, "18"},
		{"非因果自身影", "原型机_远程", 0.3, "1"},
	} {
		id := uint64(41 + i)
		role := unit.RoleHelper
		if i == 1 {
			role = unit.RoleMinion
		}
		units = append(units, unit.Snapshot{ID: id, Kind: entry.kind, Role: role, Mortal: i == 1, Radius: 16, HP: entry.hp, MaxHP: 100, OwnerID: 4, Slot: 1, X: float64(i)*85 - 85, Y: -100, NoHealthNumbers: true, Marks: []unit.Mark{{Kind: entry.skin}}})
		expected[fmt.Sprintf("acausal-hp-%d", id)] = entry.want
	}
	frame := map[string]any{"phase": "running", "team": true, "slots": [2][]string{{"小骑士", "无下限术士", "无下限术士"}, {"非因果", "小骑士"}}, "units": units, "looks": unit.Looks(), "packs": unit.Packs(), "field": "六边形", "fieldShape": "hex", "hexRadius": 280, "time": 0, "effects": []any{}}
	initial, err := json.Marshal(frame)
	if err != nil {
		t.Fatal(err)
	}
	frames := []json.RawMessage{initial}
	tl := newTimeline(replay.Recording{Initial: initial, Frames: frames})
	tl.TitleFrames, tl.HoldStartFrames, tl.EndFrames = 3, 2, 3
	tl.TotalFrames = tl.TitleFrames + tl.HoldStartFrames + tl.BattleFrames + tl.EndFrames
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dir := t.TempDir()
	job := &Job{ID: "health", Initial: initial, Frames: frames, Timeline: tl, total: tl.TotalFrames, dir: dir, ctx: ctx, cancel: cancel}
	m := &Manager{chrome: findChrome(), ffmpeg: findFFmpeg(), jobs: map[string]*Job{job.ID: job}}
	mux := http.NewServeMux()
	m.Handle(mux)
	mux.Handle("/ball/", http.StripPrefix("/ball/", http.FileServer(http.FS(unit.BallFS()))))
	mux.Handle("/", http.FileServer(http.FS(web.FS)))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	chrome, err := launchChrome(ctx, m.chrome, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer chrome.close()
	chrome.page.call(ctx, "Page.enable", nil, nil)
	chrome.page.call(ctx, "Runtime.enable", nil, nil)
	want, _ := json.Marshal(expected)
	for _, mode := range []string{"export", "live", "team"} {
		t.Run(mode, func(t *testing.T) {
			url := srv.URL + "/"
			if mode == "export" {
				url += "?export=" + job.ID
				chrome.page.setViewport(ctx, 1080, 1920)
			} else {
				chrome.page.setViewport(ctx, 1100, 850)
			}
			if mode == "team" {
				url += "team.html"
			}
			if err := chrome.page.navigate(ctx, url); err != nil {
				t.Fatal(err)
			}
			if mode == "export" {
				if err := chrome.page.waitFor(ctx, "window.__exportReady === true", 45*time.Second); err != nil {
					t.Fatal(err)
				}
				if _, err := chrome.page.eval(ctx, fmt.Sprintf("window.__exportStep(%d)", tl.TitleFrames+tl.HoldStartFrames)); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := chrome.page.waitFor(ctx, "typeof window.__exportRender === 'function'", 15*time.Second); err != nil {
					t.Fatal(err)
				}
				if _, err := chrome.page.eval(ctx, fmt.Sprintf("(async()=>{window.__exportRender(%s);await window.__packsWait();window.__exportRender(%s)})()", initial, initial)); err != nil {
					t.Fatal(err)
				}
			}
			result, err := chrome.page.evalString(ctx, fmt.Sprintf(`(()=>{
 const want=%s, errors=[];
 const reference=getComputedStyle(document.getElementById('hp-1'));
 for(const [id,text] of Object.entries(want)) {
   const el=document.getElementById(id);
   if(!el||el.textContent!==text) {errors.push(id+': '+el?.textContent+' != '+text);continue;}
   const style=getComputedStyle(el), body=document.getElementById('u-'+id.split('-').at(-1));
   if(id.startsWith('hp-')&&el.parentElement!==body.parentElement) errors.push(id+': layer');
   for(const key of ['color','fontSize','fontWeight','fontVariantNumeric','textShadow']) if(style[key]!==reference[key]) errors.push(id+': '+key);
   const matrix=new DOMMatrix(style.transform);
   if(matrix.a!==1||matrix.b!==0||matrix.c!==0||matrix.d!==1||Math.abs(matrix.e+parseFloat(style.width)/2)>0.1||Math.abs(matrix.f+parseFloat(style.height))>0.1) errors.push(id+': anchor');
   if(style.color!=='rgb(255, 255, 255)'||parseFloat(style.height)<=0||style.borderTopWidth!=='0px') errors.push(id+': invisible/non-white');
   if(Math.abs(parseFloat(el.style.top)-(parseFloat(body.style.top)-parseFloat(body.style.height)/2-4))>0.001) errors.push(id+': position');
 }
 if(document.querySelectorAll('.hp-float').length!==Object.keys(want).length) errors.push('duplicate/missing numbers');
 if(document.querySelector('.minion-hp,.stand-hp,.miu-hp-num,.maggot-hp')) errors.push('private health UI remains');
 for(const id of [2,3,13,50,51,52,53]) if(document.getElementById('hp-'+id)) errors.push('unexpected hp-'+id);
 if(!document.querySelector('%s')) errors.push('outside health bar removed');
 return JSON.stringify(errors);
})()`, want, map[bool]string{true: ".export-hp-bar", false: ".hp"}[mode == "export"]))
			if err != nil {
				t.Fatal(err)
			}
			if result != "[]" {
				t.Fatalf("health display: %s", result)
			}
			if root := os.Getenv("XQDJ_HEALTH_ARTIFACTS"); root != "" {
				if err := os.MkdirAll(root, 0755); err != nil {
					t.Fatal(err)
				}
				png, err := chrome.page.screenshot(ctx)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, mode+".png"), png, 0644); err != nil {
					t.Fatal(err)
				}
			}
			ok, err := chrome.page.evalBool(ctx, `(()=>{
 const units=state.units.map(u=>({...u}));
 units.find(u=>u.id===1).noHealthNumbers=true;
 units.find(u=>u.id===3).hp=20;
 units.find(u=>u.id===21).hp=0.4;
 units.find(u=>u.id===42).hp=0.4;
 window.__exportRender({...state,units,time:1});
 if(document.getElementById('hp-1')||document.getElementById('hp-12').textContent!=='20'||document.getElementById('hp-21').textContent!=='1'||document.getElementById('acausal-hp-42').textContent!=='1') return false;
 window.__exportRender({...state,units:[]});
 return document.querySelectorAll('.hp-float').length===0;
})()`)
			if err != nil || !ok {
				t.Fatalf("update/hide/cleanup: %v %v", ok, err)
			}
		})
	}
}
