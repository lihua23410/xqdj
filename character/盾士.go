package character

import (
	"math"
	"math/rand/v2"
	"xqdj/internal/unit"
)

const KindWarden = "盾士"

const (
	wardenRadius = 18.0
	wardenSpeed  = 165.0
	wardenHP     = 100.0
	wardenVision = 9999.0

	kindShield     = "盾"
	kindShard      = "盾碎片"
	kindWeakShard  = "弱化碎片"
	shieldRadius   = 20.0
	shardRadius    = 6.0
	shardMinSpeed  = 90.0
	shardMaxSpeed  = 280.0
	fullShardN     = 8
	weakShardN     = 6
	fullShardDmg   = 6.0
	weakShardDmg   = 4.0
	shardWallHits  = 2 // 弹 1 次，第 2 次撞墙消失
	shieldRefresh  = 8.0
	shieldLostTrig = 40.0
)

func init() {
	unit.Register(unit.Spec{
		Kind:    KindWarden,
		Role:    unit.RoleFighter,
		Radius:  wardenRadius,
		MaxHP:   wardenHP,
		Speed:   wardenSpeed,
		Vision:  wardenVision,
		Fighter: true,
	}, func(unit.SpawnInfo) unit.Actor {
		return &盾士{refreshAt: shieldRefresh}
	})
	unit.Register(unit.Spec{
		Kind:    kindShield,
		Role:    unit.RoleProjectile,
		Radius:  shieldRadius,
		MaxHP:   1,
		Speed:   0,
		Vision:  0,
		Fighter: false,
		Shell:   true,
	}, func(unit.SpawnInfo) unit.Actor {
		return &盾{}
	})
	unit.Register(unit.Spec{
		Kind:    kindShard,
		Role:    unit.RoleProjectile,
		Radius:  shardRadius,
		MaxHP:   1,
		Speed:   shardMaxSpeed,
		Vision:  0,
		Fighter: false,
	}, func(info unit.SpawnInfo) unit.Actor {
		return &盾碎片{owner: info.OwnerID, dmg: fullShardDmg}
	})
	unit.Register(unit.Spec{
		Kind:    kindWeakShard,
		Role:    unit.RoleProjectile,
		Radius:  shardRadius,
		MaxHP:   1,
		Speed:   shardMaxSpeed,
		Vision:  0,
		Fighter: false,
	}, func(info unit.SpawnInfo) unit.Actor {
		return &盾碎片{owner: info.OwnerID, dmg: weakShardDmg}
	})
}

type 盾士 struct {
	refreshAt float64
	lost      float64
	armed     bool
	cover     bool
	booted    bool
	x, y      float64
	slot      int
}

func (s *盾士) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.IncomingDamage:
		if s.armed || s.cover {
			unit.BlockHit(ctx, e)
			s.cover = false
			s.breakFull(ctx)
			return
		}
		unit.ConfirmHit(ctx, e)
		s.lost += e.Amount
	case unit.GuardBreak:
		s.cover = true
		s.breakFull(ctx)
	case unit.Sense:
		s.cover = false
		s.x, s.y = e.Self.X, e.Self.Y
		s.slot = e.Self.Slot
		if !s.booted {
			s.booted = true
			s.putShield(ctx, e)
		}
		s.maybeRefresh(ctx, e)
	}
}

func (s *盾士) maybeRefresh(ctx unit.Context, sense unit.Sense) {
	if sense.Time+1e-9 < s.refreshAt && s.lost < shieldLostTrig {
		return
	}
	if s.armed {
		ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: kindShield}
		s.armed = false
		s.burst(ctx, kindWeakShard, weakShardN)
	}
	s.putShield(ctx, sense)
	s.refreshAt = sense.Time + shieldRefresh
	s.lost = 0
}

func (s *盾士) putShield(ctx unit.Context, sense unit.Sense) {
	ctx.Out <- unit.Spawn{
		Kind:    kindShield,
		X:       sense.Self.X,
		Y:       sense.Self.Y,
		VX:      sense.Self.VX,
		VY:      sense.Self.VY,
		OwnerID: ctx.ID,
		Slot:    sense.Self.Slot,
	}
	s.armed = true
}

func (s *盾士) breakFull(ctx unit.Context) {
	if !s.armed {
		return
	}
	s.armed = false
	ctx.Out <- unit.DespawnOwned{OwnerID: ctx.ID, Kind: kindShield}
	s.burst(ctx, kindShard, fullShardN)
}

func (s *盾士) burst(ctx unit.Context, kind string, n int) {
	ctx.Out <- unit.FX{
		Name: "shatter", Kind: kind,
		X: s.x, Y: s.y, Slot: s.slot,
	}
	base := rand.Float64() * 2 * math.Pi
	gap := wardenRadius + shardRadius + 1.5
	for i := 0; i < n; i++ {
		ang := base + float64(i)*2*math.Pi/float64(n)
		ux, uy := math.Cos(ang), math.Sin(ang)
		sp := shardMinSpeed + rand.Float64()*(shardMaxSpeed-shardMinSpeed)
		ctx.Out <- unit.Spawn{
			Kind:    kind,
			X:       s.x + ux*gap,
			Y:       s.y + uy*gap,
			VX:      ux * sp,
			VY:      uy * sp,
			OwnerID: ctx.ID,
			Slot:    s.slot,
		}
	}
}

type 盾 struct{}

func (*盾) Handle(unit.Context, unit.Event) {}

type 盾碎片 struct {
	owner   uint64
	dmg     float64
	bounces int
}

func (b *盾碎片) Handle(ctx unit.Context, ev unit.Event) {
	switch e := ev.(type) {
	case unit.Collision:
		b.onHit(ctx, e.Other)
	case unit.WallHit:
		b.bounces++
		if b.bounces >= shardWallHits {
			ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		}
	}
}

func (b *盾碎片) onHit(ctx unit.Context, other unit.Snapshot) {
	if other.ID == b.owner {
		return
	}
	if other.Role != unit.RoleFighter {
		ctx.Out <- unit.Despawn{UnitID: ctx.ID}
		return
	}
	ctx.Out <- unit.Damage{From: ctx.ID, To: other.ID, Amount: b.dmg}
	ctx.Out <- unit.Despawn{UnitID: ctx.ID}
}
