package r缪

import (
	"testing"
	"xqdj/internal/unit"
)

func TestStarMiuBootsOnCD(t *testing.T) {
	old := miuStarRoll
	miuStarRoll = func() float64 { return 0 }
	defer func() { miuStarRoll = old }()

	out := make(chan unit.Cmd, 32)
	r := &R缪{}
	ctx := unit.Context{ID: 1, Kind: KindRMiu, Out: out}
	r.Handle(ctx, unit.Sense{Time: 0, Self: bodySelf(0, 0, 100, 0, 0)})
	if !r.starMiu {
		t.Fatal("5% roll hit should grant 天杀星缪")
	}
	if r.starReadyAt != starMiuCD {
		t.Fatalf("readyAt=%v want %v", r.starReadyAt, starMiuCD)
	}
	marked := false
	for _, c := range drain(out) {
		if v, ok := c.(unit.StackMark); ok && v.Kind == starMiuKind && v.Icon == starMiuIcon {
			marked = true
		}
		if _, ok := c.(unit.Damage); ok {
			t.Fatal("开局 CD，不应立刻斩")
		}
	}
	if !marked {
		t.Fatal("missing 天杀星缪 icon")
	}
}

func TestStarMiuStrikesClone(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	r := &R缪{
		booted: true, starMiu: true, starReadyAt: 5,
		speed: miuBaseSpeed, meleeReadyAt: 999, fireReadyAt: 999,
	}
	ctx := unit.Context{ID: 1, Kind: KindRMiu, Out: out}
	r.Handle(ctx, unit.Sense{
		Time:   5,
		Self:   bodySelf(0, 0, 100, 0, 0),
		Nearby: []unit.Snapshot{cloneAt(2, 40, 0, 100)},
	})
	var dmg *unit.Damage
	scream := false
	for _, c := range drain(out) {
		switch v := c.(type) {
		case unit.Damage:
			if v.To == 2 {
				cp := v
				dmg = &cp
			}
		case unit.FX:
			if v.Name == "scream" {
				scream = true
			}
		}
	}
	if dmg == nil || dmg.Amount != starMiuDmg || dmg.From != 1 {
		t.Fatalf("strike=%v", dmg)
	}
	if !scream {
		t.Fatal("missing scream")
	}
	if r.starReadyAt != 5+starMiuCD {
		t.Fatalf("next CD=%v", r.starReadyAt)
	}
}

func TestStarMiuLethalDies(t *testing.T) {
	out := make(chan unit.Cmd, 16)
	r := &R缪{
		starMiu: true, hp: 8,
		hasBest: true, bestID: 2, bestHP: 90,
	}
	ctx := unit.Context{ID: 1, Kind: KindRMiu, Out: out}
	r.Handle(ctx, unit.IncomingDamage{Token: 7, From: 99, Amount: 20})
	cmds := drain(out)
	confirm := false
	for _, c := range cmds {
		switch c.(type) {
		case unit.ConfirmDamage:
			confirm = true
		case unit.BlockDamage, unit.Teleport, unit.SetHP:
			t.Fatalf("天杀星缪 must not swap: %v", cmds)
		}
	}
	if !confirm {
		t.Fatal("lethal should ConfirmHit")
	}
}

func TestStarMiuKeepsOwnBody(t *testing.T) {
	out := make(chan unit.Cmd, 32)
	r := &R缪{
		booted: true, starMiu: true, starReadyAt: 999,
		speed: miuBaseSpeed, meleeReadyAt: 999, fireReadyAt: 999,
	}
	ctx := unit.Context{ID: 1, Kind: KindRMiu, Out: out}
	r.Handle(ctx, unit.Sense{
		Time:   1,
		Self:   bodySelf(0, 0, 40, 170, 0),
		Nearby: []unit.Snapshot{cloneAt(2, 50, 0, 90)},
	})
	for _, c := range drain(out) {
		if _, ok := c.(unit.Teleport); ok {
			t.Fatal("天杀星缪 must not swap with healthier clone")
		}
	}
}
