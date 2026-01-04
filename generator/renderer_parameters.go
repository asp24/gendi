package generator

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"

	di "github.com/asp24/gendi"
)

type RendererParameters struct {
	importManager *ImportManager
}

func (r *RendererParameters) literalExpr(lit di.Literal) (string, error) {
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

func (r *RendererParameters) Render(params map[string]di.Parameter, w *bytes.Buffer) error {
	if len(params) == 0 {
		return nil
	}

	r.importManager.ReserveAliases("parameters")

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	w.WriteString("var DefaultParameters = parameters.NewProviderMap(map[string]any{\n")
	for _, name := range keys {
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
