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
		disabled  map[string]struct{}
		passes    []di.OptionalPass
		wantNames []string
	}{
		{
			name:      "default-on pass is included",
			passes:    []di.OptionalPass{makePass("a", true)},
			wantNames: []string{"a"},
		},
		{
			name:      "default-off pass is excluded",
			passes:    []di.OptionalPass{makePass("a", false)},
			wantNames: []string{},
		},
		{
			name:      "default-off pass included when enabled",
			enabled:   map[string]struct{}{"a": {}},
			passes:    []di.OptionalPass{makePass("a", false)},
			wantNames: []string{"a"},
		},
		{
			name:      "default-on pass excluded when disabled",
			disabled:  map[string]struct{}{"a": {}},
			passes:    []di.OptionalPass{makePass("a", true)},
			wantNames: []string{},
		},
		{
			name:      "duplicate pass name runs only once",
			passes:    []di.OptionalPass{makePass("a", true), makePass("a", true)},
			wantNames: []string{"a"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pc := PassConfig{Enabled: tc.enabled, Disabled: tc.disabled}
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
		name     string
		enabled  map[string]struct{}
		disabled map[string]struct{}
		passes   []di.OptionalPass
	}{
		{
			name:    "unknown name in --enable-pass",
			enabled: map[string]struct{}{"unknown": {}},
			passes:  []di.OptionalPass{makePass("foo", true)},
		},
		{
			name:     "unknown name in --disable-pass",
			disabled: map[string]struct{}{"unknown": {}},
			passes:   []di.OptionalPass{makePass("foo", true)},
		},
		{
			name:     "same name in both --enable-pass and --disable-pass",
			enabled:  map[string]struct{}{"foo": {}},
			disabled: map[string]struct{}{"foo": {}},
			passes:   []di.OptionalPass{makePass("foo", true)},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pc := PassConfig{Enabled: tc.enabled, Disabled: tc.disabled}
			_, err := pc.resolvePasses(tc.passes)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}
