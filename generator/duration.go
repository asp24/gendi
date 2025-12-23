package generator

import (
	"fmt"
	"go/types"
	"time"

	di "github.com/asp24/gendi"
)

func isTimeDuration(t types.Type) bool {
	for {
		ptr, ok := t.(*types.Pointer)
		if !ok {
			break
		}
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Name() != "Duration" {
		return false
	}
	pkg := obj.Pkg()
	return pkg != nil && pkg.Path() == "time"
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
