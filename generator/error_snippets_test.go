package generator

import "testing"

func TestCountFormatSpecifiers(t *testing.T) {
	tests := []struct {
		name   string
		format string
		want   int
	}{
		{"empty string", "", 0},
		{"no specifiers", "hello world", 0},
		{"single %s", "%s", 1},
		{"single %d", "%d", 1},
		{"single %q", "%q", 1},
		{"multiple specifiers", "%s %d %q", 3},
		{"escaped percent", "%%", 0},
		{"escaped percent with text", "100%% complete", 0},
		{"mixed escaped and real", "%%s %s %%", 1},
		{"specifier after escaped", "%% %s", 1},
		{"multiple escaped", "%%%% test", 0},
		{"arg format", "arg[%d]", 1},
		{"param format", "param %q", 1},
		{"no args text", "constructor", 0},
		{"trailing percent", "test%", 1},
		{"complex format", "%s: %d%% of %q", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countFormatSpecifiers(tt.format)
			if got != tt.want {
				t.Errorf("countFormatSpecifiers(%q) = %d, want %d", tt.format, got, tt.want)
			}
		})
	}
}

func TestErrorSnippetBuilderWithEscapedPercent(t *testing.T) {
	// This test verifies that escaped percent signs don't cause argument misalignment
	snippet := NewErrorSnippet("test-service").
		WithContext("progress 100%%").
		WithContext("param %q", "timeout").
		Build()

	// The snippet should contain the parameter name "timeout" properly quoted
	if snippet == "" {
		t.Error("Expected non-empty snippet")
	}

	// Verify it contains the timeout parameter
	expected := `"timeout"`
	if !contains(snippet, expected) {
		t.Errorf("Expected snippet to contain %s, got: %s", expected, snippet)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
