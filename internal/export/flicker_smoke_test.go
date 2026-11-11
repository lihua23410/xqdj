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

	_ "xqdj/character/flicker"
	"xqdj/internal/replay"
	"xqdj/internal/unit"
	"xqdj/internal/web"
)

// 真实 Chromium 验证单挑、团队赛和导出画布；普通 go test 不启动浏览器。
func TestFlickerRenderSmoke(t *testing.T) {
	if os.Getenv("XQDJ_EXPORT_SMOKE") != "1" {
		t.Skip("set XQDJ_EXPORT_SMOKE=1 for browser verification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	dir := t.TempDir()
	units := []unit.Snapshot{
		{ID: 1, Kind: "闪烁", Role: unit.RoleFighter, Slot: 0, X: -80, Y: 0, Radius: 18, HP: 100, MaxHP: 100, AimPriority: 15},
		{ID: 2, Kind: "原型机_远程", Role: unit.RoleFighter, Slot: 1, X: 100, Y: 0, Radius: 18, HP: 80, MaxHP: 100, AimPriority: 15},
		{ID: 10, Kind: "闪烁子弹", Role: unit.RoleProjectile, OwnerID: 1, X: -60, Y: 100, VX: 580, Radius: 3, HP: 1},
		{ID: 11, Kind: "闪烁智能弹", Role: unit.RoleProjectile, OwnerID: 1, X: -60, Y: 50, VX: 580, Radius: 3, HP: 1},
		{ID: 12, Kind: "闪烁强化智能弹", Role: unit.RoleProjectile, OwnerID: 1, X: -60, Y: -65, VX: 580, Radius: 3, HP: 1},
	}
	visual := json.RawMessage(`{"Phase":2,"ReadyAt":1.4,"SideAt":0,"SideUntil":0.111,"FocusUntil":0.461,"Aim":true,"AimX":100,"AimY":0,"ThreatX":-30,"ThreatY":0,"Trail":[{"X":-80,"Y":70,"T":0},{"X":-80,"Y":48,"T":0.033},{"X":-80,"Y":24,"T":0.066},{"X":-80,"Y":0,"T":0.1}],"Beams":[]}`)
	frame := map[string]any{"phase": "running", "slots": [2][]string{{"闪烁"}, {"原型机_远程"}}, "units": units, "looks": unit.Looks(), "packs": unit.Packs(), "field": "六边形", "fieldShape": "hex", "hexRadius": 280, "time": .18, "seq": 1, "effects": []unit.FX{{Name: "flicker-state", Kind: "闪烁", UnitID: 1, Data: visual}, {Name: "shot", Kind: "闪烁", UnitID: 1}}}
	initial, _ := json.Marshal(frame)
	tl := newTimeline(replay.Recording{Initial: initial, Frames: []json.RawMessage{initial}})
	tl.TitleFrames, tl.HoldStartFrames, tl.EndFrames = 1, 1, 1
	tl.TotalFrames = tl.TitleFrames + tl.HoldStartFrames + tl.BattleFrames + tl.EndFrames
	j := &Job{ID: "flicker", Initial: initial, Frames: []json.RawMessage{initial}, Timeline: tl, total: tl.TotalFrames, dir: dir, ctx: ctx, cancel: cancel}
	m := &Manager{chrome: findChrome(), jobs: map[string]*Job{j.ID: j}}
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
	defer func() {
		chrome.close()
		// Windows 的浏览器子进程可能稍晚才释放继承的日志句柄。
		// 只重试本次测试创建的日志，不改变浏览器进程管理。
		log := filepath.Join(dir, "chrome.log")
		until := time.Now().Add(3 * time.Second)
		for time.Now().Before(until) {
			if err := os.Remove(log); err == nil || os.IsNotExist(err) {
				break
			}
			time.Sleep(25 * time.Millisecond)
		}
	}()
	for _, mode := range []string{"live", "team", "export"} {
		t.Run(mode, func(t *testing.T) {
			url := srv.URL + "/"
			if mode == "team" {
				url += "team.html"
			}
			if mode == "export" {
				url += "?export=" + j.ID
				chrome.page.setViewport(ctx, 1080, 1920)
			} else {
				chrome.page.setViewport(ctx, 1100, 850)
			}
			if err := chrome.page.navigate(ctx, url); err != nil {
				t.Fatal(err)
			}
			if mode == "export" {
				if err := chrome.page.waitFor(ctx, "window.__exportReady === true", 30*time.Second); err != nil {
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
			ok, err := chrome.page.evalBool(ctx, `(()=>{
 const base=state.units;
 for(let i=1;i<=8;i++) {
   const units=base.map(u=>u.id<10?u:{...u,x:u.x+i*6,y:u.y+(u.id===10?0:(u.id===11?1:-1)*i*i*.22),vy:u.id===10?0:(u.id===11?1:-1)*i*26});
   window.__exportRender({...state,units,time:.18+i/60,effects:[]});
 }
 if(document.querySelectorAll('.flicker-round').length!==3 || document.querySelectorAll('.flicker-flight canvas').length!==2)return false;
 for(const id of [10,11,12]) {
   const el=document.getElementById('u-'+id),style=getComputedStyle(el);
   if(style.backgroundColor!=='rgba(0, 0, 0, 0)'||style.boxShadow!=='none')return false;
 }
 const smart=document.getElementById('flicker-flight-11');
 if(smart._flight.points.length<6)return false;
 const path=smart._flight.points;
 if(path[1].y-path[0].y===path.at(-1).y-path.at(-2).y)return false;
 const el=document.querySelector('#flicker-state-1 canvas');
 if(!el||el.width<200||!document.getElementById('u-1').classList.contains('flicker-focus'))return false;
 const pixels=el.getContext('2d').getImageData(0,0,el.width,el.height).data;
 let lit=0;for(let i=3;i<pixels.length;i+=4)if(pixels[i]>30)lit++;
 return lit>100 && document.querySelectorAll('.flicker-scene').length===1;
})()`)
			if err != nil || !ok {
				t.Fatalf("visible effects: %v %v", ok, err)
			}
			if root := os.Getenv("XQDJ_FLICKER_ARTIFACTS"); root != "" {
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
			ok, err = chrome.page.evalBool(ctx, `(()=>{
 const canvas=document.querySelector('#flicker-state-1 canvas'),before=canvas.toDataURL();
 const trail=document.querySelector('#flicker-flight-11 canvas'),trailBefore=trail.toDataURL();
 const frozen={...state,effects:[]};window.__exportRender(frozen);
 if(canvas.toDataURL()!==before||trail.toDataURL()!==trailBefore)return false;
 window.__exportRender({...frozen,time:.1});
 if(document.getElementById('flicker-flight-11')._flight.points.length!==1)return false;
 window.__exportRender({...frozen,units:[]});
 return !document.querySelector('.flicker-scene,.flicker-flight,.flicker-round');
})()`)
			if err != nil || !ok {
				t.Fatalf("pause and cleanup: %v %v", ok, err)
			}
		})
	}
}
