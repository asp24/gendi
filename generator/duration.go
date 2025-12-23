package generator

import (
	"fmt"
	"go/types"
	"time"

	"gopkg.in/yaml.v3"
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

func durationLiteral(node yaml.Node) (int64, error) {
	if node.Tag != "!!str" {
		return 0, fmt.Errorf("duration literal must be string, got %q", node.Tag)
	}
	d, err := time.ParseDuration(node.Value)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", node.Value, err)
	}
	return int64(d), nil
}
