package ir

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"
	"testing"

	di "github.com/asp24/gendi"
)

// testResolver implements TypeResolver for arg_resolver tests.
type testResolver struct {
	vars map[string]types.Object // "pkg.Name" → Object
}

func (r *testResolver) LookupType(typeStr string) (types.Type, error) {
	return types.Typ[types.String], nil
}
func (r *testResolver) LookupFunc(pkgPath, name string) (*types.Func, error) {
	return nil, fmt.Errorf("not found")
}
func (r *testResolver) LookupMethod(recv types.Type, name string) (*types.Func, error) {
	return nil, fmt.Errorf("not found")
}
func (r *testResolver) InstantiateFunc(fn *types.Func, typeArgs []string) (*types.Signature, []types.Type, error) {
	return nil, nil, fmt.Errorf("not supported")
}
func (r *testResolver) LookupVar(pkgPath, name string) (types.Object, error) {
	key := pkgPath + "." + name
	if obj, ok := r.vars[key]; ok {
		return obj, nil
	}
	return nil, fmt.Errorf("symbol %s not found in %s", name, pkgPath)
}

// makeStruct creates a named struct type with the given fields.
// Each field is (name string, type types.Type, exported bool).
func makeStruct(pkgName string, fields ...any) *types.Struct {
	pkg := types.NewPackage("test/"+pkgName, pkgName)
	var flds []*types.Var
	for i := 0; i < len(fields); i += 2 {
		name := fields[i].(string)
		typ := fields[i+1].(types.Type)
		flds = append(flds, types.NewField(token.NoPos, pkg, name, typ, false))
	}
	return types.NewStruct(flds, nil)
}

// makePkgVar creates a package-level *types.Var for use with the testResolver.
func makePkgVar(pkgPath, name string, typ types.Type) types.Object {
	pkg := types.NewPackage(pkgPath, pkgPath[strings.LastIndex(pkgPath, "/")+1:])
	return types.NewVar(token.NoPos, pkg, name, typ)
}

func TestTaggedElementTypeAssignable(t *testing.T) {
	container := NewContainer()
	r := &argResolver{}
	arg := di.Argument{Kind: di.ArgTagged, Value: "tag.test"}

	if _, err := r.resolve(container, "svc.one", 0, arg, types.NewSlice(types.Typ[types.Int])); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	emptyIface := types.NewInterfaceType(nil, nil)
	emptyIface.Complete()
	if _, err := r.resolve(container, "svc.two", 0, arg, types.NewSlice(emptyIface)); err != nil {
		t.Fatalf("expected assignable element type, got %v", err)
	}
}

