package generator

import (
	"testing"

	di "github.com/asp24/gendi"
	"github.com/asp24/gendi/ir"
)

func TestLiteralExpr_Float(t *testing.T) {
	r := &ParametersRenderer{}

	tests := []struct {
		name  string
		input float64
		want  string
	}{
		{"whole number", 42.0, "42.0"},
		{"decimal", 3.14, "3.14"},
		{"zero", 0.0, "0.0"},
		{"scientific large", 1e10, "1e+10"},
		{"negative", -1.5, "-1.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.literalExpr(di.NewFloatLiteral(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("literalExpr(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLiteralValueExpr_Float(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  string
	}{
		{"whole number", 42.0, "42.0"},
		{"decimal", 3.14, "3.14"},
		{"zero", 0.0, "0.0"},
		{"scientific large", 1e10, "1e+10"},
		{"negative", -1.5, "-1.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lit := ir.LiteralValue{Type: ir.FloatLiteral, Value: tt.input}
			got, err := literalValueExpr(lit)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("literalValueExpr(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
