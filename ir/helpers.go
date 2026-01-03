package ir

import (
	"fmt"
	"go/types"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/typeres"
)

// convertLiteral converts a di.Literal to IR LiteralValue
func convertLiteral(lit di.Literal, targetType types.Type) (LiteralValue, error) {
	if typeres.IsDuration(targetType) {
		return convertDurationLiteral(lit)
	}

	switch lit.Kind {
	case di.LiteralString:
		return LiteralValue{Type: StringLiteral, Value: lit.String()}, nil
	case di.LiteralInt:
		return LiteralValue{Type: IntLiteral, Value: lit.Int()}, nil
	case di.LiteralFloat:
		return LiteralValue{Type: FloatLiteral, Value: lit.Float()}, nil
	case di.LiteralBool:
		return LiteralValue{Type: BoolLiteral, Value: lit.Bool()}, nil
	case di.LiteralNull:
		return LiteralValue{Type: NullLiteral, Value: nil}, nil
	default:
		return LiteralValue{}, fmt.Errorf("unsupported literal kind %d", lit.Kind)
	}
}

// convertDurationLiteral converts a duration literal (string "1s" or int nanoseconds)
func convertDurationLiteral(lit di.Literal) (LiteralValue, error) {
	switch lit.Kind {
	case di.LiteralString:
		// Parse as duration string - will be handled by generator
		return LiteralValue{Type: DurationLiteral, Value: lit.String()}, nil
	case di.LiteralInt:
		return LiteralValue{Type: DurationLiteral, Value: lit.Int()}, nil
	default:
		return LiteralValue{}, fmt.Errorf("duration must be string or int")
	}
}
