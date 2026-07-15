package ir

import (
	"go/types"
	"strings"
	"testing"

	di "github.com/gendi-org/gendi"
)

func buildParamUsageContainer(paramName string, target types.Type) *Container {
	container := NewContainer()
	param := &Parameter{Name: paramName}
	container.Parameters[paramName] = param
	container.Services["svc"] = &Service{
		ID: "svc",
		Constructor: &Constructor{
			Args: []*Argument{
				{Kind: ParamRefArg, Type: target, Parameter: param},
			},
		},
	}
	return container
}

func TestParamDefaultValidation(t *testing.T) {
	tests := []struct {
		name    string
		value   di.Literal
		target  types.Type
		wantErr string
	}{
		{"string default to string", di.NewStringLiteral("x"), types.Typ[types.String], ""},
		{"numeric string to int", di.NewStringLiteral("42"), types.Typ[types.Int], ""},
		{"bad string to int", di.NewStringLiteral("abc"), types.Typ[types.Int], "cannot cast"},
		{"duration string to duration", di.NewStringLiteral("5s"),
			newNamedType("time", "time", "Duration", types.Typ[types.Int64]), ""},
		{"bad duration", di.NewStringLiteral("5x"),
			newNamedType("time", "time", "Duration", types.Typ[types.Int64]), "cannot cast"},
		{"int overflow to int8", di.NewIntLiteral(300), types.Typ[types.Int8], "overflows"},
		{"float to int rejected", di.NewFloatLiteral(1.5), types.Typ[types.Int], "cannot cast"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &di.Config{
				Parameters: map[string]di.Parameter{
					"p": {Value: tt.value},
				},
			}
			container := buildParamUsageContainer("p", tt.target)
			err := (&paramDefaultValidatorPhase{}).Apply(cfg, container)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestParamDefaultValidationSkipsRuntimeParams(t *testing.T) {
	cfg := &di.Config{Parameters: map[string]di.Parameter{}}
	container := buildParamUsageContainer("runtime_only", types.Typ[types.Int])
	if err := (&paramDefaultValidatorPhase{}).Apply(cfg, container); err != nil {
		t.Fatalf("runtime-only parameter must not be validated, got %v", err)
	}
}
