// Package audit enforces the boundary between body contact and attack damage.
// It is a regression guard, not a sandbox for hostile Go code.
package audit

import (
	"fmt"
	"xqdj/internal/unit"
)

// Origin is supplied by the engine, never by the actor's command payload.
type Origin struct {
	ID             uint64
	Kind, Role     string
	Collision, Fan bool
}

func CheckCommand(o Origin, cmd unit.Cmd) {
	if hit, ok := cmd.(unit.FanContact); ok && (!o.Fan || hit.From != o.ID) {
		panic(fmt.Sprintf("audit: forged fan contact from %s #%d (claimed From=%d)", o.Kind, o.ID, hit.From))
	}
	if d, ok := cmd.(unit.Damage); ok {
		if o.Role == unit.RoleFighter && o.Collision {
			panic(fmt.Sprintf("audit: forbidden fighter body collision damage: %s #%d (claimed From=%d); use an attack fan", o.Kind, o.ID, d.From))
		}
		if o.Fan && d.From != o.ID {
			panic(fmt.Sprintf("audit: fan %s #%d must quote damage from itself, got From=%d", o.Kind, o.ID, d.From))
		}
	}
}
