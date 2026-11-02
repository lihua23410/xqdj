package sim

import (
	"xqdj/internal/audit"
	unitpkg "xqdj/internal/unit"
)

type actorBegin struct{ origin audit.Origin }
type actorEnd struct{ done chan struct{} }
type actorCommand struct {
	origin  audit.Origin
	command unitpkg.Cmd
}

// One private output stream per actor. Begin/end markers share the same FIFO
// as commands so an asynchronous next event cannot relabel an earlier command.
func actorOutput(u *unit, engine chan<- unitpkg.Cmd) (chan unitpkg.Cmd, <-chan struct{}) {
	out := make(chan unitpkg.Cmd)
	exited := make(chan struct{})
	go func() {
		defer close(exited)
		var origin audit.Origin
		for c := range out {
			switch c := c.(type) {
			case actorBegin:
				origin = c.origin
			case actorEnd:
				close(c.done)
			default:
				engine <- actorCommand{origin: origin, command: c}
			}
		}
	}()
	return out, exited
}
