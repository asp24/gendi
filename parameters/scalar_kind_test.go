package parameters

import (
	"reflect"
	"strings"
	"testing"
)

var allScalarKinds = []ScalarKind{
	ScalarString, ScalarBool,
	ScalarInt, ScalarInt8, ScalarInt16, ScalarInt32, ScalarInt64,
	ScalarUint, ScalarUint8, ScalarUint16, ScalarUint32, ScalarUint64,
	ScalarFloat32, ScalarFloat64,
	ScalarDuration, ScalarTime,
}

func TestScalarKindCastDispatch(t *testing.T) {
	c := StandardCaster{}
	for _, k := range allScalarKinds {
		// "1" converts to every numeric/bool/string target; Duration and
		// Time fail on parse, which still proves Cast reached the caster.
		got, err := k.Cast(c, "1")
		switch k {
		case ScalarDuration, ScalarTime:
			if err == nil {
				t.Fatalf("kind %d: expected parse error for %q, got %v", k, "1", got)
			}
		default:
			if err != nil {
				t.Fatalf("kind %d: unexpected error: %v", k, err)
			}
			if got == nil {
				t.Fatalf("kind %d: expected value", k)
			}
		}
	}

	if _, err := ScalarKind(999).Cast(c, "1"); err == nil || !strings.Contains(err.Error(), "unknown scalar kind") {
		t.Fatalf("expected unknown kind error, got %v", err)
	}
}

func TestScalarKindResolverMethod(t *testing.T) {
	resolverType := reflect.TypeOf(&Resolver{})
	for _, k := range allScalarKinds {
		name := k.ResolverMethod()
		if name == "" {
			t.Fatalf("kind %d: empty resolver method", k)
		}
		method, ok := resolverType.MethodByName(name)
		if !ok {
			t.Fatalf("kind %d: Resolver has no method %q", k, name)
		}
		// Every resolver method takes a name and returns (T, error).
		if method.Type.NumIn() != 2 || method.Type.In(1).Kind() != reflect.String || method.Type.NumOut() != 2 {
			t.Fatalf("kind %d: unexpected signature for %q: %v", k, name, method.Type)
		}
	}

	if got := ScalarKind(999).ResolverMethod(); got != "" {
		t.Fatalf("expected empty method for unknown kind, got %q", got)
	}
}