func TestTaggedElementTypeNotAssignable(t *testing.T) {
	container := NewContainer()
	r := &argResolver{}
	arg := di.Argument{Kind: di.ArgTagged, Value: "tag.test"}

	emptyIface := types.NewInterfaceType(nil, nil)
	emptyIface.Complete()
	if _, err := r.resolve(container, "svc.one", 0, arg, types.NewSlice(emptyIface)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := r.resolve(container, "svc.two", 0, arg, types.NewSlice(types.Typ[types.Int]))
	if err == nil || !strings.Contains(err.Error(), "not assignable") {
		t.Fatalf("expected element type mismatch error, got %v", err)
	}
}

func TestResolve(t *testing.T) {
	stringType := types.Typ[types.String]
	intType := types.Typ[types.Int]

	resolver := &testResolver{
		vars: map[string]types.Object{
			"mypkg.MyVar": makePkgVar("mypkg", "MyVar", stringType),
		},
	}

	container := NewContainer()
	container.Services["dep"] = &Service{ID: "dep", Type: stringType}
	container.Parameters["db.host"] = &Parameter{Name: "db.host", Type: stringType}

	// Also add a tagged service for spread test
	container.Services["handler_svc"] = &Service{
		ID:   "handler_svc",
		Type: types.NewSlice(intType),
	}

	tests := []struct {
		name      string
		arg       di.Argument
		paramType types.Type
		wantKind  ArgumentKind
		wantErr   string
	}{
		{
			name:      "service_ref_ok",
			arg:       di.Argument{Kind: di.ArgServiceRef, Value: "dep"},
			paramType: stringType,
			wantKind:  ServiceRefArg,
		},
		{
			name:      "service_ref_unknown",
			arg:       di.Argument{Kind: di.ArgServiceRef, Value: "missing"},
			paramType: stringType,
			wantErr:   "unknown service",
		},
		{
			name:      "inner_error",
			arg:       di.Argument{Kind: di.ArgInner, Value: "@.inner"},
			paramType: stringType,
			wantErr:   "DecoratorPass",
		},
		{
			name:      "param_found",
			arg:       di.Argument{Kind: di.ArgParam, Value: "db.host"},
			paramType: stringType,
			wantKind:  ParamRefArg,
		},
		{
			name:      "param_not_found_runtime",
			arg:       di.Argument{Kind: di.ArgParam, Value: "unknown.param"},
			paramType: intType,
			wantKind:  ParamRefArg,
		},
		{
			name:      "param_type_mismatch",
			arg:       di.Argument{Kind: di.ArgParam, Value: "db.host"},
			paramType: intType,
			wantErr:   "type mismatch",
		},
		{
			name:      "tagged_not_slice",
			arg:       di.Argument{Kind: di.ArgTagged, Value: "sometag"},
			paramType: stringType,
			wantErr:   "requires slice type",
		},
		{
			name:      "spread_not_slice",
			arg:       di.Argument{Kind: di.ArgSpread, Value: "@dep"},
			paramType: stringType,
			wantErr:   "variadic parameters",
		},
		{
			name:      "spread_ok",
			arg:       di.Argument{Kind: di.ArgSpread, Value: "@handler_svc"},
			paramType: types.NewSlice(intType),
			wantKind:  SpreadArg,
		},
		{
			name:      "spread_inner_error",
			arg:       di.Argument{Kind: di.ArgSpread, Value: "@missing"},
			paramType: types.NewSlice(intType),
			wantErr:   "unknown service",
		},
		{
			name:      "goref_ok",
			arg:       di.Argument{Kind: di.ArgGoRef, Value: "mypkg.MyVar"},
			paramType: stringType,
			wantKind:  GoRefArg,
		},
		{
			name:      "goref_invalid_name",
			arg:       di.Argument{Kind: di.ArgGoRef, Value: "noDotHere"},
			paramType: stringType,
			wantErr:   "invalid",
		},
		{
			name:      "goref_lookup_error",
			arg:       di.Argument{Kind: di.ArgGoRef, Value: "mypkg.Missing"},
			paramType: stringType,
			wantErr:   "not found",
		},
		{
			name:      "goref_type_mismatch",
			arg:       di.Argument{Kind: di.ArgGoRef, Value: "mypkg.MyVar"},
			paramType: intType,
			wantErr:   "not assignable",
		},
		{
			name:      "literal_string",
			arg:       di.Argument{Kind: di.ArgLiteral, Literal: di.NewStringLiteral("hello")},
			paramType: stringType,
			wantKind:  LiteralArg,
		},
		{
			name:      "unknown_kind",
			arg:       di.Argument{Kind: di.ArgumentKind(99)},
			paramType: stringType,
			wantErr:   "unknown argument kind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &argResolver{typeResolver: resolver}
			result, err := r.resolve(container, "svc", 0, tt.arg, tt.paramType)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got: %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Kind != tt.wantKind {
				t.Fatalf("expected kind %d, got %d", tt.wantKind, result.Kind)
			}
		})
	}
}

