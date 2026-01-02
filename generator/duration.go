package generator

import (
	"fmt"
	"time"

	"github.com/asp24/gendi/ir"
)

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
