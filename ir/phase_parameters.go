package ir

import "fmt"

// parameterPhase builds parameters from config
type parameterPhase struct{}

// build converts config parameters to IR parameters
func (p *parameterPhase) build(ctx *buildContext) error {
	for name, param := range ctx.cfg.Parameters {
		if param.Type == "" {
			return fmt.Errorf("parameter %q missing type", name)
		}
		paramType, err := ctx.resolver.LookupType(param.Type)
		if err != nil {
			return fmt.Errorf("parameter %q type: %w", name, err)
		}

		litVal, err := convertLiteral(param.Value, paramType)
		if err != nil {
			return fmt.Errorf("parameter %q value: %w", name, err)
		}

		ctx.parameters[name] = &Parameter{
			Name:  name,
			Type:  paramType,
			Value: litVal,
		}
	}
	return nil
}
