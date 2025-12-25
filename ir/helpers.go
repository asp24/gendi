package ir

import (
	"fmt"
	"go/types"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/internal/typeutil"
)

// validateConstructorSignature validates that a signature returns T or (T, error)
func validateConstructorSignature(sig *types.Signature) (types.Type, bool, error) {
	res := sig.Results()
	if res.Len() == 0 || res.Len() > 2 {
		return nil, false, fmt.Errorf("constructor must return T or (T, error)")
	}
	resType := res.At(0).Type()
	returnsErr := false
	if res.Len() == 2 {
		errType := res.At(1).Type()
		if !types.Identical(errType, types.Universe.Lookup("error").Type()) {
			return nil, false, fmt.Errorf("second return value must be error")
		}
		returnsErr = true
	}
	return resType, returnsErr, nil
}

// signatureParams extracts parameter types from a function signature
func signatureParams(sig *types.Signature) []types.Type {
	params := make([]types.Type, sig.Params().Len())
	for i := 0; i < sig.Params().Len(); i++ {
		params[i] = sig.Params().At(i).Type()
	}
	return params
}

// convertLiteral converts a di.Literal to IR LiteralValue
func convertLiteral(lit di.Literal, targetType types.Type) (LiteralValue, error) {
	if typeutil.IsDuration(targetType) {
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

// resolutionTracker tracks service resolution to detect circular references
type resolutionTracker struct {
	resolving map[string]bool
	resolved  map[string]bool
}

// newResolutionTracker creates a new resolution tracker
func newResolutionTracker() *resolutionTracker {
	return &resolutionTracker{
		resolving: make(map[string]bool),
		resolved:  make(map[string]bool),
	}
}
