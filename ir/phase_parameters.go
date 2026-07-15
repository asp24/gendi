package ir

import (
	di "github.com/gendi-org/gendi"
	"github.com/gendi-org/gendi/srcloc"
)

// parameterPhase registers declared parameters (defaults) from config.
// Target types are contextual — they come from each usage — so declaration
// carries no type of its own.
type parameterPhase struct{}

// Apply converts config parameters to IR parameters
func (p *parameterPhase) Apply(cfg *di.Config, container *Container) error {
	for name, param := range cfg.Parameters {
		if param.Value.IsNull() {
			return srcloc.Errorf(param.SourceLoc, "parameter %q: null value is not supported", name)
		}
		container.Parameters[name] = &Parameter{Name: name}
	}
	return nil
}
