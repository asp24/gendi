package ir

import (
	"strings"
	"testing"
)

func TestDetectDecoratorCycles(t *testing.T) {
	tests := []struct {
		name        string
		decoratesBy map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "no cycle - linear chain",
			decoratesBy: map[string]string{
				"dec1": "base",
				"dec2": "base",
			},
			expectError: false,
		},
		{
			name: "simple cycle - A decorates B, B decorates A",
			decoratesBy: map[string]string{
				"decA": "decB",
				"decB": "decA",
			},
			expectError: true,
			errorMsg:    "circular decorator chain",
		},
		{
			name: "three-way cycle - A -> B -> C -> A",
			decoratesBy: map[string]string{
				"decA": "decB",
				"decB": "decC",
				"decC": "decA",
			},
			expectError: true,
			errorMsg:    "circular decorator chain",
		},
		{
			name: "self-decoration",
			decoratesBy: map[string]string{
				"dec": "dec",
			},
			expectError: true,
			errorMsg:    "circular decorator chain",
		},
		{
			name:        "no decorators",
			decoratesBy: map[string]string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &decoratorResolver{}
			err := resolver.detectDecoratorCycles(tt.decoratesBy)

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
