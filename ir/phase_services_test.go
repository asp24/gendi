package ir

import (
	"strings"
	"testing"

	di "github.com/asp24/gendi"
)

func TestServicePhaseValidatesEmptyID(t *testing.T) {
	tests := []struct {
		name        string
		serviceID   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty service ID",
			serviceID:   "",
			expectError: true,
			errorMsg:    "service ID cannot be empty",
		},
		{
			name:        "whitespace-only service ID - single space",
			serviceID:   " ",
			expectError: true,
			errorMsg:    "cannot be whitespace-only",
		},
		{
			name:        "whitespace-only service ID - multiple spaces",
			serviceID:   "   ",
			expectError: true,
			errorMsg:    "cannot be whitespace-only",
		},
		{
			name:        "whitespace-only service ID - tab",
			serviceID:   "\t",
			expectError: true,
			errorMsg:    "cannot be whitespace-only",
		},
		{
			name:        "valid service ID",
			serviceID:   "my-service",
			expectError: false,
		},
		{
			name:        "valid service ID with spaces",
			serviceID:   "my service",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &di.Config{
				Services: map[string]*di.Service{
					tt.serviceID: {
						Public: true,
					},
				},
			}

			ctx := &buildContext{
				cfg:      cfg,
				services: make(map[string]*Service),
				tags:     make(map[string]*Tag),
			}

			phase := &servicePhase{}
			err := phase.build(ctx)

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
