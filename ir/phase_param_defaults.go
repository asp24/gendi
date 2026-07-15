package ir

import (
	"fmt"
	"go/types"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/parameters"
	"github.com/asp24/gendi/srcloc"
	"github.com/asp24/gendi/xmaps"
)

// paramDefaultValidatorPhase checks every declared parameter default against
// the target type of each injection site by executing the standard caster,
// so conversion failures surface at generation time instead of runtime.
// Runtime-provided values remain checked at construction time.
type paramDefaultValidatorPhase struct{}

func (p *paramDefaultValidatorPhase) Apply(cfg *di.Config, container *Container) error {
	for _, id := range xmaps.OrderedKeys(container.Services) {
		svc := container.Services[id]
		if svc.Constructor == nil {
			continue
		}
		for i, arg := range svc.Constructor.Args {
			if err := p.validateArg(cfg, svc.ID, i, arg); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *paramDefaultValidatorPhase) validateArg(cfg *di.Config, svcID string, idx int, arg *Argument) error {
	if arg == nil {
		return nil
	}
	if arg.Kind == SpreadArg {
		return p.validateArg(cfg, svcID, idx, arg.Inner)
	}
	if arg.Kind != ParamRefArg || arg.Parameter == nil {
		return nil
	}
	decl, ok := cfg.Parameters[arg.Parameter.Name]
	if !ok {
		return nil // runtime-only parameter; the value is checked at construction time
	}
	if err := castDefault(decl.Value, arg.Type); err != nil {
		return srcloc.Errorf(decl.SourceLoc, "service %q arg[%d]: parameter %q default: %v",
			svcID, idx, arg.Parameter.Name, err)
	}
	return nil
}

// castDefault runs the standard caster on a declared default exactly as the
// generated container will at runtime. The raw value fed here (int64 for int
// literals) and the value stored in the generated defaults map (untyped int
// constant) differ only in width, never in conversion outcome.
func castDefault(lit di.Literal, target types.Type) error {
	var raw any
	switch lit.Kind {
	case di.LiteralString:
		raw = lit.String()
	case di.LiteralInt:
		raw = lit.Int()
	case di.LiteralFloat:
		raw = lit.Float()
	case di.LiteralBool:
		raw = lit.Bool()
	default:
		return fmt.Errorf("unsupported default literal kind %d", lit.Kind)
	}
	method, _, err := CasterMethod(target)
	if err != nil {
		return err
	}
	c := parameters.StandardCaster{}
	switch method {
	case "ToString":
		_, err = c.ToString(raw)
	case "ToBool":
		_, err = c.ToBool(raw)
	case "ToInt":
		_, err = c.ToInt(raw)
	case "ToInt8":
		_, err = c.ToInt8(raw)
	case "ToInt16":
		_, err = c.ToInt16(raw)
	case "ToInt32":
		_, err = c.ToInt32(raw)
	case "ToInt64":
		_, err = c.ToInt64(raw)
	case "ToUint":
		_, err = c.ToUint(raw)
	case "ToUint8":
		_, err = c.ToUint8(raw)
	case "ToUint16":
		_, err = c.ToUint16(raw)
	case "ToUint32":
		_, err = c.ToUint32(raw)
	case "ToUint64":
		_, err = c.ToUint64(raw)
	case "ToFloat32":
		_, err = c.ToFloat32(raw)
	case "ToFloat64":
		_, err = c.ToFloat64(raw)
	case "ToDuration":
		_, err = c.ToDuration(raw)
	case "ToTime":
		_, err = c.ToTime(raw)
	default:
		err = fmt.Errorf("unknown caster method %s", method)
	}
	return err
}
