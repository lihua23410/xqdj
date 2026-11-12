package 闪烁

import (
	"math"
	"xqdj/internal/unit"
)

const (
	forecastLimit = .65
	sampleStep    = 1.0 / 120
)

type danger struct {
	unit.Threat
	body unit.Snapshot // projectile or attached fan, if present
}

func nearbyDangers(state *unit.MatchState, s unit.Sense) []danger {
	var all []danger
	for _, d := range s.Threats {
		if d.Slot != s.Self.Slot && d.From != s.Self.ID {
			all = append(all, danger{Threat: d})
		}
	}
	for _, o := range s.Nearby {
		if o.Role != unit.RoleProjectile || o.Nonsolid {
			continue
		}
		if o.Slot == s.Self.Slot {
			spec, _ := unit.Lookup(o.Kind)
			if !spec.AttachFriends || o.ArcSpan > 0 {
				continue
			}
		}
		speed := math.Hypot(o.VX, o.VY)
		if o.ArcSpan == 0 && speed < 1 {
			continue
		}
		if math.Hypot(o.X-s.Self.X, o.Y-s.Self.Y) > o.Radius+s.Self.Radius+(speed+sideSpeed)*anticipation {
			continue
		}
		until := anticipation
		if o.ArcSpan > 0 {
			end, possible := unit.AttackFanThreat(state, o, s.Self.ID, s.Time)
			if !possible {
				continue
			}
			if end > 0 {
				until = math.Min(until, end-s.Time)
			}
		}
		if o.ArcSpan == 0 && !o.PassWalls {
			walls := s.Walls
			if o.BreakWalls {
				walls = nil
			}
			field := s.Field
			if o.BreakWalls {
				field.Walls = nil
			}
			until = clipReach(o.X, o.Y, o.VX/speed, o.VY/speed, speed*until, o.Radius, field, walls) / speed
		}
		if until <= 1e-9 {
			continue
		}
		source := o.OwnerID
		if source == 0 {
			source = o.ID
		}
		all = append(all, danger{Threat: unit.Threat{
			From: source, Slot: o.Slot, X: o.X, Y: o.Y, Radius: o.Radius,
			VX: o.VX, VY: o.VY, ObservedAt: s.Time, ActiveAt: s.Time, Until: s.Time + until,
		}, body: o})
	}
	return all
}

type route struct {
	x, y, dx, dy, speed, after, fastFor, length float64
}

func (r route) distance(t float64) float64 {
	return r.speed*math.Min(t, r.fastFor) + r.after*math.Max(0, t-r.fastFor)
}

func (r route) at(t float64) (float64, float64) {
	d := math.Min(r.distance(t), r.length)
	return r.x + r.dx*d, r.y + r.dy*d
}

func predictRoute(s unit.Sense, dx, dy, speed, after, fastFor, horizon float64) route {
	r := route{x: s.Self.X, y: s.Self.Y, dx: dx, dy: dy, speed: speed, after: after, fastFor: fastFor}
	r.length = clipReach(r.x, r.y, dx, dy, r.distance(horizon), s.Self.Radius, s.Field, s.Walls)
	return r
}

func (d danger) impact(s unit.Sense, r route, horizon float64) (float64, bool) {
	// Use the full warning for scheduled attacks; already active dangers use
	// the short contact horizon.
	t := math.Max(0, math.Max(d.ActiveAt, d.ReadyAt[s.Self.ID])-s.Time)
	end := math.Min(horizon, math.Max(anticipation, t+sampleStep))
	end = math.Min(end, d.Until-s.Time)
	if end < t-1e-9 {
		return 0, false
	}
	for {
		x, y := r.at(t)
		if d.covers(s, s.Time+t, x, y) {
			return t, true
		}
		if t >= end {
			return 0, false
		}
		t = math.Min(t+sampleStep, end)
	}
}

