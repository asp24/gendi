package generator

import (
	"bytes"
	"fmt"
	"strconv"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/xmaps"
)

type ParametersRenderer struct {
	importManager *ImportManager
}

func NewParametersRenderer(importManager *ImportManager) *ParametersRenderer {
	return &ParametersRenderer{importManager: importManager}
}

func (r *ParametersRenderer) literalExpr(lit di.Literal) (string, error) {
	switch lit.Kind {
	case di.LiteralString:
		return strconv.Quote(lit.String()), nil
	case di.LiteralInt:
		return fmt.Sprintf("%d", lit.Int()), nil
	case di.LiteralFloat:
		return fmt.Sprintf("%v", lit.Float()), nil
	case di.LiteralBool:
		return fmt.Sprintf("%t", lit.Bool()), nil
	case di.LiteralNull:
		return "nil", nil
	default:
		return "", fmt.Errorf("unsupported literal kind %d", lit.Kind)
	}
}

func (r *ParametersRenderer) Render(params map[string]di.Parameter, containerName string, w *bytes.Buffer) error {
	if len(params) == 0 {
		return nil
	}

	r.importManager.ReserveAliases("parameters")

	fmt.Fprintf(w, "var %s = parameters.NewProviderMap(map[string]any{\n", defaultParametersName(containerName))
	for _, name := range xmaps.OrderedKeys(params) {
		param := params[name]
		lit, err := r.literalExpr(param.Value)
		if err != nil {
			return fmt.Errorf("parameter %q: %w", name, err)
		}
		fmt.Fprintf(w, "\t%q: %s,\n", name, lit)
	}
	w.WriteString("})\n\n")

	return nil
}