func TestResolveFieldAccess(t *testing.T) {
	stringType := types.Typ[types.String]
	intType := types.Typ[types.Int]

	// Inner struct: type Inner struct { DSN string }
	innerStruct := makeStruct("cfg", "DSN", stringType)
	// Outer struct: type Config struct { Host string; Port int; Database Inner; secret string }
	pkg := types.NewPackage("test/cfg", "cfg")
	outerStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "Host", stringType, false),
		types.NewField(token.NoPos, pkg, "Port", intType, false),
		types.NewField(token.NoPos, pkg, "Database", innerStruct, false),
		types.NewField(token.NoPos, pkg, "secret", stringType, false), // unexported
	}, nil)
	ptrOuter := types.NewPointer(outerStruct)

	resolver := &testResolver{
		vars: map[string]types.Object{
			"mypkg.DefaultCfg": makePkgVar("mypkg", "DefaultCfg", ptrOuter),
		},
	}

	container := NewContainer()
	container.Services["config"] = &Service{ID: "config", Type: ptrOuter}
	container.Services["config.db"] = &Service{ID: "config.db", Type: ptrOuter} // dotted ID

	tests := []struct {
		name      string
		arg       di.Argument
		paramType types.Type
		wantErr   string
	}{
		// Service field access
		{
			name:      "service_field_ok",
			arg:       di.Argument{Kind: di.ArgFieldAccessService, Value: "config.Host"},
			paramType: stringType,
		},
		{
			name:      "service_nested_field",
			arg:       di.Argument{Kind: di.ArgFieldAccessService, Value: "config.Database.DSN"},
			paramType: stringType,
		},
		{
			name:      "service_dotted_id",
			arg:       di.Argument{Kind: di.ArgFieldAccessService, Value: "config.db.Host"},
			paramType: stringType,
		},
		{
			name:      "service_no_field",
			arg:       di.Argument{Kind: di.ArgFieldAccessService, Value: "config"},
			paramType: stringType,
			wantErr:   "requires at least one field",
		},
		{
			name:      "service_not_found",
			arg:       di.Argument{Kind: di.ArgFieldAccessService, Value: "missing.Host"},
			paramType: stringType,
			wantErr:   "no matching service",
		},
		{
			name:      "service_unknown_field",
			arg:       di.Argument{Kind: di.ArgFieldAccessService, Value: "config.NoSuch"},
			paramType: stringType,
			wantErr:   "not found",
		},
		{
			name:      "service_unexported_field",
			arg:       di.Argument{Kind: di.ArgFieldAccessService, Value: "config.secret"},
			paramType: stringType,
			wantErr:   "not found", // unexported fields are not found by LookupFieldOrMethod with nil package
		},
		{
			name:      "service_type_mismatch",
			arg:       di.Argument{Kind: di.ArgFieldAccessService, Value: "config.Host"},
			paramType: intType,
			wantErr:   "not assignable",
		},
		// Go ref field access
		{
			name:      "goref_field_ok",
			arg:       di.Argument{Kind: di.ArgFieldAccessGo, Value: "mypkg.DefaultCfg.Host"},
			paramType: stringType,
		},
		{
			name:      "goref_nested_field",
			arg:       di.Argument{Kind: di.ArgFieldAccessGo, Value: "mypkg.DefaultCfg.Database.DSN"},
			paramType: stringType,
		},
		{
			name:      "goref_too_few_parts",
			arg:       di.Argument{Kind: di.ArgFieldAccessGo, Value: "x.y"},
			paramType: stringType,
			wantErr:   "requires at least one field",
		},
		{
			name:      "goref_symbol_not_found",
			arg:       di.Argument{Kind: di.ArgFieldAccessGo, Value: "mypkg.Missing.Field"},
			paramType: stringType,
			wantErr:   "no matching package-level symbol",
		},
		{
			name:      "goref_unknown_field",
			arg:       di.Argument{Kind: di.ArgFieldAccessGo, Value: "mypkg.DefaultCfg.NoSuch"},
			paramType: stringType,
			wantErr:   "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &argResolver{typeResolver: resolver}
			result, err := r.resolve(container, "svc", 0, tt.arg, tt.paramType)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got: %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Kind != FieldAccessArg {
				t.Fatalf("expected FieldAccessArg, got %d", result.Kind)
			}
		})
	}
}
