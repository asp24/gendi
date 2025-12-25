package generator

import (
	"fmt"
	"strconv"

	"github.com/asp24/gendi/ir"
)

func literalValueExpr(lit ir.LiteralValue) (string, error) {
	switch lit.Type {
	case ir.StringLiteral:
		v, ok := lit.Value.(string)
		if !ok {
			return "", fmt.Errorf("string literal must be string")
		}
		return strconv.Quote(v), nil
	case ir.IntLiteral:
		v, ok := lit.Value.(int64)
		if !ok {
			return "", fmt.Errorf("int literal must be int64")
		}
		return fmt.Sprintf("%d", v), nil
	case ir.FloatLiteral:
		v, ok := lit.Value.(float64)
		if !ok {
			return "", fmt.Errorf("float literal must be float64")
		}
		return fmt.Sprintf("%v", v), nil
	case ir.BoolLiteral:
		v, ok := lit.Value.(bool)
		if !ok {
			return "", fmt.Errorf("bool literal must be bool")
		}
		return fmt.Sprintf("%t", v), nil
	case ir.NullLiteral:
		return "nil", nil
	case ir.DurationLiteral:
		return "", fmt.Errorf("duration literals require conversion")
	default:
		return "", fmt.Errorf("unsupported literal kind %d", lit.Type)
	}
}
