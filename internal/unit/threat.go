package unit

type ThreatShape uint8

const (
	ThreatCircle ThreatShape = iota
	ThreatCapsule
	ThreatRect
	ThreatRay // clipped at walls, with flat start and end as in a radar sweep
)

// Threat describes an existing spatial hit test. Times use the match clock.
// ActiveAt == Until describes a single impact. ReadyAt is optional and
// contains per-target hit cooldowns. Published slices and maps are immutable.
type Threat struct {
	Shape                ThreatShape
	From                 uint64
	Target               uint64 // optional target when a forecast only applies to one unit
	Slot                 int    // negative for neutral hazards
	X, Y, X2, Y2, Radius float64
	VX, VY, Spin         float64 // translation and rotation around X,Y
	ObservedAt           float64
	ActiveAt, Until      float64
	ReadyAt              map[uint64]float64
}

// SetThreats replaces the publishing unit's threats without changing its attacks.
type SetThreats struct {
	UnitID uint64
	Areas  []Threat
}

func ThreatReadyAt(last map[uint64]float64, gap, now float64) map[uint64]float64 {
	var ready map[uint64]float64
	for id, t := range last {
		if t+gap > now {
			if ready == nil {
				ready = make(map[uint64]float64)
			}
			ready[id] = t + gap
		}
	}
	return ready
}
