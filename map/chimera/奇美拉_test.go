package 奇美拉

import (
	"testing"
)

func TestContourClosedAndInside(t *testing.T) {
	pts := contour()
	if len(pts) < 20 {
		t.Fatalf("contour too short: %d", len(pts))
	}
	area := 0.0
	for i := range pts {
		j := (i + 1) % len(pts)
		area += pts[i].x*pts[j].y - pts[j].x*pts[i].y
	}
	if area <= 0 {
		t.Fatalf("want CCW area, got %v", area/2)
	}
	f := Field()

	if !f.OutlineContains(0, 0, 18) {
		t.Fatal("hub center should be inside")
	}
	nx, ny := North()
	if !f.OutlineContains(nx, ny, 18) {
		t.Fatal("volcano center should be inside")
	}
	sx, sy := South()
	if !f.OutlineContains(sx, sy, 18) {
		t.Fatal("chamber center should be inside")
	}
	ex, ey := East()
	if !f.OutlineContains(ex, ey, 18) {
		t.Fatal("circle center should be inside")
	}
	wx, wy := West()
	if !f.OutlineContains(wx, wy, 18) {
		t.Fatal("octagon center should be inside")
	}
	// 斜边外：枢纽东北空隙。
	apo := hexApo()
	if f.OutlineContains(apo, apo, 0) {
		t.Fatal("sealed corner should be outside")
	}
	// 北通道中段在场内。
	if !f.OutlineContains(0, hexApo()+neck/2, 18) {
		t.Fatal("north neck should be inside")
	}
	// 火山和圆之间的空隙在场外。
	if f.OutlineContains(ex*0.55, ny*0.55, 0) {
		t.Fatal("gap between regions should be outside")
	}
	if !RegionContains(0, wx, wy, 18) || RegionContains(0, 0, 0, 18) {
		t.Fatal("west region")
	}
	if !RegionContains(1, ex, ey, 18) || RegionContains(1, 0, 0, 18) {
		t.Fatal("east region")
	}
}

func TestDoorDoesNotSweep(t *testing.T) {
	o := newChimeraOutline(0)
	// 从枢纽向北穿过通道，不该撞场边。
	y0 := hexApo() - 40
	y1 := yFarN() + 40
	dt := 1.0
	vy := y1 - y0
	if _, _, _, hit := o.Sweep(0, y0, 0, vy, 0, 1, 18, dt, false); hit {
		t.Fatal("north door should be open")
	}
	// 上平边在门洞两侧仍是墙。
	if _, _, _, hit := o.Sweep(120, hexApo()-40, 0, 200, 0, 1, 18, 1, false); !hit {
		t.Fatal("north flat outside the door should be a wall")
	}
}
