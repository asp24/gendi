package yaml

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/asp24/gendi/srcloc"
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

func TestSyntaxError_BadIndent_ProducesSrclocError(t *testing.T) {
	// Indentation under "services" is broken: "foo" and "baz" disagree.
	yaml := "services:\n foo: bar\n  baz: qux\n"
	path := writeYAML(t, yaml)

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
	if le.Loc.Line < 1 {
		t.Errorf("Loc.Line = %d, want >= 1", le.Loc.Line)
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

func TestSyntaxError_UnclosedQuote_ProducesSrclocError(t *testing.T) {
	yaml := `parameters:
  bad:
    type: string
    value: "unclosed
`
	path := writeYAML(t, yaml)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error")
	}
	var le *srcloc.Error
	if !errors.As(err, &le) {
		t.Fatalf("expected *srcloc.Error, got %T: %v", err, err)
	}
	if le.Loc == nil {
		t.Fatal("expected non-nil Loc")
	}
}

func TestSyntaxError_TypeMismatch_ProducesSrclocError(t *testing.T) {
	// decoration_priority is int, not string. Triggers a yaml.Error
	// that is NOT a *yaml.SyntaxError (decode error path).
	yaml := `services:
  foo:
    type: string
    constructor:
      func: pkg.New
    decoration_priority: "abc"
`
	path := writeYAML(t, yaml)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error")
	}
	var le *srcloc.Error
	if !errors.As(err, &le) {
		t.Fatalf("expected *srcloc.Error, got %T: %v", err, err)
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

func TestBlockScalar_Param_Pipe(t *testing.T) {
	yaml := `parameters:
  greeting:
    type: string
    value: |
      hello
      world
`
	path := writeYAML(t, yaml)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := cfg.Parameters["greeting"].Value.String()
	want := "hello\nworld\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBlockScalar_Param_Folded(t *testing.T) {
	yaml := `parameters:
  greeting:
    type: string
    value: >
      hello
      world
`
	path := writeYAML(t, yaml)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := cfg.Parameters["greeting"].Value.String()
	want := "hello world\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBlockScalar_Arg_Pipe(t *testing.T) {
	yaml := `services:
  echo:
    type: string
    constructor:
      func: pkg.New
      args:
        - |
          hello
          world
`
	path := writeYAML(t, yaml)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	args := cfg.Services["echo"].Constructor.Args
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
	got := args[0].Literal.String()
	want := "hello\nworld\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBlockScalar_Arg_Folded(t *testing.T) {
	yaml := `services:
  echo:
    type: string
    constructor:
      func: pkg.New
      args:
        - >
          hello
          world
`
	path := writeYAML(t, yaml)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	args := cfg.Services["echo"].Constructor.Args
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
	got := args[0].Literal.String()
	want := "hello world\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
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
