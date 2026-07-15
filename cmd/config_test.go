package cmd

import (
	"testing"

	di "github.com/gendi-org/gendi"
)

type testPass struct {
	name string
}

func (p *testPass) Name() string                               { return p.name }
func (p *testPass) Process(cfg *di.Config) (*di.Config, error) { return cfg, nil }

func makePass(name string) *testPass {
	return &testPass{name: name}
}

func TestConfig_ResolvePasses(t *testing.T) {
	cases := []struct {
		name             string
		enabled          map[string]struct{}
		passes           []di.Pass
		selectablePasses []di.Pass
		wantNames        []string
	}{
		{
			name:      "always-included pass is included",
			passes:    []di.Pass{makePass("a")},
			wantNames: []string{"a"},
		},
		{
			name:             "selectable pass without enable flag is excluded",
			selectablePasses: []di.Pass{makePass("a")},
			wantNames:        []string{},
		},
		{
			name:             "selectable pass with enable flag is included",
			enabled:          map[string]struct{}{"a": {}},
			selectablePasses: []di.Pass{makePass("a")},
			wantNames:        []string{"a"},
		},
		{
			name:      "duplicate always-included pass name runs only once",
			passes:    []di.Pass{makePass("a"), makePass("a")},
			wantNames: []string{"a"},
		},
		{
			name:             "duplicate selectable pass name runs only once",
			enabled:          map[string]struct{}{"a": {}},
			selectablePasses: []di.Pass{makePass("a"), makePass("a")},
			wantNames:        []string{"a"},
		},
		{
			name:             "always-included pass wins over selectable pass with same name",
			enabled:          map[string]struct{}{"a": {}},
			passes:           []di.Pass{makePass("a")},
			selectablePasses: []di.Pass{makePass("a")},
			wantNames:        []string{"a"},
		},
		{
			name:             "selectable passes are appended after always-included passes",
			enabled:          map[string]struct{}{"b": {}},
			passes:           []di.Pass{makePass("a")},
			selectablePasses: []di.Pass{makePass("b")},
			wantNames:        []string{"a", "b"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{EnabledPasses: tc.enabled}
			result, err := cfg.resolvePasses(tc.passes, tc.selectablePasses)
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

func TestConfig_ResolvePasses_Errors(t *testing.T) {
	cases := []struct {
		name             string
		enabled          map[string]struct{}
		passes           []di.Pass
		selectablePasses []di.Pass
	}{
		{
			name:             "unknown name in --enable-pass",
			enabled:          map[string]struct{}{"unknown": {}},
			selectablePasses: []di.Pass{makePass("foo")},
		},
		{
			name:    "always-included pass is not selectable",
			enabled: map[string]struct{}{"foo": {}},
			passes:  []di.Pass{makePass("foo")},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{EnabledPasses: tc.enabled}
			_, err := cfg.resolvePasses(tc.passes, tc.selectablePasses)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestConfig_ResolvePasses_UnknownErrorIsDeterministic(t *testing.T) {
	cfg := Config{EnabledPasses: map[string]struct{}{"zeta": {}, "alpha": {}, "mu": {}}}
	_, err := cfg.resolvePasses(nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	want := `--enable-pass: unknown pass "alpha"`
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}
