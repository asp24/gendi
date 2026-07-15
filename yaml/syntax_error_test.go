package yaml

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	di "github.com/gendi-org/gendi"
	"github.com/gendi-org/gendi/srcloc"
)

func writeYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "gendi.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp yaml: %v", err)
	}
	return path
}

// TestSyntaxError covers the various ways a malformed YAML config
// produces a *srcloc.Error with a usable Loc through LoadConfig.
// Each row asserts the structural shape only — see
// TestSyntaxError_BadIndent_ExactLocation for the one exact-position
// regression test against the pinned goccy version.
func TestSyntaxError(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "bad_indent",
			yaml: "services:\n foo: bar\n  baz: qux\n",
		},
		{
			name: "unclosed_quote",
			yaml: "parameters:\n  bad: \"unclosed\n",
		},
		{
			// decoration_priority is int, not string. Triggers a
			// yaml.Error that is NOT a *yaml.SyntaxError (decode error
			// path), exercising the toSrclocError yaml.Error branch.
			name: "decode_type_mismatch",
			yaml: "services:\n  foo:\n    type: string\n    constructor:\n      func: pkg.New\n    decoration_priority: \"abc\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeYAML(t, tt.yaml)

			_, err := LoadConfig(path)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var le *srcloc.Error
			if !errors.As(err, &le) {
				t.Fatalf("expected *srcloc.Error, got %T: %v", err, err)
			}
			if le.Loc == nil {
				t.Fatal("expected non-nil Loc")
			}
			absPath, _ := filepath.Abs(path)
			if le.Loc.File != absPath {
				t.Errorf("Loc.File = %q, want %q", le.Loc.File, absPath)
			}
		})
	}
}

// Spec §Testing #1: exact line/column assertion (catches off-by-one
// regressions when goccy version changes). Values were confirmed by
// running against goccy v1.19.2.
func TestSyntaxError_BadIndent_ExactLocation(t *testing.T) {
	// Carefully-controlled YAML: the second mapping key under "services"
	// has one extra space of indent. goccy reports the error at the
	// second ':' (line 2, column 7) where the inconsistent indent is
	// detected.
	//
	// Line 1: services:
	// Line 2:  foo: bar       (1-space indent — error reported here)
	// Line 3:   baz: qux      (2-space indent — the conflict)
	yaml := "services:\n foo: bar\n  baz: qux\n"
	path := writeYAML(t, yaml)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error")
	}
	var le *srcloc.Error
	if !errors.As(err, &le) {
		t.Fatalf("expected *srcloc.Error, got %T", err)
	}
	if le.Loc == nil {
		t.Fatal("nil Loc")
	}
	// Locked-in for goccy v1.19.2; bump deliberately if the version pin
	// in go.mod changes.
	const wantLine, wantCol = 2, 7
	if le.Loc.Line != wantLine || le.Loc.Column != wantCol {
		t.Errorf("Loc = %d:%d, want %d:%d", le.Loc.Line, le.Loc.Column, wantLine, wantCol)
	}
}

func TestSyntaxError_RenderingProducesSnippet(t *testing.T) {
	// End-to-end: feed a known-bad YAML, render the error, expect a
	// caret-style snippet output.
	yaml := "services:\n foo: bar\n  baz: qux\n"
	path := writeYAML(t, yaml)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error")
	}
	rendered := srcloc.NewRenderer().RenderError(err, 2)
	if !strings.Contains(rendered, "^") {
		t.Errorf("expected caret in rendered output, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "gendi.yaml:") {
		t.Errorf("expected file:line:col prefix, got:\n%s", rendered)
	}
}

func TestSyntaxError_EmptyFile_DoesNotPanic(t *testing.T) {
	path := writeYAML(t, "")
	// Empty file should either return nil config (if goccy treats it as
	// nil doc) or a srcloc.Error without panic. The point is: no panic.
	_, _ = LoadConfig(path)
}

