package ir

import (
	"strings"
	"testing"

	di "github.com/asp24/gendi"
)

func TestDetectDecoratorCycles(t *testing.T) {
	tests := []struct {
		name        string
		services    map[string]*Service
		expectError bool
		errorMsg    string
	}{
		{
			name: "no cycle - linear chain",
			services: map[string]*Service{
				"base": {ID: "base", Public: true},
				"dec1": {ID: "dec1", Decorates: &Service{ID: "base"}},
				"dec2": {ID: "dec2", Decorates: &Service{ID: "base"}},
			},
			expectError: false,
		},
		{
			name: "simple cycle - A decorates B, B decorates A",
			services: map[string]*Service{
				"decA": {ID: "decA"},
				"decB": {ID: "decB"},
			},
			expectError: true,
			errorMsg:    "circular decorator chain",
		},
		{
			name: "three-way cycle - A -> B -> C -> A",
			services: map[string]*Service{
				"decA": {ID: "decA"},
				"decB": {ID: "decB"},
				"decC": {ID: "decC"},
			},
			expectError: true,
			errorMsg:    "circular decorator chain",
		},
		{
			name: "self-decoration",
			services: map[string]*Service{
				"dec": {ID: "dec"},
			},
			expectError: true,
			errorMsg:    "circular decorator chain",
		},
		{
			name: "no decorators",
			services: map[string]*Service{
				"svc1": {ID: "svc1", Public: true},
				"svc2": {ID: "svc2"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup circular references for cycle test cases
			if tt.name == "simple cycle - A decorates B, B decorates A" {
				tt.services["decA"].Decorates = tt.services["decB"]
				tt.services["decB"].Decorates = tt.services["decA"]
			}
			if tt.name == "three-way cycle - A -> B -> C -> A" {
				tt.services["decA"].Decorates = tt.services["decB"]
				tt.services["decB"].Decorates = tt.services["decC"]
				tt.services["decC"].Decorates = tt.services["decA"]
			}
			if tt.name == "self-decoration" {
				tt.services["dec"].Decorates = tt.services["dec"]
			}

			ctx := &buildContext{
				cfg:      &di.Config{},
				services: tt.services,
			}

			v := &validator{}
			err := v.detectDecoratorCycles(ctx)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateIncludesDecoratorCycleCheck(t *testing.T) {
	// Test that the main validate() function calls decorator cycle detection
	decA := &Service{ID: "decA"}
	decB := &Service{ID: "decB"}
	decA.Decorates = decB
	decB.Decorates = decA

	ctx := &buildContext{
		cfg: &di.Config{},
		services: map[string]*Service{
			"decA":   decA,
			"decB":   decB,
			"public": {ID: "public", Public: true}, // Need at least one public service
		},
	}

	v := &validator{}
	err := v.validate(ctx)

	if err == nil {
		t.Error("expected validation to catch circular decorator chain")
	}
	if !strings.Contains(err.Error(), "circular decorator chain") {
		t.Errorf("expected circular decorator chain error, got: %v", err)
	}
}
