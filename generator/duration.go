package generator

import (
	"fmt"
	"go/types"
	"time"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/internal/typeutil"
	"github.com/asp24/gendi/ir"
)

func isTimeDuration(t types.Type) bool {
	return typeutil.IsDuration(t)
}

func durationLiteral(lit di.Literal) (int64, error) {
	if lit.Kind != di.LiteralString {
		return 0, fmt.Errorf("duration literal must be string")
	}
	d, err := time.ParseDuration(lit.String())
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", lit.String(), err)
	}
	return int64(d), nil
}

func durationLiteralValue(lit ir.LiteralValue) (int64, error) {
	if lit.Type != ir.DurationLiteral {
		return 0, fmt.Errorf("duration literal must be string or int")
	}
	switch v := lit.Value.(type) {
	case string:
		d, err := time.ParseDuration(v)
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q: %w", v, err)
		}
		return int64(d), nil
	case int64:
		return v, nil
	default:
		return 0, fmt.Errorf("duration literal must be string or int")
	}
}