// TestBlockScalar covers the cross-product {parameter value, constructor
// arg} × {block (|), folded (>)} so that LiteralNode handling stays
// equivalent to plain string scalars after the goccy migration.
func TestBlockScalar(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		extract func(t *testing.T, cfg *di.Config) string
		want    string
	}{
		{
			name: "param_pipe",
			yaml: "parameters:\n  greeting: |\n    hello\n    world\n",
			extract: func(_ *testing.T, cfg *di.Config) string {
				return cfg.Parameters["greeting"].Value.String()
			},
			want: "hello\nworld\n",
		},
		{
			name: "param_folded",
			yaml: "parameters:\n  greeting: >\n    hello\n    world\n",
			extract: func(_ *testing.T, cfg *di.Config) string {
				return cfg.Parameters["greeting"].Value.String()
			},
			want: "hello world\n",
		},
		{
			name: "arg_pipe",
			yaml: "services:\n  echo:\n    type: string\n    constructor:\n      func: pkg.New\n      args:\n        - |\n          hello\n          world\n",
			extract: func(t *testing.T, cfg *di.Config) string {
				args := cfg.Services["echo"].Constructor.Args
				if len(args) != 1 {
					t.Fatalf("expected 1 arg, got %d", len(args))
				}
				return args[0].Literal.String()
			},
			want: "hello\nworld\n",
		},
		{
			name: "arg_folded",
			yaml: "services:\n  echo:\n    type: string\n    constructor:\n      func: pkg.New\n      args:\n        - >\n          hello\n          world\n",
			extract: func(t *testing.T, cfg *di.Config) string {
				args := cfg.Services["echo"].Constructor.Args
				if len(args) != 1 {
					t.Fatalf("expected 1 arg, got %d", len(args))
				}
				return args[0].Literal.String()
			},
			want: "hello world\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeYAML(t, tt.yaml)
			cfg, err := LoadConfig(path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := tt.extract(t, cfg)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestE2E_LoadConfigPath_NoDoubleLocation(t *testing.T) {
	// Triggers convertLiteral overflow inside a parameter value.
	// Chain: convertLiteral -> parser:43 (no wrapping for parameter
	// value-decode location since convertLiteral already locates) ->
	// parser:45 AddContext("parameter %q") ->
	// loader:100 AddContext("convert %s") ->
	// (caller would invoke cli:34 AddContext("load config"))
	yaml := `parameters:
  bignum:
    type: int
    value: 9999999999999999999
`
	path := writeYAML(t, yaml)
	absPath, _ := filepath.Abs(path)

	loaderErr := func() error {
		_, err := LoadConfig(path)
		return err
	}()
	if loaderErr == nil {
		t.Fatal("expected loader error")
	}

	// Simulate cli:34 wrapper.
	withCLI := srcloc.AddContext(loaderErr, "load config")

	var le *srcloc.Error
	if !errors.As(withCLI, &le) {
		t.Fatalf("expected *srcloc.Error, got %T: %v", withCLI, withCLI)
	}
	if le.Loc == nil {
		t.Fatal("expected non-nil Loc")
	}
	if le.Loc.File != absPath {
		t.Errorf("Loc.File = %q, want %q", le.Loc.File, absPath)
	}

	msg := withCLI.Error()

	// Structural single-location guarantee: the `file:line:col:`
	// header appears exactly once, at the start. The path string
	// itself may appear again later because the loader's `convert <abs>`
	// AddContext prefix happens to contain it; that is just a prefix
	// string, not a second Loc.
	header := le.Loc.String() + ":"
	if !strings.HasPrefix(msg, header) {
		t.Errorf("expected message to start with %q, got %q", header, msg)
	}
	if strings.Count(msg, header) != 1 {
		t.Errorf("expected location header %q exactly once in %q", header, msg)
	}

	// Visible prefix preserves wrapper context in outermost-first order.
	expectedSubstrings := []string{
		"load config",
		"convert " + absPath,
		"parameter \"bignum\"",
	}
	for _, want := range expectedSubstrings {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q in %q", want, msg)
		}
	}

	// Outermost prefix comes first after the location prefix.
	loadIdx := strings.Index(msg, "load config")
	convertIdx := strings.Index(msg, "convert ")
	paramIdx := strings.Index(msg, "parameter ")
	if !(loadIdx < convertIdx && convertIdx < paramIdx) {
		t.Errorf("prefix ordering wrong in %q (load=%d convert=%d param=%d)",
			msg, loadIdx, convertIdx, paramIdx)
	}

	// Renderer produces snippet with caret on the bignum line.
	rendered := srcloc.NewRenderer().RenderError(withCLI, 2)
	if !strings.Contains(rendered, "^") {
		t.Errorf("expected caret in:\n%s", rendered)
	}
	if !strings.Contains(rendered, "9999999999999999999") {
		t.Errorf("expected source line in:\n%s", rendered)
	}
}
