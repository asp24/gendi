package ir

import (
	"strings"
	"testing"
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

			expander := &decoratorExpander{}
			err := expander.detectDecoratorCycles(tt.services)

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
