package ir

import (
	"fmt"

	di "github.com/asp24/gendi"
)

// parameterPhase builds parameters from config
type parameterPhase struct {
	resolver TypeResolver
}

// Apply converts config parameters to IR parameters
func (p *parameterPhase) Apply(cfg *di.Config, container *Container) error {
	for name, param := range cfg.Parameters {
		if param.Type == "" {
			return fmt.Errorf("parameter %q missing type", name)
		}
		paramType, err := p.resolver.LookupType(param.Type)
		if err != nil {
			return fmt.Errorf("parameter %q type: %w", name, err)
		}

		litVal, err := convertLiteral(param.Value, paramType)
		if err != nil {
			return fmt.Errorf("parameter %q value: %w", name, err)
		}

		container.Parameters[name] = &Parameter{
			Name:  name,
			Type:  paramType,
			Value: litVal,
		}
	}
	return nil
}
