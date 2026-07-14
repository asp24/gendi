package gomod

import "testing"

func TestParseModulePath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "module example.com/m\n", "example.com/m"},
		{"trailing comment", "module example.com/m // main module\n", "example.com/m"},
		{"quoted", "module \"example.com/m\"\n", "example.com/m"},
		{"backquoted", "module `example.com/m`\n", "example.com/m"},
		{"after go directive", "go 1.22\nmodule example.com/m\n", "example.com/m"},
		{"missing", "go 1.22\n", ""},
		{"commented out", "// module example.com/other\nmodule example.com/m\n", "example.com/m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseModulePath([]byte(tt.in)); got != tt.want {
				t.Errorf("ParseModulePath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
