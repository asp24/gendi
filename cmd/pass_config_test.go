package cmd

import (
	"testing"

	di "github.com/asp24/gendi"
)

type testPass struct {
	name         string
	runByDefault bool
}

func (p *testPass) Name() string                              { return p.name }
func (p *testPass) RunByDefault() bool                        { return p.runByDefault }
func (p *testPass) Process(cfg *di.Config) (*di.Config, error) { return cfg, nil }

func makePass(name string, runByDefault bool) *testPass {
	return &testPass{name: name, runByDefault: runByDefault}
}

func TestPassConfig_ResolvePasses(t *testing.T) {
	cases := []struct {
		name      string
		enabled   map[string]struct{}
		passes    []di.SelectablePass
		wantNames []string
	}{
		{
			name:      "default-on pass is included",
			passes:    []di.SelectablePass{makePass("a", true)},
			wantNames: []string{"a"},
		},
		{
			name:      "default-off pass is excluded",
			passes:    []di.SelectablePass{makePass("a", false)},
			wantNames: []string{},
		},
		{
			name:      "default-off pass included when enabled",
			enabled:   map[string]struct{}{"a": {}},
			passes:    []di.SelectablePass{makePass("a", false)},
			wantNames: []string{"a"},
		},
		{
			name:      "duplicate pass name runs only once",
			passes:    []di.SelectablePass{makePass("a", true), makePass("a", true)},
			wantNames: []string{"a"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pc := PassConfig{Enabled: tc.enabled}
			result, err := pc.resolvePasses(tc.passes)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != len(tc.wantNames) {
				t.Fatalf("got %d passes, want %d", len(result), len(tc.wantNames))
			}
			for i, p := range result {
				if p.Name() != tc.wantNames[i] {
					t.Errorf("pass[%d] name = %q, want %q", i, p.Name(), tc.wantNames[i])
				}
			}
		})
	}
}

func TestPassConfig_ResolvePasses_Errors(t *testing.T) {
	cases := []struct {
		name    string
		enabled map[string]struct{}
		passes  []di.SelectablePass
	}{
		{
			name:    "unknown name in --enable-pass",
			enabled: map[string]struct{}{"unknown": {}},
			passes:  []di.SelectablePass{makePass("foo", true)},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pc := PassConfig{Enabled: tc.enabled}
			_, err := pc.resolvePasses(tc.passes)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}
