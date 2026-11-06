// Package api is the role-local seam for independently maintained pump parts.
package api

import "xqdj/internal/unit"

type Definition struct {
	ID        int
	Key, Name string
	Install   func(*Build)
	Weapon    func() unit.Actor
	Register  func(*unit.Pack)
}

// Pulse is an assembly-local contraction event, not an engine extension.
type Pulse struct {
	Sense       unit.Sense
	Beat        int
	Echo        bool
	SeekingPart bool // The core reserved this contraction's movement for a loose part.
}
type Detached struct{}

// Build is reconstructed from the currently installed set. Removing a part
// removes its entire contribution without inverse arithmetic or global state.
type Build struct {
	Period, Range, Damage, Pull, Push, Dash float64
	Echo, Capture                           bool
	Flow                                    func(x, y, force float64) (float64, float64)
	Steer                                   func(x, y float64, beat int) (float64, float64)
	Guards                                  []func(Frame) Guard
	WinchSpeed                              float64
}

type Frame struct {
	Time, Phase       float64
	Self              unit.Snapshot
	Suspended, Loaded bool
}

type Guard struct {
	Key     string  `json:"key"`
	Inner   float64 `json:"inner"`
	Outer   float64 `json:"outer"`
	Span    float64 `json:"span"`
	Angle   float64 `json:"angle"`
	Capture bool    `json:"capture"`
}

func Base() Build { return Build{Period: 1, Range: 108, Damage: 3, Pull: 250, Push: 185, Dash: 370} }