func (d danger) covers(s unit.Sense, t, x, y float64) bool {
	dt := t - d.ObservedAt
	ox, oy := d.X+d.VX*dt, d.Y+d.VY*dt
	x, y = x-ox, y-oy
	r := s.Self.Radius
	if d.body.ArcSpan > 0 {
		distance := math.Hypot(x, y)
		if distance > d.Radius+r || distance < d.body.ArcInner-r {
			return false
		}
		face := math.Atan2(d.body.FaceY, d.body.FaceX)
		half := d.body.ArcSpan / 2
		delta := math.Atan2(math.Sin(math.Atan2(y, x)-face), math.Cos(math.Atan2(y, x)-face))
		if math.Abs(delta) <= half {
			return true
		}
		edge := face + math.Copysign(half, delta)
		c, sn := math.Cos(edge), math.Sin(edge)
		return segDist(x, y, d.body.ArcInner*c, d.body.ArcInner*sn, d.Radius*c, d.Radius*sn) <= r
	}
	switch d.Shape {
	case unit.ThreatRect:
		dx := x - math.Max(0, math.Min(x, d.X2-d.X))
		dy := y - math.Max(0, math.Min(y, d.Y2-d.Y))
		return math.Hypot(dx, dy) <= r
	case unit.ThreatCapsule, unit.ThreatRay:
		dx, dy := d.segmentAxis(t)
		if d.Shape == unit.ThreatRay {
			length := math.Hypot(dx, dy)
			if length <= 1e-9 {
				return false
			}
			ux, uy := dx/length, dy/length
			along := x*ux + y*uy
			return along >= 0 && along <= length && math.Abs(x*uy-y*ux) <= d.Radius+r &&
				clipReach(ox, oy, ux, uy, along, d.Radius, s.Field, s.Walls)+1e-9 >= along
		}
		return segDist(x, y, 0, 0, dx, dy) <= d.Radius+r
	default:
		return math.Hypot(x, y) <= d.Radius+r
	}
}

func (d danger) segmentAxis(t float64) (float64, float64) {
	dx, dy := d.X2-d.X, d.Y2-d.Y
	if d.Spin == 0 {
		return dx, dy
	}
	angle := (t - d.ObservedAt) * d.Spin
	c, sn := math.Cos(angle), math.Sin(angle)
	return dx*c - dy*sn, dx*sn + dy*c
}

func (d danger) dodgeNormal(s unit.Sense, t float64) (float64, float64) {
	// Moving attacks use their direction of travel.
	ax, ay := d.VX, d.VY
	if d.Shape == unit.ThreatCapsule || d.Shape == unit.ThreatRay {
		ax, ay = d.segmentAxis(t)
	} else if math.Hypot(ax, ay) < 1e-9 {
		if d.body.ArcSpan > 0 {
			ax, ay = d.body.FaceX, d.body.FaceY
		} else {
			ax, ay = s.Self.VX, s.Self.VY
		}
	}
	if math.Hypot(ax, ay) < 1e-9 {
		x, y := d.X, d.Y
		if d.Shape == unit.ThreatRect {
			x, y = (x+d.X2)/2, (y+d.Y2)/2
		}
		ax, ay = s.Self.X-x, s.Self.Y-y
	}
	if n := math.Hypot(ax, ay); n > 1e-9 {
		return -ay / n, ax / n
	}
	return 0, 1
}

func dodgeScale(s unit.Sense) float64 {
	if s.VelocityScale > 0 {
		return s.VelocityScale
	}
	return 1
}

func planDodge(state *unit.MatchState, s unit.Sense, remaining float64) (dx, dy float64, threat unit.Threat, ok bool) {
	all := nearbyDangers(state, s)
	if len(all) == 0 {
		return 0, 0, unit.Threat{}, false
	}
	v := math.Hypot(s.Self.VX, s.Self.VY)
	var ux, uy float64
	if v > 1e-9 {
		ux, uy = s.Self.VX/v, s.Self.VY/v
	}
	scale, after := dodgeScale(s), v
	if remaining > 0 {
		after = cruise * scale
	}
	normal := predictRoute(s, ux, uy, v, after, remaining, forecastLimit)
	first, hitAt := -1, forecastLimit+1
	for i, d := range all {
		if t, hit := d.impact(s, normal, forecastLimit); hit {
			if t < hitAt-1e-9 || (first >= 0 && math.Abs(t-hitAt) <= 1e-9 && d.From < all[first].From) {
				first, hitAt = i, t
			}
		}
	}
	if first < 0 || hitAt <= 1e-9 {
		return 0, 0, unit.Threat{}, false
	}
	d := all[first]
	horizon := math.Max(anticipation, hitAt+sampleStep)
	nx, ny := d.dodgeNormal(s, s.Time+hitAt)
	// A route relying on an unpredicted bounce before the hit is not safe.
	needed := scale * (sideDist + cruise*math.Max(0, hitAt-sideDuration))
	for _, side := range [2]float64{1, -1} {
		x, y := side*nx, side*ny
		r := predictRoute(s, x, y, sideSpeed*scale, cruise*scale, sideDuration, horizon)
		if r.length+1e-9 < needed {
			continue
		}
		safe := true
		for _, other := range all {
			if _, hit := other.impact(s, r, horizon); hit {
				safe = false
				break
			}
		}
		if safe {
			return x, y, d.Threat, true
		}
	}
	return 0, 0, unit.Threat{}, false
}
